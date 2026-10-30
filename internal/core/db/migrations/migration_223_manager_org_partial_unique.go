package migrations

import (
	"fmt"
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate223ManagerOrgPartialUnique 修复「删除机构后无法重建同部门机构」的问题。
//
// 背景：sys_manager_org.dept_id 上的唯一索引 idx_sys_manager_org_dept_id 是
// GORM AutoMigrate 按 uniqueIndex tag 建的普通唯一索引，不带 deleted_at 条件。
// 而 ManagerOrg 走 GORM 软删——页面上删除的机构行仍在表里继续占用 dept_id，
// 导致「从部门架构生成」/手动新增同部门机构时撞
// duplicate key ... idx_sys_manager_org_dept_id (SQLSTATE 23505)。
//
// 修复（三步，全部幂等，双方言通用——PostgreSQL 与 SQLite 均支持部分索引）：
//  1. 物理清理已软删的机构行（这些行是用户明确删除的配置数据，operlog 已留审计）；
//  2. DROP 旧普通唯一索引；
//  3. 重建同名索引为部分唯一索引 WHERE deleted_at IS NULL（软删行不再占坑）。
//
// 服务层已同步把 DeleteOrg 改为硬删（Unscoped），本索引作为双保险，
// 防御未来任何软删路径回归。
//
// 非阻断档：与 migration_219/220 同档，失败仅 WARN 不阻断启动。
func Migrate223ManagerOrgPartialUnique(db *gorm.DB) error {
	log.Println("Running migration 223: manager org dept_id partial unique index")

	if !db.Migrator().HasTable(&models.ManagerOrg{}) {
		log.Println("Warning: sys_manager_org 不存在，跳过 migration 223（先启动建表后再重启即可）")
		return nil
	}

	// 1) 物理清理软删行
	res := db.Exec("DELETE FROM sys_manager_org WHERE deleted_at IS NOT NULL")
	if res.Error != nil {
		log.Printf("Warning: 清理 sys_manager_org 软删行失败: %v", res.Error)
	} else if res.RowsAffected > 0 {
		log.Printf("[迁移] 已物理清理 %d 条已软删的机构行", res.RowsAffected)
	}

	// 2) 删旧普通唯一索引（幂等）
	if err := db.Exec("DROP INDEX IF EXISTS idx_sys_manager_org_dept_id").Error; err != nil {
		log.Printf("Warning: 删除旧索引 idx_sys_manager_org_dept_id 失败: %v", err)
	}

	// 3) 重建为部分唯一索引（同名，AutoMigrate 视为已存在不会改回普通索引）
	if err := db.Exec(
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_sys_manager_org_dept_id " +
			"ON sys_manager_org (dept_id) WHERE deleted_at IS NULL",
	).Error; err != nil {
		return fmt.Errorf("重建 sys_manager_org 部分唯一索引失败: %w", err)
	}

	log.Println("Migration 223 completed: dept_id unique now scoped to deleted_at IS NULL")
	return nil
}
