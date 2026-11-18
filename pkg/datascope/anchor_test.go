package datascope

// anchor_test.go —— 机构锚点维度（v1.54）契约回归。
//
// 覆盖 .planning/notes/261118-anchor-scope-contract.md §九 的 6 条验收用例：
//   A. 刘洋(综合管理部, ancestors 含武汉中支) + level=2 → 含武汉中支全部下级(含硚口支公司)
//   B. 同上 + level=3(支公司) → 未命中 → fail-closed 空集
//   C. C1 回归：All(1) 用户 + 有效锚点 → 仍 All=true
//   D. Self(5) 用户 + 有效锚点 → Hook=None 且 DeptIDs 非空
//   E. C2 零回归：无锚点来源 → Merge 结果与原 Resolve 逐字段相等
//   F. ValidateAnchor(1/4) 报错；mode=5 的非法 param 报错；mode=3 无外部机构时报错
//
// 夹具刻意对齐生产数据形态：
//   - sys_dept.ancestors 为「根→父」序、逗号分隔、无首尾逗号
//   - sys_user.dept_id 为 TEXT（生产为 varchar(64)）
//   - 全部查询两侧 CAST(... AS TEXT)，故 SQLite 夹具与 PG 行为一致

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// newAnchorTestDB 建最小夹具：部门树 + 用户 + 角色/岗位 + 两张关联表。
//
// 组织树（对齐用户真实场景）：
//
//	D-root  湖北分公司  level=1  ancestors=""
//	D-wh    武汉中支    level=2  ancestors="D-root"
//	D-qk    硚口支公司  level=3  ancestors="D-root,D-wh"   is_external=1
//	D-zh    综合管理部  level=0  ancestors="D-root,D-wh"    ← 刘洋在此
//	D-cs    长沙中支    level=2  ancestors="D-root"
//
// 用户：
//
//	U-liu   刘洋   dept=D-zh   （武汉中支下的科室）
//	U-wang  老王   dept=D-qk   （硚口支公司本部）
//	U-li    小李   dept=D-cs   （长沙中支）
func newAnchorTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "anchor.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})

	stmts := []string{
		`CREATE TABLE sys_dept (id TEXT PRIMARY KEY, dept_name TEXT, dept_code TEXT,
			parent_id TEXT, ancestors TEXT DEFAULT '', org_level INTEGER DEFAULT 0,
			is_external_org INTEGER DEFAULT 0, status INTEGER DEFAULT 0,
			deleted_at DATETIME)`,
		`CREATE TABLE sys_user (id TEXT PRIMARY KEY, username TEXT, nickname TEXT,
			dept_id TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE sys_role (id TEXT PRIMARY KEY, role_name TEXT, role_key TEXT,
			data_scope INTEGER DEFAULT 1, anchor_mode INTEGER DEFAULT 0,
			anchor_param TEXT DEFAULT '', status INTEGER DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE sys_post (id TEXT PRIMARY KEY, post_code TEXT, post_name TEXT,
			anchor_mode INTEGER DEFAULT 0, anchor_param TEXT DEFAULT '',
			status INTEGER DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE sys_user_role (user_id TEXT, role_id TEXT)`,
		`CREATE TABLE sys_user_post (user_id TEXT, post_id TEXT)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}

	// 部门树
	require.NoError(t, db.Exec(`INSERT INTO sys_dept
		(id, dept_name, dept_code, parent_id, ancestors, org_level, is_external_org, status) VALUES
		('D-root','湖北分公司','HB',   NULL,    '',            1, 1, 0),
		('D-wh',  '武汉中支',  'WH',   'D-root','D-root',      2, 1, 0),
		('D-qk',  '硚口支公司','QK',   'D-wh',  'D-root,D-wh', 3, 1, 0),
		('D-zh',  '综合管理部','ZH',   'D-wh',  'D-root,D-wh', 0, 0, 0),
		('D-cs',  '长沙中支',  'CS',   'D-root','D-root',      2, 1, 0)`).Error)

	// 用户
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, nickname, dept_id, status) VALUES
		('U-liu','liuyang','刘洋','D-zh',0),
		('U-wang','wangwu','老王','D-qk',0),
		('U-li','lixiao','小李','D-cs',0),
		('U-nod','nodept','无部门',NULL,0)`).Error)

	return db
}

