package common

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	rootcommon "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
)

var textReplacementKeys = map[string]struct{}{
	"content":           {},
	"input":             {},
	"input_text":        {},
	"instructions":      {},
	"output_text":       {},
	"parts":             {},
	"prompt":            {},
	"reasoning_content": {},
	"system":            {},
	"text":              {},
	"text_delta":        {},
	"thinking":          {},
}

func ApplyTextReplacementsJSON(data []byte, config dto.TextReplacementConfig, response bool) ([]byte, error) {
	if config.Empty() || len(data) == 0 {
		return data, nil
	}
	var value any
	if err := rootcommon.Unmarshal(data, &value); err != nil {
		return data, nil
	}
	value = rewriteTextValuesInPlace(value, false, config, response)
	return rootcommon.Marshal(value)
}

func rewriteTextValuesInPlace(value any, active bool, config dto.TextReplacementConfig, response bool) any {
	switch current := value.(type) {
	case map[string]any:
		for key, child := range current {
			_, keyAllowsText := textReplacementKeys[strings.ToLower(key)]
			current[key] = rewriteTextValuesInPlace(child, active || keyAllowsText, config, response)
		}
	case []any:
		for i, child := range current {
			current[i] = rewriteTextValuesInPlace(child, active, config, response)
		}
	case string:
		if active {
			return config.Apply(current, response)
		}
	}
	return value
}

func TransformStoredRequestBody(storage rootcommon.BodyStorage, config dto.TextReplacementConfig) (rootcommon.ReplayableBody, io.Closer, error) {
	if storage == nil {
		return nil, nil, fmt.Errorf("request body storage is nil")
	}
	data, err := storage.Bytes()
	if err != nil {
		return nil, nil, err
	}
	transformed, err := ApplyTextReplacementsJSON(data, config, false)
	if err != nil {
		return nil, nil, err
	}
	if bytes.Equal(data, transformed) {
		// The request context owns the original storage and closes it at the end
		// of the request. Returning it here would let an HTTP handler close the
		// shared storage too early, breaking later billing/usage reads.
		return rootcommon.NewReplayableBodyReader(storage), io.NopCloser(bytes.NewReader(nil)), nil
	}
	return newOutboundTextReplacementBody(transformed)
}

func newOutboundTextReplacementBody(data []byte) (rootcommon.ReplayableBody, io.Closer, error) {
	storage, err := rootcommon.CreateBodyStorage(data)
	if err != nil {
		return nil, nil, err
	}
	return rootcommon.NewReplayableBodyReader(storage), storage, nil
}

func TransformResponseBody(resp *http.Response, config dto.TextReplacementConfig) error {
	if resp == nil || resp.Body == nil || config.Empty() {
		return nil
	}
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		return nil
	}
	data, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return err
	}
	transformed, err := ApplyTextReplacementsJSON(data, config, true)
	if err != nil {
		return err
	}
	resp.Body = io.NopCloser(bytes.NewReader(transformed))
	resp.ContentLength = int64(len(transformed))
	resp.Header.Set("Content-Length", fmt.Sprint(len(transformed)))
	return nil
}
