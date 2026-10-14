package controller

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/console_setting"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

var completionRatioMetaOptionKeys = []string{
	"ModelPrice",
	"ModelRatio",
	"CompletionRatio",
	"CacheRatio",
	"CreateCacheRatio",
	"ImageRatio",
	"AudioRatio",
	"AudioCompletionRatio",
}

func isPaymentComplianceOptionKey(key string) bool {
	return strings.HasPrefix(key, "payment_setting.compliance_")
}

func isPositiveOptionValue(value string) bool {
	intValue, err := strconv.Atoi(strings.TrimSpace(value))
	if err == nil {
		return intValue > 0
	}
	floatValue, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return err == nil && floatValue > 0
}

func collectModelNamesFromOptionValue(raw string, modelNames map[string]struct{}) {
	if strings.TrimSpace(raw) == "" {
		return
	}

	var parsed map[string]any
	if err := common.UnmarshalJsonStr(raw, &parsed); err != nil {
		return
	}

	for modelName := range parsed {
		modelNames[modelName] = struct{}{}
	}
}

func buildCompletionRatioMetaValue(optionValues map[string]string) string {
	modelNames := make(map[string]struct{})
	for _, key := range completionRatioMetaOptionKeys {
		collectModelNamesFromOptionValue(optionValues[key], modelNames)
	}

	meta := make(map[string]ratio_setting.CompletionRatioInfo, len(modelNames))
	for modelName := range modelNames {
		meta[modelName] = ratio_setting.GetCompletionRatioInfo(modelName)
	}

	jsonBytes, err := common.Marshal(meta)
	if err != nil {
		return "{}"
	}
	return string(jsonBytes)
}

func GetOptions(c *gin.Context) {
	action := systemSettingsAction(c)
	if !canAccessSystemSettingsAction(c, action) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权访问该系统设置"})
		return
	}
	var options []*model.Option
	optionValues := make(map[string]string)
	common.OptionMapRWMutex.Lock()
	for k, v := range common.OptionMap {
		if k == "theme.frontend" || k == "billing_setting.billing_mode" || k == "billing_setting.billing_expr" {
			continue
		}
		if c.GetInt("role") < common.RoleRootUser && !optionAllowedForSystemSettingsAction(k, action) {
			continue
		}
		value := common.Interface2String(v)
		isSensitiveKey := strings.HasSuffix(k, "Token") ||
			strings.HasSuffix(k, "Secret") ||
			strings.HasSuffix(k, "Key") ||
			strings.HasSuffix(k, "secret") ||
			strings.HasSuffix(k, "api_key")
		if isSensitiveKey {
			continue
		}
		options = append(options, &model.Option{
			Key:   k,
			Value: value,
		})
		if slices.Contains(completionRatioMetaOptionKeys, k) {
			optionValues[k] = value
		}
	}
	common.OptionMapRWMutex.Unlock()
	// Display the same effective expressions used by pricing and settlement,
	// including built-in defaults absent from persisted administrator options.
	for key, values := range map[string]map[string]string{
		"billing_setting.billing_mode": billing_setting.GetBillingModeCopy(),
		"billing_setting.billing_expr": billing_setting.GetBillingExprCopy(),
	} {
		if c.GetInt("role") < common.RoleRootUser && !optionAllowedForSystemSettingsAction(key, action) {
			continue
		}
		encoded, err := common.Marshal(values)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
			return
		}
		options = append(options, &model.Option{Key: key, Value: string(encoded)})
	}
	options = append(options, &model.Option{
		Key:   "CompletionRatioMeta",
		Value: buildCompletionRatioMetaValue(optionValues),
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    options,
	})
}

type OptionUpdateRequest struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

func recordPasskeyDomainAudit(c *gin.Context, change *model.PasskeyDomainChange, confirmed bool, err error) {
	params := map[string]any{"confirmed": confirmed}
	if change != nil {
		params["removed_rp_ids"] = change.RemovedRPIDs
		params["affected_credentials"] = change.AffectedCredentials
		params["unknown_credentials"] = change.UnknownCredentials
	}
	if err != nil {
		params["success"] = false
		params["error"] = err.Error()
	} else {
		params["success"] = true
	}
	recordManageAudit(c, "passkey.domain_update", params)
}

