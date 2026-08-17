package addomain

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupManagerSyncDB 构造内存 sqlite，建 SyncManagersToAD 依赖的三张表。
// 参考 dept_ou_mapper_test.go 的风格：直接 db.Exec CREATE TABLE，避免 AutoMigrate 外键问题。
func setupManagerSyncDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	// 三表均含 deleted_at：ADConfig/User 嵌入 BaseModel，GORM First/Find 会自动追加
	// `deleted_at IS NULL` 软删除过滤，缺列会导致 no such column。
	db.Exec(`CREATE TABLE sys_ad_config (id TEXT, sync_enabled INTEGER, status INTEGER, deleted_at DATETIME)`)
	db.Exec(`CREATE TABLE sys_dept (id TEXT, dept_name TEXT, dept_code TEXT, parent_id TEXT, ancestors TEXT, leader TEXT, status INTEGER, deleted_at DATETIME)`)
	db.Exec(`CREATE TABLE sys_user (id TEXT, username TEXT, dept_id TEXT, ad_dn TEXT, ad_synced_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`)
	return db
}

func insertDeptRow(db *gorm.DB, id, parentID, ancestors, leader string) {
	db.Exec(`INSERT INTO sys_dept (id, parent_id, ancestors, leader) VALUES (?, ?, ?, ?)`, id, parentID, ancestors, leader)
}

// insertUserRow 插入用户；deptID/adDN 为空串时写 NULL（leader 用户常无部门）。
func insertUserRow(db *gorm.DB, id, username, deptID, adDN string) {
	if deptID == "" && adDN == "" {
		db.Exec(`INSERT INTO sys_user (id, username) VALUES (?, ?)`, id, username)
	} else if deptID == "" {
		db.Exec(`INSERT INTO sys_user (id, username, ad_dn) VALUES (?, ?, ?)`, id, username, adDN)
	} else {
		db.Exec(`INSERT INTO sys_user (id, username, dept_id, ad_dn) VALUES (?, ?, ?, ?)`, id, username, deptID, adDN)
	}
}

func insertEnabledADConfig(db *gorm.DB) {
	db.Exec(`INSERT INTO sys_ad_config (id, sync_enabled, status) VALUES ('cfg-1', 1, 0)`)
}

func strPtr(s string) *string { return &s }

// deptWith 构造带 ID/Ancestors/Leader 的 Department（用于直接测 resolveLeaderByAncestors）。
func deptWith(id, ancestors string, leader *string) *models.Department {
	return &models.Department{
		BaseModel: models.BaseModel{ID: id},
		Ancestors: ancestors,
		Leader:    leader,
	}
}

// ============ splitAncestorIDs ============

func TestSplitAncestorIDs(t *testing.T) {
	// 标准 "0,root,parent" + self=parent → 仅 root（过滤 0 和 self）
	assert.Equal(t, []string{"root-id"}, splitAncestorIDs("0,root-id,parent-id", "parent-id"))

	// 空字符串
	assert.Nil(t, splitAncestorIDs("", "x"))

	// 多个 0 + 空值 → 全过滤
	assert.Equal(t, []string{"a", "b"}, splitAncestorIDs("0,,a,0,b", "self"))

	// 仅 0 根占位符
	assert.Nil(t, splitAncestorIDs("0", "self"))

	// 空格容错
	assert.Equal(t, []string{"a"}, splitAncestorIDs(" 0 , a ", "self"))
}

// ============ resolveLeaderByAncestors ============

func TestResolveLeaderByAncestors_CurrentDeptHasLeader(t *testing.T) {
	db := setupManagerSyncDB(t)
	svc := NewUserADSyncService(db, nil, nil, nil)

	// 当前部门有 leader → 直接返回，不查 DB
	dept := deptWith("dept-1", "0", strPtr("leader-1"))
	leaderID, err := svc.resolveLeaderByAncestors(context.Background(), dept)
	assert.NoError(t, err)
	assert.Equal(t, "leader-1", leaderID)
}

