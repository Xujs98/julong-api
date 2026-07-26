package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	EmailCampaignModeImmediate   = "immediate"
	EmailCampaignModeScheduled   = "scheduled"
	EmailCampaignModeConditional = "conditional"

	EmailCampaignTargetAllUsers          = "all_users"
	EmailCampaignTargetActiveSubscribers = "active_subscribers"
	EmailCampaignTargetSelectedUsers     = "selected_users"

	EmailCampaignTriggerSubscriptionExpiring = "subscription_expiring"

	EmailCampaignStatusDraft         = "draft"
	EmailCampaignStatusScheduled     = "scheduled"
	EmailCampaignStatusActive        = "active"
	EmailCampaignStatusRunning       = "running"
	EmailCampaignStatusCompleted     = "completed"
	EmailCampaignStatusPartialFailed = "partial_failed"
	EmailCampaignStatusPaused        = "paused"

	EmailDeliveryStatusPending = "pending"
	EmailDeliveryStatusSending = "sending"
	EmailDeliveryStatusSent    = "sent"
	EmailDeliveryStatusFailed  = "failed"
	EmailDeliveryStatusSkipped = "skipped"
)

type EmailCampaign struct {
	Id             int64  `json:"id" gorm:"primaryKey"`
	Name           string `json:"name" gorm:"type:varchar(128);not null"`
	Subject        string `json:"subject" gorm:"type:varchar(255);not null"`
	Content        string `json:"content" gorm:"type:text;not null"`
	Mode           string `json:"mode" gorm:"type:varchar(32);not null;index"`
	TargetType     string `json:"target_type" gorm:"type:varchar(32);not null"`
	TargetUserIds  string `json:"-" gorm:"type:text;column:target_user_ids"`
	TriggerType    string `json:"trigger_type" gorm:"type:varchar(64);default:''"`
	TriggerDays    int    `json:"trigger_days" gorm:"type:int;default:0"`
	ScheduledAt    int64  `json:"scheduled_at" gorm:"type:bigint;default:0"`
	NextRunAt      int64  `json:"next_run_at" gorm:"type:bigint;default:0;index:idx_email_campaign_due,priority:2"`
	LastRunAt      int64  `json:"last_run_at" gorm:"type:bigint;default:0"`
	Status         string `json:"status" gorm:"type:varchar(32);not null;index:idx_email_campaign_due,priority:1"`
	CreatedBy      int    `json:"created_by" gorm:"type:int;not null;index"`
	RecipientCount int64  `json:"recipient_count" gorm:"type:bigint;default:0"`
	SuccessCount   int64  `json:"success_count" gorm:"type:bigint;default:0"`
	FailedCount    int64  `json:"failed_count" gorm:"type:bigint;default:0"`
	SkippedCount   int64  `json:"skipped_count" gorm:"type:bigint;default:0"`
	LastError      string `json:"last_error" gorm:"type:text"`
	CreatedAt      int64  `json:"created_at" gorm:"type:bigint;index"`
	UpdatedAt      int64  `json:"updated_at" gorm:"type:bigint;index"`
}

type EmailDelivery struct {
	Id                  int64  `json:"id" gorm:"primaryKey"`
	CampaignId          int64  `json:"campaign_id" gorm:"type:bigint;not null;index"`
	UserId              int    `json:"user_id" gorm:"type:int;not null;index"`
	Email               string `json:"email" gorm:"type:varchar(255);not null;index"`
	Username            string `json:"username" gorm:"type:varchar(64);default:''"`
	DisplayName         string `json:"display_name" gorm:"type:varchar(64);default:''"`
	SubscriptionId      int    `json:"subscription_id" gorm:"type:int;default:0"`
	SubscriptionTitle   string `json:"subscription_title" gorm:"type:varchar(128);default:''"`
	SubscriptionEndTime int64  `json:"subscription_end_time" gorm:"type:bigint;default:0"`
	DedupeKey           string `json:"-" gorm:"type:varchar(191);not null;uniqueIndex"`
	Status              string `json:"status" gorm:"type:varchar(32);not null;index"`
	AttemptCount        int    `json:"attempt_count" gorm:"type:int;default:0"`
	LastError           string `json:"last_error" gorm:"type:text"`
	SentAt              int64  `json:"sent_at" gorm:"type:bigint;default:0"`
	CreatedAt           int64  `json:"created_at" gorm:"type:bigint;index"`
	UpdatedAt           int64  `json:"updated_at" gorm:"type:bigint;index"`
}

