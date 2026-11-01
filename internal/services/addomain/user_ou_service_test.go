package addomain

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// ============================================================================
// user_ou_service_test.go — internal/services/addomain/user_ou_service.go 覆盖
//
// 该文件是 v1.51 从 GitHub 线上生产线回流本地的生产文件（此前本地从未覆盖，
// 导致 addomain 包覆盖率从 69.6% 掉到 67.5%）。本文件覆盖其 8 个函数：
//
//	NewUserOUService / HandleUserLoginAD / updateUserDeptAndADInfo
//	restoreDeletedUserWithADInfo / GetUserDeptByADOU / createDeptFromOUDN
//	buildAncestors / generateUniqueDeptCode
//
// 测试库：SQLite 内存库（每用例独立共享缓存实例，避免 :memory: 的连接池
// 多连接各持一份空库问题 —— mapper.UpsertMapping 内部会开事务）。
// ============================================================================

// createUserOUDeptTable 为 UserOUService 测试准备的 sys_dept 建表语句。
// 未直接复用 test_fixtures_test.go 的 createDeptTable：那份 fixture 缺少
// models.Department 的 leader / phone / email / is_external_org / remark /
// created_by / updated_by / version 列，而 GORM 读写 models.Department 会带上
// 这些列（缺列直接报 no such column）。
const createUserOUDeptTable = `CREATE TABLE sys_dept (
	id TEXT PRIMARY KEY,
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME,
	created_by TEXT DEFAULT '',
	updated_by TEXT DEFAULT '',
	version INTEGER DEFAULT 0,
	dept_name TEXT DEFAULT '',
	dept_code TEXT DEFAULT '',
	parent_id TEXT,
	ancestors TEXT DEFAULT '',
	order_num INTEGER DEFAULT 0,
	leader TEXT,
	phone TEXT,
	email TEXT,
	is_external_org INTEGER DEFAULT 0,
	status INTEGER DEFAULT 0,
	remark TEXT DEFAULT ''
)`

// createUserOUADConfigTable 对齐 models.ADConfig 的全部列。
const createUserOUADConfigTable = `CREATE TABLE sys_ad_config (
	id TEXT PRIMARY KEY,
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME,
	created_by TEXT DEFAULT '',
	updated_by TEXT DEFAULT '',
	version INTEGER DEFAULT 0,
	config_name TEXT,
	server_address TEXT,
	server_port INTEGER DEFAULT 389,
	domain_name TEXT,
	base_dn TEXT,
	admin_username TEXT,
	admin_password TEXT,
	use_ssl INTEGER DEFAULT 0,
	use_tls INTEGER DEFAULT 0,
	sync_enabled INTEGER DEFAULT 1,
	sync_interval INTEGER DEFAULT 3600,
	member_ou_dn TEXT,
	last_sync_at DATETIME,
	status INTEGER DEFAULT 0
)`

// createUserOURoleTable 对齐 models.Role 的全部列。
const createUserOURoleTable = `CREATE TABLE sys_role (
	id TEXT PRIMARY KEY,
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME,
	created_by TEXT DEFAULT '',
	updated_by TEXT DEFAULT '',
	version INTEGER DEFAULT 0,
	role_name TEXT,
	role_key TEXT,
	role_sort INTEGER DEFAULT 0,
	data_scope INTEGER DEFAULT 1,
	menu_check_strictly INTEGER DEFAULT 1,
	dept_check_strictly INTEGER DEFAULT 1,
	status INTEGER DEFAULT 0,
	remark TEXT DEFAULT ''
)`

// createUserOUserRoleTable 对齐 models.UserRole 的全部列。
const createUserOUserRoleTable = `CREATE TABLE sys_user_role (
	user_id TEXT NOT NULL,
	role_id TEXT NOT NULL,
	created_at DATETIME
)`

// userOUStatements 为 UserOUService 测试准备的建表语句集合。
// sys_user / sys_dept_ou_mapping 复用 test_fixtures_test.go 的共享 fixture
// （零副本约定），其余四张表在本文件内补齐。
var userOUStatements = []string{
	createUserTable,
	createDeptOUMappingTable,
	createUserOUDeptTable,
	createUserOUADConfigTable,
	createUserOURoleTable,
	createUserOUserRoleTable,
}

// userOUDBSeq 保证每次 setup 拿到独立的内存库（-count=2 重复运行时也不会
// 撞到上一轮残留的同名共享缓存库）。
var userOUDBSeq atomic.Int64

// setupUserOUServiceDB 构造 UserOUService 测试所需的 SQLite 内存库与服务实例。
func setupUserOUServiceDB(t *testing.T) (*gorm.DB, *UserOUService) {
	t.Helper()

	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared&_busy_timeout=5000",
		name, userOUDBSeq.Add(1))

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "打开 sqlite 内存库失败")

	sqlDB, err := db.DB()
	require.NoError(t, err, "获取底层连接池失败")
	t.Cleanup(func() { _ = sqlDB.Close() })

	for i, stmt := range userOUStatements {
		require.NoError(t, db.Exec(stmt).Error, "建表语句 %d 执行失败", i)
	}

	return db, NewUserOUService(db, NewDeptOUmapper(db))
}

