package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

const (
	mediaAssetDirEnv       = "MEDIA_ASSET_STORAGE_DIR"
	mediaAssetMaxBytes     = 50 << 20
	mediaAssetMaxTotalSize = 128 << 20
)

var errMediaAssetTooLarge = errors.New("media asset exceeds the 50 MB per-file or 128 MB request limit")

func mediaAssetDir() string {
	if configured := strings.TrimSpace(os.Getenv(mediaAssetDirEnv)); configured != "" {
		return configured
	}
	return filepath.Join("data", "media-assets")
}

func mediaAssetRetentionDays() int {
	if common.MediaAssetRetentionDays <= 0 {
		return 1
	}
	return common.MediaAssetRetentionDays
}

func mediaAssetPublicURL(assetID string) string {
	base := strings.TrimRight(strings.TrimSpace(system_setting.TaskPublicAddress), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(system_setting.ServerAddress), "/")
	}
	return base + "/api/media-assets/" + url.PathEscape(assetID) + "/content"
}

func BuildMediaAssetURL(assetID string) string {
	return mediaAssetPublicURL(assetID)
}

func mediaTypeForMime(mimeType string) string {
	mimeType = strings.ToLower(mimeType)
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	}
	if strings.HasPrefix(mimeType, "video/") {
		return "video"
	}
	if strings.HasPrefix(mimeType, "audio/") {
		return "audio"
	}
	return ""
}

func parseMediaDataURL(value string, remainingBytes int64) ([]byte, string, bool, error) {
	if !strings.HasPrefix(value, "data:") {
		return nil, "", false, nil
	}
	comma := strings.IndexByte(value, ',')
	if comma < 0 {
		return nil, "", true, errors.New("invalid media data URL")
	}
	meta, payload := value[5:comma], value[comma+1:]
	parts := strings.Split(meta, ";")
	mimeType := strings.TrimSpace(parts[0])
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	encoded := false
	for _, part := range parts[1:] {
		if strings.EqualFold(part, "base64") {
			encoded = true
		}
	}
	if !encoded {
		return nil, "", true, errors.New("media data URL must use base64 encoding")
	}
	// Reject over-budget payloads before the decoder allocates its output. Both
	// standard and unpadded Base64 accept CR/LF, which do not consume the budget.
	limit := int(max(int64(0), min(int64(mediaAssetMaxBytes), remainingBytes)))
	encodedLength, padding := 0, 0
	for i := range len(payload) {
		if payload[i] == '\r' || payload[i] == '\n' {
			continue
		}
		encodedLength++
		if encodedLength > base64.StdEncoding.EncodedLen(limit) {
			return nil, "", true, errMediaAssetTooLarge
		}
		if payload[i] == '=' {
			padding++
		} else {
			padding = 0
		}
	}
	decodedLength := encodedLength/4*3 + encodedLength%4*6/8
	if encodedLength%4 == 0 {
		decodedLength -= min(padding, 2)
	}
	if decodedLength > limit {
		return nil, "", true, errMediaAssetTooLarge
	}
	if encodedLength != len(payload) {
		var normalized strings.Builder
		normalized.Grow(encodedLength)
		for i := range len(payload) {
			if payload[i] != '\r' && payload[i] != '\n' {
				normalized.WriteByte(payload[i])
			}
		}
		payload = normalized.String()
	}
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(payload)
	}
	if err != nil {
		return nil, "", true, errors.New("invalid base64 media data URL")
	}
	if len(decoded) == 0 || len(decoded) > limit {
		return nil, "", true, errMediaAssetTooLarge
	}
	return decoded, mimeType, true, nil
}

// PersistTaskMedia converts data URLs in a task request to short-lived public
// URLs. Public URLs are recorded for the dashboard but are never downloaded or
// deleted by this service.
func PersistTaskMedia(ctx context.Context, userID int, value any) (any, error) {
	total := int64(0)
	created := make([]*model.MediaAsset, 0)
	result, err := persistTaskMediaValue(ctx, userID, "", "", value, &total, &created)
	if err != nil {
		removeCreatedMediaAssets(created)
		return nil, err
	}
	return result, nil
}

