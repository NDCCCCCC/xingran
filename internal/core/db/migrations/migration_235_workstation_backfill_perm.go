package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate235WorkstationBackfillPerm —— seed 工位「按人员反推部门」独立按钮权限。
//
// 背景（quick-261106-wsb 用户决策变更）：backfill-dept 原复用 ops:workstation:edit，
// 用户裁定批量反推影响面为全库，不应与单工位编辑共用权限 → 独立码
// ops:workstation:backfill，可按角色独立授权/回收。
//
// 复用 migration 226 的幂等 seed 机制（ensureActionPermissionMenu 按 perms
// 查既有行复用；grantActionPermissionToParentHolders NOT EXISTS 幂等授权）：
//   - 建菜单 F 型按钮行「工位部门反推」，挂在工位管理 C 菜单（perms=
//     ops:workstation:list）下；
//   - bootstrap 授权给持有工位菜单的角色（对齐 226 的工位导入/导出既有做法），
//     上线后可在角色管理里按需回收。
//
// 幂等：重复执行复用既有按钮行，不重复授权。
func Migrate235WorkstationBackfillPerm(db *gorm.DB) error {
	log.Println("Running migration 235: ops:workstation:backfill 按钮权限 seed")

	seed := actionPermissionSeed{
		name:        "工位部门反推",
		perms:       "ops:workstation:backfill",
		parentPerms: "ops:workstation:list",
		orderNum:    11,
	}

	parents, err := resolveActionPermissionParents(db, seed)
	if err != nil {
		log.Printf("Migration 235: WARNING 查询父菜单失败: %v, 跳过", err)
		return nil
	}
	if len(parents) == 0 {
		log.Printf("Migration 235: WARNING 未找到 live 父菜单(ops:workstation:list), 跳过创建与授权")
		return nil
	}

	menuID, err := ensureActionPermissionMenu(db, seed, parents[0].ID)
	if err != nil {
		log.Printf("Migration 235: WARNING 创建/获取按钮菜单失败: %v, 跳过授权", err)
		return nil
	}

	parentIDs := make([]string, 0, len(parents))
	for _, p := range parents {
		parentIDs = append(parentIDs, p.ID)
	}
	if err := grantActionPermissionToParentHolders(db, menuID, parentIDs); err != nil {
		log.Printf("Migration 235: WARNING 授权按钮菜单失败: %v", err)
		return nil
	}

	log.Println("Migration 235 completed: ops:workstation:backfill seeded")
	return nil
}
