-- ========================================
-- 菜单路径和组件路径统一迁移
-- ========================================
-- 目的：统一数据库中的菜单路径和组件路径，与前端页面文件结构保持一致
-- 约定：component 字段不包含 "pages/" 前缀，前端在加载组件时自动添加
-- 日期：2026-01-24
-- ========================================

-- 设置搜索路径
SET search_path TO public;

-- ========================================
-- 第一步：全局去掉 component 字段中的 pages/ 前缀
-- ========================================
-- 将所有带 pages/ 前缀的组件路径去掉前缀
UPDATE sys_menu
SET component = SUBSTRING(component FROM 7)  -- 去掉 'pages/' 前缀（7个字符）
WHERE component LIKE 'pages/%'
  AND menu_type = 'C';  -- 只修改菜单类型

-- ========================================
-- 第二步：修复仪表盘路径（最关键的修复）
-- ========================================
-- 仪表盘路径应该是 dashboard，组件是 dashboard-system/index
UPDATE sys_menu
SET path = 'dashboard',
    component = 'dashboard-system/index'
WHERE menu_name = '仪表盘'
  AND (path IS NULL OR path = '' OR path = 'dashboard-system');

-- ========================================
-- 第三步：修复监控模块组件路径
-- ========================================
-- 监控仪表盘
UPDATE sys_menu
SET component = 'monitor/dashboard/index'
WHERE menu_name = '监控仪表盘' AND component NOT LIKE 'monitor/dashboard/%';

-- 服务监控
UPDATE sys_menu
SET component = 'monitor/server/index'
WHERE menu_name = '服务监控' AND component NOT LIKE 'monitor/server/%';

-- 缓存管理
UPDATE sys_menu
SET component = 'monitor/cache/index'
WHERE menu_name = '缓存管理' AND component NOT LIKE 'monitor/cache/%';

-- 定时任务
UPDATE sys_menu
SET component = 'monitor/job/index'
WHERE menu_name = '定时任务' AND component NOT LIKE 'monitor/job/%';

-- 日志管理
UPDATE sys_menu
SET component = 'monitor/logs/index'
WHERE menu_name = '日志管理' AND component NOT LIKE 'monitor/logs/%';

-- ========================================
-- 第四步：修复系统管理模块组件路径
-- ========================================
-- 用户管理
UPDATE sys_menu
SET component = 'system/user/index'
WHERE menu_name = '用户管理' AND component NOT LIKE 'system/user/%';

-- 角色管理
UPDATE sys_menu
SET component = 'system/role/index'
WHERE menu_name = '角色管理' AND component NOT LIKE 'system/role/%';

-- 菜单管理
UPDATE sys_menu
SET component = 'system/menu/index'
WHERE menu_name = '菜单管理' AND component NOT LIKE 'system/menu/%';

-- 部门管理
UPDATE sys_menu
SET component = 'system/dept/index'
WHERE menu_name = '部门管理' AND component NOT LIKE 'system/dept/%';

-- 岗位管理
UPDATE sys_menu
SET component = 'system/post/index'
WHERE menu_name = '岗位管理' AND component NOT LIKE 'system/post/%';

-- 字典管理
UPDATE sys_menu
SET component = 'system/dict/index'
WHERE menu_name = '字典管理' AND component NOT LIKE 'system/dict/%';

-- 参数配置
UPDATE sys_menu
SET component = 'system/config/index'
WHERE menu_name = '参数配置' AND component NOT LIKE 'system/config/%';

-- 通知公告
UPDATE sys_menu
SET component = 'system/notice/index'
WHERE menu_name = '通知公告' AND component NOT LIKE 'system/notice/%';

-- 背景图查询（验证码背景）
UPDATE sys_menu
SET path = 'system/captcha-background',
    component = 'system/captcha-background/index'
WHERE menu_name = '背景图查询' AND component NOT LIKE 'system/captcha-background/%';

-- 系统设置
UPDATE sys_menu
SET component = 'system/settings-page/index'
WHERE menu_name = '系统设置' AND component NOT LIKE 'system/settings-page/%';

