package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"time"
)

// Authentication middleware records the caller's activity. No state is exposed.
func UserActivityHeartbeat(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
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
