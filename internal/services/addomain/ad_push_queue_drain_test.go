//go:build !skip_db_tests
// +build !skip_db_tests

// Phase 58 Wave 0 —— 排水器（ad_push_queue drainer）测试基座。
//
// 本文件由 58-01 Task 2 建立，是后续 compare-and-move / 终态分流 / 退避重试
// 用例的共同环境。所有本 phase 追加的用例命名一律 `TestPhase58_*`，
// 便于 `go test ./internal/services/addomain/ -run TestPhase58 -count=1` 一把捞起。
//
// 五张表：
//   - ad_push_queue        —— 直调 migrations.Migrate215CreateADPushQueue 建表（含
//     uq_ad_push_queue_active 部分唯一索引；手写 DDL 会制造
//     第二份事实源，与模型"不声明 uniqueIndex"的红线冲突）
//   - sys_user             —— 复用 user_ou_service_test.go 的 createUserTable（零副本）
//   - sys_ad_config        —— 本文件的 createDrainADConfigTable（含 base_dn / config_name，F-F）
//   - sys_dept             —— 复用 user_ou_service_test.go 的 createDeptTable（零副本）
//   - sys_dept_ou_mapping  —— 复用 user_ou_service_test.go 的 createDeptOUMappingTable（零副本）
//
// 编译约束：本文件引用同包 fakePool78（定义于 failover_client_78_07_test.go，
// 带 `!skip_db_tests` build tag），故本文件同样携带该 tag，避免 `-tags skip_db_tests`
// 构建时出现 undefined: fakePool78。
package addomain

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/db/migrations"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// createDrainADConfigTable 排水测试专用的 sys_ad_config 建表语句。
//
// ⚠ F-F（Phase 57 遗留缺口）：必须含 base_dn 与 config_name 两列 ——
// D-02 的最长后缀匹配（按 sys_user.ad_dn 反查归属配置）要读 base_dn，
// 缺列则相关用例只能绕开 GORM 退化成一堆裸 SQL 断言。
//
// 时间列一律 DATETIME（不写 timestamptz）：glebarez/go-sqlite 仅在
// declType ∈ {date, datetime, time, timestamp} 时把列扫成 time.Time，
// 写 timestamptz 会扫成 string，Scan 到 *time.Time 时报错。
//
// deleted_at 必须存在：models.ADConfig 内嵌 BaseModel，GORM 软删作用域引用它。
const createDrainADConfigTable = `CREATE TABLE sys_ad_config (
	id TEXT PRIMARY KEY,
	config_name TEXT,
	base_dn TEXT,
	server_address TEXT,
	server_port INTEGER,
	domain_name TEXT,
	admin_username TEXT,
	admin_password TEXT,
	use_ssl INTEGER NOT NULL DEFAULT 0,
	use_tls INTEGER NOT NULL DEFAULT 0,
	sync_enabled INTEGER NOT NULL DEFAULT 1,
	sync_interval INTEGER,
	member_ou_dn TEXT,
	last_sync_at DATETIME,
	status INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME
)`

// setupDrainTestDB 构建排水器测试所需的内存 sqlite DB（五张表）。
//
// sys_user / sys_dept / sys_dept_ou_mapping 三张表**复用** user_ou_service_test.go
// 中已有的包级 const，不在此处再声明副本 —— 两处副本一旦漂移，会让
// "某一套环境缺列" 类故障以业务 bug 的面目出现。
func setupDrainTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(createUserTable).Error)          // 复用 user_ou_service_test.go
	require.NoError(t, db.Exec(createDeptOUMappingTable).Error) // 复用 user_ou_service_test.go
	require.NoError(t, db.Exec(createDeptTable).Error)          // 复用 user_ou_service_test.go（sys_dept）
	require.NoError(t, db.Exec(createDrainADConfigTable).Error)
	// 队列表走迁移：uq_ad_push_queue_active / idx_ad_push_queue_status_retry /
	// idx_ad_push_queue_entity_id 三个索引由迁移建，不手写。
	require.NoError(t, migrations.Migrate215CreateADPushQueue(db))
	return db
}

// ==================== seed helper（导出给后续 plan 复用） ====================

// drainNow 统一的时间字面量（sqlite DATETIME 可读格式）。
func drainNow() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05")
}

// drainBoolToInt 把 Go bool 收敛成 sqlite INTEGER 的 0/1，
// 避免依赖驱动对 bool 的隐式转换。
func drainBoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// seedDrainADConfig 向 sys_ad_config 插入一条 AD 配置。
//
// syncEnabled / status 显式写入，供 IsADSyncEnabled 前置门控与
// D-02 最长后缀匹配用例分别调参。deleted_at 写 NULL（软删作用域外）。
func seedDrainADConfig(t *testing.T, db *gorm.DB, id, configName, baseDN string, syncEnabled bool, status int) {
	t.Helper()
	now := drainNow()
	require.NoError(t, db.Exec(
		`INSERT INTO sys_ad_config (id, config_name, base_dn, sync_enabled, status, created_at, updated_at, deleted_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, NULL)`,
		id, configName, baseDN, drainBoolToInt(syncEnabled), status, now, now,
	).Error)
}

