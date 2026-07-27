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
	EmailTemplateEventGeneralNotification        = "notification.general"
	EmailTemplateEventVerifyCode                 = "auth.verify_code"
	EmailTemplateEventPasswordReset              = "auth.password_reset"
	EmailTemplateEventSubscriptionExpiryReminder = "subscription.expiry_reminder"
	EmailTemplateEventLowBalance                 = "balance.low"
	EmailTemplateEventAccountQuotaAlert          = "account.quota_alert"
	EmailTemplateEventChannelAnomalyDisabled     = "channel.anomaly_disabled"
	EmailTemplateEventDashboardReport            = "dashboard.report"
	EmailTemplateEventUserQuotaAdjustment        = "user.quota_adjustment"
	EmailTemplateLocaleChinese                   = "zh"
	EmailTemplateLocaleEnglish                   = "en"
	maxEmailTemplateSubjectLength                = 255
	maxEmailTemplateContentLength                = 200000
	emailTemplateLocalizedStorageKeySeparator    = "::"
)

type EmailTemplateDefinition struct {
	Event              string   `json:"event"`
	Label              string   `json:"label"`
	Description        string   `json:"description"`
	Category           string   `json:"category"`
	CampaignCompatible bool     `json:"campaign_compatible"`
	Placeholders       []string `json:"placeholders"`
}

type EmailTemplate struct {
	EmailTemplateDefinition
	Locale   string `json:"locale"`
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
	emailTemplateLocales            = []string{EmailTemplateLocaleChinese, EmailTemplateLocaleEnglish}
	emailTemplateOrder              = []string{
		EmailTemplateEventGeneralNotification,
		EmailTemplateEventVerifyCode,
		EmailTemplateEventPasswordReset,
		EmailTemplateEventSubscriptionExpiryReminder,
		EmailTemplateEventLowBalance,
		EmailTemplateEventAccountQuotaAlert,
		EmailTemplateEventChannelAnomalyDisabled,
		EmailTemplateEventDashboardReport,
		EmailTemplateEventUserQuotaAdjustment,
	}
	emailTemplateDefinitions = map[string]EmailTemplateDefinition{
		EmailTemplateEventGeneralNotification: {
			Event:              EmailTemplateEventGeneralNotification,
			Label:              "General notification",
			Description:        "A reusable template for email campaigns and general notices.",
			Category:           "Notifications",
			CampaignCompatible: true,
			Placeholders:       []string{"system_name", "username", "display_name", "email"},
		},
		EmailTemplateEventVerifyCode: {
			Event:        EmailTemplateEventVerifyCode,
			Label:        "Email verification code",
			Description:  "Sent for registration, email binding, and email verification flows.",
			Category:     "Authentication",
			Placeholders: []string{"system_name", "username", "display_name", "email", "verification_code", "expires_in_minutes"},
		},
		EmailTemplateEventPasswordReset: {
			Event:        EmailTemplateEventPasswordReset,
			Label:        "Password reset email",
			Description:  "Sent when a registered user requests a password reset link.",
			Category:     "Authentication",
			Placeholders: []string{"system_name", "username", "display_name", "email", "reset_url", "expires_in_minutes"},
		},
		EmailTemplateEventSubscriptionExpiryReminder: {
			Event:              EmailTemplateEventSubscriptionExpiryReminder,
			Label:              "Subscription expiry reminder",
			Description:        "Sent 7, 3, and 1 days before an active subscription expires.",
			Category:           "Subscriptions",
			CampaignCompatible: true,
			Placeholders:       []string{"system_name", "username", "display_name", "email", "subscription_name", "subscription_end_time", "days_remaining"},
		},
		EmailTemplateEventLowBalance: {
			Event:        EmailTemplateEventLowBalance,
			Label:        "Low balance reminder",
			Description:  "Sent when a user's wallet or subscription balance falls below the configured threshold.",
			Category:     "Billing",
			Placeholders: []string{"system_name", "username", "display_name", "email", "balance_type", "current_balance", "warning_threshold", "recharge_url"},
		},
		EmailTemplateEventAccountQuotaAlert: {
			Event:        EmailTemplateEventAccountQuotaAlert,
			Label:        "Account quota alert",
			Description:  "Sent to selected administrators when a channel account balance crosses below the warning threshold.",
			Category:     "Operations",
			Placeholders: []string{"system_name", "username", "display_name", "email", "channel_id", "channel_name", "channel_type", "current_balance", "warning_threshold", "checked_at"},
		},
		EmailTemplateEventChannelAnomalyDisabled: {
			Event:        EmailTemplateEventChannelAnomalyDisabled,
			Label:        "Channel anomaly alert",
			Description:  "Sent only when a channel is automatically disabled after an anomaly; manual shutdowns are excluded.",
			Category:     "Operations",
			Placeholders: []string{"system_name", "username", "display_name", "email", "channel_id", "channel_name", "channel_type", "channel_base_url", "failure_reason", "disabled_at"},
		},
		EmailTemplateEventDashboardReport: {
			Event:        EmailTemplateEventDashboardReport,
			Label:        "Data dashboard report",
			Description:  "Sent to selected administrators with a completed dashboard usage summary.",
			Category:     "Operations",
			Placeholders: []string{"system_name", "username", "display_name", "email", "report_type", "report_period", "generated_at", "total_consumption", "total_quota", "total_requests", "total_tokens", "active_users", "active_models", "active_channels", "active_groups", "top_models", "top_users", "group_analysis"},
		},
		EmailTemplateEventUserQuotaAdjustment: {
			Event:        EmailTemplateEventUserQuotaAdjustment,
			Label:        "Quota adjustment notice",
			Description:  "Sent after an administrator adjusts user quota in bulk.",
			Category:     "Billing",
			Placeholders: []string{"system_name", "username", "display_name", "email", "operation", "adjustment_amount", "previous_quota", "current_quota", "operator_name", "adjusted_at"},
		},
	}
	emailTemplateDefaults = newEmailTemplateDefaults()
)

