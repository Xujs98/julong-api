package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	UserPresenceEventOnline             = "online"
	UserPresenceEventOffline            = "offline"
	UserPresenceActivitySourceLogin     = "login"
	UserPresenceActivitySourceDashboard = "dashboard"
	UserPresenceActivitySourceAPI       = "api"
	defaultUserPresenceOfflineMinutes   = 5
	minUserPresenceOfflineMinutes       = 1
	maxUserPresenceOfflineMinutes       = 1440
	maxUserPresenceMonitoredUsers       = 500
	userPresenceActivityWriteInterval   = 30 * time.Second
	userPresenceMaximumSourceLength     = 32
	userPresenceMaximumIPLength         = 45
	userPresenceMaximumUserAgentLength  = 512
)

type UserPresenceEmailDispatchResult struct {
	OfflineUserCount int `json:"offline_user_count"`
	RecipientCount   int `json:"recipient_count"`
}

type UserPresenceTestEmailResult struct {
	RecipientCount  int      `json:"recipient_count"`
	EmailCount      int      `json:"email_count"`
	MonitoredUserId int      `json:"monitored_user_id"`
	Events          []string `json:"events"`
}

type userPresenceRuntimeConfig struct {
	Enabled          bool
	Events           map[string]struct{}
	MonitoredUserIDs []int
	MonitoredUsers   map[int]struct{}
	RecipientUserIDs []int
	OfflineMinutes   int
}

var (
	userPresenceRuntimeMu        sync.RWMutex
	userPresenceRuntimeSignature string
	userPresenceRuntime          userPresenceRuntimeConfig
	userPresenceActivityMu       sync.Mutex
	userPresenceActivityWrites   = map[int]int64{}
)

func currentUserPresenceRuntimeConfig() userPresenceRuntimeConfig {
	common.OptionMapRWMutex.RLock()
	enabledRaw := common.OptionMap[common.UserPresenceEmailEnabledOptionKey]
	eventsRaw := common.OptionMap[common.UserPresenceEmailEventsOptionKey]
	monitoredRaw := common.OptionMap[common.UserPresenceEmailMonitoredUserIDsOptionKey]
	recipientsRaw := common.OptionMap[common.UserPresenceEmailRecipientUserIDsOptionKey]
	offlineMinutesRaw := common.OptionMap[common.UserPresenceOfflineMinutesOptionKey]
	common.OptionMapRWMutex.RUnlock()

	signature := strings.Join([]string{enabledRaw, eventsRaw, monitoredRaw, recipientsRaw, offlineMinutesRaw}, "\x00")
	userPresenceRuntimeMu.RLock()
	if signature == userPresenceRuntimeSignature {
		config := userPresenceRuntime
		userPresenceRuntimeMu.RUnlock()
		return config
	}
	userPresenceRuntimeMu.RUnlock()

	events := []string{}
	if strings.TrimSpace(eventsRaw) != "" {
		if err := common.UnmarshalJsonStr(eventsRaw, &events); err != nil {
			common.SysError("failed to load user presence email events: " + err.Error())
		}
	}
	events = normalizeUserPresenceEmailEvents(events)
	if len(events) == 0 && strings.TrimSpace(eventsRaw) == "" {
		events = []string{UserPresenceEventOnline, UserPresenceEventOffline}
	}
	monitoredUserIDs := decodeUserPresenceIDsForRuntime(monitoredRaw, "monitored users")
	recipientUserIDs := decodeUserPresenceIDsForRuntime(recipientsRaw, "recipients")
	offlineMinutes := defaultUserPresenceOfflineMinutes
	if parsed, err := strconv.Atoi(strings.TrimSpace(offlineMinutesRaw)); err == nil && parsed >= minUserPresenceOfflineMinutes && parsed <= maxUserPresenceOfflineMinutes {
		offlineMinutes = parsed
	}
	config := userPresenceRuntimeConfig{
		Enabled:          strings.EqualFold(strings.TrimSpace(enabledRaw), "true"),
		Events:           make(map[string]struct{}, len(events)),
		MonitoredUserIDs: monitoredUserIDs,
		MonitoredUsers:   make(map[int]struct{}, len(monitoredUserIDs)),
		RecipientUserIDs: recipientUserIDs,
		OfflineMinutes:   offlineMinutes,
	}
	for _, event := range events {
		config.Events[event] = struct{}{}
	}
	for _, userId := range monitoredUserIDs {
		config.MonitoredUsers[userId] = struct{}{}
	}

	userPresenceRuntimeMu.Lock()
	userPresenceRuntimeSignature = signature
	userPresenceRuntime = config
	userPresenceRuntimeMu.Unlock()
	return config
}

