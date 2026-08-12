-- =============================================
-- 修复AD域电脑设备管理的组件路径
-- 迁移版本: 073
-- 描述: 修正电脑设备管理的component字段，使其符合前端路由规范
-- =============================================

-- 更新电脑设备管理的component字段
-- 原值: /ad-domain/computers（错误的绝对路径）
-- 新值: ad-domain/computers（相对路径，不带pages/前缀）
UPDATE sys_menu
SET component = 'ad-domain/computers'
WHERE menu_name = '电脑设备管理'
  AND component = '/ad-domain/computers';

-- 验证修改结果
SELECT
    id,
    menu_name,
    path,
    component,
    menu_type,
    visible,
    status
FROM sys_menu
WHERE menu_name = '电脑设备管理';

SELECT
    '073_fix_ad_computer_component_path.sql completed' AS status;
