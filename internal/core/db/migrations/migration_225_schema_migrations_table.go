package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// Migrate225SchemaMigrationsTable 建立迁移记录表 sys_schema_migrations 并批量 seed
// 当前全部活跃迁移版本（Phase 84 PMODEL-04）。
//
// 背景：本仓迁移机制是「幂等重跑」而非「应用一次」——每次启动无条件重跑全部
// MigrateNNN，此前全仓没有任何已应用版本的持久记录（「已应用」的唯一凭证是启动日志）。
// 本迁移只做「记录 + 对账」的地基：建表 + seed，**不参与任何迁移是否执行的判断**
// （无 skip 语义，现有幂等重跑行为零改变）。
//
// 表结构：sys_schema_migrations(version int PRIMARY KEY, name text NOT NULL,
// applied_at timestamptz DEFAULT now())。
//
// seed 妥协说明：历史版本（175-224）的真实应用时间无法考证（编号迁移此前不留记录），
// 统一以 applied_at = now() 批量 seed，仅表示「该版本在本库视为已应用」。
// 另注：ActiveMigrations 含少量 PG-only 版本（175/176/202-206/211/214），在 sqlite 库
// 上它们也一并 seed —— 对账时若在 sqlite 库报这些版本 missing/extra，属方言差异的预期
// 信息，不是漂移。
//
// 幂等：CREATE TABLE IF NOT EXISTS + 逐版本 INSERT 冲突忽略（ON CONFLICT DO NOTHING /
// INSERT OR IGNORE 按方言守卫），可重跑。
func Migrate225SchemaMigrationsTable(db *gorm.DB) error {
	log.Println("Running migration 225: create sys_schema_migrations table + seed active versions")

	createSQL := `CREATE TABLE IF NOT EXISTS sys_schema_migrations (
		version int PRIMARY KEY,
		name text NOT NULL,
		applied_at timestamptz DEFAULT now()
	)`
	if !isPostgreSQL(db) {
		// sqlite 无 now() 函数，时间默认值降级为 CURRENT_TIMESTAMP；
		// timestamptz 类型名 sqlite 按亲和性接受，无需改名。
		createSQL = `CREATE TABLE IF NOT EXISTS sys_schema_migrations (
			version int PRIMARY KEY,
			name text NOT NULL,
			applied_at timestamptz DEFAULT CURRENT_TIMESTAMP
		)`
	}
	if err := db.Exec(createSQL).Error; err != nil {
		return fmt.Errorf("创建 sys_schema_migrations 表失败: %w", err)
	}

	// 批量 seed 活跃版本（除 225 自身 —— 225 由调用块的 runAndRecord 在成功后记录）。
	// 单事务包裹：seed 属纯记录行为，事务内失败整体 WARN 由调用块按非阻断档处理。
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("开启 seed 事务失败: %w", tx.Error)
	}
	seeded := 0
	for _, m := range ActiveMigrations {
		if m.Version == 225 {
			continue
		}
		var sql string
		if isPostgreSQL(db) {
			sql = `INSERT INTO sys_schema_migrations (version, name, applied_at)
				VALUES (?, ?, now()) ON CONFLICT (version) DO NOTHING`
		} else {
			sql = `INSERT OR IGNORE INTO sys_schema_migrations (version, name, applied_at)
				VALUES (?, ?, CURRENT_TIMESTAMP)`
		}
		if err := tx.Exec(sql, m.Version, m.Name).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("seed 版本 %d (%s) 失败: %w", m.Version, m.Name, err)
		}
		seeded++
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交 seed 事务失败: %w", err)
	}

	log.Printf("Migration 225 completed: sys_schema_migrations ready, %d historical versions seeded", seeded)
	return nil
}
