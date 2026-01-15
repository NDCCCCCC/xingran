-- 删除工位表的工位编码和容量字段
-- 创建时间: 2025-01-14
-- 说明: 简化工位管理，去掉工位编码和容量字段

-- 删除 workstation_code 列（如果存在）
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'sys_workstation' AND column_name = 'workstation_code'
    ) THEN
        ALTER TABLE sys_workstation DROP COLUMN workstation_code;
    END IF;
END $$;

-- 删除 capacity 列（如果存在）
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'sys_workstation' AND column_name = 'capacity'
    ) THEN
        ALTER TABLE sys_workstation DROP COLUMN capacity;
    END IF;
END $$;
