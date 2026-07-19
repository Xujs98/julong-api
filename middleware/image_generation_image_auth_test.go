package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestImageGenerationTaskImageAuthWhitelistBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousEnabled := common.ImageGenerationLogImageAuthWhitelistEnabled
	previousWhitelist := common.ImageGenerationLogImageAuthWhitelist
	t.Cleanup(func() {
		common.ImageGenerationLogImageAuthWhitelistEnabled = previousEnabled
		common.ImageGenerationLogImageAuthWhitelist = previousWhitelist
	})

	common.ImageGenerationLogImageAuthWhitelistEnabled = true
	common.ImageGenerationLogImageAuthWhitelist = "images.example.com"

	router := gin.New()
	router.GET("/image", ImageGenerationTaskImageAuth(), func(c *gin.Context) {
		require.True(t, c.GetBool(string(constant.ContextKeyImageGenerationImageAuthBypassed)))
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/image", nil)
	request.Header.Set("Referer", "https://images.example.com/tasks/1")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestImageGenerationTaskImageAuthRequiresTokenWhenDisabledOrMismatched(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousEnabled := common.ImageGenerationLogImageAuthWhitelistEnabled
	previousWhitelist := common.ImageGenerationLogImageAuthWhitelist
	t.Cleanup(func() {
		common.ImageGenerationLogImageAuthWhitelistEnabled = previousEnabled
		common.ImageGenerationLogImageAuthWhitelist = previousWhitelist
	})

	for _, enabled := range []bool{false, true} {
		common.ImageGenerationLogImageAuthWhitelistEnabled = enabled
		common.ImageGenerationLogImageAuthWhitelist = "images.example.com"
		router := gin.New()
		router.GET("/image", ImageGenerationTaskImageAuth(), func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})

		request := httptest.NewRequest(http.MethodGet, "/image", nil)
		request.Header.Set("Referer", "https://other.example.com/tasks/1")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusUnauthorized, recorder.Code)
	}
}
