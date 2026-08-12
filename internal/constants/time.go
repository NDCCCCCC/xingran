// 时间相关常量:统一管理项目中使用的时间常量,避免硬编码。

package constants

import "time"

// 缓存相关时间常量
const (
	// DefaultCacheExpire 默认缓存过期时间
	DefaultCacheExpire = 5 * time.Minute

	// CacheConfigExpire 缓存配置过期时间(用于动态配置的缓存值)
	CacheConfigExpire = 30 * time.Minute

	// CaptchaCacheExpire 验证码缓存过期时间
	CaptchaCacheExpire = 5 * time.Minute

	// BackgroundCacheExpire 验证码背景缓存过期时间
	BackgroundCacheExpire = 24 * time.Hour

	// LoginLockExpire 登录锁定缓存过期时间
	LoginLockExpire = 30 * time.Minute
)

// 网络超时时间常量
const (
	// DefaultHTTPTimeout 默认 HTTP 请求超时时间
	DefaultHTTPTimeout = 30 * time.Second

	// SNMPTimeout SNMP 查询超时时间
	SNMPTimeout = 5 * time.Second

	// DeviceInitTimeout 设备连接初始化超时时间
	DeviceInitTimeout = 30 * time.Second

	// DeviceTaskTimeout 设备任务默认超时时间
	DeviceTaskTimeout = 5 * time.Minute

	// DeviceTaskExtraTimeout 设备任务额外超时时间(在任务超时后额外等待)
	DeviceTaskExtraTimeout = 1 * time.Minute

	// DeviceMaxIdleTime 设备连接池最大空闲时间
	DeviceMaxIdleTime = 5 * time.Minute

	// RetryDelay 重试延迟时间
	RetryDelay = 1 * time.Second
)

// 系统时间常量
const (
	// OneDay 一天的时间
	OneDay = 24 * time.Hour

	// ShutdownTimeout 服务关闭超时时间
	ShutdownTimeout = 10 * time.Second

	// WebSocketHeartbeatInterval WebSocket 心跳间隔
	WebSocketHeartbeatInterval = 54 * time.Second
)

// JWT Token 过期时间常量(默认值,实际值由配置文件决定)
const (
	// DefaultAccessTokenExpire 默认访问令牌过期时间(2小时)
	DefaultAccessTokenExpire = 2 * time.Hour

	// DefaultRefreshTokenExpire 默认刷新令牌过期时间(7天)
	DefaultRefreshTokenExpire = 7 * 24 * time.Hour
)
