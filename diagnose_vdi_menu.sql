-- 检查 VDI 菜单是否插入成功
SELECT id, menu_name, parent_id, menu_type, visible, status, path, component
FROM sys_menu
WHERE id LIKE '770e8400%' OR parent_id IN (
    SELECT id FROM sys_menu WHERE id LIKE '770e8400%'
)
ORDER BY parent_id NULLS LAST, order_num;

-- 检查菜单总数
SELECT COUNT(*) as vdi_menu_count
FROM sys_menu
WHERE id LIKE '770e8400%';

-- 检查是否有其他菜单使用了 vdi 路径
SELECT id, menu_name, path, component
FROM sys_menu
WHERE path LIKE '%vdi%' OR component LIKE '%vdi%';
