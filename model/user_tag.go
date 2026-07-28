package model

import (
	"errors"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

const UserTagNameMaxLength = 32

const (
	UserTagRiskMediumId = -1
	UserTagRiskHighId   = -2
)

const (
	UserTagRiskMediumColor = "#C2410C"
	UserTagRiskHighColor   = "#B91C1C"
)

var userTagColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

type UserTag struct {
	Id        int    `json:"id"`
	Name      string `json:"name" gorm:"type:varchar(32);uniqueIndex"`
	Color     string `json:"color" gorm:"type:char(7)"`
	BuiltIn   bool   `json:"built_in,omitempty" gorm:"-:all"`
	RiskLevel string `json:"risk_level,omitempty" gorm:"-:all"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt int64  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

func GetBuiltInUserTag(tagId int) (UserTag, bool) {
	switch tagId {
	case UserTagRiskMediumId:
		return UserTag{
			Id:        UserTagRiskMediumId,
			Name:      "Medium risk",
			Color:     UserTagRiskMediumColor,
			BuiltIn:   true,
			RiskLevel: UserRiskLevelMedium,
		}, true
	case UserTagRiskHighId:
		return UserTag{
			Id:        UserTagRiskHighId,
			Name:      "High risk",
			Color:     UserTagRiskHighColor,
			BuiltIn:   true,
			RiskLevel: UserRiskLevelHigh,
		}, true
	default:
		return UserTag{}, false
	}
}

func IsBuiltInUserTagId(tagId int) bool {
	_, exists := GetBuiltInUserTag(tagId)
	return exists
}

func NormalizeUserTag(tag *UserTag) error {
	if tag == nil {
		return errors.New("标签不能为空")
	}
	tag.Name = strings.TrimSpace(tag.Name)
	tag.Color = strings.ToUpper(strings.TrimSpace(tag.Color))
	if tag.Name == "" {
		return errors.New("标签名称不能为空")
	}
	if len([]rune(tag.Name)) > UserTagNameMaxLength {
		return errors.New("标签名称不能超过 32 个字符")
	}
	if !userTagColorPattern.MatchString(tag.Color) {
		return errors.New("标签颜色必须为 #RRGGBB 格式")
	}
	return nil
}

func ListUserTags() ([]UserTag, error) {
	mediumRiskTag, _ := GetBuiltInUserTag(UserTagRiskMediumId)
	highRiskTag, _ := GetBuiltInUserTag(UserTagRiskHighId)
	tags := []UserTag{mediumRiskTag, highRiskTag}
	var customTags []UserTag
	if err := DB.Order("id asc").Find(&customTags).Error; err != nil {
		return nil, err
	}
	return append(tags, customTags...), nil
}

func CreateUserTag(tag *UserTag) error {
	if err := NormalizeUserTag(tag); err != nil {
		return err
	}
	return DB.Create(tag).Error
}

func UpdateUserTag(tag *UserTag) error {
	if tag == nil || tag.Id <= 0 {
		return errors.New("标签不存在")
	}
	if err := NormalizeUserTag(tag); err != nil {
		return err
	}
	var existing UserTag
	if err := DB.First(&existing, tag.Id).Error; err != nil {
		return err
	}
	if err := DB.Model(&existing).Updates(map[string]interface{}{
		"name":  tag.Name,
		"color": tag.Color,
	}).Error; err != nil {
		return err
	}
	return DB.First(tag, tag.Id).Error
}

func DeleteUserTag(tagId int) error {
	if tagId <= 0 {
		return errors.New("标签不存在")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var tag UserTag
		if err := tx.First(&tag, tagId).Error; err != nil {
			return err
		}
		if err := tx.Model(&User{}).Where("tag_id = ?", tagId).Update("tag_id", 0).Error; err != nil {
			return err
		}
		return tx.Delete(&tag).Error
	})
}

func SetUserTag(userId int, tagId int) error {
	if userId <= 0 || tagId < 0 {
		return errors.New("用户或标签不存在")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if tagId > 0 {
			var count int64
			if err := tx.Model(&UserTag{}).Where("id = ?", tagId).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return gorm.ErrRecordNotFound
			}
		}
		var user User
		if err := tx.Select("id").First(&user, userId).Error; err != nil {
			return err
		}
		return tx.Model(&user).Update("tag_id", tagId).Error
	})
}
