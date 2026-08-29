package services

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// DutyScheduleService 排班管理服务
type DutyScheduleService struct {
	db *gorm.DB
}

// NewDutyScheduleService 创建排班管理服务
func NewDutyScheduleService(db *gorm.DB) *DutyScheduleService {
	return &DutyScheduleService{db: db}
}

// ==================== 排班管理 ====================

// GenerateSchedule 生成排班
func (s *DutyScheduleService) GenerateSchedule(ctx context.Context, req *GenerateScheduleRequest, creatorID string) (int, error) {
	// 获取值班池
	poolService := NewDutyPoolService(s.db)
	pool, err := poolService.GetDutyPoolByID(ctx, req.PoolID)
	if err != nil {
		return 0, err
	}

	if len(pool.Members) == 0 {
		return 0, fmt.Errorf("值班池没有成员")
	}

	// 解析日期
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return 0, fmt.Errorf("开始日期格式错误: %w", err)
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return 0, fmt.Errorf("结束日期格式错误: %w", err)
	}

	// 获取节假日列表
	var holidays []models.Holiday
	if queryErr := s.db.WithContext(ctx).
		Where("holiday_date >= ? AND holiday_date <= ?", startDate, endDate).
		Find(&holidays).Error; queryErr != nil {
		return 0, fmt.Errorf("获取节假日失败: %w", queryErr)
	}

	// 构建节假日映射
	holidayMap := make(map[string]models.Holiday)
	for _, h := range holidays {
		dateStr := h.HolidayDate.Format("2006-01-02")
		holidayMap[dateStr] = h
	}

	var scheduleCount int

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 如果需要清除已有排班
		if req.ClearExists {
			if deleteErr := tx.Where("pool_id = ? AND schedule_date >= ? AND schedule_date <= ?",
				req.PoolID, startDate, endDate).
				Delete(&models.DutySchedule{}).Error; deleteErr != nil {
				return fmt.Errorf("清除已有排班失败: %w", deleteErr)
			}
		}

		// 生成排班
		memberCount := len(pool.Members)
		memberIndex := 0
		var schedules []models.DutySchedule

		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			// 判断当天实际的日期类型
			actualDutyType := s.getDutyType(d, holidayMap)

			// 只对符合用户选择的值班类型的日期进行排班
			if actualDutyType != req.DutyType {
				continue
			}

			// 轮询选择值班人员
			for i := 0; i < pool.DailyCount; i++ {
				member := pool.Members[memberIndex%memberCount]
				memberIndex++

				schedule := models.DutySchedule{
					BaseModel:    models.BaseModel{CreatedBy: creatorID},
					PoolID:       req.PoolID,
					UserID:       member.UserID,
					ScheduleDate: d,
					DutyType:     models.ScheduleMode(actualDutyType),
					Status:       models.DutyStatusNormal,
					IsManual:     false,
				}

				schedules = append(schedules, schedule)
			}
		}

		// 批量插入排班记录
		if len(schedules) > 0 {
			if createErr := tx.Create(&schedules).Error; createErr != nil {
				return fmt.Errorf("保存排班记录失败: %w", createErr)
			}
		}

		scheduleCount = len(schedules)
		return nil
	})

	return scheduleCount, err
}

// getDutyType 判断日期类型
func (s *DutyScheduleService) getDutyType(date time.Time, holidayMap map[string]models.Holiday) string {
	dateStr := date.Format("2006-01-02")

	// 检查是否为节假日
	if h, exists := holidayMap[dateStr]; exists {
		if h.IsOffday {
			return string(models.ScheduleModeHoliday)
		}
	}

	// 检查是否为周末
	if isWeekend(date) {
		return string(models.ScheduleModeWeekend)
	}

	return string(models.ScheduleModeWeekday)
}

