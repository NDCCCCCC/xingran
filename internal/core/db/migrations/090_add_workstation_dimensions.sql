-- 添加工位尺寸和桌型字段
-- Migration: 090
-- Description: 为工位表添加宽度、深度和桌型字段，用于平面图编辑器

-- 添加 width 列（工位宽度，单位：毫米）
ALTER TABLE sys_workstation
ADD COLUMN IF NOT EXISTS width INT DEFAULT 160;

COMMENT ON COLUMN sys_workstation.width IS '工位宽度（毫米），默认1600mm';

-- 添加 depth 列（工位深度，单位：毫米）
ALTER TABLE sys_workstation
ADD COLUMN IF NOT EXISTS depth INT DEFAULT 70;

COMMENT ON COLUMN sys_workstation.depth IS '工位深度（毫米），默认700mm';

-- 添加 desk_type 列（桌型：0=一字型, 1=L型）
ALTER TABLE sys_workstation
ADD COLUMN IF NOT EXISTS desk_type INT DEFAULT 0;

COMMENT ON COLUMN sys_workstation.desk_type IS '桌型：0=一字型, 1=L型，默认为一字型';

-- 为已有工位设置默认值（如果width或depth为NULL）
UPDATE sys_workstation
SET width = 160
WHERE width IS NULL;

UPDATE sys_workstation
SET depth = 70
WHERE depth IS NULL;

UPDATE sys_workstation
SET desk_type = 0
WHERE desk_type IS NULL;
