package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserImageGenerationLogAccess(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	userId := 8801

	allowed, limit, err := GetUserImageGenerationLogAccess(userId)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Zero(t, limit)

	require.NoError(t, DB.Create(&UserSubscription{
		UserId: userId, Status: "active", EndTime: now + 3600,
		AllowImageGenerationLogs: true, ImageGenerationLogLimit: 20,
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		UserId: userId, Status: "active", EndTime: now + 3600,
		AllowImageGenerationLogs: true, ImageGenerationLogLimit: 50,
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		UserId: userId, Status: "expired", EndTime: now - 1,
		AllowImageGenerationLogs: true, ImageGenerationLogLimit: 100,
	}).Error)

	allowed, limit, err = GetUserImageGenerationLogAccess(userId)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 50, limit)

	require.NoError(t, DB.Create(&UserSubscription{
		UserId: userId, Status: "active", EndTime: now + 3600,
		AllowImageGenerationLogs: true, ImageGenerationLogLimit: 0,
	}).Error)
	allowed, limit, err = GetUserImageGenerationLogAccess(userId)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Zero(t, limit)
}