func newEmailTemplateDefaults() map[string]storedEmailTemplate {
	defaults := map[string]storedEmailTemplate{}
	add := func(event, locale, subject, title, accent, body string) {
		defaults[emailTemplateStorageKey(event, locale)] = storedEmailTemplate{
			Subject: subject,
			Content: buildLocalizedEmailTemplateCard(locale, title, accent, body),
		}
	}

	add(EmailTemplateEventGeneralNotification, EmailTemplateLocaleChinese,
		"[{{system_name}}] 重要通知", "重要通知", "#0f766e", `
<p>{{display_name}}，您好：</p>
<p>这里填写需要发送给用户的通知内容。</p>
<p style="color:#6b7280;">感谢您对 {{system_name}} 的支持。</p>`)
	add(EmailTemplateEventGeneralNotification, EmailTemplateLocaleEnglish,
		"[{{system_name}}] Important notice", "Important notice", "#0f766e", `
<p>Hello {{display_name}},</p>
<p>Replace this paragraph with the announcement you want to send.</p>
<p style="color:#6b7280;">Thank you for using {{system_name}}.</p>`)

	add(EmailTemplateEventVerifyCode, EmailTemplateLocaleChinese,
		"[{{system_name}}] 邮箱验证码", "邮箱验证码", "#2563eb", `
<p>{{display_name}}，您好：</p>
<p>您的邮箱验证码是：</p>
<p style="margin:24px 0;font-size:32px;font-weight:700;letter-spacing:8px;text-align:center;color:#111827;">{{verification_code}}</p>
<p>验证码将在 <strong>{{expires_in_minutes}}</strong> 分钟后失效。</p>
<p style="color:#6b7280;">如果不是您本人操作，请忽略此邮件。</p>`)
	add(EmailTemplateEventVerifyCode, EmailTemplateLocaleEnglish,
		"[{{system_name}}] Email verification code", "Email verification code", "#2563eb", `
<p>Hello {{display_name}},</p>
<p>Your email verification code is:</p>
<p style="margin:24px 0;font-size:32px;font-weight:700;letter-spacing:8px;text-align:center;color:#111827;">{{verification_code}}</p>
<p>This code expires in <strong>{{expires_in_minutes}}</strong> minutes.</p>
<p style="color:#6b7280;">Ignore this email if you did not request the code.</p>`)

	add(EmailTemplateEventPasswordReset, EmailTemplateLocaleChinese,
		"[{{system_name}}] 密码重置请求", "密码重置", "#7c3aed", `
<p>{{display_name}}，您好：</p>
<p>我们收到了您的密码重置请求，请点击下方按钮设置新密码。</p>
<p style="margin:24px 0;text-align:center;"><a href="{{reset_url}}" style="display:inline-block;padding:10px 18px;border-radius:6px;background:#7c3aed;color:#ffffff;text-decoration:none;font-weight:600;">重置密码</a></p>
<p>此链接将在 <strong>{{expires_in_minutes}}</strong> 分钟后失效。</p>
<p style="color:#6b7280;word-break:break-all;">如果按钮无法点击，请复制以下链接到浏览器中打开：<br>{{reset_url}}</p>`)
	add(EmailTemplateEventPasswordReset, EmailTemplateLocaleEnglish,
		"[{{system_name}}] Password reset request", "Reset your password", "#7c3aed", `
<p>Hello {{display_name}},</p>
<p>We received a request to reset your password.</p>
<p style="margin:24px 0;text-align:center;"><a href="{{reset_url}}" style="display:inline-block;padding:10px 18px;border-radius:6px;background:#7c3aed;color:#ffffff;text-decoration:none;font-weight:600;">Reset password</a></p>
<p>This link expires in <strong>{{expires_in_minutes}}</strong> minutes.</p>
<p style="color:#6b7280;word-break:break-all;">If the button does not work, open this link:<br>{{reset_url}}</p>`)

	add(EmailTemplateEventSubscriptionExpiryReminder, EmailTemplateLocaleChinese,
		"[{{system_name}}] 订阅将在 {{days_remaining}} 天后到期", "订阅到期提醒", "#ea580c", `
<p>{{display_name}}，您好：</p>
<p>您的 <strong>{{subscription_name}}</strong> 订阅将在 <strong>{{days_remaining}}</strong> 天后到期。</p>
<p>到期时间：<strong>{{subscription_end_time}}</strong></p>
<p style="color:#6b7280;">如需继续使用订阅权益，请及时续订。</p>`)
	add(EmailTemplateEventSubscriptionExpiryReminder, EmailTemplateLocaleEnglish,
		"[{{system_name}}] Subscription expires in {{days_remaining}} days", "Subscription expiry reminder", "#ea580c", `
<p>Hello {{display_name}},</p>
<p>Your <strong>{{subscription_name}}</strong> subscription expires in <strong>{{days_remaining}}</strong> days.</p>
<p>Expiry time: <strong>{{subscription_end_time}}</strong></p>
<p style="color:#6b7280;">Renew in time to keep your subscription benefits active.</p>`)

	add(EmailTemplateEventLowBalance, EmailTemplateLocaleChinese,
		"[{{system_name}}] {{balance_type}}不足提醒", "余额不足提醒", "#dc2626", `
<p>{{display_name}}，您好：</p>
<p>您的{{balance_type}}已低于提醒阈值，请及时处理以免影响服务。</p>
<p>当前剩余：<strong>{{current_balance}}</strong><br>提醒阈值：<strong>{{warning_threshold}}</strong></p>
<p style="margin:24px 0;text-align:center;"><a href="{{recharge_url}}" style="display:inline-block;padding:10px 18px;border-radius:6px;background:#dc2626;color:#ffffff;text-decoration:none;font-weight:600;">前往充值</a></p>`)
	add(EmailTemplateEventLowBalance, EmailTemplateLocaleEnglish,
		"[{{system_name}}] Low {{balance_type}} reminder", "Low balance reminder", "#dc2626", `
<p>Hello {{display_name}},</p>
<p>Your {{balance_type}} has fallen below the warning threshold.</p>
<p>Remaining: <strong>{{current_balance}}</strong><br>Warning threshold: <strong>{{warning_threshold}}</strong></p>
<p style="margin:24px 0;text-align:center;"><a href="{{recharge_url}}" style="display:inline-block;padding:10px 18px;border-radius:6px;background:#dc2626;color:#ffffff;text-decoration:none;font-weight:600;">Recharge now</a></p>`)

	add(EmailTemplateEventAccountQuotaAlert, EmailTemplateLocaleChinese,
		"[{{system_name}}] 渠道 {{channel_name}} 账号额度告警", "账号限额通知", "#d97706", `
<p>{{display_name}}，您好：</p>
<p>渠道 <strong>{{channel_name}}</strong>（#{{channel_id}}）的上游账号余额已低于告警阈值。</p>
<p>渠道类型：{{channel_type}}<br>当前余额：<strong>{{current_balance}}</strong><br>告警阈值：<strong>{{warning_threshold}}</strong><br>检查时间：{{checked_at}}</p>`)
	add(EmailTemplateEventAccountQuotaAlert, EmailTemplateLocaleEnglish,
		"[{{system_name}}] Channel {{channel_name}} quota alert", "Account quota alert", "#d97706", `
<p>Hello {{display_name}},</p>
<p>The upstream account balance for channel <strong>{{channel_name}}</strong> (#{{channel_id}}) is below the warning threshold.</p>
<p>Channel type: {{channel_type}}<br>Current balance: <strong>{{current_balance}}</strong><br>Warning threshold: <strong>{{warning_threshold}}</strong><br>Checked at: {{checked_at}}</p>`)

	add(EmailTemplateEventChannelAnomalyDisabled, EmailTemplateLocaleChinese,
		"[{{system_name}}] 渠道 {{channel_name}} 已异常关闭", "渠道异常提醒", "#b91c1c", `
<p>{{display_name}}，您好：</p>
<p>渠道 <strong>{{channel_name}}</strong>（#{{channel_id}}）因异常被系统自动关闭。</p>
<p>渠道类型：{{channel_type}}<br>接口地址：{{channel_base_url}}<br>关闭时间：{{disabled_at}}</p>
<p style="padding:12px;border-radius:6px;background:#fef2f2;color:#991b1b;word-break:break-word;">{{failure_reason}}</p>`)
	add(EmailTemplateEventChannelAnomalyDisabled, EmailTemplateLocaleEnglish,
		"[{{system_name}}] Channel {{channel_name}} was automatically disabled", "Channel anomaly alert", "#b91c1c", `
<p>Hello {{display_name}},</p>
<p>Channel <strong>{{channel_name}}</strong> (#{{channel_id}}) was automatically disabled after an anomaly.</p>
<p>Channel type: {{channel_type}}<br>Base URL: {{channel_base_url}}<br>Disabled at: {{disabled_at}}</p>
<p style="padding:12px;border-radius:6px;background:#fef2f2;color:#991b1b;word-break:break-word;">{{failure_reason}}</p>`)

	add(EmailTemplateEventDashboardReport, EmailTemplateLocaleChinese,
		"[{{system_name}}] {{report_type}}数据报表 {{report_period}}", "数据看板报表", "#2563eb", `
<p>{{display_name}}，您好：</p>
<p>以下是 <strong>{{report_period}}</strong> 的数据看板汇总。</p>
<table role="presentation" style="width:100%;margin:24px 0;border-collapse:separate;border-spacing:8px;table-layout:fixed;">
  <tr><td style="padding:16px;border-radius:6px;background:#f3f4f6;"><span style="display:block;color:#6b7280;font-size:12px;">总消费</span><strong style="font-size:20px;">{{total_consumption}}</strong></td><td style="padding:16px;border-radius:6px;background:#f3f4f6;"><span style="display:block;color:#6b7280;font-size:12px;">请求数</span><strong style="font-size:20px;">{{total_requests}}</strong></td></tr>
  <tr><td style="padding:16px;border-radius:6px;background:#f3f4f6;"><span style="display:block;color:#6b7280;font-size:12px;">总 Token</span><strong style="font-size:20px;">{{total_tokens}}</strong></td><td style="padding:16px;border-radius:6px;background:#f3f4f6;"><span style="display:block;color:#6b7280;font-size:12px;">活跃用户</span><strong style="font-size:20px;">{{active_users}}</strong></td></tr>
</table>
<p>原始额度：<strong>{{total_quota}}</strong><br>活跃模型：<strong>{{active_models}}</strong>，活跃渠道：<strong>{{active_channels}}</strong>，活跃分组：<strong>{{active_groups}}</strong></p>
<p style="margin-bottom:8px;font-weight:600;">消费最高的模型</p>
<pre style="margin:0;padding:14px;border-radius:6px;background:#f8fafc;color:#334155;font-family:inherit;white-space:pre-wrap;word-break:break-word;">{{top_models}}</pre>
<p style="margin:20px 0 8px;font-weight:600;">用户统计（按消费排行）</p>
<pre style="margin:0;padding:14px;border-radius:6px;background:#f8fafc;color:#334155;font-family:inherit;white-space:pre-wrap;word-break:break-word;">{{top_users}}</pre>
<p style="margin:20px 0 8px;font-weight:600;">分组数据分析（按消费排行）</p>
<pre style="margin:0;padding:14px;border-radius:6px;background:#f8fafc;color:#334155;font-family:inherit;white-space:pre-wrap;word-break:break-word;">{{group_analysis}}</pre>
<p style="margin-top:24px;color:#6b7280;font-size:12px;">生成时间：{{generated_at}}</p>`)
	add(EmailTemplateEventDashboardReport, EmailTemplateLocaleEnglish,
		"[{{system_name}}] {{report_type}} dashboard report {{report_period}}", "Data dashboard report", "#2563eb", `
<p>Hello {{display_name}},</p>
<p>Here is the completed data dashboard summary for <strong>{{report_period}}</strong>.</p>
<table role="presentation" style="width:100%;margin:24px 0;border-collapse:separate;border-spacing:8px;table-layout:fixed;">
  <tr><td style="padding:16px;border-radius:6px;background:#f3f4f6;"><span style="display:block;color:#6b7280;font-size:12px;">Consumption</span><strong style="font-size:20px;">{{total_consumption}}</strong></td><td style="padding:16px;border-radius:6px;background:#f3f4f6;"><span style="display:block;color:#6b7280;font-size:12px;">Requests</span><strong style="font-size:20px;">{{total_requests}}</strong></td></tr>
  <tr><td style="padding:16px;border-radius:6px;background:#f3f4f6;"><span style="display:block;color:#6b7280;font-size:12px;">Total tokens</span><strong style="font-size:20px;">{{total_tokens}}</strong></td><td style="padding:16px;border-radius:6px;background:#f3f4f6;"><span style="display:block;color:#6b7280;font-size:12px;">Active users</span><strong style="font-size:20px;">{{active_users}}</strong></td></tr>
</table>
<p>Raw quota: <strong>{{total_quota}}</strong><br>Active models: <strong>{{active_models}}</strong>, active channels: <strong>{{active_channels}}</strong>, active groups: <strong>{{active_groups}}</strong></p>
<p style="margin-bottom:8px;font-weight:600;">Top models by consumption</p>
<pre style="margin:0;padding:14px;border-radius:6px;background:#f8fafc;color:#334155;font-family:inherit;white-space:pre-wrap;word-break:break-word;">{{top_models}}</pre>
<p style="margin:20px 0 8px;font-weight:600;">User analytics by consumption</p>
<pre style="margin:0;padding:14px;border-radius:6px;background:#f8fafc;color:#334155;font-family:inherit;white-space:pre-wrap;word-break:break-word;">{{top_users}}</pre>
<p style="margin:20px 0 8px;font-weight:600;">Group data analysis by consumption</p>
<pre style="margin:0;padding:14px;border-radius:6px;background:#f8fafc;color:#334155;font-family:inherit;white-space:pre-wrap;word-break:break-word;">{{group_analysis}}</pre>
<p style="margin-top:24px;color:#6b7280;font-size:12px;">Generated at {{generated_at}}</p>`)

	add(EmailTemplateEventUserQuotaAdjustment, EmailTemplateLocaleChinese,
		"[{{system_name}}] 额度调整通知", "额度调整通知", "#0f766e", `
<p>{{display_name}}，您好：</p>
<p>管理员 <strong>{{operator_name}}</strong> 已对您的账户额度执行<strong>{{operation}}</strong>操作。</p>
<table role="presentation" style="width:100%;margin:24px 0;border-collapse:separate;border-spacing:8px;table-layout:fixed;">
  <tr><td style="padding:16px;border-radius:6px;background:#f3f4f6;"><span style="display:block;color:#6b7280;font-size:12px;">调整前</span><strong style="font-size:20px;">{{previous_quota}}</strong></td><td style="padding:16px;border-radius:6px;background:#ecfdf5;"><span style="display:block;color:#047857;font-size:12px;">调整后</span><strong style="font-size:20px;color:#065f46;">{{current_quota}}</strong></td></tr>
</table>
<p>调整额度：<strong>{{adjustment_amount}}</strong><br>调整时间：{{adjusted_at}}</p>`)
	add(EmailTemplateEventUserQuotaAdjustment, EmailTemplateLocaleEnglish,
		"[{{system_name}}] Quota adjustment notice", "Quota adjustment notice", "#0f766e", `
<p>Hello {{display_name}},</p>
<p>Administrator <strong>{{operator_name}}</strong> performed a quota <strong>{{operation}}</strong> on your account.</p>
<table role="presentation" style="width:100%;margin:24px 0;border-collapse:separate;border-spacing:8px;table-layout:fixed;">
  <tr><td style="padding:16px;border-radius:6px;background:#f3f4f6;"><span style="display:block;color:#6b7280;font-size:12px;">Before</span><strong style="font-size:20px;">{{previous_quota}}</strong></td><td style="padding:16px;border-radius:6px;background:#ecfdf5;"><span style="display:block;color:#047857;font-size:12px;">After</span><strong style="font-size:20px;color:#065f46;">{{current_quota}}</strong></td></tr>
</table>
<p>Adjustment amount: <strong>{{adjustment_amount}}</strong><br>Adjusted at: {{adjusted_at}}</p>`)

	return defaults
}

