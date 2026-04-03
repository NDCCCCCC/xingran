-- 为 sys_workstation 表添加楼宇相关字段
-- 执行日期: 2026-02-13

-- 添加楼宇ID字段（关联ops_buildings）
ALTER TABLE sys_workstation ADD COLUMN IF NOT EXISTS building_id VARCHAR(64);

-- 添加楼宇名称字段
ALTER TABLE sys_workstation ADD COLUMN IF NOT EXISTS building_name VARCHAR(100);

-- 为新字段添加注释
COMMENT ON COLUMN sys_workstation.building_id IS '所属楼宇ID（关联ops_buildings）';
COMMENT ON COLUMN sys_workstation.building_name IS '所属楼宇名称';

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_sys_workstation_building_id ON sys_workstation(building_id) WHERE building_id IS NOT NULL;

-- 根据现有的楼层信息，自动填充楼宇信息（可选）
-- 注意：需要类型转换，因为 ops_buildings.building_id 是 varchar，需要转换为 uuid
UPDATE sys_workstation ws
SET building_id = f.building_id::uuid,
    building_name = b.name
FROM ops_floors f
JOIN ops_buildings b ON b.id = f.building_id::uuid
WHERE ws.floor_id = f.id::text
  AND ws.building_id IS NULL;
