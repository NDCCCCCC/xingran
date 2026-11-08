package system

// Phase 134 / FIX-01：sys 侧恢复 / 状态 1→0 启用时 AD push enable 入队的 service 接线用例。
//
// 覆盖：
//  1. UserService.UpdateStatus 1→0：同事务入队 enable（业务写与队列行同生共死）；
//  2. UserService.Update 1→0（编辑表单）：同事务入队 enable；
//  3. 1→0 入队失败 → 业务启用整体回滚（D-02 第二层 / QUEUE-01 红线）；
//  4. 软删恢复（SyncUserFromAD 登录路径）：一致 DN → 恢复 + 同事务入队 enable；
//  5. 软删恢复身份不一致：SAFE-01 拒绝 → 不入队（既有 Phase 60 断言用 Skipped/Error 覆盖）；
//  6. 批量恢复（BatchSyncADUsers 阶段 5c）：一致 DN → 恢复 + 入队 enable（SAFE-02 双路径一致）；
//  7. disable→enable 最新意图胜出在 service 层的端到端体现。
//
// 复用：setupPhase57QueueDB / seedADConfigEnabled / seedUserRow / newTestUserService /
// countRowsByAction / countQueueRows / strPtr（system 包既有测试基座）。

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/core/db/migrations"
	"github.com/xingran-next/xingran-go-backend/internal/core/security"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
)

// ==================== 1. UpdateStatus 1→0 ====================

func TestPhase134_UpdateStatus_Reenable_EnqueuesEnable(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)
	userID := seedUserRow(t, db, "phase134-reenable-status", 1) // 已停用
	svc := newTestUserService(t, db)

	require.NoError(t, svc.UpdateStatus(context.Background(), userID, int(models.UserStatusEnabled)))

	assert.Equal(t, int64(1), countRowsByAction(t, db, userID, models.ADPushActionEnable),
		"状态 1→0 启用必须同事务入队 enable")
	assert.Equal(t, int64(0), countRowsByAction(t, db, userID, models.ADPushActionDisable))

	var row models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ?", userID).First(&row).Error)
	assert.Equal(t, models.ADPushStatusPending, row.Status)
	assert.Equal(t, models.ADPushActionEnable, row.ActionType)

	var status int
	require.NoError(t, db.Raw("SELECT status FROM sys_user WHERE id = ?", userID).Scan(&status).Error)
	assert.Equal(t, int(models.UserStatusEnabled), status, "业务启用与队列行同生共死")
}

func TestPhase134_UpdateStatus_Reenable_RollsBackWhenQueueInsertFails(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)
	userID := seedUserRow(t, db, "phase134-reenable-rollback", 1)
	svc := newTestUserService(t, db)
	require.NoError(t, db.Exec("DROP TABLE ad_push_queue").Error)

	err := svc.UpdateStatus(context.Background(), userID, int(models.UserStatusEnabled))
	require.Error(t, err, "队列表不可写时启用必须整体失败")

	var status int
	require.NoError(t, db.Raw("SELECT status FROM sys_user WHERE id = ?", userID).Scan(&status).Error)
	assert.Equal(t, int(models.UserStatusDisabled), status, "事务必须回滚：用户不得被启用")

	require.NoError(t, migrations.Migrate215CreateADPushQueue(db))
}

func TestPhase134_UpdateStatus_Reenable_NoADConfig_NoEnqueue(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 1) // AD 配置停用
	userID := seedUserRow(t, db, "phase134-reenable-nocfg", 1)
	svc := newTestUserService(t, db)

	require.NoError(t, svc.UpdateStatus(context.Background(), userID, int(models.UserStatusEnabled)),
		"无启用配置时状态变更必须放行")
	assert.Equal(t, int64(0), countQueueRows(t, db, userID), "无启用配置时不应入队")
}

// ==================== 2. Update（编辑表单）1→0 ====================

func TestPhase134_Update_Reenable_EnqueuesEnable(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)
	userID := seedUserRow(t, db, "phase134-reenable-form", 1)
	svc := newTestUserService(t, db)

	require.NoError(t, svc.Update(context.Background(), &requests.UserUpdateRequest{
		ID: userID, Status: models.UserStatusEnabled,
	}))

	assert.Equal(t, int64(1), countRowsByAction(t, db, userID, models.ADPushActionEnable),
		"编辑表单把停用用户改回启用必须同事务入队 enable")
	assert.Equal(t, int64(0), countRowsByAction(t, db, userID, models.ADPushActionDisable))
}

// ==================== 3. 最新意图胜出（service 层端到端） ====================

