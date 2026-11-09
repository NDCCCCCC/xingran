//go:build !skip_db_tests
// +build !skip_db_tests

package addomain

// Phase 134 / FIX-01：AD push 队列 enable 方向（入队 + 压制 + 身份校验 + 排水）sqlite 用例。
//
// 覆盖（对应 REQUIREMENTS.md FIX-01 验收口径）：
//  1. adPushOppositeAction 纯函数：enable↔disable 互认相反意图，其余动作无互斥（单测可红断言）；
//  2. EnqueueEnableOnStatusTransition：仅 1→0 入队 enable；D-02 无配置降级；
//  3. EnqueueEnableOnRestore：SAFE-01 身份校验（一致放行入队 / 不一致不入队+WARN / 空存量 DN 放行）；
//  4. F-01 压制「最新意图胜出」：enable 压制同实体活跃 disable；disable 压制同实体活跃 enable
//     （双向），终态行退出唯一谓词、不打回活跃集合；
//  5. drainRow enable 分支：真实调用 EnableUser（UAC 544）并转 pushed；AD 失败转 failed 退避；
//     软删用户（恢复后 ad_dn 仍在）的 enable 行也能推出去（Unscoped）。
//
// 复用：setupEnqueueTestDB / seedEnabledConfig / countQueueRows（ad_push_queue_enqueue_test.go），
// setupDrainTestDB / seedDrainADConfig / seedDrainUser / seedDrainQueueRow / newDrainTestDrainer /
// reloadDrainRow / newDrainTestLDAP / drainTestAccount（ad_push_queue_drain_test.go）。
// ⚠ 排水用例所在文件带 !skip_db_tests build tag（依赖 fakePool78），故本文件同样携带该 tag。

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ==================== 1. 相反意图纯函数 ====================

func TestPhase134_OppositeAction(t *testing.T) {
	assert.Equal(t, models.ADPushActionDisable, adPushOppositeAction(models.ADPushActionEnable))
	assert.Equal(t, models.ADPushActionEnable, adPushOppositeAction(models.ADPushActionDisable))
	// 其余动作与 disable/enable 无互斥关系，保持 v1.21 既有语义
	assert.Equal(t, models.ADPushActionType(""), adPushOppositeAction(models.ADPushActionUpdateUser))
	assert.Equal(t, models.ADPushActionType(""), adPushOppositeAction(models.ADPushActionUpdateManager))
	assert.Equal(t, models.ADPushActionType(""), adPushOppositeAction("unknown"))
}

// ==================== 2. 状态 1→0 入队 enable ====================

func TestPhase134_EnqueueEnableOnStatusTransition(t *testing.T) {
	db := setupEnqueueTestDB(t)
	e := NewADPushEnqueuer()

	// 无启用配置 → D-02 降级放行，不入队
	require.NoError(t, e.EnqueueEnableOnStatusTransition(db, models.UserStatusDisabled, models.UserStatusEnabled, "u-1"))
	assert.Equal(t, int64(0), countQueueRows(t, db, models.ADPushActionEnable, "u-1"),
		"无启用 AD 配置时启用入队必须 WARN 放行、不入队")

	seedEnabledConfig(t, db)

	// 空 userID / 状态不变 / 0→1 停用方向 → 全部不入队 enable
	require.NoError(t, e.EnqueueEnableOnStatusTransition(db, models.UserStatusDisabled, models.UserStatusEnabled, ""))
	require.NoError(t, e.EnqueueEnableOnStatusTransition(db, models.UserStatusEnabled, models.UserStatusEnabled, "u-1"))
	require.NoError(t, e.EnqueueEnableOnStatusTransition(db, models.UserStatusDisabled, models.UserStatusDisabled, "u-1"))
	require.NoError(t, e.EnqueueEnableOnStatusTransition(db, models.UserStatusEnabled, models.UserStatusDisabled, "u-1"))
	assert.Equal(t, int64(0), countQueueRows(t, db, models.ADPushActionEnable, "u-1"),
		"仅 1→0 才入队 enable")

	// 1→0 → 一行 pending enable
	require.NoError(t, e.EnqueueEnableOnStatusTransition(db, models.UserStatusDisabled, models.UserStatusEnabled, "u-1"))
	assert.Equal(t, int64(1), countQueueRows(t, db, models.ADPushActionEnable, "u-1"))
	var row models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ? AND action_type = ?", "u-1", models.ADPushActionEnable).First(&row).Error)
	assert.Equal(t, models.ADPushStatusPending, row.Status)

	// 重复入队合并为一行（ON CONFLICT）
	require.NoError(t, e.EnqueueEnableOnStatusTransition(db, models.UserStatusDisabled, models.UserStatusEnabled, "u-1"))
	assert.Equal(t, int64(1), countQueueRows(t, db, models.ADPushActionEnable, "u-1"))
}

