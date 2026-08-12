-- 手动插入部门到AD OU映射的SQL脚本
-- 用于快速解决"未找到OU DN对应部门"的问题
--
-- 使用说明：
-- 1. 根据实际AD配置ID修改下面的 ad_config_id 值
-- 2. 根据实际部门ID修改 dept_id 值
-- 3. 根据AD OU结构修改 ou_dn 值

-- 示例：为用户 chenchao-076 所在的OU创建映射
-- AD OU: OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn
-- 系统部门: 分公司本部-科技创新部-基础运维科

-- 首先查询部门ID（替换"基础运维科"为实际部门名）
SELECT id, dept_name, parent_id FROM sys_dept WHERE dept_name = '基础运维科';

-- 然后查询AD配置ID
SELECT id, config_name, enabled FROM sys_ad_config WHERE enabled = true;

-- 最后插入映射关系（替换下面的ID为实际值）
INSERT INTO sys_dept_ou_mapping (
    id,
    dept_id,
    ad_config_id,
    ou_dn,
    ou_name,
    parent_ou_dn,
    sync_enabled,
    sync_status,
    created_at,
    updated_at
) VALUES (
    gen_random_uuid(),                          -- 自动生成UUID
    'DEPT_ID_HERE',                             -- 替换为实际的部门ID
    'AD_CONFIG_ID_HERE',                        -- 替换为实际的AD配置ID
    'OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn',
    '基础运维科',                                -- OU名称
    'OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn', -- 父OU DN
    true,                                       -- 启用同步
    'synced',                                   -- 同步状态
    NOW(),                                      -- 创建时间
    NOW()                                       -- 更新时间
) ON CONFLICT (dept_id, ad_config_id)
DO UPDATE SET
    ou_dn = EXCLUDED.ou_dn,
    ou_name = EXCLUDED.ou_name,
    sync_enabled = EXCLUDED.sync_enabled,
    updated_at = NOW();

-- 验证插入结果
SELECT
    m.id,
    d.dept_name,
    m.ou_dn,
    m.ou_name,
    m.sync_enabled,
    m.sync_status
FROM sys_dept_ou_mapping m
JOIN sys_dept d ON m.dept_id = d.id
ORDER BY m.created_at DESC
LIMIT 10;

-- 批量插入示例（为整个部门树创建映射）
-- 需要先执行上面的查询获取正确的ID

-- 1. 湖北分公司
-- INSERT INTO sys_dept_ou_mapping (id, dept_id, ad_config_id, ou_dn, ou_name, parent_ou_dn, sync_enabled, sync_status, created_at, updated_at)
-- VALUES (gen_random_uuid(), 'HUBEI_BRANCH_ID', 'AD_CONFIG_ID', 'OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn', '湖北分公司', 'OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn', true, 'synced', NOW(), NOW());

-- 2. 分公司本部
-- INSERT INTO sys_dept_ou_mapping (id, dept_id, ad_config_id, ou_dn, ou_name, parent_ou_dn, sync_enabled, sync_status, created_at, updated_at)
-- VALUES (gen_random_uuid(), 'BRANCH_HQ_ID', 'AD_CONFIG_ID', 'OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn', '分公司本部', 'OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn', true, 'synced', NOW(), NOW());

-- 3. 科技创新部
-- INSERT INTO sys_dept_ou_mapping (id, dept_id, ad_config_id, ou_dn, ou_name, parent_ou_dn, sync_enabled, sync_status, created_at, updated_at)
-- VALUES (gen_random_uuid(), 'TECH_INNOVATION_ID', 'AD_CONFIG_ID', 'OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn', '科技创新部', 'OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn', true, 'synced', NOW(), NOW());

-- 4. 基础运维科
-- INSERT INTO sys_dept_ou_mapping (id, dept_id, ad_config_id, ou_dn, ou_name, parent_ou_dn, sync_enabled, sync_status, created_at, updated_at)
-- VALUES (gen_random_uuid(), 'BASIC_OPS_ID', 'AD_CONFIG_ID', 'OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn', '基础运维科', 'OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn', true, 'synced', NOW(), NOW());