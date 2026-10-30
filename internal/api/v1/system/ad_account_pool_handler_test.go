package system

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	coredb "github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	addomainServices "github.com/xingran-next/xingran-go-backend/internal/services/addomain"
)

// =====================================================================
// Phase 74-04: ad_account_handler (Phase 36 pool endpoints) tests.
//
// ADAccountHandler.pool is the addomainServices.AccountPool interface —
// fully mockable via per-method *Func fields. Create/Update additionally
// need core.SM4Cipher (fake PasswordCipher) and a sqlite DB (Update reads
// the existing row via core.GetDB()).
// =====================================================================

// fakeADPasswordCipher is a deterministic stand-in for the SM4 cipher.
type fakeADPasswordCipher struct{ fail bool }

func (f *fakeADPasswordCipher) Encrypt(plaintext string) (string, error) {
	if f.fail {
		return "", errors.New("encrypt boom")
	}
	return "enc:" + plaintext, nil
}
func (f *fakeADPasswordCipher) Decrypt(ciphertext string) (string, error) {
	return ciphertext, nil
}

type mockAccountPool struct {
	PickAvailableFunc      func(ctx context.Context, configID string) (*models.ADServiceAccount, error)
	ListAvailableFunc      func(ctx context.Context, configID string) ([]models.ADServiceAccount, error)
	ListAllFunc            func(ctx context.Context, configID string, page, pageSize int, statusFilter *int) ([]models.ADServiceAccount, int64, error)
	CountByStatusFunc      func(ctx context.Context, configID string) (total, available, disabled, circuitBroken int64, err error)
	PickFirstAvailableFunc func(ctx context.Context, configID string) (*models.ADServiceAccount, error)
	CreateFunc             func(ctx context.Context, account *models.ADServiceAccount) error
	UpdateFunc             func(ctx context.Context, account *models.ADServiceAccount) error
	DeleteFunc             func(ctx context.Context, accountID string) error
	MarkSuccessFunc        func(ctx context.Context, accountID string) error
	MarkFailureFunc        func(ctx context.Context, accountID, reason string) error
	ManualUnlockFunc       func(ctx context.Context, accountID, operator, reason string) error
	SetEnabledFunc         func(ctx context.Context, accountID string, enabled bool) error
	RecoverExpiredFunc     func(ctx context.Context) (int, error)
	InvalidateCacheCalled  bool
	StartHotReloadFunc     func(ctx context.Context) error
}

