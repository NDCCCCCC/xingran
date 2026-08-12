//go:build archive_skip


package migrations

import (
	"fmt"
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate168ReconciliationTables Phase 42 R1: 资产对账观测底座 — 主表 + 物化视图
//
// 业务背景:
//   - D-01 / D-08: R1 物化视图 `reconciliation_normalized` 反向推导物理链路
//     (ops_asset.mac1 → sys_port_mac → sys_info_point → sys_workstation_info_point → sys_workstation.user_id),
//     用 mac1 优先(mac2 备选)的 COALESCE 语义,避免单资产 2 行。
//   - D-03: R1 IP 字段最小化,asset_ip 直接取 ops_asset.machine_ip 单值,不做多 IP 解析链 / CIDR 例外。
//   - D-11: 防 R1 告警风暴 — partial unique index uniq_recon_asset_type_open
//     (asset_id, conflict_type) WHERE resolved_at IS NULL AND deleted_at IS NULL。
//
// 步骤清单:
//   1. GORM AutoMigrate 创建 sys_data_reconciliation + sys_reconciliation_exception 两张表
//   2. 物化视图 reconciliation_normalized — 反向推导物理链路
//   3. unique index idx_recon_norm_asset (CONCURRENTLY 必需)
//   4. partial unique index uniq_recon_asset_type_open(显式命名,D-11)
//   5. 验证:SELECT COUNT(*) FROM reconciliation_normalized 不报错
//
// R1 已知约束:
//   - sys_port_mac / sys_info_point / sys_workstation_info_point 尚未在项目 schema 中落地,
//     物化视图 SQL 暂以 `ops_asset LEFT JOIN sys_user` 起步,R2 增加物理链路 JOIN 时再扩展。
//     CREATE MATERIALIZED VIEW 在表不存在时会失败,所以先 AutoMigrate 上游表,
//     后续 R2 计划(migration_NNN)若引入新表,本 MV 在 refresh 时自然引入。
//   - 为保证 R1 启动不阻塞,本迁移对 PG-only 步骤做 dialect 校验,SQLite 直接跳过物化视图部分。
//
// 参考实现:
//   - migration_148_create_ops_asset_table.go:DO $$ 块显式命名 unique constraint
//   - migration_152_mac_matview.go:CREATE MATERIALIZED VIEW + UNIQUE INDEX 模式
//   - migration_165_sys_dept_location_alias.go:GORM AutoMigrate + seed 组合
func Migrate168ReconciliationTables(db *gorm.DB) error {
	log.Println("Running migration 168: Phase 42 R1 reconciliation tables + materialized view + unique index")

	// 1. GORM AutoMigrate 两张主表 + 旁路 sys_reconciliation_exception
	//    GORM 会按 tag 创建列(包括 JSONB / INET / CIDR / text[]),然后我们在下面补 partial unique index。
	if err := db.AutoMigrate(
		&models.SysDataReconciliation{},
		&models.SysReconciliationException{},
	); err != nil {
		log.Printf("Migration 168: AutoMigrate reconciliation tables failed: %v", err)
		return fmt.Errorf("AutoMigrate 对账表失败: %w", err)
	}
	log.Println("Migration 168: sys_data_reconciliation + sys_reconciliation_exception 已创建/已存在")

	// 2-5 仅在 PostgreSQL 执行(SQLite 不支持 INET / CIDR / 物化视图)
	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] reconciliation 物化视图 + 部分唯一索引 跳过(非 PostgreSQL 数据库)")
		log.Println("Migration 168 completed (non-PostgreSQL dialect: tables only)")
		return nil
	}

	// 2. 物化视图 reconciliation_normalized
	//    R1 简化版:仅 LEFT JOIN ops_asset ↔ sys_user ↔ sys_ad_user。
	//    物理链路 sys_port_mac/sys_info_point/sys_workstation_info_point 在 R2 引入。
	//
	//    关键修正:
	//      1) sys_user.id (uuid) ≠ ops_asset.user_id (varchar(64)),必须 ::uuid 显式转换
	//         (PG 强类型不会自动 cast,SQLSTATE 42883 "operator does not exist")
	//      2) sys_ad_user.username 可能有重复(同账号在多 OU/历史残留),
	//         用 DISTINCT ON (a.id) 保证每 asset 一行,避免后续 unique index SQLSTATE 23505
	//      3) SELECT 字段顺序:asset 主字段 → JOIN 字段 → 预留字段
	//
	//    DROP IF EXISTS + CREATE:迁移在二次启动时需要应用修正后的 SQL(原 MV 已创建但
	//    数据不符合新 DISTINCT 约束,且 unique index 未创建)。IF NOT EXISTS 会跳过
	//    CREATE 导致旧 MV 仍在;DROP + CREATE 保证每次迁移启动都按最新 SQL 重建。
	dropMatViewSQL := `DROP MATERIALIZED VIEW IF EXISTS reconciliation_normalized;`

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
    -- R2 字段预留(物理链路反推 user_id)
    NULL::uuid                                        AS physical_user_id,
    NULL::varchar                                     AS physical_username,
    -- AD 字段(DISTINCT 决定只取首条 ad 记录)
    ad.id                                             AS ad_id,
    ad.username                                       AS ad_username,
    ad.is_enabled                                     AS ad_is_enabled,
    -- 检测时间戳(MV 刷新时统一)
    NOW()                                             AS mv_refreshed_at
