-- ============================================
-- 运维工单模块数据库迁移
-- 文件: 011_create_workorder_tables.sql
-- 说明: 创建运维工单管理相关表
-- ============================================

-- ============================================
-- 1. 工单分类表 sys_workorder_category
-- ============================================

CREATE TABLE IF NOT EXISTS sys_workorder_category (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_name VARCHAR(100) NOT NULL,
    description VARCHAR(500),
    status INT DEFAULT 0,
    sort_order INT DEFAULT 0,
    parent_id UUID,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(64),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(64)
);

-- 添加外键约束（父分类自关联）
ALTER TABLE sys_workorder_category ADD CONSTRAINT fk_workorder_category_parent
    FOREIGN KEY (parent_id) REFERENCES sys_workorder_category(id) ON DELETE SET NULL;

-- 添加索引（使用GORM期望的名称）
CREATE UNIQUE INDEX IF NOT EXISTS idx_wo_category_name ON sys_workorder_category(category_name);
CREATE INDEX IF NOT EXISTS idx_workorder_category_parent ON sys_workorder_category(parent_id);
CREATE INDEX IF NOT EXISTS idx_workorder_category_status ON sys_workorder_category(status);

-- 添加表和字段注释
COMMENT ON TABLE sys_workorder_category IS '工单分类表';
COMMENT ON COLUMN sys_workorder_category.id IS '主键ID';
COMMENT ON COLUMN sys_workorder_category.category_name IS '分类名称';
COMMENT ON COLUMN sys_workorder_category.description IS '描述';
COMMENT ON COLUMN sys_workorder_category.status IS '状态: 0=启用 1=停用';
COMMENT ON COLUMN sys_workorder_category.sort_order IS '排序';
COMMENT ON COLUMN sys_workorder_category.parent_id IS '父分类ID';

-- ============================================
-- 2. 工单主表 sys_workorder
-- ============================================

CREATE TABLE IF NOT EXISTS sys_workorder (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(200) NOT NULL,
    work_order_no VARCHAR(50) NOT NULL,
    category_id UUID NOT NULL,
    type VARCHAR(20) NOT NULL,
    priority INT DEFAULT 1,
    status INT DEFAULT 0,
    description TEXT,
    solution TEXT,
    submitter_id UUID NOT NULL,
    assignee_id UUID,
    dept_id UUID,
    expected_resolve_at TIMESTAMP,
    resolved_at TIMESTAMP,
    closed_at TIMESTAMP,
    attachment_ids VARCHAR(1000),
    -- 自动分配相关字段
    is_auto_assigned BOOLEAN DEFAULT FALSE,
    duty_pool_id UUID,
    duty_type VARCHAR(20),
    assign_strategy VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(64),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(64)
);

-- 添加外键约束
ALTER TABLE sys_workorder ADD CONSTRAINT fk_workorder_category
    FOREIGN KEY (category_id) REFERENCES sys_workorder_category(id) ON DELETE RESTRICT;
ALTER TABLE sys_workorder ADD CONSTRAINT fk_workorder_submitter
    FOREIGN KEY (submitter_id) REFERENCES sys_user(id) ON DELETE RESTRICT;
ALTER TABLE sys_workorder ADD CONSTRAINT fk_workorder_assignee
    FOREIGN KEY (assignee_id) REFERENCES sys_user(id) ON DELETE SET NULL;
ALTER TABLE sys_workorder ADD CONSTRAINT fk_workorder_dept
    FOREIGN KEY (dept_id) REFERENCES sys_dept(id) ON DELETE SET NULL;

-- 添加索引（使用GORM期望的名称）
CREATE UNIQUE INDEX IF NOT EXISTS idx_wo_no ON sys_workorder(work_order_no);
CREATE INDEX IF NOT EXISTS idx_wo_category ON sys_workorder(category_id);
CREATE INDEX IF NOT EXISTS idx_wo_status ON sys_workorder(status);
CREATE INDEX IF NOT EXISTS idx_wo_submitter ON sys_workorder(submitter_id);
CREATE INDEX IF NOT EXISTS idx_wo_assignee ON sys_workorder(assignee_id);
CREATE INDEX IF NOT EXISTS idx_wo_created_at ON sys_workorder(created_at);