type EmailRecipientCandidate struct {
	UserId              int    `json:"user_id"`
	Email               string `json:"email"`
	Username            string `json:"username"`
	DisplayName         string `json:"display_name"`
	SubscriptionId      int    `json:"subscription_id"`
	SubscriptionTitle   string `json:"subscription_title"`
	SubscriptionEndTime int64  `json:"subscription_end_time"`
}

type EmailCampaignDeliveryStats struct {
	RecipientCount int64
	SuccessCount   int64
	FailedCount    int64
	SkippedCount   int64
}

type EmailCampaignStats struct {
	CampaignCount  int64 `json:"campaign_count"`
	RecipientCount int64 `json:"recipient_count"`
	SuccessCount   int64 `json:"success_count"`
	FailedCount    int64 `json:"failed_count"`
}

type EmailCampaignUserOption struct {
	Id          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

func (campaign *EmailCampaign) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	if campaign.CreatedAt == 0 {
		campaign.CreatedAt = now
	}
	if campaign.UpdatedAt == 0 {
		campaign.UpdatedAt = now
	}
	return nil
}

func (campaign *EmailCampaign) BeforeUpdate(_ *gorm.DB) error {
	campaign.UpdatedAt = common.GetTimestamp()
	return nil
}

func (delivery *EmailDelivery) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	if delivery.CreatedAt == 0 {
		delivery.CreatedAt = now
	}
	if delivery.UpdatedAt == 0 {
		delivery.UpdatedAt = now
	}
	return nil
}

func (delivery *EmailDelivery) BeforeUpdate(_ *gorm.DB) error {
	delivery.UpdatedAt = common.GetTimestamp()
	return nil
}

func (campaign *EmailCampaign) SetTargetUserIDList(userIDs []int) error {
	data, err := common.Marshal(userIDs)
	if err != nil {
		return err
	}
	campaign.TargetUserIds = string(data)
	return nil
}

func (campaign *EmailCampaign) TargetUserIDList() ([]int, error) {
	if strings.TrimSpace(campaign.TargetUserIds) == "" {
		return []int{}, nil
	}
	var userIDs []int
	if err := common.UnmarshalJsonStr(campaign.TargetUserIds, &userIDs); err != nil {
		return nil, err
	}
	return userIDs, nil
}

func CreateEmailCampaign(campaign *EmailCampaign) error {
	return DB.Create(campaign).Error
}

func UpdateEmailCampaign(campaign *EmailCampaign) error {
	return DB.Save(campaign).Error
}

func GetEmailCampaignById(id int64) (*EmailCampaign, error) {
	var campaign EmailCampaign
	err := DB.Where("id = ?", id).First(&campaign).Error
	return &campaign, err
}

func ListEmailCampaigns(startIdx, pageSize int, search, status string) ([]*EmailCampaign, int64, error) {
	query := DB.Model(&EmailCampaign{})
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR subject LIKE ?", like, like)
	}
	if status = strings.TrimSpace(status); status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var campaigns []*EmailCampaign
	err := query.Order("id DESC").Offset(startIdx).Limit(pageSize).Find(&campaigns).Error
	return campaigns, total, err
}

func GetEmailCampaignStats() (EmailCampaignStats, error) {
	var stats EmailCampaignStats
	err := DB.Model(&EmailCampaign{}).Select(
		"COUNT(*) AS campaign_count, COALESCE(SUM(recipient_count), 0) AS recipient_count, COALESCE(SUM(success_count), 0) AS success_count, COALESCE(SUM(failed_count), 0) AS failed_count",
	).Scan(&stats).Error
	return stats, err
}

