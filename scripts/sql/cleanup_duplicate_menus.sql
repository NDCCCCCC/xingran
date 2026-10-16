-- 清理 sys_menu 重复数据脚本
-- 生成时间: 2026-05-19
-- 保留规则: 每组保留创建时间最早的记录
-- ⚠️ 执行前请备份数据库！
-- 备份命令: pg_dump -h 10.62.10.34 -U postgres -d xingran -t sys_menu > backup_sys_menu.sql

BEGIN;

-- 显示清理前的重复情况
SELECT '=== 清理前重复菜单统计 ===' as info;
SELECT menu_name, COUNT(*) as count
FROM sys_menu
WHERE deleted_at IS NULL
GROUP BY menu_name
HAVING COUNT(*) > 1
ORDER BY count DESC;

-- 删除角色菜单关联（先删除外键关联）
DELETE FROM sys_role_menu
WHERE menu_id IN (
    SELECT id FROM (
        SELECT id, ROW_NUMBER() OVER (PARTITION BY menu_name ORDER BY created_at) as rn
        FROM sys_menu
        WHERE deleted_at IS NULL
    ) t WHERE rn > 1
);

-- 删除重复菜单（保留每组中创建时间最早的）
DELETE FROM sys_menu
WHERE id IN (
    SELECT id FROM (
        SELECT id, ROW_NUMBER() OVER (PARTITION BY menu_name ORDER BY created_at) as rn
        FROM sys_menu
        WHERE deleted_at IS NULL
    ) t WHERE rn > 1
);

-- 验证结果
SELECT '=== 清理后统计 ===' as info;
SELECT '清理后菜单总数: ' || COUNT(*) as result
FROM sys_menu
WHERE deleted_at IS NULL;

SELECT '=== 检查是否还有重复 ===' as info;
SELECT menu_name, COUNT(*) as count
FROM sys_menu
WHERE deleted_at IS NULL
GROUP BY menu_name
HAVING COUNT(*) > 1;

COMMIT;

-- 如果需要回滚，使用: ROLLBACK;
