package system

// anchor_guard_handler_coverage_test.go —— 补齐 anchor_grant_guard.go、
// user_permission_preview_handler.go、role/post PreviewAnchor handler 的 diff 分支。
//
// 全部用 gin.CreateTestContext + 最小 core（仅注入 DB）构造，不启动 HTTP 服务。

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	coredb "github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

func newGuardTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "guard.db")), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})
	for _, ddl := range []string{
		`CREATE TABLE sys_dept (id TEXT PRIMARY KEY, dept_name TEXT, ancestors TEXT DEFAULT '',
			org_level INTEGER DEFAULT 0, is_external_org INTEGER DEFAULT 0, status INTEGER DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE sys_user (id TEXT PRIMARY KEY, username TEXT, nickname TEXT, dept_id TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE sys_role (id TEXT PRIMARY KEY, role_name TEXT, role_key TEXT, data_scope INTEGER DEFAULT 1,
			anchor_mode INTEGER DEFAULT 0, anchor_param TEXT DEFAULT '', status INTEGER DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE sys_post (id TEXT PRIMARY KEY, post_code TEXT, post_name TEXT, post_sort INTEGER DEFAULT 0,
			anchor_mode INTEGER DEFAULT 0, anchor_param TEXT DEFAULT '', status INTEGER DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE sys_user_role (user_id TEXT, role_id TEXT)`,
		`CREATE TABLE sys_user_post (user_id TEXT, post_id TEXT)`,
		`CREATE TABLE sys_role_menu (role_id TEXT, menu_id TEXT)`,
		`CREATE TABLE sys_menu (id TEXT PRIMARY KEY, menu_name TEXT, perms TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(ddl).Error)
	}
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, ancestors, org_level, is_external_org, status) VALUES
		('D-hb','湖北分公司','',1,1,0), ('D-wh','武汉中支','D-hb',2,1,0), ('D-zh','综合部','D-hb,D-wh',0,0,0)`).Error)
	return db
}

func newGuardHandler(t *testing.T, db *gorm.DB) *UserHandler {
	t.Helper()
	h := NewUserHandler(nil)
	h.core = &core.Core{
		CoreInfra:    &core.CoreInfra{DB: &coredb.Database{DB: db}},
		CoreServices: &core.CoreServices{},
	}
	return h
}

func newGuardContext(t *testing.T, userID string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString("{}"))
	if userID != "" {
		c.Set("user_id", userID)
	}
	return c, w
}

func parseRespCode(t *testing.T, w *httptest.ResponseRecorder) int {
	t.Helper()
	var r response.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &r), "body=%s", w.Body.String())
	return r.Code
}

func TestGuardPostGrants_NoCoreAndEmptyIDs(t *testing.T) {
	// core 未注入 → 明确 400，不放行（行 109-113）。
	h := NewUserHandler(nil)
	c, w := newGuardContext(t, "op")
	assert.False(t, h.guardPostGrants(c, []string{"P"}, "target"))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 无岗位 → 直接放行，无需 core（行 106/107）。
	c2, w2 := newGuardContext(t, "op")
	assert.True(t, h.guardPostGrants(c2, nil, "target"))
	assert.Empty(t, w2.Body.String(), "放行时不应写响应")
}

