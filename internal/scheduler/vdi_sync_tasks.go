package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// ==================== VDI 同步任务注册函数 ====================

// RegisterVDISyncTasks 注册 VDI 同步相关定时任务
func RegisterVDISyncTasks(scheduler *Scheduler) {
	// VDI 虚拟机数据同步任务 - 从 VDI 服务器同步虚拟机列表
	scheduler.RegisterTask("vdi_vm_sync", func(ctx context.Context, params map[string]interface{}) error {
		return executeVDIVMSyncTask(ctx, params)
	})
}

// executeVDIVMSyncTask 执行 VDI 虚拟机同步任务
// 从所有启用的 VDI 服务器同步虚拟机数据到本地数据库
func executeVDIVMSyncTask(ctx context.Context, params map[string]interface{}) error {
	if GlobalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	db := GlobalDB.GetDB()

	// 从参数中获取服务器ID（可选）
	serverID, _ := params["param"].(string)

	// 如果没有指定服务器，同步所有启用的服务器
	if serverID == "" || serverID == "auto" {
		return syncAllEnabledVDIServers(ctx, db)
	}

	// 同步指定的服务器
	return syncSingleVDIServer(ctx, db, serverID)
}

// syncAllEnabledVDIServers 同步所有启用的 VDI 服务器
func syncAllEnabledVDIServers(ctx context.Context, db *gorm.DB) error {
	// 查询所有启用的 VDI 服务器
	var servers []models.VDIServer
	if err := db.Where("status = ?", 0).Find(&servers).Error; err != nil {
		return fmt.Errorf("查询 VDI 服务器失败: %w", err)
	}

	if len(servers) == 0 {
		applogger.Infof("没有启用的 VDI 服务器，跳过同步")
		return nil
	}

	applogger.Infof("开始同步 VDI 虚拟机数据，共 %d 个服务器", len(servers))

	successCount := 0
	failCount := 0

	for _, server := range servers {
		if err := syncVDIServerVMs(ctx, db, &server); err != nil {
			applogger.Infof("同步 VDI 服务器失败 [%s]: %v", server.Name, err)
			failCount++
		} else {
			successCount++
		}
	}

	applogger.Infof("VDI 虚拟机数据同步完成: 成功=%d, 失败=%d", successCount, failCount)
	return nil
}

// syncSingleVDIServer 同步单个 VDI 服务器
func syncSingleVDIServer(ctx context.Context, db *gorm.DB, serverID string) error {
	var server models.VDIServer
	if err := db.Where("id = ?", serverID).First(&server).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("VDI 服务器不存在: %s", serverID)
		}
		return fmt.Errorf("查询 VDI 服务器失败: %w", err)
	}

	if server.Status != 0 {
		return fmt.Errorf("VDI 服务器未启用: %s", server.Name)
	}

	return syncVDIServerVMs(ctx, db, &server)
}

// syncVDIServerVMs 同步指定 VDI 服务器的虚拟机数据
// 调用 VDI 服务层执行实际的同步操作
func syncVDIServerVMs(ctx context.Context, db *gorm.DB, server *models.VDIServer) error {
	startTime := time.Now()
	applogger.Infof("开始同步 VDI 服务器 [%s] 的虚拟机数据", server.Name)

	// 获取 VDI 服务实例
	vmService := GetVDIVMService()
	if vmService == nil {
		return fmt.Errorf("VDI 虚拟机服务未初始化，请检查 Core 初始化配置")
	}

	// 调用 VDI 服务执行实际同步
	if err := vmService.SyncVMsFromVDIByServer(ctx, server); err != nil {
		return fmt.Errorf("VDI 服务器 [%s] 同步失败: %w", server.Name, err)
	}

	// 记录同步日志
	duration := time.Since(startTime)
	applogger.Infof("VDI 服务器 [%s] 同步完成，耗时: %v", server.Name, duration)

	// 更新服务器的最后同步时间
	if err := db.Model(server).Update("last_sync_time", time.Now()).Error; err != nil {
		applogger.Infof("更新服务器同步时间失败: %v", err)
	}

	return nil
}

// SyncVDIVMsManually 手动触发 VDI 虚拟机同步（供 API 调用）
// 这个函数提供了从外部触发同步的入口点
func SyncVDIVMsManually(ctx context.Context, db *gorm.DB, serverID string) error {
	if serverID == "" || serverID == "auto" {
		return syncAllEnabledVDIServers(ctx, db)
	}
	return syncSingleVDIServer(ctx, db, serverID)
}
