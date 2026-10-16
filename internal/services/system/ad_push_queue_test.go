package system

// Phase 57 Wave 0：ad_push_queue 迁移与模型行为基线
//
// 覆盖 HYGIENE-06（sqlite 迁移建表路径）与 QUEUE-02（部分唯一索引合并语义）。
// 本文件是 Phase 57 三个 plan 新增测试的唯一落地文件，57-02 / 57-03 在此追加。

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/db/migrations"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
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
func TestPhase57_Migration215_BareInsertWithoutID(t *testing.T) {
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
