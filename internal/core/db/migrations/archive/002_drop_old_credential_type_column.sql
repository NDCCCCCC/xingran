-- ==========================================
-- 删除旧的 credential_type 列
-- ==========================================
-- 说明：credential_type 已被 protocol_type 替代
-- 这个迁移完成 001_redesign_credential_model.sql 中被注释的删除操作
-- ==========================================

-- 先为可能为空的 credential_type 设置默认值
UPDATE sys_auth_credential
SET credential_type = 'ssh'
WHERE credential_type IS NULL;

-- 删除旧的 credential_type 列
ALTER TABLE sys_auth_credential DROP COLUMN IF EXISTS credential_type;

-- 删除旧的 snmp_community 列（如果还存在）
ALTER TABLE sys_auth_credential DROP COLUMN IF EXISTS snmp_community;

-- 验证结果
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'sys_auth_credential'
ORDER BY ordinal_position;
