package migrations

import (
	"fmt"
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate219UserADSyncFields 补齐「AD 域控用户同步修复」所需的用户表与影子表列。
//
// 背景（问题清单）：
//   - sys_user 缺 mobile / title 列 → AD 同步过来的手机号/职位被直接丢弃；
//   - sys_user 缺 ad_config_id 列 → 多 AD 域场景下无法按域隔离用户，同名用户串味；
//   - sys_ad_user 影子表缺 manager 列 → 域控里实际的 manager 取值（DN 格式）无法回显。
//
// 双方言幂等实现要点：
//   - PostgreSQL 支持 ALTER TABLE ... ADD COLUMN IF NOT EXISTS，但 SQLite 不支持该语法；
//     故统一走 GORM Migrator 的 HasColumn + AddColumn 组合（先例 migration_140/141），
//     两个方言都由驱动生成对应 DDL：列已存在 → 跳过；不存在 → ALTER TABLE ADD COLUMN。
//     重复执行不报错。
//   - 目标表不存在（如全新库尚未跑 AutoMigrate）时降级为 Warning 跳过，不阻断启动；
//     启动期 AutoMigrate 兜底（models.User / models.ADUser 均已注册进 MigrateModelList）。
//
// 非阻断档：列补建失败只让 AD 同步丢弃手机号/职位、多域隔离退化为按用户名匹配，
// 不影响登录 / 入队 / 排水主链路（与 migration_216/217/218 同档）。
func Migrate219UserADSyncFields(db *gorm.DB) error {
	log.Println("Running migration 219: user/ad_user AD sync columns")

	// 1) sys_user 补 mobile / title / ad_config_id
	if err := migrate219AddColumnsIfMissing(db, &models.User{}, "sys_user",
		[]string{"Mobile", "Title", "AdConfigID"}); err != nil {
		return err
	}

	// 2) sys_ad_user 补 manager（域控 manager 属性回显）
	if err := migrate219AddColumnsIfMissing(db, &models.ADUser{}, "sys_ad_user",
		[]string{"Manager"}); err != nil {
		return err
	}

	// 3) ad_config_id 索引：AutoMigrate 路径由 model tag 自动建；
	//    SKIP_AUTOMIGRATE 旁路不跑 AutoMigrate，需在此显式补建（HasIndex 判存，幂等）。
	if err := migrate219EnsureFieldIndexIfMissing(db, &models.User{}, "sys_user", "AdConfigID"); err != nil {
		return err
	}

	log.Println("Migration 219 completed: user/ad_user AD sync columns ready")
	return nil
}

// migrate219AddColumnsIfMissing 给指定模型补列（幂等，PostgreSQL / SQLite 通用）。
//
// 实现要点：不用 ALTER TABLE ... ADD COLUMN IF NOT EXISTS —— 该语法 SQLite 不支持；
// 改用 GORM Migrator 的 HasColumn + AddColumn，列已存在则跳过。
// 表不存在时降级为 Warning 跳过（不阻断启动），由 AutoMigrate 兜底。
func migrate219AddColumnsIfMissing(db *gorm.DB, model interface{}, table string, cols []string) error {
	if !db.Migrator().HasTable(model) {
		log.Printf("Warning: 表 %s 不存在, migration_219 跳过补列 %v (由 AutoMigrate 兜底)", table, cols)
		return nil
	}
	for _, col := range cols {
		if db.Migrator().HasColumn(model, col) {
			continue
		}
		if err := db.Migrator().AddColumn(model, col); err != nil {
			return fmt.Errorf("表 %s 补列 %s 失败: %w", table, col, err)
		}
		log.Printf("Added column %s.%s", table, col)
	}
	return nil
}

// migrate219EnsureFieldIndexIfMissing 幂等补建模型字段上声明的索引。
//
// 必要性：GORM v1.30 的 Migrator.AddColumn 只发 ALTER TABLE ADD COLUMN，不建索引；
// AutoMigrate 路径会按 model tag 建索引，但 SKIP_AUTOMIGRATE 旁路不跑 AutoMigrate，
// 需在此显式补建（CreateIndex 本身不判存，故先 HasIndex 避免重复建报错）。
// 表不存在时直接跳过（建索引会因缺表报错，且该场景由 AutoMigrate 兜底）。
func migrate219EnsureFieldIndexIfMissing(db *gorm.DB, model interface{}, table string, field string) error {
	if !db.Migrator().HasTable(model) {
		return nil
	}
	if db.Migrator().HasIndex(model, field) {
		return nil
	}
	if err := db.Migrator().CreateIndex(model, field); err != nil {
		return fmt.Errorf("表 %s 补建索引 %s 失败: %w", table, field, err)
	}
	log.Printf("Created index for field %s.%s", table, field)
	return nil
}
