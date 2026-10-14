package model

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestInvoiceApplicationClaimsEligibleCredits(t *testing.T) {
	setupInvoiceTest(t)
	user := User{
		Username: "invoice-user",
		Password: "password",
		Email:    "invoice@example.com",
		Status:   common.UserStatusEnabled,
		AffCode:  "invoice-aff-code",
	}
	require.NoError(t, DB.Create(&user).Error)

	logs := []Log{
		{UserId: user.Id, Type: LogTypeQuotaIncrease, CreatedAt: 100, Quota: 100, RequestId: "invoice-online", Other: common.MapToJsonStr(map[string]any{"source": QuotaIncreaseSourceOnlineRecharge})},
		{UserId: user.Id, Type: LogTypeQuotaIncrease, CreatedAt: 200, Quota: 200, RequestId: "invoice-redemption", Other: common.MapToJsonStr(map[string]any{"source": QuotaIncreaseSourceRedemption})},
		{UserId: user.Id, Type: LogTypeQuotaIncrease, CreatedAt: 300, Quota: 400, RequestId: "invoice-admin", Other: common.MapToJsonStr(map[string]any{"source": QuotaIncreaseSourceAdminAdjustment})},
		{UserId: user.Id, Type: LogTypeQuotaIncrease, CreatedAt: 400, Quota: 900, RequestId: "invoice-checkin", Other: common.MapToJsonStr(map[string]any{"source": QuotaIncreaseSourceCheckin})},
	}
	require.NoError(t, LOG_DB.Create(&logs).Error)

	credits, total, err := GetInvoiceCredits(user.Id, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, credits, 3)
	assert.Equal(t, "invoice-admin", credits[0].RequestId)

	application, err := CreateInvoiceApplication(
		user.Id,
		user.Email,
		"USD",
		300,
		[]string{"invoice-online", "invoice-redemption"},
	)
	require.NoError(t, err)
	assert.EqualValues(t, 300, application.AmountQuota)
	assert.InDelta(t, invoiceAmountFromQuota(300), application.Amount, 1e-12)
	assert.Equal(t, InvoiceStatusPending, application.Status)
	var persisted InvoiceApplication
	require.NoError(t, DB.First(&persisted, application.Id).Error)
	assert.NotEmpty(t, persisted.AmountValue)
	assert.InDelta(t, invoiceAmountFromQuota(300), persisted.Amount, 1e-12)

	credits, total, err = GetInvoiceCredits(user.Id, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, credits, 1)
	assert.Equal(t, "invoice-admin", credits[0].RequestId)

	_, err = CreateInvoiceApplication(
		user.Id,
		user.Email,
		"USD",
		100,
		[]string{"invoice-online"},
	)
	assert.ErrorIs(t, err, ErrInvoiceSourcesClaimed)
}

func TestInvoiceMigrationIsIdempotent(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&InvoiceApplication{}, &InvoiceItem{}, &InvoiceAttachment{}, &InvoicePaymentProof{}))
	require.NoError(t, DB.AutoMigrate(&InvoiceApplication{}, &InvoiceItem{}, &InvoiceAttachment{}, &InvoicePaymentProof{}))
}

func TestInvoiceMigrationReleasesLegacyRejectedSources(t *testing.T) {
	setupInvoiceTest(t)
	require.NoError(t, DB.Exec(
		"INSERT INTO invoice_applications (user_id, email, status, email_sending_at) VALUES (?, ?, ?, NULL)",
		1, "legacy@example.com", InvoiceStatusRejected,
	).Error)
	application := InvoiceApplication{}
	require.NoError(t, DB.Where("email = ?", "legacy@example.com").First(&application).Error)
	item := InvoiceItem{ApplicationId: application.Id, UserId: 1, SourceRequestId: "legacy-rejected-source"}
	require.NoError(t, DB.Create(&item).Error)

	require.NoError(t, finalizeInvoiceItemMigration())
	var emailSendingAt int64
	require.NoError(t, DB.Model(&InvoiceApplication{}).Select("email_sending_at").Where("id = ?", application.Id).Scan(&emailSendingAt).Error)
	assert.Zero(t, emailSendingAt)
	require.NoError(t, DB.First(&item, item.Id).Error)
	assert.Positive(t, item.ReleasedAt)
	assert.Zero(t, item.FrozenUntil)

	firstReleasedAt := item.ReleasedAt
	require.NoError(t, finalizeInvoiceItemMigration())
	require.NoError(t, DB.First(&item, item.Id).Error)
	assert.Equal(t, firstReleasedAt, item.ReleasedAt)
}

