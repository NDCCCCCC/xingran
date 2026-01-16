package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ==================== 枚举定义 ====================

// ScheduleMode 排班模式枚举
type ScheduleMode string

const (
	ScheduleModeWeekday ScheduleMode = "weekday" // 工作日
	ScheduleModeWeekend ScheduleMode = "weekend" // 周末
	ScheduleModeHoliday ScheduleMode = "holiday" // 节假日
)

// DutyStatus 值班状态枚举
type DutyStatus int

const (
	DutyStatusNormal    DutyStatus = 0 // 正常
	DutyStatusExchanged DutyStatus = 1 // 已调换
	DutyStatusCancelled DutyStatus = 2 // 已取消
)

// HolidayType 节假日类型枚举
type HolidayType string

const (
	HolidayTypeLegal   HolidayType = "legal"   // 法定节假日
	HolidayTypeWorkday HolidayType = "workday" // 调休工作日
	HolidayTypeCustom  HolidayType = "custom"  // 自定义节假日
)

// DutyPoolStatus 值班池状态枚举
type DutyPoolStatus int

const (
	DutyPoolStatusEnabled  DutyPoolStatus = 0 // 启用
	DutyPoolStatusDisabled DutyPoolStatus = 1 // 停用
)

// ==================== 模型定义 ====================

// DutyPool 值班人员池
type DutyPool struct {
	BaseModel
	PoolName    string         `gorm:"size:100;not null" json:"poolName"`
	DeptID      *string        `gorm:"type:uuid" json:"deptId,omitempty"`
	Description string         `gorm:"size:500" json:"description,omitempty"`
	Status      DutyPoolStatus `gorm:"default:0" json:"status"`
	DailyCount  int            `gorm:"default:1" json:"dailyCount"` // 每日值班人数

	// 关联
	Dept    *Department      `gorm:"foreignKey:DeptID" json:"department,omitempty"`
	Members []DutyPoolMember `gorm:"foreignKey:PoolID" json:"members,omitempty"`
}

// TableName 指定表名
func (DutyPool) TableName() string {
	return "sys_duty_pool"
}

