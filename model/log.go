package model

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func applyExplicitLogTextFilter(tx *gorm.DB, column string, value string) (*gorm.DB, error) {
	if value == "" {
		return tx, nil
	}
	if strings.Contains(value, "%") {
		condition, pattern, err := buildLogLikeCondition(column, value)
		if err != nil {
			return nil, err
		}
		return tx.Where(condition, pattern), nil
	}
	return tx.Where(column+" = ?", value), nil
}

func buildLogLikeCondition(column string, value string) (string, string, error) {
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		pattern, err := sanitizeClickHouseLikePattern(value)
		if err != nil {
			return "", "", err
		}
		return column + " LIKE ?", pattern, nil
	}

	pattern, err := sanitizeLikePattern(value)
	if err != nil {
		return "", "", err
	}
	return column + " LIKE ? ESCAPE '!'", pattern, nil
}

func sanitizeClickHouseLikePattern(input string) (string, error) {
	input = strings.ReplaceAll(input, `\`, `\\`)
	input = strings.ReplaceAll(input, `_`, `\_`)

	if err := validateLikePattern(input); err != nil {
		return "", err
	}
	return input, nil
}

type Log struct {
	Id                    int      `json:"id" gorm:"index:idx_created_at_id,priority:2;index:idx_user_id_id,priority:2"`
	UserId                int      `json:"user_id" gorm:"index;index:idx_user_id_id,priority:1"`
	CreatedAt             int64    `json:"created_at" gorm:"bigint;index:idx_created_at_id,priority:1;index:idx_created_at_type"`
	Type                  int      `json:"type" gorm:"index:idx_created_at_type"`
	Content               string   `json:"content"`
	Username              string   `json:"username" gorm:"index;index:index_username_model_name,priority:2;default:''"`
	TokenName             string   `json:"token_name" gorm:"index;default:''"`
	ModelName             string   `json:"model_name" gorm:"index;index:index_username_model_name,priority:1;default:''"`
	Quota                 int      `json:"quota" gorm:"default:0"`
	PromptTokens          int      `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens      int      `json:"completion_tokens" gorm:"default:0"`
	UseTime               int      `json:"use_time" gorm:"default:0"`
	IsStream              bool     `json:"is_stream"`
	ChannelId             int      `json:"channel" gorm:"index"`
	ChannelName           string   `json:"channel_name" gorm:"->"`
	TokenId               int      `json:"token_id" gorm:"default:0;index"`
	Group                 string   `json:"group" gorm:"index"`
	UserDisplayGroupRatio *float64 `json:"user_display_group_ratio,omitempty"`
	Ip                    string   `json:"ip" gorm:"index;default:''"`
	RequestId             string   `json:"request_id,omitempty" gorm:"type:varchar(64);index:idx_logs_request_id;default:''"`
	UpstreamRequestId     string   `json:"upstream_request_id,omitempty" gorm:"type:varchar(128);index:idx_logs_upstream_request_id;default:''"`
	Other                 string   `json:"other"`
}

type UserLoginIPStat struct {
	IP              string `json:"ip"`
	LastLoginAt     int64  `json:"last_login_at"`
	LoginCount      int64  `json:"login_count"`
	Blocked         bool   `json:"blocked" gorm:"-:all"`
	SharedUserCount int64  `json:"shared_user_count" gorm:"-:all"`
}

// don't use iota, avoid change log type value
const (
	LogTypeUnknown       = 0
	LogTypeTopup         = 1
	LogTypeConsume       = 2
	LogTypeManage        = 3
	LogTypeSystem        = 4
	LogTypeError         = 5
	LogTypeRefund        = 6
	LogTypeLogin         = 7
	LogTypeQuotaIncrease = 8
)

const (
	QuotaIncreaseSourceRedemption            = "redemption"
	QuotaIncreaseSourceCheckin               = "checkin"
	QuotaIncreaseSourceAdminAdjustment       = "admin_adjustment"
	QuotaIncreaseSourceOnlineRecharge        = "online_recharge"
	QuotaIncreaseSourceRefund                = "refund"
	QuotaIncreaseSourceRegistrationBonus     = "registration_bonus"
	QuotaIncreaseSourceInvitationBonus       = "invitation_bonus"
	QuotaIncreaseSourceAffiliateTransfer     = "affiliate_transfer"
	QuotaIncreaseSourceAgentRedemptionRefund = "agent_redemption_refund"
)

func ensureLogRequestId(log *Log) {
	if log != nil && log.RequestId == "" {
		log.RequestId = common.NewRequestId()
	}
}

func createLog(log *Log) error {
	snapshotUserDisplayGroupRatio(log)
	ensureLogRequestId(log)
	return LOG_DB.Create(log).Error
}

func clickHouseLogOrder(prefix string) string {
	return prefix + "created_at desc, " + prefix + "request_id desc"
}

func assignDisplayLogIds(logs []*Log, startIdx int) {
	for i := range logs {
		logs[i].Id = startIdx + i + 1
	}
}

type logGroupRatioDisplay struct {
	applicable bool
	visible    bool
	value      float64
}

func isFiniteLogRatio(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func actualLogGroupRatioDisplay(otherMap map[string]interface{}) logGroupRatioDisplay {
	groupRatio, hasGroupRatio := otherMap["group_ratio"].(float64)
	userGroupRatio, hasUserGroupRatio := otherMap["user_group_ratio"].(float64)
	if (!hasGroupRatio || !isFiniteLogRatio(groupRatio)) &&
		(!hasUserGroupRatio || !isFiniteLogRatio(userGroupRatio)) {
		return logGroupRatioDisplay{}
	}

	display := logGroupRatioDisplay{applicable: true}
	if hasUserGroupRatio && isFiniteLogRatio(userGroupRatio) && userGroupRatio != -1 {
		display.value = userGroupRatio
		display.visible = true
	} else if hasGroupRatio && isFiniteLogRatio(groupRatio) {
		display.value = groupRatio
		display.visible = groupRatio != 1
	}
	return display
}

func configuredUserDisplayGroupRatio(log *Log, otherMap map[string]interface{}) logGroupRatioDisplay {
	actualDisplay := actualLogGroupRatioDisplay(otherMap)
	if !actualDisplay.applicable {
		return actualDisplay
	}

	switch common.UserLogGroupRatioDisplayMode {
	case common.UserLogGroupRatioDisplayModePricingGroup:
		group := strings.TrimSpace(log.Group)
		if group == "" {
			group, _ = otherMap["group"].(string)
		}
		if ratio_setting.ContainsGroupRatio(group) {
			ratio := ratio_setting.GetGroupRatio(group)
			if ratio >= 0 && isFiniteLogRatio(ratio) {
				return logGroupRatioDisplay{applicable: true, visible: true, value: ratio}
			}
		}
	case common.UserLogGroupRatioDisplayModeManual:
		if common.UserLogGroupRatioManualValue >= 0 && isFiniteLogRatio(common.UserLogGroupRatioManualValue) {
			return logGroupRatioDisplay{
				applicable: true,
				visible:    true,
				value:      common.UserLogGroupRatioManualValue,
			}
		}
	}
	return actualDisplay
}

func snapshotUserDisplayGroupRatio(log *Log) {
	if log == nil || log.UserDisplayGroupRatio != nil {
		return
	}
	otherMap, _ := common.StrToMap(log.Other)
	if otherMap == nil {
		return
	}
	display := configuredUserDisplayGroupRatio(log, otherMap)
	if !display.visible {
		return
	}
	value := display.value
	log.UserDisplayGroupRatio = &value
}

func storedUserDisplayGroupRatio(log *Log, otherMap map[string]interface{}) logGroupRatioDisplay {
	if log.UserDisplayGroupRatio != nil &&
		*log.UserDisplayGroupRatio >= 0 &&
		isFiniteLogRatio(*log.UserDisplayGroupRatio) {
		return logGroupRatioDisplay{
			applicable: true,
			visible:    true,
			value:      *log.UserDisplayGroupRatio,
		}
	}
	return actualLogGroupRatioDisplay(otherMap)
}

func applyUserLogDisplayMetrics(log *Log, otherMap map[string]interface{}, actual logGroupRatioDisplay, display logGroupRatioDisplay) {
	if log.Type != LogTypeConsume || !actual.applicable || !display.visible || actual.value <= 0 {
		return
	}
	scale := display.value / actual.value
	if scale < 0 || !isFiniteLogRatio(scale) || scale == 1 {
		return
	}

	log.Quota = common.QuotaRound(float64(log.Quota) * scale)
	log.PromptTokens = common.QuotaRound(float64(log.PromptTokens) * scale)
	log.CompletionTokens = common.QuotaRound(float64(log.CompletionTokens) * scale)
	if feeQuota, ok := otherMap["fee_quota"].(float64); ok && feeQuota >= 0 && isFiniteLogRatio(feeQuota) {
		otherMap["fee_quota"] = common.QuotaRound(feeQuota * scale)
	}
	for _, key := range []string{
		"cache_tokens",
		"cache_creation_tokens",
		"cache_creation_tokens_5m",
		"cache_creation_tokens_1h",
	} {
		value, ok := otherMap[key].(float64)
		if !ok || value < 0 || !isFiniteLogRatio(value) {
			continue
		}
		otherMap[key] = common.QuotaRound(value * scale)
	}
}

func userVisibleLogDataMetrics(log *Log) (quota int, tokenUsed int) {
	if log == nil {
		return 0, 0
	}
	displayLog := *log
	otherMap, _ := common.StrToMap(displayLog.Other)
	if otherMap != nil {
		actualDisplay := actualLogGroupRatioDisplay(otherMap)
		display := storedUserDisplayGroupRatio(&displayLog, otherMap)
		applyUserLogDisplayMetrics(&displayLog, otherMap, actualDisplay, display)
	}
	tokenUsed = common.QuotaFromFloat(float64(displayLog.PromptTokens) + float64(displayLog.CompletionTokens))
	return displayLog.Quota, tokenUsed
}

func formatAdminLogGroupRatioDisplay(logs []*Log) {
	for i := range logs {
		otherMap, _ := common.StrToMap(logs[i].Other)
		if otherMap == nil {
			continue
		}

		delete(otherMap, "user_group_ratio_display_enabled")
		delete(otherMap, "user_group_ratio_display_mode")
		delete(otherMap, "user_group_ratio_display_value")
		display := storedUserDisplayGroupRatio(logs[i], otherMap)
		if display.applicable {
			otherMap["user_group_ratio_display_enabled"] = common.UserLogGroupRatioDisplayEnabled
			if common.UserLogGroupRatioDisplayEnabled && display.visible {
				otherMap["user_group_ratio_display_value"] = display.value
			}
		}
		logs[i].Other = common.MapToJsonStr(otherMap)
	}
}

func formatUserLogs(logs []*Log, startIdx int) {
	for i := range logs {
		logs[i].ChannelName = ""
		var otherMap map[string]interface{}
		otherMap, _ = common.StrToMap(logs[i].Other)
		if otherMap != nil {
			actualDisplay := actualLogGroupRatioDisplay(otherMap)
			display := storedUserDisplayGroupRatio(logs[i], otherMap)
			applyUserLogDisplayMetrics(logs[i], otherMap, actualDisplay, display)
			if adminInfo, ok := otherMap["admin_info"].(map[string]interface{}); ok {
				if adjustment, ok := adminInfo["model_token_adjustment"].(map[string]interface{}); ok {
					if billedUsage, ok := adjustment["billed"].(map[string]interface{}); ok {
						otherMap["billed_token_usage"] = billedUsage
					}
				}
			}
			// Remove admin-only debug fields.
			delete(otherMap, "admin_info")
			// Remove operation-audit details (operator/route info), admin-only.
			delete(otherMap, "audit_info")
			delete(otherMap, "group_ratio_display_mode")
			delete(otherMap, "user_group_ratio_display_enabled")
			delete(otherMap, "user_group_ratio_display_mode")
			delete(otherMap, "user_group_ratio_display_value")
			delete(otherMap, "group_ratio")
			delete(otherMap, "user_group_ratio")
			logs[i].UserDisplayGroupRatio = nil
			if common.UserLogGroupRatioDisplayEnabled && display.visible {
				value := display.value
				logs[i].UserDisplayGroupRatio = &value
				otherMap["group_ratio"] = value
				otherMap["group_ratio_display_mode"] = "snapshot"
			}
			// delete(otherMap, "reject_reason")
			delete(otherMap, "stream_status")
		}
		logs[i].Other = common.MapToJsonStr(otherMap)
	}
	assignDisplayLogIds(logs, startIdx)
}

func GetLogByTokenId(tokenId int) (logs []*Log, err error) {
	order := "id desc"
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		order = clickHouseLogOrder("")
	}
	err = LOG_DB.Model(&Log{}).Where("token_id = ?", tokenId).Order(order).Limit(common.MaxRecentItems).Find(&logs).Error
	formatUserLogs(logs, 0)
	return logs, err
}

func RecordLog(userId int, logType int, content string) {
	if logType == LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(userId, false)
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      logType,
		Content:   content,
	}
	err := createLog(log)
	if err != nil {
		common.SysLog("failed to record log: " + err.Error())
	}
}

