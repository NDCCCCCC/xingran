-- 清理孤立的部门OU映射数据
-- 问题：当部门被删除后，sys_dept_ou_mapping 中的映射记录没有级联删除
-- 导致新部门无法映射到已被使用的 OU DN
-- 解决：删除所有 dept_id 不在 sys_dept 表中的映射记录

-- 删除孤立的映射记录（部门不存在）
DELETE FROM sys_dept_ou_mapping
WHERE dept_id NOT IN (
    SELECT id FROM sys_dept WHERE deleted_at IS NULL
);

-- 删除重复的映射记录（保留最新的）
-- PostgreSQL DISTINCT ON 要求 ORDER BY 必须包含所有 DISTINCT ON 列
DELETE FROM sys_dept_ou_mapping
WHERE id NOT IN (
    SELECT DISTINCT ON (dept_id, ad_config_id) id
    FROM sys_dept_ou_mapping
    ORDER BY dept_id, ad_config_id, updated_at DESC
);

-- 添加注释
COMMENT ON TABLE sys_dept_ou_mapping IS '部门-AD域OU映射表（已清理孤立数据）';