func TestResolveLeaderByAncestors_ParentFallback(t *testing.T) {
	db := setupManagerSyncDB(t)
	// 祖先链：root(无leader) → parent(有leader)。当前 dept-1 无 leader。
	insertDeptRow(db, "root-id", "", "0", "")
	insertDeptRow(db, "parent-id", "root-id", "0,root-id", "parent-leader")

	svc := NewUserADSyncService(db, nil, nil, nil)
	dept := deptWith("dept-1", "0,root-id,parent-id", nil)

	leaderID, err := svc.resolveLeaderByAncestors(context.Background(), dept)
	assert.NoError(t, err)
	assert.Equal(t, "parent-leader", leaderID, "应从最近的父部门取 leader")
}

func TestResolveLeaderByAncestors_NoLeaderAnywhere(t *testing.T) {
	db := setupManagerSyncDB(t)
	insertDeptRow(db, "root-id", "", "0", "")
	insertDeptRow(db, "parent-id", "root-id", "0,root-id", "")

	svc := NewUserADSyncService(db, nil, nil, nil)
	dept := deptWith("dept-1", "0,root-id,parent-id", nil)

	leaderID, err := svc.resolveLeaderByAncestors(context.Background(), dept)
	assert.NoError(t, err)
	assert.Empty(t, leaderID, "祖先链均无 leader → 返回空")
}

// ============ SyncManagersToAD ============

func TestSyncManagersToAD_BasicSync(t *testing.T) {
	db := setupManagerSyncDB(t)
	insertEnabledADConfig(db)
	insertDeptRow(db, "dept-1", "", "0", "leader-1")
	insertUserRow(db, "leader-1", "boss", "", "CN=boss,DC=x")   // leader 有 ad_dn
	insertUserRow(db, "user-1", "alice", "dept-1", "CN=alice,DC=x")

	var mu sync.Mutex
	var calls []string
	svc := NewUserADSyncService(db, nil, nil, nil)
	svc.updateUserAttributeFn = func(userDN string, attrs map[string]string) error {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, userDN+"|"+attrs["manager"])
		return nil
	}

	result, err := svc.SyncManagersToAD(context.Background(), nil)
	assert.NoError(t, err)
	// boss(leader) 有 ad_dn 故计入候选(Total=2)，但其无部门 → skipped；
	// alice 有部门且 leader=boss(有 ad_dn) → synced。验证正常同步路径。
	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 1, result.Synced)
	assert.Equal(t, 1, result.Skipped)
	assert.Equal(t, 0, result.Failed)
	assert.Len(t, calls, 1)
	assert.Equal(t, "CN=alice,DC=x|CN=boss,DC=x", calls[0])
}

func TestSyncManagersToAD_ParentLeaderFallback(t *testing.T) {
	db := setupManagerSyncDB(t)
	insertEnabledADConfig(db)
	// dept-1 无 leader，父 dept-root 有 leader
	insertDeptRow(db, "dept-root", "", "0", "leader-1")
	insertDeptRow(db, "dept-1", "dept-root", "0,dept-root", "")
	insertUserRow(db, "leader-1", "boss", "", "CN=boss,DC=x")
	insertUserRow(db, "user-1", "alice", "dept-1", "CN=alice,DC=x")

	var synced int
	var mu sync.Mutex
	svc := NewUserADSyncService(db, nil, nil, nil)
	svc.updateUserAttributeFn = func(userDN string, attrs map[string]string) error {
		mu.Lock()
		synced++
		mu.Unlock()
		assert.Equal(t, "CN=boss,DC=x", attrs["manager"], "应用父部门 leader 的 ad_dn")
		return nil
	}

	result, err := svc.SyncManagersToAD(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Synced)
	assert.Equal(t, 1, synced)
}

func TestSyncManagersToAD_SelfLeaderSkipped(t *testing.T) {
	db := setupManagerSyncDB(t)
	insertEnabledADConfig(db)
	// 部门 leader = 用户自己 → 跳过（保持原值）
	insertDeptRow(db, "dept-1", "", "0", "user-1")
	insertUserRow(db, "user-1", "alice", "dept-1", "CN=alice,DC=x")

	called := false
	svc := NewUserADSyncService(db, nil, nil, nil)
	svc.updateUserAttributeFn = func(userDN string, attrs map[string]string) error {
		called = true
		return nil
	}

	result, err := svc.SyncManagersToAD(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Skipped, "自指 leader 应跳过")
	assert.Equal(t, 0, result.Synced)
	assert.False(t, called, "自指时不应调用 updateAttr")
}