// ---------------------------------------------------------------------------
// 测试数据辅助
// ---------------------------------------------------------------------------

// insertDept 插入一个部门。
func insertDept(t *testing.T, db *gorm.DB, id, name, code string, parentID *string, ancestors string) models.Department {
	t.Helper()
	dept := models.Department{
		BaseModel: models.BaseModel{ID: id},
		DeptName:  name,
		DeptCode:  code,
		ParentID:  parentID,
		Ancestors: ancestors,
		Status:    models.DeptStatusNormal,
	}
	require.NoError(t, db.Create(&dept).Error, "插入部门 %s 失败", name)
	return dept
}

// insertSoftDeletedDept 插入一个软删除部门。
func insertSoftDeletedDept(t *testing.T, db *gorm.DB, id, name, code string) models.Department {
	t.Helper()
	dept := models.Department{
		BaseModel: models.BaseModel{
			ID:        id,
			DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true},
		},
		DeptName: name,
		DeptCode: code,
		Status:   models.DeptStatusNormal,
	}
	require.NoError(t, db.Create(&dept).Error, "插入软删除部门 %s 失败", name)
	return dept
}

// insertUser 插入一个系统用户；deleted=true 时写入软删除标记。
func insertUser(t *testing.T, db *gorm.DB, username string, deptID *string, deleted bool) models.User {
	t.Helper()
	user := models.User{
		Username: username,
		Password: "test-pwd",
		Salt:     "test-salt",
		DeptID:   deptID,
	}
	if deleted {
		user.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
	}
	require.NoError(t, db.Create(&user).Error, "插入用户 %s 失败", username)
	return user
}

// insertADConfig 插入一条启用的 AD 配置。
func insertADConfig(t *testing.T, db *gorm.DB, id string) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO sys_ad_config (
			id, config_name, server_address, server_port, domain_name, base_dn,
			admin_username, admin_password, use_ssl, use_tls, sync_enabled,
			sync_interval, member_ou_dn, last_sync_at, status,
			created_at, updated_at, created_by, updated_by, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, 0, 1, 3600, NULL, NULL, 0,
			datetime('now'), datetime('now'), '', '', 0)`,
		id, "测试域控", "ldap://ad.local", 389, "cpic.com", "DC=cpic,DC=com",
		"admin", "secret").Error, "插入 AD 配置失败")
}

// insertRole 插入一个角色。
func insertRole(t *testing.T, db *gorm.DB, id, name, key string) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO sys_role (
			id, role_name, role_key, role_sort, data_scope,
			menu_check_strictly, dept_check_strictly, status, remark,
			created_at, updated_at, created_by, updated_by, version)
		VALUES (?, ?, ?, 0, 1, 1, 1, 0, '', datetime('now'), datetime('now'), '', '', 0)`,
		id, name, key).Error, "插入角色 %s 失败", key)
}

// insertOUInsertMapping 通过真实的 mapper.UpsertMapping 建立部门-OU 映射。
func insertOUInsertMapping(t *testing.T, db *gorm.DB, deptID, adConfigID, ouDN, ouName string) {
	t.Helper()
	now := time.Now()
	mapping := &models.DeptOUMapping{
		DeptID:      deptID,
		ADConfigID:  adConfigID,
		OUDN:        ouDN,
		OUName:      ouName,
		SyncEnabled: true,
		SyncStatus:  "synced",
		LastSyncAt:  &now,
	}
	require.NoError(t, NewDeptOUmapper(db).UpsertMapping(context.Background(), mapping),
		"插入 OU 映射失败")
}

// countRows 统计指定表的行数（不套软删除作用域）。
func countRows(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Table(table).Count(&n).Error, "统计 %s 行数失败", table)
	return n
}

// ---------------------------------------------------------------------------
// NewUserOUService
// ---------------------------------------------------------------------------

func TestNewUserOUService_注入db与mapper(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)

	require.NotNil(t, svc, "构造函数不应返回 nil")
	assert.Same(t, db, svc.db, "构造函数应原样持有传入的 db")
	require.NotNil(t, svc.mapper, "构造函数应持有传入的 mapper")
	assert.Same(t, db, svc.mapper.db, "mapper 应与 service 共用同一个 db")
}

// ---------------------------------------------------------------------------
// HandleUserLoginAD
// ---------------------------------------------------------------------------

func TestHandleUserLoginAD_命中OU映射_写入部门与AD信息(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertDept(t, db, "dept-1", "基础运维科", "基础运维科", nil, "")
	user := insertUser(t, db, "zhangsan", nil, false)

	const ouDN = "OU=基础运维科,DC=cpic,DC=com"
	insertOUInsertMapping(t, db, "dept-1", "ad-config-1", ouDN, "基础运维科")

	const userDN = "CN=zhangsan,OU=基础运维科,DC=cpic,DC=com"
	require.NoError(t, svc.HandleUserLoginAD(ctx, "zhangsan", userDN, ouDN))

	var got models.User
	require.NoError(t, db.Where("id = ?", user.ID).First(&got).Error)
	require.NotNil(t, got.DeptID, "登录后应绑定映射到的部门")
	assert.Equal(t, "dept-1", *got.DeptID)
	require.NotNil(t, got.AdDn, "应回写 AD 用户 DN")
	assert.Equal(t, userDN, *got.AdDn)
	require.NotNil(t, got.AdOuDn, "应回写 AD OU DN")
	assert.Equal(t, ouDN, *got.AdOuDn)
	require.NotNil(t, got.AdSyncedAt, "应记录 AD 同步时间")
}

