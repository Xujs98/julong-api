package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedFlowQuotaData(t *testing.T, quotaData QuotaData) {
	t.Helper()
	require.NoError(t, DB.Create(&quotaData).Error)
}

func seedFlowLookupData(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.Create(&Channel{Id: 1, Name: "east"}).Error)
	require.NoError(t, DB.Create(&Channel{Id: 2, Name: "west"}).Error)
	require.NoError(t, DB.Create(&Token{Id: 11, UserId: 1, Key: "sk-primary", Name: "primary"}).Error)
	require.NoError(t, DB.Create(&Token{Id: 22, UserId: 2, Key: "sk-backup", Name: "backup"}).Error)
	require.NoError(t, DB.Delete(&Token{Id: 11}).Error)
}

func TestGetFlowQuotaDataUsesQuotaDataRoleSpecificDimensions(t *testing.T) {
	truncateTables(t)
	seedFlowLookupData(t)

	seedFlowQuotaData(t, QuotaData{
		UserID:    1,
		Username:  "alice",
		NodeName:  "node-a",
		TokenID:   11,
		UseGroup:  "vip",
		ModelName: "gpt-a",
		ChannelID: 1,
		CreatedAt: 1000,
		Count:     2,
		Quota:     100,
		TokenUsed: 40,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID:    1,
		Username:  "alice",
		NodeName:  "node-a",
		TokenID:   11,
		UseGroup:  "vip",
		ModelName: "gpt-a",
		ChannelID: 1,
		CreatedAt: 1100,
		Count:     1,
		Quota:     50,
		TokenUsed: 20,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID:    1,
		Username:  "alice",
		NodeName:  "node-a",
		TokenID:   11,
		UseGroup:  "vip",
		ModelName: "gpt-a",
		ChannelID: 2,
		CreatedAt: 1200,
		Count:     1,
		Quota:     25,
		TokenUsed: 10,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID:    2,
		Username:  "bob",
		NodeName:  "node-b",
		TokenID:   22,
		UseGroup:  "default",
		ModelName: "gpt-b",
		ChannelID: 1,
		CreatedAt: 1300,
		Count:     3,
		Quota:     70,
		TokenUsed: 30,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID:    1,
		Username:  "alice",
		ModelName: "legacy",
		CreatedAt: 1400,
		Count:     99,
		Quota:     999,
		TokenUsed: 999,
	})

	rootRows, err := GetFlowQuotaData(900, 2000, "", 0, common.RoleRootUser)
	require.NoError(t, err)
	require.Len(t, rootRows, 3)
	// Token 11 was soft-deleted, so its name is intentionally left empty for the
	// frontend to render a localized "deleted (id)" label instead.
	require.Equal(t, FlowQuotaData{
		UserID:      1,
		Username:    "alice",
		NodeName:    "node-a",
		TokenID:     11,
		TokenName:   "",
		UseGroup:    "vip",
		ChannelID:   1,
		ChannelName: "east",
		ModelName:   "gpt-a",
		TokenUsed:   60,
		Count:       3,
		Quota:       150,
	}, *rootRows[0])
	// A token that still exists resolves to its current name.
	require.Equal(t, 22, rootRows[1].TokenID)
	require.Equal(t, "backup", rootRows[1].TokenName)

	adminRows, err := GetFlowQuotaData(900, 2000, "alice", 0, common.RoleAdminUser)
	require.NoError(t, err)
	require.Len(t, adminRows, 2)
	require.Equal(t, 0, adminRows[0].TokenID)
	require.Empty(t, adminRows[0].TokenName)
	require.Empty(t, adminRows[0].NodeName)
	require.Equal(t, "alice", adminRows[0].Username)
	require.Equal(t, "vip", adminRows[0].UseGroup)
	require.Equal(t, "east", adminRows[0].ChannelName)
	require.Equal(t, 150, adminRows[0].Quota)

	selfRows, err := GetFlowQuotaData(900, 2000, "", 1, common.RoleCommonUser)
	require.NoError(t, err)
	require.Len(t, selfRows, 1)
	require.Empty(t, selfRows[0].Username)
	require.Equal(t, 0, selfRows[0].ChannelID)
	require.Empty(t, selfRows[0].ChannelName)
	require.Empty(t, selfRows[0].TokenName)
	require.Equal(t, "vip", selfRows[0].UseGroup)
	require.Equal(t, 175, selfRows[0].Quota)
}

