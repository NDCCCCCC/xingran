-- 运维管理模块菜单数据
-- 运维管理菜单
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('550e8400-e29b-41d4-a716-446655441001', '运维管理', NULL, 4, 'ops', 'Layout', 'M', '1', '0', '', 'Control', '运维管理目录', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 楼宇管理
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('550e8400-e29b-41d4-a716-446655441002', '楼宇管理', '550e8400-e29b-41d4-a716-446655441001', 1, 'buildings', 'operations/buildings/index', 'C', '1', '0', 'ops:buildings:list', 'BuildOutlined', '楼宇管理菜单', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 楼宇管理按钮
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('550e8400-e29b-41d4-a716-446655441003', '楼宇查询', '550e8400-e29b-41d4-a716-446655441002', 1, '', '', 'F', '1', '0', 'ops:buildings:query', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441004', '楼宇新增', '550e8400-e29b-41d4-a716-446655441002', 2, '', '', 'F', '1', '0', 'ops:buildings:add', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441005', '楼宇修改', '550e8400-e29b-41d4-a716-446655441002', 3, '', '', 'F', '1', '0', 'ops:buildings:edit', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441006', '楼宇删除', '550e8400-e29b-41d4-a716-446655441002', 4, '', '', 'F', '1', '0', 'ops:buildings:remove', '#', '', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 楼层管理
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('550e8400-e29b-41d4-a716-446655441007', '楼层管理', '550e8400-e29b-41d4-a716-446655441001', 2, 'floors', 'operations/floors/index', 'C', '1', '0', 'ops:floors:list', 'ApartmentOutlined', '楼层管理菜单', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 楼层管理按钮
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('550e8400-e29b-41d4-a716-446655441008', '楼层查询', '550e8400-e29b-41d4-a716-446655441007', 1, '', '', 'F', '1', '0', 'ops:floors:query', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441009', '楼层新增', '550e8400-e29b-41d4-a716-446655441007', 2, '', '', 'F', '1', '0', 'ops:floors:add', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441010', '楼层修改', '550e8400-e29b-41d4-a716-446655441007', 3, '', '', 'F', '1', '0', 'ops:floors:edit', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441011', '楼层删除', '550e8400-e29b-41d4-a716-446655441007', 4, '', '', 'F', '1', '0', 'ops:floors:remove', '#', '', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 工位管理
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('550e8400-e29b-41d4-a716-446655441012', '工位管理', '550e8400-e29b-41d4-a716-446655441001', 3, 'workstations', 'operations/workstations/index', 'C', '1', '0', 'ops:workstations:list', 'DesktopOutlined', '工位管理菜单', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 工位管理按钮
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('550e8400-e29b-41d4-a716-446655441013', '工位查询', '550e8400-e29b-41d4-a716-446655441012', 1, '', '', 'F', '1', '0', 'ops:workstations:query', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441014', '工位新增', '550e8400-e29b-41d4-a716-446655441012', 2, '', '', 'F', '1', '0', 'ops:workstations:add', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441015', '工位修改', '550e8400-e29b-41d4-a716-446655441012', 3, '', '', 'F', '1', '0', 'ops:workstations:edit', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441016', '工位删除', '550e8400-e29b-41d4-a716-446655441012', 4, '', '', 'F', '1', '0', 'ops:workstations:remove', '#', '', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 机房管理
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('550e8400-e29b-41d4-a716-446655441017', '机房管理', '550e8400-e29b-41d4-a716-446655441001', 4, 'server-rooms', 'operations/server-rooms/index', 'C', '1', '0', 'ops:server-rooms:list', 'CloudServerOutlined', '机房管理菜单', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 机房管理按钮
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('550e8400-e29b-41d4-a716-446655441018', '机房查询', '550e8400-e29b-41d4-a716-446655441017', 1, '', '', 'F', '1', '0', 'ops:server-rooms:query', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441019', '机房新增', '550e8400-e29b-41d4-a716-446655441017', 2, '', '', 'F', '1', '0', 'ops:server-rooms:add', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441020', '机房修改', '550e8400-e29b-41d4-a716-446655441017', 3, '', '', 'F', '1', '0', 'ops:server-rooms:edit', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441021', '机房删除', '550e8400-e29b-41d4-a716-446655441017', 4, '', '', 'F', '1', '0', 'ops:server-rooms:remove', '#', '', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 专线管理
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('550e8400-e29b-41d4-a716-446655441022', '专线管理', '550e8400-e29b-41d4-a716-446655441001', 5, 'dedicated-lines', 'operations/dedicated-lines/index', 'C', '1', '0', 'ops:dedicated-lines:list', 'LineChartOutlined', '专线管理菜单', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 专线管理按钮
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('550e8400-e29b-41d4-a716-446655441023', '专线查询', '550e8400-e29b-41d4-a716-446655441022', 1, '', '', 'F', '1', '0', 'ops:dedicated-lines:query', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441024', '专线新增', '550e8400-e29b-41d4-a716-446655441022', 2, '', '', 'F', '1', '0', 'ops:dedicated-lines:add', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441025', '专线修改', '550e8400-e29b-41d4-a716-446655441022', 3, '', '', 'F', '1', '0', 'ops:dedicated-lines:edit', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441026', '专线删除', '550e8400-e29b-41d4-a716-446655441022', 4, '', '', 'F', '1', '0', 'ops:dedicated-lines:remove', '#', '', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 机房设备管理
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('550e8400-e29b-41d4-a716-446655441027', '机房设备管理', '550e8400-e29b-41d4-a716-446655441001', 6, 'room-devices', 'operations/room-devices/index', 'C', '1', '0', 'ops:room-devices:list', 'AppstoreOutlined', '机房设备管理菜单', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 机房设备管理按钮
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('550e8400-e29b-41d4-a716-446655441028', '设备查询', '550e8400-e29b-41d4-a716-446655441027', 1, '', '', 'F', '1', '0', 'ops:room-devices:query', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441029', '设备新增', '550e8400-e29b-41d4-a716-446655441027', 2, '', '', 'F', '1', '0', 'ops:room-devices:add', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441030', '设备修改', '550e8400-e29b-41d4-a716-446655441027', 3, '', '', 'F', '1', '0', 'ops:room-devices:edit', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441031', '设备删除', '550e8400-e29b-41d4-a716-446655441027', 4, '', '', 'F', '1', '0', 'ops:room-devices:remove', '#', '', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;
