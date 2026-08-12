package migrations

import (
	"fmt"
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate204AddDot1xUserLimit 加列 sys_device_port_status.dot1x_user_limit
//
// 背景:
//   - 锐捷 RGOS 在同一 interface view 下 dot1x port-control auto 与
//     dot1x default-user-limit N 是两条独立但耦合的命令。
//   - 设备行为不对称:
//     * no dot1x port-control auto 自动清除 default-user-limit (disable 路径无影响)
//     * dot1x port-control auto 不自动恢复 default-user-limit (enable 必须显式恢复 N)
//   - 旧 renderRuijieDot1xEnable 硬编码 `default-user-limit 1`,非 1 端口被错置为 1。
//
// 解决:
//   - 加 nullable int 列缓存 MAX_USER (TextFSM `show dot1x port-control` 已能解析)。
//   - renderRuijieDot1xEnable 读缓存值下发; nil (设备无配置 / 尚未采集) 兜底 1。
//   - 100% 向后兼容: 历史端口 dot1x_user_limit=0,被兜底, 与硬编码 1 行为一致。
//
// 范围: 本迁移只动 sys_device_port_status。华为 dot1x profile 模板名错位
// (不同端口可能用不同 profile name, undo authentication-profile 不带名字, 重新
// enable 时必须读到原 profile 名) 属另一范畴,留待后续 Phase。
func Migrate204AddDot1xUserLimit(db *gorm.DB) error {
	log.Println("Running migration 204: 加 sys_device_port_status.dot1x_user_limit")

	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] 204 跳过(非 PostgreSQL)")
		log.Println("Migration 204 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// 幂等性: 列已存在则跳过 (IF NOT EXISTS 在 PG 9.6+ 支持)
	var err error
	if err = db.Exec(`
ALTER TABLE sys_device_port_status
ADD COLUMN IF NOT EXISTS dot1x_user_limit INTEGER NOT NULL DEFAULT 0`).Error; err != nil {
		return fmt.Errorf("加 dot1x_user_limit 列失败: %w", err)
	}

	applogger.Infof("[迁移] 204 dot1x_user_limit 列已加 (INTEGER NOT NULL DEFAULT 0)")
	log.Println("Migration 204 completed")
	return nil
}
