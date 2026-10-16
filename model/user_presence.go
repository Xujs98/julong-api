package model

import (
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"time"

	"gorm.io/gorm/clause"
)

type UserPresence struct {
	UserId            int    `json:"user_id" gorm:"primaryKey;column:user_id"`
	IsOnline          bool   `json:"is_online" gorm:"type:bool;not null;column:is_online;index"`
	LastActivityAt    int64  `json:"last_activity_at" gorm:"type:bigint;not null;column:last_activity_at;index"`
	LastWebActivityAt int64  `json:"last_web_activity_at" gorm:"type:bigint;not null;default:0;index"`
	LastAPIActivityAt int64  `json:"last_api_activity_at" gorm:"type:bigint;not null;default:0;index"`
	LastSource        string `json:"last_source" gorm:"type:varchar(32);not null;column:last_source"`
	LastIP            string `json:"last_ip" gorm:"type:varchar(45);not null;column:last_ip"`
	LastUserAgent     string `json:"last_user_agent" gorm:"type:varchar(512);not null;column:last_user_agent"`
	LastChangedAt     int64  `json:"last_changed_at" gorm:"type:bigint;not null;column:last_changed_at"`
	UpdatedAt         int64  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
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
	activityColumn := "last_web_activity_at"
	if source == "api" {
		activityColumn = "last_api_activity_at"
		presence.LastAPIActivityAt = activityAt
	} else {
		presence.LastWebActivityAt = activityAt
	}
	created := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&presence)
	if created.Error != nil {
		return UserPresence{}, false, created.Error
	}
	if created.RowsAffected == 1 {
		return presence, true, nil
	}
	// Sources advance independently, including requests in the same second.
	if err := DB.Model(&UserPresence{}).
		Where("user_id = ? AND "+activityColumn+" < ?", userId, activityAt).
		UpdateColumn(activityColumn, activityAt).Error; err != nil {
		return UserPresence{}, false, err
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
		Where("id IN ? AND status = ? AND deleted_at IS NULL AND role <> ?", userIds, common.UserStatusEnabled, common.RoleRootUser).
		Order("id ASC").
		Find(&users).Error
	return users, err
}

// UserOnlineWindowSeconds is independent of notification configuration.
const UserOnlineWindowSeconds int64 = 5 * 60

func (options UserSortOptions) ApplyPresence(query *gorm.DB, now int64) *gorm.DB {
	if options.Presence != "online" && options.Presence != "offline" {
		return query
	}
	recent := query.Session(&gorm.Session{NewDB: true}).Model(&UserPresence{}).Select("user_id").Where("(last_web_activity_at > ? AND last_web_activity_at <= ?) OR (last_api_activity_at > ? AND last_api_activity_at <= ?)", now-UserOnlineWindowSeconds, now, now-UserOnlineWindowSeconds, now)
	// Exclude Root from both sides, preventing state inference by filtering.
	query = query.Where("role <> ?", common.RoleRootUser)
	if options.Presence == "online" {
		return query.Where("status = ? AND deleted_at IS NULL AND id IN (?)", common.UserStatusEnabled, recent)
	}
	return query.Where("(status <> ? OR deleted_at IS NOT NULL OR id NOT IN (?))", common.UserStatusEnabled, recent)
}

func FillUserOnlineStatus(users []*User, now int64) error {
	ids := make([]int, 0, len(users))
	for _, user := range users {
		user.IsOnline = nil
		user.OnlineSource = ""
		if user.Role != common.RoleRootUser {
			ids = append(ids, user.Id)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	var presences []UserPresence
	if err := DB.Select("user_id", "last_web_activity_at", "last_api_activity_at").Where("user_id IN ?", ids).Find(&presences).Error; err != nil {
		return err
	}
	active := make(map[int]string, len(presences))
	for _, presence := range presences {
		web := presence.LastWebActivityAt > now-UserOnlineWindowSeconds && presence.LastWebActivityAt <= now
		api := presence.LastAPIActivityAt > now-UserOnlineWindowSeconds && presence.LastAPIActivityAt <= now
		switch {
		case web && api:
			active[presence.UserId] = "both"
		case web:
			active[presence.UserId] = "web"
		case api:
			active[presence.UserId] = "api"
		}
	}
	for _, user := range users {
		if user.Role == common.RoleRootUser {
			continue
		}
		online := user.Status == common.UserStatusEnabled && !user.DeletedAt.Valid && active[user.Id] != ""
		user.IsOnline = &online
		user.OnlineSource = "offline"
		if online {
			user.OnlineSource = active[user.Id]
		}
	}
	return nil
}

type UserActivitySummary struct {
	DailyActiveUsers    int64 `json:"daily_active_users"`
	OnlineUsers         int64 `json:"online_users"`
	DayStart            int64 `json:"day_start"`
	GeneratedAt         int64 `json:"generated_at"`
	OnlineWindowSeconds int64 `json:"online_window_seconds"`
}

func GetUserActivitySummary(now time.Time) (*UserActivitySummary, error) {
	local := now.In(time.FixedZone("Asia/Shanghai", 8*60*60))
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location()).Unix()
	result := UserActivitySummary{DayStart: start, GeneratedAt: now.Unix(), OnlineWindowSeconds: UserOnlineWindowSeconds}
	query := DB.Model(&UserPresence{}).Joins("JOIN users ON users.id = user_presences.user_id").Where("users.role <> ? AND users.deleted_at IS NULL", common.RoleRootUser)
	if err := query.Session(&gorm.Session{}).Where("user_presences.last_activity_at >= ? AND user_presences.last_activity_at <= ?", start, now.Unix()).Count(&result.DailyActiveUsers).Error; err != nil {
		return nil, err
	}
	if err := query.Session(&gorm.Session{}).Where("users.status = ?", common.UserStatusEnabled).
		Where("(user_presences.last_web_activity_at > ? AND user_presences.last_web_activity_at <= ?) OR (user_presences.last_api_activity_at > ? AND user_presences.last_api_activity_at <= ?)", now.Unix()-UserOnlineWindowSeconds, now.Unix(), now.Unix()-UserOnlineWindowSeconds, now.Unix()).Count(&result.OnlineUsers).Error; err != nil {
		return nil, err
	}
	return &result, nil
}
