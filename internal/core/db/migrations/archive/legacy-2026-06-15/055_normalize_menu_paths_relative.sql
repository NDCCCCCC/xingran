-- ========================================
-- 菜单路径标准化：统一使用相对路径
-- ========================================
-- 规则：每个菜单的 path 字段只存储当前层级的路径段
--       不包含父路径前缀，不包含前导斜杠
--       通过父子关系在运行时拼接完整路径
--
-- 示例：
--   父菜单（运维管理）: path = "ops"
--   子菜单（工单管理）: path = "workorder"  → 完整路径: ops/workorder
--   孙菜单（工单列表）: path = "orders"     → 完整路径: ops/workorder/orders
-- ========================================

-- 设置搜索路径
SET search_path TO public;

-- ========================================
-- 第一步：统一使用相对路径
-- ========================================

-- 1.1 系统管理模块
-- 系统管理下的子菜单：system/xxx → xxx
UPDATE sys_menu
SET path = 'user'
WHERE menu_name = '用户管理' AND path = 'system/user';

UPDATE sys_menu
SET path = 'role'
WHERE menu_name = '角色管理' AND path = 'system/role';

UPDATE sys_menu
SET path = 'menu'
WHERE menu_name = '菜单管理' AND path = 'system/menu';

UPDATE sys_menu
SET path = 'dept'
WHERE menu_name = '部门管理' AND path = 'system/dept';

UPDATE sys_menu
SET path = 'post'
WHERE menu_name = '岗位管理' AND path = 'system/post';

UPDATE sys_menu
SET path = 'dict'
WHERE menu_name = '字典管理' AND path = 'system/dict';

UPDATE sys_menu
SET path = 'config'
WHERE menu_name = '参数配置' AND path = 'system/config';

UPDATE sys_menu
SET path = 'notice'
WHERE menu_name = '通知公告' AND path = 'system/notice';

UPDATE sys_menu
SET path = 'captcha-background'
WHERE menu_name = '背景图查询' AND path = 'background';

UPDATE sys_menu
SET path = 'settings-page'
WHERE menu_name = '系统设置' AND path = 'system/settings-page';

-- 1.2 系统监控模块
-- 系统监控下的子菜单：monitor/xxx → xxx
UPDATE sys_menu
SET path = 'dashboard'
WHERE menu_name = '监控仪表盘' AND path = 'monitor/dashboard';

UPDATE sys_menu
SET path = 'server'
WHERE menu_name = '服务监控' AND path = 'monitor/server';

UPDATE sys_menu
SET path = 'cache'
WHERE menu_name = '缓存管理' AND path = 'monitor/cache';

UPDATE sys_menu
SET path = 'job'
WHERE menu_name = '定时任务' AND path = 'monitor/job';

UPDATE sys_menu
SET path = 'logs'
WHERE menu_name = '日志管理' AND path = 'monitor/logs';

UPDATE sys_menu
SET path = 'operLog'
WHERE menu_name = '操作日志' AND path = 'monitor/operLog';

UPDATE sys_menu
SET path = 'loginLog'
WHERE menu_name = '登录日志' AND path = 'monitor/loginLog';

-- 1.3 网络设备管理模块
-- 网络设备管理下的子菜单：network/xxx → xxx
UPDATE sys_menu
SET path = 'discoveries'
WHERE menu_name = '设备发现' AND path = 'network/discoveries';

UPDATE sys_menu
SET path = 'devices'
WHERE menu_name = '设备管理' AND path = 'network/devices';

UPDATE sys_menu
SET path = 'ports'
WHERE menu_name = '端口状态' AND path = 'network/ports';

UPDATE sys_menu
SET path = 'templates'
WHERE menu_name = '配置模板' AND path = 'network/templates';

UPDATE sys_menu
SET path = 'credentials'
WHERE menu_name = '授权凭证' AND path = 'network/credentials';

UPDATE sys_menu
SET path = 'executions'
WHERE menu_name = '配置执行' AND path = 'network/executions';

UPDATE sys_menu
SET path = 'backups'
WHERE menu_name = '配置备份' AND path = 'network/backups';

