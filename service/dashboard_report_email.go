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
	DashboardReportFrequencyDaily   = "daily"
	DashboardReportFrequencyWeekly  = "weekly"
	DashboardReportFrequencyMonthly = "monthly"
	maxDashboardReportSchedules     = 20
	maxDashboardReportSendTimes     = 12
)

type DashboardReportEmailSchedule struct {
	ID        string   `json:"id"`
	Frequency string   `json:"frequency"`
	SendTimes []string `json:"send_times"`
	Weekday   int      `json:"weekday"`
	MonthDay  int      `json:"month_day"`
}

type DashboardReportEmailDispatchResult struct {
	RecipientCount int    `json:"recipient_count"`
	Period         string `json:"period"`
	ScheduledTime  string `json:"scheduled_time,omitempty"`
}

type dashboardReportPeriod struct {
	Start     time.Time
	End       time.Time
	PeriodKey string
	TypeZH    string
	TypeEN    string
}

type dashboardReportDispatch struct {
	Period        dashboardReportPeriod
	DispatchKey   string
	ScheduledTime string
}

type dashboardReportDispatchHistory map[string]int64

func validateDashboardReportEmailSchedules(schedules []DashboardReportEmailSchedule) error {
	if len(schedules) == 0 {
		return errors.New("at least one dashboard report schedule is required")
	}
	if len(schedules) > maxDashboardReportSchedules {
		return fmt.Errorf("dashboard report schedule count cannot exceed %d", maxDashboardReportSchedules)
	}
	seenIDs := make(map[string]struct{}, len(schedules))
	for _, schedule := range schedules {
		if schedule.ID == "" || len(schedule.ID) > 64 {
			return errors.New("dashboard report schedule ID is invalid")
		}
		if _, exists := seenIDs[schedule.ID]; exists {
			return errors.New("dashboard report schedule IDs must be unique")
		}
		seenIDs[schedule.ID] = struct{}{}
		switch schedule.Frequency {
		case DashboardReportFrequencyDaily, DashboardReportFrequencyWeekly, DashboardReportFrequencyMonthly:
		default:
			return errors.New("dashboard report frequency must be daily, weekly, or monthly")
		}
		if len(schedule.SendTimes) == 0 || len(schedule.SendTimes) > maxDashboardReportSendTimes {
			return fmt.Errorf("each dashboard report schedule must have between 1 and %d send times", maxDashboardReportSendTimes)
		}
		for _, sendTime := range schedule.SendTimes {
			if _, err := time.Parse("15:04", sendTime); err != nil {
				return errors.New("dashboard report send time must use HH:mm format")
			}
		}
		if schedule.Weekday < 1 || schedule.Weekday > 7 {
			return errors.New("dashboard report weekday must be between 1 and 7")
		}
		if schedule.MonthDay < 1 || schedule.MonthDay > 31 {
			return errors.New("dashboard report month day must be between 1 and 31")
		}
	}
	return nil
}

func IsDashboardReportEmailDue(now time.Time) bool {
	if !emailOptionBool(common.DashboardReportEmailEnabledOptionKey, false) {
		return false
	}
	config, err := GetEmailSettingsConfig()
	if err != nil || !config.DashboardReportEmailEnabled {
		return false
	}
	history := loadDashboardReportDispatchHistory()
	_, due := nextDashboardReportDispatch(config.DashboardReportEmailSchedules, now, history)
	return due
}

func SendDashboardReportTestEmails(userIDs []int) (DashboardReportEmailDispatchResult, error) {
	return sendDashboardReportTestEmailsAt(time.Now(), userIDs, common.SendEmail)
}

func sendDashboardReportTestEmailsAt(now time.Time, userIDs []int, sender EmailCampaignSender) (DashboardReportEmailDispatchResult, error) {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	period := dashboardReportPeriod{
		Start:  start,
		End:    now,
		TypeZH: "实时",
		TypeEN: "Real-time",
	}
	return sendDashboardReportEmails(context.Background(), period, userIDs, sender)
}

func DispatchDashboardReportEmails(ctx context.Context, sender EmailCampaignSender) (DashboardReportEmailDispatchResult, error) {
	return dispatchDashboardReportEmailsAt(ctx, time.Now(), sender)
}

func dispatchDashboardReportEmailsAt(ctx context.Context, now time.Time, sender EmailCampaignSender) (DashboardReportEmailDispatchResult, error) {
	config, err := GetEmailSettingsConfig()
	if err != nil {
		return DashboardReportEmailDispatchResult{}, err
	}
	history := loadDashboardReportDispatchHistory()
	dispatch, due := nextDashboardReportDispatch(config.DashboardReportEmailSchedules, now, history)
	if !config.DashboardReportEmailEnabled || !due {
		return DashboardReportEmailDispatchResult{}, nil
	}
	result, err := sendDashboardReportEmails(ctx, dispatch.Period, config.DashboardReportEmailRecipientIDs, sender)
	if err != nil {
		return result, err
	}
	result.ScheduledTime = dispatch.ScheduledTime
	history[dispatch.DispatchKey] = now.Unix()
	if err := persistDashboardReportDispatchHistory(history, now); err != nil {
		return result, err
	}
	return result, nil
}

