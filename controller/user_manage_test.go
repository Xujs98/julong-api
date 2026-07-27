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
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupManageUserTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousRedisEnabled := common.RedisEnabled
	previousMainDatabaseType, previousLogDatabaseType := common.MainDatabaseType(), common.LogDatabaseType()
	common.RedisEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB, model.LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(
		&model.UserTag{}, &model.User{}, &model.UserSession{}, &model.Token{}, &model.Log{}, &model.CasbinRule{}, &model.AuthzRole{},
	))

	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.RedisEnabled = previousRedisEnabled
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func performManageUserRequest(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/user/manage", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 9999)
	c.Set("role", common.RoleRootUser)
	c.Set("username", "root-operator")
	ManageUser(c)
	return recorder
}

func TestManageUserDisableAdvancesAuthVersionOnceAndRevokesSession(t *testing.T) {
	db := setupManageUserTestDB(t)
	now := time.Now().Unix()
	user := model.User{
		Username: "managed-disable-user", Password: "password", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1,
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&model.UserSession{
		SID: "managed-disable-session", UserID: user.Id, Version: 1, UserAuthVersion: 1,
		Status: model.UserSessionStatusActive, RefreshHash: "refresh-hash", LoginMethod: "password",
		LastActiveAt: now, ExpiresAt: now + 3600,
	}).Error)

	recorder := performManageUserRequest(t, fmt.Sprintf(`{"id":%d,"action":"disable"}`, user.Id))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":true`)

	var updated model.User
	require.NoError(t, db.First(&updated, user.Id).Error)
	assert.Equal(t, common.UserStatusDisabled, updated.Status)
	assert.EqualValues(t, 2, updated.AuthVersion)
	var session model.UserSession
	require.NoError(t, db.First(&session, "sid = ?", "managed-disable-session").Error)
	assert.Equal(t, model.UserSessionStatusRevoked, session.Status)
}

func TestManageUserDemoteAdvancesAuthVersionAndRevokesSessionsOnce(t *testing.T) {
	db := setupManageUserTestDB(t)
	previousMaster := common.IsMasterNode
	common.IsMasterNode = false
	t.Cleanup(func() { common.IsMasterNode = previousMaster })
	require.NoError(t, authz.Init(db))

	now := time.Now().Unix()
	user := model.User{
		Username: "managed-demote-user", Password: "password", Role: common.RoleAdminUser,
		Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1,
	}
	require.NoError(t, db.Create(&user).Error)
	for _, sid := range []string{"managed-demote-session-one", "managed-demote-session-two"} {
		require.NoError(t, db.Create(&model.UserSession{
			SID: sid, UserID: user.Id, Version: 1, UserAuthVersion: 1,
			Status: model.UserSessionStatusActive, RefreshHash: "refresh-" + sid, LoginMethod: "password",
			LastActiveAt: now, ExpiresAt: now + 3600,
		}).Error)
	}

	sessionUpdateCount := 0
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("test:count_demote_session_updates", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "user_sessions" {
			sessionUpdateCount++
		}
	}))

	recorder := performManageUserRequest(t, fmt.Sprintf(`{"id":%d,"action":"demote"}`, user.Id))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":true`)

	var updated model.User
	require.NoError(t, db.First(&updated, user.Id).Error)
	assert.Equal(t, common.RoleCommonUser, updated.Role)
	assert.EqualValues(t, 2, updated.AuthVersion)
	var sessions []model.UserSession
	require.NoError(t, db.Where("user_id = ?", user.Id).Order("sid asc").Find(&sessions).Error)
	require.Len(t, sessions, 2)
	for _, session := range sessions {
		assert.Equal(t, model.UserSessionStatusRevoked, session.Status)
		assert.Equal(t, "admin_demote", session.RevokedReason)
	}
	assert.Equal(t, 1, sessionUpdateCount)
}

func TestManageUserDeleteReturnsImmediatelyAndUnknownActionFails(t *testing.T) {
	db := setupManageUserTestDB(t)
	deleted := model.User{
		Username: "managed-delete-user", Password: "password", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1, AffCode: "delete-aff",
	}
	require.NoError(t, db.Create(&deleted).Error)

	recorder := performManageUserRequest(t, fmt.Sprintf(`{"id":%d,"action":"delete"}`, deleted.Id))
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	var deletedCount int64
	require.NoError(t, db.Unscoped().Model(&model.User{}).Where("id = ? AND deleted_at IS NOT NULL", deleted.Id).Count(&deletedCount).Error)
	assert.EqualValues(t, 1, deletedCount)

	unchanged := model.User{
		Username: "managed-unknown-user", Password: "password", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1, AffCode: "unknown-aff",
	}
	require.NoError(t, db.Create(&unchanged).Error)
	recorder = performManageUserRequest(t, fmt.Sprintf(`{"id":%d,"action":"unknown"}`, unchanged.Id))
	assert.Contains(t, recorder.Body.String(), `"success":false`)
	require.NoError(t, db.First(&unchanged, unchanged.Id).Error)
	assert.EqualValues(t, 1, unchanged.AuthVersion)
	assert.Equal(t, common.UserStatusEnabled, unchanged.Status)
}