func TestLogQuotaDataSplitsRowsByUseGroupTokenChannelAndNode(t *testing.T) {
	truncateTables(t)
	CacheQuotaDataLock.Lock()
	CacheQuotaData = make(map[string]*QuotaData)
	CacheQuotaDataLock.Unlock()

	LogQuotaData(QuotaDataLogParams{
		UserID:               1,
		Username:             "alice",
		ModelName:            "gpt-a",
		CreatedAt:            3661,
		UseGroup:             "vip",
		TokenID:              11,
		ChannelID:            1,
		NodeName:             "node-a",
		Quota:                100,
		TokenUsed:            40,
		UserDisplayQuota:     common.GetPointer(10),
		UserDisplayTokenUsed: common.GetPointer(4),
	})
	LogQuotaData(QuotaDataLogParams{
		UserID:               1,
		Username:             "alice",
		ModelName:            "gpt-a",
		CreatedAt:            3700,
		UseGroup:             "vip",
		TokenID:              11,
		ChannelID:            1,
		NodeName:             "node-a",
		Quota:                50,
		TokenUsed:            20,
		UserDisplayQuota:     common.GetPointer(5),
		UserDisplayTokenUsed: common.GetPointer(2),
	})
	LogQuotaData(QuotaDataLogParams{
		UserID:    1,
		Username:  "alice",
		ModelName: "gpt-a",
		CreatedAt: 3700,
		UseGroup:  "default",
		TokenID:   11,
		ChannelID: 1,
		NodeName:  "node-a",
		Quota:     25,
		TokenUsed: 10,
	})

	SaveQuotaDataCache()

	var rows []QuotaData
	require.NoError(t, DB.Order("quota DESC").Find(&rows).Error)
	require.Len(t, rows, 2)
	require.Equal(t, int64(3600), rows[0].CreatedAt)
	require.Equal(t, "vip", rows[0].UseGroup)
	require.Equal(t, 11, rows[0].TokenID)
	require.Equal(t, 1, rows[0].ChannelID)
	require.Equal(t, "node-a", rows[0].NodeName)
	require.Equal(t, 2, rows[0].Count)
	require.Equal(t, 150, rows[0].Quota)
	require.Equal(t, 60, rows[0].TokenUsed)
	require.NotNil(t, rows[0].UserDisplayQuota)
	require.Equal(t, 15, *rows[0].UserDisplayQuota)
	require.NotNil(t, rows[0].UserDisplayTokenUsed)
	require.Equal(t, 6, *rows[0].UserDisplayTokenUsed)
	require.Equal(t, "default", rows[1].UseGroup)
	require.Equal(t, 25, rows[1].Quota)
	require.NotNil(t, rows[1].UserDisplayQuota)
	require.Equal(t, 25, *rows[1].UserDisplayQuota)
	require.NotNil(t, rows[1].UserDisplayTokenUsed)
	require.Equal(t, 10, *rows[1].UserDisplayTokenUsed)
}

