-- 修复 API 密钥菜单显示问题
-- 确保"密钥列表"菜单在侧边栏正确显示

-- 第一步：检查并修复"密钥列表"菜单的状态和可见性
UPDATE sys_menu
SET
    status = 0,        -- 0 = 正常启用
    visible = 1,       -- 1 = 可见
    parent_id = 'd67f4240-f887-481a-b345-94fb36782500',  -- 系统管理
    path = 'apikeys',
    component = 'system/apikeys/index',
    menu_type = 'C',
    order_num = 11,
    updated_at = NOW()
WHERE menu_name = '密钥列表';

-- 第二步：确保管理员角色有权限
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT 'd67f4241-f887-481a-b345-94fb36782500', id
FROM sys_menu
WHERE menu_name = '密钥列表'
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- 第三步：删除任何残留的"API密钥管理"目录菜单（如果存在）
DELETE FROM sys_menu WHERE menu_name = 'API密钥管理';
DELETE FROM sys_menu WHERE menu_name = '使用日志';

-- 验证结果
SELECT
    '修复后的菜单配置' as "说明",
    id as "菜单ID",
    menu_name as "菜单名称",
    parent_id as "父菜单ID",
    path as "路径",
    status as "状态(0=正常)",
    visible as "可见(1=显示)",
    menu_type as "类型"
FROM sys_menu
WHERE menu_name = '密钥列表';

-- 检查权限分配
SELECT
    rm.role_id,
    rm.menu_id,
    m.menu_name
FROM sys_role_menu rm
JOIN sys_menu m ON m.id = rm.menu_id
WHERE m.menu_name = '密钥列表';

SELECT '✅ 菜单显示问题修复完成' AS status;
