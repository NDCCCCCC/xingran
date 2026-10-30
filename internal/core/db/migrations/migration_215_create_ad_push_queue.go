package migrations

import (
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate215CreateADPushQueue 创建 ad_push_queue AD 推送队列表
// (Phase 57 D-05/D-06/D-07: outbox-lite 同事务入队的存储载体)。
//
// 设计要点:
//   - 载荷 = 最小动作行 (entity_type / entity_id / action_type),禁用 JSON 快照
//     (D-05: 快照过期是已判死刑的时间戳仲裁土壤);排水闭包重读 DB 后全量推送。
//   - 合并键走部分唯一索引 (D-06): UNIQUE(entity_type, entity_id, action_type)
//     WHERE status IN ('pending', 'failed')。插入即夺锁,禁用"先查后写"应用层合并;
//     终态行 (pushed/dead) 退出索引、留表可查 (QUEUE-03)。
//   - 退避列一次建齐 (D-07): attempts / next_retry_at / last_error / failed_at,
//     避免 Phase 58 再出迁移。
//   - entity_id 用 varchar(64) 而非 uuid: sys_user.dept_id 是 varchar(64),
//     跨实体类型混用;sqlite 下 uuid declType 亦有 affinity 坑。
//   - 时间列分方言: glebarez/go-sqlite 仅在 declType 为 date|datetime|time|timestamp
//     时扫成 time.Time,写 timestamptz 会扫成 string 并在 Scan 到 *time.Time 时报错。
//   - 全部幂等 (IF NOT EXISTS);database.go 调用点失败仅打日志非阻断 (T-57-04)。
//
// ⚠ 约束红线 (D-08 / W-1): 对应模型 models.ADPushQueue 禁止声明 uniqueIndex tag,
// 否则 GORM AutoMigrate 的 DROP (无 IF EXISTS) 与本迁移建的索引名冲突会导致启动 FATA。
func Migrate215CreateADPushQueue(db *gorm.DB) error {
	log.Println("Running migration 215: create ad_push_queue + active-unique partial index")

	// 1. 建表 (双方言分方言)
	const ddlPG = `CREATE TABLE IF NOT EXISTS ad_push_queue (
		id uuid PRIMARY KEY,
		entity_type varchar(32) NOT NULL,
		entity_id varchar(64) NOT NULL,
		action_type varchar(32) NOT NULL,
		status varchar(20) NOT NULL DEFAULT 'pending',
		attempts integer NOT NULL DEFAULT 0,
		next_retry_at timestamptz,
		last_error text,
		created_at timestamptz,
		updated_at timestamptz,
		pushed_at timestamptz,
		failed_at timestamptz
	)`

	// sqlite 版: 时间列必须写 datetime,否则 time.Time 扫不回 (见上方注释)
	const ddlSQLite = `CREATE TABLE IF NOT EXISTS ad_push_queue (
		id uuid PRIMARY KEY,
		entity_type varchar(32) NOT NULL,
		entity_id varchar(64) NOT NULL,
		action_type varchar(32) NOT NULL,
		status varchar(20) NOT NULL DEFAULT 'pending',
		attempts integer NOT NULL DEFAULT 0,
		next_retry_at datetime,
		last_error text,
		created_at datetime,
		updated_at datetime,
		pushed_at datetime,
		failed_at datetime
	)`

	ddl := ddlSQLite
	if isPostgreSQL(db) {
		ddl = ddlPG
	}
	if err := db.Exec(ddl).Error; err != nil {
		applogger.Errorf("migration 215: create ad_push_queue 失败: %v", err)
		return err
	}

	// 2. 索引 (双方言共用;sqlite 3.35+ / modernc v1.40.1 支持 partial unique index)
	//
	// ⚠ 谓词文本不得改写、不得加表名限定: 取自单一事实源 models.ADPushQueueActivePredicate,
	// 与 Plan 02 的 ON CONFLICT TargetWhere 复用同一常量 (写错即 PG 42P10)。
	// 两侧若要改必须同时改该常量 —— 漂移由 TestPhase57_ActiveRowPredicate_MatchesIndexDDL 守护。
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_ad_push_queue_active
ON ad_push_queue (entity_type, entity_id, action_type)
WHERE ` + models.ADPushQueueActivePredicate).Error; err != nil {
		applogger.Errorf("migration 215: create uq_ad_push_queue_active 失败: %v", err)
		return err
	}
	// 退避拾取索引 (D-07): Phase 58 drainer 按 status + next_retry_at 扫
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_ad_push_queue_status_retry
ON ad_push_queue (status, next_retry_at)`).Error; err != nil {
		applogger.Errorf("migration 215: create idx_ad_push_queue_status_retry 失败: %v", err)
		return err
	}
	// 实体维度查询 (Phase 59 登录守卫按 user_id 查有无 pending/failed 行)
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_ad_push_queue_entity_id
ON ad_push_queue (entity_id)`).Error; err != nil {
		applogger.Errorf("migration 215: create idx_ad_push_queue_entity_id 失败: %v", err)
		return err
	}

	return nil
}
