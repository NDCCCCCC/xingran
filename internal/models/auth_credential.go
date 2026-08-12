package models

import "github.com/lib/pq"

// ProtocolType 协议类型枚举 (SSH 或 Telnet)
type ProtocolType string

const (
	ProtocolTypeSSH    ProtocolType = "ssh"    // SSH
	ProtocolTypeTelnet ProtocolType = "telnet" // Telnet
)

// SNMPVersion SNMP版本枚举
type SNMPVersion string

const (
	SNMPVersionV1  SNMPVersion = "v1"  // SNMP v1
	SNMPVersionV2c SNMPVersion = "v2c" // SNMP v2c
	SNMPVersionV3  SNMPVersion = "v3"  // SNMP v3
)

// AuthCredential 授权凭证模型
// 支持同时配置 SSH/Telnet 和 SNMP，用于设备连接和监控
type AuthCredential struct {
	BaseModel
	CredentialName  string         `gorm:"size:100;not null" json:"credentialName"`
	ProtocolType    ProtocolType   `gorm:"size:10;not null;default:ssh" json:"protocolType"` // SSH 或 Telnet
	Username        string         `gorm:"size:100" json:"username,omitempty"`
	Password        string         `gorm:"size:255" json:"-"`                            // 加密存储，不返回给前端
	EnablePassword  string         `gorm:"size:255" json:"-"`                            // SSH 特权模式密码，加密存储
	SNMPCommunities pq.StringArray `gorm:"type:text[]" json:"snmpCommunities,omitempty"` // 多个 SNMP Community
	SNMPVersion     SNMPVersion    `gorm:"size:10;default:v2c" json:"snmpVersion"`
	Description     string         `gorm:"type:text" json:"description,omitempty"`
	IsDefault       bool           `gorm:"default:false" json:"isDefault"`
}

// TableName 设置表名
func (AuthCredential) TableName() string {
	return "sys_auth_credential"
}
