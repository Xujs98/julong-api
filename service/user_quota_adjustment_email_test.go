package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendUserQuotaAdjustmentEmailsReportsDeliveryOutcomes(t *testing.T) {
	adjustments := []model.UserQuotaAdjustment{
		{UserId: 1, Username: "alice", DisplayName: "Alice", Email: "alice@example.com", Mode: model.UserQuotaAdjustModeAdd, Value: 50, PreviousQuota: 100, CurrentQuota: 150},
		{UserId: 2, Username: "bob", DisplayName: "Bob", Email: "bob@example.com", Mode: model.UserQuotaAdjustModeSubtract, Value: 25, PreviousQuota: 100, CurrentQuota: 75},
		{UserId: 3, Username: "no-email", Email: "invalid", Mode: model.UserQuotaAdjustModeOverride, Value: 0, PreviousQuota: 100, CurrentQuota: 0},
	}
	var deliveredContent string
	result := sendUserQuotaAdjustmentEmails(
		adjustments,
		"root",
		EmailTemplateLocaleChinese,
		"[{{system_name}}] {{operation}}",
		"<p>{{display_name}} {{operation}} {{current_quota}}</p>",
		func(subject, receiver, content string) error {
			if receiver == "bob@example.com" {
				return errors.New("SMTP rejected recipient")
			}
			require.Contains(t, subject, "增加")
			deliveredContent = content
			return nil
		},
	)

	assert.Equal(t, 1, result.SuccessCount)
	assert.Equal(t, 1, result.FailedCount)
	assert.Equal(t, 1, result.SkippedCount)
	assert.True(t, strings.Contains(deliveredContent, "Alice 增加"))
}
