package addomain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

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
		phone TEXT,
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

func TestHandleUserLoginAD_UserNotFound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	db.Exec(createDeptOUMappingTable)
	db.Exec(createUserTable)

	mapper := NewDeptOUmapper(db)
	service := NewUserOUService(db, mapper)

	ctx := context.Background()

	// 测试用户不存在场景（应返回nil，不报错）
	err = service.HandleUserLoginAD(ctx, "nonexistent", "CN=user,OU=TestDept,DC=company,DC=com", "OU=TestDept,DC=company,DC=com")
	assert.NoError(t, err) // 不应该报错，只记录日志
}

func TestHandleUserLoginAD_MappingNotFound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	db.Exec(createDeptOUMappingTable)
	db.Exec(createUserTable)

	// 插入测试用户
	db.Exec(`INSERT INTO sys_user (id, username, password, salt, auth_source) VALUES ('user-1', 'testuser', '', '', 'local')`)

	mapper := NewDeptOUmapper(db)
	service := NewUserOUService(db, mapper)

	ctx := context.Background()

	// 测试映射不存在场景（应返回nil，不报错）
	err = service.HandleUserLoginAD(ctx, "testuser", "CN=testuser,OU=NonExistent,DC=company,DC=com", "OU=NonExistent,DC=company,DC=com")
	assert.NoError(t, err) // 不应该报错，只记录警告日志
}

func TestHandleUserLoginAD_Success(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	db.Exec(createDeptOUMappingTable)
	db.Exec(createUserTable)

	// Phase 40 修复后 FindDeptByOUDN JOIN sys_dept 过滤软删除，测试需补建该表
	db.Exec(`CREATE TABLE sys_dept (
		id TEXT PRIMARY KEY,
		dept_name TEXT NOT NULL,
		dept_code TEXT NOT NULL UNIQUE,
		parent_id TEXT,
		ancestors TEXT DEFAULT '',
		order_num INTEGER DEFAULT 0,
		leader TEXT,
		leader_name TEXT,
		leader_username TEXT,
		phone TEXT,
		email TEXT,
		is_external_org INTEGER DEFAULT 0,
		status INTEGER DEFAULT 0,
		remark TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		created_by TEXT,
		updated_by TEXT,
		version INTEGER DEFAULT 0
	)`)
	db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, status) VALUES ('dept-1', 'TestDept', 'TEST001', 0)`)

	mapper := NewDeptOUmapper(db)
	service := NewUserOUService(db, mapper)

	ctx := context.Background()

	// 准备映射数据
	db.Exec(`INSERT INTO sys_dept_ou_mapping (id, dept_id, ad_config_id, ou_dn, ou_name, sync_enabled, sync_status)
		VALUES ('map-1', 'dept-1', 'config-1', 'OU=TestDept,DC=company,DC=com', 'TestDept', 1, 'synced')`)

	// 插入测试用户
	db.Exec(`INSERT INTO sys_user (id, username, password, salt, auth_source) VALUES ('user-1', 'testuser', '', '', 'local')`)

	// 测试成功场景
	err = service.HandleUserLoginAD(ctx, "testuser", "CN=testuser,OU=TestDept,DC=company,DC=com", "OU=TestDept,DC=company,DC=com")
	assert.NoError(t, err)

	// 验证用户信息已更新
	var deptID, adUserDN, adOUDN string
	row := db.Raw(`SELECT dept_id, ad_dn, ad_ou_dn FROM sys_user WHERE username = ?`, "testuser").Row()
	row.Scan(&deptID, &adUserDN, &adOUDN)

	assert.Equal(t, "dept-1", deptID)
	assert.Equal(t, "CN=testuser,OU=TestDept,DC=company,DC=com", adUserDN)
	assert.Equal(t, "OU=TestDept,DC=company,DC=com", adOUDN)
}

func TestGetUserDeptByADOU(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	db.Exec(createUserTable)

	mapper := NewDeptOUmapper(db)
	service := NewUserOUService(db, mapper)

	ctx := context.Background()

	// 测试用户不存在场景
	dept, err := service.GetUserDeptByADOU(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, dept)
}

func TestUpdateUserDeptAndADInfo(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	db.Exec(createUserTable)

	mapper := NewDeptOUmapper(db)
	service := NewUserOUService(db, mapper)

	ctx := context.Background()

	// 插入测试用户
	db.Exec(`INSERT INTO sys_user (id, username, password, salt, auth_source) VALUES ('user-1', 'testuser', '', '', 'local')`)

	// 测试更新用户信息
	err = service.updateUserDeptAndADInfo(ctx, "user-1", "dept-1", "CN=testuser,OU=TestDept,DC=company,DC=com", "OU=TestDept,DC=company,DC=com")
	assert.NoError(t, err)

	// 验证更新
	var deptID, adUserDN, adOUDN string
	row := db.Raw(`SELECT dept_id, ad_dn, ad_ou_dn FROM sys_user WHERE id = ?`, "user-1").Row()
	row.Scan(&deptID, &adUserDN, &adOUDN)

	assert.Equal(t, "dept-1", deptID)
	assert.Equal(t, "CN=testuser,OU=TestDept,DC=company,DC=com", adUserDN)
	assert.Equal(t, "OU=TestDept,DC=company,DC=com", adOUDN)
}

// createDeptTable 为 buildAncestors / generateUniqueDeptCode 测试创建 sys_dept 表。
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

// setupUserOUSvcDB 构建内存 DB 并返回已注入的 UserOUService。
func setupUserOUSvcDB(t *testing.T) (*UserOUService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	db.Exec(createDeptTable)
	mapper := NewDeptOUmapper(db)
	return NewUserOUService(db, mapper), db
}

// TestBuildAncestors_NilParent 验证无父部门时返回空字符串。
func TestBuildAncestors_NilParent(t *testing.T) {
	svc, _ := setupUserOUSvcDB(t)
	ctx := context.Background()

	result := svc.buildAncestors(ctx, nil)

	assert.Equal(t, "", result, "nil parent should yield empty ancestors")
}

// TestBuildAncestors_WithParent 验证能正确拼接父部门的 ancestors。
func TestBuildAncestors_WithParent(t *testing.T) {
	svc, db := setupUserOUSvcDB(t)
	ctx := context.Background()

	// 插入父部门：ancestors="root-id"
	db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, ancestors) VALUES ('parent-1', '父部门', 'P1', 'root-id')`)

	parentID := "parent-1"
	result := svc.buildAncestors(ctx, &parentID)

	assert.Equal(t, "root-id,parent-1", result, "ancestors should be parent's ancestors + parent id")
}

