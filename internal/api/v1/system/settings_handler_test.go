package system

// =====================================================================
// Method Enumeration (Plan 72-11 Task 1)
//
// settings_handler.go (91 lines) — method coverage:
//   - GetUserPreferences  GET /system/settings/preferences
//   - UpdateUserPreferences PUT /system/settings/preferences
// =====================================================================

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// setupSettingsTestDB creates in-memory SQLite with sys_user_preference schema.
//
// 建表 SQL 显式带上与 PG 侧一致的 CHECK 约束（theme_style / layout_type /
// layout_density / page_size）。SQLite 原生支持 CHECK，只是 GORM AutoMigrate 不生成；
// 不写出来的话本 bug 在 SQLite 上表现为「静默写空串」而非报错，测不出根因。
//
// 不加 NOT NULL：既有 TC4/TC7 依赖「不指定该列时插入 NULL」；SQLite 的 CHECK
// 对 NULL 结果放行（NULL → UNKNOWN → 通过），不影响既有断言。
func setupSettingsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user_preference (
			id TEXT PRIMARY KEY,
			user_id TEXT UNIQUE,
			theme TEXT,
			theme_style TEXT CHECK (theme_style IN ('minimal','glassmorphism','neumorphism','flat2.0','luxury-quiet')),
			layout_type TEXT CHECK (layout_type IN ('classic','hybrid','innovative')),
			layout_density TEXT CHECK (layout_density IN ('compact','comfortable','spacious')),
			sidebar_width INTEGER,
			sidebar_collapsed_width INTEGER,
			sidebar_collapsed INTEGER,
			page_size INTEGER CHECK (page_size IN (5,10,20,50,100)),
			custom_primary_color TEXT,
			custom_sidebar_color TEXT,
			language TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error)
	return db
}

// newSettingsTestHandler wires a real SettingsService into the handler.
func newSettingsTestHandler(t *testing.T, db *gorm.DB) (*SettingsHandler, *gorm.DB) {
	t.Helper()
	svc := systemServices.NewSettingsService(db)
	h := NewSettingsHandler(svc)
	h.core = &core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{},
	}
	return h, db
}

// invokeAuthHandler is a variant of invokeHandler that injects user_id into context.
func invokeAuthHandler(t *testing.T, method, path string, body interface{}, params gin.Params,
	handler func(*gin.Context), userID string) *httpRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var req *http.Request
	if body != nil {
		var raw []byte
		if s, ok := body.(string); ok {
			raw = []byte(s)
		} else {
			raw, _ = json.Marshal(body)
		}
		req = httptest.NewRequest(method, path, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	c.Request = req
	if params != nil {
		c.Params = params
	}
	if userID != "" {
		c.Set("user_id", userID)
	}
	if handler != nil {
		handler(c)
	}
	return w
}

// httpRecorder is an alias for httptest.ResponseRecorder.
type httpRecorder = httptest.ResponseRecorder

// TC1: GetUserPreferences - returns defaults when no record
func TestSettingsHandler_GetUserPreferences_Defaults(t *testing.T) {
	h, _ := newSettingsTestHandler(t, setupSettingsTestDB(t))

	w := invokeAuthHandler(t, "GET", "/system/settings/preferences", nil, nil, h.GetUserPreferences, "user-1")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "light", resp.Data["theme"])
}

