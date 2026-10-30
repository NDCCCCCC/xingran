package services

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// DutyHolidayService 节假日管理服务
type DutyHolidayService struct {
	db *gorm.DB
}

// NewDutyHolidayService 创建节假日管理服务
func NewDutyHolidayService(db *gorm.DB) *DutyHolidayService {
	return &DutyHolidayService{db: db}
}

// CreateHoliday 创建节假日
func (s *DutyHolidayService) CreateHoliday(ctx context.Context, holiday *models.Holiday, creatorID string) error {
	holiday.CreatedBy = creatorID
	if err := s.db.WithContext(ctx).Create(holiday).Error; err != nil {
		return fmt.Errorf("创建节假日失败: %w", err)
	}
	return nil
}

// GetHolidayList 获取节假日列表
func (s *DutyHolidayService) GetHolidayList(ctx context.Context, year int) ([]models.Holiday, error) {
	var holidays []models.Holiday

	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	endDate := time.Date(year, 12, 31, 23, 59, 59, 0, time.Local)

	if err := s.db.WithContext(ctx).
		Where("holiday_date >= ? AND holiday_date <= ?", startDate, endDate).
		Order("holiday_date ASC").
		Find(&holidays).Error; err != nil {
		return nil, fmt.Errorf("查询节假日列表失败: %w", err)
	}

	return holidays, nil
}

// UpdateHoliday 更新节假日
func (s *DutyHolidayService) UpdateHoliday(ctx context.Context, holiday *models.Holiday, updaterID string) error {
	holiday.UpdatedBy = updaterID
	if err := s.db.WithContext(ctx).Save(holiday).Error; err != nil {
		return fmt.Errorf("更新节假日失败: %w", err)
	}
	return nil
}

// DeleteHoliday 删除节假日
func (s *DutyHolidayService) DeleteHoliday(ctx context.Context, holidayID string) error {
	if err := s.db.WithContext(ctx).Delete(&models.Holiday{}, "id = ?", holidayID).Error; err != nil {
		return fmt.Errorf("删除节假日失败: %w", err)
	}
	return nil
}

// GetHolidayYears 获取所有有节假日数据的年份列表
func (s *DutyHolidayService) GetHolidayYears(ctx context.Context) ([]int, error) {
	var years []int

	if err := s.db.WithContext(ctx).
		Model(&models.Holiday{}).
		Distinct("year").
		Pluck("year", &years).Error; err != nil {
		return nil, fmt.Errorf("查询年份列表失败: %w", err)
	}

	// 降序排序（最近的年份在前）
	sort.Sort(sort.Reverse(sort.IntSlice(years)))

	return years, nil
}

// BatchCreateHolidays 批量创建节假日
func (s *DutyHolidayService) BatchCreateHolidays(ctx context.Context, holidays []models.Holiday, creatorID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range holidays {
			holidays[i].CreatedBy = creatorID
		}

		if err := tx.Create(&holidays).Error; err != nil {
			return fmt.Errorf("批量创建节假日失败: %w", err)
		}

		return nil
	})
}
