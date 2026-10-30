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
	"context"
	"errors"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/glebarez/sqlite"
	"github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core/db/migrations"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
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

// seedDrainOUMapping 向 sys_dept_ou_mapping 插入一行（Plan 04 的 compare-and-move 用例铺底）。
//
// 唯一键是 (dept_id, ad_config_id)：同一 dept_id 在不同 ad_config_id 下可以有多行
// （F-B 的测试构造正是靠这一点）。syncEnabled 控制 sync_enabled 列。
func seedDrainOUMapping(t *testing.T, db *gorm.DB, id, deptID, adConfigID, ouDN string, syncEnabled bool) {
	t.Helper()
	now := drainNow()
	require.NoError(t, db.Exec(
		`INSERT INTO sys_dept_ou_mapping
		 (id, dept_id, ad_config_id, ou_dn, ou_name, sync_enabled, sync_status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 'synced', ?, ?)`,
		id, deptID, adConfigID, ouDN, ouDN, drainBoolToInt(syncEnabled), now, now,
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

// ==================== Phase 58 Plan 03 Task 1：退避纯函数 + 拾取谓词 ====================

// TestPhase58_BackoffDuration_Curve 退避曲线 60s→2m→4m→8m→16m（D-05）。
func TestPhase58_BackoffDuration_Curve(t *testing.T) {
	assert.Equal(t, 60*time.Second, backoffDuration(1), "第 1 次失败 → 60s")
	assert.Equal(t, 2*time.Minute, backoffDuration(2), "第 2 次失败 → 2min")
	assert.Equal(t, 4*time.Minute, backoffDuration(3), "第 3 次失败 → 4min")
	assert.Equal(t, 8*time.Minute, backoffDuration(4), "第 4 次失败 → 8min")
	assert.Equal(t, 16*time.Minute, backoffDuration(5), "第 5 次失败 → 16min")
}

// TestPhase58_BackoffDuration_CappedAt30m 封顶 30min（D-05）+ <=0 回落 60s。
func TestPhase58_BackoffDuration_CappedAt30m(t *testing.T) {
	assert.Equal(t, 30*time.Minute, backoffDuration(6), "第 6 次失败 → 封顶 30min")
	assert.Equal(t, 30*time.Minute, backoffDuration(99), "超上限仍封顶 30min")
	assert.Equal(t, 60*time.Second, backoffDuration(0), "attempts<=0 → 60s")
}

// TestPhase58_PickRows_NullNextRetryIsPickable D-05：next_retry_at IS NULL = 立即可拾取。
func TestPhase58_PickRows_NullNextRetryIsPickable(t *testing.T) {
	db := setupDrainTestDB(t)
	for i, id := range []string{"u-1", "u-2", "u-3"} {
		seedDrainQueueRow(t, db, "", id, string(models.ADPushActionUpdateUser),
			string(models.ADPushStatusPending), 0, nil)
		_ = i
	}
	d := NewADPushDrainer(db, newDrainTestPool())
	rows, err := d.pickRows(context.Background(), time.Now().UTC())
	require.NoError(t, err)
	assert.Len(t, rows, 3, "next_retry_at IS NULL 的行必须全部可拾取（反向语义会制造僵尸行）")
}

// TestPhase58_PickRows_FutureNextRetrySkipped 退避未到期的行不被拾取。
func TestPhase58_PickRows_FutureNextRetrySkipped(t *testing.T) {
	db := setupDrainTestDB(t)
	future := time.Now().UTC().Add(time.Hour)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser),
		string(models.ADPushStatusFailed), 1, &future)
	d := NewADPushDrainer(db, newDrainTestPool())
	rows, err := d.pickRows(context.Background(), time.Now().UTC())
	require.NoError(t, err)
	assert.Empty(t, rows, "next_retry_at 未到期的行不得被拾取")
}

// TestPhase58_PickRows_PastNextRetryPicked failed 且退避已到期的行可被拾取。
func TestPhase58_PickRows_PastNextRetryPicked(t *testing.T) {
	db := setupDrainTestDB(t)
	past := time.Now().UTC().Add(-time.Minute)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser),
		string(models.ADPushStatusFailed), 1, &past)
	d := NewADPushDrainer(db, newDrainTestPool())
	rows, err := d.pickRows(context.Background(), time.Now().UTC())
	require.NoError(t, err)
	assert.Len(t, rows, 1, "退避已到期的 failed 行必须可被拾取")
}

// TestPhase58_PickRows_Limit100 单轮批量上限 100（D-05）。
func TestPhase58_PickRows_Limit100(t *testing.T) {
	db := setupDrainTestDB(t)
	for i := 0; i < 120; i++ {
		seedDrainQueueRow(t, db, "", "u-"+string(rune('a'+i%26))+string(rune('a'+(i/26))),
			string(models.ADPushActionUpdateUser), string(models.ADPushStatusPending), 0, nil)
	}
	d := NewADPushDrainer(db, newDrainTestPool())
	rows, err := d.pickRows(context.Background(), time.Now().UTC())
	require.NoError(t, err)
	assert.Len(t, rows, 100, "单轮上限 100（drainBatchLimit）")
}

// TestPhase58_PickRows_OrderedByCreatedAt 按 created_at ASC 返回（F-01 压制后的入队先后）。
func TestPhase58_PickRows_OrderedByCreatedAt(t *testing.T) {
	db := setupDrainTestDB(t)
	r1 := seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser),
		string(models.ADPushStatusPending), 0, nil)
	r2 := seedDrainQueueRow(t, db, "", "u-2", string(models.ADPushActionUpdateUser),
		string(models.ADPushStatusPending), 0, nil)
	r3 := seedDrainQueueRow(t, db, "", "u-3", string(models.ADPushActionUpdateUser),
		string(models.ADPushStatusPending), 0, nil)
	// 显式把 created_at 拉开（GORM Create 会在同一秒内落库，顺序不可辨）。
	require.NoError(t, db.Exec(`UPDATE ad_push_queue SET created_at = ? WHERE id = ?`, "2026-01-01 00:00:03", r3.ID).Error)
	require.NoError(t, db.Exec(`UPDATE ad_push_queue SET created_at = ? WHERE id = ?`, "2026-01-01 00:00:01", r1.ID).Error)
	require.NoError(t, db.Exec(`UPDATE ad_push_queue SET created_at = ? WHERE id = ?`, "2026-01-01 00:00:02", r2.ID).Error)

	d := NewADPushDrainer(db, newDrainTestPool())
	rows, err := d.pickRows(context.Background(), time.Now().UTC())
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Equal(t, r1.ID, rows[0].ID, "created_at ASC 第一个")
	assert.Equal(t, r2.ID, rows[1].ID)
	assert.Equal(t, r3.ID, rows[2].ID)
}

// TestPhase58_PickRows_TerminalRowsNotPicked 终态行（pushed/dead）退出活跃谓词，不再重试（T-58-03-07）。
func TestPhase58_PickRows_TerminalRowsNotPicked(t *testing.T) {
	db := setupDrainTestDB(t)
	seedDrainQueueRow(t, db, "", "u-pushed", string(models.ADPushActionUpdateUser),
		string(models.ADPushStatusPushed), 0, nil)
	seedDrainQueueRow(t, db, "", "u-dead", string(models.ADPushActionUpdateUser),
		string(models.ADPushStatusDead), 0, nil)
	d := NewADPushDrainer(db, newDrainTestPool())
	rows, err := d.pickRows(context.Background(), time.Now().UTC())
	require.NoError(t, err)
	assert.Empty(t, rows, "pushed / dead 终态行不得被拾取")
}

// TestPhase58_Drain_NoConfigDeferWithoutDead Q3(i)：全局零启用配置 → 整轮 defer，不动任何行。
// ⚠ 与"单用户匹配不到配置"（转 dead）必须区分 —— 混淆会误伤已入队的推送。
func TestPhase58_Drain_NoConfigDeferWithoutDead(t *testing.T) {
	db := setupDrainTestDB(t)
	// 刻意不 seed 任何 sys_ad_config 行（或只 seed 停用的）。
	seedDrainADConfig(t, db, "cfg-disabled", "停用域", "DC=corp,DC=com", true, int(models.ADConfigStatusDisabled))
	seedDrainUser(t, db, "u-1", "alice", "CN=alice,DC=corp,DC=com", "DC=corp,DC=com", nil)
	row := seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser),
		string(models.ADPushStatusPending), 0, nil)

	// 本用例在分组阶段即 defer，不会触达 LDAP；直接构造即可（Task 2 才引入
	// newDrainTestDrainer 这一注入 helper）。
	d := NewADPushDrainer(db, newDrainTestPool(drainTestAccount("a1", "cfg-disabled")))
	require.NoError(t, d.DrainOnce(context.Background()))

	var got models.ADPushQueue
	require.NoError(t, db.Where("id = ?", row.ID).First(&got).Error)
	assert.Equal(t, models.ADPushStatusPending, got.Status, "零启用配置时行状态必须原样保留（defer 而非 dead）")

	var deadCount int64
	require.NoError(t, db.Model(&models.ADPushQueue{}).Where("status = ?", string(models.ADPushStatusDead)).Count(&deadCount).Error)
	assert.Equal(t, int64(0), deadCount, "defer 分支不得把任何行打成 dead")
}

// ==================== Phase 58 Plan 03 Task 2：drainRow 状态跃迁 ====================

