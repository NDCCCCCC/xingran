//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate167WorkstationBackfillStatus
// 修复历史数据:user_id 有值但 status 仍为 Available(0) 的工位
//
// 触发条件:
//   - user_id IS NOT NULL AND user_id != ''
//   - status = 0 (空闲,与 user_id 不一致)
//   - deleted_at IS NULL (仅修复活数据)
//   - **Maintain(2) 不修改**,保留维护语义
//
// 修复后: status = 1 (占用),updated_at = NOW()
//
// 背景 (见 .planning/quick/260626-n71-workstation-status-occupied-from-user/):
// 本次 n71 引入 user_id↔status 联动规则,服务端 Save 前会自动覆盖 status,
// 但导入前已存在的不一致历史数据需要一次性回填。Service 层 helper 不影响
// 已有数据(只在写入路径生效)。
func Migrate167WorkstationBackfillStatus(db *gorm.DB) error {
	log.Println("Running migration 167: workstation backfill status from user_id")

	result := db.Exec(`
		UPDATE sys_workstation
		SET status = 1, updated_at = NOW()
		WHERE user_id IS NOT NULL
		  AND user_id != ''
		  AND status = 0
		  AND deleted_at IS NULL
	`)

	if result.Error != nil {
		log.Printf("Migration 167 failed: %v", result.Error)
		return result.Error
	}

	log.Printf("Migration 167: backfilled %d workstation row(s) status 0 → 1 (user_id 有值)", result.RowsAffected)
	log.Println("Migration 167 completed: workstation status backfill done")
	return nil
}