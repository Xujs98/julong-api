package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateTextQuotaSummaryTracksBillingRevenueCounterfactuals(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	inputAdjustment := 0.1
	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "revenue-model",
		PriceData: types.PriceData{
			ModelRatio:      1,
			CompletionRatio: 1,
			ModelTokenAdjustment: types.ModelTokenAdjustment{
				Input: &inputAdjustment,
			},
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio:        2,
				GroupSpecialRatio: 2,
				PricingGroupRatio: 1.5,
				HasSpecialRatio:   true,
			},
		},
		StartTime: time.Now(),
	}
	usage := &dto.Usage{PromptTokens: 100, TotalTokens: 100}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)
	audit := buildBillingRevenueAudit(summary.Quota, summary.BillingCounterfactuals)

	require.NotNil(t, audit)
	require.NotNil(t, audit.OriginalQuota)
	require.NotNil(t, audit.GroupSpecialRatio)
	require.NotNil(t, audit.ModelTokenAdjustment)
	assert.Equal(t, 220, summary.Quota)
	assert.Equal(t, int64(150), *audit.OriginalQuota)
	assert.Equal(t, int64(55), *audit.GroupSpecialRatio)
	assert.Equal(t, int64(20), *audit.ModelTokenAdjustment)
}

func TestTieredBillingRevenueUsesUnadjustedTokensAndPricingGroup(t *testing.T) {
	inputAdjustment := 0.1
	expr := `tier("default", p * 2)`
	relayInfo := &relaycommon.RelayInfo{
		PriceData: types.PriceData{
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio:        2,
				GroupSpecialRatio: 2,
				PricingGroupRatio: 1.5,
				HasSpecialRatio:   true,
			},
		},
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:  "tiered_expr",
			ExprString:   expr,
			ExprHash:     billingexpr.ExprHashString(expr),
			GroupRatio:   2,
			QuotaPerUnit: 500_000,
			ModelTokenAdjustment: types.ModelTokenAdjustment{
				Input: &inputAdjustment,
			},
		},
	}
	params := billingexpr.TokenParams{P: 100, Len: 100}
	ok, finalQuota, result := TryTieredSettle(relayInfo, params)
	require.True(t, ok)
	require.NotNil(t, result)

	counterfactuals := calculateTieredBillingRevenueCounterfactuals(relayInfo, params, result, decimal.Zero)
	audit := buildBillingRevenueAudit(finalQuota, counterfactuals)

	require.NotNil(t, audit)
	require.NotNil(t, audit.OriginalQuota)
	require.NotNil(t, audit.GroupSpecialRatio)
	require.NotNil(t, audit.ModelTokenAdjustment)
	assert.Equal(t, 220, finalQuota)
	assert.Equal(t, int64(150), *audit.OriginalQuota)
	assert.Equal(t, int64(55), *audit.GroupSpecialRatio)
	assert.Equal(t, int64(20), *audit.ModelTokenAdjustment)
}

func TestAudioBillingRevenueTracksOriginalQuota(t *testing.T) {
	inputAdjustment := 0.1
	relayInfo := &relaycommon.RelayInfo{
		PriceData: types.PriceData{
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio:        2,
				GroupSpecialRatio: 2,
				PricingGroupRatio: 1.5,
				HasSpecialRatio:   true,
			},
		},
	}
	quotaInfo := QuotaInfo{
		InputDetails: TokenDetails{TextTokens: 100},
		ModelName:    "revenue-model",
		ModelRatio:   1,
		GroupRatio:   2,
		ModelTokenAdjustment: types.ModelTokenAdjustment{
			Input: &inputAdjustment,
		},
	}
	finalQuota, _ := calculateAudioQuota(quotaInfo)
	counterfactuals := calculateAudioBillingRevenueCounterfactuals(relayInfo, quotaInfo)
	audit := buildBillingRevenueAudit(finalQuota, counterfactuals)

	require.NotNil(t, audit)
	require.NotNil(t, audit.OriginalQuota)
	assert.Equal(t, 220, finalQuota)
	assert.Equal(t, int64(150), *audit.OriginalQuota)
}

func TestBuildBillingRevenueAuditPreservesDiscountsAsNegativeRevenue(t *testing.T) {
	withoutSpecialRatio := 100
	audit := buildBillingRevenueAudit(80, billingRevenueCounterfactuals{
		WithoutGroupSpecialRatio: &withoutSpecialRatio,
	})

	require.NotNil(t, audit)
	require.NotNil(t, audit.GroupSpecialRatio)
	assert.Equal(t, int64(-20), *audit.GroupSpecialRatio)
}
