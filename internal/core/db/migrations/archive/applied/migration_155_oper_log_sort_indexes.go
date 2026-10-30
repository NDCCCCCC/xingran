//go:build archive_skip


package migrations

import (
	"fmt"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate155OperLogSortIndexes 为操作日志表创建排序字段索引
//
// 用途:Phase B 全部分页排序基建。sys_oper_log 列表按 oper_time DESC + status 筛选为高频组合,
// 缺失索引会全表扫描 + 排序(日志表行数常达百万级)。
//
// 索引:
//   - idx_sys_oper_log_oper_time_desc: 单列 oper_time DESC,支持默认排序
//   - idx_sys_oper_log_status_time: 复合 (status, oper_time DESC),覆盖"按状态+时间"组合
//   - idx_sys_oper_log_oper_name_time: 复合 (oper_name, oper_time DESC),覆盖"按操作人+时间"
//
// 仅 PostgreSQL;用 IF NOT EXISTS 保证幂等。
func Migrate155OperLogSortIndexes(db *gorm.DB) error {
	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] sys_oper_log 排序索引跳过创建（非PostgreSQL数据库）")
		return nil
	}

	applogger.Infof("[迁移] 开始创建 sys_oper_log 服务端排序索引")

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_sys_oper_log_oper_time_desc
		    ON sys_oper_log (oper_time DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_sys_oper_log_status_time
		    ON sys_oper_log (status, oper_time DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_sys_oper_log_oper_name_time
		    ON sys_oper_log (oper_name, oper_time DESC);`,
	}

	for _, sql := range indexes {
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("创建 sys_oper_log 排序索引失败: %w (sql=%s)", err, sql)
		}
	}

	applogger.Infof("[迁移] sys_oper_log 服务端排序索引已创建（3 个）")
	return nil
}