// isWeekend 判断是否为周末
func isWeekend(date time.Time) bool {
	weekday := date.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

// dutyScheduleAllowedSortFields 排班列表可排序字段白名单（对应 sys_duty_schedule 表列名）。
var dutyScheduleAllowedSortFields = map[string]string{
	"scheduleDate": "schedule_date",
	"dutyType":     "duty_type",
	"status":       "status",
}

// GetDutyScheduleList 获取排班列表
func (s *DutyScheduleService) GetDutyScheduleList(ctx context.Context, req *DutyScheduleListRequest) ([]models.DutySchedule, int64, error) {
	var schedules []models.DutySchedule
	var total int64

	query := s.db.WithContext(ctx).Model(&models.DutySchedule{})

	// 筛选条件
	if req.PoolID != nil && *req.PoolID != "" {
		query = query.Where("pool_id = ?", *req.PoolID)
	}
	if req.UserID != nil && *req.UserID != "" {
		query = query.Where("user_id = ?", *req.UserID)
	}
	if req.StartDate != nil && *req.StartDate != "" {
		if startDate, err := time.Parse("2006-01-02", *req.StartDate); err == nil {
			query = query.Where("schedule_date >= ?", startDate)
		}
	}
	if req.EndDate != nil && *req.EndDate != "" {
		if endDate, err := time.Parse("2006-01-02", *req.EndDate); err == nil {
			query = query.Where("schedule_date <= ?", endDate.AddDate(0, 0, 1))
		}
	}
	if req.DutyType != nil && *req.DutyType != "" {
		query = query.Where("duty_type = ?", *req.DutyType)
	}
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}
	// 过期状态筛选：0=未过期，1=已过期
	if req.Expired != nil {
		today := time.Now().Format("2006-01-02")
		if *req.Expired == 0 {
			// 未过期：排班日期 >= 今天
			query = query.Where("schedule_date >= ?", today)
		} else if *req.Expired == 1 {
			// 已过期：排班日期 < 今天
			query = query.Where("schedule_date < ?", today)
		}
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计排班数量失败: %w", err)
	}

	// 分页查询
	if req.Current == 0 {
		req.Current = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	offset := (req.Current - 1) * req.PageSize
	// 用户排序（白名单）优先，无 OrderByColumn 时保留 schedule_date ASC 默认
	query = base.ApplySort(query, req.BaseListRequest, dutyScheduleAllowedSortFields)
	if req.OrderByColumn == "" {
		query = query.Order("schedule_date ASC")
	}
	if err := query.Preload("Pool").Preload("User").
		Offset(offset).Limit(req.PageSize).
		Find(&schedules).Error; err != nil {
		return nil, 0, fmt.Errorf("查询排班列表失败: %w", err)
	}

	return schedules, total, nil
}

// GetTodayDuty 获取今日值班人员
func (s *DutyScheduleService) GetTodayDuty(ctx context.Context) ([]TodayDutyMember, error) {
	// 使用本地时间获取今天的日期，与数据库 TimeZone=Asia/Shanghai 配置一致
	today := time.Now().Local().Format("2006-01-02")

	var schedules []models.DutySchedule
	if err := s.db.WithContext(ctx).
		Preload("Pool").
		Preload("User").
		Where("DATE(schedule_date) = ? AND status = ?", today, models.DutyStatusNormal).
		Find(&schedules).Error; err != nil {
		return nil, fmt.Errorf("查询今日值班失败: %w", err)
	}

	if len(schedules) == 0 {
		return nil, fmt.Errorf("今日无值班人员")
	}

	var members []TodayDutyMember
	for _, s := range schedules {
		poolName := ""
		if s.Pool != nil {
			poolName = s.Pool.PoolName
		}

		username := ""
		phone := ""
		if s.User != nil {
			// 统一显示格式：昵称（用户名）
			if s.User.Nickname != nil && *s.User.Nickname != "" && *s.User.Nickname != s.User.Username {
				username = fmt.Sprintf("%s (%s)", *s.User.Nickname, s.User.Username)
			} else {
				username = s.User.Username
			}
			if s.User.Phone != nil {
				phone = *s.User.Phone
			}
		}

		members = append(members, TodayDutyMember{
			ScheduleID: s.ID,
			PoolID:     s.PoolID,
			PoolName:   poolName,
			UserID:     s.UserID,
			Username:   username,
			Phone:      phone,
			DutyType:   string(s.DutyType),
		})
	}

	return members, nil
}

// GetMonthlyDutySchedule 获取月度值班排班
func (s *DutyScheduleService) GetMonthlyDutySchedule(ctx context.Context, year int, month int) (map[string][]TodayDutyMember, error) {
	// 计算月份的起止日期
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endDate := startDate.AddDate(0, 1, 0).Add(-time.Second)

	var schedules []models.DutySchedule
	if err := s.db.WithContext(ctx).
		Preload("Pool").
		Preload("User").
		Where("schedule_date >= ? AND schedule_date <= ? AND status = ? AND deleted_at IS NULL", startDate, endDate, models.DutyStatusNormal).
		Order("schedule_date ASC").
		Find(&schedules).Error; err != nil {
		return nil, fmt.Errorf("查询月度值班失败: %w", err)
	}

	result := make(map[string][]TodayDutyMember)
	for _, s := range schedules {
		dateStr := s.ScheduleDate.Format("2006-01-02")

		poolName := ""
		if s.Pool != nil {
			poolName = s.Pool.PoolName
		}

		username := ""
		phone := ""
		if s.User != nil {
			// 统一显示格式：昵称（用户名）
			if s.User.Nickname != nil && *s.User.Nickname != "" && *s.User.Nickname != s.User.Username {
				username = fmt.Sprintf("%s (%s)", *s.User.Nickname, s.User.Username)
			} else {
				username = s.User.Username
			}
			if s.User.Phone != nil {
				phone = *s.User.Phone
			}
		}

		result[dateStr] = append(result[dateStr], TodayDutyMember{
			ScheduleID: s.ID,
			PoolID:     s.PoolID,
			PoolName:   poolName,
			UserID:     s.UserID,
			Username:   username,
			Phone:      phone,
			DutyType:   string(s.DutyType),
		})
	}

	return result, nil
}

// ==================== 调班管理 ====================

// SwapDuty 调班
func (s *DutyScheduleService) SwapDuty(ctx context.Context, req *SwapDutyRequest, operatorID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取两个排班记录
		var fromSchedule, toSchedule models.DutySchedule
		if err := tx.Where("id = ?", req.FromScheduleID).First(&fromSchedule).Error; err != nil {
			return fmt.Errorf("原排班记录不存在")
		}
		if err := tx.Where("id = ?", req.ToScheduleID).First(&toSchedule).Error; err != nil {
			return fmt.Errorf("目标排班记录不存在")
		}

		// 交换值班人员ID
		fromUserID := fromSchedule.UserID
		toUserID := toSchedule.UserID

		// 更新排班记录
		fromSchedule.UserID = toUserID
		fromSchedule.Status = models.DutyStatusExchanged
		fromSchedule.IsManual = true
		fromSchedule.SwapFromDate = &fromSchedule.ScheduleDate
		fromSchedule.SwapReason = req.Reason
		fromSchedule.UpdatedBy = operatorID

		toSchedule.UserID = fromUserID
		toSchedule.Status = models.DutyStatusExchanged
		toSchedule.IsManual = true
		toSchedule.SwapFromDate = &toSchedule.ScheduleDate
		toSchedule.SwapReason = req.Reason
		toSchedule.UpdatedBy = operatorID

		if err := tx.Save(&fromSchedule).Error; err != nil {
			return fmt.Errorf("更新原排班记录失败: %w", err)
		}
		if err := tx.Save(&toSchedule).Error; err != nil {
			return fmt.Errorf("更新目标排班记录失败: %w", err)
		}

		// 记录调班历史
		exchange := &models.DutyExchange{
			ScheduleID:     req.FromScheduleID,
			OriginalUserID: fromUserID,
			NewUserID:      toUserID,
			ExchangeDate:   time.Now(),
			Reason:         req.Reason,
			CreatedBy:      operatorID,
		}
		if err := tx.Create(exchange).Error; err != nil {
			return fmt.Errorf("记录调班历史失败: %w", err)
		}

		return nil
	})
}

