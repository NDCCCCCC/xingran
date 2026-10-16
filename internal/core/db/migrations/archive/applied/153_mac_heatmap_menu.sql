-- =============================================
-- MAC 端口使用热力图 菜单与权限
-- Migration version: 153
-- Description: Phase 15 PERF-04 — 在 网络管理/MAC地址历史 父菜单下挂载 端口使用热力图 子菜单
-- 权限点: network:mac:heatmap (D-18 锁定)
-- 父菜单候选: '历史查询' 或 'MAC地址历史' (Phase 13 已创建)
--
-- Schema 注: Go 版 sys_menu 已用 Meta JSONB 字段统一元数据
-- 不存在 XingRan-Java 原版的 query / is_frame / is_cache 字段, 仅插入实际存在的列
-- =============================================

-- ================================
-- 1. 主菜单项: 端口使用热力图
-- ================================
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
)
SELECT
    gen_random_uuid(),
    '端口使用热力图',
    p.id,
    COALESCE((SELECT MAX(order_num) + 1 FROM sys_menu WHERE parent_id = p.id), 1),
    'heatmap',
    'network/mac/heatmap',
    'C',
    1,
    0,
    'network:mac:heatmap',
    'heat-map',
    'MAC 端口使用热力图 (Phase 15 PERF-04, 数据源 MV-04)',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu p
WHERE p.menu_name IN ('历史查询', 'MAC地址历史')
    AND p.menu_type = 'C'
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '端口使用热力图'
            AND parent_id = p.id
    )
ORDER BY CASE WHEN p.menu_name = 'MAC地址历史' THEN 0 ELSE 1 END
LIMIT 1;

-- ================================
-- 2. 按钮权限: 热力图查询
-- ================================
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
)
SELECT
    gen_random_uuid(),
    '热力图查询',
    m.id,
    1,
    '',
    NULL,
    'F',
    1,
    0,
    'network:mac:heatmap:query',
    '#',
    '查询 MAC 端口使用热力图按钮权限',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '端口使用热力图'
    AND m.menu_type = 'C'
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '热力图查询'
            AND parent_id = m.id
    );
