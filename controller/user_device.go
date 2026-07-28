package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type updateUserDevicesRequest struct {
	DeviceIDs []string `json:"device_ids"`
	Blocked   bool     `json:"blocked"`
}

func AdminGetUserLoginDevices(c *gin.Context) {
	user, ok := getManageableUserFromParam(c)
	if !ok {
		return
	}
	devices, err := model.GetUserLoginDeviceStats(user.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, devices)
}

func AdminUpdateUserLoginDevices(c *gin.Context) {
	user, ok := getManageableUserFromParam(c)
	if !ok {
		return
	}
	var request updateUserDevicesRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || len(request.DeviceIDs) == 0 || len(request.DeviceIDs) > 100 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	deviceIDs, err := model.NormalizeDeviceIDs(request.DeviceIDs)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	revokedCount := int64(0)
	if request.Blocked {
		if err := model.BlockUserDevices(user.Id, deviceIDs, c.GetInt("id"), "manually blocked"); err != nil {
			common.ApiError(c, err)
			return
		}
		revokedCount, err = model.RevokeUserSessionsByDeviceIDs(user.Id, deviceIDs, "admin_device_blocked")
	} else {
		err = model.UnblockUserDevices(user.Id, deviceIDs)
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, user.Id, "user.device_block_update", map[string]interface{}{
		"blocked":       request.Blocked,
		"count":         len(deviceIDs),
		"revoked_count": revokedCount,
		"user_id":       user.Id,
	})
	common.ApiSuccess(c, gin.H{"revoked_count": revokedCount})
}
