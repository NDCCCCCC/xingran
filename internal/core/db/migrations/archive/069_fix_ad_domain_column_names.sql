-- =============================================
-- AD域管理表列名修正
-- 迁移版本: 069
-- 描述: 移除 oun 列的 NOT NULL 约束，迁移数据后删除列
-- =============================================

-- 步骤1: 移除 oun 列的 NOT NULL 约束
ALTER TABLE sys_ad_ou ALTER COLUMN oun DROP NOT NULL;
ALTER TABLE sys_ad_group ALTER COLUMN oun DROP NOT NULL;
ALTER TABLE sys_ad_user ALTER COLUMN oun DROP NOT NULL;

-- 步骤2: 如果 ou_dn 为空但 oun 有数据，则复制数据
UPDATE sys_ad_ou SET ou_dn = oun WHERE ou_dn IS NULL AND oun IS NOT NULL;

UPDATE sys_ad_group SET ou_dn = oun WHERE ou_dn IS NULL AND oun IS NOT NULL;

UPDATE sys_ad_user SET ou_dn = oun WHERE ou_dn IS NULL AND oun IS NOT NULL;

-- 步骤3: 删除多余的 oun 列
ALTER TABLE sys_ad_ou DROP COLUMN IF EXISTS oun;
ALTER TABLE sys_ad_group DROP COLUMN IF EXISTS oun;
ALTER TABLE sys_ad_user DROP COLUMN IF EXISTS oun;

-- 重建索引（确保使用 ou_dn）
DROP INDEX IF EXISTS idx_ad_group_ou CASCADE;
CREATE INDEX idx_ad_group_ou ON sys_ad_group(ou_dn) WHERE deleted_at IS NULL;

DROP INDEX IF EXISTS idx_ad_user_ou CASCADE;
CREATE INDEX idx_ad_user_ou ON sys_ad_user(ou_dn) WHERE deleted_at IS NULL;

-- 添加列注释
COMMENT ON COLUMN sys_ad_ou.ou_dn IS 'OU的LDAP DN';
COMMENT ON COLUMN sys_ad_group.ou_dn IS '所属OU的DN';
COMMENT ON COLUMN sys_ad_user.ou_dn IS '所属OU的DN';
COMMENT ON COLUMN sys_ad_group.group_dn IS '组的LDAP DN';
COMMENT ON COLUMN sys_ad_user.user_dn IS '用户的LDAP DN';
