-- 最终修复：创建API密钥管理菜单
-- 使用正确的系统管理ID: d67f4240-f887-481a-b345-94fb36782500

-- 清理任何可能存在的错误API密钥菜单记录
DELETE FROM sys_role_menu
WHERE menu_id IN (
    SELECT id FROM sys_menu
    WHERE menu_name IN ('API密钥管理', '密钥列表', '使用日志')
);

DELETE FROM sys_menu
WHERE menu_name IN ('API密钥管理', '密钥列表', '使用日志');

-- 创建"API密钥管理"目录菜单（二级，仿照"验证码背景图"的M类型）
-- 使用DO块确保获取正确的系统管理ID和创建子菜单
DO $$
DECLARE
    v_sys_manage_id UUID := 'd67f4240-f887-481a-b345-94fb36782500';
    v_apikey_dir_id UUID;
    v_keylist_menu_id UUID;
    v_logs_menu_id UUID;
    v_admin_role_id UUID;
BEGIN
    -- 验证系统管理菜单存在
    IF NOT EXISTS (SELECT 1 FROM sys_menu WHERE id = v_sys_manage_id) THEN
        RAISE EXCEPTION '系统管理菜单不存在: %', v_sys_manage_id;
    END IF;

    -- 创建API密钥管理目录（二级菜单，类型M）
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
        'API密钥管理',
        v_sys_manage_id,
        11,
        'apikeys',
        NULL,
        'M',
        1,
        0,
        'KeyOutlined',
        'API密钥管理目录',
        NOW(),
        NOW()
    )
    RETURNING id INTO v_apikey_dir_id;

    RAISE NOTICE '创建API密钥管理目录成功，ID: %', v_apikey_dir_id;

    -- 创建"密钥列表"子菜单（三级菜单，类型C）
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
        perms,
        icon,
        remark,
        created_at,
        updated_at
    ) VALUES (
        gen_random_uuid(),
        '密钥列表',
        v_apikey_dir_id,
        1,
        'system/apikeys',
        'system/apikeys/index',
        'C',
        1,
        0,
        'system:apikey:list',
        NULL,
        'API密钥列表页面',
        NOW(),
        NOW()
    )
    RETURNING id INTO v_keylist_menu_id;

    RAISE NOTICE '创建密钥列表菜单成功，ID: %', v_keylist_menu_id;

    -- 创建"使用日志"子菜单（三级菜单，类型C）
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
        perms,
        icon,
        remark,
        created_at,
        updated_at
    ) VALUES (
        gen_random_uuid(),
        '使用日志',
        v_apikey_dir_id,
        2,
        'system/apikeys/logs',
        'system/apikeys/LogsModal/index',
        'C',
        1,
        0,
        'system:apikey:logs',
        NULL,
        'API密钥使用日志页面',
        NOW(),
        NOW()
    )
    RETURNING id INTO v_logs_menu_id;

    RAISE NOTICE '创建使用日志菜单成功，ID: %', v_logs_menu_id;

    -- 获取管理员角色ID
    SELECT id INTO v_admin_role_id
    FROM sys_role
    WHERE role_key = 'admin'
    LIMIT 1;

    IF v_admin_role_id IS NOT NULL THEN
        -- 为管理员角色分配菜单权限
        INSERT INTO sys_role_menu (role_id, menu_id) VALUES
            (v_admin_role_id, v_apikey_dir_id),
            (v_admin_role_id, v_keylist_menu_id),
            (v_admin_role_id, v_logs_menu_id)
        ON CONFLICT (role_id, menu_id) DO NOTHING;

        RAISE NOTICE '为管理员角色分配API密钥菜单权限成功';
    ELSE
        RAISE WARNING '未找到管理员角色，跳过权限分配';
    END IF;

    RAISE NOTICE 'API密钥管理菜单创建完成！';
END $$;

-- 验证结果
SELECT
    m1.menu_name as "一级菜单",
    m2.menu_name as "二级菜单",
    m2.menu_type as "二级类型",
    m2.id as "二级ID",
    m3.menu_name as "三级菜单",
    m3.menu_type as "三级类型",
    m3.id as "三级ID",
    m3.parent_id as "三级parent_id"
FROM sys_menu m1
LEFT JOIN sys_menu m2 ON m2.parent_id = m1.id
LEFT JOIN sys_menu m3 ON m3.parent_id = m2.id
WHERE m1.id = 'd67f4240-f887-481a-b345-94fb36782500'
  AND m2.menu_name = 'API密钥管理'
ORDER BY m2.order_num, m3.order_num;

SELECT 'API密钥管理菜单创建完成！结构：系统管理 > API密钥管理(M) > 密钥列表(C)/使用日志(C)' AS status;
