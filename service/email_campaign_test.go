package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupEmailCampaignTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.SubscriptionPlan{},
		&model.UserSubscription{},
		&model.EmailCampaign{},
		&model.EmailDelivery{},
	))
	oldDB := model.DB
	model.DB = db
	t.Cleanup(func() {
		model.DB = oldDB
		_ = sqlDB.Close()
	})
	return db
}

func createEmailCampaignTestUser(t *testing.T, db *gorm.DB, username, email string) model.User {
	t.Helper()
	user := model.User{
		Username: username,
		Password: "password",
		Email:    email,
		Status:   common.UserStatusEnabled,
		Role:     common.RoleCommonUser,
	}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func TestConditionalEmailCampaignDeduplicatesExpiryAndSendsAfterRenewal(t *testing.T) {
	db := setupEmailCampaignTestDB(t)
	now := common.GetTimestamp()
	user := createEmailCampaignTestUser(t, db, "subscriber", "subscriber@example.com")
	plan := model.SubscriptionPlan{Title: "Pro", PriceAmount: 1, Currency: "USD"}
	require.NoError(t, db.Create(&plan).Error)
	subscription := model.UserSubscription{
		UserId:    user.Id,
		PlanId:    plan.Id,
		StartTime: now - 3600,
		EndTime:   now + 2*24*60*60,
		Status:    "active",
	}
	require.NoError(t, db.Create(&subscription).Error)

	campaign := model.EmailCampaign{
		Name:        "Expiry reminder",
		Subject:     "{{subscription_name}} expires soon",
		Content:     "Hello {{username}}, {{days_remaining}} days remain.",
		Mode:        model.EmailCampaignModeConditional,
		TargetType:  model.EmailCampaignTargetActiveSubscribers,
		TriggerType: model.EmailCampaignTriggerSubscriptionExpiring,
		TriggerDays: 3,
		Status:      model.EmailCampaignStatusActive,
		NextRunAt:   now,
		CreatedBy:   1,
	}
	require.NoError(t, campaign.SetTargetUserIDList(nil))
	require.NoError(t, model.CreateEmailCampaign(&campaign))

	var subjects []string
	sender := func(subject, receiver, content string) error {
		subjects = append(subjects, subject)
		assert.Equal(t, user.Email, receiver)
		assert.Contains(t, content, "Hello subscriber")
		return nil
	}
	result, err := DispatchDueEmailCampaigns(context.Background(), sender)
	require.NoError(t, err)
	assert.Equal(t, 1, result.CampaignCount)
	assert.Equal(t, []string{"Pro expires soon"}, subjects)

	require.NoError(t, db.Model(&model.EmailCampaign{}).Where("id = ?", campaign.Id).Updates(map[string]any{
		"status":      model.EmailCampaignStatusActive,
		"next_run_at": common.GetTimestamp(),
	}).Error)
	_, err = DispatchDueEmailCampaigns(context.Background(), sender)
	require.NoError(t, err)
	assert.Len(t, subjects, 1, "the same subscription expiry must not send twice")

	newEndTime := common.GetTimestamp() + 24*60*60
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("id = ?", subscription.Id).Update("end_time", newEndTime).Error)
	require.NoError(t, db.Model(&model.EmailCampaign{}).Where("id = ?", campaign.Id).Updates(map[string]any{
		"status":      model.EmailCampaignStatusActive,
		"next_run_at": common.GetTimestamp(),
	}).Error)
	_, err = DispatchDueEmailCampaigns(context.Background(), sender)
	require.NoError(t, err)
	assert.Len(t, subjects, 2, "a renewed expiry batch must be eligible for a new reminder")

	var deliveries int64
	require.NoError(t, db.Model(&model.EmailDelivery{}).Where("campaign_id = ?", campaign.Id).Count(&deliveries).Error)
	assert.Equal(t, int64(2), deliveries)
}

func TestSubscriptionExpiryReminderUsesSevenThreeOneDayWindowsAndDeduplicates(t *testing.T) {
	db := setupEmailCampaignTestDB(t)
	now := common.GetTimestamp()
	user := createEmailCampaignTestUser(t, db, "subscriber", "subscriber@example.com")
	plan := model.SubscriptionPlan{Title: "Pro", PriceAmount: 1, Currency: "USD"}
	require.NoError(t, db.Create(&plan).Error)
	for _, remainingHours := range []int64{156, 60, 12, 96} {
		subscription := model.UserSubscription{
			UserId:    user.Id,
			PlanId:    plan.Id,
			StartTime: now - 3600,
			EndTime:   now + remainingHours*60*60,
			Status:    "active",
		}
		require.NoError(t, db.Create(&subscription).Error)
	}

	common.OptionMapRWMutex.Lock()
	oldOptions := common.OptionMap
	common.OptionMap = map[string]string{
		common.SubscriptionExpiryReminderEnabledOptionKey: "true",
	}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = oldOptions
		common.OptionMapRWMutex.Unlock()
	})

	var subjects []string
	sender := func(subject, receiver, _ string) error {
		subjects = append(subjects, subject)
		assert.Equal(t, user.Email, receiver)
		return nil
	}
	result, err := DispatchSubscriptionExpiryReminders(context.Background(), sender)
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.SentCount)
	assert.Len(t, subjects, 3)
	assert.Contains(t, subjects[0]+subjects[1]+subjects[2], "7")
	assert.Contains(t, subjects[0]+subjects[1]+subjects[2], "3")
	assert.Contains(t, subjects[0]+subjects[1]+subjects[2], "1")

	result, err = DispatchSubscriptionExpiryReminders(context.Background(), sender)
	require.NoError(t, err)
	assert.Zero(t, result.SentCount)
	assert.Len(t, subjects, 3)

	var deliveries int64
	require.NoError(t, db.Model(&model.EmailDelivery{}).Where("campaign_id = ?", systemSubscriptionExpiryCampaignID).Count(&deliveries).Error)
	assert.Equal(t, int64(3), deliveries)
}

