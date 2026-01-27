-- Migration: 080_add_dept_code_field.sql
-- Description: 为sys_dept表添加dept_code字段，用于Excel导入时的部门关联
-- Date: 2025-01-27

-- 添加dept_code字段
ALTER TABLE sys_dept ADD COLUMN IF NOT EXISTS dept_code VARCHAR(50);

-- 添加唯一索引，确保部门编码唯一（排除已删除记录）
CREATE UNIQUE INDEX IF NOT EXISTS idx_sys_dept_code
ON sys_dept(dept_code)
WHERE deleted_at IS NULL;

-- 添加注释
COMMENT ON COLUMN sys_dept.dept_code IS '部门编码，用于Excel导入等场景，唯一标识部门';

-- 为现有数据生成默认dept_code（基于ID的简化编码）
-- 注意：实际使用时应该由管理员设置有意义的部门编码
UPDATE sys_dept
SET dept_code = 'DEPT_' || UPPER(SUBSTRING(encode(id, 'base64'), 1, 8))
WHERE dept_code IS NULL;

-- 设置dept_code为NOT NULL（已有数据已填充）
ALTER TABLE sys_dept ALTER COLUMN dept_code SET NOT NULL;

-- 记录迁移完成
INSERT INTO schema_migrations (version, description)
VALUES (80, 'add_dept_code_field')
ON CONFLICT (version) DO NOTHING;
