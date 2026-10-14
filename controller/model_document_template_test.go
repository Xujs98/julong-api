package controller

import (
	"archive/zip"
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelDocumentUploadFormatsAndLimits(t *testing.T) {
	markdown := "# Video API\n\n| Field | Value |\n| --- | --- |\n| seconds | 8 |\n\n```json\n{\"seconds\":8}\n```\n"
	content, format, err := service.ParseModelDocumentUpload("guide.MD", []byte(markdown))
	require.NoError(t, err)
	assert.Equal(t, markdown, content)
	assert.Equal(t, "markdown", format)

	var word bytes.Buffer
	writer := zip.NewWriter(&word)
	part, err := writer.Create("word/document.xml")
	require.NoError(t, err)
	_, err = part.Write([]byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Video API</w:t></w:r></w:p><w:p><w:pPr><w:pStyle w:val="Code"/></w:pPr><w:r><w:t>curl -X POST /v1/videos</w:t></w:r></w:p><w:tbl><w:tr><w:tc><w:p><w:r><w:t>Field</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>Value</w:t></w:r></w:p></w:tc></w:tr><w:tr><w:tc><w:p><w:r><w:t>seconds</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>8</w:t></w:r></w:p></w:tc></w:tr></w:tbl><w:p><w:pPr><w:numPr/></w:pPr><w:r><w:t>Poll task</w:t></w:r></w:p></w:body></w:document>`))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	content, format, err = service.ParseModelDocumentUpload("guide.docx", word.Bytes())
	require.NoError(t, err)
	assert.Equal(t, "docx", format)
	assert.Contains(t, content, "# Video API")
	assert.Contains(t, content, "````\ncurl -X POST /v1/videos\n````")
	assert.Contains(t, content, "| Field | Value |\n| --- | --- |\n| seconds | 8 |")
	assert.Contains(t, content, "- Poll task")
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"empty.md", nil},
		{"binary.md", []byte{0xff, 0xfe, 0x01}},
		{"nul.md", []byte("hello\x00world")},
		{"too-large.md", bytes.Repeat([]byte{'x'}, model.MaxDocumentTemplateContentBytes+1)},
		{"bad.docx", []byte("not a ZIP")},
		{"legacy.doc", []byte("binary doc")},
		{"active.html", []byte("<script>alert(1)</script>")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := service.ParseModelDocumentUpload(tc.name, tc.data)
			require.Error(t, err)
		})
	}
	var bomb bytes.Buffer
	zipWriter := zip.NewWriter(&bomb)
	part, err = zipWriter.Create("word/document.xml")
	require.NoError(t, err)
	_, err = part.Write(bytes.Repeat([]byte{'a'}, 8*1024*1024+1))
	require.NoError(t, err)
	require.NoError(t, zipWriter.Close())
	_, _, err = service.ParseModelDocumentUpload("compressed-too-large.docx", bomb.Bytes())
	require.Error(t, err)
}

func TestModelDocumentTemplateDatabaseMatrix(t *testing.T) {
	for _, dialect := range []struct{ kind, env string }{{"sqlite", ""}, {"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"}} {
		t.Run(dialect.kind, func(t *testing.T) {
			if dialect.env != "" && os.Getenv(dialect.env) == "" {
				t.Skip("set " + dialect.env + " to run this database")
			}
			database := modelManagementDB(t, dialect.kind, os.Getenv(dialect.env))
			require.NoError(t, database.AutoMigrate(&model.ModelDocumentTemplate{}))
			require.NoError(t, database.AutoMigrate(&model.Model{}, &model.ModelDocumentTemplate{}))
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("file", "video-guide.md")
			require.NoError(t, err)
			// More than MySQL TEXT's 64 KiB must survive unchanged.
			content := "# Video API\n\n" + strings.Repeat("Document body.\n", 6000)
			_, err = part.Write([]byte(content))
			require.NoError(t, err)
			require.NoError(t, writer.Close())
			response := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(response)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/model-document-templates", &body)
			ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())
			ctx.Set("role", common.RoleRootUser)
			CreateModelDocumentTemplate(ctx)
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			var saved struct {
				Success bool
				Data    model.ModelDocumentTemplate
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &saved))
			require.True(t, saved.Success, response.Body.String())
			require.Positive(t, saved.Data.ID)
			assert.Equal(t, content, string(saved.Data.Content))
			assert.Equal(t, "video-guide", saved.Data.Name)
			assert.Equal(t, "markdown", saved.Data.Format)
			assert.Positive(t, saved.Data.CreatedTime)
			record := model.Model{ModelName: "video-model", Status: 1, SyncOfficial: 1, DocumentTemplateID: saved.Data.ID}
			require.NoError(t, record.Insert())
			previousGroups := setting.UserUsableGroups2JSONString()
			require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default group"}`))
			t.Cleanup(func() { require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(previousGroups)) })
			channel := model.Channel{Name: "documentation-test", Type: 1, Key: "not-a-real-key", Status: common.ChannelStatusEnabled}
			require.NoError(t, database.Create(&channel).Error)
			require.NoError(t, database.Create(&model.Ability{Group: "default", Model: record.ModelName, ChannelId: channel.Id, Enabled: true}).Error)
			privateModel := model.Model{ModelName: "private-video", Status: 1, DocumentTemplateID: saved.Data.ID}
			require.NoError(t, privateModel.Insert())
			require.NoError(t, database.Create(&model.Ability{Group: "private", Model: privateModel.ModelName, ChannelId: channel.Id, Enabled: true}).Error)
			model.InvalidatePricingCache()
			t.Cleanup(model.InvalidatePricingCache)
			var public struct {
				Success bool
				Data    *model.ModelDocumentTemplate
			}
			modelManagementRequest(t, GetModelDocumentation, http.MethodGet, "/api/model-documentation?model=video-model", nil, &public)
			require.True(t, public.Success)
			require.NotNil(t, public.Data)
			assert.Equal(t, content, string(public.Data.Content))
			for _, name := range []string{"private-video", "absent-video"} {
				result := modelManagementRequest(t, GetModelDocumentation, http.MethodGet, "/api/model-documentation?model="+name, nil, nil)
				assert.Equal(t, http.StatusNotFound, result.Code, result.Body.String())
				assert.NotContains(t, result.Body.String(), "Document body")
			}
			document, err := model.GetModelDocumentTemplate(record.ModelName)
			require.NoError(t, err)
			require.NotNil(t, document)
			assert.Equal(t, content, string(document.Content))

			var listing struct {
				Success bool
				Data    struct {
					Items []model.ModelDocumentTemplate
					Total int
				}
			}
			modelManagementRequest(t, ListModelDocumentTemplates, http.MethodGet, "/api/model-document-templates?page=1&page_size=20", nil, &listing)
			require.True(t, listing.Success)
			require.Len(t, listing.Data.Items, 1)
			assert.Empty(t, listing.Data.Items[0].Content, "list must not load whole document contents")
			assert.Equal(t, 1, listing.Data.Total)

			for _, update := range []map[string]string{{"name": "Renamed guide"}, {"content": "# Updated\n\n```json\n{}\n```"}} {
				encoded, err := common.Marshal(update)
				require.NoError(t, err)
				response = httptest.NewRecorder()
				ctx, _ = gin.CreateTestContext(response)
				ctx.Request = httptest.NewRequest(http.MethodPut, "/api/model-document-templates/"+strconv.Itoa(saved.Data.ID), bytes.NewReader(encoded))
				ctx.Request.Header.Set("Content-Type", "application/json")
				ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(saved.Data.ID)}}
				ctx.Set("role", common.RoleRootUser)
				UpdateModelDocumentTemplate(ctx)
				require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			}
			document, err = model.GetModelDocumentTemplate(record.ModelName)
			require.NoError(t, err)
			require.NotNil(t, document)
			assert.Equal(t, "Renamed guide", document.Name)
			assert.Equal(t, "# Updated\n\n```json\n{}\n```", string(document.Content))
			// Hiding a model also hides its documentation; restoring it does not
			// require changing or duplicating the shared template.
			require.NoError(t, database.Model(&record).Update("status", 0).Error)
			model.InvalidatePricingCache()
			result := modelManagementRequest(t, GetModelDocumentation, http.MethodGet, "/api/model-documentation?model=video-model", nil, nil)
			assert.Equal(t, http.StatusNotFound, result.Code, result.Body.String())
			require.NoError(t, database.Model(&record).Update("status", 1).Error)
			model.InvalidatePricingCache()
			// Metadata-only upstream sync must not overwrite the local binding.
			require.NoError(t, database.Model(&record).Update("description", "synced description").Error)
			require.NoError(t, database.First(&record, record.Id).Error)
			assert.Equal(t, saved.Data.ID, record.DocumentTemplateID)
			record.DocumentTemplateID = saved.Data.ID + 10000
			assert.Error(t, record.Update())
			require.NoError(t, model.DeleteModelDocumentTemplate(saved.Data.ID))
			require.NoError(t, database.First(&record, record.Id).Error)
			assert.Zero(t, record.DocumentTemplateID)
			document, err = model.GetModelDocumentTemplate(record.ModelName)
			require.NoError(t, err)
			assert.Nil(t, document)
			assert.Error(t, model.DeleteModelDocumentTemplate(saved.Data.ID))
		})
	}
}
