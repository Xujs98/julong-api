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
