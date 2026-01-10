package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
)

// ==================== 工单服务 ====================

// WorkOrderService 工单服务
type WorkOrderService struct {
	db *gorm.DB
}

// NewWorkOrderService 创建工单服务
func NewWorkOrderService(db *gorm.DB) *WorkOrderService {
	return &WorkOrderService{db: db}
}

// ==================== 请求/响应结构 ====================

// WorkOrderListRequest 工单列表查询请求
type WorkOrderListRequest struct {
	Current      int    `json:"current"`
	PageSize     int    `json:"pageSize"`
	WorkOrderNo  string `json:"workOrderNo"`
	Title        string `json:"title"`
	CategoryID   string `json:"categoryId"`
	Type         string `json:"type"`
	Priority     *int   `json:"priority"`
	Status       *int   `json:"status"`
	SubmitterID  string `json:"submitterId"`
	AssigneeID   string `json:"assigneeId"`
	DeptID       string `json:"deptId"`
	StartDate    string `json:"startDate"`
	EndDate      string `json:"endDate"`
}

// WorkOrderCreateRequest 创建工单请求
type WorkOrderCreateRequest struct {
	Title             string                      `json:"title" binding:"required,max=200"`
	CategoryID        string                      `json:"categoryId" binding:"required,uuid"`
	Type              models.WorkOrderType        `json:"type" binding:"required"`
	Priority          models.WorkOrderPriority    `json:"priority"`
	Description       string                      `json:"description"`
	DeptID            *string                     `json:"deptId"`
	ExpectedResolveAt *string                     `json:"expectedResolveAt"` // YYYY-MM-DD HH:mm:ss
	AttachmentIDs     string                      `json:"attachmentIds"`
	IsAutoAssigned    bool                        `json:"isAutoAssigned"`    // 是否自动分配（默认根据配置决定）
	AssigneeID        *string                     `json:"assigneeId"`        // 手动指定处理人（当不自动分配时使用）
}

// WorkOrderUpdateRequest 更新工单请求
type WorkOrderUpdateRequest struct {
	ID                string                      `json:"id" binding:"required,uuid"`
	Title             *string                     `json:"title"`
	CategoryID        *string                     `json:"categoryId"`
	Type              *models.WorkOrderType       `json:"type"`
	Priority          *models.WorkOrderPriority   `json:"priority"`
	Status            *models.WorkOrderStatus     `json:"status"`
	Description       *string                     `json:"description"`
	Solution          *string                     `json:"solution"`
	AssigneeID        *string                     `json:"assigneeId"`
	DeptID            *string                     `json:"deptId"`
	ExpectedResolveAt *string                     `json:"expectedResolveAt"`
	AttachmentIDs     *string                     `json:"attachmentIds"`
	ResolvedAt        *string                     `json:"resolvedAt"`
	ClosedAt          *string                     `json:"closedAt"`
}

// AssignWorkOrderRequest 分配工单请求
type AssignWorkOrderRequest struct {
	AssigneeID string `json:"assigneeId" binding:"required,uuid"`
	Remark     string `json:"remark"`
}

// UpdateStatusRequest 更新状态请求
type UpdateStatusRequest struct {
	Status   models.WorkOrderStatus `json:"status" binding:"required"`
	Solution string                 `json:"solution"`
	Remark   string                 `json:"remark"`
}

// AddCommentRequest 添加评论请求
type AddCommentRequest struct {
	Content    string `json:"content" binding:"required"`
	IsInternal bool   `json:"isInternal"`
}

// WorkOrderStatistics 工单统计数据
type WorkOrderStatistics struct {
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
	AvgProcessTime float64              `json:"avgProcessTime"` // 平均处理时长（小时）
}

// AssigneeStatistics 处理人统计
type AssigneeStatistics struct {
	AssigneeID   string  `json:"assigneeId"`
	AssigneeName string  `json:"assigneeName"`
	TotalCount   int     `json:"totalCount"`
	PendingCount int     `json:"pendingCount"`
	DoneCount    int     `json:"doneCount"`
	AvgProcessTime float64 `json:"avgProcessTime"`
}

// DepartmentStatistics 部门统计
type DepartmentStatistics struct {
	DeptID    string `json:"deptId"`
	DeptName  string `json:"deptName"`
	TotalCount int   `json:"totalCount"`
	DoneCount  int   `json:"doneCount"`
}

// TrendData 趋势数据
type TrendData struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// ==================== 工单CRUD方法 ====================

// GetWorkOrderList 分页查询工单列表
func (s *WorkOrderService) GetWorkOrderList(ctx context.Context, req *WorkOrderListRequest) ([]models.WorkOrder, int64, error) {
	var list []models.WorkOrder
	var total int64

	query := s.db.WithContext(ctx).Model(&models.WorkOrder{})

	// 构建查询条件
	if req.WorkOrderNo != "" {
		query = query.Where("work_order_no LIKE ?", "%"+req.WorkOrderNo+"%")
	}
	if req.Title != "" {
		query = query.Where("title LIKE ?", "%"+req.Title+"%")
	}
	if req.CategoryID != "" {
		query = query.Where("category_id = ?", req.CategoryID)
	}
	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}
	if req.Priority != nil {
		query = query.Where("priority = ?", *req.Priority)
	}
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}
	if req.SubmitterID != "" {
		query = query.Where("submitter_id = ?", req.SubmitterID)
	}
	if req.AssigneeID != "" {
		query = query.Where("assignee_id = ?", req.AssigneeID)
	}
	if req.DeptID != "" {
		query = query.Where("dept_id = ?", req.DeptID)
	}
	if req.StartDate != "" {
		query = query.Where("created_at >= ?", req.StartDate)
	}
	if req.EndDate != "" {
		query = query.Where("created_at <= ?", req.EndDate)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询工单总数失败: %w", err)
	}

	// 分页查询
	current := req.Current
	if current <= 0 {
		current = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (current - 1) * pageSize

	// 预加载关联数据
	if err := query.
		Preload("Category").
		Preload("Submitter").
		Preload("Assignee").
		Preload("Dept").
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("查询工单列表失败: %w", err)
	}

	return list, total, nil
}

// GetMyPendingWorkOrdersRequest 获取待办工单请求
type GetMyPendingWorkOrdersRequest struct {
	Limit int `json:"limit"` // 限制返回数量，默认5
}

// GetMyPendingWorkOrders 获取当前用户的待办工单（待处理和处理中）
func (s *WorkOrderService) GetMyPendingWorkOrders(ctx context.Context, req *GetMyPendingWorkOrdersRequest, userID string) ([]models.WorkOrder, int64, error) {
	var list []models.WorkOrder
	var total int64

	// 默认返回5条
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 5
	}

	// 构建查询：分配给当前用户且状态为待处理(0)或处理中(1)的工单
	query := s.db.WithContext(ctx).
		Preload("Category").
		Preload("Submitter").
		Preload("Assignee")

	// 获取总数
	if err := query.Model(&models.WorkOrder{}).
		Where("assignee_id = ?", userID).
		Where("status IN ?", []int{0, 1}). // 待处理或处理中
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询待办工单总数失败: %w", err)
	}

	// 获取列表，按优先级降序、创建时间降序排序
	if err := query.
		Where("assignee_id = ?", userID).
		Where("status IN ?", []int{0, 1}).
		Order("priority DESC"). // 优先级高的在前
		Order("created_at DESC").
		Limit(limit).
		Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("查询待办工单列表失败: %w", err)
	}

	return list, total, nil
}

