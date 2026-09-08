package router

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestInvoiceAPIRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)

	routes := make(map[string]struct{}, len(engine.Routes()))
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	for _, route := range []string{
		http.MethodGet + " /api/user/invoice",
		http.MethodPost + " /api/user/invoice/apply",
		http.MethodPost + " /api/user/invoice/applications/:id/cancel",
		http.MethodGet + " /api/user/invoice/applications/:id/attachment",
		http.MethodGet + " /api/user/invoice/applications",
		http.MethodGet + " /api/user/invoice/email-templates",
		http.MethodPut + " /api/user/invoice/applications/:id",
		http.MethodPost + " /api/user/invoice/applications/:id/complete",
		http.MethodPost + " /api/user/invoice/applications/:id/withdraw",
	} {
		_, exists := routes[route]
		assert.True(t, exists, route)
	}
}
