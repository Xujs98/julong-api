package controller

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func performCompleteInvoiceRequest(t *testing.T, applicationId int) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("locale", "zh"))
	require.NoError(t, writer.WriteField("admin_note", "sent by test"))
	file, err := writer.CreateFormFile("attachment", "invoice.pdf")
	require.NoError(t, err)
	_, err = file.Write([]byte("invoice attachment"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/user/invoice/applications/1/complete", &body)
	context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	context.Params = gin.Params{{Key: "id", Value: strconv.Itoa(applicationId)}}
	context.Set("id", 99)
	context.Set("role", common.RoleRootUser)
	context.Set("username", "invoice-root")
	AdminCompleteInvoiceApplication(context)
	return recorder
}

func TestAdminUpdateInvoiceApplicationAcceptsFiveHundredCharacterNote(t *testing.T) {
	db := setupManageUserTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.InvoiceApplication{}, &model.InvoiceItem{}, &model.InvoiceAttachment{}, &model.InvoicePaymentProof{}))
	application := model.InvoiceApplication{
		UserId: 1,
		Email:  "invoice@example.com",
		Amount: 25,
		Unit:   "USD",
		Status: model.InvoiceStatusPending,
	}
	require.NoError(t, db.Create(&application).Error)

	note := strings.Repeat("a", 500)
	body, err := common.Marshal(map[string]string{
		"status":     model.InvoiceStatusProcessing,
		"admin_note": note,
	})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/user/invoice/applications/1", strings.NewReader(string(body)))
	context.Params = gin.Params{{Key: "id", Value: "1"}}
	context.Set("id", 99)
	context.Set("role", common.RoleRootUser)
	context.Set("username", "invoice-root")

	AdminUpdateInvoiceApplication(context)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	var persisted model.InvoiceApplication
	require.NoError(t, db.First(&persisted, application.Id).Error)
	assert.Equal(t, note, persisted.AdminNote)
}

