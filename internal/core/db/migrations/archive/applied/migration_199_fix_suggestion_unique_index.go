//go:build archive_skip


package migrations

import (
	"fmt"
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate199FixSuggestionUniqueIndex Phase 46 R5: 修复建议 partial unique index
//
// 业务背景:
//   - D-B3: 1 对多版本化 — 同一 exception_id 可有多条历史建议
//     (新建版本不删旧版本,旧版本 superseded_at 标记)
//
//   - D-B4: partial unique index uniq_fix_suggestion_pending_per_exception
//     阻止同 exception 同时存在多个 pending 状态建议。
//     用 partial unique (WHERE fix_status='pending' AND superseded_at IS NULL AND deleted_at IS NULL)
//     即可满足"同 exception 1 个 pending"约束,又不破坏 D-B3 的多版本化语义
//     (历史 rejected/applied/rolled_back 记录可保留)。
//
// 独立成 migration_199 文件的原因:
//   - 部分唯一索引在 AutoMigrate 阶段(本文件 198)不创建
//   - 若 198 失败后重跑,199 可单独执行(幂等 DO $$ 块)
//   - 与 migration_168 的 uniq_recon_asset_type_open 模式一致
//
// 命名严格遵循 uni_*_* 规范(参考 MEMORY `xingran-gorm-sql-constraint-naming-conflict`):
//   - GORM uniqueIndex 期望 `uni_*`
//   - SQL inline UNIQUE 会被 PG 自动命名为 `<table>_<col>_key`
//   - 显式命名 `uni_fix_suggestion_pending_per_exception` 避免冲突
//
// 仅 PG-only(SQLite 不支持部分唯一索引的 WHERE 子句)
func Migrate199FixSuggestionUniqueIndex(db *gorm.DB) error {
	log.Println("Running migration 199: Phase 46 R5 partial unique index uniq_fix_suggestion_pending_per_exception")

	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] partial unique index uniq_fix_suggestion_pending_per_exception 跳过(非 PostgreSQL 数据库)")
		log.Println("Migration 199 completed (non-PostgreSQL dialect: skipped)")
		return nil
	}

	partialUniqueSQL := `
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'uniq_fix_suggestion_pending_per_exception'
          AND schemaname = 'public'
    ) THEN
        EXECUTE 'CREATE UNIQUE INDEX uniq_fix_suggestion_pending_per_exception
                 ON sys_reconciliation_fix_suggestion (exception_id)
                 WHERE fix_status = ''pending'' AND superseded_at IS NULL AND deleted_at IS NULL';
    END IF;
END$$;
`
	if err := db.Exec(partialUniqueSQL).Error; err != nil {
		return fmt.Errorf("创建 partial unique index uniq_fix_suggestion_pending_per_exception 失败: %w", err)
	}
	applogger.Infof("[迁移] partial unique index uniq_fix_suggestion_pending_per_exception 已就位(D-B4 防同 exception 多 pending)")

	log.Println("Migration 199 completed: partial unique index uniq_fix_suggestion_pending_per_exception ready")
	return nil
}
