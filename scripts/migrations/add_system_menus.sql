-- ========================================
-- 系统管理模块菜单数据
-- 创建时间: 2024-12-24
-- 说明: 添加系统管理下子模块的菜单和权限
-- ========================================

-- 首先获取系统管理父菜单ID（假设系统管理菜单已存在）
-- 系统管理菜单ID变量：$system_menu_id

-- ========================================
-- 1. 字典管理模块
-- ========================================

-- 字典管理菜单（一级）
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '字典管理',
   (SELECT id FROM sys_menu WHERE menu_name = '系统管理' LIMIT 1),
   6, 'dict', 'system/dict/index', 'C', 1, 0,
   'system:dict:list', 'dict', '字典管理菜单', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (menu_name, parent_id) DO NOTHING;

-- 获取字典管理菜单ID
-- 字典类型查询
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '字典类型查询',
   (SELECT id FROM sys_menu WHERE path = 'dict' AND component = 'system/dict/index' LIMIT 1),
   1, '', '', 'F', 1, 0,
   'system:dict:list', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- 字典类型新增
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '字典类型新增',
   (SELECT id FROM sys_menu WHERE path = 'dict' AND component = 'system/dict/index' LIMIT 1),
   2, '', '', 'F', 1, 0,
   'system:dict:add', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- 字典类型修改
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '字典类型修改',
   (SELECT id FROM sys_menu WHERE path = 'dict' AND component = 'system/dict/index' LIMIT 1),
   3, '', '', 'F', 1, 0,
   'system:dict:edit', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- 字典类型删除
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '字典类型删除',
   (SELECT id FROM sys_menu WHERE path = 'dict' AND component = 'system/dict/index' LIMIT 1),
   4, '', '', 'F', 1, 0,
   'system:dict:remove', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- 字典数据查询
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '字典数据查询',
   (SELECT id FROM sys_menu WHERE path = 'dict' AND component = 'system/dict/index' LIMIT 1),
   5, '', '', 'F', 1, 0,
   'system:dict:list', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- ========================================
-- 2. 参数配置模块
-- ========================================

-- 参数配置菜单（一级）
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '参数设置',
   (SELECT id FROM sys_menu WHERE menu_name = '系统管理' LIMIT 1),
   7, 'config', 'system/config/index', 'C', 1, 0,
   'system:config:list', 'edit', '参数设置菜单', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (menu_name, parent_id) DO NOTHING;

-- 参数查询
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '参数查询',
   (SELECT id FROM sys_menu WHERE path = 'config' AND component = 'system/config/index' LIMIT 1),
   1, '', '', 'F', 1, 0,
   'system:config:list', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- 参数新增
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '参数新增',
   (SELECT id FROM sys_menu WHERE path = 'config' AND component = 'system/config/index' LIMIT 1),
   2, '', '', 'F', 1, 0,
   'system:config:add', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- 参数修改
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '参数修改',
   (SELECT id FROM sys_menu WHERE path = 'config' AND component = 'system/config/index' LIMIT 1),
   3, '', '', 'F', 1, 0,
   'system:config:edit', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- 参数删除
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '参数删除',
   (SELECT id FROM sys_menu WHERE path = 'config' AND component = 'system/config/index' LIMIT 1),
   4, '', '', 'F', 1, 0,
   'system:config:remove', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- ========================================
-- 3. 通知公告模块
-- ========================================

-- 通知公告菜单（一级）
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '通知公告',
   (SELECT id FROM sys_menu WHERE menu_name = '系统管理' LIMIT 1),
   8, 'notice', 'system/notice/index', 'C', 1, 0,
   'system:notice:list', 'message', '通知公告菜单', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (menu_name, parent_id) DO NOTHING;

