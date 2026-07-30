package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLedgerSummarySumsOperatingCostsWithoutMultiplyingSelectedDays(t *testing.T) {
	truncate(t)
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	previous, existed := common.OptionMap[common.UserQuotaSummaryExcludedUserIDsOptionKey]
	previousEstimateRatio, estimateRatioExisted := common.OptionMap[common.LedgerEstimateRatioOptionKey]
	common.OptionMap[common.UserQuotaSummaryExcludedUserIDsOptionKey] = "[]"
	common.OptionMap[common.LedgerEstimateRatioOptionKey] = "2"
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		if existed {
			common.OptionMap[common.UserQuotaSummaryExcludedUserIDsOptionKey] = previous
		} else {
			delete(common.OptionMap, common.UserQuotaSummaryExcludedUserIDsOptionKey)
		}
		if estimateRatioExisted {
			common.OptionMap[common.LedgerEstimateRatioOptionKey] = previousEstimateRatio
		} else {
			delete(common.OptionMap, common.LedgerEstimateRatioOptionKey)
		}
		common.OptionMapRWMutex.Unlock()
	})

	users := []model.User{
		{Username: "ledger-user-a", Password: "password", Group: "default", Quota: 500_000, AffCode: "ledger-user-a"},
		{Username: "ledger-user-b", Password: "password", Group: "default", Quota: 1_000_000, AffCode: "ledger-user-b"},
	}
	require.NoError(t, model.DB.Create(&users).Error)
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local).Unix()
	end := time.Date(2026, 7, 3, 23, 59, 59, 0, time.Local).Unix()
	other, err := common.Marshal(map[string]interface{}{"group_ratio": 2.0})
	require.NoError(t, err)
	require.NoError(t, model.LOG_DB.Create(&model.Log{
		Type: model.LogTypeConsume, CreatedAt: start + 60, Quota: 500_000, Other: string(other),
	}).Error)
	entries := []model.LedgerEntry{
		{Platform: "OpenAI", Account: "plus", Type: "plus", Quota: 500_000, CostPrice: decimal.NewFromInt(2), Quantity: 2, OccurredAt: start, CreatedBy: 1},
		{Platform: "Anthropic", Account: "pro", Type: "pro", Quota: 500_000, CostPrice: decimal.NewFromInt(3), Quantity: 1, OccurredAt: start + 1, CreatedBy: 1},
	}
	require.NoError(t, model.DB.Create(&entries).Error)

	summary, err := GetLedgerSummary(start, end)
	require.NoError(t, err)
	assert.Equal(t, int64(1_500_000), summary.UserQuota.Real)
	assert.Equal(t, float64(750_000), summary.UserQuota.Estimated)
	assert.Equal(t, int64(500_000), summary.UsageQuota.Real)
	assert.Equal(t, float64(250_000), summary.UsageQuota.Estimated)
	assert.True(t, decimal.NewFromInt(2).Equal(summary.EstimateRatio))
	assert.True(t, decimal.NewFromInt(7).Equal(summary.DailyOperatingCost))
	assert.True(t, decimal.NewFromInt(7).Equal(summary.TotalOperatingCost))
	assert.Equal(t, 3, summary.Days)
	assert.Equal(t, int64(2), summary.LedgerEntryCount)
	assert.Equal(t, int64(2), summary.IncludedUserCount)
	assert.Equal(t, int64(1_500_000), summary.OperationalQuota.Real)
	require.NotNil(t, summary.OperationalQuota.CostRatio)
	assert.True(t, decimal.RequireFromString("2.333333").Equal(*summary.OperationalQuota.CostRatio))
	require.NotNil(t, summary.CostRatios["plus"])
	assert.True(t, decimal.NewFromInt(2).Equal(*summary.CostRatios["plus"]))
	require.NotNil(t, summary.CostRatios["pro"])
	assert.True(t, decimal.NewFromInt(3).Equal(*summary.CostRatios["pro"]))
	assert.Nil(t, summary.CostRatios["k12"])
}
