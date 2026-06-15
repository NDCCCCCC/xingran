-- Migration: 048_add_isp_dict.sql
-- Description: 添加运营商数据字典
-- Date: 2026-01-21

-- 插入运营商字典类型
INSERT INTO sys_dict_type (id, dict_name, dict_type, status, remark, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    '运营商',
    'ops_isp',
    0,
    '运营商字典：电信、移动、联通等',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
)
ON CONFLICT (dict_type) DO NOTHING;

-- 插入运营商字典数据
INSERT INTO sys_dict_data (id, dict_sort, dict_label, dict_value, dict_type, css_class, list_class, is_default, status, remark, created_at, updated_at)
VALUES
    (
        gen_random_uuid(),
        1,
        '电信',
        'telecom',
        'ops_isp',
        NULL,
        'primary',
        true,
        0,
        '中国电信',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        gen_random_uuid(),
        2,
        '移动',
        'mobile',
        'ops_isp',
        NULL,
        'success',
        false,
        0,
        '中国移动',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        gen_random_uuid(),
        3,
        '联通',
        'unicom',
        'ops_isp',
        NULL,
        'warning',
        false,
        0,
        '中国联通',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        gen_random_uuid(),
        4,
        '广电',
        'broadcast',
        'ops_isp',
        NULL,
        'processing',
        false,
        0,
        '中国广电',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        gen_random_uuid(),
        5,
        '其他',
        'other',
        'ops_isp',
        NULL,
        'default',
        false,
        0,
        '其他运营商',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    )
ON CONFLICT DO NOTHING;
