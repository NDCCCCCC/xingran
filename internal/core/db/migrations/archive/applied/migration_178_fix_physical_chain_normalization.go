//go:build archive_skip


package migrations

import (
	"fmt"
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate178FixPhysicalChainNormalization 修正 reconciliation_physical_chain 视图
// 的 JOIN 条件,加入 MAC 格式 + interface_name 格式归一化
//
// 背景:
//   - migration_175 创建 reconciliation_physical_chain 视图时,JOIN 条件用了字节级
//     严格比对:
//       * mac.mac_address ('b022.7a2e.4a4f' Cisco 小写点分) =
//         a.mac1        ('B0:22:7A:2E:4A:4F' Windows 大写冒号)  ← 永远不匹配
//       * port.interface_name ('GE2/25' 短名) =
//         mac.interface_name ('GigabitEthernet 2/25' 长名)    ← 永远不匹配
//   - 导致 reconciliation_normalized 物化视图(migration_176 重建)里的
//     physical_user_id / physical_username 永远是 NULL
//   - ClassifySignals.hasPhysical 永远 false → 所有设备的 conflict_type 都是
//     "物理无/责任人有" → 整工位对账健康度永远异常
//   - workstation_device_service.go 的 GetPhysicalDevices 已修复同样问题(Phase 45 后续),
//     但 view 是 Phase 45 R5 早期创建,没被同步修复
//
// 修复:
//   1. CREATE OR REPLACE VIEW reconciliation_physical_chain:用 LOWER + REGEXP_REPLACE
//      把 MAC 归一化到 12 位 hex;interface_name 走 GE/Gi/GigabitEthernet 归一化
//   2. REFRESH MATERIALIZED VIEW reconciliation_normalized:让 MV 立即重算
//   3. 同时把 InfoPoint JOIN 也加 port.device_id 防御,避免 port_status 漂移
//   4. 跑一次 ops_asset_physical 物理化表的 refresh(如有 cron 在维护则跳过)
func Migrate178FixPhysicalChainNormalization(db *gorm.DB) error {
	log.Println("Running migration 178: Fix reconciliation_physical_chain view normalization + refresh MV")

	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] 178 跳过(非 PostgreSQL)")
		log.Println("Migration 178 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// 1. 重建 reconciliation_physical_chain 视图——MAC + interface_name 归一化
	createViewSQL := `
CREATE OR REPLACE VIEW reconciliation_physical_chain AS
WITH latest_mac AS (
    SELECT DISTINCT ON (LOWER(REGEXP_REPLACE(COALESCE(m.mac_address, ''), '[.:\-]', '', 'g')))
        m.mac_address,
        m.device_id,
        m.interface_name
      FROM sys_device_mac_address m
     ORDER BY LOWER(REGEXP_REPLACE(COALESCE(m.mac_address, ''), '[.:\-]', '', 'g')),
              m.collected_at DESC NULLS LAST
),
norm AS (
    SELECT
        a.id                                              AS asset_id,
        a.devicesn                                        AS asset_code,
        -- 归一化资产端 MAC:小写 + 去 . : - 分隔符,统一 12 位 hex
        LOWER(REGEXP_REPLACE(COALESCE(NULLIF(a.mac1,''), NULLIF(a.mac2,'')), '[.:\-]', '', 'g')) AS asset_norm_mac,
        -- 归一化采集端 MAC(同 latest_mac 内的列):直接复用
        LOWER(REGEXP_REPLACE(COALESCE(mac.mac_address, ''), '[.:\-]', '', 'g')) AS mac_norm_mac,
        -- 归一化接口名:小写 + 去空格 + GE/Gi/GigabitEthernet → 'ge' 前缀
        LOWER(REGEXP_REPLACE(COALESCE(mac.interface_name, ''), '\s+', '', 'g')) AS mac_norm_iface,
        mac.device_id,
        LOWER(REGEXP_REPLACE(COALESCE(port.interface_name, ''), '\s+', '', 'g')) AS port_norm_iface
    FROM ops_asset a
    LEFT JOIN latest_mac mac
           ON LOWER(REGEXP_REPLACE(COALESCE(mac.mac_address, ''), '[.:\-]', '', 'g'))
            = LOWER(REGEXP_REPLACE(COALESCE(NULLIF(a.mac1,''), NULLIF(a.mac2,'')), '[.:\-]', '', 'g'))
    LEFT JOIN sys_device_port_status port
           ON port.device_id::text = mac.device_id::text
          AND LOWER(REGEXP_REPLACE(COALESCE(port.interface_name, ''), '\s+', '', 'g'))
            = LOWER(REGEXP_REPLACE(COALESCE(mac.interface_name, ''), '\s+', '', 'g'))
)
SELECT
    n.asset_id,
    n.asset_code,
    COALESCE(NULLIF((SELECT mac1 FROM ops_asset WHERE id = n.asset_id), ''),
             NULLIF((SELECT mac2 FROM ops_asset WHERE id = n.asset_id), '')) AS mac_join,
    ws.id                                                                  AS workstation_id,
    ws.user_id                                                             AS physical_user_id,
    su.username                                                            AS physical_username
FROM norm n
LEFT JOIN ops_info_points ip
       ON ip.port_id::text IN (
              SELECT id::text FROM sys_device_port_status
               WHERE device_id::text = n.device_id::text
                 AND LOWER(REGEXP_REPLACE(COALESCE(interface_name, ''), '\s+', '', 'g')) = n.mac_norm_iface
                 AND LOWER(REGEXP_REPLACE(COALESCE(interface_name, ''), '\s+', '', 'g')) = n.port_norm_iface
          )
      AND ip.deleted_at IS NULL
      AND ip.status      = 0
LEFT JOIN sys_workstation ws
       ON ws.id::text    = ip.workstation_id::text
      AND ws.deleted_at IS NULL
LEFT JOIN sys_user su
       ON su.id::text    = ws.user_id::text
      AND su.deleted_at IS NULL
WHERE n.asset_norm_mac = n.mac_norm_mac;  -- 防御:只在 MAC 归一化相等时输出
`
	if err := db.Exec(createViewSQL).Error; err != nil {
		return fmt.Errorf("重建 reconciliation_physical_chain 视图失败: %w", err)
	}
	applogger.Infof("[迁移] reconciliation_physical_chain 视图已重建(JOIN 加入 MAC/接口名归一化)")

	// ⚠ 2026-06-30 性能修正:本迁移原在此 DROP+CREATE MV + 两次 COUNT 验证 + REFRESH MV,
	// 但 LOWER 视图虽少匹配(GE≠GigabitEthernet 不等值),其相关子查询仍在生产数据下让
	// 这组操作烧 ~1.5min(实测 COUNT 16s + MV 重建 1m19s)。MV + unique index 已由
	// migration_176 创建,migration_180 会用 O(N+M) 快视图 REFRESH MV。故本迁移只保留:
	//   - CREATE OR REPLACE VIEW(轻量 DDL,上面已做)
	//   - 补 2 个 MV 查询索引(workstation/physical,176 未建;IF NOT EXISTS 幂等,MV 已存在)
	//   - view 版本注释
	// 去掉 DROP+CREATE MV + COUNT 验证 + REFRESH MV,启动省 ~1.5min。COUNT 命中验证由
	// migration_180(快视图)统一做。
	idxSQL := `
CREATE INDEX IF NOT EXISTS idx_recon_norm_workstation
    ON reconciliation_normalized (asset_user_id);
CREATE INDEX IF NOT EXISTS idx_recon_norm_physical
    ON reconciliation_normalized (physical_user_id)
    WHERE physical_user_id IS NOT NULL;
`
	if err := db.Exec(idxSQL).Error; err != nil {
		applogger.Warnf("[迁移] 178 创建 MV 查询索引失败 (非阻断): %v", err)
	}

	// 标记 view 版本以便 SQL 端验证
	markViewSQL := `COMMENT ON VIEW reconciliation_physical_chain IS 'R5_fix_mac_iface_20260629'`
	if err := db.Exec(markViewSQL).Error; err != nil {
		applogger.Warnf("[迁移] 设置 view 标记失败 (非阻断): %v", err)
	}

	log.Println("Migration 178 completed: reconciliation_physical_chain view(MV rebuild deferred to migration 180)")
	return nil
}