func TestOperatorGrantContextBranches(t *testing.T) {
	db := newGuardTestDB(t)
	h := newGuardHandler(t, db)

	// 无 user_id → 保守拒绝（行 57/58）。
	c, _ := newGuardContext(t, "")
	uid, hasEdit, isAdmin := h.operatorGrantContext(c)
	assert.Empty(t, uid)
	assert.False(t, hasEdit)
	assert.False(t, isAdmin)

	// 有 user_id 但 core 为 nil → 同上。
	hNil := NewUserHandler(nil)
	c2, _ := newGuardContext(t, "U-op")
	uid, hasEdit, isAdmin = hNil.operatorGrantContext(c2)
	assert.Equal(t, "U-op", uid)
	assert.False(t, hasEdit)
	assert.False(t, isAdmin)

	// 普通用户：无 admin 角色、无 post:edit 权限。
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, status) VALUES ('U-plain','plain',0)`).Error)
	c3, _ := newGuardContext(t, "U-plain")
	_, hasEdit, isAdmin = h.operatorGrantContext(c3)
	assert.False(t, isAdmin)
	assert.False(t, hasEdit)

	// 持启用角色的普通管理员：有 post:edit 菜单权限。
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, status) VALUES ('U-mgr','mgr',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, status) VALUES ('R-post','岗位管理员','post_mgr',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_menu (id, menu_name, perms, status) VALUES ('M-post','岗位编辑','system:post:edit',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES ('U-mgr','R-post')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('R-post','M-post')`).Error)
	c4, _ := newGuardContext(t, "U-mgr")
	_, hasEdit, isAdmin = h.operatorGrantContext(c4)
	assert.True(t, hasEdit, "启用角色下的 system:post:edit 应识别")
	assert.False(t, isAdmin)

	// 超管（role_key=admin，启用）→ isSuperAdmin + 天然 post:edit（行 91-93），旁路菜单行。
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, status) VALUES ('U-admin','admin',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, status) VALUES ('R-admin','超管','admin',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES ('U-admin','R-admin')`).Error)
	c5, _ := newGuardContext(t, "U-admin")
	_, hasEdit, isAdmin = h.operatorGrantContext(c5)
	assert.True(t, isAdmin)
	assert.True(t, hasEdit, "超管天然具备 post:edit")

	// 停用的 admin 角色不算超管（与 middleware 同口径）。
	require.NoError(t, db.Exec(`UPDATE sys_role SET status = 1 WHERE id = 'R-admin'`).Error)
	c6, _ := newGuardContext(t, "U-admin")
	_, _, isAdmin = h.operatorGrantContext(c6)
	assert.False(t, isAdmin)
}

func TestOperatorGrantContextDBErrors(t *testing.T) {
	db := newGuardTestDB(t)
	h := newGuardHandler(t, db)

	// 缺 sys_user_role → 超管查询失败，保守返回（行 68/69）。
	require.NoError(t, db.Exec(`DROP TABLE sys_user_role`).Error)
	c, _ := newGuardContext(t, "U-1")
	_, hasEdit, isAdmin := h.operatorGrantContext(c)
	assert.False(t, hasEdit)
	assert.False(t, isAdmin)

	// 重建关联表但缺 sys_menu → 权限查询失败，保留 isSuperAdmin（行 85/86）。
	require.NoError(t, db.Exec(`CREATE TABLE sys_user_role (user_id TEXT, role_id TEXT)`).Error)
	require.NoError(t, db.Exec(`DROP TABLE sys_menu`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, status) VALUES ('R','r','admin',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES ('U-1','R')`).Error)
	c2, _ := newGuardContext(t, "U-1")
	_, hasEdit, isAdmin = h.operatorGrantContext(c2)
	assert.False(t, hasEdit)
	assert.True(t, isAdmin, "权限查询失败时保留已判定的超管身份，但 post:edit 保守为 false")
}

