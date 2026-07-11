package model

import "time"

type ErrorReport struct {
	Id          int    `json:"id"`
	CreatedAt   int64  `json:"created_at" gorm:"index"`
	UserId      int    `json:"user_id" gorm:"index"`
	Username    string `json:"username" gorm:"size:64;default:''"`
	Title       string `json:"title" gorm:"size:200;default:''"`
	Message     string `json:"message" gorm:"type:text"`
	PageUrl     string `json:"page_url" gorm:"type:text"`
	ErrorStatus int    `json:"error_status" gorm:"default:500;index"`
	UserAgent   string `json:"user_agent" gorm:"type:text"`
	Stack       string `json:"stack" gorm:"type:text"`
	Ip          string `json:"ip" gorm:"size:64;default:''"`
}

func CreateErrorReport(report *ErrorReport) error {
	report.CreatedAt = time.Now().Unix()
	return DB.Create(report).Error
}

func GetErrorReports(startIdx int, pageSize int) ([]*ErrorReport, int64, error) {
	var reports []*ErrorReport
	var total int64
	if err := DB.Model(&ErrorReport{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := DB.Order("id desc").Limit(pageSize).Offset(startIdx).Find(&reports).Error
	return reports, total, err
}

func GetErrorReportById(id int) (*ErrorReport, error) {
	var report ErrorReport
	err := DB.Where("id = ?", id).First(&report).Error
	return &report, err
}
