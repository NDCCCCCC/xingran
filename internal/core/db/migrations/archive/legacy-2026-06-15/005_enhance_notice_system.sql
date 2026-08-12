-- 企业级通知公告系统数据库迁移
-- 文件: 005_enhance_notice_system.sql
-- 说明: 扩展sys_notice表并创建关联表，支持定向推送、阅读统计、附件等功能

-- ============================================
-- 1. 扩展通知主表 sys_notice
-- ============================================

-- 优先级: 0=普通 1=重要 2=紧急
ALTER TABLE sys_notice ADD COLUMN IF NOT EXISTS priority INT DEFAULT 0;

-- 定时发布时间
ALTER TABLE sys_notice ADD COLUMN IF NOT EXISTS publish_time TIMESTAMP;

-- 发布状态: 0=草稿 1=已发布 2=定时发布中 3=已撤回
ALTER TABLE sys_notice ADD COLUMN IF NOT EXISTS publish_status INT DEFAULT 0;

-- 目标类型: 0=全部用户 1=指定部门 2=指定角色 3=指定用户
ALTER TABLE sys_notice ADD COLUMN IF NOT EXISTS target_type INT DEFAULT 0;

-- 创建人姓名（冗余存储，提升查询性能）
ALTER TABLE sys_notice ADD COLUMN IF NOT EXISTS created_by_name VARCHAR(64);

-- 是否为Markdown格式
ALTER TABLE sys_notice ADD COLUMN IF NOT EXISTS is_markdown BOOLEAN DEFAULT FALSE;

-- 添加索引优化查询性能
CREATE INDEX IF NOT EXISTS idx_notice_publish_status ON sys_notice(publish_status);
CREATE INDEX IF NOT EXISTS idx_notice_publish_time ON sys_notice(publish_time);
CREATE INDEX IF NOT EXISTS idx_notice_priority ON sys_notice(priority);
CREATE INDEX IF NOT EXISTS idx_notice_target_type ON sys_notice(target_type);

-- 添加字段注释
COMMENT ON COLUMN sys_notice.priority IS '优先级: 0=普通 1=重要 2=紧急';
COMMENT ON COLUMN sys_notice.publish_time IS '定时发布时间';
COMMENT ON COLUMN sys_notice.publish_status IS '发布状态: 0=草稿 1=已发布 2=定时发布中 3=已撤回';
COMMENT ON COLUMN sys_notice.target_type IS '目标类型: 0=全部用户 1=指定部门 2=指定角色 3=指定用户';
COMMENT ON COLUMN sys_notice.created_by_name IS '创建人姓名';
COMMENT ON COLUMN sys_notice.is_markdown IS '是否为Markdown格式';

-- ============================================
-- 2. 创建通知接收范围表 sys_notice_target
-- ============================================

CREATE TABLE IF NOT EXISTS sys_notice_target (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notice_id UUID NOT NULL,
    target_type VARCHAR(20) NOT NULL,
    target_id UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uni_sys_notice_target_notice_type_target UNIQUE (notice_id, target_type, target_id)
);

-- 添加外键约束
ALTER TABLE sys_notice_target DROP CONSTRAINT IF EXISTS fk_notice_target_notice;
ALTER TABLE sys_notice_target ADD CONSTRAINT fk_notice_target_notice
    FOREIGN KEY (notice_id) REFERENCES sys_notice(id) ON DELETE CASCADE;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_notice_target_notice_id ON sys_notice_target(notice_id);
CREATE INDEX IF NOT EXISTS idx_notice_target_target ON sys_notice_target(target_type, target_id);

-- 添加表和字段注释
COMMENT ON TABLE sys_notice_target IS '通知接收范围表';
COMMENT ON COLUMN sys_notice_target.id IS '主键ID';
COMMENT ON COLUMN sys_notice_target.notice_id IS '通知ID';
COMMENT ON COLUMN sys_notice_target.target_type IS '目标类型: dept/role/user';
COMMENT ON COLUMN sys_notice_target.target_id IS '目标ID（部门ID/角色ID/用户ID）';
COMMENT ON COLUMN sys_notice_target.created_at IS '创建时间';

-- ============================================
-- 3. 创建通知阅读记录表 sys_notice_read
-- ============================================

