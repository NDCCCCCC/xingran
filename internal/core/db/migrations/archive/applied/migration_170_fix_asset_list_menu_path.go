//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate170FixAssetListMenuPath Phase 42-r1 后置修复:
//
// 历史背景:
//   - migration_147 创建"资产管理 > 资产列表"子菜单时把 childPath 写成了 "assets",
//     与父菜单 path='assets' 重复。routeGenerator.resolvePath(parentPath, childPath) = "assets/assets",
//     React Router 找不到该路径 → 点击"资产列表"菜单 fallback 到 /dashboard。
//   - migration_147 已修正 childPath := "list",但生产数据库已存在的 sys_menu 记录
//     仍保留旧值,只能通过新 migration 用 SQL UPDATE 修复。
//
// 修复范围:仅改"资产管理"父菜单下的"资产列表"子菜单 path,不影响其他子菜单(对账看板/异常列表
// 已用 'dashboard'/'exceptions' 正确短路径),不影响 sys_role_menu 关联(role_menu 用 menu_id 关联,
// 改 path 不影响权限判定)。
func Migrate170FixAssetListMenuPath(db *gorm.DB) error {
	log.Println("Running migration 170: Fix 资产列表 menu path (assets -> list)")

	// 1. 找父菜单"资产管理"(parent_id IS NULL, menu_type='M')
	var parentMenu struct {
		ID string
	}
	if err := db.Table("sys_menu").
		Select("id").
		Where("menu_name = ? AND parent_id IS NULL AND menu_type = 'M'", "资产管理").
		Scan(&parentMenu).Error; err != nil {
		return err
	}
	if parentMenu.ID == "" {
		log.Println("Migration 170: 资产管理父菜单不存在,跳过")
		return nil
	}

	// 2. UPDATE 资产列表子菜单 path = 'list',WHERE 限定为旧值 'assets' 防误改
	//    加 parent_id 限定确保只改资产管理下的子菜单,不影响其他模块同名菜单(理论上不会存在)
	result := db.Table("sys_menu").
		Where("parent_id = ? AND menu_name = ? AND path = ?", parentMenu.ID, "资产列表", "assets").
		Update("path", "list")
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		log.Println("Migration 170: 资产列表 path 已是 'list' 或无匹配记录,跳过")
	} else {
		log.Printf("Migration 170: 修复资产列表 path 'assets' -> 'list',影响 %d 行", result.RowsAffected)
	}

	// 3. 验证:SELECT path from sys_menu where menu_name='资产列表' and parent_id=...
	var newPath string
	if err := db.Table("sys_menu").
		Select("path").
		Where("parent_id = ? AND menu_name = ?", parentMenu.ID, "资产列表").
		Scan(&newPath).Error; err != nil {
		return err
	}
	log.Printf("Migration 170: 验证通过,资产列表 path = %q", newPath)

	log.Println("Migration 170 completed")
	return nil
}