-- ========================================
-- 第五步：修复运维管理模块路径
-- ========================================
-- 楼宇管理
UPDATE sys_menu
SET path = 'operations/buildings',
    component = 'operations/buildings/index'
WHERE menu_name = '楼宇管理' AND path = 'buildings';

-- 楼层管理
UPDATE sys_menu
SET path = 'operations/floors',
    component = 'operations/floors/index'
WHERE menu_name = '楼层管理' AND path = 'floors';

-- 机房管理
UPDATE sys_menu
SET path = 'operations/server-rooms',
    component = 'operations/server-rooms/index'
WHERE menu_name = '机房管理' AND path = 'server-rooms';

-- 工位管理
UPDATE sys_menu
SET path = 'operations/workstations',
    component = 'operations/workstations/index'
WHERE menu_name = '工位管理' AND path = 'workstations';

-- 楼宇空间
UPDATE sys_menu
SET path = 'operations/building-spaces',
    component = 'operations/building-spaces/index'
WHERE menu_name = '楼宇空间' AND (path = 'ops/building-spaces' OR path = 'building-spaces');

-- 楼宇空间3D
UPDATE sys_menu
SET path = 'operations/building-spaces-3d',
    component = 'operations/building-spaces-3d/index'
WHERE menu_name = '楼宇空间3D' AND (path = 'ops/building-spaces-3d' OR path = 'building-spaces-3d');

-- 机房设备管理
UPDATE sys_menu
SET path = 'operations/room-devices',
    component = 'operations/room-devices/index'
WHERE menu_name = '机房设备管理' AND path = 'room-devices';

-- 信息点管理
UPDATE sys_menu
SET path = 'operations/info-points',
    component = 'operations/info-points/index'
WHERE menu_name = '信息点管理' AND path = 'info-points';

-- 专线管理
UPDATE sys_menu
SET path = 'operations/dedicated-lines',
    component = 'operations/dedicated-lines/index'
WHERE menu_name = '专线管理' AND path = 'dedicated-lines';

-- ========================================
-- 第六步：修复网络设备模块组件路径
-- ========================================
-- 设备发现（使用复数形式 discoveries）
UPDATE sys_menu
SET component = 'network/discoveries/index'
WHERE menu_name = '设备发现' AND path = 'network/discoveries';

-- 设备管理（使用复数形式 devices）
UPDATE sys_menu
SET component = 'network/devices/index'
WHERE menu_name = '设备管理' AND path = 'network/devices';

-- 端口状态（使用复数形式 ports）
UPDATE sys_menu
SET component = 'network/ports/index'
WHERE menu_name = '端口状态' AND path = 'network/ports';

-- 配置模板（使用复数形式 templates）
UPDATE sys_menu
SET component = 'network/templates/index'
WHERE menu_name = '配置模板' AND path = 'network/templates';

-- 授权凭证（使用复数形式 credentials）
UPDATE sys_menu
SET component = 'network/credentials/index'
WHERE menu_name = '授权凭证' AND path = 'network/credentials';

-- 配置执行（使用复数形式 executions）
UPDATE sys_menu
SET component = 'network/executions/index'
WHERE menu_name = '配置执行' AND path = 'network/executions';

-- 配置备份（使用复数形式 backups）
UPDATE sys_menu
SET component = 'network/backups/index'
WHERE menu_name = '配置备份' AND path = 'network/backups';

-- 命令分发
UPDATE sys_menu
SET component = 'network/command/index'
WHERE menu_name = '命令分发' AND path = 'network/command';

-- MAC地址
UPDATE sys_menu
SET component = 'network/mac/index'
WHERE menu_name = 'MAC地址' AND path = 'network/mac';

-- ========================================
-- 第七步：修复工单模块路径
-- ========================================
-- 工单管理
UPDATE sys_menu
SET path = 'workorder/orders',
    component = 'workorder/orders/index'
WHERE menu_name = '工单管理' AND path = 'orders';

-- 工单分类
UPDATE sys_menu
SET path = 'workorder/categories',
    component = 'workorder/categories/index'
WHERE menu_name = '工单分类' AND path = 'categories';

