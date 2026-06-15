-- Phase 32 P2-A4 source-tracking:
--   Original commit: a3032b2e
--   Created: 2026-01-16
--   Note: Conflicts with 029_add_building_spaces_menu.sql — both share prefix 029. Runner uses Go code ordering, not filename sort; conflict is harmless.

-- 添加楼宇地址和坐标相关字段
-- 执行此脚本前请先备份数据库

-- 1. 添加城市代码字段
ALTER TABLE ops_buildings ADD COLUMN IF NOT EXISTS city_code VARCHAR(20);

-- 2. 添加城市名称字段
ALTER TABLE ops_buildings ADD COLUMN IF NOT EXISTS city_name VARCHAR(50);

-- 3. 添加详细地址字段
ALTER TABLE ops_buildings ADD COLUMN IF NOT EXISTS address VARCHAR(200);

-- 4. 添加经度字段
ALTER TABLE ops_buildings ADD COLUMN IF NOT EXISTS longitude DECIMAL(11, 8);

-- 5. 添加纬度字段
ALTER TABLE ops_buildings ADD COLUMN IF NOT EXISTS latitude DECIMAL(11, 8);

-- 6. 添加父级楼宇ID字段
ALTER TABLE ops_buildings ADD COLUMN IF NOT EXISTS parent_id VARCHAR(64);

-- 7. 添加层级字段
ALTER TABLE ops_buildings ADD COLUMN IF NOT EXISTS level INT DEFAULT 2;

-- 8. 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_ops_buildings_city_code ON ops_buildings(city_code);
CREATE INDEX IF NOT EXISTS idx_ops_buildings_parent_id ON ops_buildings(parent_id);
CREATE INDEX IF NOT EXISTS idx_ops_buildings_longitude_latitude ON ops_buildings(longitude, latitude);

-- 9. 为现有数据设置默认值（武汉市）
UPDATE ops_buildings
SET city_code = 'wuhan',
    city_name = '武汉市',
    level = 2
WHERE city_code IS NULL OR city_code = '';

-- 验证字段添加成功
SELECT column_name, data_type, character_maximum_length
FROM information_schema.columns
WHERE table_name = 'ops_buildings'
  AND column_name IN ('city_code', 'city_name', 'address', 'longitude', 'latitude', 'parent_id', 'level')
ORDER BY ordinal_position;