// newDrainTestDrainer 返回已注入 mock LDAP 与假账号池的排水器（所有 DrainOnce 用例的统一入口）。
//
// ⚠ 缺 clientFactoryOverride 注入 → executeGroup 内 NewFailoverClient 的 clientFactory 为 nil
// → newClient 真拨号 → 用例落 MarkFailure 必红（且会污染同包其它用例的账号池状态）。
// Plan 04 的 DrainOnce 用例也必须走这个 helper，不得自行 NewADPushDrainer 后忘记注入。
func newDrainTestDrainer(t *testing.T, db *gorm.DB, mock *mockLDAPClient, accounts ...models.ADServiceAccount) *ADPushDrainer {
	t.Helper()
	d := NewADPushDrainer(db, newDrainTestPool(accounts...))
	d.clientFactoryOverride = func(*models.ADConfig, *models.ADServiceAccount) LDAPClientIface { return mock }
	return d
}

// reloadDrainRow 读回队列行（终态断言用）。
func reloadDrainRow(t *testing.T, db *gorm.DB, entityID string) models.ADPushQueue {
	t.Helper()
	var row models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ?", entityID).First(&row).Error)
	return row
}

// setDrainUserAttrs 给已 seed 的用户补 nickname/email/phone/dept_id（update_user 路径需要
// 至少一个属性非空，否则 syncUserAttributes 会在 len(attributes)==0 处提前返回、不触达 LDAP）。
func setDrainUserAttrs(t *testing.T, db *gorm.DB, userID, nickname, email, phone, deptID string) {
	t.Helper()
	require.NoError(t, db.Exec(
		`UPDATE sys_user SET nickname = ?, email = ?, phone = ?, dept_id = ? WHERE id = ?`,
		nickname, email, phone, deptID, userID).Error)
}

// TestPhase58_Drain_DisableSuccessWritesPushedAt disable 动作真实执行 DisableUser 并转 pushed。
func TestPhase58_Drain_DisableSuccessWritesPushedAt(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainUser(t, db, "u-1", "alice", "CN=alice,OU=研发部,DC=corp,DC=com", "OU=研发部,DC=corp,DC=com", nil)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionDisable), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	assert.Equal(t, 1, mock.disableUserCalls, "disable 动作必须真实调用 DisableUser（UAC 514）")
	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusPushed, row.Status)
	assert.NotNil(t, row.PushedAt, "成功终态必须写 pushed_at")
	assert.Nil(t, row.NextRetryAt, "成功终态必须清空 next_retry_at")
}

// TestPhase58_Drain_DisableSoftDeletedUserStillDrains P2 红线：读 sys_user 必须 Unscoped。
// 软删用户对应的 disable 行仍必须能推出去（未 Unscoped 时本用例必红）。
func TestPhase58_Drain_DisableSoftDeletedUserStillDrains(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainUser(t, db, "u-1", "alice", "CN=alice,DC=corp,DC=com", "DC=corp,DC=com", nil)
	require.NoError(t, db.Exec(`UPDATE sys_user SET deleted_at = ? WHERE id = ?`, drainNow(), "u-1").Error)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionDisable), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	assert.Equal(t, 1, mock.disableUserCalls, "软删用户的 disable 行仍必须推到 AD")
	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusPushed, row.Status)
}

// TestPhase58_Drain_OrphanDisableRowGoesDead F-07：entity_id 指向不存在用户 → dead，且不烧 attempts。
func TestPhase58_Drain_OrphanDisableRowGoesDead(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainQueueRow(t, db, "", "ghost-user", string(models.ADPushActionDisable), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	row := reloadDrainRow(t, db, "ghost-user")
	assert.Equal(t, models.ADPushStatusDead, row.Status)
	assert.Equal(t, 0, row.Attempts, "不可恢复分支不得烧 attempts")
	assert.Contains(t, row.LastError, "unrecoverable:user-not-found")
	assert.Equal(t, 0, mock.disableUserCalls)
}

// TestPhase58_Drain_UpdateUserSuccessWritesPushedAt update_user 属性路径真实执行。
//
// ⚠ Phase 58 Plan 04 起 update_user 路径叠加了 compare-and-move：用户有 dept_id 时
// 必须有对应的 OU 映射行，否则按 Q2 决策转 dead（no-ou-mapping）而非 pushed。
// 本用例的语义是"属性路径成功 → pushed"，故铺一行与用户当前 ad_ou_dn 归一化相等的
// 映射（目标一致 → 跳过 MoveUser，落 markPushed），断言强度不变。
func TestPhase58_Drain_UpdateUserSuccessWritesPushedAt(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainDept(t, db, "dept-1", "研发部")
	seedDrainUser(t, db, "u-1", "alice", "CN=alice,OU=研发部,DC=corp,DC=com", "OU=研发部,DC=corp,DC=com", nil)
	setDrainUserAttrs(t, db, "u-1", "Alice", "alice@corp.com", "13800000000", "dept-1")
	seedDrainOUMapping(t, db, "m-1", "dept-1", cfgID, "OU=研发部,DC=corp,DC=com", true)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	assert.Equal(t, 1, mock.updateUserAttrCalls, "属性推送必须真实调用 UpdateUserAttribute")
	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusPushed, row.Status)
	assert.NotNil(t, row.PushedAt)
}

// TestPhase58_Drain_FailureSchedulesBackoff 可恢复失败 → failed + attempts+1 + 退避 60s。
func TestPhase58_Drain_FailureSchedulesBackoff(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainUser(t, db, "u-1", "alice", "CN=alice,DC=corp,DC=com", "DC=corp,DC=com", nil)
	setDrainUserAttrs(t, db, "u-1", "Alice", "", "", "")
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	mock.updateUserAttrErr = errors.New("boom")
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))

	before := time.Now().UTC()
	require.NoError(t, d.DrainOnce(context.Background()))

	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusFailed, row.Status)
	assert.Equal(t, 1, row.Attempts)
	require.NotNil(t, row.NextRetryAt)
	backoff := row.NextRetryAt.Sub(before)
	assert.GreaterOrEqual(t, backoff, 55*time.Second, "退避应约 60s（容差 ±5s）")
	assert.LessOrEqual(t, backoff, 65*time.Second, "退避应约 60s（容差 ±5s）")
	assert.NotNil(t, row.FailedAt)
	assert.True(t, len(row.LastError) >= len("retry:update_user:"), "last_error 应有前缀")
	assert.Equal(t, "retry:update_user:", row.LastError[:len("retry:update_user:")])
}

// TestPhase58_MarkFailed_SanitizesNulInLastError 缺陷X 回归（2026-10-21 UAT 实测）：
// LDAP Result Code 50 的 err.Error() 实测含 "\x00"（problem 4003, data 0\n\0）→
// markFailed/markDead 直写 last_error → PG 拒收 SQLSTATE 22021 → UPDATE 整体失败 →
// 行留 pending/attempts=0/next_retry_at=NULL 每 tick 无限重试。两入口必须先过
// sanitizeFailureReason（与账号池 MarkFailure 同款），保证 last_error 恒为合法 UTF-8
// 且不含 0x00。
func TestPhase58_MarkFailed_SanitizesNulInLastError(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainUser(t, db, "u-1", "alice", "CN=alice,DC=corp,DC=com", "DC=corp,DC=com", nil)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser), string(models.ADPushStatusPending), 0, nil)
	row := reloadDrainRow(t, db, "u-1")

	// 实测 LDAP 50 报文形态：含 NUL（data 0\n\0）与非法 UTF-8 序列
	dirty := "retry:update_user:LDAP Result Code 50 \"Insufficient Access Rights\": 00002098: DSID-031514A2, problem 4003, data 0\n\x00\x80\x01bad"
	require.Contains(t, dirty, "\x00", "构造的报文必须真的含 NUL")
	d := NewADPushDrainer(db, newDrainTestPool(drainTestAccount("a1", cfgID)))
	now := time.Now().UTC()

	// markFailed 入口（failed 分支）
	require.NoError(t, d.markFailed(context.Background(), row, now, false, dirty))
	got := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusFailed, got.Status)
	assert.NotContains(t, got.LastError, "\x00", "last_error 不得含 0x00（缺陷X：PG SQLSTATE 22021）")
	assert.True(t, utf8.ValidString(got.LastError), "last_error 必须是合法 UTF-8")
	assert.Contains(t, got.LastError, "LDAP Result Code 50", "清洗只去坏字节，保留可排障正文")

	// markDead 入口（终态分支）
	require.NoError(t, d.markDead(context.Background(), row.ID, now, 5, "dead:retry:"+dirty))
	got = reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusDead, got.Status)
	assert.NotContains(t, got.LastError, "\x00", "markDead 的 last_error 同样不得含 0x00")
	assert.True(t, utf8.ValidString(got.LastError), "markDead 的 last_error 必须是合法 UTF-8")
}

// TestPhase58_Drain_MaxAttemptsGoesDead F-11 主用例：连败 4 次后本轮再失败 → dead 且 attempts 必须落到 5。
//
// ⚠ 这条断言就是 markDead 必须带 attempts 入参的验收依据：若 markDead 只写 status/时间戳/reason
// 不写 attempts，DB 里仍是 4，本用例必红。
func TestPhase58_Drain_MaxAttemptsGoesDead(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainUser(t, db, "u-1", "alice", "CN=alice,DC=corp,DC=com", "DC=corp,DC=com", nil)
	setDrainUserAttrs(t, db, "u-1", "Alice", "", "", "")
	// 模拟已连败 4 次
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser), string(models.ADPushStatusFailed), 4, nil)

	mock := newDrainTestLDAP()
	mock.updateUserAttrErr = errors.New("boom")
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusDead, row.Status)
	assert.Equal(t, 5, row.Attempts, "达上限分支必须把 newAttempts 落到 DB")
	assert.Nil(t, row.NextRetryAt)
	assert.Equal(t, "dead:max-attempts", row.LastError)
}