func TestInvoiceApplicationAdminListAndStatus(t *testing.T) {
	setupInvoiceTest(t)
	user := User{
		Username: "invoice-admin-list-user",
		Password: "password",
		Email:    "admin-list@example.com",
		Status:   common.UserStatusEnabled,
		AffCode:  "invoice-admin-list-aff",
	}
	require.NoError(t, DB.Create(&user).Error)
	application := InvoiceApplication{
		UserId:      user.Id,
		Email:       user.Email,
		AmountQuota: 500,
		Unit:        "USD",
		Status:      InvoiceStatusPending,
	}
	require.NoError(t, DB.Create(&application).Error)
	item := InvoiceItem{
		ApplicationId: application.Id, UserId: user.Id, SourceRequestId: "invoice-admin-detail",
		Source: QuotaIncreaseSourceOnlineRecharge, Quota: 250, SourceCreatedAt: 123,
	}
	require.NoError(t, DB.Create(&item).Error)
	detailLog := Log{
		UserId: user.Id, Type: LogTypeQuotaIncrease, CreatedAt: 123, Quota: 250,
		RequestId: item.SourceRequestId, Content: "online payment detail",
	}
	require.NoError(t, LOG_DB.Create(&detailLog).Error)

	applications, total, err := GetInvoiceApplications(0, 0, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, applications, 1)
	assert.Equal(t, user.Username, applications[0].Username)
	require.Len(t, applications[0].Items, 1)
	assert.InDelta(t, invoiceAmountFromQuota(250), applications[0].Items[0].Amount, 1e-12)
	assert.Equal(t, "online payment detail", applications[0].Items[0].Content)

	updated, err := UpdateInvoiceApplication(application.Id, InvoiceStatusProcessing, "reviewing", 0)
	require.NoError(t, err)
	assert.Equal(t, InvoiceStatusProcessing, updated.Status)
	assert.Equal(t, "reviewing", updated.AdminNote)
	assert.Zero(t, updated.CompletedAt)
	userApplications, _, err := GetInvoiceApplications(user.Id, 0, 10)
	require.NoError(t, err)
	require.Len(t, userApplications, 1)
	assert.Empty(t, userApplications[0].AdminNote)
	require.Len(t, userApplications[0].Items, 1)
	assert.Empty(t, userApplications[0].Items[0].Content)
	_, err = UpdateInvoiceApplication(application.Id, InvoiceStatusCompleted, "", 0)
	assert.Error(t, err)

	_, err = UpdateInvoiceApplication(application.Id, "unknown", "", 0)
	assert.Error(t, err)
}

func TestRejectedInvoiceReleasesCreditsAfterFreeze(t *testing.T) {
	setupInvoiceTest(t)
	user := User{Username: "invoice-rejected-user", Password: "password", Email: "rejected@example.com", Status: common.UserStatusEnabled, AffCode: "invoice-rejected-aff"}
	require.NoError(t, DB.Create(&user).Error)
	log := Log{UserId: user.Id, Type: LogTypeQuotaIncrease, CreatedAt: 100, Quota: 500, RequestId: "invoice-rejected-credit", Other: common.MapToJsonStr(map[string]any{"source": QuotaIncreaseSourceOnlineRecharge})}
	require.NoError(t, LOG_DB.Create(&log).Error)
	application, err := CreateInvoiceApplication(user.Id, user.Email, "USD", 500, []string{log.RequestId})
	require.NoError(t, err)

	updated, err := UpdateInvoiceApplication(application.Id, InvoiceStatusRejected, "invalid title", 24)
	require.NoError(t, err)
	assert.Equal(t, InvoiceStatusRejected, updated.Status)
	require.Len(t, updated.Items, 1)
	assert.Positive(t, updated.Items[0].ReleasedAt)
	assert.Greater(t, updated.Items[0].FrozenUntil, common.GetTimestamp())

	credits, total, err := GetInvoiceCredits(user.Id, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, credits, 1)
	assert.Equal(t, updated.Items[0].FrozenUntil, credits[0].FrozenUntil)
	_, err = CreateInvoiceApplication(user.Id, user.Email, "USD", 500, []string{log.RequestId})
	assert.ErrorIs(t, err, ErrInvoiceSourcesClaimed)

	require.NoError(t, DB.Model(&InvoiceItem{}).Where("application_id = ?", application.Id).Update("frozen_until", common.GetTimestamp()-1).Error)
	second, err := CreateInvoiceApplication(user.Id, user.Email, "USD", 500, []string{log.RequestId})
	require.NoError(t, err)
	assert.NotEqual(t, application.Id, second.Id)
	_, err = UpdateInvoiceApplication(application.Id, InvoiceStatusProcessing, "", 0)
	assert.ErrorIs(t, err, ErrInvoiceStatusTerminal)
}

