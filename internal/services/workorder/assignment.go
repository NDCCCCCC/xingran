package workorder

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// AssignmentService 分配服务
type AssignmentService struct {
	db *gorm.DB
}

// NewAssignmentService 创建分配服务
func NewAssignmentService(db *gorm.DB) *AssignmentService {
	return &AssignmentService{db: db}
}

// AssignRequest 分配工单请求
type AssignRequest struct {
	AssigneeID string `json:"assigneeId" binding:"required,uuid"`
	Remark     string `json:"remark"`
}

// Assign 分配工单
func (s *AssignmentService) Assign(ctx context.Context, id string, req *AssignRequest, operatorID string) error {
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

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新工单
		updates := map[string]interface{}{
			"assignee_id": req.AssigneeID,
			"updated_by":  operatorID,
		}

		if err := tx.Model(&workOrder).Updates(updates).Error; err != nil {
			return fmt.Errorf("分配工单失败: %w", err)
		}

		// 记录历史
		baseService := NewBaseService(s.db)
		if err := baseService.recordHistory(tx, workOrder.ID, "assign", "assignee_id", oldAssignee, req.AssigneeID, req.Remark, operatorID); err != nil {
			applogger.Warnf("记录工单分配历史失败: %v", err)
		}

		return nil
	})
}

// AssignToTodayDuty 分配给当天值班人员
func (s *AssignmentService) AssignToTodayDuty(ctx context.Context, id string, operatorID string) error {
	// 获取当天值班人员
	dutyMembers, err := s.getTodayDutyMembers(ctx, nil)
	if err != nil {
		return fmt.Errorf("今天没有值班人员")
	}

	if len(dutyMembers) == 0 {
		return fmt.Errorf("今天没有值班人员")
	}

	// 简化处理：使用第一个值班人员
	assigneeID := dutyMembers[0].UserID

	req := &AssignRequest{
		AssigneeID: assigneeID,
		Remark:     "自动分配给当天值班人员",
	}

	return s.Assign(ctx, id, req, operatorID)
}

// UpdateStatusRequest 更新状态请求
type UpdateStatusRequest struct {
	Status   models.WorkOrderStatus `json:"status" binding:"required"`
	Solution string                 `json:"solution"`
	Remark   string                 `json:"remark"`
}

// UpdateStatus 更新工单状态
func (s *AssignmentService) UpdateStatus(ctx context.Context, id string, req *UpdateStatusRequest, operatorID string) error {
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

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"status":     req.Status,
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
			return fmt.Errorf("更新工单状态失败: %w", err)
		}

		// 记录历史
		remark := req.Remark
		if remark == "" {
			remark = getStatusName(req.Status)
		}
		baseService := NewBaseService(s.db)
		if err := baseService.recordHistory(tx, workOrder.ID, "update_status", "status",
			strconv.Itoa(int(workOrder.Status)), strconv.Itoa(int(req.Status)), remark, operatorID); err != nil {
			applogger.Warnf("记录工单状态变更历史失败: %v", err)
		}

		return nil
	})
}

// TodayDutyMember 当天值班成员
type TodayDutyMember struct {
	ScheduleID string `json:"scheduleId"`
	PoolID     string `json:"poolId"`
	PoolName   string `json:"poolName"`
	UserID     string `json:"userId"`
	Username   string `json:"username"`
	Phone      string `json:"phone"`
	DutyType   string `json:"dutyType"`
}

// getTodayDutyMembers 获取当天值班人员
func (s *AssignmentService) getTodayDutyMembers(ctx context.Context, poolID *string) ([]TodayDutyMember, error) {
	var members []TodayDutyMember

	// 查询今天的排班记录
	today := time.Now().Local().Format("2006-01-02")

	query := `
		SELECT ds.id as schedule_id, ds.pool_id, dp.pool_name, ds.user_id,
		       u.username, u.phone, ds.duty_type
		FROM sys_duty_schedule ds
		LEFT JOIN sys_duty_pool dp ON ds.pool_id = dp.id
		LEFT JOIN sys_user u ON ds.user_id = u.id
		WHERE DATE(ds.schedule_date) = ? AND ds.status = ? AND ds.deleted_at IS NULL
	`

	args := []interface{}{today, int(models.DutyStatusNormal)}

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
		models.WorkOrderStatusClosed: {},
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
