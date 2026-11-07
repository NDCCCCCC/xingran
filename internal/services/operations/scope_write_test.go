// Package operations: Phase 132 scope_write.go 厚单测。
// 命名前缀 ScopeWrite* 避免与其它 agent 的测试/helper 冲突。
package operations

import (
	"context"
	"net/http"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// scopeWriteCtx 构造指定可见集的 ctx。
func scopeWriteCtx(r datascope.ScopeResult) context.Context {
	return datascope.WithContext(context.Background(), r)
}

// scopeWriteDeptOnly 构造「仅看某部门」的 ScopeResult。
func scopeWriteDeptOnly(deptIDs ...string) datascope.ScopeResult {
	return datascope.ScopeResult{DeptIDs: deptIDs}
}

// scopeWriteBuildDB 用 AutoMigrate 建最小夹具表（sys_dept / ops_buildings /
// ops_floors / sys_workstation / ops_info_points），供 filterVisibleIDs 测试。
func scopeWriteBuildDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (id TEXT PRIMARY KEY, dept_name TEXT);
		CREATE TABLE ops_buildings (id TEXT PRIMARY KEY, org_id TEXT, deleted_at TEXT);
		CREATE TABLE ops_floors (id TEXT PRIMARY KEY, building_id TEXT, org_id TEXT, deleted_at TEXT);
		CREATE TABLE sys_workstation (id TEXT PRIMARY KEY, floor_id TEXT, user_id TEXT, deleted_at TEXT);
		CREATE TABLE ops_info_points (id TEXT PRIMARY KEY, workstation_id TEXT, deleted_at TEXT);
	`).Error)
	return db
}

// ---------------------------------------------------------------------
// A-1 ErrScopeDenied 必须是显式 HTTP 403 的 AppError（agent-8 P0-2）
// ---------------------------------------------------------------------

func TestScopeWrite_ErrScopeDenied_IsExplicit403(t *testing.T) {
	require.True(t, apperrors.IsAppError(ErrScopeDenied), "必须是 AppError 才能被 HandleServiceError 透传")
	appErr := apperrors.GetAppError(ErrScopeDenied)
	require.NotNil(t, appErr)
	assert.Equal(t, http.StatusForbidden, appErr.GetHTTPStatus(), "显式 403，绝不能是 codes.go 默认的 401")
	assert.Equal(t, apperrors.CodePermissionDenied, appErr.GetCode())
	assert.Equal(t, "无权操作该数据", appErr.Message)
	assert.Contains(t, appErr.Error(), "无权操作该数据")
	assert.True(t, IsScopeDenied(ErrScopeDenied))
	assert.True(t, IsScopeDenied(apperrors.Wrap(ErrScopeDenied, apperrors.CodeServerError, "包装后仍可判定")))
	assert.False(t, IsScopeDenied(nil))
	assert.False(t, IsScopeDenied(apperrors.New(apperrors.CodeParamError, "别的错误")))
}

// ---------------------------------------------------------------------
// A-2 checkWriteScope 五档 / 0 值 / 越界 / 空 orgID
// ---------------------------------------------------------------------

func TestScopeWrite_CheckWriteScope_FiveTiers(t *testing.T) {
	const (
		deptA = "aaaaaaaa-0000-0000-0000-000000000001"
		deptB = "bbbbbbbb-0000-0000-0000-000000000002"
	)
	cases := []struct {
		name       string
		scope      datascope.ScopeResult
		ok         bool // 是否注入 ctx
		orgID      string
		hasUserCol bool
		wantDeny   bool
	}{
		{"All 档短路放行（越界 orgID 也不拒）", datascope.ScopeResult{All: true}, true, deptB, false, false},
		{"0/unset 未注入 → 按 All 兜底放行", datascope.ScopeResult{}, false, deptB, false, false},
		{"Custom 命中可见集", scopeWriteDeptOnly(deptA, deptB), true, deptA, false, false},
		{"Custom 越界", scopeWriteDeptOnly(deptA), true, deptB, false, true},
		{"Dept 单部门命中", scopeWriteDeptOnly(deptA), true, deptA, false, false},
		{"DeptChild 展开集命中", scopeWriteDeptOnly(deptA, deptB), true, deptB, false, false},
		{"FailClosed 全拒（含命中集内 orgID）", datascope.ScopeResult{FailClosed: true}, true, deptA, false, true},
		{"Custom 空集 fail-closed", datascope.ScopeResult{DeptIDs: []string{}}, true, deptA, false, true},
		{"Self 档无用户列实体 → 拒", datascope.ScopeResult{ResourceHook: datascope.HookSelfUser, UserID: "u1"}, true, deptA, false, true},
		{"Self 档有用户列但仅凭 orgID → 保守拒（须资源级校验）", datascope.ScopeResult{ResourceHook: datascope.HookSelfUser, UserID: "u1"}, true, deptA, true, true},
		{"空 orgID 无法解析 → fail-closed 拒", scopeWriteDeptOnly(deptA), true, "", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.ok {
				ctx = scopeWriteCtx(tc.scope)
			}
			err := checkWriteScope(ctx, tc.orgID, tc.hasUserCol)
			if tc.wantDeny {
				assert.True(t, IsScopeDenied(err), "应返回 ErrScopeDenied")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestScopeWrite_CheckWriteScopeEffective_Inheritance(t *testing.T) {
	const (
		deptA = "aaaaaaaa-0000-0000-0000-000000000001"
		deptB = "bbbbbbbb-0000-0000-0000-000000000002"
	)
	ctx := scopeWriteCtx(scopeWriteDeptOnly(deptA))
	// 楼层 org_id 为空 → 继承楼宇 org_id=deptA → 放行。
	assert.NoError(t, checkWriteScopeEffective(ctx, deptA))
	// 楼层覆盖值 deptB 越界 → 拒。
	assert.True(t, IsScopeDenied(checkWriteScopeEffective(ctx, deptB)))
	// 归属链断链（解析不出有效机构）→ 拒。
	assert.True(t, IsScopeDenied(checkWriteScopeEffective(ctx, "")))
}

func TestScopeWrite_CheckWriteScopeSelfUser(t *testing.T) {
	const selfUser = "11111111-0000-0000-0000-0000000000a1"
	selfCtx := scopeWriteCtx(datascope.ScopeResult{
		ResourceHook: datascope.HookSelfUser, UserID: selfUser,
	})
	assert.NoError(t, checkWriteScopeSelfUser(selfCtx, selfUser), "本人工位放行")
	assert.True(t, IsScopeDenied(checkWriteScopeSelfUser(selfCtx, "other-user")), "他人拒")
	assert.True(t, IsScopeDenied(checkWriteScopeSelfUser(selfCtx, "")), "无绑定用户拒")
	// All 档零回归
	assert.NoError(t, checkWriteScopeSelfUser(scopeWriteCtx(datascope.ScopeResult{All: true}), "any"))
	// FailClosed 全拒
	assert.True(t, IsScopeDenied(checkWriteScopeSelfUser(
		scopeWriteCtx(datascope.ScopeResult{FailClosed: true}), selfUser)))
}

// ---------------------------------------------------------------------
// A-3 filterVisibleIDs 批量裁剪边界 / All 短路零查询
// ---------------------------------------------------------------------

func TestScopeWrite_FilterVisibleIDs_Basic(t *testing.T) {
	const (
		deptA = "aaaaaaaa-0000-0000-0000-000000000001"
		deptB = "bbbbbbbb-0000-0000-0000-000000000002"
	)
	db := scopeWriteBuildDB(t)
	require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, org_id) VALUES
		('b1', ?), ('b2', ?), ('b3', ?)`, deptA, deptB, deptA).Error)

	ids := []string{"b1", "b2", "b3", "missing"}
	ctx := scopeWriteCtx(scopeWriteDeptOnly(deptA))
	orgExpr := effectiveOrgExpr("org_id", "org_id")
	visible, err := filterVisibleIDs(ctx, db, "ops_buildings", ids, orgExpr)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"b1", "b3"}, visible, "越界 b2 / 不存在的 missing 均被剔除")

	// 越界集全剔除
	visibleB, err := filterVisibleIDs(ctx, db, "ops_buildings", []string{"b2"}, orgExpr)
	require.NoError(t, err)
	assert.Empty(t, visibleB)
}

