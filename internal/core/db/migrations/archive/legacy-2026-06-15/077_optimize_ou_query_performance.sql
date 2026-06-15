-- Migration: 077_optimize_ou_query_performance.sql
-- Description: 为 sys_ad_ou 表添加复合索引以提升OU查询性能
-- Version: 077
-- Date: 2025-01-27

CREATE INDEX IF NOT EXISTS idx_ad_ou_dn
    ON sys_ad_ou (ad_config_id, ou_dn);