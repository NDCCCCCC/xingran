package services

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// DutyService 值班管理服务
type DutyService struct {
	db *gorm.DB
}

// NewDutyService 创建值班管理服务
func NewDutyService(db *gorm.DB) *DutyService {
	return &DutyService{db: db}
}

// ==================== 请求/响应结构 ====================

// DutyPoolListRequest 值班池列表请求
type DutyPoolListRequest struct {
	PoolName *string `json:"poolName,omitempty"`
	DeptID   *string `json:"deptId,omitempty"`
	Status   *int    `json:"status,omitempty"`
	Current  int     `json:"current"`
	PageSize int     `json:"pageSize"`
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
	ID          string   `json:"id" binding:"required"`
	PoolName    string   `json:"poolName" binding:"required,max=100"`
	DeptID      *string  `json:"deptId,omitempty"`
	Description string   `json:"description,omitempty"`
	DailyCount  int      `json:"dailyCount" binding:"min=1,max=10"`
	Status      int      `json:"status" binding:"min=0,max=1"`
	MemberIDs   []string `json:"memberIds" binding:"required,min=1"`
}

// DutyScheduleListRequest 排班列表请求
type DutyScheduleListRequest struct {
	PoolID    *string `json:"poolId,omitempty"`
	UserID    *string `json:"userId,omitempty"`
	StartDate *string `json:"startDate,omitempty"` // YYYY-MM-DD
	EndDate   *string `json:"endDate,omitempty"`   // YYYY-MM-DD
	DutyType  *string `json:"dutyType,omitempty"`
	Status    *int    `json:"status,omitempty"`
	Current   int     `json:"current"`
	PageSize  int     `json:"pageSize"`
}

// GenerateScheduleRequest 生成排班请求
type GenerateScheduleRequest struct {
	PoolID      string `json:"poolId" binding:"required"`
	StartDate   string `json:"startDate" binding:"required"` // YYYY-MM-DD
	EndDate     string `json:"endDate" binding:"required"`   // YYYY-MM-DD
	DutyType    string `json:"dutyType" binding:"required,oneof=weekday weekend holiday"` // 值班类型：工作日/周末/节假日
	ClearExists bool   `json:"clearExists"` // 是否清除已有排班
}

// SwapDutyRequest 调班请求
type SwapDutyRequest struct {
	FromScheduleID string `json:"fromScheduleId" binding:"required"` // 原排班ID
	ToScheduleID   string `json:"toScheduleId" binding:"required"`   // 目标排班ID
	Reason         string `json:"reason"`
}

// ManualDutyRequest 手动排班请求
type ManualDutyRequest struct {
	PoolID    string   `json:"poolId" binding:"required"`
	DutyDate  string   `json:"dutyDate" binding:"required"`    // YYYY-MM-DD
	UserIDs   []string `json:"userIds" binding:"required,min=1"`
	DutyType  string   `json:"dutyType" binding:"required,oneof=weekday weekend holiday"`
	Reason    string   `json:"reason"`
}

// ==================== 值班池管理 ====================

// CreateDutyPool 创建值班池
func (s *DutyService) CreateDutyPool(ctx context.Context, req *DutyPoolCreateRequest, creatorID string) (*models.DutyPool, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查值班池名称是否已存在
	var count int64
	if err := tx.Model(&models.DutyPool{}).Where("pool_name = ?", req.PoolName).Count(&count).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("检查值班池名称失败: %w", err)
	}
	if count > 0 {
		tx.Rollback()
		return nil, fmt.Errorf("值班池名称已存在")
	}

	// 创建值班池
	pool := &models.DutyPool{
		BaseModel:   models.BaseModel{CreatedBy: creatorID},
		PoolName:    req.PoolName,
		DeptID:      req.DeptID,
		Description: req.Description,
		Status:      models.DutyPoolStatusEnabled,
		DailyCount:  req.DailyCount,
	}

	if err := tx.Create(pool).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建值班池失败: %w", err)
	}

	// 添加成员
	if len(req.MemberIDs) > 0 {
		var members []models.DutyPoolMember
		for i, userID := range req.MemberIDs {
			// 验证用户是否存在
			var user models.User
			if err := tx.Where("id = ?", userID).First(&user).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("用户不存在: %s", userID)
			}

			members = append(members, models.DutyPoolMember{
				PoolID:      pool.ID,
				UserID:      userID,
				MemberOrder: i,
			})
		}
		if err := tx.Create(&members).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("添加值班池成员失败: %w", err)
		}
	}

	tx.Commit()

	// 重新加载完整的值班池信息
	if err := s.db.WithContext(ctx).Preload("Members.User").Preload("Dept").Where("id = ?", pool.ID).First(pool).Error; err != nil {
		return nil, fmt.Errorf("查询值班池失败: %w", err)
	}

	return pool, nil
}

