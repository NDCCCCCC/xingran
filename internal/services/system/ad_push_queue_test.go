package system

// Phase 57 Wave 0：ad_push_queue 迁移与模型行为基线
//
// 覆盖 HYGIENE-06（sqlite 迁移建表路径）与 QUEUE-02（部分唯一索引合并语义）。
// 本文件是 Phase 57 三个 plan 新增测试的唯一落地文件，57-02 / 57-03 在此追加。
//
// Wave 2（57-02）追加：入队 service 的 diff 边界、前置条件降级、ON CONFLICT 合并语义。

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core/db/migrations"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
)

// setupPhase57QueueDB 创建内存 sqlite 并直调 migration_215 建表。
// 先例：internal/services/system/config_backup_service_93_02_test.go:402。
func setupPhase57QueueDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupUserServiceTestDB(t)
	require.NoError(t, migrations.Migrate215CreateADPushQueue(db))
	return db
}

// queueIndexNames 读取 sqlite_master 中 ad_push_queue 的索引名集合。
func queueIndexNames(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var names []string
	require.NoError(t, db.Raw(
		`SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='ad_push_queue'`,
	).Scan(&names).Error)
	return names
}

// TestPhase57_Migration215_SQLiteDDL 锁定迁移在 sqlite 上的建表 + 索引 + 幂等性。
func TestPhase57_Migration215_SQLiteDDL(t *testing.T) {
	db := setupPhase57QueueDB(t)

	require.True(t, db.Migrator().HasTable("ad_push_queue"), "ad_push_queue 表应存在")

	names := queueIndexNames(t, db)
	assert.Contains(t, names, "uq_ad_push_queue_active", "部分唯一索引应存在")
	assert.Contains(t, names, "idx_ad_push_queue_status_retry", "退避拾取索引应存在")
	assert.Contains(t, names, "idx_ad_push_queue_entity_id", "实体查询索引应存在")

	// 幂等：重复执行不得报错
	require.NoError(t, migrations.Migrate215CreateADPushQueue(db), "迁移必须幂等可重入")
}

// TestPhase57_Migration215_PartialUniqueIndex 锁定 D-06 合并语义。
//
// 裸 SQL 插入（不走 GORM，BeforeCreate 钩子不生效）→ id 必须显式给。
// sqlite 对非 INTEGER 主键列允许 NULL，两行都写 NULL 会让 fixture 静默退化
// （sqlite 对 NULL 主键在非 WITHOUT ROWID 表中不强制唯一），导致唯一约束断言假绿。
func TestPhase57_Migration215_PartialUniqueIndex(t *testing.T) {
	db := setupPhase57QueueDB(t)

	const insertPending = `INSERT INTO ad_push_queue
		(id, entity_type, entity_id, action_type, status, attempts, created_at, updated_at)
		VALUES (?, 'user', ?, 'update_user', 'pending', 0, '2024-01-01 00:00:00', '2024-01-01 00:00:00')`

	entityID := uuid.NewString()
	require.NoError(t, db.Exec(insertPending, uuid.NewString(), entityID).Error)

	// 同三元组第二条 pending 必须命中唯一约束
	err := db.Exec(insertPending, uuid.NewString(), entityID).Error
	require.Error(t, err, "同 (entity_type, entity_id, action_type) 第二条 pending 行必须被唯一约束拦截")
	assert.Contains(t, err.Error(), "UNIQUE constraint failed", "sqlite 应报唯一约束冲突")

	// 第一条转终态 pushed 后退出唯一索引 → 同三元组可再插入
	require.NoError(t, db.Exec(
		`UPDATE ad_push_queue SET status='pushed' WHERE entity_id=? AND status='pending'`, entityID,
	).Error)
	require.NoError(t, db.Exec(insertPending, uuid.NewString(), entityID).Error,
		"终态行应退出部分唯一索引")

	var total int64
	require.NoError(t, db.Table("ad_push_queue").Count(&total).Error)
	assert.Equal(t, int64(2), total, "应保留 1 条 pushed 终态行 + 1 条 pending 行")
}

// TestPhase57_Migration215_BareInsertWithoutID 行为记录（**非契约断言**）。
//
// 记录裸 INSERT 不带 id 时 sqlite 的实际行为：sqlite 对非 INTEGER 主键列允许 NULL，
// 而 PG 的 `id uuid PRIMARY KEY` 强制 NOT NULL（23502）。两侧语义不等价，因此生产
// 写入路径必须走 GORM（BeforeCreate 生成 UUID）。此用例只记录，不断言成功/失败。
//
// ⚠ 已 skip：需真实 PG 才能验证 uuid 列的 NOT NULL 契约（v1.22 跟进）。
// sqlite 对非 INTEGER 主键列容忍 NULL，无法复现 23502，留着跑只会制造"契约已被覆盖"的错觉。
func TestPhase57_Migration215_BareInsertWithoutID(t *testing.T) {
	t.Skip("Phase 58 F-16: 裸 INSERT 无 ID 的 PG 契约断言（uuid 列 NOT NULL → 23502）需真实 PG 才能验证；sqlite 无法复现该约束，PG 契约断言留 v1.22（与 Phase 57 D-04「留档不修」一致）")
	db := setupPhase57QueueDB(t)

	var execErr error
	assert.NotPanics(t, func() {
		execErr = db.Exec(
			`INSERT INTO ad_push_queue (entity_type, entity_id, action_type) VALUES ('user', ?, 'update_user')`,
			uuid.NewString(),
		).Error
	})

	var nullIDs int64
	require.NoError(t, db.Table("ad_push_queue").Where("id IS NULL").Count(&nullIDs).Error)
	t.Logf("裸 INSERT 不带 id: err=%v, id IS NULL 行数=%d (sqlite 容忍 NULL 主键, PG 会 23502)", execErr, nullIDs)
}

// TestPhase57_ADPushQueue_ModelRoundTrip 锁定模型与 DDL 的一致性（HYGIENE-06）。
func TestPhase57_ADPushQueue_ModelRoundTrip(t *testing.T) {
	db := setupPhase57QueueDB(t)

	q := models.ADPushQueue{
		EntityType: models.ADPushEntityUser,
		EntityID:   uuid.NewString(),
		ActionType: models.ADPushActionUpdateUser,
	}
	require.NoError(t, db.Create(&q).Error)

	var got models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ?", q.EntityID).First(&got).Error)

	parsed, err := uuid.Parse(got.ID)
	require.NoError(t, err, "BeforeCreate 应生成合法 UUID")
	assert.NotEmpty(t, parsed.String())
	assert.Equal(t, models.ADPushStatusPending, got.Status, "BeforeCreate 应兜底 status=pending")
	assert.False(t, got.CreatedAt.IsZero(), "sqlite datetime 列应可扫回 time.Time")
	assert.False(t, got.UpdatedAt.IsZero(), "sqlite datetime 列应可扫回 time.Time")

	// 不软删（D-08）：表中不得存在 deleted_at 列
	var softDeleteCols int64
	require.NoError(t, db.Raw(
		`SELECT count(*) AS c FROM pragma_table_info('ad_push_queue') WHERE name='deleted_at'`,
	).Scan(&softDeleteCols).Error)
	assert.Equal(t, int64(0), softDeleteCols, "队列表不软删，不应有 deleted_at 列")
}

// ==================== Wave 2（57-02）：入队 service 行为 ====================

// seedADConfigEnabled 写入一行 sys_ad_config（D-02 前置条件的唯一数据源）。
//
// 表由 setupUserServiceTestDB（HYGIENE-06 共享 helper）保证存在，此处只写数据。
// 固定 id + INSERT OR REPLACE：同一用例内可反复切换 sync_enabled / status
// （否则第二次 INSERT 撞主键）。
func seedADConfigEnabled(t *testing.T, db *gorm.DB, syncEnabled bool, status int) {
	t.Helper()
	require.NoError(t, db.Exec(`
		INSERT OR REPLACE INTO sys_ad_config
			(id, sync_enabled, status, created_at, updated_at, deleted_at)
		VALUES ('phase57-ad-config', ?, ?, '2024-01-01 00:00:00', '2024-01-01 00:00:00', NULL)
	`, syncEnabled, status).Error)
}

// countQueueRows 统计某实体在队列表中的行数。
func countQueueRows(t *testing.T, db *gorm.DB, entityID string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Table("ad_push_queue").Where("entity_id = ?", entityID).Count(&n).Error)
	return n
}

