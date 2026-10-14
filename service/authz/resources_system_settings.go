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
		{"billing.payment", "Payment Gateway"}, {"billing.checkin", "Check-in Rewards"}, {"billing.invoice", "Self-service Invoices"},
		{"request-policies.filtering", "Content Filtering"}, {"request-policies.routing", "Routing & Retry"}, {"request-policies.health", "Channel Health"},
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
		{"operations.email-reminders", "Subscription expiry reminders"},
		{"operations.email-templates", "Email templates"},
		{"operations.email-campaigns", "Email Campaigns"},
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

// SystemSettingsPermissionActions maps reorganized policy pages to their
// existing Julong grants so an upstream navigation change does not revoke them.
func SystemSettingsPermissionActions(action string) []string {
	switch action {
	case "request-policies.filtering":
		return []string{action, "security.sensitive-words"}
	case "request-policies.routing":
		return []string{action, "models.channel-affinity", "models.routing-reliability"}
	case "request-policies.health":
		return []string{action, "models.routing-reliability", "operations.monitoring"}
	default:
		return []string{action}
	}
}

func CanSystemSettings(userID, role int, action string) bool {
	permission := SystemSettingsPermission(action)
	if Can(userID, role, permission) {
		return true
	}
	// An explicit new-page denial takes precedence over legacy grants.
	if e := currentEnforcer(); e != nil {
		if effect, ok := explicitSubjectEffect(e, UserSubject(userID), permission); ok && effect == EffectDeny {
			return false
		}
		for _, roleKey := range resolveSubjectRoles(userID, role) {
			if effect, ok := explicitSubjectEffect(e, RoleSubject(roleKey), permission); ok && effect == EffectDeny {
				return false
			}
		}
	}
	for _, oldAction := range SystemSettingsPermissionActions(action)[1:] {
		if Can(userID, role, SystemSettingsPermission(oldAction)) {
			return true
		}
	}
	return false
}
