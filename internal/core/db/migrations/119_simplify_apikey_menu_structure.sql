-- 简化API密钥管理菜单结构
-- 删除"API密钥管理"二级目录菜单和"使用日志"子菜单
-- 将"密钥列表"直接作为系统管理下的二级菜单
-- "使用日志"功能通过密钥列表页面的操作按钮访问（Modal形式）

-- 第一步：删除角色权限关联
DELETE FROM sys_role_menu
WHERE menu_id IN (
    SELECT id FROM sys_menu WHERE menu_name IN ('API密钥管理', '使用日志')
);

-- 第二步：删除"使用日志"子菜单
DELETE FROM sys_menu WHERE menu_name = '使用日志';

-- 第三步：删除"API密钥管理"目录菜单
DELETE FROM sys_menu WHERE menu_name = 'API密钥管理';

-- 第四步：更新"密钥列表"菜单
-- 将其父菜单改为系统管理（直接作为系统管理的子菜单）
UPDATE sys_menu
SET
    parent_id = 'd67f4240-f887-481a-b345-94fb36782500',
    path = 'apikeys',
    component = 'system/apikeys/index',
    menu_type = 'C',
    order_num = 11,
    perms = 'system:apikey:list',
    icon = 'KeyOutlined',
    remark = 'API密钥管理页面（含查看、创建、编辑、删除及使用日志）',
    updated_at = NOW()
WHERE menu_name = '密钥列表';

-- 第五步：为管理员角色重新分配权限
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT 'd67f4241-f887-481a-b345-94fb36782500', id
FROM sys_menu
WHERE menu_name = '密钥列表'
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- 验证修复结果
SELECT
    '简化后的菜单结构' as "说明",
    m1.menu_name as "一级菜单",
    m2.menu_name as "二级菜单",
    m2.path as "路径",
    m2.component as "组件",
    m2.menu_type as "类型",
    m2.order_num as "排序"
FROM sys_menu m1
INNER JOIN sys_menu m2 ON m2.parent_id = m1.id
WHERE m1.id = 'd67f4240-f887-481a-b345-94fb36782500'
  AND m2.menu_name = '密钥列表';

-- 检查是否还有其他相关菜单
SELECT
    menu_name,
    path,
    component,
    menu_type,
    parent_id,
    status
FROM sys_menu
WHERE menu_name LIKE '%API%' OR menu_name LIKE '%密钥%' OR menu_name LIKE '%日志%'
ORDER BY order_num;

SELECT '✅ 简化完成：系统管理 > 密钥列表（使用日志功能在页面内）' AS status;
