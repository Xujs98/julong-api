package controller

import "testing"

func TestOptionAllowedForSystemSettingsAction(t *testing.T) {
	tests := []struct {
		key, action string
		allowed     bool
	}{
		{"SystemName", "site.system-info", true},
		{"SystemName", "operations.logs", false},
		{"ImageGenerationLogEnabled", "operations.logs", true},
		{"ModelPrice", "billing.model-pricing", true},
		{"GroupRatio", "billing.group-pricing", true},
		{"WaffoPrivateKey", "billing.payment", true},
		{"GitHubClientSecret", "auth.oauth", true},
		{"GitHubClientSecret", "content.dashboard", false},
		{"console_setting.custom_endpoints", "content.api-info", true},
		{"console_setting.custom_endpoints", "content.dashboard", false},
		{"unknown.option", "site.system-info", false},
	}
	for _, test := range tests {
		if got := optionAllowedForSystemSettingsAction(test.key, test.action); got != test.allowed {
			t.Fatalf("key=%s action=%s allowed=%v, want %v", test.key, test.action, got, test.allowed)
		}
	}
}
