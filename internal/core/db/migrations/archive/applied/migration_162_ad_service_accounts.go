//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate162ADServiceAccounts Phase 36: AD 域控服务账号池 + 自动故障切换
//
// 创建 sys_ad_service_accounts 表，将 sys_ad_config 的 admin_username/admin_password
// 迁移到新表（status=0, available），实现多账号并发可用 + 自动故障切换。
//
// 字段命名 `password_ciphertext`（而非 `encrypted_password`）以命中 operlog 现有敏感关键词
// `password`，自动触发 `******` 脱敏，避免扩展 OPERLOG-03 锁定的敏感关键词列表。
//
// sys_ad_config 的 admin_username/admin_password 字段保留并标 @Deprecated，
// v1.16 双读兼容期；v1.17 由独立清理迁移移除。
func Migrate162ADServiceAccounts(db *gorm.DB) error {
	log.Println("Running migration 162: AD service accounts pool + failover")

	// 1. CREATE TABLE
	// 幂等：IF NOT EXISTS 防止重复执行报错
	createTableSQL := `
CREATE TABLE IF NOT EXISTS sys_ad_service_accounts (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id               UUID NOT NULL REFERENCES sys_ad_config(id),
    username                VARCHAR(255) NOT NULL,
    password_ciphertext     TEXT NOT NULL,
    status                  INT NOT NULL DEFAULT 0,
    failure_count           INT NOT NULL DEFAULT 0,
    circuit_breaker_until   TIMESTAMP,
    last_success_at         TIMESTAMP,
    last_failure_at         TIMESTAMP,
    last_failure_reason     TEXT,
    manual_unlock_reason    TEXT,
    manual_unlocked_by      VARCHAR(64),
    manual_unlocked_at      TIMESTAMP,
    remark                  VARCHAR(500),
    created_at              TIMESTAMP,
    updated_at              TIMESTAMP,
    deleted_at              TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ad_acct_active
    ON sys_ad_service_accounts(config_id, status)
    WHERE deleted_at IS NULL;
`
	if err := db.Exec(createTableSQL).Error; err != nil {
		log.Printf("Migration 162: failed to create sys_ad_service_accounts table: %v", err)
		return err
	}
	log.Println("Migration 162: sys_ad_service_accounts table created (or already exists)")

	// 2. 数据迁移：从 sys_ad_config 拷贝第一行 admin 账号到新表
	// 幂等：先查现有账号数，为 0 时才迁移，避免重复插入
	var existingCount int64
	if err := db.Model(&struct{}{}).Table("sys_ad_service_accounts").
		Where("deleted_at IS NULL").
		Count(&existingCount).Error; err != nil {
		log.Printf("Migration 162: failed to count existing accounts: %v", err)
		return err
	}

	if existingCount > 0 {
		log.Printf("Migration 162: %d accounts already exist, skipping data migration", existingCount)
		return nil
	}

	// 查找第一个启用的 AD 配置（sync_enabled=true AND status=0）
	type adConfigRow struct {
		ID               string
		AdminUsername    string
		AdminPassword    string
	}
	var configRow adConfigRow
	err := db.Raw(`
		SELECT id, admin_username, admin_password
		FROM sys_ad_config
		WHERE sync_enabled = true AND status = 0
			AND admin_username IS NOT NULL
			AND admin_password IS NOT NULL
		ORDER BY created_at ASC
		LIMIT 1
	`).Scan(&configRow).Error

	if err != nil {
		log.Printf("Migration 162: failed to query sys_ad_config: %v", err)
		return err
	}

	if configRow.ID == "" {
		log.Println("Migration 162: no active AD config found, skipping data migration (pool starts empty)")
		return nil
	}

	// 插入新表（status=0 表示可用）
	insertSQL := `
INSERT INTO sys_ad_service_accounts (
    id, config_id, username, password_ciphertext,
    status, failure_count, remark,
    created_at, updated_at
) VALUES (
    gen_random_uuid(), ?, ?, ?,
    0, 0, '从 sys_ad_config 迁移（v1.16 兼容期）',
    NOW(), NOW()
)
`
	if err := db.Exec(insertSQL,
		configRow.ID,
		configRow.AdminUsername,
		configRow.AdminPassword,
	).Error; err != nil {
		log.Printf("Migration 162: failed to insert migrated account: %v", err)
		return err
	}

	log.Printf("Migration 162: migrated admin account '%s' from sys_ad_config id=%s",
		configRow.AdminUsername, configRow.ID)
	log.Println("Migration 162 completed: AD service accounts pool ready")
	return nil
}