-- ========================================
-- 094: 添加楼层平面图图片字段
-- ========================================
-- 目的：支持在楼层平面图编辑器中上传并显示底层平面图图片
-- 日期：2025-02-04
-- ========================================

SET search_path TO public;

ALTER TABLE ops_floors
ADD COLUMN IF NOT EXISTS plan_image_id UUID;

ALTER TABLE ops_floors
DROP CONSTRAINT IF EXISTS fk_floors_plan_image;

ALTER TABLE ops_floors
ADD CONSTRAINT fk_floors_plan_image
FOREIGN KEY (plan_image_id) REFERENCES sys_files(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_floors_plan_image
ON ops_floors(plan_image_id) WHERE plan_image_id IS NOT NULL;

COMMENT ON COLUMN ops_floors.plan_image_id IS '楼层平面图图片ID，关联 sys_files 表，用于在平面图编辑器中显示底层参考图';

SELECT
    '094_add_floor_plan_image.sql' AS migration,
    'plan_image_id field added to ops_floors' AS status;
