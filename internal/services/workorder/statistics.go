package workorder

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Statistics 统计数据
type Statistics struct {
	Total        int64                  `json:"total"`
	Pending      int                    `json:"pending"`
	Processing   int                    `json:"processing"`
	Completed    int                    `json:"completed"`
	Closed       int                    `json:"closed"`
	Rejected     int                    `json:"rejected"`
	ByPriority   map[string]int         `json:"byPriority"`
	ByCategory   map[string]int         `json:"byCategory"`
	ByAssignee   []AssigneeStatistics   `json:"byAssignee"`
	ByDepartment []DepartmentStatistics `json:"byDepartment"`
	Trend        []TrendData            `json:"trend"`
}

// AssigneeStatistics 处理人统计
type AssigneeStatistics struct {
	AssigneeID     string  `json:"assigneeId"`
	AssigneeName   string  `json:"assigneeName"`
	TotalCount     int     `json:"totalCount"`
	PendingCount   int     `json:"pendingCount"`
	DoneCount      int     `json:"doneCount"`
	AvgProcessTime float64 `json:"avgProcessTime"`
}

// DepartmentStatistics 部门统计
type DepartmentStatistics struct {
	DeptID     string `json:"deptId"`
	DeptName   string `json:"deptName"`
	TotalCount int    `json:"totalCount"`
	DoneCount  int    `json:"doneCount"`
}

// TrendData 趋势数据
type TrendData struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// StatisticsService 统计服务
type StatisticsService struct {
	db *gorm.DB
}

// NewStatisticsService 创建统计服务
func NewStatisticsService(db *gorm.DB) *StatisticsService {
	return &StatisticsService{db: db}
}

// Get 获取工单统计数据
func (s *StatisticsService) Get(ctx context.Context) (*Statistics, error) {
	stats := &Statistics{
		ByPriority: make(map[string]int),
		ByCategory: make(map[string]int),
	}

	// 总工单数
	if err := s.db.WithContext(ctx).Model(&models.WorkOrder{}).Count(&stats.Total).Error; err != nil {
		return nil, fmt.Errorf("查询总工单数失败: %w", err)
	}

	// 按状态统计
	var statusResults []struct {
		Status int
		Count  int
	}
	if err := s.db.WithContext(ctx).Model(&models.WorkOrder{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&statusResults).Error; err != nil {
		return nil, fmt.Errorf("按状态统计失败: %w", err)
	}

	for _, r := range statusResults {
		switch r.Status {
		case int(models.WorkOrderStatusPending):
			stats.Pending = r.Count
		case int(models.WorkOrderStatusProcessing):
			stats.Processing = r.Count
		case int(models.WorkOrderStatusCompleted):
			stats.Completed = r.Count
		case int(models.WorkOrderStatusClosed):
			stats.Closed = r.Count
		case int(models.WorkOrderStatusRejected):
			stats.Rejected = r.Count
		}
	}

	// 按优先级统计
	var priorityResults []struct {
		Priority int
		Count    int
	}
	if err := s.db.WithContext(ctx).Model(&models.WorkOrder{}).
		Select("priority, count(*) as count").
		Group("priority").
		Scan(&priorityResults).Error; err != nil {
		return nil, fmt.Errorf("按优先级统计失败: %w", err)
	}

	for _, r := range priorityResults {
		var name string
		switch r.Priority {
		case int(models.WorkOrderPriorityLow):
			name = "低"
		case int(models.WorkOrderPriorityMedium):
			name = "中"
		case int(models.WorkOrderPriorityHigh):
			name = "高"
		case int(models.WorkOrderPriorityUrgent):
			name = "紧急"
		}
		stats.ByPriority[name] = r.Count
	}

	// 按分类统计 - 使用JOIN避免N+1查询
	type CategoryStat struct {
		CategoryID   string
		CategoryName string
		Count        int
	}
	var categoryResults []CategoryStat
	if err := s.db.WithContext(ctx).Model(&models.WorkOrder{}).
		Select("sys_workorder_category.id as category_id, " +
			"sys_workorder_category.category_name, " +
			"count(*) as count").
		Joins("INNER JOIN sys_workorder_category ON " +
			"sys_workorder_category.id = sys_workorder.category_id").
		Group("sys_workorder_category.id, sys_workorder_category.category_name").
		Scan(&categoryResults).Error; err != nil {
		return nil, fmt.Errorf("按分类统计失败: %w", err)
	}

	for _, r := range categoryResults {
		stats.ByCategory[r.CategoryName] = r.Count
	}

	// 按处理人统计 - 使用JOIN避免N+1查询
	type AssigneeStat struct {
		AssigneeID   string
		AssigneeName string
		Count        int
	}
	var assigneeResults []AssigneeStat
	if err := s.db.WithContext(ctx).Model(&models.WorkOrder{}).
		Select("sys_user.id as assignee_id, " +
			"COALESCE(sys_user.nickname, sys_user.username, '') as assignee_name, " +
			"count(*) as count").
		Joins("INNER JOIN sys_user ON sys_user.id = sys_workorder.assignee_id").
		Where("sys_workorder.assignee_id IS NOT NULL").
		Group("sys_user.id, sys_user.nickname, sys_user.username").
		Scan(&assigneeResults).Error; err != nil {
		return nil, fmt.Errorf("按处理人统计失败: %w", err)
	}

	for _, r := range assigneeResults {
		stats.ByAssignee = append(stats.ByAssignee, AssigneeStatistics{
			AssigneeID:   r.AssigneeID,
			AssigneeName: r.AssigneeName,
			TotalCount:   r.Count,
		})
	}

	// 按部门统计 - 使用JOIN避免N+1查询
	type DeptStat struct {
		DeptID   string
		DeptName string
		Count    int
	}
	var deptResults []DeptStat
	if err := s.db.WithContext(ctx).Model(&models.WorkOrder{}).
		Select("sys_dept.id as dept_id, " +
			"sys_dept.dept_name, " +
			"count(*) as count").
		Joins("INNER JOIN sys_dept ON " +
			"sys_dept.id = sys_workorder.dept_id").
		Where("sys_workorder.dept_id IS NOT NULL").
		Group("sys_dept.id, sys_dept.dept_name").
		Scan(&deptResults).Error; err != nil {
		return nil, fmt.Errorf("按部门统计失败: %w", err)
	}

	for _, r := range deptResults {
		stats.ByDepartment = append(stats.ByDepartment, DepartmentStatistics{
			DeptID:     r.DeptID,
			DeptName:   r.DeptName,
			TotalCount: r.Count,
		})
	}

	// 趋势统计（最近30天）
	var trendResults []struct {
		Date  string
		Count int
	}
	if err := s.db.WithContext(ctx).Model(&models.WorkOrder{}).
		Select("DATE(created_at) as date, count(*) as count").
		Where("created_at >= ?", time.Now().AddDate(0, 0, -30)).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&trendResults).Error; err != nil {
		return nil, fmt.Errorf("趋势统计失败: %w", err)
	}

	for _, r := range trendResults {
		stats.Trend = append(stats.Trend, TrendData{
			Date:  r.Date,
			Count: r.Count,
		})
	}

	return stats, nil
}
