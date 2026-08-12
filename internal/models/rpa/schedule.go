package rpa

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"time"
)

// Schedule RPA 定时调度配置
type Schedule struct {
	models.BaseModel
	Name        string     `gorm:"size:255;not null" json:"name"`
	TaskID      string     `gorm:"type:uuid;not null" json:"taskId"`
	CronExpr    string     `gorm:"size:100;not null" json:"cronExpr"`
	Timezone    string     `gorm:"size:50;default:'Asia/Shanghai'" json:"timezone"`
	StartDate   *time.Time `json:"startDate"`
	EndDate     *time.Time `json:"endDate"`
	Status      int        `gorm:"not null;default:0" json:"status"` // 0=enabled 1=disabled
	NextRunTime *time.Time `json:"nextRunTime"`
	LastRunTime *time.Time `json:"lastRunTime"`
	RunCount    int        `gorm:"default:0" json:"runCount"`
	Description string     `gorm:"type:text" json:"description"`
}

// TableName 指定表名
func (Schedule) TableName() string {
	return "sys_rpa_schedules"
}
