package migrations

import (
	"fmt"

	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate158WorkstationSortIndexes 为工位表创建排序字段索引
//
// 用途:Phase B 全部分页排序基建。ops_workstation 列表高频筛选维度:
// 部门、楼宇、状态。order_num 字段为工位在所属楼宇内的排序号。
func Migrate158WorkstationSortIndexes(db *gorm.DB) error {
	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] ops_workstation 排序索引跳过创建（非PostgreSQL数据库）")
		return nil
	}

	applogger.Infof("[迁移] 开始创建 ops_workstation 服务端排序索引")

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_ops_workstation_dept_status
		    ON ops_workstation (dept_id, status);`,
		`CREATE INDEX IF NOT EXISTS idx_ops_workstation_building_status
		    ON ops_workstation (building_id, status);`,
		`CREATE INDEX IF NOT EXISTS idx_ops_workstation_created_at_desc
		    ON ops_workstation (created_at DESC);`,
	}

	for _, sql := range indexes {
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("创建 ops_workstation 排序索引失败: %w (sql=%s)", err, sql)
		}
	}

	applogger.Infof("[迁移] ops_workstation 服务端排序索引已创建（3 个）")
	return nil
}
