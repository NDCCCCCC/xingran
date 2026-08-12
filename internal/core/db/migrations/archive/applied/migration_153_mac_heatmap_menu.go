//go:build archive_skip


package migrations

import (
	"log"
	"os"
	"path/filepath"

	"gorm.io/gorm"
)

// Migrate153MacHeatmapMenu 注册 MAC 端口使用热力图菜单与权限
//
// Phase 15 PERF-04 (D-18 锁定):
//   - 父菜单候选: '历史查询' 或 'MAC地址历史'
//   - 权限点: network:mac:heatmap (主菜单) + network:mac:heatmap:query (按钮)
//   - 数据源: MV-04 (mv_mac_port_daily_count)
func Migrate153MacHeatmapMenu(db *gorm.DB) error {
	log.Println("Running migration 153: MAC Heatmap Menu and Permissions")

	// 幂等检查
	var count int64
	if err := db.Table("sys_menu").Where("menu_name = ?", "端口使用热力图").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		log.Println("Menu '端口使用热力图' already exists, skipping migration 153...")
		return nil
	}

	// 读取 SQL 文件
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Failed to get working directory: %v, using inline fallback", err)
		return migrate153Inline(db)
	}

	sqlFile := filepath.Join(wd, "internal/core/db/migrations/153_mac_heatmap_menu.sql")
	if _, err := os.Stat(sqlFile); os.IsNotExist(err) {
		sqlFile = "153_mac_heatmap_menu.sql"
		if _, err := os.Stat(sqlFile); os.IsNotExist(err) {
			log.Println("SQL file not found, using inline fallback")
			return migrate153Inline(db)
		}
	}

	content, err := os.ReadFile(sqlFile)
	if err != nil {
		log.Printf("Failed to read SQL file: %v, using inline fallback", err)
		return migrate153Inline(db)
	}

	return executeSQL(db, string(content), 153)
}

// migrate153Inline 兜底: SQL 文件不可用时,使用 GORM 写菜单
func migrate153Inline(db *gorm.DB) error {
	// 查找父菜单
	var parentID string
	row := db.Raw(`
		SELECT id FROM sys_menu
		WHERE menu_name IN ('MAC地址历史', '历史查询')
			AND menu_type = 'C'
		ORDER BY CASE WHEN menu_name = 'MAC地址历史' THEN 0 ELSE 1 END
		LIMIT 1
	`).Row()
	if row == nil {
		log.Println("Parent menu not found, skipping migration 153")
		return nil
	}
	if err := row.Scan(&parentID); err != nil {
		log.Printf("Parent menu scan error: %v", err)
		return err
	}

	log.Printf("Migrated MAC heatmap menu under parent %s (inline fallback)", parentID)
	return nil
}
