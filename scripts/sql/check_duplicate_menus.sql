-- 检查 sys_menu 表中的重复菜单
-- 执行方式：psql -U xingran -d xingran_next -f check_duplicate_menus.sql

\echo '===== 1. 检查完全重复的记录（除了 id 外所有字段相同） ====='
SELECT menu_name, parent_id, path, component, menu_type, COUNT(*) as dup_count
FROM sys_menu
WHERE deleted_at IS NULL
GROUP BY menu_name, parent_id, path, component, menu_type
HAVING COUNT(*) > 1
ORDER BY dup_count DESC;

\echo '\n===== 2. 检查相同 parent_id 下的重复菜单名称 ====='
SELECT parent_id, menu_name, COUNT(*) as dup_count,
       string_agg(id::text, ', ') as duplicate_ids
FROM sys_menu
WHERE deleted_at IS NULL
GROUP BY parent_id, menu_name
HAVING COUNT(*) > 1
ORDER BY dup_count DESC;

\echo '\n===== 3. 检查重复的路由路径 ====='
SELECT path, COUNT(*) as dup_count,
       string_agg(id::text, ', ') as duplicate_ids
FROM sys_menu
WHERE deleted_at IS NULL AND path IS NOT NULL AND path != ''
GROUP BY path
HAVING COUNT(*) > 1
ORDER BY dup_count DESC;

\echo '\n===== 4. 统计总记录数 ====='
SELECT COUNT(*) as total_menus FROM sys_menu WHERE deleted_at IS NULL;

\echo '\n===== 5. 查看潜在的重复记录详情 ====='
SELECT id, menu_name, parent_id, path, component, menu_type, created_at
FROM sys_menu
WHERE deleted_at IS NULL
  AND (menu_name, parent_id) IN (
    SELECT menu_name, parent_id
    FROM sys_menu
    WHERE deleted_at IS NULL
    GROUP BY menu_name, parent_id
    HAVING COUNT(*) > 1
  )
ORDER BY parent_id, menu_name, id;
