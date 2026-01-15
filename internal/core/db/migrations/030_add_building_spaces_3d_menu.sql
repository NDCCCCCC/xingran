-- 楼宇空间 3D 可视化菜单配置
-- 添加到运维管理模块（与楼宇空间平级）

-- 楼宇空间 3D 可视化菜单
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('550e8400-e29b-41d4-a716-446655441102', '楼宇空间3D', 'c50b5b01-fdbc-4821-a3e8-2e2178339b20', 8, '/ops/building-spaces-3d', 'operations/building-spaces-3d/index', 'C', '1', '0', 'ops:building:spaces:3d:list', 'EnvironmentOutlined', '楼宇空间3D可视化页面，基于百度地图展示湖北省楼宇分布', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 楼宇空间 3D 按钮权限
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('550e8400-e29b-41d4-a716-446655441103', '3D视图查询', '550e8400-e29b-41d4-a716-446655441102', 1, '', '', 'F', '1', '0', 'ops:building:spaces:3d:query', '#', '', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;
