package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/stretchr/testify/require"
)

// TestFormatUserLogsStripsQuotaSaturation verifies the admin-only quota
// saturation marker (nested under other.admin_info) is removed for non-admin
// log views, since formatUserLogs strips the whole admin_info object.
func TestFormatUserLogsStripsQuotaSaturation(t *testing.T) {
	other := common.MapToJsonStr(map[string]interface{}{
		"model_price": 0.004,
		"admin_info": map[string]interface{}{
			"quota_saturation": map[string]interface{}{
				"op":      "QuotaFromDecimal",
				"kind":    "overflow",
				"clamped": common.MaxQuota,
			},
		},
	})
	logs := []*Log{{Other: other}}

	formatUserLogs(logs, 0)

	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	_, hasAdminInfo := parsed["admin_info"]
	require.False(t, hasAdminInfo, "admin_info (and nested quota_saturation) must be stripped for non-admin views")
	// Non-admin billing fields remain visible.
	require.Contains(t, parsed, "model_price")
}

func TestFormatUserLogsGroupRatioVisibilitySetting(t *testing.T) {
	previousEnabled := common.UserLogGroupRatioDisplayEnabled
	previousMode := common.UserLogGroupRatioDisplayMode
	previousManualValue := common.UserLogGroupRatioManualValue
	previousGroupRatios := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		common.UserLogGroupRatioDisplayEnabled = previousEnabled
		common.UserLogGroupRatioDisplayMode = previousMode
		common.UserLogGroupRatioManualValue = previousManualValue
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroupRatios))
	})
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"codex-v1":0.06}`))

	newLog := func() *Log {
		return &Log{Group: "codex-v1", Other: common.MapToJsonStr(map[string]interface{}{
			"model_price":      0.004,
			"group_ratio":      0.095,
			"user_group_ratio": 0.095,
		})}
	}

	common.UserLogGroupRatioDisplayEnabled = false
	common.UserLogGroupRatioDisplayMode = common.UserLogGroupRatioDisplayModeSystem
	hiddenLog := newLog()
	formatUserLogs([]*Log{hiddenLog}, 0)
	hidden, err := common.StrToMap(hiddenLog.Other)
	require.NoError(t, err)
	require.NotContains(t, hidden, "group_ratio")
	require.NotContains(t, hidden, "user_group_ratio")
	require.Contains(t, hidden, "model_price")

	common.UserLogGroupRatioDisplayEnabled = true
	systemLog := newLog()
	formatUserLogs([]*Log{systemLog}, 0)
	systemValues, err := common.StrToMap(systemLog.Other)
	require.NoError(t, err)
	require.Equal(t, 0.095, systemValues["group_ratio"])
	require.Equal(t, 0.095, systemValues["user_group_ratio"])
	require.NotContains(t, systemValues, "group_ratio_display_mode")

	common.UserLogGroupRatioDisplayMode = common.UserLogGroupRatioDisplayModePricingGroup
	pricingGroupLog := newLog()
	formatUserLogs([]*Log{pricingGroupLog}, 0)
	pricingGroupValues, err := common.StrToMap(pricingGroupLog.Other)
	require.NoError(t, err)
	require.Equal(t, 0.06, pricingGroupValues["group_ratio"])
	require.NotContains(t, pricingGroupValues, "user_group_ratio")
	require.Equal(
		t,
		common.UserLogGroupRatioDisplayModePricingGroup,
		pricingGroupValues["group_ratio_display_mode"],
	)

	common.UserLogGroupRatioDisplayMode = common.UserLogGroupRatioDisplayModeManual
	common.UserLogGroupRatioManualValue = 1
	manualLog := newLog()
	formatUserLogs([]*Log{manualLog}, 0)
	manualValues, err := common.StrToMap(manualLog.Other)
	require.NoError(t, err)
	require.Equal(t, float64(1), manualValues["group_ratio"])
	require.NotContains(t, manualValues, "user_group_ratio")
	require.Equal(
		t,
		common.UserLogGroupRatioDisplayModeManual,
		manualValues["group_ratio_display_mode"],
	)

	nonBillingLog := &Log{Group: "codex-v1", Other: common.MapToJsonStr(map[string]interface{}{
		"model_price": 0.004,
	})}
	formatUserLogs([]*Log{nonBillingLog}, 0)
	nonBillingValues, err := common.StrToMap(nonBillingLog.Other)
	require.NoError(t, err)
	require.NotContains(t, nonBillingValues, "group_ratio")
	require.NotContains(t, nonBillingValues, "group_ratio_display_mode")
}

func TestFormatAdminLogsIncludeActualAndUserDisplayedRatios(t *testing.T) {
	previousEnabled := common.UserLogGroupRatioDisplayEnabled
	previousMode := common.UserLogGroupRatioDisplayMode
	previousGroupRatios := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		common.UserLogGroupRatioDisplayEnabled = previousEnabled
		common.UserLogGroupRatioDisplayMode = previousMode
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroupRatios))
	})
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"codex-v1":0.06}`))

	newLog := func() *Log {
		return &Log{Group: "codex-v1", Other: common.MapToJsonStr(map[string]interface{}{
			"group_ratio":      0.06,
			"user_group_ratio": 0.095,
		})}
	}

	common.UserLogGroupRatioDisplayEnabled = true
	common.UserLogGroupRatioDisplayMode = common.UserLogGroupRatioDisplayModePricingGroup
	visibleLog := newLog()
	formatAdminLogGroupRatioDisplay([]*Log{visibleLog})
	visibleValues, err := common.StrToMap(visibleLog.Other)
	require.NoError(t, err)
	require.Equal(t, 0.06, visibleValues["group_ratio"])
	require.Equal(t, 0.095, visibleValues["user_group_ratio"])
	require.Equal(t, true, visibleValues["user_group_ratio_display_enabled"])
	require.Equal(t, common.UserLogGroupRatioDisplayModePricingGroup, visibleValues["user_group_ratio_display_mode"])
	require.Equal(t, 0.06, visibleValues["user_group_ratio_display_value"])

	common.UserLogGroupRatioDisplayEnabled = false
	hiddenLog := newLog()
	formatAdminLogGroupRatioDisplay([]*Log{hiddenLog})
	hiddenValues, err := common.StrToMap(hiddenLog.Other)
	require.NoError(t, err)
	require.Equal(t, 0.06, hiddenValues["group_ratio"])
	require.Equal(t, 0.095, hiddenValues["user_group_ratio"])
	require.Equal(t, false, hiddenValues["user_group_ratio_display_enabled"])
	require.NotContains(t, hiddenValues, "user_group_ratio_display_value")
}
