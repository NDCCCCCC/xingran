package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
	"time"
)

// VDIVirtualMachine 虚拟机表
type VDIVirtualMachine struct {
	BaseModel
	VMID          string  `gorm:"type:varchar(100);uniqueIndex;not null" json:"vm_id"`
	Name          string  `gorm:"type:varchar(200);not null" json:"name"`
	ResourceID    string  `gorm:"type:varchar(100);index" json:"resource_id"`
	PowerState    string  `gorm:"type:varchar(50)" json:"power_state"`
	IPAddress     string  `gorm:"type:varchar(50)" json:"ip_address"`
	MACAddress    string  `gorm:"type:varchar(50)" json:"mac_address"`
	OSType        string  `gorm:"type:varchar(50)" json:"os_type"`
	CPUNumber     int     `json:"cpu_number"` // CPU颗数
	CPUCore       int     `json:"cpu_core"`   // 每颗CPU的核数
	CPUPer        int     `json:"cpu_per"`    // CPU使用率
	Memory        int     `json:"memory"`
	MemoryPer     int     `json:"memory_per"` // 内存使用率
	Disk          int     `json:"disk"`
	DiskPer       int     `json:"disk_per"` // 磁盘使用率
	BoundUserID   *string `gorm:"type:varchar(100)" json:"bound_user_id"`
	BoundUserName *string `gorm:"type:varchar(200)" json:"bound_user_name"`
	PolicyGroupID *string `gorm:"type:varchar(100)" json:"policy_group_id"`
	// 网络配置信息
	IPType         string     `gorm:"type:varchar(50)" json:"ip_type"`         // IP类型：STATIC/DHCP
	SubnetMask     string     `gorm:"type:varchar(50)" json:"subnet_mask"`     // 子网掩码
	DefaultGateway string     `gorm:"type:varchar(50)" json:"default_gateway"` // 默认网关
	NameServer     string     `gorm:"type:varchar(100)" json:"name_server"`    // DNS服务器
	AssignIP       string     `gorm:"type:varchar(50)" json:"assign_ip"`       // 分配的IP地址
	LastSyncAt     *time.Time `json:"last_sync_at"`
	VdiServerID    string     `gorm:"type:varchar(100);index;not null" json:"vdi_server_id"`
}

// TableName 指定表名
func (VDIVirtualMachine) TableName() string {
	return "sys_vdi_vm"
}

// VDIServerStatus VDI服务器状态枚举（0=正常, 1=停用）
type VDIServerStatus int

const (
	VDIServerStatusNormal  VDIServerStatus = 0 // 正常
	VDIServerStatusStopped VDIServerStatus = 1 // 停用
)

// VDIServer VDI服务器配置表
type VDIServer struct {
	BaseModel
	Name              string     `gorm:"type:varchar(200);not null" json:"name"`
	Endpoint          string     `gorm:"type:varchar(500);not null" json:"endpoint"`
	Username          string     `gorm:"type:varchar(100);not null" json:"username"`
	PasswordEncrypted string     `gorm:"type:varchar(500);not null" json:"-"`
	TenantID          int        `json:"tenant_id"`
	AuthToken         string     `gorm:"type:varchar(1000)" json:"-"`
	TokenExpiry       *time.Time `json:"-"`
	LastSyncTime      *time.Time `gorm:"column:last_sync_time" json:"lastSyncTime,omitempty"`
	Status            int        `gorm:"type:int;default:0" json:"status"`
}

// TableName 指定表名
func (VDIServer) TableName() string {
	return "sys_vdi_server"
}

// VDIResourceGroup 资源组表
type VDIResourceGroup struct {
	BaseModel
	ResourceGroupID string `gorm:"type:varchar(100);uniqueIndex;not null" json:"resource_group_id"` // 资源组ID（深信服返回的ID）
	Name            string `gorm:"type:varchar(200);not null" json:"name"`                          // 资源组名称
	VdiServerID     string `gorm:"type:varchar(100);index;not null" json:"vdi_server_id"`           // VDI服务器ID
	Type            string `gorm:"type:varchar(50)" json:"type"`                                    // 类型：独享桌面/池桌面
	Status          int    `gorm:"type:int;default:0" json:"status"`                                // 0=正常, 1=停用
}

// TableName 指定表名
func (VDIResourceGroup) TableName() string {
	return "sys_vdi_resource_group"
}

// VDIUserBinding 用户关联表
type VDIUserBinding struct {
	BaseModel
	UserID      string    `gorm:"type:varchar(100);index;not null" json:"user_id"`       // 用户ID
	UserName    string    `gorm:"type:varchar(200);not null" json:"user_name"`           // 用户名
	VMID        string    `gorm:"type:varchar(100);index;not null" json:"vm_id"`         // 虚拟机ID
	VdiServerID string    `gorm:"type:varchar(100);index;not null" json:"vdi_server_id"` // VDI服务器ID
	BoundAt     time.Time `gorm:"not null" json:"bound_at"`                              // 绑定时间
	Status      int       `gorm:"type:int;default:0" json:"status"`                      // 0=正常, 1=停用
}

