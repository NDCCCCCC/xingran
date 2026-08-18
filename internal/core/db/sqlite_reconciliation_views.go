package db

// sqlite_reconciliation_views.go — sqlite dev 模式下 reconciliation 三视图的等价补建。
//
// 背景(sqlite-recon-normalized-view, 2026-08-18):
//   PG 生产链路的 reconciliation_normalized(MV,migration_176+182)及其前置视图
//   reconciliation_physical_chain / reconciliation_user_lookup(普通 VIEW,migration_175,
//   最终形态见 migration_180)全部由 PG-only 迁移创建,sqlite 分支不执行
//   (179 注释:相关函数/归一化是 PG-only)。上一轮 reconciliation-sqlite-cast-400 修复
//   让 probeMaterializedView() 对 sqlite 诚实探测 sqlite_master,缺视图走设计内 fallback
//   降级(无 rn.physical_username / rn.ad_username 字段)+ 每请求 WARN 噪音。
//   用户决策:sqlite bootstrap 补建等价普通 VIEW,让 dev 环境走完整 MV 路径。
//
// 方言翻译规则(PG → sqlite):
//   - MATERIALIZED VIEW → 普通 VIEW(sqlite 无 MV;普通 VIEW 实时计算,dev 语义更优,
//     无需 REFRESH cron;唯一代价是无法建索引,dev 数据量可接受)
//   - DISTINCT ON (k) ... ORDER BY k, x NULLS LAST
//     → ROW_NUMBER() OVER (PARTITION BY k ORDER BY (x IS NULL), x) + 外层 WHERE rn=1
//     (sqlite 3.25+ 窗口函数;DESC 时 sqlite 默认 NULLs last,与 PG NULLS LAST 一致)
//   - LEFT JOIN LATERAL (SELECT a,b,c ... LIMIT 1) → 3 个关联标量子查询
//   - REGEXP_REPLACE(x, '[.:\-]', '', 'g') → REPLACE(REPLACE(REPLACE(x,'.',''),':',''),'-','')
//   - normalize_iface() PL/pgSQL 函数 → sqliteIfaceFoldCaseSQL() 生成的 CASE 表达式
//     (组内最长前缀优先,与 PG 顺序 REGEXP_REPLACE 语义逐组核对等价,见函数注释)
//   - NOW() → CURRENT_TIMESTAMP(UTC 字符串,NormalizedRow.MVRefreshedAt 可正常扫描)
//   - id::text / ::uuid cast → 直接等值(sqlite 全 TEXT,无强类型)
//   - ad.is_enabled = TRUE → = 1(sqlite bool 存 INTEGER)
//   - 偏差已记录:PG 先 REGEXP_REPLACE 去全部 \s+ 空白,sqlite 仅 REPLACE 去空格
//     (接口名实际不含制表符/换行,dev 可接受)
//
// 幂等策略:DROP VIEW IF EXISTS + CREATE VIEW(sqlite 无 CREATE OR REPLACE VIEW;
// 视图是元数据操作,瞬时完成;先 DROP 保证定义随代码版本刷新,不对齐 IF NOT EXISTS
// 的"旧定义残留"漂移风险)。DROP 顺序为依赖逆序(先 normalized 后前置),CREATE 顺序
// 为依赖正序。sqlite DROP VIEW 无 CASCADE/RESTRICT,总是成功。
//
// 仅 sqlite 分支调用(AutoMigrate 的 else 块);PG 路径零改动。

