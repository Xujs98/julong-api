package console_setting

import (
	"encoding/json"
	"testing"
	"time"
)

func TestValidateAndMatchAnnouncements(t *testing.T) {
	now := time.Now()
	announcements := []Announcement{
		{Id: 1, Title: "draft", Content: "hidden", Status: AnnouncementStatusDraft, NotificationMode: AnnouncementNotifySilent, AudienceMode: AnnouncementAudienceAll},
		{Id: 2, Title: "all", Content: "visible", Status: AnnouncementStatusActive, NotificationMode: AnnouncementNotifyPopup, StartTime: now.Add(-time.Hour).Format(time.RFC3339), EndTime: now.Add(time.Hour).Format(time.RFC3339), AudienceMode: AnnouncementAudienceAll},
		{Id: 3, Title: "plan", Content: "subscriber", Status: AnnouncementStatusActive, NotificationMode: AnnouncementNotifySilent, AudienceMode: AnnouncementAudienceRules, ConditionGroups: []AnnouncementConditionGroup{{Conditions: []AnnouncementCondition{{Type: "subscription_plan", Operator: "in", PlanIds: []int{9}}, {Type: "balance", Operator: "gte", Value: 100}}}}},
	}
	raw, err := json.Marshal(announcements)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateAnnouncements(string(raw)); err != nil {
		t.Fatal(err)
	}
	previous := GetConsoleSetting().Announcements
	GetConsoleSetting().Announcements = string(raw)
	t.Cleanup(func() { GetConsoleSetting().Announcements = previous })

	public := GetActiveAnnouncements(nil)
	if len(public) != 1 || public[0].Id != 2 {
		t.Fatalf("unexpected public announcements: %+v", public)
	}
	matched := GetActiveAnnouncements(&AnnouncementAudience{Balance: 100, PlanIds: map[int]bool{9: true}})
	if len(matched) != 2 {
		t.Fatalf("expected two matched announcements, got %+v", matched)
	}
	notMatched := GetActiveAnnouncements(&AnnouncementAudience{Balance: 99, PlanIds: map[int]bool{9: true}})
	if len(notMatched) != 1 || notMatched[0].Id != 2 {
		t.Fatalf("unexpected unmatched result: %+v", notMatched)
	}
}
