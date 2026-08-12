-- 081: 删除楼宇表中的城市代码、城市名称和父级ID字段
-- 这些字段不再需要，经纬度通过地址自动解析

-- 删除城市代码字段
ALTER TABLE ops_buildings DROP COLUMN IF EXISTS city_code;

-- 删除城市名称字段
ALTER TABLE ops_buildings DROP COLUMN IF EXISTS city_name;

-- 删除父级楼宇ID字段
ALTER TABLE ops_buildings DROP COLUMN IF EXISTS parent_id;

-- 删除相关索引（如果存在）
DROP INDEX IF EXISTS idx_ops_buildings_city_code;
DROP INDEX IF EXISTS idx_ops_buildings_parent_id;

-- 验证字段删除成功
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_name = 'ops_buildings'
ORDER BY ordinal_position;
