-- 删除专线编码字段
-- 专线编码字段不再需要，使用名称即可

-- 删除 line_code 字段
ALTER TABLE ops_dedicated_lines
DROP COLUMN IF EXISTS line_code;

-- 删除相关唯一索引（如果存在）
DROP INDEX IF EXISTS idx_ops_dedicated_lines_line_code;
