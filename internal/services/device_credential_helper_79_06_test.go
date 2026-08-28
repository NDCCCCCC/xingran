// Phase 79-06 (TAIL-01) — device_credential_helper.go coverage tests.
//
// 47 stmts, class (c) sqlite/pure per 79-RESEARCH §2: every helper is a
// credential lookup against sys_auth_credential with a default-credential
// fallback. All branches are driven through a glebarez sqlite file DB in
// t.TempDir() (79-04 newAcv7904 pattern — pq.StringArray SNMPCommunities is
// AutoMigrate-safe there).
//
// Naming: D-79-06 `<source>_79_06_test.go` + plan-suffixed helpers
// (newDch7906 / dch7906Seed*) — no collision with existing same-package
// helpers (setupTestService / loadSampleFixture / *_79_0[1-5] helpers).
//
// Shared sqlite assembly `newDB7906` also lives here: it is reused by the
// other 79-06 files and installs a test-side UUID fill callback because the
// plain-UUID models (DeviceDiscovery / ConfigExecution / ConfigExecutionDetail)
// carry no BeforeCreate hook — on PostgreSQL the PK columns carry
// gen_random_uuid() defaults, sqlite has none, so a second Create would
// collide on the empty-string primary key.
package services

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// -----------------------------------------------------------------------------
// shared sqlite assembly (used across the 79-06 files)
// -----------------------------------------------------------------------------

// newDB7906 opens a glebarez sqlite file DB under t.TempDir(), AutoMigrates the
// given models and installs the test-side UUID fill callback. The DB handle is
// closed via t.Cleanup. Callers own seeding.
func newDB7906(t *testing.T, migrateModels ...any) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "svc7906.db")), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "open sqlite temp db")
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.AutoMigrate(migrateModels...), "auto migrate %v", migrateModels)

	// test7906:fill_uuid — fill empty string PKs with UUIDs before insert so
	// models without a BeforeCreate hook (DeviceDiscovery, ConfigExecution,
	// ConfigExecutionDetail) can be created more than once per table.
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("test7906:fill_uuid", fillUUID7906))
	return db
}

// fillUUID7906 is the gorm callback body: fills an empty `ID string` field on
// the Create destination (struct, pointer-to-struct or slice thereof).
func fillUUID7906(tx *gorm.DB) {
	if tx == nil || tx.Statement == nil || tx.Statement.Dest == nil {
		return
	}
	rv := reflect.ValueOf(tx.Statement.Dest)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Struct:
		setUUIDIfEmpty7906(rv)
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			el := rv.Index(i)
			for el.Kind() == reflect.Pointer {
				if el.IsNil() {
					break
				}
				el = el.Elem()
			}
			if el.Kind() == reflect.Struct {
				setUUIDIfEmpty7906(el)
			}
		}
	}
}