func TestPhase134_DisableThenReenable_LatestIntentIsEnable(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)
	userID := seedUserRow(t, db, "phase134-latest-intent", 0)
	svc := newTestUserService(t, db)

	// 先停用 → 入队 disable
	require.NoError(t, svc.UpdateStatus(context.Background(), userID, int(models.UserStatusDisabled)))
	require.Equal(t, int64(1), countRowsByAction(t, db, userID, models.ADPushActionDisable))

	// 再启用 → 入队 enable，并压制上一条活跃 disable（dead 终态）
	require.NoError(t, svc.UpdateStatus(context.Background(), userID, int(models.UserStatusEnabled)))
	require.Equal(t, int64(1), countRowsByAction(t, db, userID, models.ADPushActionEnable))

	var oldDisable models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ? AND action_type = ?", userID, models.ADPushActionDisable).First(&oldDisable).Error)
	assert.Equal(t, models.ADPushStatusDead, oldDisable.Status,
		"先停用后启用：旧 disable 必须被压制，杜绝恢复后仍被禁用的竞态")

	var activeEnable int64
	require.NoError(t, db.Model(&models.ADPushQueue{}).
		Where("entity_id = ? AND action_type = ? AND status IN ?", userID, models.ADPushActionEnable,
			[]string{string(models.ADPushStatusPending), string(models.ADPushStatusFailed)}).
		Count(&activeEnable).Error)
	assert.Equal(t, int64(1), activeEnable, "enable 必须是唯一活跃意图")
}

// ==================== 4. 软删恢复（登录路径 SAFE-01） ====================

func TestPhase134_Restore_MatchingDN_EnqueuesEnable(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	const stored = "CN=restore,OU=IT,DC=corp,DC=com"
	const current = "cn=RESTORE,ou=it,dc=CORP,dc=COM"
	userID := seedUserRow(t, db, "phase134_restore", 0)
	// 置为软删 + 存量 ad_dn
	require.NoError(t, db.Exec(
		`UPDATE sys_user SET deleted_at = '2024-01-01 00:00:00', ad_dn = ? WHERE id = ?`,
		stored, userID).Error)

	svc := NewUserSyncService(db, security.NewPasswordManager(nil), nil)
	_, _, err := svc.SyncUserFromAD(context.Background(), phase60ADUser("phase134_restore", current), "")
	require.NoError(t, err)

	var after models.User
	require.NoError(t, db.First(&after, "id = ?", userID).Error)
	assert.False(t, after.DeletedAt.Valid, "软删必须被解除")

	assert.Equal(t, int64(1), countRowsByAction(t, db, userID, models.ADPushActionEnable),
		"软删恢复成功必须同事务入队 enable（AD 端此前被 UAC 514 禁用）")
}

func TestPhase134_Restore_MismatchedDN_NoEnable(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	const stored = "CN=restore-drift,OU=IT,DC=corp,DC=com"
	const current = "CN=someone-else,OU=IT,DC=corp,DC=com"
	userID := seedUserRow(t, db, "phase134_restore_drift", 0)
	require.NoError(t, db.Exec(
		`UPDATE sys_user SET deleted_at = '2024-01-01 00:00:00', ad_dn = ?, ad_username = ? WHERE id = ?`,
		stored, "phase134_restore_drift", userID).Error)

	svc := NewUserSyncService(db, security.NewPasswordManager(nil), nil)
	_, _, err := svc.SyncUserFromAD(context.Background(), phase60ADUser("phase134_restore_drift", current), "")
	require.Error(t, err, "身份不一致的恢复必须被 SAFE-01 拒绝")

	assert.Equal(t, int64(0), countQueueRows(t, db, userID),
		"身份不一致的恢复绝不能入队 enable（防止把 ad_dn 已漂移的账号重新启用）")
	var after models.User
	require.NoError(t, db.Unscoped().First(&after, "id = ?", userID).Error)
	assert.True(t, after.DeletedAt.Valid, "被拒后必须保持软删（D-03 零写入）")
}

// ==================== 5. 批量恢复（SAFE-02 双路径一致性） ====================

func TestPhase134_BatchRestore_MatchingDN_EnqueuesEnable(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	const stored = "CN=batch-restore,OU=IT,DC=corp,DC=com"
	const current = "CN=BATCH-RESTORE,OU=IT,DC=corp,DC=com"
	userID := seedUserRow(t, db, "phase134_batch_restore", 0)
	require.NoError(t, db.Exec(
		`UPDATE sys_user SET deleted_at = '2024-01-01 00:00:00', ad_dn = ?, ad_username = ? WHERE id = ?`,
		stored, "phase134_batch_restore", userID).Error)

	svc := NewUserSyncService(db, security.NewPasswordManager(nil), nil)
	res, err := svc.BatchSyncADUsers(context.Background(), []*ADUserInfoForSync{
		phase60ADUser("phase134_batch_restore", current),
	}, "")
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 1, res.Success)

	assert.Equal(t, int64(1), countRowsByAction(t, db, userID, models.ADPushActionEnable),
		"批量恢复必须与登录恢复路径对称地入队 enable")
}