// TestPhase58_Drain_NoBareAttemptsDeadAfterMerge F-11 禁令验收：attempts=5 的行本轮成功必须转
// pushed（而不是凭裸 attempts 判死）。
func TestPhase58_Drain_NoBareAttemptsDeadAfterMerge(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainUser(t, db, "u-1", "alice", "CN=alice,DC=corp,DC=com", "DC=corp,DC=com", nil)
	setDrainUserAttrs(t, db, "u-1", "Alice", "", "", "")
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser), string(models.ADPushStatusFailed), 5, nil)

	mock := newDrainTestLDAP() // 本轮无错误
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusPushed, row.Status, "attempts=5 的行若本轮成功必须转 pushed，不得凭裸 attempts 判死")
	assert.NotNil(t, row.PushedAt)
}

// TestPhase58_Drain_PoolExhaustedIsRetryable 账号池熔断是可重试失败，不判 dead（D-05）。
func TestPhase58_Drain_PoolExhaustedIsRetryable(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainUser(t, db, "u-1", "alice", "CN=alice,DC=corp,DC=com", "DC=corp,DC=com", nil)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionDisable), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock) // 无账号 → ErrAllAccountsUnavailable
	require.NoError(t, d.DrainOnce(context.Background()))

	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusFailed, row.Status, "池熔断必须可重试，不得转 dead")
	// ⚠ MISSED-02：池侧故障**不计入** attempts。原先此处断言 1，那编码的是缺陷 ——
	// 池侧故障会累积计数，把整批行推到"再有一次业务失败就判死"的悬崖边上。
	assert.Equal(t, 0, row.Attempts, "池侧故障不得计入 attempts（基础设施故障不是业务失败）")
	assert.True(t, row.NextRetryAt != nil, "必须排退避")
	assert.True(t, len(row.LastError) >= len("pool:"))
	assert.Equal(t, "pool:", row.LastError[:len("pool:")])
}

// TestPhase58_Drain_PoolFailureDoesNotArmTerminalState MISSED-02 回归：
// 连续多轮**池侧故障**后 attempts 必须原地不动；随后第一次**业务**失败才 +1，
// 且不应因之前的池侧故障而直接判死。
//
// 这条锁的是 H-04 与 MISSED-02 的**配对**关系：单独修 H-04 会得到
// "不判死但持续计数"的半修状态 —— 一次约 2h 的 AD 断连把 attempts 推到 12+，
// 之后任何一次业务失败立刻 dead，而 last_error 已被覆盖成 "dead:max-attempts"，
// 运维看不出前 12 次不是业务失败。
func TestPhase58_Drain_PoolFailureDoesNotArmTerminalState(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainUser(t, db, "u-1", "alice", "CN=alice,DC=corp,DC=com", "DC=corp,DC=com", nil)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionDisable), string(models.ADPushStatusPending), 0, nil)

	// --- 第一阶段：连续 12 轮池侧故障（模拟长时间 AD 断连）---
	for round := 1; round <= 12; round++ {
		d := newDrainTestDrainer(t, db, newDrainTestLDAP()) // 无账号 → ErrAllAccountsUnavailable
		require.NoError(t, d.DrainOnce(context.Background()))
		// 抹掉退避，使下一轮立即可拾取（生产靠时间推进，此处压缩时间轴）
		require.NoError(t, db.Exec(
			`UPDATE ad_push_queue SET next_retry_at = NULL WHERE entity_id = ?`, "u-1").Error)

		row := reloadDrainRow(t, db, "u-1")
		require.Equal(t, models.ADPushStatusFailed, row.Status, "第 %d 轮池侧故障后不得判死", round)
		require.Equal(t, 0, row.Attempts,
			"第 %d 轮池侧故障后 attempts 必须仍为 0（基础设施故障不计入业务失败计数）", round)
	}

	// --- 第二阶段：一次业务失败，attempts 才 +1，且不判死 ---
	mock := newDrainTestLDAP()
	mock.disableUserErr = errors.New("业务失败：AD 返回非池侧错误")
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusFailed, row.Status,
		"12 轮池侧故障后的首次业务失败不得直接判死（attempts 从 0 起算）")
	assert.Equal(t, 1, row.Attempts, "首次业务失败才把 attempts 推到 1")
	assert.NotContains(t, row.LastError, "pool:", "last_error 应反映本次真实业务失败原因")
}

// TestPhase58_Failover_AllDialFailureWrapsPoolSentinel H-04：所有账号**拨号**失败是池侧
// 可恢复故障，返回值必须满足 errors.Is(err, ErrAllAccountsDialFailed)。
//
// ⚠ 同时锁定它**不**满足 ErrAllAccountsUnavailable：两者对管理员的处置动作不同
//   （加账号/查熔断 vs 查网络与域控），Phase 78 三条既有用例正锁定这一区分。
//
// 反向半边同样重要：账号拨通了、只是业务操作全失败时**不得**带任何池侧哨兵 ——
// code 32 之类必须原样上抛到 handlePushError 的不可恢复分支。
func TestPhase58_Failover_AllDialFailureWrapsPoolSentinel(t *testing.T) {
	accounts := []models.ADServiceAccount{
		{ID: "a1", Username: "svc-1"},
		{ID: "a2", Username: "svc-2"},
	}
	cfg := &models.ADConfig{BaseModel: models.BaseModel{ID: "cfg-h04"}}

	// --- 全部拨号失败 → 包 ErrAllAccountsDialFailed ---
	pool := &fakePool78{listAvailableRes: accounts}
	fc := NewFailoverClient(pool, cfg)
	fc.clientFactory = func(*models.ADConfig, *models.ADServiceAccount) LDAPClientIface {
		return &mockLDAPClient{connectErr: errors.New("dial tcp 10.0.0.1:389: connect: connection refused")}
	}
	err := fc.ExecuteWithFailover(context.Background(), func(LDAPClientIface) error {
		t.Fatal("一个账号都没拨通时 operation 不得被执行")
		return nil
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAllAccountsDialFailed),
		"H-04：全部拨号失败必须被识别为池侧故障，否则排水器会烧 attempts 并在 ~15min 后把整批打成 dead 终态")
	assert.False(t, errors.Is(err, ErrAllAccountsUnavailable),
		"池非空时不得冒充 ErrAllAccountsUnavailable（Phase 78 契约：处置动作不同）")
	assert.Equal(t, 2, len(pool.markFailureCalls), "每个拨号失败的账号都要上报 MarkFailure")
	for _, call := range pool.markFailureCalls {
		assert.True(t, len(call.reason) >= len("dial:"), "拨号失败上报的 reason 必须保留 dial: 前缀")
	}

	// --- 拨通但业务操作全失败 → 不带任何池侧哨兵 ---
	pool2 := &fakePool78{listAvailableRes: accounts}
	fc2 := NewFailoverClient(pool2, cfg)
	fc2.clientFactory = func(*models.ADConfig, *models.ADServiceAccount) LDAPClientIface {
		return &mockLDAPClient{}
	}
	code32 := ldap.NewError(ldap.LDAPResultNoSuchObject, errors.New("no such object"))
	err = fc2.ExecuteWithFailover(context.Background(), func(LDAPClientIface) error { return code32 })
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrAllAccountsUnavailable),
		"业务失败不是池耗尽：带哨兵会把 code 32 误判成可重试、跳过不可恢复分支")
	assert.False(t, errors.Is(err, ErrAllAccountsDialFailed),
		"业务失败不是拨号失败：否则 code 32 会被误判为池侧故障而绕过 handlePushError")
	assert.True(t, ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject),
		"业务失败的原始语义必须能被 errors.As/IsErrorWithCode 穿透")
}

// TestPhase58_Drain_AllDialFailureNeverKillsBatch H-04 端到端：域控全不可达时，
// 即使行已连败到 maxDrainAttempts-1，也**不得**转 dead —— 必须 failed + 排退避，
// 且仍在活跃谓词内可被下轮拾取。
//
// ⚠ 这是本条修复的核心可观测后果：修复前这条用例会得到 status=dead，
//   一批行被永久终结（dead 行不被任何后续编辑复活）。
func TestPhase58_Drain_AllDialFailureNeverKillsBatch(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainUser(t, db, "u-1", "alice", "CN=alice,DC=corp,DC=com", "DC=corp,DC=com", nil)
	// 连败到上限前一格：本轮再失败即达 maxDrainAttempts
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionDisable),
		string(models.ADPushStatusFailed), 4, nil)

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	d.clientFactoryOverride = func(*models.ADConfig, *models.ADServiceAccount) LDAPClientIface {
		return &mockLDAPClient{dnExistsRes: true, connectErr: errors.New("connection refused")}
	}
	require.NoError(t, d.DrainOnce(context.Background()))

	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusFailed, row.Status,
		"全部账号拨号失败是可恢复故障：本轮失败数已达上限也**不得**转 dead（修复前本断言必红）")
	assert.Nil(t, row.PushedAt)
	require.NotNil(t, row.NextRetryAt, "退避必须排上，等 AD 恢复后再拾取")
	require.True(t, len(row.LastError) >= len("pool:"))
	assert.Equal(t, "pool:", row.LastError[:len("pool:")], "last_error 前缀沿用 pool:，排障口径不变")

	// 仍可拾取：退避到期后行必须重新出现在活跃谓词里（未退出 = 未被终态吃掉）
	rows, err := d.pickRows(context.Background(), d.now().Add(time.Hour))
	require.NoError(t, err)
	assert.Len(t, rows, 1, "池侧故障后的行必须在退避到期后重新可拾取，否则整批会被悄悄饿死")
}

