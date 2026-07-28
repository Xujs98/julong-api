package controller

import (
	"encoding/csv"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderUsageLogsCSVIncludesBOMAndPreventsFormulaInjection(t *testing.T) {
	payload, err := renderUsageLogsCSV([]*model.Log{{
		Id:        7,
		CreatedAt: 1,
		Type:      model.LogTypeConsume,
		Username:  "=HYPERLINK(\"https://example.com\")",
		Content:   "+SUM(1,1)",
	}})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(payload), "\xEF\xBB\xBF"))

	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(payload), "\xEF\xBB\xBF")))
	rows, err := reader.ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, []string{
		"日志 ID",
		"时间",
		"类型",
		"用户名",
		"令牌名称",
		"模型",
		"分组",
		"输入 Token",
		"输出 Token",
		"额度",
		"渠道 ID",
		"渠道名称",
		"IP 地址",
		"流式请求",
		"用时（秒）",
		"请求 ID",
		"上游请求 ID",
		"内容",
		"其他",
	}, rows[0])
	assert.Equal(t, "'=HYPERLINK(\"https://example.com\")", rows[1][3])
	assert.Equal(t, "'+SUM(1,1)", rows[1][17])
	assert.Equal(t, "consume", rows[1][2])
}

func TestUsageLogExportWindowHonorsPageParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("without pagination", func(t *testing.T) {
		context, _ := gin.CreateTestContext(httptest.NewRecorder())
		context.Request = httptest.NewRequest("GET", "/api/log/export", nil)

		window := getUsageLogExportWindow(context)

		assert.Equal(t, 0, window.startIdx)
		assert.Equal(t, usageLogExportLimit, window.pageSize)
		assert.False(t, window.paginated)
	})

	t.Run("current page", func(t *testing.T) {
		context, _ := gin.CreateTestContext(httptest.NewRecorder())
		context.Request = httptest.NewRequest("GET", "/api/log/export?p=3&page_size=100", nil)

		window := getUsageLogExportWindow(context)

		assert.Equal(t, 200, window.startIdx)
		assert.Equal(t, 100, window.pageSize)
		assert.True(t, window.paginated)
	})

	t.Run("page size is bounded", func(t *testing.T) {
		context, _ := gin.CreateTestContext(httptest.NewRecorder())
		context.Request = httptest.NewRequest("GET", "/api/log/export?p=2&page_size=99999", nil)

		window := getUsageLogExportWindow(context)

		assert.Equal(t, usageLogExportLimit, window.startIdx)
		assert.Equal(t, usageLogExportLimit, window.pageSize)
		assert.True(t, window.paginated)
	})
}
