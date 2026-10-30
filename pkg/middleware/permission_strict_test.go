// Package middleware 提供 v1.24 Phase 78（#16）严格权限中间件的回归测试。
//
// 背景：写操作权限此前存在**两个叠加病灶**——
//  ① 组级 ANY-of：/system/users 组挂 [UserList, UserAdd, UserEdit, UserView]，
//     只持 system:user:list 的用户调删除接口命中 UserList 分支 → 200 且真删除；
//  ② 模块级降级：checkUserPermission 的 Step 3 把 system:user:remove 降级匹配到
//     页面本体 system:user → 只读角色仍被放行。
//
// 本文件锁定 RequireStrictPermissions / checkUserPermissionStrict 的语义：
//   - 只做 Step 1 精确匹配（跳过 Step 2 F 型继承、Step 3 模块降级）
//   - 超管（role_key='admin'）bypass
//   - 精确命中即放行
//   - 与既有 checkUserPermission 行为**并存不冲突**（A1 同时断言两者差异）
//
// 复用 permission_inherit_test.go 的 SQLite 测试夹具
// （newTestCoreWithSQLite / createTables，同包可见）。
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// strictFixture 构造一个「用户 + 角色 + 若干菜单授权」的最小夹具。
//
// grantedPerms 为授予该角色的 sys_menu.perms 列表；
// roleKey 为角色标识（"admin" 即超管，用于验证 bypass）。
func strictFixture(t *testing.T, roleKey string, grantedPerms []string) (*core.Core, string) {
	t.Helper()

	userID := uuid.New().String()
	roleID := uuid.New().String()

	coreInst := newTestCoreWithSQLite(t, func(gormDB *gorm.DB) {
		require.NoError(t, gormDB.Exec(
			`INSERT INTO sys_role (id, role_name, role_key, status, del_flag) VALUES (?, ?, ?, 0, 0)`,
			roleID, "测试角色", roleKey,
		).Error)
		require.NoError(t, gormDB.Exec(
			`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`,
			userID, roleID,
		).Error)

		for _, perms := range grantedPerms {
			menuID := uuid.New().String()
			// 页面本体（menu_type='C'）与按钮（menu_type='F'）都建，
			// 以便同时覆盖「只持页面本体」的降级场景与「精确持有 F 权限」的场景。
			menuType := "F"
			if perms == "system:user" {
				menuType = "C"
			}
			require.NoError(t, gormDB.Exec(
				`INSERT INTO sys_menu (id, menu_name, parent_id, menu_type, perms, status) VALUES (?, ?, NULL, ?, ?, 0)`,
				menuID, "菜单-"+perms, menuType, perms,
			).Error)
			require.NoError(t, gormDB.Exec(
				`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`,
				roleID, menuID,
			).Error)
		}
	})

	return coreInst, userID
}

// runStrictMiddleware 以 RequireStrictPermissions 挂载一条路由并发起请求。
// 返回 HTTP 状态码。
func runStrictMiddleware(t *testing.T, coreInst *core.Core, userID string, perms []string) int {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	// 模拟鉴权中间件：把 user_id 放进 gin 上下文（getUserIDAsString 从此处读取）
	engine.POST("/write", func(c *gin.Context) {
		if userID != "" {
			c.Set("user_id", userID)
		}
		c.Next()
	}, RequireStrictPermissions(perms, coreInst), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/write", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	return rec.Code
}

// TestStrictPermission_SkipsModuleDowngrade（A1）
//
// 核心证据：只持「页面本体 system:user」的角色——
//   - 严格检查 system:user:remove → false（Phase 78 修好后必须拒绝）
//   - 既有 checkUserPermission → true（证明 Step 3 模块降级确实存在，是本 phase 的修复靶点）
func TestStrictPermission_SkipsModuleDowngrade(t *testing.T) {
	coreInst, userID := strictFixture(t, "readonly_role", []string{"system:user"})

	// 既有（降级）检查：Step 3 命中页面本体 → 放行 —— 这是被修掉的旧行为
	assert.True(t, checkUserPermission(coreInst, userID, "system:user:remove"),
		"既有 checkUserPermission 应因 Step 3 模块降级放行（缺陷证据）")

	// 严格检查：仅 Step 1 精确匹配 → 拒绝
	assert.False(t, checkUserPermissionStrict(coreInst, userID, "system:user:remove"),
		"严格检查不得做模块级降级（只持页面权限应被拒绝）")

	// 中间件层同样拒绝（403）
	assert.Equal(t, http.StatusForbidden,
		runStrictMiddleware(t, coreInst, userID, []string{"system:user:remove"}),
		"只持页面权限的用户调写接口应 403")
}

// TestStrictPermission_SuperAdminBypass（A2）
//
// 超管（role_key='admin'）即便没有任何菜单授权也必须放行。
func TestStrictPermission_SuperAdminBypass(t *testing.T) {
	coreInst, userID := strictFixture(t, "admin", nil)

	assert.True(t, isSuperAdmin(coreInst, userID), "role_key='admin' 应识别为超管")
	assert.Equal(t, http.StatusOK,
		runStrictMiddleware(t, coreInst, userID, []string{"system:user:remove"}),
		"超管应 bypass 严格权限检查")
}

// TestStrictPermission_ExactMatchPasses（A3）
//
// 精确持有 system:user:remove 的角色 → 放行；未持有的权限码 → 拒绝。
func TestStrictPermission_ExactMatchPasses(t *testing.T) {
	coreInst, userID := strictFixture(t, "writer_role", []string{"system:user:remove"})

	assert.True(t, checkUserPermissionStrict(coreInst, userID, "system:user:remove"),
		"精确命中应放行")
	assert.False(t, checkUserPermissionStrict(coreInst, userID, "system:user:edit"),
		"未持有的权限码应拒绝")

	assert.Equal(t, http.StatusOK,
		runStrictMiddleware(t, coreInst, userID, []string{"system:user:remove"}),
		"精确命中应通过中间件")
	assert.Equal(t, http.StatusForbidden,
		runStrictMiddleware(t, coreInst, userID, []string{"system:user:edit"}),
		"未持有的权限码应 403")
}

// TestStrictPermission_UnauthenticatedRejected 验证未认证请求被拒。
func TestStrictPermission_UnauthenticatedRejected(t *testing.T) {
	coreInst, _ := strictFixture(t, "writer_role", []string{"system:user:remove"})

	assert.Equal(t, http.StatusUnauthorized,
		runStrictMiddleware(t, coreInst, "", []string{"system:user:remove"}),
		"未认证应 401")
}

// TestStrictPermission_CacheKeyIsolated 验证严格检查与既有检查的缓存条目互不污染。
//
// 同 userID + 同权限码下：既有检查（降级）为 true，严格检查为 false ——
// 若两者共用缓存键，第二个断言必然失败。
func TestStrictPermission_CacheKeyIsolated(t *testing.T) {
	coreInst, userID := strictFixture(t, "readonly_role", []string{"system:user"})

	assert.True(t, checkUserPermission(coreInst, userID, "system:user:remove"))
	assert.False(t, checkUserPermissionStrict(coreInst, userID, "system:user:remove"),
		"严格检查缓存不得复用降级检查的条目")
}
