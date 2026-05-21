-- 修复 API 密钥表中的 JSONB 字段数据
-- 问题：现有数据存储为字符串 '"["read","write"]"' 而不是 JSON 数组 '["read","write"]'
-- 解决：使用 jsonb_populate_record 或直接更新为正确的 JSON 格式

DO $$
DECLARE
    v_record RECORD;
    v_scopes_jsonb jsonb;
    v_whitelist_jsonb jsonb;
BEGIN
    -- 遍历所有 API 密钥记录
    FOR v_record IN
        SELECT id, scopes, ip_whitelist
        FROM sys_api_keys
        WHERE deleted_at IS NULL
    LOOP
        -- 修复 scopes 字段
        IF v_record.scopes IS NOT NULL AND v_record.scopes != '' THEN
            BEGIN
                -- 尝试解析为 JSONB
                v_scopes_jsonb := v_record.scopes::jsonb;

                -- 如果是字符串（例如 '"["read"]"'），提取实际的数组
                IF jsonb_typeof(v_scopes_jsonb) = 'string' THEN
                    UPDATE sys_api_keys
                    SET scopes = v_scopes_jsonb#>>'{}'::jsonb
                    WHERE id = v_record.id;
                END IF;
            EXCEPTION WHEN OTHERS THEN
                -- 如果解析失败，设置为空数组
                UPDATE sys_api_keys
                SET scopes = '[]'::jsonb
                WHERE id = v_record.id;
            END;
        ELSE
            UPDATE sys_api_keys
            SET scopes = '[]'::jsonb
            WHERE id = v_record.id;
        END IF;

        -- 修复 ip_whitelist 字段
        IF v_record.ip_whitelist IS NOT NULL AND v_record.ip_whitelist != '' THEN
            BEGIN
                v_whitelist_jsonb := v_record.ip_whitelist::jsonb;

                IF jsonb_typeof(v_whitelist_jsonb) = 'string' THEN
                    UPDATE sys_api_keys
                    SET ip_whitelist = v_whitelist_jsonb#>>'{}'::jsonb
                    WHERE id = v_record.id;
                END IF;
            EXCEPTION WHEN OTHERS THEN
                UPDATE sys_api_keys
                SET ip_whitelist = '[]'::jsonb
                WHERE id = v_record.id;
            END;
        ELSE
            UPDATE sys_api_keys
            SET ip_whitelist = '[]'::jsonb
            WHERE id = v_record.id;
        END IF;
    END LOOP;

    RAISE NOTICE 'API 密钥 JSONB 数据修复完成！';
END $$;

-- 验证修复结果
SELECT
    id,
    name,
    jsonb_typeof(scopes) as scopes_type,
    scopes,
    jsonb_typeof(ip_whitelist) as whitelist_type,
    ip_whitelist
FROM sys_api_keys
WHERE deleted_at IS NULL
LIMIT 5;

SELECT '✅ JSONB 数据格式修复完成' AS status;
