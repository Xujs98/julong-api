package controller

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func GetImageGenerationLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	startTime, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTime, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	channelId, _ := strconv.Atoi(c.Query("channel_id"))
	role := c.GetInt("role")
	logs, total, err := model.GetImageGenerationLogs(
		c.GetInt("id"),
		role >= common.RoleAdminUser,
		pageInfo.GetStartIdx(),
		pageInfo.GetPageSize(),
		channelId,
		strings.TrimSpace(c.Query("model")),
		strings.TrimSpace(c.Query("prompt")),
		startTime,
		endTime,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	for _, log := range logs {
		refs, refErr := log.ImageRefs()
		if refErr != nil {
			continue
		}
		log.ImageUrls = make([]string, len(refs))
		for index, ref := range refs {
			if ref.Type == "remote" {
				log.ImageUrls[index] = ref.Value
			} else {
				log.ImageUrls[index] = "/api/image-generation-logs/" + strconv.Itoa(log.Id) + "/images/" + strconv.Itoa(index)
			}
		}
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
}

func GetImageGenerationLogImage(c *gin.Context) {
	id, idErr := strconv.Atoi(c.Param("id"))
	index, indexErr := strconv.Atoi(c.Param("index"))
	if idErr != nil || indexErr != nil || id <= 0 || index < 0 {
		common.ApiErrorMsg(c, "无效的图片日志参数")
		return
	}
	log, err := model.GetImageGenerationLogById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if c.GetInt("role") < common.RoleAdminUser && log.UserId != c.GetInt("id") {
		c.Status(http.StatusForbidden)
		return
	}
	refs, err := log.ImageRefs()
	if err != nil || index >= len(refs) {
		c.Status(http.StatusNotFound)
		return
	}
	ref := refs[index]
	c.Header("Cache-Control", "private, max-age=3600")
	if ref.Type == "local" {
		if filepath.Base(ref.Value) != ref.Value {
			c.Status(http.StatusBadRequest)
			return
		}
		data, readErr := os.ReadFile(filepath.Join(service.ImageGenerationLogStorageDir(), ref.Value))
		if readErr != nil {
			c.Status(http.StatusNotFound)
			return
		}
		mimeType := ref.MimeType
		if mimeType == "" {
			mimeType = http.DetectContentType(data)
		}
		c.Data(http.StatusOK, mimeType, data)
		return
	}
	c.Status(http.StatusNotFound)
}