func persistTaskMediaValue(ctx context.Context, userID int, key, inheritedMediaType string, value any, total *int64, created *[]*model.MediaAsset) (any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	mediaType := mediaTypeForKey(key)
	if mediaType == "" {
		mediaType = inheritedMediaType
	}
	switch current := value.(type) {
	case map[string]any:
		if _, isUploadedFileRef := current["__fileRef"]; isUploadedFileRef {
			return current, nil
		}
		if mediaType == "" {
			if kind, ok := current["type"].(string); ok {
				mediaType = mediaTypeForKey(kind)
			}
		}
		for childKey, child := range current {
			// Media objects also carry role, detail and voice_id metadata. Only
			// their URL member inherits the containing media field's type.
			childMediaType := ""
			if strings.EqualFold(childKey, "url") {
				childMediaType = mediaType
			}
			replaced, err := persistTaskMediaValue(ctx, userID, childKey, childMediaType, child, total, created)
			if err != nil {
				return nil, err
			}
			current[childKey] = replaced
		}
		return current, nil
	case []any:
		for i, child := range current {
			replaced, err := persistTaskMediaValue(ctx, userID, key, mediaType, child, total, created)
			if err != nil {
				return nil, err
			}
			current[i] = replaced
		}
		return current, nil
	case string:
		if mediaType == "" || strings.TrimSpace(current) == "" {
			return current, nil
		}
		data, mimeType, isData, err := parseMediaDataURL(current, mediaAssetMaxTotalSize-*total)
		if err != nil {
			return nil, err
		}
		if isData {
			kind := mediaTypeForMime(mimeType)
			if kind == "" {
				return nil, fmt.Errorf("unsupported media MIME type %q", mimeType)
			}
			if *total+int64(len(data)) > mediaAssetMaxTotalSize {
				return nil, errMediaAssetTooLarge
			}
			asset, err := saveBase64MediaAsset(ctx, userID, kind, mimeType, data)
			if err != nil {
				return nil, err
			}
			*created = append(*created, asset)
			*total += int64(len(data))
			return mediaAssetPublicURL(asset.AssetID), nil
		}
		if mediaType == "" {
			return current, nil
		}
		if err := ValidateUserMediaURL(current); err != nil {
			return nil, err
		}
		asset := &model.MediaAsset{UserID: userID, SourceType: model.MediaAssetSourcePublic, MediaType: mediaType, OriginalURL: current, MimeType: ""}
		if err := model.DB.Create(asset).Error; err != nil {
			return nil, err
		}
		*created = append(*created, asset)
		return current, nil
	default:
		return value, nil
	}
}

func removeCreatedMediaAssets(assets []*model.MediaAsset) {
	for _, asset := range assets {
		if asset == nil {
			continue
		}
		if asset.StoredPath != "" {
			_ = os.Remove(asset.StoredPath)
		}
		_ = model.DB.Delete(asset).Error
	}
}

func mediaTypeForKey(key string) string {
	// Match media payload fields, not parameters such as image_size or
	// sequential_image_generation that happen to contain a media word.
	switch strings.ToLower(key) {
	case "image", "images", "image_url", "image_urls", "image_refs",
		"reference_image", "reference_images", "referenceimages", "input_reference",
		"input_image", "input_images", "first_frame_image", "last_frame_image",
		"first_frame_url", "last_frame_url", "first_frame", "last_frame", "image_tail",
		"base_image_url", "mask_image_url", "img_url":
		return "image"
	case "video", "videos", "video_url", "video_urls", "video_refs",
		"reference_video", "reference_videos", "referencevideos", "input_video",
		"first_clip":
		return "video"
	case "audio", "audios", "audio_url", "audio_urls", "audio_refs",
		"reference_audio", "reference_audios", "referenceaudios", "input_audio",
		"driving_audio":
		return "audio"
	default:
		return ""
	}
}

func saveBase64MediaAsset(ctx context.Context, userID int, mediaType, mimeType string, data []byte) (*model.MediaAsset, error) {
	if err := os.MkdirAll(mediaAssetDir(), 0o750); err != nil {
		return nil, err
	}
	assetID, err := common.GenerateRandomCharsKey(32)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(mediaAssetDir(), "asset_"+assetID)
	asset := &model.MediaAsset{AssetID: "asset_" + assetID, UserID: userID, SourceType: model.MediaAssetSourceBase64, MediaType: mediaType, MimeType: mimeType, Size: int64(len(data)), StoredPath: path, ExpiresAt: time.Now().Unix() + int64(mediaAssetRetentionDays()*86400)}
	if err := model.DB.Create(asset).Error; err != nil {
		return nil, err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		_ = model.DB.Delete(asset).Error
		return nil, err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		_ = model.DB.Delete(asset).Error
		return nil, err
	}
	return asset, nil
}

func CleanupExpiredMediaAssets(ctx context.Context) (int64, error) {
	var assets []model.MediaAsset
	if err := model.DB.Where("source_type = ? AND expires_at > 0 AND expires_at <= ?", model.MediaAssetSourceBase64, time.Now().Unix()).Limit(500).Find(&assets).Error; err != nil {
		return 0, err
	}
	var deleted int64
	for _, asset := range assets {
		if err := ctx.Err(); err != nil {
			return deleted, err
		}
		if asset.StoredPath != "" {
			if err := os.Remove(asset.StoredPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return deleted, fmt.Errorf("remove expired media asset %s: %w", asset.AssetID, err)
			}
		}
		if err := model.DB.Delete(&asset).Error; err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}

func ValidateUserMediaURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return errors.New("media URLs must be absolute HTTP or HTTPS URLs")
	}
	if parsed.User != nil {
		return errors.New("media URLs must not contain credentials")
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return errors.New("media URLs must use a public host")
	}
	if ip := net.ParseIP(host); ip != nil {
		if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			return errors.New("media URLs must use a public host")
		}
	}
	return nil
}

func ServeMediaAsset(asset *model.MediaAsset) (*os.File, error) {
	if asset.SourceType != model.MediaAssetSourceBase64 || asset.StoredPath == "" || asset.ExpiresAt <= time.Now().Unix() {
		return nil, os.ErrNotExist
	}
	return os.Open(asset.StoredPath)
}
