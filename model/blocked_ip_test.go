package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockedIPLifecycleNormalizesAddresses(t *testing.T) {
	truncateTables(t)
	const canonical = "2001:db8::1"
	invalidateBlockedIPCache([]string{canonical})

	require.NoError(t, BlockIPAddresses(
		[]string{"2001:0db8:0:0:0:0:0:1", canonical},
		12,
		99,
		"test block",
	))

	blocked, err := IsIPBlocked(canonical)
	require.NoError(t, err)
	assert.True(t, blocked)

	var records []BlockedIP
	require.NoError(t, DB.Find(&records).Error)
	require.Len(t, records, 1)
	assert.Equal(t, canonical, records[0].IP)
	assert.Equal(t, 12, records[0].UserId)
	assert.Equal(t, 99, records[0].OperatorId)

	require.NoError(t, UnblockIPAddresses([]string{"2001:0db8::1"}))
	blocked, err = IsIPBlocked(canonical)
	require.NoError(t, err)
	assert.False(t, blocked)
}

func TestDisableUserAndBlockIPsIsAtomic(t *testing.T) {
	truncateTables(t)
	user := User{
		Username: "blocked-user",
		Password: "password",
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, DB.Create(&user).Error)
	user.Status = common.UserStatusDisabled

	require.NoError(t, DisableUserAndBlockIPs(
		&user,
		[]string{"198.51.100.10", "2001:db8::10"},
		7,
	))

	var stored User
	require.NoError(t, DB.First(&stored, user.Id).Error)
	assert.Equal(t, common.UserStatusDisabled, stored.Status)

	var count int64
	require.NoError(t, DB.Model(&BlockedIP{}).Where("user_id = ?", user.Id).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

func TestGetSharedLastLoginIPCounts(t *testing.T) {
	truncateTables(t)
	users := []User{
		{Username: "shared-one", Password: "password", AffCode: "shared-one", LastLoginIP: "203.0.113.8"},
		{Username: "shared-two", Password: "password", AffCode: "shared-two", LastLoginIP: "203.0.113.8"},
		{Username: "unique", Password: "password", AffCode: "unique", LastLoginIP: "203.0.113.9"},
	}
	require.NoError(t, DB.Create(&users).Error)

	counts, err := GetSharedLastLoginIPCounts([]string{"203.0.113.8", "203.0.113.9"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), counts["203.0.113.8"])
	assert.Equal(t, int64(1), counts["203.0.113.9"])
}