// TC2: GetUserPreferences - unauthorized when no user_id
func TestSettingsHandler_GetUserPreferences_Unauthorized(t *testing.T) {
	h, _ := newSettingsTestHandler(t, setupSettingsTestDB(t))

	w := invokeAuthHandler(t, "GET", "/system/settings/preferences", nil, nil, h.GetUserPreferences, "")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC3: GetUserPreferences - service error bubbles up
func TestSettingsHandler_GetUserPreferences_ServiceError(t *testing.T) {
	h, db := newSettingsTestHandler(t, setupSettingsTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_user_preference").Error)

	w := invokeAuthHandler(t, "GET", "/system/settings/preferences", nil, nil, h.GetUserPreferences, "user-1")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC4: UpdateUserPreferences - success
func TestSettingsHandler_UpdateUserPreferences_Success(t *testing.T) {
	h, db := newSettingsTestHandler(t, setupSettingsTestDB(t))

	body := systemServices.UserPreferences{
		Theme:        "dark",
		ThemeStyle:   "minimal",
		LayoutType:   "classic",
		PageSize:     20,
		Language:     "en-US",
	}
	w := invokeAuthHandler(t, "PUT", "/system/settings/preferences", body, nil, h.UpdateUserPreferences, "user-1")
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	// Verify persisted
	var theme string
	require.NoError(t, db.Raw("SELECT theme FROM sys_user_preference WHERE user_id = ?", "user-1").Scan(&theme).Error)
	assert.Equal(t, "dark", theme)
}

// TC5: UpdateUserPreferences - unauthorized
func TestSettingsHandler_UpdateUserPreferences_Unauthorized(t *testing.T) {
	h, _ := newSettingsTestHandler(t, setupSettingsTestDB(t))

	body := systemServices.UserPreferences{
		Theme:    "dark",
		PageSize: 20,
		Language: "en-US",
	}
	w := invokeAuthHandler(t, "PUT", "/system/settings/preferences", body, nil, h.UpdateUserPreferences, "")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC6: UpdateUserPreferences - bad JSON returns 400
func TestSettingsHandler_UpdateUserPreferences_BadJSON(t *testing.T) {
	h, _ := newSettingsTestHandler(t, setupSettingsTestDB(t))

	w := invokeAuthHandler(t, "PUT", "/system/settings/preferences", "{not-json", nil, h.UpdateUserPreferences, "user-1")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC7: UpdateUserPreferences - updates existing record
func TestSettingsHandler_UpdateUserPreferences_UpdateExisting(t *testing.T) {
	h, db := newSettingsTestHandler(t, setupSettingsTestDB(t))

	// Seed existing record
	require.NoError(t, db.Exec(`INSERT INTO sys_user_preference (id, user_id, theme, page_size, language, sidebar_width, sidebar_collapsed_width, sidebar_collapsed)
		VALUES (?, 'user-1', 'light', 10, 'zh-CN', 280, 64, 0)`, "id-1").Error)

	body := systemServices.UserPreferences{
		Theme:    "dark",
		PageSize: 50,
		Language: "en-US",
	}
	w := invokeAuthHandler(t, "PUT", "/system/settings/preferences", body, nil, h.UpdateUserPreferences, "user-1")
	assert.Equal(t, http.StatusOK, w.Code)

	var pageSize int
	require.NoError(t, db.Raw("SELECT page_size FROM sys_user_preference WHERE user_id = ?", "user-1").Scan(&pageSize).Error)
	assert.Equal(t, 50, pageSize)
}

// TC8: UpdateUserPreferences - service error
func TestSettingsHandler_UpdateUserPreferences_ServiceError(t *testing.T) {
	h, db := newSettingsTestHandler(t, setupSettingsTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_user_preference").Error)

	body := systemServices.UserPreferences{
		Theme:    "dark",
		PageSize: 20,
		Language: "en-US",
	}
	w := invokeAuthHandler(t, "PUT", "/system/settings/preferences", body, nil, h.UpdateUserPreferences, "user-1")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC9: WithCore - nil receiver returns nil
func TestSettingsHandler_WithCore_NilReceiver(t *testing.T) {
	var h *SettingsHandler
	result := h.WithCore(&core.Core{})
	assert.Nil(t, result)
}

// =====================================================================
// quick-261102-m94: HTTP 端到端回归（前端真实形态 / 越界值 400 而非 500）
// =====================================================================

// TestSettingsHandler_UpdateUserPreferences_OmittedOptionalFieldsSucceeds
// 前端真实形态：PUT 的 body 里**根本没有 themeStyle 这个 key**（Phase 65 移除多主题后
// 前端不再发送该字段）。
//
// 注意：不能用 systemServices.UserPreferences 结构体整体序列化来构造这个 body ——
// 它的 themeStyle 标签没有 omitempty，序列化出来是 "themeStyle": ""，那是「显式空串」
// 而非「字段不存在」。虽然绑定层对二者等价，但这条用例要严格模拟前端形态，所以用
// 原始 JSON 字符串。
func TestSettingsHandler_UpdateUserPreferences_OmittedOptionalFieldsSucceeds(t *testing.T) {
	h, db := newSettingsTestHandler(t, setupSettingsTestDB(t))

	require.NoError(t, db.Exec(`INSERT INTO sys_user_preference (id, user_id, theme, theme_style, layout_type, layout_density, sidebar_width, sidebar_collapsed_width, page_size, language)
		VALUES (?, 'user-1', 'light', 'glassmorphism', 'hybrid', 'compact', 320, 80, 20, 'zh-CN')`, "id-1").Error)

	// 字段严格照前端 toBackendFormat() 的实际形态：没有 themeStyle key
	// （themeStyle 是本次唯一「请求省略 + 数据库有 CHECK 约束」的字段）；
	// layoutType/layoutDensity 等前端始终发送的字段按显式值生效。
	body := `{"theme":"dark","layoutType":"classic","layoutDensity":"comfortable","sidebarWidth":280,"sidebarCollapsedWidth":64,"sidebarCollapsed":false,"pageSize":20,"language":"zh-CN"}`
	w := invokeAuthHandler(t, "PUT", "/system/settings/preferences", body, nil, h.UpdateUserPreferences, "user-1")
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var resp struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)

	var row struct {
		ThemeStyle string
		LayoutType string
	}
	require.NoError(t, db.Raw(`SELECT theme_style, layout_type FROM sys_user_preference WHERE user_id = ?`, "user-1").Scan(&row).Error)
	assert.Equal(t, "glassmorphism", row.ThemeStyle, "省略的 theme_style 应保留原值，不被写成空串")
	assert.NotEmpty(t, row.LayoutType, "layout_type 不应被清空")
	assert.Equal(t, "classic", row.LayoutType, "显式携带的 layoutType 应生效")
}

// TestSettingsHandler_UpdateUserPreferences_UnsupportedThemeStyleRejected
// 携带数据库 CHECK 约束从未允许过的主题风格值 → HTTP 400（参数错误），不是 500。
func TestSettingsHandler_UpdateUserPreferences_UnsupportedThemeStyleRejected(t *testing.T) {
	h, _ := newSettingsTestHandler(t, setupSettingsTestDB(t))

	body := `{"theme":"dark","themeStyle":"ink-amber","pageSize":20,"language":"zh-CN"}`
	w := invokeAuthHandler(t, "PUT", "/system/settings/preferences", body, nil, h.UpdateUserPreferences, "user-1")

	assert.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	assert.NotContains(t, w.Body.String(), "23514", "不应出现数据库约束冲突")
	assert.NotContains(t, w.Body.String(), "CHECK constraint", "不应出现数据库约束冲突")
}

// TestSettingsHandler_UpdateUserPreferences_UnsupportedPageSizeRejected
// pageSize=30 能过 min/max 校验但撞 chk_page_size → 绑定层已收紧为 oneof，HTTP 400。
func TestSettingsHandler_UpdateUserPreferences_UnsupportedPageSizeRejected(t *testing.T) {
	h, _ := newSettingsTestHandler(t, setupSettingsTestDB(t))

	body := `{"theme":"dark","pageSize":30,"language":"zh-CN"}`
	w := invokeAuthHandler(t, "PUT", "/system/settings/preferences", body, nil, h.UpdateUserPreferences, "user-1")

	assert.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	assert.NotContains(t, w.Body.String(), "23514", "不应出现数据库约束冲突")
	assert.NotContains(t, w.Body.String(), "CHECK constraint", "不应出现数据库约束冲突")
}