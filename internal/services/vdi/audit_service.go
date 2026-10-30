package vdi

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// AuditService VDI 审计日志服务
type AuditService struct {
	db *gorm.DB
}

// NewAuditService 创建审计服务
func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

// RecordOperation 记录 VM 操作审计日志
func (s *AuditService) RecordOperation(ctx context.Context, req *AuditRequest) error {
	auditLog := &rpa.AuditLog{
		ResourceType:  rpa.ResourceType("vdi"),
		ResourceID:    req.VMID,
		Action:        rpa.AuditAction(req.Operation),
		OldValue:      req.OldValue,
		NewValue:      req.NewValue,
		OperatorID:    req.OperatorID,
		OperatorName:  req.OperatorName,
		IPAddress:     req.OperatorIP,
		Result:        rpa.AuditResult(req.Status),
		ErrorMessage:  req.ErrorMsg,
	}

	if err := s.db.WithContext(ctx).Create(auditLog).Error; err != nil {
		return errors.DatabaseError(err)
	}

	return nil
}

// auditLogAllowedSortFields VDI 审计日志可排序字段白名单（对应 rpa_audit_log 表列名）。
var auditLogAllowedSortFields = map[string]string{
	"action":    "action",
	"result":    "result",
	"createdAt": "created_at",
}

// QueryLogs 查询审计日志
func (s *AuditService) QueryLogs(ctx context.Context, req *AuditQueryRequest) ([]*rpa.AuditLog, int64, error) {
	var logs []*rpa.AuditLog
	var total int64

	query := s.db.WithContext(ctx).Model(&rpa.AuditLog{}).
		Where("resource_type = ?", "vdi")

	// 按 VMID 筛选
	if req.VMID != "" {
		query = query.Where("resource_id = ?", req.VMID)
	}

	// 按操作类型筛选
	if req.Operation != "" {
		query = query.Where("action = ?", req.Operation)
	}

	// 按时间范围筛选
	if !req.StartTime.IsZero() {
		query = query.Where("created_at >= ?", req.StartTime)
	}
	if !req.EndTime.IsZero() {
		query = query.Where("created_at <= ?", req.EndTime)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.DatabaseError(err)
	}

	// 分页查询：用户排序（白名单）优先，无 OrderByColumn 时保留 created_at DESC 默认
	offset := (req.Page - 1) * req.PageSize
	sortReq := base.BaseListRequest{
		Current:       req.Page,
		PageSize:      req.PageSize,
		OrderByColumn: req.OrderByColumn,
		IsAsc:         req.IsAsc,
	}
	query = base.ApplySort(query, sortReq, auditLogAllowedSortFields)
	if req.OrderByColumn == "" {
		query = query.Order("created_at DESC")
	}
	if err := query.Offset(offset).
		Limit(req.PageSize).
		Find(&logs).Error; err != nil {
		return nil, 0, errors.DatabaseError(err)
	}

	return logs, total, nil
}

// GetVMOperationsSummary 获取虚拟机操作摘要
func (s *AuditService) GetVMOperationsSummary(ctx context.Context, vmID string, days int) (*VMOpsSummary, error) {
	since := time.Now().AddDate(0, 0, -days)

	var result struct {
		TotalOperations int64
		SuccessCount    int64
		FailureCount    int64
	}

	// 总操作数
	if err := s.db.WithContext(ctx).Model(&rpa.AuditLog{}).
		Where("resource_type = ?", "vdi").
		Where("resource_id = ?", vmID).
		Where("created_at >= ?", since).
		Count(&result.TotalOperations).Error; err != nil {
		return nil, errors.DatabaseError(err)
	}

	// 成功操作数
	if err := s.db.WithContext(ctx).Model(&rpa.AuditLog{}).
		Where("resource_type = ?", "vdi").
		Where("resource_id = ?", vmID).
		Where("created_at >= ?", since).
		Where("result = ?", "success").
		Count(&result.SuccessCount).Error; err != nil {
		return nil, errors.DatabaseError(err)
	}

	// 失败操作数
	result.FailureCount = result.TotalOperations - result.SuccessCount

	// 获取最近操作记录
	var recentLogs []*rpa.AuditLog
	if err := s.db.WithContext(ctx).Model(&rpa.AuditLog{}).
		Where("resource_type = ?", "vdi").
		Where("resource_id = ?", vmID).
		Order("created_at DESC").
		Limit(10).
		Find(&recentLogs).Error; err != nil {
		return nil, errors.DatabaseError(err)
	}

	return &VMOpsSummary{
		VMID:           vmID,
		TotalOperations: int(result.TotalOperations),
		SuccessCount:    int(result.SuccessCount),
		FailureCount:    int(result.FailureCount),
		RecentLogs:      recentLogs,
		Period:          fmt.Sprintf("%d days", days),
	}, nil
}

// AuditRequest 审计日志请求
type AuditRequest struct {
	VMID         string                 // 虚拟机 ID
	Operation    string                 // 操作类型：create/delete/start/stop/restart/config_ip/rename/bind_user/unbind_user
	OldValue     map[string]interface{} // 操作前的值
	NewValue     map[string]interface{} // 操作后的值
	Status       string                 // 操作状态：success/failed
	ErrorMsg     string                 // 错误消息（失败时）
	OperatorID   string                 // 操作人 ID
	OperatorName string                 // 操作人名称
	OperatorIP   string                 // 操作人 IP
}

// AuditQueryRequest 审计日志查询请求
type AuditQueryRequest struct {
	VMID          string    // 虚拟机 ID
	Operation     string    // 操作类型
	StartTime     time.Time // 开始时间
	EndTime       time.Time // 结束时间
	Page          int       // 页码
	PageSize      int       // 每页大小
	OrderByColumn string    // 排序字段（白名单）
	IsAsc         *bool     // 升降序
}

// VMOpsSummary 虚拟机操作摘要
type VMOpsSummary struct {
	VMID            string          // 虚拟机 ID
	TotalOperations int             // 总操作数
	SuccessCount    int             // 成功数
	FailureCount    int             // 失败数
	RecentLogs      []*rpa.AuditLog // 最近操作记录
	Period          string          // 统计周期
}

// VM 操作类型常量
const (
	OpVMCreate     = "create"
	OpVMDelete     = "delete"
	OpVMStart      = "start"
	OpVMStop       = "stop"
	OpVMRestart    = "restart"
	OpVMConfigIP   = "config_ip"
	OpVMRename     = "rename"
	OpVMBindUser   = "bind_user"
	OpVMUnbindUser = "unbind_user"
)

// 操作状态常量
const (
	OpStatusSuccess = "success"
	OpStatusFailed  = "failed"
)
