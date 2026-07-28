package service

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"

	"github.com/bytedance/gopkg/util/gopool"
)

const maxOperationalEmailRecipients = 100

type EmailSettingsConfig struct {
	SubscriptionExpiryReminderEnabled bool                           `json:"subscription_expiry_reminder_enabled"`
	LowBalanceEmailEnabled            bool                           `json:"low_balance_email_enabled"`
	LowBalanceEmailThreshold          int                            `json:"low_balance_email_threshold"`
	LowBalanceEmailRechargeURL        string                         `json:"low_balance_email_recharge_url"`
	AccountQuotaEmailEnabled          bool                           `json:"account_quota_email_enabled"`
	AccountQuotaEmailThreshold        float64                        `json:"account_quota_email_threshold"`
	AccountQuotaEmailRecipientUserIDs []int                          `json:"account_quota_email_recipient_user_ids"`
	ChannelAnomalyEmailEnabled        bool                           `json:"channel_anomaly_email_enabled"`
	ChannelAnomalyEmailRecipientIDs   []int                          `json:"channel_anomaly_email_recipient_user_ids"`
	DashboardReportEmailEnabled       bool                           `json:"dashboard_report_email_enabled"`
	DashboardReportEmailFrequency     string                         `json:"dashboard_report_email_frequency"`
	DashboardReportEmailSendTime      string                         `json:"dashboard_report_email_send_time"`
	DashboardReportEmailWeekday       int                            `json:"dashboard_report_email_weekday"`
	DashboardReportEmailMonthDay      int                            `json:"dashboard_report_email_month_day"`
	DashboardReportEmailRecipientIDs  []int                          `json:"dashboard_report_email_recipient_user_ids"`
	DashboardReportEmailSchedules     []DashboardReportEmailSchedule `json:"dashboard_report_email_schedules"`
	RiskUserEmailEnabled              bool                           `json:"risk_user_email_enabled"`
	RiskUserEmailLevels               []string                       `json:"risk_user_email_levels"`
	RiskUserEmailRecipientIDs         []int                          `json:"risk_user_email_recipient_user_ids"`
}

