package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserRiskReportAggregatesCurrentUserSignals(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	userId := 41
	logs := make([]Log, 0)
	for index := 0; index < 3; index++ {
		logs = append(logs, Log{
			UserId:    userId,
			CreatedAt: now - int64(index),
			Type:      LogTypeError,
			Other: common.MapToJsonStr(map[string]interface{}{
				"error_code": "sensitive_words_detected",
			}),
		})
	}
	for index := 0; index < 4; index++ {
		logs = append(logs, Log{
			UserId:    userId,
			CreatedAt: now - 20 - int64(index),
			Type:      LogTypeConsume,
			IsStream:  true,
			Other: common.MapToJsonStr(map[string]interface{}{
				"stream_status": map[string]interface{}{
					"status":     "error",
					"end_reason": "client_gone",
				},
			}),
		})
	}
	for index := 0; index < 5; index++ {
		logs = append(logs, Log{
			UserId:    userId,
			CreatedAt: now - 40 - int64(index),
			Type:      LogTypeQuotaIncrease,
			Quota:     100,
			Content:   "请求失败，返还预扣额度",
			Other: common.MapToJsonStr(map[string]interface{}{
				"source": QuotaIncreaseSourceRefund,
			}),
		})
	}
	for index := 0; index < 5; index++ {
		logs = append(logs, Log{
			UserId:    userId,
			CreatedAt: now - 60 - int64(index),
			Type:      LogTypeLogin,
			Ip:        fmt.Sprintf("192.0.2.%d", index+1),
		})
	}
	logs = append(logs,
		Log{UserId: 999, CreatedAt: now, Type: LogTypeError, Other: `{"error_code":"sensitive_words_detected"}`},
		Log{UserId: userId, CreatedAt: now - 8*24*60*60, Type: LogTypeError, Other: `{"error_code":"sensitive_words_detected"}`},
	)
	require.NoError(t, LOG_DB.Create(&logs).Error)

	report, err := GetUserRiskReport(userId, 7, now)
	require.NoError(t, err)
	assert.Equal(t, UserRiskLevelHigh, report.Level)
	assert.Equal(t, 92, report.Score)
	assert.EqualValues(t, 7, report.Summary.TotalRequests)
	assert.EqualValues(t, 3, report.Summary.ErrorCount)
	assert.InDelta(t, 3.0/7.0, report.Summary.ErrorRate, 0.0001)
	assert.EqualValues(t, 5, report.Summary.RefundCount)
	assert.EqualValues(t, 500, report.Summary.RefundQuota)
	assert.EqualValues(t, 5, report.Summary.FailedRefundCount)
	assert.EqualValues(t, 3, report.Summary.SensitiveWordAttempts)
	assert.EqualValues(t, 4, report.Summary.ClientAbortCount)
	assert.EqualValues(t, 4, report.Summary.AbnormalStreamCount)
	assert.EqualValues(t, 5, report.Summary.UniqueIPCount)
	assert.Len(t, report.Signals, 4)
}

func TestGetUserRiskReportDoesNotFlagNormalSettlementRefunds(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	logs := []Log{
		{
			UserId:    52,
			CreatedAt: now,
			Type:      LogTypeQuotaIncrease,
			Quota:     300,
			Content:   "预扣额度返还",
			Other: common.MapToJsonStr(map[string]interface{}{
				"source": QuotaIncreaseSourceRefund,
			}),
		},
	}
	require.NoError(t, LOG_DB.Create(&logs).Error)

	report, err := GetUserRiskReport(52, 1, now)
	require.NoError(t, err)
	assert.Equal(t, UserRiskLevelLow, report.Level)
	assert.Zero(t, report.Score)
	assert.EqualValues(t, 1, report.Summary.RefundCount)
	assert.Zero(t, report.Summary.FailedRefundCount)
	assert.Empty(t, report.Signals)
}

func TestGetUserRiskReportFlagsFailedRefundsAfterResponseOutput(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix() + 1
	userId := 53
	for index := 0; index < 3; index++ {
		RecordQuotaIncreaseLogWithAudit(
			userId,
			100,
			QuotaIncreaseSourceRefund,
			"audited refund",
			fmt.Sprintf("request-%d", index),
			map[string]interface{}{
				"refund_after_output": true,
				"response_bytes":      256,
			},
		)
	}

	report, err := GetUserRiskReport(userId, 1, now)
	require.NoError(t, err)
	assert.Equal(t, UserRiskLevelHigh, report.Level)
	assert.Equal(t, 60, report.Score)
	assert.EqualValues(t, 3, report.Summary.RefundAfterOutputCount)
	require.Len(t, report.Signals, 1)
	assert.Equal(t, UserRiskSignalRefundAfterOutput, report.Signals[0].Code)
	assert.Equal(t, UserRiskLevelHigh, report.Signals[0].Severity)

	var logs []Log
	require.NoError(t, LOG_DB.Where("user_id = ?", userId).Find(&logs).Error)
	require.Len(t, logs, 3)
	assert.Contains(t, logs[0].Other, `"refund_after_output":true`)
	assert.NotEmpty(t, logs[0].RequestId)
}

func TestGetUserRiskReportCountsDisconnectedFailedRefunds(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix() + 1
	userId := 54
	for index := 0; index < 3; index++ {
		RecordQuotaIncreaseLogWithAudit(
			userId,
			100,
			QuotaIncreaseSourceRefund,
			"请求失败，返还预扣额度",
			fmt.Sprintf("disconnected-request-%d", index),
			map[string]interface{}{
				"client_disconnected": true,
				"response_bytes":      0,
			},
		)
	}

	report, err := GetUserRiskReport(userId, 1, now)
	require.NoError(t, err)
	assert.EqualValues(t, 3, report.Summary.ClientAbortCount)
	assert.EqualValues(t, 3, report.Summary.FailedRefundCount)
	require.Len(t, report.Signals, 2)
	assert.Equal(t, UserRiskSignalClientAbort, report.Signals[0].Code)
	assert.Equal(t, UserRiskSignalFailedRefund, report.Signals[1].Code)
}

func TestSetUserRiskDetectionEnabled(t *testing.T) {
	truncateTables(t)
	user := User{
		Username: "risk-detection-setting",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, DB.Create(&user).Error)

	require.NoError(t, SetUserRiskDetectionEnabled(user.Id, true))
	stored, err := GetUserById(user.Id, false)
	require.NoError(t, err)
	assert.True(t, stored.RiskDetectionEnabled)

	require.NoError(t, SetUserRiskDetectionEnabled(user.Id, false))
	stored, err = GetUserById(user.Id, false)
	require.NoError(t, err)
	assert.False(t, stored.RiskDetectionEnabled)
}