func buildLocalizedEmailTemplateCard(locale, title, accent, body string) string {
	lang := "zh-CN"
	if locale == EmailTemplateLocaleEnglish {
		lang = "en"
	}
	return fmt.Sprintf(`<!doctype html>
<html lang="%s">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:24px;background:#f3f4f6;color:#1f2937;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;">
  <div style="max-width:620px;margin:0 auto;overflow:hidden;border-radius:8px;background:#ffffff;box-shadow:0 8px 24px rgba(15,23,42,.08);">
    <div style="padding:38px 40px;background:%s;">
      <h1 style="margin:0;font-size:30px;line-height:1.3;color:#ffffff;">%s</h1>
    </div>
    <div style="padding:38px 40px;line-height:1.7;">
      %s
    </div>
    <div style="padding:18px 40px;background:#f8fafc;border-top:1px solid #e5e7eb;color:#9ca3af;font-size:12px;">{{system_name}}</div>
  </div>
</body>
</html>`, lang, accent, title, strings.TrimSpace(body))
}

func SupportedEmailTemplateLocales() []string {
	return append([]string(nil), emailTemplateLocales...)
}

func NormalizeEmailTemplateLocale(locale string) string {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if strings.HasPrefix(locale, "en") {
		return EmailTemplateLocaleEnglish
	}
	return EmailTemplateLocaleChinese
}