// TableName 指定表名
func (VDIUserBinding) TableName() string {
	return "sys_vdi_user_binding"
}

// encryptVDIPassword 加密VDI服务器密码（使用AES-128-GCM）
func encryptVDIPassword(password string) string {
	const encryptionKey = "xingran-vdi-server-key-16"
	key := []byte(encryptionKey[:16])

	block, err := aes.NewCipher(key)
	if err != nil {
		return password
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return password
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return password
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(password), nil)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

// decryptVDIPassword 解密VDI服务器密码
func decryptVDIPassword(encrypted string) string {
	const encryptionKey = "xingran-vdi-server-key-16"
	key := []byte(encryptionKey[:16])

	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return encrypted
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return encrypted
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return encrypted
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return encrypted
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return encrypted
	}

	return string(plaintext)
}

// VMP服务器API数据结构

// VMPVMGroup VMP虚拟机组
type VMPVMGroup struct {
	Name string       `json:"name"` // 分组名称
	ID   string       `json:"id"`   // 分组ID
	Data []*VMPVMInfo `json:"data"` // 虚拟机列表
}

// VMPVMInfo VMP虚拟机信息
type VMPVMInfo struct {
	// 基本信息
	VMID     int64  `json:"vmid"`     // 虚拟机ID
	Name     string `json:"name"`     // 虚拟机名称
	Status   string `json:"status"`   // 状态: running/stopped/clone
	IP       string `json:"ip"`       // IP地址
	Hostname string `json:"hostname"` // 主机名
	OSType   string `json:"ostype"`   // 操作系统类型: win1064/l2664/sslvpn
	VMType   string `json:"vmtype"`   // 虚拟机类型: vm/derive/tpl

	// 硬件配置
	CoresNumber int `json:"cores_number"` // CPU核心数
	Memory      int `json:"memory"`       // 内存大小(MB)
	Graphics    int `json:"graphics"`     // 显卡

	// 分组信息
	GroupName    string `json:"groupname"`               // 分组名称
	VMGroup      string `json:"vmgroup"`                 // 分组ID
	TemplateUUID string `json:"template_uuid,omitempty"` // 模板UUID

	// 运行状态
	CPURatio string `json:"cpu_ratio,omitempty"` // CPU使用率
	MemRatio string `json:"mem_ratio,omitempty"` // 内存使用率
	IORatio  string `json:"io_ratio,omitempty"`  // IO使用率

	// 详细状态信息（仅running状态时有值）
	CPUStatus  *VMPCPUStatus  `json:"cpu_status,omitempty"`
	MemStatus  *VMPMemStatus  `json:"mem_status,omitempty"`
	DiskStatus *VMPDiskStatus `json:"disk_status,omitempty"`

	// 资源使用情况
	ResMemUsed  string `json:"res_mem_uesed,omitempty"`  // 已用内存
	ResDiskUsed int64  `json:"res_disk_uesed,omitempty"` // 已用磁盘

	// 网络流量
	FlowInfoReceive int `json:"flow_info_receive,omitempty"` // 接收流量
	FlowInfoSend    int `json:"flow_info_send,omitempty"`    // 发送流量

	// 主机信息
	Host       string `json:"host"`       // 宿主机ID
	HStatus    int    `json:"hstatus"`    // 宿主机状态
	Node       string `json:"node"`       // 节点名称
	CfgStorage string `json:"cfgstorage"` // 存储配置

	// 其他
	HA             int    `json:"ha"`                        // HA启用状态
	IfIrsOn        int    `json:"if_irs_on"`                 // IRS启用状态
	AssociatedUser string `json:"associated_user,omitempty"` // 关联用户
}

// VMPCPUStatus CPU状态信息
type VMPCPUStatus struct {
	Ratio string `json:"ratio"` // 使用率（如 "0.28"）
	MHz   int    `json:"mhz"`   // 频率
	CPUs  int    `json:"cpus"`  // 核心数
}

// VMPMemStatus 内存状态信息
type VMPMemStatus struct {
	Ratio string `json:"ratio"` // 使用率（如 "0.27"）
	Free  int64  `json:"free"`  // 剩余内存(字节)
	Total int64  `json:"total"` // 总内存(字节)
}

// VMPDiskStatus 磁盘状态信息
type VMPDiskStatus struct {
	Ratio string `json:"ratio"` // 使用率（如 "0.23"）
	Free  int64  `json:"free"`  // 剩余空间(字节)
	Total int64  `json:"total"` // 总空间(字节)
}

// VMPResponse VMP API响应
type VMPResponse struct {
	Success int           `json:"success"` // 1=成功
	Data    []*VMPVMGroup `json:"data"`    // 分组数据
}
