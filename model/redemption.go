package model

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"gorm.io/gorm"
)

type Redemption struct {
	Id                 int            `json:"id"`
	UserId             int            `json:"user_id"`
	Key                string         `json:"key" gorm:"type:char(32);uniqueIndex"`
	Status             int            `json:"status" gorm:"default:1"`
	Name               string         `json:"name" gorm:"index"`
	Quota              int            `json:"quota" gorm:"default:100"`
	CreatedTime        int64          `json:"created_time" gorm:"bigint"`
	RedeemedTime       int64          `json:"redeemed_time" gorm:"bigint"`
	Count              int            `json:"count" gorm:"-:all"` // only for api request
	UsedUserId         int            `json:"used_user_id"`
	DeletedAt          gorm.DeletedAt `gorm:"index"`
	ExpiredTime        int64          `json:"expired_time" gorm:"bigint"` // 过期时间，0 表示不过期
	AgentCharge        int            `json:"agent_charge" gorm:"default:0"`
	CreatorUsername    string         `json:"creator_username,omitempty" gorm:"-:all"`
	CreatorDisplayName string         `json:"creator_display_name,omitempty" gorm:"-:all"`
	CreatorRole        int            `json:"creator_role,omitempty" gorm:"-:all"`
}

func populateRedemptionCreators(redemptions []*Redemption) {
	userIds := make([]int, 0)
	seen := make(map[int]bool)
	for _, redemption := range redemptions {
		if redemption == nil || redemption.UserId <= 0 || seen[redemption.UserId] {
			continue
		}
		seen[redemption.UserId] = true
		userIds = append(userIds, redemption.UserId)
	}
	if len(userIds) == 0 {
		return
	}

	var users []User
	if err := DB.Select("id", "username", "display_name", "role").Where("id IN ?", userIds).Find(&users).Error; err != nil {
		common.SysError("failed to populate redemption creators: " + err.Error())
		return
	}
	userMap := make(map[int]User, len(users))
	for _, user := range users {
		userMap[user.Id] = user
	}
	for _, redemption := range redemptions {
		if redemption == nil {
			continue
		}
		if user, ok := userMap[redemption.UserId]; ok {
			redemption.CreatorUsername = user.Username
			redemption.CreatorDisplayName = user.DisplayName
			redemption.CreatorRole = user.Role
		}
	}
}

func GetAllRedemptions(startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	return GetRedemptionsByUser(0, startIdx, num)
}

func GetRedemptionsByUser(userId int, startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取总数
	query := tx.Model(&Redemption{})
	if userId > 0 {
		query = query.Where("user_id = ?", userId)
	}
	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 获取分页数据
	err = query.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}
	// 提交事务
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	populateRedemptionCreators(redemptions)

	return redemptions, total, nil
}

func SearchRedemptions(keyword string, status string, startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	return SearchRedemptionsByUser(0, keyword, status, startIdx, num)
}

func SearchRedemptionsByUser(userId int, keyword string, status string, startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&Redemption{})
	if userId > 0 {
		query = query.Where("user_id = ?", userId)
	}

	if keyword != "" {
		if id, err := strconv.Atoi(keyword); err == nil {
			query = query.Where("id = ? OR name LIKE ?", id, keyword+"%")
		} else {
			query = query.Where("name LIKE ?", keyword+"%")
		}
	}

	if status != "" {
		now := common.GetTimestamp()
		switch status {
		case "expired":
			query = query.Where(
				"status = ? AND expired_time != 0 AND expired_time < ?",
				common.RedemptionCodeStatusEnabled,
				now,
			)
		case strconv.Itoa(common.RedemptionCodeStatusEnabled):
			query = query.Where(
				"status = ? AND (expired_time = 0 OR expired_time >= ?)",
				common.RedemptionCodeStatusEnabled,
				now,
			)
		case strconv.Itoa(common.RedemptionCodeStatusDisabled):
			query = query.Where("status = ?", common.RedemptionCodeStatusDisabled)
		case strconv.Itoa(common.RedemptionCodeStatusUsed):
			query = query.Where("status = ?", common.RedemptionCodeStatusUsed)
		}
	}

	// Get total count
	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated data
	err = query.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	populateRedemptionCreators(redemptions)

	return redemptions, total, nil
}