func ListEmailTemplates() []EmailTemplate {
	templates := make([]EmailTemplate, 0, len(emailTemplateOrder)*len(emailTemplateLocales))
	for _, event := range emailTemplateOrder {
		for _, locale := range emailTemplateLocales {
			template, _ := GetEmailTemplateForLocale(event, locale)
			templates = append(templates, template)
		}
	}
	return templates
}

func GetEmailTemplate(event string) (EmailTemplate, error) {
	return GetEmailTemplateForLocale(event, EmailTemplateLocaleChinese)
}

func GetEmailTemplateForLocale(event, locale string) (EmailTemplate, error) {
	definition, ok := emailTemplateDefinitions[strings.TrimSpace(event)]
	if !ok {
		return EmailTemplate{}, errors.New("unsupported email template event")
	}
	locale = NormalizeEmailTemplateLocale(locale)
	storageKey := emailTemplateStorageKey(definition.Event, locale)
	stored := emailTemplateDefaults[storageKey]
	isCustom := false
	if custom, exists := loadCustomEmailTemplates()[storageKey]; exists {
		if err := validateEmailTemplate(definition, custom.Subject, custom.Content); err == nil {
			stored = custom
			isCustom = true
		} else {
			common.SysError(fmt.Sprintf("invalid custom email template ignored: event=%s locale=%s err=%v", definition.Event, locale, err))
		}
	}
	definition.Placeholders = append([]string(nil), definition.Placeholders...)
	return EmailTemplate{
		EmailTemplateDefinition: definition,
		Locale:                  locale,
		Subject:                 stored.Subject,
		Content:                 stored.Content,
		IsCustom:                isCustom,
	}, nil
}

