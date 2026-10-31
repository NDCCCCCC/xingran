package addomain

// DEBT-01（v1.51）：ad_push_queue_service.go 入队路径（Enqueue* 家族）行为测试补齐。
//
// 覆盖目标（断言行为，非凑覆盖率）：
//  1. HasADFieldChanged 四字段 diff 契约（仅 nickname/email/phone/deptId 参与）
//  2. DedupeEntityIDs 去重保序语义
//  3. EnqueueUpdateIfChanged：diff 短路 / D-02 无配置降级 / upsert pending /
//     failed 行合并打回（Phase 58 D-06/D-07 语义）
//  4. EnqueueDisableOnStatusTransition：仅 0→1 入队、启用方向不入队
//  5. EnqueueDisable：F-01 压制（disable 打死同实体活跃 update_user 行，
//     且压制先于 D-02 门控）、去重、批量 upsert 合并
//  6. EnqueueUpdateBatch / EnqueueUpdateManager：去重 + 批量入队 + 冲突合并
//
// 数据库层使用 SQLite 内存库 + AutoMigrate + 与生产 DDL 等价的部分唯一索引
// （uq_ad_push_queue_active），不 mock gorm 层。

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
)

// setupEnqueueTestDB 创建 SQLite 内存库并迁移队列表 + AD 配置表，
// 手工补建与 migration_215 等价的部分唯一索引（ON CONFLICT TargetWhere 依赖它）。
func setupEnqueueTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.ADPushQueue{}, &models.ADConfig{}))
	require.NoError(t, db.Exec(
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_ad_push_queue_active
		 ON ad_push_queue (entity_type, entity_id, action_type)
		 WHERE status IN ('pending','failed')`,
	).Error)
	return db
}

// seedEnabledConfig 写一条启用的 AD 配置（sync_enabled=true AND status=0）。
func seedEnabledConfig(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO sys_ad_config (id, config_name, server_address, server_port, domain_name, base_dn, sync_enabled, status, created_at, updated_at)
		 VALUES ('cfg-1', '测试配置', '127.0.0.1', 389, 'test.com', 'dc=test', 1, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`).Error)
}

func countQueueRows(t *testing.T, db *gorm.DB, action models.ADPushActionType, entityID string) int64 {
	t.Helper()
	var cnt int64
	require.NoError(t, db.Model(&models.ADPushQueue{}).
		Where("entity_type = ? AND entity_id = ? AND action_type = ?",
			models.ADPushEntityUser, entityID, action).Count(&cnt).Error)
	return cnt
}

func TestHasADFieldChanged(t *testing.T) {
	old := &models.User{
		Nickname: strPtr("张三"), Email: strPtr("a@b.c"),
		Phone: strPtr("13800000000"), DeptID: strPtr("d-1"),
	}

	// nil 防御
	if HasADFieldChanged(nil, &requests.UserUpdateRequest{}) {
		t.Fatal("oldUser=nil 应返回 false")
	}
	if HasADFieldChanged(old, nil) {
		t.Fatal("req=nil 应返回 false")
	}
	// 全部字段 nil（无变更语义）→ false
	if HasADFieldChanged(old, &requests.UserUpdateRequest{}) {
		t.Fatal("四个字段全 nil 应视为无变更")
	}
	// 四字段相同（含 TrimSpace 等价）→ false
	same := &requests.UserUpdateRequest{
		Nickname: strPtr("张三"), Email: strPtr(" a@b.c "),
		Phone: strPtr("13800000000"), DeptID: strPtr("d-1"),
	}
	if HasADFieldChanged(old, same) {
		t.Fatal("TrimSpace 后相等应视为无变更")
	}
	// 旧值为 nil 视作空串：旧 nil vs 新空串 → 归一相等 → 无变更
	if HasADFieldChanged(&models.User{}, &requests.UserUpdateRequest{Nickname: strPtr("")}) {
		t.Fatal("旧 nil vs 新空串应视为无变更（都归一为空串）")
	}
	if !HasADFieldChanged(&models.User{}, &requests.UserUpdateRequest{Nickname: strPtr("新")}) {
		t.Fatal("旧 nil vs 新值应视为变更")
	}
	// 逐字段变更 → true
	for name, req := range map[string]*requests.UserUpdateRequest{
		"nickname": {Nickname: strPtr("李四")},
		"email":    {Email: strPtr("x@y.z")},
		"phone":    {Phone: strPtr("13911111111")},
		"deptId":   {DeptID: strPtr("d-2")},
	} {
		if !HasADFieldChanged(old, req) {
			t.Errorf("%s 变更应返回 true", name)
		}
	}
	// 非 diff 字段变更 → false（工号/状态等不参与）
	if HasADFieldChanged(old, &requests.UserUpdateRequest{EmployeeNo: strPtr("E001"), Status: models.UserStatusDisabled}) {
		t.Fatal("非 diff 字段变更不应触发")
	}
}

