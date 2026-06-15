-- 修复 VDI 菜单组件路径
-- 删除旧的 VDI 菜单记录
DELETE FROM sys_menu WHERE id LIKE '770e8400%';

-- 重新插入 VDI 菜单（使用正确的组件路径）
-- 虚拟机管理一级菜单
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('770e8400-e29b-41d4-a716-446655440001', '虚拟机管理', NULL, 5, 'vdi', NULL, 'M', '1', '0', 'vdi:visit', 'CloudServerOutlined', '虚拟机管理目录', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 虚拟机列表菜单
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('770e8400-e29b-41d4-a716-446655440002', '虚拟机列表', '770e8400-e29b-41d4-a716-446655440001', 1, 'vdi/vm', 'vdi/VirtualMachineList/index', 'C', '1', '0', 'vdi:vm:list', 'DesktopOutlined', '虚拟机列表菜单', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 虚拟机列表按钮
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('770e8400-e29b-41d4-a716-446655440003', '虚拟机查询', '770e8400-e29b-41d4-a716-446655440002', 1, '', '', 'F', '1', '0', 'vdi:vm:query', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440004', '虚拟机新增', '770e8400-e29b-41d4-a716-446655440002', 2, '', '', 'F', '1', '0', 'vdi:vm:add', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440005', '虚拟机修改', '770e8400-e29b-41d4-a716-446655440002', 3, '', '', 'F', '1', '0', 'vdi:vm:edit', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440006', '虚拟机删除', '770e8400-e29b-41d4-a716-446655440002', 4, '', '', 'F', '1', '0', 'vdi:vm:remove', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440007', '虚拟机操作', '770e8400-e29b-41d4-a716-446655440002', 5, '', '', 'F', '1', '0', 'vdi:vm:operate', '#', '开关机、重启等操作', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440008', '配置IP', '770e8400-e29b-41d4-a716-446655440002', 6, '', '', 'F', '1', '0', 'vdi:vm:config', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440009', '重命名', '770e8400-e29b-41d4-a716-446655440002', 7, '', '', 'F', '1', '0', 'vdi:vm:rename', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440010', '绑定用户', '770e8400-e29b-41d4-a716-446655440002', 8, '', '', 'F', '1', '0', 'vdi:vm:bind', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440011', '同步状态', '770e8400-e29b-41d4-a716-446655440002', 9, '', '', 'F', '1', '0', 'vdi:vm:sync', '#', '从VDI服务器同步虚拟机状态', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 虚拟机详情菜单（隐藏，用于详情页访问）
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('770e8400-e29b-41d4-a716-446655440012', '虚拟机详情', '770e8400-e29b-41d4-a716-446655440001', 2, 'vdi/vm/:id', 'vdi/VirtualMachineDetail/index', 'C', '0', '0', 'vdi:vm:view', '#', '虚拟机详情页（含账号管理）', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 虚拟机详情按钮
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('770e8400-e29b-41d4-a716-446655440013', '账号查询', '770e8400-e29b-41d4-a716-446655440012', 1, '', '', 'F', '1', '0', 'vdi:account:list', '#', '查看VM账号列表', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440014', '创建账号', '770e8400-e29b-41d4-a716-446655440012', 2, '', '', 'F', '1', '0', 'vdi:account:create', '#', '在VM内创建用户账号', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440015', '重置密码', '770e8400-e29b-41d4-a716-446655440012', 3, '', '', 'F', '1', '0', 'vdi:account:reset', '#', '重置VM用户密码', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440016', '删除账号', '770e8400-e29b-41d4-a716-446655440012', 4, '', '', 'F', '1', '0', 'vdi:account:delete', '#', '删除VM用户账号', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440017', '打开终端', '770e8400-e29b-41d4-a716-446655440012', 5, '', '', 'F', '1', '0', 'vdi:vm:console', '#', '打开VM网页终端', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- VDI服务器配置菜单
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('770e8400-e29b-41d4-a716-446655440018', 'VDI服务器配置', '770e8400-e29b-41d4-a716-446655440001', 3, 'vdi/servers', 'vdi/VDIServerConfig/index', 'C', '1', '0', 'vdi:admin', 'SettingOutlined', 'VDI服务器配置菜单', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- VDI服务器配置按钮
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('770e8400-e29b-41d4-a716-446655440019', '服务器查询', '770e8400-e29b-41d4-a716-446655440018', 1, '', '', 'F', '1', '0', 'vdi:server:query', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440020', '服务器新增', '770e8400-e29b-41d4-a716-446655440018', 2, '', '', 'F', '1', '0', 'vdi:server:add', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440021', '服务器修改', '770e8400-e29b-41d4-a716-446655440018', 3, '', '', 'F', '1', '0', 'vdi:server:edit', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440022', '服务器删除', '770e8400-e29b-41d4-a716-446655440018', 4, '', '', 'F', '1', '0', 'vdi:server:remove', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440023', '测试连接', '770e8400-e29b-41d4-a716-446655440018', 5, '', '', 'F', '1', '0', 'vdi:server:test', '#', '测试VDI服务器连接', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;
