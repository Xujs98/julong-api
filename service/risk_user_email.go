package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	riskUserEmailScanBatchSize       = 500
	maxRiskUsersPerNotificationEmail = 100
)

type RiskUserEmailDispatchResult struct {
	RecipientCount int      `json:"recipient_count"`
	RiskUserCount  int      `json:"risk_user_count"`
	Levels         []string `json:"levels"`
}

type riskUserEmailEntry struct {
	User   model.User
	Report *model.UserRiskReport
}

type riskUserEmailNotificationState struct {
	Level      string `json:"level"`
	Score      int    `json:"score"`
	NotifiedAt int64  `json:"notified_at"`
}

func normalizeRiskUserEmailLevels(levels []string) []string {
	selected := make(map[string]struct{}, len(levels))
	for _, level := range levels {
		switch strings.ToLower(strings.TrimSpace(level)) {
		case model.UserRiskLevelMedium:
			selected[model.UserRiskLevelMedium] = struct{}{}
		case model.UserRiskLevelHigh:
			selected[model.UserRiskLevelHigh] = struct{}{}
		}
	}
	result := make([]string, 0, len(selected))
	for _, level := range []string{model.UserRiskLevelMedium, model.UserRiskLevelHigh} {
		if _, exists := selected[level]; exists {
			result = append(result, level)
		}
	}
	return result
}

func decodeRiskUserEmailLevels(target *[]string) error {
	raw := emailOptionString(common.RiskUserEmailLevelsOptionKey)
	if raw == "" {
		*target = []string{model.UserRiskLevelMedium, model.UserRiskLevelHigh}
		return nil
	}
	var levels []string
	if err := common.UnmarshalJsonStr(raw, &levels); err != nil {
		return fmt.Errorf("invalid risk user email levels: %w", err)
	}
	levels = normalizeRiskUserEmailLevels(levels)
	if len(levels) == 0 {
		levels = []string{model.UserRiskLevelMedium, model.UserRiskLevelHigh}
	}
	*target = levels
	return nil
}

func IsRiskUserEmailEnabled() bool {
	return emailOptionBool(common.RiskUserEmailEnabledOptionKey, false)
}

func SendRiskUserTestEmails(userIDs []int, levels []string) (RiskUserEmailDispatchResult, error) {
	return sendRiskUserTestEmailsAt(context.Background(), time.Now(), userIDs, levels, common.SendEmail)
}

func sendRiskUserTestEmailsAt(ctx context.Context, now time.Time, userIDs []int, levels []string, sender EmailCampaignSender) (RiskUserEmailDispatchResult, error) {
	if sender == nil {
		return RiskUserEmailDispatchResult{}, errors.New("email sender is required")
	}
	levels = normalizeRiskUserEmailLevels(levels)
	if len(levels) == 0 {
		return RiskUserEmailDispatchResult{}, errors.New("at least one risk level is required")
	}
	userIDs = normalizeEmailRecipientIDs(userIDs)
	if len(userIDs) > maxOperationalEmailRecipients {
		return RiskUserEmailDispatchResult{}, fmt.Errorf("recipient count cannot exceed %d", maxOperationalEmailRecipients)
	}
	if err := validateOperationalEmailRecipientIDs(userIDs); err != nil {
		return RiskUserEmailDispatchResult{}, err
	}
	recipients, err := model.GetOperationalEmailRecipientUsers(userIDs)
	if err != nil {
		return RiskUserEmailDispatchResult{}, err
	}
	if len(recipients) == 0 {
		return RiskUserEmailDispatchResult{}, errors.New("no active administrator or root recipient with an email address")
	}
	entries, _, err := collectRiskUserEmailEntries(ctx, now, levels, nil, false)
	if err != nil {
		return RiskUserEmailDispatchResult{}, err
	}
	if len(entries) == 0 {
		return RiskUserEmailDispatchResult{}, errors.New("no current medium or high risk users match the selected levels")
	}
	return sendRiskUserEmailBatch(now, recipients, entries, true, sender)
}