func RecordQuotaIncreaseLog(userId int, quota int, source string, content string) {
	recordQuotaIncreaseLog(userId, quota, source, content, "", nil)
}

func RecordQuotaIncreaseLogWithAudit(userId int, quota int, source string, content string, requestId string, audit map[string]interface{}) {
	recordQuotaIncreaseLog(userId, quota, source, content, requestId, audit)
}

func recordQuotaIncreaseLog(userId int, quota int, source string, content string, requestId string, audit map[string]interface{}) {
	if userId <= 0 || quota <= 0 {
		return
	}
	username, _ := GetUsernameById(userId, false)
	other := map[string]interface{}{
		"source": source,
	}
	if len(audit) > 0 {
		other["admin_info"] = map[string]interface{}{
			"refund_audit": audit,
		}
	}
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      LogTypeQuotaIncrease,
		Content:   content,
		Quota:     quota,
		RequestId: requestId,
		Other:     common.MapToJsonStr(other),
	}
	if err := createLog(log); err != nil {
		common.SysLog("failed to record quota increase log: " + err.Error())
	}
}

func RecordAgentRedemptionRefundLog(userId int, redemption *Redemption, refund int) {
	if userId <= 0 || redemption == nil || refund <= 0 {
		return
	}
	username, _ := GetUsernameById(userId, false)
	other := map[string]interface{}{
		"op": buildOpField("agent_redemption_refund", map[string]interface{}{
			"redemption_id":    redemption.Id,
			"redemption_name":  redemption.Name,
			"redemption_quota": redemption.Quota,
			"refund_quota":     refund,
		}),
		"redemption_id":    redemption.Id,
		"redemption_name":  redemption.Name,
		"redemption_quota": redemption.Quota,
		"refund_quota":     refund,
	}
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      LogTypeRefund,
		Content:   fmt.Sprintf("代理兑换码退款 %s，兑换码ID %d", logger.LogQuota(refund), redemption.Id),
		Quota:     refund,
		Other:     common.MapToJsonStr(other),
	}
	if err := createLog(log); err != nil {
		common.SysLog("failed to record agent redemption refund log: " + err.Error())
	}
	RecordQuotaIncreaseLog(userId, refund, QuotaIncreaseSourceAgentRedemptionRefund, log.Content)
}

