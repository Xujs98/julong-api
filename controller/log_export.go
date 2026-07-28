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
			0,
			usageLogExportLimit,
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
			0,
			usageLogExportLimit,
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
	c.Header("X-Export-Row-Limit", strconv.Itoa(usageLogExportLimit))
	c.Header("X-Export-Truncated", strconv.FormatBool(total > int64(len(logs))))
	c.Data(200, "text/csv; charset=utf-8", payload)
}

func renderUsageLogsCSV(logs []*model.Log) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{
		"ID",
		"Time",
		"Type",
		"Username",
		"Token Name",
		"Model",
		"Group",
		"Prompt Tokens",
		"Completion Tokens",
		"Quota",
		"Channel ID",
		"Channel Name",
		"IP",
		"Stream",
		"Duration Seconds",
		"Request ID",
		"Upstream Request ID",
		"Content",
		"Other",
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
