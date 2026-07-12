package service

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
)

var lastImageGenerationLogCleanup atomic.Int64

const (
	imageGenerationResultKey = "image_generation_log_results"
	maxLoggedImageBytes      = 25 * 1024 * 1024
)

func CaptureImageGenerationResult(c *gin.Context, images []dto.ImageData) {
	if c == nil || !common.ImageGenerationLogEnabled || len(images) == 0 {
		return
	}
	existing, _ := c.Get(imageGenerationResultKey)
	results, _ := existing.([]dto.ImageData)
	c.Set(imageGenerationResultKey, append(results, images...))
}

func RecordImageGenerationLog(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ImageRequest, quota int) {
	if c == nil || info == nil || request == nil || !common.ImageGenerationLogEnabled {
		return
	}
	value, exists := c.Get(imageGenerationResultKey)
	images, ok := value.([]dto.ImageData)
	if !exists || !ok || len(images) == 0 {
		return
	}

	refs := make([]model.ImageGenerationImage, 0, len(images))
	for _, image := range images {
		ref, err := persistImageGenerationResult(image)
		if err != nil {
			logger.LogError(c, "failed to persist generated image: "+err.Error())
			continue
		}
		refs = append(refs, ref)
	}
	if len(refs) == 0 {
		return
	}

	encoded, err := json.Marshal(refs)
	if err != nil {
		logger.LogError(c, "failed to encode generated image refs: "+err.Error())
		return
	}
	log := model.NewImageGenerationLog(info.UserId)
	log.TokenId = info.TokenId
	if token, tokenErr := model.GetTokenById(info.TokenId); tokenErr == nil && token != nil {
		log.TokenName = token.Name
	}
	log.ChannelId = info.ChannelId
	log.ModelName = info.OriginModelName
	log.Prompt = truncateImageLogPrompt(request.Prompt)
	log.Size = request.Size
	log.Quality = request.Quality
	log.ImageCount = len(refs)
	log.Images = string(encoded)
	log.Quota = quota
	log.RequestId = c.GetString(common.RequestIdKey)
	log.UseTime = int(time.Since(info.StartTime).Seconds())
	if err := log.Insert(); err != nil {
		logger.LogError(c, "failed to record image generation log: "+err.Error())
	}
	cleanupExpiredImageGenerationLogs(c)
}

func cleanupExpiredImageGenerationLogs(c *gin.Context) {
	days := common.ImageGenerationLogRetentionDays
	if days <= 0 {
		return
	}
	now := time.Now().Unix()
	last := lastImageGenerationLogCleanup.Load()
	if now-last < 3600 || !lastImageGenerationLogCleanup.CompareAndSwap(last, now) {
		return
	}
	cutoff := now - int64(days)*24*3600
	var logs []model.ImageGenerationLog
	if err := model.DB.Where("created_at < ?", cutoff).Find(&logs).Error; err != nil {
		logger.LogError(c, "failed to find expired image generation logs: "+err.Error())
		return
	}
	files := make([]string, 0)
	for _, log := range logs {
		refs, err := log.ImageRefs()
		if err != nil {
			continue
		}
		for _, ref := range refs {
			if ref.Type == "local" && filepath.Base(ref.Value) == ref.Value {
				files = append(files, filepath.Join(ImageGenerationLogStorageDir(), ref.Value))
			}
		}
	}
	if len(logs) > 0 {
		ids := make([]int, 0, len(logs))
		for _, log := range logs {
			ids = append(ids, log.Id)
		}
		if err := model.DB.Delete(&model.ImageGenerationLog{}, ids).Error; err != nil {
			logger.LogError(c, "failed to delete expired image generation logs: "+err.Error())
			return
		}
		for _, file := range files {
			_ = os.Remove(file)
		}
	}
}

func ImageGenerationLogStorageDir() string {
	if dir := strings.TrimSpace(os.Getenv("IMAGE_LOG_STORAGE_DIR")); dir != "" {
		return dir
	}
	return "image-generation-logs"
}

func persistImageGenerationResult(image dto.ImageData) (model.ImageGenerationImage, error) {
	if strings.TrimSpace(image.B64Json) == "" {
		url := strings.TrimSpace(image.Url)
		if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
			return model.ImageGenerationImage{}, fmt.Errorf("unsupported image URL")
		}
		return model.ImageGenerationImage{
			Type:          "remote",
			Value:         url,
			RevisedPrompt: image.RevisedPrompt,
		}, nil
	}

	encoded := image.B64Json
	if comma := strings.Index(encoded, ","); strings.HasPrefix(encoded, "data:") && comma >= 0 {
		encoded = encoded[comma+1:]
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(encoded)
	}
	if err != nil {
		return model.ImageGenerationImage{}, err
	}
	if len(data) == 0 || len(data) > maxLoggedImageBytes {
		return model.ImageGenerationImage{}, fmt.Errorf("image size is outside the allowed range")
	}
	mimeType := http.DetectContentType(data)
	if !strings.HasPrefix(mimeType, "image/") {
		return model.ImageGenerationImage{}, fmt.Errorf("generated content is not an image")
	}
	dir := ImageGenerationLogStorageDir()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return model.ImageGenerationImage{}, err
	}
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		return model.ImageGenerationImage{}, err
	}
	ext := extensionForImageMime(mimeType)
	name := fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), hex.EncodeToString(random), ext)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o640); err != nil {
		return model.ImageGenerationImage{}, err
	}
	return model.ImageGenerationImage{
		Type:          "local",
		Value:         name,
		MimeType:      mimeType,
		RevisedPrompt: image.RevisedPrompt,
	}, nil
}

func extensionForImageMime(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".png"
	}
}

func truncateImageLogPrompt(prompt string) string {
	const maxPromptRunes = 20000
	runes := []rune(prompt)
	if len(runes) <= maxPromptRunes {
		return prompt
	}
	return string(runes[:maxPromptRunes])
}
