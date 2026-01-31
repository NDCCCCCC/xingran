package requests

import (
	"github.com/ruoyi-next/ruoyi-go-backend/internal/constants"
)

// PaginationParams 分页参数（基础结构）
type PaginationParams struct {
	Current  int `json:"current"`  // 当前页码
	PageSize int `json:"pageSize"` // 每页数量
}

// GetPagination 获取分页参数，应用默认值和限制
func (p *PaginationParams) GetPagination() (current, pageSize int) {
	current = p.Current
	if current < 1 {
		current = constants.DefaultCurrent
	}
	pageSize = p.PageSize
	if pageSize < constants.MinPageSize {
		pageSize = constants.DefaultPageSize
	}
	if pageSize > constants.MaxPageSize {
		pageSize = constants.MaxPageSize
	}
	return current, pageSize
}

// GetOffset 计算分页偏移量
func (p *PaginationParams) GetOffset() int {
	current, pageSize := p.GetPagination()
	return (current - 1) * pageSize
}

// BatchOperationRequest 批量操作请求（基础结构）
type BatchOperationRequest struct {
	IDs    []string `json:"ids" binding:"required"`    // 要操作的ID列表
	Action string   `json:"action" binding:"required"` // 操作类型：delete=删除
}

// StatusRequest 状态筛选请求（基础结构）
type StatusRequest struct {
	Status *int `json:"status"` // 状态（0=正常/启用 1=停用/禁用）
}

// HasStatus 检查是否有状态筛选
func (s *StatusRequest) HasStatus() bool {
	return s.Status != nil
}

// GetStatus 获取状态值，如果未设置返回默认值
func (s *StatusRequest) GetStatus(defaultValue int) int {
	if s.Status != nil {
		return *s.Status
	}
	return defaultValue
}
