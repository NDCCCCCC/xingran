//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate164Phase38VerifyAdminMigrated Phase 38 (D-02 + D-03): 放宽 sys_ad_config 单管理员列约束 + 幂等补迁账号池。
//
// 1. Schema 放宽 (D-02): ALTER admin_username/admin_password DROP NOT NULL —— Phase 38 起新配置不再
//    写 admin 字段，DB 必须允许空值。GORM AutoMigrate 对既有列 NOT NULL→nullable 的放宽不可靠
//    （本项目既有 GORM 约束处理踩坑记录），此处显式 ALTER 兜底。
//    幂等：DROP NOT NULL 对已 nullable 的列为 no-op，每次启动安全执行。
//
// 2. 数据补迁 (D-03): 对每个启用 + 同步开启且仍持单管理员凭据的 AD 配置，若其账号池为空则补迁一条
//    （与 migration_162 同款 INSERT）。避免 Phase 38 移除单管理员路径后这些配置登录/同步失败。
//    幂等：先 count sys_ad_service_accounts，>0 则 skip。
//
// 仅含 ALTER(无索引/约束新增) + INSERT，不与 GORM uni_*_* 命名冲突（CLAUDE.md memory 警示）。
func Migrate164Phase38VerifyAdminMigrated(db *gorm.DB) error {
	log.Println("Running migration 164: Phase 38 relax admin columns + verify migrated to pool")

	// 1. 放宽 NOT NULL（幂等：已 nullable 时 no-op；SQLite 等不支持该语法的环境记日志跳过，不阻断）
	relaxSQLs := []string{
		`ALTER TABLE sys_ad_config ALTER COLUMN admin_username DROP NOT NULL`,
		`ALTER TABLE sys_ad_config ALTER COLUMN admin_password DROP NOT NULL`,
	}
	for _, sql := range relaxSQLs {
		if err := db.Exec(sql).Error; err != nil {
			log.Printf("Migration 164: relax column skipped (%v): %s", err, sql)
		}
	}

	// 2. 数据补迁：查找启用 + 同步开启且仍持单管理员凭据的配置
	type adConfigRow struct {
		ID            string
		ConfigName    string
		AdminUsername string
		AdminPassword string
	}
	var configs []adConfigRow
	if err := db.Raw(`
		SELECT id, config_name, admin_username, admin_password
		FROM sys_ad_config
		WHERE sync_enabled = true AND status = 0
			AND admin_username IS NOT NULL AND admin_password IS NOT NULL
			AND admin_username <> '' AND admin_password <> ''
	`).Scan(&configs).Error; err != nil {
		log.Printf("Migration 164: failed to query sys_ad_config: %v", err)
		return err
	}

	for _, cfg := range configs {
		// 幂等：该 config 在账号池已有账号则跳过
		var cnt int64
		if err := db.Table("sys_ad_service_accounts").
			Where("config_id = ? AND deleted_at IS NULL", cfg.ID).
			Count(&cnt).Error; err != nil {
			log.Printf("Migration 164: count accounts for config %s failed: %v", cfg.ID, err)
			continue
		}
		if cnt > 0 {
			log.Printf("Migration 164: config %s already has %d account(s), skip", cfg.ConfigName, cnt)
			continue
		}

		// admin_password 列保存的是 SM4 密文（sys_ad_config 历史加密），直接作为 password_ciphertext 入池
		insertSQL := `
INSERT INTO sys_ad_service_accounts (
    id, config_id, username, password_ciphertext,
    status, failure_count, remark,
    created_at, updated_at
) VALUES (
    gen_random_uuid(), ?, ?, ?,
    0, 0, 'Phase 38 补迁（sys_ad_config 单管理员）',
    NOW(), NOW()
)`
		if err := db.Exec(insertSQL, cfg.ID, cfg.AdminUsername, cfg.AdminPassword).Error; err != nil {
			log.Printf("Migration 164: insert account for config %s failed: %v", cfg.ID, err)
			continue
		}
		log.Printf("Migration 164: migrated admin account '%s' from sys_ad_config id=%s", cfg.AdminUsername, cfg.ID)
	}

	log.Println("Migration 164 completed: Phase 38 admin columns relaxed + pool verified")
	return nil
}
