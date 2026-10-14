package router

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		http.MethodGet + " /api/user/invoice/applications/:id/payment-proofs/:proof_id",
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

func TestInvoicePaymentProofAccessTokenRequiresWalletRead(t *testing.T) {
	rule, exists := middleware.AccessTokenRouteRule("GET /api/user/invoice/applications/:id/payment-proofs/:proof_id")
	require.True(t, exists)
	assert.Equal(t, "scope", rule.Kind())
	assert.Equal(t, "wallet:read", rule.Scope())
	assert.True(t, service.AccessTokenScopeGranted([]string{"wallet:read"}, rule.Scope()))
	assert.False(t, service.AccessTokenScopeGranted([]string{"profile:read"}, rule.Scope()))
	assert.False(t, service.AccessTokenScopeGranted(nil, rule.Scope()))
}
