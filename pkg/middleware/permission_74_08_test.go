package middleware

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// =====================================================================
// 74-08 Batch C 续: oper_log.go 纯函数 + permission.go 中间件/数据权限。
// 复用 permission_inherit_test.go 的 newTestCoreWithSQLite / createTables。
// 注意: mwSuperAdminCache / mwPermCheckCache 是进程级 30s TTL 缓存,
// 每个用例必须用全新 uuid userID,避免跨用例缓存污染。
// =====================================================================

// ---------------- oper_log.go ----------------

func TestDefaultOperLogConfig(t *testing.T) {
	cfg := DefaultOperLogConfig()
	require.NotNil(t, cfg)
	assert.Contains(t, cfg.LogPaths, "/system/user")
	assert.Contains(t, cfg.LogPaths, "/ops/building")
	assert.Contains(t, cfg.ExcludePaths, "/list")
}

func TestShouldLogOperation(t *testing.T) {
	cfg := DefaultOperLogConfig()

	// 方法门槛: 只记 POST/PUT/DELETE
	assert.False(t, shouldLogOperation("/system/user", "GET", cfg))
	assert.False(t, shouldLogOperation("/system/user", "PATCH", cfg))

	// 排除优先
	assert.False(t, shouldLogOperation("/system/user/list", "POST", cfg), "/list 排除")
	assert.False(t, shouldLogOperation("/system/user/export", "POST", cfg), "/export 排除")

	// 命中 LogPaths
	assert.True(t, shouldLogOperation("/system/user", "POST", cfg))
	assert.True(t, shouldLogOperation("/ops/building", "PUT", cfg))
	assert.True(t, shouldLogOperation("/workorder/123", "DELETE", cfg))

	// 不在列表
	assert.False(t, shouldLogOperation("/other/thing", "POST", cfg))
}

func TestGetBusinessType(t *testing.T) {
	assert.Equal(t, 1, GetBusinessType("/system/user/add", "POST"))
	assert.Equal(t, 1, GetBusinessType("/system/user/create", "POST"))
	assert.Equal(t, 2, GetBusinessType("/system/user/edit", "POST"))
	assert.Equal(t, 2, GetBusinessType("/system/user/update", "POST"))
	assert.Equal(t, 3, GetBusinessType("/system/user/delete", "POST"))
	assert.Equal(t, 3, GetBusinessType("/system/user/remove", "POST"))
	assert.Equal(t, 5, GetBusinessType("/system/user/export", "POST"))
	assert.Equal(t, 6, GetBusinessType("/system/user/import", "POST"))
	assert.Equal(t, 0, GetBusinessType("/system/user/list", "POST"))
}

func TestGetMethodDescription(t *testing.T) {
	assert.Equal(t, "新增", getMethodDescription("/x/add", "POST"))
	assert.Equal(t, "修改", getMethodDescription("/x/update", "POST"))
	assert.Equal(t, "删除", getMethodDescription("/x/delete", "POST"))
	assert.Equal(t, "导出", getMethodDescription("/x/export", "POST"))
	assert.Equal(t, "导入", getMethodDescription("/x/import", "POST"))
	assert.Equal(t, "POST", getMethodDescription("/x/other", "POST"), "未知路径回退 HTTP 方法")
}

func TestSetOperLogInfo(t *testing.T) {
	c, _ := newMWContext("POST", "/system/user")
	SetOperLogInfo(c, "用户管理", 1, "新增")
	title, ok := c.Get("oper_log_title")
	require.True(t, ok)
	assert.Equal(t, "用户管理", title)
	bt, ok := c.Get("oper_log_business_type")
	require.True(t, ok)
	assert.Equal(t, 1, bt)
}

// ---------------- permission.go 纯函数 + 缓存 ----------------

func TestIsValidDataScopeField(t *testing.T) {
	assert.True(t, isValidDataScopeField("dept_id"))
	assert.True(t, isValidDataScopeField("id"))
	assert.False(t, isValidDataScopeField("name"))
	assert.False(t, isValidDataScopeField("dept_id; DROP TABLE x--"), "注入串拒")
	assert.False(t, isValidDataScopeField(""))
}

func TestExtractModulePermission(t *testing.T) {
	assert.Equal(t, "system:menu", extractModulePermission("system:menu:list"))
	assert.Equal(t, "system:user", extractModulePermission("system:user:add"))
	assert.Equal(t, "", extractModulePermission("nocolon"))
	assert.Equal(t, "", extractModulePermission(""))
	assert.Equal(t, "", extractModulePermission(":leading"))
}

