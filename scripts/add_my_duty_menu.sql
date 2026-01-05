-- ============================================
-- 添加"我的值班"菜单到运维管理下
-- ============================================

-- 步骤1: 获取"运维管理"父菜单ID（DO块不需要$）
DO $$
DECLARE
    v_parent_id UUID;
    v_menu_id UUID;
BEGIN
    -- 获取运维管理菜单的ID
    SELECT id INTO v_parent_id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL LIMIT 1;

    IF v_parent_id IS NULL THEN
        RAISE NOTICE '未找到"运维管理"父菜单，请先创建运维管理菜单';
    ELSE
        -- 插入"我的值班"菜单
        INSERT INTO sys_menu (
            id,
            menu_name,
            parent_id,
            order_num,
            path,
            component,
            menu_type,
            visible,
            status,
            icon,
            remark,
            created_at,
            updated_at
        ) VALUES (
            gen_random_uuid(),
            '我的值班',
            v_parent_id,
            5,
            'duty/my-duty',
            NULL,
            'C',
            1,
            0,
            'CalendarOutlined',
            '查看个人值班记录和统计',
            CURRENT_TIMESTAMP,
            CURRENT_TIMESTAMP
        )
        ON CONFLICT (menu_name, parent_id) DO NOTHING
        RETURNING id INTO v_menu_id;

        RAISE NOTICE '成功添加"我的值班"菜单，ID: %', v_menu_id;

        -- 为所有启用角色分配菜单权限
        INSERT INTO sys_role_menu (role_id, menu_id)
        SELECT
            r.id,
            (SELECT id FROM sys_menu WHERE menu_name = '我的值班' AND parent_id = v_parent_id LIMIT 1)
        FROM sys_role r
        WHERE r.status = 0
            AND NOT EXISTS (
                SELECT 1 FROM sys_role_menu rm
                WHERE rm.role_id = r.id
                    AND rm.menu_id = (SELECT id FROM sys_menu WHERE menu_name = '我的值班' AND parent_id = v_parent_id LIMIT 1)
            );

        RAISE NOTICE '成功为所有启用角色分配"我的值班"菜单权限';
    END IF;
END $$;

-- 验证结果
SELECT
    m.id,
    m.menu_name,
    p.menu_name as parent_name,
    m.path,
    m.menu_type,
    m.visible,
    m.status,
    m.icon
FROM sys_menu m
LEFT JOIN sys_menu p ON m.parent_id = p.id
WHERE m.menu_name = '我的值班';
