package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetControllerUserQuotaSummarySettings(t *testing.T) {
	t.Helper()
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	previous, existed := common.OptionMap[common.UserQuotaSummaryExcludedUserIDsOptionKey]
	common.OptionMap[common.UserQuotaSummaryExcludedUserIDsOptionKey] = "[]"
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		if existed {
			common.OptionMap[common.UserQuotaSummaryExcludedUserIDsOptionKey] = previous
		} else {
			delete(common.OptionMap, common.UserQuotaSummaryExcludedUserIDsOptionKey)
		}
		common.OptionMapRWMutex.Unlock()
	})
}

func TestGetUserQuotaSummaryReturnsFilteredAggregate(t *testing.T) {
	db := setupManageUserTestDB(t)
	resetControllerUserQuotaSummarySettings(t)
	users := []model.User{
		{Username: "summary-vip", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "vip", Quota: 125, AffCode: "summary-vip-aff"},
		{Username: "summary-disabled", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusDisabled, Group: "vip", Quota: 250, AffCode: "summary-disabled-aff"},
		{Username: "summary-default", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", Quota: 500, AffCode: "summary-default-aff"},
	}
	require.NoError(t, db.Create(&users).Error)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/user/quota-summary?keyword=summary-vip&status=1", nil)
	GetUserQuotaSummary(context)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"success":true,"message":"","data":{"total_quota":125,"user_count":1}}`, recorder.Body.String())
}

func TestUpdateUserQuotaSummarySettingsPersistsSelectionAndRefreshesSummary(t *testing.T) {
	db := setupManageUserTestDB(t)
	resetControllerUserQuotaSummarySettings(t)
	users := []model.User{
		{Username: "included-user", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", Quota: 125, AffCode: "included-user-aff"},
		{Username: "excluded-user", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", Quota: 250, AffCode: "excluded-user-aff"},
	}
	require.NoError(t, db.Create(&users).Error)

	gin.SetMode(gin.TestMode)
	updateRecorder := httptest.NewRecorder()
	updateContext, _ := gin.CreateTestContext(updateRecorder)
	updateContext.Request = httptest.NewRequest(http.MethodPut, "/api/user/quota-summary/settings", strings.NewReader(fmt.Sprintf(`{"excluded_user_ids":[%d]}`, users[1].Id)))
	updateContext.Request.Header.Set("Content-Type", "application/json")
	updateContext.Set("id", 9999)
	updateContext.Set("role", common.RoleRootUser)
	updateContext.Set("username", "root-operator")
	UpdateUserQuotaSummarySettings(updateContext)

	assert.Equal(t, http.StatusOK, updateRecorder.Code)
	expectedSettingsResponse := fmt.Sprintf(`{"success":true,"message":"","data":{"excluded_user_ids":[%d]}}`, users[1].Id)
	assert.JSONEq(t, expectedSettingsResponse, updateRecorder.Body.String())

	settingsRecorder := httptest.NewRecorder()
	settingsContext, _ := gin.CreateTestContext(settingsRecorder)
	settingsContext.Request = httptest.NewRequest(http.MethodGet, "/api/user/quota-summary/settings", nil)
	GetUserQuotaSummarySettings(settingsContext)
	assert.Equal(t, http.StatusOK, settingsRecorder.Code)
	assert.JSONEq(t, expectedSettingsResponse, settingsRecorder.Body.String())

	summaryRecorder := httptest.NewRecorder()
	summaryContext, _ := gin.CreateTestContext(summaryRecorder)
	summaryContext.Request = httptest.NewRequest(http.MethodGet, "/api/user/quota-summary", nil)
	GetUserQuotaSummary(summaryContext)
	assert.JSONEq(t, `{"success":true,"message":"","data":{"total_quota":125,"user_count":1}}`, summaryRecorder.Body.String())
}
