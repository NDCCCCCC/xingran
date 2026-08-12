-- =============================================
-- AD Group Sync Permission - Menu Data Migration
-- Migration version: 131
-- Description: Add group sync permission button to AD group management menu
-- =============================================

-- Add "Sync Groups" button permission under the AD Group Management menu
-- The parent menu is the "用户组管理" menu which has perms 'ops:ad:group:view'
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
    '组同步',
    id,
    3,
    '',
    '',
    'F',
    1,
    0,
    'ops:ad:group:sync',
    '',
    'AD用户组同步权限',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE perms = 'ops:ad:group:view'
    AND menu_type = 'F'
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE perms = 'ops:ad:group:sync'
    )
LIMIT 1;
