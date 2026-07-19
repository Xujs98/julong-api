package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
)

var lastImageGenerationLogCleanup atomic.Int64

const (
	imageGenerationResultKey      = "image_generation_log_results"
	maxLoggedImageBytes           = 25 * 1024 * 1024
	maxImageTaskErrorBytes        = 256 * 1024
	ImageGenerationTaskContextKey = "image_generation_task_id"
)

type ImageGenerationTaskURLBuilder func(index int) string

func shouldCaptureImageGeneration(c *gin.Context) bool {
	return common.ImageGenerationLogEnabled || (c != nil && c.GetString(ImageGenerationTaskContextKey) != "")
}

func CaptureImageGenerationResult(c *gin.Context, images []dto.ImageData) {
	if c == nil || !shouldCaptureImageGeneration(c) || len(images) == 0 {
		return
	}
	existing, _ := c.Get(imageGenerationResultKey)
	results, _ := existing.([]dto.ImageData)
	for _, image := range images {
		duplicate := false
		for _, result := range results {
			if (image.B64Json != "" && image.B64Json == result.B64Json) ||
				(image.Url != "" && image.Url == result.Url) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			results = append(results, image)
		}
	}
	c.Set(imageGenerationResultKey, results)
}

func CaptureResponsesImageGenerationResult(c *gin.Context, response *dto.OpenAIResponsesResponse) {
	if response == nil {
		return
	}
	for i := range response.Output {
		CaptureResponsesImageGenerationOutput(c, &response.Output[i])
	}
}

func CaptureResponsesImageGenerationOutput(c *gin.Context, output *dto.ResponsesOutput) {
	if output == nil || output.Type != dto.ResponsesOutputTypeImageGenerationCall || strings.TrimSpace(output.Result) == "" {
		return
	}
	CaptureImageGenerationResult(c, []dto.ImageData{{B64Json: output.Result}})
	if output.Quality != "" {
		c.Set("image_generation_call_quality", output.Quality)
	}
	if output.Size != "" {
		c.Set("image_generation_call_size", output.Size)
	}
}

func RecordImageGenerationLog(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ImageRequest, quota int) {
	if c == nil || info == nil || request == nil || !shouldCaptureImageGeneration(c) {
		return
	}
	RecordCapturedImageGenerationLog(c, info, request.Prompt, request.Size, request.Quality, quota)
}

