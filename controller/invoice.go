package controller

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime"
	"net/http"
	"net/mail"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"golang.org/x/image/webp"
)

const (
	maxInvoiceAttachmentBytes    = 10 * 1024 * 1024
	maxInvoicePaymentProofBytes  = 5 * 1024 * 1024
	maxInvoicePaymentProofs      = 3
	maxInvoiceApplyBytes         = maxInvoicePaymentProofBytes*maxInvoicePaymentProofs + 1024*1024
	maxInvoicePaymentProofPixels = 25_000_000
)

var sendInvoiceEmail = common.SendEmailWithAttachments

type invoiceCreditsRequest struct {
	RequestIds  []string `json:"request_ids"`
	InvoiceType string   `json:"invoice_type"`
	CompanyName string   `json:"company_name"`
	TaxNumber   string   `json:"tax_number"`
	Email       string   `json:"email"`
	Notes       string   `json:"notes"`
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
	user, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"email":                  strings.TrimSpace(user.Email),
		"allowed_sources":        setting.AllowedSources,
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
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxInvoiceApplyBytes)
	var req invoiceCreditsRequest
	var proofs []model.InvoicePaymentProof
	if c.ContentType() == "multipart/form-data" {
		if err := c.Request.ParseMultipartForm(maxInvoiceApplyBytes); err != nil {
			common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Invoice application upload exceeds the allowed size"))
			return
		}
		defer c.Request.MultipartForm.RemoveAll()
		if err := common.UnmarshalJsonStr(c.PostForm("request_ids"), &req.RequestIds); err != nil {
			common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Select valid invoice payment records"))
			return
		}
		req.InvoiceType = c.PostForm("invoice_type")
		req.CompanyName = c.PostForm("company_name")
		req.TaxNumber = c.PostForm("tax_number")
		req.Email = c.PostForm("email")
		req.Notes = c.PostForm("notes")
		files := c.Request.MultipartForm.File["payment_proofs"]
		if len(files) < 1 || len(files) > maxInvoicePaymentProofs {
			common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Upload 1 to 3 payment screenshots"))
			return
		}
		for _, header := range files {
			if header.Size <= 0 || header.Size > maxInvoicePaymentProofBytes {
				common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Each payment screenshot must be no larger than 5 MB"))
				return
			}
			file, err := header.Open()
			if err != nil {
				common.ApiError(c, err)
				return
			}
			data, readErr := io.ReadAll(io.LimitReader(file, maxInvoicePaymentProofBytes+1))
			_ = file.Close()
			if readErr != nil || len(data) == 0 || len(data) > maxInvoicePaymentProofBytes {
				common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Each payment screenshot must be no larger than 5 MB"))
				return
			}
			contentType, err := validateInvoicePaymentProof(data, header.Filename)
			if err != nil {
				common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage(err.Error()))
				return
			}
			filename := filepath.Base(strings.ReplaceAll(header.Filename, "\\", "/"))
			if len([]rune(filename)) > 255 || strings.ContainsAny(filename, "\r\n") {
				common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Invalid payment screenshot filename"))
				return
			}
			proofs = append(proofs, model.InvoicePaymentProof{Filename: filename, ContentType: contentType, Data: data})
		}
	} else if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Invalid invoice application parameters"))
		return
	}
	if len(req.RequestIds) == 0 || len(req.RequestIds) > 100 {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Select valid invoice payment records"))
		return
	}
	req.InvoiceType = strings.TrimSpace(req.InvoiceType)
	req.CompanyName = strings.TrimSpace(req.CompanyName)
	req.TaxNumber = strings.TrimSpace(req.TaxNumber)
	req.Email = strings.TrimSpace(req.Email)
	req.Notes = strings.TrimSpace(req.Notes)
	if req.InvoiceType != model.InvoiceTypeOrdinary && req.InvoiceType != model.InvoiceTypeSpecial {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Select a valid invoice type"))
		return
	}
	if len([]rune(req.CompanyName)) < 1 || len([]rune(req.CompanyName)) > 200 {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Company name must contain 1 to 200 characters"))
		return
	}
	if len(req.TaxNumber) < 1 || len(req.TaxNumber) > 50 || strings.IndexFunc(req.TaxNumber, func(r rune) bool {
		return !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z'))
	}) >= 0 {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Tax number must contain 1 to 50 letters or digits"))
		return
	}
	if len([]rune(req.Notes)) > 1000 {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Invoice notes must not exceed 1000 characters"))
		return
	}
	if len(proofs) == 0 {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Upload 1 to 3 payment screenshots"))
		return
	}
	user, err := model.GetUserById(c.GetInt("id"), false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if req.Email == "" {
		req.Email = strings.TrimSpace(user.Email)
	}
	address, err := mail.ParseAddress(req.Email)
	if err != nil || address.Address != req.Email || len(req.Email) > 254 || strings.ContainsAny(req.Email, "\r\n") {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Enter a valid invoice delivery email"))
		return
	}
	minimumQuotaValue, err := common.QuotaFromFloatStrict(setting.MinimumAmount * common.QuotaPerUnit)
	if err != nil || minimumQuotaValue <= 0 {
		common.ApiErrorMsg(c, "发票金额配置无效，请联系管理员")
		return
	}
	application, err := model.CreateInvoiceApplication(user.Id, req.Email, setting.Unit, int64(minimumQuotaValue), req.RequestIds, model.InvoiceApplicationDetails{
		InvoiceType: req.InvoiceType, CompanyName: req.CompanyName, TaxNumber: req.TaxNumber, Notes: req.Notes, PaymentProofs: proofs,
	})
	if err != nil {
		switch {
		case errors.Is(err, model.ErrInvoiceSourcesClaimed):
			common.ApiErrorMsg(c, "所选充值记录已在其他申请中")
		case errors.Is(err, model.ErrInvoiceSourcesInvalid):
			common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Selected payment records are disabled, unavailable, or below the invoice minimum"))
		default:
			common.ApiError(c, err)
		}
		return
	}
	common.ApiSuccess(c, application)
}

