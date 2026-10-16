package router

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func siteAccessFixture(t *testing.T) *gin.Engine {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	previousMaster, previousRedis, previousSQLite := common.IsMasterNode, common.RedisEnabled, common.SQLitePath
	common.IsMasterNode, common.RedisEnabled = false, false
	common.SQLitePath = filepath.Join(t.TempDir(), "site.db")
	t.Setenv("SQL_DSN", os.Getenv("SITE_ACCESS_TEST_DSN"))
	t.Setenv("LOG_SQL_DSN", "")
	var dialector gorm.Dialector = sqlite.Open(common.SQLitePath)
	switch os.Getenv("SITE_ACCESS_TEST_DIALECT") {
	case "mysql":
		dialector = mysql.Open(os.Getenv("SITE_ACCESS_TEST_DSN"))
	case "postgres":
		dialector = postgres.Open(os.Getenv("SITE_ACCESS_TEST_DSN"))
	}
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.BlockedIP{}, &model.BlockedDevice{}, &model.UserAccessToken{}, &model.Token{}, &model.Option{}, &model.AuditLog{}, &model.CasbinRule{}, &model.AuthzRole{}))
	initialSQL, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, initialSQL.Close())
	require.NoError(t, model.InitDB()) // Initializes the dialect's reserved columns too.
	db = model.DB
	var logDialect gorm.Dialector = sqlite.Open(filepath.Join(t.TempDir(), "logs.db"))
	if dsn := os.Getenv("SITE_ACCESS_TEST_LOG_DSN"); dsn != "" {
		switch os.Getenv("SITE_ACCESS_TEST_DIALECT") {
		case "mysql":
			logDialect = mysql.Open(dsn)
		case "postgres":
			logDialect = postgres.Open(dsn)
		}
	}
	logDB, err := gorm.Open(logDialect, &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, logDB.AutoMigrate(&model.Log{}, &model.AuditLog{}))
	model.LOG_DB = logDB
	t.Setenv("FRONTEND_BASE_URL", "https://frontend.invalid")
	t.Cleanup(func() {
		var blocks []model.BlockedIP
		require.NoError(t, db.Find(&blocks).Error)
		for _, block := range blocks {
			require.NoError(t, model.UnblockIPAddresses([]string{block.IP}))
		}
		for _, table := range []any{&model.BlockedDevice{}, &model.UserSession{}, &model.UserAccessToken{}, &model.Token{}, &model.User{}, &model.AuditLog{}, &model.CasbinRule{}, &model.AuthzRole{}} {
			require.NoError(t, db.Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(table).Error)
		}
		require.NoError(t, logDB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.Log{}).Error)
		require.NoError(t, logDB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.AuditLog{}).Error)
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.IsMasterNode, common.RedisEnabled, common.SQLitePath = previousMaster, previousRedis, previousSQLite
		common.SetDatabaseTypes(previousMainType, previousLogType)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
		sqlLog, err := logDB.DB()
		require.NoError(t, err)
		require.NoError(t, sqlLog.Close())
	})
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	require.NoError(t, engine.SetTrustedProxies(nil))
	SetRouter(engine, WebAssets{})
	return engine
}

func TestSiteAccessBlockedIPCannotReachAnyRoute(t *testing.T) {
	engine := siteAccessFixture(t)
	require.NoError(t, model.BlockIPAddresses([]string{"198.51.100.201"}, 0, 1, "fixture"))
	for _, path := range []string{"/", "/sign-in", "/assets/app.js", "/api/status", "/api/user/self", "/v1/models", "/v1/videos", "/unknown"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.RemoteAddr = "198.51.100.201:1234"
			req.Header.Set("X-Forwarded-For", "203.0.113.10")
			out := httptest.NewRecorder()
			engine.ServeHTTP(out, req)
			assert.Equal(t, http.StatusForbidden, out.Code)
			assert.Empty(t, out.Header().Get("Location"))
			assert.Equal(t, "no-store", out.Header().Get("Cache-Control"))
		})
	}
	require.NoError(t, model.UnblockIPAddresses([]string{"198.51.100.201"}))
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	req.RemoteAddr = "198.51.100.201:1234"
	out := httptest.NewRecorder()
	engine.ServeHTTP(out, req)
	assert.Equal(t, http.StatusOK, out.Code)
}