func UpdateEmailTemplate(event, subject, content string) (EmailTemplate, error) {
	return UpdateEmailTemplateForLocale(event, EmailTemplateLocaleChinese, subject, content)
}

func UpdateEmailTemplateForLocale(event, locale, subject, content string) (EmailTemplate, error) {
	definition, ok := emailTemplateDefinitions[strings.TrimSpace(event)]
	if !ok {
		return EmailTemplate{}, errors.New("unsupported email template event")
	}
	locale = NormalizeEmailTemplateLocale(locale)
	subject = strings.TrimSpace(subject)
	if err := validateEmailTemplate(definition, subject, content); err != nil {
		return EmailTemplate{}, err
	}

	emailTemplatesMu.Lock()
	defer emailTemplatesMu.Unlock()
	custom := loadCustomEmailTemplates()
	custom[emailTemplateStorageKey(definition.Event, locale)] = storedEmailTemplate{Subject: subject, Content: content}
	if err := persistCustomEmailTemplates(custom); err != nil {
		return EmailTemplate{}, err
	}
	return GetEmailTemplateForLocale(definition.Event, locale)
}

func ResetEmailTemplate(event string) (EmailTemplate, error) {
	return ResetEmailTemplateForLocale(event, EmailTemplateLocaleChinese)
}

