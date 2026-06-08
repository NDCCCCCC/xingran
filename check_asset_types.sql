-- 检查数据库中的设备类型分布
SELECT 
    COALESCE(device_type_name, '(NULL)') as device_type_name,
    COUNT(*) as count
FROM ops_asset
GROUP BY device_type_name
ORDER BY count DESC;
