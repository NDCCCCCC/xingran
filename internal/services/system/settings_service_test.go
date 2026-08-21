package system

// =====================================================================
// settings_service_test.go — covers settings_service.go (172 lines)
// Extends existing settings_service_test.go (PRESERVED - TestBuildDefaultPreferences_HardcodedDefaults)
//
// Per Plan 72-11 Task 4
// =====================================================================

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupSettingsServiceDB creates in-memory SQLite for settings service tests.
func setupSettingsServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user_preference (
			id TEXT PRIMARY KEY,
			user_id TEXT UNIQUE,
			theme TEXT,
			theme_style TEXT,
			layout_type TEXT,
			layout_density TEXT,
			sidebar_width INTEGER,
			sidebar_collapsed_width INTEGER,
			sidebar_collapsed INTEGER,
			page_size INTEGER,
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