func TestHandleUserLoginAD_OU无映射_自动建部门链并绑定用户(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertADConfig(t, db, "ad-config-1")
	user := insertUser(t, db, "lisi", nil, false)

	const ouDN = "OU=基础运维科,OU=分公司本部,DC=cpic,DC=com"
	require.NoError(t, svc.HandleUserLoginAD(ctx, "lisi",
		"CN=lisi,OU=基础运维科,OU=分公司本部,DC=cpic,DC=com", ouDN))

	// 1) OU 缺失时应自下而上补出两级部门链
	var parent models.Department
	require.NoError(t, db.Where("dept_name = ? AND parent_id IS NULL", "分公司本部").First(&parent).Error,
		"应创建顶层部门「分公司本部」")
	var leaf models.Department
	require.NoError(t, db.Where("dept_name = ?", "基础运维科").First(&leaf).Error,
		"应创建叶子部门「基础运维科」")
	require.NotNil(t, leaf.ParentID, "叶子部门应挂在「分公司本部」下")
	assert.Equal(t, parent.ID, *leaf.ParentID)
	assert.Equal(t, parent.ID, leaf.Ancestors, "父部门无祖先链时 ancestors 取父部门 ID")

	// 2) 用户应绑定到最底层部门
	var got models.User
	require.NoError(t, db.Where("id = ?", user.ID).First(&got).Error)
	require.NotNil(t, got.DeptID)
	assert.Equal(t, leaf.ID, *got.DeptID, "用户应绑定到最底层部门")

	// 3) OU 映射应落在最底层部门上
	var mapping models.DeptOUMapping
	require.NoError(t, db.Where("ou_dn = ?", ouDN).First(&mapping).Error)
	assert.Equal(t, leaf.ID, mapping.DeptID)
	assert.Equal(t, "基础运维科", mapping.OUName, "OU 名称应取 DN 中的 OU= 段")
	assert.Equal(t, "ad-config-1", mapping.ADConfigID)
	assert.True(t, mapping.SyncEnabled)
	assert.Equal(t, "synced", mapping.SyncStatus)
}

func TestHandleUserLoginAD_用户不存在_跳过部门设置(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)

	require.NoError(t, svc.HandleUserLoginAD(context.Background(), "ghost",
		"CN=ghost,OU=基础运维科,DC=cpic,DC=com", "OU=基础运维科,DC=cpic,DC=com"))

	assert.Equal(t, int64(0), countRows(t, db, "sys_dept"), "用户不存在时不应创建部门")
	assert.Equal(t, int64(0), countRows(t, db, "sys_dept_ou_mapping"), "用户不存在时不应创建映射")
}

func TestHandleUserLoginAD_软删除用户_拒绝自动恢复(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	deleted := insertUser(t, db, "zhangsan", nil, true)

	require.NoError(t, svc.HandleUserLoginAD(ctx, "zhangsan",
		"CN=zhangsan,OU=基础运维科,DC=cpic,DC=com", "OU=基础运维科,DC=cpic,DC=com"))

	// F-22 安全约定：软删除用户再次从 AD 登录不得被自动恢复（防账号接管）
	var got models.User
	require.NoError(t, db.Unscoped().Where("id = ?", deleted.ID).First(&got).Error)
	assert.True(t, got.DeletedAt.Valid, "软删除用户不应被自动恢复")
	assert.Nil(t, got.DeptID, "软删除用户不应被绑定部门")
	assert.Equal(t, int64(0), countRows(t, db, "sys_dept"), "软删除用户不应触发建部门")
}

func TestHandleUserLoginAD_自动建部门失败_不阻断登录(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	// 没有启用的 AD 配置 → createDeptFromOUDN 失败
	insertUser(t, db, "lisi", nil, false)

	require.NoError(t, svc.HandleUserLoginAD(ctx, "lisi",
		"CN=lisi,OU=基础运维科,DC=cpic,DC=com", "OU=基础运维科,DC=cpic,DC=com"),
		"自动建部门失败不应阻断登录流程")

	var got models.User
	require.NoError(t, db.Where("username = ?", "lisi").First(&got).Error)
	assert.Nil(t, got.DeptID, "建部门失败时用户不绑定部门")
	assert.Equal(t, int64(0), countRows(t, db, "sys_dept"))
}

func TestHandleUserLoginAD_查询用户出错_返回错误(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)

	// 删表制造非 ErrRecordNotFound 的查询错误
	require.NoError(t, db.Exec("DROP TABLE sys_user").Error)

	err := svc.HandleUserLoginAD(context.Background(), "zhangsan",
		"CN=zhangsan,OU=基础运维科,DC=cpic,DC=com", "OU=基础运维科,DC=cpic,DC=com")
	require.Error(t, err, "查询用户出错应向上返回错误")
	assert.Contains(t, err.Error(), "查询用户失败")
}

