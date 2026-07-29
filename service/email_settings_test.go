package service

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupEmailSettingsTest(t *testing.T) (*gorm.DB, map[string]model.User) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.User{}, &model.QuotaData{}, &model.Log{}, &model.UserPresence{}))

	users := map[string]model.User{
		"common": {
			Username: "mail-common", Password: "password", Email: "common@example.com",
			Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "mail-common",
		},
		"admin": {
			Username: "mail-admin", Password: "password", Email: "admin@example.com",
			Role: common.RoleAdminUser, Status: common.UserStatusEnabled, AffCode: "mail-admin",
		},
		"root": {
			Username: "mail-root", Password: "password", Email: "root@example.com",
			Role: common.RoleRootUser, Status: common.UserStatusEnabled, AffCode: "mail-root",
		},
		"disabled_admin": {
			Username: "mail-disabled", Password: "password", Email: "disabled@example.com",
			Role: common.RoleAdminUser, Status: common.UserStatusDisabled, AffCode: "mail-disabled",
		},
	}
	for key, user := range users {
		require.NoError(t, db.Create(&user).Error)
		users[key] = user
	}

	oldDB := model.DB
	oldLogDB := model.LOG_DB
	oldDatabaseType := common.MainDatabaseType()
	oldLogDatabaseType := common.LogDatabaseType()
	common.OptionMapRWMutex.Lock()
	oldOptions := common.OptionMap
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()
	model.DB = db
	model.LOG_DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.SetLogDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		common.SetMainDatabaseType(oldDatabaseType)
		common.SetLogDatabaseType(oldLogDatabaseType)
		common.OptionMapRWMutex.Lock()
		common.OptionMap = oldOptions
		common.OptionMapRWMutex.Unlock()
	})
	return db, users
}

func TestUpdateEmailSettingsConfigPersistsCompleteConfiguration(t *testing.T) {
	db, users := setupEmailSettingsTest(t)
	input := EmailSettingsConfig{
		SubscriptionExpiryReminderEnabled: true,
		LowBalanceEmailEnabled:            true,
		LowBalanceEmailThreshold:          120000,
		LowBalanceEmailRechargeURL:        "https://example.com/wallet",
		AccountQuotaEmailEnabled:          true,
		AccountQuotaEmailThreshold:        12.5,
		AccountQuotaEmailRecipientUserIDs: []int{users["root"].Id, users["admin"].Id, users["admin"].Id},
		ChannelAnomalyEmailEnabled:        true,
		ChannelAnomalyEmailRecipientIDs:   []int{users["root"].Id},
		DashboardReportEmailEnabled:       true,
		DashboardReportEmailFrequency:     DashboardReportFrequencyWeekly,
		DashboardReportEmailSendTime:      "09:30",
		DashboardReportEmailWeekday:       5,
		DashboardReportEmailMonthDay:      15,
		DashboardReportEmailRecipientIDs:  []int{users["admin"].Id, users["root"].Id},
		DashboardReportEmailSchedules: []DashboardReportEmailSchedule{
			{Frequency: DashboardReportFrequencyWeekly, SendTimes: []string{"18:00", "09:30"}, Weekday: 5, MonthDay: 15},
			{Frequency: DashboardReportFrequencyDaily, SendTimes: []string{"23:00", "12:00"}, Weekday: 1, MonthDay: 1},
		},
		RiskUserEmailEnabled:              true,
		RiskUserEmailLevels:               []string{model.UserRiskLevelHigh, model.UserRiskLevelMedium, model.UserRiskLevelHigh},
		RiskUserEmailRecipientIDs:         []int{users["admin"].Id, users["root"].Id},
		UserPresenceEmailEnabled:          true,
		UserPresenceEmailEvents:           []string{UserPresenceEventOffline, UserPresenceEventOnline, UserPresenceEventOnline},
		UserPresenceEmailMonitoredUserIDs: []int{users["common"].Id, users["admin"].Id},
		UserPresenceEmailRecipientIDs:     []int{users["admin"].Id, users["root"].Id},
		UserPresenceOfflineMinutes:        5,
	}

	updated, err := UpdateEmailSettingsConfig(input)
	require.NoError(t, err)
	expectedRecipients := normalizeEmailRecipientIDs([]int{users["admin"].Id, users["root"].Id})
	assert.Equal(t, expectedRecipients, updated.AccountQuotaEmailRecipientUserIDs)
	assert.Equal(t, []int{users["root"].Id}, updated.ChannelAnomalyEmailRecipientIDs)
	assert.Equal(t, expectedRecipients, updated.DashboardReportEmailRecipientIDs)
	assert.Equal(t, []string{model.UserRiskLevelMedium, model.UserRiskLevelHigh}, updated.RiskUserEmailLevels)
	assert.Equal(t, expectedRecipients, updated.RiskUserEmailRecipientIDs)
	assert.Equal(t, []string{UserPresenceEventOnline, UserPresenceEventOffline}, updated.UserPresenceEmailEvents)
	assert.Equal(t, normalizeEmailRecipientIDs([]int{users["common"].Id, users["admin"].Id}), updated.UserPresenceEmailMonitoredUserIDs)
	assert.Equal(t, expectedRecipients, updated.UserPresenceEmailRecipientIDs)
	assert.Equal(t, 5, updated.UserPresenceOfflineMinutes)
	assert.Equal(t, DashboardReportFrequencyWeekly, updated.DashboardReportEmailFrequency)
	assert.Equal(t, "09:30", updated.DashboardReportEmailSendTime)
	require.Len(t, updated.DashboardReportEmailSchedules, 2)
	assert.Equal(t, []string{"09:30", "18:00"}, updated.DashboardReportEmailSchedules[0].SendTimes)
	assert.Equal(t, []string{"12:00", "23:00"}, updated.DashboardReportEmailSchedules[1].SendTimes)
	assert.Equal(t, input.LowBalanceEmailThreshold, updated.LowBalanceEmailThreshold)
	assert.Equal(t, input.AccountQuotaEmailThreshold, updated.AccountQuotaEmailThreshold)

	var optionCount int64
	require.NoError(t, db.Model(&model.Option{}).Count(&optionCount).Error)
	assert.EqualValues(t, 24, optionCount)
	assert.Equal(t, "true", common.OptionMap[common.LowBalanceEmailEnabledOptionKey])
	expectedChannelRecipients, err := common.Marshal([]int{users["root"].Id})
	require.NoError(t, err)
	assert.Equal(t, string(expectedChannelRecipients), common.OptionMap[common.ChannelAnomalyEmailRecipientUserIDsOptionKey])
}

