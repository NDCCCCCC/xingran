// Package middleware — RequirePermissionsWithQuery 回归测试
//
// 背景: 「楼宇空间」/「楼宇空间3D」只读可视化页面复用了 building/floor/workstation
// 的 CRUD list 接口拼装数据, 但页面菜单权限标识 (ops:building:spaces:list) 与 CRUD
// 模块权限标识 (ops:building:list) 命名空间割裂, 导致空间管理角色访问页面时 list 接口
// 全部 403。
//
// 修复 (RequirePermissionsWithQuery): 查询类路径 (末段 list/tree) 额外接受可视化读权限,
// 写操作路径保持严格权限, 避免越权。
//
// 验证:
//   - 空间管理角色持 ops:building:spaces:list → /list 通过, 写操作 (create/update/delete) 拒绝
//   - 楼宇管理角色持 ops:building:list → /list 与写操作均通过 (原权限不破坏)
//   - 无权限用户 → /list 拒绝
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// buildingStrictPerms / spacesReadPerm 复用同一组权限, 模拟 router.go 的实际配置
var (
	buildingStrictPerms = []string{"ops:building:list", "ops:building:add", "ops:building:edit", "ops:building:delete"}
	spacesReadPerm      = []string{"ops:building:spaces:list"}
)

// setUserIDMiddleware 测试辅助: 把 user_id 注入 gin.Context (模拟 JWTAuth 之后的状态)
func setUserIDMiddleware(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

// runQueryPermCase 构造一个仅挂载 RequirePermissionsWithQuery 的引擎, 对 handlerPath 发起
// POST 请求, 返回 HTTP 状态码。每次调用建立独立引擎, 避免多路由冲突。
func runQueryPermCase(t *testing.T, coreInst *core.Core, userID, handlerPath, reqPath string) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(setUserIDMiddleware(userID))
	r.POST(handlerPath, RequirePermissionsWithQuery(buildingStrictPerms, spacesReadPerm, coreInst), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, reqPath, nil)
	r.ServeHTTP(w, req)
	return w.Code
}