func TestHandleUserLoginAD_更新用户失败_不阻断登录(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertDept(t, db, "dept-1", "基础运维科", "基础运维科", nil, "")
	insertUser(t, db, "zhangsan", nil, false)

	const ouDN = "OU=基础运维科,DC=cpic,DC=com"
	insertOUInsertMapping(t, db, "dept-1", "ad-config-1", ouDN, "基础运维科")

	// 用触发器让 sys_user 的 UPDATE 必然失败
	require.NoError(t, db.Exec(`CREATE TRIGGER trg_block_sys_user_update
		BEFORE UPDATE ON sys_user
		BEGIN SELECT RAISE(ABORT, 'update blocked by test'); END;`).Error)

	require.NoError(t, svc.HandleUserLoginAD(ctx, "zhangsan",
		"CN=zhangsan,OU=基础运维科,DC=cpic,DC=com", ouDN),
		"更新用户信息失败不应阻断登录流程")

	var got models.User
	require.NoError(t, db.Where("username = ?", "zhangsan").First(&got).Error)
	assert.Nil(t, got.DeptID, "更新失败时部门不应被写入")
}

func TestHandleUserLoginAD_映射部门已软删除_不复用旧部门(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertADConfig(t, db, "ad-config-1")
	insertSoftDeletedDept(t, db, "dept-old", "基础运维科", "基础运维科")
	user := insertUser(t, db, "zhangsan", nil, false)

	const ouDN = "OU=基础运维科,DC=cpic,DC=com"
	insertOUInsertMapping(t, db, "dept-old", "ad-config-1", ouDN, "基础运维科")

	require.NoError(t, svc.HandleUserLoginAD(ctx, "zhangsan",
		"CN=zhangsan,OU=基础运维科,DC=cpic,DC=com", ouDN))

	// Phase 40：映射指向的部门已被软删除时不得复用，应另建新部门
	var got models.User
	require.NoError(t, db.Where("id = ?", user.ID).First(&got).Error)
	require.NotNil(t, got.DeptID)
	assert.NotEqual(t, "dept-old", *got.DeptID, "不得把用户挂到已软删除的部门上")

	var fresh models.Department
	require.NoError(t, db.Where("id = ?", *got.DeptID).First(&fresh).Error)
	assert.Equal(t, "基础运维科", fresh.DeptName)
}

// ---------------------------------------------------------------------------
// updateUserDeptAndADInfo
// ---------------------------------------------------------------------------

func TestUpdateUserDeptAndADInfo_写入部门与AD字段(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	user := insertUser(t, db, "wangwu", nil, false)

	const userDN = "CN=wangwu,OU=运维科,DC=cpic,DC=com"
	const ouDN = "OU=运维科,DC=cpic,DC=com"
	require.NoError(t, svc.updateUserDeptAndADInfo(ctx, user.ID, "dept-9", userDN, ouDN))

	var got models.User
	require.NoError(t, db.Where("id = ?", user.ID).First(&got).Error)
	require.NotNil(t, got.DeptID)
	assert.Equal(t, "dept-9", *got.DeptID)
	require.NotNil(t, got.AdDn)
	assert.Equal(t, userDN, *got.AdDn)
	require.NotNil(t, got.AdOuDn)
	assert.Equal(t, ouDN, *got.AdOuDn)
	require.NotNil(t, got.AdSyncedAt)
}

func TestUpdateUserDeptAndADInfo_数据库错误_返回错误(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)

	require.NoError(t, db.Exec("DROP TABLE sys_user").Error)

	err := svc.updateUserDeptAndADInfo(context.Background(), "u-1", "dept-1", "CN=x", "OU=y")
	require.Error(t, err, "数据库错误应向上返回")
}

// ---------------------------------------------------------------------------
// restoreDeletedUserWithADInfo
// ---------------------------------------------------------------------------

// TestRestoreDeletedUserWithADInfo_软删除用户现状语义 锁定该函数当前的生产行为。
//
// 背景：F-22 安全修复后 HandleUserLoginAD 不再自动恢复软删除用户，本函数已无
// 生产调用方（仅 auth.go.backup 的历史路径引用过）。实测行为：GORM 的软删除
// 作用域会给 UPDATE 语句追加 deleted_at IS NULL 条件，因此本函数第 1 步的
// "UPDATE deleted_at = NULL" 与第 3 步的 AD 信息回写都会命中 0 行 —— 软删除
// 用户实际并未被恢复，部门也不会被绑定。第 4 步的默认角色分配走原生 SQL
// （Exec），不受软删除作用域影响，仍然生效。
//
// 本测试锁定现状（防回归）。若后续修复该函数，需要同步更新这里的断言。
func TestRestoreDeletedUserWithADInfo_软删除用户现状语义(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	deleted := insertUser(t, db, "olduser", nil, true)

	err := svc.restoreDeletedUserWithADInfo(ctx, "olduser",
		"CN=olduser,OU=运维科,DC=cpic,DC=com", "OU=运维科,DC=cpic,DC=com", "dept-7")
	require.NoError(t, err, "恢复流程不应返回错误")

	var got models.User
	require.NoError(t, db.Unscoped().Where("id = ?", deleted.ID).First(&got).Error)
	assert.True(t, got.DeletedAt.Valid,
		"现状：UPDATE 被软删除作用域过滤，软删除用户并未被恢复")
	assert.Nil(t, got.DeptID,
		"现状：AD 信息回写同样被软删除作用域过滤，部门未绑定")
}

