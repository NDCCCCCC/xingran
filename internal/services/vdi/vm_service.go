package vdi

import (
	"context"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============ 请求/响应 DTO ============

// CreateVMServiceRequest 创建虚拟机请求（服务层）
type CreateVMServiceRequest struct {
	Name          string `json:"name,omitempty"`
	ResourceID    string `json:"resource_id" validate:"required"`
	VdiServerID   string `json:"vdi_server_id" validate:"required"`
	CPUNumber     int    `json:"cpu_number" validate:"min=1,max=64"`   // CPU颗数
	CPUCore       int    `json:"cpu_core" validate:"min=1,max=128"`    // 每颗CPU核数
	Memory        int    `json:"memory" validate:"min=512,max=131072"`
	Disk          int    `json:"disk" validate:"min=20,max=2000"`
	// VDI API specific fields
	VTPID         int    `json:"vtp_id" validate:"required,min=1"`           // VTP platform ID
	HostID        string `json:"host_id" validate:"required"`                 // Host position ID (father_id)
	RunPositionID string `json:"run_position_id"`                              // Run position ID (empty if id == father_id)
	DiskID        string `json:"disk_id" validate:"required"`                  // Personal disk ID
	StorageID     string `json:"storage_id" validate:"required"`               // Storage location ID
	NetworkID     string `json:"network_id" validate:"required"`               // Network interface ID
	Count         int    `json:"count" validate:"min=1,max=10"`                // Number of VMs to create
}

// UpdateVMRequest 更新虚拟机请求
type UpdateVMRequest struct {
	Name       *string `json:"name"`
	IPAddress  *string `json:"ip_address"`
	MACAddress *string `json:"mac_address"`
}

// ListVMRequest 虚拟机列表请求
type ListVMRequest struct {
	Page          int    `json:"page" validate:"min=1"`
	PageSize      int    `json:"pageSize" validate:"min=1,max=100"`
	Name          string `json:"name"`
	VdiServerID   string `json:"vdi_server_id"`
	ResourceID    string `json:"resource_id"`
	PowerState    string `json:"power_state"`
	OrderByColumn string `json:"orderByColumn,omitempty"`
	IsAsc         *bool  `json:"isAsc,omitempty"`
}

// VDIVMDTO 虚拟机数据传输对象
type VDIVMDTO struct {
	ID            string     `json:"id"`
	VMID          string     `json:"vm_id"`
	Name          string     `json:"name"`
	ResourceID    string     `json:"resource_id"`
	PowerState    string     `json:"power_state"`
	IPAddress     string     `json:"ip_address"`
	MACAddress    string     `json:"mac_address"`
	OSType        string     `json:"os_type"`
	CPUNumber     int        `json:"cpu_number"`      // CPU颗数
	CPUCore       int        `json:"cpu_core"`        // 每颗CPU的核数
	CPUPer        int        `json:"cpu_per"`         // CPU使用率
	Memory        int        `json:"memory"`
	MemoryPer     int        `json:"memory_per"`      // 内存使用率
	Disk          int        `json:"disk"`
	DiskPer       int        `json:"disk_per"`        // 磁盘使用率
	BoundUserID    *string    `json:"bound_user_id"`
	BoundUserName  *string    `json:"bound_user_name"`
	PolicyGroupID  *string    `json:"policy_group_id"`
	// 网络配置信息
	IPType         string     `json:"ip_type"`              // IP类型：STATIC/DHCP
	SubnetMask     string     `json:"subnet_mask"`          // 子网掩码
	DefaultGateway string     `json:"default_gateway"`      // 默认网关
	NameServer     string     `json:"name_server"`          // DNS服务器
	AssignIP       string     `json:"assign_ip"`            // 分配的IP地址
	VdiServerID    string     `json:"vdi_server_id"`
	LastSyncAt     *time.Time `json:"last_sync_at"`
}

// PageResult 分页结果
type PageResult struct {
	List     []VDIVMDTO `json:"list"`
	Total    int64      `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"pageSize"`
}

// VDISyncResult VDI同步结果
type VDISyncResult struct {
	Total      int                    `json:"total"`
	Success    int                    `json:"success"`
	Failed     int                    `json:"failed"`
	Duration   string                 `json:"duration"`
	Servers    []VDIServerSyncResult  `json:"servers"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
}

// VDIServerSyncResult 单个VDI服务器的同步结果
type VDIServerSyncResult struct {
	ServerID    string `json:"server_id"`
	ServerName  string `json:"server_name"`
	Success     int    `json:"success"`
	Failed      int    `json:"failed"`
	ErrorMsg    string `json:"error_msg,omitempty"`
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

// BindUserServiceRequest 绑定用户请求（服务层）
type BindUserServiceRequest struct {
	Username string `json:"username" validate:"required"`
}

// VDIResourceGroupDTO 资源组数据传输对象
type VDIResourceGroupDTO struct {
	ResourceGroupID string `json:"resource_group_id"`
	Name            string `json:"name"`
	VdiServerID     string `json:"vdi_server_id"`
	Type            string `json:"type"`
}

// VDIResourceDTO 资源数据传输对象（资源组下的具体计算资源）
type VDIResourceDTO struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Note  string `json:"note"`
	GrpID int    `json:"grp_id"`
}

// ============ VM Service 接口 ============

// VMService 虚拟机服务接口
type VMService interface {
	// CRUD操作（都会调用VDI API）
	// 资源组查询
	ListResourceGroups(ctx context.Context, vdiServerID string) ([]VDIResourceGroupDTO, error)
	// 资源查询（资源组下的具体计算资源）
	ListResources(ctx context.Context, vdiServerID string, groupID string) ([]VDIResourceDTO, error)

	// CRUD操作（都会调用VDI API）
	CreateVM(ctx context.Context, req *CreateVMServiceRequest) (*VDIVMDTO, error)
	GetVM(ctx context.Context, id string) (*VDIVMDTO, error)
	ListVMs(ctx context.Context, req *ListVMRequest, userID string, dataScope models.DataScope) (*PageResult, error)
	UpdateVM(ctx context.Context, id string, req *UpdateVMRequest) error
	DeleteVM(ctx context.Context, ids []string) error

	// VDI操作（完整实现VDI API调用）
	OperateVM(ctx context.Context, req *VMOperateRequest) error

	// 用户关联
	BindUser(ctx context.Context, vmID string, req *BindUserServiceRequest) error
	UnbindUser(ctx context.Context, vmID string) error

	// 同步操作
	SyncVMFromVDI(ctx context.Context, vmID string) error
	SyncAllVMs(ctx context.Context, serverID string) error
	SyncVMsFromVDIByServer(ctx context.Context, server *models.VDIServer) error
}