func TestScopeWrite_FilterVisibleIDs_AllShortCircuitZeroQuery(t *testing.T) {
	db := scopeWriteBuildDB(t)
	// All 档：即使表不存在 / 无数据，也必须零查询原样返回 ids。
	ctx := scopeWriteCtx(datascope.ScopeResult{All: true})
	ids := []string{"x", "y", "z"}
	visible, err := filterVisibleIDs(ctx, db, "ops_buildings", ids, effectiveOrgExpr("org_id", "org_id"))
	require.NoError(t, err)
	assert.Equal(t, ids, visible, "All 档零回归：原样返回、不查库")
	// 未注入 ctx 同样按 All 兜底
	visible2, err := filterVisibleIDs(context.Background(), db, "ops_buildings", ids, effectiveOrgExpr("org_id", "org_id"))
	require.NoError(t, err)
	assert.Equal(t, ids, visible2)
}

func TestScopeWrite_FilterVisibleIDs_FailClosedAndEmpty(t *testing.T) {
	db := scopeWriteBuildDB(t)
	orgExpr := effectiveOrgExpr("org_id", "org_id")
	cases := []struct {
		name  string
		scope datascope.ScopeResult
	}{
		{"FailClosed", datascope.ScopeResult{FailClosed: true}},
		{"空集", datascope.ScopeResult{DeptIDs: []string{}}},
		{"Self 无用户列", datascope.ScopeResult{ResourceHook: datascope.HookSelfUser, UserID: "u1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			visible, err := filterVisibleIDs(scopeWriteCtx(tc.scope), db, "ops_buildings", []string{"b1", "b2"}, orgExpr)
			require.NoError(t, err)
			assert.Empty(t, visible, "fail-closed 必须收紧为空集")
		})
	}
	// 空 ids 早退零查询
	visible, err := filterVisibleIDs(scopeWriteCtx(datascope.ScopeResult{FailClosed: true}), db, "ops_buildings", nil, orgExpr)
	require.NoError(t, err)
	assert.Empty(t, visible)
	// 非法表名拒绝（防注入）
	_, err = filterVisibleIDs(scopeWriteCtx(scopeWriteDeptOnly("d")), db, "sys_user; DROP TABLE x", []string{"b1"}, orgExpr)
	require.Error(t, err)
	assert.True(t, apperrors.IsAppError(err))
}

