package requests

import (
	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// PaginationParams 分页参数（基础结构）
// 嵌入 base.BaseListRequest 使 json 顶层自动出现 orderByColumn/isAsc 字段,
// 所有嵌入 PaginationParams 的 XxxListRequest 自动获得服务端排序能力。
type PaginationParams struct {
	base.BaseListRequest
}

// GetPagination 获取分页参数，应用默认值和限制
func (p *PaginationParams) GetPagination() (current, pageSize int) {
	return p.GetPaginationWithMax(constants.MaxListPageSize)
}

// GetPaginationWithMax 获取分页参数并按调用方指定的上限钳制 pageSize。
//
// v129-recheck C-4：workstation List 的消费方（CAD 平面图编辑器 / 3D 视图）
// 以 pageSize:1000 请求楼层全集，迁移时误用 MaxListPageSize=100 上限导致
// >100 工位的楼层静默截断。恢复迁移前 MaxOptionsPageSize=10000 口径
// （current<1 守卫保留——修复迁移前 current=0 产生负 offset 的隐患）。
func (p *PaginationParams) GetPaginationWithMax(maxPageSize int) (current, pageSize int) {
	current = p.Current
	if current < 1 {
		current = constants.DefaultCurrent
	}
	pageSize = p.PageSize
	if pageSize < constants.MinPageSize {
		pageSize = constants.DefaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
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
