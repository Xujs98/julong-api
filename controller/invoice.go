package controller

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

const maxInvoiceAttachmentBytes = 10 * 1024 * 1024

var sendInvoiceEmail = common.SendEmailWithAttachments

type invoiceCreditsRequest struct {
	RequestIds []string `json:"request_ids"`
}

type invoiceUpdateRequest struct {
	Status    string `json:"status"`
	AdminNote string `json:"admin_note"`
}

func invoiceResponse(c *gin.Context, userId int) {
	setting := operation_setting.GetInvoiceSetting()
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil || pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	creditPage, err := strconv.Atoi(c.DefaultQuery("credit_page", "1"))
	if err != nil || creditPage < 1 {
		creditPage = 1
	}
	applicationPage, err := strconv.Atoi(c.DefaultQuery("application_page", "1"))
	if err != nil || applicationPage < 1 {
		applicationPage = 1
	}
	credits, total, err := model.GetInvoiceCredits(userId, (creditPage-1)*pageSize, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	applications, applicationTotal, err := model.GetInvoiceApplications(userId, (applicationPage-1)*pageSize, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"enabled":                setting.Enabled,
		"minimum_amount":         setting.MinimumAmount,
		"unit":                   setting.Unit,
		"processing_days":        setting.ProcessingDays,
		"rejection_freeze_hours": setting.RejectionFreezeHours,
		"credits":                credits,
		"credits_total":          total,
		"applications":           applications,
		"applications_total":     applicationTotal,
		"page_size":              pageSize,
	})
}

func GetInvoiceInfo(c *gin.Context) {
	invoiceResponse(c, c.GetInt("id"))
}

func CancelInvoiceApplication(c *gin.Context) {
	applicationId, err := strconv.Atoi(c.Param("id"))
	if err != nil || applicationId <= 0 {
		common.ApiErrorMsg(c, "无效的申请 ID")
		return
	}
	application, err := model.CancelInvoiceApplication(applicationId, c.GetInt("id"))
	if err != nil {
		if errors.Is(err, model.ErrInvoiceNotCancellable) {
			common.ApiErrorMsg(c, "仅待确认的发票申请可以取消")
			return
		}
		common.ApiError(c, err)
		return
	}
	application.AdminNote = ""
	common.ApiSuccess(c, application)
}

func DownloadInvoiceAttachment(c *gin.Context) {
	applicationId, err := strconv.Atoi(c.Param("id"))
	if err != nil || applicationId <= 0 {
		common.ApiErrorMsg(c, "无效的申请 ID")
		return
	}
	role := c.GetInt("role")
	_, attachment, err := model.GetInvoiceAttachment(applicationId, c.GetInt("id"), role >= common.RoleAdminUser)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	contentType := strings.TrimSpace(attachment.ContentType)
	if _, _, parseErr := mime.ParseMediaType(contentType); parseErr != nil {
		contentType = "application/octet-stream"
	}
	filename := filepath.Base(strings.ReplaceAll(attachment.Filename, "\\", "/"))
	if filename == "" || filename == "." {
		filename = "invoice"
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	c.Data(http.StatusOK, contentType, attachment.Data)
}

func ApplyInvoice(c *gin.Context) {
	setting := operation_setting.GetInvoiceSetting()
	if !setting.Enabled {
		common.ApiErrorMsg(c, "自助发票功能当前未开放，仍在内测中")
		return
	}
	var req invoiceCreditsRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil || len(req.RequestIds) == 0 || len(req.RequestIds) > 100 {
		common.ApiErrorMsg(c, "请选择有效的充值记录")
		return
	}
	user, err := model.GetUserById(c.GetInt("id"), false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	email := strings.TrimSpace(user.Email)
	if email == "" {
		common.ApiErrorMsg(c, "请先完善账户邮箱后再申请发票")
		return
	}
	minimumQuotaValue, err := common.QuotaFromFloatStrict(setting.MinimumAmount * common.QuotaPerUnit)
	if err != nil || minimumQuotaValue <= 0 {
		common.ApiErrorMsg(c, "发票金额配置无效，请联系管理员")
		return
	}
	minimumQuota := int64(minimumQuotaValue)
	application, err := model.CreateInvoiceApplication(user.Id, email, setting.Unit, minimumQuota, req.RequestIds)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrInvoiceSourcesClaimed):
			common.ApiErrorMsg(c, "所选充值记录已在其他申请中")
		case errors.Is(err, model.ErrInvoiceSourcesInvalid):
			common.ApiErrorMsg(c, "所选充值记录金额未达到开票门槛或已失效")
		default:
			common.ApiError(c, err)
		}
		return
	}
	common.ApiSuccess(c, application)
}

func AdminGetInvoiceApplications(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	applications, total, err := model.GetInvoiceApplications(0, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(applications)
	common.ApiSuccess(c, pageInfo)
}

func AdminGetInvoiceEmailTemplates(c *gin.Context) {
	templates := make([]service.EmailTemplate, 0, 2)
	for _, locale := range service.SupportedEmailTemplateLocales() {
		template, err := service.GetEmailTemplateForLocale(service.EmailTemplateEventInvoiceDelivered, locale)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		templates = append(templates, template)
	}
	common.ApiSuccess(c, templates)
}

func AdminUpdateInvoiceApplication(c *gin.Context) {
	id := c.Param("id")
	applicationId, err := strconv.Atoi(id)
	if err != nil || applicationId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的申请 ID"})
		return
	}
	var req invoiceUpdateRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "无效的参数")
		return
	}
	if len([]rune(strings.TrimSpace(req.AdminNote))) > 500 {
		common.ApiErrorMsg(c, "内部备注不能超过 500 个字符")
		return
	}
	if req.Status == model.InvoiceStatusCompleted {
		common.ApiErrorMsg(c, "请选择邮件模板并上传电子发票附件后完成申请")
		return
	}
	setting := operation_setting.GetInvoiceSetting()
	application, err := model.UpdateInvoiceApplication(
		applicationId,
		req.Status,
		strings.TrimSpace(req.AdminNote),
		setting.RejectionFreezeHours,
	)
	if err != nil {
		if errors.Is(err, model.ErrInvoiceStatusTerminal) {
			common.ApiErrorMsg(c, "该发票申请已进入终态，不可再更改")
			return
		}
		if errors.Is(err, model.ErrInvoiceEmailSending) {
			common.ApiErrorMsg(c, "该发票邮件正在发送，请稍后刷新")
			return
		}
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "invoice.application.update", map[string]interface{}{
		"application_id": applicationId,
		"status":         req.Status,
	})
	common.ApiSuccess(c, application)
}

