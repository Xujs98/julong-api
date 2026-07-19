package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
)

func systemSettingsAction(c *gin.Context) string {
	return strings.TrimSpace(c.Query("section"))
}

func canAccessSystemSettingsAction(c *gin.Context, action string) bool {
	if c.GetInt("role") >= common.RoleRootUser {
		return true
	}
	return action != "" && authz.Can(c.GetInt("id"), c.GetInt("role"), authz.SystemSettingsPermission(action))
}

func optionAllowedForSystemSettingsAction(key, action string) bool {
	if action == "billing.payment" && strings.HasPrefix(key, "Waffo") {
		return true
	}
	allowed := map[string][]string{
		"site.system-info": {"theme.frontend", "SystemName", "Logo", "Footer", "About", "HomePageContent", "ServerAddress", "legal.user_agreement", "legal.privacy_policy"},
		"site.notice":      {"Notice"}, "site.header-navigation": {"HeaderNavModules"}, "site.sidebar-modules": {"SidebarModulesAdmin"},
		"auth.basic-auth":            {"PasswordLoginEnabled", "PasswordRegisterEnabled", "EmailVerificationEnabled", "RegisterEnabled", "EmailDomainRestrictionEnabled", "EmailAliasRestrictionEnabled", "EmailDomainWhitelist", "ServerAddress"},
		"auth.oauth":                 {"GitHubOAuthEnabled", "GitHubClientId", "GitHubClientSecret", "TelegramOAuthEnabled", "TelegramBotToken", "TelegramBotName", "LinuxDOOAuthEnabled", "LinuxDOClientId", "LinuxDOClientSecret", "LinuxDOMinimumTrustLevel", "WeChatAuthEnabled", "WeChatServerAddress", "WeChatServerToken", "WeChatAccountQRCodeImageURL", "discord.enabled", "discord.client_id", "discord.client_secret", "oidc.enabled", "oidc.client_id", "oidc.client_secret", "oidc.well_known", "oidc.authorization_endpoint", "oidc.token_endpoint", "oidc.user_info_endpoint"},
		"auth.passkey":               {"passkey.enabled", "passkey.rp_display_name", "passkey.rp_id", "passkey.origins", "passkey.allow_insecure_origin", "passkey.user_verification", "passkey.attachment_preference"},
		"auth.bot-protection":        {"TurnstileCheckEnabled", "TurnstileSiteKey", "TurnstileSecretKey"},
		"billing.quota":              {"QuotaForNewUser", "PreConsumedQuota", "QuotaForInviter", "QuotaForInvitee", "TopUpLink", "general_setting.docs_link", "quota_setting.enable_free_model_pre_consume", "payment_setting.compliance_confirmed", "payment_setting.compliance_terms_version"},
		"billing.currency":           {"QuotaPerUnit", "USDExchangeRate", "DisplayInCurrencyEnabled", "DisplayTokenStatEnabled", "general_setting.quota_display_type", "general_setting.custom_currency_symbol", "general_setting.custom_currency_exchange_rate"},
		"billing.model-pricing":      {"ModelPrice", "ModelRatio", "CacheRatio", "CreateCacheRatio", "CompletionRatio", "ImageRatio", "AudioRatio", "AudioCompletionRatio", "ExposeRatioEnabled", "billing_setting.billing_mode", "billing_setting.billing_expr", "billing_setting.per_request_subscription_allowed", "tool_price_setting.prices"},
		"billing.group-pricing":      {"TopupGroupRatio", "GroupRatio", "UserUsableGroups", "GroupGroupRatio", "AutoGroups", "DefaultUseAutoGroup", "group_ratio_setting.group_special_usable_group"},
		"billing.payment":            {"PayAddress", "EpayId", "EpayKey", "Price", "MinTopUp", "CustomCallbackAddress", "PayMethods", "payment_setting.amount_options", "payment_setting.amount_discount", "payment_setting.compliance_confirmed", "payment_setting.compliance_terms_version", "payment_setting.compliance_confirmed_at", "payment_setting.compliance_confirmed_by", "StripeApiSecret", "StripeWebhookSecret", "StripePriceId", "StripeUnitPrice", "StripeMinTopUp", "StripePromotionCodesEnabled", "CreemApiKey", "CreemWebhookSecret", "CreemTestMode", "CreemProducts"},
		"billing.checkin":            {"checkin_setting.enabled", "checkin_setting.min_quota", "checkin_setting.max_quota"},
		"models.global":              {"global.pass_through_request_enabled", "global.thinking_model_blacklist", "global.chat_completions_to_responses_policy", "general_setting.ping_interval_enabled", "general_setting.ping_interval_seconds"},
		"models.routing-reliability": {"RetryTimes", "ChannelDisableThreshold", "AutomaticDisableChannelEnabled", "AutomaticEnableChannelEnabled", "AutomaticDisableKeywords", "AutomaticDisableStatusCodes", "AutomaticRetryStatusCodes", "monitor_setting.auto_test_channel_enabled", "monitor_setting.auto_test_channel_minutes", "monitor_setting.channel_test_mode"},
		"models.gemini":              {"gemini.safety_settings", "gemini.version_settings", "gemini.supported_imagine_models", "gemini.thinking_adapter_enabled", "gemini.thinking_adapter_budget_tokens_percentage", "gemini.function_call_thought_signature_enabled", "gemini.remove_function_response_id_enabled"},
		"models.claude":              {"claude.model_headers_settings", "claude.default_max_tokens", "claude.thinking_adapter_enabled", "claude.thinking_adapter_budget_tokens_percentage"},
		"models.grok":                {"grok.violation_deduction_enabled", "grok.violation_deduction_amount"},
		"models.channel-affinity":    {"channel_affinity_setting.enabled", "channel_affinity_setting.switch_on_success", "channel_affinity_setting.keep_on_channel_disabled", "channel_affinity_setting.max_entries", "channel_affinity_setting.default_ttl_seconds", "channel_affinity_setting.rules"},
		"models.model-deployment":    {"model_deployment.ionet.api_key", "model_deployment.ionet.enabled"},
		"security.rate-limit":        {"ModelRequestRateLimitEnabled", "ModelRequestRateLimitCount", "ModelRequestRateLimitSuccessCount", "ModelRequestRateLimitDurationMinutes", "ModelRequestRateLimitGroup"},
		"security.sensitive-words":   {"CheckSensitiveEnabled", "CheckSensitiveOnPromptEnabled", "SensitiveWords"},
		"security.ssrf":              {"fetch_setting.enable_ssrf_protection", "fetch_setting.allow_private_ip", "fetch_setting.domain_filter_mode", "fetch_setting.ip_filter_mode", "fetch_setting.domain_list", "fetch_setting.ip_list", "fetch_setting.allowed_ports", "fetch_setting.apply_ip_filter_for_domain"},
		"security.token-limits":      {"token_setting.max_user_tokens"},
		"content.dashboard":          {"DataExportEnabled", "DataExportDefaultTime", "DataExportInterval"},
		"content.announcements":      {"console_setting.announcements", "console_setting.announcements_enabled"},
		"content.api-info":           {"console_setting.api_info", "console_setting.api_info_enabled", "console_setting.custom_endpoints"},
		"content.faq":                {"console_setting.faq", "console_setting.faq_enabled"},
		"content.uptime-kuma":        {"console_setting.uptime_kuma_groups", "console_setting.uptime_kuma_enabled"},
		"content.chat":               {"Chats"},
		"content.drawing":            {"DrawingEnabled", "MjNotifyEnabled", "MjAccountFilterEnabled", "MjForwardUrlEnabled", "MjModeClearEnabled", "MjActionCheckSuccessEnabled"},
		"operations.behavior":        {"DefaultCollapseSidebar", "DemoSiteEnabled", "SelfUseModeEnabled"},
		"operations.alerts":          {"QuotaRemindThreshold", "perf_metrics_setting.enabled", "perf_metrics_setting.flush_interval", "perf_metrics_setting.bucket_time", "perf_metrics_setting.retention_days"},
		"operations.email":           {"SMTPServer", "SMTPPort", "SMTPAccount", "SMTPFrom", "SMTPToken", "SMTPSSLEnabled", "SMTPStartTLSEnabled", "SMTPInsecureSkipVerify", "SMTPForceAuthLogin"},
		"operations.worker":          {"WorkerUrl", "WorkerValidKey", "WorkerAllowHttpImageRequestEnabled"},
		"operations.logs":            {"LogConsumeEnabled", "ImageGenerationLogEnabled", "ImageGenerationLogRetentionDays", "ImageGenerationLogPollingIntervalSeconds"},
		"operations.performance":     {"performance_setting.disk_cache_enabled", "performance_setting.disk_cache_threshold_mb", "performance_setting.disk_cache_max_size_mb", "performance_setting.disk_cache_path", "performance_setting.monitor_enabled", "performance_setting.monitor_cpu_threshold", "performance_setting.monitor_memory_threshold", "performance_setting.monitor_disk_threshold"},
	}
	for _, allowedKey := range allowed[action] {
		if key == allowedKey {
			return true
		}
	}
	return false
}
