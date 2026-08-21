package system

// =====================================================================
// apikey_service_extra_test.go — additional tests extending apikey_service_test.go
// Per Plan 72-11 Task 4. Existing apikey_service_test.go (PRESERVED).
// This file adds tests for paths not covered by the existing test suite.
// =====================================================================

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
)

// TC1: CreateAPIKey - exceeds MaxKeysPerUser
func TestAPIKeyService_Create_ExceedsMaxKeysPerUser(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	// Insert MaxKeysPerUser keys directly
	for i := 0; i < MaxKeysPerUser; i++ {
		_, _ = createTestAPIKey(t, db, user.ID, true)
	}

	req := &requests.CreateAPIKeyRequest{
		Name:   "over-limit",
		Scopes: []string{"read"},
	}
	_, err := svc.CreateAPIKey(ctx, user.ID, req)
	assert.Error(t, err, "should fail when MaxKeysPerUser reached")
	assert.Contains(t, err.Error(), "最大密钥数量限制")
	cleanupTestData(t, db)
}

// TC2: CreateAPIKey - invalid scope rejected
func TestAPIKeyService_Create_InvalidScope(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	req := &requests.CreateAPIKeyRequest{
		Name:   "invalid-scope",
		Scopes: []string{"super-admin"},
	}
	_, err := svc.CreateAPIKey(ctx, user.ID, req)
	assert.Error(t, err, "invalid scope should error")
	cleanupTestData(t, db)
}

// TC3: CreateAPIKey - invalid expiresAt format
func TestAPIKeyService_Create_InvalidExpiresAt(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	invalidTime := "not-rfc3339"
	req := &requests.CreateAPIKeyRequest{
		Name:      "bad-time",
		Scopes:    []string{"read"},
		ExpiresAt: &invalidTime,
	}
	_, err := svc.CreateAPIKey(ctx, user.ID, req)
	assert.Error(t, err, "invalid expiresAt format should error")
	cleanupTestData(t, db)
}

// TC4: CreateAPIKey - successful creation with valid expiresAt
func TestAPIKeyService_Create_ValidExpiresAt(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	futureTime := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	req := &requests.CreateAPIKeyRequest{
		Name:      "with-expiry",
		Scopes:    []string{"read"},
		ExpiresAt: &futureTime,
	}
	key, err := svc.CreateAPIKey(ctx, user.ID, req)
	require.NoError(t, err)
	require.NotNil(t, key)
	assert.NotEmpty(t, *key)
	cleanupTestData(t, db)
}

// TC5: CreateAPIKey - empty expiresAt should succeed
func TestAPIKeyService_Create_EmptyExpiresAt(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	emptyTime := ""
	req := &requests.CreateAPIKeyRequest{
		Name:      "no-expiry",
		Scopes:    []string{"read"},
		ExpiresAt: &emptyTime,
	}
	key, err := svc.CreateAPIKey(ctx, user.ID, req)
	require.NoError(t, err)
	require.NotNil(t, key)
	cleanupTestData(t, db)
}

// TC6: ValidateAPIKey - invalid format
func TestAPIKeyService_ValidateAPIKey_InvalidFormat(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()

	_, err := svc.ValidateAPIKey(ctx, "not-a-valid-key")
	assert.Error(t, err)
	cleanupTestData(t, db)
}

// TC7: ValidateAPIKey - not found
func TestAPIKeyService_ValidateAPIKey_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()

	// Use a well-formatted but non-existent key
	key := fmt.Sprintf("rec_%016x%048x", 0, 0)
	_, err := svc.ValidateAPIKey(ctx, key)
	assert.Error(t, err)
	cleanupTestData(t, db)
}