func GetRedemptionById(id int) (*Redemption, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	var err error = nil
	err = DB.First(&redemption, "id = ?", id).Error
	if err == nil {
		populateRedemptionCreators([]*Redemption{&redemption})
	}
	return &redemption, err
}

func Redeem(key string, userId int) (quota int, err error) {
	if key == "" {
		return 0, errors.New("未提供兑换码")
	}
	if userId == 0 {
		return 0, errors.New("无效的 user id")
	}
	redemption := &Redemption{}

	keyCol := "`key`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		keyCol = `"key"`
	}
	common.RandomSleep()
	err = DB.Transaction(func(tx *gorm.DB) error {
		err := lockForUpdate(tx).Where(keyCol+" = ?", key).First(redemption).Error
		if err != nil {
			return errors.New("无效的兑换码")
		}
		if redemption.Status != common.RedemptionCodeStatusEnabled {
			return errors.New("该兑换码已被使用")
		}
		if redemption.ExpiredTime != 0 && redemption.ExpiredTime < common.GetTimestamp() {
			return errors.New("该兑换码已过期")
		}
		var redeemer User
		if err := tx.Select("id", "is_agent").First(&redeemer, "id = ?", userId).Error; err != nil {
			return err
		}
		if redeemer.IsAgent {
			var creator User
			if redemption.UserId <= 0 {
				return errors.New("代理用户只能使用管理员或 root 生成的兑换码")
			}
			if err := tx.Select("id", "role").First(&creator, "id = ?", redemption.UserId).Error; err != nil {
				return err
			}
			if creator.Role < common.RoleAdminUser {
				return errors.New("代理用户只能使用管理员或 root 生成的兑换码")
			}
		}
		// Compare-and-swap on status: only the transaction that flips
		// enabled -> used may credit quota, so a concurrent redeem of the
		// same code loses here even without a row lock (e.g. on SQLite).
		result := tx.Model(&Redemption{}).
			Where("id = ? AND status = ?", redemption.Id, common.RedemptionCodeStatusEnabled).
			Updates(map[string]interface{}{
				"redeemed_time": common.GetTimestamp(),
				"status":        common.RedemptionCodeStatusUsed,
				"used_user_id":  userId,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("该兑换码已被使用")
		}
		return tx.Model(&User{}).Where("id = ?", userId).Update("quota", gorm.Expr("quota + ?", redemption.Quota)).Error
	})
	if err != nil {
		common.SysError("redemption failed: " + err.Error())
		return 0, ErrRedeemFailed
	}
	RecordLog(userId, LogTypeTopup, fmt.Sprintf("通过兑换码充值 %s，兑换码ID %d", logger.LogQuota(redemption.Quota), redemption.Id))
	return redemption.Quota, nil
}

func (redemption *Redemption) Insert() error {
	var err error
	err = DB.Create(redemption).Error
	return err
}

func CreateRedemptionsWithWalletCharge(userId int, redemption Redemption, count int, charge int) ([]string, error) {
	if userId == 0 {
		return nil, errors.New("无效的 user id")
	}
	if count <= 0 {
		return nil, errors.New("兑换码数量必须大于 0")
	}
	keys := make([]string, 0, count)
	err := DB.Transaction(func(tx *gorm.DB) error {
		if charge > 0 {
			result := tx.Model(&User{}).
				Where("id = ? AND quota >= ?", userId, charge).
				Update("quota", gorm.Expr("quota - ?", charge))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return errors.New("余额不足，无法生成兑换码")
			}
		}

		baseAgentCharge := 0
		agentChargeRemainder := 0
		if charge > 0 {
			baseAgentCharge = charge / count
			agentChargeRemainder = charge % count
		}
		for i := 0; i < count; i++ {
			key := common.GetUUID()
			agentCharge := baseAgentCharge
			if i < agentChargeRemainder {
				agentCharge++
			}
			cleanRedemption := Redemption{
				UserId:      userId,
				Name:        redemption.Name,
				Key:         key,
				CreatedTime: common.GetTimestamp(),
				Quota:       redemption.Quota,
				ExpiredTime: redemption.ExpiredTime,
				AgentCharge: agentCharge,
			}
			if err := tx.Create(&cleanRedemption).Error; err != nil {
				return err
			}
			keys = append(keys, key)
		}
		return nil
	})
	return keys, err
}

