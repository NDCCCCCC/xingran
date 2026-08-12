-- 工位管理表
-- 创建时间: 2024-12-24
-- 说明: 用于管理系统工位信息，包括工位分配、位置等

CREATE TABLE IF NOT EXISTS sys_workstation (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workstation_code  VARCHAR(50) NOT NULL UNIQUE,        -- 工位编码
    workstation_name  VARCHAR(100) NOT NULL,               -- 工位名称
    dept_id           UUID,                                -- 部门ID
    dept_name         VARCHAR(100),                        -- 部门名称（冗余）
    location          VARCHAR(200),                        -- 位置
    floor             VARCHAR(50),                         -- 楼层
    workstation_type  SMALLINT DEFAULT 0,                  -- 工位类型：0:固定工位 1:灵活工位 2:管理工位
    status            SMALLINT DEFAULT 0,                  -- 状态：0:启用 1:禁用
    capacity          INTEGER DEFAULT 1,                   -- 容量
    description       TEXT,                                -- 备注
    user_id           UUID,                                -- 分配的用户ID
    user_name         VARCHAR(100),                        -- 分配的用户名称（冗余）
    created_by        VARCHAR(64),                         -- 创建者
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_by        VARCHAR(64),                         -- 更新者
    updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP, -- 更新时间
    version           INTEGER DEFAULT 1,                   -- 版本号
    deleted_at        TIMESTAMP                             -- 软删除时间
);

-- 创建索引
CREATE INDEX idx_sys_workstation_dept_id ON sys_workstation(dept_id);
CREATE INDEX idx_sys_workstation_user_id ON sys_workstation(user_id);
CREATE INDEX idx_sys_workstation_status ON sys_workstation(status);
CREATE INDEX idx_sys_workstation_type ON sys_workstation(workstation_type);
CREATE INDEX idx_sys_workstation_deleted_at ON sys_workstation(deleted_at);

-- 添加注释
COMMENT ON TABLE sys_workstation IS '工位管理表';
COMMENT ON COLUMN sys_workstation.id IS '主键ID';
COMMENT ON COLUMN sys_workstation.workstation_code IS '工位编码';
COMMENT ON COLUMN sys_workstation.workstation_name IS '工位名称';
COMMENT ON COLUMN sys_workstation.dept_id IS '所属部门ID';
COMMENT ON COLUMN sys_workstation.dept_name IS '所属部门名称';
COMMENT ON COLUMN sys_workstation.location IS '位置';
COMMENT ON COLUMN sys_workstation.floor IS '楼层';
COMMENT ON COLUMN sys_workstation.workstation_type IS '工位类型：0:固定工位 1:灵活工位 2:管理工位';
COMMENT ON COLUMN sys_workstation.status IS '状态：0:启用 1:禁用';
COMMENT ON COLUMN sys_workstation.capacity IS '容量';
COMMENT ON COLUMN sys_workstation.description IS '备注';
COMMENT ON COLUMN sys_workstation.user_id IS '分配的用户ID';
COMMENT ON COLUMN sys_workstation.user_name IS '分配的用户名称';
