-- 009_create_sys_files_table.up.sql
CREATE TABLE IF NOT EXISTS sys_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0,

    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    file_type VARCHAR(100),
    extension VARCHAR(20),
    storage_path VARCHAR(500) NOT NULL,
    file_hash VARCHAR(64),
    uploader_id UUID NOT NULL,
    business_type VARCHAR(50),
    is_deleted BOOLEAN DEFAULT FALSE,
    delete_time TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_sys_files_uploader_id ON sys_files(uploader_id);
CREATE INDEX idx_sys_files_business_type ON sys_files(business_type);
CREATE INDEX idx_sys_files_file_hash ON sys_files(file_hash);
CREATE INDEX idx_sys_files_deleted_at ON sys_files(deleted_at);

-- 添加注释
COMMENT ON TABLE sys_files IS '文件管理表';
COMMENT ON COLUMN sys_files.file_hash IS '文件哈希(SHA256)用于去重';
COMMENT ON COLUMN sys_files.business_type IS '业务类型: room_photo/import/export等';