// TestPhase58_Drain_Code32GoesDeadAndClearsAdDn QUEUE-05：code 32 → dead + 清 ad_dn + 不烧 attempts。
func TestPhase58_Drain_Code32GoesDeadAndClearsAdDn(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainUser(t, db, "u-1", "alice", "CN=alice,DC=corp,DC=com", "DC=corp,DC=com", nil)
	setDrainUserAttrs(t, db, "u-1", "Alice", "", "", "")
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	mock.updateUserAttrErr = ldap.NewError(ldap.LDAPResultNoSuchObject, errors.New("no such object"))
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusDead, row.Status)
	assert.Equal(t, 0, row.Attempts, "code 32 是不可恢复错误，不烧 attempts")
	assert.Equal(t, "unrecoverable:target-not-found", row.LastError)

	var adDn, adOuDn, adSyncedAt *string
	require.NoError(t, db.Raw(`SELECT ad_dn, ad_ou_dn, ad_synced_at FROM sys_user WHERE id = ?`, "u-1").
		Row().Scan(&adDn, &adOuDn, &adSyncedAt))
	assert.Nil(t, adDn, "code 32 必须清空 sys_user.ad_dn")
	assert.Nil(t, adOuDn)
	assert.Nil(t, adSyncedAt)
}

// TestPhase58_Drain_DNExistsFalseIsDeadNotPushed DNExists=false 时 syncUserAttributes 清 ad_dn 后
// 返回 nil，必须重新判空 → dead（锁住 drainRow 里 reload 判空那段逻辑）。
func TestPhase58_Drain_DNExistsFalseIsDeadNotPushed(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainUser(t, db, "u-1", "alice", "CN=alice,DC=corp,DC=com", "DC=corp,DC=com", nil)
	setDrainUserAttrs(t, db, "u-1", "Alice", "", "", "")
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	mock.dnExistsRes = false // AD 端不存在
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusDead, row.Status, "AD 端不存在 → dead，不得误报 pushed")
	var adDn *string
	require.NoError(t, db.Raw(`SELECT ad_dn FROM sys_user WHERE id = ?`, "u-1").Row().Scan(&adDn))
	assert.Nil(t, adDn, "DNExists=false 会清空 ad_dn")
}

// TestPhase58_Drain_WritesUTCTimestamps F-12：注入固定 UTC 时刻，断言 pushed_at 的 Location 与值。
func TestPhase58_Drain_WritesUTCTimestamps(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainUser(t, db, "u-1", "alice", "CN=alice,DC=corp,DC=com", "DC=corp,DC=com", nil)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionDisable), string(models.ADPushStatusPending), 0, nil)

	fixed := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	d.now = func() time.Time { return fixed }
	require.NoError(t, d.DrainOnce(context.Background()))

	row := reloadDrainRow(t, db, "u-1")
	require.NotNil(t, row.PushedAt)
	assert.Equal(t, time.UTC, row.PushedAt.Location(), "pushed_at 的 Location 必须是 UTC")
	assert.True(t, row.PushedAt.Equal(fixed), "pushed_at 必须等于注入的固定 UTC 时刻")
}

// TestPhase58_Drain_MutualExclusionSkipsSecondRound mu.TryLock 语义：**同一个实例**上
// 上一轮未结束时本轮立即跳过（一次 LDAP 写都不发生）。
//
// H-01/H-02：本用例此前只证明"单实例上 TryLock 有效"，却被写成 SC1 互斥性的验收证据；
// 而生产中每轮新建实例 → 每轮一把新锁。修 H-01 后（排水器在 Start() 里只构造一次、
// 闭包捕获同一实例），"同一实例"才是生产的真实形态，断言才成立。
//
// 第三段是反向记录：另一个实例**不受**这把锁阻塞 —— 诚实的单实例语义，
// 不是"跨实例互斥"的主张（生产不存在第二个调用方；cron 轮次防重叠由
// cron.SkipIfStillRunning 承担）。
func TestPhase58_Drain_MutualExclusionSkipsSecondRound(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainUser(t, db, "u-1", "alice", "CN=alice,DC=corp,DC=com", "DC=corp,DC=com", nil)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionDisable), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))

	// --- 1) 同一实例持锁 → 本轮必须被跳过，一次 LDAP 写都不发生 ---
	d.mu.Lock() // 手动持锁模拟"上一轮未结束"
	err := d.DrainOnce(context.Background())
	d.mu.Unlock()

	require.NoError(t, err, "持锁期间 DrainOnce 必须立即返回 nil（跳过本轮）")
	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusPending, row.Status, "被跳过的轮次不得改动任何行")
	assert.Equal(t, 0, mock.disableUserCalls, "同一实例被锁期间不得发出任何 LDAP 写（mock 计数必须不前进）")
	assert.Equal(t, 0, mock.dnExistsCalls, "同一实例被锁期间不得触达 LDAP 任何接口")

	// --- 2) 同一实例放锁后仍能正常排水（证明上面是"跳过"而非"卡死"） ---
	require.NoError(t, d.DrainOnce(context.Background()))
	assert.Equal(t, 1, mock.disableUserCalls, "放锁后同一实例必须正常执行一轮")
	assert.Equal(t, models.ADPushStatusPushed, reloadDrainRow(t, db, "u-1").Status)

	// --- 3) 反向记录：另一个实例不被这把锁阻塞（单实例语义，非跨实例互斥） ---
	// 新用户 u-2 再次被编辑 → 新入队一行（u-1 已是 pushed 终态，不会被再拾取）
	seedDrainUser(t, db, "u-2", "bob", "CN=bob,DC=corp,DC=com", "DC=corp,DC=com", nil)
	seedDrainQueueRow(t, db, "", "u-2", string(models.ADPushActionDisable), string(models.ADPushStatusPending), 0, nil)

	mock2 := newDrainTestLDAP()
	other := newDrainTestDrainer(t, db, mock2, drainTestAccount("a1", cfgID))
	d.mu.Lock() // 第一个实例仍持锁
	err = other.DrainOnce(context.Background())
	d.mu.Unlock()

	require.NoError(t, err)
	assert.Equal(t, 1, mock2.disableUserCalls,
		"不同实例各持一把锁 → 互斥不跨实例；生产靠 cron 侧单实例复用 + SkipIfStillRunning")
	assert.Equal(t, models.ADPushStatusPushed, reloadDrainRow(t, db, "u-2").Status)
}

// ==================== Phase 58 Plan 04 Task 2：compare-and-move OU 移动面 ====================

// drainUserADRefs 读回 sys_user 的 ad_dn / ad_ou_dn（可空列）。
func drainUserADRefs(t *testing.T, db *gorm.DB, userID string) (adDn, adOuDn *string) {
	t.Helper()
	require.NoError(t, db.Raw(`SELECT ad_dn, ad_ou_dn FROM sys_user WHERE id = ?`, userID).
		Row().Scan(&adDn, &adOuDn))
	return adDn, adOuDn
}

// seedCompareMoveUser 铺一个 compare-and-move 用例的标准用户：
// dept_id = dept-it，ad_dn / ad_ou_dn 均落在 cfg-1 的 base_dn 下，nickname 非空
// （保证 syncUserAttributes 真的触达 LDAP）。
func seedCompareMoveUser(t *testing.T, db *gorm.DB, userID, adOuDn, deptID string) {
	t.Helper()
	seedDrainUser(t, db, userID, "zhangsan",
		"CN=zhangsan,OU=IT,DC=corp,DC=com", adOuDn, &deptID)
	setDrainUserAttrs(t, db, userID, "张三", "", "", deptID)
}

// TestPhase58_Drain_SkipsMoveWhenOUMatches QUEUE-04 幂等点：目标 OU 与当前 ad_ou_dn
// 归一化相等时**不调** MoveUser。
func TestPhase58_Drain_SkipsMoveWhenOUMatches(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedCompareMoveUser(t, db, "u-1", "OU=IT,DC=corp,DC=com", "dept-it")
	seedDrainOUMapping(t, db, "m-1", "dept-it", cfgID, "OU=IT,DC=corp,DC=com", true)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	assert.Equal(t, 0, mock.moveUserCalls, "目标 OU 与当前 ad_ou_dn 归一化相等 → 必须跳过 MoveUser（否则对已搬移源 DN 再 ModifyDN 会 code 32）")
	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusPushed, row.Status)
	assert.NotNil(t, row.PushedAt)
}

// TestPhase58_Drain_NormalizedComparisonSkipsMove 大小写/空格差异也必须跳过。
//
// 这组数据同时覆盖 **attr 名大小写**（OU vs ou）与 **属性值大小写**（IT vs it）
// 两类差异 —— AD DN 两者皆大小写不敏感，NormalizeDN 的契约是"attr 名与值统一 ToLower"。
// 未经 NormalizeDN 的裸比较在本用例下必红。
func TestPhase58_Drain_NormalizedComparisonSkipsMove(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	// 用户侧：attr 名小写 + 值小写 + 分隔逗号前后带空格
	seedCompareMoveUser(t, db, "u-1", "ou=it , dc=corp , dc=com", "dept-it")
	// 映射侧：attr 名大写 + 值大写
	seedDrainOUMapping(t, db, "m-1", "dept-it", cfgID, "OU=IT,DC=corp,DC=com", true)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	assert.Equal(t, 0, mock.moveUserCalls, "attr 名/值大小写与空格差异归一化后相等 → 必须跳过 MoveUser")
	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusPushed, row.Status)
}

