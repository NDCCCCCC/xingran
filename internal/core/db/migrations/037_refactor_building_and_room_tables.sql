-- 037: 重构楼宇和机房表结构
-- 1. 删除楼宇表的code字段和totalArea字段（如果存在）
-- 2. 添加楼宇表的totalFloors字段
-- 3. 删除机房表的code字段
-- 4. 删除楼层表的area字段

-- 删除楼宇表的code字段
ALTER TABLE ops_buildings DROP COLUMN IF EXISTS code;

-- 删除楼宇表的total_area字段
ALTER TABLE ops_buildings DROP COLUMN IF EXISTS total_area;

-- 添加楼宇表的total_floors字段
ALTER TABLE ops_buildings ADD COLUMN IF NOT EXISTS total_floors INT DEFAULT 0;
COMMENT ON COLUMN ops_buildings.total_floors IS '楼层数（根据创建的楼层自动计算）';

-- 删除机房表的code字段
ALTER TABLE ops_server_rooms DROP COLUMN IF EXISTS code;

-- 删除楼层表的area字段
ALTER TABLE ops_floors DROP COLUMN IF EXISTS area;
