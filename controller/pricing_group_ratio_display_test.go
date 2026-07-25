package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/stretchr/testify/require"
)

func TestPricingGroupRatiosForUserDisplayMode(t *testing.T) {
	previousMode := common.ModelSquareGroupRatioDisplayMode
	previousGroupRatios := ratio_setting.GroupRatio2JSONString()
	previousOverrides := ratio_setting.GroupGroupRatio2JSONString()
	t.Cleanup(func() {
		common.ModelSquareGroupRatioDisplayMode = previousMode
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroupRatios))
		require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(previousOverrides))
	})
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"codex-v1":1}`))
	require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(`{"codex-pro":{"codex-v1":2}}`))

	common.ModelSquareGroupRatioDisplayMode = common.ModelSquareGroupRatioDisplayModeActual
	require.Equal(t, float64(2), pricingGroupRatiosForUser("codex-pro")["codex-v1"])

	common.ModelSquareGroupRatioDisplayMode = common.ModelSquareGroupRatioDisplayModePricingGroup
	require.Equal(t, float64(1), pricingGroupRatiosForUser("codex-pro")["codex-v1"])
}
