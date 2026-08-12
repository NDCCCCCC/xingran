//go:build archive_skip


package migrations

import (
	"fmt"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate158WorkstationSortIndexes 为工位表创建排序字段索引
//
// 用途:Phase B 全部分页排序基建。sys_workstation 列表高频筛选维度:
// 部门、楼宇、状态。order_num 字段为工位在所属楼宇内的排序号。
func Migrate158WorkstationSortIndexes(db *gorm.DB) error {
	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] sys_workstation 排序索引跳过创建（非PostgreSQL数据库）")
		return nil
	}

	applogger.Infof("[迁移] 开始创建 sys_workstation 服务端排序索引")

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_sys_workstation_dept_status
		    ON sys_workstation (dept_id, status);`,
		`CREATE INDEX IF NOT EXISTS idx_sys_workstation_building_status
		    ON sys_workstation (building_id, status);`,
		`CREATE INDEX IF NOT EXISTS idx_sys_workstation_created_at_desc
		    ON sys_workstation (created_at DESC);`,
	}

	for _, sql := range indexes {
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("创建 sys_workstation 排序索引失败: %w (sql=%s)", err, sql)
		}
	}

	applogger.Infof("[迁移] sys_workstation 服务端排序索引已创建（3 个）")
	return nil
}