func TestNonReusableInvoiceKeepsCreditsClaimed(t *testing.T) {
	setupInvoiceTest(t)
	user := User{Username: "invoice-terminal-user", Password: "password", Email: "terminal@example.com", Status: common.UserStatusEnabled, AffCode: "invoice-terminal-aff"}
	require.NoError(t, DB.Create(&user).Error)
	log := Log{UserId: user.Id, Type: LogTypeQuotaIncrease, CreatedAt: 100, Quota: 500, RequestId: "invoice-terminal-credit", Other: common.MapToJsonStr(map[string]any{"source": QuotaIncreaseSourceRedemption})}
	require.NoError(t, LOG_DB.Create(&log).Error)
	application, err := CreateInvoiceApplication(user.Id, user.Email, "USD", 500, []string{log.RequestId})
	require.NoError(t, err)

	updated, err := UpdateInvoiceApplication(application.Id, InvoiceStatusNonReusable, "duplicate invoice", 24)
	require.NoError(t, err)
	assert.Equal(t, InvoiceStatusNonReusable, updated.Status)
	require.Len(t, updated.Items, 1)
	assert.Zero(t, updated.Items[0].ReleasedAt)
	credits, total, err := GetInvoiceCredits(user.Id, 0, 10)
	require.NoError(t, err)
	assert.Empty(t, credits)
	assert.Zero(t, total)
	_, err = UpdateInvoiceApplication(application.Id, InvoiceStatusRejected, "", 24)
	assert.ErrorIs(t, err, ErrInvoiceStatusTerminal)
}

func TestInvoiceEmailDeliveryLockAndCompletion(t *testing.T) {
	setupInvoiceTest(t)
	application := InvoiceApplication{
		UserId:      1,
		Email:       "invoice-lock@example.com",
		AmountQuota: 500,
		AmountValue: "5",
		Unit:        "USD",
		Status:      InvoiceStatusPending,
	}
	require.NoError(t, DB.Create(&application).Error)

	locked, err := BeginInvoiceEmailDelivery(application.Id)
	require.NoError(t, err)
	assert.Positive(t, locked.EmailSendingAt)
	_, err = BeginInvoiceEmailDelivery(application.Id)
	assert.ErrorIs(t, err, ErrInvoiceEmailSending)
	_, err = UpdateInvoiceApplication(application.Id, InvoiceStatusRejected, "", 24)
	assert.ErrorIs(t, err, ErrInvoiceEmailSending)

	require.NoError(t, CancelInvoiceEmailDelivery(application.Id, locked.EmailSendingAt))
	locked, err = BeginInvoiceEmailDelivery(application.Id)
	require.NoError(t, err)
	assert.Positive(t, locked.EmailSendingAt)

	completed, err := CompleteInvoiceEmailDelivery(application.Id, locked.EmailSendingAt, "sent", InvoiceAttachment{
		Filename:    "invoice.pdf",
		ContentType: "application/pdf",
		Data:        []byte("invoice attachment"),
	})
	require.NoError(t, err)
	assert.Equal(t, InvoiceStatusCompleted, completed.Status)
	assert.Equal(t, "sent", completed.AdminNote)
	assert.Positive(t, completed.CompletedAt)
	assert.Zero(t, completed.EmailSendingAt)
	assert.True(t, completed.HasAttachment)
	_, attachment, err := GetInvoiceAttachment(application.Id, application.UserId, false)
	require.NoError(t, err)
	assert.Equal(t, "invoice.pdf", attachment.Filename)
	assert.Equal(t, int64(len(attachment.Data)), attachment.Size)
	assert.Equal(t, []byte("invoice attachment"), attachment.Data)
	_, _, err = GetInvoiceAttachment(application.Id, application.UserId+1, false)
	assert.Error(t, err)
	_, _, err = GetInvoiceAttachment(application.Id, application.UserId+1, true)
	require.NoError(t, err)
	_, err = BeginInvoiceEmailDelivery(application.Id)
	assert.ErrorIs(t, err, ErrInvoiceStatusTerminal)
}

