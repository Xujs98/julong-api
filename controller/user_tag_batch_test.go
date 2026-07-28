package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func performBatchQuotaRequest(t *testing.T, role int, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/user/batch-quota", strings.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("id", 9999)
	context.Set("role", role)
	context.Set("username", "quota-operator")
	BatchAdjustUserQuota(context)
	return recorder
}

func TestUserTagFilterAndDeleteClearsAssignments(t *testing.T) {
	db := setupManageUserTestDB(t)
	red := model.UserTag{Name: "Red", Color: "#ff0000"}
	blue := model.UserTag{Name: "Blue", Color: "#0077ff"}
	require.NoError(t, model.CreateUserTag(&red))
	require.NoError(t, model.CreateUserTag(&blue))
	assert.Equal(t, "#FF0000", red.Color)

	users := []model.User{
		{Username: "tag-red-user", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: "tag-red-aff"},
		{Username: "tag-blue-user", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: "tag-blue-aff"},
	}
	require.NoError(t, db.Create(&users).Error)
	require.NoError(t, model.SetUserTag(users[0].Id, red.Id))
	require.NoError(t, model.SetUserTag(users[1].Id, blue.Id))

	filtered, total, err := model.SearchUsers("", "", nil, nil, &red.Id, 0, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, filtered, 1)
	assert.Equal(t, users[0].Id, filtered[0].Id)
	require.NotNil(t, filtered[0].Tag)
	assert.Equal(t, "Red", filtered[0].Tag.Name)

	require.NoError(t, model.DeleteUserTag(red.Id))
	var updated model.User
	require.NoError(t, db.First(&updated, users[0].Id).Error)
	assert.Zero(t, updated.TagId)
}

func TestBuiltInRiskTagsAreReadOnlyAndFilterUsersBySevenDayRisk(t *testing.T) {
	db := setupManageUserTestDB(t)
	previousGlobalEnabled := common.UserRiskDetectionEnabled
	common.UserRiskDetectionEnabled = true
	t.Cleanup(func() {
		common.UserRiskDetectionEnabled = previousGlobalEnabled
	})

	tags, err := model.ListUserTags()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(tags), 2)
	assert.Equal(t, model.UserTagRiskMediumId, tags[0].Id)
	assert.Equal(t, model.UserRiskLevelMedium, tags[0].RiskLevel)
	assert.True(t, tags[0].BuiltIn)
	assert.Equal(t, model.UserTagRiskHighId, tags[1].Id)
	assert.Equal(t, model.UserRiskLevelHigh, tags[1].RiskLevel)
	assert.True(t, tags[1].BuiltIn)
	assert.Error(t, model.UpdateUserTag(&model.UserTag{Id: model.UserTagRiskMediumId, Name: "Changed", Color: "#000000"}))
	assert.Error(t, model.DeleteUserTag(model.UserTagRiskHighId))

	users := []model.User{
		{Username: "medium-risk-user", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: "medium-risk-aff"},
		{Username: "high-risk-user", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: "high-risk-aff"},
		{Username: "low-risk-user", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: "low-risk-aff"},
	}
	require.NoError(t, db.Create(&users).Error)
	assert.Error(t, model.SetUserTag(users[0].Id, model.UserTagRiskMediumId))

	now := time.Now().Unix()
	logs := []model.Log{
		{
			UserId:    users[0].Id,
			CreatedAt: now,
			Type:      model.LogTypeError,
			Other: common.MapToJsonStr(map[string]interface{}{
				"error_code": "sensitive_words_detected",
			}),
		},
	}
	for index := 0; index < 3; index++ {
		logs = append(logs, model.Log{
			UserId:    users[1].Id,
			CreatedAt: now - int64(index),
			Type:      model.LogTypeQuotaIncrease,
			Other: common.MapToJsonStr(map[string]interface{}{
				"refund_after_output": true,
			}),
		})
	}
	require.NoError(t, db.Create(&logs).Error)

	mediumRiskTagId := model.UserTagRiskMediumId
	mediumUsers, total, err := model.SearchUsers("", "", nil, nil, &mediumRiskTagId, 0, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, mediumUsers, 1)
	assert.Equal(t, users[0].Id, mediumUsers[0].Id)
	require.NotNil(t, mediumUsers[0].RiskTag)
	assert.Equal(t, model.UserTagRiskMediumId, mediumUsers[0].RiskTag.Id)
	mediumSummary, err := model.GetUserQuotaSummary("", "", nil, nil, &mediumRiskTagId)
	require.NoError(t, err)
	assert.EqualValues(t, 1, mediumSummary.UserCount)

	highRiskTagId := model.UserTagRiskHighId
	highUsers, total, err := model.SearchUsers("", "", nil, nil, &highRiskTagId, 0, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, highUsers, 1)
	assert.Equal(t, users[1].Id, highUsers[0].Id)
	require.NotNil(t, highUsers[0].RiskTag)
	assert.Equal(t, model.UserTagRiskHighId, highUsers[0].RiskTag.Id)
	highSummary, err := model.GetUserQuotaSummary("", "", nil, nil, &highRiskTagId)
	require.NoError(t, err)
	assert.EqualValues(t, 1, highSummary.UserCount)

	common.UserRiskDetectionEnabled = false
	mediumUsers, total, err = model.SearchUsers("", "", nil, nil, &mediumRiskTagId, 0, 20)
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, mediumUsers)
	require.NoError(t, model.SetUserRiskDetectionEnabled(users[0].Id, true))
	mediumUsers, total, err = model.SearchUsers("", "", nil, nil, &mediumRiskTagId, 0, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, mediumUsers, 1)
	assert.Equal(t, users[0].Id, mediumUsers[0].Id)
}

