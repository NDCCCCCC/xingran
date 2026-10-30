package addomain

import (
	"context"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core/db/migrations"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============================================================================
// Phase 59 / GUARD-02：对账只读服务测试（分页 / 筛选 / LIKE 转义 / 白名单）
//                       + Go↔SQL 的 config_unrouted 口径一致性（D-12 核心交付）
//
// 视图不手写 DDL：一律调用 migrations.Migrate217DeptOUMismatchView ——
// 视图定义只有一份事实源，测试用的视图与生产启动建出的视图不可能漂移。
// 源表用原生 DDL（不走 GORM AutoMigrate）：sqlite 测试路径不经
// sanitizeSQLiteModelDefaults，model 里的 gen_random_uuid() / type:uuid 等
// PG-only 片段会直接报语法错误（同 migration_217 测试的口径）。
// ============================================================================

// phase59Statements 视图引用的四张表的最小列集。
var phase59Statements = []string{
	`CREATE TABLE sys_user (
	    id TEXT PRIMARY KEY,
	    username TEXT,
	    dept_id TEXT,
	    ad_ou_dn TEXT,
	    deleted_at TEXT
	)`,
	`CREATE TABLE sys_dept (
	    id TEXT PRIMARY KEY,
	    dept_name TEXT,
	    deleted_at TEXT
	)`,
	`CREATE TABLE sys_dept_ou_mapping (
	    dept_id TEXT,
	    ad_config_id TEXT,
	    ou_dn TEXT,
	    sync_enabled INTEGER DEFAULT 1
	)`,
	`CREATE TABLE sys_ad_config (
	    id TEXT PRIMARY KEY,
	    config_name TEXT,
	    base_dn TEXT,
	    status INTEGER DEFAULT 0,
	    deleted_at TEXT
	)`,
}

// freshPhase59DB 建内存 sqlite 库、建四张源表并建出 v_dept_ou_mismatch 视图。
func freshPhase59DB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "打开 sqlite 失败")

	for i, stmt := range phase59Statements {
		require.NoError(t, db.Exec(stmt).Error, "建表 %d 失败", i)
	}
	require.NoError(t, migrations.Migrate217DeptOUMismatchView(db), "建视图失败")
	return db
}

// phase59Exec 执行一批 SQL，任一条失败即终止用例。
func phase59Exec(t *testing.T, db *gorm.DB, stmts []string) {
	t.Helper()
	for i, stmt := range stmts {
		require.NoError(t, db.Exec(stmt).Error, "执行第 %d 条 SQL 失败: %s", i, stmt)
	}
}

// phase59FourClassSeed 四类口径各恰一行的受控 fixture（与 migration_217 测试同构）。
func phase59FourClassSeed() []string {
	return []string{
		`INSERT INTO sys_ad_config (id, config_name, base_dn, status, deleted_at)
		 VALUES ('cfg-1', '主域', 'DC=corp,DC=com', 0, NULL)`,
		`INSERT INTO sys_dept (id, dept_name, deleted_at) VALUES ('dept-ok', 'IT部', NULL)`,
		`INSERT INTO sys_dept (id, dept_name, deleted_at) VALUES ('dept-gone', '已删部门', '2026-01-01 00:00:00')`,
		`INSERT INTO sys_dept (id, dept_name, deleted_at) VALUES ('dept-off', '停用映射部门', NULL)`,
		`INSERT INTO sys_dept_ou_mapping (dept_id, ad_config_id, ou_dn, sync_enabled)
		 VALUES ('dept-ok',   'cfg-1', 'OU=IT,DC=corp,DC=com',   1)`,
		`INSERT INTO sys_dept_ou_mapping (dept_id, ad_config_id, ou_dn, sync_enabled)
		 VALUES ('dept-gone', 'cfg-1', 'OU=Gone,DC=corp,DC=com', 1)`,
		`INSERT INTO sys_dept_ou_mapping (dept_id, ad_config_id, ou_dn, sync_enabled)
		 VALUES ('dept-off',  'cfg-1', 'OU=Off,DC=corp,DC=com',  0)`,
		`INSERT INTO sys_user (id, username, dept_id, ad_ou_dn, deleted_at)
		 VALUES ('u-mismatch', 'zhangsan', 'dept-ok', 'OU=Sales,DC=corp,DC=com', NULL)`,
		`INSERT INTO sys_user (id, username, dept_id, ad_ou_dn, deleted_at)
		 VALUES ('u-off', 'lisi', 'dept-off', 'OU=Off,DC=corp,DC=com', NULL)`,
		`INSERT INTO sys_user (id, username, dept_id, ad_ou_dn, deleted_at)
		 VALUES ('u-unrouted', 'wangwu', 'dept-ok', 'OU=X,DC=other,DC=com', NULL)`,
	}
}

