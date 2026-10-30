package utils

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// PaginationParams 分页参数
type PaginationParams struct {
	Page     int
	PageSize int
}

// ParsePagination 解析分页参数
func ParsePagination(page, pageSize int) PaginationParams {
	if page <= 0 {
		page = DefaultPage
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
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
