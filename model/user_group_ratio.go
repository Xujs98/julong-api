package model

import (
	"errors"
	"math"
)

var ErrInvalidUserGroupRatioAdjustment = errors.New("adjusted group ratio must be a finite number greater than or equal to 0")

func ApplyUserGroupRatioAdjustment(baseRatio float64, enabled bool, adjustment float64) (float64, error) {
	if math.IsNaN(baseRatio) || math.IsInf(baseRatio, 0) || baseRatio < 0 {
		return 0, ErrInvalidUserGroupRatioAdjustment
	}
	if !enabled {
		return baseRatio, nil
	}
	if math.IsNaN(adjustment) || math.IsInf(adjustment, 0) {
		return 0, ErrInvalidUserGroupRatioAdjustment
	}

	adjustedRatio := baseRatio + adjustment
	if math.IsNaN(adjustedRatio) || math.IsInf(adjustedRatio, 0) || adjustedRatio < 0 {
		return 0, ErrInvalidUserGroupRatioAdjustment
	}
	return adjustedRatio, nil
}
