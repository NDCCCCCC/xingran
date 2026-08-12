package workorder

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// BaseService 基础工单服务
type BaseService struct {
	db *gorm.DB
}

// NewBaseService 创建基础服务
func NewBaseService(db *gorm.DB) *BaseService {
	return &BaseService{db: db}
}

// WorkOrderStatusStatistics 工单按状态的聚合统计。
// status: 0=待处理 1=处理中 2=已完成 3=已关闭 4=已拒绝(models.WorkOrderStatus)。
type WorkOrderStatusStatistics struct {
	Total      int64 `json:"total"`
	Pending    int64 `json:"pending"`    // status = 0
	Processing int64 `json:"processing"` // status = 1
	Completed  int64 `json:"completed"`  // status = 2
	Closed     int64 `json:"closed"`     // status = 3
}

// GetStatusStatistics 统计工单总数及各状态计数。
// 用条件聚合(SUM CASE)避免「按当前页 list 计算统计」的错误——旧前端用当前页(默认 10 条)
// 的 list.filter().length 算统计,多页时严重偏小。base query 与 GetList 一致。
func (s *BaseService) GetStatusStatistics(ctx context.Context) (*WorkOrderStatusStatistics, error) {
	var result WorkOrderStatusStatistics
	err := s.db.WithContext(ctx).
		Model(&models.WorkOrder{}).
		Select(
			"COUNT(*) AS total",
			"SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END) AS pending",
			"SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) AS processing",
			"SUM(CASE WHEN status = 2 THEN 1 ELSE 0 END) AS completed",
			"SUM(CASE WHEN status = 3 THEN 1 ELSE 0 END) AS closed",
		).
		Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("统计工单状态失败: %w", err)
	}
	return &result, nil
}