// GetDutyPoolList 获取值班池列表
func (s *DutyService) GetDutyPoolList(ctx context.Context, req *DutyPoolListRequest) ([]models.DutyPool, int64, error) {
	var pools []models.DutyPool
	var total int64

	query := s.db.WithContext(ctx).Model(&models.DutyPool{})

	// 筛选条件
	if req.PoolName != nil && *req.PoolName != "" {
		query = query.Where("pool_name LIKE ?", "%"+*req.PoolName+"%")
	}
	if req.DeptID != nil && *req.DeptID != "" {
		query = query.Where("dept_id = ?", *req.DeptID)
	}
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计值班池数量失败: %w", err)
	}

	// 分页查询
	if req.Current == 0 {
		req.Current = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	offset := (req.Current - 1) * req.PageSize
	if err := query.Preload("Members.User").Preload("Dept").
		Order("created_at DESC").
		Offset(offset).Limit(req.PageSize).
		Find(&pools).Error; err != nil {
		return nil, 0, fmt.Errorf("查询值班池列表失败: %w", err)
	}

	return pools, total, nil
}

// GetDutyPoolByID 根据ID获取值班池
func (s *DutyService) GetDutyPoolByID(ctx context.Context, poolID string) (*models.DutyPool, error) {
	var pool models.DutyPool
	if err := s.db.WithContext(ctx).Preload("Members.User").Preload("Dept").
		Where("id = ?", poolID).First(&pool).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("值班池不存在")
		}
		return nil, fmt.Errorf("查询值班池失败: %w", err)
	}
	return &pool, nil
}

// UpdateDutyPool 更新值班池
func (s *DutyService) UpdateDutyPool(ctx context.Context, req *DutyPoolUpdateRequest, updaterID string) error {
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查值班池是否存在
	var pool models.DutyPool
	if err := tx.Where("id = ?", req.ID).First(&pool).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("值班池不存在")
	}

	// 检查名称是否与其他池重复
	var count int64
	if err := tx.Model(&models.DutyPool{}).Where("id != ? AND pool_name = ?", req.ID, req.PoolName).Count(&count).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("检查值班池名称失败: %w", err)
	}
	if count > 0 {
		tx.Rollback()
		return fmt.Errorf("值班池名称已存在")
	}

	// 更新值班池基本信息
	updates := map[string]interface{}{
		"pool_name":   req.PoolName,
		"dept_id":     req.DeptID,
		"description": req.Description,
		"daily_count": req.DailyCount,
		"status":      req.Status,
		"updated_by":  updaterID,
	}

	if err := tx.Model(&pool).Updates(updates).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新值班池失败: %w", err)
	}

	// 更新成员：先删除旧成员，再添加新成员
	if err := tx.Where("pool_id = ?", req.ID).Delete(&models.DutyPoolMember{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除旧成员失败: %w", err)
	}

	if len(req.MemberIDs) > 0 {
		var members []models.DutyPoolMember
		for i, userID := range req.MemberIDs {
			members = append(members, models.DutyPoolMember{
				PoolID:      req.ID,
				UserID:      userID,
				MemberOrder: i,
			})
		}
		if err := tx.Create(&members).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("添加值班池成员失败: %w", err)
		}
	}

	tx.Commit()
	return nil
}

