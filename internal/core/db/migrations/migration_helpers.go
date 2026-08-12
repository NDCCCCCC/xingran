package migrations

import "gorm.io/gorm"

// isPostgreSQL 判断当前数据库方言是否为 PostgreSQL。
// 历史:原定义于 migration_mac_history.go(已删除);多个 migration 用作 SQLite 路径守卫。
func isPostgreSQL(db *gorm.DB) bool {
	return db.Config.Dialector.Name() == "postgres"
}