// RecordLogWithAdminInfo 记录操作日志，并将管理员相关信息存入 Other.admin_info，
func RecordLogWithAdminInfo(userId int, logType int, content string, adminInfo map[string]interface{}) {
	if logType == LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(userId, false)
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      logType,
		Content:   content,
	}
	if len(adminInfo) > 0 {
		other := map[string]interface{}{
			"admin_info": adminInfo,
		}
		log.Other = common.MapToJsonStr(other)
	}
	if err := createLog(log); err != nil {
		common.SysLog("failed to record log: " + err.Error())
	}
}

// buildOpField 构建语言无关的操作描述（写入 Other.op）。
// 前端依据 action(稳定操作标识) + params(结构化参数) 在渲染期用 i18n 本地化展示，
// 因此不在数据库中存储自然语言句子。
func buildOpField(action string, params map[string]interface{}) map[string]interface{} {
	op := map[string]interface{}{
		"action": action,
	}
	if len(params) > 0 {
		op["params"] = params
	}
	return op
}

// RecordLoginLog 记录用户登录成功的审计日志（type=LogTypeLogin）。
// username 由调用方传入（登录流程已持有用户对象），避免额外的数据库查询。
// content 为英文兜底文本（用于导出）；action+params 供前端本地化渲染。
// extra 可携带 login_method、user_agent 等附加信息（普通用户可见）。
func RecordLoginLog(userId int, username string, content string, ip string, action string, params map[string]interface{}, extra map[string]interface{}) {
	if normalized, err := NormalizeIPAddress(ip); err == nil {
		ip = normalized
	}
	other := map[string]interface{}{}
	for k, v := range extra {
		other[k] = v
	}
	other["op"] = buildOpField(action, params)
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      LogTypeLogin,
		Content:   content,
		Ip:        ip,
		Other:     common.MapToJsonStr(other),
	}
	if err := createLog(log); err != nil {
		common.SysLog("failed to record login log: " + err.Error())
	}
}

