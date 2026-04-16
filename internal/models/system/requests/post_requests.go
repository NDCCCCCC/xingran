package requests

import (
	"github.com/ruoyi-next/ruoyi-go-backend/internal/constants"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// PostListParams 岗位列表查询参数
type PostListParams struct {
	PostCode *string `json:"postCode"`
	PostName *string `json:"postName"`
	Status   *int    `json:"status"`
	Current  int     `json:"current"`
	PageSize int     `json:"pageSize"`
}

// DefaultPostListParams 默认列表参数
func DefaultPostListParams() PostListParams {
	return PostListParams{
		Current:  constants.DefaultCurrent,
		PageSize: constants.DefaultPageSize,
	}
}

// GetPagination 获取分页参数
func (p *PostListParams) GetPagination() (current, pageSize int) {
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

// GetOffset 计算偏移量
func (p *PostListParams) GetOffset() int {
	current, pageSize := p.GetPagination()
	return (current - 1) * pageSize
}

// PostCreateRequest 创建岗位请求
type PostCreateRequest struct {
	PostCode string            `json:"postCode" binding:"required"`
	PostName string            `json:"postName" binding:"required"`
	PostSort int               `json:"postSort"`
	Status   models.PostStatus `json:"status"`
	Remark   *string           `json:"remark"`
}

// PostUpdateRequest 更新岗位请求
type PostUpdateRequest struct {
	ID       string            `json:"id" binding:"required"`
	PostName string            `json:"postName" binding:"required"`
	PostSort int               `json:"postSort"`
	Status   models.PostStatus `json:"status"`
	Remark   *string           `json:"remark"`
}