func TestGuardPostGrants_DecisionBranches(t *testing.T) {
	db := newGuardTestDB(t)
	h := newGuardHandler(t, db)

	// 挂锚点岗位 + 有 post:edit 的操作者 → 放行（含 LogAnchorGrant 行 135-137）。
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, status) VALUES ('U-op','op',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, status) VALUES ('R-post','岗位管理员','post_mgr',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_menu (id, menu_name, perms, status) VALUES ('M-post','岗位编辑','system:post:edit',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES ('U-op','R-post')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('R-post','M-post')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_post (id, post_code, post_name, anchor_mode, anchor_param, status)
		VALUES ('P-anchor','mid','中支管理员',5,'2',0), ('P-normal','normal','普通岗位',0,'',0)`).Error)

	c, w := newGuardContext(t, "U-op")
	assert.True(t, h.guardPostGrants(c, []string{"P-anchor", "P-normal"}, "target"))
	assert.Empty(t, w.Body.String())

	// 无 post:edit 的操作者挂锚点岗位 → 400 拒绝（行 127-131）。
	c2, w2 := newGuardContext(t, "U-none")
	assert.False(t, h.guardPostGrants(c2, []string{"P-anchor"}, "target"))
	assert.Equal(t, http.StatusBadRequest, w2.Code)
	assert.Contains(t, w2.Body.String(), "system:post:edit")

	// 查询失败（缺 sys_post）→ 500，不放行也不误报权限不足（行 121-124）。
	require.NoError(t, db.Exec(`DROP TABLE sys_post`).Error)
	c3, w3 := newGuardContext(t, "U-op")
	assert.False(t, h.guardPostGrants(c3, []string{"P-normal"}, "target"))
	assert.Equal(t, http.StatusInternalServerError, w3.Code)
}

func TestUserPreviewHandlerBranches(t *testing.T) {
	db := newGuardTestDB(t)
	h := newGuardHandler(t, db)

	// JSON 解析失败 → 400（行 37-39）。
	c, w := newGuardContext(t, "op")
	c.Request = httptest.NewRequest(http.MethodPost, "/preview", bytes.NewBufferString("{bad"))
	c.Request.Header.Set("Content-Type", "application/json")
	h.PreviewUserPermission(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// core 为 nil → 400（行 41-43）。
	hNil := NewUserHandler(nil)
	c2, w2 := newGuardContext(t, "op")
	c2.Request = httptest.NewRequest(http.MethodPost, "/preview", bytes.NewBufferString(`{"deptId":"D-zh"}`))
	c2.Request.Header.Set("Content-Type", "application/json")
	hNil.PreviewUserPermission(c2)
	assert.Equal(t, http.StatusBadRequest, w2.Code)

	// 成功：表单当前值 → 预览画像（行 46-53）。
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, data_scope, status)
		VALUES ('R-self','普通用户',5,0)`).Error)
	c3, w3 := newGuardContext(t, "op")
	c3.Request = httptest.NewRequest(http.MethodPost, "/preview",
		bytes.NewBufferString(`{"deptId":"D-wh","roleIds":["R-self"]}`))
	c3.Request.Header.Set("Content-Type", "application/json")
	h.PreviewUserPermission(c3)
	var okResp response.Response
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &okResp))
	assert.Equal(t, 0, okResp.Code)
	data, _ := json.Marshal(okResp.Data)
	assert.Contains(t, string(data), "仅本人数据")

	// service 查询失败（缺 sys_role）→ 错误响应（行 49-51）。
	require.NoError(t, db.Exec(`DROP TABLE sys_role`).Error)
	c4, w4 := newGuardContext(t, "op")
	c4.Request = httptest.NewRequest(http.MethodPost, "/preview",
		bytes.NewBufferString(`{"deptId":"D-wh","roleIds":["R-self"]}`))
	c4.Request.Header.Set("Content-Type", "application/json")
	h.PreviewUserPermission(c4)
	assert.NotEqual(t, 0, parseRespCode(t, w4))
}