func GetUserLoginIPStats(userId int) ([]UserLoginIPStat, error) {
	stats := make([]UserLoginIPStat, 0)
	err := LOG_DB.Model(&Log{}).
		Select("ip, MAX(created_at) AS last_login_at, COUNT(*) AS login_count").
		Where("user_id = ? AND type = ? AND ip <> ''", userId, LogTypeLogin).
		Group("ip").
		Order("last_login_at DESC").
		Scan(&stats).Error
	return stats, err
}

type UserQuotaIncreaseLog struct {
	Id        int    `json:"id"`
	RequestId string `json:"request_id"`
	CreatedAt int64  `json:"created_at"`
	Quota     int    `json:"quota"`
	Source    string `json:"source"`
	Content   string `json:"content"`
}

func GetUserQuotaIncreaseLogs(userId int, startIdx int, num int) ([]UserQuotaIncreaseLog, int64, error) {
	query := LOG_DB.Model(&Log{}).Where("user_id = ? AND type = ?", userId, LogTypeQuotaIncrease)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := "id desc"
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		order = clickHouseLogOrder("")
	}
	var logs []Log
	if err := query.Order(order).Offset(startIdx).Limit(num).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	items := make([]UserQuotaIncreaseLog, 0, len(logs))
	for i := range logs {
		metadata := struct {
			Source string `json:"source"`
		}{}
		if logs[i].Other != "" {
			_ = common.UnmarshalJsonStr(logs[i].Other, &metadata)
		}
		items = append(items, UserQuotaIncreaseLog{
			Id:        logs[i].Id,
			RequestId: logs[i].RequestId,
			CreatedAt: logs[i].CreatedAt,
			Quota:     logs[i].Quota,
			Source:    metadata.Source,
			Content:   logs[i].Content,
		})
	}
	return items, total, nil
}

// RecordOperationAuditLog 记录管理/高危操作审计日志（type=LogTypeManage）。
// logUserId 为日志归属者，管理审计日志应归属实际操作者；目标资源/用户放入
// action params。username 内部按 logUserId 查询。content 为英文兜底文本（供导出使用）。
// action+params 写入 Other.op，供前端本地化渲染（普通用户可见，不含敏感信息）。
// adminInfo 存放操作者身份（写入 Other.admin_info，普通用户查询时剥离）；
// auditInfo 存放路由/方法/结果等中间件兜底信息（写入 Other.audit_info，普通用户查询时剥离）。
func RecordOperationAuditLog(logUserId int, content string, ip string, action string, params map[string]interface{}, adminInfo map[string]interface{}, auditInfo map[string]interface{}) {
	username, _ := GetUsernameById(logUserId, false)
	other := map[string]interface{}{
		"op": buildOpField(action, params),
	}
	if len(adminInfo) > 0 {
		other["admin_info"] = adminInfo
	}
	if len(auditInfo) > 0 {
		other["audit_info"] = auditInfo
	}
	log := &Log{
		UserId:    logUserId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      LogTypeManage,
		Content:   content,
		Ip:        ip,
		Other:     common.MapToJsonStr(other),
	}
	if err := createLog(log); err != nil {
		common.SysLog("failed to record operation audit log: " + err.Error())
	}
}

