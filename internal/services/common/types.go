package common

import (
	"context"

	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// PageResult 分页结果
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Current  int         `json:"current"`
	PageSize int         `json:"pageSize"`
}

// ListParams 列表查询基础参数
// 嵌入 base.BaseListRequest 使 json 顶层自动获得 orderByColumn/isAsc 字段,
// 所有嵌入 ListParams 的 XxxListParams 自动获得服务端排序能力。
type ListParams struct {
	base.BaseListRequest
}

// DefaultListParams 默认列表参数
func DefaultListParams() ListParams {
	return ListParams{
		BaseListRequest: base.BaseListRequest{
			Current:  1,
			PageSize: 10,
		},
	}
}

// BaseService 基础服务接口，定义通用CRUD操作
type BaseService[T any] interface {
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*T, error)
	List(ctx context.Context, params ListParams) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
}

// OperableService 可操作的服务接口（支持启用/禁用）
type OperableService interface {
	UpdateStatus(ctx context.Context, id string, status int) error
}
