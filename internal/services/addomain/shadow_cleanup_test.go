//go:build !skip_db_tests
// +build !skip_db_tests

// Phase 61 Plan 01: 影子表删除清理测试（SHADOW-01 两道闸门 + SHADOW-03 增量 no-op）。
//
// fixture 零副本：复用 78-05 helper（setupSync78DB / entry78 / newSyncSvc78 /
// insertConfig78 / closeDB），computer 用例经 78-06 的 setupComp78DB 补建
// sys_ad_computer 表（D-78-06e 禁止重定义）。零 LDAP 网络 + []*ldap.Entry 字面量驱动。
//
// 闸门语义先例：group_sync_threshold_test.go（恰 50% 放行 = 判式严格大于）。

package addomain

import (
	"context"
	"testing"

	"github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// ============================================================================
// 预置 helper（本文件私有；表建表/配置行一律复用 78-05/78-06 helper,零副本）
// ============================================================================

// seedShadowUser78 预置一条 sys_ad_user 活行（username 模型 not null,必填）。
func seedShadowUser78(t *testing.T, db *gorm.DB, configID, dn, username string) {
	t.Helper()
	user := &models.ADUser{ADConfigID: configID, UserDN: dn, Username: username}
	user.ID = uuid.NewString()
	require.NoError(t, db.Create(user).Error)
}

// seedShadowOU78 预置一条 sys_ad_ou 活行（ou_name 模型 not null,必填）。
func seedShadowOU78(t *testing.T, db *gorm.DB, configID, dn string) {
	t.Helper()
	ou := &models.ADOU{ADConfigID: configID, OUN: dn, OUName: "name-of-" + dn}
	ou.ID = uuid.NewString()
	require.NoError(t, db.Create(ou).Error)
}

// countShadowRows78 统计行数;unscoped=false 走默认作用域（活行）,true 含软删行。
func countShadowRows78(t *testing.T, db *gorm.DB, model interface{}, configID string, unscoped bool) int64 {
	t.Helper()
	q := db.Model(model).Where("ad_config_id = ?", configID)
	if unscoped {
		q = q.Unscoped()
	}
	var n int64
	require.NoError(t, q.Count(&n).Error)
	return n
}

// ============================================================================
// Task 1: user 类别 —— 两道闸门 + 软删 + 管理口径 + 增量 no-op
// ============================================================================

// TestShadowCleanup_Users_SoftDeletesVanishedRows：预置 2 活行,LDAP 返回其中 1 行
// （带 sAMAccountName）→ 另 1 行软删;Unscoped 计数不变（禁硬删）,默认作用域只剩 1 行。
func TestShadowCleanup_Users_SoftDeletesVanishedRows(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	dn1 := "CN=u1,OU=Staff,DC=example,DC=com"
	dn2 := "CN=u2,OU=Staff,DC=example,DC=com"
	seedShadowUser78(t, db, cfg.ID, dn1, "u1")
	seedShadowUser78(t, db, cfg.ID, dn2, "u2")

	entries := []*ldap.Entry{
		entry78(dn1, map[string][]string{"sAMAccountName": {"u1"}}),
	}

	result := &SyncResult{}
	require.NoError(t, svc.cleanupVanishedUsers(ctx, cfg, entries, result))

	// Unscoped 全表计数不变 —— 软删不硬删,行保留可审计（prohibition: 禁硬删）
	assert.Equal(t, int64(2), countShadowRows78(t, db, &models.ADUser{}, cfg.ID, true),
		"Unscoped 计数不变（软删非硬删）")
	// 默认作用域只剩 1 行
	assert.Equal(t, int64(1), countShadowRows78(t, db, &models.ADUser{}, cfg.ID, false))
	// 消失行 deleted_at 置位
	var u2 models.ADUser
	require.NoError(t, db.Unscoped().Where("user_dn = ? AND ad_config_id = ?", dn2, cfg.ID).First(&u2).Error)
	assert.True(t, u2.DeletedAt.Valid, "AD 端已消失的行应软删(deleted_at 非空)")
	// 存活行不受影响
	var u1 models.ADUser
	require.NoError(t, db.Where("user_dn = ? AND ad_config_id = ?", dn1, cfg.ID).First(&u1).Error)
	assert.False(t, u1.DeletedAt.Valid)
	// 可观测性计数
	assert.Equal(t, 1, result.DeletedUserCount)
}

// TestShadowCleanup_Users_RejectsEmptyLDAP：空 entries（Gate 1）→ 返回 nil 且无任何行
// deleted_at 置位 —— AD 不可达/BaseDN 配错不得伪装成「全部消失」。
func TestShadowCleanup_Users_RejectsEmptyLDAP(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	cfg := insertConfig78(t, db, "")

	seedShadowUser78(t, db, cfg.ID, "CN=a,OU=Staff,DC=example,DC=com", "a")
	seedShadowUser78(t, db, cfg.ID, "CN=b,OU=Staff,DC=example,DC=com", "b")

	result := &SyncResult{}
	require.NoError(t, svc.cleanupVanishedUsers(context.Background(), cfg, nil, result))
	require.NoError(t, svc.cleanupVanishedUsers(context.Background(), cfg, []*ldap.Entry{}, result))

	assert.Equal(t, int64(2), countShadowRows78(t, db, &models.ADUser{}, cfg.ID, false),
		"Gate 1 拒绝后所有活行原样保留")
	assert.Equal(t, 0, result.DeletedUserCount)

	var all []models.ADUser
	require.NoError(t, db.Unscoped().Where("ad_config_id = ?", cfg.ID).Find(&all).Error)
	require.Len(t, all, 2)
	for _, u := range all {
		assert.False(t, u.DeletedAt.Valid, "无任何行应被软删")
	}
}

// TestShadowCleanup_Users_RejectsOverThreshold：预置 3 活行 + LDAP 仅含 1 行 →
// 拟删 2/3（2*2=4 > 3）→ Gate 2 拒绝,无软删。
func TestShadowCleanup_Users_RejectsOverThreshold(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	cfg := insertConfig78(t, db, "")

	dn1 := "CN=u1,OU=Staff,DC=example,DC=com"
	seedShadowUser78(t, db, cfg.ID, dn1, "u1")
	seedShadowUser78(t, db, cfg.ID, "CN=u2,OU=Staff,DC=example,DC=com", "u2")
	seedShadowUser78(t, db, cfg.ID, "CN=u3,OU=Staff,DC=example,DC=com", "u3")

	entries := []*ldap.Entry{
		entry78(dn1, map[string][]string{"sAMAccountName": {"u1"}}),
	}

	result := &SyncResult{}
	require.NoError(t, svc.cleanupVanishedUsers(context.Background(), cfg, entries, result),
		"Gate 2 拒绝为非阻断:返回 nil,无错误传播")

	assert.Equal(t, int64(3), countShadowRows78(t, db, &models.ADUser{}, cfg.ID, false),
		"超阈值拒绝后全部活行保留")
	assert.Equal(t, 0, result.DeletedUserCount)
}

// TestShadowCleanup_Users_AtExactThreshold：预置 2 活行 + LDAP 仅含 1 行 →
// 恰 50% 放行,恰 1 行软删。
//
// 判式为严格大于（与先例逐字同构）:len(vanished)*2 > len(stale),即 1*2=2 > 2
// 为 FALSE → 清理放行。
func TestShadowCleanup_Users_AtExactThreshold(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	cfg := insertConfig78(t, db, "")

	dn1 := "CN=u1,OU=Staff,DC=example,DC=com"
	dn2 := "CN=u2,OU=Staff,DC=example,DC=com"
	seedShadowUser78(t, db, cfg.ID, dn1, "u1")
	seedShadowUser78(t, db, cfg.ID, dn2, "u2")

	entries := []*ldap.Entry{
		entry78(dn1, map[string][]string{"sAMAccountName": {"u1"}}),
	}

	result := &SyncResult{}
	require.NoError(t, svc.cleanupVanishedUsers(context.Background(), cfg, entries, result))

	assert.Equal(t, 1, result.DeletedUserCount, "恰 50% 放行,1 行软删")
	assert.Equal(t, int64(1), countShadowRows78(t, db, &models.ADUser{}, cfg.ID, false))
}

// TestShadowCleanup_Users_FiltersUnmanagedEntries：sAMAccountName 为空串的 entry
// 不入管理集（与 syncUsers 的 userDNs 构造同口径）→ 其对应行按已消失软删;
// 同批带用户名的 entry 正常保护其行。
//
// 预置 2 活行而非 1:单行全消失 = 100% > 50%,会被 Gate 2 保护性拒绝
// （本文件 AtExactThreshold 用例已锚定该判式）;2 行中消失 1 行恰 50% 可放行,
// 使「管理口径」的差异在放行分支内可观测。
func TestShadowCleanup_Users_FiltersUnmanagedEntries(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	cfg := insertConfig78(t, db, "")

	dnUnmanaged := "CN=ghost,OU=Staff,DC=example,DC=com"
	dnManaged := "CN=real,OU=Staff,DC=example,DC=com"
	seedShadowUser78(t, db, cfg.ID, dnUnmanaged, "ghost")
	seedShadowUser78(t, db, cfg.ID, dnManaged, "real")

	// ghost 的 entry 存在但 sAMAccountName 为空串 → 不入管理集
	entries := []*ldap.Entry{
		entry78(dnUnmanaged, map[string][]string{"displayName": {"Ghost"}}),
		entry78(dnManaged, map[string][]string{"sAMAccountName": {"real"}}),
	}

	result := &SyncResult{}
	require.NoError(t, svc.cleanupVanishedUsers(context.Background(), cfg, entries, result))

	var ghost models.ADUser
	require.NoError(t, db.Unscoped().Where("user_dn = ? AND ad_config_id = ?", dnUnmanaged, cfg.ID).First(&ghost).Error)
	assert.True(t, ghost.DeletedAt.Valid,
		"sAMAccountName 为空的 entry 不受本轮 upsert 管理,其对应行按已消失软删(清理口径=管理口径)")
	var real models.ADUser
	require.NoError(t, db.Where("user_dn = ? AND ad_config_id = ?", dnManaged, cfg.ID).First(&real).Error)
	assert.False(t, real.DeletedAt.Valid)
	assert.Equal(t, 1, result.DeletedUserCount)
}

// TestShadowCleanup_FullSyncOnly：同一 fixture 分别以增量与全量类型调编排入口 →
// 增量无软删（SHADOW-03 no-op）,全量软删（SHADOW-01 生效）。
func TestShadowCleanup_FullSyncOnly(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	dn1 := "CN=u1,OU=Staff,DC=example,DC=com"
	dn2 := "CN=u2,OU=Staff,DC=example,DC=com"
	seedShadowUser78(t, db, cfg.ID, dn1, "u1")
	seedShadowUser78(t, db, cfg.ID, dn2, "u2")

	entries := []*ldap.Entry{
		entry78(dn1, map[string][]string{"sAMAccountName": {"u1"}}),
	}

	// 增量分支:守卫短路,no-op
	resultIncr := &SyncResult{}
	require.NoError(t, svc.cleanupAfterFullSync(ctx, cfg, string(models.ADSyncTypeIncremental), nil, entries, nil, resultIncr))
	assert.Equal(t, int64(2), countShadowRows78(t, db, &models.ADUser{}, cfg.ID, false),
		"增量同步不触发清理")
	assert.Equal(t, 0, resultIncr.DeletedUserCount)

	// 全量分支:清理生效
	resultFull := &SyncResult{}
	require.NoError(t, svc.cleanupAfterFullSync(ctx, cfg, string(models.ADSyncTypeFull), nil, entries, nil, resultFull))
	assert.Equal(t, int64(1), countShadowRows78(t, db, &models.ADUser{}, cfg.ID, false),
		"全量同步末尾清理生效")
	assert.Equal(t, 1, resultFull.DeletedUserCount)
	var u2 models.ADUser
	require.NoError(t, db.Unscoped().Where("user_dn = ? AND ad_config_id = ?", dn2, cfg.ID).First(&u2).Error)
	assert.True(t, u2.DeletedAt.Valid)
}