-- 添加表和字段注释
COMMENT ON TABLE sys_workorder IS '运维工单表';
COMMENT ON COLUMN sys_workorder.id IS '主键ID';
COMMENT ON COLUMN sys_workorder.title IS '工单标题';
COMMENT ON COLUMN sys_workorder.work_order_no IS '工单编号';
COMMENT ON COLUMN sys_workorder.category_id IS '分类ID';
COMMENT ON COLUMN sys_workorder.type IS '工单类型: fault/request/change/incident/question';
COMMENT ON COLUMN sys_workorder.priority IS '优先级: 0=低 1=中 2=高 3=紧急';
COMMENT ON COLUMN sys_workorder.status IS '状态: 0=待处理 1=处理中 2=已完成 3=已关闭 4=已拒绝';
COMMENT ON COLUMN sys_workorder.description IS '工单描述';
COMMENT ON COLUMN sys_workorder.solution IS '解决方案';
COMMENT ON COLUMN sys_workorder.submitter_id IS '报告人ID';
COMMENT ON COLUMN sys_workorder.assignee_id IS '处理人ID';
COMMENT ON COLUMN sys_workorder.dept_id IS '部门ID';
COMMENT ON COLUMN sys_workorder.expected_resolve_at IS '期望解决时间';
COMMENT ON COLUMN sys_workorder.resolved_at IS '实际解决时间';
COMMENT ON COLUMN sys_workorder.closed_at IS '关闭时间';
COMMENT ON COLUMN sys_workorder.attachment_ids IS '附件ID列表（逗号分隔）';
COMMENT ON COLUMN sys_workorder.is_auto_assigned IS '是否自动分配';
COMMENT ON COLUMN sys_workorder.duty_pool_id IS '关联的值班池ID';
COMMENT ON COLUMN sys_workorder.duty_type IS '值班类型';
COMMENT ON COLUMN sys_workorder.assign_strategy IS '分配策略';

-- ============================================
-- 3. 工单评论表 sys_workorder_comment
-- ============================================

CREATE TABLE IF NOT EXISTS sys_workorder_comment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_order_id UUID NOT NULL,
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    is_internal BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 添加外键约束
ALTER TABLE sys_workorder_comment ADD CONSTRAINT fk_wo_comment_workorder
    FOREIGN KEY (work_order_id) REFERENCES sys_workorder(id) ON DELETE CASCADE;
ALTER TABLE sys_workorder_comment ADD CONSTRAINT fk_wo_comment_user
    FOREIGN KEY (user_id) REFERENCES sys_user(id) ON DELETE CASCADE;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_wo_comment_workorder ON sys_workorder_comment(work_order_id);
CREATE INDEX IF NOT EXISTS idx_wo_comment_user ON sys_workorder_comment(user_id);

-- 添加表和字段注释
COMMENT ON TABLE sys_workorder_comment IS '工单评论表';
COMMENT ON COLUMN sys_workorder_comment.id IS '主键ID';
COMMENT ON COLUMN sys_workorder_comment.work_order_id IS '工单ID';
COMMENT ON COLUMN sys_workorder_comment.user_id IS '评论用户ID';
COMMENT ON COLUMN sys_workorder_comment.content IS '评论内容';
COMMENT ON COLUMN sys_workorder_comment.is_internal IS '是否内部评论';

-- ============================================
-- 4. 工单操作历史表 sys_workorder_history
-- ============================================

CREATE TABLE IF NOT EXISTS sys_workorder_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_order_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    field VARCHAR(50),
    old_value TEXT,
    new_value TEXT,
    remark TEXT,
    operator_id UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 添加外键约束
