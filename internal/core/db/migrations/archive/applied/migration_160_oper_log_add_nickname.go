//go:build archive_skip


package migrations

import (
	"gorm.io/gorm"
)

// Migration160OperLogAddNickname 为 sys_oper_log 表添加 nickname 字段
// 支持操作日志显示为"nickname（username）"格式
func Migration160OperLogAddNickname(db *gorm.DB) error {
	// 添加 nickname 字段到 sys_oper_log 表
	// 字段为可空，历史数据的 nickname 为 NULL
	sql := `
ALTER TABLE sys_oper_log
ADD COLUMN IF NOT EXISTS nickname VARCHAR(50);

-- 为 nickname 字段创建索引，提升查询性能
CREATE INDEX IF NOT EXISTS idx_sys_oper_log_nickname
ON sys_oper_log(nickname);

-- 创建复合索引，支持按 nickname 和 oper_name 查询
CREATE INDEX IF NOT EXISTS idx_sys_oper_log_nickname_name
ON sys_oper_log(nickname, oper_name);
`

	return db.Exec(sql).Error
}