func SearchEmailCampaignUserOptions(keyword string, startIdx, pageSize int) ([]EmailCampaignUserOption, int64, error) {
	query := DB.Model(&User{}).
		Select("id, username, display_name, email").
		Where("status = ? AND email <> '' AND deleted_at IS NULL", common.UserStatusEnabled)
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		idTextExpression := "CAST(id AS TEXT)"
		if common.UsingMainDatabase(common.DatabaseTypeMySQL) {
			idTextExpression = "CAST(id AS CHAR)"
		}
		like := "%" + strings.ToLower(keyword) + "%"
		query = query.Where(
			"("+idTextExpression+" LIKE ? OR LOWER(username) LIKE ? OR LOWER(email) LIKE ?)",
			like, like, like,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var users []EmailCampaignUserOption
	err := query.Order("id DESC").Offset(startIdx).Limit(pageSize).Scan(&users).Error
	return users, total, err
}

func GetEmailCampaignUserOptionsByIds(userIDs []int) ([]EmailCampaignUserOption, error) {
	if len(userIDs) == 0 {
		return []EmailCampaignUserOption{}, nil
	}
	var users []EmailCampaignUserOption
	err := DB.Model(&User{}).
		Select("id, username, display_name, email").
		Where("id IN ? AND status = ? AND email <> '' AND deleted_at IS NULL", userIDs, common.UserStatusEnabled).
		Order("id DESC").
		Scan(&users).Error
	return users, err
}

func PauseEmailCampaign(id int64) error {
	result := DB.Model(&EmailCampaign{}).
		Where("id = ? AND status IN ?", id, []string{EmailCampaignStatusScheduled, EmailCampaignStatusActive}).
		Updates(map[string]any{"status": EmailCampaignStatusPaused, "next_run_at": 0, "updated_at": common.GetTimestamp()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("email campaign cannot be paused in its current state")
	}
	return nil
}

func DeleteEmailCampaign(id int64) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var campaign EmailCampaign
		if err := tx.Where("id = ?", id).First(&campaign).Error; err != nil {
			return err
		}
		if campaign.Status == EmailCampaignStatusRunning {
			return errors.New("running email campaign cannot be deleted")
		}
		if err := tx.Where("campaign_id = ?", id).Delete(&EmailDelivery{}).Error; err != nil {
			return err
		}
		return tx.Delete(&campaign).Error
	})
}

func RecoverStaleEmailCampaigns(staleBefore int64) error {
	var campaigns []*EmailCampaign
	if err := DB.Where("status = ? AND updated_at < ?", EmailCampaignStatusRunning, staleBefore).Find(&campaigns).Error; err != nil {
		return err
	}
	for _, campaign := range campaigns {
		status := EmailCampaignStatusScheduled
		if campaign.Mode == EmailCampaignModeConditional {
			status = EmailCampaignStatusActive
		}
		if err := DB.Model(&EmailCampaign{}).
			Where("id = ? AND status = ? AND updated_at < ?", campaign.Id, EmailCampaignStatusRunning, staleBefore).
			Updates(map[string]any{"status": status, "next_run_at": common.GetTimestamp(), "updated_at": common.GetTimestamp()}).Error; err != nil {
			return err
		}
	}
	return nil
}

func HasDueEmailCampaigns(now int64) bool {
	var count int64
	err := DB.Model(&EmailCampaign{}).
		Where("status IN ? AND next_run_at > 0 AND next_run_at <= ?", []string{EmailCampaignStatusScheduled, EmailCampaignStatusActive}, now).
		Limit(1).
		Count(&count).Error
	return err == nil && count > 0
}

