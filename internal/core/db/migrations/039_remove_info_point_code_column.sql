-- 删除信息点编码字段
-- 信息点编码字段不再需要，使用名称即可

-- 删除 code 字段
ALTER TABLE ops_info_points
DROP COLUMN IF EXISTS code;
