package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

type resolveUserManagementOptionsRequest struct {
	UserIds []int `json:"user_ids"`
}

type batchQuotaAdjustRequest struct {
	Mode         string `json:"mode"`
	Value        int    `json:"value"`
	AllUsers     bool   `json:"all_users"`
	UserIds      []int  `json:"user_ids"`
	SendEmail    bool   `json:"send_email"`
	EmailLocale  string `json:"email_locale"`
	EmailSubject string `json:"email_subject"`
	EmailContent string `json:"email_content"`
}

func SearchUserManagementOptions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	users, total, err := model.SearchManageableUserOptions(
		c.Query("keyword"),
		c.GetInt("role"),
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

func ResolveUserManagementOptions(c *gin.Context) {
	var request resolveUserManagementOptionsRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	users, err := model.ResolveManageableUserOptions(request.UserIds, c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, users)
}

func BatchAdjustUserQuota(c *gin.Context) {
	var request batchQuotaAdjustRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	request.EmailSubject = strings.TrimSpace(request.EmailSubject)
	if request.SendEmail {
		if common.SMTPServer == "" && common.SMTPAccount == "" {
			common.ApiError(c, errors.New("SMTP 服务器未配置"))
			return
		}
		if request.EmailSubject == "" || strings.TrimSpace(request.EmailContent) == "" {
			common.ApiError(c, errors.New("邮件主题和内容不能为空"))
			return
		}
		if err := service.ValidateUserQuotaAdjustmentEmail(request.EmailLocale, request.EmailSubject, request.EmailContent); err != nil {
			common.ApiError(c, err)
			return
		}
	}

	adjustments, err := model.BatchAdjustUserQuota(
		c.GetInt("role"),
		request.AllUsers,
		request.UserIds,
		request.Mode,
		request.Value,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	for _, adjustment := range adjustments {
		increase := adjustment.CurrentQuota - adjustment.PreviousQuota
		if increase > 0 {
			model.RecordQuotaIncreaseLog(
				adjustment.UserId,
				increase,
				model.QuotaIncreaseSourceAdminAdjustment,
				fmt.Sprintf(
					"管理员 %s 批量调整额度，从 %s 调整为 %s",
					c.GetString("username"),
					logger.LogQuota(adjustment.PreviousQuota),
					logger.LogQuota(adjustment.CurrentQuota),
				),
			)
		}
	}
	recordManageAudit(c, "user.batch_quota_adjust", map[string]interface{}{
		"mode":           request.Mode,
		"value":          logger.LogQuota(request.Value),
		"all_users":      request.AllUsers,
		"adjusted_count": len(adjustments),
		"send_email":     request.SendEmail,
	})

	emailResult := service.UserQuotaAdjustmentEmailResult{}
	if request.SendEmail {
		emailResult = service.SendUserQuotaAdjustmentEmails(
			adjustments,
			c.GetString("username"),
			request.EmailLocale,
			request.EmailSubject,
			request.EmailContent,
		)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"adjusted_count":      len(adjustments),
			"email_success_count": emailResult.SuccessCount,
			"email_skipped_count": emailResult.SkippedCount,
			"email_failed_count":  emailResult.FailedCount,
		},
	})
}
