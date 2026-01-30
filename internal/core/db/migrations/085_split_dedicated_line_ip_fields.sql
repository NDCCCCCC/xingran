-- 085: 拆分专线IP地址和子网掩码字段
-- 将 ip_address 拆分为 source_ip_address 和 dest_ip_address
-- 将 subnet_mask 拆分为 source_subnet_mask 和 dest_subnet_mask

-- 添加新字段（如果不存在）
ALTER TABLE ops_dedicated_lines
ADD COLUMN IF NOT EXISTS source_ip_address VARCHAR(50),
ADD COLUMN IF NOT EXISTS dest_ip_address VARCHAR(50),
ADD COLUMN IF NOT EXISTS source_subnet_mask VARCHAR(50),
ADD COLUMN IF NOT EXISTS dest_subnet_mask VARCHAR(50);

-- 迁移现有数据：将原来的 ip_address 内容复制到 source_ip_address
-- 只有当旧字段存在时才执行迁移
DO $$
BEGIN
    -- 检查 ip_address 列是否存在
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'ops_dedicated_lines'
        AND column_name = 'ip_address'
    ) THEN
        -- 迁移 IP 地址数据
        UPDATE ops_dedicated_lines
        SET source_ip_address = ip_address
        WHERE ip_address IS NOT NULL;
    END IF;

    -- 检查 subnet_mask 列是否存在
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'ops_dedicated_lines'
        AND column_name = 'subnet_mask'
    ) THEN
        -- 迁移子网掩码数据
        UPDATE ops_dedicated_lines
        SET source_subnet_mask = subnet_mask
        WHERE subnet_mask IS NOT NULL;
    END IF;
END $$;

-- 删除旧字段（如果存在）
ALTER TABLE ops_dedicated_lines
DROP COLUMN IF EXISTS ip_address,
DROP COLUMN IF EXISTS subnet_mask;

-- 添加注释
COMMENT ON COLUMN ops_dedicated_lines.source_ip_address IS '源IP地址';
COMMENT ON COLUMN ops_dedicated_lines.dest_ip_address IS '目的IP地址';
COMMENT ON COLUMN ops_dedicated_lines.source_subnet_mask IS '源子网掩码';
COMMENT ON COLUMN ops_dedicated_lines.dest_subnet_mask IS '目的子网掩码';