func TestSiteAccessDisabledCredentialsTraceNewIPsAndDevices(t *testing.T) {
	engine := siteAccessFixture(t)
	user := model.User{Username: "site-disabled", AffCode: "site-disabled", Password: "unused", Status: common.UserStatusEnabled, Role: common.RoleCommonUser, AuthVersion: 1, Group: "default"}
	require.NoError(t, model.DB.Create(&user).Error)
	device := uuid.NewString()
	bundle, err := service.CreateLoginSession(user.Id, "password", "198.51.100.202", "fixture", device)
	require.NoError(t, err)
	rawPAT := model.AccessTokenPrefix + "siteFixtureSecret"
	pat := model.UserAccessToken{UserId: user.Id, Name: "fixture", TokenHash: model.AccessTokenFingerprint(rawPAT)}
	require.NoError(t, model.DB.Create(&pat).Error)
	token := model.Token{UserId: user.Id, Key: "siteFixtureRelaySecret", Name: "fixture", Status: common.TokenStatusEnabled, ExpiredTime: -1, UnlimitedQuota: true}
	require.NoError(t, model.DB.Create(&token).Error)
	markerOut := httptest.NewRecorder()
	markerCtx, _ := gin.CreateTestContext(markerOut)
	markerCtx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	service.WriteSiteIdentityCookie(markerCtx, user.Id, device)
	marker := markerOut.Result().Cookies()[0]
	assert.True(t, marker.HttpOnly)
	assert.Equal(t, "/", marker.Path)
	assert.Empty(t, marker.Domain)
	user.Status = common.UserStatusDisabled
	require.NoError(t, model.DisableUserAndBlockIPs(&user, nil, 1))
	knownBlocked, err := model.IsUserDeviceBlocked(user.Id, device)
	require.NoError(t, err)
	assert.True(t, knownBlocked)
	knownIP, err := model.IsIPBlocked("198.51.100.202")
	require.NoError(t, err)
	assert.True(t, knownIP)
	for i, tc := range []struct{ name, path, header, value, cookie string }{
		{"revoked JWT", "/", "Authorization", "Bearer " + bundle.AccessToken, ""},
		{"PAT", "/api/status", "Authorization", "Bearer " + rawPAT, ""},
		{"API key", "/", "Authorization", "Bearer sk-" + token.Key, ""},
		{"provider key", "/v1/messages", "x-api-key", "sk-" + token.Key, ""},
		{"gemini query", "/v1beta/models?key=sk-" + token.Key, "", "", ""},
		{"signed device", "/", "", "", marker.Name + "=" + marker.Value},
		{"revoked refresh", "/api/user/auth/refresh", "", "", service.RefreshCookieName + "=" + bundle.RefreshToken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ip := fmt.Sprintf("203.0.113.%d", 110+i)
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.RemoteAddr = ip + ":1234"
			if tc.header != "" {
				req.Header.Set(tc.header, tc.value)
			}
			if tc.cookie != "" {
				req.Header.Set("Cookie", tc.cookie)
			}
			out := httptest.NewRecorder()
			engine.ServeHTTP(out, req)
			require.Equal(t, http.StatusForbidden, out.Code)
			blocked, err := model.IsIPBlocked(ip)
			require.NoError(t, err)
			assert.True(t, blocked)
			stats, err := model.GetUserLoginIPStats(user.Id)
			require.NoError(t, err)
			found := false
			for _, stat := range stats {
				if stat.IP == ip {
					found = true
					assert.True(t, stat.Blocked)
					assert.Zero(t, stat.LoginCount)
				}
			}
			assert.True(t, found)
			anonymous := httptest.NewRequest(http.MethodGet, "/", nil)
			anonymous.RemoteAddr = req.RemoteAddr
			out = httptest.NewRecorder()
			engine.ServeHTTP(out, anonymous)
			assert.Equal(t, http.StatusForbidden, out.Code)
			require.NoError(t, model.UnblockIPAddresses([]string{ip}))
		})
	}
	require.NoError(t, model.UnblockIPAddresses([]string{"198.51.100.202"}))
}