func RecordTopupLog(userId int, quota int, content string, callerIp string, paymentMethod string, callbackPaymentMethod string) {
	username, _ := GetUsernameById(userId, false)
	adminInfo := map[string]interface{}{
		"server_ip":               common.GetIp(),
		"node_name":               common.NodeName,
		"caller_ip":               callerIp,
		"payment_method":          paymentMethod,
		"callback_payment_method": callbackPaymentMethod,
		"version":                 common.Version,
	}
	other := map[string]interface{}{
		"admin_info": adminInfo,
	}
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      LogTypeTopup,
		Content:   content,
		Ip:        callerIp,
		Other:     common.MapToJsonStr(other),
	}
	err := createLog(log)
	if err != nil {
		common.SysLog("failed to record topup log: " + err.Error())
	}
	RecordQuotaIncreaseLog(userId, quota, QuotaIncreaseSourceOnlineRecharge, content)
}

func RecordErrorLog(c *gin.Context, userId int, channelId int, modelName string, tokenName string, content string, tokenId int, useTimeSeconds int,
	isStream bool, group string, other map[string]interface{}) {
	logger.LogInfo(c, fmt.Sprintf("record error log: userId=%d, channelId=%d, modelName=%s, tokenName=%s, content=%s", userId, channelId, modelName, tokenName, common.LocalLogPreview(content)))
	username := c.GetString("username")
	requestId := c.GetString(common.RequestIdKey)
	upstreamRequestId := c.GetString(common.UpstreamRequestIdKey)
	otherStr := common.MapToJsonStr(other)
	// 判断是否需要记录 IP
	needRecordIp := false
	if settingMap, err := GetUserSetting(userId, false); err == nil {
		if settingMap.RecordIpLog {
			needRecordIp = true
		}
	}
	log := &Log{
		UserId:           userId,
		Username:         username,
		CreatedAt:        common.GetTimestamp(),
		Type:             LogTypeError,
		Content:          content,
		PromptTokens:     0,
		CompletionTokens: 0,
		TokenName:        tokenName,
		ModelName:        modelName,
		Quota:            0,
		ChannelId:        channelId,
		TokenId:          tokenId,
		UseTime:          useTimeSeconds,
		IsStream:         isStream,
		Group:            group,
		Ip: func() string {
			if needRecordIp {
				return c.ClientIP()
			}
			return ""
		}(),
		RequestId:         requestId,
		UpstreamRequestId: upstreamRequestId,
		Other:             otherStr,
	}
	err := createLog(log)
	if err != nil {
		logger.LogError(c, "failed to record log: "+err.Error())
	}
}

type RecordConsumeLogParams struct {
	ChannelId        int                    `json:"channel_id"`
	PromptTokens     int                    `json:"prompt_tokens"`
	CompletionTokens int                    `json:"completion_tokens"`
	ModelName        string                 `json:"model_name"`
	TokenName        string                 `json:"token_name"`
	Quota            int                    `json:"quota"`
	Content          string                 `json:"content"`
	TokenId          int                    `json:"token_id"`
	UseTimeSeconds   int                    `json:"use_time_seconds"`
	IsStream         bool                   `json:"is_stream"`
	Group            string                 `json:"group"`
	Other            map[string]interface{} `json:"other"`
}

func RecordConsumeLog(c *gin.Context, userId int, params RecordConsumeLogParams) {
	if !common.LogConsumeEnabled {
		return
	}
	logger.LogInfo(c, fmt.Sprintf("record consume log: userId=%d, params=%s", userId, common.GetJsonString(params)))
	username := c.GetString("username")
	requestId := c.GetString(common.RequestIdKey)
	upstreamRequestId := c.GetString(common.UpstreamRequestIdKey)
	createdAt := common.GetTimestamp()
	otherStr := common.MapToJsonStr(params.Other)
	// 判断是否需要记录 IP
	needRecordIp := false
	if settingMap, err := GetUserSetting(userId, false); err == nil {
		if settingMap.RecordIpLog {
			needRecordIp = true
		}
	}
	log := &Log{
		UserId:           userId,
		Username:         username,
		CreatedAt:        createdAt,
		Type:             LogTypeConsume,
		Content:          params.Content,
		PromptTokens:     params.PromptTokens,
		CompletionTokens: params.CompletionTokens,
		TokenName:        params.TokenName,
		ModelName:        params.ModelName,
		Quota:            params.Quota,
		ChannelId:        params.ChannelId,
		TokenId:          params.TokenId,
		UseTime:          params.UseTimeSeconds,
		IsStream:         params.IsStream,
		Group:            params.Group,
		Ip: func() string {
			if needRecordIp {
				return c.ClientIP()
			}
			return ""
		}(),
		RequestId:         requestId,
		UpstreamRequestId: upstreamRequestId,
		Other:             otherStr,
	}
	err := createLog(log)
	if err != nil {
		logger.LogError(c, "failed to record log: "+err.Error())
	}
	if common.DataExportEnabled {
		userDisplayQuota, userDisplayTokenUsed := userVisibleLogDataMetrics(log)
		tokenUsed := common.QuotaFromFloat(float64(params.PromptTokens) + float64(params.CompletionTokens))
		LogQuotaData(QuotaDataLogParams{
			UserID:               userId,
			Username:             username,
			ModelName:            params.ModelName,
			Quota:                params.Quota,
			CreatedAt:            createdAt,
			TokenUsed:            tokenUsed,
			UseGroup:             params.Group,
			TokenID:              params.TokenId,
			ChannelID:            params.ChannelId,
			NodeName:             common.NodeName,
			UserDisplayQuota:     common.GetPointer(userDisplayQuota),
			UserDisplayTokenUsed: common.GetPointer(userDisplayTokenUsed),
		})
	}
}

