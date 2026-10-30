package system

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	coredb "github.com/xingran-next/xingran-go-backend/internal/core/db"
)

// =====================================================================
// Phase 74-04: helpers.go (parseInt) + fix_menu_handler.go tests.
// =====================================================================

func TestParseInt(t *testing.T) {
	assert.Equal(t, 42, parseInt("42"))
	assert.Equal(t, 0, parseInt("not-a-number"))
	assert.Equal(t, 0, parseInt(""))
	assert.Equal(t, -7, parseInt("-7"))
}

func TestFixMenuPathsHandler_FixesPaths(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
		CREATE TABLE sys_menu (
			id TEXT PRIMARY KEY,
			menu_name TEXT,
			parent_id TEXT,
			path TEXT,
			component TEXT,
			visible INTEGER DEFAULT 1,
			status INTEGER DEFAULT 0
		)
	`).Error)
	// 用户中心 root + 3 children with the legacy short paths.
	require.NoError(t, gdb.Exec(`INSERT INTO sys_menu (id, menu_name, parent_id, path) VALUES ('m-root', '用户中心', NULL, 'user')`).Error)
	require.NoError(t, gdb.Exec(`INSERT INTO sys_menu (id, menu_name, parent_id, path) VALUES ('m-1', '个人中心', 'm-root', 'profile')`).Error)
	require.NoError(t, gdb.Exec(`INSERT INTO sys_menu (id, menu_name, parent_id, path) VALUES ('m-2', '系统设置', 'm-root', 'settings')`).Error)
	require.NoError(t, gdb.Exec(`INSERT INTO sys_menu (id, menu_name, parent_id, path) VALUES ('m-3', '我的通知', 'm-root', 'my-notices')`).Error)

	coreObj := &core.Core{CoreInfra: &core.CoreInfra{DB: &coredb.Database{DB: gdb}}}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/system/fix-menu-paths", nil)

	FixMenuPathsHandler(c, coreObj)
	assert.Equal(t, 200, w.Code)

	var path string
	require.NoError(t, gdb.Raw(`SELECT path FROM sys_menu WHERE id = 'm-1'`).Scan(&path).Error)
	assert.Equal(t, "user/profile", path)
	require.NoError(t, gdb.Raw(`SELECT path FROM sys_menu WHERE id = 'm-2'`).Scan(&path).Error)
	assert.Equal(t, "user/settings", path)
	require.NoError(t, gdb.Raw(`SELECT path FROM sys_menu WHERE id = 'm-3'`).Scan(&path).Error)
	assert.Equal(t, "user/my-notices", path)
}

func TestFixMenuPathsHandler_MissingRoot(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`CREATE TABLE sys_menu (id TEXT PRIMARY KEY, menu_name TEXT, parent_id TEXT, path TEXT, component TEXT)`).Error)

	coreObj := &core.Core{CoreInfra: &core.CoreInfra{DB: &coredb.Database{DB: gdb}}}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/system/fix-menu-paths", nil)

	FixMenuPathsHandler(c, coreObj)
	// No 用户中心 root row → NotFound business error. apperrors.NotFound maps to
	// HTTP 400 in this app's response.Error (documented 74-03 quirk #1).
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "用户中心")
}
