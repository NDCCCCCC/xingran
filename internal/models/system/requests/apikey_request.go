package requests

import (
	"github.com/xingran-next/xingran-go-backend/internal/constants"
)

// CreateAPIKeyRequest 创建API密钥请求
type CreateAPIKeyRequest struct {
	Name         string   `json:"name" binding:"required,min=1,max=100"`   // 密钥名称
	Description  *string  `json:"description"`                             // 描述信息
	Scopes       []string `json:"scopes" binding:"required"`               // 作用域数组（read, write, admin）
	InheritPerms bool     `json:"inheritPerms"`                            // 是否继承用户权限
	IPWhitelist  []string `json:"ipWhitelist"`                             // IP 白名单（支持 CIDR）
	ExpiresAt    *string  `json:"expiresAt"`                               // 过期时间（RFC3339 格式）
}

// UpdateAPIKeyRequest 更新API密钥请求
type UpdateAPIKeyRequest struct {
	ID           string   `json:"id" binding:"required"`                   // 密钥ID
	Name         *string  `json:"name"`                                    // 密钥名称
	Description  *string  `json:"description"`                             // 描述信息
	Scopes       []string `json:"scopes"`                                  // 作用域数组
	InheritPerms *bool    `json:"inheritPerms"`                            // 是否继承用户权限
	IPWhitelist  []string `json:"ipWhitelist"`                             // IP 白名单
	IsActive     *bool    `json:"isActive"`                                // 是否启用
	ExpiresAt    *string  `json:"expiresAt"`                               // 过期时间
}

// ListAPIKeysParams API密钥列表查询参数
type ListAPIKeysParams struct {
	Current       int     `json:"current"`                      // 当前页
	PageSize      int     `json:"pageSize"`                     // 每页数量
	Keyword       *string `json:"keyword"`                      // 关键词搜索（名称或密钥）
	Status        *bool   `json:"status"`                       // 状态筛选（true=启用, false=禁用）
	Scope         *string `json:"scope"`                        // 作用域筛选
	OrderByColumn string  `json:"orderByColumn"`                // 服务端排序字段（白名单）
	IsAsc         *bool   `json:"isAsc"`                        // 是否升序
}

// DefaultListAPIKeysParams 默认列表参数
func DefaultListAPIKeysParams() ListAPIKeysParams {
	return ListAPIKeysParams{
		Current:  constants.DefaultCurrent,
		PageSize: constants.DefaultPageSize,
	}
}

// GetPagination 获取分页参数
func (p *ListAPIKeysParams) GetPagination() (current, pageSize int) {
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
func (p *ListAPIKeysParams) GetOffset() int {
	current, pageSize := p.GetPagination()
	return (current - 1) * pageSize
}
