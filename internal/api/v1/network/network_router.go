package network

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	networkServices "github.com/xingran-next/xingran-go-backend/internal/services/network"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
)

// strictGroup 创建一个「单一权限点 + 严格权限」的写操作路由子组
// （85-03 RBAC-01，与 internal/api/router.go 的 strictUserRouteGroup 同构：
// RequireStrictPermissions 只做 sys_menu.perms 精确匹配，跳过 F 型子权限
// 继承与模块级降级）。父组必须不含 ANY-of 权限中间件。
func strictGroup(parent *gin.RouterGroup, perm permission.PermissionCode, core *core.Core) *gin.RouterGroup {
	g := parent.Group("")
	g.Use(middleware.RequireStrictPermissions([]string{string(perm)}, core))
	return g
}

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
	// Phase 93: 异步恢复任务服务（装配点与 backupService 同源；临时导出实例不在此接线）
	restoreTaskSvc := services.NewConfigRestoreTaskService(db, backupService, core.DeviceExecutor)
	restoreTaskSvc.RecoverStaleRunningTasks(context.Background()) // 启动收敛：残留 running 一次性置 failed（失败仅记日志，不阻断装配）
	discoveryService := core.DeviceDiscoveryService

	// 创建Handler
	credentialHandler := NewCredentialHandler(credentialService).WithCore(core)
	deviceHandler := NewDeviceHandler(deviceService).WithCore(core)
	templateHandler := NewTemplateHandler(templateService).WithCore(core)
	commandHandler := NewCommandHandler(commandService, db).WithCore(core)
	executionHandler := NewExecutionHandler(executionService).WithCore(core)
	backupHandler := NewBackupHandler(backupService, restoreTaskSvc, db).WithCore(core)
	discoveryHandler := NewDiscoveryHandler(discoveryService).WithCore(core)
	exportHandler := NewNetworkExportHandler(core)

	// ==================== 网络设备路由 ====================
	// 85-03 RBAC-01：读组保留组级 ANY-of（含 OpsSelectorReadPerms queryExtra 放行），
	// 写路由搬出到**兄弟**写组逐动作 RequireStrictPermissions 单一权限点
	// （写组绝不做成读组子组——gin Group 继承父中间件链会先放行 ANY-of）。
	// 域内 strict 兄弟写组复用 router.go 的 strictUserRouteGroup 同构写法，
	// 本文件用局部 helper strictGroup 保持一致。
	devices := r.Group("/devices")
	// 查询接口(/list)额外接受 ops 读权限(OpsSelectorReadPerms): 信息点管理等运维页面
	// 关联网络设备时需要读取设备列表做选择器, 但运维角色不持有 network:device 权限。
	// 放宽只读路径; 写操作见下方 devicesWrites 严格兄弟组。
	devices.Use(middleware.RequirePermissionsWithQuery([]string{
		"network:device:list",
		"network:device:add",
		"network:device:edit",
		"network:device:delete",
	}, middleware.OpsSelectorReadPerms, core))
	{
		devices.POST("/list", deviceHandler.List)
		devices.POST("/statistics", deviceHandler.Statistics)
		devices.POST("/discover", discoveryHandler.Probe)
		devices.POST("/:id", deviceHandler.GetByID)
	}
	devicesWrites := r.Group("/devices")
	{
		devicesAdd := strictGroup(devicesWrites, permission.NetworkDeviceAdd, core)
		devicesAdd.POST("", deviceHandler.Create)
		devicesAdd.POST("/quick-create", deviceHandler.QuickCreate)
		devicesEdit := strictGroup(devicesWrites, permission.NetworkDeviceEdit, core)
		devicesEdit.POST("/:id/update", deviceHandler.Update)
		devicesDelete := strictGroup(devicesWrites, permission.NetworkDeviceDelete, core)
		devicesDelete.POST("/:id/delete", deviceHandler.Delete)
		devicesDelete.POST("/batch-delete", deviceHandler.BatchDelete)
		devicesExport := strictGroup(devicesWrites, permission.NetworkDeviceExport, core)
		devicesExport.POST("/export", exportHandler.ExportDevices)
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
		credentials.POST("/:id", credentialHandler.GetByID)
		credentials.POST("/:id/devices", credentialHandler.GetDevicesByCredential)
	}
	credentialsWrites := r.Group("/credentials")
	{
		credentialsAdd := strictGroup(credentialsWrites, permission.NetworkCredentialAdd, core)
		credentialsAdd.POST("", credentialHandler.Create)
		credentialsEdit := strictGroup(credentialsWrites, permission.NetworkCredentialEdit, core)
		credentialsEdit.POST("/:id/update", credentialHandler.Update)
		credentialsEdit.POST("/:id/set-default", credentialHandler.SetDefault)
		credentialsDelete := strictGroup(credentialsWrites, permission.NetworkCredentialDelete, core)
		credentialsDelete.POST("/:id/delete", credentialHandler.Delete)
		credentialsDelete.POST("/batch-delete", credentialHandler.BatchDelete)
		credentialsExport := strictGroup(credentialsWrites, permission.NetworkCredentialExport, core)
		credentialsExport.POST("/export", exportHandler.ExportCredentials)
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
		templates.POST("/:id", templateHandler.GetByID)
		templates.POST("/:id/variables", templateHandler.GetVariables)
	}
	templatesWrites := r.Group("/templates")
	{
		templatesAdd := strictGroup(templatesWrites, permission.NetworkTemplateAdd, core)
		templatesAdd.POST("", templateHandler.Create)
		templatesAdd.POST("/:id/clone", templateHandler.Clone) // 克隆=创建副本
		templatesEdit := strictGroup(templatesWrites, permission.NetworkTemplateEdit, core)
		templatesEdit.POST("/:id/update", templateHandler.Update)
		templatesDelete := strictGroup(templatesWrites, permission.NetworkTemplateDelete, core)
		templatesDelete.POST("/:id/delete", templateHandler.Delete)
		templatesDelete.POST("/batch-delete", templateHandler.BatchDelete)
		templatesView := strictGroup(templatesWrites, permission.NetworkTemplateView, core)
		templatesView.POST("/:id/preview", templateHandler.Preview) // 读语义端点（写路由口径挂 view 严格单码）
		templatesExport := strictGroup(templatesWrites, permission.NetworkTemplateExport, core)
		templatesExport.POST("/export", exportHandler.ExportTemplates)
	}

	// ==================== 命令分发路由 ====================
	command := r.Group("/command")
	command.Use(middleware.RequirePermissions([]string{
		"network:command:execute",
		"network:command:view",
	}, core))
	{
		command.POST("/list", commandHandler.List)
		command.POST("/statistics", commandHandler.Statistics)
		command.POST("/:id", commandHandler.GetExecutionResult)
		command.POST("/:id/device/:deviceId", commandHandler.GetDeviceExecutionDetail)
	}
	commandWrites := r.Group("/command")
	{
		commandExec := strictGroup(commandWrites, permission.NetworkCommandExecute, core)
		commandExec.POST("/dispatch", commandHandler.Dispatch)
		commandExec.POST("/quick", commandHandler.QuickCommand)
		commandExport := strictGroup(commandWrites, permission.NetworkCommandExport, core)
		commandExport.POST("/export", exportHandler.ExportCommands)
	}

	// ==================== 配置执行路由 ====================
	executions := r.Group("/executions")
	executions.Use(middleware.RequirePermissions([]string{
		"network:command:execute",
	}, core))
	{
		executions.POST("/list", executionHandler.List)
		executions.POST("/statistics", executionHandler.Statistics)
		executions.POST("/:id", executionHandler.GetByID)
		executions.POST("/:id/cancel", executionHandler.Cancel)
	}
	// 执行写操作严格组（执行记录删除/导出归 execute 域码——85-03 裁定零新码）
	executionsWrites := r.Group("/executions")
	{
		executionsExec := strictGroup(executionsWrites, permission.NetworkCommandExecute, core)
		executionsExec.POST("/template/execute", executionHandler.ExecuteByTemplate)
		executionsExec.POST("/:id/delete", executionHandler.Delete)
		executionsExec.POST("/batch-delete", executionHandler.BatchDelete)
		executionsExec.POST("/export", exportHandler.ExportExecutions)
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
		backups.POST("/content", backupHandler.GetContentFromBody)
		backups.POST("/:id", backupHandler.GetContent)
		// Phase 93: 恢复任务查询双端点（读，D-31）
		backups.POST("/restore-tasks/list", backupHandler.ListRestoreTasks)
		backups.POST("/restore-tasks/:id", backupHandler.GetRestoreTask)
		backups.GET("/statistics", backupHandler.GetStatistics)
		backups.GET("/version", backupHandler.GetByVersion)
		backups.GET("/history", backupHandler.GetHistory)
	}
	backupsWrites := r.Group("/backups")
	{
		backupsAdd := strictGroup(backupsWrites, permission.NetworkBackupAdd, core)
		backupsAdd.POST("", backupHandler.Create)
		backupsAdd.POST("/batch", backupHandler.BatchBackup)
		backupsDiff := strictGroup(backupsWrites, permission.NetworkBackupDiff, core)
		backupsDiff.POST("/diff", backupHandler.Diff)
		backupsDelete := strictGroup(backupsWrites, permission.NetworkBackupDelete, core)
		backupsDelete.POST("/:id/delete", backupHandler.Delete)
		backupsDelete.POST("/batch-delete", backupHandler.BatchDelete)
		backupsExport := strictGroup(backupsWrites, permission.NetworkBackupExport, core)
		backupsExport.POST("/export", exportHandler.ExportBackups)
	}

	// ==================== 配置恢复路由（高危下发，单独收口 v129-recheck C-3） ====================
	// 85-03：原 restores 为 backups ANY-of 读组的**子组**，父组 ANY-of 先于组内
	// 单码检查放行（且 RequirePermissions 非 Strict）。改为兄弟组 + RequireStrictPermissions
	// 单一权限点（Phase 52/56 端口写同模式），封掉 Step2 F 型继承与 Step3 模块降级。
	restores := r.Group("/backups")
	restores.Use(middleware.RequireStrictPermissions([]string{
		"network:backup:restore",
	}, core))
	{
		restores.POST("/:id/restore", backupHandler.Restore)
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
		discoveries.POST("/:id", discoveryHandler.GetByID)
		discoveries.POST("/:id/results", discoveryHandler.GetResults)
		discoveries.POST("/:id/cancel", discoveryHandler.Cancel)
	}
	discoveriesWrites := r.Group("/discoveries")
	{
		discoveryAdd := strictGroup(discoveriesWrites, permission.NetworkDiscoveryAdd, core)
		discoveryAdd.POST("/create", discoveryHandler.Create)
		discoveryAdd.POST("/:id/execute", discoveryHandler.Execute)
		discoveryAdd.POST("/:id/import", discoveryHandler.ImportDevices)
		discoveryDelete := strictGroup(discoveriesWrites, permission.NetworkDiscoveryDelete, core)
		discoveryDelete.POST("/:id/delete", discoveryHandler.Delete)
		discoveryDelete.POST("/batch-delete", discoveryHandler.BatchDelete)
		discoveryExport := strictGroup(discoveriesWrites, permission.NetworkDiscoveryExport, core)
		discoveryExport.POST("/export", exportHandler.ExportDiscoveries)
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