// seedDrainUser 向 sys_user 插入一行并返回构造好的 *models.User 供断言比对。
//
// password / salt 写空串满足 NOT NULL；auth_source 写 'local'（AD 由调用方按需覆盖）；
// deleted_at 写 NULL（F-07 孤儿用例需要软删行时用 softDeleteDrainUser）。
func seedDrainUser(t *testing.T, db *gorm.DB, id, username, adDn, adOuDn string, deptID *string) *models.User {
	t.Helper()
	now := drainNow()
	adUsername := username
	require.NoError(t, db.Exec(
		`INSERT INTO sys_user
			(id, username, password, salt, status, dept_id, auth_source, ad_username, ad_dn, ad_ou_dn, ad_synced_at, created_at, updated_at, deleted_at)
		 VALUES (?, ?, '', '', 0, ?, 'local', ?, ?, ?, NULL, ?, ?, NULL)`,
		id, username, deptID, adUsername, adDn, adOuDn, now, now,
	).Error)

	return &models.User{
		BaseModel:  models.BaseModel{ID: id},
		Username:   username,
		DeptID:     deptID,
		AuthSource: "local",
		ADUsername: &adUsername,
		AdDn:       &adDn,
		AdOuDn:     &adOuDn,
	}
}

// seedDrainDept 向 sys_dept 插入一行。
//
// 可选：只有需要断言 LDAP 侧 department 属性（同步路径读 dept.DeptName）的用例才需要；
// 没有该行时 syncUserAttributes 的 First(&dept) 报错被吞掉，只是不带 department 属性，
// 不影响其它断言。
func seedDrainDept(t *testing.T, db *gorm.DB, id, name string) {
	t.Helper()
	now := drainNow()
	require.NoError(t, db.Exec(
		`INSERT INTO sys_dept (id, dept_name, ancestors, status, created_at, updated_at, deleted_at)
		 VALUES (?, ?, '0', 0, ?, ?, NULL)`,
		id, name, now, now,
	).Error)
}

// seedDrainQueueRow 铺一行 ad_push_queue（走 GORM，BeforeCreate 兜底 UUID 与 pending）。
// id 传空串时由 BeforeCreate 生成；status 传空串时同样落到 pending。
func seedDrainQueueRow(t *testing.T, db *gorm.DB, id, entityID, actionType, status string, attempts int, nextRetryAt *time.Time) models.ADPushQueue {
	t.Helper()
	row := models.ADPushQueue{
		ID:          id,
		EntityType:  models.ADPushEntityUser,
		EntityID:    entityID,
		ActionType:  models.ADPushActionType(actionType),
		Status:      models.ADPushStatus(status),
		Attempts:    attempts,
		NextRetryAt: nextRetryAt,
	}
	require.NoError(t, db.Create(&row).Error)
	return row
}

// ==================== 装配函数（Plan 04/05 直接消费，签名已定死） ====================

// newDrainTestLDAP 返回已装配好错误注入的 mock LDAP：DNExists 默认 true。
//
// ⚠ T-58-W0-02：dnExistsRes 的零值是 false，会让 syncUserAttributes 走
// "DNExists=false → 清空 ad_dn 后返回 nil" 分支，把 code 32 用例变成假绿。
// 必须由本函数固定为 true。
func newDrainTestLDAP() *mockLDAPClient {
	return &mockLDAPClient{dnExistsRes: true}
}

// newDrainTestPool 返回账号池 fake；不传账号即 listAvailableRes == nil，
// 模拟熔断耗尽 → FailoverClient.ExecuteWithFailover 返回 ErrAllAccountsUnavailable。
func newDrainTestPool(accounts ...models.ADServiceAccount) *fakePool78 {
	return &fakePool78{listAvailableRes: accounts}
}

// drainTestAccount 构造一个可用服务账号（models.ADServiceAccount 不内嵌 BaseModel）。
func drainTestAccount(id, configID string) models.ADServiceAccount {
	return models.ADServiceAccount{
		ID:                 id,
		ConfigID:           configID,
		Username:           "svc-" + id,
		PasswordCiphertext: "not-used-in-unit-test",
		Status:             models.ADAccountStatusAvailable,
	}
}

// ==================== 基座自检 ====================