// GetWorkOrderByID 获取工单详情
func (s *WorkOrderService) GetWorkOrderByID(ctx context.Context, id string) (*models.WorkOrder, error) {
	var workOrder models.WorkOrder

	if err := s.db.WithContext(ctx).
		Preload("Category").
		Preload("Submitter").
		Preload("Assignee").
		Preload("Dept").
		Preload("Comments", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).Preload("Comments.User").
		Preload("History", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).Preload("History.Operator").
		Preload("Ratings").
		Preload("Ratings.Rater").
		Where("id = ?", id).
		First(&workOrder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("工单不存在")
		}
		return nil, fmt.Errorf("查询工单详情失败: %w", err)
	}

	return &workOrder, nil
}

// CreateWorkOrder 创建工单
func (s *WorkOrderService) CreateWorkOrder(ctx context.Context, req *WorkOrderCreateRequest, submitterID string) (*models.WorkOrder, error) {
	// 开启事务
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取工单配置
	var config models.WorkOrderConfig
	if err := tx.Where("id = ?", "default").First(&config).Error; err != nil {
		// 如果配置不存在，使用默认配置
		config = models.WorkOrderConfig{
			AutoAssignEnabled:  true,
			AutoAssignStrategy: "assign_one",
		}
	}

	workOrder := &models.WorkOrder{
		Title:         req.Title,
		CategoryID:    req.CategoryID,
		Type:          req.Type,
		Priority:      req.Priority,
		Status:        models.WorkOrderStatusPending,
		Description:   req.Description,
		SubmitterID:   submitterID,
		DeptID:        req.DeptID,
		AttachmentIDs: req.AttachmentIDs,
	}

	// 处理期望解决时间
	if req.ExpectedResolveAt != nil && *req.ExpectedResolveAt != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", *req.ExpectedResolveAt); err == nil {
			workOrder.ExpectedResolveAt = &t
		}
	}

	// 自动分配逻辑
	shouldAutoAssign := req.IsAutoAssigned && config.AutoAssignEnabled
	if shouldAutoAssign {
		// 调用值班管理服务获取当天值班人员
		dutyMembers, err := s.getTodayDutyMembers(ctx, nil)
		if err != nil {
			// 获取值班人员失败，记录警告但不阻止工单创建
			logger.Warnf("获取当天值班人员失败: %v，工单将不自动分配", err)
		} else if len(dutyMembers) > 0 {
			// 根据分配策略选择处理人
			var assigneeID string
			var dutyPoolID string
			var dutyType string

			switch config.AutoAssignStrategy {
			case "assign_all":
				// 分配给所有值班人员（创建多个工单）
				// 这里简化处理，只分配给第一个
				assigneeID = dutyMembers[0].UserID
				dutyPoolID = dutyMembers[0].PoolID
				dutyType = string(dutyMembers[0].DutyType)
			case "assign_order":
				// 按顺序分配（这里简化，使用第一个）
				assigneeID = dutyMembers[0].UserID
				dutyPoolID = dutyMembers[0].PoolID
				dutyType = string(dutyMembers[0].DutyType)
			default: // assign_one
				// 随机分配一个（这里使用第一个）
				assigneeID = dutyMembers[0].UserID
				dutyPoolID = dutyMembers[0].PoolID
				dutyType = string(dutyMembers[0].DutyType)
			}

			workOrder.AssigneeID = &assigneeID
			workOrder.IsAutoAssigned = true
			workOrder.DutyPoolID = &dutyPoolID
			workOrder.DutyType = dutyType
			workOrder.AssignStrategy = config.AutoAssignStrategy
		}
	} else if req.AssigneeID != nil {
		// 手动指定处理人
		workOrder.AssigneeID = req.AssigneeID
	}

	// 创建工单
	if err := tx.Create(workOrder).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建工单失败: %w", err)
	}

	// 记录操作历史
	s.recordWorkOrderHistory(tx, workOrder.ID, "create", "", "", "", "创建工单", submitterID)

	// 如果自动分配了，记录分配历史
	if workOrder.IsAutoAssigned && workOrder.AssigneeID != nil {
		s.recordWorkOrderHistory(tx, workOrder.ID, "auto_assign", "assignee_id", "", *workOrder.AssigneeID, "自动分配给值班人员", submitterID)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("创建工单失败: %w", err)
	}

	// 重新加载关联数据
	return s.GetWorkOrderByID(ctx, workOrder.ID)
}