func RecordCapturedImageGenerationLog(c *gin.Context, info *relaycommon.RelayInfo, prompt, size, quality string, quota int) {
	if c == nil || info == nil || !shouldCaptureImageGeneration(c) {
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
	log.Prompt = truncateImageLogPrompt(prompt)
	log.Size = size
	log.Quality = quality
	log.ImageCount = len(refs)
	log.Images = string(encoded)
	log.Quota = quota
	log.RequestId = c.GetString(common.RequestIdKey)
	log.UseTime = int(time.Since(info.StartTime).Seconds())
	if taskId := c.GetString(ImageGenerationTaskContextKey); taskId != "" {
		taskLog, taskErr := model.GetUserImageGenerationLogByTaskId(info.UserId, taskId)
		if taskErr != nil {
			logger.LogError(c, "failed to find image generation task: "+taskErr.Error())
			return
		}
		if err := taskLog.UpdateTask(map[string]any{
			"status":        model.ImageGenerationStatusSuccess,
			"token_id":      log.TokenId,
			"token_name":    log.TokenName,
			"channel_id":    log.ChannelId,
			"model_name":    log.ModelName,
			"prompt":        log.Prompt,
			"size":          log.Size,
			"quality":       log.Quality,
			"image_count":   log.ImageCount,
			"images":        log.Images,
			"quota":         log.Quota,
			"use_time":      log.UseTime,
			"error_message": "",
		}); err != nil {
			logger.LogError(c, "failed to complete image generation task: "+err.Error())
		}
	} else if err := log.Insert(); err != nil {
		logger.LogError(c, "failed to record image generation log: "+err.Error())
	}
	cleanupExpiredImageGenerationLogs(c)
}

func CreateImageGenerationTask(c *gin.Context, request *dto.ImageRequest) (*model.ImageGenerationLog, error) {
	key, err := common.GenerateRandomCharsKey(32)
	if err != nil {
		return nil, err
	}
	log := model.NewImageGenerationLog(c.GetInt("id"))
	log.TaskId = "img_" + key
	log.Status = model.ImageGenerationStatusPending
	log.TokenId = c.GetInt("token_id")
	log.TokenName = c.GetString("token_name")
	log.ChannelId = common.GetContextKeyInt(c, constant.ContextKeyChannelId)
	log.ModelName = request.Model
	log.Prompt = truncateImageLogPrompt(request.Prompt)
	log.Size = request.Size
	log.Quality = request.Quality
	log.RequestId = c.GetString(common.RequestIdKey)
	if err := log.Insert(); err != nil {
		return nil, err
	}
	return log, nil
}

func MarkImageGenerationTaskProcessing(taskId string, userId int) error {
	log, err := model.GetUserImageGenerationLogByTaskId(userId, taskId)
	if err != nil {
		return err
	}
	return log.UpdateTask(map[string]any{"status": model.ImageGenerationStatusProcessing})
}

func CompleteImageGenerationTask(taskId string, userId int, responseBody []byte) error {
	log, err := model.GetUserImageGenerationLogByTaskId(userId, taskId)
	if err != nil {
		return err
	}
	refs, err := log.ImageRefs()
	if err != nil {
		return err
	}
	if len(refs) == 0 {
		var response dto.ImageResponse
		if err := common.Unmarshal(responseBody, &response); err != nil {
			return fmt.Errorf("invalid image generation response: %w", err)
		}
		for _, image := range response.Data {
			ref, persistErr := persistImageGenerationResult(image)
			if persistErr != nil {
				return persistErr
			}
			refs = append(refs, ref)
		}
		if len(refs) == 0 {
			return fmt.Errorf("image generation response contains no images")
		}
		encodedRefs, marshalErr := json.Marshal(refs)
		if marshalErr != nil {
			return marshalErr
		}
		log.Images = string(encodedRefs)
		log.ImageCount = len(refs)
	}
	sanitized, err := sanitizeImageGenerationTaskResponse(responseBody, refs)
	if err != nil {
		return err
	}
	return log.UpdateTask(map[string]any{
		"status":        model.ImageGenerationStatusSuccess,
		"image_count":   len(refs),
		"images":        log.Images,
		"response":      sanitized,
		"error_message": "",
	})
}

func FailImageGenerationTask(taskId string, userId int, responseBody []byte, fallbackMessage string) error {
	log, err := model.GetUserImageGenerationLogByTaskId(userId, taskId)
	if err != nil {
		return err
	}
	message := imageGenerationTaskErrorMessage(responseBody, fallbackMessage)
	if len(responseBody) > maxImageTaskErrorBytes {
		responseBody = responseBody[:maxImageTaskErrorBytes]
	}
	return log.UpdateTask(map[string]any{
		"status":        model.ImageGenerationStatusFailed,
		"error_message": message,
		"response":      string(responseBody),
	})
}

func BuildImageGenerationTaskPayload(log *model.ImageGenerationLog, buildURL ImageGenerationTaskURLBuilder) (map[string]any, error) {
	refs, err := log.ImageRefs()
	if err != nil {
		return nil, err
	}
	data := make([]map[string]any, 0, len(refs))
	for index, ref := range refs {
		url := ref.Value
		if (ref.Type == "local" || ref.Type == "minio") && buildURL != nil {
			url = buildURL(index)
		}
		item := map[string]any{"url": url}
		if ref.Type == "minio" {
			item["bucket"] = ref.Bucket
			item["object_key"] = ref.Value
			item["sha256"] = ref.SHA256
			item["mime_type"] = ref.MimeType
			item["size"] = ref.Size
		}
		if ref.RevisedPrompt != "" {
			item["revised_prompt"] = ref.RevisedPrompt
		}
		data = append(data, item)
	}
	progress := 0
	switch log.Status {
	case model.ImageGenerationStatusProcessing:
		progress = 10
	case model.ImageGenerationStatusSuccess, model.ImageGenerationStatusFailed:
		progress = 100
	}
	payload := map[string]any{
		"task_id":                  log.TaskId,
		"object":                   "image.generation.task",
		"status":                   log.Status,
		"progress":                 progress,
		"created_at":               log.CreatedAt,
		"updated_at":               log.UpdatedAt,
		"request_id":               log.RequestId,
		"model":                    log.ModelName,
		"image_count":              log.ImageCount,
		"data":                     data,
		"polling_interval_seconds": common.ImageGenerationLogPollingIntervalSeconds,
	}
	if log.ErrorMessage != "" {
		payload["error"] = map[string]any{"message": log.ErrorMessage}
	} else {
		payload["error"] = nil
	}
	if log.Response != "" {
		var response map[string]any
		if err := common.Unmarshal([]byte(log.Response), &response); err == nil {
			response["data"] = data
			payload["response"] = response
		}
	}
	return payload, nil
}

func ReadImageGenerationLogImage(log *model.ImageGenerationLog, index int) ([]byte, string, error) {
	refs, err := log.ImageRefs()
	if err != nil || index < 0 || index >= len(refs) {
		return nil, "", fmt.Errorf("image not found")
	}
	ref := refs[index]
	if ref.Type == "minio" {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		data, err := ReadGeneratedImageObject(ctx, ref.Bucket, ref.Value, maxLoggedImageBytes)
		if err != nil {
			return nil, "", err
		}
		mimeType := ref.MimeType
		if mimeType == "" {
			mimeType = http.DetectContentType(data)
		}
		return data, mimeType, nil
	}
	if ref.Type != "local" || filepath.Base(ref.Value) != ref.Value {
		return nil, "", fmt.Errorf("local image not found")
	}
	data, err := os.ReadFile(filepath.Join(ImageGenerationLogStorageDir(), ref.Value))
	if err != nil {
		return nil, "", err
	}
	mimeType := ref.MimeType
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}
	return data, mimeType, nil
}