func TestAdminUpdateInvoiceApplicationRejectsNoteOverFiveHundredCharacters(t *testing.T) {
	body, err := common.Marshal(map[string]string{
		"status":     model.InvoiceStatusProcessing,
		"admin_note": strings.Repeat("界", 501),
	})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/user/invoice/applications/1", strings.NewReader(string(body)))
	context.Params = gin.Params{{Key: "id", Value: "1"}}

	AdminUpdateInvoiceApplication(context)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"success":false,"message":"内部备注不能超过 500 个字符"}`, recorder.Body.String())
}

func TestAdminCompleteInvoiceOnlyCompletesAfterEmailSucceeds(t *testing.T) {
	db := setupManageUserTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.InvoiceApplication{}, &model.InvoiceItem{}, &model.InvoiceAttachment{}, &model.InvoicePaymentProof{}))
	user := model.User{
		Username: "invoice-recipient", Password: "password", Email: "recipient@example.com",
		Status: common.UserStatusEnabled, AffCode: "invoice-recipient-aff",
	}
	require.NoError(t, db.Create(&user).Error)
	application := model.InvoiceApplication{
		UserId: user.Id, Email: user.Email, AmountQuota: 500, AmountValue: "5", Unit: "USD", Status: model.InvoiceStatusPending,
	}
	require.NoError(t, db.Create(&application).Error)

	previousSender := sendInvoiceEmail
	t.Cleanup(func() { sendInvoiceEmail = previousSender })
	sendInvoiceEmail = func(string, string, string, []common.EmailAttachment) error {
		return errors.New("SMTP failure")
	}
	failed := performCompleteInvoiceRequest(t, application.Id)
	assert.Contains(t, failed.Body.String(), `"success":false`)
	var persisted model.InvoiceApplication
	require.NoError(t, db.First(&persisted, application.Id).Error)
	assert.Equal(t, model.InvoiceStatusPending, persisted.Status)
	assert.Zero(t, persisted.EmailSendingAt)

	sendInvoiceEmail = func(subject, receiver, content string, attachments []common.EmailAttachment) error {
		assert.NotEmpty(t, subject)
		assert.Equal(t, user.Email, receiver)
		assert.Contains(t, content, "#"+strconv.Itoa(application.Id))
		require.Len(t, attachments, 1)
		assert.Equal(t, "invoice.pdf", attachments[0].Filename)
		return nil
	}
	succeeded := performCompleteInvoiceRequest(t, application.Id)
	assert.Contains(t, succeeded.Body.String(), `"success":true`)
	require.NoError(t, db.First(&persisted, application.Id).Error)
	assert.Equal(t, model.InvoiceStatusCompleted, persisted.Status)
	assert.Equal(t, "sent by test", persisted.AdminNote)
	assert.Positive(t, persisted.CompletedAt)
	var attachment model.InvoiceAttachment
	require.NoError(t, db.Where("application_id = ?", application.Id).First(&attachment).Error)
	assert.Equal(t, "invoice.pdf", attachment.Filename)
	assert.Equal(t, []byte("invoice attachment"), attachment.Data)
}

func TestAdminWithdrawCompletedInvoiceReturnsApplicationToPending(t *testing.T) {
	db := setupManageUserTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.InvoiceApplication{}, &model.InvoiceItem{}, &model.InvoiceAttachment{}, &model.InvoicePaymentProof{}))
	application := model.InvoiceApplication{
		UserId: 7, Email: "withdraw@example.com", AmountQuota: 500, AmountValue: "5", Unit: "USD",
		Status: model.InvoiceStatusCompleted, CompletedAt: 1_800_000_200,
	}
	require.NoError(t, db.Create(&application).Error)
	attachment := model.InvoiceAttachment{ApplicationId: application.Id, Filename: "wrong.pdf", ContentType: "application/pdf", Data: []byte("wrong"), Size: 5}
	require.NoError(t, db.Create(&attachment).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/user/invoice/applications/1/withdraw", nil)
	context.Params = gin.Params{{Key: "id", Value: strconv.Itoa(application.Id)}}
	context.Set("id", 99)
	context.Set("role", common.RoleRootUser)
	context.Set("username", "invoice-root")
	AdminWithdrawCompletedInvoiceApplication(context)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	assert.Contains(t, recorder.Body.String(), `"status":"pending"`)
	assert.Contains(t, recorder.Body.String(), `"has_attachment":false`)
	require.NoError(t, db.First(&application, application.Id).Error)
	assert.Equal(t, model.InvoiceStatusPending, application.Status)
	assert.Zero(t, application.CompletedAt)
	var attachmentCount int64
	require.NoError(t, db.Model(&model.InvoiceAttachment{}).Where("application_id = ?", application.Id).Count(&attachmentCount).Error)
	assert.Zero(t, attachmentCount)
}

func TestCancelInvoiceApplicationOnlyAllowsPendingOwner(t *testing.T) {
	db := setupManageUserTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.InvoiceApplication{}, &model.InvoiceItem{}, &model.InvoiceAttachment{}, &model.InvoicePaymentProof{}))
	application := model.InvoiceApplication{
		UserId: 7, Email: "owner@example.com", AmountQuota: 500, AmountValue: "5", Unit: "USD", Status: model.InvoiceStatusPending, AdminNote: "private note",
	}
	require.NoError(t, db.Create(&application).Error)
	item := model.InvoiceItem{ApplicationId: application.Id, UserId: application.UserId, SourceRequestId: "cancel-controller-credit", Source: model.QuotaIncreaseSourceOnlineRecharge, Quota: 500}
	require.NoError(t, db.Create(&item).Error)

	wrongUser := httptest.NewRecorder()
	wrongContext, _ := gin.CreateTestContext(wrongUser)
	wrongContext.Request = httptest.NewRequest(http.MethodPost, "/api/user/invoice/applications/1/cancel", nil)
	wrongContext.Params = gin.Params{{Key: "id", Value: strconv.Itoa(application.Id)}}
	wrongContext.Set("id", application.UserId+1)
	CancelInvoiceApplication(wrongContext)
	assert.Contains(t, wrongUser.Body.String(), `"success":false`)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/user/invoice/applications/1/cancel", nil)
	context.Params = gin.Params{{Key: "id", Value: strconv.Itoa(application.Id)}}
	context.Set("id", application.UserId)
	CancelInvoiceApplication(context)
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	assert.Contains(t, recorder.Body.String(), `"status":"cancelled"`)
	assert.NotContains(t, recorder.Body.String(), "private note")

	require.NoError(t, db.First(&application, application.Id).Error)
	assert.Equal(t, model.InvoiceStatusCancelled, application.Status)
	require.NoError(t, db.First(&item, item.Id).Error)
	assert.Positive(t, item.ReleasedAt)
	assert.Zero(t, item.FrozenUntil)
}

func TestDownloadInvoiceAttachmentEnforcesOwnershipAndReturnsOriginalFile(t *testing.T) {
	db := setupManageUserTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.InvoiceApplication{}, &model.InvoiceItem{}, &model.InvoiceAttachment{}, &model.InvoicePaymentProof{}))
	application := model.InvoiceApplication{
		UserId: 8, Email: "download@example.com", AmountQuota: 500, AmountValue: "5", Unit: "USD", Status: model.InvoiceStatusCompleted,
	}
	require.NoError(t, db.Create(&application).Error)
	attachment := model.InvoiceAttachment{ApplicationId: application.Id, Filename: "电子发票.pdf", ContentType: "application/pdf", Data: []byte("stored invoice bytes"), Size: 20}
	require.NoError(t, db.Create(&attachment).Error)

	request := func(userId, role int) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = httptest.NewRequest(http.MethodGet, "/api/user/invoice/applications/1/attachment", nil)
		context.Params = gin.Params{{Key: "id", Value: strconv.Itoa(application.Id)}}
		context.Set("id", userId)
		context.Set("role", role)
		DownloadInvoiceAttachment(context)
		return recorder
	}

	owner := request(application.UserId, common.RoleCommonUser)
	assert.Equal(t, http.StatusOK, owner.Code)
	assert.Equal(t, "application/pdf", owner.Header().Get("Content-Type"))
	assert.Equal(t, "private, no-store", owner.Header().Get("Cache-Control"))
	assert.Contains(t, owner.Header().Get("Content-Disposition"), "attachment")
	assert.Equal(t, "stored invoice bytes", owner.Body.String())

	otherUser := request(application.UserId+1, common.RoleCommonUser)
	assert.Contains(t, otherUser.Body.String(), `"success":false`)
	admin := request(application.UserId+1, common.RoleAdminUser)
	assert.Equal(t, "stored invoice bytes", admin.Body.String())
}

func TestApplyInvoiceRejectsLegacyRequestWithoutInvoiceMaterials(t *testing.T) {
	db := setupManageUserTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.InvoiceApplication{}, &model.InvoiceItem{}, &model.InvoiceAttachment{}, &model.InvoicePaymentProof{}))
	previous := operation_setting.GetInvoiceSetting()
	setting := config.GlobalConfig.Get("invoice_setting").(*operation_setting.InvoiceSetting)
	setting.Enabled = true
	setting.MinimumAmount = 1
	t.Cleanup(func() { *setting = previous })
	user := model.User{Username: "invoice-material-user", Password: "password", Email: "registered@example.com", Status: common.UserStatusEnabled, AffCode: "invoice-material-aff"}
	require.NoError(t, db.Create(&user).Error)
	log := model.Log{UserId: user.Id, Type: model.LogTypeQuotaIncrease, Quota: int(common.QuotaPerUnit), RequestId: "invoice-material-credit", Other: common.MapToJsonStr(map[string]any{"source": model.QuotaIncreaseSourceOnlineRecharge})}
	require.NoError(t, db.Create(&log).Error)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/user/invoice/apply", strings.NewReader(`{"request_ids":["invoice-material-credit"]}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("id", user.Id)
	ApplyInvoice(context)
	assert.Contains(t, recorder.Body.String(), `"success":false`)
	var count int64
	require.NoError(t, db.Model(&model.InvoiceApplication{}).Count(&count).Error)
	assert.Zero(t, count)
}

