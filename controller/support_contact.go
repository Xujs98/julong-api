package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type SupportContact struct {
	Id    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`
	Value string `json:"value"`
}

type updateSupportContactsRequest struct {
	Contacts []SupportContact `json:"contacts"`
}

func GetSupportContacts(c *gin.Context) {
	contacts := make([]SupportContact, 0)
	_ = json.Unmarshal([]byte(common.SupportContacts), &contacts)
	for index := range contacts {
		if contacts[index].Id == "" {
			contacts[index].Id = fmt.Sprintf("contact-%d", index+1)
		}
	}
	common.ApiSuccess(c, contacts)
}

func UpdateSupportContacts(c *gin.Context) {
	var request updateSupportContactsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "客服联系方式格式错误")
		return
	}
	if len(request.Contacts) > 30 {
		common.ApiErrorMsg(c, "客服联系方式最多 30 条")
		return
	}
	allowedTypes := map[string]bool{"qq": true, "wechat": true, "phone": true}
	for index := range request.Contacts {
		contact := &request.Contacts[index]
		contact.Id = strings.TrimSpace(contact.Id)
		if contact.Id == "" {
			contact.Id = fmt.Sprintf("contact-%d", index+1)
		}
		contact.Type = strings.ToLower(strings.TrimSpace(contact.Type))
		contact.Label = strings.TrimSpace(contact.Label)
		contact.Value = strings.TrimSpace(contact.Value)
		if !allowedTypes[contact.Type] || contact.Value == "" || len([]rune(contact.Id)) > 64 || len([]rune(contact.Value)) > 128 || len([]rune(contact.Label)) > 64 {
			common.ApiErrorMsg(c, "客服联系方式内容无效")
			return
		}
	}
	encoded, err := json.Marshal(request.Contacts)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.UpdateOption("SupportContacts", string(encoded)); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": request.Contacts})
}
