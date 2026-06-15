-- 111_remove_info_point_code_column.sql
-- 删除信息点表中的 code 列（如果存在）

-- 删除唯一约束索引（如果存在）
DROP INDEX IF EXISTS idx_ops_info_points_code;

-- 删除 code 列（如果存在）
-- 注意：如果 code 列不存在，此语句会报错，但不影响后续操作
ALTER TABLE ops_info_points DROP COLUMN IF EXISTS code;

COMMENT ON TABLE ops_info_points IS '信息点表';
