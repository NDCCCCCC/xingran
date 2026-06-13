-- Phase 32 P2-A4 source-tracking:
--   Original commit: a3032b2e
--   Created: 2026-01-16
--   Note: Conflicts with 031_update_building_coordinates.sql — both share prefix 031. Runner uses Go code ordering, not filename sort; conflict is harmless.

-- 增强机房表，添加所属楼宇字段
-- 执行日期: 2026-01-14

-- 添加所属楼宇ID字段（关联ops_buildings）
ALTER TABLE ops_server_rooms ADD COLUMN IF NOT EXISTS building_id VARCHAR(64) NOT NULL DEFAULT '';

-- 添加所属楼宇名称字段
ALTER TABLE ops_server_rooms ADD COLUMN IF NOT EXISTS building_name VARCHAR(100);

-- 添加外键约束（如果需要）
-- ALTER TABLE ops_server_rooms ADD CONSTRAINT fk_server_room_building
--   FOREIGN KEY (building_id) REFERENCES ops_buildings(id) ON DELETE RESTRICT;

-- 为新字段添加注释
COMMENT ON COLUMN ops_server_rooms.building_id IS '所属楼宇ID（关联ops_buildings）';
COMMENT ON COLUMN ops_server_rooms.building_name IS '所属楼宇名称';

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_ops_server_rooms_building_id ON ops_server_rooms(building_id);

-- 修改floor_id字段为必填
ALTER TABLE ops_server_rooms ALTER COLUMN floor_id SET NOT NULL;
ALTER TABLE ops_server_rooms ALTER COLUMN floor_id DROP DEFAULT;

-- 添加复合索引（楼宇+楼层）以优化查询
CREATE INDEX IF NOT EXISTS idx_ops_server_rooms_building_floor ON ops_server_rooms(building_id, floor_id);