func TestDedupeEntityIDs(t *testing.T) {
	got := DedupeEntityIDs([]string{"b", "a", "b", "", "a", "c", ""})
	want := []string{"b", "a", "c"}
	if len(got) != len(want) {
		t.Fatalf("去重结果长度不符: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("去重应保首次出现顺序, got %v want %v", got, want)
		}
	}
	if got := DedupeEntityIDs([]string{"", ""}); len(got) != 0 {
		t.Fatalf("全空串应去重为空, got %v", got)
	}
	if got := DedupeEntityIDs(nil); len(got) != 0 {
		t.Fatalf("nil 应返回空, got %v", got)
	}
}

func TestEnqueueUpdateIfChanged(t *testing.T) {
	db := setupEnqueueTestDB(t)
	e := NewADPushEnqueuer()
	old := &models.User{BaseModel: models.BaseModel{ID: "u-1"}, Nickname: strPtr("张三")}

	// 无变更 → 不发任何写
	require.NoError(t, e.EnqueueUpdateIfChanged(db, old, &requests.UserUpdateRequest{}))
	if n := countQueueRows(t, db, models.ADPushActionUpdateUser, "u-1"); n != 0 {
		t.Fatalf("无变更不应入队, got %d 行", n)
	}
	// 有变更但无启用配置 → D-02 降级放行，不入队
	require.NoError(t, e.EnqueueUpdateIfChanged(db, old, &requests.UserUpdateRequest{Nickname: strPtr("李四")}))
	if n := countQueueRows(t, db, models.ADPushActionUpdateUser, "u-1"); n != 0 {
		t.Fatalf("无启用配置不应入队, got %d 行", n)
	}
	// 有变更 + 启用配置 → 一行 pending
	seedEnabledConfig(t, db)
	require.NoError(t, e.EnqueueUpdateIfChanged(db, old, &requests.UserUpdateRequest{Nickname: strPtr("李四")}))
	if n := countQueueRows(t, db, models.ADPushActionUpdateUser, "u-1"); n != 1 {
		t.Fatalf("应入队一行 update_user, got %d 行", n)
	}
	var row models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ? AND action_type = ?", "u-1", models.ADPushActionUpdateUser).First(&row).Error)
	if row.Status != models.ADPushStatusPending {
		t.Fatalf("新入队行应为 pending, got %s", row.Status)
	}
	// 重复入队（活跃行已存在）→ 合并为一行，仍 pending
	require.NoError(t, e.EnqueueUpdateIfChanged(db, old, &requests.UserUpdateRequest{Nickname: strPtr("王五")}))
	if n := countQueueRows(t, db, models.ADPushActionUpdateUser, "u-1"); n != 1 {
		t.Fatalf("同实体活跃行应合并, got %d 行", n)
	}
	// failed 行 → 合并打回 pending 并清空退避状态（D-06/D-07）
	failedAt := time.Now().UTC()
	require.NoError(t, db.Model(&models.ADPushQueue{}).Where("entity_id = ? AND action_type = ?", "u-1", models.ADPushActionUpdateUser).
		Updates(map[string]interface{}{"status": models.ADPushStatusFailed, "attempts": 3, "last_error": "boom", "failed_at": failedAt, "next_retry_at": failedAt}).Error)
	require.NoError(t, e.EnqueueUpdateIfChanged(db, old, &requests.UserUpdateRequest{Nickname: strPtr("赵六")}))
	require.NoError(t, db.Where("entity_id = ? AND action_type = ?", "u-1", models.ADPushActionUpdateUser).First(&row).Error)
	if row.Status != models.ADPushStatusPending || row.Attempts != 0 || row.LastError != "" || row.FailedAt != nil || row.NextRetryAt != nil {
		t.Fatalf("failed 行应被打回 pending 并清空退避状态, got %+v", row)
	}
}

