package services

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// DutyStatsService 值班统计服务
type DutyStatsService struct {
	db *gorm.DB
}

// NewDutyStatsService 创建值班统计服务
func NewDutyStatsService(db *gorm.DB) *DutyStatsService {
	return &DutyStatsService{db: db}
}

// GetMyDutyStats 获取当前用户的值班统计
func (s *DutyStatsService) GetMyDutyStats(ctx context.Context, userID string) (*MyDutyStats, error) {
	today := time.Now().Truncate(24 * time.Hour)

	// 1. 检查今日是否值班
	var todaySchedules []models.DutySchedule
	if err := s.db.WithContext(ctx).
		Preload("Pool").
		Where("user_id = ? AND schedule_date = ? AND status = ?", userID, today, models.DutyStatusNormal).
		Find(&todaySchedules).Error; err != nil {
		return nil, fmt.Errorf("查询今日值班失败: %w", err)
	}

	isOnDutyToday := len(todaySchedules) > 0

	var todayRecords []TodayDutyMember
	if isOnDutyToday {
		for _, s := range todaySchedules {
			poolName := ""
			if s.Pool != nil {
				poolName = s.Pool.PoolName
			}
			todayRecords = append(todayRecords, TodayDutyMember{
				ScheduleID: s.ID,
				PoolID:     s.PoolID,
				PoolName:   poolName,
				UserID:     s.UserID,
				DutyType:   string(s.DutyType),
			})
		}
	}

	// 2. 统计本月值班次数
	currentYear, currentMonth, _ := today.Date()
	startOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.Local)

	var thisMonthCount int64
	if err := s.db.WithContext(ctx).Model(&models.DutySchedule{}).
		Where("user_id = ? AND schedule_date >= ? AND schedule_date <= ? AND status = ?",
			userID, startOfMonth, today, models.DutyStatusNormal).
		Count(&thisMonthCount).Error; err != nil {
		return nil, fmt.Errorf("统计本月值班失败: %w", err)
	}

	// 3. 统计总值班次数
	var totalCount int64
	if err := s.db.WithContext(ctx).Model(&models.DutySchedule{}).
		Where("user_id = ? AND status = ?", userID, models.DutyStatusNormal).
		Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("统计总值班失败: %w", err)
	}

	// 4. 查询下次值班时间
	var nextSchedule models.DutySchedule
	if err := s.db.WithContext(ctx).
		Preload("Pool").
		Where("user_id = ? AND schedule_date > ? AND status = ?", userID, today, models.DutyStatusNormal).
		Order("schedule_date ASC").
		First(&nextSchedule).Error; err != nil {
		// 没有后续排班不算错误
		nextSchedule = models.DutySchedule{}
	}

	var nextDutyDate, nextDutyPoolName *string
	if nextSchedule.ID != "" {
		dateStr := nextSchedule.ScheduleDate.Format("2006-01-02")
		nextDutyDate = &dateStr
		if nextSchedule.Pool != nil {
			poolName := nextSchedule.Pool.PoolName
			nextDutyPoolName = &poolName
		}
	}

	return &MyDutyStats{
		IsOnDutyToday:    isOnDutyToday,
		TodayDutyRecords: todayRecords,
		ThisMonthCount:   int(thisMonthCount),
		TotalCount:       int(totalCount),
		NextDutyDate:     nextDutyDate,
		NextDutyPoolName: nextDutyPoolName,
	}, nil
}
