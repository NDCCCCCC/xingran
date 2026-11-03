package system

// =====================================================================
// settings_service_test.go — covers settings_service.go (172 lines)
// Extends existing settings_service_test.go (PRESERVED - TestBuildDefaultPreferences_HardcodedDefaults)
//
// Per Plan 72-11 Task 4
// =====================================================================

import (
	"context"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupSettingsServiceDB creates in-memory SQLite for settings service tests.
//
// 建表 SQL 显式带上与 PG 侧一致的 CHECK 约束（theme_style / layout_type /
// layout_density / page_size）。SQLite 原生支持 CHECK，只是 GORM AutoMigrate 不生成；
// 不写出来的话本 bug 在 SQLite 上表现为「静默写空串」而非报错，测不出根因。
//
// 不加 NOT NULL：既有 TC2/TC7 依赖「不指定该列时插入 NULL」；SQLite 的 CHECK
// 对 NULL 结果放行（NULL → UNKNOWN → 通过），不影响既有断言。
func setupSettingsServiceDB(t *testing.T) *gorm.DB {
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

// TC1: GetUserPreferences - returns defaults when no record
func TestSettingsService_GetUserPreferences_Defaults(t *testing.T) {
	db := setupSettingsServiceDB(t)
	svc := NewSettingsService(db)

	prefs, err := svc.GetUserPreferences(context.Background(), "user-1")
	require.NoError(t, err)
	require.NotNil(t, prefs)
	assert.Equal(t, "light", prefs.Theme)
	assert.Equal(t, "minimal", prefs.ThemeStyle)
	assert.Equal(t, 280, prefs.SidebarWidth)
	assert.Equal(t, 64, prefs.SidebarCollapsedWidth)
	assert.Equal(t, 10, prefs.PageSize)
	assert.Equal(t, "zh-CN", prefs.Language)
}

// TC2: GetUserPreferences - returns existing prefs
func TestSettingsService_GetUserPreferences_Existing(t *testing.T) {
	db := setupSettingsServiceDB(t)
	svc := NewSettingsService(db)

	require.NoError(t, db.Exec(`INSERT INTO sys_user_preference
		(id, user_id, theme, theme_style, sidebar_width, sidebar_collapsed_width, page_size, language)
		VALUES (?, 'user-1', 'dark', 'glassmorphism', 320, 80, 20, 'en-US')`, "id-1").Error)

	prefs, err := svc.GetUserPreferences(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, "dark", prefs.Theme)
	assert.Equal(t, 20, prefs.PageSize)
	assert.Equal(t, "en-US", prefs.Language)
}

// TC3: GetUserPreferences - service error
func TestSettingsService_GetUserPreferences_Error(t *testing.T) {
	db := setupSettingsServiceDB(t)
	svc := NewSettingsService(db)
	require.NoError(t, db.Exec("DROP TABLE sys_user_preference").Error)

	_, err := svc.GetUserPreferences(context.Background(), "user-1")
	assert.Error(t, err)
}

// TC4: UpdateUserPreferences - creates new record
func TestSettingsService_UpdateUserPreferences_Create(t *testing.T) {
	db := setupSettingsServiceDB(t)
	svc := NewSettingsService(db)

	req := &UserPreferences{
		Theme:        "dark",
		ThemeStyle:   "minimal",
		PageSize:     20,
		Language:     "en-US",
	}
	require.NoError(t, svc.UpdateUserPreferences(context.Background(), "user-1", req))

	var theme string
	require.NoError(t, db.Raw("SELECT theme FROM sys_user_preference WHERE user_id = ?", "user-1").Scan(&theme).Error)
	assert.Equal(t, "dark", theme)
}

// TC5: UpdateUserPreferences - updates existing record
func TestSettingsService_UpdateUserPreferences_Update(t *testing.T) {
	db := setupSettingsServiceDB(t)
	svc := NewSettingsService(db)

	require.NoError(t, db.Exec(`INSERT INTO sys_user_preference
		(id, user_id, theme, page_size, language)
		VALUES (?, 'user-1', 'light', 10, 'zh-CN')`, "id-1").Error)

	req := &UserPreferences{
		Theme:    "dark",
		PageSize: 50,
		Language: "en-US",
	}
	require.NoError(t, svc.UpdateUserPreferences(context.Background(), "user-1", req))

	var pageSize int
	require.NoError(t, db.Raw("SELECT page_size FROM sys_user_preference WHERE user_id = ?", "user-1").Scan(&pageSize).Error)
	assert.Equal(t, 50, pageSize)
}

// TC6: buildDefaultPreferences - returns hardcoded defaults
func TestSettingsService_BuildDefaultPreferences(t *testing.T) {
	svc := &settingsService{}
	prefs := svc.buildDefaultPreferences()

	assert.Equal(t, "light", prefs.Theme)
	assert.Equal(t, "minimal", prefs.ThemeStyle)
	assert.Equal(t, "classic", prefs.LayoutType)
	assert.Equal(t, "comfortable", prefs.LayoutDensity)
	assert.Equal(t, 280, prefs.SidebarWidth)
	assert.Equal(t, 64, prefs.SidebarCollapsedWidth)
	assert.False(t, prefs.SidebarCollapsed)
	assert.Equal(t, 10, prefs.PageSize)
	assert.Equal(t, "zh-CN", prefs.Language)
}

// TC7: GetUserPreferences - default sidebar width when zero
func TestSettingsService_GetUserPreferences_ZeroDefaultsApplied(t *testing.T) {
	db := setupSettingsServiceDB(t)
	svc := NewSettingsService(db)

	// Seed with zero sidebar widths
	require.NoError(t, db.Exec(`INSERT INTO sys_user_preference
		(id, user_id, theme, sidebar_width, sidebar_collapsed_width, page_size, language)
		VALUES (?, 'user-1', 'dark', 0, 0, 10, 'zh-CN')`, "id-1").Error)

	prefs, err := svc.GetUserPreferences(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, 280, prefs.SidebarWidth, "zero should fallback to default 280")
	assert.Equal(t, 64, prefs.SidebarCollapsedWidth, "zero should fallback to default 64")
}

// TestBuildDefaultPreferences_HardcodedDefaults — PRESERVED from prior test file (D-08)
// Phase 72 Wave 3b Plan 72-11 extends existing test file. The original regression test
// is kept verbatim to guard against silent drift in default values.
func TestBuildDefaultPreferences_HardcodedDefaults(t *testing.T) {
	svc := &settingsService{}
	prefs := svc.buildDefaultPreferences()

	assert.Equal(t, "light", prefs.Theme)
	assert.Equal(t, "minimal", prefs.ThemeStyle)
	assert.Equal(t, "classic", prefs.LayoutType)
	assert.Equal(t, "comfortable", prefs.LayoutDensity)
	assert.Equal(t, 280, prefs.SidebarWidth)
	assert.Equal(t, 64, prefs.SidebarCollapsedWidth)
	assert.Equal(t, false, prefs.SidebarCollapsed)
	assert.Equal(t, 10, prefs.PageSize)
	assert.Equal(t, "zh-CN", prefs.Language)
}

// =====================================================================
// quick-261102-m94: 「省略即不修改」+ 允许集合自检回归
// =====================================================================

// TestSettingsService_UpdateUserPreferences_OmittedOptionalFieldsPreserved
// 已有行 + 请求省略 themeStyle/layoutType/layoutDensity/sidebarWidth/
// sidebarCollapsedWidth → DB 中这几列保持原值，不被写成空字符串。
// 这条用例在修复前会因 CHECK 约束报错而失败 —— 它就是本次故障的最小复现。
func TestSettingsService_UpdateUserPreferences_OmittedOptionalFieldsPreserved(t *testing.T) {
	db := setupSettingsServiceDB(t)
	svc := NewSettingsService(db)

	require.NoError(t, db.Exec(`INSERT INTO sys_user_preference
		(id, user_id, theme, theme_style, layout_type, layout_density, sidebar_width, sidebar_collapsed_width, page_size, language)
		VALUES (?, 'user-1', 'light', 'glassmorphism', 'hybrid', 'compact', 320, 80, 20, 'zh-CN')`, "id-1").Error)

	// 前端形态：只携带 theme / pageSize / language，其余可选字段留空（= 本次未携带）
	req := &UserPreferences{
		Theme:    "dark",
		PageSize: 20,
		Language: "zh-CN",
	}
	require.NoError(t, svc.UpdateUserPreferences(context.Background(), "user-1", req))

	var row struct {
		Theme         string
		ThemeStyle    string
		LayoutType    string
		LayoutDensity string
		SidebarWidth  int
	}
	require.NoError(t, db.Raw(`SELECT theme, theme_style, layout_type, layout_density, sidebar_width
		FROM sys_user_preference WHERE user_id = ?`, "user-1").Scan(&row).Error)

	assert.Equal(t, "dark", row.Theme, "显式携带的字段应被写入")
	assert.Equal(t, "glassmorphism", row.ThemeStyle, "省略的 theme_style 应保留原值")
	assert.Equal(t, "hybrid", row.LayoutType, "省略的 layout_type 应保留原值")
	assert.Equal(t, "compact", row.LayoutDensity, "省略的 layout_density 应保留原值")
	assert.Equal(t, 320, row.SidebarWidth, "省略的 sidebar_width 应保留原值")
}

// TestSettingsService_UpdateUserPreferences_CreateAppliesCanonicalDefaults
// 无历史行 + 请求省略全部可选字段 → 落库规范默认值，不出现空字符串。
func TestSettingsService_UpdateUserPreferences_CreateAppliesCanonicalDefaults(t *testing.T) {
	db := setupSettingsServiceDB(t)
	svc := NewSettingsService(db)

	req := &UserPreferences{
		Theme:    "dark",
		PageSize: 20,
		Language: "zh-CN",
	}
	require.NoError(t, svc.UpdateUserPreferences(context.Background(), "user-1", req))

	var row struct {
		ThemeStyle            string
		LayoutType            string
		LayoutDensity         string
		SidebarWidth          int
		SidebarCollapsedWidth int
	}
	require.NoError(t, db.Raw(`SELECT theme_style, layout_type, layout_density, sidebar_width, sidebar_collapsed_width
		FROM sys_user_preference WHERE user_id = ?`, "user-1").Scan(&row).Error)

	assert.Equal(t, "minimal", row.ThemeStyle)
	assert.Equal(t, "classic", row.LayoutType)
	assert.Equal(t, "comfortable", row.LayoutDensity)
	assert.Equal(t, 280, row.SidebarWidth)
	assert.Equal(t, 64, row.SidebarCollapsedWidth)
}

// TestSettingsService_UpdateUserPreferences_ExplicitValuesWin
// 显式携带的合法值仍然生效 —— 证明语义是「只有省略才保留」，不是「永不更新」。
func TestSettingsService_UpdateUserPreferences_ExplicitValuesWin(t *testing.T) {
	db := setupSettingsServiceDB(t)
	svc := NewSettingsService(db)

	require.NoError(t, db.Exec(`INSERT INTO sys_user_preference
		(id, user_id, theme, theme_style, page_size, language)
		VALUES (?, 'user-1', 'light', 'glassmorphism', 20, 'zh-CN')`, "id-1").Error)

	req := &UserPreferences{
		Theme:         "dark",
		ThemeStyle:    "neumorphism",
		LayoutType:    "innovative",
		LayoutDensity: "spacious",
		SidebarWidth:  300,
		PageSize:      20,
		Language:      "zh-CN",
	}
	require.NoError(t, svc.UpdateUserPreferences(context.Background(), "user-1", req))

	var row struct {
		ThemeStyle    string
		LayoutType    string
		LayoutDensity string
		SidebarWidth  int
	}
	require.NoError(t, db.Raw(`SELECT theme_style, layout_type, layout_density, sidebar_width
		FROM sys_user_preference WHERE user_id = ?`, "user-1").Scan(&row).Error)

	assert.Equal(t, "neumorphism", row.ThemeStyle)
	assert.Equal(t, "innovative", row.LayoutType)
	assert.Equal(t, "spacious", row.LayoutDensity)
	assert.Equal(t, 300, row.SidebarWidth)
}

// TestSettingsService_UpdateUserPreferences_OutOfDomainValueNeverPersisted
// 绕过 HTTP 绑定直接调服务，请求带不在允许集合内的主题风格值 →
// 服务层自检拦下，落库的是规范默认值，服务层绝不写越界值。
func TestSettingsService_UpdateUserPreferences_OutOfDomainValueNeverPersisted(t *testing.T) {
	db := setupSettingsServiceDB(t)
	svc := NewSettingsService(db)

	req := &UserPreferences{
		Theme:      "dark",
		ThemeStyle: "not-a-valid-theme-style",
		PageSize:   20,
		Language:   "zh-CN",
	}
	require.NoError(t, svc.UpdateUserPreferences(context.Background(), "user-1", req))

	var themeStyle string
	require.NoError(t, db.Raw("SELECT theme_style FROM sys_user_preference WHERE user_id = ?", "user-1").Scan(&themeStyle).Error)
	assert.Equal(t, "minimal", themeStyle, "越界值应被替换为规范默认值")
}

// parseBindingOneOf 解析结构体 binding 标签里的 oneof 列表。
func parseBindingOneOf(t *testing.T, f reflect.StructField) []string {
	t.Helper()
	tag := f.Tag.Get("binding")
	for _, part := range strings.Split(tag, ",") {
		if strings.HasPrefix(part, "oneof=") {
			return strings.Fields(strings.TrimPrefix(part, "oneof="))
		}
	}
	t.Fatalf("字段 %s 的 binding 标签缺少 oneof 列表: %q", f.Name, tag)
	return nil
}

// TestSettingsService_BindingAllowListMatchesAllowedSets
// 锁定「绑定标签的 oneof 列表 == 服务层允许集合」不再漂移 —— 绑定层与数据库
// CHECK 约束对齐是本次修复的第二道防线，漂移即意味着可能出现必然 500 的请求。
func TestSettingsService_BindingAllowListMatchesAllowedSets(t *testing.T) {
	tp := reflect.TypeOf(UserPreferences{})

	stringCases := []struct {
		field   string
		allowed []string
	}{
		{"ThemeStyle", allowedThemeStyles},
		{"LayoutType", allowedLayoutTypes},
		{"LayoutDensity", allowedLayoutDensities},
	}
	for _, tc := range stringCases {
		f, ok := tp.FieldByName(tc.field)
		require.True(t, ok, "字段 %s 不存在", tc.field)
		assert.Equal(t, tc.allowed, parseBindingOneOf(t, f),
			"字段 %s 的绑定 oneof 列表与服务层允许集合不一致", tc.field)
	}

	f, ok := tp.FieldByName("PageSize")
	require.True(t, ok, "字段 PageSize 不存在")
	wantPageSizes := make([]string, 0, len(allowedPageSizes))
	for _, v := range allowedPageSizes {
		wantPageSizes = append(wantPageSizes, strconv.Itoa(v))
	}
	assert.Equal(t, wantPageSizes, parseBindingOneOf(t, f),
		"字段 PageSize 的绑定 oneof 列表与服务层允许集合不一致")
}