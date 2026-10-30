//go:build archive_skip


package migrations

import (
	"fmt"
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate191DropInfoPointsRedundantColumns 删除 ops_info_points 的 4 个冗余物理列。
//
// 背景 (2026-07-01, 冗余字段验证):
//   - building_id / building_name / floor_name: 建表 SQL(032) 从未定义,
//     是 GORM AutoMigrate 基于 struct tag 自动 ADD 出来的"幽灵列";
//     populateRedundantFields 从不写入它们(永远 NULL), List 查询靠
//     LEFT JOIN ops_buildings/ops_floors 动态计算别名覆盖。
//   - workstation_name: populateRedundantFields 会填充, 但 List 查询同样用
//     LEFT JOIN sys_workstation.workstation_name 覆盖, 物理列冗余。
//
// 配套改动:
//   - OpsInfoPoint model 上述 4 字段 tag 改 gorm:"->;-:migration" (只读+忽略迁移),
//     保留 JSON 序列化 + JOIN scan 容器, 不再建/写物理列。
//   - InfoPointService.GetByID 补同款 JOIN, 单条详情靠 JOIN 取工位/楼层/楼宇名。
//
// 安全性:
//   - building_id/building_name/floor_name 在生产数据中全为 NULL;
//     workstation_name 的值可由 LEFT JOIN sys_workstation 完全还原 → 删除无数据损失。
//   - DROP COLUMN IF EXISTS 幂等, 重复执行安全。
//   - 非 PostgreSQL 方言跳过。
//
// 回滚 (需人工执行; 重新加列后由 AutoMigrate 维护):
//   ALTER TABLE ops_info_points ADD COLUMN IF NOT EXISTS building_id varchar(64);
//   ALTER TABLE ops_info_points ADD COLUMN IF NOT EXISTS building_name varchar(100);
//   ALTER TABLE ops_info_points ADD COLUMN IF NOT EXISTS floor_name varchar(100);
//   ALTER TABLE ops_info_points ADD COLUMN IF NOT EXISTS workstation_name varchar(100);
//   (加列后值为空; workstation_name 如需回填, 参见 populateRedundantFields 逻辑)
func Migrate191DropInfoPointsRedundantColumns(db *gorm.DB) error {
	log.Println("Running migration 190: 删除 ops_info_points 冗余列 (building_id/building_name/floor_name/workstation_name)")

	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] 190 跳过(非 PostgreSQL)")
		log.Println("Migration 190 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// 列名来自代码内常量白名单, 无注入风险, 可直接拼接。
	columns := []string{"building_id", "building_name", "floor_name", "workstation_name"}
	for _, col := range columns {
		var exists bool
		if err := db.Raw(
			`SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'ops_info_points' AND column_name = ?)`,
			col,
		).Scan(&exists).Error; err != nil {
			return fmt.Errorf("检查 ops_info_points.%s 是否存在失败: %w", col, err)
		}
		if !exists {
			applogger.Infof("[迁移] 190 ops_info_points.%s 不存在, 跳过", col)
			continue
		}
		if err := db.Exec(fmt.Sprintf("ALTER TABLE ops_info_points DROP COLUMN %s", col)).Error; err != nil {
			return fmt.Errorf("DROP ops_info_points.%s 失败: %w", col, err)
		}
		applogger.Infof("[迁移] 190 已删除 ops_info_points.%s", col)
	}

	log.Println("Migration 190 completed")
	return nil
}
