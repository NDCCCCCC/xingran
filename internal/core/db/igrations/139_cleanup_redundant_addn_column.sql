-- Migration 139: 彻底清理 sys_user 表的冗余 addn 列
-- Date: 2026-05-27
-- Description: 解决数据库中同时存在 ad_dn 和 addn 列的问题
--              确保只保留 ad_dn 列，删除 addn 列

-- 步骤1: 备份检查 - 显示当前状态
DO $$
BEGIN
    RAISE NOTICE '=== 迁移 139 开始：清理冗余 addn 列 ===';

    -- 检查列是否存在
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'sys_user' AND column_name = 'addn'
    ) THEN
        RAISE NOTICE '✓ 检测到 addn 列存在';
    ELSE
        RAISE NOTICE '✓ addn 列不存在（已清理）';
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'sys_user' AND column_name = 'ad_dn'
    ) THEN
        RAISE NOTICE '✓ 检测到 ad_dn 列存在';
    ELSE
        RAISE NOTICE '⚠ ad_dn 列不存在（将创建）';
    END IF;
END $$;

-- 步骤2: 确保 ad_dn 列存在（如果不存在则创建）
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'sys_user' AND column_name = 'ad_dn'
    ) THEN
        ALTER TABLE sys_user ADD COLUMN ad_dn TEXT;
        RAISE NOTICE '✓ 已创建 ad_dn 列';
    END IF;
END $$;

-- 步骤3: 数据迁移 - 将 addn 数据复制到 ad_dn
UPDATE sys_user
SET ad_dn = addn
WHERE addn IS NOT NULL
  AND addn != ''
  AND (ad_dn IS NULL OR ad_dn = '');

-- 记录迁移的数据量
DO $$
DECLARE
    migrated_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO migrated_count
    FROM sys_user
    WHERE addn IS NOT NULL AND addn != '';

    IF migrated_count > 0 THEN
        RAISE NOTICE '✓ 已迁移 % 条记录从 addn 到 ad_dn', migrated_count;
    END IF;
END $$;

-- 步骤4: 删除冗余的 addn 列
ALTER TABLE sys_user DROP COLUMN IF EXISTS addn;

RAISE NOTICE '✓ 已删除 addn 列';

-- 步骤5: 验证结果
DO $$
DECLARE
    ad_dn_exists BOOLEAN;
    addn_exists BOOLEAN;
BEGIN
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'sys_user' AND column_name = 'ad_dn'
    ) INTO ad_dn_exists;

    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'sys_user' AND column_name = 'addn'
    ) INTO addn_exists;

    IF ad_dn_exists AND NOT addn_exists THEN
        RAISE NOTICE '=== ✅ 迁移 139 成功：只保留 ad_dn 列 ===';
    ELSIF ad_dn_exists AND addn_exists THEN
        RAISE NOTICE '=== ⚠ 迁移 139 警告：addn 列仍然存在 ===';
    ELSE
        RAISE NOTICE '=== ❌ 迁移 139 失败：ad_dn 列不存在 ===';
    END IF;
END $$;

-- 步骤6: 显示最终列状态
SELECT
    '139_cleanup_final_state' AS migration,
    column_name,
    data_type
FROM information_schema.columns
WHERE table_name = 'sys_user'
  AND column_name IN ('ad_dn', 'addn')
ORDER BY column_name;

-- 预期结果：
-- migration                | column_name | data_type
-- ------------------------|-------------|---------
-- 139_cleanup_final_state | ad_dn       | text
-- （应该只有一行，addn 列已被删除）
