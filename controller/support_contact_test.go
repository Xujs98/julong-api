package controller

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func saveSupportFixture(t *testing.T, contacts []SupportContact, files map[string][]byte) (bool, []SupportContact) {
	t.Helper()
	var body bytes.Buffer
	contentType := "application/json"
	if files == nil {
		data, err := common.Marshal(updateSupportContactsRequest{Contacts: contacts})
		require.NoError(t, err)
		body.Write(data)
	} else {
		writer := multipart.NewWriter(&body)
		data, err := common.Marshal(contacts)
		require.NoError(t, err)
		require.NoError(t, writer.WriteField("contacts", string(data)))
		for id, data := range files {
			part, err := writer.CreateFormFile("qrcode_"+id, "untrusted.svg")
			require.NoError(t, err)
			_, err = part.Write(data)
			require.NoError(t, err)
		}
		require.NoError(t, writer.Close())
		contentType = writer.FormDataContentType()
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/support-contacts", &body)
	c.Request.Header.Set("Content-Type", contentType)
	UpdateSupportContacts(c)
	var response struct {
		Success bool
		Data    []SupportContact
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response), recorder.Body.String())
	return response.Success, response.Data
}

func TestSupportContactLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			var driver gorm.Dialector
			dbType := common.DatabaseTypeSQLite
			switch dialect {
			case "sqlite":
				driver = sqlite.Open(filepath.Join(t.TempDir(), "support.db"))
			case "mysql":
				dsn := os.Getenv("SUPPORT_TEST_MYSQL")
				if dsn == "" {
					t.Skip("dedicated MySQL test database not configured")
				}
				driver = mysql.Open(dsn)
				dbType = common.DatabaseTypeMySQL
			case "postgres":
				dsn := os.Getenv("SUPPORT_TEST_POSTGRES")
				if dsn == "" {
					t.Skip("dedicated PostgreSQL test database not configured")
				}
				driver = postgres.Open(dsn)
				dbType = common.DatabaseTypePostgreSQL
			}
			db, err := gorm.Open(driver, &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			defer sqlDB.Close()
			oldDB, oldType, oldLogType := model.DB, common.MainDatabaseType(), common.LogDatabaseType()
			oldContacts, oldMap := common.SupportContacts, common.OptionMap
			model.DB = db

			var version string
			query := "SELECT version()"
			if dialect == "sqlite" {
				query = "SELECT sqlite_version()"
			}
			require.NoError(t, db.Raw(query).Scan(&version).Error)
			t.Logf("database version: %s", version)
			common.SetDatabaseTypes(dbType, dbType)
			common.OptionMap = map[string]string{}
			common.SupportContacts = "[]"
			defer func() {
				model.DB = oldDB
				common.SetDatabaseTypes(oldType, oldLogType)
				common.SupportContacts = oldContacts
				common.OptionMap = oldMap
			}()
			require.NoError(t, db.AutoMigrate(&model.Option{}))
			dir := t.TempDir()
			t.Setenv("SITE_ASSET_STORAGE_DIR", dir)
			contacts := []SupportContact{{Id: "qq", Type: "qq", Label: "Legacy", Value: "12345"}, {Id: "group", Type: "qq_group", Label: "QQ help", Value: "67890"}, {Id: "wechat", Type: "wechat_group", Label: "WeChat help", Value: "Support"}}
			// Legacy data and both new group types persist through the existing Option contract.
			ok, saved := saveSupportFixture(t, contacts, map[string][]byte{"group": buildSiteLogoPNG(t, 16, 16), "wechat": buildSiteLogoPNG(t, 32, 32)})
			require.True(t, ok)
			require.Len(t, saved, 3)
			assert.Empty(t, saved[0].QRCodeURL)
			require.NotEmpty(t, saved[1].QRCodeURL)
			require.NotEmpty(t, saved[2].QRCodeURL)
			var option model.Option
			require.NoError(t, db.Where(&model.Option{Key: "SupportContacts"}).First(&option).Error)
			var stored []SupportContact
			require.NoError(t, common.UnmarshalJsonStr(option.Value, &stored))
			assert.Equal(t, saved, stored)
			// Public image bytes, MIME and deletion-aware caching.
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, saved[1].QRCodeURL, nil)
			c.Params = gin.Params{{Key: "filename", Value: strings.TrimPrefix(saved[1].QRCodeURL, supportQRPrefix)}}
			GetSupportQRCode(c)
			assert.Equal(t, 200, recorder.Code)
			assert.Equal(t, "image/png", recorder.Header().Get("Content-Type"))
			assert.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
			assert.Equal(t, buildSiteLogoPNG(t, 16, 16), recorder.Body.Bytes())
			oldURL := saved[1].QRCodeURL
			ok, saved = saveSupportFixture(t, saved, map[string][]byte{"group": buildSiteLogoPNG(t, 24, 24)})
			require.True(t, ok)
			assert.NotEqual(t, oldURL, saved[1].QRCodeURL)
			_, err = os.Stat(filepath.Join(dir, "support", strings.TrimPrefix(oldURL, supportQRPrefix)))
			require.True(t, os.IsNotExist(err))
			before := common.SupportContacts
			// If a later image fails validation, roll back an earlier staged upload.
			ok, _ = saveSupportFixture(t, saved, map[string][]byte{"group": buildSiteLogoPNG(t, 16, 16), "wechat": []byte("<svg onload='alert(1)'/>")})
			assert.False(t, ok)
			assert.Equal(t, before, common.SupportContacts)
			entries, err := os.ReadDir(filepath.Join(dir, "support"))
			require.NoError(t, err)
			assert.Len(t, entries, 2)
			for _, value := range []string{"https://external.invalid/a.png", supportQRPrefix + "../../secret", supportQRPrefix + "qr-00000000000000000000000000000000.png"} {
				invalid := append([]SupportContact{}, saved...)
				invalid[1].QRCodeURL = value
				ok, _ = saveSupportFixture(t, invalid, nil)
				assert.False(t, ok)
			}
			duplicate := append([]SupportContact{}, saved...)
			duplicate[1].Id = duplicate[0].Id
			ok, _ = saveSupportFixture(t, duplicate, nil)
			assert.False(t, ok)
			// Removing a QR and switching a group to a regular contact clean both files.
			saved[1].QRCodeURL = ""
			saved[2].Type = "wechat"
			ok, saved = saveSupportFixture(t, saved, nil)
			require.True(t, ok)
			assert.Empty(t, saved[2].QRCodeURL)
			entries, err = os.ReadDir(filepath.Join(dir, "support"))
			require.NoError(t, err)
			assert.Empty(t, entries)
			ok, saved = saveSupportFixture(t, nil, nil)
			require.True(t, ok)
			assert.Empty(t, saved)

			// A real persistence failure must roll back staged images as well.
			require.NoError(t, db.Migrator().DropTable(&model.Option{}))
			ok, _ = saveSupportFixture(t, contacts, map[string][]byte{"group": buildSiteLogoPNG(t, 16, 16)})
			assert.False(t, ok)
			entries, err = os.ReadDir(filepath.Join(dir, "support"))
			require.NoError(t, err)
			assert.Empty(t, entries)
			assert.Equal(t, "[]", common.SupportContacts)
		})
	}
}

func TestSupportContactRejectsInvalidUploads(t *testing.T) {
	old := common.SupportContacts
	common.SupportContacts = "[]"
	defer func() { common.SupportContacts = old }()
	dir := t.TempDir()
	t.Setenv("SITE_ASSET_STORAGE_DIR", dir)
	for _, data := range [][]byte{nil, []byte("not an image"), buildSiteLogoPNG(t, 16, 16)[:33], make([]byte, maxSiteLogoBytes+1), buildSiteLogoPNG(t, 4097, 1)} {
		ok, _ := saveSupportFixture(t, []SupportContact{{Id: "g", Type: "qq_group", Value: "123"}}, map[string][]byte{"g": data})
		assert.False(t, ok)
	}
	for _, filename := range []string{"../secret", "qr-bad.png", "logo-00000000000000000000000000000000.png"} {
		r := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(r)
		c.Params = gin.Params{{Key: "filename", Value: filename}}
		GetSupportQRCode(c)
		c.Writer.WriteHeaderNow()
		assert.Equal(t, 404, r.Code)
	}
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