func (m *mockAccountPool) PickAvailable(ctx context.Context, configID string) (*models.ADServiceAccount, error) {
	if m.PickAvailableFunc != nil {
		return m.PickAvailableFunc(ctx, configID)
	}
	return nil, errors.New("not implemented")
}
func (m *mockAccountPool) ListAvailable(ctx context.Context, configID string) ([]models.ADServiceAccount, error) {
	if m.ListAvailableFunc != nil {
		return m.ListAvailableFunc(ctx, configID)
	}
	return nil, errors.New("not implemented")
}
func (m *mockAccountPool) ListAll(ctx context.Context, configID string, page, pageSize int, statusFilter *int) ([]models.ADServiceAccount, int64, error) {
	if m.ListAllFunc != nil {
		return m.ListAllFunc(ctx, configID, page, pageSize, statusFilter)
	}
	return []models.ADServiceAccount{}, 0, nil
}
func (m *mockAccountPool) CountByStatus(ctx context.Context, configID string) (int64, int64, int64, int64, error) {
	if m.CountByStatusFunc != nil {
		return m.CountByStatusFunc(ctx, configID)
	}
	return 3, 2, 1, 0, nil
}
func (m *mockAccountPool) PickFirstAvailable(ctx context.Context, configID string) (*models.ADServiceAccount, error) {
	if m.PickFirstAvailableFunc != nil {
		return m.PickFirstAvailableFunc(ctx, configID)
	}
	return nil, errors.New("not implemented")
}
func (m *mockAccountPool) Create(ctx context.Context, account *models.ADServiceAccount) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, account)
	}
	return nil
}
func (m *mockAccountPool) Update(ctx context.Context, account *models.ADServiceAccount) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, account)
	}
	return nil
}
func (m *mockAccountPool) Delete(ctx context.Context, accountID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, accountID)
	}
	return nil
}
func (m *mockAccountPool) MarkSuccess(ctx context.Context, accountID string) error {
	if m.MarkSuccessFunc != nil {
		return m.MarkSuccessFunc(ctx, accountID)
	}
	return nil
}
func (m *mockAccountPool) MarkFailure(ctx context.Context, accountID, reason string) error {
	if m.MarkFailureFunc != nil {
		return m.MarkFailureFunc(ctx, accountID, reason)
	}
	return nil
}
func (m *mockAccountPool) ManualUnlock(ctx context.Context, accountID, operator, reason string) error {
	if m.ManualUnlockFunc != nil {
		return m.ManualUnlockFunc(ctx, accountID, operator, reason)
	}
	return nil
}
func (m *mockAccountPool) SetEnabled(ctx context.Context, accountID string, enabled bool) error {
	if m.SetEnabledFunc != nil {
		return m.SetEnabledFunc(ctx, accountID, enabled)
	}
	return nil
}
func (m *mockAccountPool) RecoverExpiredBreakers(ctx context.Context) (int, error) {
	if m.RecoverExpiredFunc != nil {
		return m.RecoverExpiredFunc(ctx)
	}
	return 0, nil
}
func (m *mockAccountPool) InvalidateCache(configID string) { m.InvalidateCacheCalled = true }
func (m *mockAccountPool) StartHotReload(ctx context.Context) error {
	if m.StartHotReloadFunc != nil {
		return m.StartHotReloadFunc(ctx)
	}
	return nil
}