func GetEmailSettingsConfig() (EmailSettingsConfig, error) {
	config := EmailSettingsConfig{
		SubscriptionExpiryReminderEnabled: emailOptionBool(common.SubscriptionExpiryReminderEnabledOptionKey, false),
		LowBalanceEmailEnabled:            emailOptionBool(common.LowBalanceEmailEnabledOptionKey, false),
		LowBalanceEmailThreshold:          emailOptionInt(common.LowBalanceEmailThresholdOptionKey, common.QuotaRemindThreshold),
		LowBalanceEmailRechargeURL:        emailOptionString(common.LowBalanceEmailRechargeURLOptionKey),
		AccountQuotaEmailEnabled:          emailOptionBool(common.AccountQuotaEmailEnabledOptionKey, false),
		AccountQuotaEmailThreshold:        emailOptionFloat(common.AccountQuotaEmailThresholdOptionKey, 5),
		ChannelAnomalyEmailEnabled:        emailOptionBool(common.ChannelAnomalyEmailEnabledOptionKey, false),
		DashboardReportEmailEnabled:       emailOptionBool(common.DashboardReportEmailEnabledOptionKey, false),
		DashboardReportEmailFrequency:     emailOptionString(common.DashboardReportEmailFrequencyOptionKey),
		DashboardReportEmailSendTime:      emailOptionString(common.DashboardReportEmailSendTimeOptionKey),
		DashboardReportEmailWeekday:       emailOptionInt(common.DashboardReportEmailWeekdayOptionKey, 1),
		DashboardReportEmailMonthDay:      emailOptionInt(common.DashboardReportEmailMonthDayOptionKey, 1),
		RiskUserEmailEnabled:              emailOptionBool(common.RiskUserEmailEnabledOptionKey, false),
	}
	if err := decodeRiskUserEmailLevels(&config.RiskUserEmailLevels); err != nil {
		return EmailSettingsConfig{}, err
	}
	if config.DashboardReportEmailFrequency == "" {
		config.DashboardReportEmailFrequency = DashboardReportFrequencyDaily
	}
	if config.DashboardReportEmailSendTime == "" {
		config.DashboardReportEmailSendTime = "08:00"
	}
	if err := decodeDashboardReportEmailSchedules(&config.DashboardReportEmailSchedules); err != nil {
		return EmailSettingsConfig{}, err
	}
	if len(config.DashboardReportEmailSchedules) == 0 {
		config.DashboardReportEmailSchedules = []DashboardReportEmailSchedule{{
			Frequency: config.DashboardReportEmailFrequency,
			SendTimes: []string{config.DashboardReportEmailSendTime},
			Weekday:   config.DashboardReportEmailWeekday,
			MonthDay:  config.DashboardReportEmailMonthDay,
		}}
	}
	config.DashboardReportEmailSchedules = normalizeDashboardReportEmailSchedules(config.DashboardReportEmailSchedules)
	applyLegacyDashboardReportEmailSchedule(&config)
	if err := decodeEmailRecipientIDs(common.AccountQuotaEmailRecipientUserIDsOptionKey, &config.AccountQuotaEmailRecipientUserIDs); err != nil {
		return EmailSettingsConfig{}, err
	}
	if err := decodeEmailRecipientIDs(common.ChannelAnomalyEmailRecipientUserIDsOptionKey, &config.ChannelAnomalyEmailRecipientIDs); err != nil {
		return EmailSettingsConfig{}, err
	}
	if err := decodeEmailRecipientIDs(common.DashboardReportEmailRecipientUserIDsOptionKey, &config.DashboardReportEmailRecipientIDs); err != nil {
		return EmailSettingsConfig{}, err
	}
	if err := decodeEmailRecipientIDs(common.RiskUserEmailRecipientUserIDsOptionKey, &config.RiskUserEmailRecipientIDs); err != nil {
		return EmailSettingsConfig{}, err
	}
	if len(config.AccountQuotaEmailRecipientUserIDs) == 0 || len(config.ChannelAnomalyEmailRecipientIDs) == 0 || len(config.DashboardReportEmailRecipientIDs) == 0 || len(config.RiskUserEmailRecipientIDs) == 0 {
		rootRecipients, err := model.GetOperationalEmailRecipientUsers(nil)
		if err != nil {
			return EmailSettingsConfig{}, err
		}
		rootIDs := make([]int, 0, len(rootRecipients))
		for _, recipient := range rootRecipients {
			rootIDs = append(rootIDs, recipient.Id)
		}
		if len(config.AccountQuotaEmailRecipientUserIDs) == 0 {
			config.AccountQuotaEmailRecipientUserIDs = append([]int{}, rootIDs...)
		}
		if len(config.ChannelAnomalyEmailRecipientIDs) == 0 {
			config.ChannelAnomalyEmailRecipientIDs = append([]int{}, rootIDs...)
		}
		if len(config.DashboardReportEmailRecipientIDs) == 0 {
			config.DashboardReportEmailRecipientIDs = append([]int{}, rootIDs...)
		}
		if len(config.RiskUserEmailRecipientIDs) == 0 {
			config.RiskUserEmailRecipientIDs = append([]int{}, rootIDs...)
		}
	}
	return config, nil
}

