package service

import (
	"testing"

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
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.User{}))

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
	oldDatabaseType := common.MainDatabaseType()
	common.OptionMapRWMutex.Lock()
	oldOptions := common.OptionMap
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()
	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		model.DB = oldDB
		common.SetMainDatabaseType(oldDatabaseType)
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
	}

	updated, err := UpdateEmailSettingsConfig(input)
	require.NoError(t, err)
	assert.Equal(t, []int{users["admin"].Id, users["root"].Id}, updated.AccountQuotaEmailRecipientUserIDs)
	assert.Equal(t, []int{users["root"].Id}, updated.ChannelAnomalyEmailRecipientIDs)
	assert.Equal(t, input.LowBalanceEmailThreshold, updated.LowBalanceEmailThreshold)
	assert.Equal(t, input.AccountQuotaEmailThreshold, updated.AccountQuotaEmailThreshold)

	var optionCount int64
	require.NoError(t, db.Model(&model.Option{}).Count(&optionCount).Error)
	assert.EqualValues(t, 9, optionCount)
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
