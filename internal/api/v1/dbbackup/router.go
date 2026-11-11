// Package dbbackup 全库备份与恢复的 HTTP API 层。
//
// 本包只负责路由注册、请求解析、响应包装、操作日志与敏感字段脱敏；
// 业务逻辑一律委托 internal/services/dbbackup.Service（由其它代理实现）。
// 路由权限中间件由 lead 在 internal/api/router.go 里统一挂载，本文件不重复挂。
package dbbackup

import (
	"github.com/gin-gonic/gin"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/dbbackup"
)

// DBBackupWriteRouteGroups 全库备份**写操作**路由分组（85-03 RBAC-01 逐动作拆分）。
// 四个分组分别由 lead 挂到 system:backup:export / system:backup:restore 权限点：
//   - Export：立即备份导出（POST /export）
//   - Restore：上传并恢复（POST /restore）
//   - Edit：保存备份计划（POST /plan）
//   - Remove：删除备份记录（POST /:id/delete）
type DBBackupWriteRouteGroups struct {
	Export  *gin.RouterGroup
	Restore *gin.RouterGroup
	Edit    *gin.RouterGroup
	Remove  *gin.RouterGroup
}

// SetupDBBackupRouter 设置全库备份**只读**路由。
// 对应契约第 1 节：list / statistics / tasks/:id / plan GET / :id/download。
// 只读端点挂在 ANY-of 读组上（权限中间件由 router.go 挂）。
func SetupDBBackupRouter(r *gin.RouterGroup, core *core.Core) {
	h := NewDBBackupHandler(core)

	// 备份记录分页
	r.POST("/list", h.List)
	// 统计卡
	r.POST("/statistics", h.Statistics)
	// 能力预检（pg_dump/pg_restore 是否可用），页面触发导出前提示
	r.GET("/capabilities", h.Capabilities)
	// 异步任务状态轮询
	r.GET("/tasks/:id", h.GetTask)
	// 读取备份计划（敏感字段脱敏后返回）
	r.GET("/plan", h.GetPlan)
	// 下载备份 zip（二进制流，不套 response 包装）
	r.GET("/:id/download", h.Download)
}

// SetupDBBackupWriteRouter 设置全库备份**写**路由（读写分离，逐动作严格权限子组）。
// 分组由 lead 在 router.go 里构造并传入，本函数只把 handler 绑到对应子组。
func SetupDBBackupWriteRouter(groups DBBackupWriteRouteGroups, core *core.Core) {
	h := NewDBBackupHandler(core)

	// 触发导出（返回 taskId）
	if groups.Export != nil {
		groups.Export.POST("/export", h.Export)
	}
	// 上传 zip 并触发恢复（返回 taskId）
	if groups.Restore != nil {
		groups.Restore.POST("/restore", h.Restore)
	}
	// 保存备份计划（存储配置、cron、开关、保留份数）
	if groups.Edit != nil {
		groups.Edit.POST("/plan", h.SavePlan)
	}
	// 删除备份记录与文件
	if groups.Remove != nil {
		groups.Remove.POST("/:id/delete", h.Delete)
	}
}

// NewDBBackupHandler 构造全库备份处理器。
//
// 从 core.Config.Database 组装 dbbackup.DatabaseConfig（数据库连接信息），
// exec 传 nil 表示使用服务层默认生产 DumpExecutor，backupDir 传空串表示
// 使用服务层默认 "data/backups"（运行时目录，已被 .gitignore 忽略，不在仓库创建）。
// core 一并注入处理器，供 operlog.Record 访问 OperLogService 与 GetDB()。
func NewDBBackupHandler(core *core.Core) *DBBackupHandler {
	dbCfg := dbbackup.DatabaseConfig{}
	if core != nil && core.Config != nil {
		cfg := core.Config.Database
		dbCfg = dbbackup.DatabaseConfig{
			Type:          cfg.Type,
			Path:          cfg.Path,
			Host:          cfg.Host,
			Port:          cfg.Port,
			User:          cfg.User,
			Password:      cfg.Password,
			DBName:        cfg.DBName,
			SSLMode:       cfg.SSLMode,
			PgDumpPath:    cfg.Backup.PgDumpPath,
			PgRestorePath: cfg.Backup.PgRestorePath,
		}
	}

	// core 由 router.go 注入，正常启动路径下必非 nil；此处保留防御，
	// 避免单测/异常装配时直接 panic。
	if core == nil {
		return &DBBackupHandler{service: nil, core: nil}
	}

	svc := dbbackup.NewService(core.GetDB(), dbCfg, nil, "")
	return &DBBackupHandler{
		service: svc,
		core:    core,
	}
}
