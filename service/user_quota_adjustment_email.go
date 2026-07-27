package service

import (
	"net/mail"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
)

type UserQuotaAdjustmentEmailResult struct {
	SuccessCount int `json:"success_count"`
	SkippedCount int `json:"skipped_count"`
	FailedCount  int `json:"failed_count"`
}

type userQuotaAdjustmentEmailSender func(subject, receiver, content string) error

func ValidateUserQuotaAdjustmentEmail(locale, subject, content string) error {
	_, err := RenderCustomEmailTemplateForLocale(
		EmailTemplateEventUserQuotaAdjustment,
		locale,
		subject,
		content,
		emailTemplateSampleValues(NormalizeEmailTemplateLocale(locale)),
	)
	return err
}

func SendUserQuotaAdjustmentEmails(adjustments []model.UserQuotaAdjustment, operatorName, locale, subject, content string) UserQuotaAdjustmentEmailResult {
	return sendUserQuotaAdjustmentEmails(adjustments, operatorName, locale, subject, content, common.SendEmail)
}

func sendUserQuotaAdjustmentEmails(adjustments []model.UserQuotaAdjustment, operatorName, locale, subject, content string, sender userQuotaAdjustmentEmailSender) UserQuotaAdjustmentEmailResult {
	result := UserQuotaAdjustmentEmailResult{}
	locale = NormalizeEmailTemplateLocale(locale)
	adjustedAt := time.Now().Format("2006-01-02 15:04:05")
	for _, adjustment := range adjustments {
		receiver := strings.TrimSpace(adjustment.Email)
		address, err := mail.ParseAddress(receiver)
		if err != nil || !strings.EqualFold(address.Address, receiver) {
			result.SkippedCount++
			continue
		}
		displayName := strings.TrimSpace(adjustment.DisplayName)
		if displayName == "" {
			displayName = adjustment.Username
		}
		operation := adjustment.Mode
		switch adjustment.Mode {
		case model.UserQuotaAdjustModeAdd:
			if locale == EmailTemplateLocaleChinese {
				operation = "增加"
			} else {
				operation = "increase"
			}
		case model.UserQuotaAdjustModeSubtract:
			if locale == EmailTemplateLocaleChinese {
				operation = "减少"
			} else {
				operation = "decrease"
			}
		case model.UserQuotaAdjustModeOverride:
			if locale == EmailTemplateLocaleChinese {
				operation = "覆盖"
			} else {
				operation = "override"
			}
		}
		rendered, err := RenderCustomEmailTemplateForLocale(
			EmailTemplateEventUserQuotaAdjustment,
			locale,
			subject,
			content,
			map[string]string{
				"username":          adjustment.Username,
				"display_name":      displayName,
				"email":             receiver,
				"operation":         operation,
				"adjustment_amount": logger.FormatQuota(adjustment.Value),
				"previous_quota":    logger.FormatQuota(adjustment.PreviousQuota),
				"current_quota":     logger.FormatQuota(adjustment.CurrentQuota),
				"operator_name":     operatorName,
				"adjusted_at":       adjustedAt,
			},
		)
		if err != nil {
			result.FailedCount++
			continue
		}
		if err := sender(rendered.Subject, receiver, rendered.Content); err != nil {
			common.SysError("failed to send quota adjustment email: " + err.Error())
			result.FailedCount++
			continue
		}
		result.SuccessCount++
	}
	return result
}