func TestInvalidateMiddlewarePermCache(t *testing.T) {
	userID := uuid.NewString()
	// 预置缓存条目
	mwCacheStore(&mwSuperAdminCache, userID, true)
	mwCacheStore(&mwPermCheckCache, userID+"|perm:a", true)
	mwCacheStore(&mwPermCheckCache, userID+"|perm:b", false)
	mwCacheStore(&mwPermCheckCache, "other-user|perm:a", true)

	InvalidateMiddlewarePermCache(userID)

	_, hit := mwCacheLookup(&mwSuperAdminCache, userID)
	assert.False(t, hit, "superadmin 缓存已清")
	_, hit = mwCacheLookup(&mwPermCheckCache, userID+"|perm:a")
	assert.False(t, hit)
	_, hit = mwCacheLookup(&mwPermCheckCache, userID+"|perm:b")
	assert.False(t, hit)
	_, hit = mwCacheLookup(&mwPermCheckCache, "other-user|perm:a")
	assert.True(t, hit, "其他用户缓存不受影响")
}

func TestMwCacheLookupExpiry(t *testing.T) {
	var m sync.Map
	key := uuid.NewString()
	_, hit := mwCacheLookup(&m, key)
	assert.False(t, hit, "未命中")

	mwCacheStore(&m, key, true)
	allow, hit := mwCacheLookup(&m, key)
	assert.True(t, hit)
	assert.True(t, allow)

	// 过期条目惰性删除
	m.Store(key, mwPermCacheEntry{allow: true, expiresAt: time.Now().Add(-time.Second)})
	_, hit = mwCacheLookup(&m, key)
	assert.False(t, hit, "过期条目不命中")
}

// ---------------- permission.go 中间件(带 sqlite core) ----------------

// seedPermData 建用户-角色-菜单链路:userID 持有 perms 权限。
// extraTables 为数据权限测试补建 sys_user/sys_dept/sys_role_dept。
func seedPermData(t *testing.T, userID string, perms ...string) (*core.Core, string) {
	t.Helper()
	roleID := uuid.NewString()
	c := newTestCoreWithSQLite(t, func(db *gorm.DB) {
		require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, status, del_flag) VALUES (?, 'r', 'test_role', 0, 0)`, roleID).Error)
		require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`, userID, roleID).Error)
		for _, p := range perms {
			menuID := uuid.NewString()
			require.NoError(t, db.Exec(`INSERT INTO sys_menu (id, menu_name, menu_type, status, visible, perms) VALUES (?, 'm', 'F', 0, 1, ?)`, menuID, p).Error)
			require.NoError(t, db.Exec(`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`, roleID, menuID).Error)
		}
	})
	return c, roleID
}

func TestPermissionMiddleware(t *testing.T) {
	// 未认证 → 401
	c, w := newMWContext("GET", "/x")
	Permission("system:user:list", nil)(c)
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 持有权限 → 放行
	userID := uuid.NewString()
	core, _ := seedPermData(t, userID, "system:user:list")
	c2, _ := newMWContext("GET", "/x")
	c2.Set("user_id", userID)
	Permission("system:user:list", core)(c2)
	assert.False(t, c2.IsAborted())

	// 无权限 → 403
	userID2 := uuid.NewString()
	core2, _ := seedPermData(t, userID2) // 无任何菜单
	c3, w3 := newMWContext("GET", "/x")
	c3.Set("user_id", userID2)
	Permission("system:user:list", core2)(c3)
	assert.True(t, c3.IsAborted())
	assert.Equal(t, http.StatusForbidden, w3.Code)
}

func TestRequirePermissions(t *testing.T) {
	// 未认证 → 401
	c, _ := newMWContext("GET", "/x")
	RequirePermissions([]string{"a:b:c"}, nil)(c)
	assert.True(t, c.IsAborted())

	// 持有任一 → 放行
	userID := uuid.NewString()
	core, _ := seedPermData(t, userID, "ops:building:list")
	c2, _ := newMWContext("GET", "/x")
	c2.Set("user_id", userID)
	RequirePermissions([]string{"ops:floor:list", "ops:building:list"}, core)(c2)
	assert.False(t, c2.IsAborted())

	// 全无 → 403
	userID2 := uuid.NewString()
	core2, _ := seedPermData(t, userID2, "system:menu:list")
	c3, w3 := newMWContext("GET", "/x")
	c3.Set("user_id", userID2)
	RequirePermissions([]string{"ops:floor:list", "ops:building:list"}, core2)(c3)
	assert.True(t, c3.IsAborted())
	assert.Equal(t, http.StatusForbidden, w3.Code)

	// 超级管理员 → 直接放行
	adminID := uuid.NewString()
	coreAdmin := newTestCoreWithSQLite(t, func(db *gorm.DB) {
		roleID := uuid.NewString()
		require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, status, del_flag) VALUES (?, 'admin', 'admin', 0, 0)`, roleID).Error)
		require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`, adminID, roleID).Error)
	})
	c4, _ := newMWContext("GET", "/x")
	c4.Set("user_id", adminID)
	RequirePermissions([]string{"any:thing:atall"}, coreAdmin)(c4)
	assert.False(t, c4.IsAborted(), "admin 角色直接放行")
}

