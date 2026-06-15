-- Migration: 080_add_dept_code_field.sql
-- Description: 为 sys_dept 表添加 dept_code 字段，用于 Excel 导入时的部门关联
-- Version: 080
-- Date: 2025-01-27

-- 添加 dept_code 字段
ALTER TABLE sys_dept ADD COLUMN IF NOT EXISTS dept_code VARCHAR(50);

-- 添加唯一索引（排除已删除记录）
CREATE UNIQUE INDEX IF NOT EXISTS idx_sys_dept_code
    ON sys_dept(dept_code)
    WHERE deleted_at IS NULL;

-- 添加字段注释
COMMENT ON COLUMN sys_dept.dept_code IS '部门编码，用于 Excel 导入等场景，唯一标识部门';

-- 为现有数据生成默认 dept_code（基于 UUID 的简化编码）
UPDATE sys_dept
SET dept_code = 'DEPT_' || UPPER(SUBSTRING(REPLACE(id::text, '-', ''), 1, 8))
WHERE dept_code IS NULL;

-- 设置 dept_code 为 NOT NULL（已有数据已填充）
ALTER TABLE sys_dept ALTER COLUMN dept_code SET NOT NULL;

-- 记录迁移完成
INSERT INTO schema_migrations (version, description)
VALUES (80, 'add_dept_code_field')
ON CONFLICT (version) DO NOTHING;
