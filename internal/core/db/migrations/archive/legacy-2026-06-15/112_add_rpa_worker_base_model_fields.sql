-- ============================================
-- 112_add_rpa_worker_base_model_fields.sql
-- 说明: 添加 BaseModel 缺失的字段到 sys_rpa_workers 表
-- ============================================

-- 添加缺失的 BaseModel 字段
ALTER TABLE sys_rpa_workers
ADD COLUMN IF NOT EXISTS created_by VARCHAR(64),
ADD COLUMN IF NOT EXISTS updated_by VARCHAR(64),
ADD COLUMN IF NOT EXISTS version INTEGER DEFAULT 0;

SELECT '112_add_rpa_worker_base_model_fields.sql migration completed' AS status;