// TC8: ValidateAPIKey - expired key
func TestAPIKeyService_ValidateAPIKey_Expired(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	apiKey, plainKey := createTestAPIKey(t, db, user.ID, true)

	// Set expired time
	pastTime := time.Now().Add(-1 * time.Hour)
	require.NoError(t, db.Exec("UPDATE sys_api_keys SET expires_at = ? WHERE id = ?",
		pastTime, apiKey.ID).Error)

	_, err := svc.ValidateAPIKey(ctx, plainKey)
	assert.Error(t, err, "expired key should error")
	cleanupTestData(t, db)
}

// TC9: ValidateAPIKey - successful
func TestAPIKeyService_ValidateAPIKey_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	_, plainKey := createTestAPIKey(t, db, user.ID, true)

	validated, err := svc.ValidateAPIKey(ctx, plainKey)
	require.NoError(t, err)
	require.NotNil(t, validated)
	assert.Equal(t, plainKey[:12], validated.KeyPrefix)
	cleanupTestData(t, db)
}

// TC10: ListAPIKeys - filter by status
func TestAPIKeyService_List_FilterByStatus(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	createTestAPIKey(t, db, user.ID, true)
	createTestAPIKey(t, db, user.ID, true)
	createTestAPIKey(t, db, user.ID, false)

	active := true
	params := requests.DefaultListAPIKeysParams()
	params.Status = &active
	result, err := svc.ListAPIKeys(ctx, user.ID, params)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.Total, int64(2))
	cleanupTestData(t, db)
}

// TC11: ListAPIKeys - filter by keyword
func TestAPIKeyService_List_FilterByKeyword(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	_, _ = createTestAPIKey(t, db, user.ID, true)

	keyword := "Test"
	params := requests.DefaultListAPIKeysParams()
	params.Keyword = &keyword
	result, err := svc.ListAPIKeys(ctx, user.ID, params)
	require.NoError(t, err)
	assert.Greater(t, result.Total, int64(0))
	cleanupTestData(t, db)
}

// TC12: ListAPIKeys - empty for user with no keys
func TestAPIKeyService_List_EmptyForUser(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()

	result, err := svc.ListAPIKeys(ctx, "no-such-user", requests.DefaultListAPIKeysParams())
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)
	cleanupTestData(t, db)
}

// TC13: ToggleAPIKeyStatus - toggles correctly
func TestAPIKeyService_ToggleStatus_Toggle(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	apiKey, _ := createTestAPIKey(t, db, user.ID, true)

	// Toggle: active → inactive
	require.NoError(t, svc.ToggleAPIKeyStatus(ctx, apiKey.ID))

	var status int
	require.NoError(t, db.Raw("SELECT is_active FROM sys_api_keys WHERE id = ?", apiKey.ID).Scan(&status).Error)
	assert.Equal(t, 0, status)

	// Toggle back
	require.NoError(t, svc.ToggleAPIKeyStatus(ctx, apiKey.ID))
	require.NoError(t, db.Raw("SELECT is_active FROM sys_api_keys WHERE id = ?", apiKey.ID).Scan(&status).Error)
	assert.Equal(t, 1, status)
	cleanupTestData(t, db)
}

// TC14: UpdateAPIKey - update only IsActive
func TestAPIKeyService_UpdateAPIKey_IsActiveOnly(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	apiKey, _ := createTestAPIKey(t, db, user.ID, true)

	inactive := false
	req := &requests.UpdateAPIKeyRequest{
		ID:       apiKey.ID,
		IsActive: &inactive,
	}
	require.NoError(t, svc.UpdateAPIKey(ctx, apiKey.ID, req))

	var status int
	require.NoError(t, db.Raw("SELECT is_active FROM sys_api_keys WHERE id = ?", apiKey.ID).Scan(&status).Error)
	assert.Equal(t, 0, status)
	cleanupTestData(t, db)
}

// TC15: GetAPIKey - not found
func TestAPIKeyService_GetAPIKey_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()

	_, err := svc.GetAPIKey(ctx, uuid.NewString())
	assert.Error(t, err)
	cleanupTestData(t, db)
}

