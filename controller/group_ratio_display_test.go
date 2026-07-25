package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/stretchr/testify/require"
)

func TestDisplayedTokenGroupRatioUsesConfiguredMode(t *testing.T) {
	previousMode := common.TokenGroupRatioDisplayMode
	previousGroupRatios := ratio_setting.GroupRatio2JSONString()
	previousOverrides := ratio_setting.GroupGroupRatio2JSONString()
	t.Cleanup(func() {
		common.TokenGroupRatioDisplayMode = previousMode
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroupRatios))
		require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(previousOverrides))
	})
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"codex-v1":1}`))
	require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(`{"codex-pro":{"codex-v1":2}}`))

	common.TokenGroupRatioDisplayMode = common.TokenGroupRatioDisplayModeActual
	require.Equal(t, float64(2), displayedTokenGroupRatio("codex-pro", "codex-v1"))

	common.TokenGroupRatioDisplayMode = common.TokenGroupRatioDisplayModePricingGroup
	require.Equal(t, float64(1), displayedTokenGroupRatio("codex-pro", "codex-v1"))
}
