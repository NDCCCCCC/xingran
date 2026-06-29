package migrations

import (
	"fmt"
	"log"

	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate179ReconciliationPhysicalIfaceFold Phase 45 R5 修复 (2026-06-29):
// reconciliation_physical_chain 视图的 interface_name 归一化真正实现跨厂商折叠
//
// 背景:
//   - migration_178 仅做了 LOWER + 空格剥离,
//     注释声称"GE/Gi/GigabitEthernet → 'ge' 前缀"折叠但实际未实现
//   - 实际行为:
//     * GigabitEthernet2/25 → gigabitethernet2/25
//     * GE2/25             → ge2/25
//     * Gi2/25             → gi2/25
//     三者在 SQL 层互不相等,JOIN 永远失败 → physical_user_id 命中率为 0
//   - workstation_device_service.go GetPhysicalDevices (Phase 45 后续修复版) 已经
//     用 REGEXP_REPLACE 链 ('gigabitethernet|gigabitether|ge|gi' → 'ge') 真正折叠
//   - migration_178 view 没跟上,留下双层遗漏(view 与 service 不一致)
//
// 关键决策:
//   - 抽出一个 PL/pgSQL 函数 normalize_iface() 作为单一权威规范,
//     view 与 service 表达式都调用它(后续 Phase 46+ 可复用)
//   - CREATE OR REPLACE VIEW 覆盖 migration_178 的版本
//   - REFRESH MATERIALIZED VIEW reconciliation_normalized 让 MV 立即重算
//   - 不回填 sys_device_port_status 历史数据(尊重真实采集状态;归一化在 view 层做)
//
// SQLite:
//   - sys_device_mac_address / ops_info_points / sys_device_port_status 均为 PostgreSQL-only,
//     SQLite 跳过本 migration
func Migrate179ReconciliationPhysicalIfaceFold(db *gorm.DB) error {
	log.Println("Running migration 179: reconciliation_physical_chain view true interface_name folding")

	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] 179 跳过(非 PostgreSQL)")
		log.Println("Migration 179 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// 1. 创建 PL/pgSQL 归一化函数(权威规范,与 GetPhysicalDevices 已落地表达式对齐)
	//
	// canonical 形式:short + lowercase(便于与 sys_device_port_status 已有短名直接比对)
	// 等价类(全部归一化为同一个串):
	//   GigabitEthernet/gigabitether/gigabitethernet/gi/ge/Gi/GE → ge
	//   TenGigE/tengig/tengige/xe/te                              → te
	//   FortyGigE/fortygig/fortygige/fo/fge                       → fo
	//   HundredGigE/hundredgig/hundredgige/hge                     → hge
	//   TwentyFiveGigE/twentyfivegig/twentyfivegige/twe/tw        → twe
	//   FastEthernet/fa                                            → fa
	//   Vlanif                                                     → vlanif
	//   Vlan/vl                                                    → vlan
	//   Loopback                                                   → loop
	//   NULL/null                                                  → null
	//
	// 注:Go normalizeInterfaceName 当前不对称(fullToShort → 'GE2/25' 大写短名,
	//     prefixList → 'GigabitEthernet2/25' 全称);本函数选 lowercase short canonical
	//     以匹配 GetPhysicalDevices 的 '^(gigabitethernet|gigabitether|ge|gi)','ge' 表达式,
	//     让 view JOIN 与 service 表达式口径一致。后续如需把 Go 改对称,只改一处即可。
	createFuncSQL := `
-- normalize_iface 跨厂商接口名归一化函数
--
-- 设计目标:
--   - 把 GE / Gi / GigabitEthernet / ge 等不同厂商短名/全称 折叠为统一规范
--     'ge2/25' / 'te2/25' / 'fo2/25' / 'hge2/25' / 'twe2/25' / 'fa2/25' 等
--   - canonical 形式选用 short + lowercase,与 GetPhysicalDevices 已落地的
--     '^(gigabitethernet|gigabitether|ge|gi)','ge' 表达式保持一致
--   - IMMUTABLE 让 PG 在 view JOIN 时可做谓词下推(同一输入稳定)
--
-- PG POSIX ERE 行为提醒(避免历史上'^[a-z]+' 占位的反模式):
--   - REGEXP_REPLACE(pattern) 的 | 选取首个匹配(不是最长),所以同一行内更长的候选放前
--   - 例如 '^(twe|tw)' 中 twe 先匹配,tw1/1 才会正确折叠为 twe1/1
--     若写成 '^(tw|twe)' 则 tw 先匹配 tw1/1 → twe1/1;但 twe1/1 会被错误折叠为 twee1/1
CREATE OR REPLACE FUNCTION normalize_iface(name TEXT)
RETURNS TEXT AS $$
DECLARE
    s TEXT;
BEGIN
    -- 1. 小写 + 去空格(任意空白折叠,含制表符等)
    s := LOWER(REGEXP_REPLACE(COALESCE(name, ''), '\s+', '', 'g'));

    -- 2. 逐族折叠(每族独立行,族内长前缀放前以避免 prefix collision)
    --    GigabitEthernet 家族
    s := REGEXP_REPLACE(s, '^(gigabitethernet|gigabitether|gigabite|gi|ge)', 'ge');
    --    TenGigE 家族
    s := REGEXP_REPLACE(s, '^(tengigabit|tengige|tengig|tenge|te|xe)',         'te');
    --    FortyGigE 家族
    s := REGEXP_REPLACE(s, '^(fortygigabit|fortygige|fortygig|fo|fge)',         'fo');
    --    HundredGigE 家族
    s := REGEXP_REPLACE(s, '^(hundredgigabit|hundredgige|hundredgig|hge)',      'hge');
    --    TwentyFiveGigE 家族(twe 早于 tw 避免 prefix collision)
    s := REGEXP_REPLACE(s, '^(twentyfivegigabit|twentyfivegige|twentyfivegig|twe|tw)', 'twe');
    --    FastEthernet 家族
    s := REGEXP_REPLACE(s, '^(fastethernet|fa)',                                'fa');
    --    单前缀家族
    s := REGEXP_REPLACE(s, '^(vlanif)',                                          'vlanif');
    s := REGEXP_REPLACE(s, '^(vlan|vl)',                                         'vlan');
    s := REGEXP_REPLACE(s, '^(loopback)',                                        'loop');
    s := REGEXP_REPLACE(s, '^(null)',                                            'null');

    -- 3. 未识别的前缀(如厂商私有命名)原样返回 lower_no_space,不强制折叠
    RETURN s;
END;
$$ LANGUAGE plpgsql IMMUTABLE;
`
	if err := db.Exec(createFuncSQL).Error; err != nil {
		return fmt.Errorf("创建 normalize_iface 函数失败: %w", err)
	}
	applogger.Infof("[迁移] normalize_iface 函数已创建")

	// 2. 用函数重写 reconciliation_physical_chain 视图
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
        -- 归一化接口名:跨厂商短名真正折叠为统一 lowercase short 形式(2026-06-29 修复)
        normalize_iface(mac.interface_name) AS mac_norm_iface,
        mac.device_id,
        normalize_iface(port.interface_name) AS port_norm_iface
    FROM ops_asset a
    LEFT JOIN latest_mac mac
           ON LOWER(REGEXP_REPLACE(COALESCE(mac.mac_address, ''), '[.:\-]', '', 'g'))
            = LOWER(REGEXP_REPLACE(COALESCE(NULLIF(a.mac1,''), NULLIF(a.mac2,'')), '[.:\-]', '', 'g'))
    LEFT JOIN sys_device_port_status port
           ON port.device_id::text = mac.device_id::text
          AND normalize_iface(port.interface_name) = normalize_iface(mac.interface_name)
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
                 AND normalize_iface(interface_name) = n.mac_norm_iface
                 AND normalize_iface(interface_name) = n.port_norm_iface
          )
      AND ip.deleted_at IS NULL
      AND ip.status      = 0
LEFT JOIN sys_workstation ws
       ON ws.id::text    = ip.workstation_id::text
      AND ws.deleted_at IS NULL
LEFT JOIN sys_user su
       ON su.id::text    = ws.user_id::text
      AND su.deleted_at IS NULL
WHERE n.asset_norm_mac = n.mac_norm_mac;
`
	if err := db.Exec(createViewSQL).Error; err != nil {
		return fmt.Errorf("重建 reconciliation_physical_chain 视图失败: %w", err)
	}
	applogger.Infof("[迁移] reconciliation_physical_chain 视图已重建(真正折叠 GE/Gi/GigabitEthernet 等跨厂商短名)")

	// 3. 验证 view 抽样 — 折叠后命中率应当 > 0
	var physicalCount int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM reconciliation_physical_chain WHERE physical_user_id IS NOT NULL
	`).Scan(&physicalCount).Error; err != nil {
		return fmt.Errorf("验证 reconciliation_physical_chain 视图失败: %w", err)
	}
	applogger.Infof("[迁移] reconciliation_physical_chain 命中 physical_user_id 的行数: %d (修复前应为 0,现应当 > 0)", physicalCount)

	// 4. 重建 reconciliation_normalized 物化视图
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
	applogger.Infof("[迁移] reconciliation_normalized MV 已重建(用真折叠 view 数据)")

	// 5. 验证 MV 抽样
	var mvPhysicalCount int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM reconciliation_normalized WHERE physical_user_id IS NOT NULL
	`).Scan(&mvPhysicalCount).Error; err != nil {
		return fmt.Errorf("验证 reconciliation_normalized 物化视图失败: %w", err)
	}
	applogger.Infof("[迁移] reconciliation_normalized MV 命中 physical_user_id 的行数: %d (修复前应为 0,现应当 > 0)", mvPhysicalCount)

	// 6. 标记新 view 版本以便 SQL 端验证
	markViewSQL := `COMMENT ON VIEW reconciliation_physical_chain IS 'R5_fix_mac_iface_fold_20260629_v179'`
	if err := db.Exec(markViewSQL).Error; err != nil {
		applogger.Warnf("[迁移] 设置 view 标记失败 (非阻断): %v", err)
	}

	log.Println("Migration 179 completed: reconciliation_physical_chain view with true interface_name folding + reconciliation_normalized MV rebuilt")
	return nil
}