// ListRequest 工单列表查询请求
type ListRequest struct {
	base.BaseListRequest
	WorkOrderNo string `json:"workOrderNo"`
	Title       string `json:"title"`
	CategoryID  string `json:"categoryId"`
	Type        string `json:"type"`
	Priority    *int   `json:"priority"`
	Status      *int   `json:"status"`
	SubmitterID string `json:"submitterId"`
	AssigneeID  string `json:"assigneeId"`
	DeptID      string `json:"deptId"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
}

// GetList 分页查询工单列表
func (s *BaseService) GetList(ctx context.Context, req *ListRequest) ([]models.WorkOrder, int64, error) {
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

	// 用户排序(白名单)优先,无 OrderByColumn 时保留 created_at DESC 默认
	query = base.ApplySort(query, req.BaseListRequest, workOrderAllowedSortFields)
	if req.OrderByColumn == "" {
		query = query.Order("created_at DESC")
	}

	// 预加载关联数据
	if err := query.
		Preload("Category").
		Preload("Submitter").
		Preload("Assignee").
		Preload("Dept").
		Limit(pageSize).
		Offset(offset).
		Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("查询工单列表失败: %w", err)
	}

	return list, total, nil
}

// workOrderAllowedSortFields 工单可排序字段白名单(对应 work_order 表列名)。
var workOrderAllowedSortFields = map[string]string{
	"workOrderNo":  "work_order_no",
	"title":        "title",
	"categoryId":   "category_id",
	"type":         "type",
	"priority":     "priority",
	"status":       "status",
	"submitterId":  "submitter_id",
	"assigneeId":   "assignee_id",
	"createdAt":    "created_at",
	"updatedAt":    "updated_at",
	"completedAt":  "completed_at",
}

// GetMyPendingRequest 获取待办工单请求
type GetMyPendingRequest struct {
	Limit int `json:"limit"` // 限制返回数量，默认5
}

// GetMyPending 获取当前用户的待办工单（待处理和处理中）
func (s *BaseService) GetMyPending(ctx context.Context, req *GetMyPendingRequest, userID string) ([]models.WorkOrder, int64, error) {
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

// GetByID 获取工单详情
func (s *BaseService) GetByID(ctx context.Context, id string) (*models.WorkOrder, error) {
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

// CreateRequest 创建工单请求
type CreateRequest struct {
	Title             string                   `json:"title" binding:"required,max=200"`
	CategoryID        string                   `json:"categoryId" binding:"required,uuid"`
	Type              models.WorkOrderType     `json:"type" binding:"required"`
	Priority          models.WorkOrderPriority `json:"priority"`
	Description       string                   `json:"description"`
	DeptID            *string                  `json:"deptId"`
	ExpectedResolveAt *string                  `json:"expectedResolveAt"` // YYYY-MM-DD HH:mm:ss
	AttachmentIDs     string                   `json:"attachmentIds"`
	AssigneeID        *string                  `json:"assigneeId"` // 手动指定处理人
}

// Create 创建工单
func (s *BaseService) Create(ctx context.Context, req *CreateRequest, submitterID string) (*models.WorkOrder, error) {
	var workOrder *models.WorkOrder

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		newWorkOrder := &models.WorkOrder{
			Title:         req.Title,
			CategoryID:    req.CategoryID,
			Type:          req.Type,
			Priority:      req.Priority,
			Status:        models.WorkOrderStatusPending,
			Description:   req.Description,
			SubmitterID:   submitterID,
			DeptID:        req.DeptID,
			AttachmentIDs: req.AttachmentIDs,
			AssigneeID:    req.AssigneeID,
		}

		// 处理期望解决时间
		if req.ExpectedResolveAt != nil && *req.ExpectedResolveAt != "" {
			// Time parsing logic (placeholder for future implementation)
			_ = req.ExpectedResolveAt
		}

		// 创建工单
		if err := tx.Create(newWorkOrder).Error; err != nil {
			return fmt.Errorf("创建工单失败: %w", err)
		}

		workOrder = newWorkOrder
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 重新加载关联数据
	return s.GetByID(ctx, workOrder.ID)
}

// UpdateRequest 更新工单请求
type UpdateRequest struct {
	ID                string                    `json:"id" binding:"required,uuid"`
	Title             *string                   `json:"title"`
	CategoryID        *string                   `json:"categoryId"`
	Type              *models.WorkOrderType     `json:"type"`
	Priority          *models.WorkOrderPriority `json:"priority"`
	Status            *models.WorkOrderStatus   `json:"status"`
	Description       *string                   `json:"description"`
	Solution          *string                   `json:"solution"`
	AssigneeID        *string                   `json:"assigneeId"`
	DeptID            *string                   `json:"deptId"`
	ExpectedResolveAt *string                   `json:"expectedResolveAt"`
	AttachmentIDs     *string                   `json:"attachmentIds"`
	ResolvedAt        *string                   `json:"resolvedAt"`
	ClosedAt          *string                   `json:"closedAt"`
}

// Update 更新工单
func (s *BaseService) Update(ctx context.Context, req *UpdateRequest, operatorID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var workOrder models.WorkOrder
		if err := tx.Where("id = ?", req.ID).First(&workOrder).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("工单不存在")
			}
			return fmt.Errorf("查询工单失败: %w", err)
		}

		// 更新字段
		if req.Title != nil {
			workOrder.Title = *req.Title
		}
		if req.CategoryID != nil {
			workOrder.CategoryID = *req.CategoryID
		}
		if req.Type != nil {
			workOrder.Type = *req.Type
		}
		if req.Priority != nil {
			workOrder.Priority = *req.Priority
		}
		if req.Status != nil {
			workOrder.Status = *req.Status
		}
		if req.Description != nil {
			workOrder.Description = *req.Description
		}
		if req.Solution != nil {
			workOrder.Solution = *req.Solution
		}
		if req.AssigneeID != nil {
			workOrder.AssigneeID = req.AssigneeID
		}
		if req.DeptID != nil {
			workOrder.DeptID = req.DeptID
		}
		if req.AttachmentIDs != nil {
			workOrder.AttachmentIDs = *req.AttachmentIDs
		}

		// 更新工单
		if err := tx.Save(&workOrder).Error; err != nil {
			return fmt.Errorf("更新工单失败: %w", err)
		}

		return nil
	})
}

// Delete 删除工单
func (s *BaseService) Delete(ctx context.Context, id string) error {
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

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除关联数据（评论、历史、评价）
		if err := tx.Where("work_order_id = ?", id).Delete(&models.WorkOrderComment{}).Error; err != nil {
			return fmt.Errorf("删除工单评论失败: %w", err)
		}
		if err := tx.Where("work_order_id = ?", id).Delete(&models.WorkOrderHistory{}).Error; err != nil {
			return fmt.Errorf("删除工单历史失败: %w", err)
		}
		if err := tx.Where("work_order_id = ?", id).Delete(&models.WorkOrderRating{}).Error; err != nil {
			return fmt.Errorf("删除工单评价失败: %w", err)
		}

		// 删除工单
		if err := tx.Delete(&workOrder).Error; err != nil {
			return fmt.Errorf("删除工单失败: %w", err)
		}

		return nil
	})
}

// BatchDelete 批量删除工单
func (s *BaseService) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("工单ID列表不能为空")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查是否所有工单都是待处理状态
		var count int64
		if err := tx.Model(&models.WorkOrder{}).Where("id IN ? AND status != ?", ids, models.WorkOrderStatusPending).Count(&count).Error; err != nil {
			return fmt.Errorf("检查工单状态失败: %w", err)
		}
		if count > 0 {
			return fmt.Errorf("只有待处理状态的工单可以删除")
		}

		// 删除关联数据
		if err := tx.Where("work_order_id IN ?", ids).Delete(&models.WorkOrderComment{}).Error; err != nil {
			return fmt.Errorf("删除工单评论失败: %w", err)
		}
		if err := tx.Where("work_order_id IN ?", ids).Delete(&models.WorkOrderHistory{}).Error; err != nil {
			return fmt.Errorf("删除工单历史失败: %w", err)
		}
		if err := tx.Where("work_order_id IN ?", ids).Delete(&models.WorkOrderRating{}).Error; err != nil {
			return fmt.Errorf("删除工单评价失败: %w", err)
		}

		// 删除工单
		if err := tx.Where("id IN ?", ids).Delete(&models.WorkOrder{}).Error; err != nil {
			return fmt.Errorf("批量删除工单失败: %w", err)
		}

		return nil
	})
}

// recordHistory 记录工单操作历史
func (s *BaseService) recordHistory(tx *gorm.DB, workOrderID, action, field, oldValue, newValue, remark, operatorID string) error {
	history := &models.WorkOrderHistory{
		ID:          uuid.New().String(),
		WorkOrderID: workOrderID,
		Action:      action,
		Field:       field,
		OldValue:    oldValue,
		NewValue:    newValue,
		Remark:      remark,
		OperatorID:  operatorID,
	}

	if err := tx.Create(history).Error; err != nil {
		return fmt.Errorf("记录操作历史失败: %w", err)
	}

	return nil
}
