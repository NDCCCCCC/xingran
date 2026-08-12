//go:build archive_skip


package migrations

import (
	"fmt"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate154UserSortIndexes 为 sys_user 表创建服务端排序字段索引
//
// 用途:Phase A 全部分页排序 (server-side sort) 基建。
// sys_user List 接口新增 user-driven ORDER BY 能力,
// 默认按 created_at DESC 排序在生产数据量下需索引支撑,
// 否则全表扫描 + 排序会拖慢列表响应。
//
// 索引策略:
//   - idx_sys_user_created_at_desc: 单列 created_at 索引,支持 ORDER BY created_at DESC
//   - idx_sys_user_status_created_at: 复合索引 (status, created_at DESC),
//     覆盖"按状态筛选 + 按创建时间排序"高频组合
//   - idx_sys_user_dept_created_at: 复合索引 (dept_id, created_at DESC),
//     覆盖"按部门筛选 + 按创建时间排序"高频组合
//
// 仅在 PostgreSQL 中执行 (SQLite 跳过);用 IF NOT EXISTS 保证幂等。
func Migrate154UserSortIndexes(db *gorm.DB) error {
	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] sys_user 排序索引跳过创建（非PostgreSQL数据库）")
		return nil
	}

	applogger.Infof("[迁移] 开始创建 sys_user 服务端排序索引")

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_sys_user_created_at_desc
		    ON sys_user (created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_sys_user_status_created_at
		    ON sys_user (status, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_sys_user_dept_created_at
		    ON sys_user (dept_id, created_at DESC);`,
	}

	for _, sql := range indexes {
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("创建 sys_user 排序索引失败: %w (sql=%s)", err, sql)
		}
	}

	applogger.Infof("[迁移] sys_user 服务端排序索引已创建（3 个）")
	return nil
}