// UpdateWorkOrder 更新工单
func (s *WorkOrderService) UpdateWorkOrder(ctx context.Context, req *WorkOrderUpdateRequest, operatorID string) error {
	// 开启事务
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var workOrder models.WorkOrder
	if err := tx.Where("id = ?", req.ID).First(&workOrder).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("工单不存在")
		}
		return fmt.Errorf("查询工单失败: %w", err)
	}

	// 记录变更
	changes := make(map[string]string)

	if req.Title != nil && *req.Title != workOrder.Title {
		changes["title"] = fmt.Sprintf("%s -> %s", workOrder.Title, *req.Title)
		workOrder.Title = *req.Title
	}
	if req.CategoryID != nil && *req.CategoryID != workOrder.CategoryID {
		changes["category_id"] = fmt.Sprintf("%s -> %s", workOrder.CategoryID, *req.CategoryID)
		workOrder.CategoryID = *req.CategoryID
	}
	if req.Type != nil && *req.Type != workOrder.Type {
		changes["type"] = fmt.Sprintf("%s -> %s", workOrder.Type, *req.Type)
		workOrder.Type = *req.Type
	}
	if req.Priority != nil && *req.Priority != workOrder.Priority {
		changes["priority"] = fmt.Sprintf("%d -> %d", workOrder.Priority, *req.Priority)
		workOrder.Priority = *req.Priority
	}
	if req.Status != nil && *req.Status != workOrder.Status {
		changes["status"] = fmt.Sprintf("%d -> %d", workOrder.Status, *req.Status)
		workOrder.Status = *req.Status

		// 更新状态时间
		if *req.Status == models.WorkOrderStatusCompleted && workOrder.ResolvedAt == nil {
			now := time.Now()
			workOrder.ResolvedAt = &now
		}
		if *req.Status == models.WorkOrderStatusClosed && workOrder.ClosedAt == nil {
			now := time.Now()
			workOrder.ClosedAt = &now
		}
	}
	if req.Description != nil && *req.Description != workOrder.Description {
		changes["description"] = "描述已更新"
		workOrder.Description = *req.Description
	}
	if req.Solution != nil && (*req.Solution != "" || workOrder.Solution != "") {
		changes["solution"] = "解决方案已更新"
		workOrder.Solution = *req.Solution
	}
	if req.AssigneeID != nil {
		oldAssignee := ""
		if workOrder.AssigneeID != nil {
			oldAssignee = *workOrder.AssigneeID
		}
		if *req.AssigneeID != oldAssignee {
			changes["assignee_id"] = fmt.Sprintf("%s -> %s", oldAssignee, *req.AssigneeID)
			workOrder.AssigneeID = req.AssigneeID
		}
	}
	if req.DeptID != nil {
		oldDept := ""
		if workOrder.DeptID != nil {
			oldDept = *workOrder.DeptID
		}
		if *req.DeptID != oldDept {
			changes["dept_id"] = fmt.Sprintf("%s -> %s", oldDept, *req.DeptID)
			workOrder.DeptID = req.DeptID
		}
	}
	if req.AttachmentIDs != nil && *req.AttachmentIDs != workOrder.AttachmentIDs {
		changes["attachment_ids"] = "附件已更新"
		workOrder.AttachmentIDs = *req.AttachmentIDs
	}
	if req.ExpectedResolveAt != nil {
		var newTime time.Time
		if t, err := time.Parse("2006-01-02 15:04:05", *req.ExpectedResolveAt); err == nil {
			newTime = t
			oldTime := ""
			if workOrder.ExpectedResolveAt != nil {
				oldTime = workOrder.ExpectedResolveAt.Format("2006-01-02 15:04:05")
			}
			if newTime.String() != oldTime {
				changes["expected_resolve_at"] = fmt.Sprintf("%s -> %s", oldTime, newTime.Format("2006-01-02 15:04:05"))
				workOrder.ExpectedResolveAt = &newTime
			}
		}
	}
	if req.ResolvedAt != nil {
		if t, err := time.Parse("2006-01-02 15:04:05", *req.ResolvedAt); err == nil {
			workOrder.ResolvedAt = &t
		}
	}
	if req.ClosedAt != nil {
		if t, err := time.Parse("2006-01-02 15:04:05", *req.ClosedAt); err == nil {
			workOrder.ClosedAt = &t
		}
	}

	// 更新工单
	if err := tx.Save(&workOrder).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新工单失败: %w", err)
	}

	// 记录变更历史
	for field, change := range changes {
		s.recordWorkOrderHistory(tx, workOrder.ID, "update", field, "", change, "", operatorID)
	}

	if len(changes) > 0 {
		s.recordWorkOrderHistory(tx, workOrder.ID, "update", "", "", "", "更新工单", operatorID)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("更新工单失败: %w", err)
	}

	return nil
}

// DeleteWorkOrder 删除工单
func (s *WorkOrderService) DeleteWorkOrder(ctx context.Context, id string) error {
	// 检查工单是否存在
	var workOrder models.WorkOrder
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&workOrder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("工单不存在")
		}
		return fmt.Errorf("查询工单失败: %w", err)
	}

	// 只有待处理状态的工单可以删除
	if workOrder.Status != models.WorkOrderStatusPending {
		return fmt.Errorf("只有待处理状态的工单可以删除")
	}

	// 开启事务
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除关联数据（评论、历史、评价）
	if err := tx.Where("work_order_id = ?", id).Delete(&models.WorkOrderComment{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除工单评论失败: %w", err)
	}
	if err := tx.Where("work_order_id = ?", id).Delete(&models.WorkOrderHistory{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除工单历史失败: %w", err)
	}
	if err := tx.Where("work_order_id = ?", id).Delete(&models.WorkOrderRating{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除工单评价失败: %w", err)
	}

	// 删除工单
	if err := tx.Delete(&workOrder).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除工单失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("删除工单失败: %w", err)
	}

	return nil
}

// BatchDeleteWorkOrders 批量删除工单
func (s *WorkOrderService) BatchDeleteWorkOrders(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("工单ID列表不能为空")
	}

	// 开启事务
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查是否所有工单都是待处理状态
	var count int64
	if err := tx.Model(&models.WorkOrder{}).Where("id IN ? AND status != ?", ids, models.WorkOrderStatusPending).Count(&count).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("检查工单状态失败: %w", err)
	}
	if count > 0 {
		tx.Rollback()
		return fmt.Errorf("只有待处理状态的工单可以删除")
	}

	// 删除关联数据 - 使用IN批量删除提升性能
	if err := tx.Where("work_order_id IN ?", ids).Delete(&models.WorkOrderComment{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除工单评论失败: %w", err)
	}
	if err := tx.Where("work_order_id IN ?", ids).Delete(&models.WorkOrderHistory{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除工单历史失败: %w", err)
	}
	if err := tx.Where("work_order_id IN ?", ids).Delete(&models.WorkOrderRating{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除工单评价失败: %w", err)
	}

	// 删除工单
	if err := tx.Where("id IN ?", ids).Delete(&models.WorkOrder{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("批量删除工单失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("批量删除工单失败: %w", err)
	}

	return nil
}

// AssignWorkOrder 分配工单
func (s *WorkOrderService) AssignWorkOrder(ctx context.Context, id string, req *AssignWorkOrderRequest, operatorID string) error {
	var workOrder models.WorkOrder
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&workOrder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("工单不存在")
		}
		return fmt.Errorf("查询工单失败: %w", err)
	}

	oldAssignee := ""
	if workOrder.AssigneeID != nil {
		oldAssignee = *workOrder.AssigneeID
	}

	// 开启事务
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 更新工单
	updates := map[string]interface{}{
		"assignee_id": req.AssigneeID,
		"updated_by":  operatorID,
	}

	if err := tx.Model(&workOrder).Updates(updates).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("分配工单失败: %w", err)
	}

	// 记录历史
	s.recordWorkOrderHistory(tx, workOrder.ID, "assign", "assignee_id", oldAssignee, req.AssigneeID, req.Remark, operatorID)

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("分配工单失败: %w", err)
	}

	return nil
}

// AssignToTodayDuty 分配给当天值班人员
func (s *WorkOrderService) AssignToTodayDuty(ctx context.Context, id string, operatorID string) error {
	// 获取当天值班人员
	dutyMembers, err := s.getTodayDutyMembers(ctx, nil)
	if err != nil {
		// 查询出错或没有值班人员，返回友好提示
		return fmt.Errorf("今天没有值班人员")
	}

	if len(dutyMembers) == 0 {
		return fmt.Errorf("今天没有值班人员")
	}

	// 获取配置
	var config models.WorkOrderConfig
	if err := s.db.WithContext(ctx).Where("id = ?", "default").First(&config).Error; err != nil {
		config.AutoAssignStrategy = "assign_one"
	}

	// 根据策略选择处理人
	var assigneeID string
	switch config.AutoAssignStrategy {
	case "assign_all":
		// 这里简化处理，只分配给第一个
		assigneeID = dutyMembers[0].UserID
	case "assign_order":
		assigneeID = dutyMembers[0].UserID
	default: // assign_one
		assigneeID = dutyMembers[0].UserID
	}

	req := &AssignWorkOrderRequest{
		AssigneeID: assigneeID,
		Remark:     "自动分配给当天值班人员",
	}

	return s.AssignWorkOrder(ctx, id, req, operatorID)
}