import (
	"fmt"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// sqliteIfaceFoldCaseSQL 生成 PG normalize_iface()(migration_179)的 sqlite CASE 等价式。
//
// col 必须是已 LOWER + 去空格后的列/表达式(PG 函数首步即 LOWER+去\s+)。
//
// 等价性论证(PG 顺序执行 10 条 REGEXP_REPLACE vs CASE 首个命中分支):
//   - 组内按最长前缀优先排列(如 'tengige' 先于 'tengig',否则 'tengige0/1' 会被
//     错折为 'tee0/1');PG POSIX 正则同位置取最长匹配,语义一致
//   - 组间唯一前缀包含关系是 'vlanif' ⊃ 'vlan'(PG 靠规则顺序,vlanif 先折;
//     且即使二次命中 'vlan' 规则也是幂等自替换);本 CASE 中 vlanif 分支先于 vlan
//   - 其余各组首字母/前缀互不构成包含(ge/gi、te/xe、fo/fge、hge、twe/tw、fa、loop、null),
//     逐组核对无交叉
func sqliteIfaceFoldCaseSQL(col string) string {
	return fmt.Sprintf(`CASE
	WHEN %[1]s LIKE 'gigabitethernet%%' THEN 'ge' || SUBSTR(%[1]s, 16)
	WHEN %[1]s LIKE 'gigabitether%%' THEN 'ge' || SUBSTR(%[1]s, 13)
	WHEN %[1]s LIKE 'gigabite%%' THEN 'ge' || SUBSTR(%[1]s, 10)
	WHEN %[1]s LIKE 'gi%%' THEN 'ge' || SUBSTR(%[1]s, 3)
	WHEN %[1]s LIKE 'ge%%' THEN 'ge' || SUBSTR(%[1]s, 3)
	WHEN %[1]s LIKE 'tengigabit%%' THEN 'te' || SUBSTR(%[1]s, 11)
	WHEN %[1]s LIKE 'tengige%%' THEN 'te' || SUBSTR(%[1]s, 8)
	WHEN %[1]s LIKE 'tengig%%' THEN 'te' || SUBSTR(%[1]s, 7)
	WHEN %[1]s LIKE 'tenge%%' THEN 'te' || SUBSTR(%[1]s, 6)
	WHEN %[1]s LIKE 'te%%' THEN 'te' || SUBSTR(%[1]s, 3)
	WHEN %[1]s LIKE 'xe%%' THEN 'te' || SUBSTR(%[1]s, 3)
	WHEN %[1]s LIKE 'fortygigabit%%' THEN 'fo' || SUBSTR(%[1]s, 12)
	WHEN %[1]s LIKE 'fortygige%%' THEN 'fo' || SUBSTR(%[1]s, 10)
	WHEN %[1]s LIKE 'fortygig%%' THEN 'fo' || SUBSTR(%[1]s, 9)
	WHEN %[1]s LIKE 'fge%%' THEN 'fo' || SUBSTR(%[1]s, 4)
	WHEN %[1]s LIKE 'fo%%' THEN 'fo' || SUBSTR(%[1]s, 3)
	WHEN %[1]s LIKE 'hundredgigabit%%' THEN 'hge' || SUBSTR(%[1]s, 14)
	WHEN %[1]s LIKE 'hundredgige%%' THEN 'hge' || SUBSTR(%[1]s, 12)
	WHEN %[1]s LIKE 'hundredgig%%' THEN 'hge' || SUBSTR(%[1]s, 11)
	WHEN %[1]s LIKE 'hge%%' THEN 'hge' || SUBSTR(%[1]s, 4)
	WHEN %[1]s LIKE 'twentyfivegigabit%%' THEN 'twe' || SUBSTR(%[1]s, 17)
	WHEN %[1]s LIKE 'twentyfivegige%%' THEN 'twe' || SUBSTR(%[1]s, 15)
	WHEN %[1]s LIKE 'twentyfivegig%%' THEN 'twe' || SUBSTR(%[1]s, 14)
	WHEN %[1]s LIKE 'twe%%' THEN 'twe' || SUBSTR(%[1]s, 4)
	WHEN %[1]s LIKE 'tw%%' THEN 'twe' || SUBSTR(%[1]s, 3)
	WHEN %[1]s LIKE 'fastethernet%%' THEN 'fa' || SUBSTR(%[1]s, 13)
	WHEN %[1]s LIKE 'fa%%' THEN 'fa' || SUBSTR(%[1]s, 3)
	WHEN %[1]s LIKE 'vlanif%%' THEN 'vlanif' || SUBSTR(%[1]s, 7)
	WHEN %[1]s LIKE 'vlan%%' THEN 'vlan' || SUBSTR(%[1]s, 5)
	WHEN %[1]s LIKE 'vl%%' THEN 'vlan' || SUBSTR(%[1]s, 3)
	WHEN %[1]s LIKE 'loopback%%' THEN 'loop' || SUBSTR(%[1]s, 9)
	WHEN %[1]s LIKE 'null%%' THEN 'null' || SUBSTR(%[1]s, 5)
	ELSE %[1]s
END`, col)
}

// sqliteMACNormSQL 等价 PG LOWER(REGEXP_REPLACE(x, '[.:\-]', '', 'g'))(MAC 去分隔符+小写)。
func sqliteMACNormSQL(col string) string {
	return fmt.Sprintf("LOWER(REPLACE(REPLACE(REPLACE(%s, '.', ''), ':', ''), '-', ''))", col)
}

// sqliteLowerStripSQL 等价 PG LOWER(REGEXP_REPLACE(x, '\s+', '', 'g')) 的近似
// (仅去空格;接口名实际不含其他空白,偏差已在文件头注释记录)。
func sqliteLowerStripSQL(col string) string {
	return fmt.Sprintf("LOWER(REPLACE(COALESCE(%s, ''), ' ', ''))", col)
}

// dropSQLiteReconciliationViews 仅 DROP 三视图(不重建)。用于 AutoMigrate 之前:
// GORM ALTER TABLE(sys_data_reconciliation 加 recon_category 列)在 sqlite 下走
// __temp → RENAME 路径,而 RENAME 会因 reconciliation_normalized 视图持有
// schema-level 引用而失败:
//   "SQL logic error: error in view reconciliation_normalized:
//    no such table: main.sys_data_reconciliation"
// 必须先 DROP 视图让 ALTER 路径无依赖完成,AutoMigrate 之后再调
// ensureSQLiteReconciliationViews 重建。
//
// 仅 sqlite 分支调用;PG 路径零改动(走 PG 自己的 advisory-lock + MV 重建流)。
func (d *Database) dropSQLiteReconciliationViews() error {
	if d.Type != "sqlite" {
		return nil
	}
	drops := []string{
		"DROP VIEW IF EXISTS reconciliation_normalized",
		"DROP VIEW IF EXISTS reconciliation_physical_chain",
		"DROP VIEW IF EXISTS reconciliation_user_lookup",
	}
	for _, stmt := range drops {
		if err := d.DB.Exec(stmt).Error; err != nil {
			return fmt.Errorf("sqlite 视图清理失败 (%s): %w", stmt, err)
		}
	}
	applogger.Infof("[sqlite] reconciliation 三视图已 DROP(为 AutoMigrate 腾出 ALTER 路径)")
	return nil
}

// ensureSQLiteReconciliationViews 在 sqlite 分支补建 reconciliation 三视图。
// 依赖表由 AutoMigrate 先行创建(本函数在 AutoMigrate 成功后的 else 块调用)。
func (d *Database) ensureSQLiteReconciliationViews() error {
	if d.Type != "sqlite" {
		return nil
	}

	ifaceFold := sqliteIfaceFoldCaseSQL("s")

	// ---- reconciliation_user_lookup(翻译 migration_175,id::text → 直接等值)----
	userLookupDDL := `
CREATE VIEW reconciliation_user_lookup AS
SELECT
    a.id           AS asset_id,
    a.nowuser_name AS nowuser_name,
    a.deptname     AS asset_deptname,
    (
        SELECT su.id
        FROM sys_user su
        JOIN sys_dept dept ON dept.id = su.dept_id AND dept.deleted_at IS NULL
        WHERE su.nickname    = a.nowuser_name
          AND dept.dept_name = a.deptname
          AND su.deleted_at IS NULL
          AND a.nowuser_name IS NOT NULL AND a.nowuser_name <> ''
          AND a.deptname     IS NOT NULL AND a.deptname     <> ''
        LIMIT 1
    ) AS user_id_by_name_and_dept,
    (
        SELECT su.id
        FROM sys_user su
        WHERE su.nickname  = a.nowuser_name
          AND su.deleted_at IS NULL
          AND a.nowuser_name IS NOT NULL AND a.nowuser_name <> ''
        LIMIT 1
    ) AS user_id_by_name
FROM ops_asset a
WHERE a.deleted_at IS NULL`

	// ---- reconciliation_physical_chain(翻译 migration_180 最终形态)----
	// DISTINCT ON (norm_mac) → ROW_NUMBER 窗口;normalize_iface → CASE;::text 去除。
	// 注意:子查询最多两层 —— 三层时 sqlite 扁平化会让外层 WHERE 引用的窗口结果列
	// 解析失败(实测 "no such column");故 latest_mac 为「内层算 s+_rn,外层 CASE+过滤」。
	// %[2]s 带 m. 限定符(仅 latest_mac 内层有 m 别名);port_norm 内层无别名,用 %[6]s。
	physicalChainDDL := fmt.Sprintf(`
CREATE VIEW reconciliation_physical_chain AS
WITH latest_mac AS (
    SELECT mac_address, device_id,
           %[1]s AS mac_norm_iface
    FROM (
        SELECT m.mac_address,
               m.device_id,
               %[2]s AS s,
               ROW_NUMBER() OVER (
                   PARTITION BY %[3]s
                   ORDER BY m.collected_at DESC
               ) AS _rn
        FROM sys_device_mac_address m
    )
    WHERE _rn = 1
),
port_norm AS (
    SELECT id, device_id, %[1]s AS norm_iface
    FROM (
        SELECT id, device_id, %[6]s AS s
        FROM sys_device_port_status
    )
),
chain AS (
    SELECT
        a.id         AS asset_id,
        a.devicesn   AS asset_code,
        %[4]s AS asset_norm_mac,
        %[5]s AS mac_norm_mac,
        COALESCE(NULLIF(a.mac1, ''), NULLIF(a.mac2, '')) AS mac_join_raw,
        port.id      AS port_id
    FROM ops_asset a
    LEFT JOIN latest_mac mac
           ON %[5]s = %[4]s
    LEFT JOIN port_norm port
           ON port.device_id  = mac.device_id
          AND port.norm_iface = mac.mac_norm_iface
)
SELECT
    c.asset_id,
    c.asset_code,
    c.mac_join_raw AS mac_join,
    ws.id          AS workstation_id,
    ws.user_id     AS physical_user_id,
    su.username    AS physical_username
FROM chain c
LEFT JOIN ops_info_points ip
       ON ip.port_id = c.port_id
      AND ip.deleted_at IS NULL
      AND ip.status      = 0
LEFT JOIN sys_workstation ws
       ON ws.id = ip.workstation_id
      AND ws.deleted_at IS NULL
LEFT JOIN sys_user su
       ON su.id = ws.user_id
      AND su.deleted_at IS NULL
WHERE c.asset_norm_mac = c.mac_norm_mac`,
		ifaceFold,
		sqliteLowerStripSQL("m.interface_name"),
		sqliteMACNormSQL("COALESCE(m.mac_address, '')"),
		sqliteMACNormSQL("COALESCE(NULLIF(a.mac1, ''), NULLIF(a.mac2, ''))"),
		sqliteMACNormSQL("COALESCE(mac.mac_address, '')"),
		sqliteLowerStripSQL("interface_name"),
	)

	// ---- reconciliation_normalized(翻译 migration_176 + 182 的 workstation_id)----
	// DISTINCT ON (a.id) → ROW_NUMBER 窗口(偏好非空 ad.id);LATERAL → 关联标量子查询;
	// NOW() → CURRENT_TIMESTAMP;is_enabled = TRUE → = 1。
	normalizedDDL := `
CREATE VIEW reconciliation_normalized AS
SELECT
    asset_id, asset_code, asset_ip, mac1, mac2, mac_join,
    asset_user_id, asset_username, asset_deleted_at,
    workstation_id, physical_user_id, physical_username,
    ad_id, ad_username, ad_is_enabled,
    mv_refreshed_at,
    last_resolved_at, last_resolved_by, last_conflict_type
FROM (
    SELECT
        a.id           AS asset_id,
        a.devicesn     AS asset_code,
        a.machine_ip   AS asset_ip,
        a.mac1         AS mac1,
        a.mac2         AS mac2,
        COALESCE(NULLIF(a.mac1, ''), NULLIF(a.mac2, '')) AS mac_join,
        COALESCE(
            NULLIF(a.user_id, ''),
            (SELECT user_id_by_name_and_dept FROM reconciliation_user_lookup WHERE asset_id = a.id LIMIT 1),
            (SELECT user_id_by_name          FROM reconciliation_user_lookup WHERE asset_id = a.id LIMIT 1)
        ) AS asset_user_id,
        COALESCE(
            (SELECT username FROM sys_user WHERE id = NULLIF(a.user_id, '') LIMIT 1),
            a.nowuser_name
        ) AS asset_username,
        a.deleted_at   AS asset_deleted_at,
        pc.workstation_id    AS workstation_id,
        pc.physical_user_id  AS physical_user_id,
        pc.physical_username AS physical_username,
        ad.id          AS ad_id,
        ad.username    AS ad_username,
        ad.is_enabled  AS ad_is_enabled,
        CURRENT_TIMESTAMP AS mv_refreshed_at,
        (SELECT r.resolved_at FROM sys_data_reconciliation r
          WHERE r.asset_id = a.id AND r.resolved_at IS NOT NULL AND r.deleted_at IS NULL
          ORDER BY r.resolved_at DESC LIMIT 1) AS last_resolved_at,
        (SELECT r.resolved_by FROM sys_data_reconciliation r
          WHERE r.asset_id = a.id AND r.resolved_at IS NOT NULL AND r.deleted_at IS NULL
          ORDER BY r.resolved_at DESC LIMIT 1) AS last_resolved_by,
        (SELECT r.conflict_type FROM sys_data_reconciliation r
          WHERE r.asset_id = a.id AND r.resolved_at IS NOT NULL AND r.deleted_at IS NULL
          ORDER BY r.resolved_at DESC LIMIT 1) AS last_conflict_type,
        ROW_NUMBER() OVER (PARTITION BY a.id ORDER BY (ad.id IS NULL), ad.id) AS _dedup_rn
    FROM ops_asset a
    LEFT JOIN reconciliation_physical_chain pc ON pc.asset_id = a.id
    LEFT JOIN sys_ad_user ad
           ON ad.username = COALESCE(
                  (SELECT username FROM sys_user WHERE id = NULLIF(a.user_id, '') LIMIT 1),
                  a.nowuser_name
              )
          AND ad.deleted_at IS NULL
          AND ad.is_enabled = 1
    WHERE a.deleted_at IS NULL
) WHERE _dedup_rn = 1`

	// DROP 逆序(先摘掉依赖方),CREATE 正序(先建被依赖方)。
	drops := []string{
		"DROP VIEW IF EXISTS reconciliation_normalized",
		"DROP VIEW IF EXISTS reconciliation_physical_chain",
		"DROP VIEW IF EXISTS reconciliation_user_lookup",
	}
	creates := []struct {
		name string
		ddl  string
	}{
		{"reconciliation_user_lookup", userLookupDDL},
		{"reconciliation_physical_chain", physicalChainDDL},
		{"reconciliation_normalized", normalizedDDL},
	}

	for _, stmt := range drops {
		if err := d.DB.Exec(stmt).Error; err != nil {
			return fmt.Errorf("sqlite 视图清理失败 (%s): %w", stmt, err)
		}
	}
	for _, c := range creates {
		if err := d.DB.Exec(c.ddl).Error; err != nil {
			return fmt.Errorf("sqlite 视图创建失败 (%s): %w", c.name, err)
		}
	}

	applogger.Infof("[sqlite] reconciliation 三视图已补建 (user_lookup / physical_chain / normalized),exception/list 走完整路径")
	return nil
}