FROM ops_asset a
LEFT JOIN sys_user ru ON ru.id = a.user_id::uuid AND ru.deleted_at IS NULL
LEFT JOIN sys_ad_user ad ON ad.username = ru.username
                       AND ad.deleted_at IS NULL
                       AND ad.is_enabled = TRUE
WHERE a.deleted_at IS NULL
ORDER BY a.id, ad.id NULLS LAST;
`
	if err := db.Exec(dropMatViewSQL).Error; err != nil {
		return fmt.Errorf("删除旧 reconciliation_normalized 物化视图失败: %w", err)
	}

	if err := db.Exec(matViewSQL).Error; err != nil {
		return fmt.Errorf("创建 reconciliation_normalized 物化视图失败: %w", err)
	}
	applogger.Infof("[迁移] 物化视图 reconciliation_normalized 已创建")

	// 3. unique index on reconciliation_normalized(asset_id)
	//    CONCURRENTLY 刷新物化视图的前置条件
	uniqueIdxSQL := `
CREATE UNIQUE INDEX IF NOT EXISTS idx_recon_norm_asset
    ON reconciliation_normalized (asset_id);
`
	if err := db.Exec(uniqueIdxSQL).Error; err != nil {
		return fmt.Errorf("创建 idx_recon_norm_asset 唯一索引失败: %w", err)
	}
	applogger.Infof("[迁移] idx_recon_norm_asset 唯一索引已就位(CONCURRENTLY 刷新前置)")

	// 辅助索引:asset_code JOIN lookup(migration_168 必做,后续 plan 用于按编号查询)
	codeIdxSQL := `
CREATE INDEX IF NOT EXISTS idx_recon_norm_asset_code
    ON reconciliation_normalized (asset_code);
