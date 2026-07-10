package controller

import (
	"errors"
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

func getRedemptionScopeUserId(c *gin.Context) (int, error) {
	if c.GetInt("role") >= common.RoleAdminUser {
		return 0, nil
	}
	user, err := model.GetUserById(c.GetInt("id"), false)
	if err != nil {
		return 0, err
	}
	if !user.IsAgent {
		return 0, errors.New("无权访问兑换码管理")
	}
	return user.Id, nil
}

func ensureAdminRedemptionAccess(c *gin.Context) bool {
	if c.GetInt("role") >= common.RoleAdminUser {
		return true
	}
	common.ApiErrorMsg(c, "无权操作兑换码")
	return false
}

func ensureRedemptionReadable(c *gin.Context, redemption *model.Redemption) bool {
	if c.GetInt("role") >= common.RoleAdminUser {
		return true
	}
	userId, err := getRedemptionScopeUserId(c)
	if err != nil || redemption.UserId != userId {
		common.ApiErrorMsg(c, "无权访问该兑换码")
		return false
	}
	return true
}

func calculateAgentRedemptionCharge(quota int, count int, discount int) int {
	if quota <= 0 || count <= 0 || discount <= 0 {
		return 0
	}
	charge := int64(quota) * int64(count) * int64(discount)
	return int((charge + 99) / 100)
}

func GetAllRedemptions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userId, err := getRedemptionScopeUserId(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	redemptions, total, err := model.GetRedemptionsByUser(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
}

func SearchRedemptions(c *gin.Context) {
	keyword := c.Query("keyword")
	status := c.Query("status")
	pageInfo := common.GetPageQuery(c)
	userId, err := getRedemptionScopeUserId(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	redemptions, total, err := model.SearchRedemptionsByUser(userId, keyword, status, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
}

func GetRedemption(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	redemption, err := model.GetRedemptionById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !ensureRedemptionReadable(c, redemption) {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    redemption,
	})
}

func AddRedemption(c *gin.Context) {
	if !operation_setting.IsPaymentComplianceConfirmed() {
		common.ApiErrorI18n(c, i18n.MsgPaymentComplianceRequired)
		return
	}

	redemption := model.Redemption{}
	err := c.ShouldBindJSON(&redemption)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if utf8.RuneCountInString(redemption.Name) == 0 || utf8.RuneCountInString(redemption.Name) > 20 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionNameLength)
		return
	}
	if redemption.Quota <= 0 {
		common.ApiErrorMsg(c, "兑换码额度必须大于 0")
		return
	}
	if redemption.Count <= 0 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionCountPositive)
		return
	}
	if redemption.Count > 100 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionCountMax)
		return
	}
	if valid, msg := validateExpiredTime(c, redemption.ExpiredTime); !valid {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
		return
	}

	creatorId := c.GetInt("id")
	charge := 0
	if c.GetInt("role") < common.RoleAdminUser {
		user, err := model.GetUserById(creatorId, false)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if !user.IsAgent {
			common.ApiErrorMsg(c, "只有代理用户可以生成兑换码")
			return
		}
		if redemption.ExpiredTime != 0 {
			common.ApiErrorMsg(c, "代理生成兑换码不允许设置过期时间")
			return
		}
		if user.AgentDiscount < 0 || user.AgentDiscount > 100 {
			common.ApiErrorMsg(c, "代理折扣配置无效")
			return
		}
		charge = calculateAgentRedemptionCharge(redemption.Quota, redemption.Count, user.AgentDiscount)
	}

	keys, err := model.CreateRedemptionsWithWalletCharge(creatorId, redemption, redemption.Count, charge)
	if err != nil {
		common.SysError("failed to create redemption: " + err.Error())
		common.ApiError(c, err)
		return
	}

	if charge > 0 {
		model.RecordLog(creatorId, model.LogTypeSystem, "代理生成兑换码扣除余额 "+logger.LogQuota(charge))
		_ = model.InvalidateUserCache(creatorId)
	}
	recordManageAudit(c, "redemption.create", map[string]interface{}{
		"name":   redemption.Name,
		"count":  redemption.Count,
		"quota":  logger.LogQuota(redemption.Quota),
		"charge": logger.LogQuota(charge),
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    keys,
	})
}

func DeleteRedemption(c *gin.Context) {
	if !ensureAdminRedemptionAccess(c) {
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	err := model.DeleteRedemptionById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func UpdateRedemption(c *gin.Context) {
	if !ensureAdminRedemptionAccess(c) {
		return
	}
	statusOnly := c.Query("status_only")
	redemption := model.Redemption{}
	err := c.ShouldBindJSON(&redemption)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanRedemption, err := model.GetRedemptionById(redemption.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if statusOnly == "" {
		if valid, msg := validateExpiredTime(c, redemption.ExpiredTime); !valid {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
			return
		}
		// If you add more fields, please also update redemption.Update()
		cleanRedemption.Name = redemption.Name
		cleanRedemption.Quota = redemption.Quota
		cleanRedemption.ExpiredTime = redemption.ExpiredTime
	}
	if statusOnly != "" {
		cleanRedemption.Status = redemption.Status
	}
	err = cleanRedemption.Update()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanRedemption,
	})
}

func DeleteInvalidRedemption(c *gin.Context) {
	if !ensureAdminRedemptionAccess(c) {
		return
	}
	rows, err := model.DeleteInvalidRedemptions()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    rows,
	})
}

func validateExpiredTime(c *gin.Context, expired int64) (bool, string) {
	if expired != 0 && expired < common.GetTimestamp() {
		return false, i18n.T(c, i18n.MsgRedemptionExpireTimeInvalid)
	}
	return true, ""
}
