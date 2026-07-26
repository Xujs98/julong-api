package service

import (
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	EmailTemplateEventVerifyCode                 = "auth.verify_code"
	EmailTemplateEventPasswordReset              = "auth.password_reset"
	EmailTemplateEventSubscriptionExpiryReminder = "subscription.expiry_reminder"

	maxEmailTemplateSubjectLength = 255
	maxEmailTemplateContentLength = 200000
)

type EmailTemplateDefinition struct {
	Event        string   `json:"event"`
	Label        string   `json:"label"`
	Description  string   `json:"description"`
	Placeholders []string `json:"placeholders"`
}

type EmailTemplate struct {
	EmailTemplateDefinition
	Subject  string `json:"subject"`
	Content  string `json:"content"`
	IsCustom bool   `json:"is_custom"`
}

type EmailTemplatePreview struct {
	Subject string `json:"subject"`
	Content string `json:"content"`
}

type storedEmailTemplate struct {
	Subject string `json:"subject"`
	Content string `json:"content"`
}

var (
	emailTemplatePlaceholderPattern = regexp.MustCompile(`\{\{([a-z][a-z0-9_]*)\}\}`)
	emailTemplatesMu                sync.Mutex
	emailTemplateOrder              = []string{
		EmailTemplateEventVerifyCode,
		EmailTemplateEventPasswordReset,
		EmailTemplateEventSubscriptionExpiryReminder,
	}
	emailTemplateDefinitions = map[string]EmailTemplateDefinition{
		EmailTemplateEventVerifyCode: {
			Event:       EmailTemplateEventVerifyCode,
			Label:       "Email verification code",
			Description: "Sent when a user requests an email verification code.",
			Placeholders: []string{
				"system_name", "username", "display_name", "email", "verification_code", "expires_in_minutes",
			},
		},
		EmailTemplateEventPasswordReset: {
			Event:       EmailTemplateEventPasswordReset,
			Label:       "Password reset email",
			Description: "Sent when a registered user requests a password reset link.",
			Placeholders: []string{
				"system_name", "username", "display_name", "email", "reset_url", "expires_in_minutes",
			},
		},
		EmailTemplateEventSubscriptionExpiryReminder: {
			Event:       EmailTemplateEventSubscriptionExpiryReminder,
			Label:       "Subscription expiry reminder",
			Description: "Sent 7, 3, and 1 days before an active subscription expires.",
			Placeholders: []string{
				"system_name", "username", "display_name", "email", "subscription_name", "subscription_end_time", "days_remaining",
			},
		},
	}
	emailTemplateDefaults = map[string]storedEmailTemplate{
		EmailTemplateEventVerifyCode: {
			Subject: "[{{system_name}}] 邮箱验证码",
			Content: buildEmailTemplateCard("邮箱验证码", "#2563eb", `
<p>{{display_name}}，您好：</p>
<p>您的邮箱验证码是：</p>
<p style="margin:24px 0;font-size:32px;font-weight:700;letter-spacing:8px;text-align:center;color:#111827;">{{verification_code}}</p>
<p>验证码将在 <strong>{{expires_in_minutes}}</strong> 分钟后失效。</p>
<p style="color:#6b7280;">如果不是您本人操作，请忽略此邮件。</p>`),
		},
		EmailTemplateEventPasswordReset: {
			Subject: "[{{system_name}}] 密码重置请求",
			Content: buildEmailTemplateCard("密码重置", "#7c3aed", `
<p>{{display_name}}，您好：</p>
<p>我们收到了您的密码重置请求，请点击下方按钮设置新密码。</p>
<p style="margin:24px 0;text-align:center;"><a href="{{reset_url}}" style="display:inline-block;padding:10px 18px;border-radius:6px;background:#7c3aed;color:#ffffff;text-decoration:none;font-weight:600;">重置密码</a></p>
<p>此链接将在 <strong>{{expires_in_minutes}}</strong> 分钟后失效。</p>
<p style="color:#6b7280;word-break:break-all;">如果按钮无法点击，请复制以下链接到浏览器中打开：<br>{{reset_url}}</p>
<p style="color:#6b7280;">如果不是您本人操作，请忽略此邮件。</p>`),
		},
		EmailTemplateEventSubscriptionExpiryReminder: {
			Subject: "[{{system_name}}] 订阅将在 {{days_remaining}} 天后到期",
			Content: buildEmailTemplateCard("订阅到期提醒", "#ea580c", `
<p>{{display_name}}，您好：</p>
<p>您的 <strong>{{subscription_name}}</strong> 订阅将在 <strong>{{days_remaining}}</strong> 天后到期。</p>
<p>到期时间：<strong>{{subscription_end_time}}</strong></p>
<p style="color:#6b7280;">如需继续使用订阅权益，请及时续订。</p>`),
		},
	}
)

