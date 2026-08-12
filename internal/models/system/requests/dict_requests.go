package requests

import (
	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// DictTypeListParams 字典类型列表查询参数
type DictTypeListParams struct {
	base.BaseListRequest
	DictName *string `json:"dictName"`
	DictType *string `json:"dictType"`
	Status   *int    `json:"status"`
}

// DefaultDictTypeListParams 默认列表参数
func DefaultDictTypeListParams() DictTypeListParams {
	return DictTypeListParams{
		BaseListRequest: base.BaseListRequest{
			Current:  constants.DefaultCurrent,
			PageSize: constants.DefaultPageSize,
		},
	}
}

// GetPagination 获取分页参数
func (p *DictTypeListParams) GetPagination() (current, pageSize int) {
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
func (p *DictTypeListParams) GetOffset() int {
	current, pageSize := p.GetPagination()
	return (current - 1) * pageSize
}

// DictTypeCreateRequest 创建字典类型请求
type DictTypeCreateRequest struct {
	DictName string  `json:"dictName" binding:"required"`
	DictType string  `json:"dictType" binding:"required"`
	Status   int     `json:"status"`
	Remark   *string `json:"remark"`
}

// DictTypeUpdateRequest 更新字典类型请求
type DictTypeUpdateRequest struct {
	ID       string  `json:"id"` // ID 从 URL 参数获取，不在请求体验证
	DictName string  `json:"dictName" binding:"required"`
	Status   int     `json:"status"`
	Remark   *string `json:"remark"`
}

// DictDataListParams 字典数据列表查询参数
type DictDataListParams struct {
	base.BaseListRequest
	DictType  string  `json:"dictType" binding:"required"`
	DictLabel *string `json:"dictLabel"`
	Status    *int    `json:"status"`
}

// DefaultDictDataListParams 默认列表参数
func DefaultDictDataListParams() DictDataListParams {
	return DictDataListParams{
		BaseListRequest: base.BaseListRequest{
			Current:  constants.DefaultCurrent,
			PageSize: constants.DefaultPageSize,
		},
	}
}

// GetPagination 获取分页参数
func (p *DictDataListParams) GetPagination() (current, pageSize int) {
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
func (p *DictDataListParams) GetOffset() int {
	current, pageSize := p.GetPagination()
	return (current - 1) * pageSize
}

// DictDataCreateRequest 创建字典数据请求
type DictDataCreateRequest struct {
	DictSort  int     `json:"dictSort"`
	DictLabel string  `json:"dictLabel" binding:"required"`
	DictValue string  `json:"dictValue" binding:"required"`
	DictType  string  `json:"dictType" binding:"required"`
	CssClass  *string `json:"cssClass"`
	ListClass *string `json:"listClass"`
	IsDefault bool    `json:"isDefault"`
	Status    int     `json:"status"`
	Remark    *string `json:"remark"`
}

// DictDataUpdateRequest 更新字典数据请求
type DictDataUpdateRequest struct {
	ID        string  `json:"id"` // ID 从 URL 参数获取，不在请求体验证
	DictSort  int     `json:"dictSort"`
	DictLabel string  `json:"dictLabel" binding:"required"`
	DictValue string  `json:"dictValue" binding:"required"`
	CssClass  *string `json:"cssClass"`
	ListClass *string `json:"listClass"`
	IsDefault bool    `json:"isDefault"`
	Status    int     `json:"status"`
	Remark    *string `json:"remark"`
}
