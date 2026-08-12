-- =============================================
-- 修复电脑设备表唯一约束
-- 迁移版本: 074
-- 描述: 将 computer_name 的全局唯一约束改为 (ad_config_id, computer_name) 组合唯一约束
--       这样同一 AD 配置下 computer_name 唯一，但不同 AD 配置可以有相同的 computer_name
-- =============================================

-- ================================
-- 1. 删除旧的唯一约束
-- ================================
ALTER TABLE sys_ad_computer
DROP CONSTRAINT IF EXISTS uni_sys_ad_computer_computer_name;

-- ================================
-- 2. 创建新的组合唯一约束
-- ================================
ALTER TABLE sys_ad_computer
ADD CONSTRAINT uni_sys_ad_computer_config_name
UNIQUE (ad_config_id, computer_name);

-- ================================
-- 验证迁移结果
-- ================================
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'uni_sys_ad_computer_config_name'
    ) THEN
        RAISE NOTICE '074_fix_computer_unique_constraint.sql migration completed successfully';
    ELSE
        RAISE EXCEPTION 'Migration failed: unique constraint was not created';
    END IF;
END $$;
