package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// ============================================================================
// Migration 260 — system:config 动作细码 F 型播种（孤儿码转正）
//
// 背景：站点品牌配置（系统设置-站点品牌 tab）上线后，超级管理员看到
// 「当前账号无 system:config:edit 权限，仅可查看品牌配置」。
// 根因不是前端判定写错，而是权限码从未播种：
//   sys_menu 里 system:config 域只有 2 行 ——
//     · 'system:config'      （参数配置 C 型菜单，粗码）
//     · 'system:config:list' （系统设置 C 型菜单）
//   而 pkg/permission/config.go 声明的 view/add/edit/remove 四个细码
//   没有任何菜单行承载 → menuService.GetUserPermissions（从 sys_menu.perms
//   Pluck）永远取不到 → 前端权限集恒不含该码 → 按钮门控判"无权限"。
//   后端 RequireStrictPermissions 先走 isSuperAdmin 旁路，所以超管实际能写、
//   前端却显示无权限，属"前后端权限源不一致"。
//   连带存量缺陷：普通角色也无法在角色授权页勾选这四个码。
//
// 修复：把四个细码补成 F 型按钮行，父 C 型菜单 perms='system:config'
// （参数配置菜单，menu_catalog_seed.sql 既有行），复用 migration_226 全套
// helper（resolveActionPermissionParents / ensureActionPermissionMenu /
// grantActionPermissionToParentHolders）。双方言（PG / sqlite 通用）、幂等、
// 单条失败仅记日志不阻断。
//
// 授权链路：新 F 行由 grantActionPermissionToParentHolders 精准授给
// 「已持有父菜单 system:config」的角色 —— 超管角色天然持有，
// 因此 migrate 后前端权限集立即含这四个码，无需手工在授权页补勾。
// ============================================================================

// F_TYPE_SEED_BEGIN
// configActionPermissionFTypeSeedCodes F 型声明块：供
// xingran-react-frontend/scripts/check-permission-codes.mjs 的 Go 侧标记解析器
// 识别（按 F 型按钮参与「菜单 F 型 vs TS 声明」对账）。必须与下方种子表逐条
// 1:1 一致；父 perms 留在块外，避免被判为 F 型按钮。
// ⚠️ 本块内不得出现标记字面量（注释里也不要写），否则正则会在注释处提前闭合。
var configActionPermissionFTypeSeedCodes = []string{
	"system:config:view",
	"system:config:add",
	"system:config:edit",
	"system:config:remove",
}

// F_TYPE_SEED_END

// configActionPermissionSeeds 种子表（与上方声明块逐条一致）。
var configActionPermissionSeeds = []actionPermissionSeed{
	{name: "参数配置查看", perms: "system:config:view", parentPerms: "system:config", orderNum: 1},
	{name: "参数配置新增", perms: "system:config:add", parentPerms: "system:config", orderNum: 2},
	{name: "参数配置编辑", perms: "system:config:edit", parentPerms: "system:config", orderNum: 3},
	{name: "参数配置删除", perms: "system:config:remove", parentPerms: "system:config", orderNum: 4},
}

// Migrate260ConfigActionPermissionFTypeSeed 播种 system:config 四个动作细码并
// 精准授权给持父菜单（system:config）的角色。双方言、幂等、非阻断降级。
func Migrate260ConfigActionPermissionFTypeSeed(db *gorm.DB) error {
	log.Println("Running migration 260: system:config action permission F type seed")

	// 0) 入口硬断言：声明块与种子表必须条数相同且逐条相等
	if len(configActionPermissionFTypeSeedCodes) != len(configActionPermissionSeeds) {
		return fmt.Errorf("migration 260 致命漂移: F_TYPE 块声明数(%d) != 种子表数(%d)",
			len(configActionPermissionFTypeSeedCodes), len(configActionPermissionSeeds))
	}
	for i, code := range configActionPermissionFTypeSeedCodes {
		if code != configActionPermissionSeeds[i].perms {
			return fmt.Errorf("migration 260 致命漂移: 第 %d 项码不匹配声明(%s) != 种子(%s)",
				i, code, configActionPermissionSeeds[i].perms)
		}
	}

	for _, seed := range configActionPermissionSeeds {
		parents, err := resolveActionPermissionParents(db, seed)
		if err != nil {
			log.Printf("Migration 260: WARNING 查询父菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}
		if len(parents) == 0 {
			log.Printf("Migration 260: WARNING 未找到 live 父菜单(perms=%s), 跳过创建与授权", seed.perms)
			continue
		}

		menuID, err := ensureActionPermissionMenu(db, seed, parents[0].ID)
		if err != nil {
			log.Printf("Migration 260: WARNING 创建/获取按钮菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}

		parentIDs := make([]string, 0, len(parents))
		for _, p := range parents {
			parentIDs = append(parentIDs, p.ID)
		}
		if err := grantActionPermissionToParentHolders(db, menuID, parentIDs); err != nil {
			log.Printf("Migration 260: WARNING 授权按钮菜单失败(perms=%s, menuID=%s): %v, 跳过授权",
				seed.perms, menuID, err)
			continue
		}
	}

	log.Println("Migration 260 completed: system:config action permissions seeded")
	return nil
}