func decodeUserPresenceIDsForRuntime(raw, label string) []int {
	userIds := []int{}
	if strings.TrimSpace(raw) == "" {
		return userIds
	}
	if err := common.UnmarshalJsonStr(raw, &userIds); err != nil {
		common.SysError(fmt.Sprintf("failed to load user presence %s: %v", label, err))
		return []int{}
	}
	return normalizeEmailRecipientIDs(userIds)
}

func decodeUserPresenceEmailEvents(target *[]string) error {
	raw := emailOptionString(common.UserPresenceEmailEventsOptionKey)
	if raw == "" {
		*target = []string{UserPresenceEventOnline, UserPresenceEventOffline}
		return nil
	}
	if err := common.UnmarshalJsonStr(raw, target); err != nil {
		return fmt.Errorf("invalid user presence email events: %w", err)
	}
	*target = normalizeUserPresenceEmailEvents(*target)
	return nil
}

func normalizeUserPresenceEmailEvents(events []string) []string {
	selected := make(map[string]struct{}, len(events))
	for _, event := range events {
		event = strings.ToLower(strings.TrimSpace(event))
		if event == UserPresenceEventOnline || event == UserPresenceEventOffline {
			selected[event] = struct{}{}
		}
	}
	result := make([]string, 0, 2)
	for _, event := range []string{UserPresenceEventOnline, UserPresenceEventOffline} {
		if _, exists := selected[event]; exists {
			result = append(result, event)
		}
	}
	return result
}

func validateUserPresenceMonitoredUserIDs(userIds []int) error {
	if len(userIds) == 0 {
		return nil
	}
	users, err := model.GetUserPresenceMonitorOptionsByIDs(userIds)
	if err != nil {
		return err
	}
	if len(users) != len(userIds) {
		return errors.New("all monitored users must be active users")
	}
	return nil
}

func newlyMonitoredUserPresenceIDs(previous userPresenceRuntimeConfig, current []int) []int {
	result := make([]int, 0, len(current))
	for _, userId := range current {
		if _, exists := previous.MonitoredUsers[userId]; !exists {
			result = append(result, userId)
		}
	}
	return result
}

func clearUserPresenceActivityWriteThrottle(userIds []int) {
	if len(userIds) == 0 {
		return
	}
	userPresenceActivityMu.Lock()
	for _, userId := range userIds {
		delete(userPresenceActivityWrites, userId)
	}
	userPresenceActivityMu.Unlock()
}

func IsUserPresenceEmailEnabled() bool {
	return currentUserPresenceRuntimeConfig().Enabled
}

func RecordUserPresenceActivity(userId int, source, ip, userAgent string) {
	config := currentUserPresenceRuntimeConfig()
	if !config.Enabled || userId <= 0 {
		return
	}
	if _, monitored := config.MonitoredUsers[userId]; !monitored {
		return
	}
	activityAt := time.Now().Unix()
	userPresenceActivityMu.Lock()
	lastWrite := userPresenceActivityWrites[userId]
	if activityAt-lastWrite < int64(userPresenceActivityWriteInterval/time.Second) {
		userPresenceActivityMu.Unlock()
		return
	}
	userPresenceActivityWrites[userId] = activityAt
	userPresenceActivityMu.Unlock()

	source = truncateUserPresenceValue(strings.TrimSpace(source), userPresenceMaximumSourceLength)
	ip = truncateUserPresenceValue(strings.TrimSpace(ip), userPresenceMaximumIPLength)
	userAgent = truncateUserPresenceValue(strings.TrimSpace(userAgent), userPresenceMaximumUserAgentLength)
	presence, transitioned, err := model.RecordUserPresenceActivity(userId, activityAt, source, ip, userAgent)
	if err != nil {
		userPresenceActivityMu.Lock()
		if userPresenceActivityWrites[userId] == activityAt {
			delete(userPresenceActivityWrites, userId)
		}
		userPresenceActivityMu.Unlock()
		common.SysError(fmt.Sprintf("failed to record user presence activity for user %d: %v", userId, err))
		return
	}
	if !transitioned {
		return
	}
	if _, enabled := config.Events[UserPresenceEventOnline]; !enabled {
		return
	}
	gopool.Go(func() {
		if err := notifyUserPresenceOnlineTransition(userId, presence, config, common.SendEmail); err != nil {
			common.SysError(fmt.Sprintf("failed to send user online email for user %d: %v", userId, err))
		}
	})
}

func recordUserPresenceActivityAt(userId int, activityAt int64, source, ip, userAgent string, config userPresenceRuntimeConfig, sender EmailCampaignSender) error {
	presence, transitioned, err := model.RecordUserPresenceActivity(userId, activityAt, source, ip, userAgent)
	if err != nil || !transitioned {
		return err
	}
	if _, enabled := config.Events[UserPresenceEventOnline]; !enabled {
		return nil
	}
	return notifyUserPresenceOnlineTransition(userId, presence, config, sender)
}

