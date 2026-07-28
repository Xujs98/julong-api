package service

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	hosttypes "github.com/QuantumNous/new-api/types"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// ToolSurchargeItem is one billable tool-call line for consume logs.
type ToolSurchargeItem struct {
	Name  string  `json:"name"`
	Count int     `json:"count"`
	Price float64 `json:"price"`
}

func appendToolSurchargeLogInfo(other map[string]interface{}, items []ToolSurchargeItem) {
	if len(items) == 0 {
		return
	}
	other["tool_surcharges"] = items
}

type textQuotaSummary struct {
	PromptTokens              int
	CompletionTokens          int
	TotalTokens               int
	CacheTokens               int
	CacheCreationTokens       int
	CacheCreationTokens5m     int
	CacheCreationTokens1h     int
	ImageTokens               int
	AudioTokens               int
	ModelName                 string
	TokenName                 string
	UseTimeSeconds            int64
	CompletionRatio           float64
	CacheRatio                float64
	ImageRatio                float64
	ModelRatio                float64
	GroupRatio                float64
	ModelPrice                float64
	CacheCreationRatio        float64
	CacheCreationRatio5m      float64
	CacheCreationRatio1h      float64
	Quota                     int
	IsClaudeUsageSemantic     bool
	UsageSemantic             string
	AudioInputPrice           float64
	ToolSurchargeItems        []ToolSurchargeItem
	ToolCallSurchargeQuota    decimal.Decimal
	ToolCallSurchargeBase     decimal.Decimal
	ModelTokenAdjustmentAudit *hosttypes.ModelTokenAdjustmentAudit
	BillingCounterfactuals    billingRevenueCounterfactuals
}

func (s *textQuotaSummary) hasBillableUsage() bool {
	return s.TotalTokens > 0 || !s.ToolCallSurchargeQuota.IsZero()
}

func cacheWriteTokensTotal(summary textQuotaSummary) int {
	if summary.CacheCreationTokens5m > 0 || summary.CacheCreationTokens1h > 0 {
		splitCacheWriteTokens := summary.CacheCreationTokens5m + summary.CacheCreationTokens1h
		if summary.CacheCreationTokens > splitCacheWriteTokens {
			return summary.CacheCreationTokens
		}
		return splitCacheWriteTokens
	}
	return summary.CacheCreationTokens
}

func isLegacyClaudeDerivedOpenAIUsage(relayInfo *relaycommon.RelayInfo, usage *dto.Usage) bool {
	if relayInfo == nil || usage == nil {
		return false
	}
	if relayInfo.GetFinalRequestRelayFormat() == types.RelayFormatClaude {
		return false
	}
	if usage.UsageSource != "" || usage.UsageSemantic != "" {
		return false
	}
	return usage.ClaudeCacheCreation5mTokens > 0 || usage.ClaudeCacheCreation1hTokens > 0
}

func collectToolSurchargeItem(items []ToolSurchargeItem, name string, count int, modelName string) []ToolSurchargeItem {
	if count <= 0 {
		return items
	}
	price := operation_setting.GetToolPriceForModel(name, modelName)
	if price <= 0 || math.IsNaN(price) || math.IsInf(price, 0) {
		return items
	}
	return append(items, ToolSurchargeItem{
		Name:  name,
		Count: count,
		Price: price,
	})
}

func mergeToolSurchargeItems(items []ToolSurchargeItem) []ToolSurchargeItem {
	if len(items) == 0 {
		return nil
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Name == items[j].Name {
			return items[i].Price < items[j].Price
		}
		return items[i].Name < items[j].Name
	})

	merged := items[:0]
	for _, item := range items {
		lastIndex := len(merged) - 1
		if lastIndex >= 0 &&
			merged[lastIndex].Name == item.Name &&
			merged[lastIndex].Price == item.Price {
			if item.Count > math.MaxInt-merged[lastIndex].Count {
				common.SysError("tool surcharge call count overflow for " + item.Name)
				merged[lastIndex].Count = math.MaxInt
			} else {
				merged[lastIndex].Count += item.Count
			}
			continue
		}
		merged = append(merged, item)
	}
	return merged
}

