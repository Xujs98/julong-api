package controller

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRelayImageGenerationAsyncLifecycle(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Token{}, &model.ImageGenerationLog{}))
	require.NoError(t, db.Create(&model.User{Id: 801, Username: "image_task_user", Status: common.UserStatusEnabled}).Error)
	previousEnabled := common.ImageGenerationLogEnabled
	common.ImageGenerationLogEnabled = true
	t.Cleanup(func() { common.ImageGenerationLogEnabled = previousEnabled })

	originalRunner := imageGenerationRelayRunner
	requestBodies := make(chan string, 1)
	imageGenerationRelayRunner = func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		requestBodies <- string(body)
		c.JSON(http.StatusOK, dto.ImageResponse{
			Created: time.Now().Unix(),
			Data: []dto.ImageData{{
				Url:           "https://example.com/generated.png",
				RevisedPrompt: "revised prompt",
			}},
		})
	}
	t.Cleanup(func() { imageGenerationRelayRunner = originalRunner })

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(
		`{"model":"gpt-image-2","prompt":"minimal poster","async":true}`,
	))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 801)
	c.Set("token_id", 802)
	c.Set("token_name", "image-task-token")
	c.Set(common.RequestIdKey, "req_async_image_controller")
	t.Cleanup(func() { common.CleanupBodyStorage(c) })

	RelayImageGeneration(c)
	require.Equal(t, http.StatusAccepted, recorder.Code)
	var submitted struct {
		TaskID                 string `json:"task_id"`
		Status                 string `json:"status"`
		PollingIntervalSeconds int    `json:"polling_interval_seconds"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &submitted))
	require.NotEmpty(t, submitted.TaskID)
	require.Equal(t, model.ImageGenerationStatusPending, submitted.Status)
	require.Equal(t, common.ImageGenerationLogPollingIntervalSeconds, submitted.PollingIntervalSeconds)

	select {
	case body := <-requestBodies:
		require.NotContains(t, body, `"async"`)
	case <-time.After(2 * time.Second):
		t.Fatal("background image generation relay did not run")
	}

	var task *model.ImageGenerationLog
	require.Eventually(t, func() bool {
		var err error
		task, err = model.GetUserImageGenerationLogByTaskId(801, submitted.TaskID)
		return err == nil && task.Status == model.ImageGenerationStatusSuccess
	}, 2*time.Second, 10*time.Millisecond)
	require.Equal(t, 1, task.ImageCount)
	require.NotContains(t, task.Response, "async")

	pollRecorder := httptest.NewRecorder()
	pollContext, _ := gin.CreateTestContext(pollRecorder)
	pollContext.Set("id", 801)
	pollContext.Params = gin.Params{{Key: "task_id", Value: submitted.TaskID}}
	GetImageGenerationTask(pollContext)
	require.Equal(t, http.StatusOK, pollRecorder.Code)
	require.Contains(t, pollRecorder.Body.String(), `"status":"success"`)
	require.Contains(t, pollRecorder.Body.String(), "https://example.com/generated.png")
}

func TestGetImageGenerationTaskEnforcesOwner(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.ImageGenerationLog{}))
	require.NoError(t, db.Create(&model.User{Id: 811, Username: "task_owner", Status: common.UserStatusEnabled}).Error)
	require.NoError(t, db.Create(&model.ImageGenerationLog{
		TaskId:    "img_owner_only",
		Status:    model.ImageGenerationStatusPending,
		UserId:    811,
		Username:  "task_owner",
		ModelName: "gpt-image-2",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}).Error)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("id", 812)
	c.Params = gin.Params{{Key: "task_id", Value: "img_owner_only"}}
	GetImageGenerationTask(c)
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestGetImageGenerationTaskImageAllowsWhitelistBypass(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.ImageGenerationLog{}))

	storageDir := t.TempDir()
	t.Setenv("IMAGE_LOG_STORAGE_DIR", storageDir)
	require.NoError(t, os.WriteFile(storageDir+"/whitelist.png", []byte("image-bytes"), 0o600))
	require.NoError(t, db.Create(&model.ImageGenerationLog{
		TaskId:    "img_whitelist_read",
		Status:    model.ImageGenerationStatusSuccess,
		UserId:    821,
		Images:    `[{"type":"local","value":"whitelist.png","mime_type":"image/png"}]`,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}).Error)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("id", 999)
	c.Set(string(constant.ContextKeyImageGenerationImageAuthBypassed), true)
	c.Params = gin.Params{
		{Key: "task_id", Value: "img_whitelist_read"},
		{Key: "index", Value: "0"},
	}

	GetImageGenerationTaskImage(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "image/png", recorder.Header().Get("Content-Type"))
	require.Equal(t, "image-bytes", recorder.Body.String())
}

func TestRelayImageGenerationRejectsEmptyPrompt(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/images/generations",
		strings.NewReader(`{"model":"gpt-image-2","async":true}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")
	t.Cleanup(func() { common.CleanupBodyStorage(c) })

	RelayImageGeneration(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "prompt is required")
}

func TestRelayImageGenerationIgnoresAsyncWhenLoggingDisabled(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.ImageGenerationLog{}))

	previousEnabled := common.ImageGenerationLogEnabled
	common.ImageGenerationLogEnabled = false
	t.Cleanup(func() { common.ImageGenerationLogEnabled = previousEnabled })

	originalRunner := imageGenerationRelayRunner
	called := false
	imageGenerationRelayRunner = func(c *gin.Context) {
		called = true
		c.JSON(http.StatusOK, dto.ImageResponse{
			Created: time.Now().Unix(),
			Data:    []dto.ImageData{{Url: "https://example.com/synchronous.png"}},
		})
	}
	t.Cleanup(func() { imageGenerationRelayRunner = originalRunner })

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/images/generations",
		strings.NewReader(`{"model":"gpt-image-2","prompt":"minimal poster","async":true}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")
	t.Cleanup(func() { common.CleanupBodyStorage(c) })

	RelayImageGeneration(c)

	require.True(t, called)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotContains(t, recorder.Body.String(), `"task_id"`)
	var taskCount int64
	require.NoError(t, db.Model(&model.ImageGenerationLog{}).Count(&taskCount).Error)
	require.Zero(t, taskCount)
}
