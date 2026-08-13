package api

import (
	"github.com/gin-gonic/gin"
	v1 "github.com/xingran-next/xingran-go-backend/internal/api/v1"
	assetV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/asset"
	dutyV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/duty"
	v1_knowledge "github.com/xingran-next/xingran-go-backend/internal/api/v1/knowledge"
	monitorV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/monitor"
	networkV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/network"
	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations"
	rpaV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/rpa"
	systemV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/system"
	vdiV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/vdi"
	agentV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/agent"
	workorderV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/workorder"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	// AUTH-03 / D-01: MultiAuth + RateLimitByScope 位于 internal/middleware,
	// 与下方 pkg/middleware（别名 middleware）同 package 名，故取别名 internalmw。
	internalmw "github.com/xingran-next/xingran-go-backend/internal/middleware"
	"github.com/xingran-next/xingran-go-backend/internal/scheduler"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/asset"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
)

const permSystemConfig = "system:config"

// 跨模块只读选择器放行权限集 (部门树/用户/字典/网络设备选择器) 已提取到
// pkg/middleware.OpsSelectorReadPerms (单一来源, router.go 与 network_router.go 共用)。

// noticeHubAdapter 将 websocket.NoticeHub 适配为 scheduler.NoticeHub 接口
type noticeHubAdapter struct {
	hub *websocket.NoticeHub
}

func (a *noticeHubAdapter) BroadcastToUsers(userIDs []string, message interface{}) {
	if msg, ok := message.(websocket.NoticeMessage); ok {
		a.hub.BroadcastToUsers(userIDs, msg)
	}
}

// setupEncryptionMiddlewares 配置请求/响应加密中间件
func setupEncryptionMiddlewares(r *gin.RouterGroup, core *core.Core) *crypto.RequestEncryptor {
	var encryptor *crypto.RequestEncryptor

	if core.Config.Security.RequestEncryption.Enabled {
		sm2PrivateKey, sm2PublicKey := core.JWTManager.GetSM2KeyPair()
		if sm2PrivateKey != nil && sm2PublicKey != nil {
			// P1 fix (P1-S2): 从 security.replay_window_sec 读取时间戳容差,
			// 默认 60s。<=0 时 NewRequestEncryptorWithConfig 会 fallback 到 DefaultReplayWindowSec。
			encryptor = crypto.NewRequestEncryptorWithConfig(sm2PrivateKey, sm2PublicKey, crypto.RequestEncryptorConfig{
				ReplayWindowSec: core.Config.Security.ReplayWindowSec,
			})
			applogger.Infof("请求体加密器已初始化（SM2+SM4 混合加密,replay_window=%ds）", encryptor.ReplayWindowSec())

			decryptionConfig := &middleware.RequestDecryptionConfig{
				Enabled:           true,
				ExcludePaths:      core.Config.Security.RequestEncryption.ExcludePaths,
				RequireEncryption: core.Config.Security.RequestEncryption.RequireEncryption,
			}

			encryptionConfig := &middleware.ResponseEncryptionConfig{
				Enabled:      true,
				ExcludePaths: core.Config.Security.RequestEncryption.ExcludePaths,
			}

			r.Use(middleware.RequestDecryption(encryptor, decryptionConfig, core.GetDB()))
			r.Use(middleware.ResponseEncryption(encryptor, encryptionConfig, core.GetDB()))
			applogger.Infof("请求/响应加密中间件已启用（共享数据库配置）")
		} else {
			applogger.Warnf("SM2 密钥对未配置，无法启用请求体加密")
		}
	}

	return encryptor
}

// setupNoticeHub 初始化通知中心
func setupNoticeHub(r *gin.RouterGroup, core *core.Core) *websocket.NoticeHub {
	noticeHub := websocket.NewNoticeHub()
	go noticeHub.Run()
	applogger.Infof("通知中心已启动")

	core.NoticeHub = noticeHub
	scheduler.SetNoticeHub(&noticeHubAdapter{hub: noticeHub})

	return noticeHub
}