// DeleteDutyPool 删除值班池
func (s *DutyService) DeleteDutyPool(ctx context.Context, poolID string) error {
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查是否有相关排班记录
	var count int64
	if err := tx.Model(&models.DutySchedule{}).Where("pool_id = ?", poolID).Count(&count).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("检查排班记录失败: %w", err)
	}
	if count > 0 {
		tx.Rollback()
		return fmt.Errorf("该值班池存在排班记录，无法删除")
	}

	// 删除值班池成员
	if err := tx.Where("pool_id = ?", poolID).Delete(&models.DutyPoolMember{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除值班池成员失败: %w", err)
	}

	// 删除值班池
	if err := tx.Delete(&models.DutyPool{}, "id = ?", poolID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除值班池失败: %w", err)
	}

	tx.Commit()
	return nil
}

// ==================== 排班管理 ====================

// GenerateSchedule 生成排班
func (s *DutyService) GenerateSchedule(ctx context.Context, req *GenerateScheduleRequest, creatorID string) (int, error) {
	// 获取值班池
	pool, err := s.GetDutyPoolByID(ctx, req.PoolID)
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
	if err := s.db.WithContext(ctx).
		Where("holiday_date >= ? AND holiday_date <= ?", startDate, endDate).
		Find(&holidays).Error; err != nil {
		return 0, fmt.Errorf("获取节假日失败: %w", err)
	}

	// 构建节假日映射
	holidayMap := make(map[string]models.Holiday)
	for _, h := range holidays {
		dateStr := h.HolidayDate.Format("2006-01-02")
		holidayMap[dateStr] = h
	}

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 如果需要清除已有排班
	if req.ClearExists {
		if err := tx.Where("pool_id = ? AND schedule_date >= ? AND schedule_date <= ?",
			req.PoolID, startDate, endDate).
			Delete(&models.DutySchedule{}).Error; err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("清除已有排班失败: %w", err)
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
				BaseModel:     models.BaseModel{CreatedBy: creatorID},
				PoolID:        req.PoolID,
				UserID:        member.UserID,
				ScheduleDate:  d,
				DutyType:      models.ScheduleMode(actualDutyType),
				Status:        models.DutyStatusNormal,
				IsManual:      false,
			}

			schedules = append(schedules, schedule)
		}
	}

	// 批量插入排班记录
	if len(schedules) > 0 {
		if err := tx.Create(&schedules).Error; err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("保存排班记录失败: %w", err)
		}
	}

	tx.Commit()
	return len(schedules), nil
}

// GetDutyType 判断日期类型
func (s *DutyService) getDutyType(date time.Time, holidayMap map[string]models.Holiday) string {
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

// GetDutyScheduleList 获取排班列表
func (s *DutyService) GetDutyScheduleList(ctx context.Context, req *DutyScheduleListRequest) ([]models.DutySchedule, int64, error) {
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
			query = query.Where("schedule_date <= ?", endDate)
		}
	}
	if req.DutyType != nil && *req.DutyType != "" {
		query = query.Where("duty_type = ?", *req.DutyType)
	}
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
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
	if err := query.Preload("Pool").Preload("User").
		Order("schedule_date ASC").
		Offset(offset).Limit(req.PageSize).
		Find(&schedules).Error; err != nil {
		return nil, 0, fmt.Errorf("查询排班列表失败: %w", err)
	}

	return schedules, total, nil
}

