-- 091: 创建平面图文本表
-- 用于存储楼层平面图中的文本标注元素

CREATE TABLE IF NOT EXISTS ops_floor_plan_texts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    floor_id UUID NOT NULL,
    position JSONB NOT NULL,
    content VARCHAR(500) NOT NULL,
    font_size INTEGER DEFAULT 14,
    color VARCHAR(20) DEFAULT '#333333',
    font_family VARCHAR(100) DEFAULT 'Arial, sans-serif',
    font_weight VARCHAR(20) DEFAULT 'normal',
    font_style VARCHAR(20) DEFAULT 'normal',
    angle INTEGER DEFAULT 0,
    remark VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0,
    CONSTRAINT fk_text_floor FOREIGN KEY (floor_id) REFERENCES ops_floors(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_text_floor_id ON ops_floor_plan_texts(floor_id);
CREATE INDEX IF NOT EXISTS idx_text_font_size ON ops_floor_plan_texts(font_size);

COMMENT ON TABLE ops_floor_plan_texts IS '平面图文本表 - 用于楼层平面图标注';
COMMENT ON COLUMN ops_floor_plan_texts.position IS '文本位置坐标JSON {x, y}';
COMMENT ON COLUMN ops_floor_plan_texts.content IS '文本内容';
COMMENT ON COLUMN ops_floor_plan_texts.font_size IS '字体大小（像素）';
COMMENT ON COLUMN ops_floor_plan_texts.color IS '文本颜色（十六进制）';
COMMENT ON COLUMN ops_floor_plan_texts.font_family IS '字体家族';
COMMENT ON COLUMN ops_floor_plan_texts.font_weight IS '字体粗细：normal, bold';
COMMENT ON COLUMN ops_floor_plan_texts.font_style IS '字体样式：normal, italic';
COMMENT ON COLUMN ops_floor_plan_texts.angle IS '旋转角度（度）';
