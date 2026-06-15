-- CAD平面图编辑器菜单配置
-- 添加到运维管理模块

-- CAD平面图编辑器菜单（使用子查询动态获取运维管理菜单ID）
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at)
SELECT
    '550e8400-e29b-41d4-a716-446655441200',
    '平面图编辑',
    id,
    10,
    'floor-plan-editor',
    'operations/floor-plan-editor/index',
    'C',
    '1',
    '0',
    'ops:floor-plan:list',
    'EditOutlined',
    'CAD风格平面图编辑器，支持墙体、门、工位的编辑',
    NOW(),
    NOW(),
    NULL
FROM sys_menu
WHERE menu_name = '运维管理' AND parent_id IS NULL
ON CONFLICT (id) DO NOTHING;

-- CAD平面图编辑器按钮权限
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('550e8400-e29b-41d4-a716-446655441201', '查询平面图', '550e8400-e29b-41d4-a716-446655441200', 1, '', '', 'F', '1', '0', 'ops:floor-plan:query', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441202', '编辑平面图', '550e8400-e29b-41d4-a716-446655441200', 2, '', '', 'F', '1', '0', 'ops:floor-plan:edit', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441203', '保存平面图', '550e8400-e29b-41d4-a716-446655441200', 3, '', '', 'F', '1', '0', 'ops:floor-plan:save', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441204', '墙体管理', '550e8400-e29b-41d4-a716-446655441200', 4, '', '', 'F', '1', '0', 'ops:walls:list', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441205', '墙体新增', '550e8400-e29b-41d4-a716-446655441200', 5, '', '', 'F', '1', '0', 'ops:walls:add', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441206', '墙体编辑', '550e8400-e29b-41d4-a716-446655441200', 6, '', '', 'F', '1', '0', 'ops:walls:edit', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441207', '墙体删除', '550e8400-e29b-41d4-a716-446655441200', 7, '', '', 'F', '1', '0', 'ops:walls:delete', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441208', '门管理', '550e8400-e29b-41d4-a716-446655441200', 8, '', '', 'F', '1', '0', 'ops:doors:list', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441209', '门新增', '550e8400-e29b-41d4-a716-446655441200', 9, '', '', 'F', '1', '0', 'ops:doors:add', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441210', '门编辑', '550e8400-e29b-41d4-a716-446655441200', 10, '', '', 'F', '1', '0', 'ops:doors:edit', '#', '', NOW(), NOW(), NULL),
('550e8400-e29b-41d4-a716-446655441211', '门删除', '550e8400-e29b-41d4-a716-446655441200', 11, '', '', 'F', '1', '0', 'ops:doors:delete', '#', '', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 为管理员角色分配权限（使用子查询动态获取管理员角色ID）
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT r.id, m.id
FROM sys_role r
CROSS JOIN sys_menu m
WHERE r.role_key = 'admin'
  AND r.status = '0'
  AND m.id IN (
    '550e8400-e29b-41d4-a716-446655441200',
    '550e8400-e29b-41d4-a716-446655441201',
    '550e8400-e29b-41d4-a716-446655441202',
    '550e8400-e29b-41d4-a716-446655441203',
    '550e8400-e29b-41d4-a716-446655441204',
    '550e8400-e29b-41d4-a716-446655441205',
    '550e8400-e29b-41d4-a716-446655441206',
    '550e8400-e29b-41d4-a716-446655441207',
    '550e8400-e29b-41d4-a716-446655441208',
    '550e8400-e29b-41d4-a716-446655441209',
    '550e8400-e29b-41d4-a716-446655441210',
    '550e8400-e29b-41d4-a716-446655441211'
  )
ON CONFLICT DO NOTHING;
