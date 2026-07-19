package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

const maxConcurrentImageGenerationTasks = 16

var imageGenerationTaskSlots = make(chan struct{}, maxConcurrentImageGenerationTasks)
var imageGenerationRelayRunner = func(c *gin.Context) {
	Relay(c, types.RelayFormatOpenAIImage)
}

func RelayImageGeneration(c *gin.Context) {
	request, err := helper.GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesGenerations)
	if err != nil {
		imageGenerationOpenAIError(c, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(request.Prompt) == "" {
		imageGenerationOpenAIError(c, http.StatusBadRequest, "prompt is required")
		return
	}
	if !request.IsAsync() || !common.ImageGenerationLogEnabled {
		imageGenerationRelayRunner(c)
		return
	}
	if request.Stream != nil && *request.Stream {
		imageGenerationOpenAIError(c, http.StatusBadRequest, "async and stream cannot both be true")
		return
	}

	requestBody, err := imageGenerationRequestWithoutAsync(c)
	if err != nil {
		imageGenerationOpenAIError(c, http.StatusBadRequest, err.Error())
		return
	}
	task, err := service.CreateImageGenerationTask(c, request)
	if err != nil {
		imageGenerationOpenAIError(c, http.StatusInternalServerError, "failed to create image generation task")
		return
	}

	backgroundContext, recorder := newImageGenerationBackgroundContext(c, requestBody, task.TaskId)
	c.JSON(http.StatusAccepted, gin.H{
		"task_id":                  task.TaskId,
		"object":                   "image.generation.task",
		"status":                   model.ImageGenerationStatusPending,
		"created_at":               task.CreatedAt,
		"poll_url":                 "/v1/images/generations/" + task.TaskId,
		"polling_interval_seconds": common.ImageGenerationLogPollingIntervalSeconds,
	})

	go runImageGenerationTask(backgroundContext, recorder, task)
}

func runImageGenerationTask(c *gin.Context, recorder *httptest.ResponseRecorder, task *model.ImageGenerationLog) {
	imageGenerationTaskSlots <- struct{}{}
	defer func() {
		<-imageGenerationTaskSlots
		common.CleanupBodyStorage(c)
		service.CleanupFileSources(c)
		if recovered := recover(); recovered != nil {
			message := fmt.Sprintf("image generation task panic: %v", recovered)
			_ = service.FailImageGenerationTask(task.TaskId, task.UserId, nil, message)
		}
	}()

	if err := service.MarkImageGenerationTaskProcessing(task.TaskId, task.UserId); err != nil {
		return
	}
	imageGenerationRelayRunner(c)
	responseBody := recorder.Body.Bytes()
	if recorder.Code < http.StatusOK || recorder.Code >= http.StatusMultipleChoices {
		_ = service.FailImageGenerationTask(task.TaskId, task.UserId, responseBody, recorder.Result().Status)
		return
	}
	if err := service.CompleteImageGenerationTask(task.TaskId, task.UserId, responseBody); err != nil {
		_ = service.FailImageGenerationTask(task.TaskId, task.UserId, responseBody, err.Error())
	}
}

func imageGenerationRequestWithoutAsync(c *gin.Context) ([]byte, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, err
	}
	requestBody, err := storage.Bytes()
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := common.Unmarshal(requestBody, &payload); err != nil {
		return nil, err
	}
	delete(payload, "async")
	return common.Marshal(payload)
}

func newImageGenerationBackgroundContext(c *gin.Context, requestBody []byte, taskId string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	backgroundContext, _ := gin.CreateTestContext(recorder)
	backgroundContext.Request = c.Request.Clone(context.Background())
	backgroundContext.Request.Body = io.NopCloser(bytes.NewReader(requestBody))
	backgroundContext.Request.ContentLength = int64(len(requestBody))
	backgroundContext.Request.Header = c.Request.Header.Clone()

	contextCopy := c.Copy()
	for key, value := range contextCopy.Keys {
		if key != common.KeyBodyStorage && key != common.KeyRequestBody {
			backgroundContext.Set(key, value)
		}
	}
	backgroundContext.Set(service.ImageGenerationTaskContextKey, taskId)
	return backgroundContext, recorder
}