func setupInvoiceControllerTest(t *testing.T) (*gorm.DB, model.User) {
	t.Helper()
	db := setupManageUserTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.InvoiceApplication{}, &model.InvoiceItem{}, &model.InvoiceAttachment{}, &model.InvoicePaymentProof{}))
	previous := operation_setting.GetInvoiceSetting()
	setting := config.GlobalConfig.Get("invoice_setting").(*operation_setting.InvoiceSetting)
	setting.Enabled = true
	setting.MinimumAmount = 1
	setting.AllowedSources = []string{model.QuotaIncreaseSourceOnlineRecharge, model.QuotaIncreaseSourceRedemption, model.QuotaIncreaseSourceAdminAdjustment}
	t.Cleanup(func() { *setting = previous })
	user := model.User{Username: "invoice-apply-user", Password: "password", Email: "registered@example.com", Status: common.UserStatusEnabled, AffCode: "invoice-apply-aff"}
	require.NoError(t, db.Create(&user).Error)
	log := model.Log{UserId: user.Id, Type: model.LogTypeQuotaIncrease, Quota: int(common.QuotaPerUnit), RequestId: "invoice-apply-credit", Other: common.MapToJsonStr(map[string]any{"source": model.QuotaIncreaseSourceOnlineRecharge})}
	require.NoError(t, db.Create(&log).Error)
	return db, user
}