func UpdateEmailSettingsConfig(config EmailSettingsConfig) (EmailSettingsConfig, error) {
	config.LowBalanceEmailRechargeURL = strings.TrimSpace(config.LowBalanceEmailRechargeURL)
	if len(config.DashboardReportEmailSchedules) == 0 {
		config.DashboardReportEmailSchedules = []DashboardReportEmailSchedule{{
			Frequency: config.DashboardReportEmailFrequency,
			SendTimes: []string{config.DashboardReportEmailSendTime},
			Weekday:   config.DashboardReportEmailWeekday,
			MonthDay:  config.DashboardReportEmailMonthDay,
		}}
	}
	config.DashboardReportEmailSchedules = normalizeDashboardReportEmailSchedules(config.DashboardReportEmailSchedules)
	applyLegacyDashboardReportEmailSchedule(&config)
	config.AccountQuotaEmailRecipientUserIDs = normalizeEmailRecipientIDs(config.AccountQuotaEmailRecipientUserIDs)
	config.ChannelAnomalyEmailRecipientIDs = normalizeEmailRecipientIDs(config.ChannelAnomalyEmailRecipientIDs)
	config.DashboardReportEmailRecipientIDs = normalizeEmailRecipientIDs(config.DashboardReportEmailRecipientIDs)
	config.RiskUserEmailLevels = normalizeRiskUserEmailLevels(config.RiskUserEmailLevels)
	config.RiskUserEmailRecipientIDs = normalizeEmailRecipientIDs(config.RiskUserEmailRecipientIDs)
	if config.LowBalanceEmailThreshold < 0 {
		return EmailSettingsConfig{}, errors.New("low balance threshold cannot be negative")
	}
	if math.IsNaN(config.AccountQuotaEmailThreshold) || math.IsInf(config.AccountQuotaEmailThreshold, 0) || config.AccountQuotaEmailThreshold < 0 {
		return EmailSettingsConfig{}, errors.New("account quota threshold must be a non-negative number")
	}
	if config.LowBalanceEmailRechargeURL != "" {
		parsed, err := url.ParseRequestURI(config.LowBalanceEmailRechargeURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return EmailSettingsConfig{}, errors.New("recharge URL must be a valid HTTP or HTTPS URL")
		}
	}
	if err := validateDashboardReportEmailSchedules(config.DashboardReportEmailSchedules); err != nil {
		return EmailSettingsConfig{}, err
	}
	if config.RiskUserEmailEnabled && len(config.RiskUserEmailLevels) == 0 {
		return EmailSettingsConfig{}, errors.New("at least one risk level is required")
	}
	if len(config.AccountQuotaEmailRecipientUserIDs) > maxOperationalEmailRecipients || len(config.ChannelAnomalyEmailRecipientIDs) > maxOperationalEmailRecipients || len(config.DashboardReportEmailRecipientIDs) > maxOperationalEmailRecipients || len(config.RiskUserEmailRecipientIDs) > maxOperationalEmailRecipients {
		return EmailSettingsConfig{}, fmt.Errorf("recipient count cannot exceed %d", maxOperationalEmailRecipients)
	}
	if err := validateOperationalEmailRecipientIDs(config.AccountQuotaEmailRecipientUserIDs); err != nil {
		return EmailSettingsConfig{}, err
	}
	if err := validateOperationalEmailRecipientIDs(config.ChannelAnomalyEmailRecipientIDs); err != nil {
		return EmailSettingsConfig{}, err
	}
	if err := validateOperationalEmailRecipientIDs(config.DashboardReportEmailRecipientIDs); err != nil {
		return EmailSettingsConfig{}, err
	}
	if err := validateOperationalEmailRecipientIDs(config.RiskUserEmailRecipientIDs); err != nil {
		return EmailSettingsConfig{}, err
	}

	accountRecipientJSON, err := common.Marshal(config.AccountQuotaEmailRecipientUserIDs)
	if err != nil {
		return EmailSettingsConfig{}, err
	}
	channelRecipientJSON, err := common.Marshal(config.ChannelAnomalyEmailRecipientIDs)
	if err != nil {
		return EmailSettingsConfig{}, err
	}
	dashboardReportRecipientJSON, err := common.Marshal(config.DashboardReportEmailRecipientIDs)
	if err != nil {
		return EmailSettingsConfig{}, err
	}
	dashboardReportSchedulesJSON, err := common.Marshal(config.DashboardReportEmailSchedules)
	if err != nil {
		return EmailSettingsConfig{}, err
	}
	riskUserLevelsJSON, err := common.Marshal(config.RiskUserEmailLevels)
	if err != nil {
		return EmailSettingsConfig{}, err
	}
	riskUserRecipientJSON, err := common.Marshal(config.RiskUserEmailRecipientIDs)
	if err != nil {
		return EmailSettingsConfig{}, err
	}
	values := map[string]string{
		common.SubscriptionExpiryReminderEnabledOptionKey:    strconv.FormatBool(config.SubscriptionExpiryReminderEnabled),
		common.LowBalanceEmailEnabledOptionKey:               strconv.FormatBool(config.LowBalanceEmailEnabled),
		common.LowBalanceEmailThresholdOptionKey:             strconv.Itoa(config.LowBalanceEmailThreshold),
		common.LowBalanceEmailRechargeURLOptionKey:           config.LowBalanceEmailRechargeURL,
		common.AccountQuotaEmailEnabledOptionKey:             strconv.FormatBool(config.AccountQuotaEmailEnabled),
		common.AccountQuotaEmailThresholdOptionKey:           strconv.FormatFloat(config.AccountQuotaEmailThreshold, 'f', -1, 64),
		common.AccountQuotaEmailRecipientUserIDsOptionKey:    string(accountRecipientJSON),
		common.ChannelAnomalyEmailEnabledOptionKey:           strconv.FormatBool(config.ChannelAnomalyEmailEnabled),
		common.ChannelAnomalyEmailRecipientUserIDsOptionKey:  string(channelRecipientJSON),
		common.DashboardReportEmailEnabledOptionKey:          strconv.FormatBool(config.DashboardReportEmailEnabled),
		common.DashboardReportEmailFrequencyOptionKey:        config.DashboardReportEmailFrequency,
		common.DashboardReportEmailSendTimeOptionKey:         config.DashboardReportEmailSendTime,
		common.DashboardReportEmailWeekdayOptionKey:          strconv.Itoa(config.DashboardReportEmailWeekday),
		common.DashboardReportEmailMonthDayOptionKey:         strconv.Itoa(config.DashboardReportEmailMonthDay),
		common.DashboardReportEmailRecipientUserIDsOptionKey: string(dashboardReportRecipientJSON),
		common.DashboardReportEmailSchedulesOptionKey:        string(dashboardReportSchedulesJSON),
		common.RiskUserEmailEnabledOptionKey:                 strconv.FormatBool(config.RiskUserEmailEnabled),
		common.RiskUserEmailLevelsOptionKey:                  string(riskUserLevelsJSON),
		common.RiskUserEmailRecipientUserIDsOptionKey:        string(riskUserRecipientJSON),
	}
	if err := model.UpdateOptionsBulk(values); err != nil {
		return EmailSettingsConfig{}, err
	}
	return GetEmailSettingsConfig()
}

