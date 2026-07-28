package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserQuotaSummaryReturnsFilteredAggregate(t *testing.T) {
	db := setupManageUserTestDB(t)
	users := []model.User{
		{Username: "summary-vip", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "vip", Quota: 125, AffCode: "summary-vip-aff"},
		{Username: "summary-disabled", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusDisabled, Group: "vip", Quota: 250, AffCode: "summary-disabled-aff"},
		{Username: "summary-default", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", Quota: 500, AffCode: "summary-default-aff"},
	}
	require.NoError(t, db.Create(&users).Error)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/user/quota-summary?group=vip&status=1", nil)
	GetUserQuotaSummary(context)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"success":true,"message":"","data":{"total_quota":125,"user_count":1}}`, recorder.Body.String())
}
