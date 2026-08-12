//go:build archive_skip


package migrations

import (
	"fmt"
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate173ReconciliationSilenceMV Phase 43 R2: 扩展 reconciliation_normalized MV
// 加 last_resolved_at / last_resolved_by / last_conflict_type 三字段(D-A3-01),
// 给 DetectLayer3 提供 7d 静默期查询能力。
//
// 业务背景:
//   - R1 (migration_168) MV 仅含物理链路 + AD 字段,无"最近已解决"上下文
//   - R2 需要:运维标记某条异常 resolved 后,7 天内同 (asset, type) 不重复触发
//     (ROADMAP SC 7)。需要在 MV 上 LEFT JOIN LATERAL 取最近一条已解决记录
//   - LEFT JOIN LATERAL 语法来自 D-A3-01 specifics 锁定:
//     LEFT JOIN LATERAL (
//         SELECT resolved_at, resolved_by, conflict_type
//         FROM sys_data_reconciliation r
//         WHERE r.asset_id = a.id
//           AND r.resolved_at IS NOT NULL
//           AND r.deleted_at IS NULL
//         ORDER BY r.resolved_at DESC
//         LIMIT 1
//     ) last_resolved ON true
//
// 与 dropDependentMaterializedViews 的关系:
//   - AutoMigrate 之前 dropDependentMaterializedViews() 会 DROP reconciliation_normalized
//   - AutoMigrate 之后 source tables (ops_asset / sys_user / sys_ad_user / sys_data_reconciliation)
//     已被 GORM ALTER 完毕,此时 CREATE MV 不会被引用阻塞
//   - 所以本 migration 可直接 CREATE(无需再次 DROP,因 dropDependent 已做过)
//
// 关键决策:
//   - DROP MATERIALIZED VIEW IF EXISTS reconciliation_normalized CASCADE — 二次启动时
//     重建,因为字段集已变(新增 last_resolved_*),IF NOT EXISTS 会跳过导致字段缺失
//   - CASCADE 同步 drop 关联 unique index idx_recon_norm_asset,避免重建时冲突
//   - CREATE UNIQUE INDEX ... 必须在 MV 创建后,PostgreSQL 对未索引 MV 限制 REFRESH CONCURRENTLY
//   - idx_recon_norm_last_resolved 部分索引 WHERE last_resolved_at IS NOT NULL:
//     7d 静默期查询 WHERE last_resolved_at > NOW() - INTERVAL '7 day' 命中此索引,
//     避免对全表做时间过滤(99% 行 last_resolved_at IS NULL)
//
// 与现有迁移的协作:
//   - migration_168 创建 MV R1 版本(含物理链路 + AD)
//   - 本 migration_173 升级 MV 含 R1 全部字段 + R2 3 字段
//   - migration_168 的 unique index idx_recon_norm_asset 重建(R1 索引在 MV 上,DROP MV CASCADE 一并 drop)
//   - 索引 idx_recon_norm_asset_code 来自 migration_168,本 migration 不重建(假设 MV DROP CASCADE 后需补建)
//     — 为简化,本 migration 在 CREATE UNIQUE INDEX 之后重建此辅助索引
//
// SQLite 兼容:
//   - SQLite 不支持 MATERIALIZED VIEW,DROP/CREATE 会语法错误
//   - 仅 PostgreSQL 执行 MV 部分(SQLite 跳过);测试场景 setupTestDB 在
//     internal/services/asset/reconciliation_test.go 已用 view 模拟,
//
//	本 migration 不影响 SQLite 测试路径
func Migrate173ReconciliationSilenceMV(db *gorm.DB) error {
	log.Println("Running migration 173: Phase 43 R2 reconciliation_normalized MV + last_resolved_* 字段")

	// 1. 仅 PostgreSQL 执行 MV 重建
	//    SQLite 不支持 MATERIALIZED VIEW,DROP/CREATE 会语法错误,跳过此分支。
	//    测试场景 setupTestDB 在 internal/services/asset/reconciliation_test.go 已用
	//    view 模拟,本 migration 不影响 SQLite 测试路径。
	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] reconciliation_normalized MV + last_resolved_* 跳过(非 PostgreSQL 数据库)")
		log.Println("Migration 173 completed (non-PostgreSQL dialect: MV unchanged)")
		return nil
	}

	// 3. DROP MATERIALIZED VIEW IF EXISTS ... CASCADE
	//    二次启动需重建:字段集已变(新增 last_resolved_*),IF NOT EXISTS 会跳过
	//    CASCADE 一并 drop 关联 unique index idx_recon_norm_asset
	//    (dropDependentMaterializedViews 已在 AutoMigrate 前 DROP,本 DROP 是双保险 —
	//     即使前面没 drop,这里也能继续往下走,符合"idempotent"原则)
	dropMatViewSQL := `DROP MATERIALIZED VIEW IF EXISTS reconciliation_normalized CASCADE;`
	if err := db.Exec(dropMatViewSQL).Error; err != nil {
		return fmt.Errorf("删除旧 reconciliation_normalized 物化视图失败: %w", err)
	}

	// 4. CREATE MATERIALIZED VIEW reconciliation_normalized
	//    R2 升级版:R1 全部字段 + LEFT JOIN LATERAL 取最近一条已解决记录
	//    LEFT JOIN LATERAL 是 PG 9.3+ 特性,语义:对每行 a,执行子查询返回单行
	//    (LIMIT 1),取出 resolved_at/resolved_by/conflict_type
	matViewSQL := `
CREATE MATERIALIZED VIEW reconciliation_normalized AS
SELECT DISTINCT ON (a.id)
    a.id                                              AS asset_id,
    a.devicesn                                        AS asset_code,
    a.machine_ip                                      AS asset_ip,
    a.mac1                                            AS mac1,
    a.mac2                                            AS mac2,
    COALESCE(NULLIF(a.mac1, ''), NULLIF(a.mac2, ''))  AS mac_join,
    a.user_id                                         AS asset_user_id,
    ru.username                                       AS asset_username,
    a.deleted_at                                      AS asset_deleted_at,
    -- 物理链路(MV 中固定 NULL,R2 进一步接入)
    NULL::uuid                                        AS physical_user_id,
    NULL::varchar                                     AS physical_username,
    -- AD 字段(DISTINCT 决定只取首条 ad 记录)
    ad.id                                             AS ad_id,
    ad.username                                       AS ad_username,
    ad.is_enabled                                     AS ad_is_enabled,
    -- MV 刷新时间戳
    NOW()                                             AS mv_refreshed_at,
    -- === Phase 43 R2 / D-A3-01 新增:最近已解决记录 ===
    last_resolved.resolved_at                         AS last_resolved_at,
    last_resolved.resolved_by                         AS last_resolved_by,
    last_resolved.conflict_type                       AS last_conflict_type
FROM ops_asset a
LEFT JOIN sys_user ru ON ru.id = a.user_id::uuid AND ru.deleted_at IS NULL
LEFT JOIN sys_ad_user ad ON ad.username = ru.username
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
	if err := db.Exec(matViewSQL).Error; err != nil {
		return fmt.Errorf("创建 reconciliation_normalized 物化视图(含 last_resolved_*)失败: %w", err)
	}
	applogger.Infof("[迁移] 物化视图 reconciliation_normalized 已重建(含 last_resolved_at / last_resolved_by / last_conflict_type)")

	// 5. unique index on reconciliation_normalized(asset_id)
	//    REFRESH CONCURRENTLY 的前置条件,必须在 MV 创建后
	uniqueIdxSQL := `
