package api

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/middleware"
)

// SetupRouter 设置路由
func SetupRouter(r *gin.RouterGroup, core *core.Core) {
	// 添加请求ID中间件
	r.Use(middleware.RequestID())

	// 系统管理模块
	system := r.Group("/system")
	{
		// 认证相关（无需认证）
		auth := system.Group("/auth")
		{
			v1.SetupAuthRouter(auth, core)
		}

		// 需要认证的接口
		authorized := system.Group("")
		authorized.Use(middleware.JWTAuth(core.JWTManager))
		{
			// 用户管理
			users := authorized.Group("/users")
			{
				v1.SetupUserRouter(users, core)
			}

			// 角色管理
			roles := authorized.Group("/roles")
			{
				v1.SetupRoleRouter(roles, core)
			}

			// 菜单管理
			menus := authorized.Group("/menus")
			{
				v1.SetupMenuRouter(menus, core)
			}

			// 部门管理
			depts := authorized.Group("/depts")
			{
				v1.SetupDeptRouter(depts, core)
			}

			// 岗位管理
			posts := authorized.Group("/posts")
			{
				v1.SetupPostRouter(posts, core)
			}

			// 字典管理
			dicts := authorized.Group("/dicts")
			{
				v1.SetupDictRouter(dicts, core)
			}

			// 参数配置
			configs := authorized.Group("/configs")
			{
				v1.SetupConfigRouter(configs, core)
			}

			// 通知公告
			notices := authorized.Group("/notices")
			{
				v1.SetupNoticeRouter(notices, core)
			}

			// 操作日志
			operLogs := authorized.Group("/oper-logs")
			{
				v1.SetupOperLogRouter(operLogs, core)
			}

			// 登录日志
			loginLogs := authorized.Group("/login-logs")
			{
				v1.SetupLoginLogRouter(loginLogs, core)
			}
		}
	}

	// 监控模块
	monitor := r.Group("/monitor")
	monitor.Use(middleware.JWTAuth(core.JWTManager))
	{
		v1.SetupMonitorRouter(monitor, core)
	}
}