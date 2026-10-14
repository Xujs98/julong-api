package operation_setting

import (
	"errors"
	"slices"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

type InvoiceSetting struct {
	AllowedSources       []string `json:"allowed_sources"`
	Enabled              bool     `json:"enabled"`
	MinimumAmount        float64  `json:"minimum_amount"`
	Unit                 string   `json:"unit"`
	ProcessingDays       string   `json:"processing_days"`
	RejectionFreezeHours int      `json:"rejection_freeze_hours"`
}

var invoiceSetting = InvoiceSetting{
	AllowedSources:       []string{"online_recharge", "redemption", "admin_adjustment"},
	Enabled:              false,
	MinimumAmount:        300,
	Unit:                 "USD",
	ProcessingDays:       "1-3",
	RejectionFreezeHours: 72,
}

func init() {
	config.GlobalConfig.Register("invoice_setting", &invoiceSetting)
}

func GetInvoiceSetting() InvoiceSetting {
	setting := invoiceSetting
	setting.AllowedSources = append([]string{}, invoiceSetting.AllowedSources...)
	return setting
}

func (setting InvoiceSetting) AllowsSource(source string) bool {
	return slices.Contains(setting.AllowedSources, source)
}

func ValidateInvoiceAllowedSources(value string) error {
	var sources []string
	if err := common.UnmarshalJsonStr(value, &sources); err != nil || sources == nil {
		return errors.New("Invoice application sources must be a JSON array")
	}
	seen := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		if source != "online_recharge" && source != "redemption" && source != "admin_adjustment" {
			return errors.New("Invalid invoice application source")
		}
		if _, exists := seen[source]; exists {
			return errors.New("Duplicate invoice application source")
		}
		seen[source] = struct{}{}
	}
	return nil
}
