package controller

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestMediaAssetListFiltersDatabaseMatrix(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			var driver gorm.Dialector
			switch dialect {
			case "sqlite":
				driver = sqlite.Open(filepath.Join(t.TempDir(), "media.db"))
			case "mysql":
				dsn := os.Getenv("MEDIA_ASSET_TEST_MYSQL_DSN")
				if dsn == "" {
					t.Skip("MEDIA_ASSET_TEST_MYSQL_DSN is not configured")
				}
				driver = mysql.Open(dsn)
			case "postgres":
				dsn := os.Getenv("MEDIA_ASSET_TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("MEDIA_ASSET_TEST_POSTGRES_DSN is not configured")
				}
				driver = postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true})
			}
			db, err := gorm.Open(driver, &gorm.Config{})
			require.NoError(t, err)
			versionQuery := "SELECT version()"
			if dialect == "sqlite" {
				versionQuery = "SELECT sqlite_version()"
			}
			var version string
			require.NoError(t, db.Raw(versionQuery).Scan(&version).Error)
			t.Logf("database: %s %s", dialect, version)
			connection, err := db.DB()
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, connection.Close()) })
			previous := model.DB
			previousMaster := common.IsMasterNode
			common.IsMasterNode = true
			t.Cleanup(func() { common.IsMasterNode = previousMaster })
			model.DB = db
			t.Cleanup(func() { model.DB = previous })
			require.NoError(t, db.AutoMigrate(&model.User{}, &model.MediaAsset{}, &model.CasbinRule{}, &model.AuthzRole{}))
			require.NoError(t, authz.Init(db))
			require.NoError(t, db.Create(&[]model.User{{Id: 7, Username: "asset_alice", Role: common.RoleAdminUser, AffCode: "assetalice"}, {Id: 8, Username: "asset_bob", AffCode: "assetbob"}}).Error)
			require.NoError(t, db.Create(&[]model.MediaAsset{
				{AssetID: "alice_before", UserID: 7, SourceType: "public", MediaType: "image", CreatedAt: 99},
				{AssetID: "alice_start", UserID: 7, SourceType: "public", MediaType: "image", CreatedAt: 100},
				{AssetID: "alice_end", UserID: 7, SourceType: "public", MediaType: "image", CreatedAt: 200},
				{AssetID: "alice_after", UserID: 7, SourceType: "public", MediaType: "image", CreatedAt: 201},
				{AssetID: "bob_middle", UserID: 8, SourceType: "public", MediaType: "video", CreatedAt: 150},
			}).Error)
			cases := []struct {
				name, query string
				role        int
				ids         []string
				total       int64
			}{
				{"admin combined inclusive bounds", "start_time=100&end_time=200&username=alice", common.RoleAdminUser, []string{"alice_end", "alice_start"}, 2},
				{"admin all users", "start_time=100&end_time=200", common.RoleAdminUser, []string{"alice_end", "bob_middle", "alice_start"}, 3},
				{"no matching username", "username=missing", common.RoleAdminUser, []string{}, 0},
				{"pagination and filtered total", "start_time=100&end_time=200&username=alice&page=2&page_size=1", common.RoleAdminUser, []string{"alice_start"}, 2},
				{"unfiltered backward compatibility", "", common.RoleAdminUser, []string{"alice_after", "alice_end", "bob_middle", "alice_start", "alice_before"}, 5},
				{"user cannot widen scope with username", "view_scope=all&start_time=100&end_time=200&username=bob", common.RoleCommonUser, []string{"alice_end", "alice_start"}, 2},
				{"admin self view", "view_scope=self&start_time=100&end_time=200&username=bob", common.RoleAdminUser, []string{"alice_end", "alice_start"}, 2},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					recorder := httptest.NewRecorder()
					context, _ := gin.CreateTestContext(recorder)
					context.Set("id", 7)
					context.Set("role", tc.role)
					context.Request = httptest.NewRequest(http.MethodGet, "/api/media-assets?"+tc.query, nil)
					ListMediaAssets(context)
					require.Equal(t, http.StatusOK, recorder.Code)
					var result struct {
						Success bool
						Data    struct {
							Items []model.MediaAsset
							Total int64
						}
					}
					require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &result))
					require.True(t, result.Success)
					assert.Equal(t, tc.total, result.Data.Total)
					ids := make([]string, 0, len(result.Data.Items))
					for _, asset := range result.Data.Items {
						ids = append(ids, asset.AssetID)
						if tc.role == common.RoleCommonUser || tc.name == "admin self view" {
							assert.Equal(t, 7, asset.UserID)
							assert.Empty(t, asset.Username)
						} else {
							assert.NotEmpty(t, asset.Username)
						}
					}
					assert.Equal(t, tc.ids, ids)
				})
			}
		})
	}
}