func sendDashboardReportEmails(ctx context.Context, period dashboardReportPeriod, userIDs []int, sender EmailCampaignSender) (DashboardReportEmailDispatchResult, error) {
	result := DashboardReportEmailDispatchResult{Period: formatDashboardReportPeriod(period)}
	if sender == nil {
		return result, errors.New("email sender is required")
	}
	userIDs = normalizeEmailRecipientIDs(userIDs)
	if len(userIDs) > maxOperationalEmailRecipients {
		return result, fmt.Errorf("recipient count cannot exceed %d", maxOperationalEmailRecipients)
	}
	if err := validateOperationalEmailRecipientIDs(userIDs); err != nil {
		return result, err
	}
	recipients, err := model.GetOperationalEmailRecipientUsers(userIDs)
	if err != nil {
		return result, err
	}
	if len(recipients) == 0 {
		return result, errors.New("no active administrator or root recipient with an email address")
	}
	report, err := model.GetDashboardReportData(period.Start.Unix(), period.End.Unix())
	if err != nil {
		return result, err
	}

	for _, recipient := range recipients {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}
		locale := NormalizeEmailTemplateLocale(recipient.GetSetting().Language)
		values := dashboardReportTemplateValues(report, period, locale)
		rendered, err := renderOperationalTemplateEmail(EmailTemplateEventDashboardReport, recipient, values)
		if err != nil {
			return result, err
		}
		if err := sender(rendered.Subject, recipient.Email, rendered.Content); err != nil {
			return result, fmt.Errorf("failed to send dashboard report email to user %d: %w", recipient.Id, err)
		}
		result.RecipientCount++
	}
	return result, nil
}

func dashboardReportPeriodForSchedule(schedule DashboardReportEmailSchedule, now time.Time) (dashboardReportPeriod, bool) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch schedule.Frequency {
	case DashboardReportFrequencyDaily:
		start := today.AddDate(0, 0, -1)
		return newDashboardReportPeriod(schedule.Frequency, start, today), true
	case DashboardReportFrequencyWeekly:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		if weekday != schedule.Weekday {
			return dashboardReportPeriod{}, false
		}
		end := today.AddDate(0, 0, -(weekday - 1))
		return newDashboardReportPeriod(schedule.Frequency, end.AddDate(0, 0, -7), end), true
	case DashboardReportFrequencyMonthly:
		lastDay := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()
		scheduledDay := schedule.MonthDay
		if scheduledDay > lastDay {
			scheduledDay = lastDay
		}
		if now.Day() != scheduledDay {
			return dashboardReportPeriod{}, false
		}
		end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return newDashboardReportPeriod(schedule.Frequency, end.AddDate(0, -1, 0), end), true
	default:
		return dashboardReportPeriod{}, false
	}
}

func nextDashboardReportDispatch(schedules []DashboardReportEmailSchedule, now time.Time, history dashboardReportDispatchHistory) (dashboardReportDispatch, bool) {
	for _, schedule := range schedules {
		period, matchesDate := dashboardReportPeriodForSchedule(schedule, now)
		if !matchesDate {
			continue
		}
		for _, sendTime := range schedule.SendTimes {
			parsed, err := time.Parse("15:04", sendTime)
			if err != nil || now.Hour() < parsed.Hour() || (now.Hour() == parsed.Hour() && now.Minute() < parsed.Minute()) {
				continue
			}
			dispatchKey := fmt.Sprintf("%s:%s:%d:%d:%s:%s", schedule.ID, schedule.Frequency, schedule.Weekday, schedule.MonthDay, sendTime, period.PeriodKey)
			if _, sent := history[dispatchKey]; sent {
				continue
			}
			return dashboardReportDispatch{Period: period, DispatchKey: dispatchKey, ScheduledTime: sendTime}, true
		}
	}
	return dashboardReportDispatch{}, false
}

func decodeDashboardReportEmailSchedules(target *[]DashboardReportEmailSchedule) error {
	raw := emailOptionString(common.DashboardReportEmailSchedulesOptionKey)
	if raw == "" {
		*target = []DashboardReportEmailSchedule{}
		return nil
	}
	if err := common.UnmarshalJsonStr(raw, target); err != nil {
		return fmt.Errorf("invalid dashboard report email schedules: %w", err)
	}
	return nil
}

