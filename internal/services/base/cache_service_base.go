package base

import (
	"time"
)

// =====================================================================
// Phase 92-01 (D-01): TTLResolver + CacheServiceBase 薄基类。
//
// 依赖倒置决策（consumer-defined interface）：base 包保持零依赖（与 Phase 91
// service.go 的"纯抽象层"定位一致），基类对 TTL 配置的依赖收敛为本包定义的
// 最小接口 TTLResolver；services.CacheConfigService 以同名方法隐式满足
// （GetDurationWithDefault 签名逐字一致），root→base 是合法依赖方向
// （root 现有 18 文件走此方向实证无环）。
//
// 红线：本包禁止 import internal/services 或 internal/services/system
//（反向依赖会引入 root↔base import cycle，破坏零依赖基线）。
// =====================================================================

// TTLResolver TTL 配置解析接口（D-01，consumer-defined）。
//
// 唯一方法签名与 internal/services/cache_config_service.go 的
// CacheConfigService.GetDurationWithDefault 逐字一致 → *CacheConfigService
// 隐式满足，无需适配层。
type TTLResolver interface {
	GetDurationWithDefault(configKey string, defaultDuration time.Duration) time.Duration
}

// CacheServiceBase 缓存服务基础结构（薄基类，自 system/cache_utils.go 迁入）。
//
// 原 Config 字段类型 *services.CacheConfigService 改为 TTLResolver 接口：
//   - 嵌入点构造字面量 CacheServiceBase{Config: config} 零改动
//     （*CacheConfigService → 接口隐式转换）
//   - nil 语义与迁移前逐字一致：Config == nil 时 GetExpiration 返回 defaultVal；
//     typed-nil（接口化后 if Config != nil 判空失效的场景）由
//     CacheConfigService.GetDurationWithDefault 顶部 nil-receiver 防护兜底
//     （Phase 92-01 配套，Pitfall 1）
type CacheServiceBase struct {
	Config TTLResolver
}

// GetExpiration 获取缓存过期时间（通用方法）
func (b *CacheServiceBase) GetExpiration(configKey string, defaultVal time.Duration) time.Duration {
	if b.Config != nil {
		return b.Config.GetDurationWithDefault(configKey, defaultVal)
	}
	return defaultVal
}
