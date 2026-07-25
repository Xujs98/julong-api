package controller

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildSiteLogoPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.RGBA{R: 42, G: 96, B: 180, A: 255})
	var data bytes.Buffer
	require.NoError(t, png.Encode(&data, img))
	return data.Bytes()
}

func buildSiteLogoUploadRequest(t *testing.T, data []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "logo.png")
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, "/api/site-assets/logo", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func TestValidateSiteLogo(t *testing.T) {
	ext, err := validateSiteLogo(buildSiteLogoPNG(t, 128, 64))
	require.NoError(t, err)
	assert.Equal(t, "png", ext)

	_, err = validateSiteLogo([]byte("not an image"))
	assert.ErrorContains(t, err, "仅支持")

	_, err = validateSiteLogo(buildSiteLogoPNG(t, maxSiteLogoDimension+1, 1))
	assert.ErrorContains(t, err, "4096")
}

func TestSiteLogoUploadReadAndDelete(t *testing.T) {
	setupManageUserTestDB(t)
	gin.SetMode(gin.TestMode)
	storageDir := t.TempDir()
	t.Setenv("SITE_ASSET_STORAGE_DIR", storageDir)

	previousLogo := common.Logo
	common.OptionMapRWMutex.Lock()
	common.Logo = ""
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.Logo = previousLogo
		common.OptionMapRWMutex.Unlock()
	})

	uploadResponse := httptest.NewRecorder()
	uploadContext, _ := gin.CreateTestContext(uploadResponse)
	uploadContext.Request = buildSiteLogoUploadRequest(t, buildSiteLogoPNG(t, 128, 128))
	uploadContext.Set("id", 1)
	uploadContext.Set("role", common.RoleRootUser)
	UploadSiteLogo(uploadContext)

	assert.Equal(t, http.StatusOK, uploadResponse.Code)
	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(uploadResponse.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	filename, ok := siteLogoFilename(payload.Data.URL)
	require.True(t, ok)
	_, err := os.Stat(filepath.Join(storageDir, filename))
	require.NoError(t, err)

	readResponse := httptest.NewRecorder()
	readContext, _ := gin.CreateTestContext(readResponse)
	readContext.Params = gin.Params{{Key: "filename", Value: filename}}
	GetSiteLogo(readContext)
	assert.Equal(t, http.StatusOK, readResponse.Code)
	assert.Equal(t, "image/png", readResponse.Header().Get("Content-Type"))
	assert.Equal(t, "nosniff", readResponse.Header().Get("X-Content-Type-Options"))

	common.OptionMapRWMutex.Lock()
	common.Logo = payload.Data.URL
	common.OptionMapRWMutex.Unlock()
	deleteResponse := httptest.NewRecorder()
	deleteContext, _ := gin.CreateTestContext(deleteResponse)
	deleteContext.Request = httptest.NewRequest(http.MethodDelete, payload.Data.URL, nil)
	deleteContext.Params = gin.Params{{Key: "filename", Value: filename}}
	DeleteSiteLogo(deleteContext)
	assert.Equal(t, http.StatusConflict, deleteResponse.Code)

	common.OptionMapRWMutex.Lock()
	common.Logo = ""
	common.OptionMapRWMutex.Unlock()
	deleteResponse = httptest.NewRecorder()
	deleteContext, _ = gin.CreateTestContext(deleteResponse)
	deleteContext.Request = httptest.NewRequest(http.MethodDelete, payload.Data.URL, nil)
	deleteContext.Params = gin.Params{{Key: "filename", Value: filename}}
	deleteContext.Set("id", 1)
	DeleteSiteLogo(deleteContext)
	assert.Equal(t, http.StatusOK, deleteResponse.Code)
	_, err = os.Stat(filepath.Join(storageDir, filename))
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestUploadSiteLogoRejectsUnsupportedContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = buildSiteLogoUploadRequest(t, []byte(strings.Repeat("x", 1024)))

	UploadSiteLogo(context)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "仅支持")
}

func TestRemoveObsoleteSiteLogoOnlyRemovesReplacedLocalFile(t *testing.T) {
	storageDir := t.TempDir()
	t.Setenv("SITE_ASSET_STORAGE_DIR", storageDir)
	filename := "logo-0123456789abcdef0123456789abcdef.png"
	require.NoError(t, os.WriteFile(filepath.Join(storageDir, filename), []byte("old"), 0o640))

	removeObsoleteSiteLogo(siteLogoURLPrefix+filename, "https://example.com/logo.png")

	_, err := os.Stat(filepath.Join(storageDir, filename))
	assert.ErrorIs(t, err, os.ErrNotExist)
}
