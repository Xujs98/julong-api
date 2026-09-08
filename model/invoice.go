package model

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	InvoiceStatusPending     = "pending"
	InvoiceStatusProcessing  = "processing"
	InvoiceStatusCompleted   = "completed"
	InvoiceStatusRejected    = "rejected"
	InvoiceStatusNonReusable = "non_reusable"
	InvoiceStatusCancelled   = "cancelled"

	maxInvoiceSourceRecords = 10000
	maxInvoiceItems         = 100
	invoiceEmailLockSeconds = int64(15 * 60)
)

var (
	ErrInvoiceSourcesInvalid  = errors.New("invoice source records are invalid")
	ErrInvoiceSourcesClaimed  = errors.New("invoice source records have already been claimed")
	ErrInvoiceStatusTerminal  = errors.New("invoice application status is terminal")
	ErrInvoiceEmailSending    = errors.New("invoice email is already being sent")
	ErrInvoiceNotCancellable  = errors.New("invoice application is not cancellable")
	ErrInvoiceNotWithdrawable = errors.New("invoice application completion is not withdrawable")
)

type InvoiceApplication struct {
	Id             int            `json:"id"`
	UserId         int            `json:"user_id" gorm:"index"`
	Username       string         `json:"username" gorm:"-:all"`
	Email          string         `json:"email" gorm:"type:varchar(255)"`
	AmountQuota    int64          `json:"amount_quota"`
	Amount         float64        `json:"amount" gorm:"-:all"`
	AmountValue    string         `json:"-" gorm:"type:varchar(64)"`
	Unit           string         `json:"unit" gorm:"type:varchar(32)"`
	Status         string         `json:"status" gorm:"type:varchar(32);index"`
	AdminNote      string         `json:"admin_note" gorm:"type:varchar(500)"`
	CreatedAt      int64          `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      int64          `json:"updated_at" gorm:"autoUpdateTime"`
	CompletedAt    int64          `json:"completed_at"`
	EmailSendingAt int64          `json:"-" gorm:"index"`
	HasAttachment  bool           `json:"has_attachment" gorm:"-:all"`
	Items          []*InvoiceItem `json:"items" gorm:"foreignKey:ApplicationId"`
}

func (application *InvoiceApplication) AfterFind(*gorm.DB) error {
	amount, err := strconv.ParseFloat(application.AmountValue, 64)
	if err != nil {
		amount = invoiceAmountFromQuota(application.AmountQuota)
	}
	application.Amount = amount
	return nil
}

type InvoiceItem struct {
	Id              int     `json:"id"`
	ApplicationId   int     `json:"application_id" gorm:"index"`
	UserId          int     `json:"user_id" gorm:"index"`
	SourceRequestId string  `json:"source_request_id" gorm:"type:varchar(64);uniqueIndex:idx_invoice_item_source_active,priority:1"`
	Source          string  `json:"source" gorm:"type:varchar(32)"`
	Quota           int     `json:"quota"`
	SourceCreatedAt int64   `json:"source_created_at"`
	ReleasedAt      int64   `json:"released_at" gorm:"uniqueIndex:idx_invoice_item_source_active,priority:2"`
	FrozenUntil     int64   `json:"frozen_until" gorm:"index"`
	Amount          float64 `json:"amount" gorm:"-:all"`
	Content         string  `json:"content" gorm:"-:all"`
}

type InvoiceAttachment struct {
	Id            int    `json:"id"`
	ApplicationId int    `json:"application_id" gorm:"uniqueIndex"`
	Filename      string `json:"filename" gorm:"type:varchar(255)"`
	ContentType   string `json:"content_type" gorm:"type:varchar(128)"`
	Data          []byte `json:"-"`
	Size          int64  `json:"size"`
	CreatedAt     int64  `json:"created_at" gorm:"autoCreateTime"`
}

type InvoiceCredit struct {
	RequestId   string  `json:"request_id"`
	CreatedAt   int64   `json:"created_at"`
	Quota       int     `json:"quota"`
	Amount      float64 `json:"amount"`
	Source      string  `json:"source"`
	Content     string  `json:"content"`
	FrozenUntil int64   `json:"frozen_until"`
}

func isInvoiceSource(source string) bool {
	return source == QuotaIncreaseSourceOnlineRecharge ||
		source == QuotaIncreaseSourceRedemption ||
		source == QuotaIncreaseSourceAdminAdjustment
}

func invoiceAmountFromQuota(quota int64) float64 {
	if common.QuotaPerUnit <= 0 {
		return 0
	}
	return float64(quota) / common.QuotaPerUnit
}

func loadInvoiceCredits(userId int) ([]InvoiceCredit, error) {
	var logs []Log
	query := LOG_DB.Model(&Log{}).
		Where("user_id = ? AND type = ? AND quota > 0", userId, LogTypeQuotaIncrease)
	order := "id desc"
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		order = clickHouseLogOrder("")
	}
	if err := query.Order(order).Limit(maxInvoiceSourceRecords).Find(&logs).Error; err != nil {
		return nil, err
	}

	credits := make([]InvoiceCredit, 0, len(logs))
	for i := range logs {
		metadata := struct {
			Source string `json:"source"`
		}{}
		if logs[i].RequestId == "" || common.UnmarshalJsonStr(logs[i].Other, &metadata) != nil || !isInvoiceSource(metadata.Source) {
			continue
		}
		credits = append(credits, InvoiceCredit{
			RequestId: logs[i].RequestId,
			CreatedAt: logs[i].CreatedAt,
			Quota:     logs[i].Quota,
			Amount:    invoiceAmountFromQuota(int64(logs[i].Quota)),
			Source:    metadata.Source,
			Content:   logs[i].Content,
		})
	}
	return credits, nil
}

func GetInvoiceCredits(userId, startIdx, num int) ([]InvoiceCredit, int, error) {
	credits, err := loadInvoiceCredits(userId)
	if err != nil {
		return nil, 0, err
	}
	var items []InvoiceItem
	if err := DB.Where("user_id = ?", userId).Order("released_at desc").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	claimedSet := make(map[string]struct{}, len(items))
	frozenUntil := make(map[string]int64, len(items))
	for i := range items {
		if items[i].ReleasedAt == 0 {
			claimedSet[items[i].SourceRequestId] = struct{}{}
			continue
		}
		if items[i].FrozenUntil > frozenUntil[items[i].SourceRequestId] {
			frozenUntil[items[i].SourceRequestId] = items[i].FrozenUntil
		}
	}
	eligible := make([]InvoiceCredit, 0, len(credits))
	for i := range credits {
		if _, exists := claimedSet[credits[i].RequestId]; exists {
			continue
		}
		credits[i].FrozenUntil = frozenUntil[credits[i].RequestId]
		eligible = append(eligible, credits[i])
	}
	total := len(eligible)
	if startIdx >= total {
		return []InvoiceCredit{}, total, nil
	}
	end := startIdx + num
	if end > total {
		end = total
	}
	return eligible[startIdx:end], total, nil
}

func CreateInvoiceApplication(userId int, email, unit string, minimumQuota int64, requestIds []string) (*InvoiceApplication, error) {
	if userId <= 0 || len(requestIds) == 0 || len(requestIds) > maxInvoiceItems {
		return nil, ErrInvoiceSourcesInvalid
	}
	requestSet := make(map[string]struct{}, len(requestIds))
	for _, requestId := range requestIds {
		requestId = strings.TrimSpace(requestId)
		if requestId == "" {
			return nil, ErrInvoiceSourcesInvalid
		}
		requestSet[requestId] = struct{}{}
	}
	if len(requestSet) != len(requestIds) {
		return nil, ErrInvoiceSourcesInvalid
	}

	credits, err := loadInvoiceCredits(userId)
	if err != nil {
		return nil, err
	}
	selected := make([]InvoiceCredit, 0, len(requestIds))
	var amountQuota int64
	for i := range credits {
		if _, exists := requestSet[credits[i].RequestId]; !exists {
			continue
		}
		selected = append(selected, credits[i])
		amountQuota += int64(credits[i].Quota)
	}
	if len(selected) != len(requestIds) || amountQuota < minimumQuota {
		return nil, ErrInvoiceSourcesInvalid
	}

	amount := invoiceAmountFromQuota(amountQuota)
	application := &InvoiceApplication{
		UserId:      userId,
		Email:       email,
		AmountQuota: amountQuota,
		Amount:      amount,
		AmountValue: strconv.FormatFloat(amount, 'g', -1, 64),
		Unit:        unit,
		Status:      InvoiceStatusPending,
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		ids := make([]string, 0, len(requestSet))
		for requestId := range requestSet {
			ids = append(ids, requestId)
		}
		var claimed int64
		if err := tx.Model(&InvoiceItem{}).Where("source_request_id IN ? AND released_at = 0", ids).Count(&claimed).Error; err != nil {
			return err
		}
		if claimed > 0 {
			return ErrInvoiceSourcesClaimed
		}
		var frozen int64
		if err := tx.Model(&InvoiceItem{}).Where("source_request_id IN ? AND frozen_until > ?", ids, common.GetTimestamp()).Count(&frozen).Error; err != nil {
			return err
		}
		if frozen > 0 {
			return ErrInvoiceSourcesClaimed
		}
		if err := tx.Create(application).Error; err != nil {
			return err
		}
		items := make([]InvoiceItem, 0, len(selected))
		for i := range selected {
			items = append(items, InvoiceItem{
				ApplicationId:   application.Id,
				UserId:          userId,
				SourceRequestId: selected[i].RequestId,
				Source:          selected[i].Source,
				Quota:           selected[i].Quota,
				SourceCreatedAt: selected[i].CreatedAt,
			})
		}
		if err := tx.Create(&items).Error; err != nil {
			return ErrInvoiceSourcesClaimed
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return application, nil
}

func GetInvoiceApplications(userId, startIdx, num int) ([]*InvoiceApplication, int64, error) {
	query := DB.Model(&InvoiceApplication{})
	if userId > 0 {
		query = query.Where("user_id = ?", userId)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var applications []*InvoiceApplication
	if err := query.Preload("Items").Order("id desc").Offset(startIdx).Limit(num).Find(&applications).Error; err != nil {
		return nil, 0, err
	}
	applicationIds := make([]int, 0, len(applications))
	requestIds := make([]string, 0)
	for _, application := range applications {
		applicationIds = append(applicationIds, application.Id)
		for _, item := range application.Items {
			item.Amount = invoiceAmountFromQuota(int64(item.Quota))
			if userId == 0 {
				requestIds = append(requestIds, item.SourceRequestId)
			}
		}
	}
	if len(applicationIds) > 0 {
		var attachmentApplicationIds []int
		if err := DB.Model(&InvoiceAttachment{}).Where("application_id IN ?", applicationIds).Pluck("application_id", &attachmentApplicationIds).Error; err != nil {
			return nil, 0, err
		}
		hasAttachment := make(map[int]struct{}, len(attachmentApplicationIds))
		for _, applicationId := range attachmentApplicationIds {
			hasAttachment[applicationId] = struct{}{}
		}
		for _, application := range applications {
			_, application.HasAttachment = hasAttachment[application.Id]
		}
	}
	if len(requestIds) > 0 {
		var logs []Log
		if err := LOG_DB.Select("user_id", "request_id", "content").Where("request_id IN ?", requestIds).Find(&logs).Error; err != nil {
			return nil, 0, err
		}
		contents := make(map[string]string, len(logs))
		for i := range logs {
			key := strconv.Itoa(logs[i].UserId) + "\x00" + logs[i].RequestId
			contents[key] = logs[i].Content
		}
		for _, application := range applications {
			for _, item := range application.Items {
				key := strconv.Itoa(item.UserId) + "\x00" + item.SourceRequestId
				item.Content = contents[key]
			}
		}
	}
	if userId > 0 {
		for _, application := range applications {
			application.AdminNote = ""
		}
	} else {
		userIds := make([]int, 0, len(applications))
		seen := make(map[int]struct{}, len(applications))
		for _, application := range applications {
			if _, exists := seen[application.UserId]; !exists {
				seen[application.UserId] = struct{}{}
				userIds = append(userIds, application.UserId)
			}
		}
		var users []User
		if len(userIds) > 0 {
			if err := DB.Unscoped().Select("id", "username").Where("id IN ?", userIds).Find(&users).Error; err != nil {
				return nil, 0, err
			}
		}
		usernames := make(map[int]string, len(users))
		for i := range users {
			usernames[users[i].Id] = users[i].Username
		}
		for _, application := range applications {
			application.Username = usernames[application.UserId]
		}
	}
	return applications, total, nil
}

func GetInvoiceApplicationById(id int) (*InvoiceApplication, error) {
	application := &InvoiceApplication{}
	if err := DB.Preload("Items").First(application, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return application, nil
}

func CancelInvoiceApplication(id, userId int) (*InvoiceApplication, error) {
	application := &InvoiceApplication{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).First(application, "id = ? AND user_id = ?", id, userId).Error; err != nil {
			return err
		}
		if application.Status != InvoiceStatusPending || application.EmailSendingAt > common.GetTimestamp()-invoiceEmailLockSeconds {
			return ErrInvoiceNotCancellable
		}
		if err := tx.Model(application).Updates(map[string]any{"status": InvoiceStatusCancelled, "completed_at": 0}).Error; err != nil {
			return err
		}
		return tx.Model(&InvoiceItem{}).
			Where("application_id = ? AND released_at = 0", application.Id).
			Updates(map[string]any{"released_at": time.Now().UnixNano(), "frozen_until": 0}).Error
	})
	if err != nil {
		return nil, err
	}
	if err := DB.Preload("Items").First(application, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return application, nil
}

func GetInvoiceAttachment(id, userId int, allowAdmin bool) (*InvoiceApplication, *InvoiceAttachment, error) {
	application := &InvoiceApplication{}
	query := DB.Where("id = ?", id)
	if !allowAdmin {
		query = query.Where("user_id = ?", userId)
	}
	if err := query.First(application).Error; err != nil {
		return nil, nil, err
	}
	if application.Status != InvoiceStatusCompleted {
		return nil, nil, ErrInvoiceStatusTerminal
	}
	attachment := &InvoiceAttachment{}
	if err := DB.First(attachment, "application_id = ?", application.Id).Error; err != nil {
		return nil, nil, err
	}
	return application, attachment, nil
}

func UpdateInvoiceApplication(id int, status, adminNote string, rejectionFreezeHours int) (*InvoiceApplication, error) {
	switch status {
	case InvoiceStatusPending, InvoiceStatusProcessing, InvoiceStatusRejected, InvoiceStatusNonReusable:
	default:
		return nil, errors.New("invalid invoice status")
	}
	application := &InvoiceApplication{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).First(application, "id = ?", id).Error; err != nil {
			return err
		}
		previousStatus := application.Status
		if application.Status == InvoiceStatusRejected || application.Status == InvoiceStatusNonReusable || application.Status == InvoiceStatusCompleted || application.Status == InvoiceStatusCancelled {
			return ErrInvoiceStatusTerminal
		}
		if application.EmailSendingAt > common.GetTimestamp()-invoiceEmailLockSeconds {
			return ErrInvoiceEmailSending
		}
		updates := map[string]any{"status": status, "admin_note": adminNote}
		if status == InvoiceStatusCompleted {
			updates["completed_at"] = common.GetTimestamp()
		} else {
			updates["completed_at"] = 0
		}
		if err := tx.Model(application).Updates(updates).Error; err != nil {
			return err
		}
		if previousStatus != InvoiceStatusRejected && status == InvoiceStatusRejected {
			frozenUntil := common.GetTimestamp() + int64(rejectionFreezeHours)*60*60
			releasedAt := time.Now().UnixNano()
			return tx.Model(&InvoiceItem{}).
				Where("application_id = ? AND released_at = 0", application.Id).
				Updates(map[string]any{"released_at": releasedAt, "frozen_until": frozenUntil}).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := DB.Preload("Items").First(application, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return application, nil
}

func BeginInvoiceEmailDelivery(id int) (*InvoiceApplication, error) {
	now := common.GetTimestamp()
	result := DB.Model(&InvoiceApplication{}).
		Where("id = ? AND status NOT IN ? AND (email_sending_at = 0 OR email_sending_at <= ?)", id, []string{InvoiceStatusRejected, InvoiceStatusNonReusable, InvoiceStatusCompleted, InvoiceStatusCancelled}, now-invoiceEmailLockSeconds).
		Update("email_sending_at", now)
	if result.Error != nil {
		return nil, result.Error
	}
	application := &InvoiceApplication{}
	if err := DB.Preload("Items").First(application, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if result.RowsAffected == 0 {
		if application.Status == InvoiceStatusRejected || application.Status == InvoiceStatusNonReusable || application.Status == InvoiceStatusCompleted || application.Status == InvoiceStatusCancelled {
			return nil, ErrInvoiceStatusTerminal
		}
		return nil, ErrInvoiceEmailSending
	}
	return application, nil
}

func CancelInvoiceEmailDelivery(id int, emailSendingAt int64) error {
	return DB.Model(&InvoiceApplication{}).
		Where("id = ? AND email_sending_at = ? AND status NOT IN ?", id, emailSendingAt, []string{InvoiceStatusRejected, InvoiceStatusNonReusable, InvoiceStatusCompleted, InvoiceStatusCancelled}).
		Update("email_sending_at", 0).Error
}

func CompleteInvoiceEmailDelivery(id int, emailSendingAt int64, adminNote string, attachment InvoiceAttachment) (*InvoiceApplication, error) {
	application := &InvoiceApplication{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).First(application, "id = ?", id).Error; err != nil {
			return err
		}
		if application.Status == InvoiceStatusRejected || application.Status == InvoiceStatusNonReusable || application.Status == InvoiceStatusCompleted || application.Status == InvoiceStatusCancelled {
			return ErrInvoiceStatusTerminal
		}
		if emailSendingAt == 0 || application.EmailSendingAt != emailSendingAt {
			return ErrInvoiceEmailSending
		}
		attachment.ApplicationId = application.Id
		attachment.Size = int64(len(attachment.Data))
		if err := tx.Create(&attachment).Error; err != nil {
			return err
		}
		return tx.Model(application).Updates(map[string]any{
			"status":           InvoiceStatusCompleted,
			"admin_note":       adminNote,
			"completed_at":     common.GetTimestamp(),
			"email_sending_at": 0,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	if err := DB.Preload("Items").First(application, "id = ?", id).Error; err != nil {
		return nil, err
	}
	application.HasAttachment = true
	return application, nil
}

func WithdrawCompletedInvoiceApplication(id int) (*InvoiceApplication, error) {
	application := &InvoiceApplication{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).First(application, "id = ?", id).Error; err != nil {
			return err
		}
		if application.Status != InvoiceStatusCompleted {
			return ErrInvoiceNotWithdrawable
		}
		if err := tx.Where("application_id = ?", application.Id).Delete(&InvoiceAttachment{}).Error; err != nil {
			return err
		}
		return tx.Model(application).Updates(map[string]any{
			"status":           InvoiceStatusPending,
			"completed_at":     0,
			"email_sending_at": 0,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	if err := DB.Preload("Items").First(application, "id = ?", id).Error; err != nil {
		return nil, err
	}
	application.HasAttachment = false
	return application, nil
}