func TestRequireAllPermissions(t *testing.T) {
	userID := uuid.NewString()
	core, _ := seedPermData(t, userID, "a:b:c", "a:b:d")

	// 全有 → 放行
	c, _ := newMWContext("GET", "/x")
	c.Set("user_id", userID)
	RequireAllPermissions([]string{"a:b:c", "a:b:d"}, core)(c)
	assert.False(t, c.IsAborted())

	// 缺一 → 403
	userID2 := uuid.NewString()
	core2, _ := seedPermData(t, userID2, "a:b:c")
	c2, w2 := newMWContext("GET", "/x")
	c2.Set("user_id", userID2)
	RequireAllPermissions([]string{"a:b:c", "a:b:d"}, core2)(c2)
	assert.True(t, c2.IsAborted())
	assert.Equal(t, http.StatusForbidden, w2.Code)

	// 未认证 → 401
	c3, _ := newMWContext("GET", "/x")
	RequireAllPermissions([]string{"a:b:c"}, nil)(c3)
	assert.True(t, c3.IsAborted())
}

func TestIsQueryPath_7408(t *testing.T) {
	assert.True(t, isQueryPath("/ops/building/list"))
	assert.True(t, isQueryPath("/system/departments/tree"))
	assert.True(t, isQueryPath("/ops/building/list/"), "尾斜杠容错")
	assert.False(t, isQueryPath("/ops/building"))
	assert.False(t, isQueryPath("/ops/building/create"))
	assert.False(t, isQueryPath("list"), "无斜杠")
}

func TestDataScopePermission(t *testing.T) {
	// 未认证 → 401
	c, _ := newMWContext("GET", "/x")
	DataScopePermission(nil)(c)
	assert.True(t, c.IsAborted())

	// admin → 直接放行(不写 data_scope)
	adminID := uuid.NewString()
	coreAdmin := newTestCoreWithSQLite(t, func(db *gorm.DB) {
		roleID := uuid.NewString()
		require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, status, del_flag) VALUES (?, 'admin', 'admin', 0, 0)`, roleID).Error)
		require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`, adminID, roleID).Error)
	})
	c2, _ := newMWContext("GET", "/x")
	c2.Set("user_id", adminID)
	DataScopePermission(coreAdmin)(c2)
	assert.False(t, c2.IsAborted())

	// 普通用户 → data_scope 写入 ctx
	userID := uuid.NewString()
	core, _ := seedPermData(t, userID) // 角色 data_scope 默认 1
	c3, _ := newMWContext("GET", "/x")
	c3.Set("user_id", userID)
	DataScopePermission(core)(c3)
	assert.False(t, c3.IsAborted())
	ds, ok := c3.Get("data_scope")
	require.True(t, ok)
	assert.Equal(t, models.DataScope(1), ds)
}

func TestGetUserMaxDataScope(t *testing.T) {
	userID := uuid.NewString()
	// data_scope=3 的角色
	core := newTestCoreWithSQLite(t, func(db *gorm.DB) {
		roleID := uuid.NewString()
		require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, data_scope, status, del_flag) VALUES (?, 'r', 'ds_role', 3, 0, 0)`, roleID).Error)
		require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`, userID, roleID).Error)
	})
	scope, err := getUserMaxDataScope(core, userID)
	require.NoError(t, err)
	assert.Equal(t, models.DataScope(3), scope)

	// 无角色 → COALESCE 兜底 DataScopeSelf(5)
	scope2, err := getUserMaxDataScope(core, uuid.NewString())
	require.NoError(t, err)
	assert.Equal(t, models.DataScopeSelf, scope2)
}