// TestPhase57_HasADFieldChanged_Boundaries 锁定 D-01 四字段 diff 的边界语义。
func TestPhase57_HasADFieldChanged_Boundaries(t *testing.T) {
	old := func() *models.User {
		return &models.User{
			BaseModel: models.BaseModel{ID: "u1"},
			Nickname:  strPtr("x"),
			Email:     strPtr("e@x"),
			Phone:     strPtr("13800000000"),
			DeptID:    strPtr("d1"),
		}
	}

	cases := []struct {
		name string
		old  *models.User
		req  *requests.UserUpdateRequest
		want bool
	}{
		{"nil 视为无变更", old(), &requests.UserUpdateRequest{}, false},
		{"旧值 nil 视空串，与空串 req 相等", &models.User{BaseModel: models.BaseModel{ID: "u2"}}, &requests.UserUpdateRequest{Nickname: strPtr("")}, false},
		{"nickname 变更", old(), &requests.UserUpdateRequest{Nickname: strPtr("y")}, true},
		{"email 变更", old(), &requests.UserUpdateRequest{Email: strPtr("z@x")}, true},
		{"email 未变", old(), &requests.UserUpdateRequest{Email: strPtr("e@x")}, false},
		{"phone 变更", old(), &requests.UserUpdateRequest{Phone: strPtr("13900000000")}, true},
		{"phone 未变", old(), &requests.UserUpdateRequest{Phone: strPtr("13800000000")}, false},
		{"deptId 变更", old(), &requests.UserUpdateRequest{DeptID: strPtr("d2")}, true},
		{"deptId 未变", old(), &requests.UserUpdateRequest{DeptID: strPtr("d1")}, false},
		{
			"纯角色/岗位/备注/状态编辑不入队",
			old(),
			&requests.UserUpdateRequest{
				RoleIds: []string{"r1"},
				PostIds: []string{"p1"},
				Remark:  strPtr("随便改改"),
				Status:  models.UserStatusDisabled,
				Gender:  models.GenderMale,
			},
			false,
		},
		{"TrimSpace 后相等不算变更", old(), &requests.UserUpdateRequest{Nickname: strPtr("  x  ")}, false},
		{"oldUser 为 nil", nil, &requests.UserUpdateRequest{Nickname: strPtr("y")}, false},
		{"req 为 nil", old(), nil, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, addomain.HasADFieldChanged(tc.old, tc.req))
		})
	}
}

// TestPhase57_EnqueueUpdate_EnqueuesOnlyOnRealChange 锁定 D-01 的入队判定。
func TestPhase57_EnqueueUpdate_EnqueuesOnlyOnRealChange(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	userID := seedUserRow(t, db, "phase57-enq-1", 0)
	enq := addomain.NewADPushEnqueuer()
	old := &models.User{BaseModel: models.BaseModel{ID: userID}, Nickname: strPtr("oldnick")}

	// 纯角色编辑 → 不入队
	require.NoError(t, enq.EnqueueUpdateIfChanged(db, old, &requests.UserUpdateRequest{
		RoleIds: []string{"r1"},
		Remark:  strPtr("note"),
	}))
	assert.Equal(t, int64(0), countQueueRows(t, db, userID), "非四字段变更不应产生队列行")

	// nickname 变更 → 入队一行
	require.NoError(t, enq.EnqueueUpdateIfChanged(db, old, &requests.UserUpdateRequest{
		Nickname: strPtr("newnick"),
	}))
	assert.Equal(t, int64(1), countQueueRows(t, db, userID))

	var got models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ?", userID).First(&got).Error)
	assert.Equal(t, models.ADPushEntityUser, got.EntityType)
	assert.Equal(t, models.ADPushActionUpdateUser, got.ActionType)
	assert.Equal(t, models.ADPushStatusPending, got.Status)
}

// TestPhase57_EnqueueUpdate_NoADConfig_SkipsEnqueue 锁定 D-02 第一层降级。
func TestPhase57_EnqueueUpdate_NoADConfig_SkipsEnqueue(t *testing.T) {
	db := setupPhase57QueueDB(t)
	userID := seedUserRow(t, db, "phase57-enq-2", 0)
	enq := addomain.NewADPushEnqueuer()
	old := &models.User{BaseModel: models.BaseModel{ID: userID}, Nickname: strPtr("oldnick")}
	req := &requests.UserUpdateRequest{Nickname: strPtr("newnick")}

	// 路径一：sync_enabled 关闭
	seedADConfigEnabled(t, db, false, 0)
	require.NoError(t, enq.EnqueueUpdateIfChanged(db, old, req), "无启用配置时业务编辑必须放行")
	assert.Equal(t, int64(0), countQueueRows(t, db, userID))

	// 路径二：status 停用
	seedADConfigEnabled(t, db, true, 1)
	require.NoError(t, enq.EnqueueUpdateIfChanged(db, old, req), "配置停用时业务编辑必须放行")
	assert.Equal(t, int64(0), countQueueRows(t, db, userID))
}

// TestPhase57_EnqueueUpdate_MergeOnSecondEdit 锁定 QUEUE-02 合并语义（D-06）。
//
// 这是 T-57-06 的第三重防线：ON CONFLICT 谓词若误放 Where（DO UPDATE 尾随谓词），
// 部分唯一索引不被推断，本用例会在第二次入队时直接报错（sqlite: does not match
// any PRIMARY KEY or UNIQUE constraint）而非合并成一行。
func TestPhase57_EnqueueUpdate_MergeOnSecondEdit(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	userID := seedUserRow(t, db, "phase57-merge", 0)
	enq := addomain.NewADPushEnqueuer()
	old := &models.User{BaseModel: models.BaseModel{ID: userID}, Nickname: strPtr("oldnick")}

	// 连续两次编辑（old 快照不变，模拟连续编辑同一用户）
	require.NoError(t, enq.EnqueueUpdateIfChanged(db, old, &requests.UserUpdateRequest{Nickname: strPtr("nick1")}))
	require.NoError(t, enq.EnqueueUpdateIfChanged(db, old, &requests.UserUpdateRequest{Nickname: strPtr("nick2")}))
	assert.Equal(t, int64(1), countQueueRows(t, db, userID), "同实体连续入队必须合并为一行")

	// 把该行改成退避态（failed），第三次入队应打回 pending 并清空退避字段
	failedAt := time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	require.NoError(t, db.Exec(`
		UPDATE ad_push_queue
		SET status='failed', attempts=3, last_error='boom', failed_at=?, next_retry_at=?
		WHERE entity_id=? AND action_type='update_user'
	`, failedAt, failedAt, userID).Error)

	require.NoError(t, enq.EnqueueUpdateIfChanged(db, old, &requests.UserUpdateRequest{Nickname: strPtr("nick3")}))
	assert.Equal(t, int64(1), countQueueRows(t, db, userID), "failed 行应被打回 pending 而非新增一行")

	var got models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ?", userID).First(&got).Error)
	assert.Equal(t, models.ADPushStatusPending, got.Status)
	assert.Equal(t, 0, got.Attempts)
	assert.Empty(t, got.LastError)
	assert.Nil(t, got.FailedAt)
	assert.Nil(t, got.NextRetryAt)
}

// TestPhase57_EnqueueUpdate_SyncsWithTerminalRow 锁定终态行退出唯一索引后新行独立存在。
func TestPhase57_EnqueueUpdate_SyncsWithTerminalRow(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	userID := seedUserRow(t, db, "phase57-terminal", 0)
	enq := addomain.NewADPushEnqueuer()
	old := &models.User{BaseModel: models.BaseModel{ID: userID}, Nickname: strPtr("oldnick")}

	require.NoError(t, enq.EnqueueUpdateIfChanged(db, old, &requests.UserUpdateRequest{Nickname: strPtr("nick1")}))
	require.NoError(t, db.Exec(
		`UPDATE ad_push_queue SET status='pushed' WHERE entity_id=? AND status='pending'`, userID,
	).Error)

	require.NoError(t, enq.EnqueueUpdateIfChanged(db, old, &requests.UserUpdateRequest{Nickname: strPtr("nick2")}))
	assert.Equal(t, int64(2), countQueueRows(t, db, userID), "终态行不被覆盖，新行独立存在")
}

