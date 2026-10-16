-- VDI 本地数据库检查脚本
-- 用于检查 VDI 服务器配置、资源组和虚拟机数据

-- 1. 检查 VDI 服务器配置
SELECT '=== VDI 服务器配置 ===' as section;
SELECT
    id,
    name,
    endpoint,
    username,
    status,
    auth_token IS NOT NULL as has_token,
    token_expiry,
    created_at,
    updated_at
FROM sys_vdi_server
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- 2. 检查本地虚拟机数据统计
SELECT '=== 虚拟机数据统计 ===' as section;
SELECT
    COUNT(*) as total_vms,
    COUNT(DISTINCT resource_id) as unique_resources,
    COUNT(CASE WHEN status = 0 THEN 1 END) as enabled_vms,
    COUNT(CASE WHEN status = 1 THEN 1 END) as disabled_vms
FROM sys_vdi_virtual_machine
WHERE deleted_at IS NULL;

-- 3. 检查虚拟机数据详情（前 10 条）
SELECT '=== 虚拟机数据详情 (前10条) ===' as section;
SELECT
    vm_id,
    name,
    resource_id,
    status,
    power_state,
    ip_address,
    os_type,
    cpu,
    memory,
    disk,
    created_at,
    updated_at
FROM sys_vdi_virtual_machine
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 10;

-- 4. 检查是否有虚拟机数据
SELECT '=== 检查是否有虚拟机数据 ===' as section;
SELECT
    CASE
        WHEN COUNT(*) > 0 THEN '有数据'
        ELSE '无数据'
    END as vm_data_status,
    COUNT(*) as count
FROM sys_vdi_virtual_machine
WHERE deleted_at IS NULL;

-- 5. 检查最近同步时间
SELECT '=== 最近同步时间 ===' as section;
SELECT
    MAX(updated_at) as last_sync_time,
    MAX(created_at) as last_create_time
FROM sys_vdi_virtual_machine
WHERE deleted_at IS NULL;

-- 6. 检查 Token 有效性
SELECT '=== Token 有效性检查 ===' as section;
SELECT
    id,
    name,
    CASE
        WHEN token_expiry IS NULL THEN '无 Token'
        WHEN token_expiry > NOW() THEN 'Token 有效'
        ELSE 'Token 已过期'
    END as token_status,
    token_expiry,
    CASE
        WHEN token_expiry IS NOT NULL THEN EXTRACT(EPOCH FROM (token_expiry - NOW())) / 60 as minutes_remaining
        ELSE NULL
    END
FROM sys_vdi_server
WHERE deleted_at IS NULL;