func TestUpdateEmailSettingsConfigRejectsNonOperationalRecipientWithoutPartialSave(t *testing.T) {
	db, users := setupEmailSettingsTest(t)
	_, err := UpdateEmailSettingsConfig(EmailSettingsConfig{
		LowBalanceEmailThreshold:          1,
		AccountQuotaEmailThreshold:        1,
		AccountQuotaEmailRecipientUserIDs: []int{users["common"].Id},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active administrators")

	var optionCount int64
	require.NoError(t, db.Model(&model.Option{}).Count(&optionCount).Error)
	assert.Zero(t, optionCount)
}

func TestGetEmailSettingsConfigFallsBackToRootRecipients(t *testing.T) {
	_, users := setupEmailSettingsTest(t)
	config, err := GetEmailSettingsConfig()
	require.NoError(t, err)
	assert.Equal(t, []int{users["root"].Id}, config.AccountQuotaEmailRecipientUserIDs)
	assert.Equal(t, []int{users["root"].Id}, config.ChannelAnomalyEmailRecipientIDs)
	assert.Equal(t, []int{users["root"].Id}, config.DashboardReportEmailRecipientIDs)
	assert.Equal(t, []int{users["root"].Id}, config.RiskUserEmailRecipientIDs)
	assert.Equal(t, []string{model.UserRiskLevelMedium, model.UserRiskLevelHigh}, config.RiskUserEmailLevels)
	require.Len(t, config.DashboardReportEmailSchedules, 1)
	assert.Equal(t, DashboardReportFrequencyDaily, config.DashboardReportEmailSchedules[0].Frequency)
	assert.Equal(t, []string{"08:00"}, config.DashboardReportEmailSchedules[0].SendTimes)
}

func TestGetEmailSettingsConfigReturnsEmptyRecipientArrays(t *testing.T) {
	db, _ := setupEmailSettingsTest(t)
	require.NoError(t, db.Model(&model.User{}).Where("role >= ?", common.RoleAdminUser).Update("email", "").Error)

	config, err := GetEmailSettingsConfig()
	require.NoError(t, err)
	assert.NotNil(t, config.AccountQuotaEmailRecipientUserIDs)
	assert.NotNil(t, config.ChannelAnomalyEmailRecipientIDs)
	assert.NotNil(t, config.DashboardReportEmailRecipientIDs)
	assert.NotNil(t, config.RiskUserEmailRecipientIDs)
	assert.NotNil(t, config.RiskUserEmailLevels)
	assert.Empty(t, config.AccountQuotaEmailRecipientUserIDs)
	assert.Empty(t, config.ChannelAnomalyEmailRecipientIDs)
	assert.Empty(t, config.DashboardReportEmailRecipientIDs)
	assert.Empty(t, config.RiskUserEmailRecipientIDs)

	payload, err := common.Marshal(config)
	require.NoError(t, err)
	assert.Contains(t, string(payload), `"account_quota_email_recipient_user_ids":[]`)
	assert.Contains(t, string(payload), `"channel_anomaly_email_recipient_user_ids":[]`)
	assert.Contains(t, string(payload), `"dashboard_report_email_recipient_user_ids":[]`)
	assert.Contains(t, string(payload), `"risk_user_email_recipient_user_ids":[]`)
}

func TestOperationalEmailRecipientSearchOnlyReturnsEnabledAdministrators(t *testing.T) {
	_, users := setupEmailSettingsTest(t)
	options, total, err := model.SearchOperationalEmailRecipientOptions("mail-", 0, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)

	ids := make([]int, 0, len(options))
	for _, option := range options {
		ids = append(ids, option.Id)
	}
	assert.ElementsMatch(t, []int{users["admin"].Id, users["root"].Id}, ids)
}

func TestSendChannelAnomalyTestEmailsUsesSelectedOperationalRecipients(t *testing.T) {
	_, users := setupEmailSettingsTest(t)
	receivers := make([]string, 0, 2)
	sent, err := sendChannelAnomalyTestEmails(
		[]int{users["root"].Id, users["admin"].Id},
		func(subject, receiver, content string) error {
			receivers = append(receivers, receiver)
			assert.Contains(t, subject, "渠道异常通知测试")
			assert.Contains(t, content, "用于验证渠道异常通知配置")
			return nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, 2, sent)
	assert.ElementsMatch(t, []string{users["admin"].Email, users["root"].Email}, receivers)
}

func TestSendChannelAnomalyTestEmailsFallsBackToRootAndRejectsCommonUser(t *testing.T) {
	_, users := setupEmailSettingsTest(t)
	receivers := make([]string, 0, 1)
	sent, err := sendChannelAnomalyTestEmails(nil, func(_, receiver, _ string) error {
		receivers = append(receivers, receiver)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, sent)
	assert.Equal(t, []string{users["root"].Email}, receivers)

	_, err = sendChannelAnomalyTestEmails([]int{users["common"].Id}, func(_, _, _ string) error {
		return nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active administrators")
}

func TestSendDashboardReportTestEmailsUsesCurrentDashboardData(t *testing.T) {
	db, users := setupEmailSettingsTest(t)
	now := time.Date(2026, time.July, 27, 10, 30, 0, 0, time.Local)
	require.NoError(t, db.Create(&model.QuotaData{
		UserID: users["admin"].Id, Username: users["admin"].Username, ModelName: "codex-v1",
		CreatedAt: now.Add(-time.Hour).Unix(), UseGroup: "default", ChannelID: 9,
		TokenUsed: 250000, Count: 12, Quota: 500000,
	}).Error)
	require.NoError(t, db.Create(&model.QuotaData{
		UserID: users["root"].Id, Username: users["root"].Username, ModelName: "gpt-4.1",
		CreatedAt: now.Add(-2 * time.Hour).Unix(), UseGroup: "vip", ChannelID: 10,
		TokenUsed: 750000, Count: 8, Quota: 1000000,
	}).Error)

	var subject, content string
	result, err := sendDashboardReportTestEmailsAt(
		now,
		[]int{users["root"].Id},
		func(sentSubject, _ string, sentContent string) error {
			subject, content = sentSubject, sentContent
			return nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, result.RecipientCount)
	assert.Contains(t, subject, "实时数据报表")
	assert.Contains(t, content, "$3.00")
	assert.Contains(t, content, "1,000,000")
	assert.Contains(t, content, "codex-v1")
	assert.Contains(t, content, "gpt-4.1")
	assert.Contains(t, content, "用户统计")
	assert.Contains(t, content, "mail-admin")
	assert.Contains(t, content, "分组数据分析")
	assert.Contains(t, content, "default")
	assert.Contains(t, content, "vip")
}

func TestDashboardReportScheduleUsesCompletedCalendarPeriods(t *testing.T) {
	schedule := DashboardReportEmailSchedule{
		Frequency: DashboardReportFrequencyDaily,
		SendTimes: []string{"08:00"},
		Weekday:   1,
		MonthDay:  1,
	}
	now := time.Date(2026, time.July, 27, 8, 30, 0, 0, time.Local)

	daily, due := dashboardReportPeriodForSchedule(schedule, now)
	require.True(t, due)
	assert.Equal(t, "2026-07-26", daily.Start.Format("2006-01-02"))
	assert.Equal(t, "2026-07-27", daily.End.Format("2006-01-02"))

	schedule.Frequency = DashboardReportFrequencyWeekly
	weekly, due := dashboardReportPeriodForSchedule(schedule, now)
	require.True(t, due)
	assert.Equal(t, "2026-07-20", weekly.Start.Format("2006-01-02"))
	assert.Equal(t, "2026-07-27", weekly.End.Format("2006-01-02"))

	schedule.Frequency = DashboardReportFrequencyMonthly
	schedule.MonthDay = 31
	monthEnd := time.Date(2026, time.February, 28, 8, 30, 0, 0, time.Local)
	monthly, due := dashboardReportPeriodForSchedule(schedule, monthEnd)
	require.True(t, due)
	assert.Equal(t, "2026-01-01", monthly.Start.Format("2006-01-02"))
	assert.Equal(t, "2026-02-01", monthly.End.Format("2006-01-02"))
}

func TestNextDashboardReportDispatchTracksEveryConditionAndSendTime(t *testing.T) {
	now := time.Date(2026, time.July, 27, 23, 30, 0, 0, time.Local)
	schedules := normalizeDashboardReportEmailSchedules([]DashboardReportEmailSchedule{
		{Frequency: DashboardReportFrequencyDaily, SendTimes: []string{"23:00", "12:00"}, Weekday: 1, MonthDay: 1},
		{Frequency: DashboardReportFrequencyWeekly, SendTimes: []string{"18:00"}, Weekday: 1, MonthDay: 1},
	})
	history := dashboardReportDispatchHistory{}

	first, due := nextDashboardReportDispatch(schedules, now, history)
	require.True(t, due)
	assert.Equal(t, "12:00", first.ScheduledTime)
	history[first.DispatchKey] = now.Unix()

	second, due := nextDashboardReportDispatch(schedules, now, history)
	require.True(t, due)
	assert.Equal(t, "23:00", second.ScheduledTime)
	history[second.DispatchKey] = now.Unix()

	third, due := nextDashboardReportDispatch(schedules, now, history)
	require.True(t, due)
	assert.Equal(t, "18:00", third.ScheduledTime)
	assert.Equal(t, "Weekly", third.Period.TypeEN)
	history[third.DispatchKey] = now.Unix()

	_, due = nextDashboardReportDispatch(schedules, now, history)
	assert.False(t, due)
}

func TestDispatchDashboardReportEmailsSendsEachDueTimeOnce(t *testing.T) {
	_, users := setupEmailSettingsTest(t)
	_, err := UpdateEmailSettingsConfig(EmailSettingsConfig{
		DashboardReportEmailEnabled:      true,
		DashboardReportEmailRecipientIDs: []int{users["root"].Id},
		DashboardReportEmailSchedules: []DashboardReportEmailSchedule{
			{Frequency: DashboardReportFrequencyDaily, SendTimes: []string{"12:00", "23:00"}, Weekday: 1, MonthDay: 1},
			{Frequency: DashboardReportFrequencyWeekly, SendTimes: []string{"18:00"}, Weekday: 1, MonthDay: 1},
		},
	})
	require.NoError(t, err)
	now := time.Date(2026, time.July, 27, 23, 30, 0, 0, time.Local)
	sent := 0
	sender := func(_, _, _ string) error {
		sent++
		return nil
	}

	first, err := dispatchDashboardReportEmailsAt(context.Background(), now, sender)
	require.NoError(t, err)
	assert.Equal(t, "12:00", first.ScheduledTime)
	second, err := dispatchDashboardReportEmailsAt(context.Background(), now, sender)
	require.NoError(t, err)
	assert.Equal(t, "23:00", second.ScheduledTime)
	third, err := dispatchDashboardReportEmailsAt(context.Background(), now, sender)
	require.NoError(t, err)
	assert.Equal(t, "18:00", third.ScheduledTime)
	fourth, err := dispatchDashboardReportEmailsAt(context.Background(), now, sender)
	require.NoError(t, err)
	assert.Zero(t, fourth.RecipientCount)
	assert.Equal(t, 3, sent)
}
