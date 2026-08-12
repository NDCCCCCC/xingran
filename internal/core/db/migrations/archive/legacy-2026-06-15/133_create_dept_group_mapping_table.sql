-- =============================================
-- Department-Group Mapping Tables - Migration
-- Migration version: 133
-- Description: Create tables for tracking relationships between system departments and AD groups
-- =============================================

-- Create main mapping table
CREATE TABLE IF NOT EXISTS sys_dept_group_mapping (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dept_id UUID NOT NULL,
    ad_group_id UUID NOT NULL,
    ad_config_id UUID NOT NULL,
    mapping_type VARCHAR(20) NOT NULL DEFAULT 'auto',
    mapping_status VARCHAR(20) NOT NULL DEFAULT 'active',
    group_dn VARCHAR(500) NOT NULL,
    group_name VARCHAR(255) NOT NULL,
    sync_enabled BOOLEAN NOT NULL DEFAULT true,
    last_sync_at TIMESTAMP,
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    -- Foreign key constraints
    CONSTRAINT fk_dept_group_mapping_dept FOREIGN KEY (dept_id)
        REFERENCES sys_dept(id) ON DELETE CASCADE,
    CONSTRAINT fk_dept_group_mapping_group FOREIGN KEY (ad_group_id)
        REFERENCES sys_ad_group(id) ON DELETE CASCADE,
    CONSTRAINT fk_dept_group_mapping_config FOREIGN KEY (ad_config_id)
        REFERENCES sys_ad_config(id) ON DELETE CASCADE,

    -- Unique constraint: one dept maps to one group
    CONSTRAINT uni_dept_group_mapping UNIQUE (dept_id, ad_group_id, deleted_at)
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_dept_group_dept ON sys_dept_group_mapping(dept_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_dept_group_group ON sys_dept_group_mapping(ad_group_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_dept_group_config ON sys_dept_group_mapping(ad_config_id);
CREATE INDEX IF NOT EXISTS idx_dept_group_status ON sys_dept_group_mapping(mapping_status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_dept_group_deleted ON sys_dept_group_mapping(deleted_at);

-- Add comments for documentation
COMMENT ON TABLE sys_dept_group_mapping IS '部门与AD组映射关系表';
COMMENT ON COLUMN sys_dept_group_mapping.mapping_type IS '映射类型：auto=自动映射，manual=手动映射';
COMMENT ON COLUMN sys_dept_group_mapping.sync_enabled IS '是否启用成员同步到AD组';

-- Create sync log table
CREATE TABLE IF NOT EXISTS sys_dept_group_mapping_sync_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mapping_id UUID NOT NULL,
    dept_id UUID NOT NULL,
    ad_group_id UUID NOT NULL,
    sync_type VARCHAR(50) NOT NULL,
    members_added INTEGER NOT NULL DEFAULT 0,
    members_removed INTEGER NOT NULL DEFAULT 0,
    total_members INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL,
    error_msg TEXT,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    duration_ms INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Foreign key constraint (no cascade - keep logs even if mapping deleted)
    CONSTRAINT fk_mapping_sync_log_mapping FOREIGN KEY (mapping_id)
        REFERENCES sys_dept_group_mapping(id) ON DELETE SET NULL
);

-- Create index for sync log queries
CREATE INDEX IF NOT EXISTS idx_mapping_sync_log_mapping ON sys_dept_group_mapping_sync_log(mapping_id);
CREATE INDEX IF NOT EXISTS idx_mapping_sync_log_dept ON sys_dept_group_mapping_sync_log(dept_id);
CREATE INDEX IF NOT EXISTS idx_mapping_sync_log_status ON sys_dept_group_mapping_sync_log(status);
CREATE INDEX IF NOT EXISTS idx_mapping_sync_log_started ON sys_dept_group_mapping_sync_log(started_at DESC);

-- Add comments for documentation
COMMENT ON TABLE sys_dept_group_mapping_sync_log IS '部门组同步日志表';
COMMENT ON COLUMN sys_dept_group_mapping_sync_log.sync_type IS '同步类型：full=全量，incremental=增量，member_sync=成员同步';