func ResetEmailTemplateForLocale(event, locale string) (EmailTemplate, error) {
	definition, ok := emailTemplateDefinitions[strings.TrimSpace(event)]
	if !ok {
		return EmailTemplate{}, errors.New("unsupported email template event")
	}
	locale = NormalizeEmailTemplateLocale(locale)

	emailTemplatesMu.Lock()
	defer emailTemplatesMu.Unlock()
	custom := loadCustomEmailTemplates()
	delete(custom, emailTemplateStorageKey(definition.Event, locale))
	if err := persistCustomEmailTemplates(custom); err != nil {
		return EmailTemplate{}, err
	}
	return GetEmailTemplateForLocale(definition.Event, locale)
}

func PreviewEmailTemplate(event, subject, content string) (EmailTemplatePreview, error) {
	return PreviewEmailTemplateForLocale(event, EmailTemplateLocaleChinese, subject, content)
}

func PreviewEmailTemplateForLocale(event, locale, subject, content string) (EmailTemplatePreview, error) {
	definition, ok := emailTemplateDefinitions[strings.TrimSpace(event)]
	if !ok {
		return EmailTemplatePreview{}, errors.New("unsupported email template event")
	}
	subject = strings.TrimSpace(subject)
	if err := validateEmailTemplate(definition, subject, content); err != nil {
		return EmailTemplatePreview{}, err
	}
	return renderStoredEmailTemplate(storedEmailTemplate{Subject: subject, Content: content}, emailTemplateSampleValues(NormalizeEmailTemplateLocale(locale))), nil
}

