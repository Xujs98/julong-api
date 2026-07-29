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

func setupEmailTemplateTest(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Option{}))
	oldDB := model.DB
	oldSystemName := common.SystemName
	common.OptionMapRWMutex.Lock()
	oldOptions := common.OptionMap
	common.OptionMap = map[string]string{common.EmailTemplatesOptionKey: "{}"}
	common.OptionMapRWMutex.Unlock()
	model.DB = db
	common.SystemName = "Test API"
	t.Cleanup(func() {
		model.DB = oldDB
		common.SystemName = oldSystemName
		common.OptionMapRWMutex.Lock()
		common.OptionMap = oldOptions
		common.OptionMapRWMutex.Unlock()
	})
	return db
}

func TestEmailTemplateUpdatePersistsAndEscapesRuntimeValues(t *testing.T) {
	db := setupEmailTemplateTest(t)
	template, err := UpdateEmailTemplate(
		EmailTemplateEventVerifyCode,
		"[{{system_name}}] {{verification_code}}",
		"<p>{{display_name}}</p><strong>{{verification_code}}</strong>",
	)
	require.NoError(t, err)
	assert.True(t, template.IsCustom)

	var option model.Option
	require.NoError(t, db.Where("key = ?", common.EmailTemplatesOptionKey).First(&option).Error)
	assert.Contains(t, option.Value, EmailTemplateEventVerifyCode)

	rendered, err := RenderEmailTemplate(EmailTemplateEventVerifyCode, map[string]string{
		"display_name":      "<script>alert(1)</script>",
		"verification_code": "123456",
	})
	require.NoError(t, err)
	assert.Equal(t, "[Test API] 123456", rendered.Subject)
	assert.NotContains(t, rendered.Content, "<script>")
	assert.Contains(t, rendered.Content, "&lt;script&gt;alert(1)&lt;/script&gt;")

	template, err = ResetEmailTemplate(EmailTemplateEventVerifyCode)
	require.NoError(t, err)
	assert.False(t, template.IsCustom)
}

func TestEmailTemplateRejectsUnavailablePlaceholder(t *testing.T) {
	setupEmailTemplateTest(t)
	_, err := UpdateEmailTemplate(
		EmailTemplateEventPasswordReset,
		"Reset {{verification_code}}",
		"<p>{{reset_url}}</p>",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "{{verification_code}}")
}

func TestEmailTemplateLocalesPersistIndependently(t *testing.T) {
	setupEmailTemplateTest(t)
	english, err := UpdateEmailTemplateForLocale(
		EmailTemplateEventLowBalance,
		EmailTemplateLocaleEnglish,
		"Low {{balance_type}}",
		"<p>{{current_balance}}</p>",
	)
	require.NoError(t, err)
	assert.Equal(t, EmailTemplateLocaleEnglish, english.Locale)
	assert.True(t, english.IsCustom)

	chinese, err := GetEmailTemplateForLocale(EmailTemplateEventLowBalance, EmailTemplateLocaleChinese)
	require.NoError(t, err)
	assert.False(t, chinese.IsCustom)
	assert.NotEqual(t, english.Subject, chinese.Subject)

	stored := loadCustomEmailTemplates()
	assert.Contains(t, stored, EmailTemplateEventLowBalance+emailTemplateLocalizedStorageKeySeparator+EmailTemplateLocaleEnglish)
	assert.NotContains(t, stored, EmailTemplateEventLowBalance)
}

func TestEmailTemplateCatalogContainsEveryEventAndLocale(t *testing.T) {
	setupEmailTemplateTest(t)
	templates := ListEmailTemplates()
	require.Len(t, templates, len(emailTemplateOrder)*len(emailTemplateLocales))

	seen := make(map[string]bool, len(templates))
	for _, template := range templates {
		seen[template.Event+":"+template.Locale] = true
	}
	for _, event := range emailTemplateOrder {
		for _, locale := range emailTemplateLocales {
			assert.True(t, seen[event+":"+locale], "missing %s/%s", event, locale)
		}
	}
}

func TestDefaultEmailTemplatesUseColoredHeaderCardStyle(t *testing.T) {
	setupEmailTemplateTest(t)
	for _, event := range emailTemplateOrder {
		for _, locale := range emailTemplateLocales {
			template, err := GetEmailTemplateForLocale(event, locale)
			require.NoError(t, err)
			assert.Contains(t, template.Content, "padding:38px 40px;background:")
			assert.Contains(t, template.Content, "color:#ffffff;")
		}
	}
}

func TestDashboardReportTemplateExposesReportMetrics(t *testing.T) {
	setupEmailTemplateTest(t)
	template, err := GetEmailTemplateForLocale(EmailTemplateEventDashboardReport, EmailTemplateLocaleChinese)
	require.NoError(t, err)
	assert.Contains(t, template.Placeholders, "report_period")
	assert.Contains(t, template.Placeholders, "total_consumption")
	assert.Contains(t, template.Placeholders, "top_models")
	assert.Contains(t, template.Placeholders, "top_users")
	assert.Contains(t, template.Placeholders, "group_analysis")
	assert.Contains(t, template.Content, "{{active_users}}")
	assert.Contains(t, template.Content, "{{top_users}}")
	assert.Contains(t, template.Content, "{{group_analysis}}")
}

func TestRiskUserTemplateExposesRiskSummary(t *testing.T) {
	setupEmailTemplateTest(t)
	template, err := GetEmailTemplateForLocale(EmailTemplateEventRiskUserDetected, EmailTemplateLocaleChinese)
	require.NoError(t, err)
	assert.Contains(t, template.Placeholders, "risk_user_count")
	assert.Contains(t, template.Placeholders, "risk_levels")
	assert.Contains(t, template.Placeholders, "risk_users")
	assert.Contains(t, template.Content, "{{window_days}}")
	assert.Contains(t, template.Content, "{{detected_at}}")
}

func TestUserPresenceTemplateExposesActivityDetails(t *testing.T) {
	setupEmailTemplateTest(t)
	template, err := GetEmailTemplateForLocale(EmailTemplateEventUserPresenceChanged, EmailTemplateLocaleChinese)
	require.NoError(t, err)
	assert.Contains(t, template.Placeholders, "monitored_user_id")
	assert.Contains(t, template.Placeholders, "presence_status")
	assert.Contains(t, template.Placeholders, "activity_source")
	assert.Contains(t, template.Content, "{{inactivity_minutes}}")
	assert.Contains(t, template.Content, "{{user_agent}}")
}

func TestShouldSendAccountQuotaEmailOnlyOnFirstLowBalanceOrDownwardCrossing(t *testing.T) {
	tests := []struct {
		name              string
		previousBalance   float64
		previousUpdatedAt int64
		currentBalance    float64
		threshold         float64
		expected          bool
	}{
		{name: "first balance check already low", previousUpdatedAt: 0, currentBalance: 2, threshold: 5, expected: true},
		{name: "crosses downward", previousBalance: 8, previousUpdatedAt: 1, currentBalance: 4, threshold: 5, expected: true},
		{name: "remains low", previousBalance: 4, previousUpdatedAt: 1, currentBalance: 3, threshold: 5, expected: false},
		{name: "remains above", previousBalance: 8, previousUpdatedAt: 1, currentBalance: 7, threshold: 5, expected: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, shouldSendAccountQuotaEmail(test.previousBalance, test.previousUpdatedAt, test.currentBalance, test.threshold))
		})
	}
}
