package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockedDeviceLifecycleAndSessionRevocation(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	user := User{Username: "device-block-user", Password: "password", AuthVersion: 1}
	require.NoError(t, DB.Create(&user).Error)
	blockedDeviceID := uuid.NewString()
	allowedDeviceID := uuid.NewString()
	sessions := []UserSession{
		{
			SID: "blocked-device-session", UserID: user.Id, Version: 1, UserAuthVersion: 1,
			Status: UserSessionStatusActive, RefreshHash: "blocked-hash", LoginMethod: "password",
			IP: "192.0.2.10", UserAgent: "Chrome on macOS", DeviceID: blockedDeviceID,
			CreatedAt: now, LastActiveAt: now, ExpiresAt: now + 3600,
		},
		{
			SID: "allowed-device-session", UserID: user.Id, Version: 1, UserAuthVersion: 1,
			Status: UserSessionStatusActive, RefreshHash: "allowed-hash", LoginMethod: "password",
			IP: "192.0.2.11", UserAgent: "Safari on iOS", DeviceID: allowedDeviceID,
			CreatedAt: now, LastActiveAt: now, ExpiresAt: now + 3600,
		},
	}
	require.NoError(t, DB.Create(&sessions).Error)

	require.NoError(t, BlockUserDevices(user.Id, []string{blockedDeviceID}, 99, "test"))
	blocked, err := IsUserDeviceBlocked(user.Id, blockedDeviceID)
	require.NoError(t, err)
	assert.True(t, blocked)

	revoked, err := RevokeUserSessionsByDeviceIDs(user.Id, []string{blockedDeviceID}, "device_blocked")
	require.NoError(t, err)
	assert.Equal(t, int64(1), revoked)

	blockedSession, err := GetUserSessionBySID("blocked-device-session")
	require.NoError(t, err)
	assert.Equal(t, UserSessionStatusRevoked, blockedSession.Status)
	allowedSession, err := GetUserSessionBySID("allowed-device-session")
	require.NoError(t, err)
	assert.Equal(t, UserSessionStatusActive, allowedSession.Status)

	require.NoError(t, UnblockUserDevices(user.Id, []string{blockedDeviceID}))
	blocked, err = IsUserDeviceBlocked(user.Id, blockedDeviceID)
	require.NoError(t, err)
	assert.False(t, blocked)
}

func TestGetUserLoginDeviceStatsAggregatesSessions(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	user := User{Username: "device-stats-user", Password: "password", AuthVersion: 1}
	require.NoError(t, DB.Create(&user).Error)
	deviceID := uuid.NewString()
	sessions := []UserSession{
		{
			SID: "device-stats-active", UserID: user.Id, Version: 1, UserAuthVersion: 1,
			Status: UserSessionStatusActive, RefreshHash: "active-hash", LoginMethod: "password",
			IP: "192.0.2.20", UserAgent: "latest-agent", DeviceID: deviceID,
			CreatedAt: now, LastActiveAt: now, ExpiresAt: now + 3600,
		},
		{
			SID: "device-stats-revoked", UserID: user.Id, Version: 1, UserAuthVersion: 1,
			Status: UserSessionStatusRevoked, RefreshHash: "revoked-hash", LoginMethod: "password",
			IP: "192.0.2.21", UserAgent: "older-agent", DeviceID: deviceID,
			CreatedAt: now - 60, LastActiveAt: now - 60, ExpiresAt: now + 3600, RevokedAt: now - 30,
		},
	}
	require.NoError(t, DB.Create(&sessions).Error)
	require.NoError(t, BlockUserDevices(user.Id, []string{deviceID}, 99, "test"))

	stats, err := GetUserLoginDeviceStats(user.Id)
	require.NoError(t, err)
	require.Len(t, stats, 1)
	assert.Equal(t, deviceID, stats[0].DeviceId)
	assert.Equal(t, "latest-agent", stats[0].UserAgent)
	assert.EqualValues(t, 2, stats[0].LoginCount)
	assert.EqualValues(t, 1, stats[0].ActiveSessionCount)
	assert.ElementsMatch(t, []string{"192.0.2.20", "192.0.2.21"}, stats[0].IPs)
	assert.True(t, stats[0].Blocked)
}

func TestGetUserLoginDeviceStatsKeepsBlockedDeviceWithoutSessionHistory(t *testing.T) {
	truncateTables(t)
	user := User{Username: "blocked-device-history-user", Password: "password", AuthVersion: 1}
	require.NoError(t, DB.Create(&user).Error)
	deviceID := uuid.NewString()
	require.NoError(t, BlockUserDevices(user.Id, []string{deviceID}, 99, "test"))

	stats, err := GetUserLoginDeviceStats(user.Id)
	require.NoError(t, err)
	require.Len(t, stats, 1)
	assert.Equal(t, deviceID, stats[0].DeviceId)
	assert.True(t, stats[0].Blocked)
	assert.Empty(t, stats[0].IPs)
	assert.Zero(t, stats[0].LoginCount)
}
