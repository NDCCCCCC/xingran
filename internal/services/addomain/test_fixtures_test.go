package addomain

// 62-01 由 user_ou_service_test.go 迁入的共享测试建表 const。
// 供 dept_ou_mapper_test.go / ad_push_queue_drain_test.go 等复用（零副本约定）。

const (
	// Shared table creation SQL for user_ou_service tests
	createDeptOUMappingTable = `CREATE TABLE sys_dept_ou_mapping (
		id TEXT PRIMARY KEY,
		dept_id TEXT NOT NULL,
		ad_config_id TEXT NOT NULL,
		ou_dn TEXT NOT NULL,
		ou_name TEXT NOT NULL,
		parent_ou_dn TEXT,
		sync_enabled INTEGER DEFAULT 1,
		sync_status TEXT DEFAULT 'pending',
		last_sync_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		UNIQUE(dept_id, ad_config_id)
	)`

	createUserTable = `CREATE TABLE sys_user (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL DEFAULT '',
		salt TEXT NOT NULL DEFAULT '',
		nickname TEXT,
		employee_no TEXT,
		email TEXT,
		phone TEXT, mobile TEXT, title TEXT, ad_config_id TEXT,
		avatar TEXT,
		gender INTEGER DEFAULT 0,
		status INTEGER DEFAULT 0,
		dept_id TEXT,
		dept_name TEXT,
		login_ip TEXT,
		login_time DATETIME,
		pwd_update_time DATETIME,
		pwd_expire_days INTEGER DEFAULT 90,
		init_flag BOOLEAN DEFAULT FALSE,
		remark TEXT DEFAULT '',
		auth_source TEXT DEFAULT 'local',
		ad_username TEXT,
		ad_dn TEXT,
		ad_ou_dn TEXT,
		ad_synced_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		created_by TEXT DEFAULT '',
		updated_by TEXT DEFAULT '',
		version INTEGER DEFAULT 0
	)`
)

// createDeptTable 为 sys_dept 测试建表（原属 user_ou_service_test.go，62-01 迁入）。
// createDeptTable 为 sys_dept 测试建表（62-01 由 user_ou_service_test.go 迁入）。
const createDeptTable = `CREATE TABLE sys_dept (
	id TEXT PRIMARY KEY,
	dept_name TEXT DEFAULT '',
	dept_code TEXT DEFAULT '',
	parent_id TEXT,
	ancestors TEXT DEFAULT '',
	order_num INTEGER DEFAULT 0,
	status INTEGER DEFAULT 0,
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME
)`