// ==================== 3. SAFE-01 身份校验（恢复入口） ====================

func TestPhase134_CheckRestoreIdentityForEnable(t *testing.T) {
	const dn = "CN=alice,OU=IT,DC=corp,DC=com"
	tests := []struct {
		name      string
		storedDN  *string
		currentDN string
		want      bool
	}{
		{"存量 DN 为 nil 放行", nil, dn, true},
		{"存量 DN 空串放行", strPtr(""), dn, true},
		{"存量 DN 纯空白放行", strPtr("   "), dn, true},
		{"完全一致放行", strPtr(dn), dn, true},
		{"大小写/段内空格差异放行", strPtr("cn=ALICE, ou=it, dc=CORP, dc=COM"), dn, true},
		{"不同 DN 拒绝", strPtr("CN=other,OU=IT,DC=corp,DC=com"), dn, false},
		{"存量非空但当前 DN 为空拒绝", strPtr(dn), "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &models.User{AdDn: tt.storedDN}
			assert.Equal(t, tt.want, CheckRestoreIdentityForEnable(user, tt.currentDN, "login"))
		})
	}
	// nil user 防御：不得 panic，视作无存量身份 → 放行
	assert.True(t, CheckRestoreIdentityForEnable(nil, dn, "login"))
}

func TestPhase134_EnqueueEnableOnRestore_IdentityGate(t *testing.T) {
	db := setupEnqueueTestDB(t)
	seedEnabledConfig(t, db)
	e := NewADPushEnqueuer()
	const dn = "CN=alice,OU=IT,DC=corp,DC=com"

	// A) 一致 → 入队 enable（path=batch 与 login 同口径）
	ok := &models.User{Username: "alice", AdDn: strPtr("cn=ALICE, ou=it, dc=CORP, dc=COM")}
	ok.ID = "u-ok"
	require.NoError(t, e.EnqueueEnableOnRestore(db, ok, dn, "batch"))
	assert.Equal(t, int64(1), countQueueRows(t, db, models.ADPushActionEnable, "u-ok"))

	// B) 身份不一致（ad_dn 已漂移）→ 不入队 + 记安全 WARN，且不回滚/不报错（策略拒绝非故障）
	drifted := &models.User{Username: "bob", AdDn: strPtr("CN=someone-else,OU=IT,DC=corp,DC=com")}
	drifted.ID = "u-drift"
	require.NoError(t, e.EnqueueEnableOnRestore(db, drifted, dn, "batch"))
	assert.Equal(t, int64(0), countQueueRows(t, db, models.ADPushActionEnable, "u-drift"),
		"身份不一致的恢复绝不能入队 enable（防止把 ad_dn 已漂移的账号重新启用）")

	// C) 空存量 DN → 放行入队（无身份可比对，D-01）
	empty := &models.User{Username: "carol"}
	empty.ID = "u-empty"
	require.NoError(t, e.EnqueueEnableOnRestore(db, empty, dn, "login"))
	assert.Equal(t, int64(1), countQueueRows(t, db, models.ADPushActionEnable, "u-empty"))

	// D) nil user / 空 ID → no-op，不 panic
	require.NoError(t, e.EnqueueEnableOnRestore(db, nil, dn, "login"))
	require.NoError(t, e.EnqueueEnableOnRestore(db, &models.User{}, dn, "login"))
}

// ==================== 4. F-01 压制「最新意图胜出」（双向） ====================

