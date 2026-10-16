package model

import (
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/cachex"

	"github.com/samber/hot"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const blockedIPCacheNamespace = "julong-api:blocked-ip:v1"

type BlockedIP struct {
	Id         int    `json:"id"`
	IP         string `json:"ip" gorm:"type:varchar(45);uniqueIndex;not null"`
	UserId     int    `json:"user_id" gorm:"index;default:0"`
	OperatorId int    `json:"operator_id" gorm:"index;default:0"`
	Reason     string `json:"reason" gorm:"type:varchar(255);default:''"`
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt  int64  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

var (
	blockedIPCacheOnce sync.Once
	blockedIPCache     *cachex.HybridCache[int]
)

func getBlockedIPCache() *cachex.HybridCache[int] {
	blockedIPCacheOnce.Do(func() {
		const ttl = time.Minute
		blockedIPCache = cachex.NewHybridCache[int](cachex.HybridCacheConfig[int]{
			Namespace: cachex.Namespace(blockedIPCacheNamespace),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.IntCodec{},
			Memory: func() *hot.HotCache[string, int] {
				return hot.NewHotCache[string, int](hot.LRU, 10000).
					WithTTL(ttl).
					WithJanitor().
					Build()
			},
		})
	})
	return blockedIPCache
}

func NormalizeIPAddress(value string) (string, error) {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return "", errors.New("invalid IP address")
	}
	return ip.String(), nil
}

func IsIPBlocked(value string) (bool, error) {
	ip, err := NormalizeIPAddress(value)
	if err != nil {
		return false, err
	}
	if DB == nil {
		return false, nil
	}
	if cached, found, cacheErr := getBlockedIPCache().Get(ip); cacheErr == nil && found {
		return cached == 1, nil
	}

	var count int64
	if err := DB.Model(&BlockedIP{}).Where("ip = ?", ip).Count(&count).Error; err != nil {
		return false, err
	}
	blocked := count > 0
	cacheValue := 0
	if blocked {
		cacheValue = 1
	}
	_ = getBlockedIPCache().SetWithTTL(ip, cacheValue, time.Minute)
	return blocked, nil
}

func NormalizeIPAddresses(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		ip, err := NormalizeIPAddress(value)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		result = append(result, ip)
	}
	return result, nil
}

func BlockIPAddressesWithTx(tx *gorm.DB, ips []string, userId int, operatorId int, reason string) error {
	for _, ip := range ips {
		record := BlockedIP{
			IP:         ip,
			UserId:     userId,
			OperatorId: operatorId,
			Reason:     strings.TrimSpace(reason),
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "ip"}},
			DoUpdates: clause.AssignmentColumns([]string{"user_id", "operator_id", "reason", "updated_at"}),
		}).Create(&record).Error; err != nil {
			return err
		}
	}
	return nil
}

func BlockIPAddresses(ips []string, userId int, operatorId int, reason string) error {
	normalized, err := NormalizeIPAddresses(ips)
	if err != nil {
		return err
	}
	if len(normalized) == 0 {
		return nil
	}
	if err := DB.Transaction(func(tx *gorm.DB) error {
		return BlockIPAddressesWithTx(tx, normalized, userId, operatorId, reason)
	}); err != nil {
		return err
	}
	invalidateBlockedIPCache(normalized)
	return nil
}

func DisableUserAndBlockIPs(user *User, ips []string, operatorId int) error {
	normalized, err := NormalizeIPAddresses(ips)
	if err != nil {
		return err
	}
	var previousAuthVersion int64
	if err := DB.Model(&User{}).Where("id = ?", user.Id).Select("auth_version").Find(&previousAuthVersion).Error; err != nil {
		return err
	}
	if err := DB.Transaction(func(tx *gorm.DB) error {
		if err := user.UpdateWithTx(tx, false); err != nil {
			return err
		}
		var sessions []UserSession
		if err := tx.Select("device_id", "ip").Where("user_id = ?", user.Id).Find(&sessions).Error; err != nil {
			return err
		}
		devices := make([]string, 0, len(sessions))
		for _, session := range sessions {
			if device, err := NormalizeDeviceID(session.DeviceID); err == nil {
				devices = append(devices, device)
			}
			if ip, err := NormalizeIPAddress(session.IP); err == nil {
				normalized = append(normalized, ip)
			}
		}
		if err := BlockUserDevicesWithTx(tx, user.Id, devices, operatorId, "user disabled"); err != nil {
			return err
		}
		return BlockIPAddressesWithTx(tx, normalized, user.Id, operatorId, "user disabled")
	}); err != nil {
		return err
	}
	invalidateBlockedIPCache(normalized)
	if err := updateUserCache(*user); err != nil {
		return err
	}
	if user.AuthVersion > previousAuthVersion {
		_, err := RevokeAllUserSessions(user.Id, "admin_disable")
		return err
	}
	return nil
}

func UnblockIPAddresses(ips []string) error {
	normalized, err := NormalizeIPAddresses(ips)
	if err != nil {
		return err
	}
	if len(normalized) == 0 {
		return nil
	}
	if err := DB.Where("ip IN ?", normalized).Delete(&BlockedIP{}).Error; err != nil {
		return err
	}
	invalidateBlockedIPCache(normalized)
	return nil
}

func invalidateBlockedIPCache(ips []string) {
	_, _ = getBlockedIPCache().DeleteMany(ips)
}

func GetBlockedIPSet(ips []string) (map[string]bool, error) {
	result := make(map[string]bool, len(ips))
	if len(ips) == 0 {
		return result, nil
	}
	var records []BlockedIP
	if err := DB.Select("ip").Where("ip IN ?", ips).Find(&records).Error; err != nil {
		return nil, err
	}
	for _, record := range records {
		result[record.IP] = true
	}
	return result, nil
}

func GetSharedLastLoginIPCounts(ips []string) (map[string]int64, error) {
	result := make(map[string]int64, len(ips))
	if len(ips) == 0 {
		return result, nil
	}
	var rows []struct {
		IP    string
		Count int64
	}
	err := DB.Model(&User{}).
		Select("last_login_ip AS ip, COUNT(*) AS count").
		Where("deleted_at IS NULL AND last_login_ip IN ?", ips).
		Group("last_login_ip").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.IP] = row.Count
	}
	return result, nil
}

// BlockDisabledUserRequest is only called after cryptographic credential checks.
// Recheck status under the user row lock so re-enabling cannot race a new ban.
func BlockDisabledUserRequest(userID int, value, deviceID string) (bool, error) {
	ip, err := NormalizeIPAddress(value)
	if err != nil {
		return false, err
	}
	blocked := false
	err = DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).Select("id", "status").First(&user, userID).Error; err != nil {
			return err
		}
		if user.Status != common.UserStatusDisabled {
			return nil
		}
		blocked = true
		record := BlockedIP{IP: ip, UserId: userID, Reason: "disabled user request"}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "ip"}}, DoNothing: true}).Create(&record).Error; err != nil {
			return err
		}
		if device, err := NormalizeDeviceID(deviceID); err == nil {
			return BlockUserDevicesWithTx(tx, userID, []string{device}, 0, "disabled user request")
		}
		return nil
	})
	if blocked && err == nil {
		invalidateBlockedIPCache([]string{ip})
	}
	return blocked, err
}

func ListBlockedIPs(ip string, offset, limit int) ([]BlockedIP, int64, error) {
	query := DB.Model(&BlockedIP{})
	if ip != "" {
		query = query.Where("ip = ?", ip)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	records := make([]BlockedIP, 0)
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&records).Error
	return records, total, err
}
