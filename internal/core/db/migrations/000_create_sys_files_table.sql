-- 通用文件表初始创建
-- 文件: 000_create_sys_files_table.sql
-- 说明: 创建sys_files通用文件管理表和sys_file_access_logs访问日志表

-- ============================================
-- 1. 创建通用文件表 sys_files
-- ============================================

CREATE TABLE IF NOT EXISTS sys_files (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_name       VARCHAR(255) NOT NULL,
    file_size       BIGINT NOT NULL,
    file_type       VARCHAR(100),
    extension       VARCHAR(50) NOT NULL,
    storage_path    VARCHAR(500) NOT NULL,
    file_hash       VARCHAR(64) NOT NULL,
    uploader_id     UUID NOT NULL,
    business_type   VARCHAR(50) NOT NULL,
    is_deleted      BOOLEAN DEFAULT FALSE,
    delete_time     TIMESTAMP,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 添加表注释
COMMENT ON TABLE sys_files IS '通用文件管理表';
COMMENT ON COLUMN sys_files.id IS '主键ID';
COMMENT ON COLUMN sys_files.file_name IS '原始文件名';
COMMENT ON COLUMN sys_files.file_size IS '文件大小（字节）';
COMMENT ON COLUMN sys_files.file_type IS '文件MIME类型';
COMMENT ON COLUMN sys_files.extension IS '文件扩展名';
COMMENT ON COLUMN sys_files.storage_path IS '存储路径（相对路径）';
COMMENT ON COLUMN sys_files.file_hash IS '文件SHA256哈希值，用于去重';
COMMENT ON COLUMN sys_files.uploader_id IS '上传者ID';
COMMENT ON COLUMN sys_files.business_type IS '业务类型（avatar/room-photo/document/import/export等）';
COMMENT ON COLUMN sys_files.is_deleted IS '是否已删除（软删除）';
COMMENT ON COLUMN sys_files.delete_time IS '删除时间';
COMMENT ON COLUMN sys_files.created_at IS '创建时间';
COMMENT ON COLUMN sys_files.updated_at IS '更新时间';

-- 添加唯一约束（文件哈希去重，未删除的文件）
CREATE UNIQUE INDEX IF NOT EXISTS idx_files_hash_undeleted
    ON sys_files(file_hash)
    WHERE is_deleted = FALSE;

-- 添加业务类型索引
CREATE INDEX IF NOT EXISTS idx_files_business_type
    ON sys_files(business_type);

-- 添加上传者索引
CREATE INDEX IF NOT EXISTS idx_files_uploader
    ON sys_files(uploader_id);

-- 添加软删除过滤索引
CREATE INDEX IF NOT EXISTS idx_files_is_deleted
    ON sys_files(is_deleted);

-- ============================================
-- 2. 创建文件访问日志表 sys_file_access_logs
-- ============================================

CREATE TABLE IF NOT EXISTS sys_file_access_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id         UUID NOT NULL,
    action_type     VARCHAR(20) NOT NULL,
    user_id         VARCHAR(64),
    user_name       VARCHAR(64),
    ip_address      VARCHAR(128),
    user_agent      VARCHAR(500),
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 添加表注释
COMMENT ON TABLE sys_file_access_logs IS '文件访问日志表';
COMMENT ON COLUMN sys_file_access_logs.id IS '主键ID';
COMMENT ON COLUMN sys_file_access_logs.file_id IS '文件ID';
COMMENT ON COLUMN sys_file_access_logs.action_type IS '操作类型（upload/download/delete/view）';
COMMENT ON COLUMN sys_file_access_logs.user_id IS '操作者ID';
COMMENT ON COLUMN sys_file_access_logs.user_name IS '操作者姓名';
COMMENT ON COLUMN sys_file_access_logs.ip_address IS 'IP地址';
COMMENT ON COLUMN sys_file_access_logs.user_agent IS '浏览器信息';
COMMENT ON COLUMN sys_file_access_logs.created_at IS '操作时间';

-- 添加外键约束
ALTER TABLE sys_file_access_logs DROP CONSTRAINT IF EXISTS fk_file_logs_file;
ALTER TABLE sys_file_access_logs ADD CONSTRAINT fk_file_logs_file
    FOREIGN KEY (file_id) REFERENCES sys_files(id) ON DELETE CASCADE;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_file_logs_file_id
    ON sys_file_access_logs(file_id);
CREATE INDEX IF NOT EXISTS idx_file_logs_action_type
    ON sys_file_access_logs(action_type);
CREATE INDEX IF NOT EXISTS idx_file_logs_user_id
    ON sys_file_access_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_file_logs_created_at
    ON sys_file_access_logs(created_at DESC);

-- ============================================
-- 迁移完成
-- ============================================

-- 验证迁移
SELECT '000_create_sys_files_table.sql migration completed' AS status;

-- 检查表是否创建成功
SELECT table_name, (SELECT COUNT(*) FROM information_schema.columns WHERE table_name = t.table_name) AS column_count
FROM information_schema.tables t
WHERE table_schema = 'public'
AND table_name IN ('sys_files', 'sys_file_access_logs');