// SetupRouter 设置路由
func SetupRouter(r *gin.RouterGroup, core *core.Core, allowedOrigins []string) {
	r.Use(middleware.RequestID())

	setupEncryptionMiddlewares(r, core)
	noticeHub := setupNoticeHub(r, core)

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
		// 使用带黑名单检查的 JWT 认证中间件
		authorized.Use(middleware.JWTAuthWithBlacklist(core.JWTManager, core.TokenBlacklistService))
		// 添加操作日志中间件
		authorized.Use(middleware.OperLogMiddleware(core.OperLogService, core))
		{
			// 个人信息相关接口（所有已登录用户可访问）
			profile := authorized.Group("/profile")
			{
				systemV1.SetupProfileRouter(profile, core)
			}

			// 文件管理相关接口（所有已登录用户可访问）
			files := authorized.Group("/files")
			{
				systemV1.SetupFileRouter(files, core)
			}

			// 系统设置相关接口（所有已登录用户可访问）
			settings := authorized.Group("/settings")
			{
				// 新架构：结构体Handler + Service层
				systemV1.SetupSettingsRouter(settings, core)
			}

			// 列配置管理（所有已登录用户可访问）
			columnConfig := authorized.Group("/column-config")
			{
				systemV1.SetupColumnConfigRouter(columnConfig, core)
			}

			// 通知配置管理（需要系统配置权限）
			notificationConfigs := authorized.Group("/settings/notification")
			notificationConfigs.Use(middleware.RequirePermissions([]string{permSystemConfig}, core))
			{
				systemV1.SetupNotificationConfigRouter(notificationConfigs, core)
			}

			// 用户管理
			users := authorized.Group("/users")
			// 查询接口(/list)额外接受工位/空间读权限: 工位页面用户选择器(useWorkstationData)需要
			// 读取用户列表做归属选择, 但运维角色通常不持有 system:user 权限。用户列表受
			// DataScopePermission 限制可见数据; 写操作(create/update/delete)保持严格权限。
			users.Use(middleware.RequirePermissionsWithQuery([]string{
				string(permission.UserList),
				string(permission.UserAdd),
				string(permission.UserEdit),
				string(permission.UserView),
			}, []string{"ops:workstation:list", "ops:building:spaces:list"}, core))
			// 添加数据权限中间件
			users.Use(middleware.DataScopePermission(core))
			{
				// 新架构：结构体Handler + Service层
				systemV1.SetupUserRouter(users, core)
				// 用户 Excel 导入由 SetupUserRouter 内的 ImportUser 处理（带 AD 域控同步触发），
				// 不复用 operations.SetupExcelRouter —— 后者的 importData 为通用导入，不触发
				// AD 同步。模板下载同样在 user_router（GET /import/template）。
				// 若将来需要用户导出(/export)，可单独接入 operations.exportData("user")。
			}

			// 用户解锁路由（管理员权限）
			userUnlock := authorized.Group("/user")
			userUnlock.Use(middleware.RequirePermissions([]string{permSystemConfig}, core))
			{
				systemV1.SetupUserUnlockRouter(userUnlock, core)
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
				// 新架构：结构体Handler + Service层
				systemV1.SetupRoleRouter(roles, core)
			}

			// 用户菜单和权限接口（所有已登录用户可访问，不需要特定权限）
			systemV1.SetupUserMenuRouter(authorized.Group("/my-menus"), core)

			// 菜单管理（需要菜单管理权限）
			menus := authorized.Group("/menus")
			menus.Use(middleware.RequirePermissions([]string{
				string(permission.MenuList),
				string(permission.MenuAdd),
				string(permission.MenuEdit),
				string(permission.MenuView),
			}, core))
			{
				// 新架构：结构体Handler + Service层
				systemV1.SetupMenuRouter(menus, core)
			}

			// 部门管理
			depts := authorized.Group("/departments")
			// 查询接口(/tree,/list,/tree-select)额外接受运维读权限(opsSelectorReadPerms):
			// 楼宇/楼层/工位/机房等运维管理页面内嵌的 <DeptTree>/<DeptSidebar> 选择器需要读取部门树,
			// 但运维角色通常不持有 system:dept 权限, 导致部门树在每个运维页面都 403。
			// 部门树 GetTree 不应用数据权限, 返回组织结构全树(低敏感); 写操作保持严格权限。
			depts.Use(middleware.RequirePermissionsWithQuery([]string{
				string(permission.DeptList),
				string(permission.DeptAdd),
				string(permission.DeptEdit),
				string(permission.DeptView),
			}, middleware.OpsSelectorReadPerms, core))
			// 添加数据权限中间件
			depts.Use(middleware.DataScopePermission(core))
			{
				// 新架构：结构体Handler + Service层
				systemV1.SetupDepartmentRouter(depts, core)
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
				// 新架构：结构体Handler + Service层
				systemV1.SetupPostRouter(posts, core)
			}
			// API密钥管理
			apikeys := authorized.Group("/apikeys")
			apikeys.Use(middleware.RequirePermissions([]string{
				"system:apikey:list",
				"system:apikey:add",
				"system:apikey:edit",
				"system:apikey:delete",
			}, core))
			// AUTH-03 / D-01: 启用 X-API-Key 认证链（挂载范围严格限定本管理面路由组，D-02）。
			// 顺序: RequirePermissions → MultiAuth → RateLimitByScope。
			// D-03: 无 X-API-Key 头时 MultiAuth 直接 c.Next() 跳过，由上游 JWT 中间件接管（router 层不加 fallback 分支）。
			// D-04: IP 白名单严格拒绝沿用 internal/middleware/apikey.go 既有 isIPAllowed（本处不改中间件代码）。
			// 决策记录: .planning/notes/260813-auth03-enable-decision.md
			apikeys.Use(internalmw.MultiAuth(
				systemServices.NewAPIKeyService(core.GetDB()),
				services.NewUsageLogger(core.GetDB()),
			))
			apikeys.Use(internalmw.RateLimitByScope(services.NewRateLimiter()))
			{
				// 新架构：结构体Handler + Service层
				systemV1.SetupAPIKeyRouter(apikeys, core)
			}

			// 字典管理
			dicts := authorized.Group("/dicts")
			// 查询接口(/list,/data/list)额外接受 ops 读权限(opsSelectorReadPerms):
			// 字典是全局基础数据, useDict hook 几乎每个页面都调用 /system/dicts/data/list,
			// 但运维角色通常不持有 system:dict 权限。放宽只读路径; 写操作(增删改字典类型/数据)保持严格。
			dicts.Use(middleware.RequirePermissionsWithQuery([]string{
				string(permission.DictTypeList),
				string(permission.DictTypeAdd),
				string(permission.DictTypeEdit),
				string(permission.DictTypeView),
			}, middleware.OpsSelectorReadPerms, core))
			{
				systemV1.SetupDictRouter(dicts, core)
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
				systemV1.SetupConfigRouter(configs, core)
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
				systemV1.SetupNoticeRouter(notices, core)
			}

			// 用户端通知（所有已登录用户可访问，不需要特定权限）
			systemV1.SetupNoticeUserRouter(authorized, core)

			// 仪表盘系统（所有已登录用户可访问）
			dashboards := authorized.Group("/dashboards")
			{
				// 新架构：结构体Handler + Service层
				systemV1.SetupDashboardRouter(dashboards, core)
			}

			// WebSocket路由（需要认证，但不需要权限验证）
			wsGroup := authorized.Group("/ws")
			// 注: WebSocket 握手不走 HTTP CORS，跨域防护由 gorilla Upgrader CheckOrigin（fail-secure）承担
			{
				v1.SetupNoticeWebSocketRouter(wsGroup, noticeHub, core, allowedOrigins)
			}
		}
	}

	// 监控模块
	monitor := r.Group("/monitor")
	monitor.Use(middleware.JWTAuth(core.JWTManager))
	{
		// 使用新架构：结构体Handler + Service层
		monitorV1.SetupServerRouter(monitor, core)
		monitorV1.SetupCacheRouter(monitor, core)

		// 定时任务管理
		v1.RegisterJobRoutes(monitor, core)

		// 操作日志管理
		operLogs := monitor.Group("/oper-logs")
		{
			monitorV1.SetupOperLogRouter(operLogs, core)
		}

		// 登录日志管理
		loginLogs := monitor.Group("/login-logs")
		{
			monitorV1.SetupLoginLogRouter(loginLogs, core)
		}
	}

	// 网络设备管理模块（统一管理）
	network := r.Group("/network")
	network.Use(middleware.JWTAuth(core.JWTManager))
	// 添加操作日志中间件
	network.Use(middleware.OperLogMiddleware(core.OperLogService, core))
	{
		networkV1.SetupNetworkRouter(network, core)
		// 拓扑管理（MAC过滤规则）
		networkV1.SetupTopologyRouter(network, core.GetDB(), core)
		// MAC历史查询
		networkV1.SetupMACHistoryRouter(network, core)
	}

	// 值班管理模块（运维管理）
	duty := r.Group("/duty")
	duty.Use(middleware.JWTAuth(core.JWTManager))
	// 添加操作日志中间件
	duty.Use(middleware.OperLogMiddleware(core.OperLogService, core))
	{
		// 值班池管理
		dutyPools := duty.Group("/pools")
		dutyPools.Use(middleware.RequirePermissions([]string{
			"ops:duty:pool:list",
			"ops:duty:pool:add",
			"ops:duty:pool:edit",
			"ops:duty:pool:delete",
		}, core))
		{
			dutyV1.SetupDutyPoolsRouter(dutyPools, core)
		}

		// 排班管理
		dutySchedules := duty.Group("/schedules")
		dutySchedules.Use(middleware.RequirePermissions([]string{
			"ops:duty:schedule:list",
			"ops:duty:schedule:add",
			"ops:duty:schedule:edit",
			"ops:duty:schedule:delete",
		}, core))
		{
			dutyV1.SetupDutySchedulesRouter(dutySchedules, core)
		}

		// 节假日管理（所有已登录用户可查看）
		dutyV1.SetupDutyHolidaysRouter(duty.Group("/holidays"), core)

		// 值班配置管理（所有已登录用户可查看，管理员可修改）
		dutyConfig := duty.Group("/config")
		dutyConfig.Use(middleware.RequirePermissions([]string{permSystemConfig}, core))
		{
			dutyV1.SetupDutyConfigRouter(dutyConfig, core)
		}

		// 我的值班（所有已登录用户可访问）
		dutyV1.SetupMyDutyRouter(duty.Group("/my-duty"), core)
	}

	// 运维工单管理模块
	workorder := r.Group("/workorder")
	workorder.Use(middleware.JWTAuth(core.JWTManager))
	// 添加操作日志中间件
	workorder.Use(middleware.OperLogMiddleware(core.OperLogService, core))
	{
		// 工单管理
		orders := workorder.Group("/orders")
		orders.Use(middleware.RequirePermissions([]string{
			"ops:workorder:list",
			"ops:workorder:add",
			"ops:workorder:edit",
			"ops:workorder:delete",
		}, core))
		{
			workorderV1.SetupWorkOrdersRouter(orders, core)
			workorderV1.SetupWorkOrderRatingsRouter(orders, core) // 评价路由
		}

		// 工单分类管理
		categories := workorder.Group("/categories")
		categories.Use(middleware.RequirePermissions([]string{
			"ops:workorder:category:list",
			"ops:workorder:category:add",
			"ops:workorder:category:edit",
			"ops:workorder:category:delete",
		}, core))
		{
			workorderV1.SetupWorkOrderCategoriesRouter(categories, core)
		}

		// 周期性工单模板管理
		periodic := workorder.Group("/periodic")
		periodic.Use(middleware.RequirePermissions([]string{
			"ops:workorder:periodic:list",
			"ops:workorder:periodic:add",
			"ops:workorder:periodic:edit",
			"ops:workorder:periodic:delete",
		}, core))
		{
			workorderV1.SetupPeriodicWorkOrderRouter(periodic, core)
		}

		// 工单配置管理（需要系统配置权限）
		workorderConfig := workorder.Group("/config")
		workorderConfig.Use(middleware.RequirePermissions([]string{permSystemConfig}, core))
		{
			workorderV1.SetupWorkOrderConfigRouter(workorderConfig, core)
		}

		// 工单统计（所有已登录用户可查看）
		workorderV1.SetupWorkOrderStatisticsRouter(workorder.Group("/statistics"), core)
	}

	// 知识库管理模块
	knowledge := r.Group("/knowledge")
	knowledge.Use(middleware.JWTAuth(core.JWTManager))
	// 添加操作日志中间件
	knowledge.Use(middleware.OperLogMiddleware(core.OperLogService, core))
	{
		// 知识库文章管理
		articles := knowledge.Group("/articles")
		articles.Use(middleware.RequirePermissions([]string{
			"ops:knowledge:article:list",
			"ops:knowledge:article:add",
			"ops:knowledge:article:edit",
			"ops:knowledge:article:delete",
		}, core))
		{
			v1_knowledge.SetupArticleRouter(articles, core)
		}

		// 知识库分类管理
		knowledgeCategories := knowledge.Group("/categories")
		knowledgeCategories.Use(middleware.RequirePermissions([]string{
			"ops:knowledge:category:list",
			"ops:knowledge:category:add",
			"ops:knowledge:category:edit",
			"ops:knowledge:category:delete",
		}, core))
		{
			v1_knowledge.SetupCategoryRouter(knowledgeCategories, core)
		}

		// 知识库标签管理
		knowledgeTags := knowledge.Group("/tags")
		knowledgeTags.Use(middleware.RequirePermissions([]string{
			"ops:knowledge:tag:list",
			"ops:knowledge:tag:add",
			"ops:knowledge:tag:edit",
			"ops:knowledge:tag:delete",
		}, core))
		{
			v1_knowledge.SetupTagRouter(knowledgeTags, core)
		}

		// 知识库工单转换（工单转知识库）
		workorders := knowledge.Group("/workorders")
		v1_knowledge.SetupWorkOrderRouter(workorders, core)

		// 知识库搜索和查看（所有已登录用户可访问）
		v1_knowledge.SetupKnowledgeViewRouter(knowledge.Group("/view"), core)
	}

	// AD域管理模块
	adDomain := r.Group("/ad-domain")
	adDomain.Use(middleware.JWTAuth(core.JWTManager))
	// 添加操作日志中间件
	adDomain.Use(middleware.OperLogMiddleware(core.OperLogService, core))
	{
		systemV1.SetupADDomainRouter(adDomain, core)
		// OU部门映射路由
		systemV1.SetupOUMappingRouter(adDomain, core)
	}

	// 运维管理模块（物理资源管理）
	ops := r.Group("/ops")
	ops.Use(middleware.JWTAuth(core.JWTManager))
	// 添加操作日志中间件
	ops.Use(middleware.OperLogMiddleware(core.OperLogService, core))
	{
		// 楼宇管理
		buildings := ops.Group("/building")
		// 查询接口(/list)额外接受「楼宇空间」可视化读权限 ops:building:spaces:list，
		// 解决只读可视化页面复用 building list 接口的权限命名空间割裂；写操作仍受严格权限保护。
		buildings.Use(middleware.RequirePermissionsWithQuery([]string{
			"ops:building:list",
			"ops:building:add",
			"ops:building:edit",
			"ops:building:delete",
		}, []string{"ops:building:spaces:list"}, core))
		{
			buildingService := opsServices.NewBuildingService(core.DB.GetDB())
			geocodingService := opsServices.NewGeocodingService(core.Config.Baidu.MapAK)
			buildingHandler := operations.NewBuildingHandler(buildingService, geocodingService).WithCore(core)

			buildings.POST("", buildingHandler.Create)
			buildings.POST("/list", buildingHandler.List)
			buildings.POST("/statistics", buildingHandler.Statistics)
			buildings.POST("/dropdown-options", buildingHandler.SearchBuildingOptions)
			buildings.POST("/:id", buildingHandler.GetByID)
			buildings.POST("/:id/update", buildingHandler.Update)
			buildings.POST("/:id/delete", buildingHandler.Delete)
			buildings.POST("/batch", buildingHandler.BatchOperation)
			buildings.POST("/geocode", buildingHandler.Geocode)

			// Excel导入导出
			operations.SetupExcelRouter(buildings, "building", core)
		}

		// 楼层管理
		floors := ops.Group("/floor")
		// 查询接口(/list,/tree)额外接受「楼宇空间」可视化读权限 ops:building:spaces:list。
		floors.Use(middleware.RequirePermissionsWithQuery([]string{
			"ops:floor:list",
			"ops:floor:add",
			"ops:floor:edit",
			"ops:floor:delete",
		}, []string{"ops:building:spaces:list"}, core))
		{
			// 创建带缓存的楼层服务
			var floorService opsServices.FloorService
			if core.DataCacheService != nil {
				// 创建缓存提供者（NewCacheProvider 已处理 nil 情况）
				cacheProvider := systemServices.NewCacheProvider(core.DataCacheService)
				floorService = opsServices.NewFloorServiceWithCache(
					core.DB.GetDB(),
					cacheProvider,
					core.CacheConfigService,
				)
			} else {
				floorService = opsServices.NewFloorService(core.DB.GetDB())
			}
			floorHandler := operations.NewFloorHandler(floorService).WithCore(core)

			// 注意：更具体的路由要放在前面，避免被通用路由匹配
			floors.POST("/list", floorHandler.List)
			floors.POST("/statistics", floorHandler.Statistics)
			floors.POST("/tree", floorHandler.GetTree)
			floors.POST("/dropdown-options", floorHandler.SearchFloorOptions)
			floors.POST("/batch", floorHandler.BatchOperation)
			floors.POST("/:id", floorHandler.GetByID)
			floors.POST("/:id/update", floorHandler.Update)
			floors.POST("/:id/delete", floorHandler.Delete)
			floors.POST("", floorHandler.Create) // 放在最后，作为兜底路由

			// Excel导入导出
			operations.SetupExcelRouter(floors, "floor", core)
		}

		// 工位管理
		workstations := ops.Group("/workstation")
		// 查询接口(/list)额外接受「楼宇空间」可视化读权限 ops:building:spaces:list。
		workstations.Use(middleware.RequirePermissionsWithQuery([]string{
			"ops:workstation:list",
			"ops:workstation:add",
			"ops:workstation:edit",
			"ops:workstation:delete",
		}, []string{"ops:building:spaces:list"}, core))
		{
			workstationService := opsServices.NewWorkstationService(core.DB.GetDB())
			// R4 (Phase 45) — 跨模块注入 ReconciliationService 到 WorkstationHandler
			// GetByID 内根据 hasReconciliationPerm 决定是否拉对账健康度 (D-A1-01/03)
			// Phase 45 R4: 注入 exceptionSvc 用于 per-asset 命中 (D-A4-02)
			exceptionSvcForWs := asset.NewReconciliationExceptionService(core.DB.GetDB())
			reconciliationSvc := asset.NewReconciliationService(core.DB.GetDB(), core.Cache, exceptionSvcForWs)
			workstationHandler := operations.NewWorkstationHandler(workstationService).
				WithCore(core).
				WithReconciliationService(reconciliationSvc)

			// 注意：更具体的路由要放在前面，避免被通用路由匹配
			workstations.POST("/list", workstationHandler.List)
			workstations.POST("/dept-options", workstationHandler.GetWorkstationDeptOptions)
			workstations.POST("/statistics", workstationHandler.Statistics)
			workstations.POST("/dropdown-options", workstationHandler.SearchWorkstationOptions)
			workstations.POST("/batch", workstationHandler.BatchOperation)
			workstations.POST("/positions", workstationHandler.BatchUpdatePositions)
			workstations.POST("/:id", workstationHandler.GetByID)
			workstations.POST("/:id/update", workstationHandler.Update)
			workstations.POST("/:id/delete", workstationHandler.Delete)
			workstations.POST("", workstationHandler.Create) // 放在最后

			// Excel导入导出
			operations.SetupExcelRouter(workstations, "workstation", core)
			// 部门名称↔代码映射表 (quick 260713-df0, 工位导入辅助)
			workstations.GET("/dept-mapping-template", operations.DownloadDeptMappingTemplate(core))
		}

		// 工位部门物理位置映射 (Phase 39)
		locationAlias := ops.Group("/location-alias")
		locationAlias.Use(middleware.RequirePermissions([]string{
			"ops:location:alias:list",
			"ops:location:alias:add",
			"ops:location:alias:edit",
			"ops:location:alias:delete",
		}, core))
		{
			locationAliasService := opsServices.NewLocationAliasService(core.DB.GetDB())

			// 构造 DepartmentService 用于 alias 写操作后触发 dept 缓存失效 (D-03 决策)。
			// core 不直接持有 DepartmentService, 在路由组现场构造 — 与 department_router.go 一致。
			aliasCacheProvider := systemServices.NewCacheProvider(core.DataCacheService)
			var aliasDeptSvc systemServices.DepartmentService
			if core.DataCacheService != nil {
				aliasDeptSvc = systemServices.NewDepartmentServiceWithCache(
					core.DB.GetDB(), aliasCacheProvider, core.CacheConfigService,
				)
			} else {
				aliasDeptSvc = systemServices.NewDepartmentService(core.DB.GetDB())
			}

			locationAliasHandler := operations.NewLocationAliasHandler(locationAliasService).
				WithCore(core).
				WithDeptCacheInvalidator(aliasDeptSvc)

			// 注意:更具体的路由放在前面,避免被通用路由匹配
			locationAlias.POST("/list", locationAliasHandler.List)
			locationAlias.POST("/:id/update", locationAliasHandler.Update)
			locationAlias.POST("/:id/delete", locationAliasHandler.Delete)
			locationAlias.POST("", locationAliasHandler.Create) // 放在最后,作为兜底路由
		}

		// 工位设备关联
		workstationDevices := ops.Group("/workstation-device")
		workstationDevices.Use(middleware.RequirePermissions([]string{
			"ops:workstation:list",
			"ops:workstation:add",
			"ops:workstation:edit",
		}, core))
		{
			workstationDeviceService := opsServices.NewWorkstationDeviceService(core.DB.GetDB())
			workstationDeviceHandler := operations.NewWorkstationDeviceHandler(workstationDeviceService).WithCore(core)

			workstationDevices.POST("/:id", workstationDeviceHandler.GetByWorkstation)
			workstationDevices.POST("/:id/ad", workstationDeviceHandler.GetADDevices)
			workstationDevices.POST("/:id/asset", workstationDeviceHandler.GetAssetDevices)
			// Phase 45 R5: 物理链路设备(MAC→port→infoPoint→workstation 反推)
			workstationDevices.POST("/:id/physical", workstationDeviceHandler.GetPhysicalDevices)
			workstationDevices.POST("/:id/set-primary-and-save", workstationDeviceHandler.SetPrimaryAndSave)
			workstationDevices.POST("/manual", workstationDeviceHandler.AddManual)
			workstationDevices.POST("/sync-ad", workstationDeviceHandler.SyncAD)
			workstationDevices.POST("/sync-asset", workstationDeviceHandler.SyncAsset)
			workstationDevices.POST("/:id/update", workstationDeviceHandler.Update)
			workstationDevices.POST("/:id/delete", workstationDeviceHandler.Delete)
			workstationDevices.POST("/:id/set-primary", workstationDeviceHandler.SetPrimary)
		}

		// 资产管理
		assets := ops.Group("/asset")
		assets.Use(middleware.RequirePermissions([]string{
			"ops:asset:list",
			"ops:asset:add",
			"ops:asset:edit",
			"ops:asset:delete",
		}, core))
		{
			assetService := opsServices.NewAssetService(core.DB.GetDB())
			assetHandler := operations.NewAssetHandler(assetService).WithCore(core)

			// 注意：更具体的路由要放在前面
			assets.GET("/search-by-serial/:serial", assetHandler.SearchBySerial)

			// Phase 48 Wave 3 (D-07): 从属组件清单 read-only 端点。
			// 复用 ops:asset:list 组级 middleware(MEMORY xingran-perm-namespace-split-readonly-page)。
			// 放在 /:id 通配之前避免被吞噬。
			assetComponentHandler := operations.NewAssetComponentHandler(core)
			assets.GET("/components", assetComponentHandler.ListComponents)

			assets.POST("", assetHandler.Create)
			assets.POST("/list", assetHandler.List)
			// 资产统计(专用 COUNT 聚合,不依赖分页列表)
			assets.POST("/statistics", assetHandler.Statistics)
			assets.POST("/:id", assetHandler.GetByID)
				assets.POST("/device-types", assetHandler.GetDeviceTypes)
					assets.POST("/device-categories", assetHandler.GetDeviceCategories)
					assets.POST("/status-values", assetHandler.GetStatusValues)
			assets.POST("/:id/update", assetHandler.Update)
			assets.POST("/:id/delete", assetHandler.Delete)
				assets.POST("/batch", assetHandler.BatchOperation)

				// Excel导入导出
				operations.SetupExcelRouter(assets, "asset", core)
			}

		// 信息点管理
		infoPoints := ops.Group("/infoPoint")
		infoPoints.Use(middleware.RequirePermissions([]string{
			"ops:infopoint:list",
			"ops:infopoint:add",
			"ops:infopoint:edit",
			"ops:infopoint:delete",
		}, core))
		{
			infoPointService := opsServices.NewInfoPointService(core.DB.GetDB())
			infoPointHandler := operations.NewInfoPointHandler(infoPointService).WithCore(core)

			// 注意：更具体的路由要放在前面
			infoPoints.POST("/list", infoPointHandler.List)
			infoPoints.POST("/statistics", infoPointHandler.Statistics)
			infoPoints.POST("/dropdown-options", infoPointHandler.SearchInfoPointOptions)
			infoPoints.POST("/batch", infoPointHandler.BatchOperation)
			infoPoints.POST("/:id", infoPointHandler.GetByID)
			infoPoints.POST("/:id/update", infoPointHandler.Update)
			infoPoints.POST("/:id/delete", infoPointHandler.Delete)
			infoPoints.POST("", infoPointHandler.Create) // 放在最后

			// Excel导入导出
			operations.SetupExcelRouter(infoPoints, "infoPoint", core)
		}

		// 机房管理
		serverRooms := ops.Group("/serverRoom")
		serverRooms.Use(middleware.RequirePermissions([]string{
			"ops:serverroom:list",
			"ops:serverroom:add",
			"ops:serverroom:edit",
			"ops:serverroom:delete",
		}, core))
		{
			serverRoomService := opsServices.NewServerRoomService(core.DB.GetDB())
			serverRoomHandler := operations.NewServerRoomHandler(serverRoomService).WithCore(core)

			// 注意：更具体的路由要放在前面
			serverRooms.POST("/list", serverRoomHandler.List)
			serverRooms.POST("/statistics", serverRoomHandler.Statistics)
			serverRooms.POST("/dropdown-options", serverRoomHandler.SearchServerRoomOptions)
			serverRooms.POST("/batch", serverRoomHandler.BatchOperation)
			serverRooms.POST("/:id", serverRoomHandler.GetByID)
			serverRooms.POST("/:id/update", serverRoomHandler.Update)
			serverRooms.POST("/:id/delete", serverRoomHandler.Delete)
			serverRooms.POST("", serverRoomHandler.Create) // 放在最后

			// Excel导入导出
			operations.SetupExcelRouter(serverRooms, "serverRoom", core)
		}

		// 机房照片管理
		roomPhotos := ops.Group("/rooms/photos")
		{
			operations.SetupRoomPhotoRouter(roomPhotos, core)
		}

		// 专线管理
		dedicatedLines := ops.Group("/dedicatedLine")
		// perms 对齐: 专线模块统一用 ops:dedicatedline:* (单数无连字符),
		// 与菜单 seed / sys_menu 一致, 避免命名空间割裂导致 403。
		dedicatedLines.Use(middleware.RequirePermissions([]string{
			"ops:dedicatedline:list",
			"ops:dedicatedline:add",
			"ops:dedicatedline:edit",
			"ops:dedicatedline:delete",
		}, core))
		{
			dedicatedLineService := opsServices.NewDedicatedLineService(core.DB.GetDB())
			dedicatedLineHandler := operations.NewDedicatedLineHandler(dedicatedLineService).WithCore(core)

			// 注意：更具体的路由要放在前面
			dedicatedLines.POST("/list", dedicatedLineHandler.List)
			dedicatedLines.POST("/statistics", dedicatedLineHandler.Statistics)
			dedicatedLines.POST("/dropdown-options", dedicatedLineHandler.SearchDedicatedLineOptions)
			dedicatedLines.POST("/batch", dedicatedLineHandler.BatchOperation)
			dedicatedLines.POST("/:id", dedicatedLineHandler.GetByID)
			dedicatedLines.POST("/:id/update", dedicatedLineHandler.Update)
			dedicatedLines.POST("/:id/delete", dedicatedLineHandler.Delete)
			dedicatedLines.POST("", dedicatedLineHandler.Create) // 放在最后

			// Excel导入导出
			operations.SetupExcelRouter(dedicatedLines, "dedicatedLine", core)
		}

		// room-devices - 机房设备管理
		roomDevices := ops.Group("/roomDevice")
		roomDevices.Use(middleware.RequirePermissions([]string{
			"ops:roomdevice:list",
			"ops:roomdevice:add",
			"ops:roomdevice:edit",
			"ops:roomdevice:delete",
		}, core))
		{
			roomDeviceService := opsServices.NewRoomDeviceService(core.DB.GetDB())
			roomDeviceHandler := operations.NewRoomDeviceHandler(roomDeviceService).WithCore(core)

			// 注意：更具体的路由要放在前面
			roomDevices.POST("/list", roomDeviceHandler.List)
			roomDevices.POST("/statistics", roomDeviceHandler.Statistics)
			roomDevices.POST("/dropdown-options", roomDeviceHandler.SearchRoomDeviceOptions)
			roomDevices.POST("/batch", roomDeviceHandler.BatchOperation)
			roomDevices.POST("/:id", roomDeviceHandler.GetByID)
			roomDevices.POST("/:id/update", roomDeviceHandler.Update)
			roomDevices.POST("/:id/delete", roomDeviceHandler.Delete)
			roomDevices.POST("", roomDeviceHandler.Create) // 放在最后

			// Excel导入导出
			operations.SetupExcelRouter(roomDevices, "roomDevice", core)
		}

		// CAD平面图编辑器（包含墙体、门等元素）
		// 使用平面图编辑器和墙体、门的权限标识符
		walls := ops.Group("/walls")
		walls.Use(middleware.RequirePermissions([]string{
			"ops:floor-plan:list",
			"ops:floor-plan:query",
			"ops:floor-plan:edit",
			"ops:floor-plan:save",
			"ops:walls:list",
			"ops:walls:add",
			"ops:walls:edit",
			"ops:walls:delete",
		}, core))
		{
			wallService := opsServices.NewWallService(core.DB.GetDB())
			wallHandler := operations.NewWallHandler(wallService).WithCore(core)

			walls.POST("/list", wallHandler.List)
			walls.POST("/batch", wallHandler.BatchOperation)
			walls.POST("/:id", wallHandler.GetByID)
			walls.POST("/:id/update", wallHandler.Update)
			walls.POST("/:id/delete", wallHandler.Delete)
			walls.POST("", wallHandler.Create)
		}

		// 门管理（CAD平面图）- 使用平面图编辑器和门权限
		doors := ops.Group("/doors")
		doors.Use(middleware.RequirePermissions([]string{
			"ops:floor-plan:list",
			"ops:floor-plan:query",
			"ops:floor-plan:edit",
			"ops:floor-plan:save",
			"ops:doors:list",
			"ops:doors:add",
			"ops:doors:edit",
			"ops:doors:delete",
		}, core))
		{
			doorService := opsServices.NewDoorService(core.DB.GetDB())
			doorHandler := operations.NewDoorHandler(doorService).WithCore(core)

			doors.POST("/list", doorHandler.List)
			doors.POST("/batch", doorHandler.BatchOperation)
			doors.POST("/:id", doorHandler.GetByID)
			doors.POST("/:id/update", doorHandler.Update)
			doors.POST("/:id/delete", doorHandler.Delete)
			doors.POST("", doorHandler.Create)
		}

		// 平面图文本管理（CAD平面图）
		texts := ops.Group("/floor-plan-texts")
		texts.Use(middleware.RequirePermissions([]string{
			"ops:floor-plan:list",
			"ops:floor-plan:query",
			"ops:floor-plan:edit",
			"ops:floor-plan:save",
		}, core))
		{
			textService := opsServices.NewFloorPlanTextService(core.DB.GetDB())
			textHandler := operations.NewFloorPlanTextHandler(textService).WithCore(core)

			texts.POST("/list", textHandler.List)
			texts.POST("/batch", textHandler.BatchOperation)
			texts.POST("/:id", textHandler.GetByID)
			texts.POST("/:id/update", textHandler.Update)
			texts.POST("/:id/delete", textHandler.Delete)
			texts.POST("", textHandler.Create)
		}
	}

			// VDI管理模块（虚拟桌面基础设施）
			vdi := r.Group("/vdi")
			vdi.Use(middleware.JWTAuth(core.JWTManager))
			// 添加操作日志中间件
			vdi.Use(middleware.OperLogMiddleware(core.OperLogService, core))
			{
				// 虚拟机管理
				vms := vdi.Group("/vms")
				vms.Use(middleware.DataScopePermission(core))
				vms.Use(middleware.RequirePermissions([]string{
					"vdi:vm:list",
					"vdi:vm:add",
					"vdi:vm:edit",
				}, core))
				{
					vdiV1.SetupVMRouter(vms, core)
				}

			// VDI服务器管理
			servers := vdi.Group("/servers")
			servers.Use(middleware.RequirePermissions([]string{
				"vdi:server:list",
				"vdi:server:add",
				"vdi:server:edit",
				"vdi:server:delete",
			}, core))
			{
				vdiV1.SetupVDIServerRouter(servers, core)
			}
		}

	// RPA 管理模块
	// 先设置公开的 worker 注册接口（不需要认证）
	rpaPublic := r.Group("/rpa")
	{
		rpaV1.SetupPublicWorkerRouter(rpaPublic, core)
	}

	// Agent 模块（无需认证，供 Agent 自动注册使用）
	agentV1.SetupAgentRouter(r, core)

	// 需要 JWT 认证的 RPA 路由
	rpa := r.Group("/rpa")
	rpa.Use(middleware.JWTAuth(core.JWTManager))
	// 添加操作日志中间件
	rpa.Use(middleware.OperLogMiddleware(core.OperLogService, core))
	{
		// 使用统一的 RPA 路由设置（排除公开的 worker 接口）
		rpaV1.SetupRPARouter(rpa, core)
	}

	// 资产对账模块 (Phase 42 R1)
	// 新增顶层 /asset/reconciliation 路由组,与 /ops/asset/* 分离避免冲突。
	// 中间件链:JWTAuth → OperLogMiddleware(D-17 写操作自动记)→ RequirePermissions(读端点同样受权限保护)。
	// 不在 router.go 预注册通用 /asset/reconciliation/* 路径 — 由 Setup*Router 函数内部注册,
	// 避免与 Excel 导入路由冲突陷阱(MEMORY: xingran-excel-import-route-conflict)。
	assetReconciliation := r.Group("/asset/reconciliation")
	assetReconciliation.Use(middleware.JWTAuth(core.JWTManager))
	// 添加操作日志中间件(D-17:全部写操作都走 operlog)。LogPaths 必须在 pkg/middleware/oper_log.go 追加 /asset/reconciliation 才会触发。
	assetReconciliation.Use(middleware.OperLogMiddleware(core.OperLogService, core))
	assetReconciliation.Use(middleware.RequirePermissions([]string{
		"asset:reconciliation:list",
		"asset:reconciliation:dashboard",
		"asset:reconciliation:export",
		// Phase 46 R5: 修复建议命名空间 5 个 perm(list/accept/reject/apply/rollback/stats)
		"asset:reconciliation:fix:list",
		"asset:reconciliation:fix:accept",
		"asset:reconciliation:fix:reject",
		"asset:reconciliation:fix:apply",
		"asset:reconciliation:fix:rollback",
		"asset:reconciliation:fix:stats",
	}, core))
	{
		// 异常列表读端点 (SetupReconciliationRouter: /exception/list, /exception/:id)
		assetV1.SetupReconciliationRouter(assetReconciliation, core)
		// 例外规则读端点 (SetupReconciliationExceptionRouter: /exception-rule/list, /exception-rule/:id)
		assetV1.SetupReconciliationExceptionRouter(assetReconciliation, core)
		// 6 个统计端点 (SetupReconciliationStatisticsRouter: /statistics/summary 等)
		assetV1.SetupReconciliationStatisticsRouter(assetReconciliation, core)
		// Phase 46 R5: 修复建议(7 个端点:list/getById/accept/reject/apply/rollback/stats)
		assetV1.SetupFixSuggestionRouter(assetReconciliation, core)
	}

	// 注意：前端静态文件路由已移至 cmd/main.go，直接注册在 engine 上
}