func sanitizeImageGenerationTaskResponse(responseBody []byte, refs []model.ImageGenerationImage) (string, error) {
	var response map[string]any
	if err := common.Unmarshal(responseBody, &response); err != nil {
		return "", err
	}
	items, _ := response["data"].([]any)
	for index, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		delete(item, "b64_json")
		if index < len(refs) {
			if refs[index].Type == "remote" {
				item["url"] = refs[index].Value
			} else {
				item["url"] = ""
			}
		}
	}
	encoded, err := common.Marshal(response)
	return string(encoded), err
}

func imageGenerationTaskErrorMessage(responseBody []byte, fallback string) string {
	var payload struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if err := common.Unmarshal(responseBody, &payload); err == nil {
		if strings.TrimSpace(payload.Error.Message) != "" {
			return payload.Error.Message
		}
		if strings.TrimSpace(payload.Message) != "" {
			return payload.Message
		}
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "image generation failed"
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
		remoteRef := model.ImageGenerationImage{
			Type:          "remote",
			Value:         url,
			RevisedPrompt: image.RevisedPrompt,
		}
		if !GetImageObjectStorageConfig().Enabled {
			return remoteRef, nil
		}
		data, mimeType, err := downloadGeneratedImage(url)
		if err != nil {
			common.SysError("failed to download generated image for MinIO: " + err.Error())
			return remoteRef, nil
		}
		stored, err := StoreGeneratedImageObject(context.Background(), data, mimeType, extensionForImageMime(mimeType))
		if err != nil {
			common.SysError("failed to store generated image in MinIO: " + err.Error())
			return remoteRef, nil
		}
		return minioImageRef(stored, image.RevisedPrompt), nil
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
	if GetImageObjectStorageConfig().Enabled {
		stored, storeErr := StoreGeneratedImageObject(context.Background(), data, mimeType, extensionForImageMime(mimeType))
		if storeErr == nil {
			return minioImageRef(stored, image.RevisedPrompt), nil
		}
		common.SysError("failed to store generated image in MinIO, falling back to local disk: " + storeErr.Error())
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

func minioImageRef(stored StoredImageObject, revisedPrompt string) model.ImageGenerationImage {
	return model.ImageGenerationImage{
		Type:          "minio",
		Value:         stored.ObjectKey,
		Bucket:        stored.Bucket,
		MimeType:      stored.MimeType,
		SHA256:        stored.SHA256,
		Size:          stored.Size,
		RevisedPrompt: revisedPrompt,
	}
}

func downloadGeneratedImage(rawURL string) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	if err := ValidateSSRFProtectedFetchURL(rawURL); err != nil {
		return nil, "", fmt.Errorf("image URL rejected: %w", err)
	}
	client := GetSSRFProtectedHTTPClient()
	if client == nil {
		return nil, "", fmt.Errorf("protected image download client is not initialized")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, "", fmt.Errorf("下载图片返回 HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxLoggedImageBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 || len(data) > maxLoggedImageBytes {
		return nil, "", fmt.Errorf("image size is outside the allowed range")
	}
	mimeType := strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
	if !strings.HasPrefix(mimeType, "image/") {
		mimeType = http.DetectContentType(data)
	}
	if !strings.HasPrefix(mimeType, "image/") {
		return nil, "", fmt.Errorf("generated content is not an image")
	}
	return data, mimeType, nil
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
