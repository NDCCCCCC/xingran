-- 140: Add last_sync_time column to sys_vdi_server table
-- This column tracks the last time VDI server synchronization was performed

ALTER TABLE sys_vdi_server
ADD COLUMN IF NOT EXISTS last_sync_time TIMESTAMP NULL;

COMMENT ON COLUMN sys_vdi_server.last_sync_time IS 'VDI服务器上次同步时间';