func TestSiteAccessForgedIdentityCannotPoisonBlacklist(t *testing.T) {
	engine := siteAccessFixture(t)
	user := model.User{Username: "forgery", AffCode: "forgery", Password: "unused", Status: common.UserStatusDisabled, Role: common.RoleCommonUser, AuthVersion: 1}
	require.NoError(t, model.DB.Create(&user).Error)
	device := uuid.NewString()
	out := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(out)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	service.WriteSiteIdentityCookie(ctx, user.Id, device)
	marker := out.Result().Cookies()[0]
	jwt, _, err := service.IssueAccessToken(service.AuthIdentity{UserID: user.Id, SessionID: uuid.NewString(), UserAuthVersion: 1, SessionVersion: 1})
	require.NoError(t, err)
	for i, tc := range []struct{ name, cookie, auth string }{
		{"raw device UUID", service.DeviceCookieName + "=" + device, ""},
		{"tampered signed device", marker.Name + "=" + marker.Value[:len(marker.Value)-2] + "xx", ""},
		{"forged refresh secret", service.RefreshCookieName + "=" + uuid.NewString() + ".unknown-secret", ""},
		{"tampered JWT", "", "Bearer " + jwt[:len(jwt)-2] + "xx"},
		{"unknown API key", "", "Bearer sk-not-a-real-key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ip := fmt.Sprintf("203.0.113.%d", 130+i)
			req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
			req.RemoteAddr = ip + ":1234"
			req.Header.Set("Cookie", tc.cookie)
			req.Header.Set("Authorization", tc.auth)
			req.Header.Set("New-Api-User", fmt.Sprint(user.Id))
			req.Header.Set("X-Forwarded-For", "198.51.100.201")
			out := httptest.NewRecorder()
			engine.ServeHTTP(out, req)
			assert.Equal(t, http.StatusOK, out.Code)
			blocked, err := model.IsIPBlocked(ip)
			require.NoError(t, err)
			assert.False(t, blocked)
		})
	}
}

func TestSiteAccessBlacklistManagementPermissionsAndValidation(t *testing.T) {
	siteAccessFixture(t)
	common.IsMasterNode = true
	require.NoError(t, authz.Init(model.DB))
	engine := gin.New()
	engine.PUT("/blocks", func(c *gin.Context) { c.Set("id", 1); c.Set("role", common.RoleRootUser) }, middleware.RequirePermission(authz.SystemSettingsPermission("security.ssrf")), controller.UpdateSiteBlockedIPs)
	permission := authz.SystemSettingsPermission("security.ssrf")
	require.NoError(t, authz.SetUserPermissions(901, authz.PermissionsMap{permission.Resource: {permission.Action: false}}))
	for _, role := range []int{common.RoleCommonUser, common.RoleAdminUser} {
		req := httptest.NewRequest(http.MethodGet, "/blocks", nil)
		out := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(out)
		ctx.Request = req
		ctx.Set("id", 901)
		ctx.Set("role", role)
		middleware.RequirePermission(permission)(ctx)
		assert.Equal(t, http.StatusForbidden, out.Code)
	}

	req := httptest.NewRequest(http.MethodPut, "/blocks", strings.NewReader(`{"ips":["203.0.113.150"],"blocked":true}`))
	req.RemoteAddr = "203.0.113.150:1234"
	out := httptest.NewRecorder()
	engine.ServeHTTP(out, req)
	assert.Equal(t, http.StatusBadRequest, out.Code)
	for _, blocked := range []bool{true, false} {
		req = httptest.NewRequest(http.MethodPut, "/blocks", strings.NewReader(fmt.Sprintf(`{"ips":["203.0.113.151"],"blocked":%v}`, blocked)))
		req.RemoteAddr = "203.0.113.150:1234"
		out = httptest.NewRecorder()
		engine.ServeHTTP(out, req)
		assert.Equal(t, http.StatusOK, out.Code)
		value, err := model.IsIPBlocked("203.0.113.151")
		require.NoError(t, err)
		assert.Equal(t, blocked, value)
	}
	require.NoError(t, model.BlockIPAddresses([]string{"203.0.113.151", "203.0.113.152"}, 42, 1, "user disabled"))
	items, total, err := model.ListBlockedIPs("", 0, 1)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	assert.Len(t, items, 1)
	items, total, err = model.ListBlockedIPs("203.0.113.151", 0, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, "203.0.113.151", items[0].IP)
	var events int64
	require.NoError(t, model.LOG_DB.Model(&model.AuditLog{}).Where("action = ?", "site.ip.update").Count(&events).Error)
	assert.EqualValues(t, 2, events)

}

