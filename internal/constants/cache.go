/**
 * 缓存相关常量
 *
 * 统一管理 Redis 缓存键前缀和格式
 */
package constants

// Redis 键前缀
const (
	// Redis 全局键前缀
	RedisKeyPrefix = "xingran:"
)

// Redis 键格式
const (
	// Token 黑名单键格式
	TokenBlacklistKeyFormat = "token:blacklist:%s"

	// 登录失败记录键格式
	LoginFailureKeyFormat = "login:failure:%s"

	// 登录锁定键格式
	LoginLockKeyFormat = "login:lock:%s"

	// 验证码存储键格式
	CaptchaStorageKeyFormat = "captcha:storage:%s"

	// 验证码验证键格式
	CaptchaVerifiedKeyFormat = "captcha:verified:%s"

	// 验证码锁定键格式
	CaptchaLockKeyFormat = "captcha:lock:%s"

	// 验证码背景缓存键格式
	CaptchaBgCacheKeyFormat = "captcha:bg:%s"

	// 验证码背景计数器键格式
	CaptchaBgCounterKeyFormat = "captcha:bg:counter"

	// 用户在线状态键格式
	UserOnlineKeyFormat = "user:online:%s"
)
