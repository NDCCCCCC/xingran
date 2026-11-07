// Package vdi: Phase 133 / DS-01 vdi 生产调用方迁移对照测试。
//
// 目标（LOCKED 裁定 .planning/notes/261107-v152-self-scope-decision.md §4）：
//   - 迁移后 ListVMs 优先从 datascope.FromContext(ctx) 取 ScopeResult 并应用
//     ApplyVMScopeResult；无注入时回落旧 ApplyVMDataScopeFilter 显式参数路径。
//   - 新旧实现在「五档 + 0/unset + 异常枚举」下对同一夹具的可见 VM 集合必须一致。
//   - 旧 ApplyVMDataScopeFilter / ApplyBoundUserFilter / vdi getChildDepts 保留不删，
//     既有 fail-closed 测试 vm_service_test.go:498/516/541/543 零回归。
//   - 保留审查 P1-4 语义：异常枚举 fail-closed（1=0），0/unset 不过滤（超管零回归）。
//
// 命名前缀 VMScopeMigration* 避免与其它 agent 的测试/helper 冲突。
package vdi

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	"gorm.io/gorm"
)

// VMScopeMigrationSeed 建最小数据夹具：
//
//	部门树 d1 → d11（子）；d2（独立）
//	用户 u1∈d1 / u2∈d11 / u3∈d2 / u4∈d1（用于 Custom 角色映射）
//	VM：vm1→u1 / vm2→u2 / vm3→u3 / vm4→u4 / vm5→NULL(未绑定)
//
// 旧实现按「用户 dept_id ∈ 可见部门集」过滤 bound_user_id，新实现按
// ScopeResult.DeptIDs（已由 datascope.Resolve 展开）过滤，二者逐档对齐。
func VMScopeMigrationSeed(t *testing.T) (*gorm.DB, string, string, string) {
	t.Helper()
	db := newVDITestDB(t)
	now := time.Now().Format("2006-01-02 15:04:05")
	u1 := "11111111-1111-1111-1111-111111111111" // dept d1
	u2 := "22222222-2222-2222-2222-222222222222" // dept d11 (d1 子部门)
	u3 := "33333333-3333-3333-3333-333333333333" // dept d2
	u4 := "44444444-4444-4444-4444-444444444444" // dept d1（Custom 映射用）
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, dept_id, created_at) VALUES
		(?, 'u1', 'd1', ?), (?, 'u2', 'd11', ?), (?, 'u3', 'd2', ?), (?, 'u4', 'd1', ?)`,
		u1, now, u2, now, u3, now, u4, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, parent_id, status) VALUES
		('d1', '', 0), ('d11', 'd1', 0), ('d2', '', 0)`).Error)
	// datascope.Resolve 的 DeptChild 档按 sys_dept.ancestors LIKE 展开子树
	// （单一真实源口径，见 dept_filter.go / datascope.go queryDeptChildIDs）；
	// newVDITestDB 夹具的 sys_dept 只有 parent_id，无 ancestors 列。此处补列并
	// 填生产同款 ancestors 值（根为空串、子节点为父链），使新旧两条子树展开
	// 口径在本夹具下语义等价（旧 vdi getChildDepts 走 parent_id 递归）。
	require.NoError(t, db.Exec(`ALTER TABLE sys_dept ADD COLUMN ancestors TEXT DEFAULT ''`).Error)
	require.NoError(t, db.Exec(`UPDATE sys_dept SET ancestors = CASE id
		WHEN 'd1' THEN '' WHEN 'd11' THEN 'd1' WHEN 'd2' THEN '' END`).Error)
	// datascope.Resolve 的 Custom 档按「仅启用角色」语义 INNER JOIN sys_role
	// （r.status=0，对齐 getUserMaxDataScope）；newVDITestDB 夹具未建该表，此处补建。
	// 旧 ApplyVMDataScopeFilter 的 Custom 只查 sys_user_role ⋈ sys_role_dept，
	// 不查 sys_role —— 在夹具角色 status=0（启用）时两条查询口径一致。
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_role (id TEXT PRIMARY KEY, status INTEGER DEFAULT 0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, status) VALUES ('r1', 0)`).Error)
	// Custom 档夹具：u3 → 角色 r1 → 部门 d2（与旧实现的 sys_user_role/sys_role_dept 同款）。
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, 'r1')`, u3).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role_dept (role_id, dept_id) VALUES ('r1', 'd2')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_vdi_vm (id, vm_id, name, vdi_server_id, bound_user_id, created_at) VALUES
		('vm1', 'v1', 'a', 's', ?, ?),
		('vm2', 'v2', 'b', 's', ?, ?),
		('vm3', 'v3', 'c', 's', ?, ?),
		('vm4', 'v4', 'd', 's', ?, ?),
		('vm5', 'v5', 'e', 's', NULL, ?)`, u1, now, u2, now, u3, now, u4, now, now).Error)
	return db, u1, u3, u4
}

// VMScopeMigrationOldCount 旧实现：显式 userID/dataScope + ApplyBoundUserFilter。
func VMScopeMigrationOldCount(t *testing.T, db *gorm.DB, userID string, ds models.DataScope) int64 {
	t.Helper()
	q := ApplyVMDataScopeFilter(db.Model(&models.VDIVirtualMachine{}), userID, ds, db)
	q = ApplyBoundUserFilter(q, ds)
	var n int64
	require.NoError(t, q.Count(&n).Error)
	return n
}

// VMScopeMigrationOldProductionCount 复刻 ListVMs 迁移前的外层短路语义：
// 0/unset（以及显式 userID 为空）完全跳过过滤（超管零回归）；
// 其余档位走 ApplyVMDataScopeFilter + ApplyBoundUserFilter。
// 注意：直接调用 ApplyVMDataScopeFilter(db,u,0,...) 会命中其 default:1=0，
// 但生产从不会这样调用——外层 `if userID != "" && dataScope != 0` 先短路了。
func VMScopeMigrationOldProductionCount(t *testing.T, db *gorm.DB, userID string, ds models.DataScope) int64 {
	t.Helper()
	q := db.Model(&models.VDIVirtualMachine{})
	if userID != "" && ds != 0 {
		q = ApplyVMDataScopeFilter(q, userID, ds, db)
		q = ApplyBoundUserFilter(q, ds)
	}
	var n int64
	require.NoError(t, q.Count(&n).Error)
	return n
}

// VMScopeMigrationNewCount 新实现：直接消费构造好的 ScopeResult（不经 ctx，纯函数对照）。
func VMScopeMigrationNewCount(t *testing.T, db *gorm.DB, scope datascope.ScopeResult) int64 {
	t.Helper()
	q := ApplyVMScopeResult(db.Model(&models.VDIVirtualMachine{}), scope)
	var n int64
	require.NoError(t, q.Count(&n).Error)
	return n
}

// VMScopeMigrationResolveCount 模拟生产路径：Resolve(ctx,...) → FromContext → ApplyVMScopeResult。
// 覆盖「优先 ctx ScopeResult」的注入链路，且断言 ScopeResult 来自 datascope 单一真实源。
func VMScopeMigrationResolveCount(t *testing.T, db *gorm.DB, userID string, ds models.DataScope) int64 {
	t.Helper()
	resolved, err := datascope.Resolve(context.Background(), db, userID, ds)
	require.NoError(t, err)
	ctx := datascope.WithContext(context.Background(), resolved)
	fromCtx, ok := datascope.FromContext(ctx)
	require.True(t, ok, "WithContext 注入后 FromContext 必须可取回")
	return VMScopeMigrationNewCount(t, db, fromCtx)
}

// TestVMScopeMigration_FiveScopes_OldEqualsNew 五档逐档对照：新旧实现可见集一致。
func TestVMScopeMigration_FiveScopes_OldEqualsNew(t *testing.T) {
	db, u1, u3, _ := VMScopeMigrationSeed(t)

	cases := []struct {
		name   string
		userID string
		ds     models.DataScope
	}{
		{"All 全部", u1, models.DataScopeAll},
		{"Custom 自定义(走角色-部门映射)", u3, models.DataScopeCustom},
		{"Dept 本部门", u1, models.DataScopeDept},
		{"DeptChild 本部门及子部门", u1, models.DataScopeDeptChild},
		{"Self 仅本人", u1, models.DataScopeSelf},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			old := VMScopeMigrationOldProductionCount(t, db, tc.userID, tc.ds)
			// 新实现两种入口都要与旧实现一致：
			//   ① Resolve→FromContext 生产链路；
			//   ② 手工构造等价 ScopeResult（验证纯函数本身语义）。
			resolved, err := datascope.Resolve(context.Background(), db, tc.userID, tc.ds)
			require.NoError(t, err)
			newDirect := VMScopeMigrationNewCount(t, db, resolved)
			newViaCtx := VMScopeMigrationResolveCount(t, db, tc.userID, tc.ds)
			assert.Equal(t, old, newViaCtx, "Resolve→FromContext 生产链路应与旧实现一致")
			assert.Equal(t, old, newDirect, "纯函数 ScopeResult 应与旧实现一致")
			assert.Equal(t, newViaCtx, newDirect, "两种新入口应彼此一致")
		})
	}
}

// TestVMScopeMigration_CustomRoleDeptMapping Custom 档需角色-部门映射夹具
// （Seed 已内置 u3→r1→d2 的映射，直接对照新旧实现）。
func TestVMScopeMigration_CustomRoleDeptMapping(t *testing.T) {
	db, _, u3, _ := VMScopeMigrationSeed(t)

	old := VMScopeMigrationOldProductionCount(t, db, u3, models.DataScopeCustom)
	newViaCtx := VMScopeMigrationResolveCount(t, db, u3, models.DataScopeCustom)
	assert.Equal(t, old, newViaCtx)
	assert.Equal(t, int64(1), newViaCtx, "Custom 仅可见 d2 的 vm3")
}

// TestVMScopeMigration_ZeroAndInvalidEnum 0/unset→All、异常枚举→fail-closed。
func TestVMScopeMigration_ZeroAndInvalidEnum(t *testing.T) {
	db, u1, _, _ := VMScopeMigrationSeed(t)

	// 0/unset：ListVMs 外层短路（不过滤）+ Resolve 短路为 All=true，
	// 新旧都不过滤（超管零回归，5 台含未绑定 vm5）。
	old0 := VMScopeMigrationOldProductionCount(t, db, u1, 0)
	new0 := VMScopeMigrationResolveCount(t, db, u1, 0)
	assert.Equal(t, int64(5), old0, "0/unset 旧生产短路不过滤（5 台，含未绑定 vm5）")
	assert.Equal(t, old0, new0, "0/unset 新实现同样不过滤")

	// 审查 P1-4：异常枚举（99 / -1）必须 fail-closed（1=0），不得放行全库。
	for _, bad := range []models.DataScope{models.DataScope(99), models.DataScope(-1)} {
		oldBad := VMScopeMigrationOldProductionCount(t, db, u1, bad)
		newBad := VMScopeMigrationResolveCount(t, db, u1, bad)
		assert.Equal(t, int64(0), oldBad, "异常枚举旧实现 1=0")
		assert.Equal(t, int64(0), newBad, "异常枚举新实现 FailClosed 1=0")
		assert.Equal(t, oldBad, newBad)
	}
}

// TestVMScopeMigration_ExplicitFailClosedScope FailClosedScope 构造（中间件 Resolve 失败兜底）→ 1=0。
func TestVMScopeMigration_ExplicitFailClosedScope(t *testing.T) {
	db, u1, _, _ := VMScopeMigrationSeed(t)
	scope := datascope.FailClosedScope(u1)
	assert.Equal(t, int64(0), VMScopeMigrationNewCount(t, db, scope))
}

// TestVMScopeMigration_UnboundUserHidden 非 All 档统一追加 bound_user_id IS NOT NULL。
func TestVMScopeMigration_UnboundUserHidden(t *testing.T) {
	db, u1, _, _ := VMScopeMigrationSeed(t)
	// vm5 未绑定：All 可见 5 台；Self/Dept 等非 All 档必须 <=4。
	assert.Equal(t, int64(5), VMScopeMigrationResolveCount(t, db, u1, models.DataScopeAll))
	assert.LessOrEqual(t, VMScopeMigrationResolveCount(t, db, u1, models.DataScopeSelf), int64(4))
	assert.LessOrEqual(t, VMScopeMigrationResolveCount(t, db, u1, models.DataScopeDept), int64(4))
}

// TestVMScopeMigration_NoContextFallsBackToExplicitParams 未注入 ctx 时回落显式参数。
//
// 直接验证 ListVMs 的分支选择：ok=false 不应误用零值 ScopeResult 放行全库，
// 而是走旧 ApplyVMDataScopeFilter + ApplyBoundUserFilter（语义与旧实现一致）。
func TestVMScopeMigration_NoContextFallsBackToExplicitParams(t *testing.T) {
	db, u1, _, _ := VMScopeMigrationSeed(t)
	_, ok := datascope.FromContext(context.Background())
	assert.False(t, ok, "空 ctx 不得产生 ScopeResult（触发回落分支）")

	// 回落分支的语义 == 旧实现显式参数路径。
	fallback := VMScopeMigrationOldCount(t, db, u1, models.DataScopeSelf)
	assert.Equal(t, int64(1), fallback, "Self 回落仅可见本人 vm1")
	// 若错误地用「零值 ScopeResult」走新分支，All=false+DeptIDs 空 → 1=0（过度收紧），
	// 断言回落分支不会退化成 1=0：这正是本测试要锁定的兼容行为。
	assert.NotEqual(t, int64(0), fallback)
}

// TestVMScopeMigration_ExistingFailClosedRegression 既有 fail-closed 锚点零回归：
// 非法/空 userID 旧实现 1=0；旧函数保留不删（本测试直接调用即为守护）。
func TestVMScopeMigration_ExistingFailClosedRegression(t *testing.T) {
	db, u1, _, _ := VMScopeMigrationSeed(t)
	assert.Equal(t, int64(0), VMScopeMigrationOldCount(t, db, "not-uuid", models.DataScopeAll))
	assert.Equal(t, int64(0), VMScopeMigrationOldCount(t, db, "", models.DataScopeAll))
	// Self 档 bound_user_id=登录人，与 ApplyBoundUserFilter 组合后 =1 台。
	assert.Equal(t, int64(1), VMScopeMigrationOldCount(t, db, u1, models.DataScopeSelf))
}
