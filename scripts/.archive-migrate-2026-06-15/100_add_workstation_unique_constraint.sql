-- 100_add_workstation_unique_constraint.sql
-- 使用命名唯一约束（不带 WHERE 条件）

-- 先删除可能存在的旧索引
DROP INDEX IF EXISTS sys_workstation_floor_name_idx;
DROP CONSTRAINT IF EXISTS sys_workstation_floor_name_unique;

-- 添加唯一约束（应用于所有行）
ALTER TABLE sys_workstation
ADD CONSTRAINT sys_workstation_floor_name_unique
UNIQUE (floor_id, workstation_name);

-- 添加注释
COMMENT ON CONSTRAINT sys_workstation_floor_name_unique ON sys_workstation IS '同一楼层下工位名称唯一';
