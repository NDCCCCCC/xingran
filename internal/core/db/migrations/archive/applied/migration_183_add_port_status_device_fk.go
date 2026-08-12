//go:build archive_skip


package migrations

import (
	"fmt"
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate183AddPortStatusDeviceFK 加 FK 约束 sys_device_port_status.device_id
// REFERENCES sys_network_device(id),防止未来 device_id 漂移到不存在的设备(orphan)。
//
// 必须先于 migration_183 执行 migration_182,否则 FK 校验会失败
// (因为 1247 行 port_status.device_id 仍指向"错但存在"的设备)
//
// 加 FK 用 NOT VALID 跳过存量校验 → VALIDATE CONSTRAINT 后台异步验证:
//   优势: 加 FK 不会锁表(毫秒级完成)
//   异步: VALIDATE 在后台跑,只对正在修改的行加 SHARE UPDATE EXCLUSIVE 锁
//
// FK 约束的局限:
//   - 只能防止"device_id 指向不存在的设备"(orphan)
//   - 无法防止"device_id 指向错误但存在的设备"(本次数据治理已修,但仍可能因新错误导入发生)
//   - 业务规则违反(port_status 指向错的物理设备)需靠 Phase 4 定时对账任务捕获
//
// ON DELETE RESTRICT: 不允许删除被 port_status 引用的 sys_network_device,
//   避免误删设备导致 port_status.dangling device_id
func Migrate183AddPortStatusDeviceFK(db *gorm.DB) error {
	log.Println("Running migration 183: 加 sys_device_port_status.device_id FK 约束")

	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] 183 跳过(非 PostgreSQL)")
		log.Println("Migration 183 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// 幂等性检查: 约束已存在则跳过
	var exists bool
	if err := db.Raw(`
SELECT EXISTS (
    SELECT 1 FROM pg_constraint
     WHERE conname = 'fk_port_status_device'
       AND conrelid = 'sys_device_port_status'::regclass
)`).Scan(&exists).Error; err != nil {
		return fmt.Errorf("检查 FK 是否已存在失败: %w", err)
	}
	if exists {
		log.Println("Migration 183: FK fk_port_status_device 已存在,跳过(幂等性保护)")
		return nil
	}

	// 1. 加 FK (NOT VALID 跳过存量校验)
	if err := db.Exec(`
ALTER TABLE sys_device_port_status
ADD CONSTRAINT fk_port_status_device
FOREIGN KEY (device_id) REFERENCES sys_network_device(id)
ON DELETE RESTRICT
NOT VALID`).Error; err != nil {
		return fmt.Errorf("加 FK 失败: %w", err)
	}
	applogger.Infof("[迁移] 183 FK fk_port_status_device 已加(NOT VALID,跳过存量校验)")

	// 2. 异步验证存量(不阻塞应用,只对正在修改的行加 SHARE UPDATE EXCLUSIVE 锁)
	if err := db.Exec(`
ALTER TABLE sys_device_port_status
VALIDATE CONSTRAINT fk_port_status_device`).Error; err != nil {
		// 验证失败非阻断(可能是被锁),提示人工重试
		applogger.Warnf("[迁移] 183 FK 异步验证失败,需手动重试 (ALTER TABLE sys_device_port_status VALIDATE CONSTRAINT fk_port_status_device): %v", err)
	} else {
		applogger.Infof("[迁移] 183 FK 异步验证完成")
	}

	log.Println("Migration 183 completed")
	return nil
}