func IsLowBalanceEmailEnabled() bool {
	return emailOptionBool(common.LowBalanceEmailEnabledOptionKey, false)
}

func LowBalanceEmailThreshold() int {
	return emailOptionInt(common.LowBalanceEmailThresholdOptionKey, common.QuotaRemindThreshold)
}

func LowBalanceEmailRechargeURL() string {
	configured := emailOptionString(common.LowBalanceEmailRechargeURLOptionKey)
	if configured != "" {
		return configured
	}
	return PaymentReturnURL("/wallet")
}

func NotifyAccountQuotaEmail(channel *model.Channel, previousBalance float64, previousUpdatedAt int64, currentBalance float64) {
	if channel == nil {
		return
	}
	config, err := GetEmailSettingsConfig()
	if err != nil || !config.AccountQuotaEmailEnabled {
		return
	}
	threshold := config.AccountQuotaEmailThreshold
	if !shouldSendAccountQuotaEmail(previousBalance, previousUpdatedAt, currentBalance, threshold) {
		return
	}
	recipients, err := model.GetOperationalEmailRecipientUsers(config.AccountQuotaEmailRecipientUserIDs)
	if err != nil || len(recipients) == 0 {
		if err != nil {
			common.SysError("failed to load account quota email recipients: " + err.Error())
		}
		return
	}
	values := map[string]string{
		"channel_id":        strconv.Itoa(channel.Id),
		"channel_name":      channel.Name,
		"channel_type":      constant.GetChannelTypeName(channel.Type),
		"current_balance":   fmt.Sprintf("$%.4f", currentBalance),
		"warning_threshold": fmt.Sprintf("$%.4f", threshold),
		"checked_at":        time.Now().Format("2006-01-02 15:04:05"),
	}
	gopool.Go(func() {
		sendOperationalTemplateEmails(EmailTemplateEventAccountQuotaAlert, recipients, values)
	})
}

func shouldSendAccountQuotaEmail(previousBalance float64, previousUpdatedAt int64, currentBalance, threshold float64) bool {
	if currentBalance > threshold {
		return false
	}
	return previousUpdatedAt == 0 || previousBalance > threshold
}

func NotifyChannelAnomalyEmail(channelError types.ChannelError, reason string) bool {
	config, err := GetEmailSettingsConfig()
	if err != nil || !config.ChannelAnomalyEmailEnabled {
		return false
	}
	recipients, err := model.GetOperationalEmailRecipientUsers(config.ChannelAnomalyEmailRecipientIDs)
	if err != nil || len(recipients) == 0 {
		if err != nil {
			common.SysError("failed to load channel anomaly email recipients: " + err.Error())
		}
		return false
	}
	baseURL := ""
	if channel, getErr := model.GetChannelById(channelError.ChannelId, false); getErr == nil {
		baseURL = channel.GetBaseURL()
	}
	values := map[string]string{
		"channel_id":       strconv.Itoa(channelError.ChannelId),
		"channel_name":     channelError.ChannelName,
		"channel_type":     constant.GetChannelTypeName(channelError.ChannelType),
		"channel_base_url": baseURL,
		"failure_reason":   reason,
		"disabled_at":      time.Now().Format("2006-01-02 15:04:05"),
	}
	gopool.Go(func() {
		sendOperationalTemplateEmails(EmailTemplateEventChannelAnomalyDisabled, recipients, values)
	})
	return true
}

func SendChannelAnomalyTestEmails(userIDs []int) (int, error) {
	return sendChannelAnomalyTestEmails(userIDs, common.SendEmail)
}

