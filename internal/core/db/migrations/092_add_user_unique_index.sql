-- 092: 为 AD 用户表、组表和电脑表添加索引以优化查询和 upsert 性能
-- Date: 2026-02-01
-- Description: 添加索引以优化批量 upsert 性能和查询性能

-- ==================== 用户表 ====================

-- 检查是否存在同名索引，如果存在则删除
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE tablename = 'sys_ad_user'
        AND indexname = 'uni_sys_ad_user_config_dn'
    ) THEN
        DROP INDEX uni_sys_ad_user_config_dn;
    END IF;
END $$;

-- 创建唯一索引 (ad_config_id, user_dn)
-- 这个索引将大幅提升批量 upsert 的性能
CREATE UNIQUE INDEX uni_sys_ad_user_config_dn
    ON sys_ad_user (ad_config_id, user_dn);

-- 添加注释
COMMENT ON INDEX uni_sys_ad_user_config_dn IS '唯一约束：同一AD配置下的用户DN必须唯一，用于优化批量upsert性能';

-- ==================== 用户组表 ====================

-- 检查是否存在同名索引，如果存在则删除
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE tablename = 'sys_ad_group'
        AND indexname = 'uni_sys_ad_group_config_dn'
    ) THEN
        DROP INDEX uni_sys_ad_group_config_dn;
    END IF;
END $$;

-- 创建唯一索引 (ad_config_id, group_dn)
CREATE UNIQUE INDEX uni_sys_ad_group_config_dn
    ON sys_ad_group (ad_config_id, group_dn);

-- 添加注释
COMMENT ON INDEX uni_sys_ad_group_config_dn IS '唯一约束：同一AD配置下的组DN必须唯一，用于优化批量upsert性能';

-- ==================== 电脑表 ====================

-- 检查是否存在同名索引，如果存在则删除
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE tablename = 'sys_ad_computer'
        AND indexname = 'idx_ad_computer_dn'
    ) THEN
        DROP INDEX idx_ad_computer_dn;
    END IF;
END $$;

-- 创建索引 (ad_config_id, distinguished_name)
-- 这个索引将大幅提升按 DN 查询的性能（IN 查询包含数千个 DN 值）
CREATE INDEX idx_ad_computer_dn
    ON sys_ad_computer (ad_config_id, distinguished_name);

-- 添加注释
COMMENT ON INDEX idx_ad_computer_dn IS '查询索引：优化按配置ID和DN查询电脑设备的性能';