// grantUserRoleMenu 插入 角色 + 用户-角色 + 菜单 + 角色-菜单, 使 userID 拥有 perm 对应的菜单权限。
func grantUserRoleMenu(t *testing.T, gormDB *gorm.DB, perm string) (userID string) {
	t.Helper()
	userID = uuid.New().String()
	roleID := uuid.New().String()
	menuID := uuid.New().String()
	assert.NoError(t, gormDB.Exec(`INSERT INTO sys_role (id, role_name, role_key, status, del_flag) VALUES (?, ?, ?, 0, 0)`, roleID, "role-"+perm, "r_"+perm).Error)
	assert.NoError(t, gormDB.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`, userID, roleID).Error)
	assert.NoError(t, gormDB.Exec(`INSERT INTO sys_menu (id, menu_name, parent_id, menu_type, perms, status) VALUES (?, ?, NULL, 'C', ?, 0)`, menuID, "m-"+perm, perm).Error)
	assert.NoError(t, gormDB.Exec(`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`, roleID, menuID).Error)
	return userID
}

// TestIsQueryPath 验证查询/写路径判定逻辑 — 安全关键: 写路径绝不能被误判为查询
func TestIsQueryPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/ops/building/list", true},
		{"/ops/floor/list", true},
		{"/ops/floor/tree", true},
		{"/ops/building/list/", true}, // 末尾斜杠容忍
		{"/ops/building", false},      // create
		{"/ops/building/abc/update", false},
		{"/ops/building/abc/delete", false},
		{"/ops/building/batch", false},
		{"/ops/building/geocode", false},
		{"/ops/building/abc", false}, // GetByID 末段为参数值
		{"list", false},              // 无斜杠, 非合法路由形态
	}
	for _, c := range cases {
		assert.Equal(t, c.want, isQueryPath(c.path), "path=%s", c.path)
	}
}

// TestRequirePermissionsWithQuery 集成测试中间件行为
func TestRequirePermissionsWithQuery(t *testing.T) {
	// 场景一: 空间管理角色 (仅持 ops:building:spaces:list)
	coreSpaces := newTestCoreWithSQLite(t, func(gormDB *gorm.DB) {
		grantUserRoleMenu(t, gormDB, "ops:building:spaces:list")
	})

	t.Run("空间角色_查询list_应通过", func(t *testing.T) {
		// 重新取一个持 spaces 权限的 user (上面 setup 里的 user 不可见, 这里再建一个)
		userID := grantUserRoleMenu(t, coreSpaces.DB.GetDB(), "ops:building:spaces:list")
		code := runQueryPermCase(t, coreSpaces, userID, "/ops/building/list", "/ops/building/list")
		assert.Equal(t, http.StatusOK, code)
	})
	t.Run("空间角色_create写操作_应拒绝(防越权)", func(t *testing.T) {
		userID := grantUserRoleMenu(t, coreSpaces.DB.GetDB(), "ops:building:spaces:list")
		code := runQueryPermCase(t, coreSpaces, userID, "/ops/building", "/ops/building")
		assert.Equal(t, http.StatusForbidden, code)
	})
	t.Run("空间角色_delete写操作_应拒绝(防越权)", func(t *testing.T) {
		userID := grantUserRoleMenu(t, coreSpaces.DB.GetDB(), "ops:building:spaces:list")
		code := runQueryPermCase(t, coreSpaces, userID, "/ops/building/:id/delete", "/ops/building/abc/delete")
		assert.Equal(t, http.StatusForbidden, code)
	})

	// 场景二: 楼宇管理角色 (持 ops:building:list) — 原权限不被破坏
	coreBuilding := newTestCoreWithSQLite(t, func(gormDB *gorm.DB) {
		grantUserRoleMenu(t, gormDB, "ops:building:list")
	})
	t.Run("楼宇角色_查询list_应通过", func(t *testing.T) {
		userID := grantUserRoleMenu(t, coreBuilding.DB.GetDB(), "ops:building:list")
		code := runQueryPermCase(t, coreBuilding, userID, "/ops/building/list", "/ops/building/list")
		assert.Equal(t, http.StatusOK, code)
	})
	t.Run("楼宇角色_create写操作_应通过(原权限)", func(t *testing.T) {
		userID := grantUserRoleMenu(t, coreBuilding.DB.GetDB(), "ops:building:list")
		code := runQueryPermCase(t, coreBuilding, userID, "/ops/building", "/ops/building")
		assert.Equal(t, http.StatusOK, code)
	})

	// 场景三: 无权限用户
	coreNone := newTestCoreWithSQLite(t, func(gormDB *gorm.DB) {
		grantUserRoleMenu(t, gormDB, "unrelated:perm") // 给一个无关权限, 确保用户存在但无 building 权限
	})
	t.Run("无权限用户_查询list_应拒绝", func(t *testing.T) {
		userID := grantUserRoleMenu(t, coreNone.DB.GetDB(), "unrelated:perm")
		code := runQueryPermCase(t, coreNone, userID, "/ops/building/list", "/ops/building/list")
		assert.Equal(t, http.StatusForbidden, code)
	})
}

// --- 跨模块选择器接口回归 (router.go: /system/departments, /system/users, /ops/serverRoom) ---
//
// 背景: 楼宇/楼层/工位/机房等运维管理页面内嵌的 <DeptTree>/<DeptSidebar>/用户选择器
// 复用了 system 模块的 /departments/tree、/users/list 接口, 但运维角色通常不持有
// system:dept / system:user 权限, 导致部门树在每个运维页面都 403。
// 修复: 这三个路由组改用 RequirePermissionsWithQuery, 只读路径(/tree,/list)额外接受
// 运维读权限 (ops:building:spaces:list 等), 写操作保持严格。

var (
	deptStrictPerms = []string{"system:dept:list", "system:dept:add", "system:dept:edit", "system:dept:view"}
	userStrictPerms = []string{"system:user:list", "system:user:add", "system:user:edit", "system:user:view"}
	// serverRoomStrictPerms 注意: 路由要求小写 ops:serverroom:* (见 router.go)
	serverRoomStrictPerms = []string{"ops:serverroom:list", "ops:serverroom:add", "ops:serverroom:edit", "ops:serverroom:delete"}
	// dept 组用 opsSelectorReadPerms (router.go 同名 var); user/serverRoom 组用各自的针对性读权限
	deptSelectorExtra       = []string{"ops:building:list", "ops:floor:list", "ops:workstation:list", "ops:serverroom:list", "ops:building:spaces:list"}
	userSelectorExtra       = []string{"ops:workstation:list", "ops:building:spaces:list"}
	serverRoomSelectorExtra = []string{"ops:building:spaces:list"}
)

// runQueryPermCasePerms 通用版: 允许自定义 strict / extra 权限, 测试不同模块的选择器接口。
func runQueryPermCasePerms(t *testing.T, coreInst *core.Core, userID, handlerPath, reqPath string, strict, extra []string) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(setUserIDMiddleware(userID))
	r.POST(handlerPath, RequirePermissionsWithQuery(strict, extra, coreInst), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, reqPath, nil)
	r.ServeHTTP(w, req)
	return w.Code
}

// TestRequirePermissionsWithQuery_SelectorEndpoints 验证跨模块选择器放行:
// 空间管理角色 (仅 ops:building:spaces:list) 可读 部门树/用户列表/机房列表,
// 但 system 部门/用户/机房的写操作被拒 (防越权)。
func TestRequirePermissionsWithQuery_SelectorEndpoints(t *testing.T) {
	coreSpaces := newTestCoreWithSQLite(t, func(gormDB *gorm.DB) {
		grantUserRoleMenu(t, gormDB, "ops:building:spaces:list")
	})

	t.Run("空间角色_读部门树_应通过", func(t *testing.T) {
		userID := grantUserRoleMenu(t, coreSpaces.DB.GetDB(), "ops:building:spaces:list")
		code := runQueryPermCasePerms(t, coreSpaces, userID, "/system/departments/tree", "/system/departments/tree", deptStrictPerms, deptSelectorExtra)
		assert.Equal(t, http.StatusOK, code)
	})
	t.Run("空间角色_读用户列表_应通过", func(t *testing.T) {
		userID := grantUserRoleMenu(t, coreSpaces.DB.GetDB(), "ops:building:spaces:list")
		code := runQueryPermCasePerms(t, coreSpaces, userID, "/system/users/list", "/system/users/list", userStrictPerms, userSelectorExtra)
		assert.Equal(t, http.StatusOK, code)
	})
	t.Run("空间角色_读机房列表_应通过", func(t *testing.T) {
		userID := grantUserRoleMenu(t, coreSpaces.DB.GetDB(), "ops:building:spaces:list")
		code := runQueryPermCasePerms(t, coreSpaces, userID, "/ops/serverRoom/list", "/ops/serverRoom/list", serverRoomStrictPerms, serverRoomSelectorExtra)
		assert.Equal(t, http.StatusOK, code)
	})
	t.Run("空间角色_部门写操作_应拒绝(防越权)", func(t *testing.T) {
		userID := grantUserRoleMenu(t, coreSpaces.DB.GetDB(), "ops:building:spaces:list")
		code := runQueryPermCasePerms(t, coreSpaces, userID, "/system/departments", "/system/departments", deptStrictPerms, deptSelectorExtra)
		assert.Equal(t, http.StatusForbidden, code)
	})
	t.Run("空间角色_用户写操作_应拒绝(防越权)", func(t *testing.T) {
		userID := grantUserRoleMenu(t, coreSpaces.DB.GetDB(), "ops:building:spaces:list")
		code := runQueryPermCasePerms(t, coreSpaces, userID, "/system/users/:id/update", "/system/users/abc/update", userStrictPerms, userSelectorExtra)
		assert.Equal(t, http.StatusForbidden, code)
	})
	t.Run("空间角色_机房写操作_应拒绝(防越权)", func(t *testing.T) {
		userID := grantUserRoleMenu(t, coreSpaces.DB.GetDB(), "ops:building:spaces:list")
		code := runQueryPermCasePerms(t, coreSpaces, userID, "/ops/serverRoom/:id/delete", "/ops/serverRoom/abc/delete", serverRoomStrictPerms, serverRoomSelectorExtra)
		assert.Equal(t, http.StatusForbidden, code)
	})

	// 跨模块: 信息点管理(ops:infopoint)关联网络设备, 需读 /network/devices/list + /network/ports/list 做选择器
	t.Run("信息点角色_读网络设备列表_应通过", func(t *testing.T) {
		userID := grantUserRoleMenu(t, coreSpaces.DB.GetDB(), "ops:infopoint:list")
		code := runQueryPermCasePerms(t, coreSpaces, userID, "/network/devices/list", "/network/devices/list",
			[]string{"network:device:list", "network:device:add", "network:device:edit", "network:device:delete"}, OpsSelectorReadPerms)
		assert.Equal(t, http.StatusOK, code)
	})
	t.Run("信息点角色_读网络端口列表_应通过", func(t *testing.T) {
		userID := grantUserRoleMenu(t, coreSpaces.DB.GetDB(), "ops:infopoint:list")
		code := runQueryPermCasePerms(t, coreSpaces, userID, "/network/ports/list", "/network/ports/list",
			[]string{"network:port:query"}, OpsSelectorReadPerms)
		assert.Equal(t, http.StatusOK, code)
	})
	t.Run("信息点角色_写网络设备_应拒绝(防越权)", func(t *testing.T) {
		userID := grantUserRoleMenu(t, coreSpaces.DB.GetDB(), "ops:infopoint:list")
		code := runQueryPermCasePerms(t, coreSpaces, userID, "/network/devices", "/network/devices",
			[]string{"network:device:list", "network:device:add", "network:device:edit", "network:device:delete"}, OpsSelectorReadPerms)
		assert.Equal(t, http.StatusForbidden, code)
	})
}
