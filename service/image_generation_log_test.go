package service

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func TestPersistImageGenerationResultStoresDecodedFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("IMAGE_LOG_STORAGE_DIR", dir)
	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	}
	ref, err := persistImageGenerationResult(dto.ImageData{
		B64Json: base64.StdEncoding.EncodeToString(png),
	})
	if err != nil {
		t.Fatalf("persist image: %v", err)
	}
	if ref.Type != "local" || ref.Value == "" {
		t.Fatalf("unexpected ref: %+v", ref)
	}
	stored, err := os.ReadFile(filepath.Join(dir, ref.Value))
	if err != nil {
		t.Fatalf("read stored image: %v", err)
	}
	if string(stored) != string(png) {
		t.Fatal("stored image differs from decoded input")
	}
}

func TestRecordImageGenerationLogStoresMetadataAndFileReference(t *testing.T) {
	truncate(t)
	seedUser(t, 701, 100000)
	seedToken(t, 702, 701, "image-log-token", 100000)
	dir := t.TempDir()
	t.Setenv("IMAGE_LOG_STORAGE_DIR", dir)
	previousEnabled := common.ImageGenerationLogEnabled
	common.ImageGenerationLogEnabled = true
	t.Cleanup(func() { common.ImageGenerationLogEnabled = previousEnabled })

	c, _ := gin.CreateTestContext(nil)
	c.Set("username", "test_user")
	c.Set(common.RequestIdKey, "req_image_log_test")
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	CaptureImageGenerationResult(c, []dto.ImageData{{
		B64Json: base64.StdEncoding.EncodeToString(png),
	}})
	info := &relaycommon.RelayInfo{
		UserId:          701,
		TokenId:         702,
		OriginModelName: "gpt-image-2",
		StartTime:       time.Now().Add(-time.Second),
		PriceData:       types.PriceData{Quota: 1234},
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: 703},
	}
	RecordImageGenerationLog(c, info, &dto.ImageRequest{
		Prompt:  "minimal product poster",
		Size:    "1024x1024",
		Quality: "high",
	}, 1234)

	var log model.ImageGenerationLog
	if err := model.DB.Where("request_id = ?", "req_image_log_test").First(&log).Error; err != nil {
		t.Fatalf("find image generation log: %v", err)
	}
	if log.ModelName != "gpt-image-2" || log.ImageCount != 1 || log.Quota != 1234 {
		t.Fatalf("unexpected image generation log: %+v", log)
	}
	refs, err := log.ImageRefs()
	if err != nil || len(refs) != 1 || refs[0].Type != "local" {
		t.Fatalf("unexpected image refs: %+v, err=%v", refs, err)
	}
}

func TestPersistImageGenerationResultKeepsRemoteReference(t *testing.T) {
	ref, err := persistImageGenerationResult(dto.ImageData{
		Url: "https://example.com/generated.png",
	})
	if err != nil {
		t.Fatalf("persist remote image: %v", err)
	}
	if ref.Type != "remote" || ref.Value != "https://example.com/generated.png" {
		t.Fatalf("unexpected ref: %+v", ref)
	}
}

func TestCaptureResponsesImageGenerationResult(t *testing.T) {
	previousEnabled := common.ImageGenerationLogEnabled
	common.ImageGenerationLogEnabled = true
	t.Cleanup(func() { common.ImageGenerationLogEnabled = previousEnabled })

	c, _ := gin.CreateTestContext(nil)
	encoded := base64.StdEncoding.EncodeToString([]byte{0x89, 0x50, 0x4e, 0x47})
	CaptureResponsesImageGenerationResult(c, &dto.OpenAIResponsesResponse{
		Output: []dto.ResponsesOutput{{
			Type:    dto.ResponsesOutputTypeImageGenerationCall,
			Result:  encoded,
			Quality: "high",
			Size:    "1024x1024",
		}},
	})

	value, exists := c.Get(imageGenerationResultKey)
	images, ok := value.([]dto.ImageData)
	if !exists || !ok || len(images) != 1 || images[0].B64Json != encoded {
		t.Fatalf("unexpected captured images: %#v", value)
	}
	if c.GetString("image_generation_call_quality") != "high" || c.GetString("image_generation_call_size") != "1024x1024" {
		t.Fatal("responses image metadata was not captured")
	}
}