func TestOneShotEmailCampaignRetriesOnlyFailedDeliveries(t *testing.T) {
	db := setupEmailCampaignTestDB(t)
	user := createEmailCampaignTestUser(t, db, "retry-user", "retry@example.com")
	now := common.GetTimestamp()
	campaign := model.EmailCampaign{
		Name:       "Notice",
		Subject:    "Notice",
		Content:    "Hello",
		Mode:       model.EmailCampaignModeImmediate,
		TargetType: model.EmailCampaignTargetAllUsers,
		Status:     model.EmailCampaignStatusScheduled,
		NextRunAt:  now,
		CreatedBy:  1,
	}
	require.NoError(t, campaign.SetTargetUserIDList(nil))
	require.NoError(t, model.CreateEmailCampaign(&campaign))

	_, err := DispatchDueEmailCampaigns(context.Background(), func(_, _, _ string) error {
		return errors.New("temporary SMTP failure")
	})
	require.NoError(t, err)
	stored, err := model.GetEmailCampaignById(campaign.Id)
	require.NoError(t, err)
	assert.Equal(t, model.EmailCampaignStatusPartialFailed, stored.Status)
	assert.Equal(t, int64(1), stored.FailedCount)

	require.NoError(t, model.RetryFailedEmailCampaign(campaign.Id, common.GetTimestamp()))
	var recipients []string
	_, err = DispatchDueEmailCampaigns(context.Background(), func(_, receiver, _ string) error {
		recipients = append(recipients, receiver)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{user.Email}, recipients)
	stored, err = model.GetEmailCampaignById(campaign.Id)
	require.NoError(t, err)
	assert.Equal(t, model.EmailCampaignStatusCompleted, stored.Status)
	assert.Equal(t, int64(1), stored.SuccessCount)
	assert.Equal(t, int64(0), stored.FailedCount)

	deliveries, total, err := model.ListEmailDeliveries(campaign.Id, 0, 10, "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, deliveries, 1)
	assert.Equal(t, 2, deliveries[0].AttemptCount)
	assert.WithinDuration(t, time.Now(), time.Unix(deliveries[0].SentAt, 0), 2*time.Second)
}

func TestSearchEmailCampaignUserOptionsUsesFuzzyIdUsernameAndEmail(t *testing.T) {
	db := setupEmailCampaignTestDB(t)
	users := []model.User{
		{Id: 1201, Username: "alpha-user", Email: "alpha@example.com", Password: "password", Status: common.UserStatusEnabled, Role: common.RoleCommonUser, AffCode: "campaign-1201"},
		{Id: 3302, Username: "billing", Email: "sales+team@example.com", Password: "password", Status: common.UserStatusEnabled, Role: common.RoleCommonUser, AffCode: "campaign-3302"},
		{Id: 4403, Username: "disabled-alpha", Email: "disabled@example.com", Password: "password", Status: common.UserStatusDisabled, Role: common.RoleCommonUser, AffCode: "campaign-4403"},
		{Id: 5504, Username: "no-email", Email: "", Password: "password", Status: common.UserStatusEnabled, Role: common.RoleCommonUser, AffCode: "campaign-5504"},
	}
	require.NoError(t, db.Create(&users).Error)

	byID, total, err := model.SearchEmailCampaignUserOptions("201", 0, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, byID, 1)
	assert.Equal(t, 1201, byID[0].Id)

	byUsername, total, err := model.SearchEmailCampaignUserOptions("ALPHA", 0, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, byUsername, 1)
	assert.Equal(t, 1201, byUsername[0].Id)

	byEmail, total, err := model.SearchEmailCampaignUserOptions("+team@", 0, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, byEmail, 1)
	assert.Equal(t, 3302, byEmail[0].Id)

	secondPage, total, err := model.SearchEmailCampaignUserOptions("", 1, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, secondPage, 1)
	assert.Equal(t, 1201, secondPage[0].Id)

	resolved, err := model.GetEmailCampaignUserOptionsByIds([]int{1201, 3302, 4403, 5504})
	require.NoError(t, err)
	require.Len(t, resolved, 2)
	assert.Equal(t, 3302, resolved[0].Id)
	assert.Equal(t, 1201, resolved[1].Id)
}
