package model

import (
	"errors"

	"gorm.io/gorm"
)

const (
	UserRiskLevelLow    = "low"
	UserRiskLevelMedium = "medium"
	UserRiskLevelHigh   = "high"
)

const (
	UserRiskSignalSensitiveWords    = "sensitive_word_attempts"
	UserRiskSignalErrorRate         = "failed_request_rate"
	UserRiskSignalClientAbort       = "client_abort"
	UserRiskSignalAbnormalStream    = "abnormal_stream"
	UserRiskSignalFailedRefund      = "failed_refunds"
	UserRiskSignalRefundAfterOutput = "refund_after_output"
	UserRiskSignalMultipleIPs       = "multiple_ips"
)

type UserRiskSignal struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Score    int    `json:"score"`
	Count    int64  `json:"count"`
	LastSeen int64  `json:"last_seen"`
}

type UserRiskSummary struct {
	TotalRequests          int64   `json:"total_requests"`
	ErrorCount             int64   `json:"error_count"`
	ErrorRate              float64 `json:"error_rate"`
	RefundCount            int64   `json:"refund_count"`
	RefundQuota            int64   `json:"refund_quota"`
	FailedRefundCount      int64   `json:"failed_refund_count"`
	RefundAfterOutputCount int64   `json:"refund_after_output_count"`
	SensitiveWordAttempts  int64   `json:"sensitive_word_attempts"`
	ClientAbortCount       int64   `json:"client_abort_count"`
	AbnormalStreamCount    int64   `json:"abnormal_stream_count"`
	UniqueIPCount          int64   `json:"unique_ip_count"`
}

type UserRiskReport struct {
	UserId        int              `json:"user_id"`
	Enabled       bool             `json:"enabled"`
	GlobalEnabled bool             `json:"global_enabled"`
	UserEnabled   bool             `json:"user_enabled"`
	WindowDays    int              `json:"window_days"`
	StartTime     int64            `json:"start_time"`
	EndTime       int64            `json:"end_time"`
	GeneratedAt   int64            `json:"generated_at"`
	Score         int              `json:"score"`
	Level         string           `json:"level"`
	Summary       UserRiskSummary  `json:"summary"`
	Signals       []UserRiskSignal `json:"signals"`
}