func TestRestoreDeletedUserWithADInfo_分配默认角色(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertRole(t, db, "role-common", "普通用户", "common")
	deleted := insertUser(t, db, "olduser", nil, true)

	require.NoError(t, svc.restoreDeletedUserWithADInfo(ctx, "olduser",
		"CN=olduser,OU=运维科,DC=cpic,DC=com", "OU=运维科,DC=cpic,DC=com", "dept-7"))

	assert.Equal(t, int64(1), countRows(t, db, "sys_user_role"),
		"无角色时应分配 role_key=common 的默认角色")

	var roleID string
	require.NoError(t, db.Table("sys_user_role").
		Where("user_id = ?", deleted.ID).
		Select("role_id").Row().Scan(&roleID))
	assert.Equal(t, "role-common", roleID)
}

func TestRestoreDeletedUserWithADInfo_已有角色不重复分配(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertRole(t, db, "role-common", "普通用户", "common")
	insertRole(t, db, "role-admin", "管理员", "admin")
	deleted := insertUser(t, db, "olduser", nil, true)
	require.NoError(t, db.Exec("INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)",
		deleted.ID, "role-admin").Error)

	require.NoError(t, svc.restoreDeletedUserWithADInfo(ctx, "olduser",
		"CN=olduser,OU=运维科,DC=cpic,DC=com", "OU=运维科,DC=cpic,DC=com", "dept-7"))

	assert.Equal(t, int64(1), countRows(t, db, "sys_user_role"),
		"已有角色时不应再分配默认角色")
}

func TestRestoreDeletedUserWithADInfo_无默认角色_不报错不分配(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertRole(t, db, "role-admin", "管理员", "admin")
	insertUser(t, db, "olduser", nil, true)

	require.NoError(t, svc.restoreDeletedUserWithADInfo(ctx, "olduser",
		"CN=olduser,OU=运维科,DC=cpic,DC=com", "OU=运维科,DC=cpic,DC=com", "dept-7"))

	assert.Equal(t, int64(0), countRows(t, db, "sys_user_role"),
		"缺少 role_key=common 的默认角色时不应插入关联")
}

func TestRestoreDeletedUserWithADInfo_用户角色表缺失_记日志不中断(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertRole(t, db, "role-common", "普通用户", "common")
	insertUser(t, db, "olduser", nil, true)

	// 角色关联表缺失 → 角色计数失败与默认角色分配失败都只记日志，不阻断
	require.NoError(t, db.Exec("DROP TABLE sys_user_role").Error)

	require.NoError(t, svc.restoreDeletedUserWithADInfo(ctx, "olduser",
		"CN=olduser,OU=运维科,DC=cpic,DC=com", "OU=运维科,DC=cpic,DC=com", "dept-7"),
		"角色表缺失不应导致恢复流程失败")
}

// 说明：第 1 步的 UPDATE 命中软删除行时受 deleted_at IS NULL 作用域限制（命中 0 行），
// 因此要让「补充 AD 信息」这一步真正执行并失败，必须用未被软删除的用户。
func TestRestoreDeletedUserWithADInfo_补充AD信息失败_返回错误(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertUser(t, db, "normaluser", nil, false)

	// 仅当写入 ad_dn 时才失败的触发器（第 1 步只写 deleted_at，不触发）
	require.NoError(t, db.Exec(`CREATE TRIGGER trg_block_ad_info_update
		BEFORE UPDATE ON sys_user WHEN NEW.ad_dn IS NOT NULL
		BEGIN SELECT RAISE(ABORT, 'ad info update blocked'); END;`).Error)

	err := svc.restoreDeletedUserWithADInfo(ctx, "normaluser",
		"CN=normaluser,OU=运维科,DC=cpic,DC=com", "OU=运维科,DC=cpic,DC=com", "dept-7")
	require.Error(t, err, "AD 信息回写失败应返回错误")
	assert.Contains(t, err.Error(), "补充AD信息失败")
}

func TestRestoreDeletedUserWithADInfo_恢复失败_返回错误(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertUser(t, db, "normaluser", nil, false)

	// 无条件拦截 UPDATE 的触发器，让第 1 步的恢复语句直接失败
	require.NoError(t, db.Exec(`CREATE TRIGGER trg_block_user_update
		BEFORE UPDATE ON sys_user
		BEGIN SELECT RAISE(ABORT, 'user update blocked'); END;`).Error)

	err := svc.restoreDeletedUserWithADInfo(ctx, "normaluser",
		"CN=normaluser,OU=运维科,DC=cpic,DC=com", "OU=运维科,DC=cpic,DC=com", "dept-7")
	require.Error(t, err, "恢复语句失败应返回错误")
	assert.Contains(t, err.Error(), "恢复用户失败")
}