func normalizeDashboardReportEmailSchedules(schedules []DashboardReportEmailSchedule) []DashboardReportEmailSchedule {
	result := make([]DashboardReportEmailSchedule, 0, len(schedules))
	for scheduleIndex, schedule := range schedules {
		schedule.ID = strings.TrimSpace(schedule.ID)
		if schedule.ID == "" {
			schedule.ID = fmt.Sprintf("schedule-%d", scheduleIndex+1)
		}
		schedule.Frequency = strings.TrimSpace(schedule.Frequency)
		if schedule.Frequency == "" {
			schedule.Frequency = DashboardReportFrequencyDaily
		}
		if schedule.Weekday == 0 {
			schedule.Weekday = 1
		}
		if schedule.MonthDay == 0 {
			schedule.MonthDay = 1
		}
		seenTimes := make(map[string]struct{}, len(schedule.SendTimes))
		sendTimes := make([]string, 0, len(schedule.SendTimes))
		for _, sendTime := range schedule.SendTimes {
			sendTime = strings.TrimSpace(sendTime)
			if sendTime == "" {
				continue
			}
			if _, exists := seenTimes[sendTime]; exists {
				continue
			}
			seenTimes[sendTime] = struct{}{}
			sendTimes = append(sendTimes, sendTime)
		}
		if len(sendTimes) == 0 {
			sendTimes = append(sendTimes, "08:00")
		}
		sort.Strings(sendTimes)
		schedule.SendTimes = sendTimes
		result = append(result, schedule)
	}
	return result
}

func applyLegacyDashboardReportEmailSchedule(config *EmailSettingsConfig) {
	if config == nil || len(config.DashboardReportEmailSchedules) == 0 {
		return
	}
	first := config.DashboardReportEmailSchedules[0]
	config.DashboardReportEmailFrequency = first.Frequency
	config.DashboardReportEmailSendTime = first.SendTimes[0]
	config.DashboardReportEmailWeekday = first.Weekday
	config.DashboardReportEmailMonthDay = first.MonthDay
}

func loadDashboardReportDispatchHistory() dashboardReportDispatchHistory {
	history := dashboardReportDispatchHistory{}
	raw := emailOptionString(common.DashboardReportEmailDispatchHistoryOptionKey)
	if raw == "" {
		return history
	}
	if err := common.UnmarshalJsonStr(raw, &history); err != nil {
		common.SysError("failed to load dashboard report dispatch history: " + err.Error())
		return dashboardReportDispatchHistory{}
	}
	return history
}

func persistDashboardReportDispatchHistory(history dashboardReportDispatchHistory, now time.Time) error {
	oldest := now.AddDate(0, 0, -90).Unix()
	for key, sentAt := range history {
		if sentAt < oldest {
			delete(history, key)
		}
	}
	data, err := common.Marshal(history)
	if err != nil {
		return err
	}
	return model.UpdateOption(common.DashboardReportEmailDispatchHistoryOptionKey, string(data))
}

func newDashboardReportPeriod(frequency string, start, end time.Time) dashboardReportPeriod {
	period := dashboardReportPeriod{Start: start, End: end, PeriodKey: frequency + ":" + start.Format("2006-01-02")}
	switch frequency {
	case DashboardReportFrequencyWeekly:
		period.TypeZH, period.TypeEN = "周报", "Weekly"
	case DashboardReportFrequencyMonthly:
		period.TypeZH, period.TypeEN = "月报", "Monthly"
	default:
		period.TypeZH, period.TypeEN = "日报", "Daily"
	}
	return period
}

func dashboardReportTemplateValues(report model.DashboardReportData, period dashboardReportPeriod, locale string) map[string]string {
	reportType := period.TypeZH
	if locale == EmailTemplateLocaleEnglish {
		reportType = period.TypeEN
	}
	topModels := make([]string, 0, len(report.TopModels))
	for index, item := range report.TopModels {
		topModels = append(topModels, fmt.Sprintf("%d. %s  %s", index+1, item.ModelName, formatDashboardReportConsumption(item.Quota)))
	}
	if len(topModels) == 0 {
		if locale == EmailTemplateLocaleEnglish {
			topModels = append(topModels, "No usage data")
		} else {
			topModels = append(topModels, "暂无使用数据")
		}
	}
	return map[string]string{
		"report_type":       reportType,
		"report_period":     formatDashboardReportPeriod(period),
		"generated_at":      time.Now().In(period.Start.Location()).Format("2006-01-02 15:04:05"),
		"total_consumption": formatDashboardReportConsumption(report.Quota),
		"total_quota":       formatDashboardReportInteger(report.Quota),
		"total_requests":    formatDashboardReportInteger(report.Count),
		"total_tokens":      formatDashboardReportInteger(report.TokenUsed),
		"active_users":      formatDashboardReportInteger(report.UserCount),
		"active_models":     formatDashboardReportInteger(report.ModelCount),
		"active_channels":   formatDashboardReportInteger(report.ChannelCount),
		"active_groups":     formatDashboardReportInteger(report.GroupCount),
		"top_models":        strings.Join(topModels, "\n"),
	}
}

func formatDashboardReportPeriod(period dashboardReportPeriod) string {
	return period.Start.Format("2006-01-02 15:04") + " - " + period.End.Format("2006-01-02 15:04")
}

func formatDashboardReportConsumption(quota int64) string {
	if common.QuotaPerUnit <= 0 {
		return "$0.00"
	}
	return fmt.Sprintf("$%.2f", float64(quota)/common.QuotaPerUnit)
}

func formatDashboardReportInteger(value int64) string {
	raw := strconv.FormatInt(value, 10)
	for index := len(raw) - 3; index > 0; index -= 3 {
		raw = raw[:index] + "," + raw[index:]
	}
	return raw
}
