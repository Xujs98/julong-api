package service

import (
	"encoding/base64"
	"os"
	"path/filepath"
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
