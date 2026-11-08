package middleware

// FIX-13 / v1.53 Phase 136：OQ-7「5 个权限码读路由恒 403」403→200 对照证据。
//
// 复刻 router.go 的 walls / doors / floor-plan-texts 读组所挂的
// RequirePermissions（ANY-of，非 Strict）语义：
//   - Before：sys_menu 无任何一行承载这 5 个读码（v1.25 遗留状态）→ 403；
//   - After ：migration_236 播种的 F 型按钮行落到 sys_menu 并授权给角色 → 200。
//
// 同时锁定 v1.25 严格权限中间件零回归：
//   - RequireStrictPermissions 仍跳过模块级降级（只持页面码不放行写码）；
//   - 读组 ANY-of 放行不改变 Strict 写组行为（兄弟分离）。
//
// 复用 permission_inherit_test.go 的 newTestCoreWithSQLite / createTables 夹具。

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
	"gorm.io/gorm"
)

// oq7ReadCodes 是 87-EVIDENCE.md §八登记的 5 个读权限码（与 migration_236 一致）。
var oq7ReadCodes = []string{
	string(permission.OpsFloorPlanList),
	string(permission.OpsFloorPlanQuery),
	string(permission.OpsFloorPlanSave),
	string(permission.OpsWallList),
	string(permission.OpsDoorList),
}

// oq7Fixture 构造「普通（非 admin）角色 + 用户」夹具。
// seedCodes 为授予该角色的 sys_menu.perms 列表（模拟 migration 236 播种后的授权结果）。
func oq7Fixture(t *testing.T, seedCodes []string) (*core.Core, string) {
	t.Helper()
	userID := uuid.New().String()
	roleID := uuid.New().String()

	coreInst := newTestCoreWithSQLite(t, func(db *gorm.DB) {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_role (id, role_name, role_key, status) VALUES (?, ?, ?, 0)`,
			roleID, "受限角色", "oq7_readonly",
		).Error)
		require.NoError(t, db.Exec(
			`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`, userID, roleID,
		).Error)

		// 父 C 菜单（ops:floor:list，migration 236 的父挂载点）
		require.NoError(t, db.Exec(
			`INSERT INTO sys_menu (id, menu_name, menu_type, status, perms) VALUES (?, ?, 'C', 0, ?)`,
			"oq7-parent", "楼层管理", "ops:floor:list",
		).Error)
		require.NoError(t, db.Exec(
			`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`, roleID, "oq7-parent",
		).Error)

		for _, perms := range seedCodes {
			menuID := uuid.New().String()
			// F 型按钮行：与 migration 226/227/236 的 ensureActionPermissionMenu 一致
			require.NoError(t, db.Exec(
				`INSERT INTO sys_menu (id, menu_name, parent_id, menu_type, status, perms) VALUES (?, ?, ?, 'F', 0, ?)`,
				menuID, "按钮-"+perms, "oq7-parent", perms,
			).Error)
			require.NoError(t, db.Exec(
				`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`, roleID, menuID,
			).Error)
		}
	})
	return coreInst, userID
}

// runOQ7ReadGroup 以 RequirePermissions(5 个读码) 挂载一条读路由并发起请求，
// 完全复刻 router.go 读组中间件语义。返回 HTTP 状态码。
func runOQ7ReadGroup(t *testing.T, coreInst *core.Core, userID string) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/walls/list", func(c *gin.Context) {
		if userID != "" {
			c.Set("user_id", userID)
		}
		c.Next()
	}, RequirePermissions(oq7ReadCodes, coreInst), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/walls/list", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec.Code
}

// TestOQ7_ReadRoutes_403BeforeSeed_200AfterSeed 是 403→200 的核心对照证据。
func TestOQ7_ReadRoutes_403BeforeSeed_200AfterSeed(t *testing.T) {
	// ---- Before：v1.25 遗留状态，sys_menu 无这 5 个码的菜单行 ----
	beforeCore, beforeUser := oq7Fixture(t, nil)
	assert.Equal(t, http.StatusForbidden, runOQ7ReadGroup(t, beforeCore, beforeUser),
		"OQ-7 修复前：无任何菜单承载 5 个读码，普通角色读路由恒 403")

	// ---- After：migration_236 播种 5 个 F 型按钮并授权给持父菜单的角色 ----
	afterCore, afterUser := oq7Fixture(t, oq7ReadCodes)
	assert.Equal(t, http.StatusOK, runOQ7ReadGroup(t, afterCore, afterUser),
		"OQ-7 修复后：5 个读码已播种/授权，持码角色读路由 200")

	// 逐个码验证「精确持有该读码即可放行」（不只是整体 ANY-of 碰巧命中）
	for _, code := range oq7ReadCodes {
		oneCore, oneUser := oq7Fixture(t, []string{code})
		assert.Equal(t, http.StatusOK, runOQ7ReadGroup(t, oneCore, oneUser),
			"单独持有 %s 的角色读路由应 200", code)
	}
}

// TestOQ7_StrictPermissionZeroRegression 锁定 v1.25 严格权限语义不被种子修复破坏：
// 只持页面本体码（ops:floor）的只读角色，写码严格检查必须拒绝（无模块级降级）。
func TestOQ7_StrictPermissionZeroRegression(t *testing.T) {
	coreInst, userID := strictFixture(t, "readonly_role", []string{"system:user"})

	// RequireStrictPermissions 跳过 Step 3 模块降级（v1.24 Phase 78 #16 语义）
	assert.False(t, checkUserPermissionStrict(coreInst, userID, "system:user:remove"),
		"严格检查不得做模块级降级")
	assert.Equal(t, http.StatusForbidden,
		runStrictMiddleware(t, coreInst, userID, []string{"system:user:remove"}),
		"只读角色调严格写路由仍须 403")

	// 既有（非严格）checkUserPermission 的降级语义保持并存不回退
	assert.True(t, checkUserPermission(coreInst, userID, "system:user:remove"),
		"非严格检查的模块级降级语义保持（与严格检查并存）")
}
