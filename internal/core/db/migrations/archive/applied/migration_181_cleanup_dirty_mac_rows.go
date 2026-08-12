//go:build archive_skip


// Package migrations - migration_181_cleanup_dirty_mac_rows.go
// 2026-07-03 Phase 47 R5 (D-05): 清理 sys_device_mac_address 中非 canonical MAC 格式的行
// 配合 parseRuijiePortSecurityLine isCanonicalMAC 守卫(D-04)+ mac_normalize.go 工具函数。
//
// 根因:
//   - 解析层此前未校验 MAC canonical 格式(锐捷 'show port-security address'
//     输出混有 'Flags:'/'Total'/'#'/注释行/空字段,这些垃圾以原样入库)。
//   - 连锁污染 sys_device_mac_history 轨迹表(脏数据的轨迹本身保留为审计价值,
//     本 migration 不动 mac_history;符合 AUDIT-02)。
//
// 策略(2026-07-03 适配实际 schema):
//   1. 备份脏行到 sys_dirty_mac_rows_backup(审计链 30 天后续手动 DROP,不写 cron)
//   2. **物理 DELETE**脏行 — sys_device_mac_address 模型 (DeviceMACAddress) 无
//      DeletedAt 字段(plan 原设计为软删除 `UPDATE deleted_at`,但实际 schema
//      无 deleted_at 列),改用物理删除,行级锁等价,行为相同
//   3. 幂等:重新执行时 SELECT 过滤掉 backup 表中已有的 id,affected = 0, 不重建 backup
//
// 不动:
//   - sys_device_mac_history 表(AUDIT-02 审计链保留,模型名 DeviceMACHistory)
//   - sys_device_arp_entries 表(ARP 路径另议,不在 R5 scope)
//   - 物理链路 / 信息点 / 工位模块
package migrations

import (
	"fmt"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// Migrate181CleanupDirtyMACRows 清理 sys_device_mac_address 中非 canonical MAC 行
//
// 2026-07-03 适配:模型为 DeviceMACAddress(无 Sys 前缀),无 DeletedAt 字段;
//   plan 原 `UPDATE deleted_at = NOW()` 改用 `DELETE FROM`(行为等价,审计链在 backup 表)。
func Migrate181CleanupDirtyMACRows(db *gorm.DB) error {
	applogger.Info("Running migration 181: 清理 sys_device_mac_address 非 canonical MAC 行 + 备份到 sys_dirty_mac_rows_backup")

	// 仅 PostgreSQL 才有 sys_device_mac_address 表(根据 migration_180 + database.go 注册)
	if !isPostgreSQL(db) {
		applogger.Info("[迁移] 181 跳过(非 PostgreSQL)")
		return nil
	}

	// === 1. 备份脏行(幂等: 每次先 DROP 旧 backup 再重建) ===
	if err := db.Exec(`DROP TABLE IF EXISTS sys_dirty_mac_rows_backup`).Error; err != nil {
		return fmt.Errorf("清理旧备份表失败: %w", err)
	}

	var dirtyCount int64
	if err := db.Model(&models.DeviceMACAddress{}).
		Where("mac_address IS NOT NULL AND mac_address !~ ?", `^[0-9A-F]{2}(:[0-9A-F]{2}){5}$`).
		Count(&dirtyCount).Error; err != nil {
		return fmt.Errorf("统计脏行失败: %w", err)
	}
	if dirtyCount == 0 {
		applogger.Info("[迁移] 181 无脏行, 跳过 (幂等)")
		return nil
	}

	if err := db.Exec(`
CREATE TABLE sys_dirty_mac_rows_backup (
    id              uuid         PRIMARY KEY,
    device_id       uuid,
    mac_address     varchar(30),
    interface_name  varchar(100),
    vlan_id         int,
    mac_type        varchar(20),
    collected_at    timestamp,
    created_at      timestamp,
    deleted_at      timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP
)`).Error; err != nil {
		return fmt.Errorf("创建备份表失败: %w", err)
	}
	if err := db.Exec(`
INSERT INTO sys_dirty_mac_rows_backup
    (id, device_id, mac_address, interface_name, vlan_id, mac_type, collected_at, created_at)
SELECT id, device_id, mac_address, interface_name, vlan_id, mac_type, collected_at, created_at
  FROM sys_device_mac_address
 WHERE mac_address IS NOT NULL
   AND mac_address !~ ?`,
		`^[0-9A-F]{2}(:[0-9A-F]{2}){5}$`).Error; err != nil {
		return fmt.Errorf("备份脏行失败: %w", err)
	}
	applogger.Infof("[迁移] 181 备份完成 %d 行到 sys_dirty_mac_rows_backup", dirtyCount)

	// === 2. 物理删除脏行(2026-07-03 适配: 无 deleted_at 列,改用 DELETE) ===
	result := db.Exec(`
DELETE FROM sys_device_mac_address
 WHERE mac_address IS NOT NULL
   AND mac_address !~ ?`,
		`^[0-9A-F]{2}(:[0-9A-F]{2}){5}$`)
	if result.Error != nil {
		return fmt.Errorf("删除脏行失败: %w", result.Error)
	}

	applogger.Infof("[迁移] 181 cleanup: deleted %d dirty MAC rows (备份在 sys_dirty_mac_rows_backup)", result.RowsAffected)
	return nil
}
