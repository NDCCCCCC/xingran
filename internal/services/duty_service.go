package services

import (
	"context"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// DutyService 值班管理服务
// 作为统一入口，内部委托给专门的服务
type DutyService struct {
	db              *gorm.DB
	poolService     *DutyPoolService
	scheduleService *DutyScheduleService
	statsService    *DutyStatsService
	holidayService  *DutyHolidayService
	configService   *DutyConfigService
}

// NewDutyService 创建值班管理服务
func NewDutyService(db *gorm.DB) *DutyService {
	return &DutyService{
		db:              db,
		poolService:     NewDutyPoolService(db),
		scheduleService: NewDutyScheduleService(db),
		statsService:    NewDutyStatsService(db),
		holidayService:  NewDutyHolidayService(db),
		configService:   NewDutyConfigService(db),
	}
}

// ==================== 请求/响应结构 ====================

// DutyPoolListRequest 值班池列表请求
type DutyPoolListRequest struct {
	base.BaseListRequest
	PoolName *string `json:"poolName,omitempty"`
	DeptID   *string `json:"deptId,omitempty"`
	Status   *int    `json:"status,omitempty"`
}

// DutyPoolCreateRequest 创建值班池请求
type DutyPoolCreateRequest struct {
	PoolName    string   `json:"poolName" binding:"required,max=100"`
	DeptID      *string  `json:"deptId,omitempty"`
	Description string   `json:"description,omitempty"`
	DailyCount  int      `json:"dailyCount" binding:"min=1,max=10"`
	MemberIDs   []string `json:"memberIds" binding:"required,min=1"`
}

// DutyPoolUpdateRequest 更新值班池请求
type DutyPoolUpdateRequest struct {
	ID          string   `json:"-"` // ID从URL参数获取，不参与JSON绑定
	PoolName    string   `json:"poolName" binding:"required,max=100"`
	DeptID      *string  `json:"deptId,omitempty"`
	Description string   `json:"description,omitempty"`
	DailyCount  int      `json:"dailyCount" binding:"min=1,max=10"`
	Status      *int     `json:"status,omitempty"` // 使用指针类型，未提供时为nil
	MemberIDs   []string `json:"memberIds" binding:"required,min=1"`
}

// DutyScheduleListRequest 排班列表请求
type DutyScheduleListRequest struct {
	base.BaseListRequest
	PoolID    *string `json:"poolId,omitempty"`
	UserID    *string `json:"userId,omitempty"`
	StartDate *string `json:"startDate,omitempty"` // YYYY-MM-DD
	EndDate   *string `json:"endDate,omitempty"`   // YYYY-MM-DD
	DutyType  *string `json:"dutyType,omitempty"`
	Status    *int    `json:"status,omitempty"`
	Expired   *int    `json:"expired,omitempty"` // 过期状态：0=未过期，1=已过期，不传=全部
}

// GenerateScheduleRequest 生成排班请求
type GenerateScheduleRequest struct {
	PoolID      string `json:"poolId" binding:"required"`
	StartDate   string `json:"startDate" binding:"required"`                              // YYYY-MM-DD
	EndDate     string `json:"endDate" binding:"required"`                                // YYYY-MM-DD
	DutyType    string `json:"dutyType" binding:"required,oneof=weekday weekend holiday"` // 值班类型：工作日/周末/节假日
	ClearExists bool   `json:"clearExists"`                                               // 是否清除已有排班
}

// SwapDutyRequest 调班请求
type SwapDutyRequest struct {
	FromScheduleID string `json:"fromScheduleId" binding:"required"` // 原排班ID
	ToScheduleID   string `json:"toScheduleId" binding:"required"`   // 目标排班ID
	Reason         string `json:"reason"`
}

// ManualDutyRequest 手动排班请求
type ManualDutyRequest struct {
	PoolID   string   `json:"poolId" binding:"required"`
	DutyDate string   `json:"dutyDate" binding:"required"` // YYYY-MM-DD
	UserIDs  []string `json:"userIds" binding:"required,min=1"`
	DutyType string   `json:"dutyType" binding:"required,oneof=weekday weekend holiday"`
	Reason   string   `json:"reason"`
}

// TodayDutyMember 今日值班人员信息
type TodayDutyMember struct {
	ScheduleID string `json:"scheduleId"`
	PoolID     string `json:"poolId"`
	PoolName   string `json:"poolName"`
	UserID     string `json:"userId"`
	Username   string `json:"username"`
	Phone      string `json:"phone"`
	DutyType   string `json:"dutyType"`
}

// MyDutyStats 我的值班统计信息
type MyDutyStats struct {
	IsOnDutyToday    bool              `json:"isOnDutyToday"`              // 今日是否值班
	TodayDutyRecords []TodayDutyMember `json:"todayDutyRecords,omitempty"` // 今日值班记录（如果今日值班）
	ThisMonthCount   int               `json:"thisMonthCount"`             // 本月值班次数
	TotalCount       int               `json:"totalCount"`                 // 总值班次数
	NextDutyDate     *string           `json:"nextDutyDate,omitempty"`     // 下次值班时间（格式：YYYY-MM-DD）
	NextDutyPoolName *string           `json:"nextDutyPoolName,omitempty"` // 下次值班池名称
}

// ==================== 值班池管理（委托给 DutyPoolService） ====================

// CreateDutyPool 创建值班池
func (s *DutyService) CreateDutyPool(ctx context.Context, req *DutyPoolCreateRequest, creatorID string) (*models.DutyPool, error) {
	return s.poolService.CreateDutyPool(ctx, req, creatorID)
}

// GetDutyPoolList 获取值班池列表
func (s *DutyService) GetDutyPoolList(ctx context.Context, req *DutyPoolListRequest) ([]models.DutyPool, int64, error) {
	return s.poolService.GetDutyPoolList(ctx, req)
}

// GetDutyPoolStatistics 统计值班池总数/启停数及成员总数
func (s *DutyService) GetDutyPoolStatistics(ctx context.Context) (*DutyPoolStatistics, error) {
	return s.poolService.GetDutyPoolStatistics(ctx)
}

// GetDutyPoolByID 根据ID获取值班池
func (s *DutyService) GetDutyPoolByID(ctx context.Context, poolID string) (*models.DutyPool, error) {
	return s.poolService.GetDutyPoolByID(ctx, poolID)
}

// UpdateDutyPool 更新值班池
func (s *DutyService) UpdateDutyPool(ctx context.Context, req *DutyPoolUpdateRequest, updaterID string) error {
	return s.poolService.UpdateDutyPool(ctx, req, updaterID)
}

// DeleteDutyPool 删除值班池
func (s *DutyService) DeleteDutyPool(ctx context.Context, poolID string) error {
	return s.poolService.DeleteDutyPool(ctx, poolID)
}

// ==================== 排班管理（委托给 DutyScheduleService） ====================

// GenerateSchedule 生成排班
func (s *DutyService) GenerateSchedule(ctx context.Context, req *GenerateScheduleRequest, creatorID string) (int, error) {
	return s.scheduleService.GenerateSchedule(ctx, req, creatorID)
}

// GetDutyScheduleList 获取排班列表
func (s *DutyService) GetDutyScheduleList(ctx context.Context, req *DutyScheduleListRequest) ([]models.DutySchedule, int64, error) {
	return s.scheduleService.GetDutyScheduleList(ctx, req)
}

// GetTodayDuty 获取今日值班人员
func (s *DutyService) GetTodayDuty(ctx context.Context) ([]TodayDutyMember, error) {
	return s.scheduleService.GetTodayDuty(ctx)
}

// GetMonthlyDutySchedule 获取月度值班排班
func (s *DutyService) GetMonthlyDutySchedule(ctx context.Context, year int, month int) (map[string][]TodayDutyMember, error) {
	return s.scheduleService.GetMonthlyDutySchedule(ctx, year, month)
}

// ==================== 调班管理（委托给 DutyScheduleService） ====================

// SwapDuty 调班
func (s *DutyService) SwapDuty(ctx context.Context, req *SwapDutyRequest, operatorID string) error {
	return s.scheduleService.SwapDuty(ctx, req, operatorID)
}

// ManualDuty 手动排班
func (s *DutyService) ManualDuty(ctx context.Context, req *ManualDutyRequest, creatorID string) error {
	return s.scheduleService.ManualDuty(ctx, req, creatorID)
}

// DeleteDutySchedule 删除排班记录
func (s *DutyService) DeleteDutySchedule(ctx context.Context, scheduleID string) error {
	return s.scheduleService.DeleteDutySchedule(ctx, scheduleID)
}

// BatchDeleteDutySchedules 批量删除排班记录
func (s *DutyService) BatchDeleteDutySchedules(ctx context.Context, scheduleIDs []string) error {
	return s.scheduleService.BatchDeleteDutySchedules(ctx, scheduleIDs)
}

// ==================== 我的值班统计（委托给 DutyStatsService） ====================

// GetMyDutyStats 获取当前用户的值班统计
func (s *DutyService) GetMyDutyStats(ctx context.Context, userID string) (*MyDutyStats, error) {
	return s.statsService.GetMyDutyStats(ctx, userID)
}

// ==================== 节假日管理（委托给 DutyHolidayService） ====================

// CreateHoliday 创建节假日
func (s *DutyService) CreateHoliday(ctx context.Context, holiday *models.Holiday, creatorID string) error {
	return s.holidayService.CreateHoliday(ctx, holiday, creatorID)
}

// GetHolidayList 获取节假日列表
func (s *DutyService) GetHolidayList(ctx context.Context, year int) ([]models.Holiday, error) {
	return s.holidayService.GetHolidayList(ctx, year)
}

// UpdateHoliday 更新节假日
func (s *DutyService) UpdateHoliday(ctx context.Context, holiday *models.Holiday, updaterID string) error {
	return s.holidayService.UpdateHoliday(ctx, holiday, updaterID)
}

// DeleteHoliday 删除节假日
func (s *DutyService) DeleteHoliday(ctx context.Context, holidayID string) error {
	return s.holidayService.DeleteHoliday(ctx, holidayID)
}

// GetHolidayYears 获取所有有节假日数据的年份列表
func (s *DutyService) GetHolidayYears(ctx context.Context) ([]int, error) {
	return s.holidayService.GetHolidayYears(ctx)
}

// BatchCreateHolidays 批量创建节假日
func (s *DutyService) BatchCreateHolidays(ctx context.Context, holidays []models.Holiday, creatorID string) error {
	return s.holidayService.BatchCreateHolidays(ctx, holidays, creatorID)
}

// ==================== 值班配置管理（委托给 DutyConfigService） ====================

// GetDutyConfig 获取值班配置（系统中只有一条配置记录）
func (s *DutyService) GetDutyConfig(ctx context.Context) (*models.DutyConfig, error) {
	return s.configService.GetDutyConfig(ctx)
}

// UpdateDutyConfig 更新值班配置
func (s *DutyService) UpdateDutyConfig(ctx context.Context, config *models.DutyConfig, updaterID string) error {
	return s.configService.UpdateDutyConfig(ctx, config, updaterID)
}
