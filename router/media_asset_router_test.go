/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMediaAssetContentPreservesRawRangesAndEmptyHEAD(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previous := model.DB
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "assets.db")), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	t.Cleanup(func() {
		model.DB = previous
		sqlDB, closeErr := db.DB()
		require.NoError(t, closeErr)
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, db.AutoMigrate(&model.MediaAsset{}))
	payload := []byte("media-fixture-for-http-range")
	path := filepath.Join(t.TempDir(), "video")
	require.NoError(t, os.WriteFile(path, payload, 0o600))
	asset := model.MediaAsset{
		AssetID: "asset_range_fixture", UserID: 1,
		SourceType: model.MediaAssetSourceBase64, MediaType: "video",
		MimeType: "video/mp4", StoredPath: path, Size: int64(len(payload)),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	require.NoError(t, db.Create(&asset).Error)
	engine := gin.New()
	SetApiRouter(engine)
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/media-assets/"+asset.AssetID+"/content", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			req.Header.Set("Range", "bytes=0-3")
			out := httptest.NewRecorder()
			engine.ServeHTTP(out, req)
			require.Equal(t, http.StatusPartialContent, out.Code)
			assert.Empty(t, out.Header().Get("Content-Encoding"))
			assert.Equal(t, "bytes 0-3/28", out.Header().Get("Content-Range"))
			assert.Equal(t, "4", out.Header().Get("Content-Length"))
			assert.Equal(t, "video/mp4", out.Header().Get("Content-Type"))
			assert.Equal(t, "sandbox allow-same-origin; default-src 'none'; media-src 'self'", out.Header().Get("Content-Security-Policy"))
			if method == http.MethodHead {
				assert.Empty(t, out.Body.Bytes())
			} else {
				assert.Equal(t, payload[:4], out.Body.Bytes())
			}
		})
	}
	// Exclusion applies only to media bytes; ordinary API JSON stays compressed.
	req := httptest.NewRequest(http.MethodGet, "/api/notice", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	out := httptest.NewRecorder()
	engine.ServeHTTP(out, req)
	assert.Equal(t, http.StatusOK, out.Code)
	assert.Equal(t, "gzip", out.Header().Get("Content-Encoding"))
}
