package console_setting

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	AnnouncementStatusDraft    = "draft"
	AnnouncementStatusActive   = "active"
	AnnouncementStatusArchived = "archived"
	AnnouncementNotifySilent   = "silent"
	AnnouncementNotifyPopup    = "popup"
	AnnouncementAudienceAll    = "all"
	AnnouncementAudienceRules  = "conditions"
)

type AnnouncementCondition struct {
	Type     string  `json:"type"`
	Operator string  `json:"operator"`
	PlanIds  []int   `json:"planIds,omitempty"`
	Value    float64 `json:"value,omitempty"`
}

type AnnouncementConditionGroup struct {
	Conditions []AnnouncementCondition `json:"conditions"`
}

type Announcement struct {
	Id               int                          `json:"id"`
	Title            string                       `json:"title"`
	Content          string                       `json:"content"`
	Status           string                       `json:"status"`
	NotificationMode string                       `json:"notificationMode"`
	StartTime        string                       `json:"startTime,omitempty"`
	EndTime          string                       `json:"endTime,omitempty"`
	AudienceMode     string                       `json:"audienceMode"`
	ConditionGroups  []AnnouncementConditionGroup `json:"conditionGroups,omitempty"`
	PublishDate      string                       `json:"publishDate,omitempty"`
}

type AnnouncementAudience struct {
	Balance float64
	PlanIds map[int]bool
}

func normalizeAnnouncement(announcement *Announcement) {
	announcement.Title = strings.TrimSpace(announcement.Title)
	if announcement.Status == "" {
		announcement.Status = AnnouncementStatusActive
	}
	if announcement.NotificationMode == "" {
		announcement.NotificationMode = AnnouncementNotifySilent
	}
	if announcement.AudienceMode == "" {
		announcement.AudienceMode = AnnouncementAudienceAll
	}
	if announcement.StartTime == "" {
		announcement.StartTime = announcement.PublishDate
	}
	if announcement.Title == "" {
		announcement.Title = "系统公告"
	}
}

func ParseAnnouncements(raw string) ([]Announcement, error) {
	if strings.TrimSpace(raw) == "" {
		return []Announcement{}, nil
	}
	var announcements []Announcement
	if err := json.Unmarshal([]byte(raw), &announcements); err != nil {
		return nil, fmt.Errorf("系统公告格式错误：%s", err.Error())
	}
	for index := range announcements {
		normalizeAnnouncement(&announcements[index])
	}
	return announcements, nil
}

func parseOptionalAnnouncementTime(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}

func ValidateAnnouncements(raw string) error {
	announcements, err := ParseAnnouncements(raw)
	if err != nil {
		return err
	}
	if len(announcements) > 100 {
		return fmt.Errorf("系统公告数量不能超过100个")
	}
	validStatuses := map[string]bool{AnnouncementStatusDraft: true, AnnouncementStatusActive: true, AnnouncementStatusArchived: true}
	validNotifications := map[string]bool{AnnouncementNotifySilent: true, AnnouncementNotifyPopup: true}
	for index, announcement := range announcements {
		if announcement.Title == "" || len([]rune(announcement.Title)) > 100 {
			return fmt.Errorf("第%d个公告标题无效", index+1)
		}
		if strings.TrimSpace(announcement.Content) == "" || len([]rune(announcement.Content)) > 5000 {
			return fmt.Errorf("第%d个公告内容无效", index+1)
		}
		if !validStatuses[announcement.Status] || !validNotifications[announcement.NotificationMode] {
			return fmt.Errorf("第%d个公告状态或通知方式无效", index+1)
		}
		start, startErr := parseOptionalAnnouncementTime(announcement.StartTime)
		end, endErr := parseOptionalAnnouncementTime(announcement.EndTime)
		if startErr != nil || endErr != nil || (!start.IsZero() && !end.IsZero() && !end.After(start)) {
			return fmt.Errorf("第%d个公告有效时间无效", index+1)
		}
		if announcement.AudienceMode != AnnouncementAudienceAll && announcement.AudienceMode != AnnouncementAudienceRules {
			return fmt.Errorf("第%d个公告展示条件无效", index+1)
		}
		if announcement.AudienceMode == AnnouncementAudienceRules {
			if len(announcement.ConditionGroups) == 0 || len(announcement.ConditionGroups) > 50 {
				return fmt.Errorf("第%d个公告条件组数量无效", index+1)
			}
			for _, group := range announcement.ConditionGroups {
				if len(group.Conditions) == 0 || len(group.Conditions) > 50 {
					return fmt.Errorf("第%d个公告 AND 条件数量无效", index+1)
				}
				for _, condition := range group.Conditions {
					if condition.Type == "subscription_plan" {
						if len(condition.PlanIds) == 0 || (condition.Operator != "in" && condition.Operator != "not_in") {
							return fmt.Errorf("第%d个公告订阅套餐条件无效", index+1)
						}
					} else if condition.Type == "balance" {
						validOperator := map[string]bool{"gte": true, "lte": true, "gt": true, "lt": true, "eq": true}
						if !validOperator[condition.Operator] || condition.Value < 0 {
							return fmt.Errorf("第%d个公告余额条件无效", index+1)
						}
					} else {
						return fmt.Errorf("第%d个公告条件类型无效", index+1)
					}
				}
			}
		}
	}
	return nil
}

func announcementIsActive(announcement Announcement, now time.Time) bool {
	if announcement.Status != AnnouncementStatusActive {
		return false
	}
	start, _ := parseOptionalAnnouncementTime(announcement.StartTime)
	end, _ := parseOptionalAnnouncementTime(announcement.EndTime)
	return (start.IsZero() || !now.Before(start)) && (end.IsZero() || now.Before(end))
}

func conditionMatches(condition AnnouncementCondition, audience AnnouncementAudience) bool {
	if condition.Type == "subscription_plan" {
		matched := false
		for _, planId := range condition.PlanIds {
			if audience.PlanIds[planId] {
				matched = true
				break
			}
		}
		if condition.Operator == "not_in" {
			return !matched
		}
		return matched
	}
	value := audience.Balance
	switch condition.Operator {
	case "gte":
		return value >= condition.Value
	case "lte":
		return value <= condition.Value
	case "gt":
		return value > condition.Value
	case "lt":
		return value < condition.Value
	case "eq":
		return value == condition.Value
	default:
		return false
	}
}

func announcementMatches(announcement Announcement, audience AnnouncementAudience) bool {
	if announcement.AudienceMode == AnnouncementAudienceAll {
		return true
	}
	for _, group := range announcement.ConditionGroups {
		matched := len(group.Conditions) > 0
		for _, condition := range group.Conditions {
			if !conditionMatches(condition, audience) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func GetActiveAnnouncements(audience *AnnouncementAudience) []Announcement {
	announcements, _ := ParseAnnouncements(GetConsoleSetting().Announcements)
	result := make([]Announcement, 0, len(announcements))
	now := time.Now()
	for _, announcement := range announcements {
		if !announcementIsActive(announcement, now) {
			continue
		}
		if audience == nil {
			if announcement.AudienceMode != AnnouncementAudienceAll {
				continue
			}
		} else if !announcementMatches(announcement, *audience) {
			continue
		}
		result = append(result, announcement)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].StartTime > result[j].StartTime })
	return result
}