// DutyPoolMember 值班池成员
type DutyPoolMember struct {
	ID          string    `gorm:"primaryKey;type:uuid" json:"id"`
	PoolID      string    `gorm:"type:uuid;not null;index:idx_duty_pool_member_pool,priority:1" json:"poolId"`
	UserID      string    `gorm:"type:uuid;not null;index:idx_duty_pool_member_user,priority:1" json:"userId"`
	MemberOrder int       `gorm:"default:0" json:"memberOrder"` // 排序顺序(用于轮询)
	CreatedAt   time.Time `json:"createdAt"`

	// 关联
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// BeforeCreate GORM钩子 - DutyPoolMember
func (dpm *DutyPoolMember) BeforeCreate(tx *gorm.DB) error {
	if dpm.ID == "" {
		dpm.ID = uuid.New().String()
	}
	if dpm.CreatedAt.IsZero() {
		dpm.CreatedAt = time.Now()
	}
	return nil
}

// TableName 指定表名
func (DutyPoolMember) TableName() string {
	return "sys_duty_pool_member"
}

// DutyScheduleConfig 排班配置
type DutyScheduleConfig struct {
	BaseModel
	ConfigName   string       `gorm:"size:100;not null" json:"configName"`
	PoolID       string       `gorm:"type:uuid;not null" json:"poolId"`
	ScheduleMode ScheduleMode `gorm:"size:20;not null" json:"scheduleMode"`
	IsActive     bool         `gorm:"default:true" json:"isActive"`
	StartDate    time.Time    `gorm:"type:date;not null" json:"startDate"`
	EndDate      *time.Time   `gorm:"type:date" json:"endDate,omitempty"`

	// 关联
	Pool *DutyPool `gorm:"foreignKey:PoolID" json:"pool,omitempty"`
}

// TableName 指定表名
func (DutyScheduleConfig) TableName() string {
	return "sys_duty_schedule_config"
}

// DutySchedule 排班记录
type DutySchedule struct {
	BaseModel
	ScheduleDate time.Time    `gorm:"type:date;not null;index:idx_duty_schedule_date,priority:1" json:"scheduleDate"`
	PoolID       string       `gorm:"type:uuid;not null;index:idx_duty_schedule_pool,priority:1" json:"poolId"`
	UserID       string       `gorm:"type:uuid;not null" json:"userId"`
	DutyType     ScheduleMode `gorm:"size:20;not null" json:"dutyType"`
	Status       DutyStatus   `gorm:"default:0" json:"status"`
	IsManual     bool         `gorm:"default:false" json:"isManual"` // 是否手动调整
	SwapFromDate *time.Time   `gorm:"type:date" json:"swapFromDate,omitempty"`
	SwapReason   string       `gorm:"size:500" json:"swapReason,omitempty"`

	// 关联
	Pool *DutyPool `gorm:"foreignKey:PoolID" json:"pool,omitempty"`
	User *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (DutySchedule) TableName() string {
	return "sys_duty_schedule"
}

// DutyExchange 调班记录
type DutyExchange struct {
	ID             string    `gorm:"primaryKey;type:uuid" json:"id"`
	ScheduleID     string    `gorm:"type:uuid;not null" json:"scheduleId"`
	OriginalUserID string    `gorm:"type:uuid;not null" json:"originalUserId"`
	NewUserID      string    `gorm:"type:uuid;not null" json:"newUserId"`
	ExchangeDate   time.Time `gorm:"type:date;not null" json:"exchangeDate"`
	Reason         string    `gorm:"size:500" json:"reason,omitempty"`
	CreatedBy      string    `gorm:"size:64" json:"createdBy"`
	CreatedAt      time.Time `json:"createdAt"`

	// 关联
	Schedule     *DutySchedule `gorm:"foreignKey:ScheduleID" json:"schedule,omitempty"`
	OriginalUser *User         `gorm:"foreignKey:OriginalUserID" json:"originalUser,omitempty"`
	NewUser      *User         `gorm:"foreignKey:NewUserID" json:"newUser,omitempty"`
}

// BeforeCreate GORM钩子 - DutyExchange
func (de *DutyExchange) BeforeCreate(tx *gorm.DB) error {
	if de.ID == "" {
		de.ID = uuid.New().String()
	}
	if de.CreatedAt.IsZero() {
		de.CreatedAt = time.Now()
	}
	return nil
}

// TableName 指定表名
func (DutyExchange) TableName() string {
	return "sys_duty_exchange"
}

// Holiday 节假日
type Holiday struct {
	BaseModel
	HolidayDate time.Time   `gorm:"type:date;not null;uniqueIndex:idx_holiday_date" json:"holidayDate"`
	HolidayName string      `gorm:"size:100;not null" json:"holidayName"`
	IsOffday    bool        `gorm:"default:true" json:"isOffday"` // true=休息日 false=调休工作日
	HolidayType HolidayType `gorm:"size:20;default:'custom'" json:"holidayType"`
	Year        int         `gorm:"not null;index:idx_holiday_year" json:"year"`
	Remark      string      `gorm:"size:500" json:"remark,omitempty"`
}

// TableName 指定表名
func (Holiday) TableName() string {
	return "sys_holiday"
}

// DutyConfig 值班配置
type DutyConfig struct {
	BaseModel
	ReminderEnabled       bool   `gorm:"default:true" json:"reminderEnabled"`                  // 是否启用值班提醒
	ReminderTime          string `gorm:"size:10;default:'08:00'" json:"reminderTime"`          // 提醒时间 HH:mm格式
	ReminderChannels      string `gorm:"size:100;default:'websocket'" json:"reminderChannels"` // 提醒渠道（websocket,email,sms，逗号分隔）
	BeforeReminderMinutes *int   `gorm:"default:0" json:"beforeReminderMinutes,omitempty"`     // 提前提醒分钟数（0=当天提醒）
}

// TableName 指定表名
func (DutyConfig) TableName() string {
	return "sys_duty_config"
}
