//go:build archive_skip


package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// Migrate193BackfillSysWorkstationBuildingId 反推回填 sys_workstation.building_id,
//
// 修复信息点管理页面"所属楼宇"对部分行显示 "-" 的根因。
//
// 背景 (2026-07-01):
//   - 现象: 信息点列表 (internal/services/operations/infopoint_service.go:206) 用
//     `LEFT JOIN ops_buildings ON ops_buildings.id = sys_workstation.building_id::uuid`
//     计算 building_name。
//   - 真实根因: 260 个工位的 sys_workstation.building_id 是 NULL(冗余字段导入时漏填)。
//     不是 ops_floors.building_id(已验证全有值)。
//   - migration_192 错把 ops_floors 当成问题(0 缺失 → no-op);本迁移修正方向。
//
// 用户 dry-run 验证 (2026-07-01 23:31):
//   SELECT ... FROM ops_info_points WHERE w.building_id IS NULL;
//   → 全部命中行的 building_name_via_floor (即 ops_floors.building_id → ops_buildings.name)
//     都返回"浙商大厦",反推源 100% 完整,本迁移必然 100% 修复。
//
// 策略 (方向逆转:
//   - 对每一个 sys_workstation.building_id IS NULL 的工位,根据其 floor_id 反向查
//     ops_floors.building_id,再 ::uuid 强转写到 sys_workstation.building_id
//   - 同楼层多个工位应该有相同的 ops_floors.building_id(因为楼层只属于一个楼宇),
//     LIMIT 1 + ORDER BY 保证重跑结果稳定
//
// 安全性:
//   - WHERE 仅匹配 building_id IS NULL 的工位,绝不覆盖已有值(幂等可重跑)
//   - EXISTS 子查询先校准"有可反推源"再 UPDATE,防子查询无值时把已有值擦空
//   - sys_workstation.building_id 是 uuid 类型,只能 IS NULL(uuid 没有空字符串)
//   - ops_floors.building_id 是 varchar(64) 类型,赋值需要 ::uuid 显式转换
//   - 仅 PostgreSQL 方言
//   - 不动 ops_floors / ops_info_points / ops_buildings 表,聚焦 sys_workstation
//
// 数据库合并备忘 (用户在解决 internal/core/db/database.go 时使用):
//   - 函数签名: Migrate193BackfillSysWorkstationBuildingId
//   - 调用顺序: 在 Migration192 之后(192 no-op 但保留作为历史记录;193 才是真修复)
func Migrate193BackfillSysWorkstationBuildingId(db *gorm.DB) error {
	log.Println("Running migration 193: 反推回填 sys_workstation.building_id(从 ops_floors.building_id)")

	if !isPostgreSQL(db) {
		log.Println("Migration 193 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// 第一步: 备份修复前缺失计数。
	var missingBefore int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM sys_workstation
		 WHERE building_id IS NULL
		   AND floor_id IS NOT NULL
	`).Scan(&missingBefore).Error; err != nil {
		return fmt.Errorf("统计修复前缺失工位数失败: %w", err)
	}
	log.Printf("Migration 193 step0: 修复前 floor_id 有值且 building_id 为 NULL 的工位数 = %d", missingBefore)
	if missingBefore == 0 {
		log.Println("Migration 193 step0: 无需要反推的工位,直接结束")
		return nil
	}

	// 第二步: 核心反推 UPDATE。
	// sys_workstation.building_id 是 uuid 类型, ops_floors.building_id 是 varchar(64),
	// 因此赋值时需要 ::uuid 显式强转。
	// 子查询 ORDER BY 提升重跑幂等稳定性(同楼层理论上 floor.building_id 全一致)。
	result := db.Exec(`
		UPDATE sys_workstation w
		   SET building_id = (
				SELECT f.building_id::uuid
				  FROM ops_floors f
				 WHERE f.id = w.floor_id::uuid
				   AND f.building_id IS NOT NULL
				   AND f.building_id <> ''
				 ORDER BY f.building_id
				 LIMIT 1
			   )
		 WHERE w.building_id IS NULL
		   AND w.floor_id IS NOT NULL
		   AND EXISTS (
				SELECT 1 FROM ops_floors f
				 WHERE f.id = w.floor_id::uuid
				   AND f.building_id IS NOT NULL
				   AND f.building_id <> ''
			   )
	`)
	if result.Error != nil {
		return fmt.Errorf("反推 sys_workstation.building_id 失败: %w", result.Error)
	}
	log.Printf("Migration 193 step2: 反推已填充 sys_workstation.building_id,影响 %d 行", result.RowsAffected)

	// 第三步: 修复后回查,仍缺失的工位数(纯孤儿楼层:sys_workstation.floor_id
	// 指向一栋 ops_floors 上根本没 building_id 的工位,理论上 dry-run 应为 0)。
	var stillMissing int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM sys_workstation
		 WHERE building_id IS NULL
		   AND floor_id IS NOT NULL
	`).Scan(&stillMissing).Error; err != nil {
		log.Printf("Migration 193 step3 回查失败(非阻断): %v", err)
	} else {
		log.Printf("Migration 193 step3: 修复后仍缺失 building_id 的工位数 = %d (期望 0)", stillMissing)
		if stillMissing > 0 {
			log.Print("⚠️  仍有未反推的工位,导出清单: SELECT id::text, workstation_name, floor_id FROM sys_workstation WHERE building_id IS NULL AND floor_id IS NOT NULL;")
		}
	}

	// 第四步: 信息点视图复检(与 infopoint 列表页 JOIN 路径 1:1 对齐)。
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
		log.Printf("Migration 193 step4 信息点回查失败(非阻断): %v", err)
	} else {
		log.Printf("✅ Migration 193 step4: 信息点 building_missing = %d / total = %d(目标=0)",
			ipCounts.BuildingMissing, ipCounts.Total)
	}

	log.Println("Migration 193 completed")
	return nil
}
