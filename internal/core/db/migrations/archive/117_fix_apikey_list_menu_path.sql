-- 修复API密钥管理菜单路径配置错误
-- 问题：密钥列表菜单的 path 配置与父菜单重复，导致前端路径拼接重复
--
-- 当前错误的配置：
--   API密钥管理 (父): path = 'apikeys'
--   密钥列表 (子): path = 'system/apikeys' 或 'apikeys'
--   结果: /system/apikeys/apikeys (路径重复)
--
-- 正确的配置：
--   API密钥管理 (父): path = 'apikeys'
--   密钥列表 (子): path = '' (空，表示父目录下的默认页面)
--   结果: /system/apikeys (正确)

-- 更新"密钥列表"菜单的路径为空字符串
UPDATE sys_menu
SET path = '',
    updated_at = NOW()
WHERE menu_name = '密钥列表';

-- 验证修复结果
SELECT
    menu_name,
    path,
    component,
    menu_type
FROM sys_menu
WHERE menu_name IN ('API密钥管理', '密钥列表')
ORDER BY order_num;

SELECT '修复完成：密钥列表菜单路径已设置为空字符串，作为 API密钥管理目录下的默认页面' AS status;
