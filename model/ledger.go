package model

import (
	"math"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type LedgerEntry struct {
	Id         int             `json:"id"`
	Platform   string          `json:"platform" gorm:"type:varchar(32);index;not null"`
	Account    string          `json:"account" gorm:"type:varchar(255);not null"`
	Email      string          `json:"email" gorm:"type:varchar(255);default:''"`
	Type       string          `json:"type" gorm:"type:varchar(64);index;not null"`
	Quota      int             `json:"quota" gorm:"type:int;not null"`
	CostPrice  decimal.Decimal `json:"cost_price" gorm:"type:decimal(20,6);not null"`
	Quantity   int             `json:"quantity" gorm:"type:int;not null"`
	OccurredAt int64           `json:"occurred_at" gorm:"type:bigint;index;not null"`
	CreatedBy  int             `json:"created_by" gorm:"type:int;index;not null"`
	CreatedAt  int64           `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt  int64           `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
	DeletedAt  gorm.DeletedAt  `json:"-" gorm:"index"`
}

type LedgerAggregateRow struct {
	Type      string
	Quota     int
	CostPrice decimal.Decimal
	Quantity  int
}

type LedgerAggregate struct {
	QuotaByType map[string]int64
	CostByType  map[string]decimal.Decimal
	TotalQuota  int64
	TotalCost   decimal.Decimal
}

type LedgerUsageQuota struct {
	Real int64
}

func ledgerDateQuery(startTimestamp int64, endTimestamp int64) *gorm.DB {
	query := DB.Model(&LedgerEntry{})
	if startTimestamp > 0 {
		query = query.Where("occurred_at >= ?", startTimestamp)
	}
	if endTimestamp > 0 {
		query = query.Where("occurred_at <= ?", endTimestamp)
	}
	return query
}

func GetLedgerEntries(startTimestamp int64, endTimestamp int64, startIdx int, limit int) ([]LedgerEntry, int64, error) {
	query := ledgerDateQuery(startTimestamp, endTimestamp)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	entries := make([]LedgerEntry, 0)
	err := query.Order("occurred_at desc").Order("id desc").Offset(startIdx).Limit(limit).Find(&entries).Error
	return entries, total, err
}

func GetLedgerEntryById(id int) (*LedgerEntry, error) {
	var entry LedgerEntry
	if err := DB.First(&entry, id).Error; err != nil {
		return nil, err
	}
	return &entry, nil
}

func CreateLedgerEntry(entry *LedgerEntry) error {
	return DB.Create(entry).Error
}

func UpdateLedgerEntry(entry *LedgerEntry) error {
	return DB.Save(entry).Error
}

func DeleteLedgerEntries(ids []int) (int64, error) {
	result := DB.Where("id IN ?", ids).Delete(&LedgerEntry{})
	return result.RowsAffected, result.Error
}

func GetLedgerAggregate(startTimestamp int64, endTimestamp int64) (*LedgerAggregate, error) {
	rows := make([]LedgerAggregateRow, 0)
	if err := ledgerDateQuery(startTimestamp, endTimestamp).
		Select("type", "quota", "cost_price", "quantity").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	aggregate := &LedgerAggregate{
		QuotaByType: make(map[string]int64),
		CostByType:  make(map[string]decimal.Decimal),
		TotalCost:   decimal.Zero,
	}
	for _, row := range rows {
		quantity := int64(row.Quantity)
		quota := int64(row.Quota) * quantity
		cost := row.CostPrice.Mul(decimal.NewFromInt(quantity))
		aggregate.TotalQuota = saturatingAddInt64(aggregate.TotalQuota, quota)
		aggregate.TotalCost = aggregate.TotalCost.Add(cost)
		aggregate.QuotaByType[row.Type] = saturatingAddInt64(aggregate.QuotaByType[row.Type], quota)
		aggregate.CostByType[row.Type] = aggregate.CostByType[row.Type].Add(cost)
	}
	return aggregate, nil
}

func GetLedgerUsageQuota(startTimestamp int64, endTimestamp int64) (*LedgerUsageQuota, error) {
	query := LOG_DB.Model(&Log{}).
		Select("quota").
		Where("type = ?", LogTypeConsume)
	if startTimestamp > 0 {
		query = query.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp > 0 {
		query = query.Where("created_at <= ?", endTimestamp)
	}

	rows, err := query.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &LedgerUsageQuota{}
	for rows.Next() {
		var quota int
		if err := rows.Scan(&quota); err != nil {
			return nil, err
		}
		if quota <= 0 {
			continue
		}
		result.Real = saturatingAddInt64(result.Real, int64(quota))
	}
	return result, rows.Err()
}

func saturatingAddInt64(left int64, right int64) int64 {
	if right > 0 && left > math.MaxInt64-right {
		return math.MaxInt64
	}
	if right < 0 && left < math.MinInt64-right {
		return math.MinInt64
	}
	return left + right
}
