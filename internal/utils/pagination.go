package utils

import "github.com/xingran-next/xingran-go-backend/internal/constants"

// PaginationParams 分页参数
type PaginationParams struct {
	Page     int
	PageSize int
}

// ParsePagination 解析分页参数。
//
// 默认值与上限统一引用 internal/constants,避免与本项目其它分页实现
// (operations/pagination_helper.go)取值分叉。
func ParsePagination(page, pageSize int) PaginationParams {
	if page <= 0 {
		page = constants.DefaultCurrent
	}
	if pageSize <= 0 {
		pageSize = constants.DefaultPageSize
	}
	if pageSize > constants.MaxListPageSize {
		pageSize = constants.MaxListPageSize
	}
	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
	}
}

// Offset 计算偏移量
func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Limit 返回限制数量
func (p PaginationParams) Limit() int {
	return p.PageSize
}

// BuildPaginationResponse 构建分页响应
func BuildPaginationResponse[T any](list []T, total int64, params PaginationParams) map[string]interface{} {
	return map[string]interface{}{
		"list":     list,
		"total":    total,
		"page":     params.Page,
		"pageSize": params.PageSize,
	}
}
