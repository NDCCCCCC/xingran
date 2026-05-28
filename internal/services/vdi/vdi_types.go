package vdi

import "time"

// ============ VDI API 请求/响应类型 ============

// AuthRequest 认证请求
type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse 认证响应
type AuthResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
}

// VDIResponse VDI API 统一响应格式
type VDIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// CreateVMRequest 创建虚拟机请求
type CreateVMRequest struct {
	VMName       string `json:"vM_NAME"`
	IP           string `json:"ip"`
	MAC          string `json:"mac"`
	Username     string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
	VolumeID     string `json:"volumeId,omitempty"`
	ImageID      string `json:"imageId,omitempty"`
	DesktopGroup string `json:"desktopGroup,omitempty"`
	Comments     string `json:"comments,omitempty"`
}

// CreateVMResponse 创建虚拟机响应
type CreateVMResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	VMID    string `json:"vM_ID,omitempty"`
}

// VMOperationRequest 虚拟机操作请求
type VMOperationRequest struct {
	VMID string `json:"vM_ID"`
}

// ConfigIPRequest 配置 IP 请求
type ConfigIPRequest struct {
	VMID string `json:"vM_ID"`
	IP   string `json:"ip"`
}

// RenameVMRequest 重命名虚拟机请求
type RenameVMRequest struct {
	VMID   string `json:"vM_ID"`
	VMName string `json:"vM_NAME"`
}

// BindUserRequest 绑定用户请求
type BindUserRequest struct {
	VMID     string `json:"vM_ID"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// VMInfo 虚拟机信息
type VMInfo struct {
	VMID        string    `json:"vM_ID"`
	VMName      string    `json:"vM_NAME"`
	IP          string    `json:"ip"`
	MAC         string    `json:"mac"`
	Status      string    `json:"status"`
	Username    string    `json:"username,omitempty"`
	Password    string    `json:"password,omitempty"`
	VolumeID    string    `json:"volumeId,omitempty"`
	ImageID     string    `json:"imageId,omitempty"`
	DesktopGroup string   `json:"desktopGroup,omitempty"`
	Comments    string    `json:"comments,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
}

// VMListResponse 虚拟机列表响应
type VMListResponse struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Data    []VMInfo `json:"data,omitempty"`
}

// VMInfoResponse 虚拟机详情响应
type VMInfoResponse struct {
	Code    int     `json:"code"`
	Message string  `json:"message"`
	Data    VMInfo  `json:"data,omitempty"`
}

// ============ VDI Client 扩展类型 ============

// VDIVMDetail 虚拟机详情
type VDIVMDetail struct {
	VMID          string  `json:"vm_id"`
	Name          string  `json:"name"`
	ResourceID    string  `json:"resource_id"`
	Status        int     `json:"status"`
	PowerState    string  `json:"power_state"`
	IPAddress     string  `json:"ip_address"`
	MACAddress    string  `json:"mac_address"`
	OSType        string  `json:"os_type"`
	CPU           int     `json:"cpu"`
	Memory        int     `json:"memory"`
	Disk          int     `json:"disk"`
	BoundUserID   *string `json:"bound_user_id"`
	BoundUserName *string `json:"bound_user_name"`
	PolicyGroupID *string `json:"policy_group_id"`
}

// VDIVMSummary 虚拟机摘要信息
type VDIVMSummary struct {
	VMID       string `json:"vm_id"`
	Name       string `json:"name"`
	PowerState string `json:"power_state"`
	Status     int    `json:"status"`
}

// ConfigIPRequestExtended 配置IP请求（扩展）
// VDI API endpoint: PUT /v1/servers/ip
// type: 0=用户自定义, 1=自动获取, 2=获取指定IP
type ConfigIPRequestExtended struct {
	IDs         string `json:"ids"`          // VM IDs separated by commas
	Type        int    `json:"type"`         // 0:custom, 1:auto, 2:specified IP
	IsRestart   int    `json:"is_restart"`   // 0:no, 1:yes
	BeginIP     string `json:"begin_ip,omitempty"`
	EndIP       string `json:"end_ip,omitempty"`
	VMNames     string `json:"vm_names,omitempty"`
	Mask        string `json:"mask,omitempty"`
	Gate        string `json:"gate,omitempty"`  // Gateway
	DNS         string `json:"dns,omitempty"`
	ReserveDNS  string `json:"reserve_dns,omitempty"`
}

