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
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
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

// ============================================================================
// Task 2: OU + Computer 类别 —— per-category 独立闸门
// ============================================================================

// TestShadowCleanup_OUs_SoftDeletesVanishedRows：预置 2 活 OU + LDAP 返回 1 →
// 恰 1 行软删（OU 管理集 = 全部 entry,无过滤）。
func TestShadowCleanup_OUs_SoftDeletesVanishedRows(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	cfg := insertConfig78(t, db, "")

	ouKeep := "OU=Keep,DC=example,DC=com"
	ouGone := "OU=Gone,DC=example,DC=com"
	seedShadowOU78(t, db, cfg.ID, ouKeep)
	seedShadowOU78(t, db, cfg.ID, ouGone)

	entries := []*ldap.Entry{entry78(ouKeep, nil)}

	result := &SyncResult{}
	require.NoError(t, svc.cleanupVanishedOUs(context.Background(), cfg, entries, result))

	assert.Equal(t, int64(2), countShadowRows78(t, db, &models.ADOU{}, cfg.ID, true),
		"Unscoped 计数不变（软删非硬删）")
	assert.Equal(t, int64(1), countShadowRows78(t, db, &models.ADOU{}, cfg.ID, false))

	var gone models.ADOU
	require.NoError(t, db.Unscoped().Where("ou_dn = ? AND ad_config_id = ?", ouGone, cfg.ID).First(&gone).Error)
	assert.True(t, gone.DeletedAt.Valid, "AD 端已消失的 OU 应软删")
	var keep models.ADOU
	require.NoError(t, db.Where("ou_dn = ? AND ad_config_id = ?", ouKeep, cfg.ID).First(&keep).Error)
	assert.False(t, keep.DeletedAt.Valid)
	assert.Equal(t, 1, result.DeletedOUCount)
}

// TestShadowCleanup_OUs_RejectsEmptyLDAP：空 entries（Gate 1）→ 无软删。
func TestShadowCleanup_OUs_RejectsEmptyLDAP(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	cfg := insertConfig78(t, db, "")

	seedShadowOU78(t, db, cfg.ID, "OU=A,DC=example,DC=com")
	seedShadowOU78(t, db, cfg.ID, "OU=B,DC=example,DC=com")

	result := &SyncResult{}
	require.NoError(t, svc.cleanupVanishedOUs(context.Background(), cfg, nil, result))

	assert.Equal(t, int64(2), countShadowRows78(t, db, &models.ADOU{}, cfg.ID, false),
		"Gate 1 拒绝后活行原样保留")
	assert.Equal(t, 0, result.DeletedOUCount)
}

// TestShadowCleanup_Computers_SoftDeletesVanishedRows：经 setupComp78DB 补建
// sys_ad_computer 表后,预置 2 活 computer（DistinguishedName + ComputerName 均填）+
// LDAP 返回 1（带 cn）→ 恰 1 行软删。
func TestShadowCleanup_Computers_SoftDeletesVanishedRows(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	svc := &SyncService{db: db, pool: NewAccountPool(db, nil)}
	cfg := insertConfig78(t, db, "")

	dnKeep := "CN=pc-keep,OU=Computers,DC=example,DC=com"
	dnGone := "CN=pc-gone,OU=Computers,DC=example,DC=com"
	insertComputer78(t, db, cfg.ID, dnKeep, "pc-keep", "OU=Computers,DC=example,DC=com", models.ComputerStatusOnline)
	insertComputer78(t, db, cfg.ID, dnGone, "pc-gone", "OU=Computers,DC=example,DC=com", models.ComputerStatusOnline)

	entries := []*ldap.Entry{
		entry78(dnKeep, map[string][]string{"cn": {"pc-keep"}}),
	}

	result := &SyncResult{}
	require.NoError(t, svc.cleanupVanishedComputers(context.Background(), cfg, entries, result))

	assert.Equal(t, int64(2), countShadowRows78(t, db, &models.ADComputer{}, cfg.ID, true),
		"Unscoped 计数不变（软删非硬删）")
	assert.Equal(t, int64(1), countShadowRows78(t, db, &models.ADComputer{}, cfg.ID, false))

	var gone models.ADComputer
	require.NoError(t, db.Unscoped().Where("distinguished_name = ? AND ad_config_id = ?", dnGone, cfg.ID).First(&gone).Error)
	assert.True(t, gone.DeletedAt.Valid, "AD 端已消失的电脑应软删")
	var keep models.ADComputer
	require.NoError(t, db.Where("distinguished_name = ? AND ad_config_id = ?", dnKeep, cfg.ID).First(&keep).Error)
	assert.False(t, keep.DeletedAt.Valid)
	assert.Equal(t, 1, result.DeletedComputerCount)
}

