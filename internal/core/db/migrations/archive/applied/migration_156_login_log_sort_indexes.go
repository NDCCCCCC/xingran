//go:build archive_skip


package migrations

import (
	"fmt"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate156LoginLogSortIndexes 为登录日志表创建排序字段索引
//
// 用途:Phase B 全部分页排序基建。sys_logininfor 列表按 login_time DESC 排序为高频操作,
// 缺失索引会全表扫描。
func Migrate156LoginLogSortIndexes(db *gorm.DB) error {
	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] sys_logininfor 排序索引跳过创建（非PostgreSQL数据库）")
		return nil
	}

	applogger.Infof("[迁移] 开始创建 sys_logininfor 服务端排序索引")

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_sys_logininfor_login_time_desc
		    ON sys_logininfor (login_time DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_sys_logininfor_status_time
		    ON sys_logininfor (status, login_time DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_sys_logininfor_user_name_time
		    ON sys_logininfor (user_name, login_time DESC);`,
	}

	for _, sql := range indexes {
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("创建 sys_logininfor 排序索引失败: %w (sql=%s)", err, sql)
		}
	}

	applogger.Infof("[迁移] sys_logininfor 服务端排序索引已创建（3 个）")
	return nil
}
