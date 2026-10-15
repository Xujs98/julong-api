package controller

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func mediaAssetAdmin(c *gin.Context) bool {
	return c.GetInt("role") >= common.RoleAdminUser
}

func mediaAssetResponse(asset model.MediaAsset) gin.H {
	publicURL := asset.OriginalURL
	if asset.SourceType == model.MediaAssetSourceBase64 {
		publicURL = service.BuildMediaAssetURL(asset.AssetID)
	}
	return gin.H{
		"asset_id": asset.AssetID, "user_id": asset.UserID, "username": asset.Username,
		"source_type": asset.SourceType, "media_type": asset.MediaType, "mime_type": asset.MimeType,
		"url": publicURL, "size": asset.Size, "created_at": asset.CreatedAt, "expires_at": asset.ExpiresAt,
	}
}

func ListMediaAssets(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	startTime, _ := strconv.ParseInt(c.Query("start_time"), 10, 64)
	endTime, _ := strconv.ParseInt(c.Query("end_time"), 10, 64)
	username := c.Query("username")
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	assets, total, err := model.ListMediaAssets(c.GetInt("id"), mediaAssetAdmin(c) && c.Query("view_scope") != "self", (page-1)*limit, limit, startTime, endTime, username)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items := make([]gin.H, 0, len(assets))
	for _, asset := range assets {
		items = append(items, mediaAssetResponse(asset))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": items, "total": total, "page": page, "page_size": limit}})
}

func DeleteMediaAsset(c *gin.Context) {
	if !mediaAssetAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "管理员权限不足"})
		return
	}
	asset, err := model.GetMediaAsset(c.Param("asset_id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "素材不存在"})
			return
		}
		common.ApiError(c, err)
		return
	}
	if asset.StoredPath != "" {
		if err := os.Remove(asset.StoredPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			common.ApiError(c, fmt.Errorf("删除素材文件失败: %w", err))
			return
		}
	}
	asset, err = model.DeleteMediaAsset(c.Param("asset_id"), c.GetInt("id"), true)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "素材不存在"})
			return
		}
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func MediaAssetContent(c *gin.Context) {
	asset, err := model.GetMediaAsset(c.Param("asset_id"))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	if asset.SourceType == model.MediaAssetSourcePublic {
		c.Status(http.StatusNotFound)
		return
	}
	if asset.ExpiresAt <= time.Now().Unix() {
		c.Status(http.StatusGone)
		return
	}
	file, err := service.ServeMediaAsset(asset)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer file.Close()
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	// Native media documents request their own URL again for playback/Range.
	// Preserve its origin for 'self' to match; scripts remain disabled.
	c.Header("Content-Security-Policy", "sandbox allow-same-origin; default-src 'none'; media-src 'self'")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("Content-Type", asset.MimeType)
	c.Header("Content-Disposition", "inline")
	http.ServeContent(c.Writer, c.Request, asset.AssetID, time.Unix(asset.CreatedAt, 0), file)
}