UPDATE sys_menu
SET path = 'command'
WHERE menu_name = '命令分发' AND path = 'network/command';

UPDATE sys_menu
SET path = 'mac'
WHERE menu_name = 'MAC地址' AND path = 'network/mac';

-- 1.4 工单模块
-- 工单系统下的子菜单：workorder/xxx → xxx
UPDATE sys_menu
SET path = 'orders'
WHERE menu_name = '工单管理' AND path = 'workorder/orders';

UPDATE sys_menu
SET path = 'categories'
WHERE menu_name = '工单分类' AND path = 'workorder/categories';

UPDATE sys_menu
SET path = 'statistics'
WHERE menu_name = '工单统计' AND path = 'workorder/statistics';

-- 1.5 值班管理模块
-- 值班管理下的子菜单：duty/xxx → xxx
UPDATE sys_menu
SET path = 'pools'
WHERE menu_name = '值班池管理' AND path = 'duty/pools';

UPDATE sys_menu
SET path = 'schedules'
WHERE menu_name = '排班管理' AND path = 'duty/schedules';

UPDATE sys_menu
SET path = 'config'
WHERE menu_name = '值班配置' AND path = 'duty/config';

UPDATE sys_menu
SET path = 'my-duty'
WHERE menu_name = '我的值班' AND path = 'duty/my-duty';

UPDATE sys_menu
SET path = 'holidays'
WHERE menu_name = '节假日管理' AND path = 'duty/holidays';

UPDATE sys_menu
SET path = 'management'
WHERE (menu_name = '配置' OR menu_name = '值班管理')
  AND path = 'duty/management';

-- 1.6 AD域管理模块
-- AD域管理下的子菜单：ad-domain/xxx → xxx
UPDATE sys_menu
SET path = 'configs'
WHERE menu_name = 'AD配置管理' AND path = 'ad-domain/configs';

UPDATE sys_menu
SET path = 'users'
WHERE menu_name = 'AD用户管理' AND path = 'ad-domain/users';

UPDATE sys_menu
SET path = 'groups'
WHERE menu_name = '用户组管理' AND path = 'ad-domain/groups';

UPDATE sys_menu
SET path = 'ous'
WHERE menu_name = 'OU组织单位' AND path = 'ad-domain/ous';

UPDATE sys_menu
SET path = 'logs'
WHERE menu_name = '同步日志' AND path = 'ad-domain/logs';

-- ========================================
-- 第二步：更新组件路径为相对路径（去掉 pages/ 前缀）
-- ========================================

-- 全局去掉 component 字段中的 pages/ 前缀
UPDATE sys_menu
SET component = SUBSTRING(component FROM 7)
WHERE component LIKE 'pages/%'
  AND menu_type = 'C';

-- ========================================
-- 第三步：验证结果
-- ========================================

-- 检查是否还有路径重复的情况
WITH path_check AS (
  SELECT
    m1.id,
    m1.menu_name,
    m1.path,
    m2.path as parent_path,
    CASE
      WHEN m2.path IS NOT NULL AND m1.path != ''
      THEN (m2.path || '/' || m1.path)
      ELSE m1.path
    END as full_path
  FROM sys_menu m1
  LEFT JOIN sys_menu m2 ON m1.parent_id = m2.id
  WHERE m1.menu_type IN ('M', 'C')
)
SELECT
  full_path,
  CASE WHEN full_path LIKE '%/%/%' THEN '⚠️  可能重复'
       ELSE '✓ 正常'
  END as status,
  COUNT(*) as count
FROM path_check
GROUP BY full_path, status
HAVING COUNT(*) > 1 OR full_path LIKE '%/%/%'
ORDER BY full_path;

-- 显示所有菜单的路径（用于验证）
-- SELECT
--   CASE WHEN parent_id IS NULL THEN '一级菜单' ELSE '子菜单' END as level,
--   menu_name,
--   path,
--   component
-- FROM sys_menu
-- WHERE menu_type IN ('M', 'C')
-- ORDER BY order_num;
