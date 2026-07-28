package model

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const maxUserQuotaSummaryExcludedUsers = 5000

type UserQuotaSummarySettings struct {
	ExcludedUserIds []int `json:"excluded_user_ids"`
}

func GetUserQuotaSummarySettings() (*UserQuotaSummarySettings, error) {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[common.UserQuotaSummaryExcludedUserIDsOptionKey]
	common.OptionMapRWMutex.RUnlock()
	if strings.TrimSpace(raw) == "" {
		raw = "[]"
	}

	var userIds []int
	if err := common.UnmarshalJsonStr(raw, &userIds); err != nil {
		return nil, err
	}
	userIds = uniquePositiveUserIds(userIds)
	sort.Ints(userIds)
	return &UserQuotaSummarySettings{ExcludedUserIds: userIds}, nil
}

func UpdateUserQuotaSummarySettings(excludedUserIds []int) (*UserQuotaSummarySettings, error) {
	userIds := uniquePositiveUserIds(excludedUserIds)
	if len(userIds) > maxUserQuotaSummaryExcludedUsers {
		return nil, errors.New("不统计用户数量不能超过 5000")
	}
	if len(userIds) > 0 {
		var count int64
		if err := DB.Unscoped().Model(&User{}).Where("id IN ?", userIds).Count(&count).Error; err != nil {
			return nil, err
		}
		if count != int64(len(userIds)) {
			return nil, errors.New("部分用户不存在")
		}
	}

	sort.Ints(userIds)
	encoded, err := common.Marshal(userIds)
	if err != nil {
		return nil, err
	}
	if err := UpdateOption(common.UserQuotaSummaryExcludedUserIDsOptionKey, string(encoded)); err != nil {
		return nil, err
	}
	return &UserQuotaSummarySettings{ExcludedUserIds: userIds}, nil
}

func SearchUserQuotaSummaryOptions(keyword string, startIdx int, limit int) ([]UserManagementOption, int64, error) {
	query := DB.Unscoped().Model(&User{})
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

func ResolveUserQuotaSummaryOptions(userIds []int) ([]UserManagementOption, error) {
	ids := uniquePositiveUserIds(userIds)
	if len(ids) == 0 {
		return []UserManagementOption{}, nil
	}
	var users []UserManagementOption
	err := DB.Unscoped().Model(&User{}).Where("id IN ?", ids).Select("id", "username", "display_name", "email").Order("id desc").Scan(&users).Error
	return users, err
}
