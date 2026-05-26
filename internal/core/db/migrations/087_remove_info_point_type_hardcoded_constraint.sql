-- Migration: 087_remove_info_point_type_hardcoded_constraint.sql
-- Description: 移除信息点类型的硬编码约束，改用字典数据动态验证
-- Date: 2026-05-22
-- Reason: 字典管理中新增的类型（如 PC）无法通过硬编码约束验证

-- 完全移除硬编码的 CHECK 约束
-- 以后由应用层通过字典服务 (sys_dict_data) 进行数据验证
ALTER TABLE ops_info_points DROP CONSTRAINT IF EXISTS chk_info_point_type;

-- 添加注释说明依赖字典验证
COMMENT ON COLUMN ops_info_points.info_point_type IS '信息点类型：参考 sys_dict_data 中 dict_type=ops_info_point_type 的字典值';
