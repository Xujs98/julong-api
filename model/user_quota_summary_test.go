package model

import (
	"fmt"
	"sort"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetUserQuotaSummarySettingsForTest(t *testing.T) {
	t.Helper()
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	previous, existed := common.OptionMap[common.UserQuotaSummaryExcludedUserIDsOptionKey]
	common.OptionMap[common.UserQuotaSummaryExcludedUserIDsOptionKey] = "[]"
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		if existed {
			common.OptionMap[common.UserQuotaSummaryExcludedUserIDsOptionKey] = previous
		} else {
			delete(common.OptionMap, common.UserQuotaSummaryExcludedUserIDsOptionKey)
		}
		common.OptionMapRWMutex.Unlock()
	})
}

func TestGetUserQuotaSummaryAppliesUserListFilters(t *testing.T) {
	truncateTables(t)
	resetUserQuotaSummarySettingsForTest(t)
	users := []User{
		{Username: "alice", DisplayName: "Alice", Email: "alice@example.com", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", Quota: 100, TagId: 7, AffCode: "quota-summary-alice"},
		{Username: "bob", DisplayName: "Bob", Email: "bob@example.com", Password: "password", Role: common.RoleAdminUser, Status: common.UserStatusDisabled, Group: "vip", Quota: 200, TagId: 7, AffCode: "quota-summary-bob"},
		{Username: "carol", DisplayName: "Carol", Email: "carol@example.com", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "vip", Quota: 300, AffCode: "quota-summary-carol"},
		{Username: "alex", DisplayName: "Alex", Email: "alex@example.com", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "vip", Quota: 400, TagId: 8, AffCode: "quota-summary-alex"},
	}
	require.NoError(t, DB.Create(&users).Error)
	require.NoError(t, DB.Delete(&users[2]).Error)

	commonRole := common.RoleCommonUser
	enabledStatus := common.UserStatusEnabled
	disabledStatus := common.UserStatusDisabled
	deletedStatus := -1
	tagSeven := 7
	tagEight := 8
	tests := []struct {
		name       string
		keyword    string
		group      string
		role       *int
		status     *int
		tagId      *int
		totalQuota int64
		userCount  int64
	}{
		{name: "all users including deleted", totalQuota: 1000, userCount: 4},
		{name: "keyword", keyword: "ali", totalQuota: 100, userCount: 1},
		{name: "group", group: "vip", totalQuota: 900, userCount: 3},
		{name: "role", role: &commonRole, totalQuota: 800, userCount: 3},
		{name: "enabled status excludes deleted", status: &enabledStatus, totalQuota: 500, userCount: 2},
		{name: "disabled status", status: &disabledStatus, totalQuota: 200, userCount: 1},
		{name: "deleted status", status: &deletedStatus, totalQuota: 300, userCount: 1},
		{name: "custom tag", tagId: &tagSeven, totalQuota: 300, userCount: 2},
		{name: "combined filters", keyword: "alex", group: "vip", role: &commonRole, status: &enabledStatus, tagId: &tagEight, totalQuota: 400, userCount: 1},
		{name: "no matches", keyword: "missing", totalQuota: 0, userCount: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			summary, err := GetUserQuotaSummary(test.keyword, test.group, test.role, test.status, test.tagId)
			require.NoError(t, err)
			assert.Equal(t, test.totalQuota, summary.TotalQuota)
			assert.Equal(t, test.userCount, summary.UserCount)
		})
	}
}

func TestUserQuotaSummarySettingsPersistAndExcludeSelectedUsers(t *testing.T) {
	truncateTables(t)
	resetUserQuotaSummarySettingsForTest(t)
	users := []User{
		{Username: "quota-user", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", Quota: 100, AffCode: "quota-settings-user"},
		{Username: "quota-admin", Password: "password", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, Group: "default", Quota: 200, AffCode: "quota-settings-admin"},
		{Username: "quota-root", Password: "password", Role: common.RoleRootUser, Status: common.UserStatusEnabled, Group: "default", Quota: 300, AffCode: "quota-settings-root"},
		{Username: "quota-deleted", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", Quota: 400, AffCode: "quota-settings-deleted"},
	}
	require.NoError(t, DB.Create(&users).Error)
	require.NoError(t, DB.Delete(&users[3]).Error)

	excludedUserIds := []int{users[3].Id, users[0].Id, users[3].Id, 0}
	settings, err := UpdateUserQuotaSummarySettings(excludedUserIds)
	require.NoError(t, err)
	expectedIds := []int{users[0].Id, users[3].Id}
	sort.Ints(expectedIds)
	assert.Equal(t, expectedIds, settings.ExcludedUserIds)

	loadedSettings, err := GetUserQuotaSummarySettings()
	require.NoError(t, err)
	assert.Equal(t, expectedIds, loadedSettings.ExcludedUserIds)
	var option Option
	require.NoError(t, DB.First(&option, "key = ?", common.UserQuotaSummaryExcludedUserIDsOptionKey).Error)
	assert.Equal(t, fmt.Sprintf("[%d,%d]", expectedIds[0], expectedIds[1]), option.Value)

	summary, err := GetUserQuotaSummary("", "", nil, nil, nil)
	require.NoError(t, err)
	assert.EqualValues(t, 500, summary.TotalQuota)
	assert.EqualValues(t, 2, summary.UserCount)

	options, total, err := SearchUserQuotaSummaryOptions("quota-", 0, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 4, total)
	require.Len(t, options, 4)
	resolved, err := ResolveUserQuotaSummaryOptions([]int{users[1].Id, users[2].Id, users[3].Id})
	require.NoError(t, err)
	require.Len(t, resolved, 3)
}

func TestUpdateUserQuotaSummarySettingsRejectsMissingUsers(t *testing.T) {
	truncateTables(t)
	resetUserQuotaSummarySettingsForTest(t)

	_, err := UpdateUserQuotaSummarySettings([]int{999999})
	assert.EqualError(t, err, "部分用户不存在")
	settings, getErr := GetUserQuotaSummarySettings()
	require.NoError(t, getErr)
	assert.Empty(t, settings.ExcludedUserIds)
}
