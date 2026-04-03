-- 110_add_info_point_unique_constraint.sql
-- 添加信息点表的唯一约束，支持 Excel 导入的 ON CONFLICT 功能
-- 约束：同一工位下信息点名称唯一

-- 添加唯一约束（排除已删除的记录）
CREATE UNIQUE INDEX IF NOT EXISTS ops_info_points_workstation_name_idx
ON ops_info_points (workstation_id, name)
WHERE deleted_at IS NULL;

-- 添加注释
COMMENT ON INDEX ops_info_points_workstation_name_idx IS '同一工位下信息点名称唯一（软删除记录除外）';

-- 注意：此迁移假设表中没有 code 列
-- 如果存在 code 列，需要先删除它及其相关约束
-- DO $$
-- BEGIN
--     IF EXISTS (
--         SELECT 1 FROM information_schema.columns
--         WHERE table_name = 'ops_info_points' AND column_name = 'code'
--     ) THEN
--         ALTER TABLE ops_info_points DROP COLUMN code;
--     END IF;
-- END $$;
