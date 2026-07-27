package model

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyUserGroupRatioAdjustment(t *testing.T) {
	tests := []struct {
		name       string
		baseRatio  float64
		enabled    bool
		adjustment float64
		expected   float64
		wantError  bool
	}{
		{name: "disabled returns base ratio", baseRatio: 0.08, enabled: false, adjustment: 0.1, expected: 0.08},
		{name: "increase applies after base ratio", baseRatio: 0.08, enabled: true, adjustment: 0.1, expected: 0.18},
		{name: "decrease applies after base ratio", baseRatio: 0.12, enabled: true, adjustment: -0.04, expected: 0.08},
		{name: "zero final ratio remains valid", baseRatio: 0.08, enabled: true, adjustment: -0.08, expected: 0},
		{name: "negative final ratio is rejected", baseRatio: 0.08, enabled: true, adjustment: -0.1, wantError: true},
		{name: "non finite adjustment is rejected", baseRatio: 1, enabled: true, adjustment: math.Inf(1), wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := ApplyUserGroupRatioAdjustment(test.baseRatio, test.enabled, test.adjustment)
			if test.wantError {
				require.ErrorIs(t, err, ErrInvalidUserGroupRatioAdjustment)
				return
			}
			require.NoError(t, err)
			assert.InDelta(t, test.expected, actual, 1e-12)
		})
	}
}
