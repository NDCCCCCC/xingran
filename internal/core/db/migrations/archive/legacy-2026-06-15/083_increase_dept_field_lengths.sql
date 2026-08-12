-- Migration: 083_increase_dept_field_lengths.sql
-- Description: 增加 sys_dept 表字段长度限制，支持三级部门结构的长编码
-- Version: 083
-- Date: 2025-01-29

-- 增加 dept_code 字段长度（50 -> 100）
ALTER TABLE sys_dept ALTER COLUMN dept_code TYPE VARCHAR(100);

-- 增加 dept_name 字段长度（50 -> 100）
ALTER TABLE sys_dept ALTER COLUMN dept_name TYPE VARCHAR(100);

-- 增加 email 字段长度（50 -> 100）
ALTER TABLE sys_dept ALTER COLUMN email TYPE VARCHAR(100);

-- 增加 leader 字段长度（36 -> 100）
ALTER TABLE sys_dept ALTER COLUMN leader TYPE VARCHAR(100);

-- 增加 phone 字段长度（20 -> 50）
ALTER TABLE sys_dept ALTER COLUMN phone TYPE VARCHAR(50);

-- 记录迁移完成
INSERT INTO schema_migrations (version, description)
VALUES (83, 'increase_dept_field_lengths')
ON CONFLICT (version) DO NOTHING;
