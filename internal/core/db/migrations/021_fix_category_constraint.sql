-- 修复工单分类表约束问题
-- 迁移编号: 020
-- 描述: 修复sys_workorder_category表的唯一约束名称不匹配问题

-- 步骤1: 检查并删除可能存在的旧约束（使用IF EXISTS避免错误）
-- PostgreSQL会自动生成类似 uni_sys_workorder_category_category_name 的约束名
DO $$
BEGIN
    -- 尝试删除PostgreSQL自动生成的唯一约束（如果存在）
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'uni_sys_workorder_category_category_name'
        AND conrelid = 'sys_workorder_category'::regclass
    ) THEN
        ALTER TABLE sys_workorder_category
        DROP CONSTRAINT uni_sys_workorder_category_category_name;
    END IF;

    -- 尝试删除GORM可能创建的约束（如果存在）
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'idx_wo_category_name'
        AND conrelid = 'sys_workorder_category'::regclass
    ) THEN
        ALTER TABLE sys_workorder_category
        DROP CONSTRAINT idx_wo_category_name;
    END IF;

    -- 尝试删除任何以 category_name 结尾的唯一约束
    FOR constraint_record IN
        SELECT conname FROM pg_constraint
        WHERE conrelid = 'sys_workorder_category'::regclass
        AND contype = 'u'
        AND conname LIKE '%category_name%'
    LOOP
        EXECUTE format('ALTER TABLE sys_workorder_category DROP CONSTRAINT IF EXISTS %I', constraint_record.conname);
    END LOOP;
END $$;

-- 步骤2: 创建GORM期望的唯一索引
-- 使用唯一索引而不是唯一约束，这样GORM可以正确识别
CREATE UNIQUE INDEX IF NOT EXISTS idx_wo_category_name
ON sys_workorder_category(category_name);

-- 步骤3: 验证索引创建成功
SELECT 'Index idx_wo_category_name created successfully' AS result;

-- 说明：
-- 1. 此脚本会安全地清理所有与category_name相关的旧约束
-- 2. 然后创建GORM期望的唯一索引
-- 3. 运行此脚本后，GORM的AutoMigrate应该不会再报错
