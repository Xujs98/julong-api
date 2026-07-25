package controller

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"golang.org/x/image/webp"
)

const (
	maxSiteLogoBytes     = 5 * 1024 * 1024
	maxSiteLogoDimension = 4096
	siteLogoURLPrefix    = "/api/site-assets/logo/"
)

var siteLogoFilenamePattern = regexp.MustCompile(`^logo-[a-f0-9]{32}\.(png|jpg|webp)$`)

func siteAssetStorageDir() string {
	if dir := strings.TrimSpace(os.Getenv("SITE_ASSET_STORAGE_DIR")); dir != "" {
		return dir
	}
	return "site-assets"
}

func siteLogoFilename(rawURL string) (string, bool) {
	path := strings.SplitN(strings.TrimSpace(rawURL), "?", 2)[0]
	if !strings.HasPrefix(path, siteLogoURLPrefix) {
		return "", false
	}
	filename := strings.TrimPrefix(path, siteLogoURLPrefix)
	if !siteLogoFilenamePattern.MatchString(filename) {
		return "", false
	}
	return filename, true
}

func validateSiteLogo(data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("徽标文件不能为空")
	}
	if len(data) > maxSiteLogoBytes {
		return "", fmt.Errorf("徽标文件不能超过 %d MB", maxSiteLogoBytes/(1024*1024))
	}

	mimeType := http.DetectContentType(data)
	var (
		width  int
		height int
		err    error
		ext    string
	)
	switch mimeType {
	case "image/png":
		config, decodeErr := png.DecodeConfig(bytes.NewReader(data))
		width, height, err, ext = config.Width, config.Height, decodeErr, "png"
	case "image/jpeg":
		config, decodeErr := jpeg.DecodeConfig(bytes.NewReader(data))
		width, height, err, ext = config.Width, config.Height, decodeErr, "jpg"
	case "image/webp":
		config, decodeErr := webp.DecodeConfig(bytes.NewReader(data))
		width, height, err, ext = config.Width, config.Height, decodeErr, "webp"
	default:
		return "", errors.New("仅支持 PNG、JPG 和 WebP 图片")
	}
	if err != nil || width <= 0 || height <= 0 {
		return "", errors.New("无法解析徽标图片")
	}
	if width > maxSiteLogoDimension || height > maxSiteLogoDimension {
		return "", fmt.Errorf("徽标图片宽高不能超过 %d 像素", maxSiteLogoDimension)
	}
	return ext, nil
}

func persistSiteLogo(data []byte, ext string) (string, error) {
	dir := siteAssetStorageDir()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}

	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	filename := "logo-" + hex.EncodeToString(random) + "." + ext
	temporary, err := os.CreateTemp(dir, "logo-upload-*")
	if err != nil {
		return "", err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0o640); err != nil {
		temporary.Close()
		return "", err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return "", err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return "", err
	}
	if err := temporary.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(temporaryPath, filepath.Join(dir, filename)); err != nil {
		return "", err
	}
	return filename, nil
}

func removeObsoleteSiteLogo(previousURL, nextURL string) {
	previousFilename, previousIsLocal := siteLogoFilename(previousURL)
	nextFilename, nextIsLocal := siteLogoFilename(nextURL)
	if !previousIsLocal || (nextIsLocal && previousFilename == nextFilename) {
		return
	}
	if err := os.Remove(filepath.Join(siteAssetStorageDir(), previousFilename)); err != nil && !errors.Is(err, os.ErrNotExist) {
		common.SysError("failed to remove obsolete site logo: " + err.Error())
	}
}

func UploadSiteLogo(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSiteLogoBytes+1024*1024)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "message": "徽标文件不能超过 5 MB"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请选择徽标图片"})
		return
	}
	if fileHeader.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "徽标文件不能为空"})
		return
	}
	if fileHeader.Size > maxSiteLogoBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "message": "徽标文件不能超过 5 MB"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无法读取徽标图片"})
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxSiteLogoBytes+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无法读取徽标图片"})
		return
	}
	ext, err := validateSiteLogo(data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	filename, err := persistSiteLogo(data, ext)
	if err != nil {
		common.SysError("failed to persist site logo: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "保存徽标图片失败"})
		return
	}

	recordManageAudit(c, "site_logo.upload", map[string]interface{}{"filename": filename})
	common.ApiSuccess(c, gin.H{"url": siteLogoURLPrefix + filename})
}

func GetSiteLogo(c *gin.Context) {
	filename := c.Param("filename")
	if !siteLogoFilenamePattern.MatchString(filename) {
		c.Status(http.StatusNotFound)
		return
	}
	data, err := os.ReadFile(filepath.Join(siteAssetStorageDir(), filename))
	if errors.Is(err, os.ErrNotExist) {
		c.Status(http.StatusNotFound)
		return
	}
	if err != nil {
		common.SysError("failed to read site logo: " + err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}
	mimeType := http.DetectContentType(data)
	if mimeType != "image/png" && mimeType != "image/jpeg" && mimeType != "image/webp" {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, mimeType, data)
}

func DeleteSiteLogo(c *gin.Context) {
	filename := c.Param("filename")
	if !siteLogoFilenamePattern.MatchString(filename) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的徽标文件"})
		return
	}
	common.OptionMapRWMutex.RLock()
	currentFilename, currentIsLocal := siteLogoFilename(common.Logo)
	common.OptionMapRWMutex.RUnlock()
	if currentIsLocal && currentFilename == filename {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "当前正在使用的徽标不能直接删除"})
		return
	}
	if err := os.Remove(filepath.Join(siteAssetStorageDir(), filename)); err != nil && !errors.Is(err, os.ErrNotExist) {
		common.SysError("failed to delete site logo: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "删除徽标图片失败"})
		return
	}
	recordManageAudit(c, "site_logo.delete", map[string]interface{}{"filename": filename})
	common.ApiSuccess(c, nil)
}