// UpdateWorkOrderStatus 更新工单状态
func (s *WorkOrderService) UpdateWorkOrderStatus(ctx context.Context, id string, req *UpdateStatusRequest, operatorID string) error {
	var workOrder models.WorkOrder
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&workOrder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("工单不存在")
		}
		return fmt.Errorf("查询工单失败: %w", err)
	}

	// 验证状态流转
	if !isValidStatusTransition(workOrder.Status, req.Status) {
		return fmt.Errorf("无效的状态流转: %d -> %d", workOrder.Status, req.Status)
	}

	// 开启事务
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	updates := map[string]interface{}{
		"status":    req.Status,
		"updated_by": operatorID,
	}

	// 更新解决方案
	if req.Solution != "" {
		updates["solution"] = req.Solution
	}

	// 更新状态时间
	if req.Status == models.WorkOrderStatusCompleted && workOrder.ResolvedAt == nil {
		now := time.Now()
		updates["resolved_at"] = &now
	}
	if req.Status == models.WorkOrderStatusClosed && workOrder.ClosedAt == nil {
		now := time.Now()
		updates["closed_at"] = &now
	}

	if err := tx.Model(&workOrder).Updates(updates).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新工单状态失败: %w", err)
	}

	// 记录历史
	remark := req.Remark
	if remark == "" {
		remark = getStatusName(req.Status)
	}
	s.recordWorkOrderHistory(tx, workOrder.ID, "update_status", "status",
		string(workOrder.Status), string(req.Status), remark, operatorID)

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("更新工单状态失败: %w", err)
	}

	return nil
}

// ==================== 工单评论方法 ====================

// AddWorkOrderComment 添加评论
func (s *WorkOrderService) AddWorkOrderComment(ctx context.Context, workOrderID string, req *AddCommentRequest, userID string) error {
	// 检查工单是否存在
	var workOrder models.WorkOrder
	if err := s.db.WithContext(ctx).Where("id = ?", workOrderID).First(&workOrder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("工单不存在")
		}
		return fmt.Errorf("查询工单失败: %w", err)
	}

	comment := &models.WorkOrderComment{
		ID:          uuid.New().String(),
		WorkOrderID: workOrderID,
		UserID:      userID,
		Content:     req.Content,
		IsInternal:  req.IsInternal,
		CreatedAt:   time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(comment).Error; err != nil {
		return fmt.Errorf("添加评论失败: %w", err)
	}

	return nil
}

// GetWorkOrderComments 获取评论列表
func (s *WorkOrderService) GetWorkOrderComments(ctx context.Context, workOrderID string) ([]models.WorkOrderComment, error) {
	var comments []models.WorkOrderComment

	if err := s.db.WithContext(ctx).
		Where("work_order_id = ?", workOrderID).
		Preload("User").
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		return nil, fmt.Errorf("查询评论列表失败: %w", err)
	}

	return comments, nil
}

// GetWorkOrderHistory 获取操作历史
func (s *WorkOrderService) GetWorkOrderHistory(ctx context.Context, workOrderID string) ([]models.WorkOrderHistory, error) {
	var history []models.WorkOrderHistory

	if err := s.db.WithContext(ctx).
		Where("work_order_id = ?", workOrderID).
		Preload("Operator").
		Order("created_at ASC").
		Find(&history).Error; err != nil {
		return nil, fmt.Errorf("查询操作历史失败: %w", err)
	}

	return history, nil
}

// recordWorkOrderHistory 记录工单操作历史（私有方法）
func (s *WorkOrderService) recordWorkOrderHistory(tx *gorm.DB, workOrderID, action, field, oldValue, newValue, remark, operatorID string) error {
	history := &models.WorkOrderHistory{
		ID:          uuid.New().String(),
		WorkOrderID: workOrderID,
		Action:      action,
		Field:       field,
		OldValue:    oldValue,
		NewValue:    newValue,
		Remark:      remark,
		OperatorID:  operatorID,
		CreatedAt:   time.Now(),
	}

	if err := tx.Create(history).Error; err != nil {
		return fmt.Errorf("记录操作历史失败: %w", err)
	}

	return nil
}

// ==================== 工单分类方法 ====================

// GetWorkOrderCategoryList 获取分类列表（树形结构）
func (s *WorkOrderService) GetWorkOrderCategoryList(ctx context.Context) ([]models.WorkOrderCategory, error) {
	var list []models.WorkOrderCategory

	if err := s.db.WithContext(ctx).
		Where("parent_id IS NULL").
		Order("sort_order ASC, created_at ASC").
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询分类列表失败: %w", err)
	}

	// 加载子分类
	for i := range list {
		s.loadChildrenCategories(ctx, &list[i])
	}

	return list, nil
}

// loadChildrenCategories 递归加载子分类
func (s *WorkOrderService) loadChildrenCategories(ctx context.Context, category *models.WorkOrderCategory) error {
	var children []models.WorkOrderCategory

	if err := s.db.WithContext(ctx).
		Where("parent_id = ?", category.ID).
		Order("sort_order ASC, created_at ASC").
		Find(&children).Error; err != nil {
		return err
	}

	category.Children = children

	// 递归加载子分类的子分类
	for i := range children {
		s.loadChildrenCategories(ctx, &children[i])
	}

	return nil
}

// GetWorkOrderCategoryByID 获取分类详情
func (s *WorkOrderService) GetWorkOrderCategoryByID(ctx context.Context, id string) (*models.WorkOrderCategory, error) {
	var category models.WorkOrderCategory

	if err := s.db.WithContext(ctx).
		Preload("Parent").
		Where("id = ?", id).
		First(&category).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("分类不存在")
		}
		return nil, fmt.Errorf("查询分类详情失败: %w", err)
	}

	return &category, nil
}

// CreateWorkOrderCategory 创建分类
func (s *WorkOrderService) CreateWorkOrderCategory(ctx context.Context, category *models.WorkOrderCategory, creatorID string) error {
	category.CreatedBy = creatorID
	category.UpdatedBy = creatorID

	if err := s.db.WithContext(ctx).Create(category).Error; err != nil {
		return fmt.Errorf("创建分类失败: %w", err)
	}

	return nil
}

// UpdateWorkOrderCategory 更新分类
func (s *WorkOrderService) UpdateWorkOrderCategory(ctx context.Context, category *models.WorkOrderCategory, operatorID string) error {
	category.UpdatedBy = operatorID

	if err := s.db.WithContext(ctx).Save(category).Error; err != nil {
		return fmt.Errorf("更新分类失败: %w", err)
	}

	return nil
}