func TestSyncManagersToAD_LeaderWithoutAdDn(t *testing.T) {
	db := setupManagerSyncDB(t)
	insertEnabledADConfig(db)
	insertDeptRow(db, "dept-1", "", "0", "leader-1")
	insertUserRow(db, "leader-1", "boss", "", "") // leader 无 ad_dn
	insertUserRow(db, "user-1", "alice", "dept-1", "CN=alice,DC=x")

	svc := NewUserADSyncService(db, nil, nil, nil)
	svc.updateUserAttributeFn = func(userDN string, attrs map[string]string) error {
		t.Errorf("leader 无 ad_dn 时不应调用 updateAttr")
		return nil
	}

	result, err := svc.SyncManagersToAD(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Skipped, "leader 无 ad_dn 应跳过")
	assert.Equal(t, 0, result.Synced)
}

func TestSyncManagersToAD_UpdateFailsNotAbort(t *testing.T) {
	db := setupManagerSyncDB(t)
	insertEnabledADConfig(db)
	insertDeptRow(db, "dept-1", "", "0", "leader-1")
	insertUserRow(db, "leader-1", "boss", "", "CN=boss,DC=x")
	insertUserRow(db, "user-1", "alice", "dept-1", "CN=alice,DC=x")
	insertUserRow(db, "user-2", "bob", "dept-1", "CN=bob,DC=x")

	var callCount int
	var mu sync.Mutex
	svc := NewUserADSyncService(db, nil, nil, nil)
	svc.updateUserAttributeFn = func(userDN string, attrs map[string]string) error {
		mu.Lock()
		callCount++
		mu.Unlock()
		if userDN == "CN=alice,DC=x" {
			return errors.New("LDAP modify failed")
		}
		return nil
	}

	result, err := svc.SyncManagersToAD(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Synced, "bob 应成功")
	assert.Equal(t, 1, result.Failed, "alice 失败")
	assert.Equal(t, 2, callCount, "单失败不中断，两个都应调用")
}

func TestSyncManagersToAD_NoADConfig(t *testing.T) {
	db := setupManagerSyncDB(t)
	// 不插 ADConfig → 返回空 result，不报错
	insertUserRow(db, "user-1", "alice", "", "CN=alice,DC=x")

	svc := NewUserADSyncService(db, nil, nil, nil)
	result, err := svc.SyncManagersToAD(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.Total, "无 AD 配置应跳过同步")
}

func TestSyncManagersToAD_UserIDsFilter(t *testing.T) {
	db := setupManagerSyncDB(t)
	insertEnabledADConfig(db)
	insertDeptRow(db, "dept-1", "", "0", "leader-1")
	insertUserRow(db, "leader-1", "boss", "", "CN=boss,DC=x")
	insertUserRow(db, "user-1", "alice", "dept-1", "CN=alice,DC=x")
	insertUserRow(db, "user-2", "bob", "dept-1", "CN=bob,DC=x")

	var mu sync.Mutex
	var calls []string
	svc := NewUserADSyncService(db, nil, nil, nil)
	svc.updateUserAttributeFn = func(userDN string, attrs map[string]string) error {
		mu.Lock()
		calls = append(calls, userDN)
		mu.Unlock()
		return nil
	}

	// 只同步 user-1
	result, err := svc.SyncManagersToAD(context.Background(), []string{"user-1"})
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Total, "userIDs 过滤后只 1 个候选")
	assert.Equal(t, 1, result.Synced)
	assert.Len(t, calls, 1)
	assert.Equal(t, "CN=alice,DC=x", calls[0])
}

func TestSyncManagersToAD_UserWithoutDeptSkipped(t *testing.T) {
	db := setupManagerSyncDB(t)
	insertEnabledADConfig(db)
	// 用户有 ad_dn 但无部门 → 跳过
	insertUserRow(db, "user-1", "alice", "", "CN=alice,DC=x")

	svc := NewUserADSyncService(db, nil, nil, nil)
	svc.updateUserAttributeFn = func(userDN string, attrs map[string]string) error {
		t.Errorf("无部门用户不应同步")
		return nil
	}

	result, err := svc.SyncManagersToAD(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Skipped)
	assert.Equal(t, 0, result.Synced)
}