func TestManageUserRecordsOnlyPositiveQuotaAdjustments(t *testing.T) {
	db := setupManageUserTestDB(t)
	user := model.User{
		Username: "managed-quota-user", Password: "password", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", Quota: 100,
	}
	require.NoError(t, db.Create(&user).Error)

	recorder := performManageUserRequest(t, fmt.Sprintf(`{"id":%d,"action":"add_quota","mode":"add","value":250}`, user.Id))
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	recorder = performManageUserRequest(t, fmt.Sprintf(`{"id":%d,"action":"add_quota","mode":"override","value":500}`, user.Id))
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	recorder = performManageUserRequest(t, fmt.Sprintf(`{"id":%d,"action":"add_quota","mode":"subtract","value":50}`, user.Id))
	assert.Contains(t, recorder.Body.String(), `"success":true`)

	logs, total, err := model.GetUserQuotaIncreaseLogs(user.Id, 0, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, logs, 2)
	assert.Equal(t, 150, logs[0].Quota)
	assert.Equal(t, 250, logs[1].Quota)
	assert.Equal(t, model.QuotaIncreaseSourceAdminAdjustment, logs[0].Source)
}

func TestAdminGetUserQuotaIncreaseLogsReturnsTargetUserPage(t *testing.T) {
	db := setupManageUserTestDB(t)
	user := model.User{
		Username: "quota-page-user", Password: "password", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default",
	}
	require.NoError(t, db.Create(&user).Error)
	model.RecordQuotaIncreaseLog(user.Id, 120, model.QuotaIncreaseSourceCheckin, "check-in reward")
	model.RecordQuotaIncreaseLog(user.Id, 340, model.QuotaIncreaseSourceRedemption, "redemption reward")

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/user/%d/quota-increases?p=1&page_size=1", user.Id), nil)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", user.Id)}}
	c.Set("id", 9999)
	c.Set("role", common.RoleRootUser)
	c.Set("username", "root-operator")

	AdminGetUserQuotaIncreaseLogs(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"total":2`)
	assert.Contains(t, recorder.Body.String(), `"page_size":1`)
	assert.Contains(t, recorder.Body.String(), `"quota":340`)
	assert.Contains(t, recorder.Body.String(), `"source":"redemption"`)
	assert.NotContains(t, recorder.Body.String(), `"quota":120`)
}

func TestAdminGetUserUsageSummaryReturnsSelectedGroupRatiosAndUsage(t *testing.T) {
	db := setupManageUserTestDB(t)
	previousGroupRatio := ratio_setting.GroupRatio2JSONString()
	previousGroupGroupRatio := ratio_setting.GroupGroupRatio2JSONString()
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":0.8,"svip":0.6,"codex-v1":0.12}`))
	require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(`{"default":{"svip":0.4,"codex-v1":0.08}}`))
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroupRatio))
		require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(previousGroupGroupRatio))
	})
	user := model.User{
		Username: "usage-summary-user", Password: "password", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default",
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&[]model.Token{
		{UserId: user.Id, Key: "usage-summary-svip", Name: "svip", Group: "svip"},
		{UserId: user.Id, Key: "usage-summary-codex", Name: "codex", Group: "codex-v1"},
	}).Error)
	require.NoError(t, db.Create(&[]model.Log{
		{UserId: user.Id, Type: model.LogTypeConsume, Group: "svip", Quota: 120, PromptTokens: 200, CompletionTokens: 100},
		{UserId: user.Id, Type: model.LogTypeConsume, Group: "svip", Quota: 30, PromptTokens: 20, CompletionTokens: 10},
		{UserId: user.Id, Type: model.LogTypeConsume, Group: "codex-v1", Quota: 80, PromptTokens: 800, CompletionTokens: 200},
		{UserId: user.Id, Type: model.LogTypeConsume, Group: "vip", Quota: 999, PromptTokens: 999, CompletionTokens: 1},
		{UserId: user.Id, Type: model.LogTypeRefund, Group: "codex-v1", Quota: -20, PromptTokens: 100, CompletionTokens: 100},
	}).Error)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/user/%d/usage-summary", user.Id), nil)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", user.Id)}}
	c.Set("id", 9999)
	c.Set("role", common.RoleRootUser)
	c.Set("username", "root-operator")

	AdminGetUserUsageSummary(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			GroupRatios map[string]float64 `json:"group_ratios"`
			GroupUsage  map[string]struct {
				Ratio     float64 `json:"ratio"`
				Quota     int64   `json:"quota"`
				TokenUsed int64   `json:"token_used"`
			} `json:"group_usage"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, map[string]float64{
		"codex-v1": 0.08,
		"svip":     0.4,
	}, response.Data.GroupRatios)
	require.Len(t, response.Data.GroupUsage, 2)
	assert.Equal(t, 0.08, response.Data.GroupUsage["codex-v1"].Ratio)
	assert.EqualValues(t, 80, response.Data.GroupUsage["codex-v1"].Quota)
	assert.EqualValues(t, 1000, response.Data.GroupUsage["codex-v1"].TokenUsed)
	assert.Equal(t, 0.4, response.Data.GroupUsage["svip"].Ratio)
	assert.EqualValues(t, 150, response.Data.GroupUsage["svip"].Quota)
	assert.EqualValues(t, 330, response.Data.GroupUsage["svip"].TokenUsed)
	assert.NotContains(t, response.Data.GroupUsage, "default")
	assert.NotContains(t, response.Data.GroupUsage, "vip")
}