// TC16: GetAPIKey - success
func TestAPIKeyService_GetAPIKey_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	apiKey, _ := createTestAPIKey(t, db, user.ID, true)

	got, err := svc.GetAPIKey(ctx, apiKey.ID)
	require.NoError(t, err)
	assert.Equal(t, apiKey.Name, got.Name)
	cleanupTestData(t, db)
}

// TC17: DeleteAPIKey - not found
func TestAPIKeyService_DeleteAPIKey_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()

	err := svc.DeleteAPIKey(ctx, uuid.NewString())
	assert.Error(t, err)
	cleanupTestData(t, db)
}

// TC18: validateScopes - all valid scopes
func TestAPIKeyService_ValidateScopes(t *testing.T) {
	assert.NoError(t, validateScopes([]string{APIKeyScopeRead}))
	assert.NoError(t, validateScopes([]string{APIKeyScopeWrite}))
	assert.NoError(t, validateScopes([]string{APIKeyScopeAdmin}))
	assert.NoError(t, validateScopes([]string{APIKeyScopeRead, APIKeyScopeWrite}))
}

// TC19: validateScopes - empty scopes allowed
func TestAPIKeyService_ValidateScopes_Empty(t *testing.T) {
	assert.NoError(t, validateScopes([]string{}))
	assert.NoError(t, validateScopes(nil))
}

// TC20: isValidKeyFormat - valid formats
func TestAPIKeyService_IsValidKeyFormat_Valid(t *testing.T) {
	validKey := fmt.Sprintf("rec_%016x%048x", 0, 0)
	assert.True(t, isValidKeyFormat(validKey))
}

// TC21: isValidKeyFormat - invalid formats
func TestAPIKeyService_IsValidKeyFormat_Invalid(t *testing.T) {
	assert.False(t, isValidKeyFormat(""))
	assert.False(t, isValidKeyFormat("not-a-key"))
	assert.False(t, isValidKeyFormat("rec_short"))
	assert.False(t, isValidKeyFormat(fmt.Sprintf("xxx_%016x%048x", 0, 0)), "wrong prefix")
}

// TC22: isKeyExpired - expired
func TestAPIKeyService_IsKeyExpired_Expired(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	assert.True(t, isKeyExpired(&past))
}

// TC23: isKeyExpired - not expired
func TestAPIKeyService_IsKeyExpired_Future(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	assert.False(t, isKeyExpired(&future))
}

// TC24: isKeyExpired - nil expiry never expires
func TestAPIKeyService_IsKeyExpired_Nil(t *testing.T) {
	assert.False(t, isKeyExpired(nil))
}

// TC25: ListUsageLogs - empty
func TestAPIKeyService_ListUsageLogs_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	apiKey, _ := createTestAPIKey(t, db, user.ID, true)

	result, err := svc.ListUsageLogs(ctx, ListUsageLogsParams{
		APIKeyID: apiKey.ID,
		Current:  1,
		PageSize: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)
	cleanupTestData(t, db)
}

// TC26: GetUsageLogSummary - empty
func TestAPIKeyService_GetUsageLogSummary_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	apiKey, _ := createTestAPIKey(t, db, user.ID, true)

	summary, err := svc.GetUsageLogSummary(ctx, apiKey.ID)
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, int64(0), summary.TotalRequests)
	cleanupTestData(t, db)
}

// TC27: ToggleAPIKeyStatus - not found
func TestAPIKeyService_ToggleAPIKeyStatus_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()

	err := svc.ToggleAPIKeyStatus(ctx, uuid.NewString())
	assert.Error(t, err)
	cleanupTestData(t, db)
}

// TC28: UpdateAPIKey - not found
func TestAPIKeyService_UpdateAPIKey_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()

	name := "x"
	err := svc.UpdateAPIKey(ctx, uuid.NewString(), &requests.UpdateAPIKeyRequest{Name: &name})
	assert.Error(t, err)
	cleanupTestData(t, db)
}

