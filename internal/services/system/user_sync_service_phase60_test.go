package system

// Phase 60 SAFE-01 / SAFE-02：软删恢复的 AD 身份校验（登录路径）。
//
// 覆盖：
//   - 共享校验入口 checkSoftDeletedRestore 的判定表（D-01 / D-02）；
//   - 登录路径 SyncUserFromAD 在 DN 一致时恢复、不一致时拒绝（D-03 零写入）；
//   - 空存量 DN 放行（D-01）；
//   - 拒绝审计 WARN 的字段完整性（D-05）。
//
// 复用既有基座（不重复定义）：setupTestDBForSync / captureLogs / strPtrSyncTest。

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// setupPhase60SoftDeletedFixture 建库并插入一行软删用户。
// storedDN 为 nil 时 ad_dn 写 NULL；否则写指针指向的字面值。
func setupPhase60SoftDeletedFixture(t *testing.T, username string, storedDN *string) (*gorm.DB, *UserSyncService, string) {
	t.Helper()

	db := setupTestDBForSync(t)
	require.NotNil(t, db, "setupTestDBForSync 必须返回可用 DB")

	userID := uuid.NewString()
	if storedDN == nil {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_user (id, username, password, salt, nickname, auth_source, status, deleted_at, ad_dn)
			 VALUES (?, ?, '', '', ?, 'local', 0, ?, NULL)`,
			userID, username, "Phase60 软删用户", time.Now()).Error)
	} else {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_user (id, username, password, salt, nickname, auth_source, status, deleted_at, ad_dn)
			 VALUES (?, ?, '', '', ?, 'local', 0, ?, ?)`,
			userID, username, "Phase60 软删用户", time.Now(), *storedDN).Error)
	}

	svc := NewUserSyncService(db, security.NewPasswordManager(nil), nil)
	return db, svc, userID
}

// phase60ADUser 构造登录路径使用的 AD 用户信息。
func phase60ADUser(username, userDN string) *ADUserInfoForSync {
	return &ADUserInfoForSync{
		UserDN:      userDN,
		Username:    username,
		DisplayName: "Phase60 AD User",
		Email:       "phase60@example.com",
		Phone:       "1234567890",
	}
}

// TestPhase60_CheckSoftDeletedRestore_Predicate 锁定共享校验入口的判定表。
func TestPhase60_CheckSoftDeletedRestore_Predicate(t *testing.T) {
	tests := []struct {
		name      string
		storedDN  *string
		currentDN string
		want      bool
	}{
		{"nil 存量 DN 放行", nil, "CN=u,DC=corp,DC=com", true},
		{"空串存量 DN 放行", strPtrSyncTest(""), "CN=u,DC=corp,DC=com", true},
		{"纯空白存量 DN 放行", strPtrSyncTest("   "), "CN=u,DC=corp,DC=com", true},
		{"完全一致放行", strPtrSyncTest("CN=u,DC=corp,DC=com"), "CN=u,DC=corp,DC=com", true},
		{"大小写不同放行", strPtrSyncTest("cn=U,dc=CORP,dc=COM"), "CN=u,DC=corp,DC=com", true},
		{"段内空格放行", strPtrSyncTest("CN=u, DC=corp, DC=com"), "CN=u,DC=corp,DC=com", true},
		{"不同 DN 拒绝", strPtrSyncTest("CN=other,DC=corp,DC=com"), "CN=u,DC=corp,DC=com", false},
		{"存量非空但当前为空拒绝", strPtrSyncTest("CN=u,DC=corp,DC=com"), "", false},
	}

	buf, restore := captureLogs(t)
	defer restore()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &UserSyncService{}
			got := svc.checkSoftDeletedRestore(
				&models.User{Username: "phase60-predicate", AdDn: tt.storedDN}, tt.currentDN, "login")
			assert.Equal(t, tt.want, got)
		})
	}

	logs := buf.String()
	// 两条「拒绝」用例各写一条 WARN；放行组不得产生任何审计 WARN。
	assert.Equal(t, 2, strings.Count(logs, "[SECURITY] 软删用户恢复被拒绝"),
		"仅拒绝分支产生审计 WARN，放行组不得写日志")
	assert.Contains(t, logs, "[SECURITY]")
	assert.Contains(t, logs, "path=login")
	assert.Contains(t, logs, "storedAdDN=")
	assert.Contains(t, logs, "currentAdDN=")
	assert.Contains(t, logs, "CN=other,DC=corp,DC=com")
	assert.Contains(t, logs, "CN=u,DC=corp,DC=com")
}