func TestScopeWrite_FilterVisibleIDs_SelfUserColumn(t *testing.T) {
	const selfUser = "11111111-0000-0000-0000-0000000000a1"
	db := scopeWriteBuildDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation (id, user_id) VALUES
		('w1', ?), ('w2', 'other'), ('w3', NULL)`, selfUser).Error)
	ctx := scopeWriteCtx(datascope.ScopeResult{ResourceHook: datascope.HookSelfUser, UserID: selfUser})
	visible, err := filterVisibleIDs(ctx, db, "sys_workstation", []string{"w1", "w2", "w3"},
		effectiveOrgExpr("org_id", "org_id"), "user_id")
	require.NoError(t, err)
	assert.Equal(t, []string{"w1"}, visible, "Self 档只放行本人工位")
}

// ---------------------------------------------------------------------
// A-4 filterVisibleIDsByExists 跨表有效机构（workstation 归属链）
// ---------------------------------------------------------------------

func TestScopeWrite_FilterVisibleIDsByExists_CrossTable(t *testing.T) {
	const (
		deptA = "aaaaaaaa-0000-0000-0000-000000000001"
		deptB = "bbbbbbbb-0000-0000-0000-000000000002"
	)
	db := scopeWriteBuildDB(t)
	require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, org_id) VALUES ('bld-a', ?), ('bld-b', ?)`, deptA, deptB).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, building_id, org_id) VALUES
		('f-a', 'bld-a', NULL), ('f-b', 'bld-b', NULL), ('f-override', 'bld-a', ?)`, deptB).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation (id, floor_id) VALUES
		('w1', 'f-a'), ('w2', 'f-b'), ('w3', 'f-override')`).Error)

	orgExpr := effectiveOrgExpr("f.org_id", "b.org_id")
	newSub := func() *gorm.DB {
		return db.Session(&gorm.Session{NewDB: true}).
			Table("ops_floors f").
			Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
			Where("CAST(f.id AS TEXT) = sys_workstation.floor_id").
			Where("f.deleted_at IS NULL AND b.deleted_at IS NULL").
			Select("1")
	}
	ids := []string{"w1", "w2", "w3"}

	// Dept A：w1 命中（继承楼宇 A）；w2 越界；w3 楼层覆盖 deptB 越界。
	visible, err := filterVisibleIDsByExists(scopeWriteCtx(scopeWriteDeptOnly(deptA)), db,
		"sys_workstation", ids, newSub(), orgExpr, "sys_workstation.user_id")
	require.NoError(t, err)
	assert.Equal(t, []string{"w1"}, visible, "有效机构 = 楼层覆盖优先、空则继承楼宇")

	// Dept B：w2 + w3 命中。
	visibleB, err := filterVisibleIDsByExists(scopeWriteCtx(scopeWriteDeptOnly(deptB)), db,
		"sys_workstation", ids, newSub(), orgExpr, "sys_workstation.user_id")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"w2", "w3"}, visibleB)

	// All 档零回归原样返回
	allVisible, err := filterVisibleIDsByExists(scopeWriteCtx(datascope.ScopeResult{All: true}), db,
		"sys_workstation", ids, newSub(), orgExpr, "sys_workstation.user_id")
	require.NoError(t, err)
	assert.ElementsMatch(t, ids, allVisible)

	// FailClosed 收紧空集
	closedVisible, err := filterVisibleIDsByExists(scopeWriteCtx(datascope.ScopeResult{FailClosed: true}), db,
		"sys_workstation", ids, newSub(), orgExpr, "sys_workstation.user_id")
	require.NoError(t, err)
	assert.Empty(t, closedVisible)
}

// ---------------------------------------------------------------------
// A-5 orgInVisibleSet O(1) 判定边界
// ---------------------------------------------------------------------

func TestScopeWrite_OrgInVisibleSet(t *testing.T) {
	deptIDs := []string{"d1", "d2"}
	assert.True(t, orgInVisibleSet("d1", deptIDs))
	assert.True(t, orgInVisibleSet("d2", deptIDs))
	assert.False(t, orgInVisibleSet("d3", deptIDs))
	assert.False(t, orgInVisibleSet("", deptIDs), "空 orgID fail-closed")
	assert.False(t, orgInVisibleSet("d1", nil))
}
