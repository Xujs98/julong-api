package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestAdminLogPermissionsMatrix(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			dir := t.TempDir()
			var mainDriver, logDriver gorm.Dialector
			databaseType := common.DatabaseTypeSQLite
			switch dialect {
			case "sqlite":
				mainDriver, logDriver = sqlite.Open(filepath.Join(dir, "main.db")), sqlite.Open(filepath.Join(dir, "logs.db"))
			case "mysql":
				mainDSN, logDSN := os.Getenv("LOG_PERMISSIONS_MYSQL_DSN"), os.Getenv("LOG_PERMISSIONS_MYSQL_LOG_DSN")
				if mainDSN == "" || logDSN == "" {
					t.Skip("dedicated MySQL test DSNs are not configured")
				}
				mainDriver, logDriver = mysql.Open(mainDSN), mysql.Open(logDSN)
				databaseType = common.DatabaseTypeMySQL
			case "postgres":
				mainDSN, logDSN := os.Getenv("LOG_PERMISSIONS_POSTGRES_DSN"), os.Getenv("LOG_PERMISSIONS_POSTGRES_LOG_DSN")
				if mainDSN == "" || logDSN == "" {
					t.Skip("dedicated PostgreSQL test DSNs are not configured")
				}
				mainDriver, logDriver = postgres.Open(mainDSN), postgres.Open(logDSN)
				databaseType = common.DatabaseTypePostgreSQL
			}
			db, err := gorm.Open(mainDriver, &gorm.Config{})
			require.NoError(t, err)
			logDB, err := gorm.Open(logDriver, &gorm.Config{})
			require.NoError(t, err)
			oldDB, oldLogDB := model.DB, model.LOG_DB
			oldMainType, oldLogType := common.MainDatabaseType(), common.LogDatabaseType()
			oldMaster, oldRedis := common.IsMasterNode, common.RedisEnabled
			model.DB, model.LOG_DB = db, logDB
			common.SetDatabaseTypes(databaseType, databaseType)
			common.IsMasterNode, common.RedisEnabled = true, false
			t.Cleanup(func() {
				model.DB, model.LOG_DB = oldDB, oldLogDB
				common.SetDatabaseTypes(oldMainType, oldLogType)
				common.IsMasterNode, common.RedisEnabled = oldMaster, oldRedis
				for _, connection := range []*gorm.DB{db, logDB} {
					sqlDB, e := connection.DB()
					require.NoError(t, e)
					require.NoError(t, sqlDB.Close())
				}
			})
			query := "SELECT version()"
			if dialect == "sqlite" {
				query = "SELECT sqlite_version()"
			}
			var version string
			require.NoError(t, db.Raw(query).Scan(&version).Error)
			t.Logf("database %s %s; independent log database", dialect, version)
			require.NoError(t, i18n.Init())
			require.NoError(t, db.AutoMigrate(&model.User{}, &model.CasbinRule{}, &model.AuthzRole{}, &model.Task{}, &model.Midjourney{}, &model.MediaAsset{}, &model.ImageGenerationLog{}, &model.Channel{}, &model.UserPresence{}))
			require.NoError(t, logDB.AutoMigrate(&model.AuditLog{}, &model.Log{}))
			require.NoError(t, authz.Init(db))
			users := []model.User{
				{Id: 1, Username: "scope_root", Role: common.RoleRootUser, AffCode: "scope1"},
				{Id: 2, Username: "scope_admin", Role: common.RoleAdminUser, AffCode: "scope2"},
				{Id: 3, Username: "scope_user", Role: common.RoleCommonUser, AffCode: "scope3"},
				{Id: 4, Username: "scope_promoted_root", Role: common.RoleRootUser, AffCode: "scope4"},
			}
			require.NoError(t, db.Create(&users).Error)
			require.NoError(t, db.Model(&model.User{}).Where("id > ?", 0).Update("status", common.UserStatusEnabled).Error)
			rootFile := filepath.Join(dir, "root-image")
			require.NoError(t, os.WriteFile(rootFile, []byte("root-file-canary"), 0600))
			for _, u := range users {
				require.NoError(t, logDB.Create(&model.Log{UserId: u.Id, Username: u.Username, Type: model.LogTypeConsume, CreatedAt: time.Now().Unix(), Quota: u.Id * 100, PromptTokens: u.Id, CompletionTokens: u.Id * 2}).Error)
				require.NoError(t, db.Create(&model.Task{UserId: u.Id, TaskID: fmt.Sprintf("scope_task_%d", u.Id), Status: model.TaskStatusSubmitted, SubmitTime: int64(u.Id)}).Error)
				require.NoError(t, db.Create(&model.Midjourney{UserId: u.Id, MjId: fmt.Sprintf("scope_mj_%d", u.Id), SubmitTime: int64(u.Id)}).Error)
				storedPath := ""
				if u.Id == 1 {
					storedPath = rootFile
				}
				require.NoError(t, db.Create(&model.MediaAsset{UserID: u.Id, AssetID: fmt.Sprintf("scope_asset_%d", u.Id), SourceType: model.MediaAssetSourcePublic, MediaType: "image", StoredPath: storedPath, CreatedAt: int64(u.Id)}).Error)
				require.NoError(t, db.Create(&model.ImageGenerationLog{Id: u.Id, UserId: u.Id, Username: u.Username, TaskId: fmt.Sprintf("scope_image_%d", u.Id), Status: "submitted", Images: "[]"}).Error)
				actorRole := u.Role
				if u.Id == 4 {
					actorRole = common.RoleCommonUser
				} // Root's records before promotion must also be hidden.
				require.NoError(t, logDB.Create(&model.AuditLog{UserId: u.Id, Username: u.Username, ActorRole: actorRole, EventId: fmt.Sprintf("scope_event_%d", u.Id), CreatedAt: int64(u.Id), Category: model.AuditCategoryOperation}).Error)
			}
			viewerID, viewerRole := 2, common.RoleAdminUser
			router := gin.New()
			router.Use(func(c *gin.Context) { c.Set("id", viewerID); c.Set("role", viewerRole); c.Next() })
			router.GET("/api/log/", GetAllLogs)
			router.GET("/api/log/stat", GetLogsStat)
			router.GET("/api/log/export", ExportAllLogs)
			router.GET("/api/log/export/stream", ExportAllLogsStream)
			router.GET("/api/user/", GetAllUsers)
			router.GET("/api/user/search", SearchUsers)
			router.GET("/api/mj/", GetAllMidjourney)
			router.GET("/api/task/", GetAllTask)
			router.GET("/api/image-generation-logs", GetImageGenerationLogs)
			router.GET("/api/media-assets", ListMediaAssets)
			router.GET("/api/audit", GetAuditLogs)
			router.GET("/api/task/:task_id/detail", GetDashboardTaskDetail)
			router.GET("/api/task/:task_id/request", GetDashboardTaskRequest)
			router.GET("/api/task/:task_id/artifacts", GetDashboardTaskArtifacts)
			router.GET("/api/image-generation-logs/:id/task", GetImageGenerationLogTask)
			router.GET("/api/image-generation-logs/:id/images/:index", GetImageGenerationLogImage)
			router.GET("/api/user/:id", GetUser)
			router.GET("/api/subscription/admin/users/:id", AdminListUserSubscriptions)
			router.DELETE("/api/media-assets/:asset_id", DeleteMediaAsset)
			request := func(method, path string) *httptest.ResponseRecorder {
				response := httptest.NewRecorder()
				router.ServeHTTP(response, httptest.NewRequest(method, path, nil))
				return response
			}
			lists := []struct {
				path       string
				permission authz.Permission
			}{
				{"/api/log/", authz.UsageLogsRead}, {"/api/mj/", authz.DrawingLogsRead}, {"/api/task/", authz.TaskLogsRead},
				{"/api/image-generation-logs", authz.ImageLogsRead}, {"/api/media-assets", authz.MediaAssetsRead}, {"/api/audit", authz.AuditRead},
			}
			require.NoError(t, authz.SetUserPermissions(2, authz.PermissionsMap{"audit": {"read": true}}))
			for _, list := range lists {
				t.Run(list.permission.Action+list.permission.Resource, func(t *testing.T) {
					response := request(http.MethodGet, list.path+"?page_size=20")
					var result struct {
						Success bool
						Data    struct {
							Total int
							Items []struct {
								UserID int `json:"user_id"`
							}
						}
					}
					require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
					require.True(t, result.Success, response.Body.String())
					expected := 4
					if list.permission == authz.AuditRead {
						expected = 3
					} // Existing actor-role protection remains in force.
					assert.Equal(t, expected, result.Data.Total)
					require.NoError(t, authz.SetUserPermissions(2, authz.PermissionsMap{"privacy": {"hide_root": true}}))
					var ids []int
					for page := 1; page <= 2; page++ {
						response = request(http.MethodGet, list.path+fmt.Sprintf("?p=%d&page=%d&page_size=1", page, page))
						require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
						require.True(t, result.Success, response.Body.String())
						require.Equal(t, 2, result.Data.Total)
						require.Len(t, result.Data.Items, 1)
						ids = append(ids, result.Data.Items[0].UserID)
					}
					assert.ElementsMatch(t, []int{2, 3}, ids)
					require.NoError(t, authz.SetUserPermissions(2, authz.PermissionsMap{list.permission.Resource: {list.permission.Action: false}}))
					response = request(http.MethodGet, list.path+"?page_size=20")
					require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
					require.True(t, result.Success, response.Body.String())
					assert.Equal(t, 1, result.Data.Total)
					require.Len(t, result.Data.Items, 1)
					assert.Equal(t, 2, result.Data.Items[0].UserID)
					// Restore one grant, leaving the privacy override enabled, then verify persistence through reseeding.
					require.NoError(t, authz.SetUserPermissions(2, authz.PermissionsMap{list.permission.Resource: {list.permission.Action: true}, "privacy": {"hide_root": false}}))
				})
			}
			require.NoError(t, authz.SetUserPermissions(2, authz.PermissionsMap{"privacy": {"hide_root": true}}))
			require.NoError(t, authz.Init(db))
			require.True(t, authz.RootHidden(2, common.RoleAdminUser))
			for _, path := range []string{"/api/task/scope_task_1/detail", "/api/task/scope_task_1/request", "/api/task/scope_task_1/artifacts"} {
				assert.Equal(t, http.StatusNotFound, request(http.MethodGet, path).Code, path)
			}
			for _, path := range []string{"/api/image-generation-logs/1/task", "/api/image-generation-logs/1/images/0", "/api/subscription/admin/users/1"} {
				assert.Equal(t, http.StatusForbidden, request(http.MethodGet, path).Code, path)
			}
			assert.Equal(t, http.StatusForbidden, request(http.MethodDelete, "/api/media-assets/scope_asset_1").Code)
			bytes, err := os.ReadFile(rootFile)
			require.NoError(t, err)
			assert.Equal(t, "root-file-canary", string(bytes))
			assert.Contains(t, request(http.MethodGet, "/api/user/1").Body.String(), `"success":false`)
			assert.Equal(t, http.StatusOK, request(http.MethodGet, "/api/task/scope_task_2/detail").Code)
			require.NoError(t, authz.SetUserPermissions(2, authz.PermissionsMap{"task_logs": {"task_read": false, "image_read": false}}))
			assert.Equal(t, http.StatusNotFound, request(http.MethodGet, "/api/task/scope_task_3/request").Code)
			assert.Equal(t, http.StatusForbidden, request(http.MethodGet, "/api/image-generation-logs/3/task").Code)
			// Ordinary accounts cannot acquire a cross-account view by carrying an override.
			assert.False(t, authz.CanViewLogOwner(3, common.RoleCommonUser, 2, authz.TaskLogsRead))
			viewerID, viewerRole = 1, common.RoleRootUser
			assert.False(t, authz.RootHidden(viewerID, viewerRole))
			for _, list := range lists {
				var result struct {
					Success bool
					Data    struct{ Total int }
				}
				response := request(http.MethodGet, list.path+"?page_size=20")
				require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
				require.True(t, result.Success)
				assert.Equal(t, 4, result.Data.Total)
			}
			t.Run("usage_exports_and_statistics", func(t *testing.T) {
				for _, tc := range []struct {
					name        string
					viewer      int
					role        int
					permissions authz.PermissionsMap
					ids         []int
					quota       int
				}{
					{"default_admin", 2, common.RoleAdminUser, authz.PermissionsMap{"privacy": {"hide_root": false}}, []int{1, 2, 3, 4}, 1000},
					{"private_admin", 2, common.RoleAdminUser, authz.PermissionsMap{"privacy": {"hide_root": true}}, []int{2, 3}, 500},
					{"self_only", 2, common.RoleAdminUser, authz.PermissionsMap{"usage_logs": {"read": false}}, []int{2}, 200},
					{"root", 1, common.RoleRootUser, nil, []int{1, 2, 3, 4}, 1000},
				} {
					t.Run(tc.name, func(t *testing.T) {
						viewerID, viewerRole = tc.viewer, tc.role
						if tc.permissions != nil {
							require.NoError(t, authz.SetUserPermissions(2, tc.permissions))
						}
						var stat struct {
							Success bool
							Data    model.Stat
						}
						response := request(http.MethodGet, "/api/log/stat")
						require.NoError(t, common.Unmarshal(response.Body.Bytes(), &stat))
						require.True(t, stat.Success, response.Body.String())
						assert.Equal(t, tc.quota, stat.Data.Quota)
						assert.Equal(t, len(tc.ids), stat.Data.Rpm)
						assert.Equal(t, tc.quota*3/100, stat.Data.Tpm)
						for _, path := range []string{"/api/log/export", "/api/log/export/stream"} {
							response = request(http.MethodGet, path)
							require.Equal(t, http.StatusOK, response.Code, response.Body.String())
							assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
							for _, u := range users {
								allowed := false
								for _, id := range tc.ids {
									if id == u.Id {
										allowed = true
									}
								}
								if allowed {
									assert.Contains(t, response.Body.String(), u.Username)
								} else {
									assert.NotContains(t, response.Body.String(), u.Username)
								}
							}
						}
						if tc.viewer == 2 {
							response = request(http.MethodGet, "/api/log/?username=scope_root")
							if len(tc.ids) < 4 {
								assert.Contains(t, response.Body.String(), `"total":0`)
							}
						}
					})
				}
			})
			t.Run("presence_and_root_privacy", func(t *testing.T) {
				viewerID, viewerRole = 1, common.RoleRootUser
				now := time.Date(2026, time.October, 16, 16, 1, 0, 0, time.UTC) // 00:01 on Oct 17 in UTC+8
				start := now.Unix() - 60
				extra := []model.User{
					{Id: 5, Username: "presence_never", AffCode: "presence5", Status: common.UserStatusEnabled},
					{Id: 6, Username: "presence_disabled", AffCode: "presence6", Status: common.UserStatusDisabled},
					{Id: 7, Username: "presence_deleted", AffCode: "presence7", Status: common.UserStatusEnabled, DeletedAt: gorm.DeletedAt{Time: now, Valid: true}},
				}
				require.NoError(t, db.Create(&extra).Error)
				for id, at := range map[int]int64{1: now.Unix(), 2: now.Unix() - 10, 3: now.Unix() - 300, 4: now.Unix(), 6: now.Unix(), 7: now.Unix()} {
					require.NoError(t, db.Create(&model.UserPresence{UserId: id, LastActivityAt: at, IsOnline: false}).Error)
				}
				summary, err := model.GetUserActivitySummary(now)
				require.NoError(t, err)
				assert.Equal(t, start, summary.DayStart)
				assert.EqualValues(t, 2, summary.DailyActiveUsers)
				assert.EqualValues(t, 1, summary.OnlineUsers)
				for _, tc := range []struct {
					state string
					ids   []int
				}{{"online", []int{2}}, {"offline", []int{3, 5, 6, 7}}} {
					var ids []int
					require.NoError(t, (model.UserSortOptions{Presence: tc.state}).ApplyPresence(db.Unscoped().Model(&model.User{}), now.Unix()).Pluck("id", &ids).Error)
					assert.ElementsMatch(t, tc.ids, ids)
				}
				var loaded []*model.User
				require.NoError(t, db.Unscoped().Find(&loaded).Error)
				require.NoError(t, model.FillUserOnlineStatus(loaded, now.Unix()))
				for _, u := range loaded {
					if u.Role == common.RoleRootUser {
						assert.Nil(t, u.IsOnline)
						data, e := common.Marshal(u)
						require.NoError(t, e)
						assert.NotContains(t, string(data), "is_online")
					} else {
						require.NotNil(t, u.IsOnline)
						assert.Equal(t, u.Id == 2, *u.IsOnline)
					}
				}
				// Verify live HTTP list/search pagination and Root field omission for both viewers.
				require.NoError(t, db.Model(&model.UserPresence{}).Where("user_id > ?", 0).Update("last_activity_at", time.Now().Unix()-600).Error)
				require.NoError(t, db.Model(&model.UserPresence{}).Where("user_id = ?", 2).Update("last_activity_at", time.Now().Unix()).Error)
				for _, role := range []int{common.RoleAdminUser, common.RoleRootUser} {
					viewerID, viewerRole = 2, role
					for _, path := range []string{"/api/user/?presence=online", "/api/user/search?status=online"} {
						response := request(http.MethodGet, path)
						var result struct {
							Success bool
							Data    struct {
								Total int
								Items []model.User
							}
						}
						require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
						require.True(t, result.Success, response.Body.String())
						assert.Equal(t, 1, result.Data.Total)
						require.Len(t, result.Data.Items, 1)
						assert.Equal(t, 2, result.Data.Items[0].Id)
						require.NotNil(t, result.Data.Items[0].IsOnline)
						assert.True(t, *result.Data.Items[0].IsOnline)
					}
					response := request(http.MethodGet, "/api/user/?page_size=20")
					assert.NotContains(t, response.Body.String(), `"is_online":null`)
					var result struct {
						Data struct{ Items []map[string]any }
					}
					require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
					for _, u := range result.Data.Items {
						if u["role"] == float64(common.RoleRootUser) {
							_, exists := u["is_online"]
							assert.False(t, exists)
						}
					}
					for _, id := range []int{1, 4} {
						response = request(http.MethodGet, fmt.Sprintf("/api/user/%d", id))
						assert.NotContains(t, response.Body.String(), "is_online")
					}
				}
				assert.True(t, strings.Contains(request(http.MethodGet, "/api/user/search?status=offline").Body.String(), `"success":true`))
			})
			// A failed Root lookup must not turn into an unrestricted query or detail response.
			viewerID, viewerRole = 2, common.RoleAdminUser
			isolated := model.DB
			unavailable, err := gorm.Open(sqlite.Open(filepath.Join(dir, "unavailable.db")), &gorm.Config{})
			require.NoError(t, err)
			model.DB = unavailable
			_, err = authz.LogScope(2, common.RoleAdminUser, authz.MediaAssetsRead)
			assert.Error(t, err)
			assert.False(t, authz.CanViewLogOwner(2, common.RoleAdminUser, 1, authz.MediaAssetsRead))
			model.DB = isolated
			connection, err := unavailable.DB()
			require.NoError(t, err)
			require.NoError(t, connection.Close())
		})
	}
}