// TestPhase58_Drain_MoveUpdatesAdDnAndAdOuDn F-A 阻断级：MoveUser 成功后必须**同时**
// 回写 ad_dn（RDN 保留 + 新 OU 父路径）与 ad_ou_dn。
func TestPhase58_Drain_MoveUpdatesAdDnAndAdOuDn(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedCompareMoveUser(t, db, "u-1", "OU=IT,DC=corp,DC=com", "dept-it2")
	seedDrainOUMapping(t, db, "m-1", "dept-it2", cfgID, "OU=IT2,DC=corp,DC=com", true)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	require.Equal(t, 1, mock.moveUserCalls, "目标 OU 不一致 → 必须移动一次")
	require.Len(t, mock.moveUserTargets, 1)
	assert.Equal(t, "OU=IT2,DC=corp,DC=com", mock.moveUserTargets[0])

	adDn, adOuDn := drainUserADRefs(t, db, "u-1")
	require.NotNil(t, adDn, "F-A：MoveUser 成功后必须回写 ad_dn（只回写 ad_ou_dn 会让下一轮按陈旧 DN 解析配置）")
	assert.Equal(t, "CN=zhangsan,OU=IT2,DC=corp,DC=com", *adDn, "新 ad_dn = 原 RDN + 新 OU 父路径")
	require.NotNil(t, adOuDn)
	assert.Equal(t, "OU=IT2,DC=corp,DC=com", *adOuDn)

	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusPushed, row.Status)
}

// TestPhase58_Drain_RepeatDrainIsIdempotent QUEUE-04 主用例（两轮）：同一行重复排水
// 第二轮必须因 ad_ou_dn 已回写而跳过 MoveUser。
func TestPhase58_Drain_RepeatDrainIsIdempotent(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedCompareMoveUser(t, db, "u-1", "OU=IT,DC=corp,DC=com", "dept-it2")
	seedDrainOUMapping(t, db, "m-1", "dept-it2", cfgID, "OU=IT2,DC=corp,DC=com", true)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))
	require.Equal(t, 1, mock.moveUserCalls, "第一轮必须移动一次")

	// 模拟"同一行被重复排水"：把行打回 pending（终态行退出活跃谓词，不会被再拾取）。
	require.NoError(t, db.Exec(
		`UPDATE ad_push_queue SET status = 'pending', next_retry_at = NULL, pushed_at = NULL WHERE entity_id = ?`,
		"u-1").Error)

	require.NoError(t, d.DrainOnce(context.Background()))

	assert.Equal(t, 1, mock.moveUserCalls, "第二轮 ad_ou_dn 已回写为目标 OU → 归一化相等 → MoveUser 不得再被调用（幂等）")
	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusPushed, row.Status)
}

// TestPhase58_Drain_UsesLatestDBValues 闭包内重读：入队后 DB 又改过的值必须被采纳
// （禁新旧值比较 —— 推的是最新值，不是入队快照）。
func TestPhase58_Drain_UsesLatestDBValues(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainDept(t, db, "dept-old", "旧部门")
	seedDrainDept(t, db, "dept-new", "新部门")
	seedCompareMoveUser(t, db, "u-1", "OU=IT,DC=corp,DC=com", "dept-old")
	seedDrainOUMapping(t, db, "m-old", "dept-old", cfgID, "OU=IT,DC=corp,DC=com", true)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser), string(models.ADPushStatusPending), 0, nil)

	// ← 入队之后才发生的改动：昵称与部门都变
	require.NoError(t, db.Exec(
		`UPDATE sys_user SET nickname = ?, dept_id = ? WHERE id = ?`,
		"最新昵称", "dept-new", "u-1").Error)
	seedDrainOUMapping(t, db, "m-new", "dept-new", cfgID, "OU=IT2,DC=corp,DC=com", true)

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	require.NotNil(t, mock.updateUserAttrLastAttrs, "属性推送必须真的触达 LDAP 才能断言最新值")
	assert.Equal(t, "最新昵称", mock.updateUserAttrLastAttrs["displayName"],
		"推送的必须是闭包内重读到的最新昵称，而非入队时刻的旧值")
	assert.Equal(t, "新部门", mock.updateUserAttrLastAttrs["department"],
		"department 属性同样取闭包内重读到的最新部门")
	require.Len(t, mock.moveUserTargets, 1)
	assert.Equal(t, "OU=IT2,DC=corp,DC=com", mock.moveUserTargets[0],
		"OU 移动目标也由最新的 dept_id 决定（不是入队快照）")
}

// TestPhase58_Drain_UsesConfigScopedOUMapping F-B：同 dept_id 在多个 AD 配置下有映射行时，
// 必须按本行解析出的配置取（禁跨域取首行）。
func TestPhase58_Drain_UsesConfigScopedOUMapping(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-a"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedCompareMoveUser(t, db, "u-1", "OU=OLD,DC=corp,DC=com", "dept-1")
	// 刻意让 cfg-b 的行 id 排序在前：不收敛 ad_config_id 的实现会取到它。
	seedDrainOUMapping(t, db, "m-aaa-cfgb", "dept-1", "cfg-b", "OU=IT,DC=na,DC=corp,DC=com", true)
	seedDrainOUMapping(t, db, "m-zzz-cfga", "dept-1", cfgID, "OU=IT,DC=corp,DC=com", true)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	require.Len(t, mock.moveUserTargets, 1)
	assert.Equal(t, "OU=IT,DC=corp,DC=com", mock.moveUserTargets[0],
		"必须取 cfg-a 的 OU（F-B）：取到 cfg-b 的那行等于把用户在 AD 树里跨域搬错位置")
}

// TestPhase58_Drain_NoOUMappingGoesDead Q2：目标 OU 映射缺失 → dead 且**不是 pushed**。
// 这条断言就是「终态被 markPushed 覆盖」的验收依据。
func TestPhase58_Drain_NoOUMappingGoesDead(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedCompareMoveUser(t, db, "u-1", "OU=IT,DC=corp,DC=com", "dept-orphan") // 无对应映射行
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusDead, row.Status,
		"无 OU 映射必须转 dead；若被 markPushed 覆盖成 pushed，本断言必红（生产上误报成功 + 运营动作无人触发）")
	assert.Equal(t, 0, row.Attempts, "不可恢复错误不得烧 attempts")
	assert.Equal(t, "unrecoverable:no-ou-mapping", row.LastError)
	assert.Nil(t, row.PushedAt, "dead 终态不得写 pushed_at")
	assert.Equal(t, 0, mock.moveUserCalls)
}

// TestPhase58_Drain_NoDeptSkipsMove 无部门不是失败也不是终态：跳过移动照常 pushed。
func TestPhase58_Drain_NoDeptSkipsMove(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainUser(t, db, "u-1", "zhangsan", "CN=zhangsan,OU=IT,DC=corp,DC=com", "OU=IT,DC=corp,DC=com", nil)
	require.NoError(t, db.Exec(`UPDATE sys_user SET nickname = ? WHERE id = ?`, "张三", "u-1").Error)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	assert.Equal(t, 0, mock.moveUserCalls, "无部门 → 无 OU 概念 → 不得移动")
	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusPushed, row.Status, "无部门不是失败，必须照常 pushed")
}

// TestPhase58_Drain_MoveCode32GoesDead QUEUE-05 第二条路径：OU 移动 code 32 →
// dead + 清 ad_dn/ad_ou_dn + 不烧 attempts。
func TestPhase58_Drain_MoveCode32GoesDead(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedCompareMoveUser(t, db, "u-1", "OU=IT,DC=corp,DC=com", "dept-it2")
	seedDrainOUMapping(t, db, "m-1", "dept-it2", cfgID, "OU=IT2,DC=corp,DC=com", true)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	mock.moveUserErr = ldap.NewError(ldap.LDAPResultNoSuchObject, errors.New("no such object"))
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusDead, row.Status, "OU 路径 code 32 与属性路径同款终态（dead）")
	assert.Equal(t, 0, row.Attempts, "code 32 不可恢复，不烧 attempts")
	assert.Equal(t, "unrecoverable:target-not-found", row.LastError)

	adDn, adOuDn := drainUserADRefs(t, db, "u-1")
	assert.Nil(t, adDn, "code 32 必须清空 sys_user.ad_dn（clearUserADRefs 生效）")
	assert.Nil(t, adOuDn)
}

// TestPhase58_Drain_MoveFailureIsRetryable 有意收紧：属性成功但 OU 移动失败 → 行转 failed
// （**不是** pushed），退避重试，last_error 前缀 retry:move_ou:。
func TestPhase58_Drain_MoveFailureIsRetryable(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedCompareMoveUser(t, db, "u-1", "OU=IT,DC=corp,DC=com", "dept-it2")
	seedDrainOUMapping(t, db, "m-1", "dept-it2", cfgID, "OU=IT2,DC=corp,DC=com", true)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	mock.moveUserErr = errors.New("boom")
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusFailed, row.Status,
		"OU 移动失败不得被属性成功掩盖成 pushed（相对 SyncUserUpdateToAD 的有意收紧）")
	assert.Equal(t, 1, row.Attempts)
	assert.NotNil(t, row.NextRetryAt, "可重试失败必须排退避")
	assert.Nil(t, row.PushedAt)
	require.True(t, len(row.LastError) >= len("retry:move_ou:"))
	assert.Equal(t, "retry:move_ou:", row.LastError[:len("retry:move_ou:")])
}

