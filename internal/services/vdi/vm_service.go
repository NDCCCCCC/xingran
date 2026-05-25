package vdi

import (
	"context"
	"time"
)

// ============ 请求/响应 DTO ============

// CreateVMServiceRequest 创建虚拟机请求（服务层）
type CreateVMServiceRequest struct {
	Name        string `json:"name" validate:"required"`
	ResourceID  string `json:"resource_id" validate:"required"`
	VdiServerID string `json:"vdi_server_id" validate:"required"`
	CPU         int    `json:"cpu" validate:"min=1,max=64"`
	Memory      int    `json:"memory" validate:"min=512,max=131072"`
	Disk        int    `json:"disk" validate:"min=20,max=2000"`
}

// UpdateVMRequest 更新虚拟机请求
type UpdateVMRequest struct {
	Name       *string `json:"name"`
	Status     *int    `json:"status"`
	IPAddress  *string `json:"ip_address"`
	MACAddress *string `json:"mac_address"`
}

// ListVMRequest 虚拟机列表请求
type ListVMRequest struct {
	Page        int    `json:"page" validate:"min=1"`
	PageSize    int    `json:"pageSize" validate:"min=1,max=100"`
	Name        string `json:"name"`
	Status      *int   `json:"status"`
	VdiServerID string `json:"vdi_server_id"`
	ResourceID  string `json:"resource_id"`
	PowerState  string `json:"power_state"`
}

// VDIVMDTO 虚拟机数据传输对象
type VDIVMDTO struct {
	ID            string     `json:"id"`
	VMID          string     `json:"vm_id"`
	Name          string     `json:"name"`
	ResourceID    string     `json:"resource_id"`
	Status        int        `json:"status"`
	PowerState    string     `json:"power_state"`
	IPAddress     string     `json:"ip_address"`
	MACAddress    string     `json:"mac_address"`
	OSType        string     `json:"os_type"`
	CPU           int        `json:"cpu"`
	Memory        int        `json:"memory"`
	Disk          int        `json:"disk"`
	BoundUserID   *string    `json:"bound_user_id"`
	BoundUserName *string    `json:"bound_user_name"`
	VdiServerID   string     `json:"vdi_server_id"`
	LastSyncAt    *time.Time `json:"last_sync_at"`
}

// PageResult 分页结果
type PageResult struct {
	List     []VDIVMDTO `json:"list"`
	Total    int64      `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"pageSize"`
}

// VMPowerAction 虚拟机电源操作
type VMPowerAction string

const (
	VMPowerOn      VMPowerAction = "start"
	VMPowerOff     VMPowerAction = "stop"
	VMPowerRestart VMPowerAction = "restart"
	VMPowerSuspend VMPowerAction = "suspend"
)

// VMOperateRequest 虚拟机操作请求
type VMOperateRequest struct {
	VMIDs  []string      `json:"vm_ids" validate:"required,min=1"`
	Action VMPowerAction `json:"action" validate:"required"`
}

// VMIPConfigRequest 虚拟机IP配置请求
type VMIPConfigRequest struct {
	VMID     string `json:"vm_id" validate:"required"`
	IPAddress string `json:"ip_address" validate:"required,ip"`
	Netmask  string `json:"netmask" validate:"required"`
	Gateway  string `json:"gateway" validate:"required"`
}

// RenameVMServiceRequest 重命名虚拟机请求（服务层）
type RenameVMServiceRequest struct {
	NewName string `json:"new_name" validate:"required,max=200"`
}

// BindUserServiceRequest 绑定用户请求（服务层）
type BindUserServiceRequest struct {
	UserID string `json:"user_id" validate:"required"`
}

// ============ VM Service 接口 ============

// VMService 虚拟机服务接口
type VMService interface {
	// CRUD操作（都会调用VDI API）
	CreateVM(ctx context.Context, req *CreateVMServiceRequest) (*VDIVMDTO, error)
	GetVM(ctx context.Context, id string) (*VDIVMDTO, error)
	ListVMs(ctx context.Context, req *ListVMRequest) (*PageResult, error)
	UpdateVM(ctx context.Context, id string, req *UpdateVMRequest) error
	DeleteVM(ctx context.Context, ids []string) error

	// VDI操作（完整实现VDI API调用）
	OperateVM(ctx context.Context, req *VMOperateRequest) error
	BatchConfigIP(ctx context.Context, req []VMIPConfigRequest) error
	RenameVM(ctx context.Context, id string, req *RenameVMServiceRequest) error

	// 用户关联
	BindUser(ctx context.Context, vmID string, req *BindUserServiceRequest) error
	UnbindUser(ctx context.Context, vmID string) error

	// 同步操作
	SyncVMFromVDI(ctx context.Context, vmID string) error
	SyncAllVMs(ctx context.Context, serverID string) error
}
