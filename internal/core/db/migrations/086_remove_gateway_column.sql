-- 086: 删除专线表的 gateway 字段
-- 该字段在原始表定义中存在，但未被实际使用
-- 由于本次 IP 字段拆分重构没有处理 gateway 字段，为保持一致性将其删除

-- 删除 gateway 列（如果存在）
ALTER TABLE ops_dedicated_lines
DROP COLUMN IF EXISTS gateway;