// TestShadowCleanup_Computers_RejectsOverThreshold：3 活 + LDAP 1 → 拟删 2/3
// （2*2=4 > 3）→ Gate 2 拒绝,无软删。
func TestShadowCleanup_Computers_RejectsOverThreshold(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	svc := &SyncService{db: db, pool: NewAccountPool(db, nil)}
	cfg := insertConfig78(t, db, "")

	dnKeep := "CN=pc-keep,OU=Computers,DC=example,DC=com"
	insertComputer78(t, db, cfg.ID, dnKeep, "pc-keep", "OU=Computers,DC=example,DC=com", models.ComputerStatusOnline)
	insertComputer78(t, db, cfg.ID, "CN=pc-b,OU=Computers,DC=example,DC=com", "pc-b", "OU=Computers,DC=example,DC=com", models.ComputerStatusOnline)
	insertComputer78(t, db, cfg.ID, "CN=pc-c,OU=Computers,DC=example,DC=com", "pc-c", "OU=Computers,DC=example,DC=com", models.ComputerStatusOnline)

	entries := []*ldap.Entry{
		entry78(dnKeep, map[string][]string{"cn": {"pc-keep"}}),
	}

	result := &SyncResult{}
	require.NoError(t, svc.cleanupVanishedComputers(context.Background(), cfg, entries, result),
		"Gate 2 拒绝为非阻断:返回 nil,无错误传播")

	assert.Equal(t, int64(3), countShadowRows78(t, db, &models.ADComputer{}, cfg.ID, false),
		"超阈值拒绝后全部活行保留")
	assert.Equal(t, 0, result.DeletedComputerCount)
}

// TestShadowCleanup_Computers_FiltersUnmanagedEntries：cn 为空串的 entry 不入管理集
// （与 syncComputers 跳过口径同源）→ 其对应行按已消失软删。
//
// 预置 2 活行而非 1:单行全消失 = 100% > 50%,会被 Gate 2 保护性拒绝
// （Users_AtExactThreshold 已锚定该判式）;2 行中消失 1 行恰 50% 可放行,
// 使「管理口径」的差异在放行分支内可观测。
func TestShadowCleanup_Computers_FiltersUnmanagedEntries(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	svc := &SyncService{db: db, pool: NewAccountPool(db, nil)}
	cfg := insertConfig78(t, db, "")

	dnUnmanaged := "CN=ghost-pc,OU=Computers,DC=example,DC=com"
	dnManaged := "CN=real-pc,OU=Computers,DC=example,DC=com"
	insertComputer78(t, db, cfg.ID, dnUnmanaged, "ghost-pc", "OU=Computers,DC=example,DC=com", models.ComputerStatusOnline)
	insertComputer78(t, db, cfg.ID, dnManaged, "real-pc", "OU=Computers,DC=example,DC=com", models.ComputerStatusOnline)

	// ghost-pc 的 entry 存在但 cn 为空串 → 不入管理集
	entries := []*ldap.Entry{
		entry78(dnUnmanaged, map[string][]string{"description": {"ghost"}}),
		entry78(dnManaged, map[string][]string{"cn": {"real-pc"}}),
	}

	result := &SyncResult{}
	require.NoError(t, svc.cleanupVanishedComputers(context.Background(), cfg, entries, result))

	var ghost models.ADComputer
	require.NoError(t, db.Unscoped().Where("distinguished_name = ? AND ad_config_id = ?", dnUnmanaged, cfg.ID).First(&ghost).Error)
	assert.True(t, ghost.DeletedAt.Valid,
		"cn 为空的 entry 不受本轮 upsert 管理,其对应行按已消失软删(清理口径=管理口径)")
	var real models.ADComputer
	require.NoError(t, db.Where("distinguished_name = ? AND ad_config_id = ?", dnManaged, cfg.ID).First(&real).Error)
	assert.False(t, real.DeletedAt.Valid)
	assert.Equal(t, 1, result.DeletedComputerCount)
}

