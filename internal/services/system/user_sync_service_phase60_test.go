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

	"github.com/xingran-next/xingran-go-backend/internal/core/security"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// insertPhase60SoftDeletedUser 在同一 DB 中插入一行软删用户并返回 userID。
// storedDN 为 nil 时 ad_dn 写 NULL；否则写指针指向的字面值。
func insertPhase60SoftDeletedUser(t *testing.T, db *gorm.DB, username string, storedDN *string) string {
	t.Helper()

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
	return userID
}

// setupPhase60SoftDeletedFixture 建库并插入一行软删用户。
// storedDN 为 nil 时 ad_dn 写 NULL；否则写指针指向的字面值。
func setupPhase60SoftDeletedFixture(t *testing.T, username string, storedDN *string) (*gorm.DB, *UserSyncService, string) {
	t.Helper()

	db := setupTestDBForSync(t)
	require.NotNil(t, db, "setupTestDBForSync 必须返回可用 DB")

	userID := insertPhase60SoftDeletedUser(t, db, username, storedDN)

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

// --- 批量路径（SAFE-01 批量侧 / SAFE-02 双路径一致性） ---
//
// 注意（Pitfall 8）：批量用例的 ADUserInfoForSync.OuDn 一律置空，避免
// resolveDeptFromOU 在 nil mapper 上 panic；defaultRoleID 传空串避开角色表依赖。

// TestPhase60_Batch_MismatchedDN_Skipped DN 不一致 → Skipped，保持软删且不降级。
func TestPhase60_Batch_MismatchedDN_Skipped(t *testing.T) {
	const stored = "CN=batch,OU=IT,DC=corp,DC=com"
	const current = "CN=someone-else,OU=IT,DC=corp,DC=com"
	const username = "phase60_batch_mismatch"

	db, svc, userID := setupPhase60SoftDeletedFixture(t, username, strPtrSyncTest(stored))

	buf, restore := captureLogs(t)
	defer restore()

	res, err := svc.BatchSyncADUsers(context.Background(), []*ADUserInfoForSync{
		phase60ADUser(username, current),
	}, "")
	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, 1, res.Total)
	assert.Equal(t, 1, res.Skipped, "DN 不一致必须计入 Skipped")
	assert.Equal(t, 0, res.Failed, "策略拒绝不得伪装成 DB 故障")
	assert.Equal(t, 0, res.Success, "被拒项不得计入 Success")
	require.Len(t, res.Errors, 1)
	assert.Equal(t, username, res.Errors[0].Username)
	assert.Contains(t, res.Errors[0].Error, "AD 身份不一致")
	assert.NotContains(t, res.Errors[0].Error, "CN=", "响应错误文本不得泄露 DN（D-05）")

	var after models.User
	require.NoError(t, db.Unscoped().First(&after, "id = ?", userID).Error)
	assert.True(t, after.DeletedAt.Valid, "拒绝后必须保持软删")
	require.NotNil(t, after.AdDn)
	assert.Equal(t, stored, *after.AdDn, "拒绝路径绝不清空存量 ad_dn（D-03）")

	logs := buf.String()
	assert.Contains(t, logs, "[SECURITY]")
	assert.Contains(t, logs, "path=batch", "批量拒绝的审计 WARN 必须标注 path=batch")
	assert.NotContains(t, logs, "降级逐个处理", "策略拒绝不得进入 runInBatchedTx 降级管道")
}

// TestPhase60_Batch_MatchingDN_Restores DN 归一化一致 → 批量恢复成功。
func TestPhase60_Batch_MatchingDN_Restores(t *testing.T) {
	const stored = "CN=batch-ok,OU=IT,DC=corp,DC=com"
	const current = "cn=BATCH-OK,ou=it,dc=CORP,dc=COM"
	const username = "phase60_batch_match"

	db, svc, userID := setupPhase60SoftDeletedFixture(t, username, strPtrSyncTest(stored))

	res, err := svc.BatchSyncADUsers(context.Background(), []*ADUserInfoForSync{
		phase60ADUser(username, current),
	}, "")
	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, 1, res.Total)
	assert.Equal(t, 1, res.Success)
	assert.Equal(t, 0, res.Skipped)
	assert.Equal(t, 0, res.Failed)

	var after models.User
	require.NoError(t, db.First(&after, "id = ?", userID).Error, "恢复后应能被活跃作用域查询到")
	assert.False(t, after.DeletedAt.Valid, "deleted_at 必须被清空")
	require.NotNil(t, after.AdDn)
	assert.Equal(t, current, *after.AdDn, "恢复时写入 AD 当前 DN")
}