func TestBatchQuotaAdjustsSelectedUsersAndRecordsIncreases(t *testing.T) {
	db := setupManageUserTestDB(t)
	users := []model.User{
		{Username: "batch-one", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", Quota: 100, AffCode: "batch-one-aff"},
		{Username: "batch-two", Password: "password", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, Group: "default", Quota: 200, AffCode: "batch-two-aff"},
	}
	require.NoError(t, db.Create(&users).Error)

	body := fmt.Sprintf(`{"mode":"add","value":50,"user_ids":[%d,%d]}`, users[0].Id, users[1].Id)
	recorder := performBatchQuotaRequest(t, common.RoleRootUser, body)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"adjusted_count":2`)

	var updated []model.User
	require.NoError(t, db.Where("id IN ?", []int{users[0].Id, users[1].Id}).Order("id asc").Find(&updated).Error)
	require.Len(t, updated, 2)
	assert.Equal(t, 150, updated[0].Quota)
	assert.Equal(t, 250, updated[1].Quota)
	for _, user := range updated {
		logs, total, err := model.GetUserQuotaIncreaseLogs(user.Id, 0, 10)
		require.NoError(t, err)
		assert.EqualValues(t, 1, total)
		require.Len(t, logs, 1)
		assert.Equal(t, 50, logs[0].Quota)
		assert.Equal(t, model.QuotaIncreaseSourceAdminAdjustment, logs[0].Source)
	}
}

func TestBatchQuotaAdminAllUsersRespectsRoleAndInvalidSelectionRollsBack(t *testing.T) {
	db := setupManageUserTestDB(t)
	users := []model.User{
		{Username: "batch-common", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", Quota: 100, AffCode: "batch-common-aff"},
		{Username: "batch-admin", Password: "password", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, Group: "default", Quota: 200, AffCode: "batch-admin-aff"},
		{Username: "batch-root", Password: "password", Role: common.RoleRootUser, Status: common.UserStatusEnabled, Group: "default", Quota: 300, AffCode: "batch-root-aff"},
	}
	require.NoError(t, db.Create(&users).Error)

	recorder := performBatchQuotaRequest(t, common.RoleAdminUser, `{"mode":"subtract","value":25,"all_users":true}`)
	assert.Contains(t, recorder.Body.String(), `"adjusted_count":1`)
	for index, expected := range []int{75, 200, 300} {
		var user model.User
		require.NoError(t, db.First(&user, users[index].Id).Error)
		assert.Equal(t, expected, user.Quota)
	}

	body := fmt.Sprintf(`{"mode":"add","value":40,"user_ids":[%d,%d]}`, users[0].Id, users[1].Id)
	recorder = performBatchQuotaRequest(t, common.RoleAdminUser, body)
	assert.Contains(t, recorder.Body.String(), `"success":false`)
	var commonUser model.User
	require.NoError(t, db.First(&commonUser, users[0].Id).Error)
	assert.Equal(t, 75, commonUser.Quota)
}