// TestPhase60_Login_MatchingDN_Restores DN 归一化一致 → 恢复成功。
func TestPhase60_Login_MatchingDN_Restores(t *testing.T) {
	const stored = "CN=u,OU=IT,DC=corp,DC=com"
	const current = "cn=U,ou=it,dc=CORP,dc=COM"

	db, svc, userID := setupPhase60SoftDeletedFixture(t, "phase60_match", strPtrSyncTest(stored))

	_, _, err := svc.SyncUserFromAD(context.Background(), phase60ADUser("phase60_match", current), "")
	require.NoError(t, err, "归一化后一致必须允许恢复")

	var after models.User
	require.NoError(t, db.First(&after, "id = ?", userID).Error, "恢复后应能被普通作用域查询到")
	assert.False(t, after.DeletedAt.Valid, "deleted_at 必须被清空")
	require.NotNil(t, after.AdDn)
	assert.Equal(t, current, *after.AdDn, "恢复时写入 AD 当前 DN")
}

// TestPhase60_Login_MismatchedDN_Refuses DN 不一致 → 拒绝且零写入（D-03）。
func TestPhase60_Login_MismatchedDN_Refuses(t *testing.T) {
	const stored = "CN=u,OU=IT,DC=corp,DC=com"
	const current = "CN=other,OU=IT,DC=corp,DC=com"

	db, svc, userID := setupPhase60SoftDeletedFixture(t, "phase60_mismatch", strPtrSyncTest(stored))
	adUser := phase60ADUser("phase60_mismatch", current)

	// 连续两次：第二次仍被拒，证明第一次没有靠清空 DN 绕过闸门。
	for attempt := 1; attempt <= 2; attempt++ {
		_, _, err := svc.SyncUserFromAD(context.Background(), adUser, "")
		require.Error(t, err, "第 %d 次登录必须被拒绝", attempt)
		assert.True(t, errors.Is(err, security.ErrSoftDeleteIdentityMismatch),
			"第 %d 次登录必须返回可被 errors.Is 识别的 sentinel", attempt)

		var after models.User
		require.NoError(t, db.Unscoped().First(&after, "id = ?", userID).Error)
		assert.True(t, after.DeletedAt.Valid, "第 %d 次拒绝后必须保持软删", attempt)
		require.NotNil(t, after.AdDn)
		assert.Equal(t, stored, *after.AdDn, "拒绝路径绝不清空/改写存量 ad_dn（D-03）")
	}
}

// TestPhase60_Login_EmptyStoredDN_Restores 存量 ad_dn 为 NULL → 放行（D-01）。
func TestPhase60_Login_EmptyStoredDN_Restores(t *testing.T) {
	const current = "CN=empty,OU=IT,DC=corp,DC=com"

	db, svc, userID := setupPhase60SoftDeletedFixture(t, "phase60_empty", nil)

	_, _, err := svc.SyncUserFromAD(context.Background(), phase60ADUser("phase60_empty", current), "")
	require.NoError(t, err, "空存量 DN 必须放行恢复（D-01）")

	var after models.User
	require.NoError(t, db.First(&after, "id = ?", userID).Error)
	assert.False(t, after.DeletedAt.Valid, "deleted_at 必须被清空")
	require.NotNil(t, after.AdDn)
	assert.Equal(t, current, *after.AdDn, "恢复时写入 AD 当前 DN")
}

// TestPhase60_Login_RefusalAuditWarn 拒绝审计 WARN 含全量诊断字段，且不含用户文案。
func TestPhase60_Login_RefusalAuditWarn(t *testing.T) {
	const stored = "CN=audit,OU=IT,DC=corp,DC=com"
	const current = "CN=someone-else,OU=IT,DC=corp,DC=com"
	const username = "phase60_audit"

	_, svc, userID := setupPhase60SoftDeletedFixture(t, username, strPtrSyncTest(stored))

	buf, restore := captureLogs(t)
	defer restore()

	_, _, err := svc.SyncUserFromAD(context.Background(), phase60ADUser(username, current), "")
	require.Error(t, err)

	logs := buf.String()
	for _, want := range []string{userID, username, stored, current, "path=login", "[SECURITY]"} {
		assert.Contains(t, logs, want, "审计 WARN 缺少字段: %s", want)
	}
	assert.NotContains(t, logs, "该账号存在 AD 身份不一致，已被安全拦截，请联系系统管理员",
		"用户可见文案不应出现在服务层日志中")
}
