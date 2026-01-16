-- 添加工位旋转角度字段
-- Migration: 035
-- Description: 为工位表添加 rotation 字段，用于平面图中工位的旋转角度

-- 添加 rotation 列
ALTER TABLE sys_workstation
ADD COLUMN IF NOT EXISTS rotation INT DEFAULT 0;

-- 添加注释
COMMENT ON COLUMN sys_workstation.rotation IS '平面图旋转角度（度），范围0-360，默认为0';