type RecordTaskBillingLogParams struct {
	UserId    int
	LogType   int
	Content   string
	ChannelId int
	ModelName string
	Quota     int
	TokenId   int
	Group     string
	Other     map[string]interface{}
	NodeName  string // 任务发起节点；为空时回退当前节点
}

func RecordTaskBillingLog(params RecordTaskBillingLogParams) {
	if params.LogType == LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(params.UserId, false)
	tokenName := ""
	if params.TokenId > 0 {
		if token, err := GetTokenById(params.TokenId); err == nil {
			tokenName = token.Name
		}
	}
	createdAt := common.GetTimestamp()
	log := &Log{
		UserId:    params.UserId,
		Username:  username,
		CreatedAt: createdAt,
		Type:      params.LogType,
		Content:   params.Content,
		TokenName: tokenName,
		ModelName: params.ModelName,
		Quota:     params.Quota,
		ChannelId: params.ChannelId,
		TokenId:   params.TokenId,
		Group:     params.Group,
		Other:     common.MapToJsonStr(params.Other),
	}
	err := createLog(log)
	if err != nil {
		common.SysLog("failed to record task billing log: " + err.Error())
	}
	if params.LogType == LogTypeConsume && common.DataExportEnabled {
		nodeName := params.NodeName
		if nodeName == "" {
			nodeName = common.NodeName
		}
		userDisplayQuota, userDisplayTokenUsed := userVisibleLogDataMetrics(log)
		LogQuotaData(QuotaDataLogParams{
			UserID:               params.UserId,
			Username:             username,
			ModelName:            params.ModelName,
			Quota:                params.Quota,
			CreatedAt:            createdAt,
			UseGroup:             params.Group,
			TokenID:              params.TokenId,
			ChannelID:            params.ChannelId,
			NodeName:             nodeName,
			UserDisplayQuota:     common.GetPointer(userDisplayQuota),
			UserDisplayTokenUsed: common.GetPointer(userDisplayTokenUsed),
		})
	}
}