-- 工单统计
UPDATE sys_menu
SET path = 'workorder/statistics',
    component = 'workorder/statistics/index'
WHERE menu_name = '工单统计' AND path = 'statistics';

-- 周期性工单
UPDATE sys_menu
SET path = 'workorder/periodic/templates',
    component = 'workorder/periodic/templates/index'
WHERE menu_name = '周期性工单' AND path = 'periodic/templates';

-- ========================================
-- 第八步：修复值班管理模块路径
-- ========================================
-- 排班管理
UPDATE sys_menu
SET path = 'duty/schedules',
    component = 'duty/schedules/index'
WHERE menu_name = '排班管理' AND path = 'schedules';

-- 值班池管理
UPDATE sys_menu
SET path = 'duty/pools',
    component = 'duty/pools/index'
WHERE menu_name = '值班池管理' AND path = 'pools';

-- 值班配置
UPDATE sys_menu
SET path = 'duty/config',
    component = 'duty/config/index'
WHERE menu_name = '值班配置' AND path = 'config';

-- 我的值班
UPDATE sys_menu
SET path = 'duty/my-duty',
    component = 'duty/my-duty/index'
WHERE menu_name = '我的值班' AND path = 'my-duty';

-- 节假日管理
UPDATE sys_menu
SET path = 'duty/holidays',
    component = 'duty/holidays/index'
WHERE menu_name = '节假日管理' AND path = 'holidays';

-- ========================================
-- 第九步：修复AD域管理模块路径
-- ========================================
-- AD配置管理
UPDATE sys_menu
SET path = 'ad-domain/configs',
    component = 'ad-domain/configs/index'
WHERE menu_name = 'AD配置管理' AND path = 'configs';

-- AD用户管理
UPDATE sys_menu
SET path = 'ad-domain/users',
    component = 'ad-domain/users/index'
WHERE menu_name = 'AD用户管理' AND path = 'users';

-- 用户组管理
UPDATE sys_menu
SET path = 'ad-domain/groups',
    component = 'ad-domain/groups/index'
WHERE menu_name = '用户组管理' AND path = 'groups';

-- OU组织单位
UPDATE sys_menu
SET path = 'ad-domain/ous',
    component = 'ad-domain/ous/index'
WHERE menu_name = 'OU组织单位' AND path = 'ous';

-- 同步日志
UPDATE sys_menu
SET path = 'ad-domain/logs',
    component = 'ad-domain/logs/index'
WHERE menu_name = '同步日志' AND path = 'logs';

-- ========================================
-- 第十步：修复知识库和特殊页面组件路径
-- ========================================
-- 知识库文章
UPDATE sys_menu
SET component = 'knowledge/articles/index'
WHERE menu_name = '知识库文章' AND component NOT LIKE 'knowledge/articles/%';

-- 知识库查看
UPDATE sys_menu
SET component = 'knowledge/view/index'
WHERE menu_name = '知识库查看' AND component NOT LIKE 'knowledge/view/%';

-- 个人中心
UPDATE sys_menu
SET component = 'profile/index'
WHERE menu_name = '个人中心' AND component NOT LIKE 'profile/%';

-- 我的通知
UPDATE sys_menu
SET component = 'my-notices/index'
WHERE menu_name = '我的通知' AND component NOT LIKE 'my-notices/%';

-- 用户设置
UPDATE sys_menu
SET component = 'settings/index'
WHERE menu_name = '用户设置' AND component NOT LIKE 'settings/%';

-- ========================================
-- 验证查询
-- ========================================
-- 检查是否有空路径的菜单
-- SELECT id, menu_name, path, component FROM sys_menu WHERE (path IS NULL OR path = '') AND menu_type = 'C';

-- 检查组件路径是否仍包含 pages/ 前缀
-- SELECT id, menu_name, component FROM sys_menu WHERE component LIKE 'pages/%' AND menu_type = 'C';

-- 显示所有菜单类型的路径和组件（用于验证）
-- SELECT id, menu_name, path, component FROM sys_menu WHERE menu_type = 'C' ORDER BY menu_name;
