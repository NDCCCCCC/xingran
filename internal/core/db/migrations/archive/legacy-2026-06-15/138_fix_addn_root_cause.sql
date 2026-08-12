-- Migration 138: 彻底修复 GORM AutoMigrate addn 列反复创建问题
-- Date: 2026-05-27
-- Root Cause: User 模型字段 ADDN (全大写) 被 GORM 命名策略转换为 "addn" 列名
-- Solution: 将字段名从 ADDN 改为 AdDn，GORM 将正确转换为 ad_dn

-- 步骤1: 确保数据迁移到 ad_dn 列
UPDATE sys_user
SET ad_dn = addn
WHERE addn IS NOT NULL
  AND addn != ''
  AND (ad_dn IS NULL OR ad_dn = '');

-- 步骤2: 删除 addn 列（GORM 将不再创建此列）
ALTER TABLE sys_user
DROP COLUMN IF EXISTS addn;

-- 步骤3: 验证修复（应该只看到 ad_dn，没有 addn）
SELECT
    '138_fix_addn_root_cause' AS migration,
    '修复完成：ad_dn 保留，addn 已删除' AS status,
    column_name,
    data_type
FROM information_schema.columns
WHERE table_name = 'sys_user'
  AND column_name IN ('ad_dn', 'addn');

-- 预期结果：
-- 只有 ad_dn 行，没有 addn 行
-- column_name | data_type
-- ad_dn       | text
