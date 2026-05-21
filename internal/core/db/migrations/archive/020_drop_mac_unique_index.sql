-- 删除MAC地址表的唯一索引
-- 迁移编号: 020
-- 描述: 删除sys_device_mac_address表的唯一索引，允许同一MAC地址在多次采集中存在

-- 删除唯一索引（如果存在）
DROP INDEX IF EXISTS idx_sys_device_mac_address_unique;

-- 说明：
-- 1. 历史数据中存在重复的MAC地址记录
-- 2. 同一设备接口可能在不同时间采集到相同的MAC地址
-- 3. 保留普通索引 idx_sys_device_mac_address_composite 以保证查询性能
