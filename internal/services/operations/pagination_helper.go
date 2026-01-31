package operations

import (
	"math"
)

// PaginationParams 分页参数
type PaginationParams struct {
	Current  int
	PageSize int
}

// PageResult 分页结果
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Current  int         `json:"current"`
	PageSize int         `json:"pageSize"`
}

// extractPagination 从参数中提取分页信息
func extractPagination(params map[string]interface{}) PaginationParams {
	current := 1
	pageSize := 10

	// 尝试从不同类型的参数中获取 current
	if c, ok := params["current"].(int); ok {
		current = c
	} else if c, ok := params["current"].(float64); ok {
		current = int(c)
	}

	// 尝试从不同类型的参数中获取 pageSize
	if ps, ok := params["pageSize"].(int); ok {
		pageSize = ps
	} else if ps, ok := params["pageSize"].(float64); ok {
		pageSize = int(ps)
	}

	// 限制 pageSize 范围
	pageSize = int(math.Max(10, math.Min(100, float64(pageSize))))

	return PaginationParams{
		Current:  current,
		PageSize: pageSize,
	}
}

// calculateOffset 计算偏移量
func calculateOffset(params PaginationParams) int {
	return (params.Current - 1) * params.PageSize
}
