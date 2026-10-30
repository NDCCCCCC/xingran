package migrations

import (
	"fmt"
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate213RecomputeBuildingTotalFloors 一次性归正 ops_buildings.total_floors 存量漂移。
//
// 背景 (261013-lqb 之前的历史缺陷, 三条写路径让该反范式列漂移):
//   - 路径 A: floor Update 换楼全程不重算 total_floors (91-03 只修了换楼工位同步,漏了计数);
//   - 路径 B: 楼宇 Excel 导入直写用户上传的 totalFloors 值;
//   - 路径 C: 楼层 Excel 导入经通用 BatchUpsert 绕过 floorService.Create 的重算。
//
// 本迁移按 ops_floors 非软删行数重算所有未删楼栋, 使排序白名单
// (building_service.go totalFloors)、楼栋卡片/概览抽屉、Excel 导出消费到正确值。
//
// 幂等性: UPDATE 结果由实时 COUNT 决定, 重复执行结果一致。
// 无常驻对象: 一次性数据修复 SQL, 跑完即退出, 不产生 TRIGGER/视图等任何常驻
// schema object (禁 TRIGGER 是用户明确偏好, user-prefers-code-fixes-no-db-triggers)。
// 双方言: 纯 SQL 关联子查询 UPDATE, PG 与 sqlite 语法一致, 无需 isPostgreSQL 守卫。
func Migrate213RecomputeBuildingTotalFloors(db *gorm.DB) error {
	log.Println("Running migration 213: recompute ops_buildings.total_floors from live ops_floors count")

	const sql = `
UPDATE ops_buildings
SET total_floors = (
	SELECT COUNT(*)
	FROM ops_floors
	WHERE ops_floors.building_id = ops_buildings.id
		AND ops_floors.deleted_at IS NULL
)
WHERE deleted_at IS NULL`
	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("ops_buildings.total_floors 存量重算失败: %w", err)
	}

	applogger.Infof("[迁移] 213 ops_buildings.total_floors 存量漂移已按实时 COUNT 归正")
	log.Println("Migration 213 completed")
	return nil
}
