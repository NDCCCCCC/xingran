package security

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// setupSecurityTestDB creates an in-memory SQLite database for testing.
// Tables are created manually to avoid PostgreSQL-specific syntax.
func setupSecurityTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "Failed to open test database")

	// Create sys_user table
	// AD 列名须与 User 模型 gorm tag 一致: ad_dn / ad_ou_dn / ad_synced_at (见 user.go:34-36)
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_user (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 1,
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			salt TEXT NOT NULL DEFAULT '',
			nickname TEXT,
			employee_no TEXT,
			email TEXT,
			phone TEXT,
			avatar TEXT,
			gender INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0,
			dept_id TEXT,
			dept_name TEXT,
			login_ip TEXT,
			login_time DATETIME,
			pwd_update_time DATETIME,
			pwd_expire_days INTEGER DEFAULT 90,
			init_flag INTEGER DEFAULT 0,
			remark TEXT DEFAULT '',
			auth_source TEXT NOT NULL DEFAULT 'local',
			ad_username TEXT,
			ad_dn TEXT,
			ad_ou_dn TEXT,
			ad_synced_at DATETIME
		)
	`).Error
	require.NoError(t, err, "Failed to create sys_user table")

	// Create sys_config table
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_config (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 1,
			config_name TEXT,
			config_key TEXT NOT NULL,
			config_value TEXT NOT NULL,
			config_type TEXT DEFAULT 'Y',
			is_system INTEGER DEFAULT 0,
			remark TEXT DEFAULT ''
		)
	`).Error
	require.NoError(t, err, "Failed to create sys_config table")

	// Create sys_ad_config table
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_ad_config (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 1,
			config_name TEXT NOT NULL,
			server_address TEXT NOT NULL,
			server_port INTEGER DEFAULT 389,
			domain_name TEXT NOT NULL,
			base_dn TEXT NOT NULL,
			admin_username TEXT NOT NULL,
			admin_password TEXT NOT NULL,
			use_ssl INTEGER DEFAULT 0,
			use_tls INTEGER DEFAULT 0,
			sync_enabled INTEGER DEFAULT 1,
			sync_interval INTEGER DEFAULT 3600,
			member_ou_dn TEXT,
			last_sync_at DATETIME,
			status INTEGER DEFAULT 0
		)
	`).Error
	require.NoError(t, err, "Failed to create sys_ad_config table")

	return db
}

// createTestUser creates a test user in the database with the given username and hashed password.
func createTestUser(t *testing.T, db *gorm.DB, username, plainPassword string, status models.UserStatus) *models.User {
	t.Helper()
	pwdMgr := NewPasswordManager(nil)
	hashedPassword, err := pwdMgr.HashPassword(plainPassword)
	require.NoError(t, err, "Failed to hash test user password")

	user := &models.User{
		BaseModel: models.BaseModel{ID: username + "-test-id"},
		Username:  username,
		Password:  hashedPassword,
		Status:    status,
	}
	require.NoError(t, db.Create(user).Error, "Failed to create test user")
	return user
}

// ========== Integration Tests: AuthStrategyFactory ==========

func TestIntegration_AuthStrategyFactory_GetLocalAuthenticator(t *testing.T) {
	db := setupSecurityTestDB(t)
	pwdMgr := NewPasswordManager(nil)
	factory := NewAuthStrategyFactory(db, pwdMgr)

	auth, err := factory.GetAuthenticator("local")
	assert.NoError(t, err, "GetAuthenticator('local') should not error")
	assert.Equal(t, "local", auth.Name(), "Authenticator name should be 'local'")

	// Verify it is a LocalAuthenticator
	_, ok := auth.(*LocalAuthenticator)
	assert.True(t, ok, "Should return a LocalAuthenticator instance")
}

func TestIntegration_AuthStrategyFactory_InvalidMode(t *testing.T) {
	db := setupSecurityTestDB(t)
	pwdMgr := NewPasswordManager(nil)
	factory := NewAuthStrategyFactory(db, pwdMgr)

	_, err := factory.GetAuthenticator("invalid")
	assert.Error(t, err, "GetAuthenticator('invalid') should return error")
	assert.Contains(t, err.Error(), "不支持的认证模式", "Error should mention unsupported mode")
}

func TestIntegration_AuthStrategyFactory_GetADAuthenticator_NoConfig(t *testing.T) {
	db := setupSecurityTestDB(t)
	pwdMgr := NewPasswordManager(nil)
	factory := NewAuthStrategyFactory(db, pwdMgr)

	// No AD config in database, should fail
	_, err := factory.GetAuthenticator("ad")
	assert.Error(t, err, "GetAuthenticator('ad') should fail when no AD config exists")
}

func TestIntegration_AuthStrategyFactory_GetADAuthenticator_WithConfig(t *testing.T) {
	db := setupSecurityTestDB(t)
	pwdMgr := NewPasswordManager(nil)
	factory := NewAuthStrategyFactory(db, pwdMgr)

	// Insert AD config
	adConfig := &models.ADConfig{
		BaseModel:     models.BaseModel{ID: "test-ad-config-id"},
		ConfigName:    "Test AD",
		ServerAddress: "192.168.1.100",
		ServerPort:    389,
		DomainName:    "test.com",
		BaseDN:        "dc=test,dc=com",
		AdminUsername: "admin",
		AdminPassword: "encrypted_password",
		Status:        0, // Enabled
	}
	require.NoError(t, db.Create(adConfig).Error, "Failed to create test AD config")

	auth, err := factory.GetAuthenticator("ad")
	assert.NoError(t, err, "GetAuthenticator('ad') should succeed with AD config")
	assert.Equal(t, "ad", auth.Name(), "Authenticator name should be 'ad'")

	_, ok := auth.(*ADAuthenticator)
	assert.True(t, ok, "Should return an ADAuthenticator instance")
}

func TestIntegration_AuthStrategyFactory_GetHybridAuthenticator_WithConfig(t *testing.T) {
	db := setupSecurityTestDB(t)
	pwdMgr := NewPasswordManager(nil)
	factory := NewAuthStrategyFactory(db, pwdMgr)

	// Insert AD config
	adConfig := &models.ADConfig{
		BaseModel:     models.BaseModel{ID: "test-hybrid-ad-config"},
		ConfigName:    "Hybrid Test AD",
		ServerAddress: "192.168.1.100",
		ServerPort:    389,
		DomainName:    "test.com",
		BaseDN:        "dc=test,dc=com",
		AdminUsername: "admin",
		AdminPassword: "encrypted_password",
		Status:        0,
	}
	require.NoError(t, db.Create(adConfig).Error)

	auth, err := factory.GetAuthenticator("hybrid")
	assert.NoError(t, err, "GetAuthenticator('hybrid') should succeed with AD config")
	assert.Equal(t, "hybrid", auth.Name(), "Authenticator name should be 'hybrid'")

	_, ok := auth.(*HybridAuthenticator)
	assert.True(t, ok, "Should return a HybridAuthenticator instance")
}

// ========== Integration Tests: LocalAuthenticator ==========

func TestIntegration_LocalAuthenticator_ValidCredentials(t *testing.T) {
	db := setupSecurityTestDB(t)
	pwdMgr := NewPasswordManager(nil)

	// Create test user
	createTestUser(t, db, "testuser", "password123", models.UserStatusEnabled)

	auth := NewLocalAuthenticator(db, pwdMgr)
	req := &AuthRequest{
		Username: "testuser",
		Password: "password123",
		IP:       "127.0.0.1",
	}

	result, err := auth.Authenticate(context.Background(), req)
	assert.NoError(t, err, "Authenticate should succeed with valid credentials")
	assert.NotNil(t, result, "Result should not be nil")
	assert.Equal(t, "local", result.AuthSource, "AuthSource should be 'local'")
	assert.NotNil(t, result.User, "User should not be nil")
	assert.Equal(t, "testuser", result.User.Username, "Username should match")
	assert.False(t, result.NeedsSync, "Local auth should not need sync")
}

func TestIntegration_LocalAuthenticator_InvalidPassword(t *testing.T) {
	db := setupSecurityTestDB(t)
	pwdMgr := NewPasswordManager(nil)

	createTestUser(t, db, "testuser2", "password123", models.UserStatusEnabled)

	auth := NewLocalAuthenticator(db, pwdMgr)
	req := &AuthRequest{
		Username: "testuser2",
		Password: "wrongpassword",
		IP:       "127.0.0.1",
	}

	result, err := auth.Authenticate(context.Background(), req)
	assert.Error(t, err, "Authenticate should fail with wrong password")
	assert.Nil(t, result, "Result should be nil on failure")
	assert.Equal(t, ErrInvalidCredentials, err, "Error should be ErrInvalidCredentials")
}

func TestIntegration_LocalAuthenticator_UserNotFound(t *testing.T) {
	db := setupSecurityTestDB(t)
	pwdMgr := NewPasswordManager(nil)

	auth := NewLocalAuthenticator(db, pwdMgr)
	req := &AuthRequest{
		Username: "nonexistent",
		Password: "password123",
		IP:       "127.0.0.1",
	}

	result, err := auth.Authenticate(context.Background(), req)
	assert.Error(t, err, "Authenticate should fail for non-existent user")
	assert.Nil(t, result, "Result should be nil")
	assert.Equal(t, ErrUserNotFound, err, "Error should be ErrUserNotFound")
}

func TestIntegration_LocalAuthenticator_UserDisabled(t *testing.T) {
	db := setupSecurityTestDB(t)
	pwdMgr := NewPasswordManager(nil)

	createTestUser(t, db, "disableduser", "password123", models.UserStatusDisabled)

	auth := NewLocalAuthenticator(db, pwdMgr)
	req := &AuthRequest{
		Username: "disableduser",
		Password: "password123",
		IP:       "127.0.0.1",
	}

	result, err := auth.Authenticate(context.Background(), req)
	assert.Error(t, err, "Authenticate should fail for disabled user")
	assert.Nil(t, result, "Result should be nil")
	assert.Equal(t, ErrUserDisabled, err, "Error should be ErrUserDisabled")
}

// ========== Integration Tests: GetDefaultAuthMode ==========

func TestIntegration_AuthStrategyFactory_GetDefaultAuthMode_NoConfig(t *testing.T) {
	db := setupSecurityTestDB(t)
	pwdMgr := NewPasswordManager(nil)
	factory := NewAuthStrategyFactory(db, pwdMgr)

	mode, err := factory.GetDefaultAuthMode(context.Background())
	assert.NoError(t, err, "GetDefaultAuthMode should not error when no config exists")
	assert.Equal(t, "local", mode, "Default mode should be 'local' when no config")
}

func TestIntegration_AuthStrategyFactory_GetDefaultAuthMode_WithConfig(t *testing.T) {
	db := setupSecurityTestDB(t)
	pwdMgr := NewPasswordManager(nil)
	factory := NewAuthStrategyFactory(db, pwdMgr)

	// Insert auth mode config
	config := &models.Config{
		BaseModel:   models.BaseModel{ID: "test-config-id"},
		ConfigKey:   "sys.auth.default.mode",
		ConfigValue: "hybrid",
	}
	require.NoError(t, db.Create(config).Error, "Failed to create test config")

	mode, err := factory.GetDefaultAuthMode(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "hybrid", mode, "Should return configured mode")
}

func TestIntegration_AuthStrategyFactory_GetDefaultAuthMode_InvalidValue(t *testing.T) {
	db := setupSecurityTestDB(t)
	pwdMgr := NewPasswordManager(nil)
	factory := NewAuthStrategyFactory(db, pwdMgr)

	config := &models.Config{
		BaseModel:   models.BaseModel{ID: "test-invalid-config-id"},
		ConfigKey:   "sys.auth.default.mode",
		ConfigValue: "invalid_mode",
	}
	require.NoError(t, db.Create(config).Error)

	mode, err := factory.GetDefaultAuthMode(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "local", mode, "Should fall back to 'local' for invalid mode value")
}

// ========== Integration Tests: HybridAuthenticator ==========

func TestIntegration_HybridAuthenticator_LocalSuccess(t *testing.T) {
	// When local auth succeeds, hybrid should return local result.
	// Since HybridAuthenticator takes concrete types, we test via the factory with real DB.
	db := setupSecurityTestDB(t)
	pwdMgr := NewPasswordManager(nil)

	// Create user so local auth succeeds
	createTestUser(t, db, "hybriduser", "hybridpass", models.UserStatusEnabled)

	// Insert AD config for hybrid mode
	adConfig := &models.ADConfig{
		BaseModel:     models.BaseModel{ID: "hybrid-test-ad-id"},
		ConfigName:    "Hybrid Test",
		ServerAddress: "192.168.1.100",
		ServerPort:    389,
		DomainName:    "test.com",
		BaseDN:        "dc=test,dc=com",
		AdminUsername: "admin",
		AdminPassword: "encrypted",
		Status:        0,
	}
	require.NoError(t, db.Create(adConfig).Error)

	factory := NewAuthStrategyFactory(db, pwdMgr)
	auth, err := factory.GetAuthenticator("hybrid")
	require.NoError(t, err)

	result, err := auth.Authenticate(context.Background(), &AuthRequest{
		Username: "hybriduser",
		Password: "hybridpass",
		IP:       "127.0.0.1",
	})

	assert.NoError(t, err, "Hybrid auth should succeed when local auth succeeds")
	assert.Equal(t, "local", result.AuthSource, "Should use local auth source")
	assert.False(t, result.NeedsSync, "Local auth should not need sync")
}

func TestIntegration_HybridAuthenticator_FallbackToAD(t *testing.T) {
	// When local auth fails (user not found), should attempt AD.
	// Without a real LDAP server, AD auth will fail, so we verify the hybrid
	// authenticator returns an error since both local and AD fail.

	db := setupSecurityTestDB(t)
	pwdMgr := NewPasswordManager(nil)

	// Insert AD config so hybrid authenticator can be created
	adConfig := &models.ADConfig{
		BaseModel:     models.BaseModel{ID: "fallback-test-ad-id"},
		ConfigName:    "Fallback Test",
		ServerAddress: "192.168.1.100",
		ServerPort:    389,
		DomainName:    "test.com",
		BaseDN:        "dc=test,dc=com",
		AdminUsername: "admin",
		AdminPassword: "encrypted",
		Status:        0,
	}
	require.NoError(t, db.Create(adConfig).Error)

	factory := NewAuthStrategyFactory(db, pwdMgr)
	auth, err := factory.GetAuthenticator("hybrid")
	require.NoError(t, err)

	// User does not exist locally, AD server unreachable -- both should fail
	result, err := auth.Authenticate(context.Background(), &AuthRequest{
		Username: "nonlocaluser",
		Password: "somepassword",
		IP:       "127.0.0.1",
	})

	// Hybrid should fail: local returns ErrUserNotFound, AD returns connection error
	// The hybrid returns the local error (first attempt)
	assert.Error(t, err, "Hybrid should fail when both local and AD fail")
	assert.Nil(t, result, "Result should be nil on failure")
}

// ========== Integration Tests: UserSyncer Interface ==========

// mockUserSyncer implements UserSyncer for testing
type mockUserSyncer struct {
	syncFunc func(ctx context.Context, adUserInfo *ADUserInfo, defaultRoleID string) (*SyncedUser, error)
}

func (m *mockUserSyncer) SyncADUser(ctx context.Context, adUserInfo *ADUserInfo, defaultRoleID string) (*SyncedUser, error) {
	if m.syncFunc != nil {
		return m.syncFunc(ctx, adUserInfo, defaultRoleID)
	}
	return &SyncedUser{
		ID:       "synced-user-id",
		Username: adUserInfo.Username,
		Status:   0,
	}, nil
}

func TestIntegration_AuthStrategyFactory_SetUserSyncer(t *testing.T) {
	db := setupSecurityTestDB(t)
	pwdMgr := NewPasswordManager(nil)
	factory := NewAuthStrategyFactory(db, pwdMgr)

	syncer := &mockUserSyncer{}
	factory.SetUserSyncer(syncer)

	// Create AD config
	adConfig := &models.ADConfig{
		BaseModel:     models.BaseModel{ID: "syncer-test-ad-id"},
		ConfigName:    "Syncer Test",
		ServerAddress: "192.168.1.100",
		ServerPort:    389,
		DomainName:    "test.com",
		BaseDN:        "dc=test,dc=com",
		AdminUsername: "admin",
		AdminPassword: "encrypted",
		Status:        0,
	}
	require.NoError(t, db.Create(adConfig).Error)

	// Get AD authenticator and verify syncer is set
	auth, err := factory.GetAuthenticator("ad")
	require.NoError(t, err)

	adAuth, ok := auth.(*ADAuthenticator)
	require.True(t, ok, "Should be ADAuthenticator")
	assert.NotNil(t, adAuth.userSyncer, "userSyncer should be set on ADAuthenticator")
}

func TestIntegration_UserSyncer_InterfaceContract(t *testing.T) {
	// Verify mockUserSyncer satisfies the UserSyncer interface at compile time
	var _ UserSyncer = (*mockUserSyncer)(nil)

	syncer := &mockUserSyncer{}
	adUserInfo := &ADUserInfo{
		Username:    "testuser",
		DisplayName: "Test User",
		Email:       "test@example.com",
	}

	result, err := syncer.SyncADUser(context.Background(), adUserInfo, "")
	assert.NoError(t, err)
	assert.Equal(t, "synced-user-id", result.ID)
	assert.Equal(t, "testuser", result.Username)
}

func TestIntegration_UserSyncer_CustomBehavior(t *testing.T) {
	syncer := &mockUserSyncer{
		syncFunc: func(ctx context.Context, adUserInfo *ADUserInfo, defaultRoleID string) (*SyncedUser, error) {
			return &SyncedUser{
				ID:       "custom-synced-id",
				Username: adUserInfo.Username,
				Nickname: strPtr(adUserInfo.DisplayName),
				Email:    strPtr(adUserInfo.Email),
				Status:   0,
				Roles:    []string{defaultRoleID},
			}, nil
		},
	}

	adUserInfo := &ADUserInfo{
		Username:    "aduser",
		DisplayName: "AD User",
		Email:       "aduser@example.com",
	}

	result, err := syncer.SyncADUser(context.Background(), adUserInfo, "role-123")
	assert.NoError(t, err)
	assert.Equal(t, "custom-synced-id", result.ID)
	assert.Equal(t, "aduser", result.Username)
	assert.NotNil(t, result.Nickname)
	assert.Equal(t, "AD User", *result.Nickname)
	assert.Equal(t, []string{"role-123"}, result.Roles)
}

// ========== Integration Tests: Error Types ==========

func TestIntegration_AuthErrors_AreDistinct(t *testing.T) {
	// Verify that all auth errors are distinct and properly defined
	errors := []error{
		ErrUserNotFound,
		ErrInvalidCredentials,
		ErrUserDisabled,
		ErrADConfigNotFound,
		ErrADConnectionFailed,
	}

	for i, e1 := range errors {
		for j, e2 := range errors {
			if i != j {
				assert.NotEqual(t, e1, e2, "Error %d should differ from error %d", i, j)
			}
		}
	}

	// Verify error messages are in Chinese (as defined)
	assert.Contains(t, ErrUserNotFound.Error(), "用户不存在")
	assert.Contains(t, ErrInvalidCredentials.Error(), "用户名或密码错误")
	assert.Contains(t, ErrUserDisabled.Error(), "用户已被禁用")
	assert.Contains(t, ErrADConfigNotFound.Error(), "AD配置不存在")
	assert.Contains(t, ErrADConnectionFailed.Error(), "AD连接失败")
}

// ========== Integration Tests: AuthRequest and AuthResult ==========

func TestIntegration_AuthRequest_Fields(t *testing.T) {
	req := &AuthRequest{
		Username: "testuser",
		Password: "testpass",
		IP:       "10.0.0.1",
	}
	assert.Equal(t, "testuser", req.Username)
	assert.Equal(t, "testpass", req.Password)
	assert.Equal(t, "10.0.0.1", req.IP)
}

func TestIntegration_AuthResult_Fields(t *testing.T) {
	result := &AuthResult{
		User: &UserResult{
			ID:       "user-id-1",
			Username: "testuser",
			Status:   0,
			Roles:    []string{"admin", "user"},
		},
		AuthSource: "local",
		NeedsSync:  false,
	}
	assert.Equal(t, "user-id-1", result.User.ID)
	assert.Equal(t, "local", result.AuthSource)
	assert.False(t, result.NeedsSync)
	assert.Len(t, result.User.Roles, 2)
}

// strPtr is a helper to create a string pointer
func strPtr(s string) *string {
	return &s
}