func (redemption *Redemption) SelectUpdate() error {
	// This can update zero values
	return DB.Model(redemption).Select("redeemed_time", "status").Updates(redemption).Error
}

// Update Make sure your token's fields is completed, because this will update non-zero values
func (redemption *Redemption) Update() error {
	var err error
	err = DB.Model(redemption).Select("name", "status", "quota", "redeemed_time", "expired_time", "agent_charge").Updates(redemption).Error
	return err
}

func (redemption *Redemption) Delete() error {
	var err error
	err = DB.Delete(redemption).Error
	return err
}

func DeleteRedemptionById(id int) (err error) {
	if id == 0 {
		return errors.New("id 为空！")
	}
	var refundUserId int
	var refundAmount int
	var refundedRedemption Redemption
	err = DB.Transaction(func(tx *gorm.DB) error {
		redemption := Redemption{Id: id}
		if err := tx.Where(redemption).First(&redemption).Error; err != nil {
			return err
		}
		userId, refund, err := refundAgentRedemptionCharge(tx, &redemption)
		if err != nil {
			return err
		}
		if refund > 0 {
			refundUserId = userId
			refundAmount = refund
			refundedRedemption = redemption
		}
		return tx.Delete(&redemption).Error
	})
	if err == nil && refundAmount > 0 {
		RecordAgentRedemptionRefundLog(refundUserId, &refundedRedemption, refundAmount)
	}
	return err
}

func DeleteInvalidRedemptions() (int64, error) {
	now := common.GetTimestamp()
	var rowsAffected int64
	type pendingRefundLog struct {
		userId     int
		refund     int
		redemption Redemption
	}
	var refundLogs []pendingRefundLog
	err := DB.Transaction(func(tx *gorm.DB) error {
		var redemptions []Redemption
		if err := tx.Where(
			"status IN ? OR (status = ? AND expired_time != 0 AND expired_time < ?)",
			[]int{common.RedemptionCodeStatusUsed, common.RedemptionCodeStatusDisabled},
			common.RedemptionCodeStatusEnabled,
			now,
		).Find(&redemptions).Error; err != nil {
			return err
		}
		if len(redemptions) == 0 {
			return nil
		}
		for i := range redemptions {
			userId, refund, err := refundAgentRedemptionCharge(tx, &redemptions[i])
			if err != nil {
				return err
			}
			if refund > 0 {
				refundLogs = append(refundLogs, pendingRefundLog{
					userId:     userId,
					refund:     refund,
					redemption: redemptions[i],
				})
			}
		}
		result := tx.Delete(&redemptions)
		rowsAffected = result.RowsAffected
		return result.Error
	})
	if err == nil {
		for _, refundLog := range refundLogs {
			RecordAgentRedemptionRefundLog(refundLog.userId, &refundLog.redemption, refundLog.refund)
		}
	}
	return rowsAffected, err
}

func refundAgentRedemptionCharge(tx *gorm.DB, redemption *Redemption) (int, int, error) {
	if redemption == nil || redemption.UserId <= 0 || redemption.Status == common.RedemptionCodeStatusUsed {
		return 0, 0, nil
	}
	var creator User
	if err := tx.Select("id", "is_agent", "agent_discount").First(&creator, "id = ?", redemption.UserId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	if !creator.IsAgent {
		return 0, 0, nil
	}
	refund := redemption.AgentCharge
	if refund <= 0 {
		refund = int((int64(redemption.Quota)*int64(creator.AgentDiscount) + 99) / 100)
	}
	if refund <= 0 {
		return 0, 0, nil
	}
	if err := tx.Model(&User{}).Where("id = ?", creator.Id).Update("quota", gorm.Expr("quota + ?", refund)).Error; err != nil {
		return 0, 0, err
	}
	if err := tx.Model(redemption).Update("agent_charge", 0).Error; err != nil {
		return 0, 0, err
	}
	return creator.Id, refund, nil
}