func TestCompletedInvoiceCanBeWithdrawnToPendingAndAttachmentIsDeleted(t *testing.T) {
	setupInvoiceTest(t)
	application := InvoiceApplication{
		UserId: 1, Email: "withdraw@example.com", AmountQuota: 500, AmountValue: "5",
		Unit: "USD", Status: InvoiceStatusCompleted, AdminNote: "keep review context",
		CompletedAt: common.GetTimestamp(), EmailSendingAt: common.GetTimestamp(),
	}
	require.NoError(t, DB.Create(&application).Error)
	item := InvoiceItem{ApplicationId: application.Id, UserId: application.UserId, SourceRequestId: "withdraw-credit", Quota: 500}
	require.NoError(t, DB.Create(&item).Error)
	attachment := InvoiceAttachment{ApplicationId: application.Id, Filename: "wrong.pdf", ContentType: "application/pdf", Data: []byte("wrong invoice"), Size: 13}
	require.NoError(t, DB.Create(&attachment).Error)

	withdrawn, err := WithdrawCompletedInvoiceApplication(application.Id)
	require.NoError(t, err)
	assert.Equal(t, InvoiceStatusPending, withdrawn.Status)
	assert.Zero(t, withdrawn.CompletedAt)
	assert.Zero(t, withdrawn.EmailSendingAt)
	assert.False(t, withdrawn.HasAttachment)
	assert.Equal(t, "keep review context", withdrawn.AdminNote)
	require.Len(t, withdrawn.Items, 1)
	assert.Zero(t, withdrawn.Items[0].ReleasedAt)

	var attachmentCount int64
	require.NoError(t, DB.Model(&InvoiceAttachment{}).Where("application_id = ?", application.Id).Count(&attachmentCount).Error)
	assert.Zero(t, attachmentCount)
	_, _, err = GetInvoiceAttachment(application.Id, application.UserId, false)
	assert.Error(t, err)
	_, err = WithdrawCompletedInvoiceApplication(application.Id)
	assert.ErrorIs(t, err, ErrInvoiceNotWithdrawable)
}

func TestPendingInvoiceCanBeCancelledByOwnerAndCreditsAreReleased(t *testing.T) {
	setupInvoiceTest(t)
	user := User{Username: "invoice-cancel-user", Password: "password", Email: "cancel@example.com", Status: common.UserStatusEnabled, AffCode: "invoice-cancel-aff"}
	require.NoError(t, DB.Create(&user).Error)
	log := Log{UserId: user.Id, Type: LogTypeQuotaIncrease, CreatedAt: 100, Quota: 500, RequestId: "invoice-cancel-credit", Other: common.MapToJsonStr(map[string]any{"source": QuotaIncreaseSourceOnlineRecharge})}
	require.NoError(t, LOG_DB.Create(&log).Error)
	application, err := CreateInvoiceApplication(user.Id, user.Email, "USD", 500, []string{log.RequestId})
	require.NoError(t, err)

	_, err = CancelInvoiceApplication(application.Id, user.Id+1)
	assert.Error(t, err)
	cancelled, err := CancelInvoiceApplication(application.Id, user.Id)
	require.NoError(t, err)
	assert.Equal(t, InvoiceStatusCancelled, cancelled.Status)
	require.Len(t, cancelled.Items, 1)
	assert.Positive(t, cancelled.Items[0].ReleasedAt)
	assert.Zero(t, cancelled.Items[0].FrozenUntil)

	credits, total, err := GetInvoiceCredits(user.Id, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, credits, 1)
	assert.Equal(t, log.RequestId, credits[0].RequestId)
	_, err = CancelInvoiceApplication(application.Id, user.Id)
	assert.ErrorIs(t, err, ErrInvoiceNotCancellable)
	_, err = UpdateInvoiceApplication(application.Id, InvoiceStatusProcessing, "", 0)
	assert.ErrorIs(t, err, ErrInvoiceStatusTerminal)
}

