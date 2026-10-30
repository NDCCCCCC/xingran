package migrations

import (
	"fmt"
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate206AddUserRoleRoleMenuIndexes 为登录菜单加载相关的关联表添加复合索引
//
// 背景:
//   - 登录后 GET /system/my-menus 在远程 Supabase PostgreSQL 上执行多个顺序查询:
//     sys_user_role 按 user_id 过滤、sys_role_menu 按 role_id 过滤、sys_menu 按 parent_id/status/visible 过滤。
//   - 这些表原无外键列索引,导致全表扫描;在跨地域/高延迟数据库上单个查询可达 3-7s,
//     累计超过前端 axios 30s 超时,触发登录失败。
//
// 解决:
//   - 为 sys_user_role(user_id, role_id) 建立复合索引,加速按用户查角色。
//   - 为 sys_role_menu(role_id, menu_id) 建立复合索引,加速按角色查菜单。
//   - 为 sys_menu(parent_id, status, visible) 建立复合索引,加速菜单树过滤。
//
// 幂等性: CREATE INDEX IF NOT EXISTS 本身幂等,重复执行不会报错。
// 对应模型层也加了 gorm index 标签,与 AutoMigrate 保持一致。
func Migrate206AddUserRoleRoleMenuIndexes(db *gorm.DB) error {
	log.Println("Running migration 206: add indexes for sys_user_role / sys_role_menu / sys_menu")

	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] 206 跳过(非 PostgreSQL)")
		log.Println("Migration 206 skipped (non-PostgreSQL dialect)")
		return nil
	}

	indexes := []struct {
		name  string
		table string
		cols  string
	}{
		{
			name:  "idx_sys_user_role_user_id_role_id",
			table: "sys_user_role",
			cols:  "user_id, role_id",
		},
		{
			name:  "idx_sys_role_menu_role_id_menu_id",
			table: "sys_role_menu",
			cols:  "role_id, menu_id",
		},
		{
			name:  "idx_sys_menu_parent_status_visible",
			table: "sys_menu",
			cols:  "parent_id, status, visible",
		},
	}

	for _, idx := range indexes {
		sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s)", idx.name, idx.table, idx.cols)
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("创建索引 %s ON %s(%s) 失败: %w", idx.name, idx.table, idx.cols, err)
		}
		applogger.Infof("[迁移] 206 索引 %s 已创建/已存在", idx.name)
		log.Printf("Migration 206: index %s created or already exists", idx.name)
	}

	log.Println("Migration 206 completed")
	return nil
}