func SetUserRiskDetectionEnabled(userId int, enabled bool) error {
	if userId <= 0 {
		return errors.New("invalid user id")
	}
	result := DB.Model(&User{}).Where("id = ?", userId).Update("risk_detection_enabled", enabled)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

type userRiskAggregate struct {
	Count    int64 `gorm:"column:count"`
	LastSeen int64 `gorm:"column:last_seen"`
	Quota    int64 `gorm:"column:quota"`
}

func scanUserRiskAggregate(query *gorm.DB) (userRiskAggregate, error) {
	var aggregate userRiskAggregate
	err := query.Select(
		"COUNT(*) AS count, COALESCE(MAX(created_at), 0) AS last_seen, COALESCE(SUM(quota), 0) AS quota",
	).Scan(&aggregate).Error
	return aggregate, err
}

func userRiskLogQuery(userId int, startTime int64, endTime int64) *gorm.DB {
	return LOG_DB.Model(&Log{}).
		Where("user_id = ?", userId).
		Where("created_at >= ? AND created_at <= ?", startTime, endTime)
}

func appendUserRiskSignal(report *UserRiskReport, code string, severity string, score int, count int64, lastSeen int64) {
	if report == nil || score <= 0 || count <= 0 {
		return
	}
	report.Score += score
	report.Signals = append(report.Signals, UserRiskSignal{
		Code:     code,
		Severity: severity,
		Score:    score,
		Count:    count,
		LastSeen: lastSeen,
	})
}

func GetUserRiskReport(userId int, windowDays int, generatedAt int64) (*UserRiskReport, error) {
	if userId <= 0 || windowDays <= 0 || generatedAt <= 0 {
		return nil, errors.New("invalid user risk report range")
	}

	report := &UserRiskReport{
		UserId:      userId,
		WindowDays:  windowDays,
		StartTime:   generatedAt - int64(windowDays)*24*60*60,
		EndTime:     generatedAt,
		GeneratedAt: generatedAt,
		Level:       UserRiskLevelLow,
		Signals:     make([]UserRiskSignal, 0),
	}
	requestAggregate, err := scanUserRiskAggregate(
		userRiskLogQuery(userId, report.StartTime, report.EndTime).
			Where("type IN ?", []int{LogTypeConsume, LogTypeError}),
	)
	if err != nil {
		return nil, err
	}
	errorAggregate, err := scanUserRiskAggregate(
		userRiskLogQuery(userId, report.StartTime, report.EndTime).
			Where("type = ?", LogTypeError),
	)
	if err != nil {
		return nil, err
	}
	sensitiveAggregate, err := scanUserRiskAggregate(
		userRiskLogQuery(userId, report.StartTime, report.EndTime).
			Where("type = ?", LogTypeError).
			Where("other LIKE ?", `%"error_code":"sensitive_words_detected"%`),
	)
	if err != nil {
		return nil, err
	}
	clientAbortAggregate, err := scanUserRiskAggregate(
		userRiskLogQuery(userId, report.StartTime, report.EndTime).
			Where(
				"(type = ? AND is_stream = ? AND other LIKE ?) OR (type = ? AND other LIKE ?)",
				LogTypeConsume,
				true,
				`%"end_reason":"client_gone"%`,
				LogTypeQuotaIncrease,
				`%"client_disconnected":true%`,
			),
	)
	if err != nil {
		return nil, err
	}
	abnormalStreamAggregate, err := scanUserRiskAggregate(
		userRiskLogQuery(userId, report.StartTime, report.EndTime).
			Where("type = ? AND is_stream = ?", LogTypeConsume, true).
			Where("other LIKE ?", `%"stream_status":%`).
			Where("other LIKE ?", `%"status":"error"%`),
	)
	if err != nil {
		return nil, err
	}
	refundAggregate, err := scanUserRiskAggregate(
		userRiskLogQuery(userId, report.StartTime, report.EndTime).
			Where("type = ?", LogTypeQuotaIncrease).
			Where("other LIKE ?", `%"source":"refund"%`),
	)
	if err != nil {
		return nil, err
	}
	failedRefundAggregate, err := scanUserRiskAggregate(
		userRiskLogQuery(userId, report.StartTime, report.EndTime).
			Where("type = ?", LogTypeQuotaIncrease).
			Where("other LIKE ?", `%"source":"refund"%`).
			Where(
				"content LIKE ? OR content LIKE ? OR content LIKE ? OR content LIKE ?",
				"%请求失败%",
				"%任务失败%",
				"%request failed%",
				"%task failed%",
			),
	)
	if err != nil {
		return nil, err
	}
	refundAfterOutputAggregate, err := scanUserRiskAggregate(
		userRiskLogQuery(userId, report.StartTime, report.EndTime).
			Where("type = ?", LogTypeQuotaIncrease).
			Where("other LIKE ?", `%"refund_after_output":true%`),
	)
	if err != nil {
		return nil, err
	}

	var uniqueIPs []string
	if err := userRiskLogQuery(userId, report.StartTime, report.EndTime).
		Where("ip <> ''").
		Distinct("ip").
		Pluck("ip", &uniqueIPs).Error; err != nil {
		return nil, err
	}
	ipAggregate, err := scanUserRiskAggregate(
		userRiskLogQuery(userId, report.StartTime, report.EndTime).
			Where("ip <> ''"),
	)
	if err != nil {
		return nil, err
	}

	report.Summary = UserRiskSummary{
		TotalRequests:          requestAggregate.Count,
		ErrorCount:             errorAggregate.Count,
		RefundCount:            refundAggregate.Count,
		RefundQuota:            refundAggregate.Quota,
		FailedRefundCount:      failedRefundAggregate.Count,
		RefundAfterOutputCount: refundAfterOutputAggregate.Count,
		SensitiveWordAttempts:  sensitiveAggregate.Count,
		ClientAbortCount:       clientAbortAggregate.Count,
		AbnormalStreamCount:    abnormalStreamAggregate.Count,
		UniqueIPCount:          int64(len(uniqueIPs)),
	}
	if requestAggregate.Count > 0 {
		report.Summary.ErrorRate = float64(errorAggregate.Count) / float64(requestAggregate.Count)
	}

	if sensitiveAggregate.Count > 0 {
		score := min(60, 30+int(sensitiveAggregate.Count-1)*10)
		severity := UserRiskLevelMedium
		if sensitiveAggregate.Count >= 3 {
			severity = UserRiskLevelHigh
		}
		appendUserRiskSignal(report, UserRiskSignalSensitiveWords, severity, score, sensitiveAggregate.Count, sensitiveAggregate.LastSeen)
	}
	if requestAggregate.Count >= 10 && report.Summary.ErrorRate >= 0.25 {
		score := 15
		severity := UserRiskLevelMedium
		if report.Summary.ErrorRate >= 0.5 {
			score = 25
			severity = UserRiskLevelHigh
		}
		appendUserRiskSignal(report, UserRiskSignalErrorRate, severity, score, errorAggregate.Count, errorAggregate.LastSeen)
	}
	if clientAbortAggregate.Count >= 3 {
		score := min(20, int(clientAbortAggregate.Count)*3)
		severity := UserRiskLevelMedium
		if clientAbortAggregate.Count >= 10 {
			severity = UserRiskLevelHigh
		}
		appendUserRiskSignal(report, UserRiskSignalClientAbort, severity, score, clientAbortAggregate.Count, clientAbortAggregate.LastSeen)
	}
	otherAbnormalCount := max(int64(0), abnormalStreamAggregate.Count-clientAbortAggregate.Count)
	if otherAbnormalCount >= 3 {
		score := min(15, int(otherAbnormalCount)*2)
		appendUserRiskSignal(report, UserRiskSignalAbnormalStream, UserRiskLevelMedium, score, otherAbnormalCount, abnormalStreamAggregate.LastSeen)
	}
	if failedRefundAggregate.Count >= 3 {
		score := min(20, 5+int(failedRefundAggregate.Count)*2)
		severity := UserRiskLevelMedium
		if failedRefundAggregate.Count >= 10 {
			severity = UserRiskLevelHigh
		}
		appendUserRiskSignal(report, UserRiskSignalFailedRefund, severity, score, failedRefundAggregate.Count, failedRefundAggregate.LastSeen)
	}
	if refundAfterOutputAggregate.Count > 0 {
		score := min(60, 35+int(refundAfterOutputAggregate.Count-1)*15)
		severity := UserRiskLevelMedium
		if refundAfterOutputAggregate.Count >= 3 {
			severity = UserRiskLevelHigh
		}
		appendUserRiskSignal(report, UserRiskSignalRefundAfterOutput, severity, score, refundAfterOutputAggregate.Count, refundAfterOutputAggregate.LastSeen)
	}
	if len(uniqueIPs) >= 5 {
		score := 15
		severity := UserRiskLevelMedium
		if len(uniqueIPs) >= 10 {
			score = 25
			severity = UserRiskLevelHigh
		}
		appendUserRiskSignal(report, UserRiskSignalMultipleIPs, severity, score, int64(len(uniqueIPs)), ipAggregate.LastSeen)
	}

	report.Score = max(0, min(100, report.Score))
	if report.Score >= 60 {
		report.Level = UserRiskLevelHigh
	} else if report.Score >= 25 {
		report.Level = UserRiskLevelMedium
	}
	return report, nil
}
