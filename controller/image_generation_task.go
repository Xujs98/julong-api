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

	"github.com/QuantumNous/new-api/common"
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
	if !request.IsAsync() {
		Relay(c, types.RelayFormatOpenAIImage)
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
		"task_id":    task.TaskId,
		"object":     "image.generation.task",
		"status":     model.ImageGenerationStatusPending,
		"created_at": task.CreatedAt,
		"poll_url":   "/v1/images/generations/" + task.TaskId,
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
	log, err := model.GetUserImageGenerationLogByTaskId(c.GetInt("id"), strings.TrimSpace(c.Param("task_id")))
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
