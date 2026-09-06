package migrations

import (
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate212CreateRestoreTaskActiveUniqueIndex 为 sys_config_restore_task 建立
// 同设备活跃任务部分唯一索引 (v129-recheck C-1: StartRestore 互斥 D-08 原子化)。
//
// 先查后写的应用层互斥存在并发窗口，两个并发 StartRestore 可各自建任务。
// 唯一索引 (device_id) WHERE status IN ('pending','running') 使插入即夺锁，
// 服务层捕获唯一冲突返回友好错误；应用层预查保留作快速路径。
//
// 双方言: PG 与 sqlite (3.8+) 均支持 partial unique index，语法一致，双方言
// 分支均注册 (幂等 IF NOT EXISTS)。
//
// 脏数据兼容: 若历史数据已存在同设备双活跃任务 (正是本索引要消灭的竞态产物)，
// 建索引失败按项目惯例非阻断 (留待人工清理后下次启动重试)。
func Migrate212CreateRestoreTaskActiveUniqueIndex(db *gorm.DB) error {
	log.Println("Running migration 212: create sys_config_restore_task active-unique partial index")

	const ddl = `CREATE UNIQUE INDEX IF NOT EXISTS uq_sys_config_restore_task_device_active
ON sys_config_restore_task (device_id)
WHERE status IN ('pending', 'running')`
	if err := db.Exec(ddl).Error; err != nil {
		applogger.Errorf("migration 212: create active-unique partial index 失败: %v", err)
		return err
	}
	return nil
}
