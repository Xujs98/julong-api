package controller

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const supportQRPrefix = "/api/support-contacts/qrcode/"

var supportQRPattern = regexp.MustCompile(`^qr-[a-f0-9]{32}\.(png|jpg|webp)$`)
var supportContactsMutex sync.Mutex

type SupportContact struct {
	Id        string `json:"id"`
	Type      string `json:"type"`
	Label     string `json:"label"`
	Value     string `json:"value"`
	QRCodeURL string `json:"qr_code_url,omitempty"`
}

type updateSupportContactsRequest struct {
	Contacts []SupportContact `json:"contacts"`
}

func supportContactsSnapshot() []SupportContact {
	common.OptionMapRWMutex.RLock()
	raw := common.SupportContacts
	common.OptionMapRWMutex.RUnlock()
	contacts := make([]SupportContact, 0)
	_ = common.UnmarshalJsonStr(raw, &contacts)
	for index := range contacts {
		if contacts[index].Id == "" {
			contacts[index].Id = fmt.Sprintf("contact-%d", index+1)
		}
	}
	return contacts
}

func GetSupportContacts(c *gin.Context) {
	common.ApiSuccess(c, supportContactsSnapshot())
}

// QR images are public by unguessable URL, like site logos, so img requests do
// not require exposing dashboard bearer tokens in URLs.
func GetSupportQRCode(c *gin.Context) {
	filename := c.Param("filename")
	if !supportQRPattern.MatchString(filename) {
		c.Status(http.StatusNotFound)
		return
	}
	path := filepath.Join(siteAssetStorageDir(), "support", filename)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "default-src 'none'; sandbox")
	c.File(path)
}

func UpdateSupportContacts(c *gin.Context) {
	// Files remain browser-local until save. Bound the entire multipart request,
	// not only individual files; failed saves remove all newly staged files.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024*1024)
	var request updateSupportContactsRequest
	if c.ContentType() == "multipart/form-data" {
		if err := c.Request.ParseMultipartForm(16 * 1024 * 1024); err != nil {
			common.ApiErrorT(c, "Invalid support contact upload (maximum total size: 16 MB)")
			return
		}
		defer c.Request.MultipartForm.RemoveAll()
		if err := common.UnmarshalJsonStr(c.PostForm("contacts"), &request.Contacts); err != nil {
			common.ApiErrorT(c, "Invalid support contacts")
			return
		}
	} else if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorT(c, "Invalid support contacts")
		return
	}
	if len(request.Contacts) > 30 {
		common.ApiErrorT(c, "Support contacts cannot exceed 30 entries")
		return
	}
	supportContactsMutex.Lock()
	defer supportContactsMutex.Unlock()
	previous := supportContactsSnapshot()
	previousURLs := make(map[string]bool)
	for _, contact := range previous {
		previousURLs[contact.QRCodeURL] = true
	}
	ids := make(map[string]bool)
	allowedTypes := map[string]bool{"qq": true, "wechat": true, "phone": true, "qq_group": true, "wechat_group": true}
	for index := range request.Contacts {
		contact := &request.Contacts[index]
		contact.Id = strings.TrimSpace(contact.Id)
		if contact.Id == "" {
			contact.Id = fmt.Sprintf("contact-%d", index+1)
		}
		contact.Type = strings.ToLower(strings.TrimSpace(contact.Type))
		contact.Label = strings.TrimSpace(contact.Label)
		contact.Value = strings.TrimSpace(contact.Value)
		group := contact.Type == "qq_group" || contact.Type == "wechat_group"
		if !allowedTypes[contact.Type] || contact.Value == "" || len([]rune(contact.Id)) > 64 || ids[contact.Id] || len([]rune(contact.Value)) > 128 || len([]rune(contact.Label)) > 64 {
			common.ApiErrorT(c, "Invalid support contacts")
			return
		}
		ids[contact.Id] = true
		if !group {
			contact.QRCodeURL = ""
		}
		if contact.QRCodeURL != "" && (!strings.HasPrefix(contact.QRCodeURL, supportQRPrefix) || !supportQRPattern.MatchString(strings.TrimPrefix(contact.QRCodeURL, supportQRPrefix)) || !previousURLs[contact.QRCodeURL]) {
			common.ApiErrorT(c, "Invalid support QR code")
			return
		}
	}
	if form := c.Request.MultipartForm; form != nil {
		for field, files := range form.File {
			valid := false
			for _, contact := range request.Contacts {
				if field == "qrcode_"+contact.Id && (contact.Type == "qq_group" || contact.Type == "wechat_group") {
					valid = true
					break
				}
			}
			if !valid || len(files) != 1 {
				common.ApiErrorT(c, "Invalid support QR code")
				return
			}
		}
	}
	dir := filepath.Join(siteAssetStorageDir(), "support")
	created := []string{}
	committed := false
	defer func() {
		if !committed {
			for _, filename := range created {
				_ = os.Remove(filepath.Join(dir, filename))
			}
		}
	}()
	for index := range request.Contacts {
		contact := &request.Contacts[index]
		form := c.Request.MultipartForm
		if form == nil || len(form.File["qrcode_"+contact.Id]) == 0 {
			continue
		}
		header := form.File["qrcode_"+contact.Id][0]
		if header.Size > maxSiteLogoBytes {
			common.ApiErrorT(c, "QR code must be a PNG, JPG or WebP image up to 5 MB and 4096 pixels")
			return
		}
		file, err := header.Open()
		if err != nil {
			common.ApiErrorT(c, "Invalid support QR code")
			return
		}
		data, err := io.ReadAll(io.LimitReader(file, maxSiteLogoBytes+1))
		file.Close()
		if err != nil {
			common.ApiErrorT(c, "Invalid support QR code")
			return
		}
		ext, err := validateSiteLogo(data)
		if err != nil {
			common.ApiErrorT(c, "QR code must be a PNG, JPG or WebP image up to 5 MB and 4096 pixels")
			return
		}
		if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
			common.ApiErrorT(c, "Invalid support QR code")
			return
		}
		filename, err := persistSiteImage(data, ext, dir, "qr-")
		if err != nil {
			common.ApiErrorT(c, "Failed to save support QR code")
			return
		}
		created = append(created, filename)
		contact.QRCodeURL = supportQRPrefix + filename
	}
	if request.Contacts == nil {
		request.Contacts = []SupportContact{}
	}
	encoded, err := common.Marshal(request.Contacts)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.UpdateOption("SupportContacts", string(encoded)); err != nil {
		common.ApiError(c, err)
		return
	}
	committed = true
	retained := make(map[string]bool)
	for _, contact := range request.Contacts {
		retained[contact.QRCodeURL] = true
	}
	for _, contact := range previous {
		filename, local := strings.CutPrefix(contact.QRCodeURL, supportQRPrefix)
		if local && supportQRPattern.MatchString(filename) && !retained[contact.QRCodeURL] {
			if err := os.Remove(filepath.Join(dir, filename)); err != nil && !os.IsNotExist(err) {
				common.SysError("failed to remove obsolete support QR code: " + err.Error())
			}
		}
	}
	common.ApiSuccess(c, request.Contacts)
}
