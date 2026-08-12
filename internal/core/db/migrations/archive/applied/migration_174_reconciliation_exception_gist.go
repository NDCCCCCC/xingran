//go:build archive_skip


package migrations

import (
	"fmt"
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate174ReconciliationExceptionGist Phase 44 R3: sys_reconciliation_exception
//
// 业务背景:
//   - migration_168 (Phase 42 R1) 已建表 sys_reconciliation_exception，但 R1 只 seed
//     全局规则，未配置 GiST 索引 + CHECK 约束。
//   - R3 引入 CIDR 段级别例外规则引擎 + 命中测试工具，需要：
//       (1) GiST inet_ops partial index 加速命中测试 SQL `ip_range >> $1::inet`（D-R3-A1-03）
//       (2) CHECK 约束确保 exception_actions ⊆ 5 白名单值 + severity_override 仅 low/medium/high
//
// 关键约束（Pitfall 1/6 + MEMORY.md `xingran-gorm-sql-constraint-naming-conflict`）：
//   - 纯 SQL `DO $$ ... END $$`，禁止 GORM `check:` tag（GORM 命名不可控，且 v1.30.5 不稳定）
//   - `CREATE INDEX` 用 `IF NOT EXISTS`（DO$$ 包装）保证幂等
//   - model `IPRange` 字段保持 `gorm:"type:cidr;not null;column:ip_range"`，**不**加 `gorm:"index"`
//     （防 AutoMigrate 误建 btree 与 GiST 冲突）
//
// Status Convention（CLAUDE.md）: 0=启用, 1=停用
//   - partial index WHERE is_active = 0 AND deleted_at IS NULL —— 仅索引启用规则
//   - 命中测试工具默认仅查启用规则，与 partial index 对齐
//
// 参考：
//   - migration_168:139-156 DO$$ partial unique index 模式
//   - 44-RESEARCH.md §Code Examples :644-734 verbatim
//   - 44-CONTEXT.md D-R3-A1-03（GiST 索引留给命中测试单点查询）
//   - Pitfall 8：severity_override 白名单不含 critical（override 是降级语义）
func Migrate174ReconciliationExceptionGist(db *gorm.DB) error {
	log.Println("Running migration 174: Phase 44 R3 sys_reconciliation_exception GiST inet_ops 索引 + CHECK 约束")

	// 仅 PostgreSQL 执行（SQLite 不支持 GiST / inet_ops / cidr）
	if !isPostgreSQL(db) {
		applogger.Infof("[迁移 174] GiST/CHECK 跳过(非 PostgreSQL 数据库)")
		log.Println("Migration 174 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// 1. GiST inet_ops partial index：仅索引启用规则（is_active=0），加速 `ip_range >> ?::inet`
	//    D-R3-A1-03 + Claude's Discretion: WHERE is_active = 0 AND deleted_at IS NULL
	//    PG 9.4+ 内置 inet_ops opclass，无需 CREATE EXTENSION
	gistIdxSQL := `
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'idx_recon_exc_active_range'
          AND schemaname = 'public'
    ) THEN
        EXECUTE 'CREATE INDEX idx_recon_exc_active_range
                 ON sys_reconciliation_exception USING gist (ip_range inet_ops)
                 WHERE is_active = 0 AND deleted_at IS NULL';
    END IF;
END$$;
`
	if err := db.Exec(gistIdxSQL).Error; err != nil {
		return fmt.Errorf("创建 GiST inet_ops 索引 idx_recon_exc_active_range 失败: %w", err)
	}
	applogger.Infof("[迁移 174] GiST inet_ops 索引 idx_recon_exc_active_range 已就位")

	// 2. CHECK chk_recon_exc_actions：exception_actions 必须是 5 白名单值的子集
	//    PG 数组操作符 <@ 表示"是...的子集"
	chkActionsSQL := `
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_recon_exc_actions'
    ) THEN
        EXECUTE 'ALTER TABLE sys_reconciliation_exception
                 ADD CONSTRAINT chk_recon_exc_actions CHECK (
                     exception_actions <@ ARRAY[''no_alert'',''no_notice'',''no_workorder'',''skip_severity'',''silence'']
                 )';
    END IF;
END$$;
`
	if err := db.Exec(chkActionsSQL).Error; err != nil {
		return fmt.Errorf("创建 CHECK 约束 chk_recon_exc_actions 失败: %w", err)
	}
	applogger.Infof("[迁移 174] CHECK 约束 chk_recon_exc_actions 已就位（5 actions 白名单）")

	// 3. CHECK chk_recon_exc_severity_override：severity_override 仅允许 low/medium/high
	//    **不含 critical**（Pitfall 8 — override 是降级语义，不能升到 critical）
	chkSevSQL := `
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_recon_exc_severity_override'
    ) THEN
        EXECUTE 'ALTER TABLE sys_reconciliation_exception
                 ADD CONSTRAINT chk_recon_exc_severity_override CHECK (
                     severity_override IS NULL OR severity_override IN (''low'',''medium'',''high'')
                 )';
    END IF;
END$$;
`
	if err := db.Exec(chkSevSQL).Error; err != nil {
		return fmt.Errorf("创建 CHECK 约束 chk_recon_exc_severity_override 失败: %w", err)
	}
	applogger.Infof("[迁移 174] CHECK 约束 chk_recon_exc_severity_override 已就位（low/medium/high，不含 critical）")

	log.Println("Migration 174 completed: sys_reconciliation_exception GiST inet_ops + CHECK 约束就位")
	return nil
}
