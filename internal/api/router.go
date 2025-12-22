package api

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/middleware"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/permission"
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
			users.Use(middleware.RequirePermissions([]string{
				string(permission.UserList),
				string(permission.UserAdd),
				string(permission.UserEdit),
				string(permission.UserView),
			}, core))
			{
				v1.SetupUserRouter(users, core)
			}

			// 角色管理
			roles := authorized.Group("/roles")
			roles.Use(middleware.RequirePermissions([]string{
				string(permission.RoleList),
				string(permission.RoleAdd),
				string(permission.RoleEdit),
				string(permission.RoleView),
			}, core))
			{
				v1.SetupRoleRouter(roles, core)
			}

			// 菜单管理
			menus := authorized.Group("/menus")
			menus.Use(middleware.RequirePermissions([]string{
				string(permission.MenuList),
				string(permission.MenuAdd),
				string(permission.MenuEdit),
				string(permission.MenuView),
			}, core))
			{
				v1.SetupMenuRouter(menus, core)
			}

			// 部门管理
			depts := authorized.Group("/depts")
			depts.Use(middleware.RequirePermissions([]string{
				string(permission.DeptList),
				string(permission.DeptAdd),
				string(permission.DeptEdit),
				string(permission.DeptView),
			}, core))
			{
				v1.SetupDeptRouter(depts, core)
			}

			// 岗位管理
			posts := authorized.Group("/posts")
			posts.Use(middleware.RequirePermissions([]string{
				string(permission.PostList),
				string(permission.PostAdd),
				string(permission.PostEdit),
				string(permission.PostView),
			}, core))
			{
				v1.SetupPostRouter(posts, core)
			}

			// 字典管理
			dicts := authorized.Group("/dicts")
			dicts.Use(middleware.RequirePermissions([]string{
				string(permission.DictTypeList),
				string(permission.DictTypeAdd),
				string(permission.DictTypeEdit),
				string(permission.DictTypeView),
			}, core))
			{
				v1.SetupDictRouter(dicts, core)
			}

			// 参数配置
			configs := authorized.Group("/configs")
			configs.Use(middleware.RequirePermissions([]string{
				string(permission.ConfigList),
				string(permission.ConfigAdd),
				string(permission.ConfigEdit),
				string(permission.ConfigView),
			}, core))
			{
				v1.SetupConfigRouter(configs, core)
			}

			// 通知公告
			notices := authorized.Group("/notices")
			notices.Use(middleware.RequirePermissions([]string{
				string(permission.NoticeList),
				string(permission.NoticeAdd),
				string(permission.NoticeEdit),
				string(permission.NoticeView),
			}, core))
			{
				v1.SetupNoticeRouter(notices, core)
			}

			// 操作日志
			operLogs := authorized.Group("/oper-logs")
			operLogs.Use(middleware.RequirePermissions([]string{
				string(permission.OperLogList),
				string(permission.OperLogView),
				string(permission.OperLogExport),
			}, core))
			{
				v1.SetupOperLogRouter(operLogs, core)
			}

			// 登录日志
			loginLogs := authorized.Group("/login-logs")
			loginLogs.Use(middleware.RequirePermissions([]string{
				string(permission.LoginLogList),
				string(permission.LoginLogView),
				string(permission.LoginLogExport),
			}, core))
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