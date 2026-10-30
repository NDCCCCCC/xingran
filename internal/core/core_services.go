// Package core 提供系统的核心功能模块
// 业务服务层：管理所有业务领域的服务实例（设备、缓存、日志、认证、API 元数据等）。
package core

import (
	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
)

// CoreServices 核心业务服务层
// 包含系统运行所需的业务服务实例：设备、缓存、日志、WebSocket 通知等。
// 这些组件通常是业务层依赖，由 Init() 流程按顺序初始化。
//
// 通过 struct embedding 嵌入到 Core 中，保持向后兼容的字段访问语法
//（core.UserService、core.DeviceExecutor、core.NoticeHub 等）。
type CoreServices struct {
	// 网络设备相关服务
	DeviceExecutor              *device.DeviceExecutor                // 设备执行器
	DeviceDiscoveryService      *services.DeviceDiscoveryService      // 设备发现服务
	DeviceInfoCollectionService *services.DeviceInfoCollectionService // 设备信息采集服务
	DeviceMonitorService        *services.DeviceMonitorService        // 设备监控服务

	// WebSocket 通知
	NoticeHub *websocket.NoticeHub // WebSocket 通知中心

	// 日志与认证
	OperLogService        services.OperLogService        // 操作日志服务
	TokenBlacklistService services.TokenBlacklistService // 令牌黑名单服务

	// 缓存服务（System 模块使用）
	DataCacheService   *services.DataCacheService   // 通用数据缓存服务
	CacheConfigService *services.CacheConfigService // 缓存配置服务
	CacheManager       *system.CacheManager         // 缓存管理器（增强功能）

	// RPA 服务
	RPAScalingService rpaScalingService // RPA 扩缩容服务（内部使用，仅在 Close() 中停止）

	// API 端点元数据
	APIEndpointService *services.APIEndpointService // API 端点元数据服务

	// MAC 历史分区
	PartitionService services.PartitionService // MAC 历史表分区管理服务
}
