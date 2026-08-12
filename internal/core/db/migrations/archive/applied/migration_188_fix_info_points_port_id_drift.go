//go:build archive_skip


package migrations

import (
	"fmt"
	"log"
	"os"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate188FixInfoPointsPortIdDrift 修复 ops_info_points.port_id 历史漂移。
//
// 背景 (2026-07-01, debug session info-point-port-drift-recheck-20260701):
//   - 1077/1483 info_point 的 port_id 指向"错设备"的同名 port
//     (Excel import DependsOn scope 失效, 跨设备 first-match 拿错 port 行)
//   - 例 5F003: ip.device_id=aca124c8 (05F#1), 但 port_id 指向 515f4c58 (04F#2) 的 GE5/44
//   - 物理链路查询 v3 已锚 ip.device_id, 显示不受影响; 但 drift 数据本身不一致
//
// 修复方向 (与已删除的 migration_182 相反 — 182 改 port_status.device_id 是错的):
//   - 改 ip.port_id 指向 (ip.device_id, 同 interface_name) 的正确 port 行
//   - 只改 ops_info_points.port_id, 不改 device_id / port_status → 安全可逆
//   - fixable 1077/1077 = 100% (每个 drift 行的 ip.device_id 下都有同 interface 的 port)
//
// 对物理链路显示无影响 (已验证 2026-07-01):
//   - workstation_device_service GetPhysicalDevices v3 锚 ip.device_id
//   - 改 port_id 前后 interface_name 相同 (correct_port.interface_name = current_port.interface_name)
//   - MAC JOIN 结果完全不变
//
// 安全护栏:
//   1. dry-run 默认开启 (MIGRATION_188_DRY_RUN 未设或 != "false" 时只预览不写)
//   2. 备份表 sys_info_points_port_id_drift_backup (ip.id, old/new port_id, interface_name)
//   3. affected_rows 上限 2000 (当前 1077, 超限中止)
//   4. 备份 + UPDATE 包在同一事务, 任一失败整体回滚
//   5. 后置验证 drift == 0
//
// 回滚指引 (log 输出, 需人工执行):
//   UPDATE ops_info_points ip SET port_id = b.old_port_id
//     FROM sys_info_points_port_id_drift_backup b WHERE ip.id = b.id;
//   DROP TABLE sys_info_points_port_id_drift_backup;
func Migrate188FixInfoPointsPortIdDrift(db *gorm.DB) error {
	log.Println("Running migration 188: 修复 ops_info_points.port_id 漂移")

	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] 188 跳过(非 PostgreSQL)")
		log.Println("Migration 188 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// dry-run 默认开启: 未设或 != "false" 时只预览。设 MIGRATION_188_DRY_RUN=false 才真实写入。
	dryRun := os.Getenv("MIGRATION_188_DRY_RUN") != "false"
	const maxAffected int64 = 2000

	// 阶段 0: 预览 drift 总数 + fixable 数
	var driftCount, fixableCount int64
	if err := db.Raw(`
SELECT COUNT(*) FROM ops_info_points ip
  JOIN sys_device_port_status cur ON cur.id::text = ip.port_id
 WHERE cur.device_id::text != ip.device_id::text
   AND ip.deleted_at IS NULL AND ip.status = 0 AND ip.device_id IS NOT NULL`).Scan(&driftCount).Error; err != nil {
		return fmt.Errorf("检查 drift 数失败: %w", err)
	}
	if err := db.Raw(`
SELECT COUNT(*) FROM ops_info_points ip
  JOIN sys_device_port_status cur ON cur.id::text = ip.port_id
  JOIN sys_device_port_status cor
    ON cor.device_id::text = ip.device_id::text
   AND cor.interface_name = cur.interface_name
   AND cor.id != cur.id
 WHERE cur.device_id::text != ip.device_id::text
   AND ip.deleted_at IS NULL AND ip.status = 0 AND ip.device_id IS NOT NULL`).Scan(&fixableCount).Error; err != nil {
		return fmt.Errorf("检查 fixable 数失败: %w", err)
	}
	applogger.Infof("[迁移] 188 预览: drift=%d fixable=%d (上限 %d) dry_run=%v",
		driftCount, fixableCount, maxAffected, dryRun)

	if fixableCount == 0 {
		log.Println("Migration 188: 无可修复行, 跳过")
		return nil
	}
	if fixableCount > maxAffected {
		return fmt.Errorf("[迁移] 188 fixable %d 超过上限 %d, 中止 (检查数据或联系开发者调高上限)",
			fixableCount, maxAffected)
	}

	if dryRun {
		applogger.Infof("[迁移] 188 DRY-RUN 模式: 预览完成, 未写入数据。设 MIGRATION_188_DRY_RUN=false 重启后端执行真实修复")
		log.Println("Migration 188: DRY-RUN preview complete, no data written")
		return nil
	}

	// 阶段 1: 备份 + UPDATE (单事务, 原子)
	err := db.Transaction(func(tx *gorm.DB) error {
		// 1a. 备份表 (幂等: 先 DROP 再 CREATE)
		if err := tx.Exec(`DROP TABLE IF EXISTS sys_info_points_port_id_drift_backup`).Error; err != nil {
			return fmt.Errorf("清理旧备份表失败: %w", err)
		}
		if err := tx.Exec(`
CREATE TABLE sys_info_points_port_id_drift_backup (
    id uuid PRIMARY KEY,
    old_port_id varchar(64) NOT NULL,
    new_port_id varchar(64) NOT NULL,
    interface_name varchar(100),
    fixed_at timestamp NOT NULL DEFAULT NOW()
)`).Error; err != nil {
			return fmt.Errorf("创建备份表失败: %w", err)
		}

		// 1b. 备份原 port_id (ip.id, old_port_id, new_port_id, interface_name)
		backupSQL := `
INSERT INTO sys_info_points_port_id_drift_backup (id, old_port_id, new_port_id, interface_name)
SELECT ip.id, ip.port_id, cor.id::text, cur.interface_name
  FROM ops_info_points ip
  JOIN sys_device_port_status cur ON cur.id::text = ip.port_id
  JOIN sys_device_port_status cor
    ON cor.device_id::text = ip.device_id::text
   AND cor.interface_name = cur.interface_name
   AND cor.id != cur.id
 WHERE cur.device_id::text != ip.device_id::text
   AND ip.deleted_at IS NULL AND ip.status = 0 AND ip.device_id IS NOT NULL`
		backupResult := tx.Exec(backupSQL)
		if backupResult.Error != nil {
			return fmt.Errorf("备份原 port_id 失败: %w", backupResult.Error)
		}
		applogger.Infof("[迁移] 188 备份完成: %d 行原 port_id 已记录", backupResult.RowsAffected)

		// 1c. UPDATE port_id → correct_port.id (ip.device_id 下同 interface_name 的 port)
		updateSQL := `
UPDATE ops_info_points ip
   SET port_id = cor.id::text
  FROM sys_device_port_status cur, sys_device_port_status cor
 WHERE ip.port_id::text = cur.id::text
   AND cor.device_id::text = ip.device_id::text
   AND cor.interface_name = cur.interface_name
   AND cor.id != cur.id
   AND cur.device_id::text != ip.device_id::text
   AND ip.deleted_at IS NULL AND ip.status = 0 AND ip.device_id IS NOT NULL`
		updResult := tx.Exec(updateSQL)
		if updResult.Error != nil {
			return fmt.Errorf("UPDATE port_id 失败: %w", updResult.Error)
		}
		applogger.Infof("[迁移] 188 UPDATE 完成: %d 行 port_id 已修正", updResult.RowsAffected)

		return nil
	})
	if err != nil {
		return err
	}

	// 阶段 2: 后置验证 (期望 drift == 0)
	var remaining int64
	if err := db.Raw(`
SELECT COUNT(*) FROM ops_info_points ip
  JOIN sys_device_port_status cur ON cur.id::text = ip.port_id
 WHERE cur.device_id::text != ip.device_id::text
   AND ip.deleted_at IS NULL AND ip.status = 0 AND ip.device_id IS NOT NULL`).Scan(&remaining).Error; err != nil {
		applogger.Warnf("[迁移] 188 后置验证查询失败(非阻断): %v", err)
	} else {
		applogger.Infof("[迁移] 188 验证: 剩余 drift = %d (期望 0)", remaining)
	}

	// 阶段 3: 回滚指引 (log, 不自动执行)
	rollbackHint := `-- 手动回滚 (如需):
-- 1. UPDATE ops_info_points ip SET port_id = b.old_port_id
--      FROM sys_info_points_port_id_drift_backup b WHERE ip.id = b.id;
-- 2. DROP TABLE sys_info_points_port_id_drift_backup;`
	applogger.Infof("[迁移] 188 回滚指引:\n%s", rollbackHint)

	log.Println("Migration 188 completed")
	return nil
}
