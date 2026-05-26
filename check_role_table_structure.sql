-- 检查 sys_role 表结构
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_name = 'sys_role'
ORDER BY ordinal_position;

-- 查看管理员角色记录（尝试不同的列名）
SELECT * FROM sys_role WHERE role_key = 'admin' LIMIT 1;

-- 查看 sys_role_menu 表结构
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_name = 'sys_role_menu'
ORDER BY ordinal_position;
