package system

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// =====================================================================
// Phase 84 Plan 04 (PMODEL-01)：用户列表 data_scope 过滤红绿测试。
//
// 夹具直接 db.Exec 摆行（不走 roleService.CreateRole——sqlite ON CONFLICT
// 方言 500 前科），部门 3 层 A→B→C 验证 ancestors 5 条件对深层子树正确。
// 用户分布：A 部 2 人（alice1/alice2）、B 部 1 人（bob）、C 部 1 人（carol），
// 无关部门 X 放 uCustom（Custom scope 测试用户，避免干扰 Dept 档）。
// =====================================================================

func newUserDataScopeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user (
			id TEXT PRIMARY KEY,
			username TEXT,
			nickname TEXT,
			phone TEXT, mobile TEXT, title TEXT, ad_config_id TEXT,
			status INTEGER NOT NULL,
			dept_id TEXT,
			password TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY,
			dept_name TEXT,
			ancestors TEXT,
			status INTEGER NOT NULL,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_user_role (user_id TEXT, role_id TEXT)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_role (
			id TEXT PRIMARY KEY, role_name TEXT, status INTEGER,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_role_dept (role_id TEXT, dept_id TEXT)`).Error)

	// 部门 3 层：A(根,ancestors='') → B(ancestors='A') → C(ancestors='A,B')，外加无关 X
	now := "2024-01-01 00:00:00"
	require.NoError(t, db.Exec(`INSERT INTO sys_dept VALUES ('A','总部','',0,?,?,NULL)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept VALUES ('B','一级子部门','A',0,?,?,NULL)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept VALUES ('C','二级子部门','A,B',0,?,?,NULL)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept VALUES ('X','无关部门','',0,?,?,NULL)`, now, now).Error)

	// A 部 2 人 + B 部 1 人 + C 部 1 人 + X 部 Custom 测试用户 1 人
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id,username,nickname,phone,status,dept_id,password,created_at,updated_at,deleted_at) VALUES
		('uA1','alice1','Alice1','13800000001',0,'A','x',?,?,NULL),
		('uA2','alice2','Alice2','13800000002',0,'A','x',?,?,NULL),
		('uB','bob','Bob','13900000000',0,'B','x',?,?,NULL),
		('uC','carol','Carol','13700000000',0,'C','x',?,?,NULL),
		('uX','custom','Custom','13600000000',0,'X','x',?,?,NULL)`,
		now, now, now, now, now, now, now, now, now, now).Error)

	// Custom 映射：角色 r1 → 部门 B、C（uCustom 持有 r1）
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES ('uX', 'r1')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, status) VALUES ('r1','自定义角色',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role_dept (role_id, dept_id) VALUES ('r1','B'),('r1','C')`).Error)

	return db
}

func userDataScopeListParams(scope models.DataScope, userID string) requests.UserListParams {
	return requests.UserListParams{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 100},
		DataScope:       scope,
		DataScopeUserID: userID,
	}
}

func userDataScopeUsernames(result *PageResult) []string {
	users, _ := result.List.([]models.User)
	names := make([]string, 0, len(users))
	for _, u := range users {
		names = append(names, u.Username)
	}
	return names
}

// TestUserServiceList_DataScope 逐档断言 list 行集 + PageResult.total 同步。
func TestUserServiceList_DataScope(t *testing.T) {
	db := newUserDataScopeTestDB(t)
	svc := &userService{db: db, pwdManager: nil}
	ctx := context.Background()

	cases := []struct {
		name       string
		scope      models.DataScope
		userID     string
		wantNames  []string
		wantTotal  int64
	}{
		{name: "Dept(3) 本部门 → 仅 A 部 2 人", scope: models.DataScopeDept, userID: "uA1",
			wantNames: []string{"alice1", "alice2"}, wantTotal: 2},
		{name: "DeptChild(4) 本部门及子 → A+B+C 共 4 人（3 层深树 ancestors 正确）", scope: models.DataScopeDeptChild, userID: "uA1",
			wantNames: []string{"alice1", "alice2", "bob", "carol"}, wantTotal: 4},
		{name: "Self(5) 仅本人 → 按 sys_user.id 只剩 alice1", scope: models.DataScopeSelf, userID: "uA1",
			wantNames: []string{"alice1"}, wantTotal: 1},
		{name: "Custom(2) 按 sys_role_dept 映射(B,C) → bob+carol", scope: models.DataScopeCustom, userID: "uX",
			wantNames: []string{"bob", "carol"}, wantTotal: 2},
		{name: "All(1) 不过滤 → 全量（含 Custom 测试用户共 5 行）", scope: models.DataScopeAll, userID: "uA1",
			wantNames: []string{"alice1", "alice2", "bob", "carol", "custom"}, wantTotal: 5},
		{name: "零值(0) 未挂中间件 → 不过滤全量 5 行", scope: models.DataScope(0), userID: "",
			wantNames: []string{"alice1", "alice2", "bob", "carol", "custom"}, wantTotal: 5},
		{name: "未知枚举(99) fail-closed → 空集 total=0", scope: models.DataScope(99), userID: "uA1",
			wantNames: []string{}, wantTotal: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.List(ctx, userDataScopeListParams(tc.scope, tc.userID))
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.ElementsMatch(t, tc.wantNames, userDataScopeUsernames(result), "行集不符")
			assert.Equal(t, tc.wantTotal, result.Total, "total 应与过滤后行集同步")
		})
	}
}

// TestDataScope_NoChangeForAllAndAdmin All/零值两路径与无过滤基准结果逐条一致（V2 回归闸门）。
func TestDataScope_NoChangeForAllAndAdmin(t *testing.T) {
	db := newUserDataScopeTestDB(t)
	svc := &userService{db: db, pwdManager: nil}
	ctx := context.Background()

	// 基准：零值（admin 早退 / 未挂中间件路径）
	baseline, err := svc.List(ctx, userDataScopeListParams(0, ""))
	require.NoError(t, err)

	for _, scope := range []models.DataScope{models.DataScopeAll, 0} {
		result, err := svc.List(ctx, userDataScopeListParams(scope, "uA1"))
		require.NoError(t, err)
		assert.Equal(t, baseline.Total, result.Total, "scope=%d 应与基准 total 一致", int(scope))
		assert.ElementsMatch(t, userDataScopeUsernames(baseline), userDataScopeUsernames(result), "scope=%d 行集应与基准一致", int(scope))
	}

	// 统计接口同口径：All/零值 total = 夹具全量 5（A2+B1+C1+Custom1）
	for _, tc := range []struct {
		scope  models.DataScope
		userID string
	}{
		{models.DataScopeAll, "uA1"}, {0, ""},
	} {
		stats, err := svc.StatisticsWithDataScope(ctx, tc.userID, tc.scope)
		require.NoError(t, err)
		assert.Equal(t, int64(5), stats.Total, "statistics scope=%d 应全量", int(tc.scope))
	}
}

// TestUserListCache_ScopeIsolation 同 params、两个不同 scope 用户先后调带缓存版 List，
// 两次结果互不串（V3，防跨用户缓存泄漏）；并断言 cache key 含 scope 维度段。
func TestUserListCache_ScopeIsolation(t *testing.T) {
	db := newUserDataScopeTestDB(t)
	mem := newInfraMemCache()
	provider := NewCacheAdapter(mem)
	svc := NewUserServiceWithCache(db, provider, nil, &mockPasswordManager{}).(*userCacheService)
	ctx := context.Background()

	// scope=3 用户 uA1（本部门 A → 2 人）与 scope=5 用户 uA2（仅本人 → 1 人），同分页参数
	deptParams := userDataScopeListParams(models.DataScopeDept, "uA1")
	selfParams := userDataScopeListParams(models.DataScopeSelf, "uA2")

	first, err := svc.List(ctx, deptParams)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"alice1", "alice2"}, userDataScopeUsernames(first))
	assert.Equal(t, int64(2), first.Total)

	second, err := svc.List(ctx, selfParams)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"alice2"}, userDataScopeUsernames(second),
		"第二次（scope=5）不得命中第一次（scope=3）的缓存——跨用户泄漏")
	assert.Equal(t, int64(1), second.Total, "第二次 total 应为 1，而非复用第一次的 2")

	// 回放第一次请求 → 命中自己的缓存，仍 2 人
	replay, err := svc.List(ctx, deptParams)
	require.NoError(t, err)
	assert.Equal(t, int64(2), replay.Total, "同 scope 回放应命中自身缓存")

	// cache key 含 scope 维度段；前缀仍以 CacheKeyUserList 开头（失效 pattern 兼容）
	deptKey := svc.buildListCacheKey(deptParams)
	selfKey := svc.buildListCacheKey(selfParams)
	assert.True(t, stringsHasPrefix(deptKey, CacheKeyUserList), "key 前缀必须仍是 CacheKeyUserList")
	assert.Contains(t, deptKey, ":ds:3:uid:uA1", "key 必须含 scope+uid 维度段")
	assert.Contains(t, selfKey, ":ds:5:uid:uA2", "key 必须含 scope+uid 维度段")
	assert.NotEqual(t, deptKey, selfKey, "不同 scope 的 key 必须不同")

	// All/零值 → uid 段统一 "all"（无个性化，key 形态稳定）
	allKey := svc.buildListCacheKey(userDataScopeListParams(models.DataScopeAll, "uA1"))
	zeroKey := svc.buildListCacheKey(userDataScopeListParams(0, ""))
	assert.Contains(t, allKey, ":ds:1:uid:all")
	assert.Contains(t, zeroKey, ":ds:0:uid:all")
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
