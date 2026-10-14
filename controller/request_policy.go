package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func requestPolicyOptionAllowed(key, action string) bool {
	switch action {
	case "request-policies.filtering", "security.sensitive-words":
		return key == "CheckSensitiveEnabled" || key == "CheckSensitiveOnPromptEnabled" || key == "SensitiveWords"
	case "request-policies.routing":
		return strings.HasPrefix(key, "channel_affinity_setting.") || key == "RetryTimes" || key == "AutomaticRetryStatusCodes"
	case "request-policies.health", "operations.monitoring":
		return strings.HasPrefix(key, "monitor_setting.") || key == "AutomaticDisableChannelEnabled" || key == "AutomaticEnableChannelEnabled" || key == "AutomaticDisableStatusCodes" || key == "AutomaticDisableKeywords" || key == "ChannelDisableThreshold"
	default:
		return optionAllowedForSystemSettingsAction(key, action)
	}
}

func GetRequestPolicy(c *gin.Context) {
	action := systemSettingsAction(c)
	if !canAccessSystemSettingsAction(c, action) {
		common.ApiErrorStatus(c, http.StatusForbidden, common.NewMessage("Unauthorized, insufficient privileges"))
		return
	}
	options := model.CurrentRequestPolicy().Options
	if c.GetInt("role") < common.RoleRootUser {
		filtered := make(map[string]string)
		for key, value := range options {
			if requestPolicyOptionAllowed(key, action) {
				filtered[key] = value
			}
		}
		options = filtered
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"options": options}})
}

func UpdateRequestPolicy(c *gin.Context) {
	action := systemSettingsAction(c)
	if !canAccessSystemSettingsAction(c, action) {
		common.ApiErrorStatus(c, http.StatusForbidden, common.NewMessage("Unauthorized, insufficient privileges"))
		return
	}
	var request struct {
		Options map[string]string `json:"options"`
	}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorStatus(c, http.StatusBadRequest, err)
		return
	}
	if c.GetInt("role") < common.RoleRootUser {
		for key := range request.Options {
			if !requestPolicyOptionAllowed(key, action) {
				common.ApiErrorStatus(c, http.StatusForbidden, common.NewMessage("Unauthorized, insufficient privileges"))
				return
			}
		}
	}
	if err := model.UpdateRequestPolicyOptions(request.Options); err != nil {
		common.ApiErrorStatus(c, http.StatusBadRequest, err)
		return
	}
	GetRequestPolicy(c)
}
