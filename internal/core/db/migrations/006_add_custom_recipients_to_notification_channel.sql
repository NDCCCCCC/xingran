-- ============================================
-- 006_add_custom_recipients_to_notification_channel.sql
-- 说明: 为通知渠道表添加自定义收件人字段
-- ============================================

-- 添加 custom_recipients 字段
DO $$
BEGIN
    -- 检查列是否存在，不存在则添加
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'sys_notification_channel'
        AND column_name = 'custom_recipients'
    ) THEN
        ALTER TABLE sys_notification_channel
        ADD COLUMN custom_recipients JSONB;

        RAISE NOTICE '已添加 custom_recipients 字段到 sys_notification_channel 表';
    ELSE
        RAISE NOTICE 'custom_recipients 字段已存在，跳过添加';
    END IF;
END $$;

-- 验证
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_name = 'sys_notification_channel'
AND column_name = 'custom_recipients';