// ============================================================================
// Phase 61 Plan 02: 软删行复活（SHADOW-01 配套）——四类 upsert 路径自愈用例
// ============================================================================

// TestSync_ResurrectOU_SameDN：软删 OU + 同 DN entry 在 AD 复现 → syncOUs 返回 nil、
// 行恢复可见（deleted_at 置空）、OUName 已更新、Unscoped 计数不变（复活而非新建）。
//
// 清理（61-01）软删行后，唯一约束 UNIQUE(ad_config_id, ou_dn) 不含 deleted_at：
// 复活缺失时软删行走 Create 分支撞约束 → 整轮同步失败（Pitfall 1）。
func TestSync_ResurrectOU_SameDN(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	ouDN := "OU=B,DC=example,DC=com"
	ou := &models.ADOU{ADConfigID: cfg.ID, OUN: ouDN, OUName: "OldName"}
	ou.ID = uuid.NewString()
	require.NoError(t, db.Create(ou).Error)
	require.NoError(t, db.Delete(ou).Error) // 软删（61-01 清理语义的产物）

	entries := []*ldap.Entry{
		entry78(ouDN, map[string][]string{"ou": {"B-Revived"}}),
	}

	require.NoError(t, svc.syncOUs(ctx, cfg, entries),
		"软删行同 DN 在 AD 复现 → syncOUs 应自愈复活,而非裸 Create 撞 UNIQUE")

	// 默认作用域可见（deleted_at 已置空）
	var row models.ADOU
	require.NoError(t, db.Where("ou_dn = ? AND ad_config_id = ?", ouDN, cfg.ID).First(&row).Error)
	assert.Equal(t, "B-Revived", row.OUName, "复活行字段已按 entry 更新")
	// Unscoped 计数不变 —— 复活而非新建
	assert.Equal(t, int64(1), countShadowRows78(t, db, &models.ADOU{}, cfg.ID, true),
		"Unscoped 计数不变:同 DN 复现走 update 复活,不产生新行")
}

// TestSync_ResurrectGroup_SameDN：软删组 + 同 DN entry 在 AD 复现 → syncGroups 返回
// nil、行恢复可见、Unscoped 计数不变（证明裸 Create 不再收到复活行,存量潜伏缺陷一并修复）。
func TestSync_ResurrectGroup_SameDN(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	gDN := "CN=g1,OU=Groups,DC=example,DC=com"
	g := &models.ADGroup{ADConfigID: cfg.ID, GroupDN: gDN, GroupName: "g1"}
	g.ID = uuid.NewString()
	require.NoError(t, db.Create(g).Error)
	require.NoError(t, db.Delete(g).Error) // 软删（handleDeletedGroups 存量行为）

	entries := []*ldap.Entry{
		entry78(gDN, map[string][]string{"cn": {"g1"}}),
	}

	require.NoError(t, svc.syncGroups(ctx, cfg, entries),
		"软删组同 DN 复现 → syncGroups 应复活该组,而非 createGroupsInBatches 撞 UNIQUE(ad_config_id, group_dn)")

	var row models.ADGroup
	require.NoError(t, db.Where("group_dn = ? AND ad_config_id = ?", gDN, cfg.ID).First(&row).Error)
	assert.False(t, row.DeletedAt.Valid, "软删组同 DN 复现 → 行恢复可见")
	assert.Equal(t, int64(1), countShadowRows78(t, db, &models.ADGroup{}, cfg.ID, true),
		"Unscoped 计数不变:复活而非新建")
}