// DeleteWorkOrderCategory 删除分类
func (s *WorkOrderService) DeleteWorkOrderCategory(ctx context.Context, id string) error {
	// 检查是否有子分类
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.WorkOrderCategory{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("检查子分类失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("该分类下有子分类，无法删除")
	}

	// 检查是否有关联工单
	if err := s.db.WithContext(ctx).Model(&models.WorkOrder{}).Where("category_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("检查关联工单失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("该分类下有关联工单，无法删除")
	}

	// 删除分类
	if err := s.db.WithContext(ctx).Delete(&models.WorkOrderCategory{}, id).Error; err != nil {
		return fmt.Errorf("删除分类失败: %w", err)
	}

	return nil
}

// GetEnabledCategories 获取启用的分类（用于下拉选择）
func (s *WorkOrderService) GetEnabledCategories(ctx context.Context) ([]models.WorkOrderCategory, error) {
	var list []models.WorkOrderCategory

	if err := s.db.WithContext(ctx).
		Where("status = ?", models.WorkOrderCategoryStatusEnabled).
		Order("sort_order ASC, created_at ASC").
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询分类列表失败: %w", err)
	}

	return list, nil
}

// ==================== 工单评价方法 ====================

// CreateWorkOrderRating 创建评价
func (s *WorkOrderService) CreateWorkOrderRating(ctx context.Context, workOrderID, ratingType string, completionScore, cooperationScore int, comment, raterID string) error {
	// 检查工单是否存在且状态是否允许评价
	var workOrder models.WorkOrder
	if err := s.db.WithContext(ctx).Where("id = ?", workOrderID).First(&workOrder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("工单不存在")
		}
		return fmt.Errorf("查询工单失败: %w", err)
	}

	// 只有已完成或已关闭的工单才能评价
	if workOrder.Status != models.WorkOrderStatusCompleted && workOrder.Status != models.WorkOrderStatusClosed {
		return fmt.Errorf("只有已完成或已关闭的工单才能评价")
	}

	// 检查是否已经评价过
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.WorkOrderRating{}).
		Where("work_order_id = ? AND rating_type = ? AND rater_id = ?", workOrderID, ratingType, raterID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("检查评价记录失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("您已经评价过该工单")
	}

	// 验证评价类型
	if ratingType != "user" && ratingType != "handler" {
		return fmt.Errorf("无效的评价类型")
	}

	// 验证评分
	if ratingType == "user" && (completionScore < 1 || completionScore > 5) {
		return fmt.Errorf("完成度评分必须在1-5之间")
	}
	if ratingType == "handler" && (cooperationScore < 1 || cooperationScore > 5) {
		return fmt.Errorf("配合度评分必须在1-5之间")
	}

	rating := &models.WorkOrderRating{
		ID:              uuid.New().String(),
		WorkOrderID:     workOrderID,
		RatingType:      ratingType,
		CompletionScore: completionScore,
		CooperationScore: cooperationScore,
		Comment:         comment,
		RaterID:         raterID,
		CreatedAt:       time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(rating).Error; err != nil {
		return fmt.Errorf("创建评价失败: %w", err)
	}

	return nil
}

// GetWorkOrderRatings 获取工单评价列表
func (s *WorkOrderService) GetWorkOrderRatings(ctx context.Context, workOrderID string) ([]models.WorkOrderRating, error) {
	var ratings []models.WorkOrderRating

	if err := s.db.WithContext(ctx).
		Where("work_order_id = ?", workOrderID).
		Preload("Rater").
		Order("created_at DESC").
		Find(&ratings).Error; err != nil {
		return nil, fmt.Errorf("查询评价列表失败: %w", err)
	}

	return ratings, nil
}

// GetRatingStatistics 获取评分统计
func (s *WorkOrderService) GetRatingStatistics(ctx context.Context, workOrderID string) (map[string]interface{}, error) {
	var ratings []models.WorkOrderRating

	if err := s.db.WithContext(ctx).
		Where("work_order_id = ?", workOrderID).
		Find(&ratings).Error; err != nil {
		return nil, fmt.Errorf("查询评价列表失败: %w", err)
	}

	stats := map[string]interface{}{
		"total_count": len(ratings),
		"user_rating":   nil,
		"handler_rating": nil,
		"avg_score":     0.0,
	}

	var userRating, handlerRating *models.WorkOrderRating
	var completionSum, cooperationSum int
	var completionCount, cooperationCount int

	for _, r := range ratings {
		if r.RatingType == "user" && userRating == nil {
			userRating = &r
			completionSum += r.CompletionScore
			completionCount++
		}
		if r.RatingType == "handler" && handlerRating == nil {
			handlerRating = &r
			cooperationSum += r.CooperationScore
			cooperationCount++
		}
	}

	stats["user_rating"] = userRating
	stats["handler_rating"] = handlerRating

	// 计算平均分
	if completionCount > 0 {
		stats["avg_completion_score"] = float64(completionSum) / float64(completionCount)
	}
	if cooperationCount > 0 {
		stats["avg_cooperation_score"] = float64(cooperationSum) / float64(cooperationCount)
	}

	return stats, nil
}

// ==================== 工单统计方法 ====================

// GetWorkOrderStatistics 获取工单统计数据
func (s *WorkOrderService) GetWorkOrderStatistics(ctx context.Context) (*WorkOrderStatistics, error) {
	stats := &WorkOrderStatistics{
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
		Select("sys_workorder_category.id as category_id, "+
			"sys_workorder_category.category_name, "+
			"count(*) as count").
		Joins("INNER JOIN sys_workorder_category ON "+
			"sys_workorder_category.id = sys_work_orders.category_id").
		Group("sys_workorder_category.id, sys_workorder_category.category_name").
		Scan(&categoryResults).Error; err != nil {
		return nil, fmt.Errorf("按分类统计失败: %w", err)
	}

	// 直接使用JOIN后的结果
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
		Select("sys_user.id as assignee_id, "+
			"COALESCE(sys_user.nickname, sys_user.username, '') as assignee_name, "+
			"count(*) as count").
		Joins("INNER JOIN sys_user ON sys_user.id = sys_work_orders.assignee_id").
		Where("sys_work_orders.assignee_id IS NOT NULL").
		Group("sys_user.id, sys_user.nickname, sys_user.username").
		Scan(&assigneeResults).Error; err != nil {
		return nil, fmt.Errorf("按处理人统计失败: %w", err)
	}

	// 直接使用JOIN后的结果
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
		Select("sys_department.id as dept_id, "+
			"sys_department.dept_name, "+
			"count(*) as count").
		Joins("INNER JOIN sys_department ON "+
			"sys_department.id = sys_work_orders.dept_id").
		Where("sys_work_orders.dept_id IS NOT NULL").
		Group("sys_department.id, sys_department.dept_name").
		Scan(&deptResults).Error; err != nil {
		return nil, fmt.Errorf("按部门统计失败: %w", err)
	}

	// 直接使用JOIN后的结果
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

// ==================== 辅助方法 ====================

// getTodayDutyMembers 获取当天值班人员（私有方法）
// poolID: 可选参数，指定值班池ID以筛选特定池的值班人员
func (s *WorkOrderService) getTodayDutyMembers(ctx context.Context, poolID *string) ([]TodayDutyMember, error) {
	var members []TodayDutyMember

	// 查询今天的排班记录 - 使用本地时间日期
	today := time.Now().Local().Format("2006-01-02")

	query := `
		SELECT ds.id as schedule_id, ds.pool_id, dp.pool_name, ds.user_id,
		       u.username, u.phone, ds.duty_type
		FROM sys_duty_schedule ds
		LEFT JOIN sys_duty_pool dp ON ds.pool_id = dp.id
		LEFT JOIN sys_user u ON ds.user_id = u.id
		WHERE DATE(ds.schedule_date) = ? AND ds.status = 0 AND ds.deleted_at IS NULL
	`

	args := []interface{}{today}

	// 如果指定了值班池ID，添加筛选条件
	if poolID != nil {
		query += ` AND ds.pool_id = ?`
		args = append(args, *poolID)
	}

	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&members).Error; err != nil {
		return nil, fmt.Errorf("查询当天值班人员失败: %w", err)
	}

	return members, nil
}

// isValidStatusTransition 验证状态流转是否有效
func isValidStatusTransition(oldStatus, newStatus models.WorkOrderStatus) bool {
	// 定义允许的状态流转
	transitions := map[models.WorkOrderStatus][]models.WorkOrderStatus{
		models.WorkOrderStatusPending: {
			models.WorkOrderStatusProcessing,
			models.WorkOrderStatusRejected,
			models.WorkOrderStatusClosed,
		},
		models.WorkOrderStatusProcessing: {
			models.WorkOrderStatusPending,
			models.WorkOrderStatusCompleted,
			models.WorkOrderStatusRejected,
		},
		models.WorkOrderStatusCompleted: {
			models.WorkOrderStatusProcessing,
			models.WorkOrderStatusClosed,
		},
		models.WorkOrderStatusRejected: {
			models.WorkOrderStatusPending,
		},
		models.WorkOrderStatusClosed: {
			// 已关闭的状态不能变更
		},
	}

	allowedStatuses, ok := transitions[oldStatus]
	if !ok {
		return false
	}

	for _, status := range allowedStatuses {
		if status == newStatus {
			return true
		}
	}

	return false
}

// getStatusName 获取状态名称
func getStatusName(status models.WorkOrderStatus) string {
	switch status {
	case models.WorkOrderStatusPending:
		return "待处理"
	case models.WorkOrderStatusProcessing:
		return "处理中"
	case models.WorkOrderStatusCompleted:
		return "已完成"
	case models.WorkOrderStatusClosed:
		return "已关闭"
	case models.WorkOrderStatusRejected:
		return "已拒绝"
	default:
		return "未知"
	}
}

// ==================== 周期性工单方法 ====================

// PeriodicTemplateListRequest 周期性工单模板列表请求
type PeriodicTemplateListRequest struct {
	Current  int    `json:"current"`
	PageSize int    `json:"pageSize"`
	Title    string `json:"title"`
	IsEnabled *bool `json:"isEnabled"`
}

// CreatePeriodicTemplateRequest 创建周期性工单模板请求
type CreatePeriodicTemplateRequest struct {
	TemplateName   string                       `json:"templateName" binding:"required,max=100"`
	WorkOrderTitle string                       `json:"workOrderTitle" binding:"required,max=200"`
	Description    string                       `json:"description"`
	CategoryID     string                       `json:"categoryId" binding:"required,uuid"`
	Type           models.WorkOrderType         `json:"type" binding:"required"`
	Priority       models.WorkOrderPriority     `json:"priority"`
	CronExpression string                       `json:"cronExpression" binding:"required"`
	AssignType     models.PeriodicWorkOrderAssignType `json:"assignType"`
	AssignTargetID *string                      `json:"assignTargetId"`
	NotifyAssignee bool                         `json:"notifyAssignee"`
}

// UpdatePeriodicTemplateRequest 更新周期性工单模板请求
type UpdatePeriodicTemplateRequest struct {
	TemplateName   *string                      `json:"templateName"`
	WorkOrderTitle *string                      `json:"workOrderTitle"`
	Description    *string                      `json:"description"`
	CategoryID     *string                      `json:"categoryId"`
	Type           *models.WorkOrderType        `json:"type"`
	Priority       *models.WorkOrderPriority    `json:"priority"`
	CronExpression *string                      `json:"cronExpression"`
	AssignType     *models.PeriodicWorkOrderAssignType `json:"assignType"`
	AssignTargetID *string                      `json:"assignTargetId"`
	IsEnabled      *bool                        `json:"isEnabled"`
	NotifyAssignee *bool                        `json:"notifyAssignee"`
}

// GetPeriodicTemplateList 获取周期性工单模板列表
func (s *WorkOrderService) GetPeriodicTemplateList(ctx context.Context, req *PeriodicTemplateListRequest) ([]models.PeriodicWorkOrderTemplate, int64, error) {
	var list []models.PeriodicWorkOrderTemplate
	var total int64

	query := s.db.WithContext(ctx).Model(&models.PeriodicWorkOrderTemplate{})

	if req.Title != "" {
		query = query.Where("template_name LIKE ?", "%"+req.Title+"%")
	}
	if req.IsEnabled != nil {
		query = query.Where("is_enabled = ?", *req.IsEnabled)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询周期性工单模板总数失败: %w", err)
	}

	current := req.Current
	if current <= 0 {
		current = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (current - 1) * pageSize

	if err := query.
		Preload("Category").
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("查询周期性工单模板列表失败: %w", err)
	}

	// DEBUG: 打印从数据库读取的时间值
	for i := range list {
		if list[i].NextRunAt != nil {
			zoneName, offset := list[i].NextRunAt.Zone()
			log.Printf("[DEBUG] 读取模板: ID=%s, Cron=%s, NextRunAt=%s (Zone=%s, Offset=%d)",
				list[i].ID,
				list[i].CronExpression,
				list[i].NextRunAt.Format("2006-01-02 15:04:05 -07:00"),
				zoneName, offset)
		}
	}

	return list, total, nil
}

// GetPeriodicTemplate 获取周期性工单模板详情
func (s *WorkOrderService) GetPeriodicTemplate(ctx context.Context, id string) (*models.PeriodicWorkOrderTemplate, error) {
	var template models.PeriodicWorkOrderTemplate

	if err := s.db.WithContext(ctx).
		Preload("Category").
		Preload("ExecutionLogs", func(db *gorm.DB) *gorm.DB {
			return db.Order("executed_at DESC").Limit(10)
		}).
		Where("id = ?", id).
		First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("周期性工单模板不存在")
		}
		return nil, fmt.Errorf("查询周期性工单模板详情失败: %w", err)
	}

	return &template, nil
}