func AdminCompleteInvoiceApplication(c *gin.Context) {
	applicationId, err := strconv.Atoi(c.Param("id"))
	if err != nil || applicationId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的申请 ID"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxInvoiceAttachmentBytes+1024*1024)
	if err := c.Request.ParseMultipartForm(maxInvoiceAttachmentBytes); err != nil {
		common.ApiErrorMsg(c, "电子发票附件不能超过 10 MB")
		return
	}
	locale := service.NormalizeEmailTemplateLocale(c.PostForm("locale"))
	adminNote := strings.TrimSpace(c.PostForm("admin_note"))
	if len([]rune(adminNote)) > 500 {
		common.ApiErrorMsg(c, "内部备注不能超过 500 个字符")
		return
	}
	fileHeader, err := c.FormFile("attachment")
	if err != nil || fileHeader.Size <= 0 || fileHeader.Size > maxInvoiceAttachmentBytes {
		common.ApiErrorMsg(c, "请上传不超过 10 MB 的电子发票附件")
		return
	}
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	allowedExtensions := map[string]struct{}{".pdf": {}, ".ofd": {}, ".xml": {}, ".png": {}, ".jpg": {}, ".jpeg": {}}
	if _, ok := allowedExtensions[ext]; !ok {
		common.ApiErrorMsg(c, "电子发票附件仅支持 PDF、OFD、XML、PNG、JPG")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	defer file.Close()
	attachment, err := io.ReadAll(io.LimitReader(file, maxInvoiceAttachmentBytes+1))
	if err != nil || len(attachment) == 0 || len(attachment) > maxInvoiceAttachmentBytes {
		common.ApiErrorMsg(c, "读取电子发票附件失败")
		return
	}
	application, err := model.BeginInvoiceEmailDelivery(applicationId)
	if err != nil {
		if errors.Is(err, model.ErrInvoiceStatusTerminal) {
			common.ApiErrorMsg(c, "该发票申请已进入终态，不可再更改")
			return
		}
		if errors.Is(err, model.ErrInvoiceEmailSending) {
			common.ApiErrorMsg(c, "该发票邮件正在发送，请稍后刷新")
			return
		}
		common.ApiError(c, err)
		return
	}
	emailSent := false
	defer func() {
		if !emailSent {
			_ = model.CancelInvoiceEmailDelivery(applicationId, application.EmailSendingAt)
		}
	}()
	user, err := model.GetUserById(application.UserId, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	now := time.Now()
	displayName := strings.TrimSpace(user.DisplayName)
	if displayName == "" {
		displayName = user.Username
	}
	rendered, err := service.RenderEmailTemplateForLocale(service.EmailTemplateEventInvoiceDelivered, locale, map[string]string{
		"username":       user.Username,
		"display_name":   displayName,
		"email":          application.Email,
		"application_id": strconv.Itoa(application.Id),
		"invoice_amount": fmt.Sprintf("%.2f", application.Amount),
		"invoice_unit":   application.Unit,
		"applied_at":     time.Unix(application.CreatedAt, 0).Format("2006-01-02 15:04:05"),
		"completed_at":   now.Format("2006-01-02 15:04:05"),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := sendInvoiceEmail(rendered.Subject, application.Email, rendered.Content, []common.EmailAttachment{{
		Filename:    fileHeader.Filename,
		ContentType: fileHeader.Header.Get("Content-Type"),
		Data:        attachment,
	}}); err != nil {
		common.ApiError(c, err)
		return
	}
	emailSent = true
	attachmentFilename := filepath.Base(strings.ReplaceAll(fileHeader.Filename, "\\", "/"))
	application, err = model.CompleteInvoiceEmailDelivery(applicationId, application.EmailSendingAt, adminNote, model.InvoiceAttachment{
		Filename:    attachmentFilename,
		ContentType: fileHeader.Header.Get("Content-Type"),
		Data:        attachment,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "invoice.application.complete", map[string]interface{}{
		"application_id":  applicationId,
		"locale":          locale,
		"attachment_name": attachmentFilename,
	})
	common.ApiSuccess(c, application)
}

func AdminWithdrawCompletedInvoiceApplication(c *gin.Context) {
	applicationId, err := strconv.Atoi(c.Param("id"))
	if err != nil || applicationId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的申请 ID"})
		return
	}
	application, err := model.WithdrawCompletedInvoiceApplication(applicationId)
	if err != nil {
		if errors.Is(err, model.ErrInvoiceNotWithdrawable) {
			common.ApiErrorMsg(c, "仅已完成的发票申请可以撤回")
			return
		}
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "invoice.application.withdraw", map[string]interface{}{
		"application_id": applicationId,
	})
	common.ApiSuccess(c, application)
}