CREATE TABLE IF NOT EXISTS sys_notice_read (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notice_id UUID NOT NULL,
    user_id UUID NOT NULL,
    read_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    read_ip VARCHAR(128),
    CONSTRAINT uni_sys_notice_read_notice_user UNIQUE (notice_id, user_id)
);

-- 添加外键约束
ALTER TABLE sys_notice_read DROP CONSTRAINT IF EXISTS fk_notice_read_notice;
ALTER TABLE sys_notice_read ADD CONSTRAINT fk_notice_read_notice
    FOREIGN KEY (notice_id) REFERENCES sys_notice(id) ON DELETE CASCADE;

ALTER TABLE sys_notice_read DROP CONSTRAINT IF EXISTS fk_notice_read_user;
ALTER TABLE sys_notice_read ADD CONSTRAINT fk_notice_read_user
    FOREIGN KEY (user_id) REFERENCES sys_user(id) ON DELETE CASCADE;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_notice_read_notice_id ON sys_notice_read(notice_id);
CREATE INDEX IF NOT EXISTS idx_notice_read_user_id ON sys_notice_read(user_id);
CREATE INDEX IF NOT EXISTS idx_notice_read_read_at ON sys_notice_read(read_at);

-- 添加表和字段注释
COMMENT ON TABLE sys_notice_read IS '通知阅读记录表';
COMMENT ON COLUMN sys_notice_read.id IS '主键ID';
COMMENT ON COLUMN sys_notice_read.notice_id IS '通知ID';
COMMENT ON COLUMN sys_notice_read.user_id IS '用户ID';
COMMENT ON COLUMN sys_notice_read.read_at IS '阅读时间';
COMMENT ON COLUMN sys_notice_read.read_ip IS '阅读IP地址';

-- ============================================
-- 4. 创建通知附件表 sys_notice_attachment
-- ============================================

CREATE TABLE IF NOT EXISTS sys_notice_attachment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notice_id UUID NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    file_type VARCHAR(100),
    upload_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    uploaded_by UUID
);

-- 添加外键约束
ALTER TABLE sys_notice_attachment DROP CONSTRAINT IF EXISTS fk_notice_attachment_notice;
ALTER TABLE sys_notice_attachment ADD CONSTRAINT fk_notice_attachment_notice
    FOREIGN KEY (notice_id) REFERENCES sys_notice(id) ON DELETE CASCADE;

ALTER TABLE sys_notice_attachment DROP CONSTRAINT IF EXISTS fk_notice_attachment_user;
ALTER TABLE sys_notice_attachment ADD CONSTRAINT fk_notice_attachment_user
    FOREIGN KEY (uploaded_by) REFERENCES sys_user(id) ON DELETE SET NULL;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_notice_attachment_notice_id ON sys_notice_attachment(notice_id);
CREATE INDEX IF NOT EXISTS idx_notice_attachment_uploaded_by ON sys_notice_attachment(uploaded_by);

-- 添加表和字段注释
COMMENT ON TABLE sys_notice_attachment IS '通知附件表';
COMMENT ON COLUMN sys_notice_attachment.id IS '主键ID';
COMMENT ON COLUMN sys_notice_attachment.notice_id IS '通知ID';
COMMENT ON COLUMN sys_notice_attachment.file_name IS '文件名';
COMMENT ON COLUMN sys_notice_attachment.file_path IS '文件路径';
COMMENT ON COLUMN sys_notice_attachment.file_size IS '文件大小（字节）';
COMMENT ON COLUMN sys_notice_attachment.file_type IS '文件MIME类型';
COMMENT ON COLUMN sys_notice_attachment.upload_time IS '上传时间';
COMMENT ON COLUMN sys_notice_attachment.uploaded_by IS '上传者ID';

-- ============================================
-- 迁移完成
-- ============================================

-- 验证迁移
SELECT '005_enhance_notice_system.sql migration completed' AS status;

-- 检查新增列
SELECT column_name, data_type, column_default
FROM information_schema.columns
WHERE table_name = 'sys_notice'
AND column_name IN ('priority', 'publish_time', 'publish_status', 'target_type', 'created_by_name', 'is_markdown')
ORDER BY ordinal_position;

-- 检查新表
SELECT table_name, (SELECT COUNT(*) FROM information_schema.columns WHERE table_name = t.table_name) AS column_count
FROM information_schema.tables t
WHERE table_schema = 'public'
AND table_name IN ('sys_notice_target', 'sys_notice_read', 'sys_notice_attachment');