// TestPhase134_Enable_SuppressesPendingDisable 恢复入队 enable 时压制同实体活跃 disable 行：
// 复现 seed §5 竞态「先 disable 后恢复却仍被禁用」——压制后排水器只会看到 enable。
func TestPhase134_Enable_SuppressesPendingDisable(t *testing.T) {
	db := setupEnqueueTestDB(t)
	seedEnabledConfig(t, db)
	e := NewADPushEnqueuer()

	// 预置 u-1 的活跃 disable（pending）与 u-2 的活跃 disable（failed）
	require.NoError(t, db.Create(&models.ADPushQueue{EntityType: models.ADPushEntityUser, EntityID: "u-1", ActionType: models.ADPushActionDisable, Status: models.ADPushStatusPending}).Error)
	require.NoError(t, db.Create(&models.ADPushQueue{EntityType: models.ADPushEntityUser, EntityID: "u-2", ActionType: models.ADPushActionDisable, Status: models.ADPushStatusFailed, Attempts: 2, LastError: "boom"}).Error)
	// 预置 u-3 的已终态 disable（pushed，不应被压制改动）
	require.NoError(t, db.Create(&models.ADPushQueue{EntityType: models.ADPushEntityUser, EntityID: "u-3", ActionType: models.ADPushActionDisable, Status: models.ADPushStatusPushed}).Error)

	// 三条恢复（u-1/u-2/u-3）分别入队 enable
	for _, id := range []string{"u-1", "u-2", "u-3"} {
		u := &models.User{Username: id, AdDn: strPtr("CN=" + id + ",DC=corp,DC=com")}
		u.ID = id
		require.NoError(t, e.EnqueueEnableOnRestore(db, u, "CN="+id+",DC=corp,DC=com", "login"))
	}

	for _, id := range []string{"u-1", "u-2"} {
		var old models.ADPushQueue
		require.NoError(t, db.Where("entity_id = ? AND action_type = ?", id, models.ADPushActionDisable).First(&old).Error)
		assert.Equal(t, models.ADPushStatusDead, old.Status, "%s 的活跃 disable 必须被打为 dead 终态", id)
		assert.Equal(t, adPushSupersededByEnableReason, old.LastError)
		assert.Nil(t, old.NextRetryAt)
		// enable 行成为唯一活跃意图
		assert.Equal(t, int64(1), countQueueRows(t, db, models.ADPushActionEnable, id))
	}
	// 终态行不被压制
	var terminal models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ? AND action_type = ?", "u-3", models.ADPushActionDisable).First(&terminal).Error)
	assert.Equal(t, models.ADPushStatusPushed, terminal.Status, "pushed 终态行不得被压制改动")
}

// TestPhase134_Disable_SuppressesPendingEnable 反向：停用/软删入队 disable 时压制活跃 enable 行。
func TestPhase134_Disable_SuppressesPendingEnable(t *testing.T) {
	db := setupEnqueueTestDB(t)
	seedEnabledConfig(t, db)
	e := NewADPushEnqueuer()

	u := &models.User{Username: "dave", AdDn: strPtr("CN=dave,DC=corp,DC=com")}
	u.ID = "u-1"
	require.NoError(t, e.EnqueueEnableOnRestore(db, u, "CN=dave,DC=corp,DC=com", "login"))
	require.Equal(t, int64(1), countQueueRows(t, db, models.ADPushActionEnable, "u-1"))

	// 紧接着软删 → EnqueueDisable（批量路径，含 F-01 压制）
	require.NoError(t, e.EnqueueDisable(db, []string{"u-1"}))

	var oldEnable models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ? AND action_type = ?", "u-1", models.ADPushActionEnable).First(&oldEnable).Error)
	assert.Equal(t, models.ADPushStatusDead, oldEnable.Status, "恢复入队 enable → 立刻又停用：旧 enable 必须被压制")
	assert.Equal(t, adPushSupersededByDisableReason, oldEnable.LastError)
	assert.Equal(t, int64(1), countQueueRows(t, db, models.ADPushActionDisable, "u-1"), "disable 成为唯一活跃意图")
}

