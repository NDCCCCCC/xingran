package requests

import (
	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// RoleListParams 角色列表查询参数
type RoleListParams struct {
	base.BaseListRequest
	RoleName string `json:"roleName"`
	RoleKey  string `json:"roleKey"`
	Status   string `json:"status"`
}

// DefaultRoleListParams 默认列表参数
func DefaultRoleListParams() RoleListParams {
	return RoleListParams{
		BaseListRequest: base.BaseListRequest{
			Current:  constants.DefaultCurrent,
			PageSize: constants.DefaultPageSize,
		},
	}
}

// GetPagination 获取分页参数
func (p *RoleListParams) GetPagination() (current, pageSize int) {
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
func (p *RoleListParams) GetOffset() int {
	current, pageSize := p.GetPagination()
	return (current - 1) * pageSize
}

// RoleCreateRequest 创建角色请求
type RoleCreateRequest struct {
	RoleName          string            `json:"roleName" binding:"required"`
	RoleKey           string            `json:"roleKey" binding:"required"`
	RoleSort          int               `json:"roleSort"`
	DataScope         models.DataScope  `json:"dataScope"`
	MenuCheckStrictly *bool             `json:"menuCheckStrictly"`
	DeptCheckStrictly *bool             `json:"deptCheckStrictly"`
	Status            models.RoleStatus `json:"status"`
	Remark            *string           `json:"remark"`
	MenuIds           []string          `json:"menuIds"`
	DeptIds           []string          `json:"deptIds"`
}

// RoleUpdateRequest 更新角色请求
type RoleUpdateRequest struct {
	ID                string            `json:"id"` // ID 从 URL 参数获取，不在请求体验证
	RoleName          string            `json:"roleName" binding:"required"`
	RoleKey           string            `json:"roleKey" binding:"required"`
	RoleSort          int               `json:"roleSort"`
	DataScope         models.DataScope  `json:"dataScope"`
	MenuCheckStrictly *bool             `json:"menuCheckStrictly"`
	DeptCheckStrictly *bool             `json:"deptCheckStrictly"`
	Status            models.RoleStatus `json:"status"`
	Remark            *string           `json:"remark"`
	MenuIds           []string          `json:"menuIds"`
	DeptIds           []string          `json:"deptIds"`
}