// countDisableRows 统计 action_type='disable' 的队列行数。
func countDisableRows(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Table("ad_push_queue").Where("action_type = ?", models.ADPushActionDisable).Count(&n).Error)
	return n
}

// TestPhase57_EnqueueDisable_SingleAndBatch 锁定 QUEUE-06 单条与批量 disable 入队。
func TestPhase57_EnqueueDisable_SingleAndBatch(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)
	enq := addomain.NewADPushEnqueuer()

	id1 := seedUserRow(t, db, "phase57-dis-1", 0)
	require.NoError(t, enq.EnqueueDisable(db, []string{id1}))
	assert.Equal(t, int64(1), countQueueRows(t, db, id1))

	var got models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ?", id1).First(&got).Error)
	assert.Equal(t, models.ADPushActionDisable, got.ActionType)
	assert.Equal(t, models.ADPushEntityUser, got.EntityType)
	assert.Equal(t, models.ADPushStatusPending, got.Status)

	ids := []string{
		seedUserRow(t, db, "phase57-dis-2", 0),
		seedUserRow(t, db, "phase57-dis-3", 0),
		seedUserRow(t, db, "phase57-dis-4", 0),
	}
	require.NoError(t, enq.EnqueueDisable(db, ids))
	assert.Equal(t, int64(4), countDisableRows(t, db))

	// 空切片 → no-op
	require.NoError(t, enq.EnqueueDisable(db, nil))
	assert.Equal(t, int64(4), countDisableRows(t, db), "空切片不应新增行")
}

// TestPhase57_DedupeEntityIDs_PureFunction 是 F-05 的**能红**断言：去重逻辑被删除/改坏
// （原样返回入参、保留空串、顺序不稳定）时本用例直接红。
//
// 对照 TestPhase57_EnqueueDisable_DedupesDuplicateIDs —— 后者在 sqlite 上是行为记录，
// 不是契约断言（sqlite 对同一条 INSERT 内重复命中的 UPSERT 行不报错，见该用例注释）。
func TestPhase57_DedupeEntityIDs_PureFunction(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"nil 入参", nil, []string{}},
		{"空切片", []string{}, []string{}},
		{"无重复原样返回", []string{"a", "b"}, []string{"a", "b"}},
		{"重复只留首次出现", []string{"a", "b", "a", "b", "a"}, []string{"a", "b"}},
		{"空串被丢弃", []string{"", "a", "", "b"}, []string{"a", "b"}},
		{"全空串结果为空", []string{"", ""}, []string{}},
		{"顺序按首次出现稳定", []string{"c", "a", "c", "b", "a"}, []string{"c", "a", "b"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := append([]string{}, tc.in...)
			got := addomain.DedupeEntityIDs(tc.in)
			assert.Equal(t, tc.want, got)
			// 纯函数：不得就地改写入参（调用方 BatchDelete 之后还要用 ids 之外的逻辑）
			if tc.in != nil {
				assert.Equal(t, before, tc.in, "去重不得修改入参切片")
			}
		})
	}
}

// TestPhase57_EnqueueDisable_DedupesDuplicateIDs —— **sqlite 行为记录，非契约断言**。
//
// 本用例只证明"当前实现去重后 sqlite 端行数符合预期"，并不能证明去重逻辑存在：
// sqlite 对同一条 INSERT 语句内重复命中的 UPSERT 行不报错（实测逐行处理并合并），
// 所以把 EnqueueDisable 里的去重循环整个删掉，本用例仍然变绿 —— RED 状态不可达。
// 真正的契约断言在 TestPhase57_DedupeEntityIDs_PureFunction（对纯函数断言）。
// 去重要防的 PG 专属故障是 "ON CONFLICT DO UPDATE command cannot affect row a second
// time"，需要 PG 测试环境才能覆盖（Phase 58 验证清单项）。
func TestPhase57_EnqueueDisable_DedupesDuplicateIDs(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)
	enq := addomain.NewADPushEnqueuer()

	id1 := seedUserRow(t, db, "phase57-dup-1", 0)
	id2 := seedUserRow(t, db, "phase57-dup-2", 0)

	require.NoError(t, enq.EnqueueDisable(db, []string{id1, id1, id2}))
	assert.Equal(t, int64(2), countDisableRows(t, db), "重复 id 必须去重")
}

// TestPhase57_EnqueueDisable_MergeOnRepeatCall 锁定重复调用合并（QUEUE-02）。
func TestPhase57_EnqueueDisable_MergeOnRepeatCall(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)
	enq := addomain.NewADPushEnqueuer()

	ids := []string{
		seedUserRow(t, db, "phase57-rep-1", 0),
		seedUserRow(t, db, "phase57-rep-2", 0),
	}
	require.NoError(t, enq.EnqueueDisable(db, ids))
	require.NoError(t, enq.EnqueueDisable(db, ids))

	assert.Equal(t, int64(2), countDisableRows(t, db), "重复调用应合并，行数不变")

	var pending int64
	require.NoError(t, db.Table("ad_push_queue").
		Where("action_type = ? AND status = ?", models.ADPushActionDisable, models.ADPushStatusPending).
		Count(&pending).Error)
	assert.Equal(t, int64(2), pending, "合并后仍为 pending")
}

// countRowsByAction 统计某实体 + 动作的队列行数。
func countRowsByAction(t *testing.T, db *gorm.DB, entityID string, action models.ADPushActionType) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Table("ad_push_queue").
		Where("entity_id = ? AND action_type = ?", entityID, string(action)).Count(&n).Error)
	return n
}

// TestPhase57_ActiveRowPredicate_MatchesIndexDDL 锁定 DDL 与 ON CONFLICT TargetWhere 的
// 谓词文本不会漂移（唯一索引谓词写错 → PG 42P10 / sqlite 冲突目标失配）。
//
// 两条防线：
//  1. models.ADPushQueueActivePredicate 的字面值 —— migration_215 的 DDL 与
//     addomain.adPushOnConflictClause 都从这一个常量取，漂移在编译期即不可能；但常量本身
//     被改写（扩谓词 / 加表名限定 / 改空格）会让两侧同时失配，故仍需锁字面值。
//  2. 实建索引的 DDL（sqlite_master.sql 原文）确实包含该谓词文本。
func TestPhase57_ActiveRowPredicate_MatchesIndexDDL(t *testing.T) {
	db := setupPhase57QueueDB(t)

	assert.Equal(t, "status IN ('pending', 'failed')", models.ADPushQueueActivePredicate,
		"活跃行谓词字面值不得改写/加表名限定（DDL 与 ON CONFLICT TargetWhere 共用）")

	var idxDDL string
	require.NoError(t, db.Raw(
		`SELECT sql FROM sqlite_master WHERE type='index' AND name='uq_ad_push_queue_active'`,
	).Scan(&idxDDL).Error)
	assert.Contains(t, idxDDL, models.ADPushQueueActivePredicate,
		"实建索引 DDL 必须含共享谓词文本")
	assert.NotContains(t, idxDDL, "ad_push_queue.status",
		"谓词不得带表名限定（sqlite partial index 冲突目标为结构化比较，加限定符失配）")
}

// TestPhase57_EnqueueDisable_SupersedesPendingUpdateUser 锁定 F-01：
// 删除必须在队列层压制同实体活跃的 update_user 行，否则软删后排水器重读 sys_user
// 失败 → 把空属性写进 AD 账号（破坏性）或变僵尸行。
func TestPhase57_EnqueueDisable_SupersedesPendingUpdateUser(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)
	enq := addomain.NewADPushEnqueuer()

	userID := seedUserRow(t, db, "phase57-supersede", 0)
	old := &models.User{BaseModel: models.BaseModel{ID: userID}, Nickname: strPtr("oldnick")}

	// 先编辑入队一行 pending update_user
	require.NoError(t, enq.EnqueueUpdateIfChanged(db, old, &requests.UserUpdateRequest{Nickname: strPtr("nick1")}))
	require.Equal(t, int64(1), countRowsByAction(t, db, userID, models.ADPushActionUpdateUser))

	// 再删除 → update_user 行必须被打终态
	require.NoError(t, enq.EnqueueDisable(db, []string{userID}))

	var upd models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ? AND action_type = ?", userID, string(models.ADPushActionUpdateUser)).
		First(&upd).Error)
	assert.Equal(t, models.ADPushStatusDead, upd.Status, "被压制的 update_user 行必须转终态")
	assert.Contains(t, upd.LastError, "superseded by disable", "终态原因必须可排障")
	assert.Nil(t, upd.NextRetryAt, "被压制行不得留在退避重试队列上")

	// disable 行本身照常入队
	require.Equal(t, int64(1), countRowsByAction(t, db, userID, models.ADPushActionDisable))

	// 终态行必须退出部分唯一索引：同实体再入队 update_user 不撞约束（dead 不在活跃谓词内）
	require.NoError(t, enq.EnqueueUpdateIfChanged(db, old, &requests.UserUpdateRequest{Nickname: strPtr("nick2")}),
		"dead 终态行必须退出 uq_ad_push_queue_active，新入队不得撞唯一约束")
	assert.Equal(t, int64(2), countRowsByAction(t, db, userID, models.ADPushActionUpdateUser),
		"终态行留表可查，新行独立存在")
}

