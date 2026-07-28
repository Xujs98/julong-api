package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type updateUserQuotaSummarySettingsRequest struct {
	ExcludedUserIds []int `json:"excluded_user_ids"`
}

type resolveUserQuotaSummaryOptionsRequest struct {
	UserIds []int `json:"user_ids"`
}

func GetUserQuotaSummarySettings(c *gin.Context) {
	settings, err := model.GetUserQuotaSummarySettings()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, settings)
}

func UpdateUserQuotaSummarySettings(c *gin.Context) {
	var request updateUserQuotaSummarySettingsRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	settings, err := model.UpdateUserQuotaSummarySettings(request.ExcludedUserIds)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "user.quota_summary_settings.update", map[string]interface{}{
		"excluded_user_count": len(settings.ExcludedUserIds),
	})
	common.ApiSuccess(c, settings)
}

func SearchUserQuotaSummaryOptions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	users, total, err := model.SearchUserQuotaSummaryOptions(c.Query("keyword"), pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(users)
	common.ApiSuccess(c, pageInfo)
}

func ResolveUserQuotaSummaryOptions(c *gin.Context) {
	var request resolveUserQuotaSummaryOptionsRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	users, err := model.ResolveUserQuotaSummaryOptions(request.UserIds)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, users)
}