ALTER TABLE sys_workorder_history ADD CONSTRAINT fk_wo_history_workorder
    FOREIGN KEY (work_order_id) REFERENCES sys_workorder(id) ON DELETE CASCADE;
ALTER TABLE sys_workorder_history ADD CONSTRAINT fk_wo_history_operator
    FOREIGN KEY (operator_id) REFERENCES sys_user(id) ON DELETE CASCADE;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_wo_history_workorder ON sys_workorder_history(work_order_id);
CREATE INDEX IF NOT EXISTS idx_wo_history_created ON sys_workorder_history(created_at);

-- 添加表和字段注释
COMMENT ON TABLE sys_workorder_history IS '工单操作历史表';
COMMENT ON COLUMN sys_workorder_history.id IS '主键ID';
COMMENT ON COLUMN sys_workorder_history.work_order_id IS '工单ID';
COMMENT ON COLUMN sys_workorder_history.action IS '操作类型';
COMMENT ON COLUMN sys_workorder_history.field IS '变更字段';
COMMENT ON COLUMN sys_workorder_history.old_value IS '旧值';
COMMENT ON COLUMN sys_workorder_history.new_value IS '新值';
COMMENT ON COLUMN sys_workorder_history.remark IS '备注';
COMMENT ON COLUMN sys_workorder_history.operator_id IS '操作人ID';

-- ============================================
-- 5. 工单评价表 sys_workorder_rating
-- ============================================