func TestSiteAccessCrossOriginCannotPoisonBlacklist(t *testing.T) {
	engine := siteAccessFixture(t)
	user := model.User{Username: "cross-origin", AffCode: "cross-origin", Password: "unused", Status: common.UserStatusDisabled, Role: common.RoleCommonUser}
	require.NoError(t, model.DB.Create(&user).Error)
	token := model.Token{UserId: user.Id, Key: "crossOriginSecret", Name: "fixture", Status: common.TokenStatusEnabled}
	require.NoError(t, model.DB.Create(&token).Error)
	for _, header := range []string{"Sec-Fetch-Site", "Origin", "Referer"} {
		req := httptest.NewRequest(http.MethodGet, "/v1beta/models?key=sk-"+token.Key, nil)
		req.RemoteAddr = "203.0.113.180:1234"
		value := "https://evil.invalid/"
		if header == "Sec-Fetch-Site" {
			value = "cross-site"
		}
		req.Header.Set(header, value)
		out := httptest.NewRecorder()
		engine.ServeHTTP(out, req)
		assert.Equal(t, http.StatusForbidden, out.Code)
		assert.Empty(t, out.Result().Cookies())
		blocked, err := model.IsIPBlocked("203.0.113.180")
		require.NoError(t, err)
		assert.False(t, blocked)
	}
}

func TestSiteAccessEnabledAndDeletedUsersDoNotBlockPublicPages(t *testing.T) {
	engine := siteAccessFixture(t)
	user := model.User{Username: "normal", AffCode: "normal", Password: "unused", Status: common.UserStatusEnabled, Role: common.RoleCommonUser}
	require.NoError(t, model.DB.Create(&user).Error)
	out := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(out)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	service.WriteSiteIdentityCookie(ctx, user.Id, uuid.NewString())
	marker := out.Result().Cookies()[0]
	for _, deleted := range []bool{false, true} {
		if deleted {
			require.NoError(t, model.DB.Delete(&user).Error)
		}
		req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
		req.RemoteAddr = "203.0.113.181:1234"
		req.AddCookie(marker)
		out := httptest.NewRecorder()
		engine.ServeHTTP(out, req)
		assert.Equal(t, http.StatusOK, out.Code)
		blocked, err := model.IsIPBlocked("203.0.113.181")
		require.NoError(t, err)
		assert.False(t, blocked)
	}
}

