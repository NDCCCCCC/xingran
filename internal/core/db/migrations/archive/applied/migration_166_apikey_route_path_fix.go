//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate166ApikeyRoutePathFix Phase 40 (TECH-03 / apikey-route-path-duplication):
// 修正 sys_menu.path 字段重复拼接导致前端 URL 变成
// `/system/apikeys/system/apikeys` 的问题。
//
// 根因 (见 .planning/debug/apikey-route-path-duplication.md)：
//   - 父菜单 "API密钥管理" path=NULL（从前端菜单树推导为 "system"）
//   - 子菜单 "密钥列表" path='system/apikeys'（含父前缀，重复拼接）
//   - 子菜单 "使用日志" path='system/apikeys/logs'（同样问题）
//   - 前端 sidebar.utils buildFullPath 把父 path 与子 path 拼接，
//     生成 `system/system/apikeys` 或 `system/apikeys/system/apikeys`
//
// 本迁移用 menu_name 锁定具体菜单行（避免覆盖其他菜单合法 path），
// 把子菜单 path 改为相对父菜单的纯段：
//   - "密钥列表": system/apikeys        → list
//   - "使用日志": system/apikeys/logs   → logs
//
// 父菜单 "API密钥管理" 保持 path='apikeys'（让顶层目录直接挂 /system/apikeys）。
//
// 幂等：再次运行时 UPDATE 命中的还是相同的相对值，不会破坏已修复的数据。
func Migrate166ApikeyRoutePathFix(db *gorm.DB) error {
	log.Println("Running migration 166: apikey route path fix (TECH-03)")

	type pathFix struct {
		menuName string
		newPath  string
		oldPath  string // 仅用于日志对比
	}

	fixes := []pathFix{
		{"API密钥管理", "apikeys", "<null or stale>"},
		{"密钥列表", "list", "system/apikeys"},
		{"使用日志", "logs", "system/apikeys/logs"},
	}

	for _, f := range fixes {
		// 用 menu_name 锁定，避免误伤其他菜单
		result := db.Table("sys_menu").
			Where("menu_name = ?", f.menuName).
			Update("path", f.newPath)
		if result.Error != nil {
			log.Printf("Migration 166: update path for %s failed: %v", f.menuName, result.Error)
			continue
		}
		log.Printf("Migration 166: %s path → %q (rows affected=%d, was %q)",
			f.menuName, f.newPath, result.RowsAffected, f.oldPath)
	}

	log.Println("Migration 166 completed: apikey menu path normalized")
	return nil
}
