//go:build archive_skip


package migrations

import (
	"fmt"
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate180ReconciliationPhysicalChainPerf Phase 45 R5 性能修复 (2026-06-30):
// reconciliation_physical_chain 视图从 O(N×M) 相关子查询重构为 O(N+M) 预算 CTE。
//
// 背景(migration_179 引入的性能回归):
//   - migration_179 把 interface_name 归一化从内联 LOWER(REGEXP_REPLACE) 换成 PL/pgSQL
//     函数 normalize_iface()(10 个顺序 REGEXP_REPLACE,单次约慢 10x)。逻辑正确。
//   - 但视图的 ops_info_points JOIN 用了"相关子查询":
//       ip.port_id::text IN (
//         SELECT id::text FROM sys_device_port_status
//          WHERE device_id::text = n.device_id::text
//            AND normalize_iface(interface_name) = n.mac_norm_iface
//            AND normalize_iface(interface_name) = n.port_norm_iface)
//     子查询引用外层 norm.n → 每个 norm 行重跑一次,全表扫 port_status 且逐行调
//     normalize_iface 2 次 → O(norm_rows × port_status_rows × 2) 函数调用。
//   - 量级估算:N=M=5000 → 5000万次调用 × 10 正则 → 数十分钟级,
//     生产 SELECT COUNT(*) FROM reconciliation_physical_chain 直接触发 statement_timeout。
//   - 加上 device_id::text 强转 UUID → 索引失效,退化为顺序扫描。
//
// 关键洞察:
//   - norm CTE 已经 JOIN 了 sys_device_port_status(为做 device+interface 匹配),只是
//     没把 port.id 透传出来,导致外层不得不再用相关子查询反查 port_status 的 id。
//   - 直接把 port.id 透传,info_points 改 JOIN port.id,整段相关子查询消失。
//
// 修复(纯结构,不动 normalize_iface 函数):
//   1. latest_mac CTE 预算 normalize_iface(interface_name) AS mac_norm_iface(一次)
//   2. 新增 port_norm CTE: 对全表 port_status 预算 normalize_iface(一次,O(N))
//   3. chain CTE: asset→mac→port JOIN 用预算列(norm_iface = mac_norm_iface),
//      device_id 去掉 ::text(uuid=uuid,索引可用),透传 port.id
//   4. 外层 ops_info_points JOIN 改 ip.port_id = chain.port_id::text(普通 JOIN,无相关子查询)
//   5. mac_join 改用已 JOIN 的 a.mac1/a.mac2(消除 2 个相关子查询)
//
// 调用量级:O(N+M) ≈ port_status 行数 + mac 行数(各调一次 normalize_iface),
//          实测 N=M=5000 → ~1万次 × 10 正则 ≈ 亚秒级。
//
// 不动 normalize_iface 函数本身:结构修复后函数成本已非主因,改函数有再次引入
// 42601 类语法 bug 的风险(见 migration_179 v1 教训),保守不改。
//
// SQLite: 依赖 sys_device_port_status / sys_device_mac_address / ops_info_points,
//         均为 PostgreSQL-only,SQLite 跳过。
func Migrate180ReconciliationPhysicalChainPerf(db *gorm.DB) error {
	log.Println("Running migration 180: reconciliation_physical_chain view O(N²)→O(N+M) perf rewrite")

	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] 180 跳过(非 PostgreSQL)")
		log.Println("Migration 180 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// 1. 重写视图:消除相关子查询 + 预算 normalize_iface + 去 device_id::text 强转
	createViewSQL := `
CREATE OR REPLACE VIEW reconciliation_physical_chain AS
WITH latest_mac AS (
    -- 同 MAC 取最新一条(migration_178/179 语义保留),预算 normalize_iface 一次
    SELECT DISTINCT ON (LOWER(REGEXP_REPLACE(COALESCE(m.mac_address, ''), '[.:\-]', '', 'g')))
        m.mac_address,
        m.device_id,
        normalize_iface(m.interface_name) AS mac_norm_iface
      FROM sys_device_mac_address m
     ORDER BY LOWER(REGEXP_REPLACE(COALESCE(m.mac_address, ''), '[.:\-]', '', 'g')),
              m.collected_at DESC NULLS LAST
),
-- 对全表 port_status 预算 normalize_iface(一次,O(N)),避免 JOIN 时逐行调用
port_norm AS (
    SELECT id,
           device_id,
           normalize_iface(interface_name) AS norm_iface
      FROM sys_device_port_status
),
chain AS (
    -- asset → mac(MAC 归一化相等) → port(同 device + 同归一化 interface)
    -- 关键:device_id 两端都是 uuid,去掉 ::text 让索引可用;norm_iface 两端都预算过
    SELECT
        a.id                                                                AS asset_id,
        a.devicesn                                                          AS asset_code,
        LOWER(REGEXP_REPLACE(COALESCE(NULLIF(a.mac1,''), NULLIF(a.mac2,'')), '[.:\-]', '', 'g')) AS asset_norm_mac,
        LOWER(REGEXP_REPLACE(COALESCE(mac.mac_address, ''), '[.:\-]', '', 'g')) AS mac_norm_mac,
        COALESCE(NULLIF(a.mac1,''), NULLIF(a.mac2,''))                       AS mac_join_raw,
        port.id                                                              AS port_id
      FROM ops_asset a
      LEFT JOIN latest_mac mac
             ON LOWER(REGEXP_REPLACE(COALESCE(mac.mac_address, ''), '[.:\-]', '', 'g'))
              = LOWER(REGEXP_REPLACE(COALESCE(NULLIF(a.mac1,''), NULLIF(a.mac2,'')), '[.:\-]', '', 'g'))
      LEFT JOIN port_norm port
             ON port.device_id = mac.device_id       -- uuid = uuid,无需 ::text,索引可用
            AND port.norm_iface = mac.mac_norm_iface  -- 两端预算,无逐行函数调用
)
SELECT
    c.asset_id,
    c.asset_code,
    c.mac_join_raw                                                            AS mac_join,
    ws.id                                                                     AS workstation_id,
    ws.user_id                                                                AS physical_user_id,
    su.username                                                               AS physical_username
FROM chain c
LEFT JOIN ops_info_points ip
       ON ip.port_id = c.port_id::text   -- ip.port_id 为 text,c.port_id 为 uuid;cast 在非索引侧
      AND ip.deleted_at IS NULL
      AND ip.status      = 0
LEFT JOIN sys_workstation ws
       ON ws.id::text    = ip.workstation_id
      AND ws.deleted_at IS NULL
LEFT JOIN sys_user su
       ON su.id::text    = ws.user_id
      AND su.deleted_at IS NULL
WHERE c.asset_norm_mac = c.mac_norm_mac;
`
	if err := db.Exec(createViewSQL).Error; err != nil {
		return fmt.Errorf("重建 reconciliation_physical_chain 视图(O(N+M) 性能版)失败: %w", err)
	}
	applogger.Infof("[迁移] reconciliation_physical_chain 视图已重构(消除相关子查询 + 预算 normalize_iface)")

	// 2. 验证视图抽样 — COUNT 应当秒级返回(修复前会触发 statement_timeout)
	var physicalCount int64
	if err := db.Raw(`SELECT COUNT(*) FROM reconciliation_physical_chain`).Scan(&physicalCount).Error; err != nil {
		return fmt.Errorf("验证 reconciliation_physical_chain 视图 COUNT 失败: %w", err)
	}
	applogger.Infof("[迁移] reconciliation_physical_chain COUNT: %d (修复前此查询会 statement_timeout)", physicalCount)

	var physicalHit int64
	if err := db.Raw(`SELECT COUNT(*) FROM reconciliation_physical_chain WHERE physical_user_id IS NOT NULL`).Scan(&physicalHit).Error; err != nil {
		return fmt.Errorf("验证 reconciliation_physical_chain 命中数失败: %w", err)
	}
	applogger.Infof("[迁移] reconciliation_physical_chain 命中 physical_user_id 的行数: %d (应 > 0,与 179 版一致)", physicalHit)

	// 3. 重建 reconciliation_normalized 物化视图(数据源 view 已更快)
	//    非 CONCURRENTLY:迁移期一次性刷新,锁可接受;view 已快,刷新也快。
	//    后续 cron 走 CONCURRENTLY(core.go:448 30s),view 快后应能在 30s 内完成。
	if err := db.Exec(`REFRESH MATERIALIZED VIEW reconciliation_normalized`).Error; err != nil {
		applogger.Warnf("[迁移] 刷新 reconciliation_normalized MV 失败 (非阻断,留待 cron): %v", err)
	} else {
		applogger.Infof("[迁移] reconciliation_normalized MV 已刷新(用 O(N+M) 性能版 view 数据)")
	}

	// 4. 标记新 view 版本
	markViewSQL := `COMMENT ON VIEW reconciliation_physical_chain IS 'R5_perf_onm_20260630_v180'`
	if err := db.Exec(markViewSQL).Error; err != nil {
		applogger.Warnf("[迁移] 设置 view 标记失败 (非阻断): %v", err)
	}

	log.Println("Migration 180 completed: reconciliation_physical_chain view O(N²)→O(N+M) + MV refreshed")
	return nil
}
