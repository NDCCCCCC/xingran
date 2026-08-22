package rpa

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
)

// fakeRPACipher addomain.PasswordCipher 假实现（前缀标记法）。
type fakeRPACipher struct{ fail bool }

func (f *fakeRPACipher) Encrypt(p string) (string, error) {
	if f.fail {
		return "", assert.AnError
	}
	return "enc:" + p, nil
}

func (f *fakeRPACipher) Decrypt(c string) (string, error) {
	if f.fail || !strings.HasPrefix(c, "enc:") {
		return "", assert.AnError
	}
	return strings.TrimPrefix(c, "enc:"), nil
}

func newCredentialTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_rpa_credentials (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			name TEXT,
			target_system TEXT,
			target_url TEXT,
			username_encrypted TEXT,
			password_encrypted TEXT,
			extra_data_encrypted TEXT,
			user_id TEXT,
			dept_id TEXT,
			is_shared BOOLEAN DEFAULT 0,
			status INTEGER DEFAULT 0,
			last_used_at DATETIME,
			last_login_at DATETIME,
			login_success_count INTEGER DEFAULT 0,
			login_fail_count INTEGER DEFAULT 0
		);
		CREATE TABLE sys_rpa_sessions (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			credential_id TEXT,
			execution_id TEXT,
			target_system TEXT,
			target_url TEXT,
			access_token_encrypted TEXT,
			refresh_token_encrypted TEXT,
			cookies_encrypted TEXT,
			session_data_encrypted TEXT,
			expires_at DATETIME,
			is_valid BOOLEAN DEFAULT 1,
			invalid_reason TEXT
		)
	`).Error)
	return db
}

func TestCredentialService_CreateGetDelete(t *testing.T) {
	db := newCredentialTestDB(t)
	svc := NewCredentialService(db, &fakeRPACipher{}, nil)

	cred, err := svc.CreateCredential(context.Background(), &rpamodels.CredentialCreateRequest{
		Name:         "c1",
		TargetSystem: "erp",
		TargetURL:    "http://erp",
		Username:     "alice",
		Password:     "pw",
		ExtraData:    map[string]interface{}{"k": "v"},
	}, "u1")
	require.NoError(t, err)
	assert.Equal(t, "alice", cred.Username, "返回时应填充明文用户名")
	assert.Equal(t, "enc:alice", cred.UsernameEncrypted)

	// GetCredential: 返回解密后的用户名
	got, err := svc.GetCredential(context.Background(), cred.ID, "u1")
	require.NoError(t, err)
	assert.Equal(t, "alice", got.Username)

	// 错误 user_id → not found
	_, err = svc.GetCredential(context.Background(), cred.ID, "other")
	require.Error(t, err)

	// Delete + 再取
	require.NoError(t, svc.DeleteCredential(context.Background(), cred.ID, "u1"))
	_, err = svc.GetCredential(context.Background(), cred.ID, "u1")
	require.Error(t, err)

	// 加密失败路径
	svcFail := NewCredentialService(db, &fakeRPACipher{fail: true}, nil)
	_, err = svcFail.CreateCredential(context.Background(), &rpamodels.CredentialCreateRequest{
		Username: "x", Password: "y",
	}, "u1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "加密凭证失败")
}

func TestCredentialService_UpdateCredential(t *testing.T) {
	db := newCredentialTestDB(t)
	svc := NewCredentialService(db, &fakeRPACipher{}, nil)
	cred, err := svc.CreateCredential(context.Background(), &rpamodels.CredentialCreateRequest{
		Name: "old", Username: "u", Password: "p",
	}, "u1")
	require.NoError(t, err)

	shared := true
	status := 1
	require.NoError(t, svc.UpdateCredential(context.Background(), cred.ID, &rpamodels.CredentialUpdateRequest{
		Name:     "new",
		IsShared: &shared,
		Status:   &status,
	}, "u1"))

	// 更新用户名/密码 → 重新加密
	require.NoError(t, svc.UpdateCredential(context.Background(), cred.ID, &rpamodels.CredentialUpdateRequest{
		Username: "u2",
	}, "u1"))

	got, err := svc.GetCredential(context.Background(), cred.ID, "u1")
	require.NoError(t, err)
	assert.Equal(t, "new", got.Name)
	assert.Equal(t, "u2", got.Username)
	assert.Equal(t, "enc:u2", got.UsernameEncrypted)

	// 不存在的凭证
	err = svc.UpdateCredential(context.Background(), "missing", &rpamodels.CredentialUpdateRequest{
		Username: "x",
	}, "u1")
	require.Error(t, err)
}

func TestCredentialService_ListCredentials(t *testing.T) {
	db := newCredentialTestDB(t)
	now := time.Now().Format("2006-01-02 15:04:05")
	insert := func(id, user, system string, shared bool, status int) {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_rpa_credentials (id, name, target_system, username_encrypted, password_encrypted, user_id, dept_id, is_shared, status, created_at) VALUES (?, ?, ?, ?, ?, ?, 'd1', ?, ?, ?)`,
			id, id, system, "enc:user", "enc:pw", user, shared, status, now,
		).Error)
	}
	insert("c1", "u1", "erp", false, 0)
	insert("c2", "u2", "erp", true, 0)
	insert("c3", "u1", "hr", false, 1)

	svc := NewCredentialService(db, &fakeRPACipher{}, nil)
	ctx := context.Background()

	// 自己 + 部门共享: c1/c3 归 u1, c2 是 d1 共享 → 3 条
	list, total, err := svc.ListCredentials(ctx, &rpamodels.CredentialListParams{Current: 1, PageSize: 10}, "u1", "d1")
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	for _, c := range list {
		assert.Equal(t, "user", c.Username, "列表返回解密用户名")
	}

	// 目标系统过滤
	list, total, err = svc.ListCredentials(ctx, &rpamodels.CredentialListParams{Current: 1, PageSize: 10, TargetSystem: "hr"}, "u1", "d1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "c3", list[0].ID)

	// 状态过滤
	status := 1
	_, total, err = svc.ListCredentials(ctx, &rpamodels.CredentialListParams{Current: 1, PageSize: 10, Status: &status}, "u1", "d1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 仅我的
	list, total, err = svc.ListCredentials(ctx, &rpamodels.CredentialListParams{Current: 1, PageSize: 10, MyCredOnly: true}, "u1", "d1")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	_ = list
}

func TestCredentialService_GetCredentialForExecution(t *testing.T) {
	db := newCredentialTestDB(t)
	now := time.Now().Format("2006-01-02 15:04:05")
	require.NoError(t, db.Exec(
		`INSERT INTO sys_rpa_credentials (id, name, target_system, username_encrypted, password_encrypted, user_id, dept_id, is_shared, status, created_at) VALUES ('own', 'n', 'erp', 'enc:u', 'enc:p', 'u1', 'd1', 0, 0, ?)`, now).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_rpa_credentials (id, name, target_system, username_encrypted, password_encrypted, user_id, dept_id, is_shared, status, created_at) VALUES ('shared', 'n', 'erp', 'enc:u', 'enc:p', 'u2', 'd1', 1, 0, ?)`, now).Error)

	svc := NewCredentialService(db, &fakeRPACipher{}, nil)
	ctx := context.Background()

	// 优先自己的
	cred, err := svc.GetCredentialForExecution(ctx, "erp", "u1", "d1")
	require.NoError(t, err)
	assert.Equal(t, "own", cred.ID)

	// 自己没有 → 部门共享
	cred, err = svc.GetCredentialForExecution(ctx, "erp", "u3", "d1")
	require.NoError(t, err)
	assert.Equal(t, "shared", cred.ID)

	// 都没有 → 报错
	_, err = svc.GetCredentialForExecution(ctx, "none", "u3", "d1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未找到有效的凭证")
}

func TestCredentialService_DecryptCredential(t *testing.T) {
	db := newCredentialTestDB(t)
	svc := NewCredentialService(db, &fakeRPACipher{}, nil)
	created, err := svc.CreateCredential(context.Background(), &rpamodels.CredentialCreateRequest{
		Username: "alice", Password: "secret", ExtraData: map[string]interface{}{"role": "admin"},
	}, "u1")
	require.NoError(t, err)

	got, err := svc.GetCredential(context.Background(), created.ID, "u1")
	require.NoError(t, err)
	data, err := svc.DecryptCredential(context.Background(), got)
	require.NoError(t, err)
	assert.Equal(t, "alice", data.Username)
	assert.Equal(t, "secret", data.Password)
	assert.Equal(t, map[string]interface{}{"role": "admin"}, data.ExtraData)

	// 解密失败 → 空串（decryptString 静默回退）
	badSvc := NewCredentialService(db, &fakeRPACipher{fail: true}, nil)
	data, err = badSvc.DecryptCredential(context.Background(), got)
	require.NoError(t, err)
	assert.Equal(t, "", data.Username)
}

func TestCredentialService_Sessions(t *testing.T) {
	db := newCredentialTestDB(t)
	svc := NewCredentialService(db, &fakeRPACipher{}, nil)
	ctx := context.Background()

	// CreateSession: token + cookies + sessionData 全量加密
	created, err := svc.CreateSession(ctx, &rpamodels.SessionCreateRequest{
		CredentialID: "c1",
		TargetSystem: "erp",
		TargetURL:    "http://erp",
		AccessToken:  "at",
		RefreshToken: "rt",
		Cookies:      []rpamodels.Cookie{{Name: "sid", Value: "1"}},
		SessionData:  map[string]interface{}{"loggedIn": true},
		ExpiresAt:    nil,
	})
	require.NoError(t, err)
	assert.Equal(t, "at", created.AccessToken, "返回时填充明文 token")
	assert.Equal(t, "enc:at", created.AccessTokenEncrypted)

	// GetValidSession: 无过期 → 命中并解密
	sess, err := svc.GetValidSession(ctx, "c1", "erp")
	require.NoError(t, err)
	assert.Equal(t, "at", sess.AccessToken)
	assert.Len(t, sess.Cookies, 1)
	assert.Equal(t, true, sess.SessionData["loggedIn"])

	// 目标系统不匹配 → not found
	_, err = svc.GetValidSession(ctx, "c1", "hr")
	require.Error(t, err)

	// InvalidateSession
	require.NoError(t, svc.InvalidateSession(ctx, created.ID, "manual"))
	_, err = svc.GetValidSession(ctx, "c1", "erp")
	require.Error(t, err, "失效后不应再命中")

	// 已过期的会话
	expired := time.Now().Add(-time.Hour)
	exp, err := svc.CreateSession(ctx, &rpamodels.SessionCreateRequest{
		CredentialID: "c2", TargetSystem: "erp", AccessToken: "t2", ExpiresAt: &expired,
	})
	require.NoError(t, err)
	_, err = svc.GetValidSession(ctx, "c2", "erp")
	require.Error(t, err, "过期会话不应命中")

	// CleanupExpiredSessions
	require.NoError(t, svc.CleanupExpiredSessions(ctx))
	var isValid bool
	require.NoError(t, db.Raw(`SELECT is_valid FROM sys_rpa_sessions WHERE id = ?`, exp.ID).Scan(&isValid).Error)
	assert.False(t, isValid)
}

func TestCredentialService_LoginTracking(t *testing.T) {
	db := newCredentialTestDB(t)
	now := time.Now().Format("2006-01-02 15:04:05")
	require.NoError(t, db.Exec(
		`INSERT INTO sys_rpa_credentials (id, name, target_system, username_encrypted, password_encrypted, user_id, status, created_at) VALUES ('c1', 'n', 'erp', 'enc:u', 'enc:p', 'u1', 0, ?)`, now).Error)

	svc := NewCredentialService(db, &fakeRPACipher{}, nil)
	ctx := context.Background()
	require.NoError(t, svc.RecordLoginSuccess(ctx, "c1"))
	require.NoError(t, svc.RecordLoginSuccess(ctx, "c1"))
	require.NoError(t, svc.RecordLoginFailure(ctx, "c1"))
	require.NoError(t, svc.UpdateLastUsed(ctx, "c1"))

	var sc, fc int
	var lastLogin, lastUsed *time.Time
	require.NoError(t, db.Raw(`SELECT login_success_count, login_fail_count FROM sys_rpa_credentials WHERE id = 'c1'`).Row().Scan(&sc, &fc))
	require.NoError(t, db.Raw(`SELECT last_login_at FROM sys_rpa_credentials WHERE id = 'c1'`).Row().Scan(&lastLogin))
	require.NoError(t, db.Raw(`SELECT last_used_at FROM sys_rpa_credentials WHERE id = 'c1'`).Row().Scan(&lastUsed))
	assert.Equal(t, 2, sc)
	assert.Equal(t, 1, fc)
	assert.NotNil(t, lastLogin)
	assert.NotNil(t, lastUsed)
}
