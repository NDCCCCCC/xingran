package portwrite

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// Phase 127 前置（PWX-04）：端口写审计查询 API 的 service 层。
//
// 纯读查询：分页 + 过滤（deviceId/portId/operator/action/status/时间范围），
// 返回 sys_port_write_audit 行集（含 before/after JSON 与设备原始响应）。

// AuditListRequest 审计查询请求（Phase 127 对齐 webfe 预期契约）。
type AuditListRequest struct {
	DeviceID string `json:"deviceId,omitempty"`
	PortID   string `json:"portId,omitempty"`
	Operator string `json:"operator,omitempty"`
	Action   string `json:"action,omitempty"`
	// Status 过滤：succeeded | failed（可空 = 全部）
	Status string `json:"status,omitempty"`
	// BatchID Phase 122 收尾：批次精确对账过滤（migration 232；历史行为 NULL 不命中）
	BatchID string `json:"batchId,omitempty"`
	// StartTime/EndTime RFC3339 或 "2006-01-02 15:04:05"（可空）
	StartTime *time.Time `json:"startTime,omitempty"`
	EndTime   *time.Time `json:"endTime,omitempty"`
	Current   int        `json:"current,omitempty"`  // 默认 1
	PageSize  int        `json:"pageSize,omitempty"` // 默认 20，上限 100
}

// AuditListResult 审计查询分页响应（项目惯例 list/total/current/pageSize）。
type AuditListResult struct {
	List     []models.PortWriteAudit `json:"list"`
	Total    int64                   `json:"total"`
	Current  int                     `json:"current"`
	PageSize int                     `json:"pageSize"`
}

// normalizeAuditQuery 入参兜底（分页默认值 + 上限）。
func normalizeAuditQuery(req AuditListRequest) AuditListRequest {
	if req.Current < 1 {
		req.Current = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	return req
}

// ListAudit 分页查询端口写审计（Phase 127 前置）。
//
// 纯 GORM chainable 查询；过滤字段全部等值匹配（时间用 >= / <= 范围），
// 按 created_at 倒序。行集直接序列化 PortWriteAudit（JSON tag 已就绪，
// beforeValue/afterValue 为 RawMessage 原样透传给前端 JSON viewer）。
func (s *portWriteServiceImpl) ListAudit(ctx context.Context, req AuditListRequest) (*AuditListResult, error) {
	req = normalizeAuditQuery(req)

	q := s.db.WithContext(ctx).Model(&models.PortWriteAudit{})
	if req.DeviceID != "" {
		q = q.Where("device_id = ?", req.DeviceID)
	}
	if req.PortID != "" {
		q = q.Where("port_id = ?", req.PortID)
	}
	if req.Operator != "" {
		q = q.Where("operator = ?", req.Operator)
	}
	if req.Action != "" {
		q = q.Where("action = ?", req.Action)
	}
	if req.Status != "" {
		q = q.Where("status = ?", req.Status)
	}
	if req.BatchID != "" {
		q = q.Where("batch_id = ?", req.BatchID)
	}
	if req.StartTime != nil {
		q = q.Where("created_at >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		q = q.Where("created_at <= ?", *req.EndTime)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count audit: %w", err)
	}

	list := make([]models.PortWriteAudit, 0, req.PageSize)
	if err := q.Order("created_at DESC").
		Offset((req.Current - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("query audit: %w", err)
	}
	return &AuditListResult{List: list, Total: total, Current: req.Current, PageSize: req.PageSize}, nil
}
