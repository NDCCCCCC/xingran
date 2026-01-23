package api

import (
	"log"

	"github.com/gin-gonic/gin"
	v1 "github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/operations"
	systemV1 "github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/system"
	workorderV1 "github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/workorder"
	dutyV1 "github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/duty"
	networkV1 "github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/network"
	monitorV1 "github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/monitor"
	v1_knowledge "github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/knowledge"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/scheduler"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/websocket"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/crypto"
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

	// 创建请求加密器（使用 SM2+SM4 混合加密）
	// SM4-CBC 加密请求体，SM2 加密 SM4 密钥
	var encryptor *crypto.RequestEncryptor
	if core.Config.Security.RequestEncryption.Enabled {
		sm2PrivateKey, sm2PublicKey := core.JWTManager.GetSM2KeyPair()
		if sm2PrivateKey != nil && sm2PublicKey != nil {
			encryptor = crypto.NewRequestEncryptor(sm2PrivateKey, sm2PublicKey)
			log.Println("请求体加密器已初始化（SM2+SM4 混合加密）")
		} else {
			log.Println("警告: SM2 密钥对未配置，无法启用请求体加密")
		}
	}

	// 解密配置
	decryptionConfig := &middleware.RequestDecryptionConfig{
		Enabled:           core.Config.Security.RequestEncryption.Enabled,
		ExcludePaths:      core.Config.Security.RequestEncryption.ExcludePaths,
		RequireEncryption: core.Config.Security.RequestEncryption.RequireEncryption,
	}

	// 响应加密配置
	encryptionConfig := &middleware.ResponseEncryptionConfig{
		Enabled:      core.Config.Security.RequestEncryption.Enabled,
		ExcludePaths: core.Config.Security.RequestEncryption.ExcludePaths,
	}

	// 添加请求解密中间件（在认证之前）
	if encryptor != nil && core.Config.Security.RequestEncryption.Enabled {
		r.Use(middleware.RequestDecryption(encryptor, decryptionConfig))
		log.Println("请求体解密中间件已启用")
	}

	// 添加响应加密中间件（在所有路由之后执行）
	if encryptor != nil && core.Config.Security.RequestEncryption.Enabled {
		r.Use(middleware.ResponseEncryption(encryptor, encryptionConfig))
		log.Println("响应体加密中间件已启用")
	}

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

			// 通知配置管理（需要系统配置权限）
			notificationConfigs := authorized.Group("/settings")
			notificationConfigs.Use(middleware.RequirePermissions([]string{
				string(permission.SystemConfig),
			}, core))
			{
				systemV1.SetupNotificationConfigRouter(notificationConfigs, core)
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
				// 新架构：结构体Handler + Service层
				systemV1.SetupUserRouter(users, core)
				// Excel导入导出
				operations.SetupExcelRouter(users, "user", core)
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
			depts.Use(middleware.RequirePermissions([]string{
				string(permission.DeptList),
				string(permission.DeptAdd),
				string(permission.DeptEdit),
				string(permission.DeptView),
			}, core))
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

			// 字典管理
			dicts := authorized.Group("/dicts")
			dicts.Use(middleware.RequirePermissions([]string{
				string(permission.DictTypeList),
				string(permission.DictTypeAdd),
				string(permission.DictTypeEdit),
				string(permission.DictTypeView),
			}, core))
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
	network.Use(middleware.RequirePermissions([]string{
		"network:device:list",
		"network:device:add",
		"network:device:edit",
		"network:device:delete",
		"network:credential:list",
		"network:credential:add",
		"network:credential:edit",
		"network:credential:delete",
		"network:template:list",
		"network:template:add",
		"network:template:edit",
		"network:template:delete",
		"network:command:execute",
		"network:command:view",
		"network:backup:list",
		"network:backup:add",
		"network:backup:restore",
		"network:backup:diff",
		"network:discovery:add",
		"network:discovery:view",
	}, core))
	{
		networkV1.SetupNetworkRouter(network, core)
	}

	// 值班管理模块（运维管理）
	duty := r.Group("/duty")
	duty.Use(middleware.JWTAuth(core.JWTManager))
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
		dutyConfig.Use(middleware.RequirePermissions([]string{
			string(permission.SystemConfig),
		}, core))
		{
			dutyV1.SetupDutyConfigRouter(dutyConfig, core)
		}

		// 我的值班（所有已登录用户可访问）
		dutyV1.SetupMyDutyRouter(duty.Group("/my-duty"), core)
	}

	// 运维工单管理模块
	workorder := r.Group("/workorder")
	workorder.Use(middleware.JWTAuth(core.JWTManager))
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
		workorderConfig.Use(middleware.RequirePermissions([]string{
			string(permission.SystemConfig),
		}, core))
		{
			workorderV1.SetupWorkOrderConfigRouter(workorderConfig, core)
		}

		// 工单统计（所有已登录用户可查看）
		workorderV1.SetupWorkOrderStatisticsRouter(workorder.Group("/statistics"), core)
	}

	// 知识库管理模块
	knowledge := r.Group("/knowledge")
	knowledge.Use(middleware.JWTAuth(core.JWTManager))
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
			v1_knowledge.SetupKnowledgeRouter(articles, core)
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
			v1_knowledge.SetupKnowledgeRouter(knowledgeCategories, core)
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
			v1_knowledge.SetupKnowledgeRouter(knowledgeTags, core)
		}

		// 知识库工单转换（工单转知识库）
		v1_knowledge.SetupKnowledgeRouter(knowledge.Group("/workorders"), core)

		// 知识库搜索和查看（所有已登录用户可访问）
		v1_knowledge.SetupKnowledgeViewRouter(knowledge.Group("/view"), core)
	}

	// AD域管理模块
	adDomain := r.Group("/ad-domain")
	adDomain.Use(middleware.JWTAuth(core.JWTManager))
	{
		systemV1.SetupADDomainRouter(adDomain, core)
	}

	// 运维管理模块（物理资源管理）
	ops := r.Group("/ops")
	ops.Use(middleware.JWTAuth(core.JWTManager))
	{
		// 楼宇管理
		buildings := ops.Group("/buildings")
		buildings.Use(middleware.RequirePermissions([]string{
			"ops:building:list",
			"ops:building:add",
			"ops:building:edit",
			"ops:building:delete",
		}, core))
		{
			buildingService := opsServices.NewBuildingService(core.DB.GetDB())
			buildingHandler := operations.NewBuildingHandler(buildingService)

			buildings.POST("", buildingHandler.Create)
			buildings.POST("/list", buildingHandler.List)
			buildings.POST("/:id", buildingHandler.GetByID)
			buildings.POST("/:id/update", buildingHandler.Update)
			buildings.POST("/:id/delete", buildingHandler.Delete)
			buildings.POST("/batch", buildingHandler.BatchOperation)

			// Excel导入导出
			operations.SetupExcelRouter(buildings, "building", core)
		}

		// 楼层管理
		floors := ops.Group("/floors")
		floors.Use(middleware.RequirePermissions([]string{
			"ops:floor:list",
			"ops:floor:add",
			"ops:floor:edit",
			"ops:floor:delete",
		}, core))
		{
			floorService := opsServices.NewFloorService(core.DB.GetDB())
			floorHandler := operations.NewFloorHandler(floorService)

			// 注意：更具体的路由要放在前面，避免被通用路由匹配
			floors.POST("/list", floorHandler.List)
			floors.POST("/tree", floorHandler.GetTree)
			floors.POST("/batch", floorHandler.BatchOperation)
			floors.POST("/:id", floorHandler.GetByID)
			floors.POST("/:id/update", floorHandler.Update)
			floors.POST("/:id/delete", floorHandler.Delete)
			floors.POST("", floorHandler.Create) // 放在最后，作为兜底路由

			// Excel导入导出
			operations.SetupExcelRouter(floors, "floor", core)
		}

		// 工位管理
		workstations := ops.Group("/workstations")
		workstations.Use(middleware.RequirePermissions([]string{
			"ops:workstation:list",
			"ops:workstation:add",
			"ops:workstation:edit",
			"ops:workstation:delete",
		}, core))
		{
			workstationService := opsServices.NewWorkstationService(core.DB.GetDB())
			workstationHandler := operations.NewWorkstationHandler(workstationService)

			// 注意：更具体的路由要放在前面，避免被通用路由匹配
			workstations.POST("/list", workstationHandler.List)
			workstations.POST("/batch", workstationHandler.BatchOperation)
			workstations.POST("/positions", workstationHandler.BatchUpdatePositions)
			workstations.POST("/:id", workstationHandler.GetByID)
			workstations.POST("/:id/update", workstationHandler.Update)
			workstations.POST("/:id/delete", workstationHandler.Delete)
			workstations.POST("", workstationHandler.Create) // 放在最后

			// Excel导入导出
			operations.SetupExcelRouter(workstations, "workstation", core)
		}

		// 信息点管理
		infoPoints := ops.Group("/info-points")
		infoPoints.Use(middleware.RequirePermissions([]string{
			"ops:infopoint:list",
			"ops:infopoint:add",
			"ops:infopoint:edit",
			"ops:infopoint:delete",
		}, core))
		{
			infoPointService := opsServices.NewInfoPointService(core.DB.GetDB())
			infoPointHandler := operations.NewInfoPointHandler(infoPointService)

			// 注意：更具体的路由要放在前面
			infoPoints.POST("/list", infoPointHandler.List)
			infoPoints.POST("/batch", infoPointHandler.BatchOperation)
			infoPoints.POST("/:id", infoPointHandler.GetByID)
			infoPoints.POST("/:id/update", infoPointHandler.Update)
			infoPoints.POST("/:id/delete", infoPointHandler.Delete)
			infoPoints.POST("", infoPointHandler.Create) // 放在最后

			// Excel导入导出
			operations.SetupExcelRouter(infoPoints, "infoPoint", core)
		}

		// 机房管理
		serverRooms := ops.Group("/server-rooms")
		serverRooms.Use(middleware.RequirePermissions([]string{
			"ops:serverroom:list",
			"ops:serverroom:add",
			"ops:serverroom:edit",
			"ops:serverroom:delete",
		}, core))
		{
			serverRoomService := opsServices.NewServerRoomService(core.DB.GetDB())
			serverRoomHandler := operations.NewServerRoomHandler(serverRoomService)

			// 注意：更具体的路由要放在前面
			serverRooms.POST("/list", serverRoomHandler.List)
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
		dedicatedLines := ops.Group("/dedicated-lines")
		dedicatedLines.Use(middleware.RequirePermissions([]string{
			"ops:line:list",
			"ops:line:add",
			"ops:line:edit",
			"ops:line:delete",
		}, core))
		{
			dedicatedLineService := opsServices.NewDedicatedLineService(core.DB.GetDB())
			dedicatedLineHandler := operations.NewDedicatedLineHandler(dedicatedLineService)

			// 注意：更具体的路由要放在前面
			dedicatedLines.POST("/list", dedicatedLineHandler.List)
			dedicatedLines.POST("/batch", dedicatedLineHandler.BatchOperation)
			dedicatedLines.POST("/:id", dedicatedLineHandler.GetByID)
			dedicatedLines.POST("/:id/update", dedicatedLineHandler.Update)
			dedicatedLines.POST("/:id/delete", dedicatedLineHandler.Delete)
			dedicatedLines.POST("", dedicatedLineHandler.Create) // 放在最后

			// Excel导入导出
			operations.SetupExcelRouter(dedicatedLines, "dedicatedLine", core)
		}

		// room-devices - 机房设备管理
		roomDevices := ops.Group("/room-devices")
		roomDevices.Use(middleware.RequirePermissions([]string{
			"ops:roomdevice:list",
			"ops:roomdevice:add",
			"ops:roomdevice:edit",
			"ops:roomdevice:delete",
		}, core))
		{
			roomDeviceService := opsServices.NewRoomDeviceService(core.DB.GetDB())
			roomDeviceHandler := operations.NewRoomDeviceHandler(roomDeviceService)

			// 注意：更具体的路由要放在前面
			roomDevices.POST("/list", roomDeviceHandler.List)
			roomDevices.POST("/batch", roomDeviceHandler.BatchOperation)
			roomDevices.POST("/:id", roomDeviceHandler.GetByID)
			roomDevices.POST("/:id/update", roomDeviceHandler.Update)
			roomDevices.POST("/:id/delete", roomDeviceHandler.Delete)
			roomDevices.POST("", roomDeviceHandler.Create) // 放在最后

			// Excel导入导出
			operations.SetupExcelRouter(roomDevices, "roomDevice", core)
		}
	}
}