`
	if err := db.Exec(codeIdxSQL).Error; err != nil {
		// 非致命:log 警告即可
		applogger.Warnf("[迁移] idx_recon_norm_asset_code 创建失败(非致命): %v", err)
	}

	// 4. partial unique index uniq_recon_asset_type_open (D-11 防告警风暴)
	//    用 DO $$ 块显式命名,避免 PG 自动命名 `<table>_<col>_key`
	//    与 GORM uniqueIndex 期望的 `uni_*` 命名规范冲突 (xingran-gorm-sql-constraint-naming-conflict 教训)
	//
	//    Idempotency 修复 (debug: recon-uniq-index-23505):
	//    migration_201 (Phase 48) 已 DROP 2 列版 uniq_recon_asset_type_open 并替换为
	//    3 列版 uniq_recon_asset_type_cat_open (asset_id, conflict_type, recon_category)
	//    WHERE resolved_at IS NULL AND deleted_at IS NULL。启动顺序中 168 在 201 之前;
	//    重启场景下 168 第二次跑时 2 列 index 不存在(被 201 DROP),DO $$ 块 IF NOT EXISTS
	//    检查通过,试图 CREATE UNIQUE INDEX 却被现存重复行 (asset_id, conflict_type)
	//    (recon_category=NULL 时 24h 节流兜底不严的历史数据) 阻断 → SQLSTATE 23505。
	//
	//    修复策略:
	//    (a) 如果 3 列版 uniq_recon_asset_type_cat_open 已存在 → 168 已接管,跳过 2 列版创建
	//        (避免无意义重建后又被 201 DROP)
	//    (b) 如果 2 列版不存在 → 先按 (asset_id, conflict_type) 去重
	//        (保留最新 detected_at 一行,其余 DELETE),再 CREATE UNIQUE INDEX 2 列版
	//        (留给 201 后续 DROP + 重建 3 列版)
	partialUniqueSQL := `
DO $$
DECLARE
    three_col_exists BOOLEAN;
BEGIN
    -- (a) 3 列版已存在 (migration_201 已运行) → 跳过 2 列版
    SELECT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'uniq_recon_asset_type_cat_open'
          AND schemaname = 'public'
    ) INTO three_col_exists;

    IF three_col_exists THEN
        RAISE NOTICE 'uniq_recon_asset_type_cat_open 已存在 (migration_201 已运行), 跳过 2 列版 uniq_recon_asset_type_open';
        RETURN;
    END IF;

    -- (b) 2 列版不存在 → 先 dedup 现有重复行 (resolved_at IS NULL AND deleted_at IS NULL)
    --     保留每组 (asset_id, conflict_type) 中 detected_at 最新的 1 行,
    --     其余 DELETE (release 2026-07-04 debug: recon-uniq-index-23505)
    DELETE FROM sys_data_reconciliation
    WHERE id IN (
        SELECT id FROM (
            SELECT id,
                   ROW_NUMBER() OVER (
                       PARTITION BY asset_id, conflict_type
                       ORDER BY detected_at DESC, created_at DESC
                   ) AS rn
            FROM sys_data_reconciliation
            WHERE resolved_at IS NULL AND deleted_at IS NULL
        ) t
        WHERE t.rn > 1
    );

    -- 2 列版不存在则创建 (后续 migration_201 会 DROP 它重建 3 列版)
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'uniq_recon_asset_type_open'
          AND schemaname = 'public'
    ) THEN
        -- 部分唯一索引不能用 ADD CONSTRAINT(只支持 UNIQUE 全列),改用 CREATE UNIQUE INDEX
        EXECUTE 'CREATE UNIQUE INDEX uniq_recon_asset_type_open
                 ON sys_data_reconciliation (asset_id, conflict_type)
                 WHERE resolved_at IS NULL AND deleted_at IS NULL';
    END IF;
END$$;
`
	if err := db.Exec(partialUniqueSQL).Error; err != nil {
		return fmt.Errorf("创建 partial unique index uniq_recon_asset_type_open 失败: %w", err)
	}
	applogger.Infof("[迁移] partial unique index uniq_recon_asset_type_open 已就位(D-11 防告警风暴)")

	// 5. 验证物化视图可读
	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM reconciliation_normalized").Scan(&count).Error; err != nil {
		return fmt.Errorf("验证 reconciliation_normalized 失败: %w", err)
	}
	applogger.Infof("[迁移] reconciliation_normalized 验证通过,初始 %d 行(空 ops_asset 时返回 0)", count)

	log.Println("Migration 168 completed: reconciliation tables + MV + partial unique index ready")
	return nil
}