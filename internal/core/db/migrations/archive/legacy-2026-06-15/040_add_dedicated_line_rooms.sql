-- 添加专线源机房和目的机房字段
-- 用于绑定源和目的机房，除了现有的源/目的设备和端口

-- 添加源机房和目的机房字段
ALTER TABLE ops_dedicated_lines
ADD COLUMN IF NOT EXISTS source_room_id VARCHAR(64),
ADD COLUMN IF NOT EXISTS source_room_name VARCHAR(100),
ADD COLUMN IF NOT EXISTS dest_room_id VARCHAR(64),
ADD COLUMN IF NOT EXISTS dest_room_name VARCHAR(100);

-- 添加注释
COMMENT ON COLUMN ops_dedicated_lines.source_room_id IS '源机房ID（关联ops_server_rooms）';
COMMENT ON COLUMN ops_dedicated_lines.source_room_name IS '源机房名称（冗余字段）';
COMMENT ON COLUMN ops_dedicated_lines.dest_room_id IS '目的机房ID（关联ops_server_rooms）';
COMMENT ON COLUMN ops_dedicated_lines.dest_room_name IS '目的机房名称（冗余字段）';

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_ops_dedicated_lines_source_room ON ops_dedicated_lines(source_room_id) WHERE source_room_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ops_dedicated_lines_dest_room ON ops_dedicated_lines(dest_room_id) WHERE dest_room_id IS NOT NULL;
