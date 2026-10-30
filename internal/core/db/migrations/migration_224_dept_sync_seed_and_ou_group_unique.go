package migrations

import (
	"fmt"
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate224DeptSyncSeedAndOUGroupUnique 域控审查 P1-1 + P1-7：
//
//  1. seed「部门到AD同步」sys_job 行（P1-1：dept_to_ad_sync 此前无 migration seed，
//     生产现有的行是用户手工建的；新部署将缺失该任务导致部门→AD OU 无自动驱动）。
//     幂等：按 invoke_target + deleted_at IS NULL 判重（migration_137 seed 范式）。
//     status=0（正常）+ concurrent=false —— 执行器已按 Concurrent 挂 SkipIfStillRunning
//     （域控审查 P0-3 修复），02:00 的低频任务天然不会重叠。
//  2. sys_ou_group_mapping 唯一键扩展（P1-7）：原 uni_ou_group_mapping_ou 只含
//     (ou_dn, ad_group_id)，未含 ad_config_id —— 多域部署下两个域的同 OU DN 无法
//     各配映射。重建为 (ad_config_id, ou_dn, ad_group_id)。模型 tag 已同步
//     （internal/models/ou_group_mapping.go），AutoMigrate 视为已存在不会改回。
//
// 非阻断档：与 219/220/222/223 同档，失败仅 WARN 不阻断启动。
func Migrate224DeptSyncSeedAndOUGroupUnique(db *gorm.DB) error {
	log.Println("Running migration 224: dept_to_ad_sync job seed + sys_ou_group_mapping unique key extension")

	// ---------- 1) seed dept_to_ad_sync 定时任务 ----------
	if db.Migrator().HasTable(&models.Job{}) {
		var jobCount int64
		if err := db.Model(&models.Job{}).
			Where("invoke_target = ? AND deleted_at IS NULL", "dept_to_ad_sync").
			Count(&jobCount).Error; err != nil {
			log.Printf("Warning: 查询 sys_job 失败，跳过任务 seed: %v", err)
		} else if jobCount == 0 {
			remark := "每天凌晨2点同步系统部门结构到AD OU（域控审查 P1-1：原无seed，此行保证新部署具备自动驱动）"
			job := &models.Job{
				JobName:        "部门到AD同步",
				JobGroup:       "DEFAULT",
				InvokeTarget:   "dept_to_ad_sync",
				CronExpression: "0 0 2 * * ?",
				Concurrent:     false,
				Status:         models.JobStatusNormal,
				Remark:         &remark,
			}
			if err := db.Create(job).Error; err != nil {
				log.Printf("Warning: seed dept_to_ad_sync 失败: %v", err)
			} else {
				log.Println("[迁移] 已 seed 定时任务: 部门到AD同步 (dept_to_ad_sync, 每天 02:00)")
			}
		}
	} else {
		log.Println("Warning: sys_job 不存在，跳过任务 seed")
	}

	// ---------- 2) sys_ou_group_mapping 唯一键扩展 ----------
	if !db.Migrator().HasTable(&models.OUGroupMapping{}) {
		log.Println("Warning: sys_ou_group_mapping 不存在，跳过索引重建（先启动建表后再重启即可）")
		return nil
	}
	// 删旧索引（幂等；AutoMigrate 按 tag 生成的名字）
	if err := db.Exec("DROP INDEX IF EXISTS uni_ou_group_mapping_ou").Error; err != nil {
		log.Printf("Warning: 删除旧索引 uni_ou_group_mapping_ou 失败: %v", err)
	}
	// 重建同名索引、新列集（PostgreSQL 与 SQLite 通用语法）
	if err := db.Exec(
		"CREATE UNIQUE INDEX IF NOT EXISTS uni_ou_group_mapping_ou " +
			"ON sys_ou_group_mapping (ad_config_id, ou_dn, ad_group_id)",
	).Error; err != nil {
		return fmt.Errorf("重建 sys_ou_group_mapping 唯一索引失败: %w", err)
	}

	log.Println("Migration 224 completed: dept_to_ad_sync seeded (if missing), unique key now includes ad_config_id")
	return nil
}
