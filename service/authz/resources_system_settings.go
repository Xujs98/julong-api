package authz

const ResourceSystemSettings = "system_settings"

const (
	ActionSupportContactsWrite = "content.support"
)

var SystemSettingsSupportContactsWrite = SystemSettingsPermission(ActionSupportContactsWrite)

func SystemSettingsPermission(action string) Permission {
	return Permission{Resource: ResourceSystemSettings, Action: action}
}

func init() {
	definitions := []struct{ action, label string }{
		{"site.system-info", "System Information"}, {"site.notice", "System Notice"},
		{"site.header-navigation", "Header navigation"}, {"site.sidebar-modules", "Sidebar modules"},
		{"auth.basic-auth", "Basic Authentication"}, {"auth.oauth", "OAuth Integrations"},
		{"auth.passkey", "Passkey Authentication"}, {"auth.bot-protection", "Bot Protection"},
		{"auth.custom-oauth", "Custom OAuth"},
		{"billing.quota", "Quota Settings"}, {"billing.currency", "Currency & Display"},
		{"billing.model-pricing", "Model Pricing"}, {"billing.group-pricing", "Group Pricing"},
		{"billing.payment", "Payment Gateway"}, {"billing.checkin", "Check-in Rewards"},
		{"models.global", "Global Model Configuration"}, {"models.routing-reliability", "Routing Reliability"},
		{"models.gemini", "Gemini"}, {"models.claude", "Claude"}, {"models.grok", "Grok"},
		{"models.channel-affinity", "Channel Affinity"}, {"models.model-deployment", "Model Deployment"},
		{"security.rate-limit", "Rate Limiting"}, {"security.sensitive-words", "Sensitive Words"},
		{"security.ssrf", "SSRF Protection"}, {"security.token-limits", "Token Limits"},
		{ActionSupportContactsWrite, "Customer Support"}, {"content.dashboard", "Data Dashboard"},
		{"content.announcements", "Announcements"}, {"content.api-info", "API Addresses"},
		{"content.faq", "FAQ"}, {"content.uptime-kuma", "Uptime Kuma"},
		{"content.chat", "Chat Presets"}, {"content.drawing", "Drawing"},
		{"operations.behavior", "System Behavior"}, {"operations.alerts", "Monitoring & Alerts"},
		{"operations.email", "SMTP Email"}, {"operations.worker", "Worker Proxy"},
		{"operations.logs", "Log Maintenance"}, {"operations.performance", "Performance"},
		{"operations.update-checker", "System maintenance"},
	}
	actions := make([]ActionDefinition, 0, len(definitions))
	for _, definition := range definitions {
		actions = append(actions, ActionDefinition{
			Action:         definition.action,
			LabelKey:       definition.label,
			DescriptionKey: "Allow this administrator to view and modify this settings page.",
		})
	}
	RegisterResource(ResourceDefinition{Resource: ResourceSystemSettings, LabelKey: "System Settings", Actions: actions})
}