// TestPhase57_EnqueueDisable_SupersedesFailedUpdateUser 覆盖 failed（退避中）行同样被压制。
func TestPhase57_EnqueueDisable_SupersedesFailedUpdateUser(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)
	enq := addomain.NewADPushEnqueuer()

	userID := seedUserRow(t, db, "phase57-supersede-failed", 0)
	old := &models.User{BaseModel: models.BaseModel{ID: userID}, Nickname: strPtr("oldnick")}
	require.NoError(t, enq.EnqueueUpdateIfChanged(db, old, &requests.UserUpdateRequest{Nickname: strPtr("nick1")}))

	failedAt := time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	require.NoError(t, db.Exec(`
		UPDATE ad_push_queue SET status='failed', attempts=2, last_error='boom', failed_at=?, next_retry_at=?
		WHERE entity_id=? AND action_type='update_user'
	`, failedAt, failedAt, userID).Error)

	require.NoError(t, enq.EnqueueDisable(db, []string{userID}))

	var upd models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ? AND action_type = ?", userID, string(models.ADPushActionUpdateUser)).
		First(&upd).Error)
	assert.Equal(t, models.ADPushStatusDead, upd.Status, "failed 行同样必须被 disable 压制")
}

// TestPhase57_EnqueueDisable_SupersedeScopedToTargets 锁定压制只作用于本次入队的实体，
// 不得误伤其他用户的活跃行。
func TestPhase57_EnqueueDisable_SupersedeScopedToTargets(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)
	enq := addomain.NewADPushEnqueuer()

	target := seedUserRow(t, db, "phase57-scope-target", 0)
	bystander := seedUserRow(t, db, "phase57-scope-bystander", 0)
	oldOf := func(id string) *models.User {
		return &models.User{BaseModel: models.BaseModel{ID: id}, Nickname: strPtr("oldnick")}
	}
	require.NoError(t, enq.EnqueueUpdateIfChanged(db, oldOf(target), &requests.UserUpdateRequest{Nickname: strPtr("n1")}))
	require.NoError(t, enq.EnqueueUpdateIfChanged(db, oldOf(bystander), &requests.UserUpdateRequest{Nickname: strPtr("n1")}))

	require.NoError(t, enq.EnqueueDisable(db, []string{target}))

	var bystanderRow models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ? AND action_type = ?", bystander, string(models.ADPushActionUpdateUser)).
		First(&bystanderRow).Error)
	assert.Equal(t, models.ADPushStatusPending, bystanderRow.Status, "非目标实体的活跃行不得被压制")
}

// TestPhase57_EnqueueDisable_NoADConfig_StillSupersedes 锁定压制不受 D-02 配置门控影响：
// AD 配置关闭时跳过入队是降级策略，但"用户已软删 + 残留 pending update_user"这一破坏性
// 故障源必须被消除，否则 AD 重新启用后排水器复现。
func TestPhase57_EnqueueDisable_NoADConfig_StillSupersedes(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)
	enq := addomain.NewADPushEnqueuer()

	userID := seedUserRow(t, db, "phase57-supersede-nocfg", 0)
	old := &models.User{BaseModel: models.BaseModel{ID: userID}, Nickname: strPtr("oldnick")}
	require.NoError(t, enq.EnqueueUpdateIfChanged(db, old, &requests.UserUpdateRequest{Nickname: strPtr("nick1")}))
	require.Equal(t, int64(1), countRowsByAction(t, db, userID, models.ADPushActionUpdateUser))
	// 之后再停用 AD 配置：模拟"入队时配置关闭"的降级分支
	require.NoError(t, db.Exec(`UPDATE sys_ad_config SET status=1 WHERE id='phase57-ad-config'`).Error)

	require.NoError(t, enq.EnqueueDisable(db, []string{userID}), "配置停用时业务删除仍放行")
	assert.Equal(t, int64(0), countRowsByAction(t, db, userID, models.ADPushActionDisable), "无启用配置时 disable 不入队")

	var upd models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ? AND action_type = ?", userID, string(models.ADPushActionUpdateUser)).
		First(&upd).Error)
	assert.Equal(t, models.ADPushStatusDead, upd.Status, "压制不因配置门控而跳过")
}

// TestPhase57_EnqueueDisable_NoADConfig_SkipsEnqueue 锁定 D-02 第一层对 disable 同样生效。
func TestPhase57_EnqueueDisable_NoADConfig_SkipsEnqueue(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 1) // status = 停用
	enq := addomain.NewADPushEnqueuer()

	id1 := seedUserRow(t, db, "phase57-dis-nocfg", 0)
	require.NoError(t, enq.EnqueueDisable(db, []string{id1}), "配置停用时业务删除必须放行")
	assert.Equal(t, int64(0), countDisableRows(t, db))
}

// TestPhase57_Enqueue_PropagatesInsertFailure 锁定 D-02 第二层：INSERT 真失败不被吞掉。
//
// 队列表写入失败必须原样返回 error，让调用方（Plan 03 接线的 user_service）能回滚业务事务；
// 若此处返回 nil，"同事务入队不丢"语义被架空。
func TestPhase57_Enqueue_PropagatesInsertFailure(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)
	enq := addomain.NewADPushEnqueuer()

	id1 := seedUserRow(t, db, "phase57-fail", 0)
	require.NoError(t, db.Exec("DROP TABLE ad_push_queue").Error)

	assert.Error(t, enq.EnqueueDisable(db, []string{id1}), "队列表写入失败必须返回 error")

	// 重建（迁移幂等），确保不影响同包其它测试
	require.NoError(t, migrations.Migrate215CreateADPushQueue(db))
}

// ==================== Wave 3（57-03）：UserService 写路径接线 ====================

// TestPhase57_UserService_Update_EnqueuesInSameTransaction 锁定 QUEUE-01 接线点。
//
// 管理员改 nickname → 同一业务事务内必须产生一行 pending update_user 队列行。
func TestPhase57_UserService_Update_EnqueuesInSameTransaction(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	userID := seedUserRow(t, db, "phase57-wiring-upd", 0)
	svc := newTestUserService(t, db)

	err := svc.Update(context.Background(), &requests.UserUpdateRequest{
		ID:       userID,
		Nickname: strPtr("newnick"),
	})
	require.NoError(t, err)

	assert.Equal(t, int64(1), countQueueRows(t, db, userID), "用户编辑必须在同事务产生队列行")

	var got models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ?", userID).First(&got).Error)
	assert.Equal(t, models.ADPushEntityUser, got.EntityType)
	assert.Equal(t, models.ADPushActionUpdateUser, got.ActionType)
	assert.Equal(t, models.ADPushStatusPending, got.Status)

	var nickname *string
	require.NoError(t, db.Raw("SELECT nickname FROM sys_user WHERE id = ?", userID).Scan(&nickname).Error)
	assert.NotNil(t, nickname, "业务写与队列行应同生共死")
	assert.Equal(t, "newnick", *nickname)
}

// TestPhase57_UserService_Update_NoADFieldChange_NoEnqueue 锁定 D-01 diff 在接线后的落地：
// status 不参与四字段 diff → 纯状态编辑**不产生 update_user 行**。
//
// ⚠ F-02 修正说明：本用例原先断言"纯状态编辑产生 0 条队列行"。停用编辑现在会额外入队
// 一行 disable（F-02 安全缺口修复：系统已停用而域账号仍可登录），故断言按动作类型收窄 ——
// update_user 仍必须为 0 行，disable 行为由 TestPhase57_UserService_Update_Disable_EnqueuesDisable 锁定。
func TestPhase57_UserService_Update_NoADFieldChange_NoEnqueue(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	userID := seedUserRow(t, db, "phase57-wiring-nodiff", 0)
	svc := newTestUserService(t, db)

	require.NoError(t, svc.Update(context.Background(), &requests.UserUpdateRequest{
		ID:     userID,
		Status: models.UserStatusDisabled,
	}))
	assert.Equal(t, int64(0), countRowsByAction(t, db, userID, models.ADPushActionUpdateUser),
		"status 不参与四字段 diff，不得产生 update_user 行")
}

