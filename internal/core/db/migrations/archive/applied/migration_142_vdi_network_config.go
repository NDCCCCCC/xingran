//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate142VDINetworkConfig 添加VDI虚拟机网络配置字段
// 修复绑定用户字段映射问题，添加网络配置信息
func Migrate142VDINetworkConfig(db *gorm.DB) error {
	log.Println("Running migration 142: VDI Network Config")

	// 使用原始SQL添加字段，避免类型冲突
	columns := []struct {
		name    string
		typeDef string
	}{
		{"ip_type", "VARCHAR(50)"},
		{"subnet_mask", "VARCHAR(50)"},
		{"default_gateway", "VARCHAR(50)"},
		{"name_server", "VARCHAR(100)"},
		{"assign_ip", "VARCHAR(50)"},
	}

	for _, col := range columns {
		// 检查列是否已存在
		var columnExists int64
		db.Raw(`
			SELECT count(*) FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'sys_vdi_vm' AND column_name = ?
		`, col.name).Scan(&columnExists)

		if columnExists > 0 {
			log.Printf("Column '%s' already exists in sys_vdi_vm, skipping...", col.name)
			continue
		}

		// 添加列
		if err := db.Exec(`ALTER TABLE sys_vdi_vm ADD COLUMN ` + col.name + ` ` + col.typeDef + ` DEFAULT ''`).Error; err != nil {
			log.Printf("Failed to add column '%s': %v", col.name, err)
			return err
		}
		log.Printf("Added column: %s", col.name)
	}

	// 修复绑定用户数据（将bound_user_id中的用户名移动到bound_user_name）
	db.Exec(`
		UPDATE sys_vdi_vm
		SET bound_user_name = bound_user_id,
		    bound_user_id = NULL
		WHERE bound_user_id IS NOT NULL
		  AND bound_user_id NOT LIKE 'user-%'
		  AND bound_user_id NOT LIKE '%-%-%-%'
	`)
	log.Println("Fixed bound_user field mapping")

	log.Println("Migration 142 completed successfully: VDI network config fields added")
	return nil
}
