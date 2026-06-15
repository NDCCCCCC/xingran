-- ============================================
-- 知识库模块数据库迁移
-- 文件: 012_create_knowledge_tables.sql
-- 说明: 创建知识库管理相关表
-- ============================================

-- ============================================
-- 1. 知识库分类表 sys_knowledge_category
-- ============================================

CREATE TABLE IF NOT EXISTS sys_knowledge_category (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_name VARCHAR(100) NOT NULL,
    description VARCHAR(500),
    icon VARCHAR(50),
    status INT DEFAULT 0,
    sort_order INT DEFAULT 0,
    parent_id UUID,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(64),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(64),
    CONSTRAINT uk_knowledge_category_name UNIQUE (category_name)
);

-- 添加外键约束（父分类自关联）
ALTER TABLE sys_knowledge_category ADD CONSTRAINT fk_knowledge_category_parent
    FOREIGN KEY (parent_id) REFERENCES sys_knowledge_category(id) ON DELETE SET NULL;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_knowledge_category_parent ON sys_knowledge_category(parent_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_category_status ON sys_knowledge_category(status);

-- 添加表和字段注释
COMMENT ON TABLE sys_knowledge_category IS '知识库分类表';
COMMENT ON COLUMN sys_knowledge_category.id IS '主键ID';
COMMENT ON COLUMN sys_knowledge_category.category_name IS '分类名称';
COMMENT ON COLUMN sys_knowledge_category.description IS '描述';
COMMENT ON COLUMN sys_knowledge_category.icon IS '图标';
COMMENT ON COLUMN sys_knowledge_category.status IS '状态: 0=启用 1=停用';
COMMENT ON COLUMN sys_knowledge_category.sort_order IS '排序';
COMMENT ON COLUMN sys_knowledge_category.parent_id IS '父分类ID';

-- ============================================
-- 2. 知识库标签表 sys_knowledge_tag
-- ============================================

CREATE TABLE IF NOT EXISTS sys_knowledge_tag (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tag_name VARCHAR(50) NOT NULL,
    color VARCHAR(20),
    use_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_knowledge_tag_name UNIQUE (tag_name)
);

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_knowledge_tag_use_count ON sys_knowledge_tag(use_count);

-- 添加表和字段注释
COMMENT ON TABLE sys_knowledge_tag IS '知识库标签表';
COMMENT ON COLUMN sys_knowledge_tag.id IS '主键ID';
COMMENT ON COLUMN sys_knowledge_tag.tag_name IS '标签名称';
COMMENT ON COLUMN sys_knowledge_tag.color IS '标签颜色';
COMMENT ON COLUMN sys_knowledge_tag.use_count IS '使用次数';

-- ============================================
-- 3. 知识库文章表 sys_knowledge_article
-- ============================================

CREATE TABLE IF NOT EXISTS sys_knowledge_article (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    summary VARCHAR(500),
    category_id UUID NOT NULL,
    status INT DEFAULT 0,
    view_count INT DEFAULT 0,
    like_count INT DEFAULT 0,
    is_top BOOLEAN DEFAULT FALSE,
    source_workorder_id UUID,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 添加外键约束
ALTER TABLE sys_knowledge_article ADD CONSTRAINT fk_knowledge_article_category
    FOREIGN KEY (category_id) REFERENCES sys_knowledge_category(id) ON DELETE RESTRICT;
ALTER TABLE sys_knowledge_article ADD CONSTRAINT fk_knowledge_article_source
    FOREIGN KEY (source_workorder_id) REFERENCES sys_workorder(id) ON DELETE SET NULL;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_knowledge_article_category ON sys_knowledge_article(category_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_article_status ON sys_knowledge_article(status);
CREATE INDEX IF NOT EXISTS idx_knowledge_article_source ON sys_knowledge_article(source_workorder_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_article_created ON sys_knowledge_article(created_at);
CREATE INDEX IF NOT EXISTS idx_knowledge_article_view_count ON sys_knowledge_article(view_count);
CREATE INDEX IF NOT EXISTS idx_knowledge_article_like_count ON sys_knowledge_article(like_count);

-- 添加全文搜索索引（PostgreSQL）
CREATE INDEX IF NOT EXISTS idx_knowledge_article_title_gin ON sys_knowledge_article USING gin(to_tsvector('simple', title));
CREATE INDEX IF NOT EXISTS idx_knowledge_article_content_gin ON sys_knowledge_article USING gin(to_tsvector('simple', content));

-- 添加表和字段注释
COMMENT ON TABLE sys_knowledge_article IS '知识库文章表';
COMMENT ON COLUMN sys_knowledge_article.id IS '主键ID';
COMMENT ON COLUMN sys_knowledge_article.title IS '文章标题';
COMMENT ON COLUMN sys_knowledge_article.content IS '文章内容';
COMMENT ON COLUMN sys_knowledge_article.summary IS '摘要';
COMMENT ON COLUMN sys_knowledge_article.category_id IS '分类ID';
COMMENT ON COLUMN sys_knowledge_article.status IS '状态: 0=草稿 1=已发布';
COMMENT ON COLUMN sys_knowledge_article.view_count IS '浏览次数';
COMMENT ON COLUMN sys_knowledge_article.like_count IS '点赞次数';
COMMENT ON COLUMN sys_knowledge_article.is_top IS '是否置顶';
COMMENT ON COLUMN sys_knowledge_article.source_workorder_id IS '来源工单ID';
COMMENT ON COLUMN sys_knowledge_article.created_by IS '创建人';
COMMENT ON COLUMN sys_knowledge_article.updated_by IS '更新人';

-- ============================================
-- 4. 文章标签关联表 sys_knowledge_article_tag
-- ============================================

CREATE TABLE IF NOT EXISTS sys_knowledge_article_tag (
    article_id UUID NOT NULL,
    tag_id UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (article_id, tag_id)
);

-- 添加外键约束
ALTER TABLE sys_knowledge_article_tag ADD CONSTRAINT fk_article_tag_article
    FOREIGN KEY (article_id) REFERENCES sys_knowledge_article(id) ON DELETE CASCADE;
ALTER TABLE sys_knowledge_article_tag ADD CONSTRAINT fk_article_tag_tag
    FOREIGN KEY (tag_id) REFERENCES sys_knowledge_tag(id) ON DELETE CASCADE;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_knowledge_article_tag_tag ON sys_knowledge_article_tag(tag_id);

-- 添加表和字段注释
COMMENT ON TABLE sys_knowledge_article_tag IS '文章标签关联表';
COMMENT ON COLUMN sys_knowledge_article_tag.article_id IS '文章ID';
COMMENT ON COLUMN sys_knowledge_article_tag.tag_id IS '标签ID';

-- ============================================
-- 5. 知识库浏览历史表 sys_knowledge_view_history
-- ============================================

CREATE TABLE IF NOT EXISTS sys_knowledge_view_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id UUID NOT NULL,
    user_id UUID NOT NULL,
    view_count INT DEFAULT 1,
    last_viewed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 添加外键约束
ALTER TABLE sys_knowledge_view_history ADD CONSTRAINT fk_knowledge_view_article
    FOREIGN KEY (article_id) REFERENCES sys_knowledge_article(id) ON DELETE CASCADE;
ALTER TABLE sys_knowledge_view_history ADD CONSTRAINT fk_knowledge_view_user
    FOREIGN KEY (user_id) REFERENCES sys_user(id) ON DELETE CASCADE;

-- 添加唯一约束（同一用户对同一文章只有一条记录）
CREATE UNIQUE INDEX idx_knowledge_view_unique ON sys_knowledge_view_history(article_id, user_id);

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_knowledge_view_user ON sys_knowledge_view_history(user_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_view_last_viewed ON sys_knowledge_view_history(last_viewed_at);

-- 添加表和字段注释
COMMENT ON TABLE sys_knowledge_view_history IS '知识库浏览历史表';
COMMENT ON COLUMN sys_knowledge_view_history.id IS '主键ID';
COMMENT ON COLUMN sys_knowledge_view_history.article_id IS '文章ID';
COMMENT ON COLUMN sys_knowledge_view_history.user_id IS '用户ID';
COMMENT ON COLUMN sys_knowledge_view_history.view_count IS '浏览次数';
COMMENT ON COLUMN sys_knowledge_view_history.last_viewed_at IS '最后浏览时间';

-- ============================================
-- 6. 知识库点赞记录表 sys_knowledge_like_history
-- ============================================

CREATE TABLE IF NOT EXISTS sys_knowledge_like_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id UUID NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 添加外键约束
ALTER TABLE sys_knowledge_like_history ADD CONSTRAINT fk_knowledge_like_article
    FOREIGN KEY (article_id) REFERENCES sys_knowledge_article(id) ON DELETE CASCADE;
ALTER TABLE sys_knowledge_like_history ADD CONSTRAINT fk_knowledge_like_user
    FOREIGN KEY (user_id) REFERENCES sys_user(id) ON DELETE CASCADE;

-- 添加唯一约束（同一用户对同一文章只能点赞一次）
CREATE UNIQUE INDEX idx_knowledge_like_unique ON sys_knowledge_like_history(article_id, user_id);

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_knowledge_like_user ON sys_knowledge_like_history(user_id);

-- 添加表和字段注释
COMMENT ON TABLE sys_knowledge_like_history IS '知识库点赞记录表';
COMMENT ON COLUMN sys_knowledge_like_history.id IS '主键ID';
COMMENT ON COLUMN sys_knowledge_like_history.article_id IS '文章ID';
COMMENT ON COLUMN sys_knowledge_like_history.user_id IS '用户ID';

-- ============================================
-- 插入默认数据
-- ============================================

-- 插入默认知识库分类
INSERT INTO sys_knowledge_category (id, category_name, description, icon, status, sort_order, created_by) VALUES
    ('00000000-0000-0000-0000-000000000001', '常见问题', '系统使用常见问题', 'question-circle', 0, 1, 'system'),
    ('00000000-0000-0000-0000-000000000002', '技术文档', '技术文档和开发指南', 'file-text', 0, 2, 'system'),
    ('00000000-0000-0000-0000-000000000003', '故障处理', '故障排查和处理方法', 'tool', 0, 3, 'system'),
    ('00000000-0000-0000-0000-000000000004', '运维指南', '日常运维操作指南', 'setting', 0, 4, 'system')
ON CONFLICT (id) DO NOTHING;

-- ============================================
-- 迁移完成
-- ============================================

SELECT '012_create_knowledge_tables.sql migration completed' AS status;