// setUUIDIfEmpty7906 fills rv.ID with a fresh UUID when the field is a settable
// empty string.
func setUUIDIfEmpty7906(rv reflect.Value) {
	f := rv.FieldByName("ID")
	if !f.IsValid() || f.Kind() != reflect.String || !f.CanSet() || f.String() != "" {
		return
	}
	f.SetString(uuid.NewString())
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

// newDch7906 assembles a DeviceCredentialHelper over a fresh sqlite DB with the
// AuthCredential + NetworkDevice chain migrated.
func newDch7906(t *testing.T) (*DeviceCredentialHelper, *gorm.DB) {
	t.Helper()
	db := newDB7906(t, &models.AuthCredential{}, &models.NetworkDevice{})
	return &DeviceCredentialHelper{db: db}, db
}

// dch7906SeedCred inserts a credential row directly (bypassing the service
// layer) and returns it.
func dch7906SeedCred(t *testing.T, db *gorm.DB, name string, isDefault bool) *models.AuthCredential {
	t.Helper()
	cred := &models.AuthCredential{
		CredentialName:  name,
		ProtocolType:    models.ProtocolTypeSSH,
		Username:        "admin",
		Password:        "cipher-" + name,
		SNMPCommunities: pq.StringArray{"public"},
		SNMPVersion:     models.SNMPVersionV2c,
		IsDefault:       isDefault,
	}
	require.NoError(t, db.Create(cred).Error, "seed credential %s", name)
	return cred
}

// dch7906DeviceIPSeq allocates unique third octets — IPAddress carries a
// uniqueIndex, so seeded devices must never collide.
var dch7906DeviceIPSeq int

// dch7906SeedDevice inserts a NetworkDevice row, optionally pointing at a
// credential (nil → no credential association).
func dch7906SeedDevice(t *testing.T, db *gorm.DB, name string, credID *string) *models.NetworkDevice {
	t.Helper()
	dch7906DeviceIPSeq++
	dev := &models.NetworkDevice{
		DeviceName:   name,
		DeviceType:   models.DeviceTypeSwitch,
		Vendor:       models.VendorHuawei,
		IPAddress:    fmt.Sprintf("10.79.%d.1", dch7906DeviceIPSeq),
		CredentialID: credID,
		Status:       models.DeviceStatusOnline,
	}
	require.NoError(t, db.Create(dev).Error, "seed device %s", name)
	return dev
}

// -----------------------------------------------------------------------------
// TestDch7906_GetDeviceCredential — 单设备凭证解析四分支
// -----------------------------------------------------------------------------

// TestDch7906_GetDeviceCredential drives GetDeviceCredential's four branches:
// associated credential hit / associated credential missing / default
// credential fallback (incl. empty-string CredentialID pointer) / no default
// available.
func TestDch7906_GetDeviceCredential(t *testing.T) {
	ctx := context.Background()
	h, db := newDch7906(t)

	cred := dch7906SeedCred(t, db, "core-cred", false)
	defaultCred := dch7906SeedCred(t, db, "default-cred", true)

	t.Run("associated_credential_hit", func(t *testing.T) {
		dev := dch7906SeedDevice(t, db, "assoc-hit", &cred.ID)
		got, err := h.GetDeviceCredential(ctx, dev)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, cred.ID, got.ID, "must return the associated credential")
	})

	t.Run("associated_credential_missing_row", func(t *testing.T) {
		ghost := uuid.NewString()
		dev := dch7906SeedDevice(t, db, "assoc-miss", &ghost)
		got, err := h.GetDeviceCredential(ctx, dev)
		require.Error(t, err, "missing associated credential row must fail")
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "查询设备关联凭证失败")
	})

	t.Run("default_fallback_no_association", func(t *testing.T) {
		dev := dch7906SeedDevice(t, db, "no-assoc", nil)
		got, err := h.GetDeviceCredential(ctx, dev)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, defaultCred.ID, got.ID, "unassociated device falls back to the default credential")
	})

	t.Run("default_fallback_empty_string_credential_id", func(t *testing.T) {
		// `*device.CredentialID != ""` guard: an empty-string pointer is treated
		// as "no association" and falls through to the default lookup.
		empty := ""
		dev := dch7906SeedDevice(t, db, "empty-assoc", &empty)
		got, err := h.GetDeviceCredential(ctx, dev)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, defaultCred.ID, got.ID)
	})

	t.Run("no_default_available", func(t *testing.T) {
		// Fresh DB without any default credential → explicit error.
		h2, db2 := newDch7906(t)
		lone := dch7906SeedCred(t, db2, "non-default-only", false)
		dev := dch7906SeedDevice(t, db2, "orphan", nil)
		_ = lone
		got, err := h2.GetDeviceCredential(ctx, dev)
		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "未找到可用凭证")
	})
}

// -----------------------------------------------------------------------------
// TestDch7906_GetCredentialByID_And_Default — 按 ID / 默认凭证查询
// -----------------------------------------------------------------------------

