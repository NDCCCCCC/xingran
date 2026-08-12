-- ============================================
-- 026_add_cache_location_field.sql
-- 说明: 添加缓存层级字段，支持区分L1(内存)和L2(Redis)缓存
-- ============================================

-- 添加 location 字段到 sys_cache_info 表
ALTER TABLE sys_cache_info
ADD COLUMN IF NOT EXISTS location VARCHAR(16)
DEFAULT 'l2'
COMMENT '缓存位置:l1/l2/both';

-- 为已有数据设置默认值
UPDATE sys_cache_info SET location = 'l2' WHERE location IS NULL;

-- ============================================
-- 验证迁移结果
-- ============================================

-- 查看表结构
SELECT column_name, data_type, column_default, character_maximum_length, is_nullable
FROM information_schema.columns
WHERE table_name = 'sys_cache_info'
AND column_name = 'location';

-- 查看数据分布
SELECT location, COUNT(*) as count
FROM sys_cache_info
GROUP BY location;
