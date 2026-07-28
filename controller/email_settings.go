package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

type emailSettingsRecipientResolveRequest struct {
	UserIDs []int `json:"user_ids"`
}

type channelAnomalyTestEmailRequest struct {
	RecipientUserIDs []int `json:"recipient_user_ids"`
}

type dashboardReportTestEmailRequest struct {
	RecipientUserIDs []int `json:"recipient_user_ids"`
}

type riskUserTestEmailRequest struct {
	RecipientUserIDs []int    `json:"recipient_user_ids"`
	RiskLevels       []string `json:"risk_levels"`
}

func GetEmailSettingsConfig(c *gin.Context) {
	config, err := service.GetEmailSettingsConfig()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, config)
}

func UpdateEmailSettingsConfig(c *gin.Context) {
	var config service.EmailSettingsConfig
	if err := common.DecodeJson(c.Request.Body, &config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	updated, err := service.UpdateEmailSettingsConfig(config)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, updated)
}

func SearchEmailSettingsRecipients(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	users, total, err := model.SearchOperationalEmailRecipientOptions(
		c.Query("keyword"),
		pageInfo.GetStartIdx(),
		pageInfo.GetPageSize(),
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(users)
	common.ApiSuccess(c, pageInfo)
}

func ResolveEmailSettingsRecipients(c *gin.Context) {
	var req emailSettingsRecipientResolveRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	if len(req.UserIDs) > 100 {
		common.ApiErrorMsg(c, "收件人数量不能超过 100")
		return
	}
	users, err := model.GetOperationalEmailRecipientOptionsByIDs(req.UserIDs)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, users)
}

func SendChannelAnomalyTestEmail(c *gin.Context) {
	var req channelAnomalyTestEmailRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	recipientCount, err := service.SendChannelAnomalyTestEmails(req.RecipientUserIDs)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"recipient_count": recipientCount})
}

func SendDashboardReportTestEmail(c *gin.Context) {
	var req dashboardReportTestEmailRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	result, err := service.SendDashboardReportTestEmails(req.RecipientUserIDs)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func SendRiskUserTestEmail(c *gin.Context) {
	var req riskUserTestEmailRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	result, err := service.SendRiskUserTestEmails(req.RecipientUserIDs, req.RiskLevels)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}