// injectSysUserUpdateFailure 让**针对 sys_user 的 UPDATE 在发出 SQL 之前**失败。
//
// 用 GORM 的 Before("gorm:update") 回调 + AddError：processor 逐个回调前检查
// tx.Error，故 SQL 根本不会执行 —— 与真实 DB 故障（连接池耗尽 / 语句超时）的
// 可观测效果一致（值没写进去 + 返回 error）。若改用 After 回调，SQL 已经落库，
// 用例就变成了"假失败"。
//
// 只拦 sys_user：队列表（ad_push_queue）的终态写入不受影响，否则连 markFailed
// 都写不进去，用例就测不出状态机了。
func injectSysUserUpdateFailure(t *testing.T, db *gorm.DB) {
	t.Helper()
	const cbName = "test_inject_sys_user_update_failure"
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register(cbName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "sys_user" {
			tx.AddError(errors.New("injected: sys_user write failure"))
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Update().Remove(cbName) })
}

// TestPhase58_Drain_MoveWriteBackFailureIsNotPushed H-05：MoveUser 已成功（AD 侧对象
// 搬到新 OU），但紧随其后的 ad_dn / ad_ou_dn 回写失败时，行**不得**被写成 pushed。
//
// 修复前该路径只 Warn 并返回 (false, nil) → 调用方 markPushed → 队列以"成功"终结，
// 而本地 ad_dn 停在旧 OU；此后该用户任何一次编辑都会走 DNExists(旧 DN)=false →
// 清空三列 → dead "unrecoverable:target-not-found"，AD 账号还在但系统永久丢失绑定。
//
// 修复后：failed + attempts+1 + 排退避，last_error 以 retry:move_ou: 开头，暴露真实原因。
func TestPhase58_Drain_MoveWriteBackFailureIsNotPushed(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedCompareMoveUser(t, db, "u-1", "OU=IT,DC=corp,DC=com", "dept-it2")
	seedDrainOUMapping(t, db, "m-1", "dept-it2", cfgID, "OU=IT2,DC=corp,DC=com", true)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionUpdateUser), string(models.ADPushStatusPending), 0, nil)

	injectSysUserUpdateFailure(t, db) // 只拦 ad_dn/ad_ou_dn 回写

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	require.Equal(t, 1, mock.moveUserCalls, "前置条件：MoveUser 必须已成功（AD 侧对象已搬走）")

	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusFailed, row.Status,
		"回写失败不得写成 pushed：pushed 是终态、退出活跃谓词，'下轮会重算'不成立")
	assert.NotEqual(t, models.ADPushStatusPushed, row.Status)
	assert.Equal(t, 1, row.Attempts)
	require.NotNil(t, row.NextRetryAt, "可重试失败必须排退避")
	assert.Nil(t, row.PushedAt)
	require.True(t, len(row.LastError) >= len("retry:move_ou:"))
	assert.Equal(t, "retry:move_ou:", row.LastError[:len("retry:move_ou:")],
		"last_error 必须暴露真实原因（回写失败），而不是被 markPushed 掩盖成成功")

	// 回写确实没发生：本地 ad_dn 仍是旧 OU（这正是 H-05 描述的陈旧绑定）
	adDn, adOuDn := drainUserADRefs(t, db, "u-1")
	require.NotNil(t, adDn)
	assert.Equal(t, "CN=zhangsan,OU=IT,DC=corp,DC=com", *adDn,
		"注入的回写失败必须真的没落库，否则本用例的前提不成立")
	require.NotNil(t, adOuDn)
	assert.Equal(t, "OU=IT,DC=corp,DC=com", *adOuDn)
}

// ==================== Phase 58-05：D-05 单旋钮读取（方案A热重载：每 tick 调用） ====================

// freshDrainSysConfigDB 构建仅含 sys_config 的内存 sqlite（LoadDrainIntervalSeconds 的
// 最小环境）。AutoMigrate 走 models.Config（与生产同构的列定义）。
func freshDrainSysConfigDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Config{}))
	return db
}

// TestPhase58_LoadDrainIntervalSeconds_FallsBackOnGarbage 锁死"启动读旋钮 + 垃圾值兜底"：
// 无行 / 非数字 / 非正数 三种情形一律回落 30；合法行按字面返回。
func TestPhase58_LoadDrainIntervalSeconds_FallsBackOnGarbage(t *testing.T) {
	const key = drainIntervalSysConfigKey

	t.Run("no-row", func(t *testing.T) {
		db := freshDrainSysConfigDB(t)
		assert.Equal(t, 30, LoadDrainIntervalSeconds(db))
	})

	t.Run("non-numeric", func(t *testing.T) {
		db := freshDrainSysConfigDB(t)
		require.NoError(t, db.Exec(
			`INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system) VALUES (?, ?, 'abc', 'N', 0)`,
			"x", key).Error)
		assert.Equal(t, 30, LoadDrainIntervalSeconds(db))
	})

	t.Run("zero", func(t *testing.T) {
		db := freshDrainSysConfigDB(t)
		require.NoError(t, db.Exec(
			`INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system) VALUES (?, ?, '0', 'N', 0)`,
			"x", key).Error)
		assert.Equal(t, 30, LoadDrainIntervalSeconds(db))
	})

	t.Run("negative", func(t *testing.T) {
		db := freshDrainSysConfigDB(t)
		require.NoError(t, db.Exec(
			`INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system) VALUES (?, ?, '-5', 'N', 0)`,
			"x", key).Error)
		assert.Equal(t, 30, LoadDrainIntervalSeconds(db))
	})

	t.Run("valid-45", func(t *testing.T) {
		db := freshDrainSysConfigDB(t)
		require.NoError(t, db.Exec(
			`INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system) VALUES (?, ?, '45', 'N', 0)`,
			"x", key).Error)
		assert.Equal(t, 45, LoadDrainIntervalSeconds(db))
	})
}

// ==================== Plan 06（T1）：EnqueueUpdateBatch 批量入队变体（D-01） ====================

// enqueueBatchSeedTwoUsers 铺「一条启用的 AD 配置 + 两个用户」，返回两个 userID。
// 导入场景只有已同步到 AD 的存量用户才有推送意义，故 ad_dn 取非空串。
func enqueueBatchSeedTwoUsers(t *testing.T, db *gorm.DB, cfgID string) (string, string) {
	t.Helper()
	seedDrainADConfig(t, db, cfgID, "cfg-"+cfgID, "DC=corp,DC=com", true, 0)
	u1 := "p06-u1-" + cfgID
	u2 := "p06-u2-" + cfgID
	seedDrainUser(t, db, u1, "p06un1-"+cfgID, "CN=p06un1,OU=Users,DC=corp,DC=com", "OU=Users,DC=corp,DC=com", nil)
	seedDrainUser(t, db, u2, "p06un2-"+cfgID, "CN=p06un2,OU=Users,DC=corp,DC=com", "OU=Users,DC=corp,DC=com", nil)
	return u1, u2
}

// enqueueBatchSeedTwoUsersNoConfig 只铺用户、不铺任何 AD 配置行。
//
// ⚠ 单独抽出是因为 enqueueBatchSeedTwoUsers 无条件铺一条「启用」配置，
// 用它会与本组用例「无启用配置」的前提直接矛盾（IsADSyncEnabled 会命中那条配置）。
func enqueueBatchSeedTwoUsersNoConfig(t *testing.T, db *gorm.DB) (string, string) {
	t.Helper()
	u1 := "p06-nocfg-u1"
	u2 := "p06-nocfg-u2"
	seedDrainUser(t, db, u1, "p06nocfg1", "CN=a,OU=Users,DC=corp,DC=com", "OU=Users,DC=corp,DC=com", nil)
	seedDrainUser(t, db, u2, "p06nocfg2", "CN=b,OU=Users,DC=corp,DC=com", "OU=Users,DC=corp,DC=com", nil)
	return u1, u2
}

// countUpdateUserRows 统计指定实体集合中的 update_user 行数。
func countUpdateUserRows(t *testing.T, db *gorm.DB, entityIDs ...string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&models.ADPushQueue{}).
		Where("entity_id IN ?", entityIDs).
		Where("action_type = ?", string(models.ADPushActionUpdateUser)).
		Count(&n).Error)
	return n
}

// TestPhase58_EnqueueUpdateBatch_DedupesAndFilters 锁定 D-01 批量变体的两条前置语义：
// 空切片 no-op + 重复 id 去重（PG 对同一条 INSERT 内重复命中的行 DO UPDATE 会报
// "cannot affect row a second time"，去重是批量入队的硬前提）。
func TestPhase58_EnqueueUpdateBatch_DedupesAndFilters(t *testing.T) {
	db := setupDrainTestDB(t)
	u1, u2 := enqueueBatchSeedTwoUsers(t, db, "p06-dedup")
	enq := NewADPushEnqueuer()

	// 1) 空切片 no-op
	require.NoError(t, enq.EnqueueUpdateBatch(db, nil))
	require.NoError(t, enq.EnqueueUpdateBatch(db, []string{}))
	assert.Equal(t, int64(0), countUpdateUserRows(t, db, u1, u2), "空切片不得入队")

	// 2) 重复 id 去重：u1 传两次只应产生一行
	require.NoError(t, enq.EnqueueUpdateBatch(db, []string{u1, u1, u2}))
	assert.Equal(t, int64(2), countUpdateUserRows(t, db, u1, u2), "重复 id 必须去重")

	// 3) 每行的三列语义
	var rows []models.ADPushQueue
	require.NoError(t, db.Where("entity_id IN ?", []string{u1, u2}).Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		assert.Equal(t, models.ADPushEntityUser, r.EntityType)
		assert.Equal(t, models.ADPushActionUpdateUser, r.ActionType)
		assert.Equal(t, models.ADPushStatusPending, r.Status)
	}

	// 4) 全空串列表（DedupeEntityIDs 过滤空串后为空）同样 no-op
	require.NoError(t, enq.EnqueueUpdateBatch(db, []string{"", ""}))
	assert.Equal(t, int64(2), countUpdateUserRows(t, db, u1, u2), "全空串列表不得入队")
}

