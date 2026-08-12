-- 114_add_building_name_unique_constraint.sql
-- 添加楼宇表的唯一约束，防止同一机构下存在同名楼宇
-- 约束：同一机构下楼宇名称唯一

-- 添加唯一约束（排除已删除的记录）
CREATE UNIQUE INDEX IF NOT EXISTS ops_buildings_org_name_idx
ON ops_buildings (org_id, name)
WHERE deleted_at IS NULL;

-- 添加注释
COMMENT ON INDEX ops_buildings_org_name_idx IS '同一机构下楼宇名称唯一（软删除记录除外）';
