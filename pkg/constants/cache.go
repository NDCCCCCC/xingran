package constants

// cache.go — Redis cache key format constants (V130R-09 D-03-10: merged from
// internal/constants/cache.go when the twin constants packages were unified).

// Redis 键格式(仅保留生产代码实际使用的格式)。
//
// 调用方应使用这些常量而非内联字面量,避免 key 拼写分叉。
const (
	// TokenBlacklistKeyFormat Token 黑名单键格式
	TokenBlacklistKeyFormat = "token:blacklist:%s"

	// LoginLockKeyFormat 登录锁定键格式
	LoginLockKeyFormat = "login:lock:%s"

	// CaptchaVerifiedKeyFormat 验证码验证键格式
	CaptchaVerifiedKeyFormat = "captcha:verified:%s"

	// CaptchaRateLimitKeyFormat 验证码限流键格式
	CaptchaRateLimitKeyFormat = "captcha:rate:%s"

	// CaptchaDataKeyFormat 验证码数据键格式
	CaptchaDataKeyFormat = "captcha:data:%s"

	// CaptchaAttemptsKeyFormat 验证码尝试次数键格式
	CaptchaAttemptsKeyFormat = "captcha:attempts:%s"

	// LoginFailKeyFormat 登录失败计数键格式
	LoginFailKeyFormat = "login:fail:%s"

	// CaptchaBgListKeyFormat 验证码背景图列表键格式
	CaptchaBgListKeyFormat = "captcha:bg:list:%s:%d"

	// CaptchaCachePoolPrefixFormat 验证码缓存池前缀格式
	CaptchaCachePoolPrefixFormat = "captcha:cache:pool:%s:%d"
)
