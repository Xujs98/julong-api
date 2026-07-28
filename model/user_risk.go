package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const UserRiskTagWindowDays = 7

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
	UserId   int   `gorm:"column:user_id"`
	Count    int64 `gorm:"column:count"`
	LastSeen int64 `gorm:"column:last_seen"`
	Quota    int64 `gorm:"column:quota"`
}

type userRiskReportAggregates struct {
	Requests          userRiskAggregate
	Errors            userRiskAggregate
	SensitiveWords    userRiskAggregate
	ClientAborts      userRiskAggregate
	AbnormalStreams   userRiskAggregate
	Refunds           userRiskAggregate
	FailedRefunds     userRiskAggregate
	RefundAfterOutput userRiskAggregate
	IPs               userRiskAggregate
}

func scanUserRiskAggregate(query *gorm.DB) (userRiskAggregate, error) {
	var aggregate userRiskAggregate
	err := query.Select(
		"COUNT(*) AS count, COALESCE(MAX(created_at), 0) AS last_seen, COALESCE(SUM(quota), 0) AS quota",
	).Scan(&aggregate).Error
	return aggregate, err
}

func scanUserRiskAggregates(query *gorm.DB) (map[int]userRiskAggregate, error) {
	var aggregates []userRiskAggregate
	err := query.Select(
		"user_id, COUNT(*) AS count, COALESCE(MAX(created_at), 0) AS last_seen, COALESCE(SUM(quota), 0) AS quota",
	).Group("user_id").Scan(&aggregates).Error
	if err != nil {
		return nil, err
	}
	aggregatesByUserId := make(map[int]userRiskAggregate, len(aggregates))
	for _, aggregate := range aggregates {
		aggregatesByUserId[aggregate.UserId] = aggregate
	}
	return aggregatesByUserId, nil
}

func userRiskLogQuery(userId int, startTime int64, endTime int64) *gorm.DB {
	return LOG_DB.Model(&Log{}).
		Where("user_id = ?", userId).
		Where("created_at >= ? AND created_at <= ?", startTime, endTime)
}

