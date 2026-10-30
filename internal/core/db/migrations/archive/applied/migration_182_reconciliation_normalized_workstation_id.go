//go:build archive_skip


package migrations

import (
	"fmt"
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate182ReconciliationNormalizedWorkstationID Phase 45 R5 收尾 (2026-06-30):
// reconciliation_normalized MV 增加 workstation_id 列。
//
// 背景(R5 production log):
//   - migration_176 创建的 reconciliation_normalized MV 只 select 了
//     reconciliation_physical_chain.pc.physical_user_id / pc.physical_username,
//     **漏掉了 pc.workstation_id** (虽然 view migration_180:101 暴露了 ws.id AS workstation_id)。
//   - internal/services/asset/reconciliation_workorder.go:WorkstationIDForException
//     查询 reconciliation_normalized.workstation_id → 触发 SQLSTATE 42703
//     "column reconciliation_normalized.workstation_id does not exist"。
//   - WorkstationIDForException 是 R4 工位健康度缓存失效的前置查询(45-02 plan B2 锁定),
//     失败导致 InvalidateWorkstationHealth 拿不到 wsID,新工单创建后工位健康度页面不刷新。
//
// 修复:
//   - DROP MATERIALIZED VIEW reconciliation_normalized CASCADE(同时清掉依赖索引)
//   - 重 CREATE,SELECT pc.workstation_id(从 migration_180 view 取)透传到 MV
//   - 重建唯一索引 + asset_code 索引(与 migration_176 一致)
//   - REFRESH MATERIALIZED VIEW(确保下游查询立即可读)
//
// 不动 view reconciliation_physical_chain — 它早就暴露了 workstation_id(migration_180:101),
// 只是 MV 没 select 它。
//
// 风险:同 migration_176 — DROP CASCADE 后 AUTO_MIGRATE 期间查询 reconciliation_normalized 会
// 失败,但 database.go:dropDependentMaterializedViews() 已经在 AutoMigrate 之前 DROP 过一遍,
// 本迁移是在 AutoMigrate 之后跑,期间只对应用层可见,无并发查询。
func Migrate182ReconciliationNormalizedWorkstationID(db *gorm.DB) error {
	log.Println("Running migration 182: reconciliation_normalized MV 增加 workstation_id 列")

	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] 182 跳过(非 PostgreSQL)")
		log.Println("Migration 182 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// 1. DROP 老 MV(含依赖索引)
	dropSQL := `DROP MATERIALIZED VIEW IF EXISTS reconciliation_normalized CASCADE;`
	if err := db.Exec(dropSQL).Error; err != nil {
		return fmt.Errorf("DROP 老 reconciliation_normalized MV 失败: %w", err)
	}
	applogger.Infof("[迁移] 182 reconciliation_normalized MV 已 DROP(含依赖索引)")

	// 2. CREATE 新 MV(与 migration_176 唯一差异:多 select pc.workstation_id)
	createMV := `
CREATE MATERIALIZED VIEW reconciliation_normalized AS
SELECT DISTINCT ON (a.id)
    a.id                                              AS asset_id,
    a.devicesn                                        AS asset_code,
    a.machine_ip                                      AS asset_ip,
    a.mac1                                            AS mac1,
    a.mac2                                            AS mac2,
    COALESCE(NULLIF(a.mac1,''), NULLIF(a.mac2,''))    AS mac_join,
    -- R5 双源 declared: user_id 优先, 否则 nowuser_name+deptname 兜底
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
    -- R5 真物理链路(MV 真 LEFT JOIN reconciliation_physical_chain)
    pc.workstation_id                                 AS workstation_id,
    pc.physical_user_id                               AS physical_user_id,
    pc.physical_username                              AS physical_username,
    -- AD 字段(DISTINCT 决定只取首条 ad 记录)
    ad.id                                             AS ad_id,
    ad.username                                       AS ad_username,
    ad.is_enabled                                     AS ad_is_enabled,
    -- MV 刷新时间戳
    NOW()                                             AS mv_refreshed_at,
    -- R2 静默期上下文
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
`
	if err := db.Exec(createMV).Error; err != nil {
		return fmt.Errorf("创建 R5 reconciliation_normalized MV(含 workstation_id)失败: %w", err)
	}
	applogger.Infof("[迁移] 182 reconciliation_normalized 已重建(含 workstation_id 列)")

	// 3. 重建索引(与 migration_176 一致)
	indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_recon_norm_asset_id ON reconciliation_normalized (asset_id);`,
		`CREATE INDEX        IF NOT EXISTS idx_recon_norm_asset_code ON reconciliation_normalized (asset_code);`,
		`CREATE INDEX        IF NOT EXISTS idx_recon_norm_last_conflict ON reconciliation_normalized (last_conflict_type, last_resolved_at DESC);`,
		`CREATE INDEX        IF NOT EXISTS idx_recon_norm_workstation_id ON reconciliation_normalized (workstation_id) WHERE workstation_id IS NOT NULL;`,
	}
	for _, idxSQL := range indexes {
		if err := db.Exec(idxSQL).Error; err != nil {
			return fmt.Errorf("创建 reconciliation_normalized 索引失败 [%s]: %w", idxSQL, err)
		}
	}
	applogger.Infof("[迁移] 182 reconciliation_normalized 索引重建完成(4 个,含 workstation_id)")

	// 4. 验证 MV 已含 workstation_id 列 + REFRESH(让 WorkstationIDForException 立即可查)
	//
	// R5 修正(2026-06-30): PG materialized view 的列**不**在 information_schema.columns
	// 里(0 行返回是 PG 标准行为,与 migration_173 验证 last_resolved_at 时踩过的坑同款)。
	// 改用 pg_attribute 查 attnum > 0 排除系统列,attisdropped 过滤已删列,真实反映 MV
	// 列存在性。SELECT a.attname 返回列名字符串,与 "workstation_id" 字面比对,既避免了
	// GORM Scan(&bool) 静默失败,又跳过了 information_schema 不覆盖 MV 的陷阱。
	//
	// 证据链:Step 3 的 CREATE INDEX idx_recon_norm_workstation_id 在 workstation_id 上
	// 已成功(否则会 SQLSTATE 42703 报错),即列一定存在;此处只是把"真在"翻译成"程序看到在"。
	var wsIDAttname string
	if err := db.Raw(`
		SELECT a.attname FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		 WHERE c.relname = 'reconciliation_normalized'
		   AND a.attname = 'workstation_id'
		   AND a.attnum > 0 AND NOT a.attisdropped
		 LIMIT 1
	`).Scan(&wsIDAttname).Error; err != nil {
		return fmt.Errorf("验证 reconciliation_normalized.workstation_id 列存在失败: %w", err)
	}
	if wsIDAttname != "workstation_id" {
		return fmt.Errorf("reconciliation_normalized.workstation_id 列创建失败(pg_attribute 查无)")
	}
	applogger.Infof("[迁移] 182 验证 reconciliation_normalized.workstation_id 列存在 ✓")

	if err := db.Exec(`REFRESH MATERIALIZED VIEW reconciliation_normalized`).Error; err != nil {
		return fmt.Errorf("REFRESH MATERIALIZED VIEW reconciliation_normalized 失败: %w", err)
	}

	var mvCount int64
	if err := db.Raw(`SELECT COUNT(*) FROM reconciliation_normalized`).Scan(&mvCount).Error; err != nil {
		return fmt.Errorf("验证 reconciliation_normalized COUNT 失败: %w", err)
	}
	applogger.Infof("[迁移] 182 reconciliation_normalized REFRESH 完成: %d 行(workstation_id 列已可查)", mvCount)

	log.Println("Migration 182 completed: reconciliation_normalized MV + workstation_id column + 4 indexes")
	return nil
}