package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

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
			require.NoError(t, db.AutoMigrate(&model.User{}, &model.CasbinRule{}, &model.AuthzRole{}, &model.Task{}, &model.Midjourney{}, &model.MediaAsset{}, &model.ImageGenerationLog{}, &model.Channel{}))
			require.NoError(t, logDB.AutoMigrate(&model.AuditLog{}))
			require.NoError(t, authz.Init(db))
			users := []model.User{
				{Id: 1, Username: "scope_root", Role: common.RoleRootUser, AffCode: "scope1"},
				{Id: 2, Username: "scope_admin", Role: common.RoleAdminUser, AffCode: "scope2"},
				{Id: 3, Username: "scope_user", Role: common.RoleCommonUser, AffCode: "scope3"},
				{Id: 4, Username: "scope_promoted_root", Role: common.RoleRootUser, AffCode: "scope4"},
			}
			require.NoError(t, db.Create(&users).Error)
			rootFile := filepath.Join(dir, "root-image")
			require.NoError(t, os.WriteFile(rootFile, []byte("root-file-canary"), 0600))
			for _, u := range users {
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
				{"/api/mj/", authz.DrawingLogsRead}, {"/api/task/", authz.TaskLogsRead},
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
	body := fmt.Sprintf(`{"id":%d,"username":"managed-user","group":"default","admin_permissions":{"task_logs":{"task_read":false},"privacy":{"hide_root":true}}}`, target.Id)
	response := adminUserRequest(http.MethodPut, "/api/user/", body, proof, identity, common.RoleRootUser, nil, UpdateUser)
	var result securityEnrollmentResponse
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
	require.True(t, result.Success, response.Body.String())
	_, _, err = service.ValidateLoginSession(oldIdentity)
	assert.ErrorIs(t, err, service.ErrLoginSessionRevoked)
	assert.True(t, authz.RootHidden(target.Id, common.RoleAdminUser))
	assert.False(t, authz.Can(target.Id, common.RoleAdminUser, authz.TaskLogsRead))
	response = adminUserRequest(http.MethodPut, "/api/user/", body, proof, identity, common.RoleRootUser, nil, UpdateUser)
	assert.Equal(t, http.StatusForbidden, response.Code)
}
