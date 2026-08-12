//go:build archive_skip


package migrations

import (
	"fmt"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate157OpsBuildingSortIndexes 为楼宇表创建排序字段索引
//
// 用途:Phase B 全部分页排序基建。ops_buildings 列表默认按 order_num ASC,
// 常见组合为"按部门过滤 + 按 order_num 排序"和"按状态过滤 + 按 order_num 排序"。
func Migrate157OpsBuildingSortIndexes(db *gorm.DB) error {
	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] ops_buildings 排序索引跳过创建（非PostgreSQL数据库）")
		return nil
	}

	applogger.Infof("[迁移] 开始创建 ops_buildings 服务端排序索引")

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_ops_buildings_org_order
		    ON ops_buildings (org_id, order_num ASC);`,
		`CREATE INDEX IF NOT EXISTS idx_ops_buildings_status_order
		    ON ops_buildings (status, order_num ASC);`,
		`CREATE INDEX IF NOT EXISTS idx_ops_buildings_created_at_desc
		    ON ops_buildings (created_at DESC);`,
	}

	for _, sql := range indexes {
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("创建 ops_buildings 排序索引失败: %w (sql=%s)", err, sql)
		}
	}

	applogger.Infof("[迁移] ops_buildings 服务端排序索引已创建（3 个）")
	return nil
}