// TestPhase59_EscapeLike 锁定 LIKE 元字符转义（顺序敏感：反斜杠必须先转）。
func TestPhase59_DeptOUMismatch_EscapeLike(t *testing.T) {
	assert.Equal(t, `a\%b`, escapeLike("a%b"), "% 必须被转义")
	assert.Equal(t, `a\_b`, escapeLike("a_b"), "_ 必须被转义")
	assert.Equal(t, `a\\b`, escapeLike(`a\b`), "反斜杠必须被转义")
	// 双重转义缺陷：先转 % 再转 \ 会把 \% 变成 \\%（% 重新变回通配符）
	assert.Equal(t, `\\\%`, escapeLike(`\%`), "反斜杠必须最先转义，否则产生双重转义")
	assert.Equal(t, "plain", escapeLike("plain"), "无元字符时原样返回")
}

// TestPhase59_ValidMismatchType 锁定四值白名单。
func TestPhase59_DeptOUMismatch_ValidMismatchType(t *testing.T) {
	for _, mt := range MismatchTypes {
		assert.True(t, validMismatchType(mt), "%s 应在白名单内", mt)
	}
	assert.False(t, validMismatchType("bogus"), "非法值必须判 false（handler 据此返回 400）")
	assert.False(t, validMismatchType(""), "空串不属于白名单（空 = 不筛选，由调用方短路）")
	assert.False(t, validMismatchType("USER_OU_MISMATCH"), "白名单是精确匹配，不做大小写归一")
	assert.Len(t, MismatchTypes, 4, "D-10 恰好四类口径")
}