func TestEnqueueDisableOnStatusTransition(t *testing.T) {
	db := setupEnqueueTestDB(t)
	seedEnabledConfig(t, db)
	e := NewADPushEnqueuer()

	// 空 userID / 状态不变 / 启用方向（1→0）/ 新状态非停用 → 全部不入队
	require.NoError(t, e.EnqueueDisableOnStatusTransition(db, models.UserStatusEnabled, models.UserStatusDisabled, ""))
	if n := countQueueRows(t, db, models.ADPushActionDisable, ""); n != 0 {
		t.Fatal("空 userID 不应入队")
	}
	require.NoError(t, e.EnqueueDisableOnStatusTransition(db, models.UserStatusEnabled, models.UserStatusEnabled, "u-1"))
	require.NoError(t, e.EnqueueDisableOnStatusTransition(db, models.UserStatusDisabled, models.UserStatusDisabled, "u-1"))
	require.NoError(t, e.EnqueueDisableOnStatusTransition(db, models.UserStatusDisabled, models.UserStatusEnabled, "u-1"))
	if n := countQueueRows(t, db, models.ADPushActionDisable, "u-1"); n != 0 {
		t.Fatalf("仅 0→1 才入队, got %d 行", n)
	}
	// 0→1 → 一行 pending disable
	require.NoError(t, e.EnqueueDisableOnStatusTransition(db, models.UserStatusEnabled, models.UserStatusDisabled, "u-1"))
	if n := countQueueRows(t, db, models.ADPushActionDisable, "u-1"); n != 1 {
		t.Fatalf("0→1 应入队一行 disable, got %d 行", n)
	}
}