func TestRoleAndPostPreviewHandlers(t *testing.T) {
	db := newGuardTestDB(t)

	// 复用真实 service，注入带 DB 的 core。
	roleH := NewRoleHandler(system.NewRoleService(db))
	roleH.core = &core.Core{CoreInfra: &core.CoreInfra{DB: &coredb.Database{DB: db}}, CoreServices: &core.CoreServices{}}
	postH := NewPostHandler(system.NewPostService(db))
	postH.core = &core.Core{CoreInfra: &core.CoreInfra{DB: &coredb.Database{DB: db}}, CoreServices: &core.CoreServices{}}

	// 空 id → 400（role 行 294-296；post 行 312-314）。
	for _, tc := range []struct {
		name    string
		handler func(*gin.Context)
	}{
		{"role", roleH.PreviewAnchor},
		{"post", postH.PreviewAnchor},
	} {
		t.Run(tc.name+"-empty-id", func(t *testing.T) {
			c, w := newGuardContext(t, "op")
			c.Params = gin.Params{}
			tc.handler(c)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}

	// service 错误（不存在）→ 非 0 code（role 行 300-302；post 行 318-320）。
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, anchor_mode, anchor_param, status)
		VALUES ('R','r',0,'',0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_post (id, post_code, post_name, anchor_mode, anchor_param, status)
		VALUES ('P','p','p',0,'',0)`).Error)
	c, w := newGuardContext(t, "op")
	c.Params = gin.Params{{Key: "id", Value: "R-missing"}}
	roleH.PreviewAnchor(c)
	assert.NotEqual(t, 0, parseRespCode(t, w))

	c2, w2 := newGuardContext(t, "op")
	c2.Params = gin.Params{{Key: "id", Value: "P-missing"}}
	postH.PreviewAnchor(c2)
	assert.NotEqual(t, 0, parseRespCode(t, w2))

	// 成功：空任职人 → 200 且 data 含 totalCount=0。
	c3, w3 := newGuardContext(t, "op")
	c3.Params = gin.Params{{Key: "id", Value: "R"}}
	roleH.PreviewAnchor(c3)
	var r response.Response
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &r))
	assert.Equal(t, 0, r.Code)
	data, _ := json.Marshal(r.Data)
	assert.Contains(t, string(data), `"totalCount":0`)

	c4, w4 := newGuardContext(t, "op")
	c4.Params = gin.Params{{Key: "id", Value: "P"}}
	postH.PreviewAnchor(c4)
	require.NoError(t, json.Unmarshal(w4.Body.Bytes(), &r))
	assert.Equal(t, 0, r.Code)
}

// TestUserHandlerWriteGuardsRejectBeforeService 验证 Create/Update 在岗位准入被拒时
// 立即返回、绝不调用 service（行 75-77 / 214-216）。
// 用 service=nil 的 handler：若 guard 失效会 panic，测试会失败。
func TestUserHandlerWriteGuardsRejectBeforeService(t *testing.T) {
	db := newGuardTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_post (id, post_code, post_name, anchor_mode, anchor_param, status)
		VALUES ('P-anchor','mid','中支管理员',5,'2',0)`).Error)
	// 故意缺 sys_menu：operatorGrantContext 权限查询失败 → 保守无 post:edit → 锚点岗位被拒。
	require.NoError(t, db.Exec(`DROP TABLE sys_menu`).Error)
	require.NoError(t, db.Exec(`DROP TABLE sys_role`).Error)
	require.NoError(t, db.Exec(`DROP TABLE sys_user_role`).Error)

	h := &UserHandler{core: &core.Core{
		CoreInfra:    &core.CoreInfra{DB: &coredb.Database{DB: db}},
		CoreServices: &core.CoreServices{},
	}}

	body := `{"username":"u","password":"p","postIds":["P-anchor"]}`
	t.Run("create", func(t *testing.T) {
		c, w := newGuardContext(t, "U-op")
		c.Request = httptest.NewRequest(http.MethodPost, "/system/users", bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
		assert.NotPanics(t, func() { h.Create(c) })
		assert.NotEqual(t, 0, parseRespCode(t, w), "锚点岗位未授权必须拒绝写入")
	})

	t.Run("update", func(t *testing.T) {
		c, w := newGuardContext(t, "U-op")
		c.Request = httptest.NewRequest(http.MethodPost, "/system/users/U-target/update", bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "U-target"}}
		assert.NotPanics(t, func() { h.Update(c) })
		assert.NotEqual(t, 0, parseRespCode(t, w), "锚点岗位未授权必须拒绝写入")
	})
}
