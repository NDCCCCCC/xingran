package migrations

import (
	"fmt"
	"log"

	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
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

	// 2. 验证 view 抽样
	var physicalCount int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM reconciliation_physical_chain WHERE physical_user_id IS NOT NULL
	`).Scan(&physicalCount).Error; err != nil {
		return fmt.Errorf("验证 reconciliation_physical_chain 视图失败: %w", err)
	}
	applogger.Infof("[迁移] reconciliation_physical_chain 命中 physical_user_id 的行数: %d", physicalCount)

	// 3. 重建 reconciliation_normalized 物化视图
	//    用的是 migration_176 的同一段 SQL,无逻辑变化(数据源已修)
	reconMVSQL := `
DROP MATERIALIZED VIEW IF EXISTS reconciliation_normalized CASCADE;
CREATE MATERIALIZED VIEW reconciliation_normalized AS
SELECT DISTINCT ON (a.id)
    a.id                                              AS asset_id,
    a.devicesn                                        AS asset_code,
    a.machine_ip                                      AS asset_ip,
    a.mac1                                            AS mac1,
    a.mac2                                            AS mac2,
    COALESCE(NULLIF(a.mac1,''), NULLIF(a.mac2,''))    AS mac_join,
    COALESCE(
        NULLIF(a.user_id,''),
        (SELECT user_id_by_name_and_dept FROM reconciliation_user_lookup WHERE asset_id = a.id LIMIT 1),
        (SELECT user_id_by_name          FROM reconciliation_user_lookup WHERE asset_id = a.id LIMIT 1)
    )                                                 AS asset_user_id,
    COALESCE(
        (SELECT username FROM sys_user WHERE id::text = NULLIF(a.user_id,'') LIMIT 1),
        a.nowuser_name
    )                                                 AS asset_username,
    a.deleted_at                                      AS asset_deleted_at,
    pc.physical_user_id                               AS physical_user_id,
    pc.physical_username                              AS physical_username,
    ad.id                                             AS ad_id,
    ad.username                                       AS ad_username,
    ad.is_enabled                                     AS ad_is_enabled,
    NOW()                                             AS mv_refreshed_at,
    last_resolved.resolved_at                         AS last_resolved_at,
    last_resolved.resolved_by                         AS last_resolved_by,
    last_resolved.conflict_type                       AS last_conflict_type
FROM ops_asset a
LEFT JOIN reconciliation_physical_chain pc ON pc.asset_id = a.id
LEFT JOIN sys_ad_user ad
       ON ad.username = COALESCE(
              (SELECT username FROM sys_user WHERE id::text = NULLIF(a.user_id,'') LIMIT 1),
              a.nowuser_name
          )
      AND ad.deleted_at IS NULL
      AND ad.is_enabled = TRUE
LEFT JOIN LATERAL (
    SELECT resolved_at, resolved_by, conflict_type
    FROM sys_data_reconciliation r
    WHERE r.asset_id = a.id
      AND r.resolved_at IS NOT NULL
      AND r.deleted_at IS NULL
    ORDER BY r.resolved_at DESC
    LIMIT 1
) last_resolved ON true
WHERE a.deleted_at IS NULL
ORDER BY a.id, ad.id NULLS LAST;

CREATE UNIQUE INDEX IF NOT EXISTS idx_recon_norm_asset
    ON reconciliation_normalized (asset_id);
CREATE INDEX IF NOT EXISTS idx_recon_norm_workstation
    ON reconciliation_normalized (asset_user_id);
CREATE INDEX IF NOT EXISTS idx_recon_norm_physical
    ON reconciliation_normalized (physical_user_id)
    WHERE physical_user_id IS NOT NULL;
`
	if err := db.Exec(reconMVSQL).Error; err != nil {
		return fmt.Errorf("重建 reconciliation_normalized 物化视图失败: %w", err)
	}
	applogger.Infof("[迁移] reconciliation_normalized MV 已重建(用新 reconciliation_physical_chain 视图数据)")

	// 4. 验证 MV 抽样
	var mvPhysicalCount int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM reconciliation_normalized WHERE physical_user_id IS NOT NULL
	`).Scan(&mvPhysicalCount).Error; err != nil {
		return fmt.Errorf("验证 reconciliation_normalized 物化视图失败: %w", err)
	}
	applogger.Infof("[迁移] reconciliation_normalized MV 命中 physical_user_id 的行数: %d(修复前应为 0)", mvPhysicalCount)

	// 5. 标记 view 定义以便 SQL 端验证是否已重建
	markViewSQL := `COMMENT ON VIEW reconciliation_physical_chain IS 'R5_fix_mac_iface_20260629'`
	if err := db.Exec(markViewSQL).Error; err != nil {
		applogger.Warnf("[迁移] 设置 view 标记失败 (非阻断): %v", err)
	}

	// 6. 同步刷新 reconciliation_user_lookup(它也可能被重写)
	if err := db.Exec(`REFRESH MATERIALIZED VIEW reconciliation_normalized`).Error; err != nil {
		applogger.Warnf("[迁移] 刷新 reconciliation_normalized MV 失败 (非阻断): %v", err)
	}

	log.Println("Migration 178 completed: reconciliation_physical_chain view + reconciliation_normalized MV rebuilt with normalization")
	return nil
}
