//go:build archive_skip


package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// Migrate192BackfillOpsFloorsBuildingId 反推回填 ops_floors.building_id,修复
// 信息点管理页面"所属楼宇"列对部分行显示 "-" 的问题。
//
// 背景 (2026-07-01):
//   - 现象: 信息点列表 (internal/services/operations/infopoint_service.go:202-209)
//     通过三级 LEFT JOIN 算 building_name:
//        ops_info_points → sys_workstation → ops_floors → ops_buildings
//   - 根因: 342 行信息点对应楼层的 ops_floors.building_id 为 NULL(用户探查
//     SQL 确认:workstation_missing=0, floor_missing=0, building_missing=342,
//     building_name_blank=342, total=1341)
//   - 关联: sys_workstation.building_id 是冗余物理列(uuid 类型,信息架构里
//     给前端展示用),82% 工位(1187/1447)有值,可作为反推源头
//
// 策略 (按"先任一工位冗余值"反推):
//   - 对每一个 building_id 为空 (NULL 或 '') 的 ops_floors,
//     从其下任意一个有 building_id 的 sys_workstation 反推
//   - 同楼层多个工位可能冗余值不同(legacy 写入 bug),取任意一个都显著优于
//     "永不显示";后续如发现仍有不一致,以同楼层"众数 building_id"二次修正
//
// 安全性:
//   - WHERE 仅匹配 NULL 或空字符串的楼层,绝不覆盖已有值(幂等可重跑)
//   - EXISTS 子查询先校准"有可反推源"再 UPDATE,避免子查询无值时把整列擦空
//   - sys_workstation.building_id 是 uuid,赋值到 ops_floors.building_id(varchar(64))
//     需要 ::text 显式转换
//   - 仅 PostgreSQL 方言 (ops_floors.building_id 强类型推断依赖 PG)
//   - UPDATE 不动 sys_workstation、不动 ops_info_points、不动 ops_buildings
//
// 验证(执行后):
//   SELECT COUNT(*) FROM ops_floors
//    WHERE building_id IS NULL OR building_id = '';
//   期望: 0(或低于修复前 — 仍可能剩少量"整层工位冗余字段全空"的孤儿楼层,
//        那部分需手工映射,见 export_unresolved_floors.sql 产物)
//
// 数据库合并备忘 (用户在解决 internal/core/db/database.go 的合并冲突时使用):
//   - 此文件函数签名: Migrate192BackfillOpsFloorsBuildingId
//   - database.go 注册位置: 紧跟 migration_191_drop_info_points_redundant_columns
//     之后(migration 编号应等于在 AutoMigrate 链中调用顺序);
//     M192 必须在 M191 之前注册,且调用顺序必须是同一 AutoMigrate 链路里
//     "192 → 191" 以确保数据补完后,再删 ops_info_points 冗余列,List 接口
//     完全切到 LEFT JOIN 路径时已有完整数据
func Migrate192BackfillOpsFloorsBuildingId(db *gorm.DB) error {
	log.Println("Running migration 192: 反推回填 ops_floors.building_id(修复信息点所属楼宇显示-)")

	if !isPostgreSQL(db) {
		log.Println("Migration 192 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// 第一步: 备份未修复前的缺失计数(用于日志对照修复效果)。
	var missingBefore int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM ops_floors
		 WHERE building_id IS NULL OR building_id = ''
	`).Scan(&missingBefore).Error; err != nil {
		return fmt.Errorf("统计修复前缺失楼层失败: %w", err)
	}
	log.Printf("Migration 192 step0: 修复前缺失 building_id 的楼层数 = %d", missingBefore)
	if missingBefore == 0 {
		log.Println("Migration 192 step0: 全部楼层已有关联楼宇,无需修复,直接结束")
		return nil
	}

	// 第二步: 核心反推 UPDATE。
	// 子查询取同楼层下任意一个有冗余 building_id 的工位;::text 把 uuid 转字符串适配
	// ops_floors.building_id(varchar(64))。LIMIT 1 防止多值时 PG 报 "more than one row"。
	// ORDER BY w.building_id 让同一楼层多次重跑结果稳定(同集合相同首行)。
	result := db.Exec(`
		UPDATE ops_floors f
		   SET building_id = (
				SELECT w.building_id::text
				  FROM sys_workstation w
				 WHERE w.floor_id::uuid = f.id
				   AND w.building_id IS NOT NULL
				 ORDER BY w.building_id
				 LIMIT 1
			   )
		 WHERE (f.building_id IS NULL OR f.building_id = '')
		   AND EXISTS (
				SELECT 1 FROM sys_workstation w
				 WHERE w.floor_id::uuid = f.id
				   AND w.building_id IS NOT NULL
			   )
	`)
	if result.Error != nil {
		return fmt.Errorf("反推 ops_floors.building_id 失败: %w", result.Error)
	}
	log.Printf("Migration 192 step2: 反推已填充 ops_floors.building_id,影响 %d 行", result.RowsAffected)

	// 第三步: 修复后回查,产出两类指标。
	type CountRow struct {
		StillMissing int64
		StillEmpty   int64
	}
	var counts CountRow
	if err := db.Raw(`
		SELECT
			COUNT(*) FILTER (WHERE building_id IS NULL)                                  AS still_missing,
			COUNT(*) FILTER (WHERE building_id IS NOT NULL AND building_id = '')         AS still_empty
		  FROM ops_floors
	`).Scan(&counts).Error; err != nil {
		// 不阻断流程,仅日志
		log.Printf("Migration 192 step3 回查失败(非阻断): %v", err)
	}

	stillTotal := counts.StillMissing + counts.StillEmpty
	if stillTotal > 0 {
		log.Printf("⚠️  Migration 192 step3: 仍有 %d 个楼层未反推到 building_id(同楼层所有工位的冗余字段也为空,需手工映射)", stillTotal)
		log.Printf("    still_missing (NULL) = %d, still_empty ('') = %d", counts.StillMissing, counts.StillEmpty)
		log.Printf("    兜底建议: 用以下 SQL 导出清单,逐条人工指定后批量 UPDATE")
		log.Printf("    SELECT id::text, name, floor_no, created_at FROM ops_floors WHERE building_id IS NULL OR building_id = '' ORDER BY name;")
	} else {
		log.Printf("✅ Migration 192 step3: ops_floors.building_id 全部反推完成 (0 缺失)")
	}

	// 第四步: 直接验证信息点列表那条统计 SQL 现在看到的"所属楼宇"覆盖率,
	// 这条 SQL 与 infopoint 列表页 (infopoint_service.go:202) 的 JOIN 路径一一对应。
	type IPCountRow struct {
		BuildingMissing int64
		Total           int64
	}
	var ipCounts IPCountRow
	if err := db.Raw(`
		SELECT
			COUNT(*) FILTER (WHERE b.id IS NULL)  AS building_missing,
			COUNT(*)                              AS total
		  FROM ops_info_points ip
		  LEFT JOIN sys_workstation w ON w.id::text = ip.workstation_id
		  LEFT JOIN ops_floors       f ON f.id = w.floor_id::uuid
		  LEFT JOIN ops_buildings    b ON b.id = w.building_id::uuid
	`).Scan(&ipCounts).Error; err != nil {
		log.Printf("Migration 192 step4 信息点回查失败(非阻断): %v", err)
	} else {
		log.Printf("Migration 192 step4: 信息点 building_missing = %d / total = %d(目标=0)",
			ipCounts.BuildingMissing, ipCounts.Total)
	}

	log.Println("Migration 192 completed")
	return nil
}