// VMIPInfo 虚拟机IP信息
type VMIPInfo struct {
	VMID      string `json:"vm_id"`
	IPAddress string `json:"ip_address"`
	Netmask   string `json:"netmask"`
	Gateway   string `json:"gateway"`
}

// VDIUser VDI用户信息
type VDIUser struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	DN       string `json:"dn"`
}

// ============ VDI API 资源和虚拟机类型 ============

// VDIResourceGroup VDI资源组
type VDIResourceGroup struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Note   string `json:"note"`
	Enable string `json:"enable"`
}

// VDIResource VDI独享桌面资源
type VDIResource struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Note   string `json:"note"`
	GrpID  int    `json:"grp_id"`
}

// VDIVMResource VDI虚拟机资源信息（完整）
type VDIVMResource struct {
	ID                  string `json:"_id"` // VDI API返回整数，但Go json会自动转换为字符串
	Status              string `json:"status"`
	Graphics            string `json:"graphics"`
	GraphicsMemAll      string `json:"graphics_mem_all"`
	GraphicsMemNow      string `json:"graphics_mem_now"`
	GraphicsMemPer      string `json:"graphics_mem_per"`
	GraphicsCardType    string `json:"graphics_card_type"`
	VMName              string `json:"vm_name"`
	VTPID               string `json:"vtp_id"`
	VTPName             string `json:"vtp_name"`
	RCID                string `json:"rc_id"`
	RCName              string `json:"rc_name"`
	IP                  string `json:"ip"`
	CPUNumber           string `json:"cpu_number"`
	CPUCore             string `json:"cpu_core"`
	CPUPer              string `json:"cpu_per"`
	MemAll              string `json:"mem_all"`
	MemNow              string `json:"mem_now"`
	MemPer              string `json:"mem_per"`
	DiscAll             string `json:"disc_all"`
	DiscNow             string `json:"disc_now"`
	DiscPer             string `json:"disc_per"`
	ApplyUser           string `json:"apply_user"`
	ApplyUserStatus     string `json:"apply_user_status"`
	ApplyAppstack       string `json:"apply_appstack"`
	ApplyAppstackID     string `json:"apply_appstack_id"`
	AgentVersion        string `json:"agent_version"`
	LastLogin           string `json:"last_login"`
	IsUsed              string `json:"is_used"`
	AssignIP            string `json:"assign_ip"`
	Subnetmask          string `json:"subnetmask"`
	DefaultGateway      string `json:"defaultgateway"`
	NameServer          string `json:"nameserver"`
	ReserveNameServer   string `json:"reserve_nameserver"`
	IPState             string `json:"ip_state"`
	AuthStatus          string `json:"authstatus"`
	OSType              string `json:"osType"`
	ErrMsg              string `json:"err_msg"`
	MAC                 string `json:"mac"`
	UserDesc            string `json:"user_desc"`
	LastResult          string `json:"last_result"`
	IsClient            bool   `json:"isClient"`
	GroupPolicyName     string `json:"group_policy_name"`
	GroupPolicyID       string `json:"group_policy_id"`
	IsEnableGroupPolicy bool `json:"is_enable_group_policy"`
	AreaID              string `json:"araeId"`
	AreaName            string `json:"areaName"`
	ApplyWsus           string `json:"apply_wsus"`
}

// VDIResourceServersResponse VDI资源服务器响应
type VDIResourceServersResponse struct {
	ErrorCode    int             `json:"error_code"`
	ErrorMessage string          `json:"error_message"`
	Data         struct {
		TotalCount string           `json:"totalCount"`
		Data       []VDIVMResource `json:"data"`
	} `json:"data"`
}

// VDIResourceGroupsResponse VDI资源组响应
type VDIResourceGroupsResponse struct {
	ErrorCode    int                `json:"error_code"`
	ErrorMessage string             `json:"error_message"`
	Data         []VDIResourceGroup `json:"data"`
}
