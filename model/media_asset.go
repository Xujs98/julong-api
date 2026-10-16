package model

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	MediaAssetSourceBase64 = "base64"
	MediaAssetSourcePublic = "public"
)

type MediaAsset struct {
	ID          int64  `json:"id" gorm:"primaryKey"`
	AssetID     string `json:"asset_id" gorm:"type:varchar(64);uniqueIndex"`
	UserID      int    `json:"user_id" gorm:"index"`
	Username    string `json:"username,omitempty" gorm:"-"`
	SourceType  string `json:"source_type" gorm:"type:varchar(16);index"`
	MediaType   string `json:"media_type" gorm:"type:varchar(16);index"`
	MimeType    string `json:"mime_type" gorm:"type:varchar(128)"`
	OriginalURL string `json:"original_url,omitempty" gorm:"type:text"`
	StoredPath  string `json:"-" gorm:"type:text"`
	Size        int64  `json:"size"`
	CreatedAt   int64  `json:"created_at" gorm:"index"`
	ExpiresAt   int64  `json:"expires_at" gorm:"index"`
}

func (asset *MediaAsset) BeforeCreate(_ *gorm.DB) error {
	if asset.CreatedAt == 0 {
		asset.CreatedAt = time.Now().Unix()
	}
	if asset.AssetID == "" {
		id, err := common.GenerateRandomCharsKey(32)
		if err != nil {
			return err
		}
		asset.AssetID = "asset_" + id
	}
	if asset.SourceType != MediaAssetSourceBase64 && asset.SourceType != MediaAssetSourcePublic {
		return errors.New("invalid media asset source type")
	}
	if asset.MediaType != "image" && asset.MediaType != "video" && asset.MediaType != "audio" {
		return errors.New("invalid media asset media type")
	}
	return nil
}

func ListMediaAssets(userID int, admin bool, offset, limit int, startTime, endTime int64, username string, scopes ...UserLogScope) ([]MediaAsset, int64, error) {
	query := DB.Model(&MediaAsset{})
	for _, scope := range scopes {
		query = scope.Apply(query)
	}
	if !admin {
		query = query.Where("user_id = ?", userID)
	} else if username != "" {
		users := DB.Model(&User{}).Select("id").Where("username LIKE ?", "%"+username+"%")
		query = query.Where("user_id IN (?)", users)
	}
	if startTime > 0 {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where("created_at <= ?", endTime)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var assets []MediaAsset
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&assets).Error; err != nil {
		return nil, 0, err
	}
	if admin && len(assets) > 0 {
		ids := make([]int, 0, len(assets))
		for _, asset := range assets {
			ids = append(ids, asset.UserID)
		}
		var users []User
		if err := DB.Select("id", "username").Where("id IN ?", ids).Find(&users).Error; err == nil {
			byID := make(map[int]string, len(users))
			for _, user := range users {
				byID[user.Id] = user.Username
			}
			for i := range assets {
				assets[i].Username = byID[assets[i].UserID]
			}
		}
	}
	return assets, total, nil
}

func GetMediaAsset(assetID string) (*MediaAsset, error) {
	var asset MediaAsset
	if err := DB.Where("asset_id = ?", assetID).First(&asset).Error; err != nil {
		return nil, err
	}
	return &asset, nil
}

func HasExpiredMediaAssets(now int64) bool {
	var count int64
	return DB.Model(&MediaAsset{}).Where("source_type = ? AND expires_at > 0 AND expires_at <= ?", MediaAssetSourceBase64, now).Limit(1).Count(&count).Error == nil && count > 0
}

func DeleteMediaAsset(assetID string, userID int, admin bool) (*MediaAsset, error) {
	asset, err := GetMediaAsset(assetID)
	if err != nil {
		return nil, err
	}
	if !admin && asset.UserID != userID {
		return nil, gorm.ErrRecordNotFound
	}
	if err := DB.Delete(asset).Error; err != nil {
		return nil, err
	}
	return asset, nil
}
