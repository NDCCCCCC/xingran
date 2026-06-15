-- Migration: 078_cleanup_duplicate_ad_users.sql
-- Description: 清理AD域同步产生的复制冲突对象
-- Version: 078
-- Date: 2025-01-27

-- 删除用户名包含 $DUPLICATE- 的冲突记录（仅限未删除记录）
DELETE FROM sys_ad_user
WHERE username LIKE '$DUPLICATE-%'
  AND deleted_at IS NULL;
