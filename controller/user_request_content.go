package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type updateUserRequestContentLoggingRequest struct {
	Enabled bool `json:"enabled"`
}

func AdminListUserRequestContentLogs(c *gin.Context) {
	user, ok := getManageableUserFromParam(c)
	if !ok {
		return
	}
	logs, err := model.GetUserRequestContentLogs(user.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"enabled":   user.RequestContentLoggingEnabled,
		"items":     logs,
		"max_items": model.MaxUserRequestContentLogs,
	})
}

func AdminUpdateUserRequestContentLogging(c *gin.Context) {
	user, ok := getManageableUserFromParam(c)
	if !ok {
		return
	}
	var request updateUserRequestContentLoggingRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := model.SetUserRequestContentLogging(user.Id, request.Enabled); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, user.Id, "user.request_content_logging", map[string]interface{}{
		"enabled":  request.Enabled,
		"username": user.Username,
	})
	common.ApiSuccess(c, gin.H{"enabled": request.Enabled})
}

func AdminGetUserRequestContentLog(c *gin.Context) {
	user, ok := getManageableUserFromParam(c)
	if !ok {
		return
	}
	logId, err := strconv.Atoi(c.Param("log_id"))
	if err != nil || logId <= 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	log, err := model.GetUserRequestContentLog(user.Id, logId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorI18n(c, i18n.MsgInvalidParams)
			return
		}
		common.ApiError(c, err)
		return
	}
	content, err := service.DecodeUserRequestContent(log)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, user.Id, "user.request_content_view", map[string]interface{}{
		"request_id": log.RequestId,
		"username":   user.Username,
	})
	common.ApiSuccess(c, gin.H{
		"log":     log,
		"content": content,
	})
}

func AdminDeleteUserRequestContentLogs(c *gin.Context) {
	user, ok := getManageableUserFromParam(c)
	if !ok {
		return
	}
	if err := model.DeleteUserRequestContentLogs(user.Id); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, user.Id, "user.request_content_clear", map[string]interface{}{
		"username": user.Username,
	})
	common.ApiSuccess(c, nil)
}
