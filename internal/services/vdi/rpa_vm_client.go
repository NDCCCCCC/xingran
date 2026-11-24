package vdi

// Phase 157 / FLEET-01：为 RPA 扩缩容（VMDIProvider）构建已认证的 VDI 客户端。
//
// 本文件不新增任何 HTTP 调用 —— 复用 client.go 的 VDIClient 与 vdi_auth_manager.go
// 的密码解密（decryptVDIPassword，未导出所以工厂必须放在本包内）。
// 服务器地址与凭据来自 sys_vdi_server 表（与 VDI 虚拟机管理共用同一份配置）。

import (
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// BuildRPAVMClient 按 serverID 从 sys_vdi_server 加载配置并构建已认证的 VDI 客户端。
// 返回的 *VDIClient 满足 rpa.VDIVMClient 接口（CreateVM/StartVM/StopVM/ListVMs/GetVMInfo）。
func BuildRPAVMClient(db *gorm.DB, serverID string) (*VDIClient, error) {
	if db == nil {
		return nil, fmt.Errorf("BuildRPAVMClient: DB 句柄为空")
	}
	if serverID == "" {
		return nil, fmt.Errorf("BuildRPAVMClient: serverID 为空")
	}

	var server models.VDIServer
	if err := db.Where("id = ?", serverID).First(&server).Error; err != nil {
		return nil, fmt.Errorf("加载 VDI 服务器配置失败 (id=%s): %w", serverID, err)
	}
	if server.Endpoint == "" {
		return nil, fmt.Errorf("VDI 服务器 endpoint 为空 (id=%s)", serverID)
	}

	client := NewVDIClient(server.Endpoint, 30*1000) // 30 秒超时，与 ClientManager 一致
	password := decryptVDIPassword(server.PasswordEncrypted)
	if password == "" {
		return nil, fmt.Errorf("VDI 服务器密码解密失败 (id=%s)，请检查 sys_vdi_server 记录", serverID)
	}
	if err := client.Authenticate(server.Username, password); err != nil {
		return nil, fmt.Errorf("VDI 认证失败 (id=%s): %w", serverID, err)
	}

	return client, nil
}
