package network

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	networkServices "github.com/xingran-next/xingran-go-backend/internal/services/network"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
)

// SetupNetworkRouter 设置网络设备管理路由
func SetupNetworkRouter(r *gin.RouterGroup, core *core.Core) {
	db := core.DB.GetDB()

	// 创建缓存提供者
	var cacheProvider systemServices.CacheProvider
	if core.DataCacheService != nil {
		cacheProvider = systemServices.NewCacheProvider(core.DataCacheService)
	} else {
		cacheProvider = &systemServices.NoOpCacheProvider{}
	}

	// 创建授权凭证服务
	credentialService := services.NewAuthCredentialService(db, core.SM4Cipher)

	// 创建网络设备服务（带缓存）
	deviceService := networkServices.NewServiceWithCache(
		db,
		core.DeviceDiscoveryService,
		core.DeviceInfoCollectionService,
		cacheProvider,
		core.CacheConfigService,
	)

	// 创建其他服务
	templateService := services.NewTemplateService(db)
	commandService := services.NewCommandDispatchService(db, core.DeviceExecutor)
	executionService := services.NewConfigExecutionService(db, core.DeviceExecutor)
	backupService := services.NewConfigBackupService(db, core.DeviceExecutor)
	discoveryService := core.DeviceDiscoveryService

	// 创建Handler
	credentialHandler := NewCredentialHandler(credentialService).WithCore(core)
	deviceHandler := NewDeviceHandler(deviceService).WithCore(core)
	templateHandler := NewTemplateHandler(templateService).WithCore(core)
	commandHandler := NewCommandHandler(commandService, db).WithCore(core)
	executionHandler := NewExecutionHandler(executionService).WithCore(core)
	backupHandler := NewBackupHandler(backupService, db).WithCore(core)
	discoveryHandler := NewDiscoveryHandler(discoveryService).WithCore(core)
	exportHandler := NewNetworkExportHandler(core)

	// ==================== 网络设备路由 ====================
	devices := r.Group("/devices")
	// 查询接口(/list)额外接受 ops 读权限(OpsSelectorReadPerms): 信息点管理等运维页面
	// 关联网络设备时需要读取设备列表做选择器, 但运维角色不持有 network:device 权限。
	// 放宽只读路径; 写操作(增删改网络设备/发现/导出)保持严格。
	devices.Use(middleware.RequirePermissionsWithQuery([]string{
		"network:device:list",
		"network:device:add",
		"network:device:edit",
		"network:device:delete",
	}, middleware.OpsSelectorReadPerms, core))
	{
		devices.POST("/list", deviceHandler.List)
		devices.POST("/statistics", deviceHandler.Statistics)
		devices.POST("", deviceHandler.Create)
		devices.POST("/discover", discoveryHandler.Probe)
		devices.POST("/quick-create", deviceHandler.QuickCreate)
		devices.POST("/:id", deviceHandler.GetByID)
		devices.POST("/:id/update", deviceHandler.Update)
		devices.POST("/:id/delete", deviceHandler.Delete)
		devices.POST("/batch-delete", deviceHandler.BatchDelete)
		devices.POST("/export", exportHandler.ExportDevices)
	}

	// ==================== 授权凭证路由 ====================
	credentials := r.Group("/credentials")
	credentials.Use(middleware.RequirePermissions([]string{
		"network:credential:list",
		"network:credential:add",
		"network:credential:edit",
		"network:credential:delete",
	}, core))
	{
		credentials.POST("/list", credentialHandler.List)
		credentials.POST("/statistics", credentialHandler.Statistics)
		credentials.POST("", credentialHandler.Create)
		credentials.POST("/:id", credentialHandler.GetByID)
		credentials.POST("/:id/update", credentialHandler.Update)
		credentials.POST("/:id/delete", credentialHandler.Delete)
		credentials.POST("/batch-delete", credentialHandler.BatchDelete)
		credentials.POST("/:id/set-default", credentialHandler.SetDefault)
		credentials.POST("/:id/devices", credentialHandler.GetDevicesByCredential)
		credentials.POST("/export", exportHandler.ExportCredentials)
	}

	// ==================== 配置模板路由 ====================
	templates := r.Group("/templates")
	templates.Use(middleware.RequirePermissions([]string{
		"network:template:list",
		"network:template:add",
		"network:template:edit",
		"network:template:delete",
	}, core))
	{
		templates.POST("/list", templateHandler.List)
		templates.POST("/statistics", templateHandler.Statistics)
		templates.POST("", templateHandler.Create)
		templates.POST("/:id", templateHandler.GetByID)
		templates.POST("/:id/update", templateHandler.Update)
		templates.POST("/:id/delete", templateHandler.Delete)
		templates.POST("/batch-delete", templateHandler.BatchDelete)
		templates.POST("/:id/preview", templateHandler.Preview)
		templates.POST("/:id/clone", templateHandler.Clone)
		templates.POST("/:id/variables", templateHandler.GetVariables)
		templates.POST("/export", exportHandler.ExportTemplates)
	}

	// ==================== 命令分发路由 ====================
	command := r.Group("/command")
	command.Use(middleware.RequirePermissions([]string{
		"network:command:execute",
		"network:command:view",
	}, core))
	{
		command.POST("/dispatch", commandHandler.Dispatch)
		command.POST("/quick", commandHandler.QuickCommand)
		command.POST("/list", commandHandler.List)
		command.POST("/statistics", commandHandler.Statistics)
		command.POST("/:id", commandHandler.GetExecutionResult)
		command.POST("/:id/device/:deviceId", commandHandler.GetDeviceExecutionDetail)
		command.POST("/export", exportHandler.ExportCommands)
	}

	// ==================== 配置执行路由 ====================
	executions := r.Group("/executions")
	executions.Use(middleware.RequirePermissions([]string{
		"network:command:execute",
	}, core))
	{
		executions.POST("/list", executionHandler.List)
		executions.POST("/statistics", executionHandler.Statistics)
		executions.POST("/template/execute", executionHandler.ExecuteByTemplate)
		executions.POST("/:id", executionHandler.GetByID)
		executions.POST("/:id/cancel", executionHandler.Cancel)
		executions.POST("/:id/delete", executionHandler.Delete)
		executions.POST("/batch-delete", executionHandler.BatchDelete)
		executions.POST("/export", exportHandler.ExportExecutions)
	}

	// ==================== 配置备份路由 ====================
	backups := r.Group("/backups")
	backups.Use(middleware.RequirePermissions([]string{
		"network:backup:list",
		"network:backup:add",
		"network:backup:restore",
		"network:backup:diff",
	}, core))
	{
		backups.POST("/list", backupHandler.List)
		backups.POST("", backupHandler.Create)
		backups.POST("/content", backupHandler.GetContentFromBody)
		backups.POST("/:id", backupHandler.GetContent)
		backups.POST("/:id/delete", backupHandler.Delete)
		backups.POST("/batch-delete", backupHandler.BatchDelete)
		backups.POST("/diff", backupHandler.Diff)
		backups.POST("/:id/restore", backupHandler.Restore)
		backups.GET("/statistics", backupHandler.GetStatistics)
		backups.POST("/batch", backupHandler.BatchBackup)
		backups.GET("/version", backupHandler.GetByVersion)
		backups.GET("/history", backupHandler.GetHistory)
		backups.POST("/export", exportHandler.ExportBackups)
	}

	// ==================== 设备发现路由 ====================
	discoveries := r.Group("/discoveries")
	discoveries.Use(middleware.RequirePermissions([]string{
		"network:discovery:add",
		"network:discovery:view",
	}, core))
	{
		discoveries.POST("/list", discoveryHandler.List)
		discoveries.POST("/statistics", discoveryHandler.Statistics)
		discoveries.POST("/create", discoveryHandler.Create)
		discoveries.POST("/:id", discoveryHandler.GetByID)
		discoveries.POST("/:id/results", discoveryHandler.GetResults)
		discoveries.POST("/:id/execute", discoveryHandler.Execute)
		discoveries.POST("/:id/cancel", discoveryHandler.Cancel)
		discoveries.POST("/:id/delete", discoveryHandler.Delete)
		discoveries.POST("/export", exportHandler.ExportDiscoveries)
		discoveries.POST("/batch-delete", discoveryHandler.BatchDelete)
		discoveries.POST("/:id/import", discoveryHandler.ImportDevices)
	}

	// ==================== MAC地址管理路由（独立权限） ====================
	mac := r.Group("/mac")
	mac.Use(middleware.RequirePermissions([]string{
		"network:mac:query",
	}, core))
	{
		SetupMACRouter(mac, core, exportHandler)
	}

	// ==================== 端口状态管理路由（独立权限） ====================
	ports := r.Group("/ports")
	// 查询接口(/list)额外接受 ops 读权限: 信息点关联网络设备端口时需要读取端口列表,
	// 但运维角色不持有 network:port 权限。放宽只读; 写操作保持严格。
	ports.Use(middleware.RequirePermissionsWithQuery([]string{
		"network:port:query",
	}, middleware.OpsSelectorReadPerms, core))
	{
		SetupPortRouter(ports, core, exportHandler)
		// Phase 52: 写操作子组 /network/ports/write/* + 组级 RequirePermissions([network:port:write])
		SetupPortWriteRouter(ports, core)
	}

	// ==================== 批量导出路由（所有实体类型通用） ====================
	r.POST("/batch-export", exportHandler.BatchExport)

}
