package duty

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"gorm.io/gorm"
)

// DutyCacheService 值班管理缓存服务接口
type DutyCacheService interface {
	// 值班池管理（委托给基础服务）
	CreateDutyPool(ctx context.Context, req *services.DutyPoolCreateRequest, creatorID string) (*models.DutyPool, error)
	GetDutyPoolList(ctx context.Context, req *services.DutyPoolListRequest) ([]models.DutyPool, int64, error)
	GetDutyPoolStatistics(ctx context.Context) (*services.DutyPoolStatistics, error)
	GetDutyPoolByID(ctx context.Context, poolID string) (*models.DutyPool, error)
	UpdateDutyPool(ctx context.Context, req *services.DutyPoolUpdateRequest, updaterID string) error
	DeleteDutyPool(ctx context.Context, poolID string) error

	// 排班管理（部分带缓存）
	GenerateSchedule(ctx context.Context, req *services.GenerateScheduleRequest, creatorID string) (int, error)
	GetDutyScheduleList(ctx context.Context, req *services.DutyScheduleListRequest) ([]models.DutySchedule, int64, error)
	GetTodayDuty(ctx context.Context) ([]services.TodayDutyMember, error)
	GetMonthlyDutySchedule(ctx context.Context, year int, month int) (map[string][]services.TodayDutyMember, error)
	SwapDuty(ctx context.Context, req *services.SwapDutyRequest, operatorID string) error
	ManualDuty(ctx context.Context, req *services.ManualDutyRequest, creatorID string) error
	DeleteDutySchedule(ctx context.Context, scheduleID string) error
	BatchDeleteDutySchedules(ctx context.Context, scheduleIDs []string) error

	// 我的值班统计（委托给基础服务）
	GetMyDutyStats(ctx context.Context, userID string) (*services.MyDutyStats, error)

	// 节假日管理（带缓存）
	CreateHoliday(ctx context.Context, holiday *models.Holiday, creatorID string) error
	GetHolidayList(ctx context.Context, year int) ([]models.Holiday, error)
	UpdateHoliday(ctx context.Context, holiday *models.Holiday, updaterID string) error
	DeleteHoliday(ctx context.Context, holidayID string) error
	GetHolidayYears(ctx context.Context) ([]int, error)
	BatchCreateHolidays(ctx context.Context, holidays []models.Holiday, creatorID string) error

	// 值班配置管理（委托给基础服务）
	GetDutyConfig(ctx context.Context) (*models.DutyConfig, error)
	UpdateDutyConfig(ctx context.Context, config *models.DutyConfig, updaterID string) error

	// 缓存失效方法
	InvalidateTodayDutyCache(ctx context.Context) error
	InvalidateMonthlyScheduleCache(ctx context.Context, year, month int) error
	InvalidateAllScheduleCache(ctx context.Context) error
	InvalidateHolidayCache(ctx context.Context, year int) error
	InvalidateAllHolidayCache(ctx context.Context) error
}

// dutyCacheServiceImpl 值班管理缓存服务实现
type dutyCacheServiceImpl struct {
	base   *services.DutyService
	cache  systemServices.CacheProvider
	config *services.CacheConfigService
}

// NewDutyServiceWithCache 创建带缓存的值班管理服务
func NewDutyServiceWithCache(
	db *gorm.DB,
	cache systemServices.CacheProvider,
	config *services.CacheConfigService,
) DutyCacheService {
	base := services.NewDutyService(db)
	return &dutyCacheServiceImpl{
		base:   base,
		cache:  cache,
		config: config,
	}
}

// getExpiration 获取缓存过期时间
func (s *dutyCacheServiceImpl) getExpiration(configKey string, defaultVal time.Duration) time.Duration {
	if s.config != nil {
		return s.config.GetDurationWithDefault(configKey, defaultVal)
	}
	return defaultVal
}

// ==================== 值班池管理（不缓存，参数多变） ====================

func (s *dutyCacheServiceImpl) CreateDutyPool(ctx context.Context, req *services.DutyPoolCreateRequest, creatorID string) (*models.DutyPool, error) {
	return s.base.CreateDutyPool(ctx, req, creatorID)
}

func (s *dutyCacheServiceImpl) GetDutyPoolList(ctx context.Context, req *services.DutyPoolListRequest) ([]models.DutyPool, int64, error) {
	return s.base.GetDutyPoolList(ctx, req)
}

// GetDutyPoolStatistics 统计值班池(总数/启停/成员数),委托给基础服务。不缓存。
func (s *dutyCacheServiceImpl) GetDutyPoolStatistics(ctx context.Context) (*services.DutyPoolStatistics, error) {
	return s.base.GetDutyPoolStatistics(ctx)
}

