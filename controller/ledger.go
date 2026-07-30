package controller

import (
	"errors"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

const (
	maxLedgerDeleteBatch = 100
	maxLedgerQuantity    = 10000
)

var ledgerPlatforms = map[string]struct{}{
	"Anthropic":   {},
	"OpenAI":      {},
	"Gemini":      {},
	"Antigravity": {},
	"Grok":        {},
}

type ledgerMutationRequest struct {
	Platform   string          `json:"platform"`
	Account    string          `json:"account"`
	Email      string          `json:"email"`
	Type       string          `json:"type"`
	Quota      int             `json:"quota"`
	CostPrice  decimal.Decimal `json:"cost_price"`
	Quantity   int             `json:"quantity"`
	OccurredAt int64           `json:"occurred_at"`
}

func GetLedgerEntries(c *gin.Context) {
	startTimestamp, endTimestamp, ok := ledgerDateRange(c)
	if !ok {
		return
	}
	pageInfo := common.GetPageQuery(c)
	entries, total, err := model.GetLedgerEntries(startTimestamp, endTimestamp, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(entries)
	common.ApiSuccess(c, pageInfo)
}

func GetLedgerSummary(c *gin.Context) {
	startTimestamp, endTimestamp, ok := ledgerDateRange(c)
	if !ok {
		return
	}
	summary, err := service.GetLedgerSummary(startTimestamp, endTimestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}

func GetLedgerSettings(c *gin.Context) {
	common.ApiSuccess(c, model.GetLedgerSettings())
}

func UpdateLedgerSettings(c *gin.Context) {
	var request struct {
		EstimateRatio decimal.Decimal `json:"estimate_ratio"`
	}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiError(c, err)
		return
	}
	settings, err := model.UpdateLedgerSettings(request.EstimateRatio)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	recordManageAudit(c, "ledger.settings.update", map[string]interface{}{
		"estimate_ratio": settings.EstimateRatio.String(),
	})
	common.ApiSuccess(c, settings)
}

func CreateLedgerEntry(c *gin.Context) {
	request, ok := decodeLedgerMutation(c)
	if !ok {
		return
	}
	entry := &model.LedgerEntry{
		Platform:   request.Platform,
		Account:    request.Account,
		Email:      request.Email,
		Type:       request.Type,
		Quota:      request.Quota,
		CostPrice:  request.CostPrice,
		Quantity:   request.Quantity,
		OccurredAt: request.OccurredAt,
		CreatedBy:  c.GetInt("id"),
	}
	if err := model.CreateLedgerEntry(entry); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "ledger.create", map[string]interface{}{
		"id":       entry.Id,
		"platform": entry.Platform,
		"type":     entry.Type,
		"quantity": entry.Quantity,
	})
	common.ApiSuccess(c, entry)
}

func UpdateLedgerEntry(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid ledger entry id")
		return
	}
	entry, err := model.GetLedgerEntryById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	request, ok := decodeLedgerMutation(c)
	if !ok {
		return
	}
	entry.Platform = request.Platform
	entry.Account = request.Account
	entry.Email = request.Email
	entry.Type = request.Type
	entry.Quota = request.Quota
	entry.CostPrice = request.CostPrice
	entry.Quantity = request.Quantity
	entry.OccurredAt = request.OccurredAt
	if err := model.UpdateLedgerEntry(entry); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "ledger.update", map[string]interface{}{
		"id":       entry.Id,
		"platform": entry.Platform,
		"type":     entry.Type,
		"quantity": entry.Quantity,
	})
	common.ApiSuccess(c, entry)
}

func DeleteLedgerEntry(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid ledger entry id")
		return
	}
	deleteLedgerEntries(c, []int{id})
}

func DeleteLedgerEntries(c *gin.Context) {
	var request struct {
		Ids []int `json:"ids"`
	}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiError(c, err)
		return
	}
	deleteLedgerEntries(c, request.Ids)
}

func deleteLedgerEntries(c *gin.Context, ids []int) {
	ids = uniqueLedgerIds(ids)
	if len(ids) == 0 || len(ids) > maxLedgerDeleteBatch {
		common.ApiErrorMsg(c, "ledger entry ids must contain between 1 and 100 unique values")
		return
	}
	deleted, err := model.DeleteLedgerEntries(ids)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "ledger.delete", map[string]interface{}{
		"ids":           ids,
		"deleted_count": deleted,
	})
	common.ApiSuccess(c, deleted)
}

func decodeLedgerMutation(c *gin.Context) (*ledgerMutationRequest, bool) {
	var request ledgerMutationRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiError(c, err)
		return nil, false
	}
	request.Platform = strings.TrimSpace(request.Platform)
	request.Account = strings.TrimSpace(request.Account)
	request.Email = strings.TrimSpace(request.Email)
	request.Type = service.NormalizeLedgerType(request.Type)
	request.CostPrice = request.CostPrice.Round(6)
	if err := validateLedgerMutation(&request); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return nil, false
	}
	return &request, true
}

func validateLedgerMutation(request *ledgerMutationRequest) error {
	if _, ok := ledgerPlatforms[request.Platform]; !ok {
		return errors.New("invalid ledger platform")
	}
	if request.Account == "" || len(request.Account) > 255 {
		return errors.New("ledger account is required and must not exceed 255 characters")
	}
	if len(request.Email) > 255 {
		return errors.New("ledger email must not exceed 255 characters")
	}
	if request.Email != "" {
		address, err := mail.ParseAddress(request.Email)
		if err != nil || address.Address != request.Email {
			return errors.New("invalid ledger email")
		}
	}
	if request.Type == "" || len(request.Type) > 64 {
		return errors.New("ledger type is required and must not exceed 64 characters")
	}
	if request.Quota <= 0 || request.Quota > common.MaxQuota {
		return errors.New("ledger quota is out of range")
	}
	if request.CostPrice.LessThanOrEqual(decimal.Zero) || request.CostPrice.GreaterThan(decimal.NewFromInt(1_000_000_000)) {
		return errors.New("ledger cost price is out of range")
	}
	if request.Quantity < 1 || request.Quantity > maxLedgerQuantity {
		return errors.New("ledger quantity must be between 1 and 10000")
	}
	if request.OccurredAt <= 0 || request.OccurredAt > time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC).Unix() {
		return errors.New("invalid ledger date")
	}
	return nil
}

func ledgerDateRange(c *gin.Context) (int64, int64, bool) {
	startTimestamp, startErr := parseOptionalLedgerTimestamp(c.Query("start_timestamp"))
	endTimestamp, endErr := parseOptionalLedgerTimestamp(c.Query("end_timestamp"))
	if startErr != nil || endErr != nil || (startTimestamp > 0 && endTimestamp > 0 && startTimestamp > endTimestamp) {
		common.ApiErrorMsg(c, "invalid ledger date range")
		return 0, 0, false
	}
	return startTimestamp, endTimestamp, true
}

func parseOptionalLedgerTimestamp(value string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	timestamp, err := strconv.ParseInt(value, 10, 64)
	if err != nil || timestamp <= 0 {
		return 0, errors.New("invalid timestamp")
	}
	return timestamp, nil
}

func uniqueLedgerIds(ids []int) []int {
	result := make([]int, 0, len(ids))
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
