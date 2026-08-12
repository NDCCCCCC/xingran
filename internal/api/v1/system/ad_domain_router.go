package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	addomainServices "github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
)

// SetupADDomainRouter 设置AD域管理路由
func SetupADDomainRouter(r *gin.RouterGroup, core *core.Core) {
	// Phase 36 / Phase 38 Wave 1: AD 服务账号池（多账号故障切换）
	// Phase 38 Wave 1: accountPool 单次创建后由 NewADDomainService 与 NewADAccountHandler
	// 共享同一实例（Pitfall 4：避免重复 New 导致内存缓存不共享，熔断后账号仍被选中）。
	accountPool := addomainServices.NewAccountPool(core.GetDB(), nil) // 无 Redis pub/sub 跨进程广播（单机部署）
	service := addomainServices.NewADDomainService(core.GetDB(), accountPool, core.SM4Cipher)
	handler := NewADDomainHandler(service, core)
	syncHandler := NewADUserSyncHandler(core)

	accountHandler := NewADAccountHandler(accountPool, core)

	// 旧的组同步路由已移除 - 使用新的OU组映射功能
	// addomainAPI.SetupGroupSyncRouter 已删除

	// AD配置管理
	configs := r.Group("/configs")
	configs.Use(middleware.RequirePermissions([]string{
		"ops:ad:config:list",
		"ops:ad:config:add",
		"ops:ad:config:edit",
		"ops:ad:config:delete",
		"ops:ad:config:test",
		"ops:ad:config:sync",
	}, core))
	{
		configs.POST("/list", handler.ListConfigs)
		configs.POST("", handler.CreateConfig)
		configs.GET("/:id", handler.GetConfig)
		configs.POST("/:id/update", handler.UpdateConfig)
		configs.POST("/:id/delete", handler.DeleteConfig)
		configs.POST("/:id/test", handler.TestConnection)
		configs.POST("/:id/sync", handler.SyncData)
	}

	// Phase 36: 服务账号池（8 个 POST 端点，按动作类型独立权限）
	// 修复 REVIEWS.md opencode H4：避免共享 2 个权限导致越权
	// 权限粒度（与 menu seed migration 163 对齐）:
	//   list/stats → ops:ad:config:account:list
	//   create/update/enable/disable/unlock → ops:ad:config:account:edit
	//   delete → ops:ad:config:account:delete
	accounts := r.Group("/accounts")
	{
		// 只读端点（list/stats）
		readGroup := accounts.Group("")
		readGroup.Use(middleware.RequirePermissions([]string{"ops:ad:config:account:list"}, core))
		{
			readGroup.POST("/list", accountHandler.List)
			readGroup.POST("/stats", accountHandler.Stats)
		}

		// 写端点（create/update/enable/disable/unlock）
		writeGroup := accounts.Group("")
		writeGroup.Use(middleware.RequirePermissions([]string{"ops:ad:config:account:edit"}, core))
		{
			writeGroup.POST("/create", accountHandler.Create)
			writeGroup.POST("/update", accountHandler.Update)
			writeGroup.POST("/enable", accountHandler.Enable)
			writeGroup.POST("/disable", accountHandler.Disable)
			writeGroup.POST("/unlock", accountHandler.Unlock)
		}

		// 删除端点（独立权限，最敏感）
		deleteGroup := accounts.Group("")
		deleteGroup.Use(middleware.RequirePermissions([]string{"ops:ad:config:account:delete"}, core))
		{
			deleteGroup.POST("/delete", accountHandler.Delete)
		}
	}

	// OU管理
	ous := r.Group("/ous")
	ous.Use(middleware.RequirePermissions([]string{"ops:ad:ou:view"}, core))
	{
		ous.POST("/tree", handler.GetOUTree)
	}

	// 用户组管理路由
	groups := r.Group("/groups")
	groups.Use(middleware.RequirePermissions([]string{
		"ops:ad:group:view",
		"ops:ad:group:edit",
	}, core))
	{
		groups.POST("/list", handler.ListGroups)
		// sync-status 已恢复 - groups/index.tsx UI 依赖此端点显示总组数/已同步/未同步/成员关系数
		// 历史说明：旧的"依赖部门-组映射"实现已迁移到 OU 组映射；handler.GetGroupSyncStatus 现直接基于 groups/dept_ou_mapping 聚合统计
		groups.POST("/sync-status", handler.GetGroupSyncStatus)
		groups.POST("/:id", handler.GetGroupDetail)
		groups.POST("/:id/update", handler.UpdateGroup)
		groups.POST("/:id/members", handler.GetGroupMembers)
	}

	// 用户管理
	users := r.Group("/users")
	users.Use(middleware.RequirePermissions([]string{
		"ops:ad:user:view",
		"ops:ad:user:edit",
	}, core))
	{
		users.POST("/list", handler.ListUsers)
		users.POST("/ids", handler.GetADUserIds)
		users.GET("/:id", handler.GetUserDetail)
		users.POST("/:id/update", handler.UpdateUser)
		users.POST("/:id/move", handler.MoveUser)
		users.POST("/:id/enable", handler.EnableUser)
		users.POST("/:id/disable", handler.DisableUser)
		users.POST("/batch-sync", syncHandler.BatchSyncADUsers)
	}

	// 同步日志
	logs := r.Group("/logs")
	logs.Use(middleware.RequirePermissions([]string{"ops:ad:log:view"}, core))
	{
		logs.POST("/list", handler.ListSyncLogs)
	}

	// 电脑设备管理
	computers := r.Group("/computers")
	computers.Use(middleware.RequirePermissions([]string{
		"ops:ad:computer:view",
		"ops:ad:computer:detail",
	}, core))
	{
		computers.POST("/list", handler.ListComputers)
		computers.POST("/detail", handler.GetComputerDetail)
	}

	// OU组映射管理
	SetupOUGroupMappingRouter(r, core)
}
