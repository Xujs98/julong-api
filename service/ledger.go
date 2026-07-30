package service

import (
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/shopspring/decimal"
)

type LedgerQuotaMetric struct {
	Real      int64   `json:"real"`
	Estimated float64 `json:"estimated"`
}

type LedgerOperationalQuotaMetric struct {
	Real      int64            `json:"real"`
	CostRatio *decimal.Decimal `json:"cost_ratio"`
}

type LedgerSummary struct {
	UserQuota          LedgerQuotaMetric            `json:"user_quota"`
	UsageQuota         LedgerQuotaMetric            `json:"usage_quota"`
	DailyOperatingCost decimal.Decimal              `json:"daily_operating_cost"`
	TotalOperatingCost decimal.Decimal              `json:"total_operating_cost"`
	OperationalQuota   LedgerOperationalQuotaMetric `json:"operational_quota"`
	CostRatios         map[string]*decimal.Decimal  `json:"cost_ratios"`
	EstimateRatio      decimal.Decimal              `json:"estimate_ratio"`
	Days               int                          `json:"days"`
	LedgerEntryCount   int64                        `json:"ledger_entry_count"`
	IncludedUserCount  int64                        `json:"included_user_count"`
}

func GetLedgerSummary(startTimestamp int64, endTimestamp int64) (*LedgerSummary, error) {
	userSummary, err := model.GetUserQuotaSummary("", "", nil, nil, nil)
	if err != nil {
		return nil, err
	}
	usageQuota, err := model.GetLedgerUsageQuota(startTimestamp, endTimestamp)
	if err != nil {
		return nil, err
	}
	aggregate, err := model.GetLedgerAggregate(startTimestamp, endTimestamp)
	if err != nil {
		return nil, err
	}
	_, ledgerEntryCount, err := model.GetLedgerEntries(startTimestamp, endTimestamp, 0, 0)
	if err != nil {
		return nil, err
	}

	settings := model.GetLedgerSettings()
	days := ledgerDayCount(startTimestamp, endTimestamp)
	operationalCostRatio := ledgerCostRatio(aggregate.TotalCost, aggregate.TotalQuota)
	costRatios := map[string]*decimal.Decimal{
		"plus": ledgerCostRatio(aggregate.CostByType["plus"], aggregate.QuotaByType["plus"]),
		"pro":  ledgerCostRatio(aggregate.CostByType["pro"], aggregate.QuotaByType["pro"]),
		"k12":  ledgerCostRatio(aggregate.CostByType["k12"], aggregate.QuotaByType["k12"]),
	}
	return &LedgerSummary{
		UserQuota: LedgerQuotaMetric{
			Real:      userSummary.TotalQuota,
			Estimated: ledgerEstimatedQuota(userSummary.TotalQuota, settings.EstimateRatio),
		},
		UsageQuota: LedgerQuotaMetric{
			Real:      usageQuota.Real,
			Estimated: ledgerEstimatedQuota(usageQuota.Real, settings.EstimateRatio),
		},
		DailyOperatingCost: aggregate.TotalCost,
		TotalOperatingCost: aggregate.TotalCost,
		OperationalQuota: LedgerOperationalQuotaMetric{
			Real:      aggregate.TotalQuota,
			CostRatio: operationalCostRatio,
		},
		CostRatios:        costRatios,
		EstimateRatio:     settings.EstimateRatio,
		Days:              days,
		LedgerEntryCount:  ledgerEntryCount,
		IncludedUserCount: userSummary.UserCount,
	}, nil
}

func ledgerEstimatedQuota(real int64, estimateRatio decimal.Decimal) float64 {
	if real <= 0 {
		return 0
	}
	estimated, _ := decimal.NewFromInt(real).Div(estimateRatio).Float64()
	return estimated
}

func ledgerCostRatio(cost decimal.Decimal, quota int64) *decimal.Decimal {
	if quota <= 0 || common.QuotaPerUnit <= 0 {
		return nil
	}
	quotaUSD := decimal.NewFromInt(quota).Div(decimal.NewFromFloat(common.QuotaPerUnit))
	if quotaUSD.IsZero() {
		return nil
	}
	ratio := cost.Div(quotaUSD).Round(6)
	return &ratio
}

func ledgerDayCount(startTimestamp int64, endTimestamp int64) int {
	if startTimestamp <= 0 || endTimestamp <= 0 || endTimestamp < startTimestamp {
		return 1
	}
	start := time.Unix(startTimestamp, 0).In(time.Local)
	end := time.Unix(endTimestamp, 0).In(time.Local)
	day := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.Local)
	days := 0
	for !day.After(endDay) {
		days++
		day = day.AddDate(0, 0, 1)
	}
	return max(days, 1)
}

func NormalizeLedgerType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