CREATE TABLE IF NOT EXISTS sys_workorder_rating (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_order_id UUID NOT NULL,
    rating_type VARCHAR(20) NOT NULL,
    completion_score INT DEFAULT 0,
    cooperation_score INT DEFAULT 0,
    comment TEXT,
    rater_id UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 添加外键约束
ALTER TABLE sys_workorder_rating ADD CONSTRAINT fk_wo_rating_workorder
    FOREIGN KEY (work_order_id) REFERENCES sys_workorder(id) ON DELETE CASCADE;
ALTER TABLE sys_workorder_rating ADD CONSTRAINT fk_wo_rating_rater
    FOREIGN KEY (rater_id) REFERENCES sys_user(id) ON DELETE CASCADE;

-- 添加唯一约束（每个用户对每种评价类型只能评价一次）
CREATE UNIQUE INDEX idx_wo_rating_unique ON sys_workorder_rating(work_order_id, rating_type, rater_id);

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_wo_rating_workorder ON sys_workorder_rating(work_order_id);

-- 添加表和字段注释
COMMENT ON TABLE sys_workorder_rating IS '工单评价表';
COMMENT ON COLUMN sys_workorder_rating.id IS '主键ID';
COMMENT ON COLUMN sys_workorder_rating.work_order_id IS '工单ID';
COMMENT ON COLUMN sys_workorder_rating.rating_type IS '评价类型: user=用户评价 handler=处理人员评价';
COMMENT ON COLUMN sys_workorder_rating.completion_score IS '完成度评分（用户评价）';
COMMENT ON COLUMN sys_workorder_rating.cooperation_score IS '配合度评分（处理人员评价）';
COMMENT ON COLUMN sys_workorder_rating.comment IS '评价内容';
COMMENT ON COLUMN sys_workorder_rating.rater_id IS '评价人ID';

-- ============================================
-- 6. 工单配置表 sys_workorder_config
-- ============================================

CREATE TABLE IF NOT EXISTS sys_workorder_config (
    id VARCHAR(50) PRIMARY KEY DEFAULT 'default',
    auto_assign_enabled BOOLEAN DEFAULT TRUE,
    auto_assign_target VARCHAR(50) DEFAULT 'duty_pool',
    auto_assign_strategy VARCHAR(50) DEFAULT 'assign_one',
    auto_close_days INT DEFAULT 7,
    allow_user_close BOOLEAN DEFAULT FALSE,
    notification_enabled BOOLEAN DEFAULT TRUE,
    email_notification BOOLEAN DEFAULT FALSE,
    sms_notification BOOLEAN DEFAULT FALSE,
    rating_enabled BOOLEAN DEFAULT TRUE,
    knowledge_convert_enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 添加表和字段注释
COMMENT ON TABLE sys_workorder_config IS '工单配置表';
COMMENT ON COLUMN sys_workorder_config.auto_assign_enabled IS '是否自动分配';
COMMENT ON COLUMN sys_workorder_config.auto_assign_target IS '分配目标';
COMMENT ON COLUMN sys_workorder_config.auto_assign_strategy IS '分配策略: assign_all/assign_one/assign_order';
COMMENT ON COLUMN sys_workorder_config.auto_close_days IS '完成后自动关闭天数（0=手动关闭）';
COMMENT ON COLUMN sys_workorder_config.allow_user_close IS '是否允许用户关闭工单';
COMMENT ON COLUMN sys_workorder_config.notification_enabled IS '是否启用通知';
COMMENT ON COLUMN sys_workorder_config.email_notification IS '邮件通知';
COMMENT ON COLUMN sys_workorder_config.sms_notification IS '短信通知';
COMMENT ON COLUMN sys_workorder_config.rating_enabled IS '评价功能开关';
COMMENT ON COLUMN sys_workorder_config.knowledge_convert_enabled IS '知识库转换开关';

-- 插入默认配置
INSERT INTO sys_workorder_config (
    auto_assign_enabled, auto_assign_target, auto_assign_strategy,
    auto_close_days, allow_user_close, notification_enabled,
    email_notification, sms_notification, rating_enabled,
    knowledge_convert_enabled
) VALUES (
    TRUE, 'duty_pool', 'assign_one', 7, FALSE, TRUE,
    FALSE, FALSE, TRUE, TRUE
) ON CONFLICT (id) DO NOTHING;

-- ============================================
-- 7. 周期性工单模板表 sys_periodic_workorder_template
-- ============================================

CREATE TABLE IF NOT EXISTS sys_periodic_workorder_template (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_name VARCHAR(100) NOT NULL,
    work_order_title VARCHAR(200) NOT NULL,
    description TEXT,
    category_id UUID NOT NULL,
    type VARCHAR(20) NOT NULL,
    priority INT DEFAULT 1,
    cron_expression VARCHAR(100) NOT NULL,
    assign_type VARCHAR(20) DEFAULT 'duty_pool',
    assign_target_id UUID,
    is_enabled BOOLEAN DEFAULT TRUE,
    next_run_at TIMESTAMP,
    job_id VARCHAR(50),
    total_generated INT DEFAULT 0,
    notify_assignee BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(64),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(64)
);

-- 添加外键约束
ALTER TABLE sys_periodic_workorder_template ADD CONSTRAINT fk_periodic_wo_template_category
    FOREIGN KEY (category_id) REFERENCES sys_workorder_category(id) ON DELETE RESTRICT;
ALTER TABLE sys_periodic_workorder_template ADD CONSTRAINT fk_periodic_wo_template_assignee
    FOREIGN KEY (assign_target_id) REFERENCES sys_user(id) ON DELETE SET NULL;
ALTER TABLE sys_periodic_workorder_template ADD CONSTRAINT fk_periodic_wo_template_job
    FOREIGN KEY (job_id) REFERENCES sys_job(id) ON DELETE SET NULL;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_periodic_wo_template_enabled ON sys_periodic_workorder_template(is_enabled);
CREATE INDEX IF NOT EXISTS idx_periodic_wo_template_job ON sys_periodic_workorder_template(job_id);

-- 添加表和字段注释
COMMENT ON TABLE sys_periodic_workorder_template IS '周期性工单模板表';
COMMENT ON COLUMN sys_periodic_workorder_template.id IS '主键ID';
COMMENT ON COLUMN sys_periodic_workorder_template.template_name IS '模板名称';
COMMENT ON COLUMN sys_periodic_workorder_template.work_order_title IS '工单标题（支持变量）';
COMMENT ON COLUMN sys_periodic_workorder_template.description IS '描述';
COMMENT ON COLUMN sys_periodic_workorder_template.category_id IS '分类ID';
COMMENT ON COLUMN sys_periodic_workorder_template.type IS '工单类型';
COMMENT ON COLUMN sys_periodic_workorder_template.priority IS '优先级';
COMMENT ON COLUMN sys_periodic_workorder_template.cron_expression IS 'Cron表达式';
COMMENT ON COLUMN sys_periodic_workorder_template.assign_type IS '分配类型: manual/duty_pool/rotation';
COMMENT ON COLUMN sys_periodic_workorder_template.assign_target_id IS '分配目标ID';
COMMENT ON COLUMN sys_periodic_workorder_template.is_enabled IS '是否启用';
COMMENT ON COLUMN sys_periodic_workorder_template.next_run_at IS '下次执行时间';
COMMENT ON COLUMN sys_periodic_workorder_template.job_id IS '定时任务ID';
COMMENT ON COLUMN sys_periodic_workorder_template.total_generated IS '已生成工单数量';
COMMENT ON COLUMN sys_periodic_workorder_template.notify_assignee IS '是否通知处理人';

-- ============================================
-- 8. 周期性工单执行记录表 sys_periodic_workorder_log
-- ============================================

CREATE TABLE IF NOT EXISTS sys_periodic_workorder_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL,
    work_order_id UUID NOT NULL,
    executed_at TIMESTAMP NOT NULL,
    job_id VARCHAR(50),
    status VARCHAR(20) DEFAULT 'success',
    result TEXT,
    error_msg TEXT
);

-- 添加外键约束
ALTER TABLE sys_periodic_workorder_log ADD CONSTRAINT fk_periodic_wo_log_template
    FOREIGN KEY (template_id) REFERENCES sys_periodic_workorder_template(id) ON DELETE CASCADE;
ALTER TABLE sys_periodic_workorder_log ADD CONSTRAINT fk_periodic_wo_log_workorder
    FOREIGN KEY (work_order_id) REFERENCES sys_workorder(id) ON DELETE CASCADE;
ALTER TABLE sys_periodic_workorder_log ADD CONSTRAINT fk_periodic_wo_log_job
    FOREIGN KEY (job_id) REFERENCES sys_job(id) ON DELETE SET NULL;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_pwo_log_template ON sys_periodic_workorder_log(template_id);
CREATE INDEX IF NOT EXISTS idx_pwo_log_workorder ON sys_periodic_workorder_log(work_order_id);
CREATE INDEX IF NOT EXISTS idx_pwo_log_executed ON sys_periodic_workorder_log(executed_at);

-- 添加表和字段注释
COMMENT ON TABLE sys_periodic_workorder_log IS '周期性工单执行记录表';
COMMENT ON COLUMN sys_periodic_workorder_log.id IS '主键ID';
COMMENT ON COLUMN sys_periodic_workorder_log.template_id IS '模板ID';
COMMENT ON COLUMN sys_periodic_workorder_log.work_order_id IS '工单ID';
COMMENT ON COLUMN sys_periodic_workorder_log.executed_at IS '执行时间';
COMMENT ON COLUMN sys_periodic_workorder_log.job_id IS '定时任务ID';
COMMENT ON COLUMN sys_periodic_workorder_log.status IS '执行状态: success/failed';
COMMENT ON COLUMN sys_periodic_workorder_log.result IS '执行结果';
COMMENT ON COLUMN sys_periodic_workorder_log.error_msg IS '错误信息';

-- ============================================
-- 迁移完成
-- ============================================

SELECT '011_create_workorder_tables.sql migration completed' AS status;