func DispatchRiskUserEmails(ctx context.Context, sender EmailCampaignSender) (RiskUserEmailDispatchResult, error) {
	if sender == nil {
		return RiskUserEmailDispatchResult{}, errors.New("email sender is required")
	}
	config, err := GetEmailSettingsConfig()
	if err != nil {
		return RiskUserEmailDispatchResult{}, err
	}
	if !config.RiskUserEmailEnabled {
		return RiskUserEmailDispatchResult{}, nil
	}
	recipients, err := model.GetOperationalEmailRecipientUsers(config.RiskUserEmailRecipientIDs)
	if err != nil {
		return RiskUserEmailDispatchResult{}, err
	}
	if len(recipients) == 0 {
		return RiskUserEmailDispatchResult{}, errors.New("no active administrator or root recipient with an email address")
	}
	state, err := loadRiskUserEmailDispatchState()
	if err != nil {
		return RiskUserEmailDispatchResult{}, err
	}
	now := time.Now()
	entries, stateChanged, err := collectRiskUserEmailEntries(ctx, now, config.RiskUserEmailLevels, state, true)
	if err != nil {
		return RiskUserEmailDispatchResult{}, err
	}
	if len(entries) == 0 {
		if stateChanged {
			return RiskUserEmailDispatchResult{}, saveRiskUserEmailDispatchState(state)
		}
		return RiskUserEmailDispatchResult{}, nil
	}
	result, err := sendRiskUserEmailBatch(now, recipients, entries, false, sender)
	if err != nil {
		return result, err
	}
	for _, entry := range entries {
		state[strconv.Itoa(entry.User.Id)] = riskUserEmailNotificationState{
			Level:      entry.Report.Level,
			Score:      entry.Report.Score,
			NotifiedAt: now.Unix(),
		}
	}
	if err := saveRiskUserEmailDispatchState(state); err != nil {
		return result, err
	}
	return result, nil
}

func collectRiskUserEmailEntries(ctx context.Context, now time.Time, levels []string, state map[string]riskUserEmailNotificationState, newOnly bool) ([]riskUserEmailEntry, bool, error) {
	levels = normalizeRiskUserEmailLevels(levels)
	selectedLevels := make(map[string]struct{}, len(levels))
	for _, level := range levels {
		selectedLevels[level] = struct{}{}
	}
	entries := make([]riskUserEmailEntry, 0)
	stateChanged := false
	afterUserId := 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, stateChanged, err
		}
		users, err := model.ListRiskDetectionUsers(afterUserId, riskUserEmailScanBatchSize, common.UserRiskDetectionEnabled)
		if err != nil {
			return nil, stateChanged, err
		}
		if len(users) == 0 {
			break
		}
		userIDs := make([]int, 0, len(users))
		for _, user := range users {
			userIDs = append(userIDs, user.Id)
		}
		reports, err := model.GetUserRiskReports(userIDs, model.UserRiskTagWindowDays, now.Unix())
		if err != nil {
			return nil, stateChanged, err
		}
		for _, user := range users {
			report := reports[user.Id]
			if report == nil || report.Level == model.UserRiskLevelLow {
				if newOnly {
					key := strconv.Itoa(user.Id)
					if _, exists := state[key]; exists {
						delete(state, key)
						stateChanged = true
					}
				}
				continue
			}
			if _, selected := selectedLevels[report.Level]; !selected {
				continue
			}
			if newOnly {
				previous, exists := state[strconv.Itoa(user.Id)]
				if exists && riskUserEmailLevelRank(report.Level) <= riskUserEmailLevelRank(previous.Level) {
					continue
				}
			}
			entries = append(entries, riskUserEmailEntry{User: user, Report: report})
		}
		afterUserId = users[len(users)-1].Id
	}
	sort.Slice(entries, func(i, j int) bool {
		left, right := entries[i], entries[j]
		if left.Report.Level != right.Report.Level {
			return riskUserEmailLevelRank(left.Report.Level) > riskUserEmailLevelRank(right.Report.Level)
		}
		if left.Report.Score != right.Report.Score {
			return left.Report.Score > right.Report.Score
		}
		return left.User.Id < right.User.Id
	})
	if len(entries) > maxRiskUsersPerNotificationEmail {
		entries = entries[:maxRiskUsersPerNotificationEmail]
	}
	return entries, stateChanged, nil
}

func riskUserEmailLevelRank(level string) int {
	switch level {
	case model.UserRiskLevelHigh:
		return 2
	case model.UserRiskLevelMedium:
		return 1
	default:
		return 0
	}
}

func sendRiskUserEmailBatch(now time.Time, recipients []model.User, entries []riskUserEmailEntry, test bool, sender EmailCampaignSender) (RiskUserEmailDispatchResult, error) {
	result := RiskUserEmailDispatchResult{
		RiskUserCount: len(entries),
		Levels:        riskUserEmailEntryLevels(entries),
	}
	for _, recipient := range recipients {
		locale := NormalizeEmailTemplateLocale(recipient.GetSetting().Language)
		values := map[string]string{
			"alert_mode":      riskUserEmailAlertMode(locale, test),
			"risk_user_count": strconv.Itoa(len(entries)),
			"risk_levels":     formatRiskUserEmailLevels(result.Levels, locale),
			"risk_users":      formatRiskUserEmailEntries(entries, locale),
			"window_days":     strconv.Itoa(model.UserRiskTagWindowDays),
			"detected_at":     now.Format("2006-01-02 15:04:05"),
		}
		rendered, err := renderOperationalTemplateEmail(EmailTemplateEventRiskUserDetected, recipient, values)
		if err != nil {
			return result, err
		}
		if err := sender(rendered.Subject, recipient.Email, rendered.Content); err != nil {
			return result, fmt.Errorf("failed to send risk user email to user %d: %w", recipient.Id, err)
		}
		result.RecipientCount++
	}
	return result, nil
}