// CreatePeriodicTemplate 创建周期性工单模板
func (s *WorkOrderService) CreatePeriodicTemplate(ctx context.Context, req *CreatePeriodicTemplateRequest, creatorID string) (*models.PeriodicWorkOrderTemplate, error) {
	// 验证Cron表达式
	// 验证Cron表达式（使用6字段格式：秒 分 时 日 月 周）
	if _, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor).Parse(req.CronExpression); err != nil {
		return nil, fmt.Errorf("无效的Cron表达式: %w", err)
	}

	template := &models.PeriodicWorkOrderTemplate{
		TemplateName:   req.TemplateName,
		WorkOrderTitle: req.WorkOrderTitle,
		Description:    req.Description,
		CategoryID:     req.CategoryID,
		Type:           req.Type,
		Priority:       req.Priority,
		CronExpression: req.CronExpression,
		AssignType:     req.AssignType,
		AssignTargetID: req.AssignTargetID,
		IsEnabled:      false, // 默认不启用，需要手动启用
		NotifyAssignee: req.NotifyAssignee,
		TotalGenerated: 0,
	}

	if err := s.db.WithContext(ctx).Create(template).Error; err != nil {
		return nil, fmt.Errorf("创建周期性工单模板失败: %w", err)
	}

	return template, nil
}

