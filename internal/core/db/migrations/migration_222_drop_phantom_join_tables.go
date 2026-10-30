package migrations

import (
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate222DropPhantomJoinTables 清理 4 张「幻影关联表」。
//
// 背景：
//
//	模型的 many2many 标签早期写错了关联表名，导致 GORM AutoMigrate 额外创建了
//	4 张永远为空的影子表，而真实关联表（由 dbprovision SQL 建表 + 对应 Join 模型
//	维护）一直在正常使用：
//
//	  - user_posts  → 真实表 sys_user_post  （models.UserPost）
//	  - user_roles  → 真实表 sys_user_role  （models.UserRole）
//	  - role_menus  → 真实表 sys_role_menu  （models.RoleMenu）
//	  - role_depts  → 真实表 sys_role_dept  （models.RoleDept）
//
//	全仓没有任何 Association(...) / Preload(...) 命中这 4 组关联，
//	业务代码一律通过显式 Join 模型（tx.Create(&models.UserPost{...})、
//	tx.Table("sys_user_post")）读写真实表，因此这 4 张影子表从创建起就是死表。
//
// 本迁移：
//   - 逐张检查存在性 → 存在则统计行数；
//   - 行数为 0 → DROP TABLE；
//   - 行数 > 0 → 只告警不删（保守：万一有人往里写过数据，留给人工确认）。
//
// 另外顺带清理 sys_dept_group_mapping：它是已下线的「部门-用户组映射」功能的
// 遗留表（models.DeptGroupMapping 已随 member_sync_service 一并删除，
// 全仓无任何 Go 引用、dbprovision SQL 也无建表语句），属于纯遗留表。
//
// 幂等：表不存在直接跳过；重复执行无副作用。
// 双方言：只用 GORM Migrator（HasTable/DropTable）+ Table(...).Count()，PG / sqlite 通用。
// 非阻断：单表 DROP 失败只告警，不阻断启动（见 database.go 注册处的档位说明）。
func Migrate222DropPhantomJoinTables(db *gorm.DB) error {
	log.Println("Running migration 222: drop phantom join tables")

	phantomTables := []string{
		"user_posts",           // → sys_user_post
		"user_roles",           // → sys_user_role
		"role_menus",           // → sys_role_menu
		"role_depts",           // → sys_role_dept
		"sys_dept_group_mapping", // 已下线「部门-用户组映射」功能的遗留表
	}

	dropped, skipped := 0, 0
	for _, table := range phantomTables {
		if !db.Migrator().HasTable(table) {
			skipped++
			continue
		}
		var count int64
		if err := db.Table(table).Count(&count).Error; err != nil {
			// 统计失败不阻断：可能是权限/方言差异，留给人工处理
			applogger.Warnf("[迁移] 统计幻影关联表 %s 行数失败,跳过删除: %v", table, err)
			skipped++
			continue
		}
		if count > 0 {
			applogger.Warnf("[迁移] 幻影关联表 %s 存在 %d 行数据,跳过删除(需人工确认)", table, count)
			skipped++
			continue
		}
		if err := db.Migrator().DropTable(table); err != nil {
			applogger.Warnf("[迁移] 删除幻影关联表 %s 失败(非阻断): %v", table, err)
			skipped++
			continue
		}
		applogger.Infof("[迁移] 已删除幻影关联表 %s (0 行)", table)
		dropped++
	}

	log.Printf("Migration 222 completed: dropped=%d skipped=%d", dropped, skipped)
	return nil
}
