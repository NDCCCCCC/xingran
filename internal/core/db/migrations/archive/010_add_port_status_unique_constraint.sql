-- ==========================================
-- 为端口状态表添加唯一约束
-- 用于支持批量 upsert 操作，提升性能
-- ==========================================

-- 步骤1: 删除重复记录（保留最新的一条）
-- 使用 CTE 找出每个 (device_id, interface_name) 组合中需要保留的记录（ID最大的）
WITH ranked_ports AS (
    SELECT
        id,
        device_id,
        interface_name,
        ROW_NUMBER() OVER (
            PARTITION BY device_id, interface_name
            ORDER BY created_at DESC, id DESC
        ) AS rn
    FROM sys_device_port_status
)
DELETE FROM sys_device_port_status
WHERE id IN (
    SELECT id FROM ranked_ports WHERE rn > 1
);

-- 步骤2: 添加唯一约束
-- 注意: 如果有重复记录，上面的步骤会先清理它们
ALTER TABLE sys_device_port_status
ADD CONSTRAINT IF NOT EXISTS uniq_device_interface
UNIQUE (device_id, interface_name);

-- 步骤3: 添加注释
COMMENT ON CONSTRAINT uniq_device_interface ON sys_device_port_status
IS '确保同一设备的同一端口只有一条最新状态记录';