func ClaimDueEmailCampaigns(now int64, limit int) ([]*EmailCampaign, error) {
	if limit <= 0 {
		limit = 10
	}
	var due []*EmailCampaign
	if err := DB.Where("status IN ? AND next_run_at > 0 AND next_run_at <= ?", []string{EmailCampaignStatusScheduled, EmailCampaignStatusActive}, now).
		Order("next_run_at ASC, id ASC").
		Limit(limit).
		Find(&due).Error; err != nil {
		return nil, err
	}
	claimed := make([]*EmailCampaign, 0, len(due))
	for _, campaign := range due {
		result := DB.Model(&EmailCampaign{}).
			Where("id = ? AND status = ? AND next_run_at = ?", campaign.Id, campaign.Status, campaign.NextRunAt).
			Updates(map[string]any{"status": EmailCampaignStatusRunning, "updated_at": now})
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 1 {
			campaign.Status = EmailCampaignStatusRunning
			campaign.UpdatedAt = now
			claimed = append(claimed, campaign)
		}
	}
	return claimed, nil
}

func emailRecipientCandidateQuery(campaign *EmailCampaign, now int64) (*gorm.DB, error) {
	selectUser := "users.id AS user_id, users.email, users.username, users.display_name"
	query := DB.Table("users").
		Select(selectUser).
		Where("users.status = ? AND users.email <> '' AND users.deleted_at IS NULL", common.UserStatusEnabled)

	if campaign.Mode == EmailCampaignModeConditional {
		if campaign.TriggerType != EmailCampaignTriggerSubscriptionExpiring {
			return nil, errors.New("unsupported email campaign trigger")
		}
		deadline := now + int64(campaign.TriggerDays*24*60*60)
		return query.
			Select(selectUser+", user_subscriptions.id AS subscription_id, subscription_plans.title AS subscription_title, user_subscriptions.end_time AS subscription_end_time").
			Joins("JOIN user_subscriptions ON user_subscriptions.user_id = users.id").
			Joins("LEFT JOIN subscription_plans ON subscription_plans.id = user_subscriptions.plan_id").
			Where("user_subscriptions.status = ? AND user_subscriptions.end_time > ? AND user_subscriptions.end_time <= ?", "active", now, deadline), nil
	}

	switch campaign.TargetType {
	case EmailCampaignTargetAllUsers:
		return query, nil
	case EmailCampaignTargetActiveSubscribers:
		return query.Where("EXISTS (SELECT 1 FROM user_subscriptions WHERE user_subscriptions.user_id = users.id AND user_subscriptions.status = ? AND user_subscriptions.end_time > ?)", "active", now), nil
	case EmailCampaignTargetSelectedUsers:
		userIDs, err := campaign.TargetUserIDList()
		if err != nil {
			return nil, err
		}
		if len(userIDs) == 0 {
			return nil, errors.New("selected user list is empty")
		}
		return query.Where("users.id IN ?", userIDs), nil
	default:
		return nil, errors.New("unsupported email campaign target")
	}
}

func CountEmailRecipientCandidates(campaign *EmailCampaign, now int64) (int64, error) {
	query, err := emailRecipientCandidateQuery(campaign, now)
	if err != nil {
		return 0, err
	}
	var count int64
	if campaign.Mode == EmailCampaignModeConditional {
		err = query.Count(&count).Error
	} else {
		err = query.Distinct("users.id").Count(&count).Error
	}
	return count, err
}

func ListEmailRecipientCandidates(campaign *EmailCampaign, now int64, offset, limit int) ([]EmailRecipientCandidate, error) {
	query, err := emailRecipientCandidateQuery(campaign, now)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 500
	}
	order := "users.id ASC"
	if campaign.Mode == EmailCampaignModeConditional {
		order += ", user_subscriptions.id ASC"
	}
	var candidates []EmailRecipientCandidate
	err = query.Order(order).Offset(offset).Limit(limit).Scan(&candidates).Error
	return candidates, err
}

