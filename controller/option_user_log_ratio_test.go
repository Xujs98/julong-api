package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestUpdateOptionRejectsInvalidUserLogGroupRatioDisplaySettings(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		message string
	}{
		{
			name:    "unknown display mode",
			body:    `{"key":"UserLogGroupRatioDisplayMode","value":"unknown"}`,
			message: "用户日志倍率展示模式无效",
		},
		{
			name:    "negative manual ratio",
			body:    `{"key":"UserLogGroupRatioManualValue","value":-0.01}`,
			message: "手动展示倍率必须是大于等于 0 的有效数字",
		},
		{
			name:    "unknown model square display mode",
			body:    `{"key":"ModelSquareGroupRatioDisplayMode","value":"unknown"}`,
			message: "模型广场倍率展示模式无效",
		},
		{
			name:    "unknown token group display mode",
			body:    `{"key":"TokenGroupRatioDisplayMode","value":"unknown"}`,
			message: "令牌分组倍率展示模式无效",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = httptest.NewRequest(
				http.MethodPut,
				"/api/option/",
				strings.NewReader(test.body),
			)
			context.Set("role", common.RoleRootUser)

			UpdateOption(context)

			assert.Equal(t, http.StatusOK, response.Code)
			assert.JSONEq(
				t,
				`{"success":false,"message":"`+test.message+`"}`,
				response.Body.String(),
			)
		})
	}
}