func sendChannelAnomalyTestEmails(userIDs []int, sender EmailCampaignSender) (int, error) {
	if sender == nil {
		return 0, errors.New("email sender is required")
	}
	userIDs = normalizeEmailRecipientIDs(userIDs)
	if len(userIDs) > maxOperationalEmailRecipients {
		return 0, fmt.Errorf("recipient count cannot exceed %d", maxOperationalEmailRecipients)
	}
	if err := validateOperationalEmailRecipientIDs(userIDs); err != nil {
		return 0, err
	}
	recipients, err := model.GetOperationalEmailRecipientUsers(userIDs)
	if err != nil {
		return 0, err
	}
	if len(recipients) == 0 {
		return 0, errors.New("no active administrator or root recipient with an email address")
	}

	sent := 0
	for _, recipient := range recipients {
		values := map[string]string{
			"channel_id":       "0",
			"channel_name":     "渠道异常通知测试",
			"channel_type":     "OpenAI",
			"channel_base_url": "https://example.com",
			"failure_reason":   "这是一封测试邮件，用于验证渠道异常通知配置。",
			"disabled_at":      time.Now().Format("2006-01-02 15:04:05"),
		}
		if NormalizeEmailTemplateLocale(recipient.GetSetting().Language) == EmailTemplateLocaleEnglish {
			values["channel_name"] = "Channel anomaly notification test"
			values["failure_reason"] = "This is a test email used to verify the channel anomaly notification configuration."
		}
		rendered, err := renderOperationalTemplateEmail(EmailTemplateEventChannelAnomalyDisabled, recipient, values)
		if err != nil {
			return sent, err
		}
		if err := sender(rendered.Subject, recipient.Email, rendered.Content); err != nil {
			return sent, fmt.Errorf("failed to send channel anomaly test email to user %d: %w", recipient.Id, err)
		}
		sent++
	}
	return sent, nil
}

func sendOperationalTemplateEmails(event string, recipients []model.User, values map[string]string) {
	for _, recipient := range recipients {
		rendered, err := renderOperationalTemplateEmail(event, recipient, values)
		if err != nil {
			common.SysError(fmt.Sprintf("failed to render operational email event %s for user %d: %v", event, recipient.Id, err))
			continue
		}
		if err := common.SendEmail(rendered.Subject, recipient.Email, rendered.Content); err != nil {
			common.SysError(fmt.Sprintf("failed to send operational email event %s to user %d: %v", event, recipient.Id, err))
		}
	}
}

func renderOperationalTemplateEmail(event string, recipient model.User, values map[string]string) (EmailTemplatePreview, error) {
	displayName := strings.TrimSpace(recipient.DisplayName)
	if displayName == "" {
		displayName = recipient.Username
	}
	recipientValues := make(map[string]string, len(values)+4)
	for key, value := range values {
		recipientValues[key] = value
	}
	recipientValues["username"] = recipient.Username
	recipientValues["display_name"] = displayName
	recipientValues["email"] = recipient.Email
	return RenderEmailTemplateForLocale(event, recipient.GetSetting().Language, recipientValues)
}

func emailOptionString(key string) string {
	common.OptionMapRWMutex.RLock()
	value := common.OptionMap[key]
	common.OptionMapRWMutex.RUnlock()
	return strings.TrimSpace(value)
}

func emailOptionBool(key string, fallback bool) bool {
	value, err := strconv.ParseBool(emailOptionString(key))
	if err != nil {
		return fallback
	}
	return value
}

func emailOptionInt(key string, fallback int) int {
	value, err := strconv.Atoi(emailOptionString(key))
	if err != nil {
		return fallback
	}
	return value
}

func emailOptionFloat(key string, fallback float64) float64 {
	value, err := strconv.ParseFloat(emailOptionString(key), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return fallback
	}
	return value
}

func decodeEmailRecipientIDs(key string, target *[]int) error {
	raw := emailOptionString(key)
	if raw == "" {
		*target = []int{}
		return nil
	}
	if err := common.UnmarshalJsonStr(raw, target); err != nil {
		return fmt.Errorf("invalid email recipient setting %s: %w", key, err)
	}
	*target = normalizeEmailRecipientIDs(*target)
	return nil
}

func normalizeEmailRecipientIDs(userIDs []int) []int {
	seen := make(map[int]struct{}, len(userIDs))
	result := make([]int, 0, len(userIDs))
	for _, userID := range userIDs {
		if userID <= 0 {
			continue
		}
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}
		result = append(result, userID)
	}
	sort.Ints(result)
	return result
}

func validateOperationalEmailRecipientIDs(userIDs []int) error {
	if len(userIDs) == 0 {
		return nil
	}
	users, err := model.GetOperationalEmailRecipientOptionsByIDs(userIDs)
	if err != nil {
		return err
	}
	if len(users) != len(userIDs) {
		return errors.New("all operational email recipients must be active administrators with an email address")
	}
	return nil
}
