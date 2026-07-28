package router

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailSettingsRoutesKeepLegacyConfigAliases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	require.NotPanics(t, func() {
		SetApiRouter(engine)
	})

	routes := make(map[string]struct{}, len(engine.Routes()))
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	assert.Contains(t, routes, http.MethodGet+" /api/email-settings")
	assert.Contains(t, routes, http.MethodPut+" /api/email-settings")
	assert.Contains(t, routes, http.MethodGet+" /api/email-settings/config")
	assert.Contains(t, routes, http.MethodPut+" /api/email-settings/config")
	assert.Contains(t, routes, http.MethodPost+" /api/email-settings/risk-user/test")
}