func GetAllLogs(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, startIdx int, num int, channel int, group string, requestId string, upstreamRequestId string) (logs []*Log, total int64, err error) {
	var tx *gorm.DB
	if logType == LogTypeUnknown {
		tx = LOG_DB
	} else {
		tx = LOG_DB.Where("logs.type = ?", logType)
	}

	if tx, err = applyExplicitLogTextFilter(tx, "logs.model_name", modelName); err != nil {
		return nil, 0, err
	}
	if tx, err = applyExplicitLogTextFilter(tx, "logs.username", username); err != nil {
		return nil, 0, err
	}
	if tokenName != "" {
		tx = tx.Where("logs.token_name = ?", tokenName)
	}
	if requestId != "" {
		tx = tx.Where("logs.request_id = ?", requestId)
	}
	if upstreamRequestId != "" {
		tx = tx.Where("logs.upstream_request_id = ?", upstreamRequestId)
	}
	if startTimestamp != 0 {
		tx = tx.Where("logs.created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("logs.created_at <= ?", endTimestamp)
	}
	if channel != 0 {
		tx = tx.Where("logs.channel_id = ?", channel)
	}
	if group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", group)
	}
	err = tx.Model(&Log{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	order := "logs.created_at desc, logs.id desc"
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		order = clickHouseLogOrder("logs.")
	}
	err = tx.Order(order).Limit(num).Offset(startIdx).Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		assignDisplayLogIds(logs, startIdx)
	}

	channelIds := types.NewSet[int]()
	for _, log := range logs {
		if log.ChannelId != 0 {
			channelIds.Add(log.ChannelId)
		}
	}

	if channelIds.Len() > 0 {
		var channels []struct {
			Id   int    `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		if common.MemoryCacheEnabled {
			// Cache get channel
			for _, channelId := range channelIds.Items() {
				if cacheChannel, err := CacheGetChannel(channelId); err == nil {
					channels = append(channels, struct {
						Id   int    `gorm:"column:id"`
						Name string `gorm:"column:name"`
					}{
						Id:   channelId,
						Name: cacheChannel.Name,
					})
				}
			}
		} else {
			// Bulk query channels from DB
			if err = DB.Table("channels").Select("id, name").Where("id IN ?", channelIds.Items()).Find(&channels).Error; err != nil {
				return logs, total, err
			}
		}
		channelMap := make(map[int]string, len(channels))
		for _, channel := range channels {
			channelMap[channel.Id] = channel.Name
		}
		for i := range logs {
			logs[i].ChannelName = channelMap[logs[i].ChannelId]
		}
	}
	formatAdminLogGroupRatioDisplay(logs)

	return logs, total, err
}

const logSearchCountLimit = 10000

func GetUserLogs(userId int, logType int, startTimestamp int64, endTimestamp int64, modelName string, tokenName string, startIdx int, num int, group string, requestId string, upstreamRequestId string) (logs []*Log, total int64, err error) {
	var tx *gorm.DB
	if logType == LogTypeUnknown {
		tx = LOG_DB.Where("logs.user_id = ?", userId)
	} else {
		tx = LOG_DB.Where("logs.user_id = ? and logs.type = ?", userId, logType)
	}

	if tx, err = applyExplicitLogTextFilter(tx, "logs.model_name", modelName); err != nil {
		return nil, 0, err
	}
	if tokenName != "" {
		tx = tx.Where("logs.token_name = ?", tokenName)
	}
	if requestId != "" {
		tx = tx.Where("logs.request_id = ?", requestId)
	}
	if upstreamRequestId != "" {
		tx = tx.Where("logs.upstream_request_id = ?", upstreamRequestId)
	}
	if startTimestamp != 0 {
		tx = tx.Where("logs.created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("logs.created_at <= ?", endTimestamp)
	}
	if group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", group)
	}
	err = tx.Model(&Log{}).Limit(logSearchCountLimit).Count(&total).Error
	if err != nil {
		common.SysError("failed to count user logs: " + err.Error())
		return nil, 0, errors.New("查询日志失败")
	}
	order := "logs.id desc"
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		order = clickHouseLogOrder("logs.")
	}
	err = tx.Order(order).Limit(num).Offset(startIdx).Find(&logs).Error
	if err != nil {
		common.SysError("failed to search user logs: " + err.Error())
		return nil, 0, errors.New("查询日志失败")
	}

	formatUserLogs(logs, startIdx)
	return logs, total, err
}

type Stat struct {
	Quota int `json:"quota"`
	Rpm   int `json:"rpm"`
	Tpm   int `json:"tpm"`
}

func SumUsedQuota(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, channel int, group string) (stat Stat, err error) {
	tx := LOG_DB.Table("logs").Select("COALESCE(sum(quota), 0) quota")

	// 为rpm和tpm创建单独的查询
	rpmTpmQuery := LOG_DB.Table("logs").Select("count(*) rpm, COALESCE(sum(prompt_tokens), 0) + COALESCE(sum(completion_tokens), 0) tpm")

	if tx, err = applyExplicitLogTextFilter(tx, "username", username); err != nil {
		return stat, err
	}
	if rpmTpmQuery, err = applyExplicitLogTextFilter(rpmTpmQuery, "username", username); err != nil {
		return stat, err
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
		rpmTpmQuery = rpmTpmQuery.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if tx, err = applyExplicitLogTextFilter(tx, "model_name", modelName); err != nil {
		return stat, err
	}
	if rpmTpmQuery, err = applyExplicitLogTextFilter(rpmTpmQuery, "model_name", modelName); err != nil {
		return stat, err
	}
	if channel != 0 {
		tx = tx.Where("channel_id = ?", channel)
		rpmTpmQuery = rpmTpmQuery.Where("channel_id = ?", channel)
	}
	if group != "" {
		tx = tx.Where(logGroupCol+" = ?", group)
		rpmTpmQuery = rpmTpmQuery.Where(logGroupCol+" = ?", group)
	}

	tx = tx.Where("type = ?", LogTypeConsume)
	rpmTpmQuery = rpmTpmQuery.Where("type = ?", LogTypeConsume)

	// 只统计最近60秒的rpm和tpm
	rpmTpmQuery = rpmTpmQuery.Where("created_at >= ?", time.Now().Add(-60*time.Second).Unix())

	// 执行查询
	if err := tx.Scan(&stat).Error; err != nil {
		common.SysError("failed to query log stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}
	if err := rpmTpmQuery.Scan(&stat).Error; err != nil {
		common.SysError("failed to query rpm/tpm stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}

	return stat, nil
}

func SumUserVisibleUsedQuota(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, channel int, group string) (stat Stat, err error) {
	stat, err = SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group)
	if err != nil {
		return stat, err
	}

	tx := LOG_DB.Model(&Log{}).
		Select("type, prompt_tokens, completion_tokens, user_display_group_ratio, other").
		Where("type = ?", LogTypeConsume).
		Where("created_at >= ?", time.Now().Add(-60*time.Second).Unix())
	if tx, err = applyExplicitLogTextFilter(tx, "username", username); err != nil {
		return stat, err
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if tx, err = applyExplicitLogTextFilter(tx, "model_name", modelName); err != nil {
		return stat, err
	}
	if channel != 0 {
		tx = tx.Where("channel_id = ?", channel)
	}
	if group != "" {
		tx = tx.Where(logGroupCol+" = ?", group)
	}

	var logs []*Log
	if err = tx.Find(&logs).Error; err != nil {
		common.SysError("failed to query user-visible tpm: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}
	var visibleTPM int64
	for _, log := range logs {
		otherMap, _ := common.StrToMap(log.Other)
		if otherMap != nil {
			actualDisplay := actualLogGroupRatioDisplay(otherMap)
			display := storedUserDisplayGroupRatio(log, otherMap)
			applyUserLogDisplayMetrics(log, otherMap, actualDisplay, display)
		}
		visibleTPM += int64(log.PromptTokens) + int64(log.CompletionTokens)
	}
	stat.Tpm = common.QuotaFromFloat(float64(visibleTPM))
	return stat, nil
}

func SumUsedToken(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string) (token int) {
	tx := LOG_DB.Table("logs").Select("COALESCE(sum(prompt_tokens), 0) + COALESCE(sum(completion_tokens), 0)")
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	tx.Where("type = ?", LogTypeConsume).Scan(&token)
	return token
}

func SumUserUsedToken(userId int) (token int64, err error) {
	err = LOG_DB.Table("logs").
		Select("COALESCE(sum(prompt_tokens), 0) + COALESCE(sum(completion_tokens), 0)").
		Where("user_id = ? AND type = ?", userId, LogTypeConsume).
		Scan(&token).Error
	return token, err
}

type UserUsageSummary struct {
	TotalTokens int64 `gorm:"column:total_tokens"`
	TotalQuota  int64 `gorm:"column:total_quota"`
}

type UserGroupUsage struct {
	UseGroup  string `gorm:"column:use_group"`
	Quota     int64  `gorm:"column:quota"`
	TokenUsed int64  `gorm:"column:token_used"`
}

func GetUserUsageSummaryBetween(userId int, startTimestamp int64, endTimestamp int64) (summary UserUsageSummary, err error) {
	err = LOG_DB.Table("logs").
		Select("COALESCE(sum(prompt_tokens), 0) + COALESCE(sum(completion_tokens), 0) AS total_tokens, COALESCE(sum(quota), 0) AS total_quota").
		Where(
			"user_id = ? AND type = ? AND created_at >= ? AND created_at < ?",
			userId,
			LogTypeConsume,
			startTimestamp,
			endTimestamp,
		).
		Scan(&summary).Error
	return summary, err
}

func GetUserGroupUsage(userId int, groups []string) ([]UserGroupUsage, error) {
	usage := make([]UserGroupUsage, 0, len(groups))
	if len(groups) == 0 {
		return usage, nil
	}
	err := LOG_DB.Table("logs").
		Clauses(clause.Select{Columns: []clause.Column{
			{Name: "group", Alias: "use_group"},
			{Name: "COALESCE(SUM(quota), 0)", Alias: "quota", Raw: true},
			{Name: "COALESCE(SUM(prompt_tokens), 0) + COALESCE(SUM(completion_tokens), 0)", Alias: "token_used", Raw: true},
		}}).
		Where(map[string]interface{}{
			"user_id": userId,
			"type":    LogTypeConsume,
			"group":   groups,
		}).
		Clauses(clause.GroupBy{Columns: []clause.Column{{Name: "group"}}}).
		Scan(&usage).Error
	return usage, err
}

func CountOldLog(ctx context.Context, targetTimestamp int64) (int64, error) {
	var total int64
	if err := LOG_DB.WithContext(ctx).Model(&Log{}).Where("created_at < ?", targetTimestamp).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func DeleteOldLogBatch(ctx context.Context, targetTimestamp int64, limit int) (int64, error) {
	if limit <= 0 {
		limit = 100
	}
	if nil != ctx.Err() {
		return 0, ctx.Err()
	}

	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		// ClickHouse DELETE is a heavy mutation that rewrites data parts, so
		// per-batch mutations would be pathologically slow. Remove all matching
		// rows in a single synchronous mutation regardless of limit; the reported
		// count lets the caller's progress loop complete in one pass.
		total, err := CountOldLog(ctx, targetTimestamp)
		if err != nil {
			return 0, err
		}
		if total == 0 {
			return 0, nil
		}
		if err := LOG_DB.WithContext(ctx).Exec(
			"ALTER TABLE logs DELETE WHERE created_at < ? SETTINGS mutations_sync = 1",
			targetTimestamp,
		).Error; err != nil {
			return 0, err
		}
		return total, nil
	}

	result := LOG_DB.WithContext(ctx).Where("created_at < ?", targetTimestamp).Limit(limit).Delete(&Log{})
	if nil != result.Error {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
