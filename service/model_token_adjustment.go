package service

import (
	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
)

type modelTokenCategoryUsage struct {
	LogInput                       int
	LogOutput                      int
	Input                          float64
	BilledInput                    float64
	Output                         float64
	BilledOutput                   float64
	CacheRead                      float64
	BilledCacheRead                float64
	CacheCreation                  float64
	BilledCacheCreation            float64
	CacheCreation5m                float64
	BilledCacheCreation5m          float64
	CacheCreation1h                float64
	BilledCacheCreation1h          float64
	IncludeCacheReadInLogInput     bool
	IncludeCacheCreationInLogInput bool
}

func buildModelTokenAdjustmentAudit(relayInfo *relaycommon.RelayInfo, adjustment types.ModelTokenAdjustment, usage modelTokenCategoryUsage) *types.ModelTokenAdjustmentAudit {
	if !adjustment.HasAny() {
		return nil
	}

	billedInput := float64(usage.LogInput) + usage.BilledInput - usage.Input
	if usage.IncludeCacheReadInLogInput {
		billedInput += usage.BilledCacheRead - usage.CacheRead
	}
	if usage.IncludeCacheCreationInLogInput {
		billedInput += usage.BilledCacheCreation - usage.CacheCreation
	}
	billedOutput := float64(usage.LogOutput) + usage.BilledOutput - usage.Output

	return &types.ModelTokenAdjustmentAudit{
		Adjustments: adjustment,
		Actual: types.ModelTokenUsage{
			Input:           usage.LogInput,
			Output:          usage.LogOutput,
			CacheRead:       roundModelTokenCount(relayInfo, usage.CacheRead),
			CacheCreation:   roundModelTokenCount(relayInfo, usage.CacheCreation),
			CacheCreation5m: roundModelTokenCount(relayInfo, usage.CacheCreation5m),
			CacheCreation1h: roundModelTokenCount(relayInfo, usage.CacheCreation1h),
		},
		Billed: types.ModelTokenUsage{
			Input:           roundModelTokenCount(relayInfo, billedInput),
			Output:          roundModelTokenCount(relayInfo, billedOutput),
			CacheRead:       roundModelTokenCount(relayInfo, usage.BilledCacheRead),
			CacheCreation:   roundModelTokenCount(relayInfo, usage.BilledCacheCreation),
			CacheCreation5m: roundModelTokenCount(relayInfo, usage.BilledCacheCreation5m),
			CacheCreation1h: roundModelTokenCount(relayInfo, usage.BilledCacheCreation1h),
		},
	}
}

func roundModelTokenCount(relayInfo *relaycommon.RelayInfo, value float64) int {
	tokens, clamp := common.QuotaRoundChecked(value)
	noteQuotaClamp(relayInfo, clamp)
	return tokens
}

func modelTokenMultiplierForVariable(multiplier float64, used bool) float64 {
	if !used {
		return 1
	}
	return multiplier
}

func attachModelTokenAdjustmentToOther(other map[string]interface{}, audit *types.ModelTokenAdjustmentAudit) {
	if other == nil || audit == nil {
		return
	}
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	if !ok || adminInfo == nil {
		adminInfo = map[string]interface{}{}
		other["admin_info"] = adminInfo
	}
	adminInfo["model_token_adjustment"] = audit
}
