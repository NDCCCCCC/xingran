package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============================================================================
// Migration 246 — 全库备份删除独立权限码 F 型播种（task-3 G4 整改）
//
// 背景（2026-11-11 granularity 审计 G4）：/system/db-backup/:id/delete 此前
// 复用 system:backup:export——持导出码即可删除备份文件，违反最小权限。
// 整改：新码 system:backup:remove（system:* 模块删除统一 remove 词汇，D-04），
// 端点接线见 internal/api/router.go dbBackupsWrites.Remove；
// 常量与端点映射见 pkg/permission/config.go；契约见 docs/API响应规范.md；
// 前端镜像与按钮门控见 constants/permissionCodes.ts systemBackup.remove 与
// pages/system/backup/components/BackupListTab.tsx 行内删除按钮。
//
// 父 C 型菜单 perms = system:backup（categories.tsx 备份与恢复菜单本体码），
// 走 parentPerms 定位。复用 migration_226 全套 helper
// （resolveActionPermissionParents / ensureActionPermissionMenu /
// grantActionPermissionToParentHolders），双方言（PG / sqlite 通用）、幂等、
// 非阻断降级。
// ============================================================================

// backupRemovePermissionFTypeSeedCodes F 型声明块（供 check-permission-codes.mjs
// 的 F_TYPE 解析器识别；必须与 backupRemovePermissionSeeds 逐条 1:1 一致）。
var backupRemovePermissionFTypeSeedCodes = []string{
	"system:backup:remove",
}

// backupRemovePermissionSeeds 种子表（与上方声明块逐条一致）。
var backupRemovePermissionSeeds = []actionPermissionSeed{
	{name: "备份删除", perms: "system:backup:remove", parentPerms: "system:backup", orderNum: 4},
}

// Migrate246BackupRemovePermissionFTypeSeed 播种 system:backup:remove F 型按钮
// 并精准授权给持父菜单（system:backup）的角色。双方言、幂等、非阻断降级。
func Migrate246BackupRemovePermissionFTypeSeed(db *gorm.DB) error {
	log.Println("Running migration 246: system backup remove F type permission seed")

	// 0) 入口硬断言：声明块与种子表必须条数相同且逐条相等
	if len(backupRemovePermissionFTypeSeedCodes) != len(backupRemovePermissionSeeds) {
		return fmt.Errorf("migration 246 致命漂移: F_TYPE 块声明数(%d) != 种子表数(%d)",
			len(backupRemovePermissionFTypeSeedCodes), len(backupRemovePermissionSeeds))
	}
	for i, code := range backupRemovePermissionFTypeSeedCodes {
		if code != backupRemovePermissionSeeds[i].perms {
			return fmt.Errorf("migration 246 致命漂移: 第 %d 项码不匹配声明(%s) != 种子(%s)",
				i, code, backupRemovePermissionSeeds[i].perms)
		}
	}

	// 逐条处理种子（单条失败仅记录日志并继续，保证非阻断）
	for _, seed := range backupRemovePermissionSeeds {
		parents, err := resolveActionPermissionParents(db, seed)
		if err != nil {
			log.Printf("Migration 246: WARNING 查询父菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}
		if len(parents) == 0 {
			log.Printf("Migration 246: WARNING 未找到 live 父菜单(perms=%s), 跳过创建与授权", seed.perms)
			continue
		}

		menuID, err := ensureActionPermissionMenu(db, seed, parents[0].ID)
		if err != nil {
			log.Printf("Migration 246: WARNING 创建/获取按钮菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}

		parentIDs := make([]string, 0, len(parents))
		for _, p := range parents {
			parentIDs = append(parentIDs, p.ID)
		}
		if err := grantActionPermissionToParentHolders(db, menuID, parentIDs); err != nil {
			log.Printf("Migration 246: WARNING 授权按钮菜单失败(perms=%s, menuID=%s): %v, 跳过授权",
				seed.perms, menuID, err)
			continue
		}
	}

	log.Println("Migration 246 completed: system backup remove F type permission seeded")
	return nil
}

// 编译期引用校验：确保 models 包引用不因重构失效（菜单类型常量由复用 helper 使用）。
var _ = models.MenuTypeButton
