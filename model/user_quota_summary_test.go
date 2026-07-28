package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserQuotaSummaryAppliesUserListFilters(t *testing.T) {
	truncateTables(t)
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
