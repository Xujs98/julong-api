package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestLedgerDateFilteringAndAggregateCalculations(t *testing.T) {
	truncateTables(t)
	entries := []LedgerEntry{
		{Platform: "OpenAI", Account: "plus-a", Type: "plus", Quota: 500_000, CostPrice: decimal.RequireFromString("1.25"), Quantity: 2, OccurredAt: 100, CreatedBy: 1},
		{Platform: "Anthropic", Account: "pro-a", Type: "pro", Quota: 1_000_000, CostPrice: decimal.RequireFromString("3"), Quantity: 1, OccurredAt: 200, CreatedBy: 1},
		{Platform: "Gemini", Account: "outside", Type: "plus", Quota: 2_000_000, CostPrice: decimal.RequireFromString("8"), Quantity: 1, OccurredAt: 300, CreatedBy: 1},
	}
	require.NoError(t, DB.Create(&entries).Error)

	filtered, total, err := GetLedgerEntries(100, 200, 0, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, filtered, 2)
	assert.Equal(t, entries[1].Id, filtered[0].Id)
	assert.Equal(t, entries[0].Id, filtered[1].Id)

	aggregate, err := GetLedgerAggregate(100, 200)
	require.NoError(t, err)
	assert.Equal(t, int64(2_000_000), aggregate.TotalQuota)
	assert.True(t, decimal.RequireFromString("5.5").Equal(aggregate.TotalCost))
	assert.Equal(t, int64(1_000_000), aggregate.QuotaByType["plus"])
	assert.Equal(t, int64(1_000_000), aggregate.QuotaByType["pro"])
	assert.True(t, decimal.RequireFromString("2.5").Equal(aggregate.CostByType["plus"]))
	assert.True(t, decimal.RequireFromString("3").Equal(aggregate.CostByType["pro"]))
}

func TestDeleteLedgerEntriesSoftDeletesSingleAndBatchTargets(t *testing.T) {
	truncateTables(t)
	entries := []LedgerEntry{
		{Platform: "OpenAI", Account: "one", Type: "plus", Quota: 1, CostPrice: decimal.NewFromInt(1), Quantity: 1, OccurredAt: 100, CreatedBy: 1},
		{Platform: "OpenAI", Account: "two", Type: "plus", Quota: 1, CostPrice: decimal.NewFromInt(1), Quantity: 1, OccurredAt: 100, CreatedBy: 1},
		{Platform: "OpenAI", Account: "three", Type: "plus", Quota: 1, CostPrice: decimal.NewFromInt(1), Quantity: 1, OccurredAt: 100, CreatedBy: 1},
	}
	require.NoError(t, DB.Create(&entries).Error)

	deleted, err := DeleteLedgerEntries([]int{entries[0].Id, entries[2].Id})
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	remaining, total, err := GetLedgerEntries(0, 0, 0, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, remaining, 1)
	assert.Equal(t, entries[1].Id, remaining[0].Id)

	_, err = GetLedgerEntryById(entries[0].Id)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	var softDeletedCount int64
	require.NoError(t, DB.Unscoped().Model(&LedgerEntry{}).
		Where("id IN ? AND deleted_at IS NOT NULL", []int{entries[0].Id, entries[2].Id}).
		Count(&softDeletedCount).Error)
	assert.Equal(t, int64(2), softDeletedCount)
}

func TestLedgerSettingsDefaultValidateAndPersistEstimateRatio(t *testing.T) {
	truncateTables(t)
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	previous, existed := common.OptionMap[common.LedgerEstimateRatioOptionKey]
	delete(common.OptionMap, common.LedgerEstimateRatioOptionKey)
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		if existed {
			common.OptionMap[common.LedgerEstimateRatioOptionKey] = previous
		} else {
			delete(common.OptionMap, common.LedgerEstimateRatioOptionKey)
		}
		common.OptionMapRWMutex.Unlock()
	})

	assert.True(t, decimal.NewFromInt(1).Equal(GetLedgerSettings().EstimateRatio))
	_, err := UpdateLedgerSettings(decimal.Zero)
	assert.Error(t, err)
	_, err = UpdateLedgerSettings(decimal.RequireFromString("1000.000001"))
	assert.Error(t, err)

	settings, err := UpdateLedgerSettings(decimal.RequireFromString("1.7500004"))
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("1.75").Equal(settings.EstimateRatio))
	assert.True(t, decimal.RequireFromString("1.75").Equal(GetLedgerSettings().EstimateRatio))

	var option Option
	require.NoError(t, DB.Where("key = ?", common.LedgerEstimateRatioOptionKey).First(&option).Error)
	assert.Equal(t, "1.75", option.Value)
}

func TestGetLedgerUsageQuotaSumsPositiveConsumptionInDateRange(t *testing.T) {
	truncateTables(t)
	logs := []Log{
		{Type: LogTypeConsume, CreatedAt: 100, Quota: 100},
		{Type: LogTypeConsume, CreatedAt: 150, Quota: 60},
		{Type: LogTypeConsume, CreatedAt: 200, Quota: 10},
		{Type: LogTypeConsume, CreatedAt: 175, Quota: -5},
		{Type: LogTypeConsume, CreatedAt: 201, Quota: 999},
		{Type: LogTypeRefund, CreatedAt: 150, Quota: 999},
	}
	require.NoError(t, LOG_DB.Create(&logs).Error)

	usage, err := GetLedgerUsageQuota(100, 200)
	require.NoError(t, err)
	assert.Equal(t, int64(170), usage.Real)
}
