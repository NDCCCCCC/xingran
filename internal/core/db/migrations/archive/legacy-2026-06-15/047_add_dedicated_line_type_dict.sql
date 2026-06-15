-- Migration: 047_add_dedicated_line_type_dict.sql
-- Description: 添加专线类型字典，替换硬编码
-- Date: 2026-01-21

-- 插入专线类型字典
INSERT INTO sys_dict_type (id, dict_name, dict_type, status, remark, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    '专线类型',
    'ops_dedicated_line_type',
    0,
    '专线类型字典：互联网专线、内网专线、云桌面专线等',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
)
ON CONFLICT (dict_type) DO NOTHING;

-- 插入专线类型字典数据
INSERT INTO sys_dict_data (id, dict_sort, dict_label, dict_value, dict_type, css_class, list_class, is_default, status, remark, created_at, updated_at)
VALUES
    (
        gen_random_uuid(),
        1,
        '互联网专线',
        'internet',
        'ops_dedicated_line_type',
        NULL,
        'primary',
        true,
        0,
        '互联网专线',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        gen_random_uuid(),
        2,
        '内网专线',
        'intranet',
        'ops_dedicated_line_type',
        NULL,
        'success',
        false,
        0,
        '内网专线',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        gen_random_uuid(),
        3,
        '云桌面专线',
        'cloud_desktop',
        'ops_dedicated_line_type',
        NULL,
        'warning',
        false,
        0,
        '云桌面专线',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        gen_random_uuid(),
        4,
        'MPLS VPN',
        'mpls',
        'ops_dedicated_line_type',
        NULL,
        'processing',
        false,
        0,
        'MPLS VPN专线',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        gen_random_uuid(),
        5,
        '光纤专线',
        'fiber',
        'ops_dedicated_line_type',
        NULL,
        'default',
        false,
        0,
        '光纤专线',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        gen_random_uuid(),
        6,
        '租用专线',
        'leased_line',
        'ops_dedicated_line_type',
        NULL,
        'default',
        false,
        0,
        '租用专线',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    )
ON CONFLICT DO NOTHING;
