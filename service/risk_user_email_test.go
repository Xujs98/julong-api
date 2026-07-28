package service

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupRiskUserEmailTest(t *testing.T) (*gorm.DB, map[string]model.User, time.Time) {
	t.Helper()
	db, users := setupEmailSettingsTest(t)
	oldGlobalRiskDetection := common.UserRiskDetectionEnabled
	common.UserRiskDetectionEnabled = false
	t.Cleanup(func() {
		common.UserRiskDetectionEnabled = oldGlobalRiskDetection
	})

	medium := users["common"]
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", medium.Id).Update("risk_detection_enabled", true).Error)
	medium.RiskDetectionEnabled = true
	users["medium"] = medium

	high := model.User{
		Username: "mail-risk-high", Password: "password", Email: "risk-high@example.com",
		Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "mail-risk-high",
		RiskDetectionEnabled: true,
	}
	require.NoError(t, db.Create(&high).Error)
	users["high"] = high

	now := time.Date(2026, time.July, 28, 12, 0, 0, 0, time.Local)
	require.NoError(t, db.Create(&model.Log{
		UserId: medium.Id, Username: medium.Username, CreatedAt: now.Unix(), Type: model.LogTypeError,
		Other: `{"error_code":"sensitive_words_detected"}`,
	}).Error)
	for range 4 {
		require.NoError(t, db.Create(&model.Log{
			UserId: high.Id, Username: high.Username, CreatedAt: now.Unix(), Type: model.LogTypeError,
			Other: `{"error_code":"sensitive_words_detected"}`,
		}).Error)
	}
	return db, users, now
}

func TestSendRiskUserTestEmailsUsesSelectedLevelsAndRealData(t *testing.T) {
	_, users, now := setupRiskUserEmailTest(t)
	contents := make([]string, 0, 2)
	sender := func(_, _ string, content string) error {
		contents = append(contents, content)
		return nil
	}

	result, err := sendRiskUserTestEmailsAt(
		context.Background(),
		now,
		[]int{users["root"].Id},
		[]string{model.UserRiskLevelMedium, model.UserRiskLevelHigh},
		sender,
	)
	require.NoError(t, err)
	assert.Equal(t, 1, result.RecipientCount)
	assert.Equal(t, 2, result.RiskUserCount)
	assert.Equal(t, []string{model.UserRiskLevelMedium, model.UserRiskLevelHigh}, result.Levels)
	require.Len(t, contents, 1)
	assert.Contains(t, contents[0], users["medium"].Username)
	assert.Contains(t, contents[0], users["high"].Username)

	contents = contents[:0]
	result, err = sendRiskUserTestEmailsAt(
		context.Background(),
		now,
		[]int{users["root"].Id},
		[]string{model.UserRiskLevelHigh},
		sender,
	)
	require.NoError(t, err)
	assert.Equal(t, 1, result.RiskUserCount)
	require.Len(t, contents, 1)
	assert.NotContains(t, contents[0], users["medium"].Username)
	assert.Contains(t, contents[0], users["high"].Username)
}

func TestDispatchRiskUserEmailsNotifiesOnEscalationAndAfterRecovery(t *testing.T) {
	db, users, _ := setupRiskUserEmailTest(t)
	require.NoError(t, db.Where("user_id = ?", users["high"].Id).Delete(&model.Log{}).Error)
	_, err := UpdateEmailSettingsConfig(EmailSettingsConfig{
		RiskUserEmailEnabled:      true,
		RiskUserEmailLevels:       []string{model.UserRiskLevelMedium, model.UserRiskLevelHigh},
		RiskUserEmailRecipientIDs: []int{users["root"].Id},
	})
	require.NoError(t, err)

	sent := 0
	sender := func(_, _, _ string) error {
		sent++
		return nil
	}

	result, err := DispatchRiskUserEmails(context.Background(), sender)
	require.NoError(t, err)
	assert.Equal(t, 1, result.RiskUserCount)
	assert.Equal(t, 1, sent)

	result, err = DispatchRiskUserEmails(context.Background(), sender)
	require.NoError(t, err)
	assert.Zero(t, result.RiskUserCount)
	assert.Equal(t, 1, sent)

	now := time.Now().Unix()
	for range 3 {
		require.NoError(t, db.Create(&model.Log{
			UserId: users["medium"].Id, Username: users["medium"].Username, CreatedAt: now,
			Type: model.LogTypeError, Other: `{"error_code":"sensitive_words_detected"}`,
		}).Error)
	}
	result, err = DispatchRiskUserEmails(context.Background(), sender)
	require.NoError(t, err)
	assert.Equal(t, []string{model.UserRiskLevelHigh}, result.Levels)
	assert.Equal(t, 2, sent)

	require.NoError(t, db.Where("user_id = ?", users["medium"].Id).Delete(&model.Log{}).Error)
	result, err = DispatchRiskUserEmails(context.Background(), sender)
	require.NoError(t, err)
	assert.Zero(t, result.RiskUserCount)
	assert.Equal(t, 2, sent)

	require.NoError(t, db.Create(&model.Log{
		UserId: users["medium"].Id, Username: users["medium"].Username, CreatedAt: time.Now().Unix(),
		Type: model.LogTypeError, Other: `{"error_code":"sensitive_words_detected"}`,
	}).Error)
	result, err = DispatchRiskUserEmails(context.Background(), sender)
	require.NoError(t, err)
	assert.Equal(t, []string{model.UserRiskLevelMedium}, result.Levels)
	assert.Equal(t, 3, sent)
}
