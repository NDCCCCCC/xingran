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
	VMID          string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"vm_id"`
	Name          string    `gorm:"type:varchar(200);not null" json:"name"`
	ResourceID    string    `gorm:"type:varchar(100);index" json:"resource_id"`
	Status        int       `gorm:"type:int;default:0" json:"status"`
	PowerState    string    `gorm:"type:varchar(50)" json:"power_state"`
	IPAddress     string    `gorm:"type:varchar(50)" json:"ip_address"`
	MACAddress    string    `gorm:"type:varchar(50)" json:"mac_address"`
	OSType        string    `gorm:"type:varchar(50)" json:"os_type"`
	CPU           int       `json:"cpu"`
	Memory        int       `json:"memory"`
	Disk          int       `json:"disk"`
	BoundUserID    *string   `gorm:"type:varchar(100)" json:"bound_user_id"`
	BoundUserName  *string   `gorm:"type:varchar(200)" json:"bound_user_name"`
	PolicyGroupID *string   `gorm:"type:varchar(100)" json:"policy_group_id"`
	LastSyncAt    *time.Time `json:"last_sync_at"`
	VdiServerID   string    `gorm:"type:varchar(100);index;not null" json:"vdi_server_id"`
}

// TableName 指定表名
func (VDIVirtualMachine) TableName() string {
	return "sys_vdi_vm"
}

// VDIServer VDI服务器配置表
type VDIServer struct {
	BaseModel
	Name              string    `gorm:"type:varchar(200);not null" json:"name"`
	Endpoint          string    `gorm:"type:varchar(500);not null" json:"endpoint"`
	Username          string    `gorm:"type:varchar(100);not null" json:"username"`
	PasswordEncrypted string    `gorm:"type:varchar(500);not null" json:"-"`
	TenantID          int       `json:"tenant_id"`
	AuthToken         string    `gorm:"type:varchar(1000)" json:"-"`
	TokenExpiry       *time.Time `json:"-"`
	Status            int       `gorm:"type:int;default:0" json:"status"`
}

// TableName 指定表名
func (VDIServer) TableName() string {
	return "sys_vdi_server"
}

// VDIResourceGroup 资源组表
type VDIResourceGroup struct {
	BaseModel
	ResourceGroupID string `gorm:"type:varchar(100);uniqueIndex;not null" json:"resource_group_id"` // 资源组ID（深信服返回的ID）
	Name            string `gorm:"type:varchar(200);not null" json:"name"`                            // 资源组名称
	VdiServerID     string `gorm:"type:varchar(100);index;not null" json:"vdi_server_id"`             // VDI服务器ID
	Type            string `gorm:"type:varchar(50)" json:"type"`                                       // 类型：独享桌面/池桌面
	Status          int    `gorm:"type:int;default:0" json:"status"`                                   // 0=正常, 1=停用
}

// TableName 指定表名
func (VDIResourceGroup) TableName() string {
	return "sys_vdi_resource_group"
}

// VDIUserBinding 用户关联表
type VDIUserBinding struct {
	BaseModel
	UserID      string    `gorm:"type:varchar(100);index;not null" json:"user_id"`         // 用户ID
	UserName    string    `gorm:"type:varchar(200);not null" json:"user_name"`             // 用户名
	VMID        string    `gorm:"type:varchar(100);index;not null" json:"vm_id"`           // 虚拟机ID
	VdiServerID string    `gorm:"type:varchar(100);index;not null" json:"vdi_server_id"`   // VDI服务器ID
	BoundAt     time.Time `gorm:"not null" json:"bound_at"`                                // 绑定时间
	Status      int       `gorm:"type:int;default:0" json:"status"`                        // 0=正常, 1=停用
}

// TableName 指定表名
func (VDIUserBinding) TableName() string {
	return "sys_vdi_user_binding"
}

// encryptVDIPassword 加密VDI服务器密码（使用AES-128-GCM）
func encryptVDIPassword(password string) string {
	const encryptionKey = "ruoyi-vdi-server-key-16"
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
	const encryptionKey = "ruoyi-vdi-server-key-16"
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
