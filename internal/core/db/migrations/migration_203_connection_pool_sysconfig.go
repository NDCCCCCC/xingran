package migrations

import (
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate203ConnectionPoolSysConfig 网络设备连接池配置移入 sys_config (web 可配)
//
// 背景:
//   - core.go 原硬编码 MaxConnections=20 (降并发避 scrapligo panic)、MaxIdle=5min。
//   - 实际部署设备数常 >20,池满 (currentCount>=max) 时 GetConnection 快速失败,导致
//     间歇性端口采集跳过 ("连接池已满: 当前=20, 最大=20")。
//   - 配套 GetConnection 已修 TOCTOU + LRU 退让;本迁移把容量与 idle 时长改为可配,
//     默认 max_connections=50 (兼顾并发与 scrapligo 稳定性), max_idle_seconds=300。
//
// ConfigType=N (非系统参数,前端"参数设置"页可编辑), IsSystem=0。
// 注意: 连接池在启动时读取配置,修改后需重启后端生效。
func Migrate203ConnectionPoolSysConfig(db *gorm.DB) error {
	log.Println("Running migration 203: connection pool sys_config seeds")

	configs := []struct {
		configName  string
		configKey   string
		configValue string
		remark      string
	}{
		{
			configName:  "连接池-最大连接数",
			configKey:   "network.connection_pool.max_connections",
			configValue: "50",
			remark:      "网络设备 SSH 连接池最大连接数;设备数较多且未触发 scrapligo panic 时可调高;修改后需重启后端生效",
		},
		{
			configName:  "连接池-空闲超时(秒)",
			configKey:   "network.connection_pool.max_idle_seconds",
			configValue: "300",
			remark:      "连接空闲超过此时长(秒)后被 cleanup 关闭释放槽位;调小可加快槽位流转;修改后需重启后端生效",
		},
	}

	for _, c := range configs {
		var existingCount int64
		if err := db.Model(&models.Config{}).Where("config_key = ?", c.configKey).Count(&existingCount).Error; err != nil {
			return err
		}
		if existingCount > 0 {
			continue
		}
		cfg := &models.Config{
			ConfigName:  c.configName,
			ConfigKey:   c.configKey,
			ConfigValue: c.configValue,
			ConfigType:  models.ConfigTypeNo, // N = 非系统参数, 前端可编辑
			IsSystem:    models.ConfigIsSystemNo,
			Remark:      c.remark,
		}
		if err := db.Create(cfg).Error; err != nil {
			log.Printf("Migration 203: create config %s failed: %v", c.configKey, err)
			continue
		}
		applogger.Infof("[迁移] sys_config seed: %s = %s", c.configKey, c.configValue)
	}

	log.Println("Migration 203 completed: connection pool configs seeded")
	return nil
}