// 用例 A：level=2 命中武汉中支 → 可见集含武汉中支全部下级（含 level=3 的硚口支公司）。
func TestResolveAnchor_Level2_HitsSubBangAndDescendants(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	anchorID, resolved, reason, err := ResolveAnchor(ctx, db, "U-liu", models.AnchorModeLevel, "2")
	require.NoError(t, err)
	require.True(t, resolved, "刘洋应命中武汉中支，reason=%s", reason)
	assert.Equal(t, "D-wh", anchorID)

	ids, err := queryDeptChildIDs(ctx, db, anchorID)
	require.NoError(t, err)

	// 武汉中支自身 + 硚口支公司 + 综合管理部，全部应包含
	assert.Contains(t, ids, "D-wh")
	assert.Contains(t, ids, "D-qk", "中支子树必须包含下级支公司")
	assert.Contains(t, ids, "D-zh")
	// 不应越级包含同级的另一个中支
	assert.NotContains(t, ids, "D-cs", "武汉中支的子树不得包含长沙中支")
}

// 用例 B：level=3（支公司）对刘洋未命中 → fail-closed 空集（不回退 dept_id）。
func TestResolveAnchor_Level3_NotMatched_FailClosed(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	anchorID, resolved, reason, err := ResolveAnchor(ctx, db, "U-liu", models.AnchorModeLevel, "3")
	require.NoError(t, err)
	assert.False(t, resolved, "刘洋部门链上没有支公司，不应命中")
	assert.Empty(t, anchorID)
	assert.Equal(t, AnchorReasonLevelMiss, reason)

	// 关键：绝不回退到用户自身 dept_id（C3）
	assert.NotEqual(t, "D-zh", anchorID)
}

// 用例 B'：老王(硚口支公司) 对 level=3 命中自身；对 level=2 上溯到武汉中支。
func TestResolveAnchor_Wang_Level3AndLevel2(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	// level=3：自身即支公司
	id3, ok3, _, err := ResolveAnchor(ctx, db, "U-wang", models.AnchorModeLevel, "3")
	require.NoError(t, err)
	assert.True(t, ok3)
	assert.Equal(t, "D-qk", id3)

	// level=2：自身不是中支，必须上溯到武汉中支（跳过 level3 的自己）
	id2, ok2, _, err := ResolveAnchor(ctx, db, "U-wang", models.AnchorModeLevel, "2")
	require.NoError(t, err)
	assert.True(t, ok2)
	assert.Equal(t, "D-wh", id2, "支公司管理员对中支级岗位应上溯到所属中支")
}

// 用例 A'：小李(长沙中支) 对 level=2 命中长沙中支，与武汉中支互不干扰。
func TestResolveAnchor_Li_HitsOwnSubBang(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	id, ok, _, err := ResolveAnchor(ctx, db, "U-li", models.AnchorModeLevel, "2")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "D-cs", id)
}

// 用例 C（C1 回归）：All(1) 用户叠加有效锚点后，结果必须仍是 All。
//
// 这是最关键的回归：MAX(data_scope) 未被改动，锚点维度也不能把 All 收窄。
func TestMerge_AllSurvivesAnchorDimension(t *testing.T) {
	roleScope := AllScope("U-liu")
	got := Merge(roleScope, []string{"D-wh", "D-qk"})
	assert.True(t, got.All, "All 用户叠加锚点后必须仍为 All（C1）")
	assert.Empty(t, got.DeptIDs)
	assert.False(t, got.FailClosed)
}

// 用例 D：Self(5) 用户 + 有效锚点 → Hook=None 且 DeptIDs 非空（机构维度胜出）。
func TestMerge_SelfOverriddenByAnchor(t *testing.T) {
	roleScope := ScopeResult{ResourceHook: HookSelfUser, UserID: "U-liu"}
	got := Merge(roleScope, []string{"D-wh", "D-qk"})
	assert.False(t, got.All)
	assert.Equal(t, HookNone, got.ResourceHook, "显式机构授予应覆盖仅本人")
	assert.Equal(t, []string{"D-qk", "D-wh"}, got.DeptIDs, "normalizeIDs 会排序")
	assert.False(t, got.FailClosed)
}

// 用例 E（C2 零回归）：无锚点来源时，Merge 结果与 Resolve 原输出逐字段相等。
func TestMerge_NoAnchor_IsIdentity(t *testing.T) {
	cases := []ScopeResult{
		AllScope("u1"),
		{FailClosed: true, UserID: "u1"},
		{DeptIDs: []string{"D-a", "D-b"}, UserID: "u1"},
		{ResourceHook: HookSelfUser, UserID: "u1"},
	}
	for _, in := range cases {
		got := Merge(in, nil)
		assert.Equal(t, in, got, "无锚点时必须逐字段等于原 Resolve 输出（C2）")
	}
}