func riskUserEmailEntryLevels(entries []riskUserEmailEntry) []string {
	levels := make([]string, 0, 2)
	for _, level := range []string{model.UserRiskLevelMedium, model.UserRiskLevelHigh} {
		for _, entry := range entries {
			if entry.Report.Level == level {
				levels = append(levels, level)
				break
			}
		}
	}
	return levels
}

func riskUserEmailAlertMode(locale string, test bool) string {
	if locale == EmailTemplateLocaleEnglish {
		if test {
			return "Test email"
		}
		return "Automatic detection"
	}
	if test {
		return "测试邮件"
	}
	return "自动检测"
}

func formatRiskUserEmailLevels(levels []string, locale string) string {
	labels := make([]string, 0, len(levels))
	for _, level := range levels {
		labels = append(labels, riskUserEmailLevelLabel(level, locale))
	}
	separator := ", "
	if locale == EmailTemplateLocaleChinese {
		separator = "、"
	}
	return strings.Join(labels, separator)
}

func formatRiskUserEmailEntries(entries []riskUserEmailEntry, locale string) string {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		displayName := strings.TrimSpace(entry.User.DisplayName)
		if displayName == "" {
			displayName = entry.User.Username
		}
		signalLabels := make([]string, 0, len(entry.Report.Signals))
		for _, signal := range entry.Report.Signals {
			signalLabels = append(signalLabels, riskUserEmailSignalLabel(signal.Code, locale))
		}
		if locale == EmailTemplateLocaleEnglish {
			lines = append(lines, fmt.Sprintf("#%d %s (%s) | %s %d pts | Requests %d | Errors %d | Refunds %d | Signals: %s",
				entry.User.Id, entry.User.Username, displayName, riskUserEmailLevelLabel(entry.Report.Level, locale), entry.Report.Score,
				entry.Report.Summary.TotalRequests, entry.Report.Summary.ErrorCount, entry.Report.Summary.RefundCount, strings.Join(signalLabels, ", ")))
			continue
		}
		lines = append(lines, fmt.Sprintf("#%d %s（%s） | %s %d 分 | 请求 %d | 错误 %d | 返还 %d | 信号：%s",
			entry.User.Id, entry.User.Username, displayName, riskUserEmailLevelLabel(entry.Report.Level, locale), entry.Report.Score,
			entry.Report.Summary.TotalRequests, entry.Report.Summary.ErrorCount, entry.Report.Summary.RefundCount, strings.Join(signalLabels, "、")))
	}
	return strings.Join(lines, "\n")
}

func riskUserEmailLevelLabel(level string, locale string) string {
	if locale == EmailTemplateLocaleEnglish {
		if level == model.UserRiskLevelHigh {
			return "High risk"
		}
		return "Medium risk"
	}
	if level == model.UserRiskLevelHigh {
		return "高风险"
	}
	return "中风险"
}

func riskUserEmailSignalLabel(code string, locale string) string {
	english := map[string]string{
		model.UserRiskSignalSensitiveWords:    "Sensitive words",
		model.UserRiskSignalErrorRate:         "High error rate",
		model.UserRiskSignalClientAbort:       "Client aborts",
		model.UserRiskSignalAbnormalStream:    "Abnormal streams",
		model.UserRiskSignalFailedRefund:      "Failed refunds",
		model.UserRiskSignalRefundAfterOutput: "Refund after output",
		model.UserRiskSignalMultipleIPs:       "Multiple IPs",
	}
	chinese := map[string]string{
		model.UserRiskSignalSensitiveWords:    "敏感词尝试",
		model.UserRiskSignalErrorRate:         "高错误率",
		model.UserRiskSignalClientAbort:       "客户端取消",
		model.UserRiskSignalAbnormalStream:    "异常流",
		model.UserRiskSignalFailedRefund:      "失败返还",
		model.UserRiskSignalRefundAfterOutput: "输出后返还",
		model.UserRiskSignalMultipleIPs:       "多 IP",
	}
	if locale == EmailTemplateLocaleEnglish {
		if label := english[code]; label != "" {
			return label
		}
		return code
	}
	if label := chinese[code]; label != "" {
		return label
	}
	return code
}

func loadRiskUserEmailDispatchState() (map[string]riskUserEmailNotificationState, error) {
	raw := emailOptionString(common.RiskUserEmailDispatchStateOptionKey)
	state := make(map[string]riskUserEmailNotificationState)
	if raw == "" {
		return state, nil
	}
	if err := common.UnmarshalJsonStr(raw, &state); err != nil {
		return nil, fmt.Errorf("invalid risk user email dispatch state: %w", err)
	}
	return state, nil
}

func saveRiskUserEmailDispatchState(state map[string]riskUserEmailNotificationState) error {
	data, err := common.Marshal(state)
	if err != nil {
		return err
	}
	return model.UpdateOption(common.RiskUserEmailDispatchStateOptionKey, string(data))
}
