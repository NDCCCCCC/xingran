-- 诊断查询：查找可能有路径问题的菜单
-- 查找父菜单是目录(menu_type='M')且有path，子菜单path包含父path前缀的情况

SELECT
    p.id as parent_id,
    p.menu_name as parent_name,
    p.path as parent_path,
    p.menu_type as parent_type,
    c.id as child_id,
    c.menu_name as child_name,
    c.path as child_path,
    c.menu_type as child_type,
    CASE
        WHEN c.path LIKE p.path || '/%' THEN 'POTENTIAL ISSUE'
        ELSE 'OK'
    END as status
FROM sys_menu p
JOIN sys_menu c ON c.parent_id = p.id
WHERE p.menu_type = 'M'  -- 父菜单是目录
  AND p.path IS NOT NULL  -- 父菜单有路径
  AND p.path != ''  -- 父菜单路径不为空
  AND c.menu_type = 'C'  -- 子菜单是菜单项
  AND c.path LIKE p.path || '/%'  -- 子菜单路径包含父路径前缀
ORDER BY p.menu_name, c.order_num;
