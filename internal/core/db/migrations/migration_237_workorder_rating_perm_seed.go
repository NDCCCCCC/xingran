package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// ============================================================================
// FIX-06 / v1.53 Phase 136：工单评价创建权限码菜单种子。
//
// SetupWorkOrderRatingsWriteRouter 把「创建评价」写路由挂到严格单一权限点
// ops:workorder:rating:create（RequireStrictPermissions，无降级）。若 sys_menu
// 无承载该码的菜单行，除 admin 外所有角色该写端点必然 403（与 OQ-7 同型）。
//
// 复用 migration 226 的幂等 seed 机制：建 F 型按钮行，挂在 live 父 C 菜单
// ops:workorder:list（工单管理），精准授权给持有该父菜单的角色。
// 不改严格权限中间件语义（RBAC 零回归），只补数据。
// ============================================================================

// F_TYPE_SEED_BEGIN
var workOrderRatingFTypeSeedCodes = []string{
	"ops:workorder:rating:create",
}
// F_TYPE_SEED_END

var workOrderRatingSeeds = []actionPermissionSeed{
	{name: "工单评价", perms: "ops:workorder:rating:create", parentPerms: "ops:workorder:list", orderNum: 40},
}

// Migrate237WorkOrderRatingPermSeed 播种工单评价创建权限（双方言 / 幂等 / 非阻断）。
func Migrate237WorkOrderRatingPermSeed(db *gorm.DB) error {
	log.Println("Running migration 237: workorder rating create permission seed (FIX-06)")

	if len(workOrderRatingFTypeSeedCodes) != len(workOrderRatingSeeds) {
		return fmt.Errorf("migration 237 致命漂移: F_TYPE_SEED 声明数(%d) != 种子表数(%d)",
			len(workOrderRatingFTypeSeedCodes), len(workOrderRatingSeeds))
	}
	for i, code := range workOrderRatingFTypeSeedCodes {
		if code != workOrderRatingSeeds[i].perms {
			return fmt.Errorf("migration 237 致命漂移: 第 %d 项码不匹配声明(%s) != 种子(%s)",
				i, code, workOrderRatingSeeds[i].perms)
		}
	}

	for _, seed := range workOrderRatingSeeds {
		parents, err := resolveActionPermissionParents(db, seed)
		if err != nil {
			log.Printf("Migration 237: WARNING 查询父菜单失败(perms=%s): %v, 跳过", seed.perms, err)
			continue
		}
		if len(parents) == 0 {
			log.Printf("Migration 237: WARNING 未找到 live 父菜单(ops:workorder:list), 跳过(perms=%s)", seed.perms)
			continue
		}

		menuID, err := ensureActionPermissionMenu(db, seed, parents[0].ID)
		if err != nil {
			log.Printf("Migration 237: WARNING 创建/获取按钮菜单失败(perms=%s): %v, 跳过", seed.perms, err)
			continue
		}

		parentIDs := make([]string, 0, len(parents))
		for _, p := range parents {
			parentIDs = append(parentIDs, p.ID)
		}
		if err := grantActionPermissionToParentHolders(db, menuID, parentIDs); err != nil {
			log.Printf("Migration 237: WARNING 授权按钮菜单失败(perms=%s, menuID=%s): %v, 跳过授权",
				seed.perms, menuID, err)
			continue
		}
	}

	log.Println("Migration 237 completed: workorder rating create permission seeded")
	return nil
}