func calculateTextToolCallSurchargeBase(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, summary *textQuotaSummary) decimal.Decimal {
	dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)

	var items []ToolSurchargeItem

	if relayInfo.ResponsesUsageInfo != nil {
		for name, tool := range relayInfo.ResponsesUsageInfo.BuiltInTools {
			if tool == nil {
				continue
			}
			items = collectToolSurchargeItem(items, name, tool.CallCount, summary.ModelName)
		}
	}
	if relayInfo.RelayMode != relayconstant.RelayModeResponses &&
		strings.HasSuffix(summary.ModelName, "search-preview") {
		items = collectToolSurchargeItem(items, dto.BuildInToolWebSearchPreview, 1, summary.ModelName)
	}

	items = collectToolSurchargeItem(
		items,
		dto.BuildInToolWebSearch,
		ctx.GetInt("claude_web_search_requests"),
		summary.ModelName,
	)

	if ctx.GetBool("gemini_google_search_call") {
		items = collectToolSurchargeItem(items, dto.BuildInToolGoogleSearch, 1, summary.ModelName)
	}

	summary.ToolSurchargeItems = mergeToolSurchargeItems(items)
	var surcharge decimal.Decimal
	for _, item := range summary.ToolSurchargeItems {
		surcharge = surcharge.Add(decimal.NewFromFloat(item.Price).
			Mul(decimal.NewFromInt(int64(item.Count))).
			Div(decimal.NewFromInt(1000)).
			Mul(dQuotaPerUnit))
	}

	return surcharge
}

func finalizeTextQuota(quota decimal.Decimal, ratio decimal.Decimal, billable bool) (int, *common.QuotaClamp) {
	value, clamp := common.QuotaFromDecimalChecked(quota)
	if !billable {
		return 0, clamp
	}
	if !ratio.IsZero() && value <= 0 {
		return 1, clamp
	}
	return value, clamp
}

// noteQuotaClamp records the first quota saturation event onto relayInfo so it
// can later be attached to the consume/task log for admin auditing. First
// non-nil clamp wins (a single request may hit multiple conversions).
func noteQuotaClamp(relayInfo *relaycommon.RelayInfo, clamp *common.QuotaClamp) {
	if clamp == nil || relayInfo == nil {
		return
	}
	if relayInfo.QuotaClamp == nil {
		relayInfo.QuotaClamp = clamp
	}
}

func composeTieredTextQuota(relayInfo *relaycommon.RelayInfo, summary textQuotaSummary, tieredQuota int, tieredResult *billingexpr.TieredResult) int {
	if summary.ToolCallSurchargeQuota.IsZero() {
		return tieredQuota
	}

	if tieredResult != nil {
		if snap := relayInfo.TieredBillingSnapshot; snap != nil {
			quota, clamp := common.QuotaFromDecimalChecked(decimal.NewFromFloat(tieredResult.ActualQuotaBeforeGroup).
				Mul(decimal.NewFromFloat(snap.GroupRatio)).
				Add(summary.ToolCallSurchargeQuota))
			noteQuotaClamp(relayInfo, clamp)
			return quota
		}
	}

	// Saturate the final sum, not just the surcharge: tieredQuota can be near
	// MaxQuota and adding the surcharge could push the total past the int32
	// quota policy bound (persisted quota columns are 32-bit).
	total, clamp := common.QuotaFromDecimalChecked(
		decimal.NewFromInt(int64(tieredQuota)).Add(summary.ToolCallSurchargeQuota),
	)
	noteQuotaClamp(relayInfo, clamp)
	return total
}

