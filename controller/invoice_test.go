package controller

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, db.AutoMigrate(&model.InvoiceApplication{}, &model.InvoiceItem{}, &model.InvoiceAttachment{}))
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
	require.NoError(t, db.AutoMigrate(&model.InvoiceApplication{}, &model.InvoiceItem{}, &model.InvoiceAttachment{}))
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
	require.NoError(t, db.AutoMigrate(&model.InvoiceApplication{}, &model.InvoiceItem{}, &model.InvoiceAttachment{}))
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
	require.NoError(t, db.AutoMigrate(&model.InvoiceApplication{}, &model.InvoiceItem{}, &model.InvoiceAttachment{}))
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
	require.NoError(t, db.AutoMigrate(&model.InvoiceApplication{}, &model.InvoiceItem{}, &model.InvoiceAttachment{}))
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