// TC29: UpdateAPIKey - clear expires_at by passing empty string
func TestAPIKeyService_UpdateAPIKey_ClearExpiresAt(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	apiKey, _ := createTestAPIKey(t, db, user.ID, true)

	// First set expiry
	future := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	require.NoError(t, svc.UpdateAPIKey(ctx, apiKey.ID, &requests.UpdateAPIKeyRequest{ExpiresAt: &future}))

	// Now clear it
	empty := ""
	require.NoError(t, svc.UpdateAPIKey(ctx, apiKey.ID, &requests.UpdateAPIKeyRequest{ExpiresAt: &empty}))

	// Verify expires_at is NULL via row count of non-null rows
	var nullCount int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM sys_api_keys WHERE id = ? AND expires_at IS NULL", apiKey.ID).Scan(&nullCount).Error)
	assert.Equal(t, int64(1), nullCount)
	cleanupTestData(t, db)
}

// TC30: CreateAPIKey - with all options
func TestAPIKeyService_Create_AllOptions(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	desc := "full options test"
	req := &requests.CreateAPIKeyRequest{
		Name:         "full-options",
		Description:  &desc,
		Scopes:       []string{"read", "write", "admin"},
		InheritPerms: true,
		IPWhitelist:  []string{"192.168.1.0/24", "10.0.0.1"},
	}
	key, err := svc.CreateAPIKey(ctx, user.ID, req)
	require.NoError(t, err)
	require.NotNil(t, key)
	assert.True(t, len(*key) > 0)

	// Verify stored values
	var name, description string
	require.NoError(t, db.Raw("SELECT name, description FROM sys_api_keys WHERE name = ?", "full-options").
		Row().Scan(&name, &description))
	assert.Equal(t, "full-options", name)
	assert.Equal(t, "full options test", description)
	cleanupTestData(t, db)
}

// TC31: ListAPIKeys - filter by scope (SQLite json_each)
func TestAPIKeyService_List_FilterByScope(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	createTestAPIKey(t, db, user.ID, true)

	scope := APIKeyScopeRead
	params := requests.DefaultListAPIKeysParams()
	params.Scope = &scope
	result, err := svc.ListAPIKeys(ctx, user.ID, params)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.Total, int64(1))
	cleanupTestData(t, db)
}

// TC32: ListAPIKeys - service error when no table
func TestAPIKeyService_List_DroppedTable(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	require.NoError(t, db.Exec("DROP TABLE sys_api_keys").Error)

	_, err := svc.ListAPIKeys(ctx, "user-1", requests.DefaultListAPIKeysParams())
	assert.Error(t, err)
	cleanupTestData(t, db)
}

// TC33: CreateAPIKey - DB error when user table missing
func TestAPIKeyService_Create_UserTableMissing(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	require.NoError(t, db.Exec("DROP TABLE sys_user").Error)

	req := &requests.CreateAPIKeyRequest{
		Name:   "x",
		Scopes: []string{"read"},
	}
	_, err := svc.CreateAPIKey(ctx, uuid.NewString(), req)
	assert.Error(t, err)
	cleanupTestData(t, db)
}

// TC34: ValidateAPIKey - inactive key rejected
func TestAPIKeyService_ValidateAPIKey_Inactive(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	_, plainKey := createTestAPIKey(t, db, user.ID, false)

	_, err := svc.ValidateAPIKey(ctx, plainKey)
	assert.Error(t, err, "inactive key should fail validation")
	cleanupTestData(t, db)
}

// TC35: Models.APIKey struct has expected fields (compile-time)
func TestAPIKeyService_ModelsStructFields(t *testing.T) {
	key := models.APIKey{
		Name:        "test",
		KeyHash:     "hash",
		Salt:        "salt",
		KeyPrefix:   "rec_12345678",
		Scopes:      []string{"read"},
		Description: stringPtr("desc"),
	}
	assert.NotEmpty(t, key.Name)
	assert.NotEmpty(t, key.KeyHash)
	assert.NotEmpty(t, key.KeyPrefix)
}

func stringPtrAPIKey(s string) *string { return &s }