// TestPhase58_EnqueueUpdateBatch_NoADConfigSkips 锁定 D-02 第一层：没有启用的 AD 配置时
// 放行（返回 nil 不报错），入队数为 0 —— 导入结果不受 AD 开关影响。
func TestPhase58_EnqueueUpdateBatch_NoADConfigSkips(t *testing.T) {
	db := setupDrainTestDB(t)
	// ⚠ 故意 seed 一条 sync_enabled=false 的配置而不是「完全不 seed」：
	//   完全无行时「无启用配置」是因为表空，本用例想验的是「有配置但未启用」这一档。
	seedDrainADConfig(t, db, "p06-cfg-off", "cfg-off", "DC=corp,DC=com", false, 0)
	u1, u2 := enqueueBatchSeedTwoUsersNoConfig(t, db)

	enq := NewADPushEnqueuer()
	require.NoError(t, enq.EnqueueUpdateBatch(db, []string{u1, u2}),
		"无启用配置必须放行（不阻断导入），不能返回 error")
	assert.Equal(t, int64(0), countUpdateUserRows(t, db, u1, u2), "无启用配置不得入队")
}

// TestPhase58_EnqueueUpdateBatch_MergeOnSecondCall 锁定 D-06 合并语义：
// 连续两次同 ids 入队，行数恒为 2（合并命中部分唯一索引，而非追加并行行）。
func TestPhase58_EnqueueUpdateBatch_MergeOnSecondCall(t *testing.T) {
	db := setupDrainTestDB(t)
	u1, u2 := enqueueBatchSeedTwoUsers(t, db, "p06-merge")
	enq := NewADPushEnqueuer()

	require.NoError(t, enq.EnqueueUpdateBatch(db, []string{u1, u2}))
	require.Equal(t, int64(2), countUpdateUserRows(t, db, u1, u2))

	require.NoError(t, enq.EnqueueUpdateBatch(db, []string{u1, u2}))
	assert.Equal(t, int64(2), countUpdateUserRows(t, db, u1, u2), "第二次入队必须合并（同实体恒一行活跃）")

	// 把两行打回 failed 再入队：合并子句应把它们打回 pending 并清零 attempts
	// （这才是"合并"而非"幂等 no-op"的判别式 —— 若实现跳过已存在行，failed 就回不来了）。
	require.NoError(t, db.Model(&models.ADPushQueue{}).
		Where("entity_id IN ?", []string{u1, u2}).
		Where("action_type = ?", string(models.ADPushActionUpdateUser)).
		Updates(map[string]interface{}{
			"status":     string(models.ADPushStatusFailed),
			"attempts":   3,
			"last_error": "prev failure",
		}).Error)
	require.NoError(t, enq.EnqueueUpdateBatch(db, []string{u1, u2}))

	var rows []models.ADPushQueue
	require.NoError(t, db.Where("entity_id IN ?", []string{u1, u2}).Find(&rows).Error)
	require.Len(t, rows, 2, "failed 行被合并而非新增并行行")
	for _, r := range rows {
		assert.Equal(t, models.ADPushStatusPending, r.Status, "合并必须把 failed 打回 pending")
		assert.Equal(t, 0, r.Attempts, "合并必须清零 attempts（F-11 归零点）")
	}
}

// TestPhase58_EnqueueUpdateBatch_NoDiffFiltering 锁定本变体与 EnqueueUpdateIfChanged 的
// 关键差异：**不做 diff**。用户属性没有任何实际变更，仍必须入队 —— 导入场景没有
// 旧值快照，全量入队才是正确语义（排水器闭包重读 DB 后推最新值）。
//
// 反向对照让本用例可证伪：同一组"同值"参数喂给 diff 变体必须判为无变更、不入队。
func TestPhase58_EnqueueUpdateBatch_NoDiffFiltering(t *testing.T) {
	db := setupDrainTestDB(t)
	u1, u2 := enqueueBatchSeedTwoUsers(t, db, "p06-nodiff")
	enq := NewADPushEnqueuer()

	setDrainUserAttrs(t, db, u1, "same-nick", "same@mail", "13800000000", "")
	var current models.User
	require.NoError(t, db.Where("id = ?", u1).First(&current).Error)
	require.NotNil(t, current.Nickname)
	require.NotNil(t, current.Email)
	require.NotNil(t, current.Phone)
	sameReq := &requests.UserUpdateRequest{
		Nickname: current.Nickname,
		Email:    current.Email,
		Phone:    current.Phone,
	}

	// 对照组：diff 变体在同值请求下判为无变更 → 不入队
	require.NoError(t, enq.EnqueueUpdateIfChanged(db, &current, sameReq))
	assert.Equal(t, int64(0), countUpdateUserRows(t, db, u1, u2),
		"对照组：diff 变体在无实际变更时不得入队")

	// 主断言：批量变体在完全相同的"无变更"前提下仍入队
	require.NoError(t, enq.EnqueueUpdateBatch(db, []string{u1, u2}))
	assert.Equal(t, int64(2), countUpdateUserRows(t, db, u1, u2),
		"无实际变更也必须入队（本变体不做 diff）")
}

// ==================== Plan 07（T1）：SC1/2/5 端到端收尾用例 ====================
//
// 下面三个用例把 QUEUE-03/04/05 串成一条完整链路：
// 入队（生产 EnqueueUpdateBatch）→ 排水（生产 DrainOnce）→ compare-and-move
// （生产 compareAndMoveOU）→ 终态（生产 markPushed/markDead）。
//
// ⚠ 与 Plan 03/04/06 的既有单腿用例的分工：那些用例各自锁定一条腿（拾取谓词、
// 退避曲线、目标一致跳过 MoveUser、单动作终态…），用 seedDrainQueueRow **直接铺行**，
// 不经入队器；本节三例是唯一从**入队器**出发、走完整条生产路径的用例。
// 因此本节刻意不重复断言单腿细节（那是上��用例的职责），只断言"链路串得通 +
// 跨腿的复合不变量（重复排水幂等 / code 32 后零 AD 写）"。

// e2eSetupCompareMoveEnv 铺端到端用例的标准环境：一个用户（当前在 OU=IT）、
// 一个目标部门（映射到 OU=IT2）、一条启用配置。返回 userID 与固定时刻。
//
// ⚠ nickname 必须非空（经 setDrainUserAttrs 写入）：syncUserAttributes 在
// len(attributes)==0 时提前 return，根本不触达 LDAP，updateUserAttrCalls 断言会假红。
// ⚠ 必须 seedDrainDept：department 属性走 sys_dept 查名，缺行时该属性被静默省略。
func e2eSetupCompareMoveEnv(t *testing.T, db *gorm.DB, cfgID, userID string) {
	t.Helper()
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainDept(t, db, "dept-it2", "IT2部")
	seedDrainUser(t, db, userID, "zhangsan",
		"CN=zhangsan,OU=IT,DC=corp,DC=com", "OU=IT,DC=corp,DC=com", ptrStr("dept-it2"))
	setDrainUserAttrs(t, db, userID, "张三", "zhangsan@corp.com", "13800000000", "dept-it2")
	seedDrainOUMapping(t, db, "m-1", "dept-it2", cfgID, "OU=IT2,DC=corp,DC=com", true)
}

// ptrStr 返回指向字面量 str 的指针（seed helper 的 deptID 入参需要 *string）。
// 单独抽出来是因为内联 &"dept-it2" 在 Go 中不合法。
func ptrStr(s string) *string { return &s }

// countQueueRowsForEntity 统计某实体的队列行总数（含终态行）。
//
// ⚠ 与 countUpdateUserRows 的区别：后者带 action_type 过滤。端到端幂等用例需要断言
// "**同一行**被重复排水"，而 reloadDrainRow 用的是 First（无 ORDER BY）—— 一旦因为
// 误用再入队而产生并行行，First 会静默返回其中一行，让断言变成随机通过/随机失败。
// 先用本 helper 把行数钉死，才让"同一行"这个前提可证伪。
func countQueueRowsForEntity(t *testing.T, db *gorm.DB, entityID string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&models.ADPushQueue{}).Where("entity_id = ?", entityID).Count(&n).Error)
	return n
}

