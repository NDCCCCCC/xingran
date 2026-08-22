package addomain

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// =====================================================================
// account_pool.go ListAll / CountByStatus / PickFirstAvailable / Create /
// Update / Delete / SetEnabled:真实 sqlite + ADServiceAccount 行。
// =====================================================================

func newAPTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:ap_"+t.Name()+"?mode=memory&cache=shared&_enable_boolean=true&_busy_timeout=5000"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_ad_service_accounts (
			id TEXT PRIMARY KEY,
			config_id TEXT NOT NULL,
			username TEXT NOT NULL,
			password_ciphertext TEXT NOT NULL,
			status INTEGER DEFAULT 0,
			failure_count INTEGER DEFAULT 0,
			circuit_breaker_until DATETIME,
			last_success_at DATETIME,
			last_failure_at DATETIME,
			last_failure_reason TEXT,
			manual_unlock_reason TEXT,
			manual_unlocked_by TEXT,
			manual_unlocked_at DATETIME,
			remark TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	return db
}

func newAP(t *testing.T, db *gorm.DB) *accountPoolImpl {
	t.Helper()
	return NewAccountPool(db, nil).(*accountPoolImpl)
}

func seedAP(t *testing.T, db *gorm.DB, configID, username string, status int) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_service_accounts (id, config_id, username, password_ciphertext, status, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, 'enc', ?, '2024-01-01', '2024-01-01', NULL)
	`, id, configID, username, status).Error)
	return id
}

func TestAP_ListAll_Pagination(t *testing.T) {
	db := newAPTestDB(t)
	p := newAP(t, db)
	ctx := context.Background()

	// 空 → 0 行
	list, total, err := p.ListAll(ctx, "ghost", 1, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, list)
	assert.Equal(t, int64(0), total)

	// 3 行
	cid := uuid.NewString()
	for i := 0; i < 3; i++ {
		seedAP(t, db, cid, "u"+string(rune('a'+i)), 0)
	}
	list, total, err = p.ListAll(ctx, cid, 1, 10, nil)
	require.NoError(t, err)
	assert.Len(t, list, 3)
	assert.Equal(t, int64(3), total)

	// statusFilter
	disabled := 1
	list, total, err = p.ListAll(ctx, cid, 1, 10, &disabled)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	_ = list

	// page=0 → 强制 1
	list, _, err = p.ListAll(ctx, cid, 0, 10, nil)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestAP_CountByStatus(t *testing.T) {
	db := newAPTestDB(t)
	p := newAP(t, db)
	ctx := context.Background()

	cid := uuid.NewString()
	seedAP(t, db, cid, "u0", 0)
	seedAP(t, db, cid, "u1", 1)
	seedAP(t, db, cid, "u2", 2)

	total, avail, disabled, broken, err := p.CountByStatus(ctx, cid)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, int64(1), avail)
	assert.Equal(t, int64(1), disabled)
	assert.Equal(t, int64(1), broken)
}

func TestAP_PickFirstAvailable(t *testing.T) {
	db := newAPTestDB(t)
	p := newAP(t, db)
	ctx := context.Background()

	// 无账号 → nil, nil err
	a, err := p.PickFirstAvailable(ctx, "ghost")
	require.NoError(t, err)
	assert.Nil(t, a)

	// 有 → 返回
	cid := uuid.NewString()
	seedAP(t, db, cid, "u", 0)
	a, err = p.PickFirstAvailable(ctx, cid)
	require.NoError(t, err)
	require.NotNil(t, a)
	assert.Equal(t, "u", a.Username)
}

func TestAP_CreateUpdateDelete(t *testing.T) {
	db := newAPTestDB(t)
	p := newAP(t, db)
	ctx := context.Background()

	// Create
	acc := &models.ADServiceAccount{
		ID:                  uuid.NewString(),
		ConfigID:            uuid.NewString(),
		Username:            "alice",
		PasswordCiphertext:  "enc",
		Status:              0,
	}
	require.NoError(t, p.Create(ctx, acc))

	// Update
	acc.Username = "alice2"
	require.NoError(t, p.Update(ctx, acc))

	// Verify via PickFirstAvailable
	a, err := p.PickFirstAvailable(ctx, acc.ConfigID)
	require.NoError(t, err)
	require.NotNil(t, a)
	assert.Equal(t, "alice2", a.Username)

	// Delete (软删)
	require.NoError(t, p.Delete(ctx, acc.ID))

	// 已删 → PickFirstAvailable 返回 nil
	a, err = p.PickFirstAvailable(ctx, acc.ConfigID)
	require.NoError(t, err)
	assert.Nil(t, a)
}

func TestAP_SetEnabled(t *testing.T) {
	db := newAPTestDB(t)
	p := newAP(t, db)
	ctx := context.Background()

	id := seedAP(t, db, uuid.NewString(), "u", 0)

	// Disable
	require.NoError(t, p.SetEnabled(ctx, id, false))

	// Verify status=1
	var status int
	require.NoError(t, db.Raw(`SELECT status FROM sys_ad_service_accounts WHERE id = ?`, id).Scan(&status).Error)
	assert.Equal(t, 1, status)

	// Enable 回 → status=0
	require.NoError(t, p.SetEnabled(ctx, id, true))
	require.NoError(t, db.Raw(`SELECT status FROM sys_ad_service_accounts WHERE id = ?`, id).Scan(&status).Error)
	assert.Equal(t, 0, status)
}