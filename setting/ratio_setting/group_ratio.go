package ratio_setting

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/types"
)

var defaultGroupRatio = map[string]float64{
	"default": 1,
	"vip":     1,
	"svip":    1,
}

var groupRatioMap = types.NewRWMap[string, float64]()

var defaultGroupGroupRatio = map[string]map[string]float64{
	"vip": {
		"edit_this": 0.9,
	},
}

var groupGroupRatioMap = types.NewRWMap[string, map[string]float64]()

var defaultGroupSpecialUsableGroup = map[string]map[string]string{}

type modelTokenRatioUserRules map[string]map[string]types.ModelTokenAdjustment

var defaultModelTokenRatio = map[string]modelTokenRatioUserRules{}

var modelTokenRatioMap = types.NewRWMap[string, modelTokenRatioUserRules]()

type GroupRatioSetting struct {
	GroupRatio              *types.RWMap[string, float64]                  `json:"group_ratio"`
	GroupGroupRatio         *types.RWMap[string, map[string]float64]       `json:"group_group_ratio"`
	GroupSpecialUsableGroup *types.RWMap[string, map[string]string]        `json:"group_special_usable_group"`
	ModelTokenRatio         *types.RWMap[string, modelTokenRatioUserRules] `json:"model_token_ratio"`
}

var groupRatioSetting GroupRatioSetting

func init() {
	groupSpecialUsableGroup := types.NewRWMap[string, map[string]string]()
	groupSpecialUsableGroup.AddAll(defaultGroupSpecialUsableGroup)
	modelTokenRatioMap.AddAll(defaultModelTokenRatio)

	groupRatioMap.AddAll(defaultGroupRatio)
	groupGroupRatioMap.AddAll(defaultGroupGroupRatio)

	groupRatioSetting = GroupRatioSetting{
		GroupSpecialUsableGroup: groupSpecialUsableGroup,
		GroupRatio:              groupRatioMap,
		GroupGroupRatio:         groupGroupRatioMap,
		ModelTokenRatio:         modelTokenRatioMap,
	}

	config.GlobalConfig.Register("group_ratio_setting", &groupRatioSetting)
}

func GetGroupRatioSetting() *GroupRatioSetting {
	if groupRatioSetting.GroupSpecialUsableGroup == nil {
		groupRatioSetting.GroupSpecialUsableGroup = types.NewRWMap[string, map[string]string]()
		groupRatioSetting.GroupSpecialUsableGroup.AddAll(defaultGroupSpecialUsableGroup)
	}
	if groupRatioSetting.ModelTokenRatio == nil {
		groupRatioSetting.ModelTokenRatio = types.NewRWMap[string, modelTokenRatioUserRules]()
		groupRatioSetting.ModelTokenRatio.AddAll(defaultModelTokenRatio)
	}
	return &groupRatioSetting
}

func GetGroupRatioCopy() map[string]float64 {
	return groupRatioMap.ReadAll()
}

func ContainsGroupRatio(name string) bool {
	_, ok := groupRatioMap.Get(name)
	return ok
}

func GroupRatio2JSONString() string {
	return groupRatioMap.MarshalJSONString()
}

func UpdateGroupRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(groupRatioMap, jsonStr)
}

func GetGroupRatio(name string) float64 {
	ratio, ok := groupRatioMap.Get(name)
	if !ok {
		common.SysLog("group ratio not found: " + name)
		return 1
	}
	return ratio
}

func GetGroupGroupRatio(userGroup, usingGroup string) (float64, bool) {
	gp, ok := groupGroupRatioMap.Get(userGroup)
	if !ok {
		return -1, false
	}
	ratio, ok := gp[usingGroup]
	if !ok {
		return -1, false
	}
	return ratio, true
}

func GroupGroupRatio2JSONString() string {
	return groupGroupRatioMap.MarshalJSONString()
}

func UpdateGroupGroupRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(groupGroupRatioMap, jsonStr)
}

func GetModelTokenAdjustment(userGroup, usingGroup, modelName string) (types.ModelTokenAdjustment, bool) {
	setting := GetGroupRatioSetting()
	userRules, ok := setting.ModelTokenRatio.Get(userGroup)
	if !ok {
		return types.ModelTokenAdjustment{}, false
	}
	modelRules, ok := userRules[usingGroup]
	if !ok {
		return types.ModelTokenAdjustment{}, false
	}
	adjustment, ok := modelRules[modelName]
	if !ok {
		adjustment, ok = modelRules[FormatMatchingModelName(modelName)]
	}
	if !ok || validateModelTokenAdjustment(adjustment) != nil {
		return types.ModelTokenAdjustment{}, false
	}
	return adjustment, true
}

func ModelTokenRatio2JSONString() string {
	return GetGroupRatioSetting().ModelTokenRatio.MarshalJSONString()
}

func UpdateModelTokenRatioByJSONString(jsonStr string) error {
	if err := CheckModelTokenRatio(jsonStr); err != nil {
		return err
	}
	return types.LoadFromJsonString(GetGroupRatioSetting().ModelTokenRatio, jsonStr)
}

func CheckGroupRatio(jsonStr string) error {
	checkGroupRatio := make(map[string]float64)
	err := common.UnmarshalJsonStr(jsonStr, &checkGroupRatio)
	if err != nil {
		return err
	}
	for name, ratio := range checkGroupRatio {
		if ratio < 0 {
			return errors.New("group ratio must be not less than 0: " + name)
		}
	}
	return nil
}

func CheckModelTokenRatio(jsonStr string) error {
	rules := make(map[string]map[string]map[string]types.ModelTokenAdjustment)
	if err := common.UnmarshalJsonStr(jsonStr, &rules); err != nil {
		return err
	}
	for userGroup, billingGroups := range rules {
		trimmedUserGroup := strings.TrimSpace(userGroup)
		if trimmedUserGroup == "" || trimmedUserGroup != userGroup {
			return fmt.Errorf("user group must not be empty or contain surrounding whitespace: %q", userGroup)
		}
		for billingGroup, modelRules := range billingGroups {
			trimmedBillingGroup := strings.TrimSpace(billingGroup)
			if trimmedBillingGroup == "" || trimmedBillingGroup != billingGroup {
				return fmt.Errorf("billing group must not be empty or contain surrounding whitespace: %q", billingGroup)
			}
			for modelName, adjustment := range modelRules {
				trimmedModelName := strings.TrimSpace(modelName)
				if trimmedModelName == "" || trimmedModelName != modelName {
					return fmt.Errorf("model name must not be empty or contain surrounding whitespace: %q", modelName)
				}
				if err := validateModelTokenAdjustment(adjustment); err != nil {
					return fmt.Errorf("invalid model token adjustment for user group %s, billing group %s, model %s: %w", userGroup, billingGroup, modelName, err)
				}
			}
		}
	}
	return nil
}

func validateModelTokenAdjustment(adjustment types.ModelTokenAdjustment) error {
	if !adjustment.HasAny() {
		return errors.New("at least one token adjustment must be configured")
	}
	values := []struct {
		name  string
		value *float64
	}{
		{name: "input", value: adjustment.Input},
		{name: "output", value: adjustment.Output},
		{name: "cache_read", value: adjustment.CacheRead},
		{name: "cache_creation", value: adjustment.CacheCreation},
	}
	for _, item := range values {
		if item.value == nil {
			continue
		}
		if math.IsNaN(*item.value) || math.IsInf(*item.value, 0) || *item.value < 0 || *item.value > types.MaxModelTokenAdjustment {
			return fmt.Errorf("%s must be between 0 and %d", item.name, types.MaxModelTokenAdjustment)
		}
	}
	return nil
}