func TestSiteAccessDisabledPasswordRequiresVerifiedCredentials(t *testing.T) {
	siteAccessFixture(t)
	previousLogin, previousEncryption := common.PasswordLoginEnabled, common.PasswordLoginEncryptionEnabled
	common.PasswordLoginEnabled, common.PasswordLoginEncryptionEnabled = true, false
	t.Cleanup(func() {
		common.PasswordLoginEnabled, common.PasswordLoginEncryptionEnabled = previousLogin, previousEncryption
	})
	hash, err := common.Password2Hash("a-long-fixture-password")
	require.NoError(t, err)
	user := model.User{Username: "password-ban", AffCode: "password-ban", Password: hash, Status: common.UserStatusDisabled, Role: common.RoleCommonUser}
	require.NoError(t, model.DB.Create(&user).Error)
	engine := gin.New()
	require.NoError(t, engine.SetTrustedProxies(nil))
	engine.POST("/login", controller.Login)
	for i, password := range []string{"wrong", "a-long-fixture-password"} {
		ip := fmt.Sprintf("203.0.113.%d", 185+i)
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(fmt.Sprintf(`{"username":"password-ban","password":"%s"}`, password)))
		req.RemoteAddr = ip + ":1234"
		out := httptest.NewRecorder()
		engine.ServeHTTP(out, req)
		assert.Contains(t, out.Body.String(), "Username or password is incorrect, or user has been banned")
		blocked, err := model.IsIPBlocked(ip)
		require.NoError(t, err)
		assert.Equal(t, i == 1, blocked)
	}
}

func TestSiteAccessCookieExpiryAndAccountReenable(t *testing.T) {
	engine := siteAccessFixture(t)
	user := model.User{Username: "reenable", AffCode: "reenable", Status: common.UserStatusDisabled, Role: common.RoleCommonUser}
	require.NoError(t, model.DB.Create(&user).Error)
	markerOut := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(markerOut)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	service.WriteSiteIdentityCookie(ctx, user.Id, uuid.NewString())
	marker := markerOut.Result().Cookies()[0]
	// Expiry is checked before a marker is used, even if the remainder looks signed.
	parts := strings.Split(marker.Value, ".")
	parts[2] = "1"
	expired := *marker
	expired.Value = strings.Join(parts, ".")
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	req.RemoteAddr = "203.0.113.190:1234"
	req.AddCookie(&expired)
	out := httptest.NewRecorder()
	engine.ServeHTTP(out, req)
	assert.Equal(t, http.StatusOK, out.Code)
	require.NoError(t, model.DB.Model(&user).Update("status", common.UserStatusEnabled).Error)
	blocked, err := model.BlockDisabledUserRequest(user.Id, "203.0.113.191", "")
	require.NoError(t, err)
	assert.False(t, blocked)
	req = httptest.NewRequest(http.MethodGet, "/api/status", nil)
	req.RemoteAddr = "203.0.113.191:1234"
	req.AddCookie(marker)
	out = httptest.NewRecorder()
	engine.ServeHTTP(out, req)
	assert.Equal(t, http.StatusOK, out.Code)
}

type siteAccessTurnstileTransport struct {
	success bool
	calls   int
}