func validateInvoicePaymentProof(data []byte, filename string) (string, error) {
	contentType := http.DetectContentType(data)
	ext := strings.ToLower(filepath.Ext(filename))
	var config image.Config
	var err error
	switch contentType {
	case "image/png":
		if ext != ".png" {
			return "", errors.New("Payment screenshots must be valid PNG, JPG, or WebP images")
		}
		config, err = png.DecodeConfig(bytes.NewReader(data))
	case "image/jpeg":
		if ext != ".jpg" && ext != ".jpeg" {
			return "", errors.New("Payment screenshots must be valid PNG, JPG, or WebP images")
		}
		config, err = jpeg.DecodeConfig(bytes.NewReader(data))
	case "image/webp":
		if ext != ".webp" {
			return "", errors.New("Payment screenshots must be valid PNG, JPG, or WebP images")
		}
		config, err = webp.DecodeConfig(bytes.NewReader(data))
	default:
		return "", errors.New("Payment screenshots must be valid PNG, JPG, or WebP images")
	}
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return "", errors.New("Payment screenshots must be valid PNG, JPG, or WebP images")
	}
	if int64(config.Width)*int64(config.Height) > maxInvoicePaymentProofPixels {
		return "", errors.New("Payment screenshot dimensions exceed the allowed size")
	}
	switch contentType {
	case "image/png":
		_, err = png.Decode(bytes.NewReader(data))
	case "image/jpeg":
		_, err = jpeg.Decode(bytes.NewReader(data))
	case "image/webp":
		_, err = webp.Decode(bytes.NewReader(data))
	}
	if err != nil {
		return "", errors.New("Payment screenshots must be valid PNG, JPG, or WebP images")
	}
	return contentType, nil
}

func DownloadInvoicePaymentProof(c *gin.Context) {
	applicationId, applicationErr := strconv.Atoi(c.Param("id"))
	proofId, proofErr := strconv.Atoi(c.Param("proof_id"))
	if applicationErr != nil || proofErr != nil || applicationId <= 0 || proofId <= 0 {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Invalid invoice payment screenshot ID"))
		return
	}
	allowAdmin := c.GetInt("role") >= common.RoleAdminUser
	if c.GetBool("use_access_token") && !c.GetBool("access_token_legacy") {
		value, _ := c.Get("access_token_scopes")
		scopes, _ := value.([]string)
		allowAdmin = allowAdmin && service.AccessTokenScopeGranted(scopes, "billing:read")
		if !allowAdmin && !service.AccessTokenScopeGranted(scopes, "wallet:read") {
			common.ApiErrorStatus(c, http.StatusForbidden, common.NewMessage("Unauthorized, insufficient privileges"))
			return
		}
	}
	proof, err := model.GetInvoicePaymentProof(applicationId, proofId, c.GetInt("id"), allowAdmin)
	if err != nil {
		common.ApiErrorStatus(c, http.StatusNotFound, common.NewMessage("Invoice payment screenshot was not found"))
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": proof.Filename}))
	c.Data(http.StatusOK, proof.ContentType, proof.Data)
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
