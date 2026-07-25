package service

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
)

const maxCapturedRequestContentBytes = 4 << 20

var capturedRequestSensitiveKeys = map[string]struct{}{
	"access_token":  {},
	"api_key":       {},
	"authorization": {},
	"client_secret": {},
	"password":      {},
	"secret":        {},
}

func CaptureUserRequestContent(c *gin.Context, info *relaycommon.RelayInfo, request dto.Request) bool {
	if c == nil || info == nil || request == nil ||
		!common.GetContextKeyBool(c, constant.ContextKeyUserRequestContentLoggingEnabled) {
		return false
	}

	raw, err := common.Marshal(request)
	if err != nil {
		logger.LogError(c, "failed to marshal request content log: "+err.Error())
		return false
	}
	originalSize := len(raw)

	var value any
	if err := common.Unmarshal(raw, &value); err != nil {
		logger.LogError(c, "failed to parse request content log: "+err.Error())
		return false
	}
	value = sanitizeCapturedRequestValue("", value)
	sanitized, err := common.Marshal(value)
	if err != nil {
		logger.LogError(c, "failed to sanitize request content log: "+err.Error())
		return false
	}

	truncated := len(sanitized) > maxCapturedRequestContentBytes
	if truncated {
		sanitized, err = truncateCapturedRequestContent(sanitized)
		if err != nil {
			logger.LogError(c, "failed to truncate request content log: "+err.Error())
			return false
		}
	}

	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(sanitized); err != nil {
		_ = writer.Close()
		logger.LogError(c, "failed to compress request content log: "+err.Error())
		return false
	}
	if err := writer.Close(); err != nil {
		logger.LogError(c, "failed to close request content log compressor: "+err.Error())
		return false
	}

	log := &model.UserRequestContentLog{
		UserId:         info.UserId,
		RequestId:      c.GetString(common.RequestIdKey),
		ModelName:      info.OriginModelName,
		TokenName:      c.GetString("token_name"),
		RequestPath:    c.Request.URL.Path,
		Status:         model.UserRequestContentStatusPending,
		OriginalSize:   originalSize,
		CapturedSize:   len(sanitized),
		Truncated:      truncated,
		CompressedJSON: compressed.Bytes(),
	}
	if err := model.CreateUserRequestContentLog(log); err != nil {
		logger.LogError(c, "failed to save request content log: "+err.Error())
		return false
	}
	return true
}

func FinishUserRequestContent(requestId string, requestErr error) {
	status := model.UserRequestContentStatusSuccess
	errorMessage := ""
	if requestErr != nil {
		status = model.UserRequestContentStatusError
		errorMessage = truncateCapturedRequestError(common.MaskSensitiveInfo(requestErr.Error()))
	}
	if err := model.FinishUserRequestContentLog(requestId, status, errorMessage); err != nil {
		common.SysError("failed to finish request content log: " + err.Error())
	}
}

func DecodeUserRequestContent(log *model.UserRequestContentLog) (any, error) {
	if log == nil || len(log.CompressedJSON) == 0 {
		return nil, fmt.Errorf("request content is empty")
	}
	reader, err := gzip.NewReader(bytes.NewReader(log.CompressedJSON))
	if err != nil {
		return nil, err
	}
	decompressed, err := io.ReadAll(io.LimitReader(reader, maxCapturedRequestContentBytes*2))
	closeErr := reader.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	var value any
	if err := common.Unmarshal(decompressed, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func sanitizeCapturedRequestValue(key string, value any) any {
	lowerKey := strings.ToLower(strings.TrimSpace(key))
	if _, sensitive := capturedRequestSensitiveKeys[lowerKey]; sensitive {
		if _, isString := value.(string); isString {
			return "[REDACTED]"
		}
	}

	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for childKey, childValue := range typed {
			result[childKey] = sanitizeCapturedRequestValue(childKey, childValue)
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for i, childValue := range typed {
			result[i] = sanitizeCapturedRequestValue(lowerKey, childValue)
		}
		return result
	case string:
		if isEmbeddedCapturedMedia(lowerKey, typed) {
			return fmt.Sprintf("[OMITTED EMBEDDED MEDIA: %d bytes]", len(typed))
		}
		return typed
	default:
		return value
	}
}

func isEmbeddedCapturedMedia(key, value string) bool {
	if len(value) < 1024 {
		return false
	}
	lowerValue := strings.ToLower(value[:min(len(value), 64)])
	if strings.HasPrefix(lowerValue, "data:") && strings.Contains(lowerValue, ";base64,") {
		return true
	}
	switch key {
	case "b64_json", "file_data", "image_data", "audio_data":
		return true
	default:
		return false
	}
}

func truncateCapturedRequestContent(sanitized []byte) ([]byte, error) {
	previewSize := min(len(sanitized), maxCapturedRequestContentBytes)
	for previewSize > 0 {
		value := map[string]any{
			"_capture_notice": "Request content exceeded the capture limit; a partial sanitized preview is stored.",
			"_preview":        string(sanitized[:previewSize]),
		}
		encoded, err := common.Marshal(value)
		if err != nil {
			return nil, err
		}
		if len(encoded) <= maxCapturedRequestContentBytes {
			return encoded, nil
		}
		previewSize = previewSize * 3 / 4
	}
	return nil, fmt.Errorf("request content preview exceeds capture limit")
}

func truncateCapturedRequestError(message string) string {
	runes := []rune(message)
	if len(runes) <= 2000 {
		return message
	}
	return string(runes[:2000]) + "..."
}