func UpdatePasskeyDomains(c *gin.Context) {
	var request struct {
		RPID                *string `json:"rp_id"`
		LegacyRPIDs         *string `json:"legacy_rp_ids"`
		Origins             *string `json:"origins"`
		Preview             bool    `json:"preview"`
		RemovalConfirmation string  `json:"removal_confirmation"`
	}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || request.RPID == nil || request.LegacyRPIDs == nil || request.Origins == nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	change, err := model.UpdatePasskeyDomainOptions(map[string]string{
		"passkey.rp_id": *request.RPID, "passkey.legacy_rp_ids": *request.LegacyRPIDs, "passkey.origins": *request.Origins,
	}, request.Preview, request.RemovalConfirmation)
	if err != nil {
		writePasskeyDomainSettingsError(c, err)
		if !request.Preview {
			recordPasskeyDomainAudit(c, change, request.RemovalConfirmation != "", err)
		}
		return
	}
	if !request.Preview {
		recordPasskeyDomainAudit(c, change, request.RemovalConfirmation != "", nil)
	}
	common.ApiSuccess(c, change)
}

func writePasskeyDomainSettingsError(c *gin.Context, err error) {
	var removal *model.PasskeyDomainRemovalError
	if errors.As(err, &removal) {
		common.ApiErrorStatus(c, http.StatusConflict, common.NewMessage("Review the affected Passkeys and confirm the domain removal. If the settings or impact have changed, confirmation is required again."), gin.H{
			"code": "PASSKEY_RP_ID_REMOVAL_CONFIRMATION_REQUIRED",
			"data": removal.Change,
		})
		return
	}
	if errors.Is(err, system_setting.ErrPasskeyRPIDInvalid) {
		writeSecurityOperationError(c, err)
		return
	}
	common.ApiError(c, err)
}

