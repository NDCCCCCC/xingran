package migrations

import (
	"log"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// Migrate232PortWriteAuditBatchID —— v1.51 收尾：sys_port_write_audit 加
// batch_id 列（TEXT 可空），Phase 122 断线对账从 deviceId+pageSize 近似升级
// 为 batch 级精确对账。
//
// 历史行 batch_id 为 NULL：audit/list 的 batchId 过滤为等值匹配，NULL 行天然
// 不命中，无需回填（历史批次本就无法重建 batchId）。
//
// 幂等：HasColumn 探测，已存在跳过；失败非阻断（调用方 WARN，下次启动重试）。
func Migrate232PortWriteAuditBatchID(db *gorm.DB) error {
	log.Println("Running migration 232: sys_port_write_audit batch_id 列")

	if db.Migrator().HasColumn(&models.PortWriteAudit{}, "batch_id") {
		log.Println("Migration 232: batch_id 列已存在，跳过")
		return nil
	}
	if err := db.Migrator().AddColumn(&models.PortWriteAudit{}, "BatchID"); err != nil {
		return err
	}
	log.Println("Migration 232: batch_id 列就绪")
	return nil
}
