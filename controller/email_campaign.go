package controller

import (
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type emailCampaignRequest struct {
	Name          string `json:"name"`
	Subject       string `json:"subject"`
	Content       string `json:"content"`
	Mode          string `json:"mode"`
	TargetType    string `json:"target_type"`
	TargetUserIds []int  `json:"target_user_ids"`
	TriggerType   string `json:"trigger_type"`
	TriggerDays   int    `json:"trigger_days"`
	ScheduledAt   int64  `json:"scheduled_at"`
	Draft         bool   `json:"draft"`
}

type emailCampaignResponse struct {
	*model.EmailCampaign
	TargetUserIds []int `json:"target_user_ids"`
}

type emailCampaignUserResolveRequest struct {
	UserIds []int `json:"user_ids"`
}

func emailCampaignToResponse(campaign *model.EmailCampaign) emailCampaignResponse {
	userIDs, err := campaign.TargetUserIDList()
	if err != nil {
		userIDs = []int{}
	}
	return emailCampaignResponse{EmailCampaign: campaign, TargetUserIds: userIDs}
}

func validateEmailCampaignRequest(req *emailCampaignRequest, requireRunnable bool) (*model.EmailCampaign, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Subject = strings.TrimSpace(req.Subject)
	req.Content = strings.TrimSpace(req.Content)
	if req.Name == "" || utf8.RuneCountInString(req.Name) > 128 {
		return nil, errors.New("任务名称不能为空且不能超过 128 个字符")
	}
	if req.Subject == "" || utf8.RuneCountInString(req.Subject) > 255 {
		return nil, errors.New("邮件主题不能为空且不能超过 255 个字符")
	}
	if req.Content == "" || len(req.Content) > 200000 {
		return nil, errors.New("邮件正文不能为空且不能超过 200000 字节")
	}

	campaign := &model.EmailCampaign{
		Name:        req.Name,
		Subject:     req.Subject,
		Content:     req.Content,
		Mode:        req.Mode,
		TargetType:  req.TargetType,
		TriggerType: req.TriggerType,
		TriggerDays: req.TriggerDays,
		ScheduledAt: req.ScheduledAt,
	}

	switch req.Mode {
	case model.EmailCampaignModeImmediate:
		campaign.ScheduledAt = 0
		campaign.TriggerType = ""
		campaign.TriggerDays = 0
	case model.EmailCampaignModeScheduled:
		if requireRunnable && req.ScheduledAt <= common.GetTimestamp() {
			return nil, errors.New("定时发送时间必须晚于当前时间")
		}
		campaign.TriggerType = ""
		campaign.TriggerDays = 0
	case model.EmailCampaignModeConditional:
		if req.TriggerType != model.EmailCampaignTriggerSubscriptionExpiring {
			return nil, errors.New("不支持的条件触发类型")
		}
		if req.TriggerDays < 1 || req.TriggerDays > 90 {
			return nil, errors.New("订阅到期提醒天数必须在 1 到 90 之间")
		}
		campaign.TargetType = model.EmailCampaignTargetActiveSubscribers
		campaign.ScheduledAt = 0
	default:
		return nil, errors.New("不支持的发送模式")
	}

	if req.Mode != model.EmailCampaignModeConditional {
		switch req.TargetType {
		case model.EmailCampaignTargetAllUsers, model.EmailCampaignTargetActiveSubscribers:
			if err := campaign.SetTargetUserIDList([]int{}); err != nil {
				return nil, err
			}
		case model.EmailCampaignTargetSelectedUsers:
			seen := make(map[int]struct{}, len(req.TargetUserIds))
			userIDs := make([]int, 0, len(req.TargetUserIds))
			for _, userID := range req.TargetUserIds {
				if userID <= 0 {
					continue
				}
				if _, exists := seen[userID]; exists {
					continue
				}
				seen[userID] = struct{}{}
				userIDs = append(userIDs, userID)
			}
			if len(userIDs) == 0 || len(userIDs) > 5000 {
				return nil, errors.New("指定用户数量必须在 1 到 5000 之间")
			}
			if err := campaign.SetTargetUserIDList(userIDs); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("不支持的收件人范围")
		}
	} else if err := campaign.SetTargetUserIDList([]int{}); err != nil {
		return nil, err
	}
	return campaign, nil
}

func ensureEmailCampaignSMTPConfigured() error {
	if strings.TrimSpace(common.SMTPServer) == "" {
		return errors.New("请先在 SMTP 邮件设置中配置邮件服务器")
	}
	return nil
}

func activateEmailCampaign(campaign *model.EmailCampaign) {
	now := common.GetTimestamp()
	if campaign.Mode == model.EmailCampaignModeConditional {
		campaign.Status = model.EmailCampaignStatusActive
		campaign.NextRunAt = now
		return
	}
	campaign.Status = model.EmailCampaignStatusScheduled
	if campaign.Mode == model.EmailCampaignModeScheduled {
		campaign.NextRunAt = campaign.ScheduledAt
	} else {
		campaign.NextRunAt = now
	}
}

func ListEmailCampaigns(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	campaigns, total, err := model.ListEmailCampaigns(pageInfo.GetStartIdx(), pageInfo.GetPageSize(), c.Query("search"), c.Query("status"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items := make([]emailCampaignResponse, 0, len(campaigns))
	for _, campaign := range campaigns {
		items = append(items, emailCampaignToResponse(campaign))
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetEmailCampaign(c *gin.Context) {
	campaign, err := getEmailCampaignFromParam(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, emailCampaignToResponse(campaign))
}

func GetEmailCampaignStats(c *gin.Context) {
	stats, err := model.GetEmailCampaignStats()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, stats)
}

func SearchEmailCampaignUsers(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	users, total, err := model.SearchEmailCampaignUserOptions(
		c.Query("keyword"),
		pageInfo.GetStartIdx(),
		pageInfo.GetPageSize(),
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(users)
	common.ApiSuccess(c, pageInfo)
}

func ResolveEmailCampaignUsers(c *gin.Context) {
	var req emailCampaignUserResolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if len(req.UserIds) > 5000 {
		common.ApiErrorMsg(c, "指定用户数量不能超过 5000")
		return
	}
	seen := make(map[int]struct{}, len(req.UserIds))
	userIDs := make([]int, 0, len(req.UserIds))
	for _, userID := range req.UserIds {
		if userID <= 0 {
			continue
		}
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}
		userIDs = append(userIDs, userID)
	}
	users, err := model.GetEmailCampaignUserOptionsByIds(userIDs)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, users)
}

func PreviewEmailCampaign(c *gin.Context) {
	var req emailCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	campaign, err := validateEmailCampaignRequest(&req, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	count, err := service.PreviewEmailCampaignAudience(campaign)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"recipient_count": count})
}

func CreateEmailCampaign(c *gin.Context) {
	var req emailCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	campaign, err := validateEmailCampaignRequest(&req, !req.Draft)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	campaign.CreatedBy = c.GetInt("id")
	if req.Draft {
		campaign.Status = model.EmailCampaignStatusDraft
	} else {
		if err := ensureEmailCampaignSMTPConfigured(); err != nil {
			common.ApiError(c, err)
			return
		}
		activateEmailCampaign(campaign)
	}
	if err := model.CreateEmailCampaign(campaign); err != nil {
		common.ApiError(c, err)
		return
	}
	if campaign.NextRunAt > 0 && campaign.NextRunAt <= common.GetTimestamp() {
		_, _, _ = service.EnqueueSystemTask(model.SystemTaskTypeEmailCampaignDispatch, nil)
	}
	common.ApiSuccess(c, emailCampaignToResponse(campaign))
}

func UpdateEmailCampaign(c *gin.Context) {
	existing, err := getEmailCampaignFromParam(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if existing.Status != model.EmailCampaignStatusDraft && existing.Status != model.EmailCampaignStatusPaused {
		common.ApiErrorMsg(c, "只有草稿或已暂停的邮件任务可以编辑")
		return
	}
	var req emailCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	updated, err := validateEmailCampaignRequest(&req, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	updated.Id = existing.Id
	updated.CreatedBy = existing.CreatedBy
	updated.CreatedAt = existing.CreatedAt
	updated.Status = existing.Status
	updated.RecipientCount = existing.RecipientCount
	updated.SuccessCount = existing.SuccessCount
	updated.FailedCount = existing.FailedCount
	updated.SkippedCount = existing.SkippedCount
	updated.LastRunAt = existing.LastRunAt
	if err := model.UpdateEmailCampaign(updated); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, emailCampaignToResponse(updated))
}

func ActivateEmailCampaign(c *gin.Context) {
	campaign, err := getEmailCampaignFromParam(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if campaign.Status != model.EmailCampaignStatusDraft && campaign.Status != model.EmailCampaignStatusPaused {
		common.ApiErrorMsg(c, "当前邮件任务不能启动")
		return
	}
	if err := ensureEmailCampaignSMTPConfigured(); err != nil {
		common.ApiError(c, err)
		return
	}
	if campaign.Mode == model.EmailCampaignModeScheduled && campaign.ScheduledAt <= common.GetTimestamp() {
		common.ApiErrorMsg(c, "定时发送时间已过，请先编辑发送时间")
		return
	}
	activateEmailCampaign(campaign)
	if err := model.UpdateEmailCampaign(campaign); err != nil {
		common.ApiError(c, err)
		return
	}
	if campaign.NextRunAt <= common.GetTimestamp() {
		_, _, _ = service.EnqueueSystemTask(model.SystemTaskTypeEmailCampaignDispatch, nil)
	}
	common.ApiSuccess(c, emailCampaignToResponse(campaign))
}

func PauseEmailCampaign(c *gin.Context) {
	id, err := parseEmailCampaignId(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.PauseEmailCampaign(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func RetryEmailCampaign(c *gin.Context) {
	id, err := parseEmailCampaignId(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := ensureEmailCampaignSMTPConfigured(); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.RetryFailedEmailCampaign(id, common.GetTimestamp()); err != nil {
		common.ApiError(c, err)
		return
	}
	_, _, _ = service.EnqueueSystemTask(model.SystemTaskTypeEmailCampaignDispatch, nil)
	common.ApiSuccess(c, nil)
}

func DeleteEmailCampaign(c *gin.Context) {
	id, err := parseEmailCampaignId(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeleteEmailCampaign(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func ListEmailCampaignDeliveries(c *gin.Context) {
	id, err := parseEmailCampaignId(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if _, err := model.GetEmailCampaignById(id); err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo := common.GetPageQuery(c)
	deliveries, total, err := model.ListEmailDeliveries(id, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), c.Query("status"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(deliveries)
	common.ApiSuccess(c, pageInfo)
}

func parseEmailCampaignId(c *gin.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("无效的邮件任务 ID")
	}
	return id, nil
}

func getEmailCampaignFromParam(c *gin.Context) (*model.EmailCampaign, error) {
	id, err := parseEmailCampaignId(c)
	if err != nil {
		return nil, err
	}
	campaign, err := model.GetEmailCampaignById(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("邮件任务不存在")
	}
	return campaign, err
}
