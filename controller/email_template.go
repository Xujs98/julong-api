package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

type emailTemplateRequest struct {
	Subject string `json:"subject"`
	Content string `json:"content"`
}

func ListEmailTemplates(c *gin.Context) {
	common.ApiSuccess(c, service.ListEmailTemplates())
}

func UpdateEmailTemplate(c *gin.Context) {
	var req emailTemplateRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	template, err := service.UpdateEmailTemplate(c.Param("event"), req.Subject, req.Content)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, template)
}

func PreviewEmailTemplate(c *gin.Context) {
	var req emailTemplateRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	preview, err := service.PreviewEmailTemplate(c.Param("event"), req.Subject, req.Content)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, preview)
}

func ResetEmailTemplate(c *gin.Context) {
	template, err := service.ResetEmailTemplate(c.Param("event"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, template)
}
