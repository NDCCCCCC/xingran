-- Phase 32 P2-A4 source-tracking:
--   Original commit: a3032b2e
--   Created: 2026-01-16
--   Note: Conflicts with 029_add_building_coordinates.sql — both share prefix 029. Runner uses Go code ordering, not filename sort; conflict is harmless.

-- 楼宇空间3D可视化菜单配置
-- 添加到运维管理模块

-- 楼宇空间菜单
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('550e8400-e29b-41d4-a716-446655441100', '楼宇空间', 'c50b5b01-fdbc-4821-a3e8-2e2178339b20', 7, '/ops/building-spaces', 'operations/building-spaces/index', 'C', '1', '0', 'ops:building:spaces:list', 'ApartmentOutlined', '楼宇空间3D可视化页面，展示楼宇、楼层、工位的3D堆叠效果', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 楼宇空间按钮权限
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('550e8400-e29b-41d4-a716-446655441101', '楼宇空间查询', '550e8400-e29b-41d4-a716-446655441100', 1, '', '', 'F', '1', '0', 'ops:building:spaces:query', '#', '', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;