// TestBuildAncestors_ParentWithEmptyAncestors 验证父部门无 ancestors 时只返回 parentID。
func TestBuildAncestors_ParentWithEmptyAncestors(t *testing.T) {
	svc, db := setupUserOUSvcDB(t)
	ctx := context.Background()

	// 父部门 ancestors 为空
	db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, ancestors) VALUES ('top-1', '顶层', 'T1', '')`)

	parentID := "top-1"
	result := svc.buildAncestors(ctx, &parentID)

	assert.Equal(t, "top-1", result, "when parent has no ancestors, result is just parent id")
}

// TestBuildAncestors_ParentNotFound 验证父部门不存在时返回空字符串（容错）。
func TestBuildAncestors_ParentNotFound(t *testing.T) {
	svc, _ := setupUserOUSvcDB(t)
	ctx := context.Background()

	parentID := "nonexistent"
	result := svc.buildAncestors(ctx, &parentID)

	assert.Equal(t, "", result, "missing parent should yield empty ancestors (fail-safe)")
}

// TestGenerateUniqueDeptCode_Available 验证名称未被占用时直接用作编码。
func TestGenerateUniqueDeptCode_Available(t *testing.T) {
	svc, _ := setupUserOUSvcDB(t)
	ctx := context.Background()

	code := svc.generateUniqueDeptCode(ctx, "新部门")

	assert.Equal(t, "新部门", code, "unused name should be returned as-is")
}

// TestGenerateUniqueDeptCode_DuplicateAddsSuffix 验证名称已占用时添加序号后缀。
func TestGenerateUniqueDeptCode_DuplicateAddsSuffix(t *testing.T) {
	svc, db := setupUserOUSvcDB(t)
	ctx := context.Background()

	// 已存在 dept_code="测试部门"
	db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, ancestors) VALUES ('d-1', '测试部门', '测试部门', '')`)

	code := svc.generateUniqueDeptCode(ctx, "测试部门")

	assert.Equal(t, "测试部门-1", code, "duplicate name should get -1 suffix")
}

// TestGenerateUniqueDeptCode_SecondDuplicateAddsIncrementedSuffix 验证多个重名时序号递增。
func TestGenerateUniqueDeptCode_SecondDuplicateAddsIncrementedSuffix(t *testing.T) {
	svc, db := setupUserOUSvcDB(t)
	ctx := context.Background()

	// 占用 "测试部门" 和 "测试部门-1"
	db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code) VALUES ('d-1', '测试部门', '测试部门')`)
	db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code) VALUES ('d-2', '测试部门', '测试部门-1')`)

	code := svc.generateUniqueDeptCode(ctx, "测试部门")

	assert.Equal(t, "测试部门-2", code, "third duplicate should get -2 suffix")
}