// TestDch7906_GetCredentialByID_And_Default table-drives GetCredentialByID and
// GetDefaultCredential hit/miss branches.
func TestDch7906_GetCredentialByID_And_Default(t *testing.T) {
	ctx := context.Background()
	h, db := newDch7906(t)

	cred := dch7906SeedCred(t, db, "byid", false)
	defaultCred := dch7906SeedCred(t, db, "def", true)

	t.Run("get_by_id_hit", func(t *testing.T) {
		got, err := h.GetCredentialByID(ctx, cred.ID)
		require.NoError(t, err)
		assert.Equal(t, cred.ID, got.ID)
	})

	t.Run("get_by_id_miss", func(t *testing.T) {
		got, err := h.GetCredentialByID(ctx, uuid.NewString())
		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "查询凭证失败")
	})

	t.Run("get_default_hit", func(t *testing.T) {
		got, err := h.GetDefaultCredential(ctx)
		require.NoError(t, err)
		assert.Equal(t, defaultCred.ID, got.ID)
	})

	t.Run("get_default_miss", func(t *testing.T) {
		h2, _ := newDch7906(t)
		got, err := h2.GetDefaultCredential(ctx)
		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "未找到默认凭证")
	})
}

// -----------------------------------------------------------------------------
// TestDch7906_GetCredentialsForDevices — 批量凭证装配(N+1 规避)
// -----------------------------------------------------------------------------

// TestDch7906_GetCredentialsForDevices drives the batch assembly: associated
// credentials resolved in one IN query, default fallback shared across
// unassociated devices, and the edge branches (empty batch, nonexistent
// credential row, no default for the fallback group).
func TestDch7906_GetCredentialsForDevices(t *testing.T) {
	ctx := context.Background()

	t.Run("mixed_batch", func(t *testing.T) {
		h, db := newDch7906(t)
		credA := dch7906SeedCred(t, db, "cred-a", false)
		credB := dch7906SeedCred(t, db, "cred-b", false)
		defaultCred := dch7906SeedCred(t, db, "cred-default", true)

		devA := dch7906SeedDevice(t, db, "dev-a", &credA.ID)
		devB := dch7906SeedDevice(t, db, "dev-b", &credB.ID)
		devC := dch7906SeedDevice(t, db, "dev-c", nil)
		devD := dch7906SeedDevice(t, db, "dev-d", nil)

		result, err := h.GetCredentialsForDevices(ctx, []models.NetworkDevice{*devA, *devB, *devC, *devD})
		require.NoError(t, err)
		require.Len(t, result, 4, "every device must get a credential")

		assert.Equal(t, credA.ID, result[devA.ID].ID, "dev-a uses its associated credential")
		assert.Equal(t, credB.ID, result[devB.ID].ID, "dev-b uses its associated credential")
		assert.Equal(t, defaultCred.ID, result[devC.ID].ID, "dev-c shares the default credential")
		assert.Equal(t, defaultCred.ID, result[devD.ID].ID, "dev-d shares the default credential")
	})

	t.Run("empty_batch", func(t *testing.T) {
		h, _ := newDch7906(t)
		result, err := h.GetCredentialsForDevices(ctx, nil)
		require.NoError(t, err)
		assert.Empty(t, result, "empty batch → empty map, no error")
	})

	t.Run("nonexistent_associated_credential_is_silently_dropped", func(t *testing.T) {
		h, db := newDch7906(t)
		defaultCred := dch7906SeedCred(t, db, "def", true)
		ghost := uuid.NewString()
		devGhost := dch7906SeedDevice(t, db, "ghost", &ghost)
		devPlain := dch7906SeedDevice(t, db, "plain", nil)

		result, err := h.GetCredentialsForDevices(ctx, []models.NetworkDevice{*devGhost, *devPlain})
		require.NoError(t, err)
		// QUIRK-79-06-A (locked, not fixed): a device whose CredentialID points
		// at a deleted credential row silently drops out of the result map —
		// the credMap lookup yields nil and no entry is written, no error is
		// raised. Callers must treat a missing key as "no credential".
		assert.NotContains(t, result, devGhost.ID, "ghost credential → device missing from map (locked quirk)")
		assert.Equal(t, defaultCred.ID, result[devPlain.ID].ID)
	})

	t.Run("fallback_without_default_errors", func(t *testing.T) {
		h, db := newDch7906(t)
		dch7906SeedCred(t, db, "plain-only", false) // no default anywhere
		dev := dch7906SeedDevice(t, db, "needs-default", nil)

		result, err := h.GetCredentialsForDevices(ctx, []models.NetworkDevice{*dev})
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "获取默认凭证失败")
	})
}
