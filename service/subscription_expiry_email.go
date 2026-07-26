package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const systemSubscriptionExpiryCampaignID int64 = 0

var subscriptionExpiryReminderDays = []int{7, 3, 1}

type SubscriptionExpiryEmailResult struct {
	SentCount    int64 `json:"sent_count"`
	FailedCount  int64 `json:"failed_count"`
	SkippedCount int64 `json:"skipped_count"`
}

func IsSubscriptionExpiryReminderEnabled() bool {
	common.OptionMapRWMutex.RLock()
	value := common.OptionMap[common.SubscriptionExpiryReminderEnabledOptionKey]
	common.OptionMapRWMutex.RUnlock()
	enabled, _ := strconv.ParseBool(value)
	return enabled
}

func DispatchSubscriptionExpiryReminders(ctx context.Context, sender EmailCampaignSender) (SubscriptionExpiryEmailResult, error) {
	result := SubscriptionExpiryEmailResult{}
	if !IsSubscriptionExpiryReminderEnabled() {
		return result, nil
	}
	if sender == nil {
		return result, errors.New("email sender is required")
	}
	now := common.GetTimestamp()
	if err := model.RecoverStaleEmailDeliveries(systemSubscriptionExpiryCampaignID, now-10*60); err != nil {
		return result, err
	}
	if err := model.ResetRetryableEmailDeliveries(systemSubscriptionExpiryCampaignID); err != nil {
		return result, err
	}

	for _, reminderDays := range subscriptionExpiryReminderDays {
		for offset := 0; ; offset += emailCampaignCandidateBatchSize {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			candidates, err := model.ListSubscriptionExpiryReminderCandidates(now, reminderDays, offset, emailCampaignCandidateBatchSize)
			if err != nil {
				return result, err
			}
			deliveries := make([]model.EmailDelivery, 0, len(candidates))
			for _, candidate := range candidates {
				status := model.EmailDeliveryStatusPending
				lastError := ""
				address, parseErr := mail.ParseAddress(strings.TrimSpace(candidate.Email))
				if parseErr != nil || !strings.EqualFold(address.Address, strings.TrimSpace(candidate.Email)) {
					status = model.EmailDeliveryStatusSkipped
					lastError = "invalid email address"
				}
				deliveries = append(deliveries, model.EmailDelivery{
					CampaignId:          systemSubscriptionExpiryCampaignID,
					UserId:              candidate.UserId,
					Email:               strings.TrimSpace(candidate.Email),
					Username:            candidate.Username,
					DisplayName:         candidate.DisplayName,
					SubscriptionId:      candidate.SubscriptionId,
					SubscriptionTitle:   candidate.SubscriptionTitle,
					SubscriptionEndTime: candidate.SubscriptionEndTime,
					DedupeKey:           fmt.Sprintf("system:subscription_expiry:%d:%d:%d:%d", reminderDays, candidate.UserId, candidate.SubscriptionId, candidate.SubscriptionEndTime),
					Status:              status,
					LastError:           lastError,
				})
			}
			if err := model.CreateEmailDeliveries(deliveries); err != nil {
				return result, err
			}
			if len(candidates) < emailCampaignCandidateBatchSize {
				break
			}
		}
	}

	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		deliveries, err := model.ListPendingEmailDeliveries(systemSubscriptionExpiryCampaignID, emailCampaignDeliveryBatchSize)
		if err != nil {
			return result, err
		}
		if len(deliveries) == 0 {
			break
		}
		for _, delivery := range deliveries {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			claimed, err := model.ClaimEmailDelivery(delivery.Id)
			if err != nil {
				return result, err
			}
			if !claimed {
				continue
			}
			current, err := model.IsSubscriptionExpiryReminderCurrent(delivery.SubscriptionId, delivery.UserId, delivery.SubscriptionEndTime, now)
			if err != nil {
				return result, err
			}
			if !current {
				if err := model.FinishEmailDelivery(delivery.Id, model.EmailDeliveryStatusSkipped, "subscription is no longer active"); err != nil {
					return result, err
				}
				result.SkippedCount++
				continue
			}

			displayName := strings.TrimSpace(delivery.DisplayName)
			if displayName == "" {
				displayName = delivery.Username
			}
			daysRemaining := int64(0)
			if seconds := delivery.SubscriptionEndTime - now; seconds > 0 {
				daysRemaining = (seconds + 24*60*60 - 1) / (24 * 60 * 60)
			}
			locale := EmailTemplateLocaleChinese
			if user, userErr := model.GetUserById(delivery.UserId, false); userErr == nil {
				locale = NormalizeEmailTemplateLocale(user.GetSetting().Language)
			}
			rendered, err := RenderEmailTemplateForLocale(EmailTemplateEventSubscriptionExpiryReminder, locale, map[string]string{
				"username":              delivery.Username,
				"display_name":          displayName,
				"email":                 delivery.Email,
				"subscription_name":     delivery.SubscriptionTitle,
				"subscription_end_time": time.Unix(delivery.SubscriptionEndTime, 0).Format("2006-01-02 15:04:05"),
				"days_remaining":        strconv.FormatInt(daysRemaining, 10),
			})
			if err != nil {
				return result, err
			}
			if err := sender(rendered.Subject, delivery.Email, rendered.Content); err != nil {
				if updateErr := model.FinishEmailDelivery(delivery.Id, model.EmailDeliveryStatusFailed, err.Error()); updateErr != nil {
					return result, updateErr
				}
				result.FailedCount++
				continue
			}
			if err := model.FinishEmailDelivery(delivery.Id, model.EmailDeliveryStatusSent, ""); err != nil {
				return result, err
			}
			result.SentCount++
		}
	}
	return result, nil
}
