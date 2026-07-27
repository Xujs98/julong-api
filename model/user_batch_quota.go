package model

import (
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	UserQuotaAdjustModeAdd      = "add"
	UserQuotaAdjustModeSubtract = "subtract"
	UserQuotaAdjustModeOverride = "override"
)

type UserManagementOption struct {
	Id          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

type UserQuotaAdjustment struct {
	UserId        int
	Username      string
	DisplayName   string
	Email         string
	Mode          string
	Value         int
	PreviousQuota int
	CurrentQuota  int
}

func manageableUsersQuery(db *gorm.DB, operatorRole int) *gorm.DB {
	query := db.Model(&User{})
	if operatorRole < common.RoleRootUser {
		query = query.Where("role < ?", operatorRole)
	}
	return query
}

func SearchManageableUserOptions(keyword string, operatorRole int, startIdx int, limit int) ([]UserManagementOption, int64, error) {
	query := manageableUsersQuery(DB, operatorRole)
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		condition := "username LIKE ? OR email LIKE ? OR display_name LIKE ?"
		args := []interface{}{"%" + keyword + "%", "%" + keyword + "%", "%" + keyword + "%"}
		if id, err := strconv.Atoi(keyword); err == nil {
			condition = "id = ? OR " + condition
			args = append([]interface{}{id}, args...)
		}
		query = query.Where("("+condition+")", args...)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []UserManagementOption
	err := query.Select("id", "username", "display_name", "email").Order("id desc").Limit(limit).Offset(startIdx).Scan(&users).Error
	return users, total, err
}

func ResolveManageableUserOptions(userIds []int, operatorRole int) ([]UserManagementOption, error) {
	ids := uniquePositiveUserIds(userIds)
	if len(ids) == 0 {
		return []UserManagementOption{}, nil
	}
	var users []UserManagementOption
	err := manageableUsersQuery(DB, operatorRole).Where("id IN ?", ids).Select("id", "username", "display_name", "email").Order("id desc").Scan(&users).Error
	return users, err
}

func BatchAdjustUserQuota(operatorRole int, allUsers bool, userIds []int, mode string, value int) ([]UserQuotaAdjustment, error) {
	if mode != UserQuotaAdjustModeAdd && mode != UserQuotaAdjustModeSubtract && mode != UserQuotaAdjustModeOverride {
		return nil, errors.New("额度调整模式无效")
	}
	if mode != UserQuotaAdjustModeOverride && value <= 0 {
		return nil, errors.New("调整额度必须大于 0")
	}
	if value < math.MinInt32 || value > math.MaxInt32 {
		return nil, errors.New("调整额度超出范围")
	}
	ids := uniquePositiveUserIds(userIds)
	if !allUsers && len(ids) == 0 {
		return nil, errors.New("请选择用户")
	}

	adjustments := make([]UserQuotaAdjustment, 0)
	err := DB.Transaction(func(tx *gorm.DB) error {
		query := manageableUsersQuery(lockForUpdate(tx), operatorRole)
		if !allUsers {
			query = query.Where("id IN ?", ids)
		}
		var users []User
		if err := query.Order("id asc").Find(&users).Error; err != nil {
			return err
		}
		if len(users) == 0 {
			return errors.New("没有可调整额度的用户")
		}
		if !allUsers && len(users) != len(ids) {
			return errors.New("部分用户不存在或无权管理")
		}

		adjustments = make([]UserQuotaAdjustment, 0, len(users))
		for _, user := range users {
			currentQuota := int64(user.Quota)
			nextQuota := int64(value)
			switch mode {
			case UserQuotaAdjustModeAdd:
				nextQuota = currentQuota + int64(value)
			case UserQuotaAdjustModeSubtract:
				nextQuota = currentQuota - int64(value)
			}
			if nextQuota < math.MinInt32 || nextQuota > math.MaxInt32 {
				return errors.New("用户额度调整后超出范围")
			}
			resolvedQuota := int(nextQuota)
			if err := tx.Model(&User{}).Where("id = ?", user.Id).Update("quota", resolvedQuota).Error; err != nil {
				return err
			}
			adjustments = append(adjustments, UserQuotaAdjustment{
				UserId:        user.Id,
				Username:      user.Username,
				DisplayName:   user.DisplayName,
				Email:         user.Email,
				Mode:          mode,
				Value:         value,
				PreviousQuota: user.Quota,
				CurrentQuota:  resolvedQuota,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, adjustment := range adjustments {
		if err := updateUserQuotaCache(adjustment.UserId, adjustment.CurrentQuota); err != nil {
			common.SysLog("failed to update user quota cache: " + err.Error())
		}
	}
	return adjustments, nil
}

func uniquePositiveUserIds(userIds []int) []int {
	seen := make(map[int]struct{}, len(userIds))
	ids := make([]int, 0, len(userIds))
	for _, userId := range userIds {
		if userId <= 0 {
			continue
		}
		if _, exists := seen[userId]; exists {
			continue
		}
		seen[userId] = struct{}{}
		ids = append(ids, userId)
	}
	sort.Ints(ids)
	return ids
}
