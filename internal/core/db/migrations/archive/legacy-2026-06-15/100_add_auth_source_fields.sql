-- Migration 100: 添加AD认证相关字段
-- Date: 2026-05-21
-- Description: 支持AD域控账号登录，添加认证源和AD用户名字段

-- 添加认证源字段（local/ad）
ALTER TABLE sys_user
ADD COLUMN IF NOT EXISTS auth_source VARCHAR(10) DEFAULT 'local';

-- 添加AD用户名字段
ALTER TABLE sys_user
ADD COLUMN IF NOT EXISTS ad_username VARCHAR(100);

-- 添加AD DN字段（可选，用于快速定位AD用户）
ALTER TABLE sys_user
ADD COLUMN IF NOT EXISTS ad_dn TEXT;

-- 创建索引：ad_username（用于快速查找AD用户）
CREATE INDEX IF NOT EXISTS idx_sys_user_ad_username
ON sys_user(ad_username);

-- 创建索引：auth_source（用于按认证源筛选用户）
CREATE INDEX IF NOT EXISTS idx_sys_user_auth_source
ON sys_user(auth_source);

-- 添加注释
COMMENT ON COLUMN sys_user.auth_source IS '认证源：local=本地认证, ad=AD域控认证';
COMMENT ON COLUMN sys_user.ad_username IS 'AD域控用户名';
COMMENT ON COLUMN sys_user.ad_dn IS 'AD域控DN（Distinguished Name）';

-- 为现有用户设置默认认证源
UPDATE sys_user
SET auth_source = 'local'
WHERE auth_source IS NULL;

-- 设置NOT NULL约束
ALTER TABLE sys_user
ALTER COLUMN auth_source SET NOT NULL;

-- 创建唯一约束：同一AD用户只能对应一个本地账号（部分索引，仅非NULL值）
CREATE UNIQUE INDEX IF NOT EXISTS idx_sys_user_ad_username_unique
ON sys_user(ad_username)
WHERE ad_username IS NOT NULL;
