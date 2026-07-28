package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrUserDeviceBlocked = errors.New("user device is blocked")

type BlockedDevice struct {
	Id         int    `json:"id"`
	UserId     int    `json:"user_id" gorm:"uniqueIndex:idx_blocked_devices_user_device,priority:1;not null"`
	DeviceId   string `json:"device_id" gorm:"type:varchar(64);uniqueIndex:idx_blocked_devices_user_device,priority:2;not null"`
	OperatorId int    `json:"operator_id" gorm:"index;default:0"`
	Reason     string `json:"reason" gorm:"type:varchar(255);default:''"`
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt  int64  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

type UserLoginDeviceStat struct {
	DeviceId           string   `json:"device_id"`
	UserAgent          string   `json:"user_agent"`
	LastLoginAt        int64    `json:"last_login_at"`
	LoginCount         int64    `json:"login_count"`
	ActiveSessionCount int64    `json:"active_session_count"`
	IPs                []string `json:"ips" gorm:"-:all"`
	Blocked            bool     `json:"blocked" gorm:"-:all"`
}

func NormalizeDeviceID(value string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", errors.New("invalid device id")
	}
	return parsed.String(), nil
}

func NormalizeDeviceIDs(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		deviceId, err := NormalizeDeviceID(value)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[deviceId]; exists {
			continue
		}
		seen[deviceId] = struct{}{}
		result = append(result, deviceId)
	}
	return result, nil
}

func IsUserDeviceBlocked(userId int, deviceId string) (bool, error) {
	if userId <= 0 || strings.TrimSpace(deviceId) == "" {
		return false, nil
	}
	normalized, err := NormalizeDeviceID(deviceId)
	if err != nil {
		return false, err
	}
	var count int64
	err = DB.Model(&BlockedDevice{}).
		Where("user_id = ? AND device_id = ?", userId, normalized).
		Count(&count).Error
	return count > 0, err
}

func BlockUserDevices(userId int, deviceIds []string, operatorId int, reason string) error {
	if userId <= 0 {
		return errors.New("invalid user id")
	}
	normalized, err := NormalizeDeviceIDs(deviceIds)
	if err != nil {
		return err
	}
	if len(normalized) == 0 {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		for _, deviceId := range normalized {
			record := BlockedDevice{
				UserId:     userId,
				DeviceId:   deviceId,
				OperatorId: operatorId,
				Reason:     strings.TrimSpace(reason),
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}, {Name: "device_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"operator_id", "reason", "updated_at"}),
			}).Create(&record).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func UnblockUserDevices(userId int, deviceIds []string) error {
	if userId <= 0 {
		return errors.New("invalid user id")
	}
	normalized, err := NormalizeDeviceIDs(deviceIds)
	if err != nil {
		return err
	}
	if len(normalized) == 0 {
		return nil
	}
	return DB.Where("user_id = ? AND device_id IN ?", userId, normalized).Delete(&BlockedDevice{}).Error
}

func GetUserLoginDeviceStats(userId int) ([]UserLoginDeviceStat, error) {
	if userId <= 0 {
		return nil, errors.New("invalid user id")
	}
	now := time.Now().Unix()
	stats := make([]UserLoginDeviceStat, 0)
	err := DB.Model(&UserSession{}).
		Select(
			"device_id, MAX(last_active_at) AS last_login_at, COUNT(*) AS login_count, "+
				"SUM(CASE WHEN status = ? AND expires_at > ? THEN 1 ELSE 0 END) AS active_session_count",
			UserSessionStatusActive,
			now,
		).
		Where("user_id = ? AND device_id <> ''", userId).
		Group("device_id").
		Order("last_login_at DESC").
		Limit(100).
		Scan(&stats).Error
	if err != nil {
		return stats, err
	}

	deviceIds := make([]string, 0, len(stats))
	statIndexes := make(map[string]int, len(stats))
	for index := range stats {
		deviceIds = append(deviceIds, stats[index].DeviceId)
		statIndexes[stats[index].DeviceId] = index
		stats[index].IPs = make([]string, 0)
	}
	if len(deviceIds) > 0 {
		var sessions []UserSession
		if err := DB.Select("device_id", "ip", "user_agent", "last_active_at").
			Where("user_id = ? AND device_id IN ?", userId, deviceIds).
			Order("last_active_at DESC").
			Limit(5000).
			Find(&sessions).Error; err != nil {
			return nil, err
		}
		seenIPs := make(map[string]map[string]struct{}, len(stats))
		for _, session := range sessions {
			index, exists := statIndexes[session.DeviceID]
			if !exists {
				continue
			}
			if stats[index].UserAgent == "" {
				stats[index].UserAgent = session.UserAgent
			}
			if session.IP == "" {
				continue
			}
			if seenIPs[session.DeviceID] == nil {
				seenIPs[session.DeviceID] = make(map[string]struct{})
			}
			if _, exists := seenIPs[session.DeviceID][session.IP]; exists {
				continue
			}
			seenIPs[session.DeviceID][session.IP] = struct{}{}
			stats[index].IPs = append(stats[index].IPs, session.IP)
		}
	}
	var blockedRecords []BlockedDevice
	if err := DB.Select("device_id").
		Where("user_id = ?", userId).
		Order("updated_at DESC").
		Limit(100).
		Find(&blockedRecords).Error; err != nil {
		return nil, err
	}
	for _, record := range blockedRecords {
		if index, exists := statIndexes[record.DeviceId]; exists {
			stats[index].Blocked = true
			continue
		}
		statIndexes[record.DeviceId] = len(stats)
		stats = append(stats, UserLoginDeviceStat{
			DeviceId: record.DeviceId,
			IPs:      make([]string, 0),
			Blocked:  true,
		})
	}
	return stats, nil
}
