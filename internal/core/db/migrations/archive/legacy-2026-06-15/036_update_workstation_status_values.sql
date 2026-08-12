-- Phase 32 P2-A4 source-tracking:
--   Original commit: a3032b2e
--   Created: 2026-01-16
--   Note: No direct filename conflict — listed for completeness. Note: gap between 031 and 036 due to Go-based migrations (032-035). Runner uses Go code ordering.

-- 更新工位状态值和注释
-- Migration: 036
-- Description: 更新工位状态枚举：0=空闲, 1=占用, 2=维护

-- 更新状态列的注释
COMMENT ON COLUMN sys_workstation.status IS '工位状态：0=空闲可分配, 1=占用已分配, 2=维护中不可用，默认为0';

-- 添加检查约束确保状态值在有效范围内
ALTER TABLE sys_workstation
DROP CONSTRAINT IF EXISTS sys_workstation_status_check;

ALTER TABLE sys_workstation
ADD CONSTRAINT sys_workstation_status_check
CHECK (status IN (0, 1, 2));
