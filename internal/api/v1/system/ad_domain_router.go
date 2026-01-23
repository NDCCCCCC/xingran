package system

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	addomainServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/middleware"
)

// SetupADDomainRouter 设置AD域管理路由
func SetupADDomainRouter(r *gin.RouterGroup, core *core.Core) {
	service := addomainServices.NewADDomainService(core.GetDB())
	handler := NewADDomainHandler(service, core)

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

	// OU管理
	ous := r.Group("/ous")
	ous.Use(middleware.RequirePermissions([]string{"ops:ad:ou:view"}, core))
	{
		ous.POST("/tree", handler.GetOUTree)
	}

	// 用户组管理
	groups := r.Group("/groups")
	groups.Use(middleware.RequirePermissions([]string{
		"ops:ad:group:view",
		"ops:ad:group:edit",
	}, core))
	{
		groups.POST("/list", handler.ListGroups)
		groups.GET("/:id", handler.GetGroupDetail)
		groups.POST("/:id/update", handler.UpdateGroup)
	}

	// 用户管理
	users := r.Group("/users")
	users.Use(middleware.RequirePermissions([]string{
		"ops:ad:user:view",
		"ops:ad:user:edit",
	}, core))
	{
		users.POST("/list", handler.ListUsers)
		users.GET("/:id", handler.GetUserDetail)
	}

	// 同步日志
	logs := r.Group("/logs")
	logs.Use(middleware.RequirePermissions([]string{"ops:ad:log:view"}, core))
	{
		logs.POST("/list", handler.ListSyncLogs)
	}
}
