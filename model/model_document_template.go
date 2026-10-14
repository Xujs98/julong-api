package model

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const MaxDocumentTemplateContentBytes = 2 * 1024 * 1024

// ModelDocumentTemplate stores normalized Markdown, not executable Word/HTML.
// Original uploads are not retained on disk or exposed through public URLs.
type ModelDocumentTemplate struct {
	ID               int      `json:"id"`
	Name             string   `json:"name" gorm:"size:128;not null"`
	Format           string   `json:"format" gorm:"size:16;not null"`
	OriginalFilename string   `json:"original_filename" gorm:"size:255"`
	Content          LongText `json:"content,omitempty" gorm:"not null"`
	CreatedTime      int64    `json:"created_time" gorm:"bigint;not null"`
	UpdatedTime      int64    `json:"updated_time" gorm:"bigint;not null"`
}

func ValidateDocumentTemplateContent(content string) error {
	if strings.TrimSpace(content) == "" {
		return common.NewMessage("Document template content cannot be empty")
	}
	if len(content) > MaxDocumentTemplateContentBytes {
		return common.NewMessage("Document template content exceeds 2 MB")
	}
	if !utf8.ValidString(content) || strings.ContainsRune(content, 0) {
		return common.NewMessage("Document template must contain valid UTF-8 text")
	}
	return nil
}

// Fields restrict partial updates so a rename cannot overwrite concurrent edits
// to the body. Omitted fields mean a deliberate full-document replacement.
func SaveModelDocumentTemplate(document *ModelDocumentTemplate, create bool, fields ...string) error {
	document.Name = strings.TrimSpace(document.Name)
	if document.Name == "" || !utf8.ValidString(document.Name) || strings.ContainsRune(document.Name, 0) || utf8.RuneCountInString(document.Name) > 128 {
		return common.NewMessage("Document template name must contain 1 to 128 characters")
	}
	if err := ValidateDocumentTemplateContent(string(document.Content)); err != nil {
		return err
	}
	return metadataTransaction(func(tx *gorm.DB) error {
		now := common.GetTimestamp()
		document.UpdatedTime = now
		if create {
			document.CreatedTime = now
			return tx.Create(document).Error
		}
		if len(fields) == 0 {
			fields = []string{"name", "format", "original_filename", "content"}
		}
		fields = append(fields, "updated_time")
		result := tx.Model(&ModelDocumentTemplate{}).Where("id = ?", document.ID).
			Select(fields).Updates(document)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			var count int64
			if err := tx.Model(&ModelDocumentTemplate{}).Where("id = ?", document.ID).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return gorm.ErrRecordNotFound
			}
		}
		return tx.First(document, document.ID).Error
	})
}

func DeleteModelDocumentTemplate(id int) error {
	return metadataTransaction(func(tx *gorm.DB) error {
		var document ModelDocumentTemplate
		if err := lockForUpdate(tx).First(&document, id).Error; err != nil {
			return err
		}
		// Sharing the metadata mutation lock with model saves prevents dangling
		// associations, including when a template is deleted concurrently.
		if err := tx.Model(&Model{}).Where("document_template_id = ?", id).
			Updates(map[string]any{"document_template_id": 0, "updated_time": common.GetTimestamp()}).Error; err != nil {
			return err
		}
		return tx.Delete(&document).Error
	})
}

func validateModelDocumentTemplate(tx *gorm.DB, id int) error {
	if id == 0 {
		return nil
	}
	if id < 0 {
		return common.NewMessage("Invalid document template")
	}
	var count int64
	if err := tx.Model(&ModelDocumentTemplate{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return common.NewMessage("Document template no longer exists")
	}
	return nil
}

// GetModelDocumentTemplate follows the same exact/prefix/suffix/contains
// metadata precedence as the public model catalog. Callers authorize the model.
func GetModelDocumentTemplate(name string) (*ModelDocumentTemplate, error) {
	var metadata []Model
	if err := DB.Select("id", "model_name", "name_rule", "status", "document_template_id").Order("id").Find(&metadata).Error; err != nil {
		return nil, err
	}
	matched := resolveModelMetadata(metadata, []string{name})[name]
	if matched == nil || matched.Status != 1 || matched.DocumentTemplateID == 0 {
		return nil, nil
	}
	var document ModelDocumentTemplate
	if err := DB.First(&document, matched.DocumentTemplateID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &document, nil
}
