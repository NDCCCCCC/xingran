-- 验证码系统配置参数初始化脚本
-- 表名: sys_config
-- 说明: 这些配置参数用于控制用户登录验证码系统的行为

-- 插入验证码配置参数
INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark) VALUES
(gen_random_uuid(), '验证码开关', 'sys.account.captchaEnabled', 'disabled', 'Y', 1, '验证码开关: disabled=停用 normal=数字验证码 slider=滑动验证码'),
(gen_random_uuid(), '验证码类型', 'sys.account.captchaType', '4', 'Y', 1, '数字字母验证码长度(4-6位)'),
(gen_random_uuid(), '验证码有效期', 'sys.account.captchaExpireTime', '5', 'Y', 1, '验证码有效期(分钟)'),
(gen_random_uuid(), '验证码最大尝试次数', 'sys.account.captchaMaxAttempts', '3', 'Y', 1, '验证码验证失败最大次数'),
(gen_random_uuid(), 'IP限流阈值', 'sys.account.ipRateLimit', '10', 'Y', 1, '每分钟获取验证码最大次数'),
(gen_random_uuid(), '登录最大重试次数', 'sys.account.loginMaxRetry', '5', 'Y', 1, '登录失败最大次数'),
(gen_random_uuid(), '账号锁定时间', 'sys.account.loginLockTime', '30', 'Y', 1, '账号锁定时间(分钟)');

-- 配置说明:
-- 1. sys.account.captchaEnabled: 控制验证码是否启用及类型
--    - disabled: 完全停用验证码
--    - normal: 使用数字字母验证码
--    - slider: 使用滑动拼图验证码
--
-- 2. sys.account.captchaType: 数字验证码的长度
--    - 可选值: 4, 5, 6
--
-- 3. sys.account.captchaExpireTime: 验证码有效时间
--    - 单位: 分钟
--    - 超时后验证码自动失效
--
-- 4. sys.account.captchaMaxAttempts: 单个验证码最大验证次数
--    - 超过此次数后验证码自动失效
--
-- 5. sys.account.ipRateLimit: IP限流
--    - 单个IP每分钟最多获取验证码次数
--    - 防止恶意刷验证码
--
-- 6. sys.account.loginMaxRetry: 登录失败最大次数
--    - 连续失败达到此次数后账号被锁定
--
-- 7. sys.account.loginLockTime: 账号锁定时间
--    - 单位: 分钟
--    - 锁定期间无法登录
