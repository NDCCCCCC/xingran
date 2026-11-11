package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============================================================================
// Migration 244 — monitor 域权限码 F 型按钮播种（monitor RBAC 全链路接线 · 后端件）
//
// 背景（2026-11-11 monitor RBAC 审计）：monitor 域 24 个动作码在
// pkg/permission/config.go 已定义、路由已挂端点级 RBAC（server_router /
// cache_router / oper_log_router / login_log_router / scheduler job_router），
// 但 sys_menu 无 F 型按钮播种 —— 权限在角色授权树上不可见，永远不会被授予。
// 本 migration 补齐 F 型播种链路，写法与授权策略对齐 migration_226
// （复用包内 resolveActionPermissionParents / ensureActionPermissionMenu /
// grantActionPermissionToParentHolders）。
//
// 父 C 型菜单（menu_catalog_seed.sql 已有）：
//   monitor:server（服务监控）/ monitor:cache（缓存管理）/
//   monitor:job（定时任务）/ monitor:log（日志管理，oper-logs 与 login-logs 共用父）
//
// 不播种：monitor:online:list / monitor:online:forceLogout（无 online C 型页面
// 载体，播种会造成无父菜单孤儿；待 online 页面立项时随页面播种）。
// ============================================================================

// monitorPermissionFTypeSeedCodes F 型声明块（供 check-permission-codes.mjs
// 的 F_TYPE 解析器识别；必须与 monitorPermissionSeeds 逐条 1:1 一致）。
// 注意：本文件声明块使用专用变量名，解析脚本按「块内权限码计入 menuFType」
// 的通用口径处理（Go migration 字面量 → c 方向存在性判断 + F 型比对）。
var monitorPermissionFTypeSeedCodes = []string{
	// 1. monitor:server (2 条)
	"monitor:server:list",
	"monitor:server:view",

	// 2. monitor:cache (3 条)
	"monitor:cache:list",
	"monitor:cache:view",
	"monitor:cache:clean",

	// 3. monitor:job (9 条)
	"monitor:job:list",
	"monitor:job:view",
	"monitor:job:add",
	"monitor:job:edit",
	"monitor:job:remove",
	"monitor:job:export",
	"monitor:job:start",
	"monitor:job:stop",
	"monitor:job:run",

	// 4. monitor:operlog (4 条，父=日志管理 monitor:log)
	"monitor:operlog:list",
	"monitor:operlog:view",
	"monitor:operlog:export",
	"monitor:operlog:remove",

	// 5. monitor:loginlog (4 条，父=日志管理 monitor:log)
	"monitor:loginlog:list",
	"monitor:loginlog:view",
	"monitor:loginlog:export",
	"monitor:loginlog:remove",
}

