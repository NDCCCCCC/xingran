-- 性能优化索引
-- 迁移编号: 019
-- 描述: 为高频查询添加索引以提升性能

-- ====================
-- 通知系统索引
-- ====================

-- 通知发布状态和状态索引（用于查询已发布的有效通知）
CREATE INDEX IF NOT EXISTS idx_sys_notice_publish_status
ON sys_notice(publish_status, status)
WHERE status = 0;

-- 通知忽略表复合索引（用于查询用户忽略的通知）
CREATE INDEX IF NOT EXISTS idx_sys_notice_ignore_user_notice
ON sys_notice_ignore(user_id, notice_id);

-- 通知目标表复合索引（用于按类型和目标ID查询）
CREATE INDEX IF NOT EXISTS idx_sys_notice_target_composite
ON sys_notice_target(target_type, target_id, notice_id);

-- 通知阅读记录索引（用于查询用户已读通知）
CREATE INDEX IF NOT EXISTS idx_sys_notice_read_user_notice
ON sys_notice_read(user_id, notice_id);

-- ====================
-- 设备管理索引
-- ====================

-- 注意：由于存在重复数据，先创建普通索引，不去重以保留历史数据
-- 如果需要创建唯一索引，请先清理重复数据

-- 设备MAC地址复合索引（允许重复，提升查询性能）
CREATE INDEX IF NOT EXISTS idx_sys_device_mac_address_composite
ON sys_device_mac_address(device_id, mac_address, interface_name);

-- MAC地址索引（用于按MAC地址搜索设备）
CREATE INDEX IF NOT EXISTS idx_sys_device_mac_address_mac
ON sys_device_mac_address(mac_address);

-- ====================
-- 工单系统索引
-- ====================
-- 实际表名：sys_workorder, sys_workorder_history, sys_workorder_comment

-- 工单历史记录索引（用于查询工单历史，按时间排序）
CREATE INDEX IF NOT EXISTS idx_sys_workorder_history_wo_created
ON sys_workorder_history(work_order_id, created_at DESC);

-- 工单评论索引（用于查询工单评论）
CREATE INDEX IF NOT EXISTS idx_sys_workorder_comment_wo_created
ON sys_workorder_comment(work_order_id, created_at DESC);

-- 工单按状态和创建时间索引（用于列表查询）
CREATE INDEX IF NOT EXISTS idx_sys_workorder_status_created
ON sys_workorder(status, created_at DESC);

-- 工单按处理人索引（用于查询分配给某用户的工单）
CREATE INDEX IF NOT EXISTS idx_sys_workorder_assignee_created
ON sys_workorder(assignee_id, created_at DESC)
WHERE assignee_id IS NOT NULL;

-- 工单按部门索引（用于查询某部门的工单）
CREATE INDEX IF NOT EXISTS idx_sys_workorder_dept_created
ON sys_workorder(dept_id, created_at DESC)
WHERE dept_id IS NOT NULL;

-- 工单按分类索引（用于查询某分类的工单）
CREATE INDEX IF NOT EXISTS idx_sys_workorder_category_created
ON sys_workorder(category_id, created_at DESC);

-- ====================
-- 值班管理索引
-- ====================

-- 值班计划按日期索引（用于查询某日期的值班）
CREATE INDEX IF NOT EXISTS idx_sys_duty_schedule_date
ON sys_duty_schedule(schedule_date);

-- 值班计划按池ID索引（用于查询某值班池的计划）
CREATE INDEX IF NOT EXISTS idx_sys_duty_schedule_pool_date
ON sys_duty_schedule(pool_id, schedule_date);

-- ====================
-- 知识库索引
-- ====================
-- 实际表名：sys_knowledge_article, sys_knowledge_category

-- 知识库文章按分类索引（用于查询某分类的文章）
CREATE INDEX IF NOT EXISTS idx_sys_knowledge_article_category_created
ON sys_knowledge_article(category_id, created_at DESC);

-- 知识库文章发布状态索引（用于查询已发布的文章）
CREATE INDEX IF NOT EXISTS idx_sys_knowledge_article_status_created
ON sys_knowledge_article(status, created_at DESC);

-- ====================
-- 用户相关索引
-- ====================

-- 用户部门索引（用于查询部门下的用户）
CREATE INDEX IF NOT EXISTS idx_sys_user_dept_status
ON sys_user(dept_id, status)
WHERE status = '0';

-- 用户角色关联索引（用于查询用户的角色）
CREATE INDEX IF NOT EXISTS idx_sys_user_role_user_role
ON sys_user_role(user_id, role_id);

-- ====================
-- 说明
-- ====================
-- 1. 使用 IF NOT EXISTS 确保幂等性
-- 2. 部分索引使用 WHERE 子句减少索引大小
-- 3. 复合索引遵循最左前缀原则
-- 4. DESC 用于排序查询优化
-- 5. 实际表名（注意：都是单数形式）：
--    - sys_workorder, sys_workorder_history, sys_workorder_comment
--    - sys_knowledge_article, sys_knowledge_category
--    - sys_duty_schedule
--    - sys_notice, sys_notice_ignore, sys_notice_target, sys_notice_read
--    - sys_user, sys_user_role
--    - sys_device_mac_address