func setupInvoiceTest(t *testing.T) {
	t.Helper()
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&InvoicePaymentProof{}))
	t.Cleanup(func() { require.NoError(t, DB.Where("1 = 1").Delete(&InvoicePaymentProof{}).Error) })
}

func TestInvoiceAllowedSourcesFilterCreditsAndBlockForgedRequests(t *testing.T) {
	setupInvoiceTest(t)
	previous := operation_setting.GetInvoiceSetting()
	setting := config.GlobalConfig.Get("invoice_setting").(*operation_setting.InvoiceSetting)
	t.Cleanup(func() { *setting = previous })
	setting.AllowedSources = []string{QuotaIncreaseSourceOnlineRecharge, QuotaIncreaseSourceRedemption, QuotaIncreaseSourceAdminAdjustment}
	user := User{Username: "invoice-source-user", Email: "source@example.com", AffCode: "invoice-source-aff"}
	require.NoError(t, DB.Create(&user).Error)
	logs := []Log{
		{UserId: user.Id, Type: LogTypeQuotaIncrease, Quota: 500, RequestId: "invoice-source-online", Other: common.MapToJsonStr(map[string]any{"source": QuotaIncreaseSourceOnlineRecharge})},
		{UserId: user.Id, Type: LogTypeQuotaIncrease, Quota: 500, RequestId: "invoice-source-redemption", Other: common.MapToJsonStr(map[string]any{"source": QuotaIncreaseSourceRedemption})},
		{UserId: user.Id, Type: LogTypeQuotaIncrease, Quota: 500, RequestId: "invoice-source-admin", Other: common.MapToJsonStr(map[string]any{"source": QuotaIncreaseSourceAdminAdjustment})},
	}
	require.NoError(t, LOG_DB.Create(&logs).Error)
	legacy, err := CreateInvoiceApplication(user.Id, user.Email, "USD", 500, []string{logs[2].RequestId})
	require.NoError(t, err)
	setting.AllowedSources = []string{QuotaIncreaseSourceOnlineRecharge}
	credits, total, err := GetInvoiceCredits(user.Id, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, credits, 1)
	assert.Equal(t, logs[0].RequestId, credits[0].RequestId)
	_, err = CreateInvoiceApplication(user.Id, user.Email, "USD", 500, []string{logs[1].RequestId})
	assert.ErrorIs(t, err, ErrInvoiceSourcesInvalid)
	setting.AllowedSources = []string{}
	credits, total, err = GetInvoiceCredits(user.Id, 0, 10)
	require.NoError(t, err)
	assert.Empty(t, credits)
	assert.Zero(t, total)
	_, err = CreateInvoiceApplication(user.Id, user.Email, "USD", 500, []string{logs[0].RequestId})
	assert.ErrorIs(t, err, ErrInvoiceSourcesInvalid)
	applications, applicationTotal, err := GetInvoiceApplications(user.Id, 0, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 1, applicationTotal)
	require.Len(t, applications, 1)
	assert.Equal(t, legacy.Id, applications[0].Id)
	assert.Empty(t, applications[0].InvoiceType)
	assert.Empty(t, applications[0].PaymentProofs)
	_, err = CancelInvoiceApplication(legacy.Id, user.Id)
	require.NoError(t, err)
}

