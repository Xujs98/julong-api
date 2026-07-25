package service

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	emailCampaignCandidateBatchSize = 500
	emailCampaignDeliveryBatchSize  = 100
)

type EmailCampaignDispatchResult struct {
	CampaignCount  int   `json:"campaign_count"`
	RecipientCount int64 `json:"recipient_count"`
	SuccessCount   int64 `json:"success_count"`
	FailedCount    int64 `json:"failed_count"`
	SkippedCount   int64 `json:"skipped_count"`
}

type EmailCampaignSender func(subject, receiver, content string) error

func DispatchDueEmailCampaigns(ctx context.Context, sender EmailCampaignSender) (EmailCampaignDispatchResult, error) {
	result := EmailCampaignDispatchResult{}
	if sender == nil {
		return result, errors.New("email sender is required")
	}
	now := common.GetTimestamp()
	if err := model.RecoverStaleEmailCampaigns(now - 10*60); err != nil {
		return result, err
	}
	campaigns, err := model.ClaimDueEmailCampaigns(now, 10)
	if err != nil {
		return result, err
	}
	for _, campaign := range campaigns {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		stats, runErr := dispatchEmailCampaign(ctx, campaign, sender, now)
		if runErr != nil {
			_ = model.FailEmailCampaignRun(campaign, runErr.Error(), common.GetTimestamp())
			return result, runErr
		}
		if err := model.FinishEmailCampaignRun(campaign, stats, common.GetTimestamp()); err != nil {
			return result, err
		}
		result.CampaignCount++
		result.RecipientCount += stats.RecipientCount
		result.SuccessCount += stats.SuccessCount
		result.FailedCount += stats.FailedCount
		result.SkippedCount += stats.SkippedCount
	}
	return result, nil
}

func PreviewEmailCampaignAudience(campaign *model.EmailCampaign) (int64, error) {
	return model.CountEmailRecipientCandidates(campaign, common.GetTimestamp())
}

func dispatchEmailCampaign(ctx context.Context, campaign *model.EmailCampaign, sender EmailCampaignSender, now int64) (model.EmailCampaignDeliveryStats, error) {
	if err := model.RecoverStaleEmailDeliveries(campaign.Id, now-10*60); err != nil {
		return model.EmailCampaignDeliveryStats{}, err
	}
	if campaign.Mode == model.EmailCampaignModeConditional {
		if err := model.ResetRetryableEmailDeliveries(campaign.Id); err != nil {
			return model.EmailCampaignDeliveryStats{}, err
		}
	}

	for offset := 0; ; offset += emailCampaignCandidateBatchSize {
		if err := ctx.Err(); err != nil {
			return model.EmailCampaignDeliveryStats{}, err
		}
		candidates, err := model.ListEmailRecipientCandidates(campaign, now, offset, emailCampaignCandidateBatchSize)
		if err != nil {
			return model.EmailCampaignDeliveryStats{}, err
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
			dedupeKey := fmt.Sprintf("%d:%d", campaign.Id, candidate.UserId)
			if campaign.Mode == model.EmailCampaignModeConditional {
				dedupeKey = fmt.Sprintf("%d:%d:%d:%d", campaign.Id, candidate.UserId, candidate.SubscriptionId, candidate.SubscriptionEndTime)
			}
			deliveries = append(deliveries, model.EmailDelivery{
				CampaignId:          campaign.Id,
				UserId:              candidate.UserId,
				Email:               strings.TrimSpace(candidate.Email),
				Username:            candidate.Username,
				DisplayName:         candidate.DisplayName,
				SubscriptionId:      candidate.SubscriptionId,
				SubscriptionTitle:   candidate.SubscriptionTitle,
				SubscriptionEndTime: candidate.SubscriptionEndTime,
				DedupeKey:           dedupeKey,
				Status:              status,
				LastError:           lastError,
			})
		}
		if err := model.CreateEmailDeliveries(deliveries); err != nil {
			return model.EmailCampaignDeliveryStats{}, err
		}
		if len(candidates) < emailCampaignCandidateBatchSize {
			break
		}
	}

	for {
		if err := ctx.Err(); err != nil {
			return model.EmailCampaignDeliveryStats{}, err
		}
		deliveries, err := model.ListPendingEmailDeliveries(campaign.Id, emailCampaignDeliveryBatchSize)
		if err != nil {
			return model.EmailCampaignDeliveryStats{}, err
		}
		if len(deliveries) == 0 {
			break
		}
		for _, delivery := range deliveries {
			if err := ctx.Err(); err != nil {
				return model.EmailCampaignDeliveryStats{}, err
			}
			claimed, err := model.ClaimEmailDelivery(delivery.Id)
			if err != nil {
				return model.EmailCampaignDeliveryStats{}, err
			}
			if !claimed {
				continue
			}
			subject := renderEmailCampaignTemplate(campaign.Subject, delivery, now, false)
			content := renderEmailCampaignTemplate(campaign.Content, delivery, now, true)
			if err := sender(subject, delivery.Email, content); err != nil {
				if updateErr := model.FinishEmailDelivery(delivery.Id, model.EmailDeliveryStatusFailed, err.Error()); updateErr != nil {
					return model.EmailCampaignDeliveryStats{}, updateErr
				}
				continue
			}
			if err := model.FinishEmailDelivery(delivery.Id, model.EmailDeliveryStatusSent, ""); err != nil {
				return model.EmailCampaignDeliveryStats{}, err
			}
		}
	}
	return model.GetEmailCampaignDeliveryStats(campaign.Id)
}

func renderEmailCampaignTemplate(template string, delivery *model.EmailDelivery, now int64, escapeHTML bool) string {
	daysRemaining := int64(0)
	endTime := ""
	if delivery.SubscriptionEndTime > 0 {
		seconds := delivery.SubscriptionEndTime - now
		if seconds > 0 {
			daysRemaining = (seconds + 24*60*60 - 1) / (24 * 60 * 60)
		}
		endTime = time.Unix(delivery.SubscriptionEndTime, 0).Format("2006-01-02 15:04:05")
	}
	values := map[string]string{
		"{{username}}":              delivery.Username,
		"{{display_name}}":          delivery.DisplayName,
		"{{email}}":                 delivery.Email,
		"{{system_name}}":           common.SystemName,
		"{{subscription_name}}":     delivery.SubscriptionTitle,
		"{{subscription_end_time}}": endTime,
		"{{days_remaining}}":        strconv.FormatInt(daysRemaining, 10),
	}
	result := template
	for placeholder, value := range values {
		if escapeHTML {
			value = html.EscapeString(value)
		}
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}
