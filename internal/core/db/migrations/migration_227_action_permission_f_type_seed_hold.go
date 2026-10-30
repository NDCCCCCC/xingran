package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// ============================================================================
// Phase 87 批次二（OQ-6 用户确认）补播：
// 包含 7 个动作权限码，其父菜单经用户明确确认全部挂在「楼层管理」（perms='ops:floor:list'）
// ============================================================================

// F_TYPE_SEED_BEGIN
var actionPermissionFTypeSeedHoldCodes = []string{
	"ops:walls:add",
	"ops:walls:edit",
	"ops:walls:delete",
	"ops:doors:add",
	"ops:doors:edit",
	"ops:doors:delete",
	"ops:floor-plan:edit",
}
// F_TYPE_SEED_END

var actionPermissionSeedsHold = []actionPermissionSeed{
	{name: "墙体新增", perms: "ops:walls:add", parentPerms: "ops:floor:list", orderNum: 20},
	{name: "墙体修改", perms: "ops:walls:edit", parentPerms: "ops:floor:list", orderNum: 21},
	{name: "墙体删除", perms: "ops:walls:delete", parentPerms: "ops:floor:list", orderNum: 22},
	{name: "门新增", perms: "ops:doors:add", parentPerms: "ops:floor:list", orderNum: 23},
	{name: "门修改", perms: "ops:doors:edit", parentPerms: "ops:floor:list", orderNum: 24},
	{name: "门删除", perms: "ops:doors:delete", parentPerms: "ops:floor:list", orderNum: 25},
	{name: "平面图编辑", perms: "ops:floor-plan:edit", parentPerms: "ops:floor:list", orderNum: 26},
}

// Migrate227ActionPermissionFTypeSeedHold 补播第二批 7 条动作级权限（OQ-6 用户确认值）。
// 双方言通用、幂等、非阻断降级、visible 归零。
func Migrate227ActionPermissionFTypeSeedHold(db *gorm.DB) error {
	log.Println("Running migration 227: action-level F type permission seed (batch 2 hold)")

	// 0) 入口硬断言：声明块与种子表条数及逐条相等
	if len(actionPermissionFTypeSeedHoldCodes) != len(actionPermissionSeedsHold) {
		return fmt.Errorf("migration 227 致命漂移: F_TYPE_SEED 块声明数(%d) != 种子表数(%d)",
			len(actionPermissionFTypeSeedHoldCodes), len(actionPermissionSeedsHold))
	}
	for i, code := range actionPermissionFTypeSeedHoldCodes {
		if code != actionPermissionSeedsHold[i].perms {
			return fmt.Errorf("migration 227 致命漂移: 第 %d 项码不匹配声明(%s) != 种子(%s)",
				i, code, actionPermissionSeedsHold[i].perms)
		}
	}

	for _, seed := range actionPermissionSeedsHold {
		// 1) 解析 live 父 C 型菜单
		parents, err := resolveActionPermissionParents(db, seed)
		if err != nil {
			log.Printf("Migration 227: WARNING 查询父菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}
		if len(parents) == 0 {
			log.Printf("Migration 227: WARNING 未找到 live 父菜单(perms=%s), 跳过创建与授权", seed.perms)
			continue
		}

		// 2) 幂等建立 F 型按钮行
		menuID, err := ensureActionPermissionMenu(db, seed, parents[0].ID)
		if err != nil {
			log.Printf("Migration 227: WARNING 创建/获取按钮菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}

		// 3) 精准授权给持有任一父 C 菜单的角色
		parentIDs := make([]string, 0, len(parents))
		for _, p := range parents {
			parentIDs = append(parentIDs, p.ID)
		}
		if err := grantActionPermissionToParentHolders(db, menuID, parentIDs); err != nil {
			log.Printf("Migration 227: WARNING 授权按钮菜单失败(perms=%s, menuID=%s): %v, 跳过授权",
				seed.perms, menuID, err)
			continue
		}
	}

	log.Println("Migration 227 completed: action-level F type permissions seeded (batch 2 hold)")
	return nil
}