func GetImageGenerationTask(c *gin.Context) {
	taskId := strings.TrimSpace(c.Param("task_id"))
	log, err := model.GetUserImageGenerationLogByTaskId(c.GetInt("id"), taskId)
	if err != nil {
		imageGenerationOpenAIError(c, http.StatusNotFound, "image generation task not found")
		return
	}
	payload, err := service.BuildImageGenerationTaskPayload(log, func(index int) string {
		return "/v1/images/generations/" + log.TaskId + "/images/" + fmt.Sprintf("%d", index)
	})
	if err != nil {
		imageGenerationOpenAIError(c, http.StatusInternalServerError, "failed to read image generation task")
		return
	}
	c.JSON(http.StatusOK, payload)
}

func GetImageGenerationTaskImage(c *gin.Context) {
	index, err := parseImageGenerationIndex(c.Param("index"))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	taskId := strings.TrimSpace(c.Param("task_id"))
	var log *model.ImageGenerationLog
	if c.GetBool(string(constant.ContextKeyImageGenerationImageAuthBypassed)) {
		log, err = model.GetImageGenerationLogByTaskId(taskId)
	} else {
		log, err = model.GetUserImageGenerationLogByTaskId(c.GetInt("id"), taskId)
	}
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	data, mimeType, err := service.ReadImageGenerationLogImage(log, index)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.Data(http.StatusOK, mimeType, data)
}

func GetImageGenerationTaskImagePresign(c *gin.Context) {
	index, err := parseImageGenerationIndex(c.Param("index"))
	if err != nil {
		imageGenerationOpenAIError(c, http.StatusBadRequest, "invalid image index")
		return
	}
	taskId := strings.TrimSpace(c.Param("task_id"))
	log, err := model.GetUserImageGenerationLogByTaskId(c.GetInt("id"), taskId)
	if err != nil {
		imageGenerationOpenAIError(c, http.StatusNotFound, "image generation task not found")
		return
	}
	refs, err := log.ImageRefs()
	if err != nil || index >= len(refs) {
		imageGenerationOpenAIError(c, http.StatusNotFound, "image not found")
		return
	}
	ref := refs[index]
	if ref.Type != "minio" {
		imageGenerationOpenAIError(c, http.StatusConflict, "image is not stored in MinIO")
		return
	}
	expiresIn := 3600
	if raw := strings.TrimSpace(c.Query("expires_in")); raw != "" {
		value, parseErr := strconv.Atoi(raw)
		if parseErr != nil || value < 60 || value > 86400 {
			imageGenerationOpenAIError(c, http.StatusBadRequest, "expires_in must be between 60 and 86400 seconds")
			return
		}
		expiresIn = value
	}
	url, expiresAt, err := service.PresignGeneratedImageObject(
		c.Request.Context(),
		ref.Bucket,
		ref.Value,
		time.Duration(expiresIn)*time.Second,
	)
	if err != nil {
		imageGenerationOpenAIError(c, http.StatusInternalServerError, "failed to create temporary image URL")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"url":        url,
		"expires_in": expiresIn,
		"expires_at": expiresAt.UTC().Format(time.RFC3339),
	})
}

func parseImageGenerationIndex(value string) (int, error) {
	index, err := strconv.Atoi(value)
	if err != nil || index < 0 {
		return 0, fmt.Errorf("invalid image index")
	}
	return index, nil
}

func imageGenerationOpenAIError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": gin.H{
		"message": common.MessageWithRequestId(message, c.GetString(common.RequestIdKey)),
		"type":    "invalid_request_error",
		"code":    "image_generation_task_error",
	}})
}
