package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
)

var maxLedgerEstimateRatio = decimal.NewFromInt(1000)

type LedgerSettings struct {
	EstimateRatio decimal.Decimal `json:"estimate_ratio"`
}

func GetLedgerSettings() *LedgerSettings {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[common.LedgerEstimateRatioOptionKey]
	common.OptionMapRWMutex.RUnlock()

	ratio, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil || ratio.LessThanOrEqual(decimal.Zero) || ratio.GreaterThan(maxLedgerEstimateRatio) {
		ratio = decimal.NewFromInt(1)
	}
	return &LedgerSettings{EstimateRatio: ratio.Round(6)}
}

func UpdateLedgerSettings(estimateRatio decimal.Decimal) (*LedgerSettings, error) {
	estimateRatio = estimateRatio.Round(6)
	if estimateRatio.LessThanOrEqual(decimal.Zero) || estimateRatio.GreaterThan(maxLedgerEstimateRatio) {
		return nil, errors.New("ledger estimate ratio must be greater than 0 and no more than 1000")
	}
	if err := UpdateOption(common.LedgerEstimateRatioOptionKey, estimateRatio.String()); err != nil {
		return nil, err
	}
	return &LedgerSettings{EstimateRatio: estimateRatio}, nil
}