// calculateTextQuotaSummary expects a usage already remapped by
// effectiveBillingUsage; PostTextConsumeQuota performs that remap once and shares
// the result with tiered billing, affinity observation and logging.
func calculateTextQuotaSummary(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage) textQuotaSummary {
	summary := textQuotaSummary{
		ModelName:            relayInfo.OriginModelName,
		TokenName:            ctx.GetString("token_name"),
		UseTimeSeconds:       time.Now().Unix() - relayInfo.StartTime.Unix(),
		CompletionRatio:      relayInfo.PriceData.CompletionRatio,
		CacheRatio:           relayInfo.PriceData.CacheRatio,
		ImageRatio:           relayInfo.PriceData.ImageRatio,
		ModelRatio:           relayInfo.PriceData.ModelRatio,
		GroupRatio:           relayInfo.PriceData.GroupRatioInfo.GroupRatio,
		ModelPrice:           relayInfo.PriceData.ModelPrice,
		CacheCreationRatio:   relayInfo.PriceData.CacheCreationRatio,
		CacheCreationRatio5m: relayInfo.PriceData.CacheCreation5mRatio,
		CacheCreationRatio1h: relayInfo.PriceData.CacheCreation1hRatio,
		UsageSemantic:        usageSemanticFromUsage(relayInfo, usage),
	}
	summary.IsClaudeUsageSemantic = summary.UsageSemantic == "anthropic"

	if usage == nil {
		usage = &dto.Usage{
			PromptTokens:     relayInfo.GetEstimatePromptTokens(),
			CompletionTokens: 0,
			TotalTokens:      relayInfo.GetEstimatePromptTokens(),
		}
	}

	summary.PromptTokens = usage.PromptTokens
	summary.CompletionTokens = usage.CompletionTokens
	summary.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	summary.CacheTokens = usage.PromptTokensDetails.CachedTokens
	summary.CacheCreationTokens = usage.PromptTokensDetails.CacheCreationTokensTotal()
	summary.CacheCreationTokens5m = usage.ClaudeCacheCreation5mTokens
	summary.CacheCreationTokens1h = usage.ClaudeCacheCreation1hTokens
	summary.ImageTokens = usage.PromptTokensDetails.ImageTokens
	summary.AudioTokens = usage.PromptTokensDetails.AudioTokens
	legacyClaudeDerived := isLegacyClaudeDerivedOpenAIUsage(relayInfo, usage)
	isOpenRouterClaudeBilling := relayInfo.ChannelMeta != nil &&
		relayInfo.ChannelType == constant.ChannelTypeOpenRouter &&
		summary.IsClaudeUsageSemantic

	if isOpenRouterClaudeBilling {
		summary.PromptTokens -= summary.CacheTokens
		isUsingCustomSettings := relayInfo.PriceData.UsePrice || hasCustomModelRatio(summary.ModelName, relayInfo.PriceData.ModelRatio)
		if summary.CacheCreationTokens == 0 && relayInfo.PriceData.CacheCreationRatio != 1 && usage.Cost != 0 && !isUsingCustomSettings {
			maybeCacheCreationTokens := CalcOpenRouterCacheCreateTokens(*usage, relayInfo.PriceData)
			if maybeCacheCreationTokens >= 0 && summary.PromptTokens >= maybeCacheCreationTokens {
				summary.CacheCreationTokens = maybeCacheCreationTokens
			}
		}
		summary.PromptTokens -= summary.CacheCreationTokens
	}

	dPromptTokens := decimal.NewFromInt(int64(summary.PromptTokens))
	dCacheTokens := decimal.NewFromInt(int64(summary.CacheTokens))
	dImageTokens := decimal.NewFromInt(int64(summary.ImageTokens))
	dAudioTokens := decimal.NewFromInt(int64(summary.AudioTokens))
	dCompletionTokens := decimal.NewFromInt(int64(summary.CompletionTokens))
	dCachedCreationTokens := decimal.NewFromInt(int64(summary.CacheCreationTokens))
	dCompletionRatio := decimal.NewFromFloat(summary.CompletionRatio)
	dCacheRatio := decimal.NewFromFloat(summary.CacheRatio)
	dImageRatio := decimal.NewFromFloat(summary.ImageRatio)
	dModelRatio := decimal.NewFromFloat(summary.ModelRatio)
	dGroupRatio := decimal.NewFromFloat(summary.GroupRatio)
	dModelPrice := decimal.NewFromFloat(summary.ModelPrice)
	dCacheCreationRatio := decimal.NewFromFloat(summary.CacheCreationRatio)
	dCacheCreationRatio5m := decimal.NewFromFloat(summary.CacheCreationRatio5m)
	dCacheCreationRatio1h := decimal.NewFromFloat(summary.CacheCreationRatio1h)
	dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
	modelTokenAdjustment := relayInfo.PriceData.ModelTokenAdjustment
	dInputTokenMultiplier := decimal.NewFromFloat(modelTokenAdjustment.InputMultiplier())
	dOutputTokenMultiplier := decimal.NewFromFloat(modelTokenAdjustment.OutputMultiplier())
	dCacheReadTokenMultiplier := decimal.NewFromFloat(modelTokenAdjustment.CacheReadMultiplier())
	dCacheCreationTokenMultiplier := decimal.NewFromFloat(modelTokenAdjustment.CacheCreationMultiplier())

	ratio := dModelRatio.Mul(dGroupRatio)
	groupRatioInfo := relayInfo.PriceData.GroupRatioInfo
	hasGroupSpecialRatio := groupRatioInfo.HasSpecialRatio && groupRatioInfo.PricingGroupRatio >= 0
	summary.ToolCallSurchargeBase = calculateTextToolCallSurchargeBase(ctx, relayInfo, &summary)
	summary.ToolCallSurchargeQuota = summary.ToolCallSurchargeBase.Mul(dGroupRatio)

	var audioInputQuotaBase decimal.Decimal
	if !relayInfo.PriceData.UsePrice {
		baseTokens := dPromptTokens

		var cachedTokensBase decimal.Decimal
		var cachedTokensWithRatio decimal.Decimal
		if !dCacheTokens.IsZero() {
			if !summary.IsClaudeUsageSemantic && !legacyClaudeDerived {
				baseTokens = baseTokens.Sub(dCacheTokens)
			}
			cachedTokensBase = dCacheTokens.Mul(dCacheRatio)
			cachedTokensWithRatio = cachedTokensBase.Mul(dCacheReadTokenMultiplier)
		}

		var cachedCreationTokensBase decimal.Decimal
		var cachedCreationTokensWithRatio decimal.Decimal
		hasSplitCacheCreationTokens := summary.CacheCreationTokens5m > 0 || summary.CacheCreationTokens1h > 0
		if !dCachedCreationTokens.IsZero() || hasSplitCacheCreationTokens {
			if !summary.IsClaudeUsageSemantic && !legacyClaudeDerived {
				baseTokens = baseTokens.Sub(dCachedCreationTokens)
				cachedCreationTokensBase = dCachedCreationTokens.Mul(dCacheCreationRatio)
			} else {
				remaining := summary.CacheCreationTokens - summary.CacheCreationTokens5m - summary.CacheCreationTokens1h
				if remaining < 0 {
					remaining = 0
				}
				cachedCreationTokensBase = decimal.NewFromInt(int64(remaining)).Mul(dCacheCreationRatio)
				cachedCreationTokensBase = cachedCreationTokensBase.Add(decimal.NewFromInt(int64(summary.CacheCreationTokens5m)).Mul(dCacheCreationRatio5m))
				cachedCreationTokensBase = cachedCreationTokensBase.Add(decimal.NewFromInt(int64(summary.CacheCreationTokens1h)).Mul(dCacheCreationRatio1h))
			}
			cachedCreationTokensWithRatio = cachedCreationTokensBase.Mul(dCacheCreationTokenMultiplier)
		}

		var imageTokensWithRatio decimal.Decimal
		if !dImageTokens.IsZero() {
			baseTokens = baseTokens.Sub(dImageTokens)
			imageTokensWithRatio = dImageTokens.Mul(dImageRatio)
		}

		if !dAudioTokens.IsZero() {
			summary.AudioInputPrice = operation_setting.GetGeminiInputAudioPricePerMillionTokens(summary.ModelName)
			if summary.AudioInputPrice > 0 {
				baseTokens = baseTokens.Sub(dAudioTokens)
				audioInputQuotaBase = decimal.NewFromFloat(summary.AudioInputPrice).
					Div(decimal.NewFromInt(1000000)).Mul(dAudioTokens).Mul(dQuotaPerUnit)
			}
		}

		// OpenAI cache-write usage reports unadjusted prefix counts, so
		// cached_tokens + cache_write_tokens can exceed prompt_tokens and the
		// remainder can go negative. Clamp at zero so overlap never turns into
		// a negative base charge.
		if baseTokens.IsNegative() {
			baseTokens = decimal.Zero
		}

		adjustedBaseTokens := baseTokens.Mul(dInputTokenMultiplier)
		promptQuota := adjustedBaseTokens.Add(cachedTokensWithRatio).Add(imageTokensWithRatio).Add(cachedCreationTokensWithRatio)
		completionQuota := dCompletionTokens.Mul(dCompletionRatio).Mul(dOutputTokenMultiplier)
		quotaBeforeGroup := promptQuota.Add(completionQuota).Mul(dModelRatio)
		quotaCalculateDecimal := quotaBeforeGroup.Mul(dGroupRatio).
			Add(audioInputQuotaBase.Mul(dGroupRatio))
		quotaCalculateDecimal = relayInfo.PriceData.ApplyOtherRatiosToDecimal(quotaCalculateDecimal)
		quotaCalculateDecimal = quotaCalculateDecimal.Add(summary.ToolCallSurchargeQuota)
		quota, clamp := finalizeTextQuota(quotaCalculateDecimal, ratio, summary.hasBillableUsage())
		summary.Quota = quota
		noteQuotaClamp(relayInfo, clamp)

		quotaBeforeGroupWithoutModelAdjustment := quotaBeforeGroup
		if modelTokenAdjustment.HasAny() {
			promptQuotaWithoutAdjustment := baseTokens.Add(cachedTokensBase).Add(imageTokensWithRatio).Add(cachedCreationTokensBase)
			completionQuotaWithoutAdjustment := dCompletionTokens.Mul(dCompletionRatio)
			quotaBeforeGroupWithoutModelAdjustment = promptQuotaWithoutAdjustment.Add(completionQuotaWithoutAdjustment).
				Mul(dModelRatio)
			withoutModelAdjustment := quotaBeforeGroupWithoutModelAdjustment.
				Mul(dGroupRatio).
				Add(audioInputQuotaBase.Mul(dGroupRatio))
			withoutModelAdjustment = relayInfo.PriceData.ApplyOtherRatiosToDecimal(withoutModelAdjustment)
			withoutModelAdjustment = withoutModelAdjustment.Add(summary.ToolCallSurchargeQuota)
			counterfactualQuota, _ := finalizeTextQuota(withoutModelAdjustment, ratio, summary.hasBillableUsage())
			summary.BillingCounterfactuals.WithoutModelTokenAdjustment = &counterfactualQuota
		}

		if hasGroupSpecialRatio {
			pricingGroupRatio := decimal.NewFromFloat(groupRatioInfo.PricingGroupRatio)
			withoutSpecialRatio := quotaBeforeGroup.Mul(pricingGroupRatio).
				Add(audioInputQuotaBase.Mul(pricingGroupRatio))
			withoutSpecialRatio = relayInfo.PriceData.ApplyOtherRatiosToDecimal(withoutSpecialRatio)
			withoutSpecialRatio = withoutSpecialRatio.Add(summary.ToolCallSurchargeBase.Mul(pricingGroupRatio))
			counterfactualQuota, _ := finalizeTextQuota(withoutSpecialRatio, dModelRatio.Mul(pricingGroupRatio), summary.hasBillableUsage())
			summary.BillingCounterfactuals.WithoutGroupSpecialRatio = &counterfactualQuota
		}
		if hasGroupSpecialRatio || modelTokenAdjustment.HasAny() {
			originalGroupRatio := dGroupRatio
			if hasGroupSpecialRatio {
				originalGroupRatio = decimal.NewFromFloat(groupRatioInfo.PricingGroupRatio)
			}
			originalQuota := quotaBeforeGroupWithoutModelAdjustment.Mul(originalGroupRatio).
				Add(audioInputQuotaBase.Mul(originalGroupRatio))
			originalQuota = relayInfo.PriceData.ApplyOtherRatiosToDecimal(originalQuota)
			originalQuota = originalQuota.Add(summary.ToolCallSurchargeBase.Mul(originalGroupRatio))
			counterfactualQuota, _ := finalizeTextQuota(originalQuota, dModelRatio.Mul(originalGroupRatio), summary.hasBillableUsage())
			summary.BillingCounterfactuals.OriginalQuota = &counterfactualQuota
		}

		cacheCreationTokens := cacheWriteTokensTotal(summary)
		promptIncludesCache := !summary.IsClaudeUsageSemantic && !legacyClaudeDerived
		summary.ModelTokenAdjustmentAudit = buildModelTokenAdjustmentAudit(relayInfo, modelTokenAdjustment, modelTokenCategoryUsage{
			LogInput:                       summary.PromptTokens,
			LogOutput:                      summary.CompletionTokens,
			Input:                          baseTokens.InexactFloat64(),
			BilledInput:                    adjustedBaseTokens.InexactFloat64(),
			Output:                         float64(summary.CompletionTokens),
			BilledOutput:                   float64(summary.CompletionTokens) * modelTokenAdjustment.OutputMultiplier(),
			CacheRead:                      float64(summary.CacheTokens),
			BilledCacheRead:                float64(summary.CacheTokens) * modelTokenAdjustment.CacheReadMultiplier(),
			CacheCreation:                  float64(cacheCreationTokens),
			BilledCacheCreation:            float64(cacheCreationTokens) * modelTokenAdjustment.CacheCreationMultiplier(),
			CacheCreation5m:                float64(summary.CacheCreationTokens5m),
			BilledCacheCreation5m:          float64(summary.CacheCreationTokens5m) * modelTokenAdjustment.CacheCreationMultiplier(),
			CacheCreation1h:                float64(summary.CacheCreationTokens1h),
			BilledCacheCreation1h:          float64(summary.CacheCreationTokens1h) * modelTokenAdjustment.CacheCreationMultiplier(),
			IncludeCacheReadInLogInput:     promptIncludesCache,
			IncludeCacheCreationInLogInput: promptIncludesCache,
		})
	} else {
		quotaBeforeGroup := dModelPrice.Mul(dQuotaPerUnit)
		quotaCalculateDecimal := quotaBeforeGroup.Mul(dGroupRatio)
		quotaCalculateDecimal = relayInfo.PriceData.ApplyOtherRatiosToDecimal(quotaCalculateDecimal)
		quotaCalculateDecimal = quotaCalculateDecimal.Add(summary.ToolCallSurchargeQuota)
		quota, clamp := finalizeTextQuota(quotaCalculateDecimal, ratio, summary.hasBillableUsage())
		summary.Quota = quota
		noteQuotaClamp(relayInfo, clamp)

		if hasGroupSpecialRatio {
			pricingGroupRatio := decimal.NewFromFloat(groupRatioInfo.PricingGroupRatio)
			withoutSpecialRatio := relayInfo.PriceData.ApplyOtherRatiosToDecimal(quotaBeforeGroup.Mul(pricingGroupRatio)).
				Add(summary.ToolCallSurchargeBase.Mul(pricingGroupRatio))
			counterfactualQuota, _ := finalizeTextQuota(withoutSpecialRatio, decimal.Zero, summary.hasBillableUsage())
			summary.BillingCounterfactuals.WithoutGroupSpecialRatio = &counterfactualQuota
			summary.BillingCounterfactuals.OriginalQuota = &counterfactualQuota
		}
	}

	return summary
}

