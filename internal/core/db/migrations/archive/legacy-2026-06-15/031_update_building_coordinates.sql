-- Phase 32 P2-A4 source-tracking:
--   Original commit: a3032b2e
--   Created: 2026-01-16
--   Note: Conflicts with 031_enhance_server_room_table.sql — both share prefix 031. Runner uses Go code ordering, not filename sort; conflict is harmless.

-- 更新楼宇地址和坐标信息
-- 注意：经纬度为根据地址估算的值，如需精确坐标请在百度地图开放平台启用 Geocoding API

-- 1. 浙商大厦：武汉市江汉区建设大道718号
UPDATE ops_buildings
SET
    address = '武汉市江汉区建设大道718号浙商大厦',
    longitude = 114.2715,
    latitude = 30.5952,
    city_code = 'wuhan',
    city_name = '武汉市',
    updated_at = NOW()
WHERE name LIKE '%浙商大厦%' OR name LIKE '%浙商%';

-- 2. 瑞通广场：湖北省武汉市江汉区建设大道847号
UPDATE ops_buildings
SET
    address = '湖北省武汉市江汉区建设大道847号瑞通广场-B座',
    longitude = 114.2748,
    latitude = 30.5928,
    city_code = 'wuhan',
    city_name = '武汉市',
    updated_at = NOW()
WHERE name LIKE '%瑞通广场%' OR name LIKE '%瑞通%';

-- 查询更新结果
SELECT
    id,
    name,
    code,
    city_name,
    address,
    longitude,
    latitude,
    status
FROM ops_buildings
WHERE name LIKE '%浙商%' OR name LIKE '%瑞通%'
ORDER BY name;