func (transport *siteAccessTurnstileTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	transport.calls++
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"success":%v}`, transport.success)))}, nil
}
func TestSiteAccessExistingTurnstileRejectsMissingInvalidAndReusedTokens(t *testing.T) {
	previousEnabled, previousClient := common.TurnstileCheckEnabled, http.DefaultClient
	common.TurnstileCheckEnabled = true
	transport := &siteAccessTurnstileTransport{}
	http.DefaultClient = &http.Client{Transport: transport}
	t.Cleanup(func() { common.TurnstileCheckEnabled, http.DefaultClient = previousEnabled, previousClient })
	engine := gin.New()
	engine.POST("/verify", middleware.TurnstileCheck(), func(c *gin.Context) { c.String(http.StatusOK, "verified") })
	for _, path := range []string{"/verify", "/verify?turnstile=invalid", "/verify?turnstile=reused"} {
		out := httptest.NewRecorder()
		engine.ServeHTTP(out, httptest.NewRequest(http.MethodPost, path, nil))
		assert.NotEqual(t, "verified", out.Body.String())
		assert.Contains(t, out.Body.String(), `"success":false`)
	}
	assert.Equal(t, 2, transport.calls)
	transport.success = true
	out := httptest.NewRecorder()
	engine.ServeHTTP(out, httptest.NewRequest(http.MethodPost, "/verify?turnstile=valid", nil))
	assert.Equal(t, "verified", out.Body.String())
}

func TestSiteAccessLoginAndRefreshIssueVerifiedDeviceMarker(t *testing.T) {
	siteAccessFixture(t)
	require.NoError(t, model.DB.AutoMigrate(&model.TwoFA{}, &model.PasskeyCredential{}, &model.AuthFlow{}))
	previousLogin, previousEncryption := common.PasswordLoginEnabled, common.PasswordLoginEncryptionEnabled
	common.PasswordLoginEnabled, common.PasswordLoginEncryptionEnabled = true, false
	t.Cleanup(func() {
		common.PasswordLoginEnabled, common.PasswordLoginEncryptionEnabled = previousLogin, previousEncryption
	})
	hash, err := common.Password2Hash("site-login-fixture-password")
	require.NoError(t, err)
	user := model.User{Username: "site-login", AffCode: "site-login", Password: hash, Status: common.UserStatusEnabled, Role: common.RoleCommonUser, AuthVersion: 1, Group: "default"}
	require.NoError(t, model.DB.Create(&user).Error)
	engine := gin.New()
	require.NoError(t, engine.SetTrustedProxies(nil))
	engine.Use(middleware.SiteAccessGuard())
	engine.POST("/api/user/login", controller.Login)
	engine.POST("/api/user/auth/refresh", controller.RefreshAuth)
	engine.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "home") })
	login := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(`{"username":"site-login","password":"site-login-fixture-password"}`))
	login.RemoteAddr = "203.0.113.210:1234"
	out := httptest.NewRecorder()
	engine.ServeHTTP(out, login)
	require.Equal(t, http.StatusOK, out.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(out.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.NotEmpty(t, response.Data.AccessToken)
	var refresh, marker *http.Cookie
	for _, cookie := range out.Result().Cookies() {
		if cookie.Name == service.RefreshCookieName {
			refresh = cookie
		}
		if cookie.Name == service.SiteIdentityCookieName {
			marker = cookie
		}
	}
	require.NotNil(t, refresh)
	require.NotNil(t, marker)
	markerReq := httptest.NewRequest(http.MethodGet, "/", nil)
	markerReq.AddCookie(marker)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = markerReq
	uid, device := service.ReadSiteIdentityCookie(ctx)
	require.Equal(t, user.Id, uid)
	require.NotEmpty(t, device)
	request := httptest.NewRequest(http.MethodPost, "/api/user/auth/refresh", nil)
	request.RemoteAddr = login.RemoteAddr
	request.AddCookie(refresh)
	out = httptest.NewRecorder()
	engine.ServeHTTP(out, request)
	require.Equal(t, http.StatusOK, out.Code)
	response.Success = false
	require.NoError(t, common.Unmarshal(out.Body.Bytes(), &response))
	require.True(t, response.Success)
	var refreshedMarker *http.Cookie
	for _, cookie := range out.Result().Cookies() {
		if cookie.Name == service.SiteIdentityCookieName {
			refreshedMarker = cookie
		}
	}
	require.NotNil(t, refreshedMarker)
	assert.True(t, refreshedMarker.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, refreshedMarker.SameSite)
	user.Status = common.UserStatusDisabled
	require.NoError(t, model.DisableUserAndBlockIPs(&user, nil, 1))
	request = httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "203.0.113.211:1234"
	request.AddCookie(refreshedMarker)
	out = httptest.NewRecorder()
	engine.ServeHTTP(out, request)
	assert.Equal(t, http.StatusForbidden, out.Code)
	blocked, err := model.IsIPBlocked("203.0.113.211")
	require.NoError(t, err)
	assert.True(t, blocked)
}
