package types

import "math"

const MaxModelTokenAdjustment = 100

type ModelTokenAdjustment struct {
	Input         *float64 `json:"input,omitempty"`
	Output        *float64 `json:"output,omitempty"`
	CacheRead     *float64 `json:"cache_read,omitempty"`
	CacheCreation *float64 `json:"cache_creation,omitempty"`
}

func (a ModelTokenAdjustment) HasAny() bool {
	return a.Input != nil || a.Output != nil || a.CacheRead != nil || a.CacheCreation != nil
}

func (a ModelTokenAdjustment) InputMultiplier() float64 {
	return modelTokenMultiplier(a.Input)
}

func (a ModelTokenAdjustment) OutputMultiplier() float64 {
	return modelTokenMultiplier(a.Output)
}

func (a ModelTokenAdjustment) CacheReadMultiplier() float64 {
	return modelTokenMultiplier(a.CacheRead)
}

func (a ModelTokenAdjustment) CacheCreationMultiplier() float64 {
	return modelTokenMultiplier(a.CacheCreation)
}

func modelTokenMultiplier(adjustment *float64) float64 {
	if adjustment == nil || math.IsNaN(*adjustment) || math.IsInf(*adjustment, 0) || *adjustment < 0 || *adjustment > MaxModelTokenAdjustment {
		return 1
	}
	return 1 + *adjustment
}

type ModelTokenUsage struct {
	Input           int `json:"input"`
	Output          int `json:"output"`
	CacheRead       int `json:"cache_read,omitempty"`
	CacheCreation   int `json:"cache_creation,omitempty"`
	CacheCreation5m int `json:"cache_creation_5m,omitempty"`
	CacheCreation1h int `json:"cache_creation_1h,omitempty"`
}

type ModelTokenAdjustmentAudit struct {
	Adjustments ModelTokenAdjustment `json:"adjustments"`
	Actual      ModelTokenUsage      `json:"actual"`
	Billed      ModelTokenUsage      `json:"billed"`
}