// TestSync_ResurrectComputer_SameDN：软删 computer（DN-A/cn=X）+ LDAP 返回同 DN 同 cn →
// syncComputers 返回 nil、行恢复可见、Unscoped 计数 1。
//
// 现状缺陷形态：existing 两查（queryExistingComputers 默认作用域 /
// queryAllComputerNames 显式 deleted_at IS NULL）都查不到软删行 → Create 分支 →
// 撞 UNIQUE(ad_config_id, computer_name) → 整轮同步失败。
func TestSync_ResurrectComputer_SameDN(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")
	compSvc := NewComputerService(db)

	dn := "CN=pc-x,OU=Computers,DC=example,DC=com"
	insertComputer78(t, db, cfg.ID, dn, "pc-x", "OU=Computers,DC=example,DC=com", models.ComputerStatusOnline)
	require.NoError(t, db.Exec(`UPDATE sys_ad_computer SET deleted_at = ? WHERE computer_name = ? AND ad_config_id = ?`,
		time.Now(), "pc-x", cfg.ID).Error) // 软删（61-01 清理语义的产物）

	entries := []*ldap.Entry{
		entry78(dn, map[string][]string{"cn": {"pc-x"}}),
	}

	require.NoError(t, compSvc.syncComputers(ctx, cfg, entries),
		"软删 computer 同 DN 复现 → syncComputers 应自愈复活,而非 Create 撞 UNIQUE")

	var row models.ADComputer
	require.NoError(t, db.Where("distinguished_name = ? AND ad_config_id = ?", dn, cfg.ID).First(&row).Error)
	assert.False(t, row.DeletedAt.Valid, "软删行恢复可见(deleted_at 置空)")
	assert.Equal(t, int64(1), countShadowRows78(t, db, &models.ADComputer{}, cfg.ID, true),
		"Unscoped 计数不变:复活而非新建")
}

// TestSync_ResurrectComputer_SameNameNewDN：软删行 DN-A/cn=X + LDAP 返回 DN-B/cn=X →
// name-map 命中走 update 分支 → 行复活且 distinguished_name 更新为 DN-B（OU 搬迁场景）。
func TestSync_ResurrectComputer_SameNameNewDN(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")
	compSvc := NewComputerService(db)

	oldDN := "CN=pc-x,OU=Old,DC=example,DC=com"
	newDN := "CN=pc-x,OU=New,DC=example,DC=com"
	insertComputer78(t, db, cfg.ID, oldDN, "pc-x", "OU=Old,DC=example,DC=com", models.ComputerStatusOnline)
	require.NoError(t, db.Exec(`UPDATE sys_ad_computer SET deleted_at = ? WHERE computer_name = ? AND ad_config_id = ?`,
		time.Now(), "pc-x", cfg.ID).Error) // 软删

	entries := []*ldap.Entry{
		entry78(newDN, map[string][]string{"cn": {"pc-x"}}),
	}

	require.NoError(t, compSvc.syncComputers(ctx, cfg, entries),
		"同名软删行经 name-map 命中 update 分支复活,而非 Create 撞 UNIQUE(ad_config_id, computer_name)")

	var row models.ADComputer
	require.NoError(t, db.Where("computer_name = ? AND ad_config_id = ?", "pc-x", cfg.ID).First(&row).Error)
	assert.False(t, row.DeletedAt.Valid, "同名软删行复活(deleted_at 置空)")
	assert.Equal(t, newDN, row.DistinguishedName, "复活行 distinguished_name 已更新为新 DN")
	assert.Equal(t, int64(1), countShadowRows78(t, db, &models.ADComputer{}, cfg.ID, true),
		"Unscoped 计数不变:复活而非新建")
}