// monitorPermissionSeeds 种子表（与上方声明块逐条一致）。
var monitorPermissionSeeds = []actionPermissionSeed{
	// 1. monitor:server (2 条)
	{name: "服务监控查询", perms: "monitor:server:list", parentPerms: "monitor:server", orderNum: 1},
	{name: "服务监控保存", perms: "monitor:server:view", parentPerms: "monitor:server", orderNum: 2},

	// 2. monitor:cache (3 条)
	{name: "缓存查询", perms: "monitor:cache:list", parentPerms: "monitor:cache", orderNum: 1},
	{name: "缓存详情", perms: "monitor:cache:view", parentPerms: "monitor:cache", orderNum: 2},
	{name: "缓存清理", perms: "monitor:cache:clean", parentPerms: "monitor:cache", orderNum: 3},

	// 3. monitor:job (9 条)
	{name: "任务查询", perms: "monitor:job:list", parentPerms: "monitor:job", orderNum: 1},
	{name: "任务详情", perms: "monitor:job:view", parentPerms: "monitor:job", orderNum: 2},
	{name: "任务新增", perms: "monitor:job:add", parentPerms: "monitor:job", orderNum: 3},
	{name: "任务编辑", perms: "monitor:job:edit", parentPerms: "monitor:job", orderNum: 4},
	{name: "任务删除", perms: "monitor:job:remove", parentPerms: "monitor:job", orderNum: 5},
	{name: "任务导出", perms: "monitor:job:export", parentPerms: "monitor:job", orderNum: 6},
	{name: "任务启用", perms: "monitor:job:start", parentPerms: "monitor:job", orderNum: 7},
	{name: "任务停用", perms: "monitor:job:stop", parentPerms: "monitor:job", orderNum: 8},
	{name: "任务执行", perms: "monitor:job:run", parentPerms: "monitor:job", orderNum: 9},

	// 4. monitor:operlog (4 条，父=日志管理 monitor:log)
	{name: "操作日志查询", perms: "monitor:operlog:list", parentPerms: "monitor:log", orderNum: 1},
	{name: "操作日志详情", perms: "monitor:operlog:view", parentPerms: "monitor:log", orderNum: 2},
	{name: "操作日志导出", perms: "monitor:operlog:export", parentPerms: "monitor:log", orderNum: 3},
	{name: "操作日志删除", perms: "monitor:operlog:remove", parentPerms: "monitor:log", orderNum: 4},

	// 5. monitor:loginlog (4 条，父=日志管理 monitor:log)
	{name: "登录日志查询", perms: "monitor:loginlog:list", parentPerms: "monitor:log", orderNum: 1},
	{name: "登录日志详情", perms: "monitor:loginlog:view", parentPerms: "monitor:log", orderNum: 2},
	{name: "登录日志导出", perms: "monitor:loginlog:export", parentPerms: "monitor:log", orderNum: 3},
	{name: "登录日志删除", perms: "monitor:loginlog:remove", parentPerms: "monitor:log", orderNum: 4},
}

// Migrate244MonitorPermissionFTypeSeed 播种 monitor 域 F 型按钮并精准授权给持父菜单的角色。
// 双方言（PG / sqlite 通用）、幂等、非阻断降级（复用 migration_226 全套 helper）。
func Migrate244MonitorPermissionFTypeSeed(db *gorm.DB) error {
	log.Println("Running migration 244: monitor RBAC F type permission seed")

	// 0) 入口硬断言：声明块与种子表必须条数相同且逐条相等
	if len(monitorPermissionFTypeSeedCodes) != len(monitorPermissionSeeds) {
		return fmt.Errorf("migration 244 致命漂移: F_TYPE 块声明数(%d) != 种子表数(%d)",
			len(monitorPermissionFTypeSeedCodes), len(monitorPermissionSeeds))
	}
	for i, code := range monitorPermissionFTypeSeedCodes {
		if code != monitorPermissionSeeds[i].perms {
			return fmt.Errorf("migration 244 致命漂移: 第 %d 项码不匹配声明(%s) != 种子(%s)",
				i, code, monitorPermissionSeeds[i].perms)
		}
	}

	// 逐条处理种子（单条失败仅记录日志并继续，保证非阻断）
	for _, seed := range monitorPermissionSeeds {
		parents, err := resolveActionPermissionParents(db, seed)
		if err != nil {
			log.Printf("Migration 244: WARNING 查询父菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}
		if len(parents) == 0 {
			log.Printf("Migration 244: WARNING 未找到 live 父菜单(perms=%s), 跳过创建与授权", seed.perms)
			continue
		}

		menuID, err := ensureActionPermissionMenu(db, seed, parents[0].ID)
		if err != nil {
			log.Printf("Migration 244: WARNING 创建/获取按钮菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}

		parentIDs := make([]string, 0, len(parents))
		for _, p := range parents {
			parentIDs = append(parentIDs, p.ID)
		}
		if err := grantActionPermissionToParentHolders(db, menuID, parentIDs); err != nil {
			log.Printf("Migration 244: WARNING 授权按钮菜单失败(perms=%s, menuID=%s): %v, 跳过授权",
				seed.perms, menuID, err)
			continue
		}
	}

	log.Println("Migration 244 completed: monitor RBAC F type permissions seeded")
	return nil
}

// 编译期引用校验：确保 models 包引用不因重构失效（菜单类型常量由复用 helper 使用）。
var _ = models.MenuTypeButton
