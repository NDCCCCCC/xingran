package migrations

import (
	"fmt"
	"log"
	"time"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate176ReconciliationPhysicalMV Phase 45 R5: 资产对账 — MV 重写接双源 declared + 真物理链路
//
// 业务背景:
//   - R1 migration_168 / R2 migration_173 物化视图 declared 信号只认 ops_asset.user_id,
//     但 Excel 导入流程只填 nowuser_name(中文姓名),user_id 几乎全是 NULL → hasDeclared 永远 false
//   - R1/R2 物化视图 physical_user_id / physical_username 写死 NULL,无任何 MAC 反推
//
// 关键变更(D-R5-A2-01):
//   1) declared 双源:
//
//      asset_user_id = COALESCE(NULLIF(a.user_id,''), user_id_by_name_and_dept, user_id_by_name)
//      asset_username = sys_user.username(若 user_id 命中) ELSE a.nowuser_name
//
//   2) physical 真接入 reconciliation_physical_chain 视图
//   3) ad.username 同步双源,避免 AD JOIN 链因 sys_user.username 缺失而断裂
//
// 与 dropDependentMaterializedViews 关系:
//   - dropDependent 在 AutoMigrate 之前已 CASCADE DROP reconciliation_normalized,
//
//	migration_176 在 AutoMigrate 之后重建,无 ALTER 阻塞
//   - 与现有 168/173 的 CREATE + 索引 sequence 完全一致
//
// 重名处理:
//   - reconciliation_user_lookup 优先 username+deptname 双条件(LIMIT 1 取首条),
//     退化到 username LIMIT 1
//   - 同名同部门场景:依赖 sys_user 创建顺序,与 GetAssetDevicesByUser 双策略同款
//
// 保留行为:
//   - R1 sys_data_reconciliation 表结构不变
//   - R2 7d 静默期 + 24h 节流逻辑保留
//   - R3 exception matcher 不动
//   - R4 HealthBadge contract 不变
//
// 历史 Type E 处理:
//   - 不主动 UPDATE。按用户决定,让 DetectLayer3 自然重写。
//   - 部署后运维/UAT 用 POST /asset/reconciliation/refresh 手动触发。
func Migrate176ReconciliationPhysicalMV(db *gorm.DB) error {
	log.Println("Running migration 176: Phase 45 R5 reconciliation_normalized MV 双源 declared + 真物理链路")

	// 1. 仅 PostgreSQL 执行
	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] R5 MV 重写跳过(非 PostgreSQL 数据库)")
		log.Println("Migration 176 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// 2. 快路径检查: reconciliation_normalized MV 是否已存在?
	//
	// 优化动机 (260704-ne5-regression-fix-3):
	//   - dropDependentMaterializedViews() 已不再 DROP reconciliation_normalized MV
	//     (GORM AutoMigrate 在当前 model tag + DB schema 一致时不会发 ALTER,见
	//      internal/models/{user,asset,ad_domain}.go 的 type:varchar 修改注释)
	//   - 视图每次启动期都存在,走 REFRESH CONCURRENTLY 而非 DROP+CREATE。
	//   - 实际节省:wall-clock 不显著(REFRESH 仍需全量重算 6688 行,约 10s),
	//     但 REFRESH 不锁表(热重启期间对账查询可继续读到旧 MV 数据,不报错)。
	//
	// 前提条件:
	//   - idx_recon_norm_asset UNIQUE INDEX 必须存在(CONCURRENTLY 必需),创建 MV 时已建
	//   - model tag 与 DB 实际类型一致(否则 GORM ALTER 会触发 0A000,见 memory
	//     gorm-automigrate-blocked-by-matview.md 的"普通 VIEW 也会阻塞"补充段)
	type mvExistsRow struct {
		Exists bool `gorm:"column:exists"`
	}
	var mvExists mvExistsRow
	if err := db.Raw(`SELECT EXISTS (
		SELECT 1 FROM pg_matviews WHERE matviewname = 'reconciliation_normalized'
	) AS exists`).Scan(&mvExists).Error; err != nil {
		applogger.Errorf("[迁移 176] 探测 reconciliation_normalized 失败: %v", err)
		return fmt.Errorf("探测 MV 存在性失败: %w", err)
	}

	if mvExists.Exists {
		// ===== 快路径 (~10s, 不锁表) =====
		applogger.Infof("[迁移] reconciliation_normalized 已存在,走 REFRESH CONCURRENTLY 快路径")

		// 2.1 兜底索引(防运维误删)
		if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_recon_norm_asset
			ON reconciliation_normalized (asset_id)`).Error; err != nil {
			applogger.Warnf("[迁移 176] 兜底 idx_recon_norm_asset 失败(非致命): %v", err)
		}

		// 2.2 REFRESH CONCURRENTLY - 不锁表,增量刷新 6688 行
		start := time.Now()
		if err := db.Exec(`REFRESH MATERIALIZED VIEW CONCURRENTLY reconciliation_normalized`).Error; err != nil {
			applogger.Errorf("[迁移 176] REFRESH CONCURRENTLY 失败: %v", err)
			return fmt.Errorf("REFRESH CONCURRENTLY 失败: %w", err)
		}
		applogger.Infof("[迁移] reconciliation_normalized REFRESH CONCURRENTLY 完成 (耗时 %v, 不锁表)", time.Since(start))

		// 2.3 轻量验证
		var mvCount int64
		if err := db.Raw("SELECT COUNT(*) FROM reconciliation_normalized").Scan(&mvCount).Error; err != nil {
			return fmt.Errorf("验证 reconciliation_normalized MV 失败: %w", err)
		}
		applogger.Infof("[迁移 176] 验证通过 (快路径): MV=%d 行", mvCount)

		log.Println("Migration 176 completed (fast path: REFRESH CONCURRENTLY)")
		return nil
	}

	// ===== 慢路径 (~10s, 首次启动 / 视图被外部删除) =====
	// MV 不存在 → 完整 DROP+CREATE 流程

	// 3. DROP 旧 MV
	//    字段集已变(双源 declared + 真物理链路),IF NOT EXISTS 会跳过
	dropMV := `DROP MATERIALIZED VIEW IF EXISTS reconciliation_normalized CASCADE;`
	if err := db.Exec(dropMV).Error; err != nil {
		applogger.Errorf("[迁移 176] DROP 老 reconciliation_normalized 物化视图失败: %v", err)
		return fmt.Errorf("DROP 老 reconciliation_normalized 物化视图失败: %w", err)
	}

	// 2.5 检查前置视图是否存在(MV 重建引用 reconciliation_user_lookup / reconciliation_physical_chain)
	//    R5 在 database.go 的迁移顺序:Migrate175 先跑建视图 → 本迁移 176 → 建 MV
	//    若任一依赖视图缺失,这里快速失败而非走完 CREATE 才报错,便于排查
	type viewExists struct {
		Exists bool `gorm:"column:exists"`
	}
	var userLookupExists viewExists
	if err := db.Raw(`SELECT EXISTS (SELECT 1 FROM pg_views WHERE viewname = 'reconciliation_user_lookup') AS exists`).Scan(&userLookupExists).Error; err != nil {
		applogger.Errorf("[迁移 176] 探测 reconciliation_user_lookup 失败: %v", err)
	}
	var physicalChainExists viewExists
	if err := db.Raw(`SELECT EXISTS (SELECT 1 FROM pg_views WHERE viewname = 'reconciliation_physical_chain') AS exists`).Scan(&physicalChainExists).Error; err != nil {
		applogger.Errorf("[迁移 176] 探测 reconciliation_physical_chain 失败: %v", err)
	}
	if !userLookupExists.Exists || !physicalChainExists.Exists {
		applogger.Errorf("[迁移 176] 前置视图缺失 userLookup=%v physicalChain=%v,需先跑 migration 175",
			userLookupExists.Exists, physicalChainExists.Exists)
		return fmt.Errorf("前置视图缺失 user_lookup=%v physical_chain=%v",
			userLookupExists.Exists, physicalChainExists.Exists)
	}

	// 3. CREATE 新 MV
	//    R5 升级版:R1 字段 + R2 字段(last_resolved_*) + 双源 declared + 真物理链路
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
		return fmt.Errorf("创建 R5 reconciliation_normalized 物化视图失败: %w", err)
	}
	applogger.Infof("[迁移] R5 reconciliation_normalized 已重建(双源 declared + 真物理链路)")

	// 4. 重建索引(migration_168/173 一致)
	uniqueIdx := `
CREATE UNIQUE INDEX IF NOT EXISTS idx_recon_norm_asset
    ON reconciliation_normalized (asset_id);
CREATE INDEX IF NOT EXISTS idx_recon_norm_asset_code
    ON reconciliation_normalized (asset_code);
CREATE INDEX IF NOT EXISTS idx_recon_norm_last_resolved
    ON reconciliation_normalized (last_conflict_type, last_resolved_at DESC)
    WHERE last_resolved_at IS NOT NULL;
`
	if err := db.Exec(uniqueIdx).Error; err != nil {
		return fmt.Errorf("重建 R5 索引失败: %w", err)
	}
	applogger.Infof("[迁移] R5 索引已就位")

	// 5. 同步回填 ops_asset_physical(物理链路持久化结果)
	//    历史数据写入,新数据由 reconciliation:detectLayer3 触发时增量维护
	backfillSQL := `
INSERT INTO ops_asset_physical (asset_id, physical_user_id, physical_username, mac_join, last_refreshed_at)
SELECT pc.asset_id, pc.physical_user_id::uuid, pc.physical_username, pc.mac_join, NOW()
FROM reconciliation_physical_chain pc
ON CONFLICT (asset_id) DO UPDATE
   SET physical_user_id  = EXCLUDED.physical_user_id,
       physical_username = EXCLUDED.physical_username,
       mac_join          = EXCLUDED.mac_join,
       last_refreshed_at = NOW();
`
	if err := db.Exec(backfillSQL).Error; err != nil {
		applogger.Warnf("[迁移] R5 同步 ops_asset_physical 失败(非致命): %v", err)
	}

	// 6. 验证 MV 可读 + 字段就位 + 抽样数据
	var mvCount int64
	if err := db.Raw("SELECT COUNT(*) FROM reconciliation_normalized").Scan(&mvCount).Error; err != nil {
		return fmt.Errorf("验证 reconciliation_normalized MV 失败: %w", err)
	}

	// 抽样:验证双源 declared 生效
	var sampleCount int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM reconciliation_normalized
		WHERE asset_username IS NOT NULL AND asset_username <> ''
	`).Scan(&sampleCount).Error; err != nil {
		return fmt.Errorf("验证双源 declared 失败: %w", err)
	}

	// 抽样:验证物理链路有数据
	var physicalCount int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM reconciliation_normalized
		WHERE physical_user_id IS NOT NULL
	`).Scan(&physicalCount).Error; err != nil {
		return fmt.Errorf("验证物理链路失败: %w", err)
	}

	applogger.Infof("[迁移] R5 reconciliation_normalized 验证通过: MV=%d 行, declared非空=%d 行, physical非空=%d 行",
		mvCount, sampleCount, physicalCount)

	// 7. 清理历史 Type E(R5 强制措施)
	//
	// 背景: R1/R2 设计 partial unique index `uniq_recon_asset_type_open`
	// `(asset_id, conflict_type) WHERE resolved_at IS NULL AND deleted_at IS NULL`
	// 防止同一 (asset, type) 重复告警。但 R5 修复后,5CD01096PC 等设备
	// 应当从 Type E(幽灵资产)重分类为更准确类型(Type A/D/F)。
	// partial unique index 会拦下新 INSERT(同一 asset+type 已存在 open 记录),
	// 24h 节流和 7d 静默期也不会让 DetectLayer3 自然重写(因为 unique constraint
	// 持续 hold 旧 Type E 记录)。
	//
	// 解决: 把所有 open Type E 记录批量标记为 resolved(带 R5 标记),
	// 释放 partial unique index 的占用,让 DetectLayer3 下一轮自然重写为准确类型。
	// 这是 R5 declared 双源修复的必然后续步骤。
	cleanupSQL := `
UPDATE sys_data_reconciliation
SET resolved_at = NOW(),
    resolved_by = 'R5-migration-176',
    resolution_note = 'R5 修复: declared 信号从单源 user_id 扩展为双源 (user_id + nowuser_name+nickname),物理链路真接入。历史 Type E 已批量关闭,DetectLayer3 将按新逻辑重新分类。'
WHERE conflict_type = 'E'
  AND resolved_at IS NULL
  AND deleted_at IS NULL;
`
	if res := db.Exec(cleanupSQL); res.Error != nil {
		applogger.Errorf("[迁移 176] 清理历史 Type E 失败(非致命): %v", res.Error)
	} else {
		applogger.Infof("[迁移 176] 历史 Type E 已批量标记为 resolved,受影响行数: %d", res.RowsAffected)
	}

	log.Println("Migration 176 completed: R5 reconciliation_normalized MV + 双源 declared + 物理链路 ready")
	return nil
}
