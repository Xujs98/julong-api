package service

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPersistTaskMediaStoresBase64AndRetainsExternalURL(t *testing.T) {
	require.NoError(t, model.DB.AutoMigrate(&model.MediaAsset{}))
	require.NoError(t, model.DB.Exec("DELETE FROM media_assets").Error)
	dir := t.TempDir()
	t.Setenv(mediaAssetDirEnv, dir)
	previousRetention := common.MediaAssetRetentionDays
	common.MediaAssetRetentionDays = 1
	t.Cleanup(func() { common.MediaAssetRetentionDays = previousRetention })

	data := []byte("fake-image-content")
	body := map[string]any{
		"images": []any{"data:image/png;base64," + base64.StdEncoding.EncodeToString(data), "https://user-media.example/image.png", map[string]any{"url": "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("nested"))}},
	}
	result, err := PersistTaskMedia(context.Background(), 42, body)
	require.NoError(t, err)
	converted := result.(map[string]any)["images"].([]any)
	assert.Equal(t, "https://user-media.example/image.png", converted[1])
	assert.Contains(t, converted[0], "/api/media-assets/asset_")

	var assets []model.MediaAsset
	require.NoError(t, model.DB.Order("id").Find(&assets).Error)
	require.Len(t, assets, 3)
	assert.Equal(t, model.MediaAssetSourceBase64, assets[0].SourceType)
	assert.Equal(t, "image", assets[0].MediaType)
	assert.Equal(t, data, mustReadFile(t, assets[0].StoredPath))
	assert.Equal(t, model.MediaAssetSourcePublic, assets[1].SourceType)
	assert.Equal(t, "https://user-media.example/image.png", assets[1].OriginalURL)
	assert.Zero(t, assets[1].ExpiresAt)
	assert.Equal(t, model.MediaAssetSourceBase64, assets[2].SourceType)
	assert.Equal(t, []byte("nested"), mustReadFile(t, assets[2].StoredPath))

	assets[0].ExpiresAt = time.Now().Unix() - 1
	require.NoError(t, model.DB.Model(&assets[0]).Update("expires_at", assets[0].ExpiresAt).Error)
	deleted, err := CleanupExpiredMediaAssets(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, 1, deleted)
	assert.NoFileExists(t, filepath.Join(dir, assets[0].AssetID))
	assert.Error(t, model.DB.Where("asset_id = ?", assets[0].AssetID).First(&model.MediaAsset{}).Error)
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

func TestPersistTaskMediaRejectsUnsupportedAndOversizedBase64(t *testing.T) {
	require.NoError(t, model.DB.AutoMigrate(&model.MediaAsset{}))
	require.NoError(t, model.DB.Exec("DELETE FROM media_assets").Error)
	dir := t.TempDir()
	t.Setenv(mediaAssetDirEnv, dir)
	valid := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("valid"))
	_, err := PersistTaskMedia(context.Background(), 1, map[string]any{"images": []any{valid, "data:text/plain;base64,eA=="}})
	assert.Error(t, err)
	var remaining int64
	require.NoError(t, model.DB.Model(&model.MediaAsset{}).Count(&remaining).Error)
	assert.Zero(t, remaining)
	entries, readErr := os.ReadDir(dir)
	require.NoError(t, readErr)
	assert.Empty(t, entries)

	tests := []struct {
		name  string
		value string
	}{
		{name: "unsupported MIME", value: "data:text/plain;base64," + base64.StdEncoding.EncodeToString([]byte("x"))},
		{name: "missing base64 marker", value: "data:image/png,x"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := PersistTaskMedia(context.Background(), 1, map[string]any{"image": testCase.value})
			assert.Error(t, err)
		})
	}
}

func TestPersistTaskMediaPreservesNonMediaFieldsAndObjectMetadata(t *testing.T) {
	require.NoError(t, model.DB.AutoMigrate(&model.MediaAsset{}))
	t.Setenv(mediaAssetDirEnv, t.TempDir())
	cases := []struct {
		name   string
		body   map[string]any
		assets int
	}{
		{
			name: "model parameters are not media URLs",
			body: map[string]any{"image_size": "1024x1024", "video_quality": "high", "sequential_image_generation": "auto", "audio_format": "mp3"},
		},
		{
			name: "voice identifiers are not audio URLs",
			body: map[string]any{"reference_audios": []any{map[string]any{"voice_id": "voice_fixture"}}},
		},
		{
			name:   "image object metadata is retained",
			body:   map[string]any{"images": []any{map[string]any{"url": "https://cdn.example/image.png", "type": "image_url", "role": "first_frame", "detail": "high"}}},
			assets: 1,
		},
		{
			name:   "frame fields and camel case aliases remain media",
			body:   map[string]any{"first_frame_image": "https://cdn.example/first.png", "last_frame_image": "https://cdn.example/last.png", "image_tail": "https://cdn.example/tail.png", "referenceVideos": []any{"https://cdn.example/reference.mp4"}},
			assets: 4,
		},
		{
			name:   "typed media object only transforms its URL",
			body:   map[string]any{"media": []any{map[string]any{"type": "reference_video", "url": "https://cdn.example/reference.mp4", "role": "reference"}}},
			assets: 1,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			require.NoError(t, model.DB.Exec("DELETE FROM media_assets").Error)
			expected, err := common.Marshal(testCase.body)
			require.NoError(t, err)
			result, err := PersistTaskMedia(context.Background(), 42, testCase.body)
			require.NoError(t, err)
			actual, err := common.Marshal(result)
			require.NoError(t, err)
			assert.JSONEq(t, string(expected), string(actual))
			var assets []model.MediaAsset
			require.NoError(t, model.DB.Find(&assets).Error)
			assert.Len(t, assets, testCase.assets)
		})
	}
}

