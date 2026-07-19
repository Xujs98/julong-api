package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

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
