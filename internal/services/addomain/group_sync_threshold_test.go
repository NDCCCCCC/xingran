//go:build !skip_db_tests
// +build !skip_db_tests

package addomain

import (
	"context"
	"testing"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	// Pure-Go SQLite driver
	_ "modernc.org/sqlite"
)

// setupGroupSyncTestDB creates an in-memory SQLite DB with the
// minimal schema required by GroupSyncService.handleDeletedGroups.
func setupGroupSyncTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default,
	})
	assert.NoError(t, err)

	// Use the production table name (sys_ad_group) so GORM queries
	// inside handleDeletedGroups resolve correctly.
	err = db.Exec(`
		CREATE TABLE sys_ad_group (
			id TEXT PRIMARY KEY,
			ad_config_id TEXT,
			group_dn TEXT,
			group_name TEXT,
			description TEXT,
			member_count INTEGER,
			ou_dn TEXT,
			group_scope TEXT,
			group_type INTEGER,
			last_sync_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
		CREATE INDEX idx_ad_groups_config ON sys_ad_group(ad_config_id);
		CREATE INDEX idx_ad_groups_dn ON sys_ad_group(group_dn);
		-- handleDeletedGroups also tries to clean up sys_ad_group_member rows
		-- matching the deleted DNs; we create an empty table to silence the
		-- "no such table" warning.
		CREATE TABLE sys_ad_group_member (
			id TEXT PRIMARY KEY,
			ad_config_id TEXT,
			group_dn TEXT,
			user_dn TEXT,
			created_at DATETIME
		);
	`).Error
	assert.NoError(t, err)

	return db
}

// seedGroups inserts n AD groups for the given config ID with deterministic
// group DNs "CN=group-<i>,DC=test,DC=local".
func seedGroups(t *testing.T, db *gorm.DB, configID string, n int) []string {
	now := time.Now().Format("2006-01-02 15:04:05")
	dns := make([]string, 0, n)
	for i := 0; i < n; i++ {
		dn := "CN=group-" + itoaForTest(i) + ",DC=test,DC=local"
		// group_type is an integer enum in the model (0=Distribution, 1=Security).
		// Use 1 to match the typical seed data.
		err := db.Exec(
			`INSERT INTO sys_ad_group (id, ad_config_id, group_dn, group_name, group_scope, group_type, created_at, updated_at)
			 VALUES (?, ?, ?, ?, 'global', 1, ?, ?)`,
			"id-"+itoaForTest(i), configID, dn, "Group "+itoaForTest(i), now, now,
		).Error
		assert.NoError(t, err)
		dns = append(dns, dn)
	}
	return dns
}