// TestPhase57_UserService_Update_Disable_EnqueuesDisable 锁定 F-02：
// 编辑表单保存带 status=1（停用）时必须同事务入队 disable，否则系统停用而域账号仍启用。
func TestPhase57_UserService_Update_Disable_EnqueuesDisable(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	userID := seedUserRow(t, db, "phase57-f02-disable", 0) // status = 0 启用
	svc := newTestUserService(t, db)

	require.NoError(t, svc.Update(context.Background(), &requests.UserUpdateRequest{
		ID: userID, Status: models.UserStatusDisabled,
	}))

	require.Equal(t, int64(1), countRowsByAction(t, db, userID, models.ADPushActionDisable),
		"编辑表单停用必须同事务入队 disable")

	var got models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ? AND action_type = ?", userID, string(models.ADPushActionDisable)).
		First(&got).Error)
	assert.Equal(t, models.ADPushStatusPending, got.Status)
	assert.Equal(t, models.ADPushEntityUser, got.EntityType)

	var status int
	require.NoError(t, db.Raw("SELECT status FROM sys_user WHERE id = ?", userID).Scan(&status).Error)
	assert.Equal(t, int(models.UserStatusDisabled), status, "sys_user 必须已停用（业务写与队列行同生共死）")
}

// TestPhase57_UserService_Update_NoStatusTransition_NoDisable 锁定 F-02 的方向性：
// 已停用用户重复保存（1→1）、启用用户保存（0→0）都不得入队 disable；
// 启用（1→0）也不入队 —— enable 动作在 v1.21 不存在。
func TestPhase57_UserService_Update_NoStatusTransition_NoDisable(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)
	enq := addomain.NewADPushEnqueuer()
	svc := newTestUserService(t, db)

	// 1) 启用用户保存（0→0）
	enabled := seedUserRow(t, db, "phase57-f02-stay-enabled", 0)
	require.NoError(t, svc.Update(context.Background(), &requests.UserUpdateRequest{
		ID: enabled, Status: models.UserStatusEnabled, Remark: strPtr("noop"),
	}))
	assert.Equal(t, int64(0), countRowsByAction(t, db, enabled, models.ADPushActionDisable), "0→0 无跃迁不得入队")

	// 2) 已停用用户重复保存（1→1）
	disabled := seedUserRow(t, db, "phase57-f02-stay-disabled", 1)
	require.NoError(t, svc.Update(context.Background(), &requests.UserUpdateRequest{
		ID: disabled, Status: models.UserStatusDisabled, Remark: strPtr("noop"),
	}))
	assert.Equal(t, int64(0), countRowsByAction(t, db, disabled, models.ADPushActionDisable), "1→1 无跃迁不得入队")

	// 3) 重新启用（1→0）：v1.21 无 enable 动作，不得入队
	require.NoError(t, svc.Update(context.Background(), &requests.UserUpdateRequest{
		ID: disabled, Status: models.UserStatusEnabled,
	}))
	assert.Equal(t, int64(0), countRowsByAction(t, db, disabled, models.ADPushActionDisable), "1→0 启用不在 v1.21 范围")

	// 纯函数级断言：只有 0→1 返回入队
	require.NoError(t, enq.EnqueueDisableOnStatusTransition(db, models.UserStatusEnabled, models.UserStatusDisabled, ""))
	assert.Equal(t, int64(0), countDisableRows(t, db), "空 userID 不得入队")
}

// TestPhase57_EnqueueDisableOnStatusTransition_RollsBackWithUser 锁定 F-02 的事务红线：
// 队列表不可写时，停用编辑必须整体回滚（sys_user.status 不得翻转）。
func TestPhase57_UserService_Update_Disable_RollsBackWhenQueueInsertFails(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	userID := seedUserRow(t, db, "phase57-f02-rollback", 0)
	svc := newTestUserService(t, db)
	require.NoError(t, db.Exec("DROP TABLE ad_push_queue").Error)

	err := svc.Update(context.Background(), &requests.UserUpdateRequest{
		ID: userID, Status: models.UserStatusDisabled,
	})
	require.Error(t, err, "队列表不可写时停用编辑必须整体失败")

	var status int
	require.NoError(t, db.Raw("SELECT status FROM sys_user WHERE id = ?", userID).Scan(&status).Error)
	assert.Equal(t, int(models.UserStatusEnabled), status, "事务必须回滚：用户不得被停用")

	require.NoError(t, migrations.Migrate215CreateADPushQueue(db))
}

// TestPhase57_UserService_Update_Disable_NoADConfig_NoEnqueue 锁定 F-02 在 D-02 第一层
// 下降级：AD 配置停用时停用编辑照常生效，但不入队。
func TestPhase57_UserService_Update_Disable_NoADConfig_NoEnqueue(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 1) // status = 停用

	userID := seedUserRow(t, db, "phase57-f02-nocfg", 0)
	svc := newTestUserService(t, db)

	require.NoError(t, svc.Update(context.Background(), &requests.UserUpdateRequest{
		ID: userID, Status: models.UserStatusDisabled,
	}), "无启用配置时业务停用必须放行")
	assert.Equal(t, int64(0), countRowsByAction(t, db, userID, models.ADPushActionDisable), "无启用配置时不应入队")

	var status int
	require.NoError(t, db.Raw("SELECT status FROM sys_user WHERE id = ?", userID).Scan(&status).Error)
	assert.Equal(t, int(models.UserStatusDisabled), status, "业务停用仍须生效")
}

// TestPhase57_UserService_Update_RollsBackWhenQueueInsertFails 锁定 D-02 第二层在接线后的落地
// （T-57-03 的直接证据）：队列表写失败 → 用户编辑整体回滚，sys_user 无任何变更。
func TestPhase57_UserService_Update_RollsBackWhenQueueInsertFails(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	userID := seedUserRow(t, db, "phase57-wiring-rollback", 0)
	svc := newTestUserService(t, db)
	require.NoError(t, db.Exec("DROP TABLE ad_push_queue").Error)

	err := svc.Update(context.Background(), &requests.UserUpdateRequest{
		ID:       userID,
		Nickname: strPtr("shouldnotpersist"),
	})
	require.Error(t, err, "队列表不可写时用户编辑必须整体失败，不得静默放行")

	var nickname *string
	require.NoError(t, db.Raw("SELECT nickname FROM sys_user WHERE id = ?", userID).Scan(&nickname).Error)
	assert.Nil(t, nickname, "业务事务必须回滚：nickname 不得被写入")

	// 重建（迁移幂等），确保不影响同包其它测试
	require.NoError(t, migrations.Migrate215CreateADPushQueue(db))
}

