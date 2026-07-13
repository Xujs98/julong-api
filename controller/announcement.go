package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/console_setting"
	"github.com/gin-gonic/gin"
)

func GetUserAnnouncements(c *gin.Context) {
	if !console_setting.GetConsoleSetting().AnnouncementsEnabled {
		common.ApiSuccess(c, []console_setting.Announcement{})
		return
	}
	userId := c.GetInt("id")
	user, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	subscriptions, err := model.GetAllActiveUserSubscriptions(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	planIds := make(map[int]bool, len(subscriptions))
	for _, summary := range subscriptions {
		if summary.Subscription != nil {
			planIds[summary.Subscription.PlanId] = true
		}
	}
	balance := float64(user.Quota)
	if common.QuotaPerUnit > 0 {
		balance /= float64(common.QuotaPerUnit)
	}
	common.ApiSuccess(c, console_setting.GetActiveAnnouncements(&console_setting.AnnouncementAudience{
		Balance: balance,
		PlanIds: planIds,
	}))
}
