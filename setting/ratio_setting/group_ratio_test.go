package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckModelTokenRatioValidatesEnabledAdjustments(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "accepts optional token categories", value: `{"default":{"codex-v1":{"gpt-test":{"input":0.1,"cache_read":0}}}}`},
		{name: "accepts a user group without rules", value: `{"default":{}}`},
		{name: "rejects a rule without enabled categories", value: `{"default":{"codex-v1":{"gpt-test":{}}}}`, wantErr: true},
		{name: "rejects a negative adjustment", value: `{"default":{"codex-v1":{"gpt-test":{"output":-0.1}}}}`, wantErr: true},
		{name: "rejects an excessive adjustment", value: `{"default":{"codex-v1":{"gpt-test":{"cache_creation":101}}}}`, wantErr: true},
		{name: "rejects surrounding user group whitespace", value: `{" default":{"codex-v1":{"gpt-test":{"input":0.1}}}}`, wantErr: true},
		{name: "rejects surrounding billing group whitespace", value: `{"default":{" codex-v1":{"gpt-test":{"input":0.1}}}}`, wantErr: true},
		{name: "rejects surrounding model whitespace", value: `{"default":{"codex-v1":{" gpt-test":{"input":0.1}}}}`, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := CheckModelTokenRatio(test.value)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGetModelTokenAdjustmentReturnsConfiguredRule(t *testing.T) {
	previous := ModelTokenRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelTokenRatioByJSONString(previous))
	})

	require.NoError(t, UpdateModelTokenRatioByJSONString(`{"default":{"codex-v1":{"gpt-test":{"input":0.1,"output":0.2}}}}`))

	adjustment, ok := GetModelTokenAdjustment("default", "codex-v1", "gpt-test")
	require.True(t, ok)
	require.NotNil(t, adjustment.Input)
	require.NotNil(t, adjustment.Output)
	assert.InDelta(t, 1.1, adjustment.InputMultiplier(), 1e-12)
	assert.InDelta(t, 1.2, adjustment.OutputMultiplier(), 1e-12)

	_, ok = GetModelTokenAdjustment("default", "other", "gpt-test")
	assert.False(t, ok)
	_, ok = GetModelTokenAdjustment("vip", "codex-v1", "gpt-test")
	assert.False(t, ok)
}
