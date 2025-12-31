package api

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/scheduler"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/websocket"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/middleware"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/permission"
)

// noticeHubAdapter 适配器，将 websocket.NoticeHub 适配为 scheduler.NoticeHub
type noticeHubAdapter struct {
	hub *websocket.NoticeHub
}

func (a *noticeHubAdapter) BroadcastToUsers(userIDs []string, message interface{}) {
	if msg, ok := message.(websocket.NoticeMessage); ok {
		a.hub.BroadcastToUsers(userIDs, msg)
	}
}

// SetupRouter 设置路由
func SetupRouter(r *gin.RouterGroup, core *core.Core) {
	// 添加请求ID中间件
	r.Use(middleware.RequestID())

	// 创建并启动通知中心
	noticeHub := websocket.NewNoticeHub()
	go noticeHub.Run()
	log.Println("通知中心已启动")
	// 将 NoticeHub 设置到 Core 中，供其他服务使用
	core.NoticeHub = noticeHub
	// 将 NoticeHub 设置到调度器全局变量，供通知发布定时任务使用
	// 使用适配器将 websocket.NoticeHub 转换为 scheduler.NoticeHub
	scheduler.SetNoticeHub(&noticeHubAdapter{hub: noticeHub})

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
			// 用户菜单和权限接口（所有已登录用户都可访问，不需要特定权限）
			// user-menus 和 user-permissions 放在权限检查之前
			v1.SetupUserMenuRouter(authorized, core)

			// 个人信息相关接口（所有已登录用户可访问）
			profile := authorized.Group("/profile")
			{
				v1.SetupProfileRouter(profile, core)
			}

			// 系统设置相关接口（所有已登录用户可访问）
			settings := authorized.Group("/settings")
			{
				v1.SetupSettingsRouter(settings, core)
			}

			// 通知配置管理（需要系统配置权限）
			notificationConfigs := authorized.Group("/settings")
			notificationConfigs.Use(middleware.RequirePermissions([]string{
				string(permission.SystemConfig),
			}, core))
			{
				v1.SetupNotificationConfigRouter(notificationConfigs, core)
			}

			// 用户管理
			users := authorized.Group("/users")
			users.Use(middleware.RequirePermissions([]string{
				string(permission.UserList),
				string(permission.UserAdd),
				string(permission.UserEdit),
				string(permission.UserView),
			}, core))
			// 添加数据权限中间件
			users.Use(middleware.DataScopePermission(core))
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

			// 菜单管理（需要菜单管理权限）
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
			depts := authorized.Group("/departments")
			depts.Use(middleware.RequirePermissions([]string{
				string(permission.DeptList),
				string(permission.DeptAdd),
				string(permission.DeptEdit),
				string(permission.DeptView),
			}, core))
			// 添加数据权限中间件
			depts.Use(middleware.DataScopePermission(core))
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

			// 工位管理
			workstations := authorized.Group("/workstations")
			workstations.Use(middleware.RequirePermissions([]string{
				string(permission.WorkstationList),
				string(permission.WorkstationAdd),
				string(permission.WorkstationEdit),
				string(permission.WorkstationView),
			}, core))
			{
				v1.SetupWorkstationRouter(workstations, core)
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

			// 验证码背景图管理（所有已登录用户可访问，作为系统设置的一部分）
			captchaBackgrounds := authorized.Group("/captcha-backgrounds")
			v1.SetupCaptchaBackgroundRouter(captchaBackgrounds, core)

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

			// 用户端通知（所有已登录用户可访问，不需要特定权限）
			v1.SetupNoticeUserRouter(authorized, core)

			// WebSocket路由（需要认证，但不需要权限验证）
			wsGroup := authorized.Group("/ws")
			wsGroup.Use(middleware.CORSForWebSocket())
			{
				v1.SetupNoticeWebSocketRouter(wsGroup, noticeHub, core)
			}
		}
	}

	// 监控模块
	monitor := r.Group("/monitor")
	monitor.Use(middleware.JWTAuth(core.JWTManager))
	{
		v1.SetupMonitorRouter(monitor, core)
	}

	// 网络设备管理模块
	network := r.Group("/network")
	network.Use(middleware.JWTAuth(core.JWTManager))
	{
		// 网络设备管理
		devices := network.Group("/devices")
		devices.Use(middleware.RequirePermissions([]string{
			"network:device:list",
			"network:device:add",
			"network:device:edit",
			"network:device:delete",
		}, core))
		{
			v1.SetupNetworkDeviceRouter(devices, core)
		}

		// 授权凭证管理
		credentials := network.Group("/credentials")
		credentials.Use(middleware.RequirePermissions([]string{
			"network:credential:list",
			"network:credential:add",
			"network:credential:edit",
			"network:credential:delete",
		}, core))
		{
			v1.SetupAuthCredentialRouter(credentials, core)
		}

		// 配置模板管理
		templates := network.Group("/templates")
		templates.Use(middleware.RequirePermissions([]string{
			"network:template:list",
			"network:template:add",
			"network:template:edit",
			"network:template:delete",
		}, core))
		{
			v1.SetupConfigTemplateRouter(templates, core)
		}

		// 命令分发
		command := network.Group("/command")
		command.Use(middleware.RequirePermissions([]string{
			"network:command:execute",
			"network:command:view",
		}, core))
		{
			v1.SetupCommandRouter(command, core)
		}

		// 配置执行
		executions := network.Group("/executions")
		executions.Use(middleware.RequirePermissions([]string{
			"network:command:execute",
			"network:command:view",
		}, core))
		{
			v1.SetupConfigExecutionRouter(executions, core)
		}

		// 配置备份
		backups := network.Group("/backups")
		backups.Use(middleware.RequirePermissions([]string{
			"network:backup:list",
			"network:backup:add",
			"network:backup:restore",
			"network:backup:diff",
		}, core))
		{
			v1.SetupConfigBackupRouter(backups, core)
		}

		// 设备发现
		discoveries := network.Group("/discoveries")
		discoveries.Use(middleware.RequirePermissions([]string{
			"network:discovery:add",
			"network:discovery:view",
		}, core))
		{
			v1.SetupDeviceDiscoveryRouter(discoveries, core)
		}

		// MAC地址管理
		macAddrs := network.Group("/mac")
		macAddrs.Use(middleware.RequirePermissions([]string{
			"network:mac:query",
		}, core))
		{
			v1.SetupMACAddressRouter(macAddrs, core)
		}

		// 端口状态管理
		ports := network.Group("/ports")
		ports.Use(middleware.RequirePermissions([]string{
			"network:port:query",
		}, core))
		{
			v1.SetupPortStatusRouter(ports, core)
		}
	}
}