func ListSubscriptionExpiryReminderCandidates(now int64, reminderDays, offset, limit int) ([]EmailRecipientCandidate, error) {
	if reminderDays <= 0 {
		return nil, errors.New("reminder days must be positive")
	}
	if limit <= 0 {
		limit = 500
	}
	daySeconds := int64(24 * 60 * 60)
	lowerBound := now + int64(reminderDays-1)*daySeconds
	upperBound := now + int64(reminderDays)*daySeconds
	selectUser := "users.id AS user_id, users.email, users.username, users.display_name"
	var candidates []EmailRecipientCandidate
	err := DB.Table("users").
		Select(selectUser+", user_subscriptions.id AS subscription_id, subscription_plans.title AS subscription_title, user_subscriptions.end_time AS subscription_end_time").
		Joins("JOIN user_subscriptions ON user_subscriptions.user_id = users.id").
		Joins("LEFT JOIN subscription_plans ON subscription_plans.id = user_subscriptions.plan_id").
		Where("users.status = ? AND users.email <> '' AND users.deleted_at IS NULL", common.UserStatusEnabled).
		Where("user_subscriptions.status = ? AND user_subscriptions.end_time > ? AND user_subscriptions.end_time <= ?", "active", lowerBound, upperBound).
		Order("users.id ASC, user_subscriptions.id ASC").
		Offset(offset).
		Limit(limit).
		Scan(&candidates).Error
	return candidates, err
}

func IsSubscriptionExpiryReminderCurrent(subscriptionId, userId int, endTime, now int64) (bool, error) {
	var count int64
	err := DB.Model(&UserSubscription{}).
		Where("id = ? AND user_id = ? AND status = ? AND end_time = ? AND end_time > ?", subscriptionId, userId, "active", endTime, now).
		Count(&count).Error
	return count > 0, err
}

func CreateEmailDeliveries(deliveries []EmailDelivery) error {
	if len(deliveries) == 0 {
		return nil
	}
	return DB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "dedupe_key"}}, DoNothing: true}).Create(&deliveries).Error
}

func RecoverStaleEmailDeliveries(campaignId, staleBefore int64) error {
	return DB.Model(&EmailDelivery{}).
		Where("campaign_id = ? AND status = ? AND updated_at < ?", campaignId, EmailDeliveryStatusSending, staleBefore).
		Updates(map[string]any{"status": EmailDeliveryStatusPending, "updated_at": common.GetTimestamp()}).Error
}

func ResetRetryableEmailDeliveries(campaignId int64) error {
	return DB.Model(&EmailDelivery{}).
		Where("campaign_id = ? AND status = ? AND attempt_count < 3", campaignId, EmailDeliveryStatusFailed).
		Updates(map[string]any{"status": EmailDeliveryStatusPending, "updated_at": common.GetTimestamp()}).Error
}

func ListPendingEmailDeliveries(campaignId int64, limit int) ([]*EmailDelivery, error) {
	if limit <= 0 {
		limit = 100
	}
	var deliveries []*EmailDelivery
	err := DB.Where("campaign_id = ? AND status = ?", campaignId, EmailDeliveryStatusPending).
		Order("id ASC").Limit(limit).Find(&deliveries).Error
	return deliveries, err
}

func ClaimEmailDelivery(id int64) (bool, error) {
	now := common.GetTimestamp()
	result := DB.Model(&EmailDelivery{}).
		Where("id = ? AND status = ?", id, EmailDeliveryStatusPending).
		Updates(map[string]any{"status": EmailDeliveryStatusSending, "attempt_count": gorm.Expr("attempt_count + 1"), "updated_at": now})
	return result.RowsAffected == 1, result.Error
}

func FinishEmailDelivery(id int64, status, errorMessage string) error {
	updates := map[string]any{
		"status":     status,
		"last_error": errorMessage,
		"updated_at": common.GetTimestamp(),
	}
	if status == EmailDeliveryStatusSent {
		updates["sent_at"] = common.GetTimestamp()
	}
	return DB.Model(&EmailDelivery{}).Where("id = ? AND status = ?", id, EmailDeliveryStatusSending).Updates(updates).Error
}

