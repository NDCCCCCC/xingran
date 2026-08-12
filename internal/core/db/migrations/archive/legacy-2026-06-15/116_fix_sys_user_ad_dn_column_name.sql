-- Migration: 116_fix_sys_user_ad_dn_column_name.sql
-- Date: 2026-05-22
-- Description: 修复 sys_user 表的 AD DN 列名，从 addn 改为 ad_dn
-- Issue: 列名与迁移文件 100_add_auth_source_fields.sql 定义不一致

-- 重命名列 addn 为 ad_dn
ALTER TABLE sys_user
RENAME COLUMN addn TO ad_dn;

-- 验证列已重命名
SELECT
    '116_fix_sys_user_ad_dn_column_name.sql' AS migration,
    '重命名 addn → ad_dn 完成' AS status,
    column_name,
    data_type
FROM information_schema.columns
WHERE table_name = 'sys_user'
  AND column_name IN ('ad_dn', 'auth_source', 'ad_username');
