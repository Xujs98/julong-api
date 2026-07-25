package model

import (
	"errors"

	"gorm.io/gorm"
)

const MaxUserRequestContentLogs = 50

const (
	UserRequestContentStatusPending = "pending"
	UserRequestContentStatusSuccess = "success"
	UserRequestContentStatusError   = "error"
)

type UserRequestContentLog struct {
	Id             int    `json:"id"`
	UserId         int    `json:"user_id" gorm:"index:idx_user_request_content_user_created,priority:1"`
	RequestId      string `json:"request_id" gorm:"type:varchar(64);uniqueIndex"`
	CreatedAt      int64  `json:"created_at" gorm:"autoCreateTime;index:idx_user_request_content_user_created,priority:2"`
	ModelName      string `json:"model_name" gorm:"type:varchar(191)"`
	TokenName      string `json:"token_name" gorm:"type:varchar(191)"`
	RequestPath    string `json:"request_path" gorm:"type:varchar(255)"`
	Status         string `json:"status" gorm:"type:varchar(16);index"`
	ErrorMessage   string `json:"error_message,omitempty" gorm:"type:text"`
	OriginalSize   int    `json:"original_size"`
	CapturedSize   int    `json:"captured_size"`
	Truncated      bool   `json:"truncated"`
	CompressedJSON []byte `json:"-"`
}

func CreateUserRequestContentLog(log *UserRequestContentLog) error {
	if log == nil || log.UserId <= 0 || log.RequestId == "" {
		return errors.New("invalid user request content log")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(log).Error; err != nil {
			return err
		}

		var staleIds []int
		if err := tx.Model(&UserRequestContentLog{}).
			Where("user_id = ?", log.UserId).
			Order("created_at DESC, id DESC").
			Offset(MaxUserRequestContentLogs).
			Limit(1000).
			Pluck("id", &staleIds).Error; err != nil {
			return err
		}
		if len(staleIds) == 0 {
			return nil
		}
		return tx.Where("id IN ?", staleIds).Delete(&UserRequestContentLog{}).Error
	})
}

func FinishUserRequestContentLog(requestId, status, errorMessage string) error {
	if requestId == "" {
		return nil
	}
	return DB.Model(&UserRequestContentLog{}).
		Where("request_id = ?", requestId).
		Updates(map[string]any{
			"status":        status,
			"error_message": errorMessage,
		}).Error
}

func GetUserRequestContentLogs(userId int) ([]*UserRequestContentLog, error) {
	logs := make([]*UserRequestContentLog, 0)
	err := DB.Model(&UserRequestContentLog{}).
		Omit("compressed_json").
		Where("user_id = ?", userId).
		Order("created_at DESC, id DESC").
		Limit(MaxUserRequestContentLogs).
		Find(&logs).Error
	return logs, err
}

func GetUserRequestContentLog(userId, logId int) (*UserRequestContentLog, error) {
	var log UserRequestContentLog
	err := DB.Where("id = ? AND user_id = ?", logId, userId).First(&log).Error
	return &log, err
}

func DeleteUserRequestContentLogs(userId int) error {
	return DB.Where("user_id = ?", userId).Delete(&UserRequestContentLog{}).Error
}

func SetUserRequestContentLogging(userId int, enabled bool) error {
	result := DB.Model(&User{}).Where("id = ?", userId).
		Update("request_content_logging_enabled", enabled)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return PublishUserAuthCache(userId)
}