func TestSelfQuotaDataUsesDisplayMetricsWhileAdminKeepsActualMetrics(t *testing.T) {
	truncateTables(t)
	seedFlowQuotaData(t, QuotaData{
		UserID:               1,
		Username:             "alice",
		UseGroup:             "vip",
		ModelName:            "gpt-a",
		CreatedAt:            3600,
		Count:                2,
		Quota:                9500,
		TokenUsed:            1140,
		UserDisplayQuota:     common.GetPointer(1000),
		UserDisplayTokenUsed: common.GetPointer(120),
	})

	selfRows, err := GetQuotaDataByUserId(1, 3600, 3600)
	require.NoError(t, err)
	require.Len(t, selfRows, 1)
	require.Equal(t, 2, selfRows[0].Count)
	require.Equal(t, 1000, selfRows[0].Quota)
	require.Equal(t, 120, selfRows[0].TokenUsed)

	selfFlowRows, err := GetFlowQuotaData(3600, 3600, "", 1, common.RoleCommonUser)
	require.NoError(t, err)
	require.Len(t, selfFlowRows, 1)
	require.Equal(t, 1000, selfFlowRows[0].Quota)
	require.Equal(t, 120, selfFlowRows[0].TokenUsed)

	adminFlowRows, err := GetFlowQuotaData(3600, 3600, "alice", 0, common.RoleAdminUser)
	require.NoError(t, err)
	require.Len(t, adminFlowRows, 1)
	require.Equal(t, 9500, adminFlowRows[0].Quota)
	require.Equal(t, 1140, adminFlowRows[0].TokenUsed)

	adminRows, err := GetAllQuotaDates(3600, 3600, "alice")
	require.NoError(t, err)
	require.Len(t, adminRows, 1)
	require.Equal(t, 9500, adminRows[0].Quota)
	require.Equal(t, 1140, adminRows[0].TokenUsed)
}

func TestGetQuotaDataGroupByUseGroupAggregatesMetricsAndDistinctUsers(t *testing.T) {
	truncateTables(t)
	seedFlowQuotaData(t, QuotaData{
		UserID: 1, Username: "alice", UseGroup: "vip", CreatedAt: 1000,
		Count: 2, Quota: 100, TokenUsed: 40,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID: 1, Username: "alice", UseGroup: "vip", CreatedAt: 1100,
		Count: 1, Quota: 50, TokenUsed: 20,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID: 2, Username: "bob", UseGroup: "vip", CreatedAt: 1200,
		Count: 3, Quota: 75, TokenUsed: 30,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID: 1, Username: "alice", UseGroup: "default", CreatedAt: 1300,
		Count: 4, Quota: 25, TokenUsed: 10,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID: 3, Username: "outside", UseGroup: "ignored", CreatedAt: 3000,
		Count: 99, Quota: 999, TokenUsed: 999,
	})

	analytics, err := GetQuotaDataGroupByUseGroup(900, 2000, "")
	require.NoError(t, err)
	require.Len(t, analytics.Items, 2)
	require.Equal(t, GroupQuotaData{
		UseGroup: "vip", Quota: 225, Count: 6, TokenUsed: 90, UserCount: 2,
	}, analytics.Items[0])
	require.Equal(t, GroupQuotaDataTotals{
		Quota: 250, Count: 10, TokenUsed: 100, UserCount: 2, GroupCount: 2,
	}, analytics.Totals)

	aliceAnalytics, err := GetQuotaDataGroupByUseGroup(900, 2000, "alice")
	require.NoError(t, err)
	require.Len(t, aliceAnalytics.Items, 2)
	require.Equal(t, int64(175), aliceAnalytics.Totals.Quota)
	require.Equal(t, int64(1), aliceAnalytics.Totals.UserCount)
}

