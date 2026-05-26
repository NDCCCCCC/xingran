-- 检查 API 密钥数据（包括已删除的）
SELECT
    id,
    name,
    key,
    is_active,
    created_at,
    updated_at,
    deleted_at,
    created_by
FROM sys_api_keys
WITH CHECK OPTION;
