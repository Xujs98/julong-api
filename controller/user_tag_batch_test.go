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
