-- 楼宇空间3D可视化菜单配置SQL
-- 执行前请先查询运维管理父菜单ID

-- 1. 查询运维管理父菜单ID
-- SELECT id, menu_name FROM sys_menu WHERE menu_name LIKE '%运维%' LIMIT 5;

-- 2. 插入楼宇空间管理菜单
-- 请将下面的 @PARENT_ID 替换为实际查询到的运维管理父菜单ID
-- 例如：如果父菜单ID是 2000，则 (2000, 6, ...)

INSERT INTO sys_menu (
  menu_name,
  parent_id,
  order_num,
  path,
  component,
  perms,
  menu_type,
  status,
  visible,
  icon,
  create_by,
  create_time,
  update_by,
  update_time,
  remark
) VALUES (
  '楼宇空间',
  (SELECT id FROM (SELECT id FROM sys_menu WHERE menu_name LIKE '%运维%' AND menu_type = 'M' LIMIT 1) AS parent),
  1,
  'building-spaces',
  'operations/building-spaces/index',
  'ops:building:spaces:list',
  'C',
  '0',
  '1',
  'apartment',
  'admin',
  NOW(),
  'admin',
  NOW(),
  '楼宇空间3D可视化页面，展示楼宇、楼层、工位的3D堆叠效果'
);

-- 3. 验证插入结果
-- SELECT id, menu_name, path, component FROM sys_menu WHERE path = 'building-spaces';

-- 注意事项：
-- 1. 菜单类型: 'M'=目录, 'C'=菜单, 'F'=按钮
-- 2. 状态: '0'=正常, '1'=停用
-- 3. 是否可见: '1'=显示, '0'=隐藏
-- 4. perms: 权限标识，格式为 模块:功能:操作
-- 5. 如果需要更多细粒度权限控制，可以添加按钮类型的子菜单
