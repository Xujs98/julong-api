package service

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserPresenceTransitionsNotifyOnceAndTimeoutAfterFiveMinutes(t *testing.T) {
	_, users := setupEmailSettingsTest(t)
	_, err := UpdateEmailSettingsConfig(EmailSettingsConfig{
		LowBalanceEmailThreshold:          1,
		AccountQuotaEmailThreshold:        1,
		UserPresenceEmailEnabled:          true,
		UserPresenceEmailEvents:           []string{UserPresenceEventOnline, UserPresenceEventOffline},
		UserPresenceEmailMonitoredUserIDs: []int{users["common"].Id},
		UserPresenceEmailRecipientIDs:     []int{users["admin"].Id},
		UserPresenceOfflineMinutes:        5,
	})
	require.NoError(t, err)

	var subjects []string
	sender := func(subject, receiver, content string) error {
		subjects = append(subjects, subject)
		assert.Equal(t, users["admin"].Email, receiver)
		assert.Contains(t, content, users["common"].Username)
		return nil
	}
	config := currentUserPresenceRuntimeConfig()
	now := time.Date(2026, time.July, 29, 12, 10, 0, 0, time.Local)
	require.NoError(t, recordUserPresenceActivityAt(
		users["common"].Id,
		now.Add(-6*time.Minute).Unix(),
		UserPresenceActivitySourceAPI,
		"203.0.113.10",
		"Codex CLI",
		config,
		sender,
	))
	require.Len(t, subjects, 1)

	require.NoError(t, recordUserPresenceActivityAt(
		users["common"].Id,
		now.Add(-5*time.Minute-30*time.Second).Unix(),
		UserPresenceActivitySourceDashboard,
		"203.0.113.10",
		"Browser",
		config,
		sender,
	))
	require.Len(t, subjects, 1)

	result, err := dispatchUserPresenceEmailsAt(context.Background(), now, sender)
	require.NoError(t, err)
	assert.Equal(t, 1, result.OfflineUserCount)
	assert.Equal(t, 1, result.RecipientCount)
	require.Len(t, subjects, 2)

	result, err = dispatchUserPresenceEmailsAt(context.Background(), now.Add(time.Minute), sender)
	require.NoError(t, err)
	assert.Zero(t, result.OfflineUserCount)
	require.Len(t, subjects, 2)

	presence, err := model.GetUserPresence(users["common"].Id)
	require.NoError(t, err)
	assert.False(t, presence.IsOnline)
	assert.Equal(t, UserPresenceActivitySourceDashboard, presence.LastSource)
}

func TestUserPresenceTestEmailUsesRealSelectedUserForBothEvents(t *testing.T) {
	_, users := setupEmailSettingsTest(t)
	var emailCount int
	result, err := sendUserPresenceTestEmails(
		[]int{users["admin"].Id},
		[]int{users["common"].Id},
		[]string{UserPresenceEventOnline, UserPresenceEventOffline},
		5,
		func(_, receiver, content string) error {
			emailCount++
			assert.Equal(t, users["admin"].Email, receiver)
			assert.Contains(t, content, users["common"].Username)
			return nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, users["common"].Id, result.MonitoredUserId)
	assert.Equal(t, 1, result.RecipientCount)
	assert.Equal(t, 2, result.EmailCount)
	assert.Equal(t, 2, emailCount)
}

func TestReEnablingUserPresenceDoesNotThrottleFirstActivity(t *testing.T) {
	db, users := setupEmailSettingsTest(t)
	now := time.Now().Unix()
	require.NoError(t, db.Create(&model.UserPresence{
		UserId:         users["common"].Id,
		IsOnline:       true,
		LastActivityAt: now,
		LastSource:     UserPresenceActivitySourceDashboard,
		LastChangedAt:  now,
	}).Error)
	userPresenceActivityMu.Lock()
	userPresenceActivityWrites[users["common"].Id] = now
	userPresenceActivityMu.Unlock()
	t.Cleanup(func() {
		clearUserPresenceActivityWriteThrottle([]int{users["common"].Id})
	})

	_, err := UpdateEmailSettingsConfig(EmailSettingsConfig{
		LowBalanceEmailThreshold:          1,
		AccountQuotaEmailThreshold:        1,
		UserPresenceEmailEnabled:          true,
		UserPresenceEmailEvents:           []string{UserPresenceEventOffline},
		UserPresenceEmailMonitoredUserIDs: []int{users["common"].Id},
		UserPresenceEmailRecipientIDs:     []int{users["admin"].Id},
		UserPresenceOfflineMinutes:        5,
	})
	require.NoError(t, err)

	presence, err := model.GetUserPresence(users["common"].Id)
	require.NoError(t, err)
	require.False(t, presence.IsOnline)

	RecordUserPresenceActivity(
		users["common"].Id,
		UserPresenceActivitySourceAPI,
		"203.0.113.20",
		"Codex CLI",
	)
	presence, err = model.GetUserPresence(users["common"].Id)
	require.NoError(t, err)
	assert.True(t, presence.IsOnline)
	assert.Equal(t, UserPresenceActivitySourceAPI, presence.LastSource)
}