// TestPhase58_EndToEnd_UpdateUserPushedAfterDrain 端到端主用例：
// 真实入队 → 排水 → 属性 Modify → compare-and-move OU → pushed 终态。
//
// 断言全部走**生产符号产生的可观测后果**（mock 计数 / sys_user 回写 / 队列行状态），
// 不做裸 SQL 断言状态之外的实现细节。
func TestPhase58_EndToEnd_UpdateUserPushedAfterDrain(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	const userID = "u-1"
	e2eSetupCompareMoveEnv(t, db, cfgID, userID)

	// --- 入队：走生产入队器（非 seedDrainQueueRow）---
	enq := NewADPushEnqueuer()
	require.NoError(t, enq.EnqueueUpdateBatch(db, []string{userID}))
	assert.Equal(t, int64(1), countUpdateUserRows(t, db, userID), "入队必须产生恰好一行 update_user")
	row0 := reloadDrainRow(t, db, userID)
	assert.Equal(t, models.ADPushStatusPending, row0.Status, "入队后初始状态必须是 pending")

	// --- 排水 ---
	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	// --- 属性 Modify 恰一次 ---
	assert.Equal(t, 1, mock.updateUserAttrCalls, "端到端必须真实推送一次属性")
	require.NotNil(t, mock.updateUserAttrLastAttrs)
	assert.Equal(t, "张三", mock.updateUserAttrLastAttrs["displayName"],
		"推的必须是闭包内重读到的最新昵称")
	assert.Equal(t, "IT2部", mock.updateUserAttrLastAttrs["department"],
		"department 属性由重读到的最新 dept_id 决定")

	// --- compare-and-move：当前 OU=IT ≠ 目标 OU=IT2 → 移动一次 ---
	require.Equal(t, 1, mock.moveUserCalls, "目标 OU 不一致 → 必须移动一次")
	require.Len(t, mock.moveUserTargets, 1)
	assert.Equal(t, "OU=IT2,DC=corp,DC=com", mock.moveUserTargets[0])

	// --- F-A：ad_dn / ad_ou_dn 双写 ---
	adDn, adOuDn := drainUserADRefs(t, db, userID)
	require.NotNil(t, adDn)
	assert.Equal(t, "CN=zhangsan,OU=IT2,DC=corp,DC=com", *adDn)
	require.NotNil(t, adOuDn)
	assert.Equal(t, "OU=IT2,DC=corp,DC=com", *adOuDn)

	// --- 终态 ---
	row := reloadDrainRow(t, db, userID)
	assert.Equal(t, models.ADPushStatusPushed, row.Status)
	assert.NotNil(t, row.PushedAt, "成功终态必须写 pushed_at（可查询确认，QUEUE-03）")
	assert.Nil(t, row.NextRetryAt)
}

// TestPhase58_EndToEnd_RepeatDrainIdempotent QUEUE-04 端到端幂等：
// 把同一行打回 pending 跑第二轮 —— MoveUser **不得**再被调用（compare-and-move
// 的幂等点跨腿成立：入队 → 移动 → 回写 → 再排水），但属性 Modify 仍每轮推最新值
// （Modify 本身幂等，且"推最新值"是 QUEUE-04 的核心承诺）。
//
// ⚠ 与 TestPhase58_Drain_RepeatDrainIsIdempotent 的分工：那条用例只断言
// moveUserCalls 不再增长；本条额外断言 updateUserAttrCalls **确实**增长到 2 ——
// 否则"第二轮什么都没干"也会让前者变绿（假幂等）。本条额外走**生产入队器**起步，
// 而那条直接 seedDrainQueueRow 铺行。
func TestPhase58_EndToEnd_RepeatDrainIdempotent(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	const userID = "u-1"
	e2eSetupCompareMoveEnv(t, db, cfgID, userID)

	enq := NewADPushEnqueuer()
	require.NoError(t, enq.EnqueueUpdateBatch(db, []string{userID}))

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))
	require.Equal(t, 1, mock.moveUserCalls, "第一轮必须移动一次")
	require.Equal(t, 1, mock.updateUserAttrCalls, "第一轮必须推一次属性")

	// 模拟"同一队列行被重复排水"：把行打回 pending。
	//
	// ⚠ 这里刻意**不用**再入队（EnqueueUpdateBatch 第二次）来制造第二轮 ——
	// 那不是"重复排水同一行"：pushed 行已退出 uq_ad_push_queue_active 部分唯一索引
	// （谓词 status IN ('pending','failed')），第二次 INSERT 不冲突，会**新增并行行**。
	// 那是 Phase 57 D-06 的正确设计（终态行必须退出索引，否则同实体的下一次编辑会撞
	// 42P10），但它会让本用例的"同一行"前提失效，且 reloadDrainRow 的 First 在两行间不确定。
	require.NoError(t, db.Exec(
		`UPDATE ad_push_queue SET status = 'pending', next_retry_at = NULL, pushed_at = NULL
		 WHERE entity_id = ?`, userID).Error)
	row := reloadDrainRow(t, db, userID)
	require.Equal(t, models.ADPushStatusPending, row.Status)
	require.Equal(t, int64(1), countQueueRowsForEntity(t, db, userID), "本用例前提是同一行被重复排水")

	require.NoError(t, d.DrainOnce(context.Background()))

	assert.Equal(t, 1, mock.moveUserCalls,
		"第二轮 ad_ou_dn 已回写为目标 OU → 归一化相等 → MoveUser 不得再被调用（端到端幂等）")
	assert.Equal(t, 2, mock.updateUserAttrCalls,
		"属性 Modify 每轮都推最新值（幂等写，仍必须真推）")

	adDn, adOuDn := drainUserADRefs(t, db, userID)
	require.NotNil(t, adDn)
	assert.Equal(t, "CN=zhangsan,OU=IT2,DC=corp,DC=com", *adDn, "第二轮不得改动 ad_dn")
	require.NotNil(t, adOuDn)
	assert.Equal(t, "OU=IT2,DC=corp,DC=com", *adOuDn, "第二轮不得改动 ad_ou_dn")
	row = reloadDrainRow(t, db, userID)
	assert.Equal(t, models.ADPushStatusPushed, row.Status)
}

// TestPhase58_EndToEnd_Code32ThenNoRetry QUEUE-05 端到端终态：
// 属性路径 code 32 → dead + 清空 sys_user.ad_dn（跨腿：排水器 + syncUserAttributes）。
//
// ⚠ 第二轮的断言口径（**不是**"不再被拾取"）：把行改回 pending 后，第二轮**会**被
// pickRows 拾取 —— 但 groupByResolvedConfig 读到用户 AdDn 已空 → 归入 deadRows 走
// no-ad-dn 分支 → 再转 dead，全程**零 AD 写**。故断言是
// 「被拾取但零 AD 写 + 终态仍为 dead」，写成"不再被拾取"会与分组逻辑的真实设计相反，
// 执行者可能为迎合断言而改坏 groupByResolvedConfig。
//
// ⚠ 更强的断言是 updateUserAttrCalls 停在 1：若第二轮真的去 AD 试写，必增。
func TestPhase58_EndToEnd_Code32ThenNoRetry(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	const userID = "u-1"
	e2eSetupCompareMoveEnv(t, db, cfgID, userID)

	enq := NewADPushEnqueuer()
	require.NoError(t, enq.EnqueueUpdateBatch(db, []string{userID}))

	mock := newDrainTestLDAP()
	mock.updateUserAttrErr = ldap.NewError(ldap.LDAPResultNoSuchObject, errors.New("no such object"))
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	// --- 第一轮：code 32 → dead ---
	row := reloadDrainRow(t, db, userID)
	assert.Equal(t, models.ADPushStatusDead, row.Status)
	assert.Equal(t, 0, row.Attempts, "不可恢复错误不烧 attempts")
	assert.Equal(t, "unrecoverable:target-not-found", row.LastError)
	assert.Nil(t, row.PushedAt, "dead 终态不得写 pushed_at")

	adDn, adOuDn := drainUserADRefs(t, db, userID)
	assert.Nil(t, adDn, "code 32 必须清空 sys_user.ad_dn（QUEUE-05 核心）")
	assert.Nil(t, adOuDn)
	assert.Equal(t, 0, mock.moveUserCalls, "属性路径就 code 32 了，OU 移动根本没走到")
	dnExistsAfterFirst := mock.dnExistsCalls

	// --- 第二轮：模拟异常回滚（行被打回 pending），断言零 AD 写 ---
	require.NoError(t, db.Model(&models.ADPushQueue{}).
		Where("entity_id = ?", userID).
		Updates(map[string]interface{}{"status": string(models.ADPushStatusPending)}).Error)

	require.NoError(t, d.DrainOnce(context.Background()))

	assert.Equal(t, 1, mock.updateUserAttrCalls,
		"code 32 之后不得再对 AD 发起写（ad_dn 已清空 → 分组阶段即判 no-ad-dn）")
	assert.Equal(t, 0, mock.moveUserCalls)
	assert.Equal(t, dnExistsAfterFirst, mock.dnExistsCalls,
		"第二轮不得再调 DNExists（同样在分组阶段即短路）")

	row = reloadDrainRow(t, db, userID)
	assert.Equal(t, models.ADPushStatusDead, row.Status, "第二轮结束后仍是 dead 终态")
	assert.Equal(t, "unrecoverable:no-ad-dn", row.LastError,
		"第二轮走的是分组阶段的 no-ad-dn 分支（与第一轮的 target-not-found 不同源，排障 SQL 靠前缀区分）")
}

// TestPhase58_EnqueueUpdateBatch_PropagatesError 锁定 D-02 第二层：队列表真写失败时
// 原样上抛 error，**禁止**吞错降级 —— 调用方据此决定回滚或降级。
func TestPhase58_EnqueueUpdateBatch_PropagatesError(t *testing.T) {
	db := setupDrainTestDB(t)
	u1, u2 := enqueueBatchSeedTwoUsers(t, db, "p06-err")
	enq := NewADPushEnqueuer()

	require.NoError(t, db.Exec("DROP TABLE ad_push_queue").Error)
	err := enq.EnqueueUpdateBatch(db, []string{u1, u2})
	require.Error(t, err, "队列表不可写时必须返回 error，不得静默降级")

	// 末尾重建队列表（与既有 RollsBack 用例同款收尾）
	require.NoError(t, migrations.Migrate215CreateADPushQueue(db))
	assert.Zero(t, countUpdateUserRows(t, db, u1, u2), "写失败不得留下任何半截行")
}
