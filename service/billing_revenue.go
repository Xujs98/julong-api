package service

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/shopspring/decimal"
)

type billingRevenueCounterfactuals struct {
	WithoutGroupSpecialRatio    *int
	WithoutModelTokenAdjustment *int
	OriginalQuota               *int
}

func calculateAudioBillingRevenueCounterfactuals(relayInfo *relaycommon.RelayInfo, quotaInfo QuotaInfo) billingRevenueCounterfactuals {
	counterfactuals := billingRevenueCounterfactuals{}
	groupRatioInfo := relayInfo.PriceData.GroupRatioInfo
	hasGroupSpecialRatio := groupRatioInfo.HasSpecialRatio && groupRatioInfo.PricingGroupRatio >= 0
	hasModelTokenAdjustment := !quotaInfo.UsePrice && quotaInfo.ModelTokenAdjustment.HasAny()
	if hasGroupSpecialRatio {
		withoutSpecialRatio := quotaInfo
		withoutSpecialRatio.GroupRatio = groupRatioInfo.PricingGroupRatio
		quota, _ := calculateAudioQuota(withoutSpecialRatio)
		counterfactuals.WithoutGroupSpecialRatio = &quota
	}
	if hasModelTokenAdjustment {
		withoutModelAdjustment := quotaInfo
		withoutModelAdjustment.ModelTokenAdjustment = types.ModelTokenAdjustment{}
		quota, _ := calculateAudioQuota(withoutModelAdjustment)
		counterfactuals.WithoutModelTokenAdjustment = &quota
	}
	if hasGroupSpecialRatio || hasModelTokenAdjustment {
		original := quotaInfo
		if hasGroupSpecialRatio {
			original.GroupRatio = groupRatioInfo.PricingGroupRatio
		}
		if hasModelTokenAdjustment {
			original.ModelTokenAdjustment = types.ModelTokenAdjustment{}
		}
		quota, _ := calculateAudioQuota(original)
		counterfactuals.OriginalQuota = &quota
	}
	return counterfactuals
}

func calculateTieredBillingRevenueCounterfactuals(relayInfo *relaycommon.RelayInfo, params billingexpr.TokenParams, actualResult *billingexpr.TieredResult, surchargeBeforeGroup decimal.Decimal) billingRevenueCounterfactuals {
	counterfactuals := billingRevenueCounterfactuals{}
	if relayInfo == nil || actualResult == nil || relayInfo.TieredBillingSnapshot == nil {
		return counterfactuals
	}

	groupRatioInfo := relayInfo.PriceData.GroupRatioInfo
	hasGroupSpecialRatio := groupRatioInfo.HasSpecialRatio && groupRatioInfo.PricingGroupRatio >= 0
	if hasGroupSpecialRatio {
		quota := tieredQuotaWithSurcharge(actualResult.ActualQuotaBeforeGroup, groupRatioInfo.PricingGroupRatio, surchargeBeforeGroup)
		counterfactuals.WithoutGroupSpecialRatio = &quota
	}

	snapshot := relayInfo.TieredBillingSnapshot
	hasModelTokenAdjustment := snapshot.ModelTokenAdjustment.HasAny()
	originalQuotaBeforeGroup := actualResult.ActualQuotaBeforeGroup
	if hasModelTokenAdjustment {
		requestInput := billingexpr.RequestInput{}
		if relayInfo.BillingRequestInput != nil {
			requestInput = *relayInfo.BillingRequestInput
		}
		withoutModelAdjustment, err := billingexpr.ComputeTieredQuotaWithRequest(snapshot, params, requestInput)
		if err != nil {
			return counterfactuals
		}
		quota := tieredQuotaWithSurcharge(withoutModelAdjustment.ActualQuotaBeforeGroup, snapshot.GroupRatio, surchargeBeforeGroup)
		counterfactuals.WithoutModelTokenAdjustment = &quota
		originalQuotaBeforeGroup = withoutModelAdjustment.ActualQuotaBeforeGroup
	}
	if hasGroupSpecialRatio || hasModelTokenAdjustment {
		originalGroupRatio := snapshot.GroupRatio
		if hasGroupSpecialRatio {
			originalGroupRatio = groupRatioInfo.PricingGroupRatio
		}
		quota := tieredQuotaWithSurcharge(originalQuotaBeforeGroup, originalGroupRatio, surchargeBeforeGroup)
		counterfactuals.OriginalQuota = &quota
	}
	return counterfactuals
}

func tieredQuotaWithSurcharge(quotaBeforeGroup float64, groupRatio float64, surchargeBeforeGroup decimal.Decimal) int {
	quota := decimal.NewFromFloat(quotaBeforeGroup).
		Add(surchargeBeforeGroup).
		Mul(decimal.NewFromFloat(groupRatio))
	return common.QuotaFromDecimal(quota)
}

func buildBillingRevenueAudit(finalQuota int, counterfactuals billingRevenueCounterfactuals) *types.BillingRevenueAudit {
	audit := &types.BillingRevenueAudit{}
	if counterfactuals.OriginalQuota != nil {
		originalQuota := int64(*counterfactuals.OriginalQuota)
		audit.OriginalQuota = &originalQuota
	}
	if counterfactuals.WithoutGroupSpecialRatio != nil {
		revenue := int64(finalQuota) - int64(*counterfactuals.WithoutGroupSpecialRatio)
		audit.GroupSpecialRatio = &revenue
	}
	if counterfactuals.WithoutModelTokenAdjustment != nil {
		revenue := int64(finalQuota) - int64(*counterfactuals.WithoutModelTokenAdjustment)
		audit.ModelTokenAdjustment = &revenue
	}
	if !audit.HasAny() {
		return nil
	}
	return audit
}

func attachBillingRevenueToOther(other map[string]interface{}, audit *types.BillingRevenueAudit) {
	if other == nil || !audit.HasAny() {
		return
	}
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	if !ok || adminInfo == nil {
		adminInfo = map[string]interface{}{}
		other["admin_info"] = adminInfo
	}
	adminInfo["billing_revenue"] = audit
}
