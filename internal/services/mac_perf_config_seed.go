package services

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// MAC 性能配置键常量 (D-06 锁定)
const (
	MACPerfConfigMatViewRefreshCron = "network.mac.perf.mat_view_refresh_cron"
	MACPerfConfigCacheTTLSeconds    = "network.mac.perf.cache_ttl_seconds"
	MACPerfConfigHeatmapTopN        = "network.mac.perf.heatmap_top_n"
)

// macPerfConfigDefaults 3 个 MAC 性能配置默认值
var macPerfConfigDefaults = []models.Config{
	{ConfigName: "MAC物化视图刷新Cron", ConfigKey: MACPerfConfigMatViewRefreshCron, ConfigValue: "0 */5 * * * *", ConfigType: models.ConfigTypeYes},
	{ConfigName: "MAC查询缓存TTL(秒)", ConfigKey: MACPerfConfigCacheTTLSeconds, ConfigValue: "300", ConfigType: models.ConfigTypeYes},
	{ConfigName: "MAC热力图TopN", ConfigKey: MACPerfConfigHeatmapTopN, ConfigValue: "100", ConfigType: models.ConfigTypeYes},
}

// SeedMACPerfConfigs 幂等 upsert 3 个 MAC 性能 sys_config 键
// 已存在则跳过 (ConfigKey 唯一索引), 不存在则插入
func SeedMACPerfConfigs(db *gorm.DB) error {
	if db == nil {
		applogger.Warnf("[SeedMACPerfConfigs] 数据库未设置，跳过")
		return nil
	}

	for _, cfg := range macPerfConfigDefaults {
		cfg := cfg
		// 用 ConfigKey 字段 (GORM 映射到 config_key) 查询
		// 已存在则不覆盖, 不存在则插入
		err := db.Where(&models.Config{ConfigKey: cfg.ConfigKey}).
			Attrs(cfg).
			FirstOrCreate(&models.Config{}).Error
		if err != nil {
			applogger.Errorf("[SeedMACPerfConfigs] upsert %s 失败: %v", cfg.ConfigKey, err)
			return err
		}
		applogger.Infof("[SeedMACPerfConfigs] 配置 %s 已就绪", cfg.ConfigKey)
	}

	return nil
}