// setupADAccountCore builds a core backed by sqlite with the pool table migrated.
func setupADAccountCore(t *testing.T, cipher addomainServices.PasswordCipher) (*core.Core, *gorm.DB) {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// Raw DDL (not AutoMigrate): the model's `default:gen_random_uuid()` tag
	// is PG-only and breaks sqlite CREATE TABLE generation.
	require.NoError(t, gdb.Exec(`
		CREATE TABLE sys_ad_service_accounts (
			id TEXT PRIMARY KEY,
			config_id TEXT NOT NULL,
			username TEXT NOT NULL,
			password_ciphertext TEXT NOT NULL,
			status INTEGER NOT NULL DEFAULT 0,
			failure_count INTEGER NOT NULL DEFAULT 0,
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
	c := &core.Core{
		CoreInfra:    &core.CoreInfra{SM4Cipher: cipher},
		CoreServices: &core.CoreServices{OperLogService: nil},
	}
	c.DB = &coredb.Database{DB: gdb}
	return c, gdb
}

func newADAccountHandlerForTest(pool addomainServices.AccountPool, c *core.Core) *ADAccountHandler {
	return NewADAccountHandler(pool, c)
}

func invokeADAccount(t *testing.T, method, path string, body interface{}, handler func(*gin.Context)) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("username", "admin-tester")
	if body != nil {
		b, _ := json.Marshal(body)
		c.Request = httptest.NewRequest(method, path, asReader(b))
	} else {
		c.Request = httptest.NewRequest(method, path, nil)
	}
	c.Request.Header.Set("Content-Type", "application/json")
	handler(c)
	return w
}

// ----------------------------------------------------------------------------
// List
// ----------------------------------------------------------------------------

func TestADAccountHandler_List_Success(t *testing.T) {
	pool := &mockAccountPool{
		ListAllFunc: func(_ context.Context, _ string, page, pageSize int, _ *int) ([]models.ADServiceAccount, int64, error) {
			assert.Equal(t, 1, page)
			assert.Equal(t, 20, pageSize)
			return []models.ADServiceAccount{{ID: "a1"}}, 1, nil
		},
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/list", map[string]interface{}{"configId": "c1"}, h.List)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_List_DefaultsPaging(t *testing.T) {
	pool := &mockAccountPool{
		ListAllFunc: func(_ context.Context, _ string, page, pageSize int, _ *int) ([]models.ADServiceAccount, int64, error) {
			assert.Equal(t, 1, page)
			assert.Equal(t, 20, pageSize)
			return nil, 0, nil
		},
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/list", map[string]interface{}{"configId": "c1", "page": 0, "pageSize": 999}, h.List)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_List_BindError(t *testing.T) {
	h := newADAccountHandlerForTest(&mockAccountPool{}, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/list", map[string]interface{}{}, h.List)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_List_Error(t *testing.T) {
	pool := &mockAccountPool{
		ListAllFunc: func(_ context.Context, _ string, _, _ int, _ *int) ([]models.ADServiceAccount, int64, error) {
			return nil, 0, errors.New("boom")
		},
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/list", map[string]interface{}{"configId": "c1"}, h.List)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// Create
// ----------------------------------------------------------------------------

func TestADAccountHandler_Create_Success(t *testing.T) {
	pool := &mockAccountPool{
		CreateFunc: func(_ context.Context, account *models.ADServiceAccount) error {
			assert.Equal(t, "enc:secret", account.PasswordCiphertext)
			return nil
		},
	}
	coreObj, _ := setupADAccountCore(t, &fakeADPasswordCipher{})
	h := newADAccountHandlerForTest(pool, coreObj)
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/create", map[string]interface{}{
		"configId": "c1", "username": "svc-ad", "password": "secret", "remark": "r",
	}, h.Create)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Create_BindError(t *testing.T) {
	coreObj, _ := setupADAccountCore(t, &fakeADPasswordCipher{})
	h := newADAccountHandlerForTest(&mockAccountPool{}, coreObj)
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/create", map[string]interface{}{"configId": "c1"}, h.Create)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Create_CipherFail(t *testing.T) {
	coreObj, _ := setupADAccountCore(t, &fakeADPasswordCipher{fail: true})
	h := newADAccountHandlerForTest(&mockAccountPool{}, coreObj)
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/create", map[string]interface{}{
		"configId": "c1", "username": "svc-ad", "password": "secret",
	}, h.Create)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Create_PoolError(t *testing.T) {
	pool := &mockAccountPool{
		CreateFunc: func(_ context.Context, _ *models.ADServiceAccount) error { return errors.New("boom") },
	}
	coreObj, _ := setupADAccountCore(t, &fakeADPasswordCipher{})
	h := newADAccountHandlerForTest(pool, coreObj)
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/create", map[string]interface{}{
		"configId": "c1", "username": "svc-ad", "password": "secret",
	}, h.Create)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// Update
// ----------------------------------------------------------------------------

func TestADAccountHandler_Update_Success(t *testing.T) {
	coreObj, gdb := setupADAccountCore(t, &fakeADPasswordCipher{})
	seed := models.ADServiceAccount{ID: "acc-1", ConfigID: "c1", Username: "old", PasswordCiphertext: "enc:old", Status: 0}
	require.NoError(t, gdb.Create(&seed).Error)

	pool := &mockAccountPool{
		UpdateFunc: func(_ context.Context, account *models.ADServiceAccount) error {
			assert.Equal(t, "newname", account.Username)
			assert.Equal(t, "enc:newpwd", account.PasswordCiphertext)
			return nil
		},
	}
	h := newADAccountHandlerForTest(pool, coreObj)
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/update", map[string]interface{}{
		"id": "acc-1", "username": "newname", "password": "newpwd",
	}, h.Update)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Update_NotFound(t *testing.T) {
	coreObj, _ := setupADAccountCore(t, &fakeADPasswordCipher{})
	h := newADAccountHandlerForTest(&mockAccountPool{}, coreObj)
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/update", map[string]interface{}{
		"id": "missing", "username": "x",
	}, h.Update)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Update_BindError(t *testing.T) {
	coreObj, _ := setupADAccountCore(t, &fakeADPasswordCipher{})
	h := newADAccountHandlerForTest(&mockAccountPool{}, coreObj)
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/update", map[string]interface{}{}, h.Update)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Update_CipherFail(t *testing.T) {
	coreObj, gdb := setupADAccountCore(t, &fakeADPasswordCipher{fail: true})
	seed := models.ADServiceAccount{ID: "acc-2", ConfigID: "c1", Username: "old", PasswordCiphertext: "enc:old", Status: 0}
	require.NoError(t, gdb.Create(&seed).Error)

	h := newADAccountHandlerForTest(&mockAccountPool{}, coreObj)
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/update", map[string]interface{}{
		"id": "acc-2", "password": "newpwd",
	}, h.Update)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Update_PoolError(t *testing.T) {
	coreObj, gdb := setupADAccountCore(t, &fakeADPasswordCipher{})
	seed := models.ADServiceAccount{ID: "acc-3", ConfigID: "c1", Username: "old", PasswordCiphertext: "enc:old", Status: 0}
	require.NoError(t, gdb.Create(&seed).Error)

	pool := &mockAccountPool{
		UpdateFunc: func(_ context.Context, _ *models.ADServiceAccount) error { return errors.New("boom") },
	}
	h := newADAccountHandlerForTest(pool, coreObj)
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/update", map[string]interface{}{
		"id": "acc-3", "username": "x",
	}, h.Update)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// Delete / Enable / Disable
// ----------------------------------------------------------------------------

func TestADAccountHandler_Delete_Success(t *testing.T) {
	h := newADAccountHandlerForTest(&mockAccountPool{}, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/delete", map[string]interface{}{"id": "a1"}, h.Delete)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Delete_BindError(t *testing.T) {
	h := newADAccountHandlerForTest(&mockAccountPool{}, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/delete", map[string]interface{}{}, h.Delete)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Delete_NotFound(t *testing.T) {
	pool := &mockAccountPool{
		DeleteFunc: func(_ context.Context, _ string) error { return addomainServices.ErrAccountNotFound },
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/delete", map[string]interface{}{"id": "a1"}, h.Delete)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Delete_Error(t *testing.T) {
	pool := &mockAccountPool{
		DeleteFunc: func(_ context.Context, _ string) error { return errors.New("boom") },
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/delete", map[string]interface{}{"id": "a1"}, h.Delete)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Enable_Success(t *testing.T) {
	pool := &mockAccountPool{
		SetEnabledFunc: func(_ context.Context, accountID string, enabled bool) error {
			assert.True(t, enabled)
			return nil
		},
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/enable", map[string]interface{}{"id": "a1"}, h.Enable)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Enable_BindError(t *testing.T) {
	h := newADAccountHandlerForTest(&mockAccountPool{}, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/enable", map[string]interface{}{}, h.Enable)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Enable_NotFound(t *testing.T) {
	pool := &mockAccountPool{
		SetEnabledFunc: func(_ context.Context, _ string, _ bool) error { return addomainServices.ErrAccountNotFound },
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/enable", map[string]interface{}{"id": "a1"}, h.Enable)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Enable_Error(t *testing.T) {
	pool := &mockAccountPool{
		SetEnabledFunc: func(_ context.Context, _ string, _ bool) error { return errors.New("boom") },
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/enable", map[string]interface{}{"id": "a1"}, h.Enable)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Disable_Success(t *testing.T) {
	pool := &mockAccountPool{
		SetEnabledFunc: func(_ context.Context, accountID string, enabled bool) error {
			assert.False(t, enabled)
			return nil
		},
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/disable", map[string]interface{}{"id": "a1"}, h.Disable)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Disable_BindError(t *testing.T) {
	h := newADAccountHandlerForTest(&mockAccountPool{}, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/disable", map[string]interface{}{}, h.Disable)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Disable_NotFound(t *testing.T) {
	pool := &mockAccountPool{
		SetEnabledFunc: func(_ context.Context, _ string, _ bool) error { return addomainServices.ErrAccountNotFound },
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/disable", map[string]interface{}{"id": "a1"}, h.Disable)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Disable_Error(t *testing.T) {
	pool := &mockAccountPool{
		SetEnabledFunc: func(_ context.Context, _ string, _ bool) error { return errors.New("boom") },
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/disable", map[string]interface{}{"id": "a1"}, h.Disable)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// Unlock
// ----------------------------------------------------------------------------

func TestADAccountHandler_Unlock_Success(t *testing.T) {
	pool := &mockAccountPool{
		ManualUnlockFunc: func(_ context.Context, accountID, operator, reason string) error {
			assert.Equal(t, "admin-tester", operator)
			assert.NotEmpty(t, reason)
			return nil
		},
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/unlock", map[string]interface{}{
		"id": "a1", "reason": "manual unlock after verification",
	}, h.Unlock)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Unlock_BindError(t *testing.T) {
	h := newADAccountHandlerForTest(&mockAccountPool{}, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/unlock", map[string]interface{}{}, h.Unlock)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Unlock_NotFound(t *testing.T) {
	pool := &mockAccountPool{
		ManualUnlockFunc: func(_ context.Context, _, _, _ string) error { return addomainServices.ErrAccountNotFound },
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/unlock", map[string]interface{}{"id": "a1", "reason": "r"}, h.Unlock)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Unlock_InvalidOperator(t *testing.T) {
	pool := &mockAccountPool{
		ManualUnlockFunc: func(_ context.Context, _, _, _ string) error { return addomainServices.ErrInvalidOperator },
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/unlock", map[string]interface{}{"id": "a1", "reason": "r"}, h.Unlock)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Unlock_InvalidReason(t *testing.T) {
	pool := &mockAccountPool{
		ManualUnlockFunc: func(_ context.Context, _, _, _ string) error { return addomainServices.ErrInvalidUnlockReason },
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/unlock", map[string]interface{}{"id": "a1", "reason": "r"}, h.Unlock)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Unlock_Error(t *testing.T) {
	pool := &mockAccountPool{
		ManualUnlockFunc: func(_ context.Context, _, _, _ string) error { return errors.New("boom") },
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/unlock", map[string]interface{}{"id": "a1", "reason": "r"}, h.Unlock)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// Stats
// ----------------------------------------------------------------------------

func TestADAccountHandler_Stats_Success(t *testing.T) {
	pool := &mockAccountPool{
		CountByStatusFunc: func(_ context.Context, _ string) (int64, int64, int64, int64, error) {
			return 5, 3, 1, 1, nil
		},
		PickFirstAvailableFunc: func(_ context.Context, _ string) (*models.ADServiceAccount, error) {
			return &models.ADServiceAccount{ID: "a1", Username: "svc-active"}, nil
		},
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/stats", map[string]interface{}{"configId": "c1"}, h.Stats)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "svc-active")
}

func TestADAccountHandler_Stats_NoActive(t *testing.T) {
	pool := &mockAccountPool{
		PickFirstAvailableFunc: func(_ context.Context, _ string) (*models.ADServiceAccount, error) {
			return nil, errors.New("none")
		},
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/stats", map[string]interface{}{"configId": "c1"}, h.Stats)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Stats_BindError(t *testing.T) {
	h := newADAccountHandlerForTest(&mockAccountPool{}, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/stats", map[string]interface{}{}, h.Stats)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestADAccountHandler_Stats_Error(t *testing.T) {
	pool := &mockAccountPool{
		CountByStatusFunc: func(_ context.Context, _ string) (int64, int64, int64, int64, error) {
			return 0, 0, 0, 0, errors.New("boom")
		},
	}
	h := newADAccountHandlerForTest(pool, &core.Core{CoreInfra: &core.CoreInfra{}, CoreServices: &core.CoreServices{}})
	w := invokeADAccount(t, "POST", "/system/ad-config/accounts/stats", map[string]interface{}{"configId": "c1"}, h.Stats)
	assert.NotEqual(t, http.StatusOK, w.Code)
}