func performApplyInvoiceRequest(t *testing.T, userId int, fields map[string]string, filenames []string, files [][]byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range fields {
		require.NoError(t, writer.WriteField(name, value))
	}
	for i, filename := range filenames {
		file, err := writer.CreateFormFile("payment_proofs", filename)
		require.NoError(t, err)
		_, err = file.Write(files[i])
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/user/invoice/apply", &body)
	context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	context.Set("id", userId)
	ApplyInvoice(context)
	return recorder
}

func TestApplyInvoicePersistsMaterialsAndAllowsSeparateDeliveryEmail(t *testing.T) {
	for _, email := range []string{"", "delivery@example.com"} {
		t.Run("delivery-email-"+email, func(t *testing.T) {
			db, user := setupInvoiceControllerTest(t)
			var pngImage, jpegImage bytes.Buffer
			require.NoError(t, png.Encode(&pngImage, image.NewRGBA(image.Rect(0, 0, 1, 1))))
			require.NoError(t, jpeg.Encode(&jpegImage, image.NewRGBA(image.Rect(0, 0, 1, 1)), nil))
			fields := map[string]string{
				"request_ids": `["invoice-apply-credit"]`, "invoice_type": "special", "company_name": "Example Company",
				"tax_number": "913100000000000000", "email": email, "notes": "Leave this invoice note",
			}
			response := performApplyInvoiceRequest(t, user.Id, fields, []string{"payment.png", "receipt.jpg"}, [][]byte{pngImage.Bytes(), jpegImage.Bytes()})
			assert.Equal(t, http.StatusOK, response.Code, response.Body.String())
			assert.Contains(t, response.Body.String(), `"success":true`)
			var applications []model.InvoiceApplication
			require.NoError(t, db.Find(&applications).Error)
			require.Len(t, applications, 1)
			application := applications[0]
			assert.Equal(t, model.InvoiceTypeSpecial, application.InvoiceType)
			assert.Equal(t, fields["company_name"], application.CompanyName)
			assert.Equal(t, fields["tax_number"], application.TaxNumber)
			assert.Equal(t, fields["notes"], application.Notes)
			wantEmail := email
			if wantEmail == "" {
				wantEmail = user.Email
			}
			assert.Equal(t, wantEmail, application.Email)
			require.NoError(t, db.First(&user, user.Id).Error)
			assert.Equal(t, "registered@example.com", user.Email)
			var proofs []model.InvoicePaymentProof
			require.NoError(t, db.Where("application_id = ?", application.Id).Order("id").Find(&proofs).Error)
			require.Len(t, proofs, 2)
			assert.Equal(t, pngImage.Bytes(), proofs[0].Data)
			assert.Equal(t, "image/png", proofs[0].ContentType)
			assert.Equal(t, jpegImage.Bytes(), proofs[1].Data)
			assert.Equal(t, "image/jpeg", proofs[1].ContentType)
			assert.NotContains(t, response.Body.String(), `"Data"`)
			assert.NotContains(t, response.Body.String(), `"data":"`)
			infoRecorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(infoRecorder)
			context.Request = httptest.NewRequest(http.MethodGet, "/api/user/invoice", nil)
			context.Set("id", user.Id)
			GetInvoiceInfo(context)
			assert.Equal(t, http.StatusOK, infoRecorder.Code)
			assert.Contains(t, infoRecorder.Body.String(), `"email":"registered@example.com"`)
			assert.Contains(t, infoRecorder.Body.String(), `"allowed_sources":["online_recharge","redemption","admin_adjustment"]`)
			assert.Contains(t, infoRecorder.Body.String(), `"payment_proofs":[`)
		})
	}
}

func TestApplyInvoiceRejectsInvalidMaterialsWithoutClaimingCredits(t *testing.T) {
	var pngImage bytes.Buffer
	require.NoError(t, png.Encode(&pngImage, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	valid := pngImage.Bytes()
	for _, tc := range []struct {
		name      string
		field     string
		value     string
		filenames []string
		files     [][]byte
	}{
		{name: "missing screenshots", filenames: []string{}, files: [][]byte{}},
		{name: "too many screenshots", filenames: []string{"1.png", "2.png", "3.png", "4.png"}, files: [][]byte{valid, valid, valid, valid}},
		{name: "oversized screenshot", filenames: []string{"1.png"}, files: [][]byte{make([]byte, maxInvoicePaymentProofBytes+1)}},
		{name: "spoofed svg", filenames: []string{"1.png"}, files: [][]byte{[]byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)}},
		{name: "spoofed extension", filenames: []string{"1.jpg"}, files: [][]byte{valid}},
		{name: "truncated image", filenames: []string{"1.png"}, files: [][]byte{valid[:35]}},
		{name: "missing company", field: "company_name", value: ""},
		{name: "long company", field: "company_name", value: strings.Repeat("界", 201)},
		{name: "missing tax number", field: "tax_number", value: ""},
		{name: "long tax number", field: "tax_number", value: strings.Repeat("1", 51)},
		{name: "invalid invoice type", field: "invoice_type", value: "other"},
		{name: "invalid email", field: "email", value: "Name <delivery@example.com>"},
		{name: "long notes", field: "notes", value: strings.Repeat("界", 1001)},
		{name: "overall request size", field: "notes", value: strings.Repeat("a", maxInvoiceApplyBytes)},
		{name: "disabled source", field: "allowed_sources", value: "disabled"},
		{name: "empty allowed sources", field: "allowed_sources", value: "empty"},
		{name: "duplicate payment record", field: "request_ids", value: `["invoice-apply-credit","invoice-apply-credit"]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, user := setupInvoiceControllerTest(t)
			fields := map[string]string{"request_ids": `["invoice-apply-credit"]`, "invoice_type": "ordinary", "company_name": "Example Company", "tax_number": "913100000000000000", "email": "delivery@example.com", "notes": ""}
			if tc.field == "allowed_sources" {
				setting := config.GlobalConfig.Get("invoice_setting").(*operation_setting.InvoiceSetting)
				if tc.value == "empty" {
					setting.AllowedSources = []string{}
				} else {
					setting.AllowedSources = []string{model.QuotaIncreaseSourceRedemption}
				}
			} else if tc.field != "" {
				fields[tc.field] = tc.value
			}
			filenames, files := tc.filenames, tc.files
			if filenames == nil {
				filenames, files = []string{"payment.png"}, [][]byte{valid}
			}
			response := performApplyInvoiceRequest(t, user.Id, fields, filenames, files)
			assert.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
			assert.Contains(t, response.Body.String(), `"success":false`)
			for _, table := range []any{&model.InvoiceApplication{}, &model.InvoiceItem{}, &model.InvoicePaymentProof{}} {
				var count int64
				require.NoError(t, db.Model(table).Count(&count).Error)
				assert.Zero(t, count)
			}
		})
	}
}

func TestApplyInvoiceJSONCannotSupplyPaymentScreenshotBytes(t *testing.T) {
	db, user := setupInvoiceControllerTest(t)
	var pngImage bytes.Buffer
	require.NoError(t, png.Encode(&pngImage, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	body, err := common.Marshal(map[string]any{
		"request_ids": []string{"invoice-apply-credit"}, "invoice_type": "ordinary", "company_name": "Example Company",
		"tax_number": "913100000000000000", "email": user.Email,
		"payment_proofs": []map[string]any{{"filename": "payment.png", "data": pngImage.Bytes()}},
	})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/user/invoice/apply", bytes.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("id", user.Id)
	ApplyInvoice(context)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `Upload 1 to 3 payment screenshots`)
	var count int64
	require.NoError(t, db.Model(&model.InvoiceApplication{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestInvoicePaymentProofDownloadScopesOwnerAndApplication(t *testing.T) {
	db, user := setupInvoiceControllerTest(t)
	applications := []model.InvoiceApplication{
		{UserId: user.Id, Email: user.Email, Status: model.InvoiceStatusPending},
		{UserId: user.Id, Email: user.Email, Status: model.InvoiceStatusPending},
	}
	require.NoError(t, db.Create(&applications).Error)
	proof := model.InvoicePaymentProof{ApplicationId: applications[0].Id, Filename: "payment.png", ContentType: "image/png", Data: []byte("private screenshot"), Size: 18}
	require.NoError(t, db.Create(&proof).Error)
	for _, tc := range []struct {
		name           string
		userId         int
		role           int
		applicationId  int
		wantStatus     int
		useAccessToken bool
		legacy         bool
		scopes         []string
	}{
		{name: "owner", userId: user.Id, role: common.RoleCommonUser, applicationId: applications[0].Id, wantStatus: http.StatusOK},
		{name: "other user", userId: user.Id + 1, role: common.RoleCommonUser, applicationId: applications[0].Id, wantStatus: http.StatusNotFound},
		{name: "admin", userId: user.Id + 1, role: common.RoleAdminUser, applicationId: applications[0].Id, wantStatus: http.StatusOK},
		{name: "wrong application", userId: user.Id, role: common.RoleCommonUser, applicationId: applications[1].Id, wantStatus: http.StatusNotFound},
		{name: "wrong application admin", userId: user.Id + 1, role: common.RoleAdminUser, applicationId: applications[1].Id, wantStatus: http.StatusNotFound},
		{name: "admin token personal wallet", userId: user.Id + 1, role: common.RoleAdminUser, applicationId: applications[0].Id, wantStatus: http.StatusNotFound, useAccessToken: true, scopes: []string{"wallet:read"}},
		{name: "admin token missing scope", userId: user.Id + 1, role: common.RoleAdminUser, applicationId: applications[0].Id, wantStatus: http.StatusForbidden, useAccessToken: true},
		{name: "root token personal wallet", userId: user.Id + 1, role: common.RoleRootUser, applicationId: applications[0].Id, wantStatus: http.StatusNotFound, useAccessToken: true, scopes: []string{"wallet:read"}},
		{name: "admin token invoice read", userId: user.Id + 1, role: common.RoleAdminUser, applicationId: applications[0].Id, wantStatus: http.StatusOK, useAccessToken: true, scopes: []string{"billing:read"}},
		{name: "owner token personal wallet", userId: user.Id, role: common.RoleAdminUser, applicationId: applications[0].Id, wantStatus: http.StatusOK, useAccessToken: true, scopes: []string{"wallet:read"}},
		{name: "legacy administrator", userId: user.Id + 1, role: common.RoleAdminUser, applicationId: applications[0].Id, wantStatus: http.StatusOK, useAccessToken: true, legacy: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(http.MethodGet, "/api/user/invoice/applications/1/payment-proofs/1", nil)
			context.Params = gin.Params{{Key: "id", Value: strconv.Itoa(tc.applicationId)}, {Key: "proof_id", Value: strconv.Itoa(proof.Id)}}
			context.Set("id", tc.userId)
			context.Set("role", tc.role)
			context.Set("use_access_token", tc.useAccessToken)
			context.Set("access_token_legacy", tc.legacy)
			context.Set("access_token_scopes", tc.scopes)
			DownloadInvoicePaymentProof(context)
			assert.Equal(t, tc.wantStatus, recorder.Code)
			if tc.wantStatus == http.StatusOK {
				assert.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
				assert.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
				assert.Equal(t, "image/png", recorder.Header().Get("Content-Type"))
				assert.Equal(t, string(proof.Data), recorder.Body.String())
			} else {
				assert.NotContains(t, recorder.Body.String(), string(proof.Data))
			}
		})
	}
}

func TestInvoicePaymentProofDownloadThroughScopedAuthentication(t *testing.T) {
	db, owner := setupInvoiceControllerTest(t)
	require.NoError(t, db.AutoMigrate(&model.UserAccessToken{}, &model.UserPresence{}, &model.AuditLog{}))
	admin := model.User{Username: "invoice-proof-admin", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, AffCode: "invoice-proof-admin-aff"}
	require.NoError(t, db.Create(&admin).Error)
	application := model.InvoiceApplication{UserId: owner.Id, Email: owner.Email, Status: model.InvoiceStatusPending}
	require.NoError(t, db.Create(&application).Error)
	proof := model.InvoicePaymentProof{ApplicationId: application.Id, Filename: "payment.png", ContentType: "image/png", Data: []byte("private payment proof"), Size: 21}
	require.NoError(t, db.Create(&proof).Error)
	router := gin.New()
	router.GET("/api/user/invoice/applications/:id/payment-proofs/:proof_id", middleware.UserAuth(), DownloadInvoicePaymentProof)
	for i, tc := range []struct {
		name       string
		userId     int
		scopes     []string
		wantStatus int
	}{
		{name: "owner wallet grant", userId: owner.Id, scopes: []string{"wallet:read"}, wantStatus: http.StatusOK},
		{name: "admin wallet grant cannot read another owner", userId: admin.Id, scopes: []string{"wallet:read"}, wantStatus: http.StatusNotFound},
		{name: "admin billing grant", userId: admin.Id, scopes: []string{"billing:read"}, wantStatus: http.StatusOK},
		{name: "owner billing grant is not a wallet grant", userId: owner.Id, scopes: []string{"billing:read"}, wantStatus: http.StatusForbidden},
		{name: "owner unrelated grant", userId: owner.Id, scopes: []string{"profile:read"}, wantStatus: http.StatusForbidden},
		{name: "admin unrelated grant", userId: admin.Id, scopes: []string{"profile:read"}, wantStatus: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := model.AccessTokenPrefix + "invoice-proof-synthetic-" + strconv.Itoa(i)
			token := model.UserAccessToken{UserId: tc.userId, TokenHash: model.AccessTokenFingerprint(raw), TokenHint: "invoice-test"}
			require.NoError(t, token.SetScopes(tc.scopes))
			require.NoError(t, db.Create(&token).Error)
			request := httptest.NewRequest(http.MethodGet, "/api/user/invoice/applications/"+strconv.Itoa(application.Id)+"/payment-proofs/"+strconv.Itoa(proof.Id), nil)
			request.Header.Set("Authorization", "Bearer "+raw)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			assert.Equal(t, tc.wantStatus, recorder.Code)
			if tc.wantStatus == http.StatusOK {
				assert.Equal(t, string(proof.Data), recorder.Body.String())
			} else {
				assert.NotContains(t, recorder.Body.String(), string(proof.Data))
			}
		})
	}
}
