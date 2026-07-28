package controller

import (
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type updateUserRiskDetectionRequest struct {
	Enabled bool `json:"enabled"`
}

func AdminGetUserRiskReport(c *gin.Context) {
	user, ok := getManageableUserFromParam(c)
	if !ok {
		return
	}
	windowDays := 7
	if value := c.Query("days"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			common.ApiErrorI18n(c, i18n.MsgInvalidParams)
			return
		}
		windowDays = parsed
	}
	if windowDays != 1 && windowDays != 7 && windowDays != 30 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	now := time.Now().Unix()
	enabled := common.UserRiskDetectionEnabled || user.RiskDetectionEnabled
	if !enabled {
		common.ApiSuccess(c, &model.UserRiskReport{
			UserId:        user.Id,
			Enabled:       false,
			GlobalEnabled: common.UserRiskDetectionEnabled,
			UserEnabled:   user.RiskDetectionEnabled,
			WindowDays:    windowDays,
			StartTime:     now - int64(windowDays)*24*60*60,
			EndTime:       now,
			GeneratedAt:   now,
			Level:         model.UserRiskLevelLow,
			Signals:       make([]model.UserRiskSignal, 0),
		})
		return
	}
	report, err := model.GetUserRiskReport(user.Id, windowDays, now)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	report.Enabled = true
	report.GlobalEnabled = common.UserRiskDetectionEnabled
	report.UserEnabled = user.RiskDetectionEnabled
	common.ApiSuccess(c, report)
}

func AdminUpdateUserRiskDetection(c *gin.Context) {
	user, ok := getManageableUserFromParam(c)
	if !ok {
		return
	}
	var request updateUserRiskDetectionRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := model.SetUserRiskDetectionEnabled(user.Id, request.Enabled); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, user.Id, "user.risk_detection", map[string]interface{}{
		"enabled":  request.Enabled,
		"username": user.Username,
	})
	common.ApiSuccess(c, gin.H{
		"enabled":        common.UserRiskDetectionEnabled || request.Enabled,
		"global_enabled": common.UserRiskDetectionEnabled,
		"user_enabled":   request.Enabled,
	})
}