func RenderEmailTemplate(event string, values map[string]string) (EmailTemplatePreview, error) {
	return RenderEmailTemplateForLocale(event, EmailTemplateLocaleChinese, values)
}

func RenderEmailTemplateForLocale(event, locale string, values map[string]string) (EmailTemplatePreview, error) {
	template, err := GetEmailTemplateForLocale(event, locale)
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

func RenderCustomEmailTemplateForLocale(event, locale, subject, content string, values map[string]string) (EmailTemplatePreview, error) {
	definition, ok := emailTemplateDefinitions[strings.TrimSpace(event)]
	if !ok {
		return EmailTemplatePreview{}, errors.New("unsupported email template event")
	}
	subject = strings.TrimSpace(subject)
	if err := validateEmailTemplate(definition, subject, content); err != nil {
		return EmailTemplatePreview{}, err
	}
	resolved := make(map[string]string, len(values)+1)
	for key, value := range values {
		resolved[key] = value
	}
	resolved["system_name"] = common.SystemName
	return renderStoredEmailTemplate(storedEmailTemplate{Subject: subject, Content: content}, resolved), nil
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

func emailTemplateStorageKey(event, locale string) string {
	if NormalizeEmailTemplateLocale(locale) == EmailTemplateLocaleChinese {
		return event
	}
	return event + emailTemplateLocalizedStorageKeySeparator + EmailTemplateLocaleEnglish
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

func persistCustomEmailTemplates(custom map[string]storedEmailTemplate) error {
	data, err := common.Marshal(custom)
	if err != nil {
		return err
	}
	return model.UpdateOption(common.EmailTemplatesOptionKey, string(data))
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

func emailTemplateSampleValues(locale string) map[string]string {
	values := map[string]string{
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
		"balance_type":          "钱包余额",
		"current_balance":       "$2.50",
		"warning_threshold":     "$5.00",
		"recharge_url":          "https://example.com/wallet",
		"channel_id":            "18",
		"channel_name":          "OpenAI Production",
		"channel_type":          "OpenAI",
		"channel_base_url":      "https://api.openai.com",
		"checked_at":            "2026-08-01 12:00:00",
		"disabled_at":           "2026-08-01 12:00:00",
		"failure_reason":        "Upstream returned HTTP 401 repeatedly.",
		"report_type":           "日报",
		"report_period":         "2026-07-26 00:00 - 2026-07-27 00:00",
		"generated_at":          "2026-07-27 08:00:00",
		"total_consumption":     "$128.50",
		"total_quota":           "64,250,000",
		"total_requests":        "12,580",
		"total_tokens":          "48,320,000",
		"active_users":          "328",
		"active_models":         "18",
		"active_channels":       "12",
		"active_groups":         "4",
		"top_models":            "1. gpt-4.1  $52.30\n2. claude-sonnet-4  $41.20\n3. gemini-2.5-pro  $23.80",
		"top_users":             "1. alice  消费 $62.10 | 请求 5,320 | Token 20,100,000\n2. bob  消费 $41.30 | 请求 3,210 | Token 15,200,000",
		"group_analysis":        "1. default  消费 $72.40 | 请求 7,100 | Token 27,800,000 | 用户 210\n2. vip  消费 $56.10 | 请求 5,480 | Token 20,520,000 | 用户 118",
		"operation":             "增加",
		"adjustment_amount":     "$10.00",
		"previous_quota":        "$20.00",
		"current_quota":         "$30.00",
		"operator_name":         "root",
		"adjusted_at":           "2026-07-27 12:00:00",
	}
	if locale == EmailTemplateLocaleEnglish {
		values["display_name"] = "Demo User"
		values["balance_type"] = "wallet balance"
		values["report_type"] = "Daily"
		values["top_users"] = "1. alice  Consumption $62.10 | Requests 5,320 | Tokens 20,100,000\n2. bob  Consumption $41.30 | Requests 3,210 | Tokens 15,200,000"
		values["group_analysis"] = "1. default  Consumption $72.40 | Requests 7,100 | Tokens 27,800,000 | Users 210\n2. vip  Consumption $56.10 | Requests 5,480 | Tokens 20,520,000 | Users 118"
		values["operation"] = "increase"
	}
	return values
}
