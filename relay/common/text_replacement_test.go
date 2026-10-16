package common

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/require"
)

func TestApplyTextReplacementsJSONOnlyChangesTextFields(t *testing.T) {
	config := dto.TextReplacementConfig{
		Request: []dto.TextReplacementRule{{From: "矩龙API启动", To: "By倾城"}},
	}
	input := []byte(`{"model":"矩龙API启动","messages":[{"role":"user","content":"矩龙API启动"}],"tools":[{"name":"矩龙API启动"}]}`)

	output, err := ApplyTextReplacementsJSON(input, config, false)
	require.NoError(t, err)
	require.JSONEq(t, `{"model":"矩龙API启动","messages":[{"role":"user","content":"By倾城"}],"tools":[{"name":"矩龙API启动"}]}`, string(output))
}

func TestTransformResponseBodyReplacesNonStreamingJSON(t *testing.T) {
	response := &http.Response{
		Header: http.Header{"Content-Type": []string{"application/json"}},
		Body:   http.NoBody,
	}
	response.Body = io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"By倾城"}}]}`))
	config := dto.TextReplacementConfig{
		Response: []dto.TextReplacementRule{{From: "By倾城", To: "矩龙API启动"}},
	}

	require.NoError(t, TransformResponseBody(response, config))
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"choices":[{"message":{"content":"矩龙API启动"}}]}`, string(body))
}

func TestTransformResponseBodyLeavesStreamingResponseForScanner(t *testing.T) {
	response := &http.Response{
		Header: http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:   io.NopCloser(strings.NewReader(`data: {"text":"By倾城"}`)),
	}
	config := dto.TextReplacementConfig{
		Response: []dto.TextReplacementRule{{From: "By倾城", To: "矩龙API启动"}},
	}

	require.NoError(t, TransformResponseBody(response, config))
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, `data: {"text":"By倾城"}`, string(body))
}