// 用例 F：ValidateAnchor 拒绝弃用编号与非法参数。
func TestValidateAnchor_Rejects(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	tests := []struct {
		name   string
		mode   int
		param  string
		errMsg string
	}{
		{"弃用模式1", 1, "2", "非法"},
		{"弃用模式4", 4, "", "非法"},
		{"未知模式9", 9, "", "非法"},
		{"level 空参数", 5, "", "必须是 1"},
		{"level 0 不可作 target", 5, "0", "必须是 1"},
		{"level 超范围", 5, "5", "必须是 1"},
		{"level 非数字", 5, "abc", "必须是 1"},
		{"level 层级不存在", 5, "4", "不存在层级"},
		{"dept 空参数", 2, "", "必须选择锚点部门"},
		{"dept 不存在", 2, "D-nope", "不存在或已删除"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAnchor(ctx, db, tt.mode, tt.param)
			require.Error(t, err, "%s 应被拒绝", tt.name)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

// 用例 F'：ValidateAnchor 通过合法配置。
func TestValidateAnchor_Accepts(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	// 不参与：任意 param 都忽略
	require.NoError(t, ValidateAnchor(ctx, db, 0, "whatever"))
	// 指定部门（存在）
	require.NoError(t, ValidateAnchor(ctx, db, 2, "D-wh"))
	// 外部机构（夹具里有 is_external_org=1）
	require.NoError(t, ValidateAnchor(ctx, db, 3, ""))
	// 层级 1/2/3 都存在
	require.NoError(t, ValidateAnchor(ctx, db, 5, "1"))
	require.NoError(t, ValidateAnchor(ctx, db, 5, "2"))
	require.NoError(t, ValidateAnchor(ctx, db, 5, "3"))
}

// 用例 F”：mode=3 但组织树无外部机构 → 配置期拒绝（把错误挡在配置时）。
func TestValidateAnchor_ExternalOrg_NoCandidates(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	require.NoError(t, db.Exec(`UPDATE sys_dept SET is_external_org = 0`).Error)

	err := ValidateAnchor(ctx, db, 3, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未标记任何「外部机构」")
}

// 用例 附加：mode=3 由近到远取最近的 is_external_org 节点。
func TestResolveAnchor_ExternalOrg_Nearest(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	// 刘洋自身 D-zh 不是外部机构，父 D-wh 是 → 命中武汉中支
	id, ok, _, err := ResolveAnchor(ctx, db, "U-liu", models.AnchorModeExternalOrg, "")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "D-wh", id)

	// 老王自身 D-qk 就是外部机构 → 命中自身，不上溯
	id2, ok2, _, err := ResolveAnchor(ctx, db, "U-wang", models.AnchorModeExternalOrg, "")
	require.NoError(t, err)
	assert.True(t, ok2)
	assert.Equal(t, "D-qk", id2)
}

// 用例 附加：无部门用户 → 未命中且原因明确（不 panic、不回退）。
func TestResolveAnchor_NoDept(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	_, ok, reason, err := ResolveAnchor(ctx, db, "U-nod", models.AnchorModeLevel, "2")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, AnchorReasonNoDept, reason)
}

// 用例 附加：mode=2 显式锚点支持跨机构授权（不要求是用户祖先）。
func TestResolveAnchor_ExplicitDept_AllowsCrossOrg(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	// 小李在长沙中支，但被显式授权到武汉中支（省公司派驻/兼任场景）
	id, ok, _, err := ResolveAnchor(ctx, db, "U-li", models.AnchorModeDept, "D-wh")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "D-wh", id, "显式锚点允许跨机构，不要求是用户祖先")
}

// 用例 附加：ResolveAnchorGrants 汇总角色 + 岗位两类来源取并集。
func TestResolveAnchorGrants_MergesRoleAndPost(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	// 刘洋：一个 role 声明 level=2（→武汉中支子树），一个 post 声明 level=3（未命中）
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, anchor_mode, anchor_param, status)
		VALUES ('R-mid','中支管理员','mid_admin', 5, '2', 0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES ('U-liu','R-mid')`).Error)

	require.NoError(t, db.Exec(`INSERT INTO sys_post (id, post_code, post_name, anchor_mode, anchor_param, status)
		VALUES ('P-off','office_admin','支公司管理员', 5, '3', 0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_post (user_id, post_id) VALUES ('U-liu','P-off')`).Error)

	ids, err := ResolveAnchorGrants(ctx, db, "U-liu")
	require.NoError(t, err)

	assert.Contains(t, ids, "D-wh")
	assert.Contains(t, ids, "D-qk")
	assert.Contains(t, ids, "D-zh")
	assert.NotContains(t, ids, "D-cs")
}

// 用例 附加：停用的角色/岗位不产生锚点（与 getUserMaxDataScope 的口径一致）。
func TestResolveAnchorGrants_DisabledSourcesIgnored(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, anchor_mode, anchor_param, status)
		VALUES ('R-off','停用角色','off_role', 5, '2', 1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES ('U-liu','R-off')`).Error)

	ids, err := ResolveAnchorGrants(ctx, db, "U-liu")
	require.NoError(t, err)
	assert.Empty(t, ids, "停用角色不得产生锚点范围")
}

