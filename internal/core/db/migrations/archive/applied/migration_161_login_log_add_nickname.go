//go:build archive_skip


package migrations

import (
	"gorm.io/gorm"
)

// Migration161LoginLogAddNickname 为 sys_logininfor 表添加 nickname 字段
// 支持登录日志显示为"nickname（username）"格式
func Migration161LoginLogAddNickname(db *gorm.DB) error {
	sql := `
ALTER TABLE sys_logininfor
ADD COLUMN IF NOT EXISTS nickname VARCHAR(50);

-- 为 nickname 字段创建索引,提升查询性能
CREATE INDEX IF NOT EXISTS idx_sys_logininfor_nickname
ON sys_logininfor(nickname);

-- 创建复合索引,支持按 nickname 和 user_name 查询
CREATE INDEX IF NOT EXISTS idx_sys_logininfor_nickname_name
ON sys_logininfor(nickname, user_name);
`
	return db.Exec(sql).Error
}
