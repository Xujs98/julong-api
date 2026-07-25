package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func preserveLogRatioDisplaySettings(t *testing.T) {
	t.Helper()
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
}

func newGroupRatioTestLog() *Log {
	return &Log{
		Type:             LogTypeConsume,
		Group:            "codex-v1",
		Quota:            9500,
		PromptTokens:     950,
		CompletionTokens: 190,
		Other: common.MapToJsonStr(map[string]interface{}{
			"model_price":           0.004,
			"group_ratio":           0.06,
			"user_group_ratio":      0.095,
			"fee_quota":             9500,
			"cache_tokens":          95,
			"cache_creation_tokens": 190,
		}),
	}
}

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

func TestSnapshotUserDisplayGroupRatioDoesNotModifyBillingFields(t *testing.T) {
	preserveLogRatioDisplaySettings(t)
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"codex-v1":0.06}`))

	tests := []struct {
		name     string
		mode     string
		manual   float64
		expected float64
	}{
		{name: "system uses actual request ratio", mode: common.UserLogGroupRatioDisplayModeSystem, expected: 0.095},
		{name: "pricing group uses current base ratio", mode: common.UserLogGroupRatioDisplayModePricingGroup, expected: 0.06},
		{name: "manual uses configured ratio", mode: common.UserLogGroupRatioDisplayModeManual, manual: 0.01, expected: 0.01},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			common.UserLogGroupRatioDisplayMode = test.mode
			common.UserLogGroupRatioManualValue = test.manual
			log := newGroupRatioTestLog()

			snapshotUserDisplayGroupRatio(log)

			require.NotNil(t, log.UserDisplayGroupRatio)
			assert.InDelta(t, test.expected, *log.UserDisplayGroupRatio, 1e-12)
			assert.Equal(t, 9500, log.Quota)
			assert.Equal(t, 950, log.PromptTokens)
			assert.Equal(t, 190, log.CompletionTokens)
			actual, err := common.StrToMap(log.Other)
			require.NoError(t, err)
			assert.Equal(t, 0.06, actual["group_ratio"])
			assert.Equal(t, 0.095, actual["user_group_ratio"])
			assert.Equal(t, 0.004, actual["model_price"])
		})
	}
}

func TestCreateLogPersistsDisplayRatioSnapshotWithoutChangingCharge(t *testing.T) {
	preserveLogRatioDisplaySettings(t)
	common.UserLogGroupRatioDisplayEnabled = false
	common.UserLogGroupRatioDisplayMode = common.UserLogGroupRatioDisplayModeManual
	common.UserLogGroupRatioManualValue = 0.01

	log := newGroupRatioTestLog()
	log.UserId = 910001
	log.Username = "display-ratio-snapshot-test"
	log.CreatedAt = common.GetTimestamp()
	log.RequestId = common.NewRequestId()
	require.NoError(t, createLog(log))
	t.Cleanup(func() {
		require.NoError(t, LOG_DB.Where("request_id = ?", log.RequestId).Delete(&Log{}).Error)
	})

	var persisted Log
	require.NoError(t, LOG_DB.Where("request_id = ?", log.RequestId).First(&persisted).Error)
	require.NotNil(t, persisted.UserDisplayGroupRatio)
	assert.InDelta(t, 0.01, *persisted.UserDisplayGroupRatio, 1e-12)
	assert.Equal(t, 9500, persisted.Quota)
	assert.Equal(t, 950, persisted.PromptTokens)
	assert.Equal(t, 190, persisted.CompletionTokens)
	persistedOther, err := common.StrToMap(persisted.Other)
	require.NoError(t, err)
	assert.Equal(t, 0.06, persistedOther["group_ratio"])
	assert.Equal(t, 0.095, persistedOther["user_group_ratio"])
	userDisplayQuota, userDisplayTokenUsed := userVisibleLogDataMetrics(&persisted)
	assert.Equal(t, 1000, userDisplayQuota)
	assert.Equal(t, 120, userDisplayTokenUsed)
	assert.Equal(t, 9500, persisted.Quota)
	assert.Equal(t, 950, persisted.PromptTokens)
	assert.Equal(t, 190, persisted.CompletionTokens)

	common.UserLogGroupRatioManualValue = 0.02
	hiddenLogs, _, err := GetUserLogs(log.UserId, LogTypeConsume, 0, 0, "", "", 0, 10, "", log.RequestId, "")
	require.NoError(t, err)
	require.Len(t, hiddenLogs, 1)
	assert.Nil(t, hiddenLogs[0].UserDisplayGroupRatio)
	hiddenOther, err := common.StrToMap(hiddenLogs[0].Other)
	require.NoError(t, err)
	assert.NotContains(t, hiddenOther, "group_ratio")
	assert.NotContains(t, hiddenOther, "user_group_ratio")
	assert.Equal(t, 1000, hiddenLogs[0].Quota)
	assert.Equal(t, 100, hiddenLogs[0].PromptTokens)
	assert.Equal(t, 20, hiddenLogs[0].CompletionTokens)
	assert.Equal(t, 10.0, hiddenOther["cache_tokens"])
	assert.Equal(t, 20.0, hiddenOther["cache_creation_tokens"])
	assert.Equal(t, 1000.0, hiddenOther["fee_quota"])

	common.UserLogGroupRatioDisplayEnabled = true
	visibleLogs, _, err := GetUserLogs(log.UserId, LogTypeConsume, 0, 0, "", "", 0, 10, "", log.RequestId, "")
	require.NoError(t, err)
	require.Len(t, visibleLogs, 1)
	require.NotNil(t, visibleLogs[0].UserDisplayGroupRatio)
	assert.InDelta(t, 0.01, *visibleLogs[0].UserDisplayGroupRatio, 1e-12)
	visibleOther, err := common.StrToMap(visibleLogs[0].Other)
	require.NoError(t, err)
	assert.Equal(t, 0.01, visibleOther["group_ratio"])
	assert.Equal(t, "snapshot", visibleOther["group_ratio_display_mode"])
	assert.NotContains(t, visibleOther, "user_group_ratio")
	assert.Equal(t, 1000, visibleLogs[0].Quota)
	assert.Equal(t, 100, visibleLogs[0].PromptTokens)
	assert.Equal(t, 20, visibleLogs[0].CompletionTokens)
	assert.Equal(t, 10.0, visibleOther["cache_tokens"])
	assert.Equal(t, 20.0, visibleOther["cache_creation_tokens"])
	assert.Equal(t, 1000.0, visibleOther["fee_quota"])

	var unchanged Log
	require.NoError(t, LOG_DB.Where("request_id = ?", log.RequestId).First(&unchanged).Error)
	require.NotNil(t, unchanged.UserDisplayGroupRatio)
	assert.InDelta(t, 0.01, *unchanged.UserDisplayGroupRatio, 1e-12)
	assert.Equal(t, 9500, unchanged.Quota)
	assert.Equal(t, 950, unchanged.PromptTokens)
	assert.Equal(t, 190, unchanged.CompletionTokens)
	unchangedOther, err := common.StrToMap(unchanged.Other)
	require.NoError(t, err)
	assert.Equal(t, 95.0, unchangedOther["cache_tokens"])
	assert.Equal(t, 190.0, unchangedOther["cache_creation_tokens"])
	assert.Equal(t, 9500.0, unchangedOther["fee_quota"])

	stats, err := SumUserVisibleUsedQuota(LogTypeConsume, 0, 0, "", log.Username, "", 0, "")
	require.NoError(t, err)
	assert.Equal(t, 9500, stats.Quota)
	assert.Equal(t, 1, stats.Rpm)
	assert.Equal(t, 120, stats.Tpm)

	actualStats, err := SumUsedQuota(LogTypeConsume, 0, 0, "", log.Username, "", 0, "")
	require.NoError(t, err)
	assert.Equal(t, 9500, actualStats.Quota)
	assert.Equal(t, 1, actualStats.Rpm)
	assert.Equal(t, 1140, actualStats.Tpm)
}

func TestPricingGroupSnapshotDoesNotFollowLaterRatioChanges(t *testing.T) {
	preserveLogRatioDisplaySettings(t)
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"codex-v1":0.06}`))
	common.UserLogGroupRatioDisplayEnabled = true
	common.UserLogGroupRatioDisplayMode = common.UserLogGroupRatioDisplayModePricingGroup
	log := newGroupRatioTestLog()
	snapshotUserDisplayGroupRatio(log)
	require.NotNil(t, log.UserDisplayGroupRatio)

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"codex-v1":0.08}`))
	formatUserLogs([]*Log{log}, 0)

	require.NotNil(t, log.UserDisplayGroupRatio)
	assert.InDelta(t, 0.06, *log.UserDisplayGroupRatio, 1e-12)
	values, err := common.StrToMap(log.Other)
	require.NoError(t, err)
	assert.Equal(t, 0.06, values["group_ratio"])
	assert.Equal(t, 6000, log.Quota)
	assert.Equal(t, 600, log.PromptTokens)
	assert.Equal(t, 120, log.CompletionTokens)
	assert.Equal(t, 60.0, values["cache_tokens"])
	assert.Equal(t, 120.0, values["cache_creation_tokens"])
	assert.Equal(t, 6000.0, values["fee_quota"])
}

func TestLegacyLogFallsBackToStoredActualRatio(t *testing.T) {
	preserveLogRatioDisplaySettings(t)
	common.UserLogGroupRatioDisplayEnabled = true
	common.UserLogGroupRatioDisplayMode = common.UserLogGroupRatioDisplayModeManual
	common.UserLogGroupRatioManualValue = 0.02
	log := newGroupRatioTestLog()

	formatUserLogs([]*Log{log}, 0)

	require.NotNil(t, log.UserDisplayGroupRatio)
	assert.InDelta(t, 0.095, *log.UserDisplayGroupRatio, 1e-12)
	values, err := common.StrToMap(log.Other)
	require.NoError(t, err)
	assert.Equal(t, 0.095, values["group_ratio"])
	assert.Equal(t, 9500, log.Quota)
	assert.Equal(t, 950, log.PromptTokens)
	assert.Equal(t, 190, log.CompletionTokens)
}

func TestManualDisplayMetricsScaleFromPlainOneRatio(t *testing.T) {
	preserveLogRatioDisplaySettings(t)
	common.UserLogGroupRatioDisplayEnabled = true
	common.UserLogGroupRatioDisplayMode = common.UserLogGroupRatioDisplayModeManual
	common.UserLogGroupRatioManualValue = 0.5
	log := &Log{
		Type:             LogTypeConsume,
		Quota:            1000,
		PromptTokens:     100,
		CompletionTokens: 50,
		Other: common.MapToJsonStr(map[string]interface{}{
			"group_ratio": 1,
		}),
	}
	snapshotUserDisplayGroupRatio(log)
	require.NotNil(t, log.UserDisplayGroupRatio)

	formatUserLogs([]*Log{log}, 0)

	assert.Equal(t, 500, log.Quota)
	assert.Equal(t, 50, log.PromptTokens)
	assert.Equal(t, 25, log.CompletionTokens)
	assert.InDelta(t, 0.5, *log.UserDisplayGroupRatio, 1e-12)
}

func TestFormatAdminLogsIncludeActualAndPersistedUserDisplayedRatios(t *testing.T) {
	preserveLogRatioDisplaySettings(t)
	common.UserLogGroupRatioDisplayEnabled = true
	common.UserLogGroupRatioDisplayMode = common.UserLogGroupRatioDisplayModeManual
	common.UserLogGroupRatioManualValue = 0.02
	displayRatio := 0.01
	log := newGroupRatioTestLog()
	log.UserDisplayGroupRatio = &displayRatio

	formatAdminLogGroupRatioDisplay([]*Log{log})

	values, err := common.StrToMap(log.Other)
	require.NoError(t, err)
	assert.Equal(t, 0.06, values["group_ratio"])
	assert.Equal(t, 0.095, values["user_group_ratio"])
	assert.Equal(t, true, values["user_group_ratio_display_enabled"])
	assert.Equal(t, 0.01, values["user_group_ratio_display_value"])
	require.NotNil(t, log.UserDisplayGroupRatio)
	assert.InDelta(t, 0.01, *log.UserDisplayGroupRatio, 1e-12)
	assert.Equal(t, 9500, log.Quota)
	assert.Equal(t, 950, log.PromptTokens)
	assert.Equal(t, 190, log.CompletionTokens)
}
