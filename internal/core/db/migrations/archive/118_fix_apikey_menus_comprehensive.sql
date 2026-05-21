-- 综合修复API密钥管理菜单
-- 问题1: "密钥列表"菜单的path应该是空字符串，不是'system/apikeys'
-- 问题2: "使用日志"菜单被错误删除，需要恢复
-- 问题3: component路径需要正确指向实际文件

DO $$
DECLARE
    v_sys_manage_id UUID := 'd67f4240-f887-481a-b345-94fb36782500';
    v_apikey_dir_id UUID;
    v_keylist_menu_id UUID;
    v_logs_menu_id UUID;
    v_admin_role_id UUID;
BEGIN
    -- 第一步：清理旧的错误配置
    DELETE FROM sys_role_menu
    WHERE menu_id IN (
        SELECT id FROM sys_menu
        WHERE menu_name IN ('API密钥管理', '密钥列表', '使用日志')
    );

    DELETE FROM sys_menu
    WHERE menu_name IN ('API密钥管理', '密钥列表', '使用日志');

    -- 第二步：创建"API密钥管理"目录菜单（二级，类型M）
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

    -- 第三步：创建"密钥列表"子菜单（三级菜单，类型C）
    -- 关键修复：path设为空字符串，表示父目录下的默认页面
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
        '',                   -- ✅ 修复：空字符串，作为默认页面
        'system/apikeys/index',
        'C',
        1,
        0,
        'system:apikey:list',
        'KeyOutlined',
        'API密钥列表页面',
        NOW(),
        NOW()
    )
    RETURNING id INTO v_keylist_menu_id;

    RAISE NOTICE '创建密钥列表菜单成功，ID: %', v_keylist_menu_id;

    -- 第四步：创建"使用日志"子菜单（三级菜单，类型C）
    -- 注意：虽然LogsModal是一个Modal组件，但保留菜单项以符合用户期望
    -- 用户点击菜单时，可以显示一个独立的日志页面或者打开Modal
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
        'logs',               -- ✅ 使用相对路径
        'system/apikeys/LogsModal',
        'C',
        1,
        0,
        'system:apikey:logs',
        'FileTextOutlined',
        'API密钥使用日志',
        NOW(),
        NOW()
    )
    RETURNING id INTO v_logs_menu_id;

    RAISE NOTICE '创建使用日志菜单成功，ID: %', v_logs_menu_id;

    -- 第五步：为管理员角色分配菜单权限
    SELECT id INTO v_admin_role_id
    FROM sys_role
    WHERE role_key = 'admin'
    LIMIT 1;

    IF v_admin_role_id IS NOT NULL THEN
        INSERT INTO sys_role_menu (role_id, menu_id) VALUES
            (v_admin_role_id, v_apikey_dir_id),
            (v_admin_role_id, v_keylist_menu_id),
            (v_admin_role_id, v_logs_menu_id)
        ON CONFLICT (role_id, menu_id) DO NOTHING;

        RAISE NOTICE '为管理员角色分配API密钥菜单权限成功';
    ELSE
        RAISE WARNING '未找到管理员角色，跳过权限分配';
    END IF;

    RAISE NOTICE 'API密钥管理菜单综合修复完成！';
END $$;

-- 验证修复结果
SELECT
    'API密钥管理' as "目录菜单",
    v_apikey_dir_id as "目录ID",
    v_apikey_dir_id || ' → ' || v_keylist_menu_id as "密钥列表完整路径",
    v_apikey_dir_id || ' → ' || v_logs_menu_id as "使用日志完整路径"
FROM (
    SELECT id AS v_apikey_dir_id
    FROM sys_menu
    WHERE menu_name = 'API密钥管理'
    LIMIT 1
) t1,
LATERAL (
    SELECT id AS v_keylist_menu_id
    FROM sys_menu
    WHERE menu_name = '密钥列表' AND parent_id = v_apikey_dir_id
    LIMIT 1
) t2,
LATERAL (
    SELECT id AS v_logs_menu_id
    FROM sys_menu
    WHERE menu_name = '使用日志' AND parent_id = v_apikey_dir_id
    LIMIT 1
) t3;

-- 详细验证菜单结构
SELECT
    m2.menu_name as "父菜单",
    m2.path as "父路径",
    m3.menu_name as "子菜单",
    m3.path as "子路径",
    m3.component as "组件路径",
    m3.menu_type as "类型"
FROM sys_menu m2
LEFT JOIN sys_menu m3 ON m3.parent_id = m2.id
WHERE m2.menu_name = 'API密钥管理'
ORDER BY m3.order_num;

SELECT '✅ 综合修复完成：密钥列表path=空，使用日志已恢复' AS status;
