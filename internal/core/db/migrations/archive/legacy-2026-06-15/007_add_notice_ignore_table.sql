-- 用户忽略通知表
-- 说明: 用户可以隐藏不想看的通知，被忽略的通知不会再显示在用户的通知列表中
-- 命名规约: 复合 unique 约束显式命名 uni_<table>_<col1>_<col2>

-- 创建用户忽略通知表
CREATE TABLE IF NOT EXISTS sys_notice_ignore (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notice_id UUID NOT NULL,
    user_id UUID NOT NULL,
    ignored_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uni_sys_notice_ignore_notice_user UNIQUE (notice_id, user_id)
);

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_notice_ignore_notice_id ON sys_notice_ignore(notice_id);
CREATE INDEX IF NOT EXISTS idx_notice_ignore_user_id ON sys_notice_ignore(user_id);

-- 添加外键约束
ALTER TABLE sys_notice_ignore DROP CONSTRAINT IF EXISTS fk_notice_ignore_notice;
ALTER TABLE sys_notice_ignore ADD CONSTRAINT fk_notice_ignore_notice
    FOREIGN KEY (notice_id) REFERENCES sys_notice(id) ON DELETE CASCADE;

ALTER TABLE sys_notice_ignore DROP CONSTRAINT IF EXISTS fk_notice_ignore_user;
ALTER TABLE sys_notice_ignore ADD CONSTRAINT fk_notice_ignore_user
    FOREIGN KEY (user_id) REFERENCES sys_user(id) ON DELETE CASCADE;

-- 添加表和字段注释
COMMENT ON TABLE sys_notice_ignore IS '用户忽略通知表';
COMMENT ON COLUMN sys_notice_ignore.id IS '主键ID';
COMMENT ON COLUMN sys_notice_ignore.notice_id IS '通知ID';
COMMENT ON COLUMN sys_notice_ignore.user_id IS '用户ID';
COMMENT ON COLUMN sys_notice_ignore.ignored_at IS '忽略时间';

-- 验证迁移
SELECT '007_add_notice_ignore_table.sql migration completed' AS status;

-- 检查新表
SELECT table_name, (SELECT COUNT(*) FROM information_schema.columns WHERE table_name = t.table_name) AS column_count
FROM information_schema.tables t
WHERE table_schema = 'public'
AND table_name = 'sys_notice_ignore';
