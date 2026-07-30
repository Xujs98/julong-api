package controller

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func ExportLedgerEntries(c *gin.Context) {
	startTimestamp, endTimestamp, ok := ledgerDateRange(c)
	if !ok {
		return
	}
	window := getUsageLogExportWindow(c)
	entries, total, err := model.GetLedgerEntries(startTimestamp, endTimestamp, window.startIdx, window.pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	payload, err := renderLedgerCSV(entries)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	filename := fmt.Sprintf("ledger-%s.csv", time.Now().UTC().Format("20060102-150405"))
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("X-Export-Row-Limit", strconv.Itoa(window.pageSize))
	c.Header("X-Export-Truncated", strconv.FormatBool(!window.paginated && total > int64(len(entries))))
	c.Data(200, "text/csv; charset=utf-8", payload)
}

func renderLedgerCSV(entries []model.LedgerEntry) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"账本 ID", "账本日期", "平台", "账号", "邮箱", "类型", "额度", "成本价（USD）", "数量", "创建人 ID", "创建时间", "更新时间"}); err != nil {
		return nil, err
	}
	for _, entry := range entries {
		row := []string{
			strconv.Itoa(entry.Id),
			time.Unix(entry.OccurredAt, 0).UTC().Format(time.RFC3339),
			entry.Platform,
			entry.Account,
			entry.Email,
			entry.Type,
			strconv.Itoa(entry.Quota),
			entry.CostPrice.StringFixed(6),
			strconv.Itoa(entry.Quantity),
			strconv.Itoa(entry.CreatedBy),
			time.Unix(entry.CreatedAt, 0).UTC().Format(time.RFC3339),
			time.Unix(entry.UpdatedAt, 0).UTC().Format(time.RFC3339),
		}
		for index := range row {
			row[index] = safeCSVCell(row[index])
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buffer.Bytes(), writer.Error()
}