// UpdatePeriodicTemplate 更新周期性工单模板
func (s *WorkOrderService) UpdatePeriodicTemplate(ctx context.Context, id string, req *UpdatePeriodicTemplateRequest, operatorID string) error {
	var template models.PeriodicWorkOrderTemplate
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("周期性工单模板不存在")
		}
		return fmt.Errorf("查询周期性工单模板失败: %w", err)
	}

	updates := map[string]interface{}{
		"updated_by": operatorID,
	}

	if req.TemplateName != nil {
		updates["template_name"] = *req.TemplateName
	}
	if req.WorkOrderTitle != nil {
		updates["work_order_title"] = *req.WorkOrderTitle
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.CategoryID != nil {
		updates["category_id"] = *req.CategoryID
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.CronExpression != nil {
		// 验证Cron表达式（使用6字段格式：秒 分 时 日 月 周）
		if _, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor).Parse(*req.CronExpression); err != nil {
			return fmt.Errorf("无效的Cron表达式: %w", err)
		}
		updates["cron_expression"] = *req.CronExpression
	}
	if req.AssignType != nil {
		updates["assign_type"] = *req.AssignType
	}
	if req.AssignTargetID != nil {
		updates["assign_target_id"] = *req.AssignTargetID
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}
	if req.NotifyAssignee != nil {
		updates["notify_assignee"] = *req.NotifyAssignee
	}

	if err := s.db.WithContext(ctx).Model(&template).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新周期性工单模板失败: %w", err)
	}

	return nil
}

// DeletePeriodicTemplate 删除周期性工单模板
func (s *WorkOrderService) DeletePeriodicTemplate(ctx context.Context, id string) error {
	var template models.PeriodicWorkOrderTemplate
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("周期性工单模板不存在")
		}
		return fmt.Errorf("查询周期性工单模板失败: %w", err)
	}

	// 检查是否启用
	if template.IsEnabled {
		return fmt.Errorf("请先禁用周期性工单模板")
	}

	// 删除执行记录
	if err := s.db.WithContext(ctx).Where("template_id = ?", id).Delete(&models.PeriodicWorkOrderLog{}).Error; err != nil {
		return fmt.Errorf("删除执行记录失败: %w", err)
	}

	// 删除模板
	if err := s.db.WithContext(ctx).Delete(&template).Error; err != nil {
		return fmt.Errorf("删除周期性工单模板失败: %w", err)
	}

	return nil
}

// EnablePeriodicTemplate 启用周期性工单模板
func (s *WorkOrderService) EnablePeriodicTemplate(ctx context.Context, id string, operatorID string) error {
	var template models.PeriodicWorkOrderTemplate
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("周期性工单模板不存在")
		}
		return fmt.Errorf("查询周期性工单模板失败: %w", err)
	}

	// 计算下次执行时间（使用6字段格式，带时区支持）
	// time.Local 已在 main.go 中设置为 Asia/Shanghai
	c := cron.New(cron.WithSeconds(), cron.WithLocation(time.Local))
	entryID, err := c.AddFunc(template.CronExpression, func() {})
	if err != nil {
		return fmt.Errorf("解析Cron表达式失败: %w", err)
	}
	// 必须启动 cron 才能获取正确的 Next 时间
	c.Start()
	defer c.Stop()
	nextRun := c.Entry(entryID).Next

	// DEBUG: 打印计算结果
	zoneName, offset := nextRun.Zone()
	log.Printf("[DEBUG] 计算下次执行时间: Cron=%s, Next=%s (Zone=%s, Offset=%d)",
		template.CronExpression, nextRun.Format("2006-01-02 15:04:05 -07:00"),
		zoneName, offset)

	// 更新模板
	updates := map[string]interface{}{
		"is_enabled": true,
		"next_run_at": nextRun,
		"updated_by":  operatorID,
	}

	if err := s.db.WithContext(ctx).Model(&template).Updates(updates).Error; err != nil {
		return fmt.Errorf("启用周期性工单模板失败: %w", err)
	}

	// TODO: 创建定时任务（需要集成到调度器中）

	return nil
}

// DisablePeriodicTemplate 禁用周期性工单模板
func (s *WorkOrderService) DisablePeriodicTemplate(ctx context.Context, id string, operatorID string) error {
	var template models.PeriodicWorkOrderTemplate
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("周期性工单模板不存在")
		}
		return fmt.Errorf("查询周期性工单模板失败: %w", err)
	}

	// 更新模板
	updates := map[string]interface{}{
		"is_enabled": false,
		"updated_by": operatorID,
	}

	if err := s.db.WithContext(ctx).Model(&template).Updates(updates).Error; err != nil {
		return fmt.Errorf("禁用周期性工单模板失败: %w", err)
	}

	// TODO: 停止定时任务（需要集成到调度器中）

	return nil
}

