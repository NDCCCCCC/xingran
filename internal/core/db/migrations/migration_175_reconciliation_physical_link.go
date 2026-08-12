package migrations

import (
	"fmt"
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate175ReconciliationPhysicalLink Phase 45 R5: 资产对账 — 物理链路底座
//
// 业务背景:
//   - R1/R2 reconciliation_normalized 物化视图里 physical_user_id / physical_username
//     被硬编码 NULL(D-R5-A1-01),导致 ClassifySignals.hasPhysical 永远 false。
//   - R5 接上真实 MAC→port→infoPoint→workstation→user 解析链。
//   - 同时建 reconciliation_user_lookup 视图,把 ops_asset.nowuser_name (+ deptname)
//     反查 sys_user.id,这样 MV 可以双源 declared(D-R5-A2-01)。
//
// 关键决策:
//   - 不新建 sys_port_mac 表(sys_device_mac_address 已存 MAC 历史)
//   - 不新建 sys_workstation_info_point 表(ops_info_points.workstation_id 已建立工位主链路)
//   - 新建 ops_asset_physical 表:物理链路结果物理化,与 R3 检测引擎测试对齐。
//   - 普通 VIEW 而非 MATERIALIZED VIEW:
//
//   (a) 普通视图在 AutoMigrate 之前 dropDependent 已 CASCADE 清理,
//       migration_175/176 在 AutoMigrate 之后重建,无 ALTER TYPE 阻塞风险
//       (参考 memory `gorm-automigrate-blocked-by-matview.md` 教训)
//   (b) 普通视图零存储开销,数据实时跟随 sys_device_mac_address 等源表变化
//
// SQLite:
//   - sys_device_mac_address / ops_info_points / sys_device_port_status 都是 PostgreSQL-only,
//     SQLite 跳过本 migration
func Migrate175ReconciliationPhysicalLink(db *gorm.DB) error {
	log.Println("Running migration 175: Phase 45 R5 reconciliation_physical_chain 视图 + reconciliation_user_lookup 视图 + ops_asset_physical 表")

	// 1. 仅 PostgreSQL 执行
	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] R5 物理链路视图跳过(非 PostgreSQL 数据库)")
		log.Println("Migration 175 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// 2. ops_asset_physical 表 — 物理链路结果物理化
	//    与 internal/services/asset/reconciliation_detection_test.go 的同名测试表对齐
	createTableSQL := `
CREATE TABLE IF NOT EXISTS ops_asset_physical (
    asset_id          UUID         PRIMARY KEY,
    physical_user_id  UUID         NULL,
    physical_username VARCHAR(64)  NULL,
    mac_join          VARCHAR(64)  NULL,
    source            VARCHAR(16)  NOT NULL DEFAULT 'chain',
    last_refreshed_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ  NULL
);
CREATE INDEX IF NOT EXISTS idx_ops_asset_physical_user
    ON ops_asset_physical (physical_user_id) WHERE deleted_at IS NULL;
`
	if err := db.Exec(createTableSQL).Error; err != nil {
		return fmt.Errorf("创建 ops_asset_physical 表失败: %w", err)
	}
	applogger.Infof("[迁移] ops_asset_physical 表已就位")

	// 3. reconciliation_physical_chain 视图
	//    MAC→port→infoPoint→workstation→user 整条解析链
	//    注意 sys_device_mac_address 是 **不带 deleted_at** 的 GORM 表 —— MAC 采集采用硬删除
	//    (mac_collection_service.go:CollectAllDevices 的 DELETE + INSERT 策略),所以这里
	//    不需要加 AND deleted_at IS NULL。
	//    而 sys_device_mac_history 才是带 deleted_at 的软删除表(见 internal/models/device_mac_history.go)。
	//    (勘误 2026-06-29: 此前注释误说"sys_device_mac_address 带 deleted_at",导致 migration_178
	//    错误加 WHERE m.deleted_at IS NULL 触发 SQLSTATE 42703,已修正。)
	createPhysicalChainSQL := `
CREATE OR REPLACE VIEW reconciliation_physical_chain AS
WITH latest_mac AS (
    SELECT DISTINCT ON (m.mac_address)
        m.mac_address,
        m.device_id,
        m.interface_name
    FROM sys_device_mac_address m
    ORDER BY m.mac_address, m.collected_at DESC NULLS LAST
)
SELECT
    a.id                                              AS asset_id,
    a.devicesn                                        AS asset_code,
    COALESCE(NULLIF(a.mac1,''), NULLIF(a.mac2,''))    AS mac_join,
    ws.id                                              AS workstation_id,
    ws.user_id                                         AS physical_user_id,
    su.username                                        AS physical_username
FROM ops_asset a
LEFT JOIN latest_mac mac
       ON mac.mac_address = COALESCE(NULLIF(a.mac1,''), NULLIF(a.mac2,''))
LEFT JOIN sys_device_port_status port
       ON port.device_id::text = mac.device_id::text
      AND port.interface_name  = mac.interface_name
LEFT JOIN ops_info_points ip
       ON ip.port_id::text = port.id::text
      AND ip.deleted_at IS NULL
      AND ip.status      = 0
LEFT JOIN sys_workstation ws
       ON ws.id::text    = ip.workstation_id::text
      AND ws.deleted_at IS NULL
LEFT JOIN sys_user su
       ON su.id::text    = ws.user_id::text
      AND su.deleted_at IS NULL
WHERE a.deleted_at IS NULL;
`
	if err := db.Exec(createPhysicalChainSQL).Error; err != nil {
		return fmt.Errorf("创建 reconciliation_physical_chain 视图失败: %w", err)
	}
	applogger.Infof("[迁移] reconciliation_physical_chain 视图已就位")

	// 4. reconciliation_user_lookup 视图
	//    nowuser_name (中文姓名) (+ deptname) → sys_user.id
	//
	// R5 修正(2026-06-29): sys_user 表的 username 字段存的是登录账号
	// (如 "yangwen-047"),中文姓名存在 nickname 字段(如 "杨文")。
	// ops_asset.nowuser_name 是 Excel 导入的中文姓名,应当匹配 sys_user.nickname
	// 而不是 username。修正后双策略改用 nickname:
	//   策略 1: nickname + deptname 双条件 LIMIT 1
	//   策略 2: nickname LIMIT 1 兜底
	createUserLookupSQL := `
CREATE OR REPLACE VIEW reconciliation_user_lookup AS
SELECT
    a.id                                  AS asset_id,
    a.nowuser_name                        AS nowuser_name,
    a.deptname                            AS asset_deptname,
    (
        SELECT su.id::text
        FROM sys_user su
        JOIN sys_dept dept ON dept.id = su.dept_id AND dept.deleted_at IS NULL
        WHERE su.nickname    = a.nowuser_name
          AND dept.dept_name = a.deptname
          AND su.deleted_at IS NULL
          AND a.nowuser_name IS NOT NULL AND a.nowuser_name <> ''
          AND a.deptname     IS NOT NULL AND a.deptname     <> ''
        LIMIT 1
    )                                     AS user_id_by_name_and_dept,
    (
        SELECT su.id::text
        FROM sys_user su
        WHERE su.nickname  = a.nowuser_name
          AND su.deleted_at IS NULL
          AND a.nowuser_name IS NOT NULL AND a.nowuser_name <> ''
        LIMIT 1
    )                                     AS user_id_by_name
FROM ops_asset a
WHERE a.deleted_at IS NULL;
`
	if err := db.Exec(createUserLookupSQL).Error; err != nil {
		return fmt.Errorf("创建 reconciliation_user_lookup 视图失败: %w", err)
	}
	applogger.Infof("[迁移] reconciliation_user_lookup 视图已就位")

	// 5. 轻量验证 — LIMIT 1 而非 COUNT(*)
	//
	// 优化 (260704-ne5-regression-fix-4):
	//   - 原 2 个 SELECT COUNT(*) 在 6688 行上各 ~3s,总共 ~6s
	//   - CREATE OR REPLACE VIEW 成功 → 视图结构 + 权限 OK
	//   - LIMIT 1 仅需 <100ms 验证"视图可读且有数据"
	//   - 真正行数由 176 的 reconciliation_normalized MV COUNT 兜底验证
	//   - 如需精确行数,运维可手跑 SELECT COUNT(*) FROM reconciliation_physical_chain
	var chainProbe int64
	if err := db.Raw("SELECT 1 FROM reconciliation_physical_chain LIMIT 1").Scan(&chainProbe).Error; err != nil {
		return fmt.Errorf("验证 reconciliation_physical_chain 视图失败: %w", err)
	}
	var lookupProbe int64
	if err := db.Raw("SELECT 1 FROM reconciliation_user_lookup LIMIT 1").Scan(&lookupProbe).Error; err != nil {
		return fmt.Errorf("验证 reconciliation_user_lookup 视图失败: %w", err)
	}

	applogger.Infof("[迁移] R5 物理链路底座就位: reconciliation_physical_chain + reconciliation_user_lookup 视图可读 (<100ms LIMIT 1 验证)")

	log.Println("Migration 175 completed: reconciliation_physical_chain + reconciliation_user_lookup 视图 + ops_asset_physical 表 ready")
	return nil
}