func TestGetDashboardReportDataIncludesTopUsersAndGroups(t *testing.T) {
	truncateTables(t)
	seedFlowQuotaData(t, QuotaData{
		UserID: 1, Username: "alice", UseGroup: "vip", ModelName: "gpt-a", ChannelID: 1,
		CreatedAt: 1000, Count: 2, Quota: 100, TokenUsed: 40,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID: 1, Username: "alice", UseGroup: "vip", ModelName: "gpt-b", ChannelID: 2,
		CreatedAt: 1100, Count: 1, Quota: 50, TokenUsed: 20,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID: 2, Username: "bob", UseGroup: "default", ModelName: "gpt-a", ChannelID: 1,
		CreatedAt: 1200, Count: 3, Quota: 75, TokenUsed: 30,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID: 3, Username: "outside", UseGroup: "ignored", ModelName: "outside", ChannelID: 3,
		CreatedAt: 3000, Count: 99, Quota: 999, TokenUsed: 999,
	})

	report, err := GetDashboardReportData(900, 2000)
	require.NoError(t, err)
	require.Len(t, report.TopUsers, 2)
	require.Len(t, report.TopGroups, 2)
	assert.Equal(t, DashboardReportUser{
		Username: "alice", Quota: 150, Count: 3, TokenUsed: 60,
	}, report.TopUsers[0])
	assert.Equal(t, DashboardReportUser{
		Username: "bob", Quota: 75, Count: 3, TokenUsed: 30,
	}, report.TopUsers[1])
	assert.Equal(t, GroupQuotaData{
		UseGroup: "vip", Quota: 150, Count: 3, TokenUsed: 60, UserCount: 1,
	}, report.TopGroups[0])
	assert.Equal(t, GroupQuotaData{
		UseGroup: "default", Quota: 75, Count: 3, TokenUsed: 30, UserCount: 1,
	}, report.TopGroups[1])
}

func TestSaveQuotaDataCacheIncrementsActualAndDisplayMetricsIndependently(t *testing.T) {
	truncateTables(t)
	CacheQuotaDataLock.Lock()
	CacheQuotaData = make(map[string]*QuotaData)
	CacheQuotaDataLock.Unlock()
	seedFlowQuotaData(t, QuotaData{
		UserID:               1,
		Username:             "alice",
		UseGroup:             "vip",
		ModelName:            "gpt-a",
		CreatedAt:            3600,
		Count:                1,
		Quota:                100,
		TokenUsed:            40,
		UserDisplayQuota:     common.GetPointer(10),
		UserDisplayTokenUsed: common.GetPointer(4),
	})
	LogQuotaData(QuotaDataLogParams{
		UserID:               1,
		Username:             "alice",
		UseGroup:             "vip",
		ModelName:            "gpt-a",
		CreatedAt:            3661,
		Quota:                50,
		TokenUsed:            20,
		UserDisplayQuota:     common.GetPointer(5),
		UserDisplayTokenUsed: common.GetPointer(2),
	})

	SaveQuotaDataCache()

	var stored QuotaData
	require.NoError(t, DB.Where("username = ?", "alice").First(&stored).Error)
	require.Equal(t, 2, stored.Count)
	require.Equal(t, 150, stored.Quota)
	require.Equal(t, 60, stored.TokenUsed)
	require.NotNil(t, stored.UserDisplayQuota)
	require.Equal(t, 15, *stored.UserDisplayQuota)
	require.NotNil(t, stored.UserDisplayTokenUsed)
	require.Equal(t, 6, *stored.UserDisplayTokenUsed)
}

func TestBackfillQuotaDataDisplayMetricsPreservesLegacyActualValues(t *testing.T) {
	truncateTables(t)
	seedFlowQuotaData(t, QuotaData{
		UserID:    1,
		Username:  "legacy",
		ModelName: "gpt-a",
		CreatedAt: 3600,
		Count:     1,
		Quota:     321,
		TokenUsed: 654,
	})

	require.NoError(t, backfillQuotaDataUserDisplayMetrics())

	var stored QuotaData
	require.NoError(t, DB.Where("username = ?", "legacy").First(&stored).Error)
	require.NotNil(t, stored.UserDisplayQuota)
	require.Equal(t, 321, *stored.UserDisplayQuota)
	require.NotNil(t, stored.UserDisplayTokenUsed)
	require.Equal(t, 654, *stored.UserDisplayTokenUsed)
}
