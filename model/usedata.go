package model

import (
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// QuotaData 柱状图数据
type QuotaData struct {
	Id                   int    `json:"id"`
	UserID               int    `json:"user_id" gorm:"index"`
	Username             string `json:"username" gorm:"index:idx_qdt_model_user_name,priority:2;size:64;default:''"`
	ModelName            string `json:"model_name" gorm:"index:idx_qdt_model_user_name,priority:1;size:64;default:''"`
	CreatedAt            int64  `json:"created_at" gorm:"bigint;index:idx_qdt_created_at,priority:2"`
	UseGroup             string `json:"use_group" gorm:"index;size:64;default:''"`
	TokenID              int    `json:"token_id" gorm:"index;default:0"`
	ChannelID            int    `json:"channel_id" gorm:"index;default:0"`
	NodeName             string `json:"node_name" gorm:"index;size:64;default:''"`
	TokenUsed            int    `json:"token_used" gorm:"default:0"`
	Count                int    `json:"count" gorm:"default:0"`
	Quota                int    `json:"quota" gorm:"default:0"`
	UserDisplayTokenUsed *int   `json:"-"`
	UserDisplayQuota     *int   `json:"-"`
}

type QuotaDataLogParams struct {
	UserID               int
	Username             string
	ModelName            string
	Quota                int
	CreatedAt            int64
	TokenUsed            int
	UseGroup             string
	TokenID              int
	ChannelID            int
	NodeName             string
	UserDisplayQuota     *int
	UserDisplayTokenUsed *int
}

type GroupQuotaData struct {
	UseGroup  string `json:"group" gorm:"column:use_group"`
	Quota     int64  `json:"quota"`
	Count     int64  `json:"count"`
	TokenUsed int64  `json:"token_used"`
	UserCount int64  `json:"user_count"`
}

type GroupQuotaDataTotals struct {
	Quota      int64 `json:"quota"`
	Count      int64 `json:"count"`
	TokenUsed  int64 `json:"token_used"`
	UserCount  int64 `json:"user_count"`
	GroupCount int   `json:"group_count"`
}

type GroupQuotaDataAnalytics struct {
	Items  []GroupQuotaData     `json:"items"`
	Totals GroupQuotaDataTotals `json:"totals"`
}

type DashboardReportModel struct {
	ModelName string `json:"model_name" gorm:"column:model_name"`
	Quota     int64  `json:"quota"`
	Count     int64  `json:"count"`
	TokenUsed int64  `json:"token_used"`
}

type DashboardReportData struct {
	Quota        int64                  `json:"quota"`
	Count        int64                  `json:"count"`
	TokenUsed    int64                  `json:"token_used"`
	UserCount    int64                  `json:"user_count"`
	ModelCount   int64                  `json:"model_count"`
	ChannelCount int64                  `json:"channel_count"`
	GroupCount   int64                  `json:"group_count"`
	TopModels    []DashboardReportModel `json:"top_models" gorm:"-"`
}

func UpdateQuotaData() {
	for {
		if common.DataExportEnabled {
			common.SysLog("正在更新数据看板数据...")
			SaveQuotaDataCache()
		}
		time.Sleep(time.Duration(common.DataExportInterval) * time.Minute)
	}
}

var CacheQuotaData = make(map[string]*QuotaData)
var CacheQuotaDataLock = sync.Mutex{}

func logQuotaDataCache(quotaData *QuotaData) {
	key := fmt.Sprintf("%d\x00%s\x00%s\x00%d\x00%s\x00%d\x00%d\x00%s",
		quotaData.UserID,
		quotaData.Username,
		quotaData.ModelName,
		quotaData.CreatedAt,
		quotaData.UseGroup,
		quotaData.TokenID,
		quotaData.ChannelID,
		quotaData.NodeName,
	)
	if quotaData.UserDisplayQuota == nil {
		quotaData.UserDisplayQuota = common.GetPointer(quotaData.Quota)
	}
	if quotaData.UserDisplayTokenUsed == nil {
		quotaData.UserDisplayTokenUsed = common.GetPointer(quotaData.TokenUsed)
	}
	cachedQuotaData, ok := CacheQuotaData[key]
	if ok {
		if cachedQuotaData.UserDisplayQuota == nil {
			cachedQuotaData.UserDisplayQuota = common.GetPointer(cachedQuotaData.Quota)
		}
		if cachedQuotaData.UserDisplayTokenUsed == nil {
			cachedQuotaData.UserDisplayTokenUsed = common.GetPointer(cachedQuotaData.TokenUsed)
		}
		cachedUserDisplayQuota := common.QuotaFromFloat(float64(*cachedQuotaData.UserDisplayQuota) + float64(*quotaData.UserDisplayQuota))
		cachedUserDisplayTokenUsed := common.QuotaFromFloat(float64(*cachedQuotaData.UserDisplayTokenUsed) + float64(*quotaData.UserDisplayTokenUsed))
		cachedQuotaData.Count += quotaData.Count
		cachedQuotaData.Quota = common.QuotaFromFloat(float64(cachedQuotaData.Quota) + float64(quotaData.Quota))
		cachedQuotaData.TokenUsed = common.QuotaFromFloat(float64(cachedQuotaData.TokenUsed) + float64(quotaData.TokenUsed))
		cachedQuotaData.UserDisplayQuota = common.GetPointer(cachedUserDisplayQuota)
		cachedQuotaData.UserDisplayTokenUsed = common.GetPointer(cachedUserDisplayTokenUsed)
		quotaData = cachedQuotaData
	}
	CacheQuotaData[key] = quotaData
}

func LogQuotaData(params QuotaDataLogParams) {
	// 只精确到小时
	createdAt := params.CreatedAt - (params.CreatedAt % 3600)
	quotaData := &QuotaData{
		UserID:               params.UserID,
		Username:             params.Username,
		ModelName:            params.ModelName,
		CreatedAt:            createdAt,
		UseGroup:             params.UseGroup,
		TokenID:              params.TokenID,
		ChannelID:            params.ChannelID,
		NodeName:             params.NodeName,
		Count:                1,
		Quota:                params.Quota,
		TokenUsed:            params.TokenUsed,
		UserDisplayQuota:     params.UserDisplayQuota,
		UserDisplayTokenUsed: params.UserDisplayTokenUsed,
	}

	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	logQuotaDataCache(quotaData)
}

func SaveQuotaDataCache() {
	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	size := len(CacheQuotaData)
	// 如果缓存中有数据，就保存到数据库中
	// 1. 先查询数据库中是否有数据
	// 2. 如果有数据，就更新数据
	// 3. 如果没有数据，就插入数据
	for _, quotaData := range CacheQuotaData {
		quotaDataDB := &QuotaData{}
		DB.Table("quota_data").
			Where("user_id = ? and username = ? and model_name = ? and created_at = ? and use_group = ? and token_id = ? and channel_id = ? and node_name = ?",
				quotaData.UserID, quotaData.Username, quotaData.ModelName, quotaData.CreatedAt, quotaData.UseGroup, quotaData.TokenID, quotaData.ChannelID, quotaData.NodeName).
			First(quotaDataDB)
		if quotaDataDB.Id > 0 {
			//quotaDataDB.Count += quotaData.Count
			//quotaDataDB.Quota += quotaData.Quota
			//DB.Table("quota_data").Save(quotaDataDB)
			increaseQuotaData(quotaData)
		} else {
			DB.Table("quota_data").Create(quotaData)
		}
	}
	CacheQuotaData = make(map[string]*QuotaData)
	common.SysLog(fmt.Sprintf("保存数据看板数据成功，共保存%d条数据", size))
}

func increaseQuotaData(quotaData *QuotaData) {
	err := DB.Table("quota_data").
		Where("user_id = ? and username = ? and model_name = ? and created_at = ? and use_group = ? and token_id = ? and channel_id = ? and node_name = ?",
			quotaData.UserID, quotaData.Username, quotaData.ModelName, quotaData.CreatedAt, quotaData.UseGroup, quotaData.TokenID, quotaData.ChannelID, quotaData.NodeName).
		Updates(map[string]interface{}{
			"count":                   gorm.Expr("count + ?", quotaData.Count),
			"quota":                   gorm.Expr("quota + ?", quotaData.Quota),
			"token_used":              gorm.Expr("token_used + ?", quotaData.TokenUsed),
			"user_display_quota":      gorm.Expr("COALESCE(user_display_quota, quota) + ?", *quotaData.UserDisplayQuota),
			"user_display_token_used": gorm.Expr("COALESCE(user_display_token_used, token_used) + ?", *quotaData.UserDisplayTokenUsed),
		}).Error
	if err != nil {
		common.SysLog(fmt.Sprintf("increaseQuotaData error: %s", err))
	}
}

func GetQuotaDataByUsername(username string, startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	// 从quota_data表中查询数据
	err = DB.Table("quota_data").
		Select("user_id, username, model_name, created_at, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used").
		Where("username = ? and created_at >= ? and created_at <= ?", username, startTime, endTime).
		Group("user_id, username, model_name, created_at").
		Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetQuotaDataByUserId(userId int, startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	// 从quota_data表中查询数据
	err = DB.Table("quota_data").
		Select("user_id, username, model_name, created_at, sum(count) as count, sum(COALESCE(user_display_quota, quota)) as quota, sum(COALESCE(user_display_token_used, token_used)) as token_used").
		Where("user_id = ? and created_at >= ? and created_at <= ?", userId, startTime, endTime).
		Group("user_id, username, model_name, created_at").
		Find(&quotaDatas).Error
	return quotaDatas, err
}

func backfillQuotaDataUserDisplayMetrics() error {
	return DB.Model(&QuotaData{}).
		Where("user_display_quota IS NULL OR user_display_token_used IS NULL").
		Updates(map[string]interface{}{
			"user_display_quota":      gorm.Expr("COALESCE(user_display_quota, quota)"),
			"user_display_token_used": gorm.Expr("COALESCE(user_display_token_used, token_used)"),
		}).Error
}

func GetQuotaDataGroupByUser(startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	err = DB.Table("quota_data").
		Select("username, created_at, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used").
		Where("created_at >= ? and created_at <= ?", startTime, endTime).
		Group("username, created_at").
		Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetQuotaDataGroupByUseGroup(startTime int64, endTime int64, username string) (*GroupQuotaDataAnalytics, error) {
	items := make([]GroupQuotaData, 0)
	itemsQuery := DB.Table("quota_data").
		Select("use_group, COALESCE(SUM(quota), 0) AS quota, COALESCE(SUM(count), 0) AS count, COALESCE(SUM(token_used), 0) AS token_used, COUNT(DISTINCT user_id) AS user_count").
		Where("created_at >= ? AND created_at <= ?", startTime, endTime)
	if username != "" {
		itemsQuery = itemsQuery.Where("username = ?", username)
	}
	if err := itemsQuery.Group("use_group").Order("quota DESC, use_group ASC").Scan(&items).Error; err != nil {
		return nil, err
	}

	var totals GroupQuotaDataTotals
	totalsQuery := DB.Table("quota_data").
		Select("COALESCE(SUM(quota), 0) AS quota, COALESCE(SUM(count), 0) AS count, COALESCE(SUM(token_used), 0) AS token_used, COUNT(DISTINCT user_id) AS user_count").
		Where("created_at >= ? AND created_at <= ?", startTime, endTime)
	if username != "" {
		totalsQuery = totalsQuery.Where("username = ?", username)
	}
	if err := totalsQuery.Scan(&totals).Error; err != nil {
		return nil, err
	}
	totals.GroupCount = len(items)

	return &GroupQuotaDataAnalytics{
		Items:  items,
		Totals: totals,
	}, nil
}

func GetDashboardReportData(startTime, endTime int64) (DashboardReportData, error) {
	var report DashboardReportData
	baseQuery := DB.Table("quota_data").Where("created_at >= ? AND created_at < ?", startTime, endTime)
	if err := baseQuery.Select(
		"COALESCE(SUM(quota), 0) AS quota, COALESCE(SUM(count), 0) AS count, COALESCE(SUM(token_used), 0) AS token_used, " +
			"COUNT(DISTINCT user_id) AS user_count, COUNT(DISTINCT CASE WHEN model_name <> '' THEN model_name END) AS model_count, " +
			"COUNT(DISTINCT CASE WHEN channel_id > 0 THEN channel_id END) AS channel_count, " +
			"COUNT(DISTINCT CASE WHEN use_group <> '' THEN use_group END) AS group_count",
	).Scan(&report).Error; err != nil {
		return DashboardReportData{}, err
	}

	report.TopModels = make([]DashboardReportModel, 0)
	if err := DB.Table("quota_data").
		Select("model_name, COALESCE(SUM(quota), 0) AS quota, COALESCE(SUM(count), 0) AS count, COALESCE(SUM(token_used), 0) AS token_used").
		Where("created_at >= ? AND created_at < ? AND model_name <> ''", startTime, endTime).
		Group("model_name").
		Order("quota DESC, model_name ASC").
		Limit(5).
		Scan(&report.TopModels).Error; err != nil {
		return DashboardReportData{}, err
	}
	return report, nil
}

func GetAllQuotaDates(startTime int64, endTime int64, username string) (quotaData []*QuotaData, err error) {
	if username != "" {
		return GetQuotaDataByUsername(username, startTime, endTime)
	}
	var quotaDatas []*QuotaData
	// 从quota_data表中查询数据
	// only select model_name, sum(count) as count, sum(quota) as quota, model_name, created_at from quota_data group by model_name, created_at;
	//err = DB.Table("quota_data").Where("created_at >= ? and created_at <= ?", startTime, endTime).Find(&quotaDatas).Error
	err = DB.Table("quota_data").Select("model_name, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used, created_at").Where("created_at >= ? and created_at <= ?", startTime, endTime).Group("model_name, created_at").Find(&quotaDatas).Error
	return quotaDatas, err
}