// TestSync_ResurrectGroup_IncrementalPath：软删组 + 同 DN entry → syncGroupEntries
// （增量路径,GroupSyncService 私有方法同包直调）返回 nil、行复活 —— 增量路径的复活证明。
func TestSync_ResurrectGroup_IncrementalPath(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	gDN := "CN=g-inc,OU=Groups,DC=example,DC=com"
	g := &models.ADGroup{ADConfigID: cfg.ID, GroupDN: gDN, GroupName: "g-inc"}
	g.ID = uuid.NewString()
	require.NoError(t, db.Create(g).Error)
	require.NoError(t, db.Delete(g).Error) // 软删（handleDeletedGroups 增量存量行为）

	groupSyncSvc := &GroupSyncService{db: db, syncService: svc}
	entries := []*ldap.Entry{
		entry78(gDN, map[string][]string{"cn": {"g-inc"}}),
	}
	result := &GroupSyncResult{}

	require.NoError(t, groupSyncSvc.syncGroupEntries(ctx, cfg, entries, result),
		"增量路径软删组同 DN 复现 → 应复活,而非裸 Create 撞 UNIQUE(ad_config_id, group_dn)")

	var row models.ADGroup
	require.NoError(t, db.Where("group_dn = ? AND ad_config_id = ?", gDN, cfg.ID).First(&row).Error)
	assert.False(t, row.DeletedAt.Valid, "增量路径软删组复活(deleted_at 置空)")
	assert.Equal(t, int64(1), countShadowRows78(t, db, &models.ADGroup{}, cfg.ID, true),
		"Unscoped 计数不变:复活而非新建")
}

// TestShadowCleanup_CategoriesIndependent：users entries 传空、ous/computers 正常 →
// 仅 user 类被 Gate 1 拒绝,OU 与 computer 照常软删（per-category 独立性断言）。
func TestShadowCleanup_CategoriesIndependent(t *testing.T) {
	ctx := context.Background()
	db := setupComp78DB(t)
	defer closeDB(t, db)
	svc := &SyncService{db: db, pool: NewAccountPool(db, nil)}
	cfg := insertConfig78(t, db, "")

	// 预置:1 活 user + 2 活 OU + 2 活 computer
	userDN := "CN=u1,OU=Staff,DC=example,DC=com"
	seedShadowUser78(t, db, cfg.ID, userDN, "u1")
	ouKeep := "OU=Keep,DC=example,DC=com"
	ouGone := "OU=Gone,DC=example,DC=com"
	seedShadowOU78(t, db, cfg.ID, ouKeep)
	seedShadowOU78(t, db, cfg.ID, ouGone)
	pcKeep := "CN=pc-keep,OU=Computers,DC=example,DC=com"
	pcGone := "CN=pc-gone,OU=Computers,DC=example,DC=com"
	insertComputer78(t, db, cfg.ID, pcKeep, "pc-keep", "OU=Computers,DC=example,DC=com", models.ComputerStatusOnline)
	insertComputer78(t, db, cfg.ID, pcGone, "pc-gone", "OU=Computers,DC=example,DC=com", models.ComputerStatusOnline)

	entries := []*ldap.Entry{
		entry78(ouKeep, nil),
		entry78(pcKeep, map[string][]string{"cn": {"pc-keep"}}),
	}

	result := &SyncResult{}
	require.NoError(t, svc.cleanupAfterFullSync(ctx, cfg, string(models.ADSyncTypeFull),
		entries, nil, entries, result),
		"user 类拒绝不阻断编排入口,整体无错误传播")

	// user 类:Gate 1 拒绝 → 行保留、计数 0
	assert.Equal(t, int64(1), countShadowRows78(t, db, &models.ADUser{}, cfg.ID, false),
		"空 user entries 只拒绝 user 类清理")
	assert.Equal(t, 0, result.DeletedUserCount)
	// OU 类:照常软删 1 行
	assert.Equal(t, int64(1), countShadowRows78(t, db, &models.ADOU{}, cfg.ID, false))
	assert.Equal(t, 1, result.DeletedOUCount)
	var ouGoneRow models.ADOU
	require.NoError(t, db.Unscoped().Where("ou_dn = ? AND ad_config_id = ?", ouGone, cfg.ID).First(&ouGoneRow).Error)
	assert.True(t, ouGoneRow.DeletedAt.Valid)
	// computer 类:照常软删 1 行
	assert.Equal(t, int64(1), countShadowRows78(t, db, &models.ADComputer{}, cfg.ID, false))
	assert.Equal(t, 1, result.DeletedComputerCount)
	var pcGoneRow models.ADComputer
	require.NoError(t, db.Unscoped().Where("distinguished_name = ? AND ad_config_id = ?", pcGone, cfg.ID).First(&pcGoneRow).Error)
	assert.True(t, pcGoneRow.DeletedAt.Valid)
}
