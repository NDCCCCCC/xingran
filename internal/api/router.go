package api

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	v1 "github.com/xingran-next/xingran-go-backend/internal/api/v1"
	agentV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/agent"
	assetV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/asset"
	dbbackupV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/dbbackup"
	dutyV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/duty"
	v1_knowledge "github.com/xingran-next/xingran-go-backend/internal/api/v1/knowledge"
	monitorV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/monitor"
	networkV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/network"
	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations"
	rpaV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/rpa"
	systemV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/system"
	vdiV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/vdi"
	workorderV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/workorder"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	// AUTH-03 / D-01: MultiAuth + RateLimitByScope 位于 internal/middleware,
	// 与下方 pkg/middleware（别名 middleware）同 package 名，故取别名 internalmw。
	internalmw "github.com/xingran-next/xingran-go-backend/internal/middleware"
	"github.com/xingran-next/xingran-go-backend/internal/scheduler"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/internal/services/asset"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"gorm.io/gorm"
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

// BroadcastToAll Phase 144：把全员事件转调 *websocket.NoticeHub.BroadcastToAll。
// 与 BroadcastToUsers 同款类型断言（scheduler 侧传 interface{}）。
func (a *noticeHubAdapter) BroadcastToAll(message interface{}) {
	if msg, ok := message.(websocket.NoticeMessage); ok {
		a.hub.BroadcastToAll(msg)
	}
}

// Phase 144：把 scheduler 全局 adapter + 真实发布链路 / webhook 通道注入到
// addomain 漂移告警服务（零循环依赖：addomain 被 scheduler/services 反向 import，
// 只能由 api 装配层注入）。WS notifier 与 scheduler.SetNoticeHub 用同一个
// adapter 实例，语义等价 PLAN「方案 A」；NoticeService / APISenderService 走
// services 根包真实实现。
func setupDriftAlertHooks(core *core.Core, adapter *noticeHubAdapter) {
	addomain.SetDriftAlertWSNotifier(adapter)
	addomain.SetDriftAlertNoticePublisher(
		func(ctx context.Context, db *gorm.DB, n addomain.DriftAlertNotice) (string, error) {
			svc := services.NewNoticeService(db)
			notice, err := svc.CreateNoticeWithTargets(ctx, &services.CreateNoticeRequest{
				NoticeTitle:   n.Title,
				NoticeType:    n.NoticeType,
				NoticeContent: n.Content,
				Priority:      models.NoticePriority(n.Priority),
				TargetType:    models.TargetType(n.TargetType),
				IsMarkdown:    false,
			}, "SYSTEM", "SYSTEM")
			if err != nil {
				return "", err
			}
			if perr := svc.PublishNotice(ctx, notice.ID); perr != nil {
				return notice.ID, perr
			}
			// 修「铃铛收不到」：发布成功后主动推 new_notice 全员事件
			// （CreateNoticeWithTargets+PublishNotice 只落库/改状态，不推 WS；
			//  scheduler.broadcastNoticeToUsers 仅服务定时发布任务）。
			// 与 ad_sync_drift_alert 双事件并存，前端 noticeStore 按 type 分流。
			func() {
				defer func() { _ = recover() }()
				adapter.BroadcastToAll(websocket.NoticeMessage{
					Type:      "new_notice",
					NoticeID:  notice.ID,
					Title:     n.Title,
					Priority:  n.Priority,
					Timestamp: time.Now().Unix(),
				})
			}()
			return notice.ID, nil
		})
	addomain.SetDriftAlertWebhookSender(
		func(ctx context.Context, db *gorm.DB, configID string, msg addomain.DriftAlertWebhookMessage) addomain.DriftAlertWebhookResult {
			sender := services.NewAPISenderService(db)
			apiMsg := &services.APIMessage{Title: msg.Title, Content: msg.Content, Data: msg.Data}
			var r *services.APISendResult
			if configID != "" {
				r = sender.Send(ctx, configID, apiMsg)
			} else {
				r = sender.SendWithDefaultConfig(ctx, models.APIConfigTypeWebhook, apiMsg)
			}
			if r == nil {
				return addomain.DriftAlertWebhookResult{Success: false, Message: "webhook 发送返回空结果"}
			}
			return addomain.DriftAlertWebhookResult{
				Success:    r.Success,
				HTTPCode:   r.HTTPCode,
				RetryCount: r.RetryCount,
				Message:    r.Message,
			}
		})
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
	adapter := &noticeHubAdapter{hub: noticeHub}
	scheduler.SetNoticeHub(adapter)
	// Phase 144：同一个 adapter 同时注入漂移告警 WS 通道 + 发布链路 / webhook。
	setupDriftAlertHooks(core, adapter)

	return noticeHub
}

// strictUserRouteGroup 创建一个「单一权限点 + 严格权限」的写操作路由子组。
//
// v1.24 Phase 78（#16）：严格权限 = 只做 sys_menu.perms 精确匹配，
// 跳过 F 型子权限继承（Step 2）与模块级降级（Step 3）——
// 只持页面本体权限（如 system:user）的只读角色不会被放行。
//
// 父组必须**不含**任何 ANY-of 权限中间件（gin 的 Group 会继承父组中间件链），
// 否则父组的 ANY-of 仍会先于严格权限放行写操作。
func strictUserRouteGroup(parent *gin.RouterGroup, perm permission.PermissionCode, core *core.Core) *gin.RouterGroup {
	g := parent.Group("")
	g.Use(middleware.RequireStrictPermissions([]string{string(perm)}, core))
	return g
}

// strictLiteralRouteGroup 创建「字面量权限码 + 严格权限」的写操作路由子组。
// 与 strictUserRouteGroup 等价，用于权限码以路由字面量挂载（无 config.go
// 常量）的模块（task-3 G1/G2：duty/workorder/knowledge/reconciliation）。
// 父组同样必须不含任何 ANY-of 权限中间件（Group 继承父组中间件链）。
func strictLiteralRouteGroup(parent *gin.RouterGroup, perm string, core *core.Core) *gin.RouterGroup {
	g := parent.Group("")
	g.Use(middleware.RequireStrictPermissions([]string{perm}, core))
	return g
}