func TestAdminLogPermissionUpdateRevokesSessions(t *testing.T) {
	_, identity, target := setupAdminUserTest(t)
	require.NoError(t, model.DB.Model(target).Update("role", common.RoleAdminUser).Error)
	require.NoError(t, model.DB.AutoMigrate(&model.CasbinRule{}, &model.AuthzRole{}))
	require.NoError(t, authz.Init(model.DB))
	bundle, err := service.CreateLoginSession(target.Id, "password", "127.0.0.1", "permissions-test")
	require.NoError(t, err)
	oldIdentity, err := service.ParseAccessToken(bundle.AccessToken)
	require.NoError(t, err)
	_, _, err = service.ValidateLoginSession(oldIdentity)
	require.NoError(t, err)
	operation := service.VerificationOperation{Scope: service.VerificationScopeAdminUserUpdate, Context: []byte(fmt.Sprintf(`{"user_id":%d}`, target.Id))}
	proof := issueSecurityEnrollmentProof(t, identity, operation, service.VerificationMethodPassword)
	body := fmt.Sprintf(`{"id":%d,"username":"managed-user","group":"default","admin_permissions":{"usage_logs":{"read":false},"task_logs":{"task_read":false},"privacy":{"hide_root":true}}}`, target.Id)
	response := adminUserRequest(http.MethodPut, "/api/user/", body, proof, identity, common.RoleRootUser, nil, UpdateUser)
	var result securityEnrollmentResponse
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
	require.True(t, result.Success, response.Body.String())
	_, _, err = service.ValidateLoginSession(oldIdentity)
	assert.ErrorIs(t, err, service.ErrLoginSessionRevoked)
	assert.True(t, authz.RootHidden(target.Id, common.RoleAdminUser))
	assert.False(t, authz.Can(target.Id, common.RoleAdminUser, authz.TaskLogsRead))
	assert.False(t, authz.Can(target.Id, common.RoleAdminUser, authz.UsageLogsRead))
	response = adminUserRequest(http.MethodPut, "/api/user/", body, proof, identity, common.RoleRootUser, nil, UpdateUser)
	assert.Equal(t, http.StatusForbidden, response.Code)
}
