-- 添加资产管理新增的 4 个字段
-- 执行时间: 2026-06-08

-- 1. 子用途
ALTER TABLE ops_asset ADD COLUMN IF NOT EXISTS sub_using_type_name VARCHAR(100);

-- 2. 存放地址
ALTER TABLE ops_asset ADD COLUMN IF NOT EXISTS storage_address VARCHAR(200);

-- 3. 最近一次盘点时间
ALTER TABLE ops_asset ADD COLUMN IF NOT EXISTS last_inventory_date TIMESTAMP;

-- 4. 盘点结果
ALTER TABLE ops_asset ADD COLUMN IF NOT EXISTS inventory_result VARCHAR(50);
