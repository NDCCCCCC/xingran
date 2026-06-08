-- 调试脚本：检查资产设备类型数据
-- 用途：验证数据库中的设备类型值是否与前端下拉选项匹配

-- 1. 查看所有不同的设备类型值
SELECT
    device_type_name,
    COUNT(*) as count
FROM ops_asset
WHERE device_type_name IS NOT NULL
    AND device_type_name != ''
GROUP BY device_type_name
ORDER BY count DESC;

-- 2. 查看包含特定关键字的设备类型（测试LIKE查询）
SELECT
    device_type_name,
    devicesn,
    device_model_name
FROM ops_asset
WHERE device_type_name LIKE '%台式机%'
LIMIT 10;

-- 3. 查看所有设备类型值（包括NULL和空字符串）
SELECT
    COALESCE(device_type_name, '(NULL)') as device_type_name,
    COUNT(*) as count
FROM ops_asset
GROUP BY device_type_name
ORDER BY count DESC;
