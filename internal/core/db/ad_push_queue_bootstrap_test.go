package db

// Phase 57 F-03/F-04: ad_push_queue 在 SKIP_AUTOMIGRATE 旁路（BootstrapMissingTables）
// 与 AutoMigrate 两条启动路径上的建表/索引 fail-fast 行为。
//
// 背景:ad_push_queue 是 AD 推送入队的事务内载体（outbox-lite）。用户编辑/删除在同事务内
// 入队，入队失败按 D-02 第二层回滚业务事务 —— 于是"队列表缺失"不是局部降级，而是
// 用户管理的新增/编辑/删除整体 500。uq_ad_push_queue_active 更是每次 ON CONFLICT 的硬前提，
// 缺失即 PG 42P10。
//
// ⚠ 本文件用 sqlite 测：migration_215 的建表与索引 DDL 双方言共用（仅时间列 declType 不同），
// 部分唯一索引在 sqlite 3.35+ / modernc v1.40.1 同样支持，故断言在两种方言上等价。

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// hasQueueIndex 读取 sqlite_master 中 ad_push_queue 的索引名集合。
func hasQueueIndex(t *testing.T, d *Database, name string) bool {
	t.Helper()
	var n int64
	require.NoError(t, d.DB.Raw(
		`SELECT count(*) FROM sqlite_master WHERE type='index' AND tbl_name='ad_push_queue' AND name = ?`,
		name,
	).Scan(&n).Error)
	return n > 0
}

// TestBootstrapMissingTables_CreatesADPushQueue 锁定 F-03：旁路路径必须补建
// ad_push_queue，且**连同部分唯一索引**（模型刻意不声明 uniqueIndex tag，裸 CreateTable
// 会留下一张没有索引的表 → 运行期 ON CONFLICT 42P10）。
func TestBootstrapMissingTables_CreatesADPushQueue(t *testing.T) {
	d := newBackfillTestDB(t)

	require.NoError(t, d.BootstrapMissingTables())

	require.True(t, d.DB.Migrator().HasTable(&models.ADPushQueue{}),
		"SKIP_AUTOMIGRATE 旁路必须补建 ad_push_queue")
	assert.True(t, hasQueueIndex(t, d, "uq_ad_push_queue_active"),
		"旁路必须同时建出部分唯一索引（ON CONFLICT 的硬前提）")
	assert.True(t, hasQueueIndex(t, d, "idx_ad_push_queue_status_retry"), "退避拾取索引应存在")
	assert.True(t, hasQueueIndex(t, d, "idx_ad_push_queue_entity_id"), "实体查询索引应存在")

	// 幂等：重复执行不得报错（旁路在每次启动都跑）
	require.NoError(t, d.BootstrapMissingTables(), "旁路补建必须幂等可重入")
}

// TestEnsureADPushQueue_SharedByBothStartupPaths 锁定两条启动路径共用同一 DDL 事实源：
// AutoMigrate 路径（PG/sqlite 分支）与旁路路径都调 ensureADPushQueue，索引谓词因此
// 不可能只在其中一条路径上写错。
func TestEnsureADPushQueue_SharedByBothStartupPaths(t *testing.T) {
	d := newBackfillTestDB(t)

	require.NoError(t, d.ensureADPushQueue())
	assert.True(t, d.DB.Migrator().HasTable(&models.ADPushQueue{}))
	assert.True(t, hasQueueIndex(t, d, "uq_ad_push_queue_active"))

	// 谓词与模型的单一事实源一致（DDL 由 migration_215 从该常量拼接）
	assert.Equal(t, "status IN ('pending', 'failed')", models.ADPushQueueActivePredicate)

	// 入队语义在旁路建出的表上必须可用：模型写入路径（BeforeCreate 生成 UUID）
	row := models.ADPushQueue{
		EntityType: models.ADPushEntityUser,
		EntityID:   "f03-entity",
		ActionType: models.ADPushActionUpdateUser,
	}
	require.NoError(t, d.DB.Create(&row).Error, "旁路建出的队列表必须可写")
	assert.Equal(t, models.ADPushStatusPending, row.Status)

	// 同三元组第二条活跃行必须撞部分唯一索引（42P10 的 sqlite 等价面）
	dup := models.ADPushQueue{
		EntityType: models.ADPushEntityUser,
		EntityID:   "f03-entity",
		ActionType: models.ADPushActionUpdateUser,
	}
	assert.Error(t, d.DB.Create(&dup).Error, "uq_ad_push_queue_active 必须生效")

	var sql string
	require.NoError(t, d.DB.Raw(
		`SELECT sql FROM sqlite_master WHERE type='index' AND name='uq_ad_push_queue_active'`,
	).Scan(&sql).Error)
	assert.Contains(t, sql, models.ADPushQueueActivePredicate, "索引 DDL 必须含共享谓词")
}