func GetEmailCampaignDeliveryStats(campaignId int64) (EmailCampaignDeliveryStats, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	if err := DB.Model(&EmailDelivery{}).Select("status, COUNT(*) AS count").Where("campaign_id = ?", campaignId).Group("status").Scan(&rows).Error; err != nil {
		return EmailCampaignDeliveryStats{}, err
	}
	stats := EmailCampaignDeliveryStats{}
	for _, item := range rows {
		stats.RecipientCount += item.Count
		switch item.Status {
		case EmailDeliveryStatusSent:
			stats.SuccessCount = item.Count
		case EmailDeliveryStatusFailed:
			stats.FailedCount = item.Count
		case EmailDeliveryStatusSkipped:
			stats.SkippedCount = item.Count
		}
	}
	return stats, nil
}

func FinishEmailCampaignRun(campaign *EmailCampaign, stats EmailCampaignDeliveryStats, now int64) error {
	status := EmailCampaignStatusCompleted
	nextRunAt := int64(0)
	if campaign.Mode == EmailCampaignModeConditional {
		status = EmailCampaignStatusActive
		nextRunAt = now + 24*60*60
	} else if stats.FailedCount > 0 {
		status = EmailCampaignStatusPartialFailed
	}
	return DB.Model(&EmailCampaign{}).Where("id = ? AND status = ?", campaign.Id, EmailCampaignStatusRunning).Updates(map[string]any{
		"status":          status,
		"next_run_at":     nextRunAt,
		"last_run_at":     now,
		"recipient_count": stats.RecipientCount,
		"success_count":   stats.SuccessCount,
		"failed_count":    stats.FailedCount,
		"skipped_count":   stats.SkippedCount,
		"last_error":      "",
		"updated_at":      now,
	}).Error
}

func FailEmailCampaignRun(campaign *EmailCampaign, errorMessage string, now int64) error {
	status := EmailCampaignStatusScheduled
	if campaign.Mode == EmailCampaignModeConditional {
		status = EmailCampaignStatusActive
	}
	return DB.Model(&EmailCampaign{}).Where("id = ? AND status = ?", campaign.Id, EmailCampaignStatusRunning).Updates(map[string]any{
		"status":      status,
		"next_run_at": now + 5*60,
		"last_error":  errorMessage,
		"updated_at":  now,
	}).Error
}

func ListEmailDeliveries(campaignId int64, startIdx, pageSize int, status string) ([]*EmailDelivery, int64, error) {
	query := DB.Model(&EmailDelivery{}).Where("campaign_id = ?", campaignId)
	if status = strings.TrimSpace(status); status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var deliveries []*EmailDelivery
	err := query.Order("id DESC").Offset(startIdx).Limit(pageSize).Find(&deliveries).Error
	return deliveries, total, err
}

func RetryFailedEmailCampaign(id int64, now int64) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var campaign EmailCampaign
		if err := tx.Where("id = ?", id).First(&campaign).Error; err != nil {
			return err
		}
		if campaign.Mode == EmailCampaignModeConditional {
			return errors.New("conditional campaign retries automatically")
		}
		if campaign.Status != EmailCampaignStatusPartialFailed && campaign.Status != EmailCampaignStatusCompleted {
			return errors.New("email campaign cannot be retried in its current state")
		}
		if err := tx.Model(&EmailDelivery{}).Where("campaign_id = ? AND status = ?", id, EmailDeliveryStatusFailed).
			Updates(map[string]any{"status": EmailDeliveryStatusPending, "updated_at": now}).Error; err != nil {
			return err
		}
		result := tx.Model(&EmailCampaign{}).Where("id = ? AND status = ?", id, campaign.Status).
			Updates(map[string]any{"status": EmailCampaignStatusScheduled, "next_run_at": now, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("email campaign %d state changed", id)
		}
		return nil
	})
}