func itoaForTest(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	buf := [20]byte{}
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

// TestHandleDeletedGroups_RejectsEmptyLDAP verifies P1-C2 gate #1:
// when LDAP returns 0 entries (e.g., server unreachable, wrong BaseDN,
// network partition), the function must NOT delete any existing groups.
// Otherwise the entire AD→group mapping table would be wiped out.
func TestHandleDeletedGroups_RejectsEmptyLDAP(t *testing.T) {
	db := setupGroupSyncTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()

	configID := "cfg-empty-ldap"
	seedGroups(t, db, configID, 10)

	svc := &GroupSyncService{db: db}
	config := &models.ADConfig{ConfigName: "test"}
	config.ID = configID
	result := &GroupSyncResult{}

	// Empty LDAP entries — simulates AD unreachable / misconfigured BaseDN
	var emptyEntries []*ldap.Entry

	err := svc.handleDeletedGroups(context.Background(), config, emptyEntries, result)

	// P1-C2 fix: empty LDAP must NOT trigger deletion. Function returns nil
	// (with a warn log) rather than deleting the entire local AD group table.
	assert.NoError(t, err, "empty LDAP should not return an error")
	assert.Equal(t, 0, result.DeletedGroups, "no groups should be deleted")

	// Verify all 10 seeded groups still exist (not soft-deleted)
	var remaining int64
	db.Model(&models.ADGroup{}).Where("ad_config_id = ? AND deleted_at IS NULL", configID).Count(&remaining)
	assert.Equal(t, int64(10), remaining, "all 10 groups should still exist after empty LDAP")
}

// TestHandleDeletedGroups_RejectsOverThreshold verifies P1-C2 gate #2:
// when the number of would-be-deleted groups exceeds 50% of the local
// table, refuse deletion (likely LDAP returned incomplete results).
func TestHandleDeletedGroups_RejectsOverThreshold(t *testing.T) {
	db := setupGroupSyncTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()

	configID := "cfg-over-threshold"
	seedGroups(t, db, configID, 20)

	svc := &GroupSyncService{db: db}
	config := &models.ADConfig{ConfigName: "test"}
	config.ID = configID
	result := &GroupSyncResult{}

	// LDAP returns only 5 of 20 groups — 15 would be deleted (75%, > 50%)
	ldapEntries := []*ldap.Entry{
		{DN: "CN=group-0,DC=test,DC=local"},
		{DN: "CN=group-1,DC=test,DC=local"},
		{DN: "CN=group-2,DC=test,DC=local"},
		{DN: "CN=group-3,DC=test,DC=local"},
		{DN: "CN=group-4,DC=test,DC=local"},
	}

	err := svc.handleDeletedGroups(context.Background(), config, ldapEntries, result)

	// P1-C2 fix: 15/20 = 75% > 50%, deletion rejected. Function returns nil
	// (warn log) rather than wiping out most of the table.
	assert.NoError(t, err)
	assert.Equal(t, 0, result.DeletedGroups, "no groups should be deleted when ratio exceeds threshold")

	// All 20 original groups must still exist
	var remaining int64
	db.Model(&models.ADGroup{}).Where("ad_config_id = ? AND deleted_at IS NULL", configID).Count(&remaining)
	assert.Equal(t, int64(20), remaining)
}

// TestHandleDeletedGroups_AllowsUnderThreshold verifies the inverse:
// when the deletion ratio is < 50%, the function should proceed
// with the soft-delete (this is the normal incremental sync case).
func TestHandleDeletedGroups_AllowsUnderThreshold(t *testing.T) {
	db := setupGroupSyncTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()

	configID := "cfg-under-threshold"
	seedGroups(t, db, configID, 20)

	svc := &GroupSyncService{db: db}
	config := &models.ADConfig{ConfigName: "test"}
	config.ID = configID
	result := &GroupSyncResult{}

	// LDAP returns 15 of 20 — 5 to delete (25%, < 50%)
	ldapEntries := make([]*ldap.Entry, 0, 15)
	for i := 0; i < 15; i++ {
		ldapEntries = append(ldapEntries, &ldap.Entry{DN: "CN=group-" + itoaForTest(i) + ",DC=test,DC=local"})
	}

	err := svc.handleDeletedGroups(context.Background(), config, ldapEntries, result)

	assert.NoError(t, err)
	assert.Equal(t, 5, result.DeletedGroups, "5 groups should be soft-deleted (25% ratio under threshold)")

	// Verify 15 remain (5 soft-deleted)
	var remaining int64
	db.Model(&models.ADGroup{}).Where("ad_config_id = ? AND deleted_at IS NULL", configID).Count(&remaining)
	assert.Equal(t, int64(15), remaining)
}

// TestHandleDeletedGroups_AtExactThreshold verifies boundary behavior:
// exactly 50% should be allowed (the gate is ">", not ">=").
func TestHandleDeletedGroups_AtExactThreshold(t *testing.T) {
	db := setupGroupSyncTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()

	configID := "cfg-exact-threshold"
	seedGroups(t, db, configID, 10)

	svc := &GroupSyncService{db: db}
	config := &models.ADConfig{ConfigName: "test"}
	config.ID = configID
	result := &GroupSyncResult{}

	// LDAP returns 5 of 10 — exactly 50% would be deleted
	ldapEntries := make([]*ldap.Entry, 0, 5)
	for i := 0; i < 5; i++ {
		ldapEntries = append(ldapEntries, &ldap.Entry{DN: "CN=group-" + itoaForTest(i) + ",DC=test,DC=local"})
	}

	err := svc.handleDeletedGroups(context.Background(), config, ldapEntries, result)

	// The production code uses len(deletedDNs)*2 > len(staleGroups), which
	// means 5*2=10 > 10 is FALSE → deletion proceeds at exactly 50%.
	assert.NoError(t, err)
	assert.Equal(t, 5, result.DeletedGroups, "50% deletion should be allowed (gate is strictly >)")
}