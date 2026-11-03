package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// TestMigrate233UserPreferenceDefaults_NormalizesEmptyValues
// 空字符串与 NULL 都被归一化为规范默认值（sqlite 方言；PG 上因 CHECK 约束
// 拦截写入而本就无违规行，该迁移为 no-op）。
func TestMigrate233UserPreferenceDefaults_NormalizesEmptyValues(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.UserPreference{}))

	require.NoError(t, db.Exec(`INSERT INTO sys_user_preference
		(id, user_id, theme, theme_style, layout_type, layout_density, page_size, language)
		VALUES ('id-1', 'user-1', 'dark', '', NULL, '', 20, 'zh-CN')`).Error)

	require.NoError(t, Migrate233UserPreferenceDefaults(db))

	var row struct {
		ThemeStyle    string
		LayoutType    string
		LayoutDensity string
	}
	require.NoError(t, db.Raw(`SELECT theme_style, layout_type, layout_density
		FROM sys_user_preference WHERE user_id = 'user-1'`).Scan(&row).Error)

	assert.Equal(t, "minimal", row.ThemeStyle)
	assert.Equal(t, "classic", row.LayoutType)
	assert.Equal(t, "comfortable", row.LayoutDensity)
}

// TestMigrate233UserPreferenceDefaults_Idempotent 连跑两次不报错，且第二次不改变已归一化的值。
func TestMigrate233UserPreferenceDefaults_Idempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.UserPreference{}))

	require.NoError(t, db.Exec(`INSERT INTO sys_user_preference
		(id, user_id, theme, theme_style, layout_type, layout_density, page_size, language)
		VALUES ('id-1', 'user-1', 'dark', '', NULL, '', 20, 'zh-CN')`).Error)

	require.NoError(t, Migrate233UserPreferenceDefaults(db))

	var first struct {
		ThemeStyle    string
		LayoutType    string
		LayoutDensity string
	}
	require.NoError(t, db.Raw(`SELECT theme_style, layout_type, layout_density
		FROM sys_user_preference WHERE user_id = 'user-1'`).Scan(&first).Error)

	// 二次执行：幂等，不报错、值不变
	require.NoError(t, Migrate233UserPreferenceDefaults(db))

	var second struct {
		ThemeStyle    string
		LayoutType    string
		LayoutDensity string
	}
	require.NoError(t, db.Raw(`SELECT theme_style, layout_type, layout_density
		FROM sys_user_preference WHERE user_id = 'user-1'`).Scan(&second).Error)

	assert.Equal(t, first, second, "重复执行不得改变已归一化的值")
	assert.Equal(t, "minimal", second.ThemeStyle)
	assert.Equal(t, "classic", second.LayoutType)
	assert.Equal(t, "comfortable", second.LayoutDensity)
}

// TestMigrate233UserPreferenceDefaults_NoTableIsNoop 表不存在时返回 nil（全新库由 AutoMigrate 建表）。
func TestMigrate233UserPreferenceDefaults_NoTableIsNoop(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, Migrate233UserPreferenceDefaults(db))
	assert.False(t, db.Migrator().HasTable("sys_user_preference"))
}
