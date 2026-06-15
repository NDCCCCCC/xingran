-- Migration 137: 清理 sys_user 表的冗余 addn 列
-- Description: 修复回归问题 - GORM 自动迁移重新创建了 addn 列
-- Created: 2026-05-26

-- 1. 将 addn 列的数据迁移到 ad_dn 列
UPDATE sys_user
SET ad_dn = addn
WHERE ad_dn IS NULL
AND addn IS NOT NULL;

-- 2. 删除 addn 列
ALTER TABLE sys_user
DROP COLUMN addn;
