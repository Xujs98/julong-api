package controller

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func ListModelDocumentTemplates(c *gin.Context) {
	page := common.GetPageQuery(c)
	var documents []model.ModelDocumentTemplate
	query := model.DB.Model(&model.ModelDocumentTemplate{})
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		common.ApiErrorStatus(c, http.StatusInternalServerError, common.NewMessage("Failed to load document templates"))
		return
	}
	if err := query.Omit("content").Order("id DESC").Offset(page.GetStartIdx()).Limit(page.GetPageSize()).Find(&documents).Error; err != nil {
		common.ApiErrorStatus(c, http.StatusInternalServerError, common.NewMessage("Failed to load document templates"))
		return
	}
	common.ApiSuccess(c, gin.H{"items": documents, "total": total, "page": page.GetPage(), "page_size": page.GetPageSize()})
}

func GetModelDocumentTemplate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Invalid document template"))
		return
	}
	var document model.ModelDocumentTemplate
	if err := model.DB.First(&document, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorStatus(c, http.StatusNotFound, common.NewMessage("Document template no longer exists"))
		} else {
			common.ApiErrorStatus(c, http.StatusInternalServerError, common.NewMessage("Failed to load document templates"))
		}
		return
	}
	common.ApiSuccess(c, document)
}

func CreateModelDocumentTemplate(c *gin.Context) {
	saveModelDocumentTemplate(c, true)
}

func UpdateModelDocumentTemplate(c *gin.Context) {
	saveModelDocumentTemplate(c, false)
}

func saveModelDocumentTemplate(c *gin.Context, create bool) {
	var document model.ModelDocumentTemplate
	var fields []string
	if !create {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Invalid document template"))
			return
		}
		if err := model.DB.First(&document, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				common.ApiErrorStatus(c, http.StatusNotFound, common.NewMessage("Document template no longer exists"))
			} else {
				common.ApiErrorStatus(c, http.StatusInternalServerError, common.NewMessage("Failed to load document templates"))
			}
			return
		}
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.MaxDocumentTemplateUploadBytes+1024*1024)
	mediaType, _, _ := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if mediaType == "multipart/form-data" {
		if err := c.Request.ParseMultipartForm(service.MaxDocumentTemplateUploadBytes + 1024*1024); err != nil {
			common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Document upload is invalid or too large"))
			return
		}
		defer c.Request.MultipartForm.RemoveAll()
		fileHeader, err := c.FormFile("file")
		if err != nil || fileHeader.Size <= 0 || fileHeader.Size > service.MaxDocumentTemplateUploadBytes {
			common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Document file must be between 1 byte and 10 MB"))
			return
		}
		file, err := fileHeader.Open()
		if err != nil {
			common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Document upload is invalid or too large"))
			return
		}
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, service.MaxDocumentTemplateUploadBytes+1))
		if err != nil {
			common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Document upload is invalid or too large"))
			return
		}
		content, format, err := service.ParseModelDocumentUpload(fileHeader.Filename, data)
		if err != nil {
			common.ApiErrorStatus(c, http.StatusBadRequest, err)
			return
		}
		document.Content, document.Format = model.LongText(content), format
		fields = append(fields, "content", "format", "original_filename")
		filename := filepath.Base(strings.ReplaceAll(fileHeader.Filename, "\\", "/"))
		if utf8.RuneCountInString(filename) > 255 {
			common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Document filename exceeds 255 characters"))
			return
		}
		document.OriginalFilename = filename
		if name, supplied := c.GetPostForm("name"); supplied {
			document.Name = name
			fields = append(fields, "name")
		} else if create {
			document.Name = strings.TrimSuffix(filename, filepath.Ext(filename))
		}
	} else {
		if create {
			common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Select a document to upload"))
			return
		}
		var patch struct {
			Name    *string `json:"name"`
			Content *string `json:"content"`
		}
		if err := common.DecodeJson(c.Request.Body, &patch); err != nil || (patch.Name == nil && patch.Content == nil) {
			common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Provide a document name or content"))
			return
		}
		if patch.Name != nil {
			document.Name = *patch.Name
			fields = append(fields, "name")
		}
		if patch.Content != nil {
			document.Content = model.LongText(*patch.Content)
			fields = append(fields, "content")
		}
	}
	if err := model.SaveModelDocumentTemplate(&document, create, fields...); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorStatus(c, http.StatusNotFound, common.NewMessage("Document template no longer exists"))
		} else {
			common.ApiErrorStatus(c, http.StatusBadRequest, err)
		}
		return
	}
	recordManageAudit(c, "model_document_template.save", map[string]any{"template_id": document.ID, "create": create, "format": document.Format})
	common.ApiSuccess(c, document)
}

func DeleteModelDocumentTemplate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Invalid document template"))
		return
	}
	if err := model.DeleteModelDocumentTemplate(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorStatus(c, http.StatusNotFound, common.NewMessage("Document template no longer exists"))
		} else {
			common.ApiErrorStatus(c, http.StatusInternalServerError, common.NewMessage("Failed to delete document template"))
		}
		return
	}
	recordManageAudit(c, "model_document_template.delete", map[string]any{"template_id": id})
	common.ApiSuccess(c, nil)
}

// GetModelDocumentation reveals a template only through a visible model in
// the caller's usable catalog. It never permits public enumeration by template ID.
func GetModelDocumentation(c *gin.Context) {
	name := strings.TrimSpace(c.Query("model"))
	if name == "" || len(name) > 1024 {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Model name cannot be empty"))
		return
	}
	group := ""
	if id, exists := c.Get("id"); exists {
		user, err := model.GetUserCache(id.(int))
		if err != nil {
			common.ApiErrorStatus(c, http.StatusInternalServerError, common.NewMessage("Failed to load document templates"))
			return
		}
		group = user.Group
	}
	catalog := filterPricingByUsableGroups(model.GetPricing(), service.GetUserUsableGroups(group))
	visible := false
	for _, item := range catalog {
		if item.ModelName == name {
			visible = true
			break
		}
	}
	if !visible {
		common.ApiErrorStatus(c, http.StatusNotFound, common.NewMessage("Model documentation is unavailable"))
		return
	}
	document, err := model.GetModelDocumentTemplate(name)
	if err != nil {
		common.ApiErrorStatus(c, http.StatusInternalServerError, common.NewMessage("Failed to load document templates"))
		return
	}
	common.ApiSuccess(c, document)
}
