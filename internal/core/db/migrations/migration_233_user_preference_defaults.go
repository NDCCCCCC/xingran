package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// Migrate233UserPreferenceDefaults —— 把 sys_user_preference 的
// theme_style / layout_type / layout_density 三列中的空字符串与 NULL
// 归一化为规范默认值 minimal / classic / comfortable。
//
// 背景（quick-261102-m94）：
// 前端自 Phase 65 移除多主题后不再发送 themeStyle，后端 DTO 用普通 string 接收，
// omitempty 让省略字段留下空字符串，UpdateUserPreferences 的 map 更新把它当成
// 「显式置空」写库，撞上 PG 的 chk_theme_style（SQLSTATE 23514）。
// 应用层已按「省略即不修改」修复（见 internal/services/system/settings_service.go）。
//
// 结论：
//   - PG 上本迁移是 no-op —— CHECK 约束在写入时就拦住了空串，PG 库里不存在违规行。
//   - 真正被它修的是 SQLite 分支 —— GORM AutoMigrate 不生成 CHECK 约束，
//     历史上被该 bug 写入过空串的行会残留，导致 GET 返回越界值。
//   - **本迁移不放宽也不新增任何 CHECK 约束**（约束本身是正确的，错误在应用层）。
//   - 不给 SQLite 补 CHECK 约束 —— GORM 的 Migrator 不支持 CHECK 类型，
//     用原始 SQL 重建表风险过高，超出本次范围。
//
// 幂等：三条 UPDATE 都只把空值替换为常量默认值，重复执行结果一致。
// 表不存在时直接返回 nil（全新库由 AutoMigrate 建表，无需归一化）。
// 任一条 UPDATE 失败返回错误，由调用方按既有惯例记日志并非阻断。
func Migrate233UserPreferenceDefaults(db *gorm.DB) error {
	log.Println("Running migration 233: sys_user_preference 空值归一化")

	if !db.Migrator().HasTable(&models.UserPreference{}) {
		log.Println("Migration 233: sys_user_preference 表不存在，跳过")
		return nil
	}

	// 双方言通用语法（不依赖 PG 专属特性）；列名与取值均为常量，不含用户输入拼接。
	normalizations := []struct {
		column string
		value  string
	}{
		{column: "theme_style", value: "minimal"},
		{column: "layout_type", value: "classic"},
		{column: "layout_density", value: "comfortable"},
	}
	for _, n := range normalizations {
		sql := fmt.Sprintf(
			"UPDATE sys_user_preference SET %s = ? WHERE %s IS NULL OR %s = ''",
			n.column, n.column, n.column,
		)
		if err := db.Exec(sql, n.value).Error; err != nil {
			return fmt.Errorf("归一化 sys_user_preference.%s 失败: %w", n.column, err)
		}
	}

	log.Println("Migration 233: 用户偏好空值归一化完成")
	return nil
}