// TestPhase58_DrainTestBase_FixtureUsable 保证后续 RED 是真的红，不是环境坏。
//
// 逐条锁住基座的关键假设：F-F 两列可读、队列表 uuid/DATETIME/nullable 列无坑、
// mock 与账号池装配的默认值正确、sys_dept.dept_name 往返可用。
func TestPhase58_DrainTestBase_FixtureUsable(t *testing.T) {
	db := setupDrainTestDB(t)

	// --- F-F：sys_ad_config 的 base_dn / config_name 两列必须真的可被读取 ---
	seedDrainADConfig(t, db, "cfg-1", "总部域", "DC=corp,DC=com", true, 0)

	var cfgCount int64
	require.NoError(t, db.Table("sys_ad_config").
		Where("base_dn = ?", "DC=corp,DC=com").Count(&cfgCount).Error)
	assert.Equal(t, int64(1), cfgCount,
		"F-F 补列：base_dn 必须可被 WHERE 命中（D-02 最长后缀匹配依赖它）")

	var cfgName string
	require.NoError(t, db.Raw(`SELECT config_name FROM sys_ad_config WHERE id = ?`, "cfg-1").
		Scan(&cfgName).Error)
	assert.Equal(t, "总部域", cfgName, "F-F 补列：config_name 必须可读")

	// --- 队列表：GORM 写入 + 原生 SELECT 读回（uuid / DATETIME / nullable 列）---
	seedDrainQueueRow(t, db, "", "user-1", string(models.ADPushActionUpdateUser),
		string(models.ADPushStatusPending), 0, nil)

	var q struct {
		EntityID    string     `gorm:"column:entity_id"`
		Status      string     `gorm:"column:status"`
		Attempts    int        `gorm:"column:attempts"`
		NextRetryAt *time.Time `gorm:"column:next_retry_at"`
	}
	require.NoError(t, db.Raw(
		`SELECT entity_id, status, attempts, next_retry_at FROM ad_push_queue WHERE entity_id = ?`,
		"user-1").Scan(&q).Error)
	assert.Equal(t, "user-1", q.EntityID)
	assert.Equal(t, "pending", q.Status)
	assert.Equal(t, 0, q.Attempts)
	assert.Nil(t, q.NextRetryAt, "未设退避时间的行 next_retry_at 必须为 NULL（= 立即可拾取）")

	// --- 装配函数默认值 ---
	ldapMock := newDrainTestLDAP()
	assert.True(t, ldapMock.dnExistsRes,
		"T-58-W0-02：newDrainTestLDAP 必须固定 dnExistsRes=true，否则 code 32 用例假绿")
	assert.Empty(t, ldapMock.moveUserCalls)
	assert.Empty(t, ldapMock.disableUserCalls)
	assert.Empty(t, ldapMock.updateUserAttrCalls)

	emptyPool := newDrainTestPool()
	assert.Nil(t, emptyPool.listAvailableRes, "不传账号 → ListAvailable 返回 nil，驱动 ErrAllAccountsUnavailable")

	onePool := newDrainTestPool(drainTestAccount("acc-1", "cfg-1"))
	require.Len(t, onePool.listAvailableRes, 1)

	acct := drainTestAccount("acc-1", "cfg-1")
	assert.Equal(t, models.ADAccountStatusAvailable, acct.Status)
	assert.Equal(t, "cfg-1", acct.ConfigID)
	assert.Equal(t, "svc-acc-1", acct.Username)
	assert.NotEmpty(t, acct.PasswordCiphertext, "password_ciphertext 列 NOT NULL，测试构造也必须非空")

	// --- sys_dept / sys_user 往返：锁住同步路径 "读 dept.DeptName" 的基座假设 ---
	// 若 sys_dept 缺 dept_name 列，First(&dept) 报错会被 syncUserAttributes 吞掉，
	// 后续 wave 的 department 属性断言会以"缺属性"失败，看起来像业务 bug 实为基座缺口。
	seedDrainDept(t, db, "dept-1", "研发部")
	deptID := "dept-1"
	seedDrainUser(t, db, "user-dept", "deptuser",
		"CN=deptuser,OU=研发部,DC=corp,DC=com", "OU=研发部,DC=corp,DC=com", &deptID)

	var deptName string
	require.NoError(t, db.Raw(`SELECT dept_name FROM sys_dept WHERE id = ?`, "dept-1").
		Scan(&deptName).Error)
	assert.Equal(t, "研发部", deptName, "sys_dept.dept_name 必须可读（同步路径 department 属性依赖）")

	var gotDeptID string
	require.NoError(t, db.Raw(`SELECT dept_id FROM sys_user WHERE id = ?`, "user-dept").
		Scan(&gotDeptID).Error)
	assert.Equal(t, "dept-1", gotDeptID, "seedDrainUser 的 deptID 必须真的落库（F-07/QUEUE-04 路由依赖）")
}
