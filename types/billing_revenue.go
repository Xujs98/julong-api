package types

type BillingRevenueAudit struct {
	OriginalQuota        *int64 `json:"original_quota,omitempty"`
	GroupSpecialRatio    *int64 `json:"group_special_ratio,omitempty"`
	ModelTokenAdjustment *int64 `json:"model_token_adjustment,omitempty"`
}

func (a *BillingRevenueAudit) HasAny() bool {
	return a != nil && (a.OriginalQuota != nil || a.GroupSpecialRatio != nil || a.ModelTokenAdjustment != nil)
}
