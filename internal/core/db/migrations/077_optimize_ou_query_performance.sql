-- =============================================
-- 优化OU查询性能
-- 迁移版本: 077
-- 描述: 为 sys_ad_ou 表的 ou_dn 字段添加索引以提升IN查询性能
-- =============================================

-- 添加索引（如果不存在）
CREATE INDEX IF NOT EXISTS idx_ad_ou_dn
ON sys_ad_ou (ad_config_id, ou_dn);

-- 验证索引
SELECT '077_optimize_ou_query_performance.sql migration completed' AS status;