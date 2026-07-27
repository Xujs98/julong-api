package types

type BillingRevenueAudit struct {
	GroupSpecialRatio    *int64 `json:"group_special_ratio,omitempty"`
	ModelTokenAdjustment *int64 `json:"model_token_adjustment,omitempty"`
}

func (a *BillingRevenueAudit) HasAny() bool {
	return a != nil && (a.GroupSpecialRatio != nil || a.ModelTokenAdjustment != nil)
}