func TestEnqueueDisable_SuppressesActiveUpdateRows(t *testing.T) {
	db := setupEnqueueTestDB(t)
	e := NewADPushEnqueuer()

	// 预置 u-1 的活跃 update_user 行（pending）与 u-2 的 failed update_user 行
	require.NoError(t, db.Create(&models.ADPushQueue{EntityType: models.ADPushEntityUser, EntityID: "u-1", ActionType: models.ADPushActionUpdateUser, Status: models.ADPushStatusPending}).Error)
	require.NoError(t, db.Create(&models.ADPushQueue{EntityType: models.ADPushEntityUser, EntityID: "u-2", ActionType: models.ADPushActionUpdateUser, Status: models.ADPushStatusFailed, Attempts: 2, LastError: "x"}).Error)
	// 预置 u-3 的已终态 update_user 行（pushed，不应被压制改动）
	require.NoError(t, db.Create(&models.ADPushQueue{EntityType: models.ADPushEntityUser, EntityID: "u-3", ActionType: models.ADPushActionUpdateUser, Status: models.ADPushStatusPushed}).Error)

	// 场景 A：无启用配置 → 压制仍发生（先于 D-02 门控），但不入队 disable
	dup := []string{"u-1", "u-1", "u-2", "u-3", ""}
	require.NoError(t, e.EnqueueDisable(db, dup))
	// ⚠ 每次查询用独立变量：复用已填充的 struct 再 First 会把主键残留为查询条件
	var row1, row2, row3 models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ? AND action_type = ?", "u-1", models.ADPushActionUpdateUser).First(&row1).Error)
	if row1.Status != models.ADPushStatusDead {
		t.Fatalf("F-01：pending update_user 应被打死, got %s", row1.Status)
	}
	require.NoError(t, db.Where("entity_id = ? AND action_type = ?", "u-2", models.ADPushActionUpdateUser).First(&row2).Error)
	// 实现契约：压制打 dead 终态 + last_error 标记压制原因 + 清空 next_retry_at/failed_at
	//（attempts 数值保留——行已退出活跃谓词，数值无语义影响）
	if row2.Status != models.ADPushStatusDead || row2.NextRetryAt != nil || row2.FailedAt != nil {
		t.Fatalf("failed update_user 应被打死且清空退避时间, got %+v", row2)
	}
	require.NoError(t, db.Where("entity_id = ? AND action_type = ?", "u-3", models.ADPushActionUpdateUser).First(&row3).Error)
	if row3.Status != models.ADPushStatusPushed {
		t.Fatalf("终态行不应被压制, got %s", row3.Status)
	}
	if n := countQueueRows(t, db, models.ADPushActionDisable, "u-1"); n != 0 {
		t.Fatalf("无启用配置不应入队 disable, got %d 行", n)
	}

	// 场景 B：启用配置 → 去重后入队 disable（u-1、u-2、u-3 各一行）
	seedEnabledConfig(t, db)
	require.NoError(t, e.EnqueueDisable(db, dup))
	for _, id := range []string{"u-1", "u-2", "u-3"} {
		if n := countQueueRows(t, db, models.ADPushActionDisable, id); n != 1 {
			t.Fatalf("去重后 %s 应恰有一行 disable, got %d 行", id, n)
		}
	}
	// 空/全空切片 → 不动库
	require.NoError(t, e.EnqueueDisable(db, nil))
	require.NoError(t, e.EnqueueDisable(db, []string{"", ""}))
}

func TestEnqueueUpdateBatch_And_Manager(t *testing.T) {
	db := setupEnqueueTestDB(t)
	e := NewADPushEnqueuer()

	// 空切片（无配置也直接短路）
	require.NoError(t, e.EnqueueUpdateBatch(db, nil))
	require.NoError(t, e.EnqueueUpdateManager(db, nil))
	// 无配置降级
	require.NoError(t, e.EnqueueUpdateBatch(db, []string{"u-1"}))
	require.NoError(t, e.EnqueueUpdateManager(db, []string{"u-1"}))
	if n := countQueueRows(t, db, models.ADPushActionUpdateUser, "u-1"); n != 0 {
		t.Fatal("无配置不应入队")
	}

	seedEnabledConfig(t, db)
	ids := []string{"u-2", "u-1", "u-2"}
	require.NoError(t, e.EnqueueUpdateBatch(db, ids))
	if n := countQueueRows(t, db, models.ADPushActionUpdateUser, "u-1"); n != 1 {
		t.Fatalf("批量 update 应入队 u-1, got %d 行", n)
	}
	if n := countQueueRows(t, db, models.ADPushActionUpdateUser, "u-2"); n != 1 {
		t.Fatalf("重复 id 应合并为一行, got %d 行", n)
	}
	// 二次批量入队同实体 → 仍一行（冲突合并）
	require.NoError(t, e.EnqueueUpdateBatch(db, []string{"u-1"}))
	if n := countQueueRows(t, db, models.ADPushActionUpdateUser, "u-1"); n != 1 {
		t.Fatalf("二次入队应合并, got %d 行", n)
	}

	// update_manager 独立 action 维度
	require.NoError(t, e.EnqueueUpdateManager(db, ids))
	if n := countQueueRows(t, db, models.ADPushActionUpdateManager, "u-1"); n != 1 {
		t.Fatalf("update_manager 应入队 u-1, got %d 行", n)
	}
	if n := countQueueRows(t, db, models.ADPushActionUpdateManager, "u-2"); n != 1 {
		t.Fatalf("update_manager 重复 id 应合并, got %d 行", n)
	}
}
