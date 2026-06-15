-- Migration: 095_add_floor_area_field.sql
-- Description: 添加楼层面积和平面图字段

-- 添加面积字段
ALTER TABLE ops_floors
ADD COLUMN IF NOT EXISTS area NUMERIC(10,2);

-- 添加平面图ID字段
ALTER TABLE ops_floors
ADD COLUMN IF NOT EXISTS plan_image_id VARCHAR(64);

-- 添加平面图URL字段
ALTER TABLE ops_floors
ADD COLUMN IF NOT EXISTS plan_image_url VARCHAR(500);

-- 添加外键约束到文件表
ALTER TABLE ops_floors
ADD CONSTRAINT fk_floor_plan_image
FOREIGN KEY (plan_image_id) REFERENCES sys_files(id)
ON DELETE SET NULL
ON UPDATE CASCADE;

-- 添加字段注释
COMMENT ON COLUMN ops_floors.area IS '楼层面积（平方米）';
COMMENT ON COLUMN ops_floors.plan_image_id IS '平面图文件ID';
COMMENT ON COLUMN ops_floors.plan_image_url IS '平面图URL';

-- 验证迁移
SELECT
    '095_add_floor_area_field.sql' AS migration,
    'area, plan_image_id, plan_image_url fields added to ops_floors' AS status;