func buildEmailTemplateCard(title, accent, body string) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:24px;background:#f3f4f6;color:#1f2937;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;">
  <div style="max-width:620px;margin:0 auto;overflow:hidden;border:1px solid #e5e7eb;border-radius:8px;background:#ffffff;">
    <div style="height:4px;background:%s;"></div>
    <div style="padding:28px;line-height:1.7;">
      <h1 style="margin:0 0 20px;font-size:22px;line-height:1.3;color:#111827;">%s</h1>
      %s
    </div>
    <div style="padding:16px 28px;border-top:1px solid #e5e7eb;color:#9ca3af;font-size:12px;">{{system_name}}</div>
  </div>
</body>
</html>`, accent, title, strings.TrimSpace(body))
}

func ListEmailTemplates() []EmailTemplate {
	templates := make([]EmailTemplate, 0, len(emailTemplateOrder))
	for _, event := range emailTemplateOrder {
		template, _ := GetEmailTemplate(event)
		templates = append(templates, template)
	}
	return templates
}

func GetEmailTemplate(event string) (EmailTemplate, error) {
	definition, ok := emailTemplateDefinitions[strings.TrimSpace(event)]
	if !ok {
		return EmailTemplate{}, errors.New("unsupported email template event")
	}
	stored := emailTemplateDefaults[definition.Event]
	isCustom := false
	if custom, exists := loadCustomEmailTemplates()[definition.Event]; exists {
		if err := validateEmailTemplate(definition, custom.Subject, custom.Content); err == nil {
			stored = custom
			isCustom = true
		} else {
			common.SysError(fmt.Sprintf("invalid custom email template ignored: event=%s err=%v", definition.Event, err))
		}
	}
	definition.Placeholders = append([]string(nil), definition.Placeholders...)
	return EmailTemplate{
		EmailTemplateDefinition: definition,
		Subject:                 stored.Subject,
		Content:                 stored.Content,
		IsCustom:                isCustom,
	}, nil
}

func UpdateEmailTemplate(event, subject, content string) (EmailTemplate, error) {
	definition, ok := emailTemplateDefinitions[strings.TrimSpace(event)]
	if !ok {
		return EmailTemplate{}, errors.New("unsupported email template event")
	}
	subject = strings.TrimSpace(subject)
	if err := validateEmailTemplate(definition, subject, content); err != nil {
		return EmailTemplate{}, err
	}

	emailTemplatesMu.Lock()
	defer emailTemplatesMu.Unlock()
	custom := loadCustomEmailTemplates()
	custom[definition.Event] = storedEmailTemplate{Subject: subject, Content: content}
	data, err := common.Marshal(custom)
	if err != nil {
		return EmailTemplate{}, err
	}
	if err := model.UpdateOption(common.EmailTemplatesOptionKey, string(data)); err != nil {
		return EmailTemplate{}, err
	}
	return GetEmailTemplate(definition.Event)
}

func ResetEmailTemplate(event string) (EmailTemplate, error) {
	definition, ok := emailTemplateDefinitions[strings.TrimSpace(event)]
	if !ok {
		return EmailTemplate{}, errors.New("unsupported email template event")
	}

	emailTemplatesMu.Lock()
	defer emailTemplatesMu.Unlock()
	custom := loadCustomEmailTemplates()
	delete(custom, definition.Event)
	data, err := common.Marshal(custom)
	if err != nil {
		return EmailTemplate{}, err
	}
	if err := model.UpdateOption(common.EmailTemplatesOptionKey, string(data)); err != nil {
		return EmailTemplate{}, err
	}
	return GetEmailTemplate(definition.Event)
}

func PreviewEmailTemplate(event, subject, content string) (EmailTemplatePreview, error) {
	definition, ok := emailTemplateDefinitions[strings.TrimSpace(event)]
	if !ok {
		return EmailTemplatePreview{}, errors.New("unsupported email template event")
	}
	subject = strings.TrimSpace(subject)
	if err := validateEmailTemplate(definition, subject, content); err != nil {
		return EmailTemplatePreview{}, err
	}
	return renderStoredEmailTemplate(storedEmailTemplate{Subject: subject, Content: content}, emailTemplateSampleValues()), nil
}

func RenderEmailTemplate(event string, values map[string]string) (EmailTemplatePreview, error) {
	template, err := GetEmailTemplate(event)
	if err != nil {
		return EmailTemplatePreview{}, err
	}
	resolved := make(map[string]string, len(values)+1)
	for key, value := range values {
		resolved[key] = value
	}
	resolved["system_name"] = common.SystemName
	return renderStoredEmailTemplate(storedEmailTemplate{Subject: template.Subject, Content: template.Content}, resolved), nil
}

func validateEmailTemplate(definition EmailTemplateDefinition, subject, content string) error {
	if strings.TrimSpace(subject) == "" || strings.TrimSpace(content) == "" {
		return errors.New("email subject and content are required")
	}
	if len(subject) > maxEmailTemplateSubjectLength {
		return fmt.Errorf("email subject cannot exceed %d characters", maxEmailTemplateSubjectLength)
	}
	if strings.ContainsAny(subject, "\r\n") {
		return errors.New("email subject cannot contain line breaks")
	}
	if len(content) > maxEmailTemplateContentLength {
		return fmt.Errorf("email content cannot exceed %d bytes", maxEmailTemplateContentLength)
	}
	allowed := make(map[string]struct{}, len(definition.Placeholders))
	for _, placeholder := range definition.Placeholders {
		allowed[placeholder] = struct{}{}
	}
	for _, source := range []string{subject, content} {
		for _, match := range emailTemplatePlaceholderPattern.FindAllStringSubmatch(source, -1) {
			if _, ok := allowed[match[1]]; !ok {
				return fmt.Errorf("placeholder %s is not available for this template", match[0])
			}
		}
	}
	return nil
}

func loadCustomEmailTemplates() map[string]storedEmailTemplate {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[common.EmailTemplatesOptionKey]
	common.OptionMapRWMutex.RUnlock()
	custom := map[string]storedEmailTemplate{}
	if strings.TrimSpace(raw) == "" {
		return custom
	}
	if err := common.UnmarshalJsonStr(raw, &custom); err != nil {
		common.SysError("failed to load email templates: " + err.Error())
		return map[string]storedEmailTemplate{}
	}
	return custom
}

func renderStoredEmailTemplate(template storedEmailTemplate, values map[string]string) EmailTemplatePreview {
	subject := template.Subject
	content := template.Content
	for key, value := range values {
		token := "{{" + key + "}}"
		subject = strings.ReplaceAll(subject, token, value)
		content = strings.ReplaceAll(content, token, html.EscapeString(value))
	}
	return EmailTemplatePreview{Subject: subject, Content: content}
}

func emailTemplateSampleValues() map[string]string {
	return map[string]string{
		"system_name":           common.SystemName,
		"username":              "demo_user",
		"display_name":          "示例用户",
		"email":                 "user@example.com",
		"verification_code":     "123456",
		"expires_in_minutes":    "15",
		"reset_url":             "https://example.com/user/reset?token=preview",
		"subscription_name":     "Pro",
		"subscription_end_time": "2026-08-01 12:00:00",
		"days_remaining":        "3",
	}
}