func (s *dutyCacheServiceImpl) GetDutyPoolByID(ctx context.Context, poolID string) (*models.DutyPool, error) {
	return s.base.GetDutyPoolByID(ctx, poolID)
}

func (s *dutyCacheServiceImpl) UpdateDutyPool(ctx context.Context, req *services.DutyPoolUpdateRequest, updaterID string) error {
	return s.base.UpdateDutyPool(ctx, req, updaterID)
}

func (s *dutyCacheServiceImpl) DeleteDutyPool(ctx context.Context, poolID string) error {
	return s.base.DeleteDutyPool(ctx, poolID)
}

// ==================== 排班管理（部分缓存） ====================

func (s *dutyCacheServiceImpl) GenerateSchedule(ctx context.Context, req *services.GenerateScheduleRequest, creatorID string) (int, error) {
	count, err := s.base.GenerateSchedule(ctx, req, creatorID)
	if err != nil {
		return 0, err
	}
	// 清除相关月份的排班缓存
	if len(req.StartDate) >= 7 {
		year := parseInt(req.StartDate[:4])
		month := parseInt(req.StartDate[5:7])
		_ = s.InvalidateMonthlyScheduleCache(ctx, year, month)
	}
	return count, nil
}

func (s *dutyCacheServiceImpl) GetDutyScheduleList(ctx context.Context, req *services.DutyScheduleListRequest) ([]models.DutySchedule, int64, error) {
	return s.base.GetDutyScheduleList(ctx, req)
}