CREATE UNIQUE INDEX IF NOT EXISTS idx_recon_norm_asset
    ON reconciliation_normalized (asset_id);
`
	if err := db.Exec(uniqueIdxSQL).Error; err != nil {
		return fmt.Errorf("创建 idx_recon_norm_asset 唯一索引失败: %w", err)
	}
	applogger.Infof("[迁移] idx_recon_norm_asset 唯一索引已就位")

	// 6. 辅助索引:asset_code JOIN lookup(migration_168 已建,这里 IF NOT EXISTS 兜底)
	codeIdxSQL := `
CREATE INDEX IF NOT EXISTS idx_recon_norm_asset_code
    ON reconciliation_normalized (asset_code);
`
	if err := db.Exec(codeIdxSQL).Error; err != nil {
		applogger.Warnf("[迁移] idx_recon_norm_asset_code 创建失败(非致命): %v", err)
	}

	// 7. 部分索引 idx_recon_norm_last_resolved — 加速 7d 静默期查询
	//    WHERE last_resolved_at IS NOT NULL 排除 NULL 行(99% 历史资产无 resolved 记录)
	//    DetectLayer3 查询模式:
	//      SELECT ... FROM reconciliation_normalized
	//      WHERE last_resolved_at IS NOT NULL
	//        AND last_resolved_at > NOW() - INTERVAL '7 day'
	//        AND last_conflict_type = ?
	//    此部分索引显著缩小扫描范围(只索引有 resolved 记录的 asset)
	silenceIdxSQL := `
CREATE INDEX IF NOT EXISTS idx_recon_norm_last_resolved
    ON reconciliation_normalized (last_conflict_type, last_resolved_at DESC)
    WHERE last_resolved_at IS NOT NULL;
`
	if err := db.Exec(silenceIdxSQL).Error; err != nil {
		applogger.Warnf("[迁移] idx_recon_norm_last_resolved 部分索引创建失败(非致命): %v", err)
	}

	// 8. 验证 MV 可读 + 字段就位
	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM reconciliation_normalized").Scan(&count).Error; err != nil {
		return fmt.Errorf("验证 reconciliation_normalized 失败: %w", err)
	}

	// 抽样验证 last_resolved_at 字段存在
	//
	// R5 修正(2026-06-29): PG materialized view 的列**不**在 information_schema.columns
	// 里(0 行返回是 PG 标准行为)。改用 pg_attribute 查 attnum > 0 排除系统列,
	// attisdropped 过滤已删列,真实反映 MV 列存在性。
	//
	// 旧代码用 information_schema.columns,MV 一直返回 0 行被误判为"列缺失",
	// 导致 173 永远报 "MV 缺少 last_resolved_at"。本次 R5 调整后,
	// 验证通过的判定基于 pg_attribute 是否找到 last_resolved_at 这条 attname。
	var lastResolvedAttname string
	if err := db.Raw(`
		SELECT a.attname FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		WHERE c.relname = 'reconciliation_normalized'
		  AND a.attname = 'last_resolved_at'
		  AND a.attnum > 0 AND NOT a.attisdropped
		LIMIT 1
	`).Scan(&lastResolvedAttname).Error; err != nil {
		applogger.Warnf("[迁移 173] 探测 last_resolved_at 列失败(非致命): %v", err)
	} else if lastResolvedAttname != "last_resolved_at" {
		return fmt.Errorf("MV 缺少 last_resolved_at 列(R2/R5 字段),pg_attribute 查无 (migration 176 应已重建)")
	} else {
		applogger.Infof("[迁移] reconciliation_normalized 验证通过: %d 行 + last_resolved_at 列就位", count)
	}

	log.Println("Migration 173 completed: reconciliation_normalized MV R2 字段 + 静默期索引 ready")
	return nil
}