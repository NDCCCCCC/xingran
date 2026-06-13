-- Phase 32 P2-A4 source-tracking:
--   Original commit: a3032b2e
--   Created: 2026-01-16
--   Note: Conflicts with 030_add_building_spaces_3d_menu.sql and 030_create_workstation_device.sql — three files share prefix 030. Runner uses Go code ordering, not filename sort; conflict is harmless.

-- 增强系统工位表，添加运维管理所需字段
-- 执行日期: 2026-01-14

-- 添加楼层ID字段（关联ops_floors）
ALTER TABLE sys_workstation ADD COLUMN IF NOT EXISTS floor_id VARCHAR(64);

-- 添加楼层名称字段
ALTER TABLE sys_workstation ADD COLUMN IF NOT EXISTS floor_name VARCHAR(100);

-- 添加电话信息点字段
ALTER TABLE sys_workstation ADD COLUMN IF NOT EXISTS phone_point VARCHAR(100);

-- 添加设备配置字段（JSON格式）
ALTER TABLE sys_workstation ADD COLUMN IF NOT EXISTS equipment VARCHAR(500);

-- 添加网络信息字段
ALTER TABLE sys_workstation ADD COLUMN IF NOT EXISTS network_info VARCHAR(500);

-- 添加平面图X坐标字段
ALTER TABLE sys_workstation ADD COLUMN IF NOT EXISTS position_x INTEGER;

-- 添加平面图Y坐标字段
ALTER TABLE sys_workstation ADD COLUMN IF NOT EXISTS position_y INTEGER;

-- 为新字段添加注释
COMMENT ON COLUMN sys_workstation.floor_id IS '所属楼层ID（关联ops_floors）';
COMMENT ON COLUMN sys_workstation.floor_name IS '所属楼层名称';
COMMENT ON COLUMN sys_workstation.phone_point IS '电话信息点';
COMMENT ON COLUMN sys_workstation.equipment IS '设备配置（JSON格式）';
COMMENT ON COLUMN sys_workstation.network_info IS '网络信息';
COMMENT ON COLUMN sys_workstation.position_x IS '平面图X坐标';
COMMENT ON COLUMN sys_workstation.position_y IS '平面图Y坐标';

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_sys_workstation_floor_id ON sys_workstation(floor_id) WHERE floor_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sys_workstation_user_id ON sys_workstation(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sys_workstation_dept_id ON sys_workstation(dept_id) WHERE dept_id IS NOT NULL;
