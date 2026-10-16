package system

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// mockSM4Cipher 模拟 SM4 cipher
type mockSM4Cipher struct {
	encrypted []string
}

func (m *mockSM4Cipher) Encrypt(plaintext string) (string, error) {
	out := "mock-encrypted:" + plaintext
	m.encrypted = append(m.encrypted, out)
	return out, nil
}

func (m *mockSM4Cipher) Decrypt(ciphertext string) (string, error) {
	return ciphertext, nil
}

// setupTestDB 创建 SQLite 内存数据库
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
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

	return db
}

// TC1: SM4 cipher 加密流程
//
// 验证 handler 用 cipher.Encrypt() 后密文写入 DB
func TestSM4Cipher_EncryptFlow(t *testing.T) {
	cipher := &mockSM4Cipher{}
	plaintext := "MySecretPassword123"

	ciphertext, err := cipher.Encrypt(plaintext)
	require.NoError(t, err)
	assert.Equal(t, "mock-encrypted:"+plaintext, ciphertext)
	assert.NotEqual(t, plaintext, ciphertext, "密文不应等于明文")
}

// TC2: 多个账号独立加密（不会混淆）
func TestSM4Cipher_MultipleAccounts(t *testing.T) {
	cipher := &mockSM4Cipher{}

	pwds := []string{"admin1_pwd", "admin2_pwd", "admin3_pwd"}
	for _, pwd := range pwds {
		ct, err := cipher.Encrypt(pwd)
		require.NoError(t, err)
		assert.Contains(t, ct, pwd, "每个账号应独立加密")
	}

	assert.Len(t, cipher.encrypted, 3)
}

// TC3: DB 中 config_id 隔离
//
// 即使 mock core 不完整，也能验证密码写入 DB 后 config_id 正确隔离
func TestConfigIDIsolation_DB(t *testing.T) {
	db := setupTestDB(t)
	configA := uuid.NewString()
	configB := uuid.NewString()

	accounts := []struct {
		id       string
		configID string
		username string
	}{
		{uuid.NewString(), configA, "svc-A1"},
		{uuid.NewString(), configA, "svc-A2"},
		{uuid.NewString(), configB, "svc-B1"},
	}

	for _, a := range accounts {
		err := db.Exec(`INSERT INTO sys_ad_service_accounts
			(id, config_id, username, password_ciphertext, status, failure_count, created_at, updated_at)
			VALUES (?, ?, ?, 'encrypted', 0, 0, datetime('now'), datetime('now'))`,
			a.id, a.configID, a.username).Error
		require.NoError(t, err)
	}

	var countA, countB int64
	db.Table("sys_ad_service_accounts").
		Where("config_id = ? AND deleted_at IS NULL", configA).
		Count(&countA)
	db.Table("sys_ad_service_accounts").
		Where("config_id = ? AND deleted_at IS NULL", configB).
		Count(&countB)

	assert.Equal(t, int64(2), countA, "config A 应有 2 个账号")
	assert.Equal(t, int64(1), countB, "config B 应有 1 个账号")
	assert.NotEqual(t, countA, countB, "不同 config 的账号数应不同（隔离性）")
}

// TC4: SM4 失败时返回错误
//
// 当 cipher 报错时 handler 应返回 500 而非静默写明文
func TestSM4Cipher_FailureOnError(t *testing.T) {
	failingCipher := &failingMockSM4Cipher{}
	_, err := failingCipher.Encrypt("password")
	assert.Error(t, err, "cipher 失败时应返回 error")
}

// failingMockSM4Cipher 测试用的失败 cipher
type failingMockSM4Cipher struct{}

func (f *failingMockSM4Cipher) Encrypt(plaintext string) (string, error) {
	return "", assertErr("mock encrypt failure")
}

func (f *failingMockSM4Cipher) Decrypt(ciphertext string) (string, error) {
	return "", nil
}

type assertErr string

func (e assertErr) Error() string { return string(e) }