// seedDataScopeTables 补建数据权限测试所需的 sys_user/sys_dept/sys_role_dept。
func seedDataScopeTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_user (id TEXT PRIMARY KEY, dept_id TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_dept (id TEXT PRIMARY KEY, parent_id TEXT, dept_id TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_role_dept (role_id TEXT, dept_id TEXT)`).Error)
}

func TestApplyDataScope(t *testing.T) {
	userID := uuid.NewString()
	roleID := uuid.NewString()

	core := newTestCoreWithSQLite(t, func(db *gorm.DB) {
		seedDataScopeTables(t, db)
		require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, status, del_flag) VALUES (?, 'r', 'x', 0, 0)`, roleID).Error)
		require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`, userID, roleID).Error)
		require.NoError(t, db.Exec(`INSERT INTO sys_user (id, dept_id) VALUES (?, 'd1')`, userID).Error)
		// 部门树: d1 → d2 → d3
		require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, parent_id, dept_id) VALUES ('d1', NULL, 'd1')`).Error)
		require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, parent_id, dept_id) VALUES ('d2', 'd1', 'd2')`).Error)
		require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, parent_id, dept_id) VALUES ('d3', 'd2', 'd3')`).Error)
		// 自定义数据权限: role → d9
		require.NoError(t, db.Exec(`INSERT INTO sys_role_dept (role_id, dept_id) VALUES (?, 'd9')`, roleID).Error)
	})
	

	q := func() *gorm.DB { return core.GetDB().Table("sys_dept") }

	// 非法字段 → 1=0
	require.NotPanics(t, func() {
		_ = ApplyDataScope(q(), userID, models.DataScopeAll, core, "evil;drop").Find(&[]map[string]interface{}{})
	})

	// DataScopeAll → 不过滤
	var rows []map[string]interface{}
	require.NoError(t, ApplyDataScope(q(), userID, models.DataScopeAll, core, "dept_id").Find(&rows).Error)
	assert.Len(t, rows, 3, "全部数据权限不过滤")

	// DataScopeCustom → rd 表的 d9
	rows = nil
	require.NoError(t, ApplyDataScope(q(), userID, models.DataScopeCustom, core, "id").Find(&rows).Error)
	assert.Len(t, rows, 0, "d9 不在 sys_dept,0 行但 SQL 合法")

	// DataScopeDept → dept_id = d1
	rows = nil
	require.NoError(t, ApplyDataScope(q(), userID, models.DataScopeDept, core, "dept_id").Find(&rows).Error)
	assert.Len(t, rows, 1, "d2 的 parent 是 d1")

	// DataScopeDeptChild → d1 及后代
	rows = nil
	require.NoError(t, ApplyDataScope(q(), userID, models.DataScopeDeptChild, core, "id").Find(&rows).Error)
	assert.Len(t, rows, 3, "d1/d2/d3 全部")

	// DataScopeSelf → dept_id = userID
	rows = nil
	require.NoError(t, ApplyDataScope(q(), userID, models.DataScopeSelf, core, "dept_id").Find(&rows).Error)
	assert.Len(t, rows, 0)

	// 未知 scope → 1=0
	rows = nil
	require.NoError(t, ApplyDataScope(q(), userID, models.DataScope(99), core, "dept_id").Find(&rows).Error)
	assert.Len(t, rows, 0)
}

func TestApplyDataScopeFromContext(t *testing.T) {
	userID := uuid.NewString()
	core := newTestCoreWithSQLite(t, func(db *gorm.DB) {
		seedDataScopeTables(t, db)
	})

	// 无 user_id → 原样返回
	c, _ := newMWContext("GET", "/x")
	base := core.GetDB().Table("sys_dept")
	out := ApplyDataScopeFromContext(c, base, core, "dept_id")
	assert.NotNil(t, out)

	// 非 admin + 有 data_scope → 走 ApplyDataScope(Self)
	c2, _ := newMWContext("GET", "/x")
	c2.Set("user_id", userID)
	c2.Set("data_scope", models.DataScopeSelf)
	out2 := ApplyDataScopeFromContext(c2, base, core, "dept_id")
	require.NotNil(t, out2)
	var rows []map[string]interface{}
	require.NoError(t, out2.Find(&rows).Error)
	assert.Len(t, rows, 0, "DataScopeSelf: dept_id=userID 无匹配")

	// 无 data_scope → 不过滤
	c3, _ := newMWContext("GET", "/x")
	c3.Set("user_id", userID)
	out3 := ApplyDataScopeFromContext(c3, base, core, "dept_id")
	require.NotNil(t, out3)
}

func TestHasUserPermission(t *testing.T) {
	c, _ := newMWContext("GET", "/x")
	assert.False(t, HasUserPermission(nil, nil, "x"), "nil ctx → false")
	assert.False(t, HasUserPermission(c, nil, "x"), "nil core → false")
	assert.False(t, HasUserPermission(c, nil, ""), "空 perm → false")

	// 无 user_id → false
	core := newTestCoreWithSQLite(t, nil)
	assert.False(t, HasUserPermission(c, core, "a:b:c"))

	// 持有权限 → true
	userID := uuid.NewString()
	core2, _ := seedPermData(t, userID, "asset:reconciliation:list")
	c2, _ := newMWContext("GET", "/x")
	c2.Set("user_id", userID)
	assert.True(t, HasUserPermission(c2, core2, "asset:reconciliation:list"))
	assert.False(t, HasUserPermission(c2, core2, "asset:reconciliation:delete"))
}
