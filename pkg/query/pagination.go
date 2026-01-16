// Package query 提供通用的查询构建和分页辅助函数
package query

import "gorm.io/gorm"

// PaginatedResult 分页结果
type PaginatedResult struct {
	Total      int64       `json:"total"`
	Current    int         `json:"current"`
	PageSize   int         `json:"pageSize"`
	TotalPages int         `json:"totalPages"`
	Data       interface{} `json:"data"`
}

// NewPaginatedResult 创建分页结果
func NewPaginatedResult(total int64, current, pageSize int, data interface{}) *PaginatedResult {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &PaginatedResult{
		Total:      total,
		Current:    current,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Data:       data,
	}
}

// PaginationRequest 分页请求
type PaginationRequest struct {
	Current  int `json:"current" binding:"min=1"`
	PageSize int `json:"pageSize" binding:"min=1,max=100"`
}

// Normalize 规范化分页参数
func (p *PaginationRequest) Normalize() {
	if p.Current <= 0 {
		p.Current = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 10
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
}

// GetOffset 获取偏移量
func (p *PaginationRequest) GetOffset() int {
	p.Normalize()
	return (p.Current - 1) * p.PageSize
}

// ListParams 列表查询参数（分页 + 排序）
type ListParams struct {
	PaginationRequest
	OrderBy  string `json:"orderBy"`
	OrderDir string `json:"orderDir"` // ASC 或 DESC
}

// Normalize 规范化列表参数
func (p *ListParams) Normalize() {
	p.PaginationRequest.Normalize()

	// 默认排序字段
	if p.OrderBy == "" {
		p.OrderBy = "created_at"
	}

	// 默认排序方向
	if p.OrderDir != "ASC" && p.OrderDir != "DESC" {
		p.OrderDir = "DESC"
	}
}

// ApplyOrder 应用排序到查询
func (p *ListParams) ApplyOrder(query *gorm.DB) *gorm.DB {
	p.Normalize()
	if p.OrderBy != "" {
		return query.Order(p.OrderBy + " " + p.OrderDir)
	}
	return query
}

// ApplyPagination 应用分页到查询
func (p *ListParams) ApplyPagination(query *gorm.DB) *gorm.DB {
	p.Normalize()
	return query.Offset(p.GetOffset()).Limit(p.PageSize)
}

// QueryExecutor 查询执行器接口
type QueryExecutor interface {
	// Execute 执行查询并返回分页结果
	Execute(query *gorm.DB, dest interface{}) (*PaginatedResult, error)
}

// DefaultQueryExecutor 默认查询执行器
type DefaultQueryExecutor struct {
	Current  int
	PageSize int
}

// NewDefaultQueryExecutor 创建默认查询执行器
func NewDefaultQueryExecutor(current, pageSize int) *DefaultQueryExecutor {
	return &DefaultQueryExecutor{
		Current:  current,
		PageSize: pageSize,
	}
}

// Execute 执行查询
func (e *DefaultQueryExecutor) Execute(query *gorm.DB, dest interface{}) (*PaginatedResult, error) {
	total, err := CountAndQuery(query, dest, e.Current, e.PageSize)
	if err != nil {
		return nil, err
	}

	return NewPaginatedResult(total, e.Current, e.PageSize, dest), nil
}
