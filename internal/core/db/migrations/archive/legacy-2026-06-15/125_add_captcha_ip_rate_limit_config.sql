-- 添加验证码 IP 限流配置参数
-- 配置键：sys.captcha.ip_rate_limit
-- 说明：每个 IP 每分钟最多获取验证码的次数
-- 默认值：10

INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    '验证码IP限流配置',
    'sys.captcha.ip_rate_limit',
    '10',
    'Y',
    1,
    '每个IP每分钟最多获取验证码的次数（默认10次）',
    NOW(),
    NOW()
)
ON CONFLICT (config_key) DO NOTHING;
