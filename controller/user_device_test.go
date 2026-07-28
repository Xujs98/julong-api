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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminUpdateUserLoginDevicesBlocksAndRevokesSessions(t *testing.T) {
	db := setupManageUserTestDB(t)
	user := model.User{
		Username: "device-controller-user", Password: "password", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1,
	}
	require.NoError(t, db.Create(&user).Error)
	deviceID := uuid.NewString()
	now := time.Now().Unix()
	require.NoError(t, db.Create(&model.UserSession{
		SID: "device-controller-session", UserID: user.Id, Version: 1, UserAuthVersion: 1,
		Status: model.UserSessionStatusActive, RefreshHash: "device-controller-hash", LoginMethod: "password",
		IP: "192.0.2.30", UserAgent: "Mozilla/5.0 Chrome/120.0 Mac OS", DeviceID: deviceID,
		CreatedAt: now, LastActiveAt: now, ExpiresAt: now + 3600,
	}).Error)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPut,
		fmt.Sprintf("/api/user/%d/login-devices", user.Id),
		strings.NewReader(fmt.Sprintf(`{"device_ids":[%q],"blocked":true}`, deviceID)),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", user.Id)}}
	context.Set("id", 9999)
	context.Set("role", common.RoleRootUser)
	context.Set("username", "root-operator")

	AdminUpdateUserLoginDevices(context)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	assert.Contains(t, recorder.Body.String(), `"revoked_count":1`)
	blocked, err := model.IsUserDeviceBlocked(user.Id, deviceID)
	require.NoError(t, err)
	assert.True(t, blocked)
	var session model.UserSession
	require.NoError(t, db.First(&session, "sid = ?", "device-controller-session").Error)
	assert.Equal(t, model.UserSessionStatusRevoked, session.Status)
}
