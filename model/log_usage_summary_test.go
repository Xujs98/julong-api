package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserUsageSummaryBetweenUsesConsumeLogsAndHalfOpenRange(t *testing.T) {
	truncateTables(t)
	logs := []Log{
		{UserId: 1, Type: LogTypeConsume, CreatedAt: 1000, PromptTokens: 10, CompletionTokens: 5, Quota: 100},
		{UserId: 1, Type: LogTypeConsume, CreatedAt: 1999, PromptTokens: 20, CompletionTokens: 10, Quota: 200},
		{UserId: 1, Type: LogTypeConsume, CreatedAt: 2000, PromptTokens: 100, CompletionTokens: 100, Quota: 1000},
		{UserId: 1, Type: LogTypeRefund, CreatedAt: 1500, PromptTokens: 50, CompletionTokens: 50, Quota: 500},
		{UserId: 2, Type: LogTypeConsume, CreatedAt: 1500, PromptTokens: 50, CompletionTokens: 50, Quota: 900},
	}
	require.NoError(t, LOG_DB.Create(&logs).Error)

	summary, err := GetUserUsageSummaryBetween(1, 1000, 2000)
	require.NoError(t, err)
	require.Equal(t, int64(45), summary.TotalTokens)
	require.Equal(t, int64(300), summary.TotalQuota)
}

func TestGetUserQuotaIncreaseLogsPaginatesStructuredCredits(t *testing.T) {
	truncateTables(t)
	users := []User{
		{Username: "quota-detail-user", Password: "password", Status: common.UserStatusEnabled, AffCode: "quota-detail-one"},
		{Username: "other-quota-detail-user", Password: "password", Status: common.UserStatusEnabled, AffCode: "quota-detail-two"},
	}
	require.NoError(t, DB.Create(&users).Error)

	RecordQuotaIncreaseLog(users[0].Id, 100, QuotaIncreaseSourceCheckin, "check-in reward")
	RecordQuotaIncreaseLog(users[0].Id, 200, QuotaIncreaseSourceRedemption, "redemption reward")
	RecordQuotaIncreaseLog(users[0].Id, 300, QuotaIncreaseSourceAdminAdjustment, "admin adjustment")
	RecordQuotaIncreaseLog(users[1].Id, 900, QuotaIncreaseSourceOnlineRecharge, "other user recharge")
	RecordQuotaIncreaseLog(users[0].Id, 0, QuotaIncreaseSourceRefund, "ignored zero increase")
	RecordLog(users[0].Id, LogTypeTopup, "legacy top-up without structured quota increase")

	items, total, err := GetUserQuotaIncreaseLogs(users[0].Id, 1, 1)
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	require.Len(t, items, 1)
	assert.Equal(t, 200, items[0].Quota)
	assert.Equal(t, QuotaIncreaseSourceRedemption, items[0].Source)
	assert.Equal(t, "redemption reward", items[0].Content)
}
