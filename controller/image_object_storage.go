package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type imageStorageCleanupTaskPayload struct {
	Manual    bool `json:"manual,omitempty"`
	DeleteAll bool `json:"delete_all,omitempty"`
}

func GetImageObjectStorageConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": service.PublicImageObjectStorageConfig()})
}

func SaveImageObjectStorageConfig(c *gin.Context) {
	var config service.ImageObjectStorageConfig
	if err := common.DecodeJson(c.Request.Body, &config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的 MinIO 配置"})
		return
	}
	if err := service.SaveImageObjectStorageConfig(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "MinIO 配置已保存", "data": service.PublicImageObjectStorageConfig()})
}

func TestImageObjectStorageConfig(c *gin.Context) {
	var config service.ImageObjectStorageConfig
	if err := common.DecodeJson(c.Request.Body, &config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的 MinIO 配置"})
		return
	}
	if err := service.TestImageObjectStorage(c.Request.Context(), config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "MinIO 连接成功"})
}

func GetImageObjectStorageStats(c *gin.Context) {
	stats, err := service.GetImageObjectStorageStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

func StartImageObjectStorageCleanup(c *gin.Context) {
	config := service.GetImageObjectStorageConfig()
	if !config.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请先启用 MinIO 生图存储"})
		return
	}
	if config.RetentionDays <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "MinIO 图片当前配置为永久保留"})
		return
	}
	task, created, err := service.EnqueueSystemTask(model.SystemTaskTypeImageStorageCleanup, imageStorageCleanupTaskPayload{Manual: true})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	message := "MinIO 图片清理任务已创建"
	if !created {
		message = "MinIO 图片清理任务正在运行"
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
		"data":    task.ToResponse(),
	})
}

func StartImageObjectStoragePurge(c *gin.Context) {
	config := service.GetImageObjectStorageConfig()
	if !config.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请先启用 MinIO 生图存储"})
		return
	}
	activeTask, err := model.GetActiveSystemTask(model.SystemTaskTypeImageStorageCleanup)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if activeTask != nil {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "已有 MinIO 图片清理任务正在运行，请稍后再试",
			"data":    activeTask.ToResponse(),
		})
		return
	}
	task, created, err := service.EnqueueSystemTask(
		model.SystemTaskTypeImageStorageCleanup,
		imageStorageCleanupTaskPayload{Manual: true, DeleteAll: true},
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !created {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "已有 MinIO 图片清理任务正在运行，请稍后再试",
			"data":    task.ToResponse(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "MinIO 图片全量清空任务已创建",
		"data":    task.ToResponse(),
	})
}
