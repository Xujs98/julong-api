package controller

import (
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// Legacy GET heartbeats are inert, so cached clients cannot keep idle users online.
func UserActivityHeartbeat(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if c.Request.Method == http.MethodPost && !c.GetBool("use_access_token") && c.GetString("session_id") != "" {
		service.RecordUserPresenceActivity(c.GetInt("id"), service.UserPresenceActivitySourceDashboard, c.ClientIP(), c.Request.UserAgent())
	}
	common.ApiSuccess(c, nil)
}

func GetUserActivitySummary(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	result, err := model.GetUserActivitySummary(time.Now())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}