// TestPhase134_LatestIntentWins_EndToEnd 端到端行序语义：
// disable → enable → 队列中该实体只剩一条 pending enable；二次 disable → 只剩 pending disable。
func TestPhase134_LatestIntentWins_EndToEnd(t *testing.T) {
	db := setupEnqueueTestDB(t)
	seedEnabledConfig(t, db)
	e := NewADPushEnqueuer()
	const id = "u-1"

	require.NoError(t, e.EnqueueDisable(db, []string{id}))
	require.NoError(t, e.EnqueueEnableOnStatusTransition(db, models.UserStatusDisabled, models.UserStatusEnabled, id))
	assert.Equal(t, int64(0), countActiveRows(t, db, id, models.ADPushActionDisable), "旧 disable 必须退出活跃集合")
	assert.Equal(t, int64(1), countActiveRows(t, db, id, models.ADPushActionEnable), "enable 是最新意图")

	require.NoError(t, e.EnqueueDisableOnStatusTransition(db, models.UserStatusEnabled, models.UserStatusDisabled, id))
	assert.Equal(t, int64(0), countActiveRows(t, db, id, models.ADPushActionEnable), "旧 enable 必须退出活跃集合")
	assert.Equal(t, int64(1), countActiveRows(t, db, id, models.ADPushActionDisable), "disable 是最新意图")
}

// countActiveRows 统计某实体 + 动作处于活跃谓词（pending/failed）内的行数。
func countActiveRows(t *testing.T, db *gorm.DB, entityID string, action models.ADPushActionType) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&models.ADPushQueue{}).
		Where("entity_type = ? AND entity_id = ? AND action_type = ? AND status IN ?",
			models.ADPushEntityUser, entityID, action,
			[]string{string(models.ADPushStatusPending), string(models.ADPushStatusFailed)}).
		Count(&n).Error)
	return n
}

// ==================== 5. 排水器 enable 分支 ====================

func TestPhase134_Drain_EnableSuccessCallsEnableUser(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainUser(t, db, "u-1", "alice", "CN=alice,OU=研发部,DC=corp,DC=com", "OU=研发部,DC=corp,DC=com", nil)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionEnable), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	assert.Equal(t, 1, mock.enableUserCalls, "enable 动作必须真实调用 EnableUser（UAC 544 Replace）")
	assert.Equal(t, 0, mock.disableUserCalls, "enable 行绝不能触发 DisableUser")
	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusPushed, row.Status)
	assert.NotNil(t, row.PushedAt)
}

func TestPhase134_Drain_EnableLDAPFailureGoesFailed(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainUser(t, db, "u-1", "alice", "CN=alice,DC=corp,DC=com", "DC=corp,DC=com", nil)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionEnable), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	mock.enableUserErr = errors.New("ldap: enable failed")
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusFailed, row.Status, "LDAP 失败必须转 failed + 退避，不得误报 pushed")
	assert.Contains(t, row.LastError, "enable")
	assert.NotNil(t, row.NextRetryAt)
}

func TestPhase134_Drain_EnableSoftDeletedUserStillDrains(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainUser(t, db, "u-1", "alice", "CN=alice,DC=corp,DC=com", "DC=corp,DC=com", nil)
	// 恢复路径入队时用户仍是软删态（applyRestoreSoftDeleted 会清 deleted_at，但队列行与
	// 业务写同事务；此处模拟"历史软删 + 恢复动作尚在队列"的边界，loadUserUnscoped 必须读到）
	require.NoError(t, db.Exec(`UPDATE sys_user SET deleted_at = ? WHERE id = ?`, drainNow(), "u-1").Error)
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionEnable), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	assert.Equal(t, 1, mock.enableUserCalls, "软删用户的 enable 行仍必须推到 AD（Unscoped 读）")
	assert.Equal(t, models.ADPushStatusPushed, reloadDrainRow(t, db, "u-1").Status)
}

func TestPhase134_Drain_EnableNoAdDnGoesDead(t *testing.T) {
	db := setupDrainTestDB(t)
	const cfgID = "cfg-1"
	seedDrainADConfig(t, db, cfgID, "总部域", "DC=corp,DC=com", true, 0)
	seedDrainUser(t, db, "u-1", "alice", "", "", nil) // ad_dn 空
	seedDrainQueueRow(t, db, "", "u-1", string(models.ADPushActionEnable), string(models.ADPushStatusPending), 0, nil)

	mock := newDrainTestLDAP()
	d := newDrainTestDrainer(t, db, mock, drainTestAccount("a1", cfgID))
	require.NoError(t, d.DrainOnce(context.Background()))

	row := reloadDrainRow(t, db, "u-1")
	assert.Equal(t, models.ADPushStatusDead, row.Status)
	assert.Contains(t, row.LastError, "unrecoverable:no-ad-dn")
	assert.Equal(t, 0, mock.enableUserCalls, "无 ad_dn 不得调用 EnableUser")
}
