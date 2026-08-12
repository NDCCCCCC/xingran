//go:build archive_skip


package migrations

import (
	"fmt"
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate198CreateFixSuggestionTable Phase 46 R5: 资产对账半自动修复建议表
//
// 业务背景:
//   - D-B1: 独立表 sys_reconciliation_fix_suggestion,与 sys_data_reconciliation 1:N
//   - D-B2: 6 状态状态机(pending/accepted/rejected/applied/rolled_back/failed)
//   - D-B3: 1 对多版本化(exception_id 索引 FK,不设唯一约束)
//   - D-B4: partial unique index uniq_fix_suggestion_pending_per_exception
//     (由 migration_199 独立建立,本迁移只建主表 + 普通索引 + CHECK)
//
// 步骤清单:
//   1. GORM AutoMigrate 创建 sys_reconciliation_fix_suggestion 主表
//      (GORM 从 model struct tags 自动建 2 个索引:
//       idx_fix_suggestion_exception / idx_fix_suggestion_status)
//   2. 手动建 2 个补充索引(W-1 修订:仅 status_created + applied_at,
//      GORM 不会再建 _exception / _status 重复):
//      - idx_fix_suggestion_status_created on (fix_status, created_at)
//        用于 7d 窗口 stats 过滤
//      - idx_fix_suggestion_applied_at on (applied_at)
//        用于 stats applied 计数(W-2 修订)
//   3. 2 个 CHECK 约束(PG-only,SQLite 跳过):
//      - chk_fix_suggestion_status:fix_status IN (6 状态)
//      - chk_fix_suggestion_conflict_type:conflict_type IN ('A','B','C','D','E','F')
//
// 重要 W-1 修订说明:
//   - 早期设计是建 4 个补充索引,GORM 再建 2 个 = 6 个索引会重复
//   - 现仅手动建 2 个补充索引,GORM 自动建 2 个,共 4 个索引
//   - 不建 idx_fix_suggestion_rollback_window(stats 用 applied_at 过滤,逐条比
//     rollback_window_until 不需要索引)
func Migrate198CreateFixSuggestionTable(db *gorm.DB) error {
	log.Println("Running migration 198: Phase 46 R5 sys_reconciliation_fix_suggestion table + 2 supplementary indexes + 2 CHECK constraints")

	// 1. GORM AutoMigrate 主表
	if err := db.AutoMigrate(&models.SysReconciliationFixSuggestion{}); err != nil {
		log.Printf("Migration 198: AutoMigrate sys_reconciliation_fix_suggestion failed: %v", err)
		return fmt.Errorf("AutoMigrate sys_reconciliation_fix_suggestion 失败: %w", err)
	}
	log.Println("Migration 198: sys_reconciliation_fix_suggestion 已创建/已存在(GORM 自动建 idx_fix_suggestion_exception / idx_fix_suggestion_status)")

	// 2. 手动建 2 个补充索引(W-1 修订:仅 2 个)
	statusCreatedIdxSQL := `
CREATE INDEX IF NOT EXISTS idx_fix_suggestion_status_created
    ON sys_reconciliation_fix_suggestion (fix_status, created_at);
`
	if err := db.Exec(statusCreatedIdxSQL).Error; err != nil {
		return fmt.Errorf("创建 idx_fix_suggestion_status_created 索引失败: %w", err)
	}
	applogger.Infof("[迁移] idx_fix_suggestion_status_created 补充索引已就位")

	appliedAtIdxSQL := `
CREATE INDEX IF NOT EXISTS idx_fix_suggestion_applied_at
    ON sys_reconciliation_fix_suggestion (applied_at);
`
	if err := db.Exec(appliedAtIdxSQL).Error; err != nil {
		return fmt.Errorf("创建 idx_fix_suggestion_applied_at 索引失败: %w", err)
	}
	applogger.Infof("[迁移] idx_fix_suggestion_applied_at 补充索引已就位(W-2 修订:用于 applied 计数)")

	// 3. CHECK 约束(PG-only,SQLite 跳过;用 isPostgreSQL 守卫)
	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] 修复建议表 CHECK 约束 跳过(非 PostgreSQL 数据库)")
		log.Println("Migration 198 completed (non-PostgreSQL dialect: table + indexes only)")
		return nil
	}

	// chk_fix_suggestion_status
	statusCheckSQL := `
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_fix_suggestion_status'
    ) THEN
        EXECUTE 'ALTER TABLE sys_reconciliation_fix_suggestion
                 ADD CONSTRAINT chk_fix_suggestion_status
                 CHECK (fix_status IN (''pending'',''accepted'',''rejected'',''applied'',''rolled_back'',''failed''))';
    END IF;
END$$;
`
	if err := db.Exec(statusCheckSQL).Error; err != nil {
		return fmt.Errorf("创建 chk_fix_suggestion_status CHECK 约束失败: %w", err)
	}
	applogger.Infof("[迁移] chk_fix_suggestion_status CHECK 约束已就位")

	// chk_fix_suggestion_conflict_type
	conflictTypeCheckSQL := `
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_fix_suggestion_conflict_type'
    ) THEN
        EXECUTE 'ALTER TABLE sys_reconciliation_fix_suggestion
                 ADD CONSTRAINT chk_fix_suggestion_conflict_type
                 CHECK (conflict_type IN (''A'',''B'',''C'',''D'',''E'',''F''))';
    END IF;
END$$;
`
	if err := db.Exec(conflictTypeCheckSQL).Error; err != nil {
		return fmt.Errorf("创建 chk_fix_suggestion_conflict_type CHECK 约束失败: %w", err)
	}
	applogger.Infof("[迁移] chk_fix_suggestion_conflict_type CHECK 约束已就位")

	log.Println("Migration 198 completed: sys_reconciliation_fix_suggestion table + 2 supplementary indexes + 2 CHECK constraints ready")
	return nil
}
