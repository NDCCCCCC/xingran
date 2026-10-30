package migrations

import (
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate216ADPushDrainSysConfig ad_push_queue 排水间隔移入 sys_config（Phase 58 D-05 单旋钮）。
//
// 背景: 排水间隔原为硬编码 30s。本迁移把它变为可配，给"域控吃不消"留放宽到 5min 的逃生门
//      （默认 30s 保证"管理员编辑 → AD 生效感知 ≤1min"）。
//
// ConfigType=N（非系统参数，前端「参数设置」页可编辑），IsSystem=0。
// ⚠ 方案A热重载（2026-10-21）：排水器每 30s tick 读取该配置（drainGate 门控），
//   修改后**无需重启**，≤30s 生效。
// ⚠ 非阻断：缺旋钮只会让 LoadDrainIntervalSeconds 回落默认 30s，不影响业务 ——
//   对照 migration 215 建表的"失败阻断启动"档，seed 类迁移不该阻断。
func Migrate216ADPushDrainSysConfig(db *gorm.DB) error {
	log.Println("Running migration 216: ad_push_queue drain interval sys_config seed")

	const (
		seedKey    = "ad_push_queue.drain_interval_seconds"
		seedValue  = "30"
		seedName   = "AD推送队列-排水间隔(秒)"
		seedRemark = "ad_push_queue 排水器轮询间隔(秒);建议 30,域控压力大时可放宽到 300;修改后约30秒内生效(无需重启)"
	)

	var existingCount int64
	if err := db.Model(&models.Config{}).Where("config_key = ?", seedKey).Count(&existingCount).Error; err != nil {
		return err
	}
	if existingCount > 0 { // 幂等：已存在则跳过（管理员可能已改过值，绝不覆盖）
		return nil
	}
	cfg := &models.Config{
		ConfigName:  seedName,
		ConfigKey:   seedKey,
		ConfigValue: seedValue,
		ConfigType:  models.ConfigTypeNo,
		IsSystem:    models.ConfigIsSystemNo,
		Remark:      seedRemark,
	}
	if err := db.Create(cfg).Error; err != nil {
		log.Printf("Migration 216: create config %s failed: %v", seedKey, err)
		return nil // 非阻断：回落默认 30s 即可
	}
	applogger.Infof("[迁移] sys_config seed: %s = %s", seedKey, seedValue)
	return nil
}
