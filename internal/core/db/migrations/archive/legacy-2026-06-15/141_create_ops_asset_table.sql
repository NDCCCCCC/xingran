-- Migration: Create ops_asset table for asset management
-- Purpose: Support comprehensive asset tracking with 40 fields
-- Date: 2026-06-08
-- 命名规约: unique 约束显式命名 uni_<table>_<col> 与 GORM Asset.DeviceSN `uniqueIndex` 对齐
-- 注: 此 .sql 文件当前不被自动加载, 实际建表逻辑见 migration_148_create_ops_asset_table.go

-- Create ops_asset table
CREATE TABLE IF NOT EXISTS ops_asset (
    -- Primary key and timestamps
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    -- 核心标识 (3 fields)
    devicesn VARCHAR(200) NOT NULL,                      -- 设备序列号（唯一标识）
    sequenceno VARCHAR(100),                             -- 序列号
    fixassetno VARCHAR(100),                             -- 固定资产编号

    -- 设备信息 (4 fields)
    device_model_name VARCHAR(200),                      -- 型号
    device_type_name VARCHAR(100),                       -- 类型
    device_category_second_name VARCHAR(100),            -- 中类
    device_basic_type_name VARCHAR(50),                  -- 是否固定资产

    -- 用户关联 (4 fields)
    deviceuser_name VARCHAR(100),                        -- 领取人
    nowuser_name VARCHAR(100),                           -- 责任人
    nowuser_p13 VARCHAR(100),                            -- 责任人p13
    deviceuser_p13 VARCHAR(100),                         -- 领取人p13

    -- 部门关联 (3 fields)
    deptname VARCHAR(100),                               -- 受益部门
    nowuser_dept_code VARCHAR(100),                      -- 部门编码
    xndept_code VARCHAR(100),                             -- 受益部门编码

    -- 状态标识 (4 fields)
    usestatus_label VARCHAR(50),                         -- 状态
    new_flag_label VARCHAR(50),                           -- 新设备标识
    print_flag_name VARCHAR(50),                         -- 打印状态
    nbf_status INTEGER DEFAULT 0,                       -- 是否拟报废 (0=否, 1=是)

    -- 时间字段 (6 fields)
    drawing_date TIMESTAMP,                              -- 接收日期
    use_date TIMESTAMP,                                  -- 发放日期
    storage_datetime TIMESTAMP,                          -- 入库日期
    last_update_date TIMESTAMP,                          -- APP扫码时间
    y07_update_time TIMESTAMP,                           -- Y07更新时间
    machine_uptime TIMESTAMP,                            -- 最后上线时间

    -- 网络信息 (4 fields)
    mac1 VARCHAR(100),                                   -- 有线MAC
    mac2 VARCHAR(100),                                   -- 无线MAC
    machine_ip VARCHAR(50),                              -- 加域IP
    machine_bs VARCHAR(50),                             -- 加域标识

    -- 合同与属性 (2 fields)
    contractno VARCHAR(100),                            -- 合同号
    attribute_value VARCHAR(500),                       -- 设备属性

    -- 位置与归属 (6 fields)
    scan_site VARCHAR(200),                              -- AAP扫码地理位置
    remark VARCHAR(1000),                                -- 备注
    qudao_name VARCHAR(100),                             -- 设备渠道
    using_type_name VARCHAR(100),                        -- 用途
    orgno_name VARCHAR(100),                             -- 使用机构
    storeroom_name VARCHAR(100),                         -- 存放地址

    -- 机构与标准 (3 fields)
    sign_orgno_name VARCHAR(100),                        -- 归属机构
    is_no_standard_name VARCHAR(100),                   -- 申请标准名称
    error_flag_name VARCHAR(50),                         -- 异常标识

    -- 外部与部门用户 (4 fields)
    outer_user VARCHAR(100),                            -- 使用人
    useful_dept_name VARCHAR(100),                       -- 部门名称
    nowuser_job_name VARCHAR(100),                       -- 责任人岗位
    user_name VARCHAR(100),                              -- APP扫码账号

    -- 系统关联字段 (3 fields)
    dept_id VARCHAR(64),                                 -- 关联 sys_dept.id
    user_id VARCHAR(64),                                 -- 关联 sys_user.id
    machine_user_id VARCHAR(100),                       -- 最后上线账号

    -- 状态字段
    status INTEGER DEFAULT 0                             -- 0=正常, 1=停用
);

-- Add comments for documentation
COMMENT ON TABLE ops_asset IS '资产管理表：包含设备序列号、型号、用户关联、部门关联等40个字段';
COMMENT ON COLUMN ops_asset.devicesn IS '设备序列号，唯一标识，用于Excel导入时判断更新或新增';
COMMENT ON COLUMN ops_asset.dept_id IS '关联 sys_dept.id，通过部门名称匹配自动转换';
COMMENT ON COLUMN ops_asset.user_id IS '关联 sys_user.id，通过用户名匹配自动转换';
COMMENT ON COLUMN ops_asset.status IS '资产状态：0=正常, 1=停用';

-- Create indexes for common queries
CREATE INDEX idx_asset_devicesn ON ops_asset(devicesn);
CREATE INDEX idx_asset_dept_id ON ops_asset(dept_id);
CREATE INDEX idx_asset_user_id ON ops_asset(user_id);
CREATE INDEX idx_asset_status ON ops_asset(status);
CREATE INDEX idx_asset_deleted_at ON ops_asset(deleted_at);

-- Create composite index for department filtering
CREATE INDEX idx_asset_dept_status ON ops_asset(dept_id, status) WHERE deleted_at IS NULL;