// GenerateWorkOrderFromTemplate 根据模板生成工单
func (s *WorkOrderService) GenerateWorkOrderFromTemplate(ctx context.Context, templateID string) (*models.WorkOrder, error) {
	var template models.PeriodicWorkOrderTemplate
	if err := s.db.WithContext(ctx).Where("id = ?", templateID).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("周期性工单模板不存在")
		}
		return nil, fmt.Errorf("查询周期性工单模板失败: %w", err)
	}

	// 替换标题中的变量
	title := s.replaceTemplateVariables(template.WorkOrderTitle)

	// 替换描述中的变量
	description := s.replaceTemplateVariables(template.Description)

	// 确定处理人
	var assigneeID *string
	var isAutoAssigned bool
	var dutyPoolID *string
	var dutyType string

	switch template.AssignType {
	case models.PeriodicAssignTypeDutyPool:
		// 获取当天值班人员（根据模板中选择的值班池筛选）
		dutyMembers, err := s.getTodayDutyMembers(ctx, template.AssignTargetID)
		if err != nil {
			return nil, fmt.Errorf("获取当天值班人员失败: %w", err)
		}
		if len(dutyMembers) == 0 {
			return nil, fmt.Errorf("所选值班池今天没有值班人员")
		}
		assigneeID = &dutyMembers[0].UserID
		isAutoAssigned = true
		dutyPoolID = &dutyMembers[0].PoolID
		dutyType = dutyMembers[0].DutyType
	case models.PeriodicAssignTypeManual:
		assigneeID = template.AssignTargetID
		isAutoAssigned = false
	case models.PeriodicAssignTypeRotation:
		// TODO: 实现轮询逻辑
		assigneeID = template.AssignTargetID
		isAutoAssigned = false
	}

	// 使用模板的创建者作为自动生成工单的提交者
	// 注意：CreatedBy 存储的是用户名，需要转换为用户ID（UUID）
	submitterID := ""
	if template.CreatedBy != "" {
		var submitter models.User
		if err := s.db.WithContext(ctx).Where("username = ?", template.CreatedBy).First(&submitter).Error; err == nil {
			submitterID = submitter.ID
		}
	}

	// 如果没有找到提交者，使用系统管理员用户
	if submitterID == "" {
		var adminUser models.User
		if err := s.db.WithContext(ctx).Where("username = ?", "admin").First(&adminUser).Error; err == nil {
			submitterID = adminUser.ID
		} else {
			return nil, fmt.Errorf("无法确定工单提交者：未找到模板创建者或系统管理员")
		}
	}

	// 创建工单
	workOrder := &models.WorkOrder{
		Title:          title,
		CategoryID:     template.CategoryID,
		Type:           template.Type,
		Priority:       template.Priority,
		Status:         models.WorkOrderStatusPending,
		Description:    description,
		SubmitterID:    submitterID,
		AssigneeID:     assigneeID,
		IsAutoAssigned: isAutoAssigned,
		DutyPoolID:     dutyPoolID,
		DutyType:       dutyType,
		AssignStrategy: string(template.AssignType),
	}

	// 设置审计字段
	workOrder.CreatedBy = template.CreatedBy
	workOrder.UpdatedBy = template.CreatedBy

	if err := s.db.WithContext(ctx).Create(workOrder).Error; err != nil {
		return nil, fmt.Errorf("创建工单失败: %w", err)
	}

	// 更新模板的生成计数
	s.db.WithContext(ctx).Model(&template).UpdateColumn("total_generated", gorm.Expr("total_generated + 1"))

	// 计算下次执行时间（使用6字段格式，带时区支持）
	// time.Local 已在 main.go 中设置为 Asia/Shanghai
	c := cron.New(cron.WithSeconds(), cron.WithLocation(time.Local))
	entryID, _ := c.AddFunc(template.CronExpression, func() {})
	c.Start()
	defer c.Stop()
	nextRun := c.Entry(entryID).Next
	s.db.WithContext(ctx).Model(&template).UpdateColumn("next_run_at", nextRun)

	// 记录执行日志
	execLog := &models.PeriodicWorkOrderLog{
		ID:         uuid.New().String(),
		TemplateID: templateID,
		WorkOrderID: workOrder.ID,
		ExecutedAt: time.Now(),
		Status:     "success",
		Result:     fmt.Sprintf("成功创建工单: %s", workOrder.WorkOrderNo),
	}
	s.db.WithContext(ctx).Create(execLog)

	return workOrder, nil
}

// replaceTemplateVariables 替换模板变量
func (s *WorkOrderService) replaceTemplateVariables(text string) string {
	now := time.Now()

	// 定义变量映射
	variables := map[string]string{
		"{date}":    now.Format("2006-01-02"),
		"{year}":    now.Format("2006"),
		"{month}":   now.Format("01"),
		"{day}":     now.Format("02"),
		"{weekday}": getWeekdayName(now.Weekday()),
		"{hour}":    now.Format("15"),
		"{minute}":  now.Format("04"),
	}

	// 替换变量
	result := text
	for key, value := range variables {
		result = strings.ReplaceAll(result, key, value)
	}

	return result
}

// getWeekdayName 获取星期名称
func getWeekdayName(wd time.Weekday) string {
	weekdays := map[time.Weekday]string{
		time.Sunday:    "周日",
		time.Monday:    "周一",
		time.Tuesday:   "周二",
		time.Wednesday: "周三",
		time.Thursday:  "周四",
		time.Friday:    "周五",
		time.Saturday:  "周六",
	}
	return weekdays[wd]
}

// GetPeriodicLogs 获取周期性工单执行记录列表
func (s *WorkOrderService) GetPeriodicLogs(ctx context.Context, templateID string) ([]models.PeriodicWorkOrderLog, error) {
	var logs []models.PeriodicWorkOrderLog

	if err := s.db.WithContext(ctx).
		Preload("Template").
		Preload("WorkOrder").
		Where("template_id = ?", templateID).
		Order("executed_at DESC").
		Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("查询执行记录失败: %w", err)
	}

	return logs, nil
}

// GetWorkOrderConfig 获取工单配置
func (s *WorkOrderService) GetWorkOrderConfig(ctx context.Context) (*models.WorkOrderConfig, error) {
	var config models.WorkOrderConfig

	// 尝试从数据库获取配置
	err := s.db.WithContext(ctx).Where("id = ?", "default").First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建默认配置
			config = models.WorkOrderConfig{
				ID:                     "default",
				AutoAssignEnabled:       true,
				AutoAssignTarget:        "duty_pool",
				AutoAssignStrategy:      "assign_one",
				AutoCloseDays:           7,
				AllowUserClose:          false,
				NotificationEnabled:     true,
				EmailNotification:       false,
				SmsNotification:         false,
				RatingEnabled:           true,
				KnowledgeConvertEnabled: true,
			}
			s.db.WithContext(ctx).Create(&config)
		} else {
			return nil, fmt.Errorf("查询工单配置失败: %w", err)
		}
	}

	return &config, nil
}

// UpdateWorkOrderConfig 更新工单配置
func (s *WorkOrderService) UpdateWorkOrderConfig(ctx context.Context, config *models.WorkOrderConfig, operatorID string) error {
	config.ID = "default"

	// 使用 GORM 的 OnConflict 来处理 upsert
	if err := s.db.WithContext(ctx).
		Where("id = ?", "default").
		Assign(config).
		FirstOrCreate(&config).Error; err != nil {
		return fmt.Errorf("更新工单配置失败: %w", err)
	}

	return nil
}

// ==================== JSON 序列化辅助函数 ====================

func (s *WorkOrderService) toJSON(v interface{}) string {
	bytes, _ := json.Marshal(v)
	return string(bytes)
}
