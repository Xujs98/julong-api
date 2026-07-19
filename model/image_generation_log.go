package model

import (
	"encoding/json"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	ImageGenerationStatusPending    = "pending"
	ImageGenerationStatusProcessing = "processing"
	ImageGenerationStatusSuccess    = "success"
	ImageGenerationStatusFailed     = "failed"
)

type ImageGenerationImage struct {
	Type          string `json:"type"`
	Value         string `json:"value"`
	Bucket        string `json:"bucket,omitempty"`
	MimeType      string `json:"mime_type,omitempty"`
	SHA256        string `json:"sha256,omitempty"`
	Size          int64  `json:"size,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

func (log *ImageGenerationLog) ImageRefs() ([]ImageGenerationImage, error) {
	var refs []ImageGenerationImage
	if log.Images == "" {
		return refs, nil
	}
	if err := json.Unmarshal([]byte(log.Images), &refs); err != nil {
		return nil, err
	}
	return refs, nil
}

type ImageGenerationLog struct {
	Id           int      `json:"id"`
	TaskId       string   `json:"task_id" gorm:"type:varchar(64);index"`
	Status       string   `json:"status" gorm:"type:varchar(20);index;default:success"`
	UserId       int      `json:"user_id" gorm:"index"`
	Username     string   `json:"username" gorm:"type:varchar(64);index"`
	TokenId      int      `json:"token_id" gorm:"index"`
	TokenName    string   `json:"token_name" gorm:"type:varchar(128)"`
	ChannelId    int      `json:"channel_id" gorm:"index"`
	ModelName    string   `json:"model_name" gorm:"type:varchar(191);index"`
	Prompt       string   `json:"prompt" gorm:"type:text"`
	Size         string   `json:"size" gorm:"type:varchar(32)"`
	Quality      string   `json:"quality" gorm:"type:varchar(32)"`
	ImageCount   int      `json:"image_count"`
	Images       string   `json:"-" gorm:"type:text"`
	ImageUrls    []string `json:"image_urls" gorm:"-:all"`
	Quota        int      `json:"quota"`
	RequestId    string   `json:"request_id" gorm:"type:varchar(64);index"`
	CreatedAt    int64    `json:"created_at" gorm:"index"`
	UpdatedAt    int64    `json:"updated_at" gorm:"index"`
	UseTime      int      `json:"use_time"`
	ErrorMessage string   `json:"error_message,omitempty" gorm:"type:text"`
	Response     string   `json:"-" gorm:"type:text"`
	ChannelName  string   `json:"channel_name" gorm:"-:all"`
}

func (log *ImageGenerationLog) Insert() error {
	return DB.Create(log).Error
}

func GetImageGenerationLogs(userId int, isAdmin bool, visibleLimit, startIdx, pageSize, channelId int, modelName, prompt string, startTime, endTime int64) ([]*ImageGenerationLog, int64, error) {
	query := DB.Model(&ImageGenerationLog{})
	if !isAdmin {
		query = query.Where("user_id = ?", userId)
		if visibleLimit > 0 {
			query = query.Where("id IN (?)", DB.Model(&ImageGenerationLog{}).
				Select("id").Where("user_id = ?", userId).Order("id DESC").Limit(visibleLimit))
		}
	}
	if modelName != "" {
		query = query.Where("model_name LIKE ?", "%"+modelName+"%")
	}
	if channelId > 0 {
		query = query.Where("channel_id = ?", channelId)
	}
	if prompt != "" {
		query = query.Where("prompt LIKE ?", "%"+prompt+"%")
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
	var logs []*ImageGenerationLog
	if err := query.Order("id DESC").Offset(startIdx).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	for _, log := range logs {
		if channel, err := GetChannelById(log.ChannelId, true); err == nil && channel != nil {
			log.ChannelName = channel.Name
		}
	}
	return logs, total, nil
}

func IsImageGenerationLogVisibleToUser(logId, userId, visibleLimit int) (bool, error) {
	if visibleLimit <= 0 {
		return true, nil
	}
	var count int64
	err := DB.Model(&ImageGenerationLog{}).
		Where("user_id = ? AND id >= ?", userId, logId).
		Count(&count).Error
	return count > 0 && count <= int64(visibleLimit), err
}

func GetImageGenerationLogById(id int) (*ImageGenerationLog, error) {
	var log ImageGenerationLog
	if err := DB.First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

func GetImageGenerationLogByTaskId(taskId string) (*ImageGenerationLog, error) {
	var log ImageGenerationLog
	if err := DB.Where("task_id = ?", taskId).First(&log).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

func GetUserImageGenerationLogByTaskId(userId int, taskId string) (*ImageGenerationLog, error) {
	var log ImageGenerationLog
	if err := DB.Where("user_id = ? AND task_id = ?", userId, taskId).First(&log).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

func (log *ImageGenerationLog) UpdateTask(fields map[string]any) error {
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["updated_at"] = time.Now().Unix()
	return DB.Model(log).Updates(fields).Error
}

func NewImageGenerationLog(userId int) *ImageGenerationLog {
	username, _ := GetUsernameById(userId, false)
	return &ImageGenerationLog{
		UserId:    userId,
		Username:  username,
		Status:    ImageGenerationStatusSuccess,
		CreatedAt: common.GetTimestamp(),
		UpdatedAt: common.GetTimestamp(),
	}
}
