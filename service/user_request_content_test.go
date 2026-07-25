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
