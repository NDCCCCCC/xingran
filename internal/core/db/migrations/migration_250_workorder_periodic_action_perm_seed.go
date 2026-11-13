package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// ============================================================================
// Migration 250 — 周期工单模板动作权限码播种（RBAC-EXC-01 清偿）
//
// RBAC-EXC-01：周期工单模板的 enable/disable/generate/logs 4 条端点此前挂在
// ANY-of 读组（ops:workorder:periodic:list/add/edit/delete），仅持只读 list 码的
// 角色也能调用状态变更 / 生成工单等写操作，违反 #87 最小权限口径。
// 现已按逐动作口径迁至 RequireStrictPermissions 严格写组，权限码为：
//
//	ops:workorder:periodic:enable    启用模板（写）
//	ops:workorder:periodic:disable   停用模板（写）
//	ops:workorder:periodic:generate  立即生成工单（写）
//	ops:workorder:periodic:logs      执行记录（只读语义，独立码）
//
// 若 sys_menu 无承载这些码的 F 型按钮行，除 admin 外所有角色调用对应端点
// 必然 403（与 OQ-7 / migration_237 同型）。故本迁移补齐 F 型授权载体：
// 建 F 行挂 live 父 C 菜单「周期性工单」（perms=ops:workorder:periodic:list，
// menu_catalog_seed.sql 行 149），并精准授权给持有该父菜单的角色。
//
// 写法与授权策略复用 migration_226 全套 helper（resolveActionPermissionParents /
// ensureActionPermissionMenu / grantActionPermissionToParentHolders），
// 双方言（PG / sqlite 通用）、幂等、非阻断降级。
// 不改严格权限中间件语义（RBAC 零回归），只补数据。
// ============================================================================

// F_TYPE_SEED_BEGIN
var workOrderPeriodicActionFTypeSeedCodes = []string{
	"ops:workorder:periodic:enable",
	"ops:workorder:periodic:disable",
	"ops:workorder:periodic:generate",
	"ops:workorder:periodic:logs",
}

// F_TYPE_SEED_END

var workOrderPeriodicActionSeeds = []actionPermissionSeed{
	{name: "启用周期工单模板", perms: "ops:workorder:periodic:enable", parentPerms: "ops:workorder:periodic:list", orderNum: 1},
	{name: "停用周期工单模板", perms: "ops:workorder:periodic:disable", parentPerms: "ops:workorder:periodic:list", orderNum: 2},
	{name: "立即生成周期工单", perms: "ops:workorder:periodic:generate", parentPerms: "ops:workorder:periodic:list", orderNum: 3},
	{name: "周期工单执行记录", perms: "ops:workorder:periodic:logs", parentPerms: "ops:workorder:periodic:list", orderNum: 4},
}

// Migrate250WorkOrderPeriodicActionPermSeed 播种周期工单模板动作权限码
// （双方言 / 幂等 / 非阻断）。
func Migrate250WorkOrderPeriodicActionPermSeed(db *gorm.DB) error {
	log.Println("Running migration 250: workorder periodic action permission seed (RBAC-EXC-01)")

	if len(workOrderPeriodicActionFTypeSeedCodes) != len(workOrderPeriodicActionSeeds) {
		return fmt.Errorf("migration 250 致命漂移: F_TYPE 块声明数(%d) != 种子表数(%d)",
			len(workOrderPeriodicActionFTypeSeedCodes), len(workOrderPeriodicActionSeeds))
	}
	for i, code := range workOrderPeriodicActionFTypeSeedCodes {
		if code != workOrderPeriodicActionSeeds[i].perms {
			return fmt.Errorf("migration 250 致命漂移: 第 %d 项码不匹配声明(%s) != 种子(%s)",
				i, code, workOrderPeriodicActionSeeds[i].perms)
		}
	}

	for _, seed := range workOrderPeriodicActionSeeds {
		parents, err := resolveActionPermissionParents(db, seed)
		if err != nil {
			log.Printf("Migration 250: WARNING 查询父菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}
		if len(parents) == 0 {
			log.Printf("Migration 250: WARNING 未找到 live 父菜单(ops:workorder:periodic:list), 跳过创建与授权(perms=%s)", seed.perms)
			continue
		}

		menuID, err := ensureActionPermissionMenu(db, seed, parents[0].ID)
		if err != nil {
			log.Printf("Migration 250: WARNING 创建/获取按钮菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}

		parentIDs := make([]string, 0, len(parents))
		for _, p := range parents {
			parentIDs = append(parentIDs, p.ID)
		}
		if err := grantActionPermissionToParentHolders(db, menuID, parentIDs); err != nil {
			log.Printf("Migration 250: WARNING 授权按钮菜单失败(perms=%s, menuID=%s): %v, 跳过授权",
				seed.perms, menuID, err)
			continue
		}
	}

	log.Println("Migration 250 completed: workorder periodic action permissions seeded")
	return nil
}
