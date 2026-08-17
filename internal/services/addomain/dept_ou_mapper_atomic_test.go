//go:build !skip_db_tests
// +build !skip_db_tests

package addomain

import (
	"context"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/assert"
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

	err = mapper.UpsertMapping(ctx, newMapping)
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

// TestUpsertMapping_DifferentOUTriggersDeleteAndInsert verifies that
// reassigning the same OU_DN to a different dept correctly deletes
// the old mapping and inserts the new one — atomically.
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

	// Reassign OU=A to dept-B (different dept)
	m2 := &models.DeptOUMapping{
		ID:         "row-B",
		DeptID:     "dept-B",
		ADConfigID: "cfg-1",
		OUDN:       "OU=A,DC=corp",
		OUName:     "DeptB",
	}
	err = mapper.UpsertMapping(ctx, m2)
	assert.NoError(t, err)

	// dept-A mapping should be gone
	var countA int64
	db.Model(&models.DeptOUMapping{}).Where("dept_id = ?", "dept-A").Count(&countA)
	assert.Equal(t, int64(0), countA, "dept-A mapping should be deleted after reassignment")

	// dept-B should now own OU=A
	var countB int64
	db.Model(&models.DeptOUMapping{}).Where("dept_id = ? AND ou_dn = ?", "dept-B", "OU=A,DC=corp").Count(&countB)
	assert.Equal(t, int64(1), countB, "dept-B should now have OU=A mapping")
}