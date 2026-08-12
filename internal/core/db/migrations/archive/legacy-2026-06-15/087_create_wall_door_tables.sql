-- 087: 创建墙体和门表（CAD风格平面图）
-- 支持运维管理中的楼层平面图绘制功能

-- 墙体表
CREATE TABLE IF NOT EXISTS ops_walls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    floor_id UUID NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'straight',
    points JSONB NOT NULL,
    thickness INTEGER DEFAULT 10,
    height DECIMAL(10,2) DEFAULT 3.0,
    color VARCHAR(20) DEFAULT '#5C6BC0',
    name VARCHAR(100),
    remark VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0,
    CONSTRAINT fk_wall_floor FOREIGN KEY (floor_id) REFERENCES ops_floors(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_wall_floor_id ON ops_walls(floor_id);
CREATE INDEX IF NOT EXISTS idx_wall_type ON ops_walls(type);
COMMENT ON TABLE ops_walls IS '墙体表 - CAD风格平面图';
COMMENT ON COLUMN ops_walls.points IS '墙体坐标点JSON数组';
COMMENT ON COLUMN ops_walls.thickness IS '墙体厚度（像素）';

-- 门表
CREATE TABLE IF NOT EXISTS ops_doors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    floor_id UUID NOT NULL,
    wall_id UUID,
    position JSONB NOT NULL,
    angle INTEGER DEFAULT 0,
    type VARCHAR(20) NOT NULL DEFAULT 'single',
    direction VARCHAR(20) NOT NULL DEFAULT 'left',
    width INTEGER DEFAULT 80,
    length INTEGER DEFAULT 50,
    color VARCHAR(20) DEFAULT '#FF7043',
    name VARCHAR(100),
    remark VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0,
    CONSTRAINT fk_door_floor FOREIGN KEY (floor_id) REFERENCES ops_floors(id) ON DELETE CASCADE,
    CONSTRAINT fk_door_wall FOREIGN KEY (wall_id) REFERENCES ops_walls(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_door_floor_id ON ops_doors(floor_id);
CREATE INDEX IF NOT EXISTS idx_door_wall_id ON ops_doors(wall_id);
CREATE INDEX IF NOT EXISTS idx_door_type ON ops_doors(type);
COMMENT ON TABLE ops_doors IS '门表 - CAD风格平面图';
COMMENT ON COLUMN ops_doors.position IS '门位置坐标JSON';
COMMENT ON COLUMN ops_doors.angle IS '旋转角度（度）';
