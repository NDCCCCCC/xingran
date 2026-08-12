-- Migration: 049_remove_dedicated_line_contract_no.sql
-- Description: 删除专线管理表的合同编号字段
-- Date: 2026-01-21

-- 删除 ops_dedicated_lines 表的 contract_no 字段
ALTER TABLE ops_dedicated_lines DROP COLUMN IF EXISTS contract_no;
