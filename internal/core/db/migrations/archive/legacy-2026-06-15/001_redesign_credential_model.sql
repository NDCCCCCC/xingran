-- ==========================================
-- 授权凭证模型重新设计迁移脚本
-- ==========================================
-- 变更说明：
-- 1. 移除 credential_type 字段，改为 protocol_type (ssh/telnet)
-- 2. snmp_community 单值字段改为 snmp_communities 数组
-- 3. 一个凭证可以同时包含 SSH/Telnet 配置和 SNMP 配置
-- ==========================================

-- 第一步：添加新列
ALTER TABLE sys_auth_credential
ADD COLUMN IF NOT EXISTS protocol_type VARCHAR(10);

ALTER TABLE sys_auth_credential
ADD COLUMN IF NOT EXISTS snmp_communities TEXT[];

-- 第二步：根据旧的 credential_type 设置 protocol_type
-- SSH 类型凭证 -> protocol_type = 'ssh'
UPDATE sys_auth_credential
SET protocol_type = 'ssh'
WHERE credential_type = 'ssh';

-- Telnet 类型凭证 -> protocol_type = 'telnet'
UPDATE sys_auth_credential
SET protocol_type = 'telnet'
WHERE credential_type = 'telnet';

-- SNMP 类型凭证 -> protocol_type = 'ssh' (默认使用 SSH 协议)
UPDATE sys_auth_credential
SET protocol_type = 'ssh'
WHERE credential_type = 'snmp' AND protocol_type IS NULL;

-- 第三步：将旧的单个 snmp_community 转换为数组
UPDATE sys_auth_credential
SET snmp_communities = ARRAY[snmp_community]
WHERE snmp_community IS NOT NULL
  AND snmp_community != ''
  AND snmp_community != ' ';

-- 对于没有 SNMP community 的记录，初始化空数组
UPDATE sys_auth_credential
SET snmp_communities = ARRAY[]::TEXT[]
WHERE snmp_communities IS NULL;

-- 第四步：设置默认值
-- 为没有设置 protocol_type 的记录设置默认值
UPDATE sys_auth_credential
SET protocol_type = 'ssh'
WHERE protocol_type IS NULL;

-- 第五步：验证数据（迁移前检查）
SELECT
    id,
    credential_name,
    credential_type AS old_credential_type,
    protocol_type AS new_protocol_type,
    snmp_community AS old_snmp_community,
    snmp_communities AS new_snmp_communities,
    username,
    snmp_version
FROM sys_auth_credential
ORDER BY id;

-- 第六步：（验证通过后执行）删除旧列
-- 取消以下注释以执行删除操作
-- ALTER TABLE sys_auth_credential DROP COLUMN credential_type;
-- ALTER TABLE sys_auth_credential DROP COLUMN snmp_community;

-- 第七步：添加约束
ALTER TABLE sys_auth_credential
ADD CONSTRAINT chk_protocol_type
CHECK (protocol_type IN ('ssh', 'telnet'));

ALTER TABLE sys_auth_credential
ADD CONSTRAINT chk_snmp_version
CHECK (snmp_version IN ('v1', 'v2c', 'v3'));

-- 第八步：添加注释
COMMENT ON COLUMN sys_auth_credential.protocol_type IS 'SSH/Telnet 协议选择';
COMMENT ON COLUMN sys_auth_credential.snmp_communities IS 'SNMP Community 列表（支持多个）';

-- ==========================================
-- 回滚脚本（如需回滚，请执行以下语句）
-- ==========================================
/*
-- 删除新列
ALTER TABLE sys_auth_credential DROP COLUMN IF EXISTS protocol_type;
ALTER TABLE sys_auth_credential DROP COLUMN IF EXISTS snmp_communities;

-- 恢复旧列
ALTER TABLE sys_auth_credential ADD COLUMN credential_type VARCHAR(50);
ALTER TABLE sys_auth_credential ADD COLUMN snmp_community VARCHAR(100);

-- 恢复数据
UPDATE sys_auth_credential
SET credential_type = protocol_type
WHERE protocol_type IN ('ssh', 'telnet');

UPDATE sys_auth_credential
SET snmp_community = snmp_communities[1]
WHERE snmp_communities IS NOT NULL AND array_length(snmp_communities, 1) > 0;

-- 添加约束
ALTER TABLE sys_auth_credential
ADD CONSTRAINT chk_credential_type
CHECK (credential_type IN ('ssh', 'telnet', 'snmp'));
*/
