package controller

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type submitErrorReportRequest struct {
	Title       string `json:"title"`
	Message     string `json:"message"`
	PageUrl     string `json:"page_url"`
	ErrorStatus int    `json:"error_status"`
	Stack       string `json:"stack"`
}

func trimRunes(value string, max int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

func SubmitErrorReport(c *gin.Context) {
	var req submitErrorReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	message := trimRunes(req.Message, 5000)
	if message == "" {
		common.ApiErrorMsg(c, "错误信息不能为空")
		return
	}

	errorStatus := req.ErrorStatus
	if errorStatus <= 0 {
		errorStatus = 500
	}

	report := &model.ErrorReport{
		UserId:      c.GetInt("id"),
		Username:    c.GetString("username"),
		Title:       trimRunes(req.Title, 200),
		Message:     message,
		PageUrl:     trimRunes(req.PageUrl, 2000),
		ErrorStatus: errorStatus,
		UserAgent:   trimRunes(c.GetHeader("User-Agent"), 1000),
		Stack:       trimRunes(req.Stack, 10000),
		Ip:          c.ClientIP(),
	}

	if err := model.CreateErrorReport(report); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, report)
}

func GetErrorReports(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	reports, total, err := model.GetErrorReports(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(reports)
	common.ApiSuccess(c, pageInfo)
}

func GetErrorReport(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	report, err := model.GetErrorReportById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, report)
}