func usageSemanticFromUsage(relayInfo *relaycommon.RelayInfo, usage *dto.Usage) string {
	if usage != nil && usage.UsageSemantic != "" {
		return usage.UsageSemantic
	}
	if relayInfo != nil && relayInfo.GetFinalRequestRelayFormat() == types.RelayFormatClaude {
		return "anthropic"
	}
	return "openai"
}

func PostTextConsumeQuota(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage, extraContent []string) int {
	originUsage := usage
	billingUsage := effectiveBillingUsage(usage)
	if usage == nil {
		extraContent = append(extraContent, "上游无计费信息")
	}
	if originUsage != nil {
		ObserveChannelAffinityUsageCacheByRelayFormat(ctx, billingUsage, relayInfo.GetFinalRequestRelayFormat())
	}

	adminRejectReason := common.GetContextKeyString(ctx, constant.ContextKeyAdminRejectReason)
	summary := calculateTextQuotaSummary(ctx, relayInfo, billingUsage)

	var tieredResult *billingexpr.TieredResult
	tieredBillingApplied := false
	if originUsage != nil {
		var tieredUsedVars map[string]bool
		if snap := relayInfo.TieredBillingSnapshot; snap != nil {
			tieredUsedVars = billingexpr.UsedVars(snap.ExprString)
		}
		tieredParams := BuildTieredTokenParams(billingUsage, summary.IsClaudeUsageSemantic, tieredUsedVars)
		tieredOk, tieredQuota, tieredRes := TryTieredSettle(relayInfo, tieredParams)
		if tieredOk {
			tieredBillingApplied = true
			tieredResult = tieredRes
			summary.Quota = composeTieredTextQuota(relayInfo, summary, tieredQuota, tieredRes)
			summary.BillingCounterfactuals = calculateTieredBillingRevenueCounterfactuals(
				relayInfo,
				tieredParams,
				tieredRes,
				summary.ToolCallSurchargeBase,
			)
			if snap := relayInfo.TieredBillingSnapshot; snap != nil {
				adjustedParams := billingexpr.ApplyModelTokenAdjustment(tieredParams, snap.ModelTokenAdjustment)
				billedCacheRead := tieredParams.CR
				if tieredUsedVars["cr"] {
					billedCacheRead = adjustedParams.CR
				}
				billedCacheCreation := tieredParams.CC
				if tieredUsedVars["cc"] {
					billedCacheCreation = adjustedParams.CC
				}
				billedCacheCreation1h := tieredParams.CC1h
				if tieredUsedVars["cc1h"] {
					billedCacheCreation1h = adjustedParams.CC1h
				}
				promptIncludesCache := !summary.IsClaudeUsageSemantic
				summary.ModelTokenAdjustmentAudit = buildModelTokenAdjustmentAudit(relayInfo, snap.ModelTokenAdjustment, modelTokenCategoryUsage{
					LogInput:                       summary.PromptTokens,
					LogOutput:                      summary.CompletionTokens,
					Input:                          tieredParams.P,
					BilledInput:                    adjustedParams.P,
					Output:                         tieredParams.C,
					BilledOutput:                   adjustedParams.C,
					CacheRead:                      tieredParams.CR,
					BilledCacheRead:                billedCacheRead,
					CacheCreation:                  tieredParams.CC + tieredParams.CC1h,
					BilledCacheCreation:            billedCacheCreation + billedCacheCreation1h,
					CacheCreation5m:                float64(summary.CacheCreationTokens5m),
					BilledCacheCreation5m:          float64(summary.CacheCreationTokens5m) * modelTokenMultiplierForVariable(snap.ModelTokenAdjustment.CacheCreationMultiplier(), tieredUsedVars["cc"]),
					CacheCreation1h:                float64(summary.CacheCreationTokens1h),
					BilledCacheCreation1h:          float64(summary.CacheCreationTokens1h) * modelTokenMultiplierForVariable(snap.ModelTokenAdjustment.CacheCreationMultiplier(), tieredUsedVars["cc1h"]),
					IncludeCacheReadInLogInput:     promptIncludesCache && tieredUsedVars["cr"],
					IncludeCacheCreationInLogInput: promptIncludesCache && (tieredUsedVars["cc"] || tieredUsedVars["cc1h"]),
				})
			}
		}
	}

	for _, item := range summary.ToolSurchargeItems {
		q := decimal.NewFromFloat(item.Price).
			Mul(decimal.NewFromInt(int64(item.Count))).
			Div(decimal.NewFromInt(1000)).
			Mul(decimal.NewFromFloat(summary.GroupRatio)).
			Mul(decimal.NewFromFloat(common.QuotaPerUnit))
		extraContent = append(extraContent, fmt.Sprintf(
			"%s 调用 %d 次，调用花费 %s",
			item.Name,
			item.Count,
			logger.LogQuota(common.QuotaFromDecimal(q)),
		))
	}
	if summary.AudioInputPrice > 0 && summary.AudioTokens > 0 {
		q := decimal.NewFromFloat(summary.AudioInputPrice).Div(decimal.NewFromInt(1000000)).Mul(decimal.NewFromInt(int64(summary.AudioTokens))).Mul(decimal.NewFromFloat(summary.GroupRatio)).Mul(decimal.NewFromFloat(common.QuotaPerUnit))
		extraContent = append(extraContent, fmt.Sprintf("Audio Input 花费 %s", logger.LogQuota(common.QuotaFromDecimal(q))))
	}

	if !summary.hasBillableUsage() {
		extraContent = append(extraContent, "上游没有返回计费信息，无法扣费（可能是上游超时）")
		logger.LogError(ctx, fmt.Sprintf("total tokens is 0, cannot consume quota, userId %d, channelId %d, tokenId %d, model %s， pre-consumed quota %d", relayInfo.UserId, relayInfo.ChannelId, relayInfo.TokenId, summary.ModelName, relayInfo.FinalPreConsumedQuota))
	} else {
		model.UpdateUserUsedQuotaAndRequestCount(relayInfo.UserId, summary.Quota)
		model.UpdateChannelUsedQuota(relayInfo.ChannelId, summary.Quota)
	}

	if err := SettleBilling(ctx, relayInfo, summary.Quota); err != nil {
		logger.LogError(ctx, "error settling billing: "+err.Error())
	}

	logModel := summary.ModelName
	if strings.HasPrefix(logModel, "gpt-4-gizmo") {
		logModel = "gpt-4-gizmo-*"
		extraContent = append(extraContent, fmt.Sprintf("模型 %s", summary.ModelName))
	}
	if strings.HasPrefix(logModel, "gpt-4o-gizmo") {
		logModel = "gpt-4o-gizmo-*"
		extraContent = append(extraContent, fmt.Sprintf("模型 %s", summary.ModelName))
	}

	logContent := strings.Join(extraContent, ", ")
	var other map[string]interface{}
	if summary.IsClaudeUsageSemantic {
		other = GenerateClaudeOtherInfo(ctx, relayInfo,
			summary.ModelRatio, summary.GroupRatio, summary.CompletionRatio,
			summary.CacheTokens, summary.CacheRatio,
			summary.CacheCreationTokens, summary.CacheCreationRatio,
			summary.CacheCreationTokens5m, summary.CacheCreationRatio5m,
			summary.CacheCreationTokens1h, summary.CacheCreationRatio1h,
			summary.ModelPrice, relayInfo.PriceData.GroupRatioInfo.GroupSpecialRatio)
		other["usage_semantic"] = "anthropic"
	} else {
		other = GenerateTextOtherInfo(ctx, relayInfo, summary.ModelRatio, summary.GroupRatio, summary.CompletionRatio, summary.CacheTokens, summary.CacheRatio, summary.ModelPrice, relayInfo.PriceData.GroupRatioInfo.GroupSpecialRatio)
	}
	appendUsageBillingPathForLog(other, common.GetContextKeyBool(ctx, constant.ContextKeyLocalCountTokens), originUsage)
	if adminRejectReason != "" {
		other["reject_reason"] = adminRejectReason
	}
	if summary.ImageTokens != 0 {
		other["image"] = true
		other["image_ratio"] = summary.ImageRatio
		other["image_output"] = summary.ImageTokens
	}
	appendToolSurchargeLogInfo(other, summary.ToolSurchargeItems)
	if summary.AudioInputPrice > 0 && summary.AudioTokens > 0 {
		other["audio_input_seperate_price"] = true
		other["audio_input_token_count"] = summary.AudioTokens
		other["audio_input_price"] = summary.AudioInputPrice
	}
	if summary.CacheCreationTokens > 0 {
		other["cache_creation_tokens"] = summary.CacheCreationTokens
		other["cache_creation_ratio"] = summary.CacheCreationRatio
	}
	if summary.CacheCreationTokens5m > 0 {
		other["cache_creation_tokens_5m"] = summary.CacheCreationTokens5m
		other["cache_creation_ratio_5m"] = summary.CacheCreationRatio5m
	}
	if summary.CacheCreationTokens1h > 0 {
		other["cache_creation_tokens_1h"] = summary.CacheCreationTokens1h
		other["cache_creation_ratio_1h"] = summary.CacheCreationRatio1h
	}
	cacheWriteTokens := cacheWriteTokensTotal(summary)
	if cacheWriteTokens > 0 {
		// cache_write_tokens: normalized cache creation total for UI display.
		// If split 5m/1h values are present, this is their sum; otherwise it falls back
		// to cache_creation_tokens.
		other["cache_write_tokens"] = cacheWriteTokens
	}
	if relayInfo.GetFinalRequestRelayFormat() != types.RelayFormatClaude && billingUsage != nil && billingUsage.UsageSource != "" && billingUsage.InputTokens > 0 {
		// input_tokens_total: explicit normalized total input used by the usage log UI.
		// Only write this field when upstream/current conversion has already provided a
		// reliable total input value and tagged the usage source. Do not infer it from
		// prompt/cache fields here, otherwise old upstream payloads may be double-counted.
		other["input_tokens_total"] = billingUsage.InputTokens
	}
	if tieredBillingApplied {
		InjectTieredBillingInfo(other, relayInfo, tieredResult)
	}
	attachModelTokenAdjustmentToOther(other, summary.ModelTokenAdjustmentAudit)
	attachBillingRevenueToOther(other, buildBillingRevenueAudit(summary.Quota, summary.BillingCounterfactuals))

	attachQuotaSaturation(ctx, relayInfo, other)

	model.RecordConsumeLog(ctx, relayInfo.UserId, model.RecordConsumeLogParams{
		ChannelId:        relayInfo.ChannelId,
		PromptTokens:     summary.PromptTokens,
		CompletionTokens: summary.CompletionTokens,
		ModelName:        logModel,
		TokenName:        summary.TokenName,
		Quota:            summary.Quota,
		Content:          logContent,
		TokenId:          relayInfo.TokenId,
		UseTimeSeconds:   int(summary.UseTimeSeconds),
		IsStream:         relayInfo.IsStream,
		Group:            relayInfo.UsingGroup,
		Other:            other,
	})
	gopool.Go(func() {
		perfmetrics.RecordRelaySample(relayInfo, true, int64(summary.CompletionTokens))
	})
	return summary.Quota
}
