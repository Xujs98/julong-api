package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func GetImageGenerationLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	startTime, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTime, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	channelId, _ := strconv.Atoi(c.Query("channel_id"))
	role := c.GetInt("role")
	visibleLimit := 0
	if role < common.RoleAdminUser {
		allowed, limit, accessErr := model.GetUserImageGenerationLogAccess(c.GetInt("id"))
		if accessErr != nil {
			common.ApiError(c, accessErr)
			return
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "当前订阅不包含生图日志查看权限"})
			return
		}
		visibleLimit = limit
	}
	logs, total, err := model.GetImageGenerationLogs(
		c.GetInt("id"),
		role >= common.RoleAdminUser,
		visibleLimit,
		pageInfo.GetStartIdx(),
		pageInfo.GetPageSize(),
		channelId,
		strings.TrimSpace(c.Query("model")),
		strings.TrimSpace(c.Query("prompt")),
		startTime,
		endTime,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	for _, log := range logs {
		refs, refErr := log.ImageRefs()
		if refErr != nil {
			continue
		}
		log.ImageUrls = make([]string, len(refs))
		for index, ref := range refs {
			if ref.Type == "remote" {
				log.ImageUrls[index] = ref.Value
			} else {
				log.ImageUrls[index] = "/api/image-generation-logs/" + strconv.Itoa(log.Id) + "/images/" + strconv.Itoa(index)
			}
		}
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
}

func GetImageGenerationLogImage(c *gin.Context) {
	id, idErr := strconv.Atoi(c.Param("id"))
	index, indexErr := strconv.Atoi(c.Param("index"))
	if idErr != nil || indexErr != nil || id <= 0 || index < 0 {
		common.ApiErrorMsg(c, "无效的图片日志参数")
		return
	}
	log, err := model.GetImageGenerationLogById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if c.GetInt("role") < common.RoleAdminUser {
		userId := c.GetInt("id")
		allowed, limit, accessErr := model.GetUserImageGenerationLogAccess(userId)
		if accessErr != nil || !allowed || log.UserId != userId {
			c.Status(http.StatusForbidden)
			return
		}
		visible, visibleErr := model.IsImageGenerationLogVisibleToUser(log.Id, userId, limit)
		if visibleErr != nil || !visible {
			c.Status(http.StatusForbidden)
			return
		}
	}
	data, mimeType, err := service.ReadImageGenerationLogImage(log, index)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.Data(http.StatusOK, mimeType, data)
}

func GetImageGenerationLogTask(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "无效的生图日志参数")
		return
	}
	log, err := model.GetImageGenerationLogById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !canAccessImageGenerationLog(c, log) {
		c.Status(http.StatusForbidden)
		return
	}
	payload, err := service.BuildImageGenerationTaskPayload(log, func(index int) string {
		return fmt.Sprintf("/api/image-generation-logs/%d/images/%d", log.Id, index)
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, payload)
}

func canAccessImageGenerationLog(c *gin.Context, log *model.ImageGenerationLog) bool {
	if c.GetInt("role") >= common.RoleAdminUser {
		return true
	}
	userId := c.GetInt("id")
	allowed, limit, err := model.GetUserImageGenerationLogAccess(userId)
	if err != nil || !allowed || log.UserId != userId {
		return false
	}
	visible, err := model.IsImageGenerationLogVisibleToUser(log.Id, userId, limit)
	return err == nil && visible
}
