-- =============================================
-- 添加电脑设备数量到同步日志表
-- 迁移版本: 071
-- 描述: 为sys_ad_sync_log表添加computer_count字段
-- =============================================

-- 添加电脑设备数量字段
ALTER TABLE sys_ad_sync_log ADD COLUMN IF NOT EXISTS computer_count INTEGER DEFAULT 0;

-- 添加注释
COMMENT ON COLUMN sys_ad_sync_log.computer_count IS '同步的电脑设备数量';