// TestPhase60_Batch_MixedUsers_Conservation 同批一允一拒 → 总量守恒。
func TestPhase60_Batch_MixedUsers_Conservation(t *testing.T) {
	const okUser = "phase60_mixed_ok"
	const badUser = "phase60_mixed_bad"
	const okStored = "CN=mixed-ok,OU=IT,DC=corp,DC=com"
	const okCurrent = "cn=mixed-OK,ou=it,dc=corp,dc=com"
	const badStored = "CN=mixed-bad,OU=IT,DC=corp,DC=com"
	const badCurrent = "CN=other,OU=IT,DC=corp,DC=com"

	db, svc, _ := setupPhase60SoftDeletedFixture(t, okUser, strPtrSyncTest(okStored))
	insertPhase60SoftDeletedUser(t, db, badUser, strPtrSyncTest(badStored))

	res, err := svc.BatchSyncADUsers(context.Background(), []*ADUserInfoForSync{
		phase60ADUser(okUser, okCurrent),
		phase60ADUser(badUser, badCurrent),
	}, "")
	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, 2, res.Total)
	assert.Equal(t, 1, res.Success, "一致项恢复成功")
	assert.Equal(t, 1, res.Skipped, "不一致项跳过")
	assert.Equal(t, 0, res.Failed)
	assert.Equal(t, res.Total, res.Success+res.Failed+res.Skipped, "成功+失败+跳过必须恒等于总数")
}

// TestPhase60_PathConsistency_MismatchRefusedBoth 同一输入下登录与批量判定及终态一致。
func TestPhase60_PathConsistency_MismatchRefusedBoth(t *testing.T) {
	const stored = "CN=path-mismatch,OU=IT,DC=corp,DC=com"
	const current = "CN=other,OU=IT,DC=corp,DC=com"
	const username = "phase60_path_mismatch"

	dbLogin, svcLogin, idLogin := setupPhase60SoftDeletedFixture(t, username, strPtrSyncTest(stored))
	dbBatch, svcBatch, idBatch := setupPhase60SoftDeletedFixture(t, username, strPtrSyncTest(stored))

	// A: 登录路径 → sentinel 拒绝
	_, _, loginErr := svcLogin.SyncUserFromAD(context.Background(), phase60ADUser(username, current), "")
	require.Error(t, loginErr)
	assert.True(t, errors.Is(loginErr, security.ErrSoftDeleteIdentityMismatch))

	// B: 批量路径 → Skipped
	res, batchErr := svcBatch.BatchSyncADUsers(context.Background(), []*ADUserInfoForSync{
		phase60ADUser(username, current),
	}, "")
	require.NoError(t, batchErr)
	require.NotNil(t, res)
	assert.Equal(t, 1, res.Skipped)
	assert.Equal(t, 0, res.Failed)

	// 两库终态一致：均保持软删，且 ad_dn 原值逐字不变
	for _, c := range []struct {
		name string
		db   *gorm.DB
		id   string
	}{
		{"login", dbLogin, idLogin},
		{"batch", dbBatch, idBatch},
	} {
		var after models.User
		require.NoError(t, c.db.Unscoped().First(&after, "id = ?", c.id).Error)
		assert.True(t, after.DeletedAt.Valid, "%s 路径拒绝后必须保持软删", c.name)
		require.NotNil(t, after.AdDn)
		assert.Equal(t, stored, *after.AdDn, "%s 路径拒绝后 ad_dn 必须原值保留", c.name)
	}
}

// TestPhase60_PathConsistency_MatchingRestoredBoth 一致场景下两路径均恢复且终态一致。
func TestPhase60_PathConsistency_MatchingRestoredBoth(t *testing.T) {
	const stored = "CN=path-match,OU=IT,DC=corp,DC=com"
	const current = "cn=path-MATCH,ou=it,dc=CORP,dc=COM"
	const username = "phase60_path_match"

	dbLogin, svcLogin, idLogin := setupPhase60SoftDeletedFixture(t, username, strPtrSyncTest(stored))
	dbBatch, svcBatch, idBatch := setupPhase60SoftDeletedFixture(t, username, strPtrSyncTest(stored))

	_, _, loginErr := svcLogin.SyncUserFromAD(context.Background(), phase60ADUser(username, current), "")
	require.NoError(t, loginErr, "登录路径归一化一致必须恢复")

	res, batchErr := svcBatch.BatchSyncADUsers(context.Background(), []*ADUserInfoForSync{
		phase60ADUser(username, current),
	}, "")
	require.NoError(t, batchErr)
	require.NotNil(t, res)
	assert.Equal(t, 1, res.Success)
	assert.Equal(t, 0, res.Skipped)
	assert.Equal(t, 0, res.Failed)

	for _, c := range []struct {
		name string
		db   *gorm.DB
		id   string
	}{
		{"login", dbLogin, idLogin},
		{"batch", dbBatch, idBatch},
	} {
		var after models.User
		require.NoError(t, c.db.First(&after, "id = ?", c.id).Error, "%s 路径恢复后应能被活跃作用域查询到", c.name)
		assert.False(t, after.DeletedAt.Valid, "%s 路径必须清空 deleted_at", c.name)
		require.NotNil(t, after.AdDn)
		assert.Equal(t, current, *after.AdDn, "%s 路径恢复后 ad_dn 应写入 AD 当前 DN", c.name)
	}
}
