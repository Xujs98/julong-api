package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetSiteBlockedIPs(c *gin.Context) {
	ip := strings.TrimSpace(c.Query("ip"))
	if ip != "" {
		normalized, err := model.NormalizeIPAddress(ip)
		if err != nil {
			common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Invalid parameters"))
			return
		}
		ip = normalized
	}
	page := common.GetPageQuery(c)
	records, total, err := model.ListBlockedIPs(ip, page.GetStartIdx(), page.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	page.SetTotal(int(total))
	page.SetItems(records)
	common.ApiSuccess(c, page)
}

func UpdateSiteBlockedIPs(c *gin.Context) {
	var request UpdateUserLoginIPsRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || len(request.IPs) == 0 || len(request.IPs) > 100 {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Invalid parameters"))
		return
	}
	ips, err := model.NormalizeIPAddresses(request.IPs)
	if err != nil {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Invalid parameters"))
		return
	}
	if request.Blocked {
		// Prevent accidental lockout of the operator. Other blocked addresses have no bypass.
		for _, ip := range ips {
			if ip == c.ClientIP() {
				common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Cannot block your current IP address"))
				return
			}
		}
		err = model.BlockIPAddresses(ips, 0, c.GetInt("id"), "manually blocked")
	} else {
		err = model.UnblockIPAddresses(ips)
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, c.GetInt("id"), "site.ip.update", map[string]any{"ips": ips, "blocked": request.Blocked})
	common.ApiSuccess(c, nil)
}
