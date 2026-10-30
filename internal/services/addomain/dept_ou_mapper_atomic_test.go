//go:build !skip_db_tests
// +build !skip_db_tests

package addomain

import (
	"context"
	"errors"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestUpsertMapping_AtomicDeleteInsert verifies P1-C3 fix:
// when the new mapping's INSERT fails mid-transaction, the prior
// mapping (which would normally be DELETEd first) must STILL EXIST.
//
// Pre-fix behavior: DELETE and CREATE were two independent transactions.
// A crash/network blip between them left the dept → OU mapping missing
// entirely, breaking user login flow for users in that dept.
//
// Post-fix behavior: both ops wrapped in single GORM Transaction;
// any failure rolls back the DELETE.
//
// We force the INSERT to fail by injecting a gorm.io/gorm hook that
// returns an error on the DeptOUMapping Create call inside the
// transaction. The hook is installed after the original insert and
// before the second upsert; the hook will see all Create calls until
// removed.
func TestUpsertMapping_AtomicDeleteInsert(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE sys_dept_ou_mapping (
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
			UNIQUE(dept_id, ad_config_id),
			UNIQUE(ad_config_id, ou_dn)
		);
	`).Error
	assert.NoError(t, err)

	mapper := NewDeptOUmapper(db)
	ctx := context.Background()

	// Seed an existing mapping
	original := &models.DeptOUMapping{
		ID:         "orig-1",
		DeptID:     "dept-original",
		ADConfigID: "config-1",
		OUDN:       "OU=Original,DC=corp,DC=local",
		OUName:     "OriginalDept",
	}
	err = mapper.UpsertMapping(ctx, original)
	assert.NoError(t, err)

	// Verify the original exists
	var count int64
	db.Model(&models.DeptOUMapping{}).Where("dept_id = ?", "dept-original").Count(&count)
	assert.Equal(t, int64(1), count, "original mapping should exist after first upsert")

	// Install a hook that will make any subsequent Create on
	// DeptOUMapping fail. This simulates a DB write failure that
	// occurs mid-UpsertMapping-transaction.
	failNext := true
	db.Callback().Create().Before("gorm:create").Register("fail-create-hook", func(tx *gorm.DB) {
		if !failNext {
			return
		}
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "sys_dept_ou_mapping" {
			tx.Error = assert.AnError
			return
		}
	})

	// New mapping reassigns the OU_DN to a different dept. UpsertMapping
	// will (1) find existing OU_DN, (2) DELETE it, (3) try to CREATE new
	// one — and the hook will force step (3) to fail. The transaction
	// must roll back step (2).
	newMapping := &models.DeptOUMapping{
		ID:         "new-1",
		DeptID:     "dept-new",
		ADConfigID: "config-1",
		OUDN:       "OU=Original,DC=corp,DC=local",
		OUName:     "RenamedDept",
	}

	// 62-03 HYGIENE-05 严格化后 UpsertMapping 异部门冲突直接返回 OUDNConflictError（不删不建），
	// P1 atomic rollback（删旧→建新失败→回滚）语义由 UpsertMappingForce 承接，此处改走 Force 驱动同一路径。
	err = mapper.UpsertMappingForce(ctx, newMapping)
	failNext = false // disarm for any subsequent unrelated calls

	// We expect an error (Create failed)
	assert.Error(t, err, "UpsertMapping should fail when Create hook rejects")

	// PRIMARY ASSERTION: dept-original mapping MUST still exist (atomic rollback).
	var origCount int64
	db.Model(&models.DeptOUMapping{}).Where("dept_id = ?", "dept-original").Count(&origCount)
	assert.Equal(t, int64(1), origCount,
		"dept-original mapping must survive UpsertMapping failure (P1-C3 atomic rollback)")

	// newMapping should NOT exist
	var newCount int64
	db.Model(&models.DeptOUMapping{}).Where("id = ?", "new-1").Count(&newCount)
	assert.Equal(t, int64(0), newCount, "new-1 row must not exist after rollback")
}

// TestUpsertMapping_TransactionSuccess verifies the happy path:
// successful upsert (same dept_id) updates fields in place and does
// NOT create a duplicate row. Validates that the transaction wrapper
// doesn't break the standard idempotent upsert semantics.
func TestUpsertMapping_TransactionSuccess(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE sys_dept_ou_mapping (
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
			UNIQUE(dept_id, ad_config_id),
			UNIQUE(ad_config_id, ou_dn)
		);
	`).Error
	assert.NoError(t, err)

	mapper := NewDeptOUmapper(db)
	ctx := context.Background()

	// First upsert creates
	m1 := &models.DeptOUMapping{
		ID:         "row-1",
		DeptID:     "dept-A",
		ADConfigID: "cfg-1",
		OUDN:       "OU=A,DC=corp",
		OUName:     "DeptA",
	}
	err = mapper.UpsertMapping(ctx, m1)
	assert.NoError(t, err)

	// Second upsert with same dept_id+config but updated OU name → in-place update
	m2 := &models.DeptOUMapping{
		ID:         "row-1-updated",
		DeptID:     "dept-A",
		ADConfigID: "cfg-1",
		OUDN:       "OU=A,DC=corp",
		OUName:     "DeptA-Renamed",
	}
	err = mapper.UpsertMapping(ctx, m2)
	assert.NoError(t, err)

	// Verify exactly one row (idempotent)
	var count int64
	db.Model(&models.DeptOUMapping{}).Where("dept_id = ?", "dept-A").Count(&count)
	assert.Equal(t, int64(1), count, "second upsert should be in-place update, not duplicate")

	// Verify the name was updated
	var mapping models.DeptOUMapping
	err = db.Where("dept_id = ?", "dept-A").First(&mapping).Error
	assert.NoError(t, err)
	assert.Equal(t, "DeptA-Renamed", mapping.OUName, "OU name should be updated by second upsert")
}