// TestPhase59_DeptOUMismatch_FourClasses 锁定服务层能查出四类口径，
// 且分页返回的四行互不重复（确定性排序 = 翻页不漏不重）。
func TestPhase59_DeptOUMismatch_FourClasses(t *testing.T) {
	db := freshPhase59DB(t)
	phase59Exec(t, db, phase59FourClassSeed())

	svc := NewDeptOUMismatchService(db)
	rows, total, err := svc.ListDeptOUMismatch(context.Background(), &DeptOUMismatchQuery{
		Current: 1, PageSize: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(4), total, "四类口径各一行")

	byType := map[string]int{}
	for _, r := range rows {
		byType[r.MismatchType]++
	}
	for _, mt := range MismatchTypes {
		assert.Equal(t, 1, byType[mt], "mismatch_type=%s 应恰一行", mt)
	}

	// 分页边界稳定：两页各 2 行，并集必须等于全集且无重复
	page1, total1, err := svc.ListDeptOUMismatch(context.Background(), &DeptOUMismatchQuery{Current: 1, PageSize: 2})
	require.NoError(t, err)
	page2, total2, err := svc.ListDeptOUMismatch(context.Background(), &DeptOUMismatchQuery{Current: 2, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(4), total1)
	assert.Equal(t, int64(4), total2)
	require.Len(t, page1, 2)
	require.Len(t, page2, 2)

	seen := map[string]bool{}
	for _, r := range append(append([]DeptOUMismatchRow{}, page1...), page2...) {
		key := r.MismatchType + "|" + derefOr(r.Username)
		assert.False(t, seen[key], "翻页出现重复行: %s", key)
		seen[key] = true
	}
	assert.Len(t, seen, 4, "两页并集应覆盖全部 4 行")
}

// TestPhase59_DeptOUMismatch_MismatchTypeFilter 锁定类型筛选只返回对应类。
func TestPhase59_DeptOUMismatch_MismatchTypeFilter(t *testing.T) {
	db := freshPhase59DB(t)
	phase59Exec(t, db, phase59FourClassSeed())

	svc := NewDeptOUMismatchService(db)
	for _, mt := range MismatchTypes {
		rows, total, err := svc.ListDeptOUMismatch(context.Background(), &DeptOUMismatchQuery{
			Current: 1, PageSize: 10, MismatchType: mt,
		})
		require.NoError(t, err)
		require.Equal(t, int64(1), total, "mismatch_type=%s 应恰一行", mt)
		require.Len(t, rows, 1)
		assert.Equal(t, mt, rows[0].MismatchType)
	}
}

// TestPhase59_DeptOUMismatch_DeptIDFilter 锁定部门筛选。
func TestPhase59_DeptOUMismatch_DeptIDFilter(t *testing.T) {
	db := freshPhase59DB(t)
	phase59Exec(t, db, phase59FourClassSeed())

	svc := NewDeptOUMismatchService(db)
	rows, total, err := svc.ListDeptOUMismatch(context.Background(), &DeptOUMismatchQuery{
		Current: 1, PageSize: 10, DeptID: "dept-off",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total, "dept-off 只有一条映射停用记录")
	require.Len(t, rows, 1)
	assert.Equal(t, "mapping_disabled", rows[0].MismatchType)
	require.NotNil(t, rows[0].DeptID)
	assert.Equal(t, "dept-off", *rows[0].DeptID)
}

// TestPhase59_DeptOUMismatch_KeywordLiteralAndCaseInsensitive 锁定：
//   - keyword 大小写不敏感（对 username / dept_name / mapped_ou_dn / user_ad_ou_dn 包含匹配）
//   - keyword 里的 `_` / `%` 被当作**字面量**，不退化成 LIKE 通配符
//
// 若 `_` 未转义，`%_%` 会匹配所有非空字符串（两行全中）；
// 若 `%` 未转义，`%%%` 会匹配所有行 —— 两条断言都能把该类缺陷变红。
func TestPhase59_DeptOUMismatch_KeywordLiteralAndCaseInsensitive(t *testing.T) {
	db := freshPhase59DB(t)
	phase59Exec(t, db, []string{
		`INSERT INTO sys_ad_config (id, config_name, base_dn, status, deleted_at)
		 VALUES ('cfg-1', '主域', 'DC=corp,DC=com', 0, NULL)`,
		`INSERT INTO sys_dept (id, dept_name, deleted_at) VALUES ('dept-1', 'IT部', NULL)`,
		`INSERT INTO sys_dept_ou_mapping (dept_id, ad_config_id, ou_dn, sync_enabled)
		 VALUES ('dept-1', 'cfg-1', 'OU=IT,DC=corp,DC=com', 1)`,
		`INSERT INTO sys_user (id, username, dept_id, ad_ou_dn, deleted_at)
		 VALUES ('u-1', 'zhangsan', 'dept-1', 'OU=Sales,DC=corp,DC=com', NULL)`,
		`INSERT INTO sys_user (id, username, dept_id, ad_ou_dn, deleted_at)
		 VALUES ('u-2', 'zhang_san', 'dept-1', 'OU=Sales,DC=corp,DC=com', NULL)`,
	})

	svc := NewDeptOUMismatchService(db)

	t.Run("case_insensitive_username", func(t *testing.T) {
		rows, total, err := svc.ListDeptOUMismatch(context.Background(), &DeptOUMismatchQuery{
			Current: 1, PageSize: 10, Keyword: "ZHANGSAN",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total, "大写关键字应命中小写用户名")
		require.Len(t, rows, 1)
		require.NotNil(t, rows[0].Username)
		assert.Equal(t, "zhangsan", *rows[0].Username)
	})

	t.Run("underscore_is_literal", func(t *testing.T) {
		rows, total, err := svc.ListDeptOUMismatch(context.Background(), &DeptOUMismatchQuery{
			Current: 1, PageSize: 10, Keyword: "_",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total, "转义后 `_` 只应命中字面含下划线的用户名（未转义会命中全部 2 行）")
		require.Len(t, rows, 1)
		require.NotNil(t, rows[0].Username)
		assert.Equal(t, "zhang_san", *rows[0].Username)
	})

	t.Run("percent_is_literal", func(t *testing.T) {
		_, total, err := svc.ListDeptOUMismatch(context.Background(), &DeptOUMismatchQuery{
			Current: 1, PageSize: 10, Keyword: "%",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), total, "没有任何行含字面 `%`（未转义会命中全部 2 行）")
	})

	t.Run("matches_mapped_ou_dn", func(t *testing.T) {
		_, total, err := svc.ListDeptOUMismatch(context.Background(), &DeptOUMismatchQuery{
			Current: 1, PageSize: 10, Keyword: "ou=it,dc=corp",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), total, "mapped_ou_dn 也参与关键字匹配")
	})

	t.Run("matches_dept_name", func(t *testing.T) {
		_, total, err := svc.ListDeptOUMismatch(context.Background(), &DeptOUMismatchQuery{
			Current: 1, PageSize: 10, Keyword: "IT部",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), total, "dept_name 也参与关键字匹配")
	})
}

// TestPhase59_DeptOUMismatch_EmptyResultIsNonNullSlice 锁定空结果序列化为 `[]`。
func TestPhase59_DeptOUMismatch_EmptyResultIsNonNullSlice(t *testing.T) {
	db := freshPhase59DB(t)
	phase59Exec(t, db, phase59FourClassSeed())

	svc := NewDeptOUMismatchService(db)
	rows, total, err := svc.ListDeptOUMismatch(context.Background(), &DeptOUMismatchQuery{
		Current: 1, PageSize: 10, DeptID: "no-such-dept",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.NotNil(t, rows, "空结果必须是非 nil 空切片（否则 JSON 序列化成 null）")
	assert.Empty(t, rows)
}

// ============================================================================
// D-12 一致性测试：Go MatchADConfigByDN 与 SQL 视图的 config_unrouted 口径
//
// 这是本 phase 唯一一处「同一规则两份实现」（Go 最长后缀匹配 / 视图 routed CTE）。
// 判定式：Go `err == nil` **必须等价于** 视图对该用户不产出 config_unrouted 行。
// ============================================================================

// phase59MatchCase 一组一致性用例的输入与预期。
type phase59MatchCase struct {
	name string
	// configs 每条形如 "id|base_dn|status|deleted_at"，模拟 sys_ad_config 行
	configs      []string
	userOUDN     string
	mappingCfgID string // 映射行指向的配置（routed 用例据此断言路由到的配置 ID）
	expectRouted bool
}

func phase59MatchCases() []phase59MatchCase {
	return []phase59MatchCase{
		{
			// 父子域共享后缀：两个候选都命中，最长后缀（子域）胜出
			name: "longest_suffix_child_domain",
			configs: []string{
				"cfg-parent|DC=corp,DC=com|0|",
				"cfg-child|DC=na,DC=corp,DC=com|0|",
			},
			userOUDN:     "OU=IT,DC=na,DC=corp,DC=com",
			mappingCfgID: "cfg-child",
			expectRouted: true,
		},
		{
			// 逗号边界：xcorp 不得被 corp 误命中（裸 HasSuffix 会误命中）
			name:         "comma_boundary_xcorp",
			configs:      []string{"cfg-1|DC=corp,DC=com|0|"},
			userOUDN:     "OU=IT,DC=xcorp,DC=com",
			mappingCfgID: "cfg-1",
			expectRouted: false,
		},
		{
			name:         "no_match_other_domain",
			configs:      []string{"cfg-1|DC=corp,DC=com|0|"},
			userOUDN:     "OU=IT,DC=other,DC=com",
			mappingCfgID: "cfg-1",
			expectRouted: false,
		},
		{
			// 空 base_dn 恒命中任何 DN，必须被两侧同时跳过
			name: "empty_base_dn_skipped",
			configs: []string{
				"cfg-empty||0|",
				"cfg-real|DC=corp,DC=com|0|",
			},
			userOUDN:     "OU=IT,DC=corp,DC=com",
			mappingCfgID: "cfg-real",
			expectRouted: true,
		},
		{
			name:         "disabled_config_not_candidate",
			configs:      []string{"cfg-1|DC=corp,DC=com|1|"},
			userOUDN:     "OU=IT,DC=corp,DC=com",
			mappingCfgID: "cfg-1",
			expectRouted: false,
		},
		{
			name:         "soft_deleted_config_not_candidate",
			configs:      []string{"cfg-1|DC=corp,DC=com|0|2026-01-01 00:00:00"},
			userOUDN:     "OU=IT,DC=corp,DC=com",
			mappingCfgID: "cfg-1",
			expectRouted: false,
		},
		{
			// 等长歧义（同域重复配置）：Go 返回 error，SQL 用 cnt = 1 复刻 → unrouted
			name: "ambiguous_equal_length_same_domain",
			configs: []string{
				"cfg-a|DC=corp,DC=com|0|",
				"cfg-b|dc=CORP,dc=COM|0|",
			},
			userOUDN:     "OU=IT,DC=corp,DC=com",
			mappingCfgID: "cfg-a",
			expectRouted: false,
		},
	}
}

// TestPhase59_MatchConsistency 七组边界输入上比对 Go 与 SQL 的口径。
func TestPhase59_MatchConsistency(t *testing.T) {
	for _, tc := range phase59MatchCases() {
		t.Run(tc.name, func(t *testing.T) {
			db := freshPhase59DB(t)

			stmts := []string{
				`INSERT INTO sys_dept (id, dept_name, deleted_at) VALUES ('dept-1', 'IT部', NULL)`,
				`INSERT INTO sys_dept_ou_mapping (dept_id, ad_config_id, ou_dn, sync_enabled)
				 VALUES ('dept-1', '` + tc.mappingCfgID + `', 'OU=Other,DC=corp,DC=com', 1)`,
				`INSERT INTO sys_user (id, username, dept_id, ad_ou_dn, deleted_at)
				 VALUES ('u-1', 'zhangsan', 'dept-1', '` + tc.userOUDN + `', NULL)`,
			}
			for _, c := range tc.configs {
				parts := strings.SplitN(c, "|", 4)
				require.Len(t, parts, 4, "配置 fixture 形状错误: %s", c)
				deletedAt := "NULL"
				if parts[3] != "" {
					deletedAt = "'" + parts[3] + "'"
				}
				stmts = append(stmts, `INSERT INTO sys_ad_config (id, config_name, base_dn, status, deleted_at)
					VALUES ('`+parts[0]+`', 'cfg', '`+parts[1]+`', `+parts[2]+`, `+deletedAt+`)`)
			}
			phase59Exec(t, db, stmts)

			// Go 侧：与视图 cfg CTE 同款过滤（status=enabled + 未软删）。
			// 空 base_dn 的过滤由 MatchADConfigByDN 内部完成，不在此重复。
			var configs []models.ADConfig
			require.NoError(t, db.Raw(
				"SELECT id, base_dn, status FROM sys_ad_config WHERE status = 0 AND deleted_at IS NULL").
				Scan(&configs).Error)

			cfg, goErr := MatchADConfigByDN(tc.userOUDN, configs)
			goRouted := goErr == nil

			var unrouted int64
			require.NoError(t, db.Raw(
				`SELECT COUNT(*) FROM v_dept_ou_mismatch
				  WHERE mismatch_type = 'config_unrouted' AND user_id = 'u-1'`).
				Scan(&unrouted).Error)
			sqlRouted := unrouted == 0

			assert.Equal(t, tc.expectRouted, goRouted, "Go 侧路由结论不符")
			assert.Equal(t, goRouted, sqlRouted,
				"Go/SQL 口径不一致: Go err=%v, SQL unrouted 行数=%d", goErr, unrouted)

			if goRouted {
				require.NotNil(t, cfg)
				var gotConfigID string
				require.NoError(t, db.Raw(
					`SELECT ad_config_id FROM v_dept_ou_mismatch
					  WHERE mismatch_type = 'user_ou_mismatch' AND user_id = 'u-1'`).
					Row().Scan(&gotConfigID))
				assert.Equal(t, cfg.ID, gotConfigID, "SQL 路由到的配置应与 Go 一致")
			}
		})
	}
}

// derefOr 解引用可选字符串，nil 时返回空串（仅用于测试内的去重键）。
func derefOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
