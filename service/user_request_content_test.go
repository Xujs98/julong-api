package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeCapturedRequestValueRedactsCredentialsAndEmbeddedMedia(t *testing.T) {
	input := map[string]any{
		"instructions": "review the repository",
		"api_key":      "secret-key",
		"input": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "input_text", "text": "fix the bug"},
					map[string]any{"type": "input_image", "image_data": "data:image/png;base64," + strings.Repeat("a", 2048)},
				},
			},
		},
	}

	sanitized := sanitizeCapturedRequestValue("", input)
	encoded, err := common.Marshal(sanitized)
	require.NoError(t, err)

	text := string(encoded)
	assert.Contains(t, text, "review the repository")
	assert.Contains(t, text, "fix the bug")
	assert.Contains(t, text, "[REDACTED]")
	assert.Contains(t, text, "[OMITTED EMBEDDED MEDIA:")
	assert.NotContains(t, text, "secret-key")
	assert.NotContains(t, text, "data:image/png;base64")
}

func TestTruncateCapturedRequestErrorUsesRuneBoundary(t *testing.T) {
	message := "错误" + strings.Repeat("文", 2100)
	truncated := truncateCapturedRequestError(message)

	assert.LessOrEqual(t, len([]rune(truncated)), 2003)
	assert.Contains(t, truncated, "错误")
	assert.Contains(t, truncated, "...")
}

func TestSanitizedTruncatedPreviewCanStayWithinCaptureLimit(t *testing.T) {
	sanitized := []byte(`{"prompt":"` + strings.Repeat(`\"`, maxCapturedRequestContentBytes) + `"}`)
	encoded, err := truncateCapturedRequestContent(sanitized)
	require.NoError(t, err)

	assert.LessOrEqual(t, len(encoded), maxCapturedRequestContentBytes)
	assert.Contains(t, string(encoded), "_capture_notice")
	assert.Contains(t, string(encoded), "_preview")
}

func TestTaskRequestSnapshotRedactsCredentialsSignedURLsAndMedia(t *testing.T) {
	snapshot, truncated := TaskRequestSnapshot(map[string]any{
		"model":         "seedance",
		"prompt":        "a short prompt",
		"api_key":       "secret-key",
		"authorization": "Bearer secret-token",
		"video_url":     "https://cdn.example/video.mp4?x-amz-signature=secret",
		"input_image":   "data:image/png;base64," + strings.Repeat("x", 2048),
	})
	require.False(t, truncated)
	assert.LessOrEqual(t, len(snapshot), 32<<10)
	text := string(snapshot)
	assert.Contains(t, text, "[REDACTED]")
	assert.Contains(t, text, "[OMITTED EMBEDDED MEDIA:")
	assert.NotContains(t, text, "secret-key")
	assert.NotContains(t, text, "secret-token")
	assert.NotContains(t, text, "x-amz-signature=secret")
}

func TestTaskRequestSnapshotBoundsLargePayload(t *testing.T) {
	snapshot, truncated := TaskRequestSnapshot(map[string]any{
		"prompt": strings.Repeat("x", 128<<10),
	})
	require.True(t, truncated)
	assert.LessOrEqual(t, len(snapshot), 32<<10)
}
