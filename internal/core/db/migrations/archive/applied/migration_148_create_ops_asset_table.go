//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate148CreateOpsAssetTable creates the ops_asset table for asset management
func Migrate148CreateOpsAssetTable(db *gorm.DB) error {
	log.Println("Running migration 148: Create ops_asset table")

	// Check if table already exists
	if db.Migrator().HasTable(&OpsAsset{}) {
		log.Println("Table ops_asset already exists, skipping migration 148...")
		return nil
	}

	// Create the table using SQL for precision (40 fields)
	sql := `
	CREATE TABLE IF NOT EXISTS ops_asset (
		-- Primary key and timestamps
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP,

		-- 核心标识 (3 fields)
		-- 命名约束: 使用 GORM 命名规范 uni_<table>_<column> 与 models.Asset.DeviceSN uniqueIndex 对齐
		-- 避免 PG 默认命名 ops_asset_devicesn_key 与 GORM 期望 uni_ops_asset_devicesn 冲突
		devicesn VARCHAR(200) NOT NULL,
		sequenceno VARCHAR(100),
		fixassetno VARCHAR(100),

		-- 设备信息 (4 fields)
		device_model_name VARCHAR(200),
		device_type_name VARCHAR(100),
		device_category_second_name VARCHAR(100),
		device_basic_type_name VARCHAR(50),

		-- 用户关联 (4 fields)
		deviceuser_name VARCHAR(100),
		nowuser_name VARCHAR(100),
		nowuser_p13 VARCHAR(100),
		deviceuser_p13 VARCHAR(100),

		-- 部门关联 (3 fields)
		deptname VARCHAR(100),
		nowuser_dept_code VARCHAR(100),
		xndept_code VARCHAR(100),

		-- 状态标识 (4 fields)
		usestatus_label VARCHAR(50),
		new_flag_label VARCHAR(50),
		print_flag_name VARCHAR(50),
		nbf_status INTEGER DEFAULT 0,

		-- 时间字段 (6 fields)
		drawing_date TIMESTAMP,
		use_date TIMESTAMP,
		storage_datetime TIMESTAMP,
		last_update_date TIMESTAMP,
		y07_update_time TIMESTAMP,
		machine_uptime TIMESTAMP,

		-- 网络信息 (4 fields)
		mac1 VARCHAR(100),
		mac2 VARCHAR(100),
		machine_ip VARCHAR(50),
		machine_bs VARCHAR(50),

		-- 合同与属性 (2 fields)
		contractno VARCHAR(100),
		attribute_value VARCHAR(500),

		-- 位置与归属 (6 fields)
		scan_site VARCHAR(200),
		remark VARCHAR(1000),
		qudao_name VARCHAR(100),
		using_type_name VARCHAR(100),
		orgno_name VARCHAR(100),
		storeroom_name VARCHAR(100),

		-- 机构与标准 (3 fields)
		sign_orgno_name VARCHAR(100),
		is_no_standard_name VARCHAR(100),
		error_flag_name VARCHAR(50),

		-- 外部与部门用户 (4 fields)
		outer_user VARCHAR(100),
		useful_dept_name VARCHAR(100),
		nowuser_job_name VARCHAR(100),
		user_name VARCHAR(100),

		-- 系统关联字段 (3 fields)
		dept_id VARCHAR(64),
		user_id VARCHAR(64),
		machine_user_id VARCHAR(100),

		-- 状态字段
		status INTEGER DEFAULT 0
	);

	-- Add comments
	COMMENT ON TABLE ops_asset IS '资产管理表：包含设备序列号、型号、用户关联、部门关联等40个字段';
	COMMENT ON COLUMN ops_asset.devicesn IS '设备序列号，唯一标识，用于Excel导入时判断更新或新增';
	COMMENT ON COLUMN ops_asset.dept_id IS '关联 sys_dept.id，通过部门名称匹配自动转换';
	COMMENT ON COLUMN ops_asset.user_id IS '关联 sys_user.id，通过用户名匹配自动转换';
	COMMENT ON COLUMN ops_asset.status IS '资产状态：0=正常, 1=停用';

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_asset_devicesn ON ops_asset(devicesn);
	CREATE INDEX IF NOT EXISTS idx_asset_dept_id ON ops_asset(dept_id);
	CREATE INDEX IF NOT EXISTS idx_asset_user_id ON ops_asset(user_id);
	CREATE INDEX IF NOT EXISTS idx_asset_status ON ops_asset(status);
	CREATE INDEX IF NOT EXISTS idx_asset_deleted_at ON ops_asset(deleted_at);
	CREATE INDEX IF NOT EXISTS idx_asset_dept_status ON ops_asset(dept_id, status) WHERE deleted_at IS NULL;

	-- 显式命名 unique constraint (与 GORM uniqueIndex 命名规范一致,避免 AutoMigrate 重建冲突)
	-- 注意: 用 DO 块保证幂等,避免重复运行报错
	DO $$
	BEGIN
		IF NOT EXISTS (
			SELECT 1 FROM pg_constraint
			WHERE conname = 'uni_ops_asset_devicesn'
			  AND conrelid = 'ops_asset'::regclass
		) THEN
			ALTER TABLE ops_asset ADD CONSTRAINT uni_ops_asset_devicesn UNIQUE (devicesn);
		END IF;
	END$$;
	`

	if err := db.Exec(sql).Error; err != nil {
		log.Printf("Failed to create ops_asset table: %v", err)
		return err
	}

	log.Println("Migration 148 completed successfully")
	return nil
}

// OpsAsset represents the asset table structure (for GORM to check table existence)
type OpsAsset struct {
	ID       string `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Devicesn string `gorm:"size:200;not null;unique"`
}
