// Package dbbackup 全库备份与恢复的 HTTP API 层。
//
// 本包只负责路由注册、请求解析、响应包装、操作日志与敏感字段脱敏；
// 业务逻辑一律委托 internal/services/dbbackup.Service（由其它代理实现）。
// 路由权限中间件由 lead 在 internal/api/router.go 里统一挂载，本文件不重复挂。
package dbbackup

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/dbbackup"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
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

	// 备份计划凭据加密器（dbbackup.RegisterCredentialCipher）已在 core.Init 注册：
	// core.Init 不依赖路由装配，定时任务 svc 与任意部署形态都可用；此处不再重复注册。

	// 注册恢复收尾的进程内菜单/权限缓存清理钩子。
	// 整库恢复替换 sys_user/sys_menu/sys_role 等鉴权表后，必须清掉缓存里的旧
	// 菜单树/角色权限/用户详情/权限判定结果，否则恢复后仍按旧鉴权放行。
	// 这里按缓存键前缀逐模式删除（core.Cache 自动补 "xingran:" 前缀，故用不带
	// 前缀的 pattern），覆盖 system/cache_keys.go 中 menu/role/user/perm/dept/config
	// 与 middleware 的严格权限缓存前缀。失败仅返回 error 由收尾处记日志、不阻断终态。
	dbbackup.RegisterRestoreCacheFlusher(func(ctx context.Context) error {
		return flushRestoreCaches(ctx, core.Cache)
	})

	svc := dbbackup.NewService(core.GetDB(), dbCfg, nil, "")
	return &DBBackupHandler{
		service: svc,
		core:    core,
	}
}

// flushRestoreCaches 恢复收尾按前缀清理进程内菜单/权限缓存（具名包级函数，
// 便于单测直接驱动全部成功/失败分支；注册闭包仅做参数捕获）。
func flushRestoreCaches(ctx context.Context, c cache.Cache) error {
	if c == nil {
		return fmt.Errorf("core.Cache 未初始化，无法清理恢复后缓存")
	}
	patterns := []string{
		"menu:*", "role:*", "user:*", "dept:*", "post:*", "dict:*",
		"config:*", "permission:*",
		// "strict|" 是 pkg/middleware/permission.go strictCacheKeyPrefix 的真实键前缀
		// （竖线分隔符，不是笔误），尾部 * 为 Redis glob 通配。
		"strict|*", "cache:*",
	}
	var firstErr error
	for _, pattern := range patterns {
		keys, err := c.Keys(ctx, pattern)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("查询缓存键 %q: %w", pattern, err)
			}
			applogger.Warnf("[全库备份] 恢复收尾查询缓存键 %q 失败: %v", pattern, err)
			continue
		}
		if len(keys) == 0 {
			continue
		}
		if err := c.MDelete(ctx, keys...); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("删除缓存键 %q: %w", pattern, err)
			}
			applogger.Warnf("[全库备份] 恢复收尾删除缓存键 %q 失败: %v", pattern, err)
		}
	}
	return firstErr
}
