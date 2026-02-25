-- 099_add_workstation_unique_constraint.sql
-- 添加工位表唯一约束，支持 Excel 导入的 ON CONFLICT 功能
-- 约束：同一楼层下工位名称唯一

-- 添加唯一索引（排除已删除的记录）
CREATE UNIQUE INDEX IF NOT EXISTS sys_workstation_floor_name_idx
ON sys_workstation (floor_id, workstation_name)
WHERE deleted_at IS NULL;

-- 添加注释
COMMENT ON INDEX sys_workstation_floor_name_idx IS '同一楼层下工位名称唯一（软删除记录除外）';
