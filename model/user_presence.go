package model

import (
	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm/clause"
)

type UserPresence struct {
	UserId         int    `json:"user_id" gorm:"primaryKey;column:user_id"`
	IsOnline       bool   `json:"is_online" gorm:"type:bool;not null;column:is_online;index"`
	LastActivityAt int64  `json:"last_activity_at" gorm:"type:bigint;not null;column:last_activity_at;index"`
	LastSource     string `json:"last_source" gorm:"type:varchar(32);not null;column:last_source"`
	LastIP         string `json:"last_ip" gorm:"type:varchar(45);not null;column:last_ip"`
	LastUserAgent  string `json:"last_user_agent" gorm:"type:varchar(512);not null;column:last_user_agent"`
	LastChangedAt  int64  `json:"last_changed_at" gorm:"type:bigint;not null;column:last_changed_at"`
	UpdatedAt      int64  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

func RecordUserPresenceActivity(userId int, activityAt int64, source, ip, userAgent string) (UserPresence, bool, error) {
	presence := UserPresence{
		UserId:         userId,
		IsOnline:       true,
		LastActivityAt: activityAt,
		LastSource:     source,
		LastIP:         ip,
		LastUserAgent:  userAgent,
		LastChangedAt:  activityAt,
		UpdatedAt:      activityAt,
	}
	created := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&presence)
	if created.Error != nil {
		return UserPresence{}, false, created.Error
	}
	if created.RowsAffected == 1 {
		return presence, true, nil
	}

	updates := map[string]any{
		"is_online":        true,
		"last_activity_at": activityAt,
		"last_source":      source,
		"last_ip":          ip,
		"last_user_agent":  userAgent,
		"last_changed_at":  activityAt,
		"updated_at":       activityAt,
	}
	transitioned := DB.Model(&UserPresence{}).
		Where("user_id = ? AND is_online = ?", userId, false).
		Updates(updates)
	if transitioned.Error != nil {
		return UserPresence{}, false, transitioned.Error
	}
	if transitioned.RowsAffected == 1 {
		return presence, true, nil
	}

	updated := DB.Model(&UserPresence{}).
		Where("user_id = ? AND is_online = ? AND last_activity_at < ?", userId, true, activityAt).
		Updates(map[string]any{
			"last_activity_at": activityAt,
			"last_source":      source,
			"last_ip":          ip,
			"last_user_agent":  userAgent,
			"updated_at":       activityAt,
		})
	if updated.Error != nil {
		return UserPresence{}, false, updated.Error
	}
	if err := DB.Where("user_id = ?", userId).First(&presence).Error; err != nil {
		return UserPresence{}, false, err
	}
	return presence, false, nil
}

func MarkTimedOutUserPresencesOffline(userIds []int, cutoff, changedAt int64) ([]UserPresence, error) {
	if len(userIds) == 0 {
		return []UserPresence{}, nil
	}
	var candidates []UserPresence
	if err := DB.Where("user_id IN ? AND is_online = ? AND last_activity_at <= ?", userIds, true, cutoff).
		Order("user_id ASC").
		Find(&candidates).Error; err != nil {
		return nil, err
	}

	result := make([]UserPresence, 0, len(candidates))
	for _, presence := range candidates {
		updated := DB.Model(&UserPresence{}).
			Where("user_id = ? AND is_online = ? AND last_activity_at <= ?", presence.UserId, true, cutoff).
			Updates(map[string]any{
				"is_online":       false,
				"last_changed_at": changedAt,
				"updated_at":      changedAt,
			})
		if updated.Error != nil {
			return result, updated.Error
		}
		if updated.RowsAffected == 1 {
			presence.IsOnline = false
			presence.LastChangedAt = changedAt
			presence.UpdatedAt = changedAt
			result = append(result, presence)
		}
	}
	return result, nil
}

func ResetUserPresencesOffline(userIds []int, changedAt int64) error {
	if len(userIds) == 0 {
		return nil
	}
	return DB.Model(&UserPresence{}).
		Where("user_id IN ? AND is_online = ?", userIds, true).
		Updates(map[string]any{
			"is_online":       false,
			"last_changed_at": changedAt,
			"updated_at":      changedAt,
		}).Error
}

func GetUserPresence(userId int) (*UserPresence, error) {
	var presence UserPresence
	if err := DB.Where("user_id = ?", userId).First(&presence).Error; err != nil {
		return nil, err
	}
	return &presence, nil
}

func ListUserPresenceUsers(userIds []int) ([]User, error) {
	if len(userIds) == 0 {
		return []User{}, nil
	}
	var users []User
	err := DB.Select("id", "username", "display_name", "email", "status", "setting").
		Where("id IN ? AND status = ? AND deleted_at IS NULL", userIds, common.UserStatusEnabled).
		Order("id ASC").
		Find(&users).Error
	return users, err
}
