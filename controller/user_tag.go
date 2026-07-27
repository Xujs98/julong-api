package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type userTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type userTagAssignmentRequest struct {
	TagId int `json:"tag_id"`
}

func ListUserTags(c *gin.Context) {
	tags, err := model.ListUserTags()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, tags)
}

func CreateUserTag(c *gin.Context) {
	var request userTagRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tag := model.UserTag{Name: request.Name, Color: request.Color}
	if err := model.CreateUserTag(&tag); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "user.tag_create", map[string]interface{}{
		"tag_id": tag.Id,
		"name":   tag.Name,
		"color":  tag.Color,
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": tag})
}

func UpdateUserTag(c *gin.Context) {
	tagId, err := strconv.Atoi(c.Param("tag_id"))
	if err != nil || tagId <= 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	var request userTagRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tag := model.UserTag{Id: tagId, Name: request.Name, Color: request.Color}
	if err := model.UpdateUserTag(&tag); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "user.tag_update", map[string]interface{}{
		"tag_id": tag.Id,
		"name":   tag.Name,
		"color":  tag.Color,
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": tag})
}

func DeleteUserTag(c *gin.Context) {
	tagId, err := strconv.Atoi(c.Param("tag_id"))
	if err != nil || tagId <= 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := model.DeleteUserTag(tagId); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "user.tag_delete", map[string]interface{}{"tag_id": tagId})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

func UpdateUserTagAssignment(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil || userId <= 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	var request userTagAssignmentRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || request.TagId < 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	user, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !canManageTargetRole(c.GetInt("role"), user.Role) {
		common.ApiErrorI18n(c, i18n.MsgUserNoPermissionHigherLevel)
		return
	}
	if err := model.SetUserTag(userId, request.TagId); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, userId, "user.tag_assign", map[string]interface{}{
		"tag_id": request.TagId,
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}