// 用例 附加：无任何锚点来源 → 空集（不是 All，也不是 dept 回退）。
func TestResolveAnchorGrants_NoSources(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	ids, err := ResolveAnchorGrants(ctx, db, "U-nod")
	require.NoError(t, err)
	assert.Empty(t, ids)
}

// ===========================================================================
// verifier 补充（v1.54 独立核验）
//
// 以下用例覆盖契约 §九 用例 B 的**真实语义**，以及 Merge 规则里前面没被锁死的
// 三条分支（规则 2 / 4 / 5）。开发侧原测试只覆盖了规则 1（All）与规则 3（并集）。
// ===========================================================================

// TestContractCaseB_Level3Miss_AnchorEmptyAndNoDeptFallback 契约 §九 用例 B。
//
// 契约原文「param="3" → 未命中 → FailClosed 空集」。必须区分两个层面：
//
//  1. **锚点维度**：ResolveAnchorGrants 返回空集，且**绝不回退到用户 dept_id**（C3）；
//  2. **合并后的 ScopeResult**：是否 FailClosed 取决于角色维度 —— Merge 规则 2/5
//     只在「角色维度本身为空」时才会 FailClosed。若用户还有 Dept/Custom 角色，
//     合并结果保留角色维度（不收窄、也不 FailClosed），这是 C2「纯增量」的必然结果。
//
// 本测试把这两层都钉死，避免后人把「锚点未命中」误读成「整条请求 fail-closed」。
func TestContractCaseB_Level3Miss_AnchorEmptyAndNoDeptFallback(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	// 刘洋：一个 level=3（支公司）的岗位锚点 —— 他的部门链上没有支公司 → 未命中
	require.NoError(t, db.Exec(`INSERT INTO sys_post (id, post_code, post_name, anchor_mode, anchor_param, status)
		VALUES ('P-branch','branch_admin','支公司管理员', 5, '3', 0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_post (user_id, post_id) VALUES ('U-liu','P-branch')`).Error)

	// 第 1 层：锚点维度必须为空集，且不回退到 D-zh（综合管理部）
	ids, err := ResolveAnchorGrants(ctx, db, "U-liu")
	require.NoError(t, err)
	assert.Empty(t, ids, "level=3 对刘洋未命中 → 锚点维度空集")
	assert.NotContains(t, ids, "D-zh", "C3：绝不回退到用户自身 dept_id")

	// 第 2 层 a：角色维度也为空（无 Dept 角色 + 无 Hook）→ Merge 落规则 5 → FailClosed
	emptyRole := ScopeResult{UserID: "U-liu"}
	merged := Merge(emptyRole, ids)
	assert.True(t, merged.FailClosed, "角色维度为空 + 锚点未命中 → FailClosed")
	assert.Empty(t, merged.DeptIDs)
	assert.Equal(t, HookNone, merged.ResourceHook)

	// 第 2 层 b：用户还有 Dept 角色（DeptIDs 非空）→ 保留角色维度，**不** FailClosed
	deptRole := ScopeResult{DeptIDs: []string{"D-zh"}, UserID: "U-liu"}
	merged2 := Merge(deptRole, ids)
	assert.False(t, merged2.FailClosed, "锚点未命中不得把既有角色维度打成 fail-closed（C2）")
	assert.Equal(t, []string{"D-zh"}, merged2.DeptIDs)
}

// TestMerge_Rule2_FailClosedWithoutAnchorIsPreserved 契约 §三 3.4 规则 2。
//
// 异常枚举（FailClosed）且**无锚点** → 原样返回，不得放宽成 All 或空 Hook。
func TestMerge_Rule2_FailClosedWithoutAnchorIsPreserved(t *testing.T) {
	in := FailClosedScope("u1")
	got := Merge(in, nil)
	assert.Equal(t, in, got, "规则 2：异常枚举 + 无锚点必须原样 fail-closed")
	assert.True(t, got.FailClosed)
}

