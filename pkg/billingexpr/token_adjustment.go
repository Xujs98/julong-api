package billingexpr

import "github.com/QuantumNous/new-api/types"

func ApplyModelTokenAdjustment(params TokenParams, adjustment types.ModelTokenAdjustment) TokenParams {
	params.P *= adjustment.InputMultiplier()
	params.C *= adjustment.OutputMultiplier()
	params.CR *= adjustment.CacheReadMultiplier()
	params.CC *= adjustment.CacheCreationMultiplier()
	params.CC1h *= adjustment.CacheCreationMultiplier()
	return params
}
