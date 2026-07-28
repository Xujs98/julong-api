package controller

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

const usageLogExportLimit = 5000

type usageLogExportWindow struct {
	startIdx  int
	pageSize  int
	paginated bool
}

func ExportAllLogs(c *gin.Context) {
	exportUsageLogs(c, true)
}

func ExportUserLogs(c *gin.Context) {
	exportUsageLogs(c, false)
}

func exportUsageLogs(c *gin.Context, isAdmin bool) {
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	modelName := c.Query("model_name")
	tokenName := c.Query("token_name")
	group := c.Query("group")
	requestId := c.Query("request_id")
	upstreamRequestId := c.Query("upstream_request_id")
	exportWindow := getUsageLogExportWindow(c)

	var logs []*model.Log
	var total int64
	var err error
	if isAdmin {
		channel, _ := strconv.Atoi(c.Query("channel"))
		logs, total, err = model.GetAllLogs(
			logType,
			startTimestamp,
			endTimestamp,
			modelName,
			c.Query("username"),
			tokenName,
			exportWindow.startIdx,
			exportWindow.pageSize,
			channel,
			group,
			requestId,
			upstreamRequestId,
		)
	} else {
		logs, total, err = model.GetUserLogs(
			c.GetInt("id"),
			logType,
			startTimestamp,
			endTimestamp,
			modelName,
			tokenName,
			exportWindow.startIdx,
			exportWindow.pageSize,
			group,
			requestId,
			upstreamRequestId,
		)
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}

	payload, err := renderUsageLogsCSV(logs)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	filename := fmt.Sprintf("usage-logs-%s.csv", time.Now().UTC().Format("20060102-150405"))
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("X-Export-Row-Limit", strconv.Itoa(exportWindow.pageSize))
	c.Header("X-Export-Truncated", strconv.FormatBool(!exportWindow.paginated && total > int64(len(logs))))
	c.Data(200, "text/csv; charset=utf-8", payload)
}

func getUsageLogExportWindow(c *gin.Context) usageLogExportWindow {
	window := usageLogExportWindow{pageSize: usageLogExportLimit}
	page, pageErr := strconv.Atoi(c.Query("p"))
	pageSize, pageSizeErr := strconv.Atoi(c.Query("page_size"))
	if pageErr != nil || pageSizeErr != nil || page < 1 || pageSize < 1 {
		return window
	}
	if pageSize > usageLogExportLimit {
		pageSize = usageLogExportLimit
	}
	maxInt := int(^uint(0) >> 1)
	if page-1 > maxInt/pageSize {
		window.startIdx = maxInt
	} else {
		window.startIdx = (page - 1) * pageSize
	}
	window.pageSize = pageSize
	window.paginated = true
	return window
}

func renderUsageLogsCSV(logs []*model.Log) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{
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
	}); err != nil {
		return nil, err
	}
	for _, log := range logs {
		if log == nil {
			continue
		}
		row := []string{
			strconv.Itoa(log.Id),
			time.Unix(log.CreatedAt, 0).UTC().Format(time.RFC3339),
			usageLogTypeName(log.Type),
			log.Username,
			log.TokenName,
			log.ModelName,
			log.Group,
			strconv.Itoa(log.PromptTokens),
			strconv.Itoa(log.CompletionTokens),
			strconv.Itoa(log.Quota),
			strconv.Itoa(log.ChannelId),
			log.ChannelName,
			log.Ip,
			strconv.FormatBool(log.IsStream),
			strconv.Itoa(log.UseTime),
			log.RequestId,
			log.UpstreamRequestId,
			log.Content,
			log.Other,
		}
		for index := range row {
			row[index] = safeCSVCell(row[index])
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func safeCSVCell(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed == "" {
		return value
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		return "'" + value
	default:
		return value
	}
}

func usageLogTypeName(logType int) string {
	switch logType {
	case model.LogTypeTopup:
		return "topup"
	case model.LogTypeConsume:
		return "consume"
	case model.LogTypeManage:
		return "manage"
	case model.LogTypeSystem:
		return "system"
	case model.LogTypeError:
		return "error"
	case model.LogTypeRefund:
		return "refund"
	case model.LogTypeLogin:
		return "login"
	case model.LogTypeQuotaIncrease:
		return "quota_increase"
	default:
		return "unknown"
	}
}
