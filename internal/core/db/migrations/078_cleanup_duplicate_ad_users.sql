-- 清理AD域中的复制冲突对象
-- 迁移版本: 078
-- 描述: 删除AD同步产生的冲突对象（用户名包含$DUPLICATE-的记录）

-- 删除用户名包含 DUPLICATE 的AD用户记录
DELETE FROM sys_ad_user
WHERE username LIKE '$DUPLICATE-%'
AND deleted_at IS NULL;

-- 验证删除结果
SELECT 'cleanup_duplicate_users migration completed' AS status,
       COUNT(*) AS remaining_duplicate_users
FROM sys_ad_user
WHERE username LIKE '$DUPLICATE-%'
AND deleted_at IS NULL;
