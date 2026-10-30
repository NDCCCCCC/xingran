//go:build archive_skip


package migrations

import (
	"fmt"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate151MACPerfIndexes 为 MAC 历史表创建复合 B-tree 索引
//
// Phase 15 PERF-01 (D-07/D-08 锁定):
//   - 复合索引列顺序 (device_id, mac_address, first_seen)
//   - 高基数列在前, first_seen 末位支持时间范围 + ORDER BY
//   - 与 Phase 12 BRIN 索引互补 (BRIN 走时间扫描, 复合索引走点查 + 排序)
//   - 仅在 PostgreSQL 中执行 (SQLite 跳过)
func Migrate151MACPerfIndexes(db *gorm.DB) error {
	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] 复合索引跳过创建（非PostgreSQL数据库）")
		return nil
	}

	applogger.Infof("[迁移] 开始创建 MAC 历史复合 B-tree 索引 (device_id, mac_address, first_seen)")

	sql := `
CREATE INDEX IF NOT EXISTS idx_mac_history_device_mac_first_seen
ON sys_device_mac_history (device_id, mac_address, first_seen);
`
	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("创建复合索引 idx_mac_history_device_mac_first_seen 失败: %w", err)
	}

	applogger.Infof("[迁移] 复合索引 idx_mac_history_device_mac_first_seen 已创建")
	return nil
}
