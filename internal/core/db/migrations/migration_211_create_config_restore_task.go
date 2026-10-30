package migrations

import (
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate211CreateConfigRestoreTask 创建 sys_config_restore_task 配置恢复任务表
// (Phase 93 D-15: 异步恢复任务的生命周期载体,与备份版本链职责分离)。
//
// 双方言: 仅 PG 分支执行 (CREATE TABLE IF NOT EXISTS,幂等);sqlite 分支由
// MigrateModelList 的 AutoMigrate 建表,调用方无需在 sqlite 分支注册本迁移。
// 索引: device_id (同设备互斥查询 D-08) 与 status (状态机流转查询)。
func Migrate211CreateConfigRestoreTask(db *gorm.DB) error {
	log.Println("Running migration 211: create sys_config_restore_task")

	const ddl = `CREATE TABLE IF NOT EXISTS sys_config_restore_task (
		id uuid PRIMARY KEY,
		device_id uuid NOT NULL,
		backup_id uuid NOT NULL,
		status varchar(20) NOT NULL,
		total_lines integer,
		sent_lines integer,
		failed_line text,
		result_json text,
		error_message text,
		started_at timestamptz,
		completed_at timestamptz,
		created_at timestamptz,
		updated_at timestamptz,
		created_by varchar(64)
	)`
	if err := db.Exec(ddl).Error; err != nil {
		applogger.Errorf("migration 211: create sys_config_restore_task 失败: %v", err)
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_sys_config_restore_task_device_id ON sys_config_restore_task (device_id)").Error; err != nil {
		applogger.Errorf("migration 211: create device_id index 失败: %v", err)
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_sys_config_restore_task_status ON sys_config_restore_task (status)").Error; err != nil {
		applogger.Errorf("migration 211: create status index 失败: %v", err)
		return err
	}
	return nil
}