// SetupRouter 设置路由
func SetupRouter(r *gin.RouterGroup, core *core.Core, allowedOrigins []string) {
	r.Use(middleware.RequestID())

	// 整库恢复期写入阻断（A8 / #87）：必须挂在最靠近入口的全局位置，
	// 早于 JWT / 权限 / 请求解密 / handler。dbbackup.IsRestoring() 为 true 时
	// 拦截所有写请求并直接返回 503，避免恢复动作覆盖并发写入；标志复位后
	// 逐请求动态读取、自动放行。详见 internal/api/restore_write_guard.go。
	r.Use(RestoreWriteGuard())

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

			// 默认主题配置（task-7：前端 default-theme.tsx 依赖的 3 条端点）。
			// 管理操作，权限与相邻 notificationConfigs 一致（system:config，不新增码）。
			themeConfig := authorized.Group("/settings/config/theme")
			themeConfig.Use(middleware.RequirePermissions([]string{permSystemConfig}, core))
			{
				systemV1.SetupDefaultThemeRouter(themeConfig, core)
			}

			// 通知配置管理（需要系统配置权限）
			notificationConfigs := authorized.Group("/settings/notification")
			notificationConfigs.Use(middleware.RequirePermissions([]string{permSystemConfig}, core))
			{
				systemV1.SetupNotificationConfigRouter(notificationConfigs, core)
			}

			// Phase 144 同步漂移告警（5 个 POST 端点，读/写权限分组在 setup 内）
			{
				systemV1.SetupAdDriftAlertRouter(authorized, core)
			}

			// 用户管理
			//
			// v1.24 Phase 78（#16）：读组 / 写组**兄弟分离**。
			//   ① 组级 ANY-of 病灶：写操作必须脱离 [list/add/edit/view] 的 ANY-of 组 ——
			//      否则只持 system:user:list 的用户调 POST /system/users/{id}/delete
			//      会命中 UserList 分支被放行（真删除）。
			//   ② 模块级降级病灶：写操作改用 RequireStrictPermissions（跳过 Step 2 F 型
			//      继承与 Step 3 模块降级）——否则只读角色会因持有页面本体 system:user 被放行。
			//
			// ⚠️ gin 的 Group() 会**继承父组中间件链**，因此写组必须是 authorized 的
			//    **兄弟组**（而不是读组的子组）；若写成 usersRead.Group("")，读组的 ANY-of
			//    依旧会作用在写路由上，病灶① 不会被修掉。
			usersRead := authorized.Group("/users")
			// 查询接口(/list)额外接受工位/空间读权限: 工位页面用户选择器(useWorkstationData)需要
			// 读取用户列表做归属选择, 但运维角色通常不持有 system:user 权限。用户列表受
			// DataScopePermission 限制可见数据; 写操作走独立严格子组(见下)。
			usersRead.Use(middleware.RequirePermissionsWithQuery([]string{
				string(permission.UserList),
				string(permission.UserView),
			}, []string{"ops:workstation:list", "ops:building:spaces:list"}, core))
			// 添加数据权限中间件
			usersRead.Use(middleware.DataScopePermission(core))
			{
				// 新架构：结构体Handler + Service层（读路由）
				systemV1.SetupUserRouter(usersRead, core)
				// 用户 Excel 导入由 SetupUserWriteRouter 内的 ImportUser 处理（带 AD 域控同步触发），
				// 不复用 operations.SetupExcelRouter —— 后者的 importData 为通用导入，不触发
				// AD 同步。模板下载同样在 user_router（GET /import/template）。
				// 用户导出（FIX-03）由下方 usersWrite 的 Export 严格子组接线，导出字段与
				// 导入模板同源（excel_config.go "user" 配置）。
			}

			// 用户写操作组：逐动作**单一权限点** + 严格权限（不做任何降级）。
			// 逐动作映射（v1.24 #16 定稿，与 pkg/permission/config.go 常量一一对应）：
			//   新增            POST /system/users                 → system:user:add
			//   导入            POST /system/users/import          → system:user:import
			//   导入模板        GET  /system/users/import/template → system:user:import
			//   编辑            POST /system/users/:id/update      → system:user:edit
			//   状态变更        POST /system/users/:id/status      → system:user:edit
			//   同步经理        POST /system/users/sync-managers   → system:user:edit
			//   删除            POST /system/users/:id/delete      → system:user:remove
			//   批量删除        POST /system/users/batch-delete    → system:user:remove
			//   重置密码        POST /system/users/:id/reset-password → system:user:resetPwd
			//   导出            POST /system/users/export          → system:user:export（FIX-03）
			// 导出为业务写操作（审计），与其余写操作一样走严格单一权限点；读组 ANY-of
			// 不覆盖它（读/写兄弟分离），持 system:user:export 的角色方可导出。
			usersWrite := authorized.Group("/users")
			// 数据权限中间件沿用原行为（读组原本覆盖全部用户路由）
			usersWrite.Use(middleware.DataScopePermission(core))
			{
				systemV1.SetupUserWriteRouter(systemV1.UserWriteRouteGroups{
					Add:      strictUserRouteGroup(usersWrite, permission.UserAdd, core),
					Import:   strictUserRouteGroup(usersWrite, permission.UserImport, core),
					Edit:     strictUserRouteGroup(usersWrite, permission.UserEdit, core),
					Remove:   strictUserRouteGroup(usersWrite, permission.UserRemove, core),
					ResetPwd: strictUserRouteGroup(usersWrite, permission.UserResetPwd, core),
					Export:   strictUserRouteGroup(usersWrite, permission.UserExport, core),
				}, core)
			}

			// 用户解锁路由（管理员权限）
			userUnlock := authorized.Group("/user")
			userUnlock.Use(middleware.RequirePermissions([]string{permSystemConfig}, core))
			{
				systemV1.SetupUserUnlockRouter(userUnlock, core)
			}

			// 角色管理
			//
			// 85-02 RBAC-04：读组 / 写组**兄弟分离**（范式对齐 users，Phase 78 #16）。
			// gin 的 Group() 会继承父组中间件链，写组必须是 authorized 的兄弟组——
			// 若写成 rolesRead.Group("")，读组 ANY-of（含 list！）仍会先放行写操作。
			// 写组逐动作 RequireStrictPermissions 单一权限点（窗口期声明见 85-02-PLAN：
			// 87 播种前无 F 型码角色写操作 fail-closed = 预期行为）。
			rolesRead := authorized.Group("/roles")
			rolesRead.Use(middleware.RequirePermissions([]string{
				string(permission.RoleList),
				string(permission.RoleView),
			}, core))
			{
				// 新架构：结构体Handler + Service层（读路由）
				systemV1.SetupRoleRouter(rolesRead, core)
			}
			rolesWrite := authorized.Group("/roles")
			{
				systemV1.SetupRoleWriteRouter(systemV1.RoleWriteRouteGroups{
					Add:    strictUserRouteGroup(rolesWrite, permission.RoleAdd, core),
					Edit:   strictUserRouteGroup(rolesWrite, permission.RoleEdit, core),
					Remove: strictUserRouteGroup(rolesWrite, permission.RoleRemove, core),
				}, core)
			}

			// 用户菜单和权限接口（所有已登录用户可访问，不需要特定权限）
			systemV1.SetupUserMenuRouter(authorized.Group("/my-menus"), core)

			// 菜单管理（需要菜单管理权限）
			//
			// 85-02 RBAC-04：读组 / 写组**兄弟分离**（同 roles，写组不得为读组子组）。
			menusRead := authorized.Group("/menus")
			menusRead.Use(middleware.RequirePermissions([]string{
				string(permission.MenuList),
				string(permission.MenuView),
			}, core))
			{
				// 新架构：结构体Handler + Service层（读路由）
				systemV1.SetupMenuRouter(menusRead, core)
			}
			menusWrite := authorized.Group("/menus")
			{
				systemV1.SetupMenuWriteRouter(systemV1.MenuWriteRouteGroups{
					Add:    strictUserRouteGroup(menusWrite, permission.MenuAdd, core),
					Edit:   strictUserRouteGroup(menusWrite, permission.MenuEdit, core),
					Remove: strictUserRouteGroup(menusWrite, permission.MenuRemove, core),
				}, core)
			}

			// 部门管理
			//
			// 85-02 RBAC-04：读组 / 写组**兄弟分离**。读组保留 queryExtra
			// （OpsSelectorReadPerms）与 DataScopePermission（运维页面部门树选择器
			// 依赖，不得删）；写组为 authorized 兄弟组 + DataScopePermission
			// （对齐 usersWrite 先例），逐动作严格单一权限点，完全脱离 queryExtra 作用面。
			deptsRead := authorized.Group("/departments")
			// 查询接口(/tree,/list,/tree-select)额外接受运维读权限(opsSelectorReadPerms):
			// 楼宇/楼层/工位/机房等运维管理页面内嵌的 <DeptTree>/<DeptSidebar> 选择器需要读取部门树,
			// 但运维角色通常不持有 system:dept 权限, 导致部门树在每个运维页面都 403。
			// 部门树 GetTree 不应用数据权限, 返回组织结构全树(低敏感); 写操作走独立严格兄弟组(见下)。
			deptsRead.Use(middleware.RequirePermissionsWithQuery([]string{
				string(permission.DeptList),
				string(permission.DeptView),
			}, middleware.OpsSelectorReadPerms, core))
			// 添加数据权限中间件
			deptsRead.Use(middleware.DataScopePermission(core))
			{
				// 新架构：结构体Handler + Service层（读路由）
				systemV1.SetupDepartmentRouter(deptsRead, core)
			}
			deptsWrite := authorized.Group("/departments")
			// 数据权限中间件沿用原行为（读组原本覆盖全部部门路由）
			deptsWrite.Use(middleware.DataScopePermission(core))
			{
				systemV1.SetupDepartmentWriteRouter(systemV1.DeptWriteRouteGroups{
					Add:    strictUserRouteGroup(deptsWrite, permission.DeptAdd, core),
					Edit:   strictUserRouteGroup(deptsWrite, permission.DeptEdit, core),
					Remove: strictUserRouteGroup(deptsWrite, permission.DeptRemove, core),
					Import: strictUserRouteGroup(deptsWrite, permission.DeptImport, core),
					Export: strictUserRouteGroup(deptsWrite, permission.DeptExport, core),
				}, core)
			}

			// 岗位管理
			//
			// 85-02 RBAC-04：读组 / 写组**兄弟分离**（同 roles，写组不得为读组子组）。
			postsRead := authorized.Group("/posts")
			postsRead.Use(middleware.RequirePermissions([]string{
				string(permission.PostList),
				string(permission.PostView),
			}, core))
			{
				// 新架构：结构体Handler + Service层（读路由）
				systemV1.SetupPostRouter(postsRead, core)
			}
			postsWrite := authorized.Group("/posts")
			{
				systemV1.SetupPostWriteRouter(systemV1.PostWriteRouteGroups{
					Add:    strictUserRouteGroup(postsWrite, permission.PostAdd, core),
					Edit:   strictUserRouteGroup(postsWrite, permission.PostEdit, core),
					Remove: strictUserRouteGroup(postsWrite, permission.PostRemove, core),
				}, core)
			}
			// 经理同步规则/机构维护（权限在 SetupManagerRuleRouter 内按动作细分）
			systemV1.SetupManagerRuleRouter(authorized, core)
			// API密钥管理
			// 85-03 RBAC-01：读组保留 ANY-of + MultiAuth/RateLimit 链；写操作走
			// 下方 apikeysWrites 严格兄弟组（写组同为管理面，保留 MultiAuth——
			// X-API-Key 客户端仍可调写端点，行为面不收紧）。
			apikeys := authorized.Group("/apikeys")
			// WR-01 修复(Phase 61 review): 删除权限词汇统一为 system:* 模块的 remove
			// 约定(与 pkg/permission/config.go APIKeyRemove 常量 + resource_action_map
			// D-04 一致)。历史值 "system:apikey:delete" 无菜单授权来源,永不命中。
			// task-3 口径记录（2026-11-11 granularity 审计）：
			//   ① 冗余说明——本组读组 ANY-of 同时含 add/edit/remove 写码，与下方
			//      apikeysWrites 严格兄弟组并存属于历史形态保留（写组 strict 已闭环，
			//      读组多出的写码只影响读端点可达性，无提权风险），不回改。
			//   ② "system:apikey:delete" 标注 @Deprecated——历史词汇，无菜单授权
			//      来源、永不命中，任何新代码不得引用（统一用 system:apikey:remove）。
			apikeys.Use(middleware.RequirePermissions([]string{
				"system:apikey:list",
				"system:apikey:add",
				"system:apikey:edit",
				"system:apikey:remove",
			}, core))
			// AUTH-03 / D-01: 启用 X-API-Key 认证链（挂载范围严格限定本管理面路由组，D-02）。
			// 顺序: RequirePermissions → MultiAuth → RateLimitByScope。
			// D-03: 无 X-API-Key 头时 MultiAuth 直接 c.Next() 跳过，由上游 JWT 中间件接管（router 层不加 fallback 分支）。
			// D-04: IP 白名单严格拒绝沿用 internal/middleware/apikey.go 既有 isIPAllowed（本处不改中间件代码）。
			// 决策记录: .planning/notes/260813-auth03-enable-decision.md
			apikeys.Use(internalmw.MultiAuth(
				systemServices.NewAPIKeyService(core.GetDB()),
				services.NewUsageLogger(core.GetDB()),
				permission.NewService(),
				core.GetDB(),
			))
			// Phase 61 / D-11/D-18: action 参数("list" → read 档) + RateLimiter 配置驱动,
			// 从 core.CacheConfigService 字段(非 getter)读 sys_config rate_limit.* 阈值。
			apikeys.Use(internalmw.RateLimitByScope(services.NewRateLimiter(core.CacheConfigService), "list"))
			{
				// 新架构：结构体Handler + Service层
				systemV1.SetupAPIKeyRouter(apikeys, core)
			}

			// API密钥写操作组（85-03 RBAC-01：逐动作 RequireStrictPermissions 单一权限点，
			// 兄弟组保留 MultiAuth/X-API-Key 认证链，见上注释）
			apikeysWrites := authorized.Group("/apikeys")
			apikeysWrites.Use(internalmw.MultiAuth(
				systemServices.NewAPIKeyService(core.GetDB()),
				services.NewUsageLogger(core.GetDB()),
				permission.NewService(),
				core.GetDB(),
			))
			{
				systemV1.SetupAPIKeyWriteRouter(systemV1.APIKeyWriteRouteGroups{
					Add:    strictUserRouteGroup(apikeysWrites, permission.APIKeyAdd, core),
					Edit:   strictUserRouteGroup(apikeysWrites, permission.APIKeyEdit, core),
					Remove: strictUserRouteGroup(apikeysWrites, permission.APIKeyRemove, core),
				}, core)
			}

			// 字典管理
			// 85-03 RBAC-01：读组保留 ANY-of（含 OpsSelectorReadPerms queryExtra），
			// 写操作见下方 dictsWrites 严格兄弟组。
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

			// 字典写操作组（85-03 RBAC-01：类型/数据子资源共用 system:dict:* 码族）
			dictsWrites := authorized.Group("/dicts")
			{
				systemV1.SetupDictWriteRouter(systemV1.DictWriteRouteGroups{
					TypesAdd:    strictUserRouteGroup(dictsWrites, permission.DictTypeAdd, core),
					TypesEdit:   strictUserRouteGroup(dictsWrites, permission.DictTypeEdit, core),
					TypesRemove: strictUserRouteGroup(dictsWrites, permission.DictTypeRemove, core),
					DataAdd:     strictUserRouteGroup(dictsWrites, permission.DictTypeAdd, core),
					DataEdit:    strictUserRouteGroup(dictsWrites, permission.DictTypeEdit, core),
					DataRemove:  strictUserRouteGroup(dictsWrites, permission.DictTypeRemove, core),
				}, core)
			}

			// 字典刷新缓存（URL 交叉校验 FIX：前端字典页「刷新缓存」按钮活跃调用
			// /system/dicts/refresh-cache，此前无后端端点）。
			// 失效字典类型/字典数据全族缓存键，与 dictTypeCacheService.invalidateCache
			// （dict_cache_impl.go invalidateCache → base.InvalidatePattern）同源键族；
			// 归 dicts 编辑码严格组（与 dicts 写组口径一致）。
			dictRefreshCache := strictUserRouteGroup(dictsWrites, permission.DictTypeEdit, core)
			dictRefreshCache.POST("/refresh-cache", func(c *gin.Context) {
				provider := systemServices.NewCacheProvider(core.DataCacheService)
				base.InvalidatePattern(c.Request.Context(), provider, []string{
					systemServices.CacheKeyDictType + "*",
					systemServices.CacheKeyDictData + "*",
					systemServices.CacheKeyDictDataByType + "*",
				}, "DICT")
				operlog.Record(c, core.OperLogService, core.GetDB(), "字典管理", operlog.OperTypeClean)
				response.Success(c, gin.H{"refreshed": true})
			})

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

			// 参数配置写操作组（85-03 RBAC-01 严格兄弟组）
			configsWrites := authorized.Group("/configs")
			{
				systemV1.SetupConfigWriteRouter(systemV1.ConfigWriteRouteGroups{
					Add:    strictUserRouteGroup(configsWrites, permission.ConfigAdd, core),
					Edit:   strictUserRouteGroup(configsWrites, permission.ConfigEdit, core),
					Remove: strictUserRouteGroup(configsWrites, permission.ConfigRemove, core),
				}, core)
			}

			// 全库备份与恢复（quick 261110-tbq）：读组 ANY-of，写组严格兄弟组。
			// 下载/上传端点已在 security.request_encryption.exclude_paths 中豁免
			// （二进制流不能被 SM2+SM4 响应加密中间件当 JSON 加密）。
			dbBackups := authorized.Group("/db-backup")
			dbBackups.Use(middleware.RequirePermissions([]string{
				string(permission.SystemBackupList),
				string(permission.SystemBackupExport),
				string(permission.SystemBackupRestore),
			}, core))
			{
				dbbackupV1.SetupDBBackupRouter(dbBackups, core)
			}
			dbBackupsWrites := authorized.Group("/db-backup")
			{
				dbbackupV1.SetupDBBackupWriteRouter(dbbackupV1.DBBackupWriteRouteGroups{
					Export:  strictUserRouteGroup(dbBackupsWrites, permission.SystemBackupExport, core),
					Restore: strictUserRouteGroup(dbBackupsWrites, permission.SystemBackupRestore, core),
					Edit:    strictUserRouteGroup(dbBackupsWrites, permission.SystemBackupExport, core),
					// G4 整改（task-3，2026-11-11）：删除不再复用 export 码——此前持
					// system:backup:export 即可删除备份文件，违反最小权限；现拆独立
					// system:backup:remove（F 型播种见 migration_246，契约见
					// docs/API响应规范.md，前端门控见 BackupListTab 删除按钮）。
					Remove: strictUserRouteGroup(dbBackupsWrites, permission.SystemBackupRemove, core),
				}, core)
			}

			// 验证码背景图管理（所有已登录用户可访问，作为系统设置的一部分）
			captchaBackgrounds := authorized.Group("/captcha-backgrounds")
			v1.SetupCaptchaBackgroundRouter(captchaBackgrounds, core)

			// 通知公告
			// 85-03 RBAC-01：读组保留 ANY-of，写操作见下方 noticesWrites 严格兄弟组。
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

			// 通知公告写操作组（85-03 RBAC-01 严格兄弟组；publish/withdraw/channels=编辑语义）
			noticesWrites := authorized.Group("/notices")
			{
				systemV1.SetupNoticeWriteRouter(systemV1.NoticeWriteRouteGroups{
					Add:    strictUserRouteGroup(noticesWrites, permission.NoticeAdd, core),
					Edit:   strictUserRouteGroup(noticesWrites, permission.NoticeEdit, core),
					Remove: strictUserRouteGroup(noticesWrites, permission.NoticeRemove, core),
				}, core)
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
	//
	// RBAC 接线（2026-11-11 monitor 域权限码全链路）：本组端点级权限在各子 setup
	// 内挂载（server/cache/oper-logs/login-logs + jobs），对齐 vdi/vm_router.go
	// 端点级中间件先例——读端点 ANY-of 读码、写端点 RequireStrictPermissions 单码。
	// 权限码定义见 pkg/permission/config.go monitor 域；F 型按钮播种见
	// migration_244（monitor RBAC F 型按钮）。
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
		// 值班池管理（task-3 G1 整改：读 ANY-of / 写 strict 兄弟分离。
		// 此前整组 4 码 ANY-of，持 ops:duty:pool:list 即可调 delete。）
		dutyPools := duty.Group("/pools")
		dutyPools.Use(middleware.RequirePermissions([]string{
			"ops:duty:pool:list",
		}, core))
		{
			dutyV1.SetupDutyPoolsRouter(dutyPools, core)
		}
		dutyPoolsWrites := duty.Group("/pools")
		{
			dutyV1.SetupDutyPoolsWriteRouter(dutyV1.DutyPoolsWriteRouteGroups{
				Add:    strictLiteralRouteGroup(dutyPoolsWrites, "ops:duty:pool:add", core),
				Edit:   strictLiteralRouteGroup(dutyPoolsWrites, "ops:duty:pool:edit", core),
				Delete: strictLiteralRouteGroup(dutyPoolsWrites, "ops:duty:pool:delete", core),
			}, core)
		}

		// 排班管理（task-3 G1 整改：读 ANY-of / 写 strict 兄弟分离）
		dutySchedules := duty.Group("/schedules")
		dutySchedules.Use(middleware.RequirePermissions([]string{
			"ops:duty:schedule:list",
		}, core))
		{
			dutyV1.SetupDutySchedulesRouter(dutySchedules, core)
		}
		dutySchedulesWrites := duty.Group("/schedules")
		{
			dutyV1.SetupDutySchedulesWriteRouter(dutyV1.DutySchedulesWriteRouteGroups{
				Edit:   strictLiteralRouteGroup(dutySchedulesWrites, "ops:duty:schedule:edit", core),
				Delete: strictLiteralRouteGroup(dutySchedulesWrites, "ops:duty:schedule:delete", core),
			}, core)
		}

		// 节假日管理（所有已登录用户可查看）
		dutyV1.SetupDutyHolidaysRouter(duty.Group("/holidays"), core)

		// 值班配置管理（所有已登录用户可查看，管理员可修改）。
		// task-3 G1 整改：读/写兄弟分离（沿用 system:config 单码，不新增码——
		// 单码组 ANY-of 与 strict 对持码者等价，拆分是为对齐全局读写分离形态）。
		dutyConfig := duty.Group("/config")
		dutyConfig.Use(middleware.RequirePermissions([]string{permSystemConfig}, core))
		{
			dutyV1.SetupDutyConfigRouter(dutyConfig, core)
		}
		dutyConfigWrites := duty.Group("/config")
		{
			dutyConfigWrite := strictLiteralRouteGroup(dutyConfigWrites, permSystemConfig, core)
			dutyV1.SetupDutyConfigWriteRouter(dutyConfigWrite, core)
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
		// 工单管理（task-3 G1 整改：读 ANY-of / 写 strict 兄弟分离。
		// 此前整组 4 码 ANY-of，持 ops:workorder:list 即可调 delete/assign。）
		orders := workorder.Group("/orders")
		orders.Use(middleware.RequirePermissions([]string{
			"ops:workorder:list",
		}, core))
		{
			workorderV1.SetupWorkOrdersRouter(orders, core)
			workorderV1.SetupWorkOrderRatingsRouter(orders, core) // 评价读路由（FIX-06）
		}
		ordersWrites := workorder.Group("/orders")
		{
			workorderV1.SetupWorkOrdersWriteRouter(workorderV1.WorkOrdersWriteRouteGroups{
				Add:    strictLiteralRouteGroup(ordersWrites, "ops:workorder:add", core),
				Edit:   strictLiteralRouteGroup(ordersWrites, "ops:workorder:edit", core),
				Delete: strictLiteralRouteGroup(ordersWrites, "ops:workorder:delete", core),
			}, core)
		}

		// 工单评价写路由（FIX-06 / Phase 136）：评价创建为业务写操作，走独立
		// 兄弟严格写组（RequireStrictPermissions 单一权限点），与上面 orders 读组
		// 的 ANY-of 解耦，保持 v1.24 Phase 78 #16 读/写兄弟分离语义零回归。
		ratingsWrite := workorder.Group("/orders")
		{
			ratingsCreate := strictUserRouteGroup(ratingsWrite, permission.WorkOrderRatingCreate, core)
			workorderV1.SetupWorkOrderRatingsWriteRouter(ratingsCreate, core)
		}

		// 工单分类管理（task-3 G1 整改：读 ANY-of / 写 strict 兄弟分离）
		categories := workorder.Group("/categories")
		categories.Use(middleware.RequirePermissions([]string{
			"ops:workorder:category:list",
		}, core))
		{
			workorderV1.SetupWorkOrderCategoriesRouter(categories, core)
		}
		categoriesWrites := workorder.Group("/categories")
		{
			workorderV1.SetupWorkOrderCategoriesWriteRouter(workorderV1.WorkOrderCategoriesWriteRouteGroups{
				Add:    strictLiteralRouteGroup(categoriesWrites, "ops:workorder:category:add", core),
				Edit:   strictLiteralRouteGroup(categoriesWrites, "ops:workorder:category:edit", core),
				Delete: strictLiteralRouteGroup(categoriesWrites, "ops:workorder:category:delete", core),
			}, core)
		}

		// 周期性工单模板管理（task-3 G1 整改：仅模板主体 CRUD 拆 strict 写组；
		// enable/disable/generate/logs 4 条 86 新路由按任务书豁免，保持继承组口径）。
		// RBAC 例外 RBAC-EXC-01：理由/风险/清偿窗口见 scripts/RBAC-EXCEPTIONS.md，
		// 不得静默保留例外；此处 ANY-of 与 #87 严格写组口径的差异以该文档为准。
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
		periodicWrites := workorder.Group("/periodic/templates")
		{
			workorderV1.SetupPeriodicWorkOrderWriteRouter(workorderV1.PeriodicWorkOrderWriteRouteGroups{
				Add:    strictLiteralRouteGroup(periodicWrites, "ops:workorder:periodic:add", core),
				Edit:   strictLiteralRouteGroup(periodicWrites, "ops:workorder:periodic:edit", core),
				Delete: strictLiteralRouteGroup(periodicWrites, "ops:workorder:periodic:delete", core),
			}, core)
		}

		// 工单配置管理（需要系统配置权限）
		// 工单配置管理（task-3 G1 整改：读/写兄弟分离，沿用 system:config 单码）
		workorderConfig := workorder.Group("/config")
		workorderConfig.Use(middleware.RequirePermissions([]string{permSystemConfig}, core))
		{
			workorderV1.SetupWorkOrderConfigRouter(workorderConfig, core)
		}
		workorderConfigWrites := workorder.Group("/config")
		{
			workorderConfigWrite := strictLiteralRouteGroup(workorderConfigWrites, permSystemConfig, core)
			workorderV1.SetupWorkOrderConfigWriteRouter(workorderConfigWrite, core)
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
		// 知识库文章（task-3 G1 整改：读 ANY-of / 写 strict 兄弟分离）
		articles := knowledge.Group("/articles")
		articles.Use(middleware.RequirePermissions([]string{
			"ops:knowledge:article:list",
		}, core))
		{
			v1_knowledge.SetupArticleRouter(articles, core)
		}
		articlesWrites := knowledge.Group("/articles")
		{
			v1_knowledge.SetupArticleWriteRouter(v1_knowledge.ArticleWriteRouteGroups{
				Add:    strictLiteralRouteGroup(articlesWrites, "ops:knowledge:article:add", core),
				Edit:   strictLiteralRouteGroup(articlesWrites, "ops:knowledge:article:edit", core),
				Delete: strictLiteralRouteGroup(articlesWrites, "ops:knowledge:article:delete", core),
			}, core)
		}

		// 知识库分类管理
		// 知识库分类（task-3 G1 整改：读 ANY-of / 写 strict 兄弟分离）
		knowledgeCategories := knowledge.Group("/categories")
		knowledgeCategories.Use(middleware.RequirePermissions([]string{
			"ops:knowledge:category:list",
		}, core))
		{
			v1_knowledge.SetupCategoryRouter(knowledgeCategories, core)
		}
		knowledgeCategoriesWrites := knowledge.Group("/categories")
		{
			v1_knowledge.SetupCategoryWriteRouter(v1_knowledge.CategoryWriteRouteGroups{
				Add:    strictLiteralRouteGroup(knowledgeCategoriesWrites, "ops:knowledge:category:add", core),
				Edit:   strictLiteralRouteGroup(knowledgeCategoriesWrites, "ops:knowledge:category:edit", core),
				Delete: strictLiteralRouteGroup(knowledgeCategoriesWrites, "ops:knowledge:category:delete", core),
			}, core)
		}

		// 知识库标签管理
		// 知识库标签（task-3 G1 整改：读 ANY-of / 写 strict 兄弟分离）
		knowledgeTags := knowledge.Group("/tags")
		knowledgeTags.Use(middleware.RequirePermissions([]string{
			"ops:knowledge:tag:list",
		}, core))
		{
			v1_knowledge.SetupTagRouter(knowledgeTags, core)
		}
		knowledgeTagsWrites := knowledge.Group("/tags")
		{
			v1_knowledge.SetupTagWriteRouter(v1_knowledge.TagWriteRouteGroups{
				Add:    strictLiteralRouteGroup(knowledgeTagsWrites, "ops:knowledge:tag:add", core),
				Edit:   strictLiteralRouteGroup(knowledgeTagsWrites, "ops:knowledge:tag:edit", core),
				Delete: strictLiteralRouteGroup(knowledgeTagsWrites, "ops:knowledge:tag:delete", core),
			}, core)
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
	//
	// task-3 词汇口径记录（2026-11-11 granularity 审计）：楼宇空间可视化读权限
	// 存在 ops:building:spaces:list 与 ops:building:spaces:query 两后缀并存
	// （router.go /list 借码处用 :list，pkg/permission/config.go 常量为 :query）。
	// 仅记录不统一——两码均未在端点鉴权主链冲突，词汇统一归
	// check-permission-codes 盲区任务范围。
	ops := r.Group("/ops")
	ops.Use(middleware.JWTAuth(core.JWTManager))
	// 添加操作日志中间件
	ops.Use(middleware.OperLogMiddleware(core.OperLogService, core))
	// Phase 151 / ARCH-01：ScopedDB 资源归属注册表装配（唯一查询起链点由 pkg/query 承载）。
	opsServices.MustSetupScopedDBRegistry()
	{
		// 楼宇管理
		// 查询接口(/list)额外接受「楼宇空间」可视化读权限 ops:building:spaces:list，
		// 解决只读可视化页面复用 building list 接口的权限命名空间割裂；
		// 写操作见下方 buildingWrites 严格兄弟组（85-03 RBAC-01）。
		buildingService := opsServices.NewBuildingService(core.DB.GetDB())
		geocodingService := opsServices.NewGeocodingService(core.Config.Baidu.MapAK)
		buildingHandler := operations.NewBuildingHandler(buildingService, geocodingService).WithCore(core)
		buildings := ops.Group("/building")
		buildings.Use(middleware.RequirePermissionsWithQuery([]string{
			"ops:building:list",
			"ops:building:add",
			"ops:building:edit",
			"ops:building:delete",
		}, []string{"ops:building:spaces:list"}, core))
		// DS-02: 权限校验之后、handler 之前注入 data_scope/user_id（含 request context）。
		buildings.Use(middleware.DataScopePermission(core))
		{
			buildings.POST("/list", buildingHandler.List)
			buildings.POST("/statistics", buildingHandler.Statistics)
			buildings.POST("/dropdown-options", buildingHandler.SearchBuildingOptions)
			buildings.POST("/:id", buildingHandler.GetByID)
			// 读语义端点（POST 但仅为地址解析查询，带缓存），留在读组
			buildings.POST("/geocode", buildingHandler.Geocode)

			// Excel 模板下载（读操作，保留在读组）
			operations.SetupExcelReadRouter(buildings, "building", core)
		}

		// 楼宇管理写路由（85-03 RBAC-01：逐动作 RequireStrictPermissions 单一权限点，
		// 与读组互为兄弟组——写组绝不做成读组子组，防 ANY-of 父中间件先放行）
		buildingWrites := ops.Group("/building")
		// DS-02: 写组为读组的兄弟组（不继承读组 ANY-of），注入 data_scope/user_id 供严格写端点取范围。
		buildingWrites.Use(middleware.DataScopePermission(core))
		{
			buildingAdd := strictUserRouteGroup(buildingWrites, permission.OpsBuildingAdd, core)
			buildingAdd.POST("", buildingHandler.Create)
			buildingEdit := strictUserRouteGroup(buildingWrites, permission.OpsBuildingEdit, core)
			buildingEdit.POST("/:id/update", buildingHandler.Update)
			buildingDeletes := strictUserRouteGroup(buildingWrites, permission.OpsBuildingDelete, core)
			buildingDeletes.POST("/:id/delete", buildingHandler.Delete)
			buildingDeletes.POST("/batch", buildingHandler.BatchOperation) // action=delete 批量删除
			buildingImport := strictUserRouteGroup(buildingWrites, permission.OpsBuildingImport, core)
			operations.SetupExcelImportRouter(buildingImport, "building", core)
			buildingExport := strictUserRouteGroup(buildingWrites, permission.OpsBuildingExport, core)
			operations.SetupExcelExportRouter(buildingExport, "building", core)
		}

		// 楼层管理
		// 查询接口(/list,/tree)额外接受「楼宇空间」可视化读权限 ops:building:spaces:list。
		// 写操作见下方 floorWrites 严格兄弟组（85-03 RBAC-01）。
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
		floors := ops.Group("/floor")
		floors.Use(middleware.RequirePermissionsWithQuery([]string{
			"ops:floor:list",
			"ops:floor:add",
			"ops:floor:edit",
			"ops:floor:delete",
		}, []string{"ops:building:spaces:list"}, core))
		// DS-02: 权限校验之后、handler 之前注入 data_scope/user_id（含 request context）。
		floors.Use(middleware.DataScopePermission(core))
		{
			// 注意：更具体的路由要放在前面，避免被通用路由匹配
			floors.POST("/list", floorHandler.List)
			floors.POST("/statistics", floorHandler.Statistics)
			floors.POST("/tree", floorHandler.GetTree)
			floors.POST("/dropdown-options", floorHandler.SearchFloorOptions)
			floors.POST("/:id", floorHandler.GetByID)

			// Excel 模板下载（读操作，保留在读组）
			operations.SetupExcelReadRouter(floors, "floor", core)
		}

		// 楼层管理写路由（85-03 RBAC-01 严格兄弟组）
		floorWrites := ops.Group("/floor")
		// DS-02: 写组为读组的兄弟组（不继承读组 ANY-of），注入 data_scope/user_id 供严格写端点取范围。
		floorWrites.Use(middleware.DataScopePermission(core))
		{
			floorAdd := strictUserRouteGroup(floorWrites, permission.OpsFloorAdd, core)
			floorAdd.POST("", floorHandler.Create) // 兜底路由
			floorEdit := strictUserRouteGroup(floorWrites, permission.OpsFloorEdit, core)
			floorEdit.POST("/:id/update", floorHandler.Update)
			floorDeletes := strictUserRouteGroup(floorWrites, permission.OpsFloorDelete, core)
			floorDeletes.POST("/:id/delete", floorHandler.Delete)
			floorDeletes.POST("/batch", floorHandler.BatchOperation)
			floorImport := strictUserRouteGroup(floorWrites, permission.OpsFloorImport, core)
			operations.SetupExcelImportRouter(floorImport, "floor", core)
			floorExport := strictUserRouteGroup(floorWrites, permission.OpsFloorExport, core)
			operations.SetupExcelExportRouter(floorExport, "floor", core)
		}

		// 工位管理
		// 查询接口(/list)额外接受「楼宇空间」可视化读权限 ops:building:spaces:list。
		// 写操作见下方 workstationWrites 严格兄弟组（85-03 RBAC-01）。
		// quick-261106-wsb: 批量反推部门执行成功后需失效工位缓存。
		// DataCacheService 为 nil 时 NewCacheProvider 返回安全空实现（既有语义）。
		workstationService := opsServices.NewWorkstationServiceWithCache(
			core.DB.GetDB(), systemServices.NewCacheProvider(core.DataCacheService))
		// R4 (Phase 45) — 跨模块注入 ReconciliationService 到 WorkstationHandler
		// GetByID 内根据 hasReconciliationPerm 决定是否拉对账健康度 (D-A1-01/03)
		// Phase 45 R4: 注入 exceptionSvc 用于 per-asset 命中 (D-A4-02)
		// Phase 103 CONV-02 (D-103-20): core.Cache → systemServices.NewCacheProvider(core.DataCacheService)
		exceptionSvcForWs := asset.NewReconciliationExceptionService(core.DB.GetDB())
		reconciliationSvc := asset.NewReconciliationService(core.DB.GetDB(), systemServices.NewCacheProvider(core.DataCacheService), exceptionSvcForWs)
		workstationHandler := operations.NewWorkstationHandler(workstationService).
			WithCore(core).
			WithReconciliationService(reconciliationSvc)
		workstations := ops.Group("/workstation")
		workstations.Use(middleware.RequirePermissionsWithQuery([]string{
			"ops:workstation:list",
			"ops:workstation:add",
			"ops:workstation:edit",
			"ops:workstation:delete",
		}, []string{"ops:building:spaces:list"}, core))
		// DS-02: 权限校验之后、handler 之前注入 data_scope/user_id（含 request context）。
		workstations.Use(middleware.DataScopePermission(core))
		{
			// 注意：更具体的路由要放在前面，避免被通用路由匹配
			workstations.POST("/list", workstationHandler.List)
			workstations.POST("/dept-options", workstationHandler.GetWorkstationDeptOptions)
			workstations.POST("/statistics", workstationHandler.Statistics)
			workstations.POST("/dropdown-options", workstationHandler.SearchWorkstationOptions)
			workstations.POST("/counts-by-floor", workstationHandler.CountsByFloor)
			workstations.POST("/:id", workstationHandler.GetByID)

			// V130R-09 D-03-6: 楼层全部工位全集端点（CAD/3D/平面图专用，无分页，
			// 替代前端以 pageSize:1000/10000 滥用 List 分页做全集下拉的反模式）
			workstations.GET("/:floorId/workstations-all", workstationHandler.GetFloorWorkstationsAll)

			// Excel 模板下载（读操作，保留在读组）
			operations.SetupExcelReadRouter(workstations, "workstation", core)
			// 部门名称↔代码映射表 (quick 260713-df0, 工位导入辅助，读)
			workstations.GET("/dept-mapping-template", operations.DownloadDeptMappingTemplate(core))
		}

		// 工位管理写路由（85-03 RBAC-01 严格兄弟组；positions 为批量更新物理位置=编辑）
		workstationWrites := ops.Group("/workstation")
		// DS-02: 写组为读组的兄弟组（不继承读组 ANY-of），注入 data_scope/user_id 供严格写端点取范围。
		workstationWrites.Use(middleware.DataScopePermission(core))
		{
			workstationAdd := strictUserRouteGroup(workstationWrites, permission.OpsWorkstationAdd, core)
			workstationAdd.POST("", workstationHandler.Create) // 兜底路由
			workstationEdits := strictUserRouteGroup(workstationWrites, permission.OpsWorkstationEdit, core)
			workstationEdits.POST("/:id/update", workstationHandler.Update)
			workstationEdits.POST("/positions", workstationHandler.BatchUpdatePositions) // 批量更新物理位置=编辑
			workstationDeletes := strictUserRouteGroup(workstationWrites, permission.OpsWorkstationDelete, core)
			workstationDeletes.POST("/:id/delete", workstationHandler.Delete)
			workstationDeletes.POST("/batch", workstationHandler.BatchOperation)
			workstationImport := strictUserRouteGroup(workstationWrites, permission.OpsWorkstationImport, core)
			operations.SetupExcelImportRouter(workstationImport, "workstation", core)
			workstationExport := strictUserRouteGroup(workstationWrites, permission.OpsWorkstationExport, core)
			operations.SetupExcelExportRouter(workstationExport, "workstation", core)
			// quick-261106-wsb：按人员反推部门 —— 独立权限项（批量全库写不与编辑共用）。
			workstationBackfill := strictUserRouteGroup(workstationWrites, permission.OpsWorkstationBackfill, core)
			workstationBackfill.POST("/backfill-dept", workstationHandler.BatchBackfillDeptFromUser)
		}

		// 工位部门物理位置映射 (Phase 39)
		// 写操作见下方 locationAliasWrites 严格兄弟组（85-03 RBAC-01）。
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

		locationAlias := ops.Group("/location-alias")
		locationAlias.Use(middleware.RequirePermissions([]string{
			"ops:location:alias:list",
			"ops:location:alias:add",
			"ops:location:alias:edit",
			"ops:location:alias:delete",
		}, core))
		// DS-02: 权限校验之后、handler 之前注入 data_scope/user_id（含 request context）。
		locationAlias.Use(middleware.DataScopePermission(core))
		{
			// 注意:更具体的路由放在前面,避免被通用路由匹配
			locationAlias.POST("/list", locationAliasHandler.List)
		}

		// 位置映射写路由（85-03 RBAC-01 严格兄弟组）
		locationAliasWrites := ops.Group("/location-alias")
		// DS-02: 写组为读组的兄弟组（不继承读组 ANY-of），注入 data_scope/user_id 供严格写端点取范围。
		locationAliasWrites.Use(middleware.DataScopePermission(core))
		{
			locationAliasAdd := strictUserRouteGroup(locationAliasWrites, permission.OpsLocationAliasAdd, core)
			locationAliasAdd.POST("", locationAliasHandler.Create) // 兜底路由
			locationAliasEdit := strictUserRouteGroup(locationAliasWrites, permission.OpsLocationAliasEdit, core)
			locationAliasEdit.POST("/:id/update", locationAliasHandler.Update)
			locationAliasDelete := strictUserRouteGroup(locationAliasWrites, permission.OpsLocationAliasDelete, core)
			locationAliasDelete.POST("/:id/delete", locationAliasHandler.Delete)
		}

		// 工位设备关联（域码借用 ops:workstation:*；写操作见下方严格兄弟组 85-03 RBAC-01）
		// task-3 口径记录（G5 借码说明，2026-11-11）：本组无独立 ops:workstation-device:* 码，
		// 有意借用 ops:workstation:* 域码（工位设备关联属工位域管理面），不新增码、不拆组。
		workstationDevices := ops.Group("/workstation-device")
		workstationDevices.Use(middleware.RequirePermissions([]string{
			"ops:workstation:list",
			"ops:workstation:add",
			"ops:workstation:edit",
		}, core))
		// DS-02: 权限校验之后、handler 之前注入 data_scope/user_id（含 request context）。
		workstationDevices.Use(middleware.DataScopePermission(core))
		{
			workstationDeviceService := opsServices.NewWorkstationDeviceService(core.DB.GetDB())
			workstationDeviceHandler := operations.NewWorkstationDeviceHandler(workstationDeviceService).WithCore(core)

			workstationDevices.POST("/:id", workstationDeviceHandler.GetByWorkstation)
			workstationDevices.POST("/:id/ad", workstationDeviceHandler.GetADDevices)
			workstationDevices.POST("/:id/asset", workstationDeviceHandler.GetAssetDevices)
			// Phase 45 R5: 物理链路设备(MAC→port→infoPoint→workstation 反推)
			workstationDevices.POST("/:id/physical", workstationDeviceHandler.GetPhysicalDevices)
			// Phase 73 / DBG-07: 物理链路 + 空因诊断(未绑定用户/信息点错位/设备或MAC缺失)。
			// 新增端点，旧 /physical 的返回结构保持不变(向后兼容)
			workstationDevices.POST("/:id/physical-diagnosis", workstationDeviceHandler.GetPhysicalDevicesWithDiagnosis)
		}

		// 工位设备关联写路由（85-03 RBAC-01 严格兄弟组；manual=新增绑定、
		// sync-*/update/set-primary=编辑、delete=删除，均挂 ops:workstation:* 域码）
		workstationDeviceService := opsServices.NewWorkstationDeviceService(core.DB.GetDB())
		workstationDeviceHandler := operations.NewWorkstationDeviceHandler(workstationDeviceService).WithCore(core)
		workstationDeviceWrites := ops.Group("/workstation-device")
		// DS-02: 写组为读组的兄弟组（不继承读组 ANY-of），注入 data_scope/user_id 供严格写端点取范围。
		workstationDeviceWrites.Use(middleware.DataScopePermission(core))
		{
			workstationDeviceAdd := strictUserRouteGroup(workstationDeviceWrites, permission.OpsWorkstationAdd, core)
			workstationDeviceAdd.POST("/manual", workstationDeviceHandler.AddManual)
			wdEdits := strictUserRouteGroup(workstationDeviceWrites, permission.OpsWorkstationEdit, core)
			wdEdits.POST("/sync-ad", workstationDeviceHandler.SyncAD)
			wdEdits.POST("/sync-asset", workstationDeviceHandler.SyncAsset)
			wdEdits.POST("/:id/update", workstationDeviceHandler.Update)
			wdEdits.POST("/:id/set-primary", workstationDeviceHandler.SetPrimary)
			wdEdits.POST("/:id/set-primary-and-save", workstationDeviceHandler.SetPrimaryAndSave)
			wdDeletes := strictUserRouteGroup(workstationDeviceWrites, permission.OpsWorkstationDelete, core)
			wdDeletes.POST("/:id/delete", workstationDeviceHandler.Delete)
		}

		// 资产管理（写操作见下方 assetWrites 严格兄弟组，85-03 RBAC-01）
		assetService := opsServices.NewAssetService(core.DB.GetDB())
		assetHandler := operations.NewAssetHandler(assetService).WithCore(core)
		assets := ops.Group("/asset")
		assets.Use(middleware.RequirePermissions([]string{
			"ops:asset:list",
			"ops:asset:add",
			"ops:asset:edit",
			"ops:asset:delete",
		}, core))
		// DS-02: 权限校验之后、handler 之前注入 data_scope/user_id（含 request context）。
		assets.Use(middleware.DataScopePermission(core))
		{
			// 注意：更具体的路由要放在前面
			assets.GET("/search-by-serial/:serial", assetHandler.SearchBySerial)

			// Phase 48 Wave 3 (D-07): 从属组件清单 read-only 端点。
			// 复用 ops:asset:list 组级 middleware(MEMORY xingran-perm-namespace-split-readonly-page)。
			// 放在 /:id 通配之前避免被吞噬。
			assetComponentHandler := operations.NewAssetComponentHandler(core)
			assets.GET("/components", assetComponentHandler.ListComponents)

			assets.POST("/list", assetHandler.List)
			// 资产统计(专用 COUNT 聚合,不依赖分页列表)
			assets.POST("/statistics", assetHandler.Statistics)
			assets.POST("/:id", assetHandler.GetByID)
			assets.POST("/device-types", assetHandler.GetDeviceTypes)
			assets.POST("/device-categories", assetHandler.GetDeviceCategories)
			assets.POST("/status-values", assetHandler.GetStatusValues)

			// Excel 模板下载（读操作，保留在读组）
			operations.SetupExcelReadRouter(assets, "asset", core)
		}

		// 资产管理写路由（85-03 RBAC-01 严格兄弟组）
		assetWrites := ops.Group("/asset")
		// DS-02: 写组为读组的兄弟组（不继承读组 ANY-of），注入 data_scope/user_id 供严格写端点取范围。
		assetWrites.Use(middleware.DataScopePermission(core))
		{
			assetAdd := strictUserRouteGroup(assetWrites, permission.OpsAssetAdd, core)
			assetAdd.POST("", assetHandler.Create)
			assetEdit := strictUserRouteGroup(assetWrites, permission.OpsAssetEdit, core)
			assetEdit.POST("/:id/update", assetHandler.Update)
			assetDeletes := strictUserRouteGroup(assetWrites, permission.OpsAssetDelete, core)
			assetDeletes.POST("/:id/delete", assetHandler.Delete)
			assetDeletes.POST("/batch", assetHandler.BatchOperation)
			assetImport := strictUserRouteGroup(assetWrites, permission.OpsAssetImport, core)
			operations.SetupExcelImportRouter(assetImport, "asset", core)
			assetExport := strictUserRouteGroup(assetWrites, permission.OpsAssetExport, core)
			operations.SetupExcelExportRouter(assetExport, "asset", core)
		}

		// 信息点管理（写操作见下方 infoPointWrites 严格兄弟组，85-03 RBAC-01）
		infoPointService := opsServices.NewInfoPointService(core.DB.GetDB())
		infoPointHandler := operations.NewInfoPointHandler(infoPointService).WithCore(core)
		infoPoints := ops.Group("/infoPoint")
		infoPoints.Use(middleware.RequirePermissions([]string{
			"ops:infopoint:list",
			"ops:infopoint:add",
			"ops:infopoint:edit",
			"ops:infopoint:delete",
		}, core))
		// DS-02: 权限校验之后、handler 之前注入 data_scope/user_id（含 request context）。
		infoPoints.Use(middleware.DataScopePermission(core))
		{
			// 注意：更具体的路由要放在前面
			infoPoints.POST("/list", infoPointHandler.List)
			infoPoints.POST("/statistics", infoPointHandler.Statistics)
			infoPoints.POST("/dropdown-options", infoPointHandler.SearchInfoPointOptions)
			infoPoints.POST("/:id", infoPointHandler.GetByID)

			// Excel 模板下载（读操作，保留在读组）
			operations.SetupExcelReadRouter(infoPoints, "infoPoint", core)
		}

		// 信息点管理写路由（85-03 RBAC-01 严格兄弟组）
		infoPointWrites := ops.Group("/infoPoint")
		// DS-02: 写组为读组的兄弟组（不继承读组 ANY-of），注入 data_scope/user_id 供严格写端点取范围。
		infoPointWrites.Use(middleware.DataScopePermission(core))
		{
			infoPointAdd := strictUserRouteGroup(infoPointWrites, permission.OpsInfoPointAdd, core)
			infoPointAdd.POST("", infoPointHandler.Create) // 兜底路由
			infoPointEdit := strictUserRouteGroup(infoPointWrites, permission.OpsInfoPointEdit, core)
			infoPointEdit.POST("/:id/update", infoPointHandler.Update)
			infoPointDeletes := strictUserRouteGroup(infoPointWrites, permission.OpsInfoPointDelete, core)
			infoPointDeletes.POST("/:id/delete", infoPointHandler.Delete)
			infoPointDeletes.POST("/batch", infoPointHandler.BatchOperation)
			infoPointImport := strictUserRouteGroup(infoPointWrites, permission.OpsInfoPointImport, core)
			operations.SetupExcelImportRouter(infoPointImport, "infoPoint", core)
			infoPointExport := strictUserRouteGroup(infoPointWrites, permission.OpsInfoPointExport, core)
			operations.SetupExcelExportRouter(infoPointExport, "infoPoint", core)
		}

		// 机房管理（写操作见下方 serverRoomWrites 严格兄弟组，85-03 RBAC-01）
		serverRoomService := opsServices.NewServerRoomService(core.DB.GetDB())
		serverRoomHandler := operations.NewServerRoomHandler(serverRoomService).WithCore(core)
		serverRooms := ops.Group("/serverRoom")
		serverRooms.Use(middleware.RequirePermissions([]string{
			"ops:serverroom:list",
			"ops:serverroom:add",
			"ops:serverroom:edit",
			"ops:serverroom:delete",
		}, core))
		// DS-02: 权限校验之后、handler 之前注入 data_scope/user_id（含 request context）。
		serverRooms.Use(middleware.DataScopePermission(core))
		{
			// 注意：更具体的路由要放在前面
			serverRooms.POST("/list", serverRoomHandler.List)
			serverRooms.POST("/statistics", serverRoomHandler.Statistics)
			serverRooms.POST("/dropdown-options", serverRoomHandler.SearchServerRoomOptions)
			serverRooms.POST("/:id", serverRoomHandler.GetByID)

			// Excel 模板下载（读操作，保留在读组）
			operations.SetupExcelReadRouter(serverRooms, "serverRoom", core)
		}

		// 机房管理写路由（85-03 RBAC-01 严格兄弟组）
		serverRoomWrites := ops.Group("/serverRoom")
		// DS-02: 写组为读组的兄弟组（不继承读组 ANY-of），注入 data_scope/user_id 供严格写端点取范围。
		serverRoomWrites.Use(middleware.DataScopePermission(core))
		{
			serverRoomAdd := strictUserRouteGroup(serverRoomWrites, permission.OpsServerRoomAdd, core)
			serverRoomAdd.POST("", serverRoomHandler.Create) // 兜底路由
			serverRoomEdit := strictUserRouteGroup(serverRoomWrites, permission.OpsServerRoomEdit, core)
			serverRoomEdit.POST("/:id/update", serverRoomHandler.Update)
			serverRoomDeletes := strictUserRouteGroup(serverRoomWrites, permission.OpsServerRoomDelete, core)
			serverRoomDeletes.POST("/:id/delete", serverRoomHandler.Delete)
			serverRoomDeletes.POST("/batch", serverRoomHandler.BatchOperation)
			serverRoomImport := strictUserRouteGroup(serverRoomWrites, permission.OpsServerRoomImport, core)
			operations.SetupExcelImportRouter(serverRoomImport, "serverRoom", core)
			serverRoomExport := strictUserRouteGroup(serverRoomWrites, permission.OpsServerRoomExport, core)
			operations.SetupExcelExportRouter(serverRoomExport, "serverRoom", core)
		}

		// 机房照片管理
		roomPhotos := ops.Group("/rooms/photos")
		// RBAC（task-4）：照片端点级权限在 SetupRoomPhotoRouter 内挂载（读 ANY-of
		// ops:serverroom:list/query、写 strict(edit)、删 strict(delete)，域借用注释见彼处）。
		// DS-02: 权限校验之后、handler 之前注入 data_scope/user_id（含 request context）。
		roomPhotos.Use(middleware.DataScopePermission(core))
		{
			operations.SetupRoomPhotoRouter(roomPhotos, core)
		}

		// 专线管理
		// perms 对齐: 专线模块统一用 ops:dedicatedline:* (单数无连字符),
		// 与菜单 seed / sys_menu 一致, 避免命名空间割裂导致 403。
		// 写操作见下方 dedicatedLineWrites 严格兄弟组（85-03 RBAC-01）。
		dedicatedLineService := opsServices.NewDedicatedLineService(core.DB.GetDB())
		dedicatedLineHandler := operations.NewDedicatedLineHandler(dedicatedLineService).WithCore(core)
		dedicatedLines := ops.Group("/dedicatedLine")
		dedicatedLines.Use(middleware.RequirePermissions([]string{
			"ops:dedicatedline:list",
			"ops:dedicatedline:add",
			"ops:dedicatedline:edit",
			"ops:dedicatedline:delete",
		}, core))
		// DS-02: 权限校验之后、handler 之前注入 data_scope/user_id（含 request context）。
		dedicatedLines.Use(middleware.DataScopePermission(core))
		{
			// 注意：更具体的路由要放在前面
			dedicatedLines.POST("/list", dedicatedLineHandler.List)
			dedicatedLines.POST("/statistics", dedicatedLineHandler.Statistics)
			dedicatedLines.POST("/dropdown-options", dedicatedLineHandler.SearchDedicatedLineOptions)
			dedicatedLines.POST("/:id", dedicatedLineHandler.GetByID)

			// Excel 模板下载（读操作，保留在读组）
			operations.SetupExcelReadRouter(dedicatedLines, "dedicatedLine", core)
		}

		// 专线管理写路由（85-03 RBAC-01 严格兄弟组）
		dedicatedLineWrites := ops.Group("/dedicatedLine")
		// DS-02: 写组为读组的兄弟组（不继承读组 ANY-of），注入 data_scope/user_id 供严格写端点取范围。
		dedicatedLineWrites.Use(middleware.DataScopePermission(core))
		{
			dedicatedLineAdd := strictUserRouteGroup(dedicatedLineWrites, permission.OpsDedicatedLineAdd, core)
			dedicatedLineAdd.POST("", dedicatedLineHandler.Create) // 兜底路由
			dedicatedLineEdit := strictUserRouteGroup(dedicatedLineWrites, permission.OpsDedicatedLineEdit, core)
			dedicatedLineEdit.POST("/:id/update", dedicatedLineHandler.Update)
			dedicatedLineDeletes := strictUserRouteGroup(dedicatedLineWrites, permission.OpsDedicatedLineDelete, core)
			dedicatedLineDeletes.POST("/:id/delete", dedicatedLineHandler.Delete)
			dedicatedLineDeletes.POST("/batch", dedicatedLineHandler.BatchOperation)
			dedicatedLineImport := strictUserRouteGroup(dedicatedLineWrites, permission.OpsDedicatedLineImport, core)
			operations.SetupExcelImportRouter(dedicatedLineImport, "dedicatedLine", core)
			dedicatedLineExport := strictUserRouteGroup(dedicatedLineWrites, permission.OpsDedicatedLineExport, core)
			operations.SetupExcelExportRouter(dedicatedLineExport, "dedicatedLine", core)
		}

		// room-devices - 机房设备管理（写操作见下方 roomDeviceWrites 严格兄弟组，85-03 RBAC-01）
		roomDeviceService := opsServices.NewRoomDeviceService(core.DB.GetDB())
		roomDeviceHandler := operations.NewRoomDeviceHandler(roomDeviceService).WithCore(core)
		roomDevices := ops.Group("/roomDevice")
		roomDevices.Use(middleware.RequirePermissions([]string{
			"ops:roomdevice:list",
			"ops:roomdevice:add",
			"ops:roomdevice:edit",
			"ops:roomdevice:delete",
		}, core))
		// DS-02: 权限校验之后、handler 之前注入 data_scope/user_id（含 request context）。
		roomDevices.Use(middleware.DataScopePermission(core))
		{
			// 注意：更具体的路由要放在前面
			roomDevices.POST("/list", roomDeviceHandler.List)
			roomDevices.POST("/statistics", roomDeviceHandler.Statistics)
			roomDevices.POST("/dropdown-options", roomDeviceHandler.SearchRoomDeviceOptions)
			roomDevices.POST("/:id", roomDeviceHandler.GetByID)

			// Excel 模板下载（读操作，保留在读组）
			operations.SetupExcelReadRouter(roomDevices, "roomDevice", core)
		}

		// 机房设备管理写路由（85-03 RBAC-01 严格兄弟组）
		roomDeviceWrites := ops.Group("/roomDevice")
		// DS-02: 写组为读组的兄弟组（不继承读组 ANY-of），注入 data_scope/user_id 供严格写端点取范围。
		roomDeviceWrites.Use(middleware.DataScopePermission(core))
		{
			roomDeviceAdd := strictUserRouteGroup(roomDeviceWrites, permission.OpsRoomDeviceAdd, core)
			roomDeviceAdd.POST("", roomDeviceHandler.Create) // 兜底路由
			roomDeviceEdit := strictUserRouteGroup(roomDeviceWrites, permission.OpsRoomDeviceEdit, core)
			roomDeviceEdit.POST("/:id/update", roomDeviceHandler.Update)
			roomDeviceDeletes := strictUserRouteGroup(roomDeviceWrites, permission.OpsRoomDeviceDelete, core)
			roomDeviceDeletes.POST("/:id/delete", roomDeviceHandler.Delete)
			roomDeviceDeletes.POST("/batch", roomDeviceHandler.BatchOperation)
			roomDeviceImport := strictUserRouteGroup(roomDeviceWrites, permission.OpsRoomDeviceImport, core)
			operations.SetupExcelImportRouter(roomDeviceImport, "roomDevice", core)
			roomDeviceExport := strictUserRouteGroup(roomDeviceWrites, permission.OpsRoomDeviceExport, core)
			operations.SetupExcelExportRouter(roomDeviceExport, "roomDevice", core)
		}

		// CAD平面图编辑器（包含墙体、门等元素）
		// 使用平面图编辑器和墙体、门的权限标识符
		// 写操作见下方 wallsWrites 严格兄弟组（85-03 RBAC-01）。
		wallService := opsServices.NewWallService(core.DB.GetDB())
		wallHandler := operations.NewWallHandler(wallService).WithCore(core)
		walls := ops.Group("/walls")
		// FIX-13：5 个读码改引用 permission.* 常量（单一真相源），
		// 由 migration_236 播种 F 型菜单，使持对应权限码的角色可读（403→200）。
		walls.Use(middleware.RequirePermissions([]string{
			string(permission.OpsFloorPlanList),
			string(permission.OpsFloorPlanQuery),
			string(permission.OpsFloorPlanEdit),
			string(permission.OpsFloorPlanSave),
			string(permission.OpsWallList),
			string(permission.OpsWallAdd),
			string(permission.OpsWallEdit),
			string(permission.OpsWallDelete),
		}, core))
		// DS-02: 权限校验之后、handler 之前注入 data_scope/user_id（含 request context）。
		walls.Use(middleware.DataScopePermission(core))
		{
			walls.POST("/list", wallHandler.List)
			walls.POST("/:id", wallHandler.GetByID)
		}

		// 墙体写路由（85-03 RBAC-01 严格兄弟组）
		wallsWrites := ops.Group("/walls")
		// DS-02: 写组为读组的兄弟组（不继承读组 ANY-of），注入 data_scope/user_id 供严格写端点取范围。
		wallsWrites.Use(middleware.DataScopePermission(core))
		{
			wallsAdd := strictUserRouteGroup(wallsWrites, permission.OpsWallAdd, core)
			wallsAdd.POST("", wallHandler.Create)
			wallsEdit := strictUserRouteGroup(wallsWrites, permission.OpsWallEdit, core)
			wallsEdit.POST("/:id/update", wallHandler.Update)
			wallsDeletes := strictUserRouteGroup(wallsWrites, permission.OpsWallDelete, core)
			wallsDeletes.POST("/:id/delete", wallHandler.Delete)
			wallsDeletes.POST("/batch", wallHandler.BatchOperation)
		}

		// 门管理（CAD平面图）- 使用平面图编辑器和门权限
		// 写操作见下方 doorsWrites 严格兄弟组（85-03 RBAC-01）。
		doorService := opsServices.NewDoorService(core.DB.GetDB())
		doorHandler := operations.NewDoorHandler(doorService).WithCore(core)
		doors := ops.Group("/doors")
		doors.Use(middleware.RequirePermissions([]string{
			string(permission.OpsFloorPlanList),
			string(permission.OpsFloorPlanQuery),
			string(permission.OpsFloorPlanEdit),
			string(permission.OpsFloorPlanSave),
			string(permission.OpsDoorList),
			string(permission.OpsDoorAdd),
			string(permission.OpsDoorEdit),
			string(permission.OpsDoorDelete),
		}, core))
		// DS-02: 权限校验之后、handler 之前注入 data_scope/user_id（含 request context）。
		doors.Use(middleware.DataScopePermission(core))
		{
			doors.POST("/list", doorHandler.List)
			doors.POST("/:id", doorHandler.GetByID)
		}

		// 门写路由（85-03 RBAC-01 严格兄弟组）
		doorsWrites := ops.Group("/doors")
		// DS-02: 写组为读组的兄弟组（不继承读组 ANY-of），注入 data_scope/user_id 供严格写端点取范围。
		doorsWrites.Use(middleware.DataScopePermission(core))
		{
			doorsAdd := strictUserRouteGroup(doorsWrites, permission.OpsDoorAdd, core)
			doorsAdd.POST("", doorHandler.Create)
			doorsEdit := strictUserRouteGroup(doorsWrites, permission.OpsDoorEdit, core)
			doorsEdit.POST("/:id/update", doorHandler.Update)
			doorsDeletes := strictUserRouteGroup(doorsWrites, permission.OpsDoorDelete, core)
			doorsDeletes.POST("/:id/delete", doorHandler.Delete)
			doorsDeletes.POST("/batch", doorHandler.BatchOperation)
		}

		// 平面图文本管理（CAD平面图）
		// 写操作见下方 textsWrites 严格兄弟组（85-03 RBAC-01，归平面图编辑码）。
		textService := opsServices.NewFloorPlanTextService(core.DB.GetDB())
		textHandler := operations.NewFloorPlanTextHandler(textService).WithCore(core)
		texts := ops.Group("/floor-plan-texts")
		// task-3 口径记录（G5 借码说明，2026-11-11）：平面图文本组借用 ops:floor-plan:*
		// 全 4 码 ANY-of（list/query/edit/save），读写同组未拆 strict——平面图编辑场景
		// 本就整组授权（画布协同），风险可接受，只记录不整改。
		texts.Use(middleware.RequirePermissions([]string{
			string(permission.OpsFloorPlanList),
			string(permission.OpsFloorPlanQuery),
			string(permission.OpsFloorPlanEdit),
			string(permission.OpsFloorPlanSave),
		}, core))
		// DS-02: 权限校验之后、handler 之前注入 data_scope/user_id（含 request context）。
		texts.Use(middleware.DataScopePermission(core))
		{
			texts.POST("/list", textHandler.List)
			texts.POST("/:id", textHandler.GetByID)
		}

		// 平面图文本写路由（85-03 RBAC-01 严格兄弟组；域内无专属写码，归 ops:floor-plan:edit）
		textsWrites := ops.Group("/floor-plan-texts")
		// DS-02: 写组为读组的兄弟组（不继承读组 ANY-of），注入 data_scope/user_id 供严格写端点取范围。
		textsWrites.Use(middleware.DataScopePermission(core))
		{
			textsEdits := strictUserRouteGroup(textsWrites, permission.OpsFloorPlanEdit, core)
			textsEdits.POST("", textHandler.Create)
			textsEdits.POST("/:id/update", textHandler.Update)
			textsEdits.POST("/:id/delete", textHandler.Delete)
			textsEdits.POST("/batch", textHandler.BatchOperation)
		}
	}

	// VDI管理模块（虚拟桌面基础设施）
	vdi := r.Group("/vdi")
	vdi.Use(middleware.JWTAuth(core.JWTManager))
	// 添加操作日志中间件
	vdi.Use(middleware.OperLogMiddleware(core.OperLogService, core))
	{
		// 虚拟机管理
		// 虚拟机管理
		// task-3 G3 口径记录（2026-11-11 granularity 审计）：vms/servers 组为
		// ANY-of（含写码）且未拆 strict 写组——持 vdi:vm:list 即可调虚拟机写端点。
		// 属已知低危残留（VDI 管理面角色本就整组授权），本轮只记录不整改，
		// 后续若拆 strict 需对齐 vdi/vm_router.go 端点级先例形态。
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
	//
	// task-3 G2 整改（2026-11-11）：拆「读 ANY-of / 写 strict」兄弟组。
	// 此前 9 码平铺整组 ANY-of，持 list 即可调 fix:apply/rollback。整改后：
	//   读组（下方）ANY-of 仅读码：list / dashboard / fix:list / fix:stats；
	//   strict 写组（下方 assetReconStrict）逐动作 strict：fix:accept/reject/apply/
	//   rollback 与 exception-rule export/template（export 码）；
	//   通用写组（assetReconWrites）ANY-of 写语义码集兜底 resolve/refresh 等
	//   无独立动作码的写端点——读码持有者无法触达。
	// 口径记录：SetupReconciliationExceptionRouter 内 exception-rule
	// create/update/delete/test/import 与 baseline/snapshot 等端点保留端点级
	// RequirePermissions(exception:create/update/delete/test) 自护，继续挂读组
	//（端点级拦截晚于组级放行，安全语义不变）。
	assetReconciliation := r.Group("/asset/reconciliation")
	assetReconciliation.Use(middleware.JWTAuth(core.JWTManager))
	// 添加操作日志中间件(D-17:全部写操作都走 operlog)。LogPaths 必须在 pkg/middleware/oper_log.go 追加 /asset/reconciliation 才会触发。
	assetReconciliation.Use(middleware.OperLogMiddleware(core.OperLogService, core))
	assetReconciliation.Use(middleware.RequirePermissions([]string{
		"asset:reconciliation:list",
		"asset:reconciliation:dashboard",
		// Phase 46 R5: 修复建议命名空间 5 个 perm 中的读码（写码迁移见下方 strict 组）
		"asset:reconciliation:fix:list",
		"asset:reconciliation:fix:stats",
	}, core))
	{
		// 异常列表读端点 (SetupReconciliationRouter: /exception/list, /exception/:id)
		assetV1.SetupReconciliationRouter(assetReconciliation, core)
		// 例外规则端点（读 + 自护写端点，口径见上方注释）
		assetV1.SetupReconciliationExceptionRouter(assetReconciliation, core)
		// 6 个统计端点 (SetupReconciliationStatisticsRouter: /statistics/summary 等)
		assetV1.SetupReconciliationStatisticsRouter(assetReconciliation, core)
		// Phase 46 R5: 修复建议读端点 (list/getById/stats)
		assetV1.SetupFixSuggestionRouter(assetReconciliation, core)
	}

	// 对账通用写组（task-3 G2）：resolve/refresh 等无独立动作码的写端点，
	// 以写语义码集 ANY-of 兜底（任一写码持有者可调；读码持有者不可达）。
	assetReconWrites := r.Group("/asset/reconciliation")
	assetReconWrites.Use(middleware.JWTAuth(core.JWTManager))
	assetReconWrites.Use(middleware.OperLogMiddleware(core.OperLogService, core))
	assetReconWrites.Use(middleware.RequirePermissions([]string{
		"asset:reconciliation:export",
		"asset:reconciliation:fix:accept",
		"asset:reconciliation:fix:reject",
		"asset:reconciliation:fix:apply",
		"asset:reconciliation:fix:rollback",
	}, core))
	{
		assetV1.SetupReconciliationWriteRouter(assetReconWrites, core)
	}

	// 对账严格写组（task-3 G2）：逐动作 RequireStrictPermissions 单一权限点。
	assetReconStrict := r.Group("/asset/reconciliation")
	assetReconStrict.Use(middleware.JWTAuth(core.JWTManager))
	assetReconStrict.Use(middleware.OperLogMiddleware(core.OperLogService, core))
	{
		assetV1.SetupFixSuggestionWriteRouter(assetV1.FixSuggestionWriteRouteGroups{
			Accept:   strictLiteralRouteGroup(assetReconStrict, "asset:reconciliation:fix:accept", core),
			Reject:   strictLiteralRouteGroup(assetReconStrict, "asset:reconciliation:fix:reject", core),
			Apply:    strictLiteralRouteGroup(assetReconStrict, "asset:reconciliation:fix:apply", core),
			Rollback: strictLiteralRouteGroup(assetReconStrict, "asset:reconciliation:fix:rollback", core),
		}, core)
		// 例外规则导出/模板下载迁 strict export 码（导出为业务写操作，users 先例口径）
		assetReconExport := strictLiteralRouteGroup(assetReconStrict, "asset:reconciliation:export", core)
		assetV1.SetupReconciliationExceptionExportRouter(assetReconExport, core)
	}

	// 注意：前端静态文件路由已移至 cmd/main.go，直接注册在 engine 上
}