// TestUpsertMapping_DifferentOUTriggersDeleteAndInsert（62-03 翻转）：
// HYGIENE-05 后异部门争用同一 OU DN 必须返回 *OUDNConflictError 且旧行原样——
// 不再静默删旧建新。显式覆盖语义由 TestUpsertMapping_ForceRebindReplacesOld 锁定。
func TestUpsertMapping_DifferentOUTriggersDeleteAndInsert(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE sys_dept_ou_mapping (
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
			UNIQUE(dept_id, ad_config_id),
			UNIQUE(ad_config_id, ou_dn)
		);
	`).Error
	assert.NoError(t, err)

	mapper := NewDeptOUmapper(db)
	ctx := context.Background()

	// Initial mapping: dept-A → OU=A
	m1 := &models.DeptOUMapping{
		ID:         "row-A",
		DeptID:     "dept-A",
		ADConfigID: "cfg-1",
		OUDN:       "OU=A,DC=corp",
		OUName:     "DeptA",
	}
	err = mapper.UpsertMapping(ctx, m1)
	assert.NoError(t, err)

	// Reassign OU=A to dept-B (different dept) → must CONFLICT, not overwrite
	m2 := &models.DeptOUMapping{
		ID:         "row-B",
		DeptID:     "dept-B",
		ADConfigID: "cfg-1",
		OUDN:       "OU=A,DC=corp",
		OUName:     "DeptB",
	}
	err = mapper.UpsertMapping(ctx, m2)
	require.Error(t, err, "异部门争用同一 OU DN 必须返回错误")
	var conflict *OUDNConflictError
	require.True(t, errors.As(err, &conflict), "错误必须可 errors.As 为 *OUDNConflictError")
	assert.Equal(t, "dept-A", conflict.ExistingDeptID)
	assert.Equal(t, "dept-B", conflict.IncomingDeptID)
	assert.True(t, errors.Is(err, ErrOUDNConflict), "Unwrap 必须到 ErrOUDNConflict 哨兵")

	// dept-A mapping must SURVIVE untouched
	var countA int64
	db.Model(&models.DeptOUMapping{}).Where("dept_id = ?", "dept-A").Count(&countA)
	assert.Equal(t, int64(1), countA, "冲突后 incumbent 映射必须原样保留")

	// dept-B must NOT own OU=A
	var countB int64
	db.Model(&models.DeptOUMapping{}).Where("dept_id = ? AND ou_dn = ?", "dept-B", "OU=A,DC=corp").Count(&countB)
	assert.Equal(t, int64(0), countB, "冲突方不得落行")
}

// TestUpsertMapping_ForceRebindReplacesOld 锁定 Force 显式覆盖语义（62-03）：
// 同冲突场景走 UpsertMappingForce → 旧行被删、新行生效（原 UpsertMapping 行为由 Force 承接）。
func TestUpsertMapping_ForceRebindReplacesOld(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	err = db.Exec(`
		CREATE TABLE sys_dept_ou_mapping (
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
			UNIQUE(dept_id, ad_config_id),
			UNIQUE(ad_config_id, ou_dn)
		);
	`).Error
	assert.NoError(t, err)

	mapper := NewDeptOUmapper(db)
	ctx := context.Background()

	m1 := &models.DeptOUMapping{ID: "row-A", DeptID: "dept-A", ADConfigID: "cfg-1", OUDN: "OU=A,DC=corp", OUName: "DeptA"}
	assert.NoError(t, mapper.UpsertMapping(ctx, m1))

	m2 := &models.DeptOUMapping{ID: "row-B", DeptID: "dept-B", ADConfigID: "cfg-1", OUDN: "OU=A,DC=corp", OUName: "DeptB"}
	assert.NoError(t, mapper.UpsertMappingForce(ctx, m2), "Force 必须成功完成重绑")

	var countA int64
	db.Model(&models.DeptOUMapping{}).Where("dept_id = ?", "dept-A").Count(&countA)
	assert.Equal(t, int64(0), countA, "Force 后旧映射被删除")
	var countB int64
	db.Model(&models.DeptOUMapping{}).Where("dept_id = ? AND ou_dn = ?", "dept-B", "OU=A,DC=corp").Count(&countB)
	assert.Equal(t, int64(1), countB, "Force 后新映射生效")
}

// TestUpsertMapping_ConflictCarriesDeptNames 冲突错误在 sys_dept 可查时携带双方部门名（HYGIENE-05 告警素材）。
func TestUpsertMapping_ConflictCarriesDeptNames(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	err = db.Exec(`
		CREATE TABLE sys_dept_ou_mapping (
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
			UNIQUE(dept_id, ad_config_id),
			UNIQUE(ad_config_id, ou_dn)
		);
		CREATE TABLE sys_dept (
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
		);
		INSERT INTO sys_dept (id, dept_name) VALUES ('dept-A', '财务部'), ('dept-B', '人资部');
	`).Error
	assert.NoError(t, err)

	mapper := NewDeptOUmapper(db)
	ctx := context.Background()
	m1 := &models.DeptOUMapping{ID: "row-A", DeptID: "dept-A", ADConfigID: "cfg-1", OUDN: "OU=同名,DC=corp", OUName: "同名"}
	assert.NoError(t, mapper.UpsertMapping(ctx, m1))
	m2 := &models.DeptOUMapping{ID: "row-B", DeptID: "dept-B", ADConfigID: "cfg-1", OUDN: "OU=同名,DC=corp", OUName: "同名"}
	err = mapper.UpsertMapping(ctx, m2)
	require.Error(t, err)
	var conflict *OUDNConflictError
	require.True(t, errors.As(err, &conflict))
	assert.Equal(t, "财务部", conflict.ExistingDeptName, "冲突错误必须携带 incumbent 部门名")
	assert.Equal(t, "人资部", conflict.IncomingDeptName, "冲突错误必须携带 incoming 部门名")
	assert.Contains(t, err.Error(), "财务部")
	assert.Contains(t, err.Error(), "人资部")
}

// TestUpsertMapping_SameDeptUpdateStillWorks 防回归护栏：同部门幂等更新不受严格化影响。
func TestUpsertMapping_SameDeptUpdateStillWorks(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	err = db.Exec(`
		CREATE TABLE sys_dept_ou_mapping (
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
			UNIQUE(dept_id, ad_config_id),
			UNIQUE(ad_config_id, ou_dn)
		);
	`).Error
	assert.NoError(t, err)

	mapper := NewDeptOUmapper(db)
	ctx := context.Background()
	m1 := &models.DeptOUMapping{ID: "row-A", DeptID: "dept-A", ADConfigID: "cfg-1", OUDN: "OU=A,DC=corp", OUName: "DeptA"}
	assert.NoError(t, mapper.UpsertMapping(ctx, m1))
	m2 := &models.DeptOUMapping{ID: "row-A2", DeptID: "dept-A", ADConfigID: "cfg-1", OUDN: "OU=A,DC=corp", OUName: "DeptA-Renamed"}
	assert.NoError(t, mapper.UpsertMapping(ctx, m2), "同部门重复 upsert 必须幂等成功")
	var count int64
	db.Model(&models.DeptOUMapping{}).Where("dept_id = ?", "dept-A").Count(&count)
	assert.Equal(t, int64(1), count)
	var mapping models.DeptOUMapping
	assert.NoError(t, db.Where("dept_id = ?", "dept-A").First(&mapping).Error)
	assert.Equal(t, "DeptA-Renamed", mapping.OUName)
}