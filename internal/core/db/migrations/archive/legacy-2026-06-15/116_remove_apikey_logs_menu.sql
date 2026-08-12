-- 移除API密钥管理的"使用日志"菜单项
-- 原因：LogsModal 是一个 Modal 弹窗组件，不是独立页面
-- 它应该在 API 密钥列表页面中通过按钮打开，而不是作为单独的菜单项

-- 删除使用日志菜单及其权限关联
DELETE FROM sys_role_menu
WHERE menu_id IN (
    SELECT id FROM sys_menu WHERE menu_name = '使用日志'
);

DELETE FROM sys_menu
WHERE menu_name = '使用日志';

-- 验证结果
SELECT '使用日志菜单已移除，因为它是 Modal 组件而非独立页面' AS status;

-- 确认 API密钥管理目录和密钥列表菜单仍然存在
SELECT
    m1.menu_name as "父菜单",
    m2.menu_name as "子菜单",
    m2.path as "路径",
    m2.menu_type as "类型"
FROM sys_menu m1
LEFT JOIN sys_menu m2 ON m2.parent_id = m1.id
WHERE m1.menu_name = 'API密钥管理'
ORDER BY m2.order_num;