// ManualDuty 手动排班
func (s *DutyScheduleService) ManualDuty(ctx context.Context, req *ManualDutyRequest, creatorID string) error {
	// 解析日期
	dutyDate, err := time.Parse("2006-01-02", req.DutyDate)
	if err != nil {
		return fmt.Errorf("值班日期格式错误: %w", err)
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除该日期的已有排班（如果存在）
		tx.Where("pool_id = ? AND schedule_date = ?", req.PoolID, dutyDate).Delete(&models.DutySchedule{})

		// 创建新的排班记录
		for _, userID := range req.UserIDs {
			schedule := models.DutySchedule{
				BaseModel:    models.BaseModel{CreatedBy: creatorID},
				PoolID:       req.PoolID,
				UserID:       userID,
				ScheduleDate: dutyDate,
				DutyType:     models.ScheduleMode(req.DutyType),
				Status:       models.DutyStatusNormal,
				IsManual:     true,
				SwapReason:   req.Reason,
			}

			if err := tx.Create(&schedule).Error; err != nil {
				return fmt.Errorf("创建排班记录失败: %w", err)
			}
		}

		return nil
	})
}

// DeleteDutySchedule 删除排班记录
func (s *DutyScheduleService) DeleteDutySchedule(ctx context.Context, scheduleID string) error {
	if err := s.db.WithContext(ctx).Delete(&models.DutySchedule{}, "id = ?", scheduleID).Error; err != nil {
		return fmt.Errorf("删除排班记录失败: %w", err)
	}
	return nil
}

// BatchDeleteDutySchedules 批量删除排班记录
func (s *DutyScheduleService) BatchDeleteDutySchedules(ctx context.Context, scheduleIDs []string) error {
	if err := s.db.WithContext(ctx).Delete(&models.DutySchedule{}, scheduleIDs).Error; err != nil {
		return fmt.Errorf("批量删除排班记录失败: %w", err)
	}
	return nil
}
