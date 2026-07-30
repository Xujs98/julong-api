package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheDisablesCachingForMissingAPIRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Cache())
	router.NoRoute(func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/new-route", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusNotFound, response.Code)
	assert.Equal(
		t,
		"no-store, no-cache, must-revalidate, private, max-age=0",
		response.Header().Get("Cache-Control"),
	)
	assert.Equal(t, "no-cache", response.Header().Get("Pragma"))
	assert.Equal(t, "0", response.Header().Get("Expires"))
}