// ---------------------------------------------------------------------------
// GetUserDeptByADOU
// ---------------------------------------------------------------------------

func TestGetUserDeptByADOU_返回用户部门(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertDept(t, db, "dept-1", "基础运维科", "基础运维科", nil, "")
	deptID := "dept-1"
	insertUser(t, db, "zhangsan", &deptID, false)

	dept, err := svc.GetUserDeptByADOU(ctx, "zhangsan")
	require.NoError(t, err)
	require.NotNil(t, dept)
	assert.Equal(t, "dept-1", dept.ID)
	assert.Equal(t, "基础运维科", dept.DeptName)
}

func TestGetUserDeptByADOU_用户未设置部门_返回错误(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertUser(t, db, "zhangsan", nil, false)

	dept, err := svc.GetUserDeptByADOU(ctx, "zhangsan")
	require.Error(t, err)
	assert.Nil(t, dept)
	assert.Contains(t, err.Error(), "未设置部门")
}

func TestGetUserDeptByADOU_用户不存在_返回错误(t *testing.T) {
	_, svc := setupUserOUServiceDB(t)

	dept, err := svc.GetUserDeptByADOU(context.Background(), "ghost")
	require.Error(t, err)
	assert.Nil(t, dept)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// ---------------------------------------------------------------------------
// createDeptFromOUDN
// ---------------------------------------------------------------------------

func TestCreateDeptFromOUDN_多级OU_自下而上建链并写映射(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertADConfig(t, db, "ad-config-1")

	const ouDN = "OU=基础运维科,OU=科技创新部,OU=分公司本部,DC=cpic,DC=com"
	deptID, err := svc.createDeptFromOUDN(ctx, ouDN, "CN=zhangsan,OU=基础运维科,OU=科技创新部,OU=分公司本部,DC=cpic,DC=com")
	require.NoError(t, err)
	require.NotEmpty(t, deptID)

	// 三级部门链：分公司本部 → 科技创新部 → 基础运维科
	var root models.Department
	require.NoError(t, db.Where("dept_name = ? AND parent_id IS NULL", "分公司本部").First(&root).Error)
	var mid models.Department
	require.NoError(t, db.Where("dept_name = ?", "科技创新部").First(&mid).Error)
	var leaf models.Department
	require.NoError(t, db.Where("id = ?", deptID).First(&leaf).Error)

	assert.Equal(t, "基础运维科", leaf.DeptName, "最底层部门名应取 DN 中最内层的 OU")
	require.NotNil(t, mid.ParentID)
	assert.Equal(t, root.ID, *mid.ParentID)
	require.NotNil(t, leaf.ParentID)
	assert.Equal(t, mid.ID, *leaf.ParentID)
	assert.Equal(t, root.ID+","+mid.ID, leaf.Ancestors, "祖先链应自顶向下累积")

	// 部门编码取部门名且唯一
	assert.Equal(t, "基础运维科", leaf.DeptCode)
	assert.Equal(t, "分公司本部", root.DeptCode)

	// OU 映射落在最底层部门
	var mapping models.DeptOUMapping
	require.NoError(t, db.Where("ou_dn = ?", ouDN).First(&mapping).Error)
	assert.Equal(t, leaf.ID, mapping.DeptID)
	assert.Equal(t, "基础运维科", mapping.OUName)
	assert.Equal(t, "ad-config-1", mapping.ADConfigID)
	assert.True(t, mapping.SyncEnabled)
	assert.Equal(t, "synced", mapping.SyncStatus)
	require.NotNil(t, mapping.LastSyncAt)
}

func TestCreateDeptFromOUDN_容器OU也建部门_DC段被过滤(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertADConfig(t, db, "ad-config-1")

	// 取生产代码注释里的真实 OU DN 形态：OU=CX 是容器 OU，DC= 段是域名部分
	const ouDN = "OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn"
	deptID, err := svc.createDeptFromOUDN(ctx, ouDN, "CN=zhangsan,OU=基础运维科,DC=cpic,DC=com")
	require.NoError(t, err)
	require.NotEmpty(t, deptID)

	// DC= 段不建部门；5 个 OU= 段各建一个部门
	assert.Equal(t, int64(5), countRows(t, db, "sys_dept"))

	var leaf models.Department
	require.NoError(t, db.Where("id = ?", deptID).First(&leaf).Error)
	assert.Equal(t, "基础运维科", leaf.DeptName)

	// 祖先链自顶向下：CX → 湖北分公司 → 分公司本部 → 科技创新部 → 基础运维科
	var cx models.Department
	require.NoError(t, db.Where("dept_name = ? AND parent_id IS NULL", "CX").First(&cx).Error)
	var chain []string
	for id := leaf.ID; ; {
		var d models.Department
		require.NoError(t, db.Where("id = ?", id).First(&d).Error)
		chain = append([]string{d.DeptName}, chain...)
		if d.ParentID == nil {
			break
		}
		id = *d.ParentID
	}
	assert.Equal(t, []string{"CX", "湖北分公司", "分公司本部", "科技创新部", "基础运维科"}, chain)
}

func TestCreateDeptFromOUDN_同名部门不同分支_互不串扰(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertADConfig(t, db, "ad-config-1")

	const ouDN1 = "OU=同名部门,OU=甲分部,DC=cpic,DC=com"
	const ouDN2 = "OU=同名部门,OU=乙分部,DC=cpic,DC=com"

	deptID1, err := svc.createDeptFromOUDN(ctx, ouDN1, "CN=u1,OU=同名部门,OU=甲分部,DC=cpic,DC=com")
	require.NoError(t, err)
	deptID2, err := svc.createDeptFromOUDN(ctx, ouDN2, "CN=u2,OU=同名部门,OU=乙分部,DC=cpic,DC=com")
	require.NoError(t, err)

	// flip-flop 修复：同名叶子部门在不同父部门下必须是两条独立记录
	assert.NotEqual(t, deptID1, deptID2, "不同分支下的同名部门不应复用同一条记录")
	assert.Equal(t, int64(4), countRows(t, db, "sys_dept"), "两个分支各建 2 级部门")

	var d1, d2 models.Department
	require.NoError(t, db.Where("id = ?", deptID1).First(&d1).Error)
	require.NoError(t, db.Where("id = ?", deptID2).First(&d2).Error)
	assert.Equal(t, "同名部门", d1.DeptName)
	assert.Equal(t, "同名部门", d2.DeptName)
	require.NotNil(t, d1.ParentID)
	require.NotNil(t, d2.ParentID)
	assert.NotEqual(t, *d1.ParentID, *d2.ParentID)

	// 部门编码必须唯一递增
	assert.Equal(t, "同名部门", d1.DeptCode)
	assert.Equal(t, "同名部门-1", d2.DeptCode)
}

func TestCreateDeptFromOUDN_重复调用_复用已有部门(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertADConfig(t, db, "ad-config-1")

	const ouDN = "OU=基础运维科,OU=分公司本部,DC=cpic,DC=com"
	first, err := svc.createDeptFromOUDN(ctx, ouDN, "CN=u1,OU=基础运维科,OU=分公司本部,DC=cpic,DC=com")
	require.NoError(t, err)
	second, err := svc.createDeptFromOUDN(ctx, ouDN, "CN=u2,OU=基础运维科,OU=分公司本部,DC=cpic,DC=com")
	require.NoError(t, err)

	assert.Equal(t, first, second, "重复调用应复用已存在的部门")
	assert.Equal(t, int64(2), countRows(t, db, "sys_dept"), "不应重复建部门")
	assert.Equal(t, int64(1), countRows(t, db, "sys_dept_ou_mapping"), "映射应幂等")
}

func TestCreateDeptFromOUDN_无效OUDN_返回错误(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertADConfig(t, db, "ad-config-1")

	deptID, err := svc.createDeptFromOUDN(ctx, "", "CN=u1")
	require.Error(t, err)
	assert.Empty(t, deptID)
	assert.Contains(t, err.Error(), "无效的OU DN")
}

func TestCreateDeptFromOUDN_嵌套超限_返回错误(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertADConfig(t, db, "ad-config-1")

	// 闸门统计的是 DN 段总数（OU= 与 DC= 一起算，ParseOUDN 不做过滤），
	// 17 段（15 个 OU + 2 个 DC）超过 maxOUDepth=16
	parts := make([]string, 0, 17)
	for i := 0; i < 15; i++ {
		parts = append(parts, fmt.Sprintf("OU=L%d", i))
	}
	parts = append(parts, "DC=cpic", "DC=com")

	deptID, err := svc.createDeptFromOUDN(ctx, strings.Join(parts, ","), "CN=u1")
	require.Error(t, err)
	assert.Empty(t, deptID)
	assert.Contains(t, err.Error(), "嵌套层数超过上限")
	assert.Equal(t, int64(0), countRows(t, db, "sys_dept"), "超限时不应创建部门")
}

func TestCreateDeptFromOUDN_嵌套刚好到上限_允许创建(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertADConfig(t, db, "ad-config-1")

	// 16 段（14 个 OU + 2 个 DC）正好等于 maxOUDepth，应放行
	parts := make([]string, 0, 16)
	for i := 0; i < 14; i++ {
		parts = append(parts, fmt.Sprintf("OU=L%d", i))
	}
	parts = append(parts, "DC=cpic", "DC=com")

	deptID, err := svc.createDeptFromOUDN(ctx, strings.Join(parts, ","), "CN=u1")
	require.NoError(t, err)
	require.NotEmpty(t, deptID)
	assert.Equal(t, int64(14), countRows(t, db, "sys_dept"))
}

func TestCreateDeptFromOUDN_缺少AD配置_返回错误(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	deptID, err := svc.createDeptFromOUDN(ctx, "OU=基础运维科,DC=cpic,DC=com", "CN=u1")
	require.Error(t, err)
	assert.Empty(t, deptID)
	assert.Contains(t, err.Error(), "获取AD配置失败")
	assert.Equal(t, int64(0), countRows(t, db, "sys_dept"))
}

func TestCreateDeptFromOUDN_部门数达上限_拒绝创建(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertADConfig(t, db, "ad-config-1")

	// 批量灌到 maxAutoCreatedDepts=5000 闸门
	require.NoError(t, db.Exec(`WITH RECURSIVE cnt(x) AS (
			SELECT 1 UNION ALL SELECT x + 1 FROM cnt WHERE x < 5000)
		INSERT INTO sys_dept (id, dept_name, dept_code, status, created_at, updated_at)
		SELECT 'bulk-' || x, 'bulkdept' || x, 'bulkdept' || x, 0, datetime('now'), datetime('now')
		FROM cnt`).Error, "批量插入部门失败")
	require.Equal(t, int64(5000), countRows(t, db, "sys_dept"))

	deptID, err := svc.createDeptFromOUDN(ctx, "OU=新部门,DC=cpic,DC=com", "CN=u1")
	require.Error(t, err)
	assert.Empty(t, deptID)
	assert.Contains(t, err.Error(), "系统部门数已达上限")
	assert.Equal(t, int64(5000), countRows(t, db, "sys_dept"), "超限时不应新增部门")
}

func TestCreateDeptFromOUDN_DN无OU段_返回错误(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertADConfig(t, db, "ad-config-1")

	// 只有 DC= 段 → 解析不出任何部门名 → 无法确定部门 ID
	deptID, err := svc.createDeptFromOUDN(ctx, "DC=cpic,DC=com", "CN=u1")
	require.Error(t, err)
	assert.Empty(t, deptID)
	assert.Contains(t, err.Error(), "无法确定部门ID")
	assert.Equal(t, int64(0), countRows(t, db, "sys_dept"))
}

func TestCreateDeptFromOUDN_建部门失败_返回错误(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertADConfig(t, db, "ad-config-1")
	require.NoError(t, db.Exec(`CREATE TRIGGER trg_block_dept_insert
		BEFORE INSERT ON sys_dept
		BEGIN SELECT RAISE(ABORT, 'dept insert blocked'); END;`).Error)

	deptID, err := svc.createDeptFromOUDN(ctx, "OU=基础运维科,DC=cpic,DC=com", "CN=u1")
	require.Error(t, err)
	assert.Empty(t, deptID)
	assert.Contains(t, err.Error(), "创建部门 基础运维科 失败")
}

func TestCreateDeptFromOUDN_查询部门失败_返回错误(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	insertADConfig(t, db, "ad-config-1")
	// 删表让部门查询报非「记录不存在」的错误
	require.NoError(t, db.Exec("DROP TABLE sys_dept").Error)

	deptID, err := svc.createDeptFromOUDN(ctx, "OU=基础运维科,DC=cpic,DC=com", "CN=u1")
	require.Error(t, err)
	assert.Empty(t, deptID)
	assert.Contains(t, err.Error(), "查询部门 基础运维科 失败")
}

// ---------------------------------------------------------------------------
// buildAncestors
// ---------------------------------------------------------------------------

func TestBuildAncestors(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	// 顶层部门：没有父部门 → 空串
	assert.Equal(t, "", svc.buildAncestors(ctx, nil))

	root := insertDept(t, db, "dept-root", "总部", "总部", nil, "")

	// 父部门存在但无祖先链 → 取父部门 ID
	assert.Equal(t, root.ID, svc.buildAncestors(ctx, &root.ID))

	// 父部门已有祖先链 → 祖先链 + 父部门 ID
	child := insertDept(t, db, "dept-child", "研发部", "研发部", &root.ID, root.ID)
	assert.Equal(t, root.ID+","+child.ID, svc.buildAncestors(ctx, &child.ID))

	// 父部门不存在 → 空串
	missing := "dept-missing"
	assert.Equal(t, "", svc.buildAncestors(ctx, &missing))
}

// ---------------------------------------------------------------------------
// generateUniqueDeptCode
// ---------------------------------------------------------------------------

func TestGenerateUniqueDeptCode(t *testing.T) {
	db, svc := setupUserOUServiceDB(t)
	ctx := context.Background()

	// 编码未被占用 → 直接用部门名
	assert.Equal(t, "运维部", svc.generateUniqueDeptCode(ctx, "运维部"))

	// 编码已被占用 → 追加序号
	insertDept(t, db, "dept-1", "运维部", "运维部", nil, "")
	assert.Equal(t, "运维部-1", svc.generateUniqueDeptCode(ctx, "运维部"))

	// 序号 1 也被占用 → 继续递增
	insertDept(t, db, "dept-2", "运维部副本", "运维部-1", nil, "")
	assert.Equal(t, "运维部-2", svc.generateUniqueDeptCode(ctx, "运维部"))

	// 1..100 序号全部被占用 → 回退到时间戳编码
	for i := 1; i <= 100; i++ {
		insertDept(t, db, fmt.Sprintf("dept-full-%03d", i), fmt.Sprintf("满位部门%d", i),
			fmt.Sprintf("运维部-%d", i), nil, "")
	}
	fallback := svc.generateUniqueDeptCode(ctx, "运维部")
	assert.Regexp(t, `^运维部-\d{9,}$`, fallback, "序号耗尽时应回退到时间戳编码")
}
