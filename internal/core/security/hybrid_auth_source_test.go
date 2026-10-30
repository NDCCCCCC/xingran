package security

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// setupAuthSourceTestDB 创建独立的内存 SQLite 测试库
// 自包含实现（不依赖其它测试文件的 helper），列定义与 models.User 保持一致，
// 含 AD 同步补充字段 mobile / title / ad_config_id
func setupAuthSourceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "打开测试数据库失败")

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
			ad_synced_at DATETIME,
			mobile TEXT,
			title TEXT,
			ad_config_id TEXT
		)
	`).Error
	require.NoError(t, err, "创建 sys_user 表失败")

	// sys_ad_config 表：工厂构造 hybrid 认证器时需要读取启用的 AD 配置
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
	require.NoError(t, err, "创建 sys_ad_config 表失败")

	// sys_config 表：工厂读取 sys.auth.ad.config_id 等配置
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
	require.NoError(t, err, "创建 sys_config 表失败")

	return db
}

// createUserWithAuthSource 在测试库中创建一个指定认证来源（auth_source）的账号
// plainPassword 为明文密码，落库前用 SM3 哈希
func createUserWithAuthSource(t *testing.T, db *gorm.DB, username, plainPassword, authSource string) *models.User {
	t.Helper()

	pwdMgr := NewPasswordManager(nil)
	hashed, err := pwdMgr.HashPassword(plainPassword)
	require.NoError(t, err, "hash password failed")

	user := &models.User{
		BaseModel:  models.BaseModel{ID: username + "-auth-source-id"},
		Username:   username,
		Password:   hashed,
		Salt:       "",
		Status:     models.UserStatusEnabled,
		AuthSource: authSource,
	}
	require.NoError(t, db.Create(user).Error, "create test user failed")
	return user
}

// spyAuthenticator 带调用计数的认证器替身
type spyAuthenticator struct {
	name   string
	calls  int
	result *AuthResult
	err    error
}

func (s *spyAuthenticator) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

func (s *spyAuthenticator) Name() string { return s.name }

// ========== hybrid：按 auth_source 分流 ==========

// TestHybridAuthenticator_ADSource_ADUnavailable 验证 P0-1：
// auth_source=ad 的账号，hybrid 下只走 AD 认证；AD 不可用时必须认证失败，
// 不能因为本地密码恰好正确就放行（AD 侧禁用/离职/改密必须立即生效）
func TestHybridAuthenticator_ADSource_ADUnavailable(t *testing.T) {
	db := setupAuthSourceTestDB(t)
	createUserWithAuthSource(t, db, "aduser", "password123", "ad")

	// 本地认证器替身：模拟"若没有来源分流，本地密码正确会通过"的情形
	localSpy := &spyAuthenticator{
		name:   "local",
		result: &AuthResult{User: &UserResult{Username: "aduser"}, AuthSource: "local"},
	}
	// AD 认证器替身：AD 不可用
	adSpy := &spyAuthenticator{name: "ad", err: ErrInvalidCredentials}

	auth := NewHybridAuthenticator(localSpy, adSpy, db)
	result, err := auth.Authenticate(context.Background(), MockAuthRequest("aduser", "password123"))

	assert.Error(t, err, "AD 来源账号在 AD 不可用时必须认证失败")
	assert.Nil(t, result)
	assert.Equal(t, 0, localSpy.calls, "AD 来源账号不得走本地认证")
	assert.Equal(t, 1, adSpy.calls, "AD 来源账号必须走 AD 认证")
}

// TestHybridAuthenticator_ADSource_ADAvailable 验证：
// auth_source=ad 的账号，AD 可用时认证成功，且 NeedsSync=true
func TestHybridAuthenticator_ADSource_ADAvailable(t *testing.T) {
	db := setupAuthSourceTestDB(t)
	createUserWithAuthSource(t, db, "aduser", "password123", "ad")

	localSpy := &spyAuthenticator{
		name:   "local",
		result: &AuthResult{User: &UserResult{Username: "aduser"}, AuthSource: "local"},
	}
	adSpy := &spyAuthenticator{
		name:   "ad",
		result: &AuthResult{User: &UserResult{Username: "aduser"}, AuthSource: "ad"},
	}

	auth := NewHybridAuthenticator(localSpy, adSpy, db)
	result, err := auth.Authenticate(context.Background(), MockAuthRequest("aduser", "password123"))

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "ad", result.AuthSource)
	assert.True(t, result.NeedsSync, "AD 认证成功必须标记 NeedsSync")
	assert.Equal(t, 0, localSpy.calls, "AD 来源账号不得走本地认证")
	assert.Equal(t, 1, adSpy.calls)
}

// TestHybridAuthenticator_ADSource_WithRealLocalAuthenticator 验证：
// 使用真实本地认证器时，AD 来源账号的本地密码通道已被关闭，
// hybrid 仍然只走 AD 认证
func TestHybridAuthenticator_ADSource_WithRealLocalAuthenticator(t *testing.T) {
	db := setupAuthSourceTestDB(t)
	createUserWithAuthSource(t, db, "aduser", "password123", "ad")

	pwdMgr := NewPasswordManager(nil)
	adSpy := &spyAuthenticator{
		name:   "ad",
		result: &AuthResult{User: &UserResult{Username: "aduser"}, AuthSource: "ad"},
	}

	auth := NewHybridAuthenticator(NewLocalAuthenticator(db, pwdMgr), adSpy, db)
	result, err := auth.Authenticate(context.Background(), MockAuthRequest("aduser", "password123"))

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "ad", result.AuthSource)
	assert.True(t, result.NeedsSync)
	assert.Equal(t, 1, adSpy.calls)
}

// TestHybridAuthenticator_LocalSource_LocalSuccess 验证：
// auth_source=local 的账号只走本地认证，AD 不再被调用
func TestHybridAuthenticator_LocalSource_LocalSuccess(t *testing.T) {
	db := setupAuthSourceTestDB(t)
	createUserWithAuthSource(t, db, "localuser", "password123", "local")

	pwdMgr := NewPasswordManager(nil)
	adSpy := &spyAuthenticator{
		name:   "ad",
		result: &AuthResult{User: &UserResult{Username: "localuser"}, AuthSource: "ad"},
	}

	auth := NewHybridAuthenticator(NewLocalAuthenticator(db, pwdMgr), adSpy, db)
	result, err := auth.Authenticate(context.Background(), MockAuthRequest("localuser", "password123"))

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "local", result.AuthSource)
	assert.False(t, result.NeedsSync)
	assert.Equal(t, 0, adSpy.calls, "本地来源账号不得调用 AD 认证")
}

// TestHybridAuthenticator_LocalSource_LocalFailed 验证：
// auth_source=local 的账号本地认证失败时直接失败，不再拿 AD 兜底
// （避免 AD 侧同名账号接管本地账号）
func TestHybridAuthenticator_LocalSource_LocalFailed(t *testing.T) {
	db := setupAuthSourceTestDB(t)
	createUserWithAuthSource(t, db, "localuser", "password123", "local")

	pwdMgr := NewPasswordManager(nil)
	adSpy := &spyAuthenticator{
		name:   "ad",
		result: &AuthResult{User: &UserResult{Username: "localuser"}, AuthSource: "ad"},
	}

	auth := NewHybridAuthenticator(NewLocalAuthenticator(db, pwdMgr), adSpy, db)
	result, err := auth.Authenticate(context.Background(), MockAuthRequest("localuser", "wrongpassword"))

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, adSpy.calls, "本地来源账号不得回退到 AD 认证")
}

// TestHybridAuthenticator_UserNotFound_FallbackToAD 验证：
// 系统中查无此用户时，保持"先本地后 AD"的兜底语义
func TestHybridAuthenticator_UserNotFound_FallbackToAD(t *testing.T) {
	db := setupAuthSourceTestDB(t)

	localSpy := &spyAuthenticator{name: "local", err: ErrUserNotFound}
	adSpy := &spyAuthenticator{
		name:   "ad",
		result: &AuthResult{User: &UserResult{Username: "ghostuser"}, AuthSource: "ad"},
	}

	auth := NewHybridAuthenticator(localSpy, adSpy, db)
	result, err := auth.Authenticate(context.Background(), MockAuthRequest("ghostuser", "password123"))

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "ad", result.AuthSource)
	assert.True(t, result.NeedsSync)
	assert.Equal(t, 1, localSpy.calls, "查无此用户时必须先尝试本地认证")
	assert.Equal(t, 1, adSpy.calls, "本地失败后必须降级到 AD 认证")
}

// TestHybridAuthenticator_SoftDeletedUser_Fallback 验证：
// 软删账号视为不存在，走"先本地后 AD"兜底语义
func TestHybridAuthenticator_SoftDeletedUser_Fallback(t *testing.T) {
	db := setupAuthSourceTestDB(t)
	user := createUserWithAuthSource(t, db, "deleteduser", "password123", "ad")
	require.NoError(t, db.Delete(&models.User{}, "id = ?", user.ID).Error)

	localSpy := &spyAuthenticator{name: "local", err: ErrUserNotFound}
	adSpy := &spyAuthenticator{
		name:   "ad",
		result: &AuthResult{User: &UserResult{Username: "deleteduser"}, AuthSource: "ad"},
	}

	auth := NewHybridAuthenticator(localSpy, adSpy, db)
	result, err := auth.Authenticate(context.Background(), MockAuthRequest("deleteduser", "password123"))

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "ad", result.AuthSource)
	assert.Equal(t, 1, localSpy.calls)
	assert.Equal(t, 1, adSpy.calls)
}

// TestHybridAuthenticator_DBSourceLookupError_Fallback 验证：
// 查库失败时保守降级为"先本地后 AD"
func TestHybridAuthenticator_DBSourceLookupError_Fallback(t *testing.T) {
	db := setupAuthSourceTestDB(t)
	// 破坏表结构，使查询账号来源时报错
	require.NoError(t, db.Exec("DROP TABLE sys_user").Error)

	localSpy := &spyAuthenticator{name: "local", err: ErrUserNotFound}
	adSpy := &spyAuthenticator{
		name:   "ad",
		result: &AuthResult{User: &UserResult{Username: "anyuser"}, AuthSource: "ad"},
	}

	auth := NewHybridAuthenticator(localSpy, adSpy, db)
	result, err := auth.Authenticate(context.Background(), MockAuthRequest("anyuser", "password123"))

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "ad", result.AuthSource)
	assert.Equal(t, 1, localSpy.calls, "查库失败时必须降级为先本地后 AD")
	assert.Equal(t, 1, adSpy.calls)
}

// TestHybridAuthenticator_InvalidAuthSource_Fallback 验证：
// auth_source 为空或非法时走兜底语义
func TestHybridAuthenticator_InvalidAuthSource_Fallback(t *testing.T) {
	db := setupAuthSourceTestDB(t)
	// 绕过 GORM 的 default 标签，直接写入非法来源
	require.NoError(t, db.Exec(
		"INSERT INTO sys_user (id, username, password, salt, status, auth_source) VALUES (?, ?, ?, ?, 0, ?)",
		"weird-user-id", "weirduser", "hash", "", "unknown",
	).Error)

	localSpy := &spyAuthenticator{name: "local", err: ErrUserNotFound}
	adSpy := &spyAuthenticator{
		name:   "ad",
		result: &AuthResult{User: &UserResult{Username: "weirduser"}, AuthSource: "ad"},
	}

	auth := NewHybridAuthenticator(localSpy, adSpy, db)
	result, err := auth.Authenticate(context.Background(), MockAuthRequest("weirduser", "password123"))

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "ad", result.AuthSource)
	assert.Equal(t, 1, localSpy.calls)
	assert.Equal(t, 1, adSpy.calls)
}

// TestHybridAuthenticator_NoDB_Fallback 验证：
// 未注入数据库时保持历史"先本地后 AD"兜底语义（兼容既有调用方）
func TestHybridAuthenticator_NoDB_Fallback(t *testing.T) {
	localSpy := &spyAuthenticator{name: "local", err: ErrUserNotFound}
	adSpy := &spyAuthenticator{
		name:   "ad",
		result: &AuthResult{User: &UserResult{Username: "anyuser"}, AuthSource: "ad"},
	}

	auth := NewHybridAuthenticator(localSpy, adSpy)
	result, err := auth.Authenticate(context.Background(), MockAuthRequest("anyuser", "password123"))

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "ad", result.AuthSource)
	assert.Equal(t, 1, localSpy.calls)
	assert.Equal(t, 1, adSpy.calls)
}

// ========== local 认证器：拒绝 AD 来源账号 ==========

// TestLocalAuthenticator_ADSource_Rejected 验证 P0-2：
// AD 来源账号的本地密码（随机不可知）不得通过本地通道认证
func TestLocalAuthenticator_ADSource_Rejected(t *testing.T) {
	db := setupAuthSourceTestDB(t)
	createUserWithAuthSource(t, db, "aduser", "password123", "ad")

	pwdMgr := NewPasswordManager(nil)
	auth := NewLocalAuthenticator(db, pwdMgr)

	result, err := auth.Authenticate(context.Background(), MockAuthRequest("aduser", "password123"))

	assert.Error(t, err, "AD 来源账号不得通过本地密码认证")
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, ErrInvalidCredentials), "应返回 ErrInvalidCredentials")
}

// TestLocalAuthenticator_LocalSource_Success 回归：
// 本地来源账号本地密码认证正常
func TestLocalAuthenticator_LocalSource_Success(t *testing.T) {
	db := setupAuthSourceTestDB(t)
	createUserWithAuthSource(t, db, "localuser", "password123", "local")

	pwdMgr := NewPasswordManager(nil)
	auth := NewLocalAuthenticator(db, pwdMgr)

	result, err := auth.Authenticate(context.Background(), MockAuthRequest("localuser", "password123"))

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "local", result.AuthSource)
	assert.Equal(t, "localuser", result.User.Username)
	assert.False(t, result.NeedsSync)
}

// TestLocalAuthenticator_EmptyAuthSource_TreatedAsLocal 回归：
// auth_source 为空（历史数据）时按本地账号处理
func TestLocalAuthenticator_EmptyAuthSource_TreatedAsLocal(t *testing.T) {
	db := setupAuthSourceTestDB(t)

	pwdMgr := NewPasswordManager(nil)
	hashed, err := pwdMgr.HashPassword("password123")
	require.NoError(t, err)
	require.NoError(t, db.Exec(
		"INSERT INTO sys_user (id, username, password, salt, status, auth_source) VALUES (?, ?, ?, ?, 0, '')",
		"empty-source-id", "emptysource", hashed, "",
	).Error)

	auth := NewLocalAuthenticator(db, pwdMgr)
	result, err := auth.Authenticate(context.Background(), MockAuthRequest("emptysource", "password123"))

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "emptysource", result.User.Username)
}

// ========== 工厂：hybrid 分支注入数据库 ==========

// TestAuthStrategyFactory_HybridInjectsDB 验证：
// 工厂构造 hybrid 认证器时注入数据库，使其能按 auth_source 分流
func TestAuthStrategyFactory_HybridInjectsDB(t *testing.T) {
	db := setupAuthSourceTestDB(t)
	pwdMgr := NewPasswordManager(nil)

	adConfig := &models.ADConfig{
		BaseModel:     models.BaseModel{ID: "hybrid-ad-config-id"},
		ConfigName:    "Test AD",
		ServerAddress: "192.168.1.100",
		ServerPort:    389,
		DomainName:    "test.com",
		BaseDN:        "dc=test,dc=com",
		AdminUsername: "admin",
		AdminPassword: "encrypted_password",
		Status:        models.ADConfigStatusEnabled,
	}
	require.NoError(t, db.Create(adConfig).Error)

	factory := NewAuthStrategyFactory(db, pwdMgr)
	auth, err := factory.GetAuthenticator("hybrid")
	require.NoError(t, err)

	hybrid, ok := auth.(*HybridAuthenticator)
	require.True(t, ok, "应返回 HybridAuthenticator 实例")
	assert.NotNil(t, hybrid.db, "hybrid 认证器必须注入数据库才能按 auth_source 分流")
	assert.Equal(t, "hybrid", auth.Name())
}