func UpdateOption(c *gin.Context) {
	var option OptionUpdateRequest
	err := common.DecodeJson(c.Request.Body, &option)
	if err != nil {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Invalid parameters"))
		return
	}
	if c.GetInt("role") < common.RoleRootUser {
		action := systemSettingsAction(c)
		if !canAccessSystemSettingsAction(c, action) || !optionAllowedForSystemSettingsAction(option.Key, action) {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权修改该系统设置"})
			return
		}
	}
	switch option.Value.(type) {
	case bool:
		option.Value = common.Interface2String(option.Value.(bool))
	case float64:
		option.Value = common.Interface2String(option.Value.(float64))
	case int:
		option.Value = common.Interface2String(option.Value.(int))
	default:
		option.Value = fmt.Sprintf("%v", option.Value)
	}
	previousLogo := ""
	if option.Key == "Logo" {
		common.OptionMapRWMutex.RLock()
		previousLogo = common.Logo
		common.OptionMapRWMutex.RUnlock()
	}
	switch option.Key {
	case "QuotaForInviter", "QuotaForInvitee":
		if isPositiveOptionValue(option.Value.(string)) && !operation_setting.IsPaymentComplianceConfirmed() {
			common.ApiErrorT(c, "Payment, redemption, subscription, and invitation reward features are disabled. The administrator must confirm compliance terms before enabling them.")
			return
		}
	case "MediaAssetRetentionDays":
		retentionDays, parseErr := strconv.Atoi(option.Value.(string))
		if parseErr != nil || retentionDays < 1 || retentionDays > 365 {
			common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Media asset retention must be between 1 and 365 days"))
			return
		}
	default:
		if isPaymentComplianceOptionKey(option.Key) {
			common.ApiErrorT(c, "Compliance confirmation fields cannot be changed through the general settings API")
			return
		}
	}
	if option.Key == "TaskPublicAddress" && option.Value.(string) != "" {
		if err := service.ValidateTaskArtifactBaseURL(option.Value.(string)); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	switch option.Key {
	case "GitHubOAuthEnabled":
		if option.Value == "true" && common.GitHubClientId == "" {
			common.ApiErrorT(c, "Cannot enable {{provider}}. Enter the {{provider}} Client ID and Client Secret first.", map[string]any{"provider": "GitHub OAuth"})
			return
		}
	case "discord.enabled":
		if option.Value == "true" && system_setting.GetDiscordSettings().ClientId == "" {
			common.ApiErrorT(c, "Cannot enable {{provider}}. Enter the {{provider}} Client ID and Client Secret first.", map[string]any{"provider": "Discord OAuth"})
			return
		}
	case "oidc.enabled":
		if option.Value == "true" && system_setting.GetOIDCSettings().ClientId == "" {
			common.ApiErrorT(c, "Cannot enable {{provider}}. Enter the {{provider}} Client ID and Client Secret first.", map[string]any{"provider": "OIDC"})
			return
		}
	case "LinuxDOOAuthEnabled":
		if option.Value == "true" && common.LinuxDOClientId == "" {
			common.ApiErrorT(c, "Cannot enable {{provider}}. Enter the {{provider}} Client ID and Client Secret first.", map[string]any{"provider": "LinuxDO OAuth"})
			return
		}
	case "EmailDomainRestrictionEnabled":
		if option.Value == "true" && len(common.EmailDomainWhitelist) == 0 {
			common.ApiErrorT(c, "Cannot enable the email domain restriction. Enter the allowed email domains first.")
			return
		}
	case "WeChatAuthEnabled":
		if option.Value == "true" && common.WeChatServerAddress == "" {
			common.ApiErrorT(c, "Cannot enable WeChat login. Enter the WeChat login settings first.")
			return
		}
	case "TurnstileCheckEnabled":
		if option.Value == "true" && common.TurnstileSiteKey == "" {
			common.ApiErrorT(c, "Cannot enable Turnstile verification. Enter the Turnstile settings first.")
			return
		}
	case "TelegramOAuthEnabled":
		if option.Value == "true" && !system_setting.GetTelegramSettings().IsConfigured() {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"code":    "TELEGRAM_OAUTH_NOT_CONFIGURED",
				"message": "Telegram OAuth is not configured or enabled. Please contact your administrator.",
			})
			return
		}
	case "theme.frontend":
		if option.Value != "default" {
			common.ApiErrorT(c, "The Classic frontend has been removed. The theme can only be set to default")
			return
		}
	case "GroupRatio":
		err = ratio_setting.CheckGroupRatio(option.Value.(string))
		if err != nil {
			common.ApiError(c, err)
			return
		}
	case "gemini.safety_settings":
		err = model_setting.ValidateGeminiSafetySettings(option.Value.(string))
		if err != nil {
			common.ApiError(c, err)
			return
		}
	case "claude.default_max_tokens":
		err = model_setting.ValidateClaudeDefaultMaxTokens(option.Value.(string))
		if err != nil {
			common.ApiError(c, err)
			return
		}
	case operation_setting.ToolPriceOptionKey:
		err = operation_setting.ValidateToolPricesJSON(option.Value.(string))
		if err != nil {
			common.ApiError(c, err)
			return
		}
	case "group_ratio_setting.model_token_ratio":
		err = ratio_setting.CheckModelTokenRatio(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "模型特殊倍率规则设置失败: " + err.Error(),
			})
			return
		}
	case "ImageRatio":
		err = ratio_setting.UpdateImageRatioByJSONString(option.Value.(string))
		if err != nil {
			common.ApiErrorT(c, "Failed to set the image ratio: {{error}}", map[string]any{"error": err.Error()})
			return
		}
	case "AudioRatio":
		err = ratio_setting.UpdateAudioRatioByJSONString(option.Value.(string))
		if err != nil {
			common.ApiErrorT(c, "Failed to set the audio ratio: {{error}}", map[string]any{"error": err.Error()})
			return
		}
	case "AudioCompletionRatio":
		err = ratio_setting.UpdateAudioCompletionRatioByJSONString(option.Value.(string))
		if err != nil {
			common.ApiErrorT(c, "Failed to set the audio completion ratio: {{error}}", map[string]any{"error": err.Error()})
			return
		}
	case "CreateCacheRatio":
		err = ratio_setting.UpdateCreateCacheRatioByJSONString(option.Value.(string))
		if err != nil {
			common.ApiErrorT(c, "Failed to set the cache creation ratio: {{error}}", map[string]any{"error": err.Error()})
			return
		}
	case "ModelRequestRateLimitGroup":
		err = setting.CheckModelRequestRateLimitGroup(option.Value.(string))
		if err != nil {
			common.ApiError(c, err)
			return
		}
	case "AutomaticDisableStatusCodes":
		_, err = operation_setting.ParseHTTPStatusCodeRanges(option.Value.(string))
		if err != nil {
			common.ApiError(c, err)
			return
		}
	case "AutomaticRetryStatusCodes":
		_, err = operation_setting.ParseHTTPStatusCodeRanges(option.Value.(string))
		if err != nil {
			common.ApiError(c, err)
			return
		}
	case "ImageGenerationLogPollingIntervalSeconds":
		interval, parseErr := strconv.Atoi(option.Value.(string))
		if parseErr != nil || interval < 5 || interval > 3600 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "生图任务刷新频率必须为 5 到 3600 秒之间的整数",
			})
			return
		}
	case common.SubscriptionExpiryReminderEnabledOptionKey:
		if option.Value != "true" && option.Value != "false" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "订阅到期提醒开关值无效",
			})
			return
		}
	case "UserLogGroupRatioDisplayMode":
		mode := option.Value.(string)
		if mode != common.UserLogGroupRatioDisplayModeSystem &&
			mode != common.UserLogGroupRatioDisplayModePricingGroup &&
			mode != common.UserLogGroupRatioDisplayModeManual {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "用户日志倍率展示模式无效",
			})
			return
		}
	case "UserLogGroupRatioManualValue":
		ratio, parseErr := strconv.ParseFloat(option.Value.(string), 64)
		if parseErr != nil || ratio < 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "手动展示倍率必须是大于等于 0 的有效数字",
			})
			return
		}
	case "ModelSquareGroupRatioDisplayMode":
		mode := option.Value.(string)
		if mode != common.ModelSquareGroupRatioDisplayModeActual &&
			mode != common.ModelSquareGroupRatioDisplayModePricingGroup {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "模型广场倍率展示模式无效",
			})
			return
		}
	case "TokenGroupRatioDisplayMode":
		mode := option.Value.(string)
		if mode != common.TokenGroupRatioDisplayModeActual &&
			mode != common.TokenGroupRatioDisplayModePricingGroup {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "令牌分组倍率展示模式无效",
			})
			return
		}
	case "ImageGenerationLogImageAuthWhitelist":
		normalized, normalizeErr := common.NormalizeImageGenerationLogImageAuthWhitelist(option.Value.(string))
		if normalizeErr != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": normalizeErr.Error()})
			return
		}
		option.Value = normalized
	case "ImageGenerationLogImageAuthWhitelistEnabled":
		if option.Value == "true" && strings.TrimSpace(common.ImageGenerationLogImageAuthWhitelist) == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "请先保存至少一个图片读取白名单域名或 IP",
			})
			return
		}
	case "invoice_setting.enabled":
		if option.Value != "true" && option.Value != "false" {
			common.ApiErrorMsg(c, "自助发票开关值无效")
			return
		}
	case "invoice_setting.allowed_sources":
		if err := operation_setting.ValidateInvoiceAllowedSources(option.Value.(string)); err != nil {
			common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage(err.Error()))
			return
		}
	case "invoice_setting.minimum_amount":
		amount, parseErr := strconv.ParseFloat(option.Value.(string), 64)
		if parseErr != nil || amount <= 0 || amount > 1000000000 || math.IsNaN(amount) || math.IsInf(amount, 0) {
			common.ApiErrorMsg(c, "发票最低申请金额必须是大于 0 且不超过 1000000000 的有效数字")
			return
		}
		quota, quotaErr := common.QuotaFromFloatStrict(amount * common.QuotaPerUnit)
		if quotaErr != nil || quota <= 0 {
			common.ApiErrorMsg(c, "发票最低申请金额超过系统额度范围")
			return
		}
	case "invoice_setting.unit":
		unit := strings.TrimSpace(option.Value.(string))
		if len([]rune(unit)) == 0 || len([]rune(unit)) > 16 {
			common.ApiErrorMsg(c, "发票金额单位长度必须为 1 到 16 个字符")
			return
		}
		option.Value = unit
	case "invoice_setting.processing_days":
		days := strings.TrimSpace(option.Value.(string))
		if len([]rune(days)) == 0 || len([]rune(days)) > 32 {
			common.ApiErrorMsg(c, "发票处理工作日说明长度必须为 1 到 32 个字符")
			return
		}
		option.Value = days
	case "invoice_setting.rejection_freeze_hours":
		hours, parseErr := strconv.Atoi(strings.TrimSpace(option.Value.(string)))
		if parseErr != nil || hours < 0 || hours > 8760 {
			common.ApiErrorMsg(c, "发票驳回冷冻时间必须是 0 到 8760 之间的整数小时")
			return
		}
		option.Value = strconv.Itoa(hours)
	case "console_setting.api_info":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "ApiInfo")
		if err != nil {
			common.ApiError(c, err)
			return
		}
	case "console_setting.custom_endpoints":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "CustomEndpoints")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
	case "console_setting.announcements":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "Announcements")
		if err != nil {
			common.ApiError(c, err)
			return
		}
	case "console_setting.faq":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "FAQ")
		if err != nil {
			common.ApiError(c, err)
			return
		}
	case "console_setting.uptime_kuma_groups":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "UptimeKumaGroups")
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}
	if model.IsPasskeyDomainOption(option.Key) {
		change, updateErr := model.UpdatePasskeyDomainOptions(map[string]string{option.Key: option.Value.(string)}, false, "")
		if updateErr != nil {
			writePasskeyDomainSettingsError(c, updateErr)
			recordPasskeyDomainAudit(c, change, false, updateErr)
			return
		}
		recordPasskeyDomainAudit(c, change, false, nil)
		common.ApiSuccess(c, change)
		return
	}
	err = model.UpdateOption(option.Key, option.Value.(string))
	if err != nil {
		if errors.Is(err, system_setting.ErrPasskeyRPIDInvalid) {
			writeSecurityOperationError(c, err)
		} else {
			common.ApiError(c, err)
		}
		return
	}
	if option.Key == "Logo" {
		removeObsoleteSiteLogo(previousLogo, option.Value.(string))
	}
	// 出于安全考虑只记录被修改的配置项名称，不记录配置值（可能含密钥等敏感信息）。
	recordManageAudit(c, "option.update", map[string]any{
		"key": option.Key,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}
