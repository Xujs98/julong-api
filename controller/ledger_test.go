package controller

import (
	"encoding/csv"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateLedgerMutationEnforcesInputBounds(t *testing.T) {
	valid := ledgerMutationRequest{
		Platform: "OpenAI", Account: "account", Email: "owner@example.com", Type: "plus",
		Quota: 500_000, CostPrice: decimal.RequireFromString("12.5"), Quantity: 1, OccurredAt: 1_800_000_000,
	}
	require.NoError(t, validateLedgerMutation(&valid))

	tests := []struct {
		name   string
		mutate func(*ledgerMutationRequest)
	}{
		{name: "unknown platform", mutate: func(request *ledgerMutationRequest) { request.Platform = "Unknown" }},
		{name: "missing account", mutate: func(request *ledgerMutationRequest) { request.Account = "" }},
		{name: "invalid email", mutate: func(request *ledgerMutationRequest) { request.Email = "not-an-email" }},
		{name: "missing type", mutate: func(request *ledgerMutationRequest) { request.Type = "" }},
		{name: "zero quota", mutate: func(request *ledgerMutationRequest) { request.Quota = 0 }},
		{name: "quota over database limit", mutate: func(request *ledgerMutationRequest) { request.Quota = common.MaxQuota + 1 }},
		{name: "zero cost", mutate: func(request *ledgerMutationRequest) { request.CostPrice = decimal.Zero }},
		{name: "cost over limit", mutate: func(request *ledgerMutationRequest) { request.CostPrice = decimal.NewFromInt(1_000_000_001) }},
		{name: "zero quantity", mutate: func(request *ledgerMutationRequest) { request.Quantity = 0 }},
		{name: "quantity over limit", mutate: func(request *ledgerMutationRequest) { request.Quantity = maxLedgerQuantity + 1 }},
		{name: "invalid date", mutate: func(request *ledgerMutationRequest) { request.OccurredAt = 0 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := valid
			test.mutate(&request)
			assert.Error(t, validateLedgerMutation(&request))
		})
	}
}

func TestDecodeLedgerMutationRoundsCostPriceToStoragePrecision(t *testing.T) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest("POST", "/api/ledger", strings.NewReader(
		`{"platform":"OpenAI","account":"account","type":"plus","quota":500000,"cost_price":"12.3456789","quantity":1,"occurred_at":1800000000}`,
	))
	request, ok := decodeLedgerMutation(context)
	require.True(t, ok)
	require.NotNil(t, request)
	assert.True(t, decimal.RequireFromString("12.345679").Equal(request.CostPrice))

	recorder = httptest.NewRecorder()
	context, _ = gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest("POST", "/api/ledger", strings.NewReader(
		`{"platform":"OpenAI","account":"account","type":"plus","quota":500000,"cost_price":"0.0000001","quantity":1,"occurred_at":1800000000}`,
	))
	request, ok = decodeLedgerMutation(context)
	assert.False(t, ok)
	assert.Nil(t, request)
}

func TestRenderLedgerCSVIncludesBOMAndPreventsFormulaInjection(t *testing.T) {
	payload, err := renderLedgerCSV([]model.LedgerEntry{{
		Id: 7, OccurredAt: 1, Platform: "OpenAI", Account: "=HYPERLINK(\"https://example.com\")",
		Email: "+owner@example.com", Type: "plus", Quota: 500_000,
		CostPrice: decimal.RequireFromString("1.5"), Quantity: 2, CreatedBy: 1, CreatedAt: 1, UpdatedAt: 1,
	}})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(payload), "\xEF\xBB\xBF"))

	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(payload), "\xEF\xBB\xBF")))
	rows, err := reader.ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "'=HYPERLINK(\"https://example.com\")", rows[1][3])
	assert.Equal(t, "'+owner@example.com", rows[1][4])
	assert.Equal(t, "1.500000", rows[1][7])
}