// TestEnsureADPushQueue_FailsFastOnIndexFailure 锁定 F-04：部分唯一索引创建失败必须
// 阻断启动（返回 error），不得只打日志。
//
// 故障注入：sqlite 里索引名与表名同命名空间 —— 先建一张名为 uq_ad_push_queue_active 的
// 表，CREATE UNIQUE INDEX 就会失败，从而在不依赖 PG 的前提下复现"权限不足/并发 DDL/
// 定义冲突"等真实失败场景的同一失败面。
func TestEnsureADPushQueue_FailsFastOnIndexFailure(t *testing.T) {
	d := newBackfillTestDB(t)
	require.NoError(t, d.DB.Exec(`CREATE TABLE uq_ad_push_queue_active (x INTEGER)`).Error)

	err := d.ensureADPushQueue()
	require.Error(t, err, "uq_ad_push_queue_active 创建失败必须阻断启动，不得只打日志继续")
	assert.Contains(t, err.Error(), "uq_ad_push_queue_active", "错误信息须点名硬前提索引名")

	// 索引确实没建成 → 若继续启动，运行期每次 ON CONFLICT 都会 42P10
	assert.False(t, hasQueueIndex(t, d, "uq_ad_push_queue_active"))

	// 该失败同样不得被 BootstrapMissingTables 吞掉
	assert.Error(t, d.BootstrapMissingTables(), "旁路路径同样必须传播队列表 DDL 失败")
}

// TestAutoMigrate_FailsFastWhenADPushQueueIndexCannotBeCreated 锁定 F-04 在启动路径上的
// 行为：AutoMigrate 里队列表/索引 DDL 失败必须**阻断启动**（返回 error），而不是打一行
// ERROR 日志继续 —— 后者会让运行期第一次用户编辑撞 42P10 并回滚业务事务，而真实原因被埋在
// 几小时前的启动日志里。
//
// 故障注入同 TestEnsureADPushQueue_FailsFastOnIndexFailure：sqlite 索引名与表名共享命名
// 空间，先建一张同名表即可让 CREATE UNIQUE INDEX 失败。
func TestAutoMigrate_FailsFastWhenADPushQueueIndexCannotBeCreated(t *testing.T) {
	d := newBackfillTestDB(t)
	require.NoError(t, d.DB.Exec(`CREATE TABLE uq_ad_push_queue_active (x INTEGER)`).Error)

	err := d.AutoMigrate()
	require.Error(t, err, "uq_ad_push_queue_active 创建失败必须阻断启动（AutoMigrate 返回 error）")
	assert.Contains(t, err.Error(), "ad_push_queue", "错误信息须指向队列表")
}

// TestAutoMigrate_SucceedsAndBuildsQueueIndexes 正常路径：AutoMigrate 结束必须已把队列表
// 与三条索引建齐（旁路与 AutoMigrate 两条路径等价 —— 共享 ensureADPushQueue 的收益）。
func TestAutoMigrate_SucceedsAndBuildsQueueIndexes(t *testing.T) {
	d := newBackfillTestDB(t)

	require.NoError(t, d.AutoMigrate())
	require.True(t, d.DB.Migrator().HasTable(&models.ADPushQueue{}))
	for _, idx := range []string{
		"uq_ad_push_queue_active",
		"idx_ad_push_queue_status_retry",
		"idx_ad_push_queue_entity_id",
	} {
		assert.True(t, hasQueueIndex(t, d, idx), "AutoMigrate 必须建出索引 %s", idx)
	}
}

// compile-time: ensureADPushQueue 必须在 sqlite 上也可调用（旁路既服务 PG 也服务 sqlite 单测）
var _ = func(d *Database, tx *gorm.DB) error { return d.ensureADPushQueue() }
