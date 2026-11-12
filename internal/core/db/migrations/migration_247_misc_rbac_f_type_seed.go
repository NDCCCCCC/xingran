package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// ============================================================================
// Migration 247 — 杂项 RBAC F 型按钮播种（task-4）
//
// 两批：
//   1. network:mac 域 3 个新码（task-4 A 项 mac/mac-history/topology 组接线新增，
//      常量见 config.go NetworkMacCollect/NetworkMacRemove/NetworkMacRuleEdit），
//      父 C 型菜单 = 「MAC地址」（perms=network:mac，menu_catalog_seed.sql 行 198）。
//   2. system:captchaBackground 域 5 码（task-4 C 项孤儿转正：captcha-backgrounds
//      路由已挂端点级权限，前端 system/settings/captcha-background 页活跃调用），
//      父 C 型菜单 = 「系统设置」（perms=system:config:list，行 203）。
//
// 写法与授权策略复用 migration_226 全套 helper（resolveActionPermissionParents /
// ensureActionPermissionMenu / grantActionPermissionToParentHolders），
// 双方言（PG / sqlite 通用）、幂等、非阻断降级。
// ============================================================================

var miscRbacFTypeSeedCodes = []string{
	// 1. network:mac 域 (4 条；query 为既有码，接线后首次播种)
	"network:mac:query",
	"network:mac:collect",
	"network:mac:remove",
	"network:mac:rule:edit",

	// 2. system:captchaBackground 孤儿转正 (5 条)
	"system:captchaBackground:list",
	"system:captchaBackground:view",
	"system:captchaBackground:add",
	"system:captchaBackground:edit",
	"system:captchaBackground:remove",
}

var miscRbacPermissionSeeds = []actionPermissionSeed{
	// 1. network:mac 域 (4 条)
	{name: "MAC查询", perms: "network:mac:query", parentPerms: "network:mac", orderNum: 1},
	{name: "MAC采集", perms: "network:mac:collect", parentPerms: "network:mac", orderNum: 2},
	{name: "MAC清理", perms: "network:mac:remove", parentPerms: "network:mac", orderNum: 3},
	{name: "MAC过滤规则管理", perms: "network:mac:rule:edit", parentPerms: "network:mac", orderNum: 4},

	// 2. system:captchaBackground 孤儿转正 (5 条，父=系统设置 system:config:list)
	{name: "背景图查询", perms: "system:captchaBackground:list", parentPerms: "system:config:list", orderNum: 1},
	{name: "背景图详情", perms: "system:captchaBackground:view", parentPerms: "system:config:list", orderNum: 2},
	{name: "背景图上传", perms: "system:captchaBackground:add", parentPerms: "system:config:list", orderNum: 3},
	{name: "背景图编辑", perms: "system:captchaBackground:edit", parentPerms: "system:config:list", orderNum: 4},
	{name: "背景图删除", perms: "system:captchaBackground:remove", parentPerms: "system:config:list", orderNum: 5},
}

// Migrate247MiscRbacFTypeSeed 播种杂项 RBAC F 型按钮并精准授权给持父菜单的角色。
func Migrate247MiscRbacFTypeSeed(db *gorm.DB) error {
	log.Println("Running migration 247: misc RBAC F type permission seed")

	if len(miscRbacFTypeSeedCodes) != len(miscRbacPermissionSeeds) {
		return fmt.Errorf("migration 247 致命漂移: F_TYPE 块声明数(%d) != 种子表数(%d)",
			len(miscRbacFTypeSeedCodes), len(miscRbacPermissionSeeds))
	}
	for i, code := range miscRbacFTypeSeedCodes {
		if code != miscRbacPermissionSeeds[i].perms {
			return fmt.Errorf("migration 247 致命漂移: 第 %d 项码不匹配声明(%s) != 种子(%s)",
				i, code, miscRbacPermissionSeeds[i].perms)
		}
	}

	for _, seed := range miscRbacPermissionSeeds {
		parents, err := resolveActionPermissionParents(db, seed)
		if err != nil {
			log.Printf("Migration 247: WARNING 查询父菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}
		if len(parents) == 0 {
			log.Printf("Migration 247: WARNING 未找到 live 父菜单(perms=%s), 跳过创建与授权", seed.perms)
			continue
		}

		menuID, err := ensureActionPermissionMenu(db, seed, parents[0].ID)
		if err != nil {
			log.Printf("Migration 247: WARNING 创建/获取按钮菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}

		parentIDs := make([]string, 0, len(parents))
		for _, p := range parents {
			parentIDs = append(parentIDs, p.ID)
		}
		if err := grantActionPermissionToParentHolders(db, menuID, parentIDs); err != nil {
			log.Printf("Migration 247: WARNING 授权按钮菜单失败(perms=%s, menuID=%s): %v, 跳过授权",
				seed.perms, menuID, err)
			continue
		}
	}

	log.Println("Migration 247 completed: misc RBAC F type permissions seeded")
	return nil
}