func TestInvoiceSourceSettingPersistsEmptyAndRejectsUnknownValues(t *testing.T) {
	setupInvoiceTest(t)
	common.OptionMapRWMutex.Lock()
	previousOptions := common.OptionMap
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptions
		common.OptionMapRWMutex.Unlock()
	})
	previous := operation_setting.GetInvoiceSetting()
	setting := config.GlobalConfig.Get("invoice_setting").(*operation_setting.InvoiceSetting)
	t.Cleanup(func() { *setting = previous })
	require.NoError(t, UpdateOption("invoice_setting.allowed_sources", `[]`))
	assert.Empty(t, operation_setting.GetInvoiceSetting().AllowedSources)
	for _, value := range []string{`null`, `{}`, `[1]`, `["checkin"]`, `["redemption","redemption"]`} {
		assert.Error(t, UpdateOption("invoice_setting.allowed_sources", value), value)
	}
	var option Option
	require.NoError(t, DB.First(&option, "key = ?", "invoice_setting.allowed_sources").Error)
	assert.Equal(t, `[]`, option.Value)
	require.NoError(t, UpdateOption("invoice_setting.allowed_sources", `["redemption"]`))
	assert.Equal(t, []string{QuotaIncreaseSourceRedemption}, operation_setting.GetInvoiceSetting().AllowedSources)
}

func TestInvoiceApplicationMaterialsAreAtomicAndPrivate(t *testing.T) {
	setupInvoiceTest(t)
	user := User{Username: "invoice-proof-user", Email: "proof@example.com", AffCode: "invoice-proof-aff"}
	require.NoError(t, DB.Create(&user).Error)
	log := Log{UserId: user.Id, Type: LogTypeQuotaIncrease, Quota: 500, RequestId: "invoice-proof-online", Other: common.MapToJsonStr(map[string]any{"source": QuotaIncreaseSourceOnlineRecharge})}
	require.NoError(t, LOG_DB.Create(&log).Error)
	details := InvoiceApplicationDetails{
		InvoiceType: InvoiceTypeSpecial, CompanyName: "Example Company", TaxNumber: "913100000000000000", Notes: "request notes",
		PaymentProofs: []InvoicePaymentProof{{Filename: "payment.png", ContentType: "image/png", Data: []byte("private payment evidence")}},
	}
	failProofWrite := true
	require.NoError(t, DB.Callback().Create().Before("gorm:create").Register("invoice_test_proof_write_failure", func(tx *gorm.DB) {
		if failProofWrite && tx.Statement.Schema != nil && tx.Statement.Schema.Name == "InvoicePaymentProof" {
			tx.AddError(errors.New("payment evidence write failed"))
		}
	}))
	t.Cleanup(func() { require.NoError(t, DB.Callback().Create().Remove("invoice_test_proof_write_failure")) })
	_, err := CreateInvoiceApplication(user.Id, "delivery@example.com", "USD", 500, []string{log.RequestId}, details)
	require.Error(t, err)
	for _, table := range []any{&InvoiceApplication{}, &InvoiceItem{}, &InvoicePaymentProof{}} {
		var count int64
		require.NoError(t, DB.Model(table).Count(&count).Error)
		assert.Zero(t, count)
	}
	failProofWrite = false
	application, err := CreateInvoiceApplication(user.Id, "delivery@example.com", "USD", 500, []string{log.RequestId}, details)
	require.NoError(t, err)
	assert.Equal(t, details.InvoiceType, application.InvoiceType)
	assert.Equal(t, details.CompanyName, application.CompanyName)
	assert.Equal(t, details.TaxNumber, application.TaxNumber)
	assert.Equal(t, details.Notes, application.Notes)
	require.Len(t, application.PaymentProofs, 1)
	assert.Empty(t, application.PaymentProofs[0].Data)
	applications, _, err := GetInvoiceApplications(user.Id, 0, 10)
	require.NoError(t, err)
	require.Len(t, applications, 1)
	require.Len(t, applications[0].PaymentProofs, 1)
	proof := applications[0].PaymentProofs[0]
	assert.Empty(t, proof.Data)
	assert.EqualValues(t, len(details.PaymentProofs[0].Data), proof.Size)
	storedProof, err := GetInvoicePaymentProof(application.Id, proof.Id, user.Id, false)
	require.NoError(t, err)
	assert.Equal(t, details.PaymentProofs[0].Data, storedProof.Data)
	_, err = GetInvoicePaymentProof(application.Id, proof.Id, user.Id+1, false)
	assert.Error(t, err)
	_, err = GetInvoicePaymentProof(application.Id+1, proof.Id, user.Id, true)
	assert.Error(t, err)
	require.NoError(t, DB.First(&user, user.Id).Error)
	assert.Equal(t, "proof@example.com", user.Email)
}
