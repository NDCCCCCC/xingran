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
)
