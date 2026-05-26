-- 将测试 API 密钥关联到当前用户并启用
-- 请替换 YOUR_USER_ID 为实际的用户 ID（从 sys_user 表中获取）

-- 1. 查看当前登录用户的 ID
SELECT id, user_name, nick_name FROM sys_user ORDER BY created_at DESC LIMIT 5;

-- 2. 更新测试 API 密钥（将 YOUR_USER_ID 替换为实际的用户 ID）
UPDATE sys_api_keys
SET
    is_active = true,
    user_id = 'YOUR_USER_ID',
    updated_at = NOW()
WHERE id = '3a58624d-5653-4084-a6db-4d00b83e3cb8';

-- 3. 验证更新结果
SELECT id, name, LEFT(key, 20) as key_preview, is_active, user_id
FROM sys_api_keys
WHERE id = '3a58624d-5653-4084-a6db-4d00b83e3cb8';
