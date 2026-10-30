package migrations

import (
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// ============================================================================
// migration_217: 部门↔OU 对账普通视图 v_dept_ou_mismatch（Phase 59 / GUARD-02）
//
// D-09/D-10/D-11/D-12/D-13 锁定：
//   - 单视图、UNION ALL 四类口径，每行带 mismatch_type 区分（D-09/D-10）
//   - 普通 SQL VIEW，实时计算（**禁用 MV**，D-13）：无刷新 cron、无缓存、无 TTL
//   - 视图列顺序固定 11 列（D-11），列名由第一分支决定：
//     dept_id, dept_name, dept_deleted_at, ad_config_id, config_name, mapped_ou_dn,
//     user_id, username, user_ad_ou_dn, mismatch_type, mismatch_detail
//
// 双方言：
//   - PG:     CREATE OR REPLACE VIEW（幂等，重复启动不报错）
//   - sqlite: 无 CREATE OR REPLACE VIEW（见 sqlite_reconciliation_views.go:30 注释），
//     必须 DROP VIEW IF EXISTS + CREATE VIEW
//   两侧共用同一份 SELECT body（buildDeptOUMismatchViewSelect），语义不漂移。
//
// ⚠ 与 AutoMigrate 的关系（Pitfall 3，本仓库历史事故：视图阻塞 GORM ALTER TYPE）：
//   本视图引用 sys_user / sys_dept / sys_dept_ou_mapping / sys_ad_config。
//   AutoMigrate 每次启动都可能对这四张表发 ALTER COLUMN TYPE；只要视图还在，
//   PG 会报 SQLSTATE 0A000 "cannot alter type of a column used by a view or rule"
//   导致启动 FATA；sqlite 走 __temp → RENAME 路径时报 "error in view"。
//   因此 database.go 在 AutoMigrate **之前**定向 DROP 本视图
//   （PG dropDependentMaterializedViews / sqlite dropSQLiteReconciliationViews），
//   再由本迁移在 AutoMigrate 之后重建。**三处启动注册路径缺一不可**
//   （PG 分支 / sqlite 分支 / SKIP_AUTOMIGRATE 旁路 BootstrapMissingTables）。
//
// ⚠ SQL 方言红线（不得违反）：
//   - 不得使用 ILIKE / REGEXP_REPLACE / :: 强转（sqlite 不可用）
//   - UNION ALL 每个分支的 ID/时间列一律 CAST(... AS TEXT)
//     （PG 的 uuid 与 varchar 不能直接 UNION，会报 "UNION types ... cannot be matched"）
//   - 布尔列一律写 NOT col，**不得**写 = 0（PG 是 boolean，sqlite 是 INTEGER）
//   - DN 后缀匹配用 SUBSTR + LENGTH，**不得用 LIKE**
//     （OU 名里的 _ / % 会被当通配符污染匹配）
// ============================================================================

const (
	// deptOUMismatchViewName 视图名（D-09 锁定；端点/前端/迁移三处依赖，改名需同步）。
	deptOUMismatchViewName = "v_dept_ou_mismatch"

	// pgDeptOUMismatchViewCreatePrefix PG DDL 前缀。
	// CREATE OR REPLACE VIEW 保证「重复启动 / 定义变更」都不报错、无需先 DROP。
	// ⚠ 该常量是全文件中 PG 建视图字面量的**唯一**处。
	// 注释里刻意不重复写出「前缀 + 视图名」的完整拼串：plan 的验收断言用 grep 计数，
	// 注释自指会把计数从 1 顶到 2（Phase 58 同款矛盾，裁决口径 = 以可证伪断言为准）。
	pgDeptOUMismatchViewCreatePrefix = "CREATE OR REPLACE VIEW v_dept_ou_mismatch AS "

	// sqliteDeptOUMismatchViewCreatePrefix sqlite DDL 前缀（sqlite 只支持 CREATE VIEW）。
	sqliteDeptOUMismatchViewCreatePrefix = "CREATE VIEW v_dept_ou_mismatch AS "

	// deptOUMismatchViewDropSQL DDL 文本常量。
	// 由三处共用：database.go 的 PG AutoMigrate 前置 DROP、sqlite_reconciliation_views.go 的
	// 前置 DROP 列表、以及本文件 sqlite 分支重建前的 DROP。
	// 视图是元数据操作，DROP 瞬时完成；先 DROP 保证定义随代码版本刷新，
	// 不对齐 IF NOT EXISTS 的「旧定义残留」漂移风险。
	deptOUMismatchViewDropSQL = "DROP VIEW IF EXISTS v_dept_ou_mismatch"
)

// pgDeptOUMismatchViewDDL 产出 PG 侧建视图 DDL。
//
// 抽成纯函数是为了**可断言**：sqlite 测试环境跑不到 PG 的 db.Exec 分支，
// 但可以断言本函数的文本产物（TestPhase59_Migration217_PGDialectDDL）。
func pgDeptOUMismatchViewDDL(body string) string {
	return pgDeptOUMismatchViewCreatePrefix + body
}

// sqliteDeptOUMismatchViewDDLs 产出 sqlite 侧 DDL 序列：先 DROP 再 CREATE。
// 返回切片而非单串，是为了让「DROP 必须在 CREATE 之前」这一顺序在类型层面可读、可断言。
func sqliteDeptOUMismatchViewDDLs(body string) []string {
	return []string{
		deptOUMismatchViewDropSQL,
		sqliteDeptOUMismatchViewCreatePrefix + body,
	}
}

// deptOUMismatchNormalizeSQL 生成 DN 归一化的 SQL 表达式，
// 比较口径对齐 addomain.NormalizeDN（internal/services/addomain/dn_util.go:36-56）：
// 整体 TrimSpace + 按 "," 分段 TrimSpace + "=" 两侧去空格 + 属性名与属性值双侧 ToLower。
//
// 实现约束：
//   - 纯字符串替换，**不用 REGEXP_REPLACE**（sqlite 无此函数）
//   - 组内顺序固定：先折叠连续空格，再逐个吃掉逗号/等号两侧空格
//   - 整组连做两遍：单遍只把 "  " 折成 " "，第二遍才能让 " ," 规则命中
//     （例："OU=IT  ,DC=x" → 第一遍 "OU=IT ,DC=x" → 第二遍 "OU=IT,DC=x"）
//   - 最外层 LOWER：NormalizeDN 对 attr 名与属性值都 ToLower
//     （RFC4514 转义序列不含大小写语义，ToLower 安全 —— 见 dn_util.go 注释）
//
// ⚠ 与 NormalizeDN 的已知差异：NormalizeDN 对无 "=" 的段也整段 TrimSpace+ToLower，
// 本表达式对整串做 LOWER + 段边界去空格，对合法 DN 输入结论一致；
// 逐输入等价性由 59-03 的 Go/SQL 一致性测试锁定（本计划只保证形状与 RESEARCH 一致）。
func deptOUMismatchNormalizeSQL(col string) string {
	expr := "TRIM(COALESCE(" + col + ", ''))"
	for i := 0; i < 2; i++ {
		expr = "REPLACE(" + expr + ", '  ', ' ')"
		expr = "REPLACE(" + expr + ", ' ,', ',')"
		expr = "REPLACE(" + expr + ", ', ', ',')"
		expr = "REPLACE(" + expr + ", ' =', '=')"
		expr = "REPLACE(" + expr + ", '= ', '=')"
	}
	return "LOWER(" + expr + ")"
}

// buildDeptOUMismatchViewSelect 返回不含方言分叉的视图 SELECT body（两方言共用）。
//
// 列顺序即 D-11 的 11 列契约；第一分支决定列名，后续 UNION ALL 分支逐列对齐。
//
// 四类口径（D-10）：
//
//	① user_ou_mismatch   —— 用户 ad_ou_dn 与「其路由到的配置」下本部门映射的 ou_dn 不等（归一化比较）
//	② dept_soft_deleted  —— 映射行的 dept_id 指向 sys_dept.deleted_at IS NOT NULL 的部门（部门级）
//	③ mapping_disabled   —— 映射 sync_enabled=false 但该部门仍有活跃用户（部门级）
//	④ config_unrouted    —— 用户 ad_ou_dn 无法归属任何 status=enabled 配置的 base_dn（用户级）
//
// 部门级分支（②③）的 user_id / username / user_ad_ou_dn 恒为 NULL（D-09）。
//
// ⚠ ② 与 ③ 可能对同一条映射行各出一行（软删部门 + 映射停用）—— 这是 UNION ALL 的预期语义，
// 刻意不做互斥裁剪（plan objective 的 Decision Reconciliation 第 4 条）。
//
// ⚠ Known Gap（不扩口径）：D-10 四类**不含**「部门/用户在路由配置下没有任何映射」的用户；
// 本 phase 只做可观测，不实现第五类（plan objective 第 2 条）。
//
// 最长后缀匹配（routed CTE）复刻 addomain.MatchADConfigByDN 的语义（D-12）：
//   - 逗号边界：SUBSTR(...) = ',' || base 且 LENGTH(dn) > LENGTH(base)
//   - 最长优先：best.nbl = MAX(nbl)
//   - 等长并列 → 歧义 → 不可路由：best_cnt.cnt = 1 才产出 routed
//     （Go 侧同名歧义返回 error，两者在「同域重复配置」输入上结论一致）
//
// 一致性测试落在 59-03（本计划只锁定 SQL 侧形状与语义）。
func buildDeptOUMismatchViewSelect() string {
	nBase := deptOUMismatchNormalizeSQL("base_dn")
	nUserOU := deptOUMismatchNormalizeSQL("ad_ou_dn")
	nMappingOU := deptOUMismatchNormalizeSQL("ou_dn")

	return `
WITH cfg AS (
    SELECT id, config_name, base_dn,
           ` + nBase + ` AS nb
    FROM sys_ad_config
    WHERE status = 0
      AND deleted_at IS NULL
      AND TRIM(COALESCE(base_dn, '')) <> ''
),
usr AS (
    SELECT id, username, dept_id, ad_ou_dn,
           ` + nUserOU + ` AS nu
    FROM sys_user
    WHERE deleted_at IS NULL
      AND dept_id IS NOT NULL
      AND ad_ou_dn IS NOT NULL
      AND ad_ou_dn <> ''
),
mp AS (
    SELECT dept_id, ad_config_id, ou_dn, sync_enabled,
           ` + nMappingOU + ` AS nm
    FROM sys_dept_ou_mapping
),
mt AS (
    SELECT u.id AS user_id, c.id AS config_id, LENGTH(c.nb) AS nbl
    FROM usr u
    JOIN cfg c
      ON LENGTH(u.nu) > LENGTH(c.nb)
     AND SUBSTR(u.nu, LENGTH(u.nu) - LENGTH(c.nb)) = ',' || c.nb
),
best AS (
    SELECT user_id, MAX(nbl) AS nbl FROM mt GROUP BY user_id
),
best_cnt AS (
    SELECT m.user_id, COUNT(*) AS cnt
    FROM mt m
    JOIN best b ON b.user_id = m.user_id AND b.nbl = m.nbl
    GROUP BY m.user_id
),
routed AS (
    SELECT b.user_id, m.config_id
    FROM best b
    JOIN best_cnt bc ON bc.user_id = b.user_id AND bc.cnt = 1
    JOIN mt m ON m.user_id = b.user_id AND m.nbl = b.nbl
)
SELECT CAST(u.dept_id AS TEXT)          AS dept_id,
       d.dept_name                      AS dept_name,
       CAST(d.deleted_at AS TEXT)       AS dept_deleted_at,
       CAST(c.id AS TEXT)               AS ad_config_id,
       c.config_name                    AS config_name,
       mp.ou_dn                         AS mapped_ou_dn,
       CAST(u.id AS TEXT)               AS user_id,
       u.username                       AS username,
       u.ad_ou_dn                       AS user_ad_ou_dn,
       'user_ou_mismatch'               AS mismatch_type,
       ('用户 OU「' || u.ad_ou_dn || '」≠ 部门映射 OU「' || mp.ou_dn || '」') AS mismatch_detail
FROM routed r
JOIN usr u ON u.id = r.user_id
JOIN cfg c ON c.id = r.config_id
-- ⚠ 两侧都 CAST：sys_dept_ou_mapping.dept_id 是 uuid（archived DDL 126），
-- sys_user.dept_id 是 varchar（model tag size:64）。PG 没有 uuid = varchar 运算符，
-- 只 CAST 一侧会报 "operator does not exist"；两侧同转 TEXT 则两种类型都成立。
JOIN mp ON CAST(mp.dept_id AS TEXT) = CAST(u.dept_id AS TEXT)
       AND CAST(mp.ad_config_id AS TEXT) = CAST(r.config_id AS TEXT)
LEFT JOIN sys_dept d ON CAST(d.id AS TEXT) = CAST(u.dept_id AS TEXT)
WHERE mp.nm <> u.nu

UNION ALL

SELECT CAST(mp.dept_id AS TEXT)          AS dept_id,
       d.dept_name                       AS dept_name,
       CAST(d.deleted_at AS TEXT)        AS dept_deleted_at,
       CAST(mp.ad_config_id AS TEXT)     AS ad_config_id,
       c.config_name                     AS config_name,
       mp.ou_dn                          AS mapped_ou_dn,
       CAST(NULL AS TEXT)                AS user_id,
       CAST(NULL AS TEXT)                AS username,
       CAST(NULL AS TEXT)                AS user_ad_ou_dn,
       'dept_soft_deleted'               AS mismatch_type,
       ('部门「' || d.dept_name || '」已软删（' || CAST(d.deleted_at AS TEXT) || '），映射仍指向该部门') AS mismatch_detail
FROM mp
JOIN sys_dept d ON d.id = mp.dept_id AND d.deleted_at IS NOT NULL
LEFT JOIN sys_ad_config c ON c.id = mp.ad_config_id

UNION ALL

SELECT CAST(mp.dept_id AS TEXT)          AS dept_id,
       d.dept_name                       AS dept_name,
       CAST(d.deleted_at AS TEXT)        AS dept_deleted_at,
       CAST(mp.ad_config_id AS TEXT)     AS ad_config_id,
       c.config_name                     AS config_name,
       mp.ou_dn                          AS mapped_ou_dn,
       CAST(NULL AS TEXT)                AS user_id,
       CAST(NULL AS TEXT)                AS username,
       CAST(NULL AS TEXT)                AS user_ad_ou_dn,
       'mapping_disabled'                AS mismatch_type,
       ('部门「' || d.dept_name || '」的映射已停用，但仍有用户归属') AS mismatch_detail
FROM mp
JOIN sys_dept d ON d.id = mp.dept_id
LEFT JOIN sys_ad_config c ON c.id = mp.ad_config_id
WHERE NOT mp.sync_enabled
  AND EXISTS (
      SELECT 1 FROM sys_user u2
      WHERE CAST(u2.dept_id AS TEXT) = CAST(mp.dept_id AS TEXT)
        AND u2.deleted_at IS NULL
  )

UNION ALL

SELECT CAST(u.dept_id AS TEXT)           AS dept_id,
       d.dept_name                       AS dept_name,
       CAST(d.deleted_at AS TEXT)        AS dept_deleted_at,
       CAST(NULL AS TEXT)                AS ad_config_id,
       CAST(NULL AS TEXT)                AS config_name,
       CAST(NULL AS TEXT)                AS mapped_ou_dn,
       CAST(u.id AS TEXT)                AS user_id,
       u.username                        AS username,
       u.ad_ou_dn                        AS user_ad_ou_dn,
       'config_unrouted'                 AS mismatch_type,
       ('用户 OU「' || u.ad_ou_dn || '」不匹配任何启用配置的 base_dn') AS mismatch_detail
FROM usr u
LEFT JOIN sys_dept d ON CAST(d.id AS TEXT) = CAST(u.dept_id AS TEXT)
WHERE NOT EXISTS (SELECT 1 FROM routed r WHERE r.user_id = u.id)`
}

// Migrate217DeptOUMismatchView 创建/刷新部门↔OU 对账普通视图（Phase 59 GUARD-02 / D-09~D-13）。
//
// 幂等：
//   - PG 走 CREATE OR REPLACE VIEW，重复执行不报错、定义立即刷新
//   - sqlite 无 CREATE OR REPLACE VIEW，先 DROP VIEW IF EXISTS 再 CREATE VIEW
//
// ⚠ 非阻断调用方：database.go 三处路径均以 applogger.Errorf 档调用（视图缺失只让对账端点
// 500，不影响登录/入队主链路）；对照 ensureADPushQueue 的「失败阻断启动」档。
func Migrate217DeptOUMismatchView(db *gorm.DB) error {
	log.Printf("Running migration 217: create view %s", deptOUMismatchViewName)

	body := buildDeptOUMismatchViewSelect()

	if isPostgreSQL(db) {
		if err := db.Exec(pgDeptOUMismatchViewDDL(body)).Error; err != nil {
			applogger.Errorf("migration 217: create view %s (PG) 失败: %v", deptOUMismatchViewName, err)
			return err
		}
		return nil
	}

	for _, ddl := range sqliteDeptOUMismatchViewDDLs(body) {
		if err := db.Exec(ddl).Error; err != nil {
			applogger.Errorf("migration 217: create view %s (sqlite) 失败: %v", deptOUMismatchViewName, err)
			return err
		}
	}
	return nil
}