// TestMerge_Rule2_FailClosedWithAnchorWidens 契约 §三 3.4 规则 2 的边界。
//
// ⚠️ 安全语义显式记录：一个「异常枚举（脏 data_scope）」用户，若同时持有有效锚点，
// 实现会走规则 3 把结果放宽为锚点子树（FailClosed=false）。
// 这是 C2「宽者胜 / 纯增量」的直接推论，但也意味着**脏枚举不再兜底全空**。
// 本测试锁定当前实现，供后续安全评审随时改判。
func TestMerge_Rule2_FailClosedWithAnchorWidens(t *testing.T) {
	got := Merge(FailClosedScope("u1"), []string{"D-a"})
	assert.False(t, got.FailClosed, "FailClosed + 有效锚点 → 规则 3 放宽（C2 宽者胜）")
	assert.Equal(t, []string{"D-a"}, got.DeptIDs)
	assert.Equal(t, HookNone, got.ResourceHook)
	assert.False(t, got.All, "放宽绝不等于 All")
}

// TestMerge_Rule4_SelfWithoutAnchorIsPreserved 契约 §三 3.4 规则 4。
func TestMerge_Rule4_SelfWithoutAnchorIsPreserved(t *testing.T) {
	in := ScopeResult{ResourceHook: HookSelfUser, UserID: "u1"}
	got := Merge(in, nil)
	assert.Equal(t, in, got, "规则 4：Self + 无锚点必须保持「仅本人」")
	assert.Equal(t, HookSelfUser, got.ResourceHook)
	assert.Empty(t, got.DeptIDs)
}

// TestMerge_Rule5_EmptyAndNoHookIsFailClosed 契约 §三 3.4 规则 5。
func TestMerge_Rule5_EmptyAndNoHookIsFailClosed(t *testing.T) {
	got := Merge(ScopeResult{UserID: "u1"}, nil)
	assert.True(t, got.FailClosed, "规则 5：并集空 + 无 Hook → FailClosed")
	assert.Equal(t, "u1", got.UserID, "UserID 必须原样透传")
	assert.Empty(t, got.DeptIDs)
}

// TestResolveAnchor_LevelParamToleratesWhitespace mode=5 参数的前后空白必须被容忍
// （入库前 sanitizeAnchorParam 用 TrimSpace，解析侧 parseAnchorLevel 也必须同口径，
// 否则「先存 ' 2' 再解析」会出现存得进但解析不到的静默空集）。
func TestResolveAnchor_LevelParamToleratesWhitespace(t *testing.T) {
	db := newAnchorTestDB(t)

	// ResolveAnchor 侧
	id, ok, _, err := ResolveAnchor(context.Background(), db, "U-liu", models.AnchorModeLevel, " 2 ")
	require.NoError(t, err)
	assert.True(t, ok, "带空白的 level 参数应被 TrimSpace 后正常解析")
	assert.Equal(t, "D-wh", id)

	// ValidateAnchor 侧必须同口径放行
	require.NoError(t, ValidateAnchor(context.Background(), db, 5, " 2 "))
}

// TestValidateAnchor_DeptModeAcceptsAnyExistingDeptID 记录一条**契约文本偏差**：
//
// 契约 §五 第 3 条要求 mode=2 的 param「非空且为合法 UUID」。实现只校验
// 「非空 + sys_dept 中确实存在该 id」，没有做 UUID 正则前置校验。
//
// PG 下 sys_dept.id 是 uuid 类型，非 UUID 串查不到任何行 → 仍然被拒绝，故生产
// 行为与契约等价；但 SQLite（测试/嵌入）下 id 是 TEXT，非 UUID 形状的行可通过，
// 与契约文本不一致。本测试把现状钉死，供后续决定「补 UUID 正则」还是「改契约」。
func TestValidateAnchor_DeptModeAcceptsAnyExistingDeptID(t *testing.T) {
	db := newAnchorTestDB(t)
	ctx := context.Background()

	// 夹具里的 D-wh 不是 UUID 形状，但确实存在 → 当前实现放行
	require.NoError(t, ValidateAnchor(ctx, db, 2, "D-wh"),
		"现状：mode=2 只校验部门存在，未校验 UUID 形状（契约文本要求合法 UUID）")

	// 非 UUID 且不存在 → 拒绝（这一半是契约要求的）
	require.Error(t, ValidateAnchor(ctx, db, 2, "not-a-uuid"))
}