func TestMediaDataURLRespectsRemainingBudgetAndEncoding(t *testing.T) {
	cases := []struct {
		name      string
		payload   string
		remaining int64
		expected  string
		tooLarge  bool
		invalid   bool
	}{
		{name: "padded exact boundary", payload: "eA==", remaining: 1, expected: "x"},
		{name: "unpadded exact boundary", payload: "eA", remaining: 1, expected: "x"},
		{name: "CRLF does not consume budget", payload: "e\r\nA=\n=\r\n", remaining: 1, expected: "x"},
		{name: "full quantum boundary", payload: "YWJj", remaining: 3, expected: "abc"},
		{name: "padded exceeds remaining budget", payload: "eHk=", remaining: 1, tooLarge: true},
		{name: "unpadded exceeds remaining budget", payload: "eHk", remaining: 1, tooLarge: true},
		{name: "exhausted request budget", payload: "eA==", tooLarge: true},
		{name: "invalid encoding", payload: "!A==", remaining: 3, invalid: true},
		{name: "empty encoding", remaining: 3, tooLarge: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, mime, isData, err := parseMediaDataURL("data:image/png;base64,"+tc.payload, tc.remaining)
			assert.True(t, isData)
			if tc.tooLarge {
				assert.ErrorIs(t, err, errMediaAssetTooLarge)
				assert.Nil(t, data)
			} else if tc.invalid {
				assert.EqualError(t, err, "invalid base64 media data URL")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, string(data))
				assert.Equal(t, "image/png", mime)
			}
		})
	}
	// An encoded file beyond the per-file ceiling is rejected before decoding.
	oversized := "data:video/mp4;base64," + strings.Repeat("A", base64.StdEncoding.EncodedLen(mediaAssetMaxBytes)+4)
	data, _, _, err := parseMediaDataURL(oversized, mediaAssetMaxTotalSize)
	assert.ErrorIs(t, err, errMediaAssetTooLarge)
	assert.Nil(t, data)
}

func TestPersistTaskMediaRejectsExhaustedBudgetAndCancellationWithoutFiles(t *testing.T) {
	require.NoError(t, model.DB.AutoMigrate(&model.MediaAsset{}))
	require.NoError(t, model.DB.Exec("DELETE FROM media_assets").Error)
	dir := t.TempDir()
	t.Setenv(mediaAssetDirEnv, dir)
	total := int64(mediaAssetMaxTotalSize - 2)
	var created []*model.MediaAsset
	_, err := persistTaskMediaValue(context.Background(), 42, "image", "", "data:image/png;base64,YWJj", &total, &created)
	assert.ErrorIs(t, err, errMediaAssetTooLarge)
	assert.Empty(t, created)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = PersistTaskMedia(ctx, 42, map[string]any{"images": []any{"data:image/png;base64,eA=="}})
	assert.ErrorIs(t, err, context.Canceled)
	var count int64
	require.NoError(t, model.DB.Model(&model.MediaAsset{}).Count(&count).Error)
	assert.Zero(t, count)
	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestMediaAssetSchemaCompatibility(t *testing.T) {
	cases := []struct {
		name string
		env  string
		open func(string) gorm.Dialector
	}{
		{name: "mysql", env: "TEST_MYSQL_DSN", open: mysql.Open},
		{name: "postgres", env: "TEST_POSTGRES_DSN", open: func(dsn string) gorm.Dialector {
			return postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true})
		}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			dsn := os.Getenv(testCase.env)
			if dsn == "" {
				t.Skipf("%s is not configured", testCase.env)
			}
			db, err := gorm.Open(testCase.open(dsn), &gorm.Config{})
			require.NoError(t, err)
			require.NoError(t, db.AutoMigrate(&model.MediaAsset{}))
			require.NoError(t, db.AutoMigrate(&model.MediaAsset{}))
			asset := &model.MediaAsset{UserID: 9, SourceType: model.MediaAssetSourcePublic, MediaType: "video", OriginalURL: "https://example.test/video.mp4"}
			require.NoError(t, db.Create(asset).Error)
			var loaded model.MediaAsset
			require.NoError(t, db.Where("asset_id = ?", asset.AssetID).First(&loaded).Error)
			assert.Equal(t, asset.OriginalURL, loaded.OriginalURL)
			require.NoError(t, db.Delete(&loaded).Error)
		})
	}
}

func TestValidateUserMediaURL(t *testing.T) {
	for _, value := range []string{
		"https://cdn.example/video.mp4",
		"http://8.8.8.8/reference.png",
	} {
		assert.NoError(t, ValidateUserMediaURL(value), value)
	}
	for _, value := range []string{
		"file:///etc/passwd",
		"https://localhost/reference.png",
		"http://127.0.0.1/reference.png",
		"http://192.168.1.10/reference.png",
		"https://user:secret@cdn.example/reference.png",
	} {
		assert.Error(t, ValidateUserMediaURL(value), value)
	}
}
