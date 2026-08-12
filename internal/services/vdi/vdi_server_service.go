package vdi

import (
	"context"
	"time"
)

// ============ 请求/响应 DTO ============

// CreateVDIServerRequest 创建VDI服务器请求
type CreateVDIServerRequest struct {
	Name     string `json:"name" validate:"required"`
	Endpoint string `json:"endpoint" validate:"required,url"`
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	TenantID int    `json:"tenant_id"`
	Status   int    `json:"status" validate:"oneof=0 1"`
}

// UpdateVDIServerRequest 更新VDI服务器请求
type UpdateVDIServerRequest struct {
	Name     *string `json:"name"`
	Endpoint *string `json:"endpoint" validate:"omitempty,url"`
	Username *string `json:"username"`
	Password *string `json:"password"`
	TenantID *int    `json:"tenant_id"`
	Status   *int    `json:"status" validate:"omitempty,oneof=0 1"`
}

// VDIServerDTO VDI服务器数据传输对象
type VDIServerDTO struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Endpoint    string     `json:"endpoint"`
	Username    string     `json:"username"`
	TenantID    int        `json:"tenant_id"`
	Status      int        `json:"status"`
	TokenExpiry *time.Time `json:"token_expiry"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// VDIServerPageResult VDI服务器分页结果
type VDIServerPageResult struct {
	List     []VDIServerDTO `json:"list"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

// ============ VDI Server Service 接口 ============

// VDIServerService VDI服务器服务接口
type VDIServerService interface {
	// CRUD操作
	CreateServer(ctx context.Context, req *CreateVDIServerRequest) (*VDIServerDTO, error)
	GetServer(ctx context.Context, id string) (*VDIServerDTO, error)
	ListServers(ctx context.Context, page, pageSize int, orderByColumn string, isAsc *bool) (*VDIServerPageResult, error)
	UpdateServer(ctx context.Context, id string, req *UpdateVDIServerRequest) error
	DeleteServer(ctx context.Context, id string) error

	// 连接测试
	TestConnection(ctx context.Context, id string) error
}