// GetTodayDuty 获取今日值班人员（带缓存）
func (s *dutyCacheServiceImpl) GetTodayDuty(ctx context.Context) ([]services.TodayDutyMember, error) {
	cacheKey := "duty:today"
	var result []services.TodayDutyMember

	expiration := s.getExpiration("cache.duty.today", 5*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.base.GetTodayDuty(ctx)
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetMonthlyDutySchedule 获取月度值班排班（带缓存）
func (s *dutyCacheServiceImpl) GetMonthlyDutySchedule(ctx context.Context, year int, month int) (map[string][]services.TodayDutyMember, error) {
	cacheKey := fmt.Sprintf("duty:monthly:%d:%d", year, month)
	var result map[string][]services.TodayDutyMember

	expiration := s.getExpiration("cache.duty.monthly", 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.base.GetMonthlyDutySchedule(ctx, year, month)
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *dutyCacheServiceImpl) SwapDuty(ctx context.Context, req *services.SwapDutyRequest, operatorID string) error {
	err := s.base.SwapDuty(ctx, req, operatorID)
	if err != nil {
		return err
	}
	// 清除今日值班缓存
	_ = s.InvalidateTodayDutyCache(ctx)
	return nil
}

func (s *dutyCacheServiceImpl) ManualDuty(ctx context.Context, req *services.ManualDutyRequest, creatorID string) error {
	err := s.base.ManualDuty(ctx, req, creatorID)
	if err != nil {
		return err
	}
	// 解析日期获取年月
	if len(req.DutyDate) >= 7 {
		year := req.DutyDate[:4]
		month := req.DutyDate[5:7]
		_ = s.InvalidateMonthlyScheduleCache(ctx, parseInt(year), parseInt(month))
		_ = s.InvalidateTodayDutyCache(ctx)
	}
	return nil
}

func (s *dutyCacheServiceImpl) DeleteDutySchedule(ctx context.Context, scheduleID string) error {
	err := s.base.DeleteDutySchedule(ctx, scheduleID)
	if err != nil {
		return err
	}
	// 清除所有排班缓存
	_ = s.InvalidateAllScheduleCache(ctx)
	return nil
}

func (s *dutyCacheServiceImpl) BatchDeleteDutySchedules(ctx context.Context, scheduleIDs []string) error {
	err := s.base.BatchDeleteDutySchedules(ctx, scheduleIDs)
	if err != nil {
		return err
	}
	// 清除所有排班缓存
	_ = s.InvalidateAllScheduleCache(ctx)
	return nil
}

// ==================== 我的值班统计（不缓存） ====================

func (s *dutyCacheServiceImpl) GetMyDutyStats(ctx context.Context, userID string) (*services.MyDutyStats, error) {
	return s.base.GetMyDutyStats(ctx, userID)
}

// ==================== 节假日管理（带缓存） ====================

// CreateHoliday 创建节假日（带缓存失效）
func (s *dutyCacheServiceImpl) CreateHoliday(ctx context.Context, holiday *models.Holiday, creatorID string) error {
	err := s.base.CreateHoliday(ctx, holiday, creatorID)
	if err != nil {
		return err
	}
	// 清除该年份的节假日缓存
	return s.InvalidateHolidayCache(ctx, holiday.Year)
}

// GetHolidayList 获取节假日列表（带缓存）
func (s *dutyCacheServiceImpl) GetHolidayList(ctx context.Context, year int) ([]models.Holiday, error) {
	cacheKey := fmt.Sprintf("duty:holidays:%d", year)
	var result []models.Holiday

	expiration := s.getExpiration("cache.duty.holidays", 60*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.base.GetHolidayList(ctx, year)
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *dutyCacheServiceImpl) UpdateHoliday(ctx context.Context, holiday *models.Holiday, updaterID string) error {
	err := s.base.UpdateHoliday(ctx, holiday, updaterID)
	if err != nil {
		return err
	}
	// 清除该年份的节假日缓存
	return s.InvalidateHolidayCache(ctx, holiday.Year)
}

func (s *dutyCacheServiceImpl) DeleteHoliday(ctx context.Context, holidayID string) error {
	err := s.base.DeleteHoliday(ctx, holidayID)
	if err != nil {
		return err
	}
	// 清除所有节假日缓存
	return s.InvalidateAllHolidayCache(ctx)
}

func (s *dutyCacheServiceImpl) GetHolidayYears(ctx context.Context) ([]int, error) {
	return s.base.GetHolidayYears(ctx)
}

func (s *dutyCacheServiceImpl) BatchCreateHolidays(ctx context.Context, holidays []models.Holiday, creatorID string) error {
	err := s.base.BatchCreateHolidays(ctx, holidays, creatorID)
	if err != nil {
		return err
	}
	// 清除所有涉及年份的节假日缓存
	yearMap := make(map[int]bool)
	for _, holiday := range holidays {
		yearMap[holiday.Year] = true
	}
	for year := range yearMap {
		_ = s.InvalidateHolidayCache(ctx, year)
	}
	return nil
}

// ==================== 值班配置管理（不缓存） ====================

func (s *dutyCacheServiceImpl) GetDutyConfig(ctx context.Context) (*models.DutyConfig, error) {
	return s.base.GetDutyConfig(ctx)
}

func (s *dutyCacheServiceImpl) UpdateDutyConfig(ctx context.Context, config *models.DutyConfig, updaterID string) error {
	return s.base.UpdateDutyConfig(ctx, config, updaterID)
}

// ==================== 缓存失效方法 ====================

// InvalidateTodayDutyCache 失效今日值班缓存
func (s *dutyCacheServiceImpl) InvalidateTodayDutyCache(ctx context.Context) error {
	keys := []string{"duty:today"}
	systemServices.InvalidateCacheByKey(ctx, s.cache, keys, "DUTY")
	return nil
}

// InvalidateMonthlyScheduleCache 失效指定月份排班缓存
func (s *dutyCacheServiceImpl) InvalidateMonthlyScheduleCache(ctx context.Context, year, month int) error {
	keys := []string{fmt.Sprintf("duty:monthly:%d:%d", year, month)}
	systemServices.InvalidateCacheByKey(ctx, s.cache, keys, "DUTY")
	return nil
}

// InvalidateAllScheduleCache 失效所有排班缓存
func (s *dutyCacheServiceImpl) InvalidateAllScheduleCache(ctx context.Context) error {
	keys := []string{"duty:*"}
	systemServices.InvalidateCacheByPattern(ctx, s.cache, keys, "DUTY")
	return nil
}

// InvalidateHolidayCache 失效指定年份节假日缓存
func (s *dutyCacheServiceImpl) InvalidateHolidayCache(ctx context.Context, year int) error {
	keys := []string{fmt.Sprintf("duty:holidays:%d", year)}
	systemServices.InvalidateCacheByKey(ctx, s.cache, keys, "DUTY")
	return nil
}

// InvalidateAllHolidayCache 失效所有节假日缓存
func (s *dutyCacheServiceImpl) InvalidateAllHolidayCache(ctx context.Context) error {
	keys := []string{"duty:holidays:*"}
	systemServices.InvalidateCacheByPattern(ctx, s.cache, keys, "DUTY")
	return nil
}

// parseInt 辅助函数：安全地将字符串解析为整数
func parseInt(s string) int {
	var result int
	if len(s) >= 4 {
		for i := 0; i < len(s) && i < 4; i++ {
			if s[i] >= '0' && s[i] <= '9' {
				result = result*10 + int(s[i]-'0')
			}
		}
	}
	return result
}
