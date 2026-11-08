package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// ============================================================================
// FIX-13 / v1.53 Phase 136：OQ-7「5 个权限码读路由恒 403」seed 修复。
//
// 根因（87-EVIDENCE.md §八）：以下 5 个读权限码
//   ops:floor-plan:list / ops:floor-plan:query / ops:floor-plan:save /
//   ops:walls:list / ops:doors:list
// 在 router.go 的 walls / doors / floor-plan-texts 读组里以 ANY-of 字面量引用，
// 但 sys_menu 里**没有任何菜单行承载这些码**，前端 permissionCodes.ts 也未声明。
// 除 admin（isSuperAdmin bypass）外的所有角色访问这些读端点必然 403。
//
// 修复：沿用 migration 226/227/235 的幂等 seed 机制，把 5 个码建成 F 型按钮行，
// 挂在 live 父 C 菜单 ops:floor:list（与 227 同款父菜单，OQ-6 用户已确认），
// 并精准授权给所有持有该父菜单的角色（NOT EXISTS 幂等）。
// 这样持楼层管理菜单的受限角色 checkUserPermission 精确匹配即放行 → 403→200。
//
// 零回归：不改 RequireStrictPermissions / RequirePermissions 语义，不改读组/写组
// 兄弟分离结构，只补数据（菜单种子）。写操作码（226/227 已播种）不受影响。
// ============================================================================

// F_TYPE_SEED_BEGIN
var actionPermissionFTypeReadCodes = []string{
	"ops:floor-plan:list",
	"ops:floor-plan:query",
	"ops:floor-plan:save",
	"ops:walls:list",
	"ops:doors:list",
}
// F_TYPE_SEED_END

var actionPermissionSeedsRead = []actionPermissionSeed{
	{name: "平面图查看", perms: "ops:floor-plan:list", parentPerms: "ops:floor:list", orderNum: 30},
	{name: "平面图查询", perms: "ops:floor-plan:query", parentPerms: "ops:floor:list", orderNum: 31},
	{name: "平面图保存", perms: "ops:floor-plan:save", parentPerms: "ops:floor:list", orderNum: 32},
	{name: "墙体查看", perms: "ops:walls:list", parentPerms: "ops:floor:list", orderNum: 33},
	{name: "门查看", perms: "ops:doors:list", parentPerms: "ops:floor:list", orderNum: 34},
}

// Migrate236OQ7ReadPermSeed 播种 OQ-7 的 5 个读权限码（双方言 / 幂等 / 非阻断）。
func Migrate236OQ7ReadPermSeed(db *gorm.DB) error {
	log.Println("Running migration 236: OQ-7 5 read permission codes seed (FIX-13)")

	// 入口硬断言：F_TYPE_SEED 声明块与种子表条数相同且逐条相等
	if len(actionPermissionFTypeReadCodes) != len(actionPermissionSeedsRead) {
		return fmt.Errorf("migration 236 致命漂移: F_TYPE_SEED 声明数(%d) != 种子表数(%d)",
			len(actionPermissionFTypeReadCodes), len(actionPermissionSeedsRead))
	}
	for i, code := range actionPermissionFTypeReadCodes {
		if code != actionPermissionSeedsRead[i].perms {
			return fmt.Errorf("migration 236 致命漂移: 第 %d 项码不匹配声明(%s) != 种子(%s)",
				i, code, actionPermissionSeedsRead[i].perms)
		}
	}

	for _, seed := range actionPermissionSeedsRead {
		parents, err := resolveActionPermissionParents(db, seed)
		if err != nil {
			log.Printf("Migration 236: WARNING 查询父菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}
		if len(parents) == 0 {
			log.Printf("Migration 236: WARNING 未找到 live 父菜单(ops:floor:list), 跳过(perms=%s)", seed.perms)
			continue
		}

		menuID, err := ensureActionPermissionMenu(db, seed, parents[0].ID)
		if err != nil {
			log.Printf("Migration 236: WARNING 创建/获取按钮菜单失败(perms=%s): %v, 跳过", seed.perms, err)
			continue
		}

		parentIDs := make([]string, 0, len(parents))
		for _, p := range parents {
			parentIDs = append(parentIDs, p.ID)
		}
		if err := grantActionPermissionToParentHolders(db, menuID, parentIDs); err != nil {
			log.Printf("Migration 236: WARNING 授权按钮菜单失败(perms=%s, menuID=%s): %v, 跳过授权",
				seed.perms, menuID, err)
			continue
		}
	}

	log.Println("Migration 236 completed: OQ-7 read permission codes seeded")
	return nil
}