// GetTodayDuty 获取今日值班人员
func (s *DutyService) GetTodayDuty(ctx context.Context) ([]TodayDutyMember, error) {
	today := time.Now().Truncate(24 * time.Hour)

	var schedules []models.DutySchedule
	if err := s.db.WithContext(ctx).
		Preload("Pool").
		Preload("User").
		Where("schedule_date = ? AND status = ?", today, models.DutyStatusNormal).
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
			username = s.User.Username
			if s.User.Nickname != nil && *s.User.Nickname != "" {
				username = *s.User.Nickname
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

// GetMonthlyDutySchedule 获取月度值班排班
func (s *DutyService) GetMonthlyDutySchedule(ctx context.Context, year int, month int) (map[string][]TodayDutyMember, error) {
	// 计算月份的起止日期
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endDate := startDate.AddDate(0, 1, 0).Add(-time.Second)

	var schedules []models.DutySchedule
	if err := s.db.WithContext(ctx).
		Preload("Pool").
		Preload("User").
		Where("schedule_date >= ? AND schedule_date <= ? AND status = ?", startDate, endDate, models.DutyStatusNormal).
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
			username = s.User.Username
			if s.User.Nickname != nil && *s.User.Nickname != "" {
				username = *s.User.Nickname
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
func (s *DutyService) SwapDuty(ctx context.Context, req *SwapDutyRequest, operatorID string) error {
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取两个排班记录
	var fromSchedule, toSchedule models.DutySchedule
	if err := tx.Where("id = ?", req.FromScheduleID).First(&fromSchedule).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("原排班记录不存在")
	}
	if err := tx.Where("id = ?", req.ToScheduleID).First(&toSchedule).Error; err != nil {
		tx.Rollback()
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
		tx.Rollback()
		return fmt.Errorf("更新原排班记录失败: %w", err)
	}
	if err := tx.Save(&toSchedule).Error; err != nil {
		tx.Rollback()
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
		tx.Rollback()
		return fmt.Errorf("记录调班历史失败: %w", err)
	}

	tx.Commit()
	return nil
}

// ManualDuty 手动排班
func (s *DutyService) ManualDuty(ctx context.Context, req *ManualDutyRequest, creatorID string) error {
	// 解析日期
	dutyDate, err := time.Parse("2006-01-02", req.DutyDate)
	if err != nil {
		return fmt.Errorf("值班日期格式错误: %w", err)
	}

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除该日期的已有排班（如果存在）
	tx.Where("pool_id = ? AND schedule_date = ?", req.PoolID, dutyDate).Delete(&models.DutySchedule{})

	// 创建新的排班记录
	for _, userID := range req.UserIDs {
		schedule := models.DutySchedule{
			BaseModel:     models.BaseModel{CreatedBy: creatorID},
			PoolID:        req.PoolID,
			UserID:        userID,
			ScheduleDate:  dutyDate,
			DutyType:      models.ScheduleMode(req.DutyType),
			Status:        models.DutyStatusNormal,
			IsManual:      true,
			SwapReason:    req.Reason,
		}

		if err := tx.Create(&schedule).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("创建排班记录失败: %w", err)
		}
	}

	tx.Commit()
	return nil
}

// DeleteDutySchedule 删除排班记录
func (s *DutyService) DeleteDutySchedule(ctx context.Context, scheduleID string) error {
	if err := s.db.WithContext(ctx).Delete(&models.DutySchedule{}, "id = ?", scheduleID).Error; err != nil {
		return fmt.Errorf("删除排班记录失败: %w", err)
	}
	return nil
}

// BatchDeleteDutySchedules 批量删除排班记录
func (s *DutyService) BatchDeleteDutySchedules(ctx context.Context, scheduleIDs []string) error {
	if err := s.db.WithContext(ctx).Delete(&models.DutySchedule{}, scheduleIDs).Error; err != nil {
		return fmt.Errorf("批量删除排班记录失败: %w", err)
	}
	return nil
}

// ==================== 我的值班统计 ====================

// MyDutyStats 我的值班统计信息
type MyDutyStats struct {
	IsOnDutyToday    bool              `json:"isOnDutyToday"`    // 今日是否值班
	TodayDutyRecords []TodayDutyMember `json:"todayDutyRecords,omitempty"` // 今日值班记录（如果今日值班）
	ThisMonthCount   int               `json:"thisMonthCount"`   // 本月值班次数
	TotalCount       int               `json:"totalCount"`       // 总值班次数
	NextDutyDate     *string           `json:"nextDutyDate,omitempty"`     // 下次值班时间（格式：YYYY-MM-DD）
	NextDutyPoolName *string           `json:"nextDutyPoolName,omitempty"` // 下次值班池名称
}

// GetMyDutyStats 获取当前用户的值班统计
func (s *DutyService) GetMyDutyStats(ctx context.Context, userID string) (*MyDutyStats, error) {
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

// ==================== 节假日管理 ====================

// CreateHoliday 创建节假日
func (s *DutyService) CreateHoliday(ctx context.Context, holiday *models.Holiday, creatorID string) error {
	holiday.CreatedBy = creatorID
	if err := s.db.WithContext(ctx).Create(holiday).Error; err != nil {
		return fmt.Errorf("创建节假日失败: %w", err)
	}
	return nil
}

// GetHolidayList 获取节假日列表
func (s *DutyService) GetHolidayList(ctx context.Context, year int) ([]models.Holiday, error) {
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
func (s *DutyService) UpdateHoliday(ctx context.Context, holiday *models.Holiday, updaterID string) error {
	holiday.UpdatedBy = updaterID
	if err := s.db.WithContext(ctx).Save(holiday).Error; err != nil {
		return fmt.Errorf("更新节假日失败: %w", err)
	}
	return nil
}

// DeleteHoliday 删除节假日
func (s *DutyService) DeleteHoliday(ctx context.Context, holidayID string) error {
	if err := s.db.WithContext(ctx).Delete(&models.Holiday{}, "id = ?", holidayID).Error; err != nil {
		return fmt.Errorf("删除节假日失败: %w", err)
	}
	return nil
}

// GetHolidayYears 获取所有有节假日数据的年份列表
func (s *DutyService) GetHolidayYears(ctx context.Context) ([]int, error) {
	var years []int

	if err := s.db.WithContext(ctx).
		Model(&models.Holiday{}).
		Distinct("year").
		Pluck("year", &years).Error; err != nil {
		return nil, fmt.Errorf("查询年份列表失败: %w", err)
	}

	// 降序排序（最近的年份在前）
	for i, j := 0, len(years)-1; i < j; i, j = i+1, j-1 {
		years[i], years[j] = years[j], years[i]
	}

	return years, nil
}

// BatchCreateHolidays 批量创建节假日
func (s *DutyService) BatchCreateHolidays(ctx context.Context, holidays []models.Holiday, creatorID string) error {
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for i := range holidays {
		holidays[i].CreatedBy = creatorID
	}

	if err := tx.Create(&holidays).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("批量创建节假日失败: %w", err)
	}

	tx.Commit()
	return nil
}

// ==================== 值班配置管理 ====================

// GetDutyConfig 获取值班配置（系统中只有一条配置记录）
func (s *DutyService) GetDutyConfig(ctx context.Context) (*models.DutyConfig, error) {
	var config models.DutyConfig
	err := s.db.WithContext(ctx).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果不存在，返回默认配置
			return &models.DutyConfig{
				ReminderEnabled:      true,
				ReminderTime:         "08:00",
				ReminderChannels:     "websocket",
				BeforeReminderMinutes: nil,
			}, nil
		}
		return nil, fmt.Errorf("查询值班配置失败: %w", err)
	}
	return &config, nil
}

// UpdateDutyConfig 更新值班配置
func (s *DutyService) UpdateDutyConfig(ctx context.Context, config *models.DutyConfig, updaterID string) error {
	// 检查是否已存在配置
	var existing models.DutyConfig
	err := s.db.WithContext(ctx).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// 不存在，创建新配置
		config.CreatedBy = updaterID
		config.UpdatedBy = updaterID
		if err := s.db.WithContext(ctx).Create(config).Error; err != nil {
			return fmt.Errorf("创建值班配置失败: %w", err)
		}
		return nil
	}

	if err != nil {
		return fmt.Errorf("查询值班配置失败: %w", err)
	}

	// 更新现有配置
	config.ID = existing.ID
	config.CreatedBy = existing.CreatedBy
	config.UpdatedBy = updaterID
	if err := s.db.WithContext(ctx).Save(config).Error; err != nil {
		return fmt.Errorf("更新值班配置失败: %w", err)
	}

	return nil
}