func TestImageGenerationTaskLifecycleStoresSanitizedResponse(t *testing.T) {
	truncate(t)
	seedUser(t, 711, 100000)
	seedToken(t, 712, 711, "image-task-token", 100000)
	t.Setenv("IMAGE_LOG_STORAGE_DIR", t.TempDir())

	c, _ := gin.CreateTestContext(nil)
	c.Set("id", 711)
	c.Set("token_id", 712)
	c.Set("token_name", "image-task-token")
	c.Set(common.RequestIdKey, "req_image_task_test")
	task, err := CreateImageGenerationTask(c, &dto.ImageRequest{
		Model:  "gpt-image-2",
		Prompt: "minimal product poster",
		Size:   "1024x1024",
	})
	if err != nil {
		t.Fatalf("create image generation task: %v", err)
	}
	if task.Status != model.ImageGenerationStatusPending || task.TaskId == "" {
		t.Fatalf("unexpected pending task: %+v", task)
	}
	if err := MarkImageGenerationTaskProcessing(task.TaskId, 711); err != nil {
		t.Fatalf("mark processing: %v", err)
	}

	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	encoded := base64.StdEncoding.EncodeToString(png)
	responseBody, _ := json.Marshal(dto.ImageResponse{
		Created: 123456,
		Data:    []dto.ImageData{{B64Json: encoded, RevisedPrompt: "revised"}},
	})
	if err := CompleteImageGenerationTask(task.TaskId, 711, responseBody); err != nil {
		t.Fatalf("complete task: %v", err)
	}

	completed, err := model.GetUserImageGenerationLogByTaskId(711, task.TaskId)
	if err != nil {
		t.Fatalf("reload completed task: %v", err)
	}
	if completed.Status != model.ImageGenerationStatusSuccess || completed.ImageCount != 1 {
		t.Fatalf("unexpected completed task: %+v", completed)
	}
	if strings.Contains(completed.Response, encoded) || strings.Contains(completed.Response, "b64_json") {
		t.Fatalf("response still contains base64 image data: %s", completed.Response)
	}
	payload, err := BuildImageGenerationTaskPayload(completed, func(index int) string {
		return "/v1/images/generations/" + completed.TaskId + "/images/0"
	})
	if err != nil {
		t.Fatalf("build task payload: %v", err)
	}
	data := payload["data"].([]map[string]any)
	if data[0]["url"] == "" || payload["status"] != model.ImageGenerationStatusSuccess {
		t.Fatalf("unexpected task payload: %+v", payload)
	}
	response := payload["response"].(map[string]any)
	responseData := response["data"].([]map[string]any)
	if responseData[0]["url"] != data[0]["url"] {
		t.Fatalf("response URL was not rewritten: %+v", response)
	}
}

func TestFailImageGenerationTaskStoresError(t *testing.T) {
	truncate(t)
	seedUser(t, 721, 100000)
	seedToken(t, 722, 721, "failed-image-task-token", 100000)
	c, _ := gin.CreateTestContext(nil)
	c.Set("id", 721)
	c.Set("token_id", 722)
	task, err := CreateImageGenerationTask(c, &dto.ImageRequest{Model: "gpt-image-2", Prompt: "poster"})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := FailImageGenerationTask(task.TaskId, 721, []byte(`{"error":{"message":"upstream failed"}}`), "fallback"); err != nil {
		t.Fatalf("fail task: %v", err)
	}
	failed, err := model.GetUserImageGenerationLogByTaskId(721, task.TaskId)
	if err != nil {
		t.Fatalf("reload failed task: %v", err)
	}
	if failed.Status != model.ImageGenerationStatusFailed || failed.ErrorMessage != "upstream failed" {
		t.Fatalf("unexpected failed task: %+v", failed)
	}
}
