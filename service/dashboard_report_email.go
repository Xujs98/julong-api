package service

import (
	"context"
	"errors"
	"fmt"
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
)

type DashboardReportEmailDispatchResult struct {
	RecipientCount int    `json:"recipient_count"`
	Period         string `json:"period"`
}

type dashboardReportPeriod struct {
	Start     time.Time
	End       time.Time
	PeriodKey string
	TypeZH    string
	TypeEN    string
}

func validateDashboardReportEmailSchedule(config EmailSettingsConfig) error {
	switch config.DashboardReportEmailFrequency {
	case DashboardReportFrequencyDaily, DashboardReportFrequencyWeekly, DashboardReportFrequencyMonthly:
	default:
		return errors.New("dashboard report frequency must be daily, weekly, or monthly")
	}
	if _, err := time.Parse("15:04", config.DashboardReportEmailSendTime); err != nil {
		return errors.New("dashboard report send time must use HH:mm format")
	}
	if config.DashboardReportEmailWeekday < 1 || config.DashboardReportEmailWeekday > 7 {
		return errors.New("dashboard report weekday must be between 1 and 7")
	}
	if config.DashboardReportEmailMonthDay < 1 || config.DashboardReportEmailMonthDay > 31 {
		return errors.New("dashboard report month day must be between 1 and 31")
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
	period, due := dashboardReportPeriodForSchedule(config, now)
	return due && emailOptionString(common.DashboardReportEmailLastPeriodOptionKey) != period.PeriodKey
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
	config, err := GetEmailSettingsConfig()
	if err != nil {
		return DashboardReportEmailDispatchResult{}, err
	}
	period, due := dashboardReportPeriodForSchedule(config, time.Now())
	if !config.DashboardReportEmailEnabled || !due || emailOptionString(common.DashboardReportEmailLastPeriodOptionKey) == period.PeriodKey {
		return DashboardReportEmailDispatchResult{}, nil
	}
	result, err := sendDashboardReportEmails(ctx, period, config.DashboardReportEmailRecipientIDs, sender)
	if err != nil {
		return result, err
	}
	if err := model.UpdateOption(common.DashboardReportEmailLastPeriodOptionKey, period.PeriodKey); err != nil {
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

func dashboardReportPeriodForSchedule(config EmailSettingsConfig, now time.Time) (dashboardReportPeriod, bool) {
	sendTime, err := time.Parse("15:04", config.DashboardReportEmailSendTime)
	if err != nil || now.Hour() < sendTime.Hour() || (now.Hour() == sendTime.Hour() && now.Minute() < sendTime.Minute()) {
		return dashboardReportPeriod{}, false
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch config.DashboardReportEmailFrequency {
	case DashboardReportFrequencyDaily:
		start := today.AddDate(0, 0, -1)
		return newDashboardReportPeriod(config.DashboardReportEmailFrequency, start, today), true
	case DashboardReportFrequencyWeekly:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		if weekday != config.DashboardReportEmailWeekday {
			return dashboardReportPeriod{}, false
		}
		end := today.AddDate(0, 0, -(weekday - 1))
		return newDashboardReportPeriod(config.DashboardReportEmailFrequency, end.AddDate(0, 0, -7), end), true
	case DashboardReportFrequencyMonthly:
		lastDay := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()
		scheduledDay := config.DashboardReportEmailMonthDay
		if scheduledDay > lastDay {
			scheduledDay = lastDay
		}
		if now.Day() != scheduledDay {
			return dashboardReportPeriod{}, false
		}
		end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return newDashboardReportPeriod(config.DashboardReportEmailFrequency, end.AddDate(0, -1, 0), end), true
	default:
		return dashboardReportPeriod{}, false
	}
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
