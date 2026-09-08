package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type InvoiceSetting struct {
	Enabled              bool    `json:"enabled"`
	MinimumAmount        float64 `json:"minimum_amount"`
	Unit                 string  `json:"unit"`
	ProcessingDays       string  `json:"processing_days"`
	RejectionFreezeHours int     `json:"rejection_freeze_hours"`
}

var invoiceSetting = InvoiceSetting{
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
	return invoiceSetting
}