// TestPhase57_UserService_Update_PartialRequest_PreservesOmittedColumns 锁定 F-06：
// 省略字段 = 未提供 = 不写库。原实现用 tx.Save(&user) 全字段覆盖，任何只带部分字段的请求
// （curl / 第三方脚本 / 前端漏传）都会把 nickname/email/phone/dept_id/dept_name/employee_no/
// remark 静默清空 —— 而入队 diff 把 nil 当"无变更、不入队"，两条路径对 nil 的解释不一致。
//
// 附加价值：本用例同时证明写路径与 diff 路径语义一致（清库 → 无队列行已不再可能）。
func TestPhase57_UserService_Update_PartialRequest_PreservesOmittedColumns(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_dept (id, dept_name, ancestors, status, created_at, updated_at, deleted_at)
		 VALUES ('phase57-dept', '研发部', '', 0, '2024-01-01 00:00:00', '2024-01-01 00:00:00', NULL)`,
	).Error)

	userID := seedUserRow(t, db, "phase57-f06", 0)
	require.NoError(t, db.Exec(`
		UPDATE sys_user SET nickname='keepnick', email='keep@x', phone='13800000000',
			dept_id='phase57-dept', dept_name='研发部', employee_no='EMP001', remark='keepremark'
		WHERE id = ?`, userID).Error)

	svc := newTestUserService(t, db)

	// 只带 nickname 一个字段的请求（无 email/phone/deptId/employeeNo/remark）
	require.NoError(t, svc.Update(context.Background(), &requests.UserUpdateRequest{
		ID: userID, Nickname: strPtr("newnick"),
	}))

	var row struct {
		Nickname   *string
		Email      *string
		Phone      *string
		DeptID     *string
		DeptName   *string
		EmployeeNo *string
		Remark     *string
	}
	require.NoError(t, db.Raw(
		`SELECT nickname, email, phone, dept_id, dept_name, employee_no, remark FROM sys_user WHERE id = ?`,
		userID,
	).Scan(&row).Error)

	require.NotNil(t, row.Nickname)
	assert.Equal(t, "newnick", *row.Nickname, "显式提供的字段必须写入")
	assert.Equal(t, "keep@x", *row.Email, "省略的 email 不得被清空")
	assert.Equal(t, "13800000000", *row.Phone, "省略的 phone 不得被清空")
	assert.Equal(t, "phase57-dept", *row.DeptID, "省略的 dept_id 不得被清空")
	assert.Equal(t, "研发部", *row.DeptName, "省略的 dept_id 时 dept_name 不得被清空")
	assert.Equal(t, "EMP001", *row.EmployeeNo, "省略的 employee_no 不得被清空")
	assert.Equal(t, "keepremark", *row.Remark, "省略的 remark 不得被清空")
}

// TestPhase57_UserService_Update_ExplicitFieldsStillWrite 锁定 F-06 的另一半契约：
// 显式提供的字段照常写入（含显式空串 = 清空），不能因为收窄成 selective Updates 就
// 变成"什么都写不进去"。
func TestPhase57_UserService_Update_ExplicitFieldsStillWrite(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	userID := seedUserRow(t, db, "phase57-f06-explicit", 0)
	require.NoError(t, db.Exec(
		`UPDATE sys_user SET nickname='n', email='e@x', phone='1', remark='r' WHERE id = ?`, userID,
	).Error)

	svc := newTestUserService(t, db)

	require.NoError(t, svc.Update(context.Background(), &requests.UserUpdateRequest{
		ID:       userID,
		Nickname: strPtr(""),
		Email:    strPtr("new@x"),
		Phone:    strPtr("2"),
		Remark:   strPtr(""),
	}))

	var row struct {
		Nickname *string
		Email    *string
		Phone    *string
		Remark   *string
	}
	require.NoError(t, db.Raw(
		`SELECT nickname, email, phone, remark FROM sys_user WHERE id = ?`, userID,
	).Scan(&row).Error)
	require.NotNil(t, row.Nickname)
	assert.Equal(t, "", *row.Nickname, "显式空串应写入（清空 nickname）")
	require.NotNil(t, row.Email)
	assert.Equal(t, "new@x", *row.Email)
	require.NotNil(t, row.Phone)
	assert.Equal(t, "2", *row.Phone)
	require.NotNil(t, row.Remark)
	assert.Equal(t, "", *row.Remark, "显式空串应写入（清空 remark）")
}

// TestPhase57_UserService_Update_OmittedField_NeverEnqueues 锁定写/diff 语义一致：
// 省略字段既不写库也不入队（此前是"清库 + 不入队"）。
func TestPhase57_UserService_Update_OmittedField_NeverEnqueues(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	userID := seedUserRow(t, db, "phase57-f06-nodiff", 0)
	require.NoError(t, db.Exec(`UPDATE sys_user SET nickname='oldnick' WHERE id = ?`, userID).Error)

	svc := newTestUserService(t, db)
	// 只改 remark —— 不在 AD 四字段内，且省略的 nickname 不得被清空/入队
	require.NoError(t, svc.Update(context.Background(), &requests.UserUpdateRequest{
		ID: userID, Remark: strPtr("仅备注"),
	}))

	assert.Equal(t, int64(0), countQueueRows(t, db, userID), "非四字段编辑不得入队")

	var nickname *string
	require.NoError(t, db.Raw("SELECT nickname FROM sys_user WHERE id = ?", userID).Scan(&nickname).Error)
	require.NotNil(t, nickname, "省略的 nickname 不得被清空")
	assert.Equal(t, "oldnick", *nickname)
}

// TestPhase57_UserService_Delete_EnqueuesDisable 锁定 QUEUE-06 接线点（T-57-01）。
func TestPhase57_UserService_Delete_EnqueuesDisable(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	userID := seedUserRow(t, db, "phase57-wiring-del", 0)
	svc := newTestUserService(t, db)

	require.NoError(t, svc.Delete(context.Background(), userID))

	assert.Equal(t, int64(1), countDisableRows(t, db), "软删必须同事务产生 disable 队列行")

	var got models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ? AND action_type = ?", userID, models.ADPushActionDisable).
		First(&got).Error)
	assert.Equal(t, models.ADPushStatusPending, got.Status)

	var deletedAt *string
	require.NoError(t, db.Raw("SELECT deleted_at FROM sys_user WHERE id = ?", userID).Scan(&deletedAt).Error)
	assert.NotNil(t, deletedAt, "用户必须已软删")
}

// TestPhase57_UserService_BatchDelete_EnqueuesDisable 锁定批量软删的 disable 入队。
func TestPhase57_UserService_BatchDelete_EnqueuesDisable(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	id2 := seedUserRow(t, db, "phase57-wiring-bdel-2", 0)
	id3 := seedUserRow(t, db, "phase57-wiring-bdel-3", 0)
	svc := newTestUserService(t, db)

	require.NoError(t, svc.BatchDelete(context.Background(), []string{id2, id3}))

	assert.Equal(t, int64(2), countDisableRows(t, db), "批量软删必须为每个用户产生一行 disable")

	for _, id := range []string{id2, id3} {
		assert.Equal(t, int64(1), countQueueRows(t, db, id), "每个被删用户都应有队列行")
		var deletedAt *string
		require.NoError(t, db.Raw("SELECT deleted_at FROM sys_user WHERE id = ?", id).Scan(&deletedAt).Error)
		assert.NotNil(t, deletedAt, "用户必须已软删")
	}
}

// TestPhase57_UserService_Delete_NoADConfig_NoEnqueue 锁定 D-02 第一层在接线后的落地：
// AD 配置停用时业务删除照常（软删生效），但不入队。
func TestPhase57_UserService_Delete_NoADConfig_NoEnqueue(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 1) // status = 停用

	userID := seedUserRow(t, db, "phase57-wiring-nocfg", 0)
	svc := newTestUserService(t, db)

	require.NoError(t, svc.Delete(context.Background(), userID), "配置停用时业务删除必须放行")

	assert.Equal(t, int64(0), countDisableRows(t, db), "无启用配置时不应入队")

	var deletedAt *string
	require.NoError(t, db.Raw("SELECT deleted_at FROM sys_user WHERE id = ?", userID).Scan(&deletedAt).Error)
	assert.NotNil(t, deletedAt, "业务软删仍须生效")
}

// ==================== GAP-01：UpdateStatus 入口接线 ====================

// TestPhase57_UserService_UpdateStatus_Disable_EnqueuesDisable 锁定 GAP-01：
// 列表页 Switch 走的 POST /system/users/:id/status → UserService.UpdateStatus
// 必须与 Update 入口一样，在停用（0→1）时同事务入队 disable。
func TestPhase57_UserService_UpdateStatus_Disable_EnqueuesDisable(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	userID := seedUserRow(t, db, "phase57-gap01-status", 0) // status = 0 启用
	svc := newTestUserService(t, db)

	require.NoError(t, svc.UpdateStatus(context.Background(), userID, int(models.UserStatusDisabled)))

	require.Equal(t, int64(1), countRowsByAction(t, db, userID, models.ADPushActionDisable),
		"UpdateStatus 停用必须同事务入队 disable（列表页 Switch 是最高频停用入口）")

	var got models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ?", userID).First(&got).Error)
	assert.Equal(t, models.ADPushStatusPending, got.Status)
	assert.Equal(t, models.ADPushEntityUser, got.EntityType)

	var status int
	require.NoError(t, db.Raw("SELECT status FROM sys_user WHERE id = ?", userID).Scan(&status).Error)
	assert.Equal(t, int(models.UserStatusDisabled), status, "业务写与队列行应同生共死")
}

// TestPhase57_UserService_UpdateStatus_Disable_RollsBackWhenQueueInsertFails 锁定 GAP-01
// 的事务红线：队列表不可写时状态变更必须整体回滚（不得出现"停用了但没记录"）。
func TestPhase57_UserService_UpdateStatus_Disable_RollsBackWhenQueueInsertFails(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	userID := seedUserRow(t, db, "phase57-gap01-status-rollback", 0)
	svc := newTestUserService(t, db)
	require.NoError(t, db.Exec("DROP TABLE ad_push_queue").Error)

	err := svc.UpdateStatus(context.Background(), userID, int(models.UserStatusDisabled))
	require.Error(t, err, "队列表不可写时停用必须整体失败")

	var status int
	require.NoError(t, db.Raw("SELECT status FROM sys_user WHERE id = ?", userID).Scan(&status).Error)
	assert.Equal(t, int(models.UserStatusEnabled), status, "事务必须回滚：用户不得被停用")

	require.NoError(t, migrations.Migrate215CreateADPushQueue(db))
}

// TestPhase57_UserService_UpdateStatus_NoTransition_NoDisable 锁定方向性与 no-op：
// 1→0（重新启用）、0→0、1→1 三种情况都不得入队 disable —— v1.21 没有 enable 动作。
func TestPhase57_UserService_UpdateStatus_NoTransition_NoDisable(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)
	svc := newTestUserService(t, db)

	// 1) 启用用户重复启用（0→0）
	enabled := seedUserRow(t, db, "phase57-gap01-stay-enabled", 0)
	require.NoError(t, svc.UpdateStatus(context.Background(), enabled, int(models.UserStatusEnabled)))
	assert.Equal(t, int64(0), countRowsByAction(t, db, enabled, models.ADPushActionDisable), "0→0 无跃迁不得入队")

	// 2) 已停用用户重复停用（1→1）
	disabled := seedUserRow(t, db, "phase57-gap01-stay-disabled", 1)
	require.NoError(t, svc.UpdateStatus(context.Background(), disabled, int(models.UserStatusDisabled)))
	assert.Equal(t, int64(0), countRowsByAction(t, db, disabled, models.ADPushActionDisable), "1→1 无跃迁不得入队")

	// 3) 重新启用（1→0）：v1.21 无 enable 动作，不得入队
	reenable := seedUserRow(t, db, "phase57-gap01-reenable", 1)
	require.NoError(t, svc.UpdateStatus(context.Background(), reenable, int(models.UserStatusEnabled)))
	assert.Equal(t, int64(0), countRowsByAction(t, db, reenable, models.ADPushActionDisable), "1→0 启用不在 v1.21 范围")
	assert.Equal(t, int64(0), countQueueRows(t, db, reenable), "重新启用不得产生任何队列行")
}

// TestPhase57_UserService_UpdateStatus_Disable_NoADConfig_NoEnqueue 锁定 D-02 第一层
// 在 UpdateStatus 入口的落地：AD 配置停用时状态变更照常生效，但不入队。
func TestPhase57_UserService_UpdateStatus_Disable_NoADConfig_NoEnqueue(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 1) // status = 停用

	userID := seedUserRow(t, db, "phase57-gap01-status-nocfg", 0)
	svc := newTestUserService(t, db)

	require.NoError(t, svc.UpdateStatus(context.Background(), userID, int(models.UserStatusDisabled)),
		"无启用配置时状态变更必须放行")
	assert.Equal(t, int64(0), countQueueRows(t, db, userID), "无启用配置时不应入队")

	var status int
	require.NoError(t, db.Raw("SELECT status FROM sys_user WHERE id = ?", userID).Scan(&status).Error)
	assert.Equal(t, int(models.UserStatusDisabled), status, "业务停用仍须生效")
}

// ==================== GAP-02：profileService.UpdateUserInfo 自助路径接线 ====================

// seedUserProfileRow 在 sys_user 插入带 AD 同步字段初值的一行，返回 id。
func seedUserProfileRow(t *testing.T, db *gorm.DB, username, nickname, email, phone string) string {
	t.Helper()
	id := seedUserRow(t, db, username, 0)
	require.NoError(t, db.Exec(
		`UPDATE sys_user SET nickname = ?, email = ?, phone = ? WHERE id = ?`,
		nickname, email, phone, id,
	).Error)
	return id
}

// TestPhase57_ProfileService_UpdateUserInfo_Change_Enqueues 锁定 GAP-02：
// 自助资料修改（个人中心）改到 AD 同步字段（nickname/email/phone）时，
// 必须与管理员 Update 路径一样同事务入队 update_user —— 否则 AD 同步用户的
// email/phone 会被下次 AD 同步静默回滚，且 Phase 59 守卫把分叉判成"一致"。
func TestPhase57_ProfileService_UpdateUserInfo_Change_Enqueues(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	userID := seedUserProfileRow(t, db, "phase57-gap02-profile", "oldnick", "old@x", "13800000000")
	svc := NewProfileService(db)

	newNick := "newnick"
	require.NoError(t, svc.UpdateUserInfo(context.Background(), userID, &UpdateUserInfoRequest{
		Nickname: &newNick,
		Gender:   1,
	}))

	require.Equal(t, int64(1), countRowsByAction(t, db, userID, models.ADPushActionUpdateUser),
		"自助改 AD 同步字段必须同事务入队 update_user")

	var got models.ADPushQueue
	require.NoError(t, db.Where("entity_id = ?", userID).First(&got).Error)
	assert.Equal(t, models.ADPushEntityUser, got.EntityType)
	assert.Equal(t, models.ADPushActionUpdateUser, got.ActionType)
	assert.Equal(t, models.ADPushStatusPending, got.Status)

	var nickname *string
	require.NoError(t, db.Raw("SELECT nickname FROM sys_user WHERE id = ?", userID).Scan(&nickname).Error)
	assert.NotNil(t, nickname)
	assert.Equal(t, "newnick", *nickname, "业务写与队列行应同生共死")
}

// TestPhase57_ProfileService_UpdateUserInfo_NoADFieldChange_NoEnqueue 锁定 D-01 在
// 自助路径的落地：只改 gender / remark（不在 AD 四字段契约内）不得产生队列行。
func TestPhase57_ProfileService_UpdateUserInfo_NoADFieldChange_NoEnqueue(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	userID := seedUserProfileRow(t, db, "phase57-gap02-noad", "oldnick", "old@x", "13800000000")
	svc := NewProfileService(db)

	remark := "just a note"
	require.NoError(t, svc.UpdateUserInfo(context.Background(), userID, &UpdateUserInfoRequest{
		Remark: &remark,
		Gender: 1,
	}))
	assert.Equal(t, int64(0), countQueueRows(t, db, userID),
		"gender/remark 不在 AD 四字段契约内，不得入队")
}

// TestPhase57_ProfileService_UpdateUserInfo_SameValue_NoEnqueue 锁定"实际变更"语义：
// 提供了字段但值（TrimSpace 后）与旧值相同 → 不入队。
func TestPhase57_ProfileService_UpdateUserInfo_SameValue_NoEnqueue(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	userID := seedUserProfileRow(t, db, "phase57-gap02-same", "oldnick", "old@x", "13800000000")
	svc := NewProfileService(db)

	same := "  oldnick  " // TrimSpace 后与旧值相同
	require.NoError(t, svc.UpdateUserInfo(context.Background(), userID, &UpdateUserInfoRequest{
		Nickname: &same,
		Gender:   1,
	}))
	assert.Equal(t, int64(0), countQueueRows(t, db, userID),
		"值未实际变更（TrimSpace 后相同）不得入队")
}

// TestPhase57_ProfileService_UpdateUserInfo_RollsBackWhenQueueInsertFails 锁定 GAP-02
// 的事务红线：队列表不可写时自助编辑必须整体回滚（QUEUE-01，与管理员路径同款）。
func TestPhase57_ProfileService_UpdateUserInfo_RollsBackWhenQueueInsertFails(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	userID := seedUserProfileRow(t, db, "phase57-gap02-rollback", "oldnick", "old@x", "13800000000")
	svc := NewProfileService(db)
	require.NoError(t, db.Exec("DROP TABLE ad_push_queue").Error)

	newNick := "shouldnotpersist"
	err := svc.UpdateUserInfo(context.Background(), userID, &UpdateUserInfoRequest{
		Nickname: &newNick,
		Gender:   1,
	})
	require.Error(t, err, "队列表不可写时自助编辑必须整体失败")

	var nickname *string
	require.NoError(t, db.Raw("SELECT nickname FROM sys_user WHERE id = ?", userID).Scan(&nickname).Error)
	require.NotNil(t, nickname)
	assert.Equal(t, "oldnick", *nickname, "事务必须回滚：nickname 必须保持旧值")

	require.NoError(t, migrations.Migrate215CreateADPushQueue(db))
}

// TestPhase57_ProfileService_UpdateUserInfo_NoADConfig_NoEnqueue 锁定 D-02 第一层在
// 自助路径的落地：AD 配置停用时自助编辑照常生效，但不入队。
func TestPhase57_ProfileService_UpdateUserInfo_NoADConfig_NoEnqueue(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 1) // status = 停用

	userID := seedUserProfileRow(t, db, "phase57-gap02-nocfg", "oldnick", "old@x", "13800000000")
	svc := NewProfileService(db)

	newNick := "newnick"
	require.NoError(t, svc.UpdateUserInfo(context.Background(), userID, &UpdateUserInfoRequest{
		Nickname: &newNick,
		Gender:   1,
	}), "无启用配置时自助编辑必须放行")
	assert.Equal(t, int64(0), countQueueRows(t, db, userID), "无启用配置时不应入队")

	var nickname *string
	require.NoError(t, db.Raw("SELECT nickname FROM sys_user WHERE id = ?", userID).Scan(&nickname).Error)
	assert.NotNil(t, nickname)
	assert.Equal(t, "newnick", *nickname, "业务写仍须生效")
}

// ==================== Phase 58 Plan 06（T3）：F-07 入队侧 Pluck + F-10 回滚用例 ====================

// seedUserRoleRows 向 sys_user_role 插 n 行关联（role_id 任意非空）。
// F-10 断言"关联删除与软删同事务回滚"需要真实存在的关联行 —— 0 行时
// "行数不变" 恒真，用例会变成假绿。
func seedUserRoleRows(t *testing.T, db *gorm.DB, userID string, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		require.NoError(t, db.Exec(
			"INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)",
			userID, uuid.NewString()).Error)
	}
}

func countUserRoleRows(t *testing.T, db *gorm.DB, userID string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw("SELECT count(*) FROM sys_user_role WHERE user_id = ?", userID).Scan(&n).Error)
	return n
}

func countSoftDeletedUsers(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw("SELECT count(*) FROM sys_user WHERE deleted_at IS NOT NULL").Scan(&n).Error)
	return n
}

func userDeletedAt(t *testing.T, db *gorm.DB, userID string) *string {
	t.Helper()
	var deletedAt *string
	require.NoError(t, db.Raw("SELECT deleted_at FROM sys_user WHERE id = ?", userID).Scan(&deletedAt).Error)
	return deletedAt
}

// TestPhase58_UserService_BatchDelete_EnqueuesOnlyExisting 锁定 F-07 入队侧掐根：
// 传入含不存在 id 的列表时，只对真实存在的行入队 disable。
//
// ⚠ 这是"Pluck 必须排在软删之前"的**行为判别式**：若 Pluck 被移到软删之后，
// GORM 默认软删作用域会把本轮刚软删的行过滤掉 → existing 恒空 → 入队 0 行 → 本用例红。
func TestPhase58_UserService_BatchDelete_EnqueuesOnlyExisting(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	u1 := seedUserRow(t, db, "phase58-f07-1", 0)
	u2 := seedUserRow(t, db, "phase58-f07-2", 0)
	missingID := uuid.NewString() // 故意不 seed —— 不存在的 id
	svc := newTestUserService(t, db)

	require.NoError(t, svc.BatchDelete(context.Background(), []string{u1, u2, missingID}))

	assert.Equal(t, int64(2), countDisableRows(t, db),
		"不存在的 id 不得入队（只有真实存在的行才入队）")

	// 入队的两行必须是 u1 / u2，不含 missingID
	var entityIDs []string
	require.NoError(t, db.Model(&models.ADPushQueue{}).
		Where("action_type = ?", string(models.ADPushActionDisable)).
		Pluck("entity_id", &entityIDs).Error)
	assert.ElementsMatch(t, []string{u1, u2}, entityIDs, "入队行必须恰好对应两个存在的用户")

	// 口径断言：入队 disable 行数 == 实际被软删的用户数 == len(existing)
	assert.Equal(t, int64(2), countSoftDeletedUsers(t, db), "实际软删的必须是两个用户")

	// 重复 id 也只入队一行（去重 + 存在性过滤叠加）
	db2 := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db2, true, 0)
	v1 := seedUserRow(t, db2, "phase58-f07-dup", 0)
	require.NoError(t, newTestUserService(t, db2).BatchDelete(context.Background(), []string{v1, v1, v1}))
	assert.Equal(t, int64(1), countDisableRows(t, db2), "重复 id 必须去重后只入队一行")
	assert.Equal(t, int64(1), countSoftDeletedUsers(t, db2), "重复 id 不得重复软删计数")
}

// TestPhase58_UserService_Delete_RollsBackWhenQueueInsertFails 锁定 F-10 单条删除：
// 队列表不可写时整个删除事务回滚 —— 软删不得落地、角色关联不得消失。
//
// 复制源：TestPhase57_UserService_Update_Disable_RollsBackWhenQueueInsertFails（Phase 57 已覆盖 Update 路径，
// 删除路径此前无回滚证据）。
func TestPhase58_UserService_Delete_RollsBackWhenQueueInsertFails(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	userID := seedUserRow(t, db, "phase58-f10-delete", 0)
	seedUserRoleRows(t, db, userID, 2)
	rolesBefore := countUserRoleRows(t, db, userID)
	require.Equal(t, int64(2), rolesBefore, "前置条件：必须真的有角色关联行，否则回滚断言恒真")

	svc := newTestUserService(t, db)
	require.NoError(t, db.Exec("DROP TABLE ad_push_queue").Error)

	err := svc.Delete(context.Background(), userID)
	require.Error(t, err, "队列表不可写时删除用户必须整体失败")

	assert.Nil(t, userDeletedAt(t, db, userID), "事务必须回滚：用户不得被软删")
	assert.Equal(t, rolesBefore, countUserRoleRows(t, db, userID), "角色关联的删除也必须回滚")

	require.NoError(t, migrations.Migrate215CreateADPushQueue(db))
}

// TestPhase58_UserService_BatchDelete_RollsBackWhenQueueInsertFails 锁定 F-10 批量删除：
// 与单条同款语义 —— 两个用户的软删与角色关联删除全部回滚。
func TestPhase58_UserService_BatchDelete_RollsBackWhenQueueInsertFails(t *testing.T) {
	db := setupPhase57QueueDB(t)
	seedADConfigEnabled(t, db, true, 0)

	u1 := seedUserRow(t, db, "phase58-f10-bdel-1", 0)
	u2 := seedUserRow(t, db, "phase58-f10-bdel-2", 0)
	seedUserRoleRows(t, db, u1, 2)
	seedUserRoleRows(t, db, u2, 1)
	rolesBeforeU1 := countUserRoleRows(t, db, u1)
	rolesBeforeU2 := countUserRoleRows(t, db, u2)
	require.Equal(t, int64(2), rolesBeforeU1)
	require.Equal(t, int64(1), rolesBeforeU2)

	svc := newTestUserService(t, db)
	require.NoError(t, db.Exec("DROP TABLE ad_push_queue").Error)

	err := svc.BatchDelete(context.Background(), []string{u1, u2})
	require.Error(t, err, "队列表不可写时批量删除必须整体失败")

	assert.Nil(t, userDeletedAt(t, db, u1), "事务必须回滚：u1 不得被软删")
	assert.Nil(t, userDeletedAt(t, db, u2), "事务必须回滚：u2 不得被软删")
	assert.Equal(t, rolesBeforeU1, countUserRoleRows(t, db, u1), "u1 的角色关联删除必须回滚")
	assert.Equal(t, rolesBeforeU2, countUserRoleRows(t, db, u2), "u2 的角色关联删除必须回滚")
	assert.Equal(t, int64(0), countSoftDeletedUsers(t, db), "整体不得有任何软删落地")

	require.NoError(t, migrations.Migrate215CreateADPushQueue(db))
}