-- 通知查询
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '通知查询',
   (SELECT id FROM sys_menu WHERE path = 'notice' AND component = 'system/notice/index' LIMIT 1),
   1, '', '', 'F', 1, 0,
   'system:notice:list', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- 通知新增
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '通知新增',
   (SELECT id FROM sys_menu WHERE path = 'notice' AND component = 'system/notice/index' LIMIT 1),
   2, '', '', 'F', 1, 0,
   'system:notice:add', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- 通知修改
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '通知修改',
   (SELECT id FROM sys_menu WHERE path = 'notice' AND component = 'system/notice/index' LIMIT 1),
   3, '', '', 'F', 1, 0,
   'system:notice:edit', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- 通知删除
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '通知删除',
   (SELECT id FROM sys_menu WHERE path = 'notice' AND component = 'system/notice/index' LIMIT 1),
   4, '', '', 'F', 1, 0,
   'system:notice:remove', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- ========================================
-- 4. 工位管理模块
-- ========================================

-- 工位管理菜单（一级）
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '工位管理',
   (SELECT id FROM sys_menu WHERE menu_name = '系统管理' LIMIT 1),
   9, 'workstation', 'system/workstation/index', 'C', 1, 0,
   'system:workstation:list', 'apartment', '工位管理菜单', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (menu_name, parent_id) DO NOTHING;

-- 工位查询
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '工位查询',
   (SELECT id FROM sys_menu WHERE path = 'workstation' AND component = 'system/workstation/index' LIMIT 1),
   1, '', '', 'F', 1, 0,
   'system:workstation:list', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- 工位新增
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '工位新增',
   (SELECT id FROM sys_menu WHERE path = 'workstation' AND component = 'system/workstation/index' LIMIT 1),
   2, '', '', 'F', 1, 0,
   'system:workstation:add', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- 工位修改
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '工位修改',
   (SELECT id FROM sys_menu WHERE path = 'workstation' AND component = 'system/workstation/index' LIMIT 1),
   3, '', '', 'F', 1, 0,
   'system:workstation:edit', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- 工位删除
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '工位删除',
   (SELECT id FROM sys_menu WHERE path = 'workstation' AND component = 'system/workstation/index' LIMIT 1),
   4, '', '', 'F', 1, 0,
   'system:workstation:remove', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- ========================================
-- 5. 日志管理模块（系统监控下）
-- ========================================

-- 操作日志菜单（一级）
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '操作日志',
   (SELECT id FROM sys_menu WHERE menu_name = '系统监控' LIMIT 1),
   1, 'operLog', 'monitor/operLog/index', 'C', 1, 0,
   'monitor:operlog:list', 'form', '操作日志菜单', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (menu_name, parent_id) DO NOTHING;

-- 操作日志查询
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '操作日志查询',
   (SELECT id FROM sys_menu WHERE path = 'operLog' AND component = 'monitor/operLog/index' LIMIT 1),
   1, '', '', 'F', 1, 0,
   'monitor:operlog:list', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- 操作日志删除
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '操作日志删除',
   (SELECT id FROM sys_menu WHERE path = 'operLog' AND component = 'monitor/operLog/index' LIMIT 1),
   2, '', '', 'F', 1, 0,
   'monitor:operlog:remove', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- 登录日志菜单（一级）
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '登录日志',
   (SELECT id FROM sys_menu WHERE menu_name = '系统监控' LIMIT 1),
   2, 'loginLog', 'monitor/loginLog/index', 'C', 1, 0,
   'monitor:loginlog:list', 'login', '登录日志菜单', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (menu_name, parent_id) DO NOTHING;

-- 登录日志查询
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '登录日志查询',
   (SELECT id FROM sys_menu WHERE path = 'loginLog' AND component = 'monitor/loginLog/index' LIMIT 1),
   1, '', '', 'F', 1, 0,
   'monitor:loginlog:list', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;

-- 登录日志删除
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
VALUES
  (gen_random_uuid(), '登录日志删除',
   (SELECT id FROM sys_menu WHERE path = 'loginLog' AND component = 'monitor/loginLog/index' LIMIT 1),
   2, '', '', 'F', 1, 0,
   'monitor:loginlog:remove', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;