func notifyUserPresenceOnlineTransition(userId int, presence model.UserPresence, config userPresenceRuntimeConfig, sender EmailCampaignSender) error {
	users, err := model.ListUserPresenceUsers([]int{userId})
	if err != nil {
		return err
	}
	if len(users) != 1 {
		return errors.New("monitored user is no longer active")
	}
	recipients, err := model.GetOperationalEmailRecipientUsers(config.RecipientUserIDs)
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		return errors.New("no active administrator or root recipient with an email address")
	}
	_, err = sendUserPresenceTransitionEmails(UserPresenceEventOnline, users[0], presence, recipients, sender)
	return err
}

func DispatchUserPresenceEmails(ctx context.Context, sender EmailCampaignSender) (UserPresenceEmailDispatchResult, error) {
	return dispatchUserPresenceEmailsAt(ctx, time.Now(), sender)
}

func dispatchUserPresenceEmailsAt(ctx context.Context, now time.Time, sender EmailCampaignSender) (UserPresenceEmailDispatchResult, error) {
	config := currentUserPresenceRuntimeConfig()
	if !config.Enabled {
		return UserPresenceEmailDispatchResult{}, nil
	}
	presences, err := model.MarkTimedOutUserPresencesOffline(
		config.MonitoredUserIDs,
		now.Add(-time.Duration(config.OfflineMinutes)*time.Minute).Unix(),
		now.Unix(),
	)
	if err != nil {
		return UserPresenceEmailDispatchResult{}, err
	}
	result := UserPresenceEmailDispatchResult{OfflineUserCount: len(presences)}
	if len(presences) == 0 {
		return result, nil
	}
	if _, enabled := config.Events[UserPresenceEventOffline]; !enabled {
		return result, nil
	}
	if sender == nil {
		return result, errors.New("email sender is required")
	}
	recipients, err := model.GetOperationalEmailRecipientUsers(config.RecipientUserIDs)
	if err != nil {
		return result, err
	}
	if len(recipients) == 0 {
		return result, errors.New("no active administrator or root recipient with an email address")
	}
	userIds := make([]int, 0, len(presences))
	for _, presence := range presences {
		userIds = append(userIds, presence.UserId)
	}
	users, err := model.ListUserPresenceUsers(userIds)
	if err != nil {
		return result, err
	}
	usersByID := make(map[int]model.User, len(users))
	for _, user := range users {
		usersByID[user.Id] = user
	}
	for _, presence := range presences {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		user, exists := usersByID[presence.UserId]
		if !exists {
			continue
		}
		sent, err := sendUserPresenceTransitionEmails(UserPresenceEventOffline, user, presence, recipients, sender)
		result.RecipientCount += sent
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func SendUserPresenceTestEmails(recipientUserIds, monitoredUserIds []int, events []string, offlineMinutes int) (UserPresenceTestEmailResult, error) {
	return sendUserPresenceTestEmails(recipientUserIds, monitoredUserIds, events, offlineMinutes, common.SendEmail)
}

func sendUserPresenceTestEmails(recipientUserIds, monitoredUserIds []int, events []string, offlineMinutes int, sender EmailCampaignSender) (UserPresenceTestEmailResult, error) {
	if sender == nil {
		return UserPresenceTestEmailResult{}, errors.New("email sender is required")
	}
	if offlineMinutes == 0 {
		offlineMinutes = defaultUserPresenceOfflineMinutes
	}
	if offlineMinutes < minUserPresenceOfflineMinutes || offlineMinutes > maxUserPresenceOfflineMinutes {
		return UserPresenceTestEmailResult{}, fmt.Errorf("user presence offline minutes must be between %d and %d", minUserPresenceOfflineMinutes, maxUserPresenceOfflineMinutes)
	}
	events = normalizeUserPresenceEmailEvents(events)
	if len(events) == 0 {
		return UserPresenceTestEmailResult{}, errors.New("at least one user presence event is required")
	}
	monitoredUserIds = normalizeEmailRecipientIDs(monitoredUserIds)
	if len(monitoredUserIds) == 0 {
		return UserPresenceTestEmailResult{}, errors.New("at least one monitored user is required")
	}
	if len(monitoredUserIds) > maxUserPresenceMonitoredUsers {
		return UserPresenceTestEmailResult{}, fmt.Errorf("monitored user count cannot exceed %d", maxUserPresenceMonitoredUsers)
	}
	if err := validateUserPresenceMonitoredUserIDs(monitoredUserIds); err != nil {
		return UserPresenceTestEmailResult{}, err
	}
	recipientUserIds = normalizeEmailRecipientIDs(recipientUserIds)
	if len(recipientUserIds) > maxOperationalEmailRecipients {
		return UserPresenceTestEmailResult{}, fmt.Errorf("recipient count cannot exceed %d", maxOperationalEmailRecipients)
	}
	if err := validateOperationalEmailRecipientIDs(recipientUserIds); err != nil {
		return UserPresenceTestEmailResult{}, err
	}
	recipients, err := model.GetOperationalEmailRecipientUsers(recipientUserIds)
	if err != nil {
		return UserPresenceTestEmailResult{}, err
	}
	if len(recipients) == 0 {
		return UserPresenceTestEmailResult{}, errors.New("no active administrator or root recipient with an email address")
	}
	users, err := model.ListUserPresenceUsers([]int{monitoredUserIds[0]})
	if err != nil {
		return UserPresenceTestEmailResult{}, err
	}
	if len(users) != 1 {
		return UserPresenceTestEmailResult{}, errors.New("monitored user is no longer active")
	}
	now := time.Now()
	result := UserPresenceTestEmailResult{
		RecipientCount:  len(recipients),
		MonitoredUserId: users[0].Id,
		Events:          events,
	}
	for _, event := range events {
		presence := model.UserPresence{
			UserId:         users[0].Id,
			IsOnline:       event == UserPresenceEventOnline,
			LastActivityAt: now.Unix(),
			LastSource:     UserPresenceActivitySourceAPI,
			LastIP:         "127.0.0.1",
			LastUserAgent:  "Email settings SMTP test",
			LastChangedAt:  now.Unix(),
		}
		if event == UserPresenceEventOffline {
			presence.LastActivityAt = now.Add(-time.Duration(offlineMinutes) * time.Minute).Unix()
		}
		sent, err := sendUserPresenceTransitionEmails(event, users[0], presence, recipients, sender)
		result.EmailCount += sent
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func sendUserPresenceTransitionEmails(event string, monitoredUser model.User, presence model.UserPresence, recipients []model.User, sender EmailCampaignSender) (int, error) {
	if sender == nil {
		return 0, errors.New("email sender is required")
	}
	sent := 0
	for _, recipient := range recipients {
		locale := NormalizeEmailTemplateLocale(recipient.GetSetting().Language)
		values := userPresenceTemplateValues(event, monitoredUser, presence, locale)
		rendered, err := renderOperationalTemplateEmail(EmailTemplateEventUserPresenceChanged, recipient, values)
		if err != nil {
			return sent, err
		}
		if err := sender(rendered.Subject, recipient.Email, rendered.Content); err != nil {
			return sent, fmt.Errorf("failed to send user presence email to user %d: %w", recipient.Id, err)
		}
		sent++
	}
	return sent, nil
}

func userPresenceTemplateValues(event string, monitoredUser model.User, presence model.UserPresence, locale string) map[string]string {
	displayName := strings.TrimSpace(monitoredUser.DisplayName)
	if displayName == "" {
		displayName = monitoredUser.Username
	}
	status := "在线"
	source := userPresenceSourceLabel(presence.LastSource, locale)
	offlineAt := "-"
	inactivityMinutes := "0"
	if event == UserPresenceEventOffline {
		status = "离线"
		offlineAt = time.Unix(presence.LastChangedAt, 0).Format("2006-01-02 15:04:05")
		minutes := (presence.LastChangedAt - presence.LastActivityAt) / 60
		if minutes < 0 {
			minutes = 0
		}
		inactivityMinutes = strconv.FormatInt(minutes, 10)
	}
	if locale == EmailTemplateLocaleEnglish {
		status = "Online"
		if event == UserPresenceEventOffline {
			status = "Offline"
		}
	}
	return map[string]string{
		"monitored_user_id":      strconv.Itoa(monitoredUser.Id),
		"monitored_username":     monitoredUser.Username,
		"monitored_display_name": displayName,
		"monitored_email":        monitoredUser.Email,
		"presence_status":        status,
		"activity_source":        source,
		"activity_ip":            presence.LastIP,
		"user_agent":             presence.LastUserAgent,
		"activity_at":            time.Unix(presence.LastActivityAt, 0).Format("2006-01-02 15:04:05"),
		"offline_at":             offlineAt,
		"inactivity_minutes":     inactivityMinutes,
	}
}

func userPresenceSourceLabel(source, locale string) string {
	if locale == EmailTemplateLocaleEnglish {
		switch source {
		case UserPresenceActivitySourceLogin:
			return "Login"
		case UserPresenceActivitySourceAPI:
			return "API call"
		default:
			return "Dashboard activity"
		}
	}
	switch source {
	case UserPresenceActivitySourceLogin:
		return "登录"
	case UserPresenceActivitySourceAPI:
		return "API 调用"
	default:
		return "后台操作"
	}
}

func truncateUserPresenceValue(value string, maxLength int) string {
	runes := []rune(value)
	if len(runes) <= maxLength {
		return value
	}
	return string(runes[:maxLength])
}