func userRiskLogsQuery(userIds []int, startTime int64, endTime int64) *gorm.DB {
	return LOG_DB.Model(&Log{}).
		Where("user_id IN ?", userIds).
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

func buildUserRiskReport(userId int, windowDays int, generatedAt int64, aggregates userRiskReportAggregates) *UserRiskReport {
	report := &UserRiskReport{
		UserId:      userId,
		WindowDays:  windowDays,
		StartTime:   generatedAt - int64(windowDays)*24*60*60,
		EndTime:     generatedAt,
		GeneratedAt: generatedAt,
		Level:       UserRiskLevelLow,
		Signals:     make([]UserRiskSignal, 0),
	}
	report.Summary = UserRiskSummary{
		TotalRequests:          aggregates.Requests.Count,
		ErrorCount:             aggregates.Errors.Count,
		RefundCount:            aggregates.Refunds.Count,
		RefundQuota:            aggregates.Refunds.Quota,
		FailedRefundCount:      aggregates.FailedRefunds.Count,
		RefundAfterOutputCount: aggregates.RefundAfterOutput.Count,
		SensitiveWordAttempts:  aggregates.SensitiveWords.Count,
		ClientAbortCount:       aggregates.ClientAborts.Count,
		AbnormalStreamCount:    aggregates.AbnormalStreams.Count,
		UniqueIPCount:          aggregates.IPs.Count,
	}
	if aggregates.Requests.Count > 0 {
		report.Summary.ErrorRate = float64(aggregates.Errors.Count) / float64(aggregates.Requests.Count)
	}

	if aggregates.SensitiveWords.Count > 0 {
		score := min(60, 30+int(aggregates.SensitiveWords.Count-1)*10)
		severity := UserRiskLevelMedium
		if aggregates.SensitiveWords.Count >= 3 {
			severity = UserRiskLevelHigh
		}
		appendUserRiskSignal(report, UserRiskSignalSensitiveWords, severity, score, aggregates.SensitiveWords.Count, aggregates.SensitiveWords.LastSeen)
	}
	if aggregates.Requests.Count >= 10 && report.Summary.ErrorRate >= 0.25 {
		score := 15
		severity := UserRiskLevelMedium
		if report.Summary.ErrorRate >= 0.5 {
			score = 25
			severity = UserRiskLevelHigh
		}
		appendUserRiskSignal(report, UserRiskSignalErrorRate, severity, score, aggregates.Errors.Count, aggregates.Errors.LastSeen)
	}
	if aggregates.ClientAborts.Count >= 3 {
		score := min(20, int(aggregates.ClientAborts.Count)*3)
		severity := UserRiskLevelMedium
		if aggregates.ClientAborts.Count >= 10 {
			severity = UserRiskLevelHigh
		}
		appendUserRiskSignal(report, UserRiskSignalClientAbort, severity, score, aggregates.ClientAborts.Count, aggregates.ClientAborts.LastSeen)
	}
	otherAbnormalCount := max(int64(0), aggregates.AbnormalStreams.Count-aggregates.ClientAborts.Count)
	if otherAbnormalCount >= 3 {
		score := min(15, int(otherAbnormalCount)*2)
		appendUserRiskSignal(report, UserRiskSignalAbnormalStream, UserRiskLevelMedium, score, otherAbnormalCount, aggregates.AbnormalStreams.LastSeen)
	}
	if aggregates.FailedRefunds.Count >= 3 {
		score := min(20, 5+int(aggregates.FailedRefunds.Count)*2)
		severity := UserRiskLevelMedium
		if aggregates.FailedRefunds.Count >= 10 {
			severity = UserRiskLevelHigh
		}
		appendUserRiskSignal(report, UserRiskSignalFailedRefund, severity, score, aggregates.FailedRefunds.Count, aggregates.FailedRefunds.LastSeen)
	}
	if aggregates.RefundAfterOutput.Count > 0 {
		score := min(60, 35+int(aggregates.RefundAfterOutput.Count-1)*15)
		severity := UserRiskLevelMedium
		if aggregates.RefundAfterOutput.Count >= 3 {
			severity = UserRiskLevelHigh
		}
		appendUserRiskSignal(report, UserRiskSignalRefundAfterOutput, severity, score, aggregates.RefundAfterOutput.Count, aggregates.RefundAfterOutput.LastSeen)
	}
	if aggregates.IPs.Count >= 5 {
		score := 15
		severity := UserRiskLevelMedium
		if aggregates.IPs.Count >= 10 {
			score = 25
			severity = UserRiskLevelHigh
		}
		appendUserRiskSignal(report, UserRiskSignalMultipleIPs, severity, score, aggregates.IPs.Count, aggregates.IPs.LastSeen)
	}

	report.Score = max(0, min(100, report.Score))
	if report.Score >= 60 {
		report.Level = UserRiskLevelHigh
	} else if report.Score >= 25 {
		report.Level = UserRiskLevelMedium
	}
	return report
}

func GetUserRiskReport(userId int, windowDays int, generatedAt int64) (*UserRiskReport, error) {
	if userId <= 0 || windowDays <= 0 || generatedAt <= 0 {
		return nil, errors.New("invalid user risk report range")
	}
	startTime := generatedAt - int64(windowDays)*24*60*60
	requestAggregate, err := scanUserRiskAggregate(
		userRiskLogQuery(userId, startTime, generatedAt).
			Where("type IN ?", []int{LogTypeConsume, LogTypeError}),
	)
	if err != nil {
		return nil, err
	}
	errorAggregate, err := scanUserRiskAggregate(
		userRiskLogQuery(userId, startTime, generatedAt).
			Where("type = ?", LogTypeError),
	)
	if err != nil {
		return nil, err
	}
	sensitiveAggregate, err := scanUserRiskAggregate(
		userRiskLogQuery(userId, startTime, generatedAt).
			Where("type = ?", LogTypeError).
			Where("other LIKE ?", `%"error_code":"sensitive_words_detected"%`),
	)
	if err != nil {
		return nil, err
	}
	clientAbortAggregate, err := scanUserRiskAggregate(
		userRiskLogQuery(userId, startTime, generatedAt).
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
		userRiskLogQuery(userId, startTime, generatedAt).
			Where("type = ? AND is_stream = ?", LogTypeConsume, true).
			Where("other LIKE ?", `%"stream_status":%`).
			Where("other LIKE ?", `%"status":"error"%`),
	)
	if err != nil {
		return nil, err
	}
	refundAggregate, err := scanUserRiskAggregate(
		userRiskLogQuery(userId, startTime, generatedAt).
			Where("type = ?", LogTypeQuotaIncrease).
			Where("other LIKE ?", `%"source":"refund"%`),
	)
	if err != nil {
		return nil, err
	}
	failedRefundAggregate, err := scanUserRiskAggregate(
		userRiskLogQuery(userId, startTime, generatedAt).
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
		userRiskLogQuery(userId, startTime, generatedAt).
			Where("type = ?", LogTypeQuotaIncrease).
			Where("other LIKE ?", `%"refund_after_output":true%`),
	)
	if err != nil {
		return nil, err
	}

	var uniqueIPs []string
	if err := userRiskLogQuery(userId, startTime, generatedAt).
		Where("ip <> ''").
		Distinct("ip").
		Pluck("ip", &uniqueIPs).Error; err != nil {
		return nil, err
	}
	ipAggregate, err := scanUserRiskAggregate(
		userRiskLogQuery(userId, startTime, generatedAt).
			Where("ip <> ''"),
	)
	if err != nil {
		return nil, err
	}
	ipAggregate.Count = int64(len(uniqueIPs))
	return buildUserRiskReport(userId, windowDays, generatedAt, userRiskReportAggregates{
		Requests:          requestAggregate,
		Errors:            errorAggregate,
		SensitiveWords:    sensitiveAggregate,
		ClientAborts:      clientAbortAggregate,
		AbnormalStreams:   abnormalStreamAggregate,
		Refunds:           refundAggregate,
		FailedRefunds:     failedRefundAggregate,
		RefundAfterOutput: refundAfterOutputAggregate,
		IPs:               ipAggregate,
	}), nil
}

func GetUserRiskReports(userIds []int, windowDays int, generatedAt int64) (map[int]*UserRiskReport, error) {
	if windowDays <= 0 || generatedAt <= 0 {
		return nil, errors.New("invalid user risk report range")
	}
	uniqueUserIds := make([]int, 0, len(userIds))
	seenUserIds := make(map[int]struct{}, len(userIds))
	for _, userId := range userIds {
		if userId <= 0 {
			continue
		}
		if _, exists := seenUserIds[userId]; exists {
			continue
		}
		seenUserIds[userId] = struct{}{}
		uniqueUserIds = append(uniqueUserIds, userId)
	}
	if len(uniqueUserIds) == 0 {
		return map[int]*UserRiskReport{}, nil
	}

	startTime := generatedAt - int64(windowDays)*24*60*60
	requestAggregates, err := scanUserRiskAggregates(
		userRiskLogsQuery(uniqueUserIds, startTime, generatedAt).
			Where("type IN ?", []int{LogTypeConsume, LogTypeError}),
	)
	if err != nil {
		return nil, err
	}
	errorAggregates, err := scanUserRiskAggregates(
		userRiskLogsQuery(uniqueUserIds, startTime, generatedAt).
			Where("type = ?", LogTypeError),
	)
	if err != nil {
		return nil, err
	}
	sensitiveAggregates, err := scanUserRiskAggregates(
		userRiskLogsQuery(uniqueUserIds, startTime, generatedAt).
			Where("type = ?", LogTypeError).
			Where("other LIKE ?", `%"error_code":"sensitive_words_detected"%`),
	)
	if err != nil {
		return nil, err
	}
	clientAbortAggregates, err := scanUserRiskAggregates(
		userRiskLogsQuery(uniqueUserIds, startTime, generatedAt).
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
	abnormalStreamAggregates, err := scanUserRiskAggregates(
		userRiskLogsQuery(uniqueUserIds, startTime, generatedAt).
			Where("type = ? AND is_stream = ?", LogTypeConsume, true).
			Where("other LIKE ?", `%"stream_status":%`).
			Where("other LIKE ?", `%"status":"error"%`),
	)
	if err != nil {
		return nil, err
	}
	refundAggregates, err := scanUserRiskAggregates(
		userRiskLogsQuery(uniqueUserIds, startTime, generatedAt).
			Where("type = ?", LogTypeQuotaIncrease).
			Where("other LIKE ?", `%"source":"refund"%`),
	)
	if err != nil {
		return nil, err
	}
	failedRefundAggregates, err := scanUserRiskAggregates(
		userRiskLogsQuery(uniqueUserIds, startTime, generatedAt).
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
	refundAfterOutputAggregates, err := scanUserRiskAggregates(
		userRiskLogsQuery(uniqueUserIds, startTime, generatedAt).
			Where("type = ?", LogTypeQuotaIncrease).
			Where("other LIKE ?", `%"refund_after_output":true%`),
	)
	if err != nil {
		return nil, err
	}
	var ipAggregates []userRiskAggregate
	err = userRiskLogsQuery(uniqueUserIds, startTime, generatedAt).
		Where("ip <> ''").
		Select("user_id, COUNT(DISTINCT ip) AS count, COALESCE(MAX(created_at), 0) AS last_seen").
		Group("user_id").
		Scan(&ipAggregates).Error
	if err != nil {
		return nil, err
	}
	ipAggregatesByUserId := make(map[int]userRiskAggregate, len(ipAggregates))
	for _, aggregate := range ipAggregates {
		ipAggregatesByUserId[aggregate.UserId] = aggregate
	}

	reports := make(map[int]*UserRiskReport, len(uniqueUserIds))
	for _, userId := range uniqueUserIds {
		reports[userId] = buildUserRiskReport(userId, windowDays, generatedAt, userRiskReportAggregates{
			Requests:          requestAggregates[userId],
			Errors:            errorAggregates[userId],
			SensitiveWords:    sensitiveAggregates[userId],
			ClientAborts:      clientAbortAggregates[userId],
			AbnormalStreams:   abnormalStreamAggregates[userId],
			Refunds:           refundAggregates[userId],
			FailedRefunds:     failedRefundAggregates[userId],
			RefundAfterOutput: refundAfterOutputAggregates[userId],
			IPs:               ipAggregatesByUserId[userId],
		})
	}
	return reports, nil
}

func ListRiskDetectionUsers(afterUserId int, limit int, globalEnabled bool) ([]User, error) {
	if limit <= 0 {
		limit = 500
	}
	if limit > 500 {
		limit = 500
	}
	query := DB.Model(&User{}).
		Select("id", "username", "display_name", "risk_detection_enabled").
		Where("id > ?", afterUserId).
		Where("deleted_at IS NULL")
	if !globalEnabled {
		query = query.Where("risk_detection_enabled = ?", true)
	}
	var users []User
	err := query.Order("id asc").Limit(limit).Find(&users).Error
	return users, err
}

func fillUserRiskTagInfo(users []*User, generatedAt int64) {
	if len(users) == 0 || generatedAt <= 0 {
		return
	}
	enabledUserIds := make([]int, 0, len(users))
	for _, user := range users {
		if common.UserRiskDetectionEnabled || user.RiskDetectionEnabled {
			enabledUserIds = append(enabledUserIds, user.Id)
		}
	}
	reports, err := GetUserRiskReports(enabledUserIds, UserRiskTagWindowDays, generatedAt)
	if err != nil {
		return
	}
	fillUserRiskTagsFromReports(users, reports)
}

func fillUserRiskTagsFromReports(users []*User, reports map[int]*UserRiskReport) {
	for _, user := range users {
		report := reports[user.Id]
		if report == nil {
			continue
		}
		var tagId int
		switch report.Level {
		case UserRiskLevelMedium:
			tagId = UserTagRiskMediumId
		case UserRiskLevelHigh:
			tagId = UserTagRiskHighId
		default:
			continue
		}
		tag, exists := GetBuiltInUserTag(tagId)
		if exists {
			tagCopy := tag
			user.RiskTag = &tagCopy
		}
	}
}
