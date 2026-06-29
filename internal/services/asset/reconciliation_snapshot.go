package asset

// Package asset - Phase 42 R1 (Asset Reconciliation Reconcile Engine)
//
// reconciliation_snapshot.go 提供物化视图 reconciliation_normalized 的刷新与状态查询。
//
// 设计要点(R1 stub 阶段):
//   - D-01: REFRESH MATERIALIZED VIEW CONCURRENTLY 5min 定时执行
//   - D-02: 失败仅 log 警告并返回 error,具体失败处理由调用方决定
//     (cron 写 job_log,handler 透传 500)
//   - D-10: cron 注册走 sys_job 表(InvokeTarget 字符串),无新 internal/scheduler/ 文件
//
// 4 个全局函数(ExecuteRefreshViewTask / ExecuteDetectLayer3Task /
// ExecuteDetectExpiredSilenceTask / ExecuteCleanupExpiredExceptionsTask)为
// sys_job.InvokeTarget 的字符串调度入口。R1 阶段是 stub,真实 service 绑定
// 在 42-04 / 42-06 plan 注入(避免 asset 包直接 import core 造成循环依赖)。

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ReconciliationSnapshot 物化视图快照服务
type ReconciliationSnapshot interface {
	// RefreshView REFRESH MATERIALIZED VIEW CONCURRENTLY reconciliation_normalized
	// D-01: 5min 定时刷新,失败仅 log 警告(D-02)
	RefreshView(ctx context.Context) error
	// LastRefreshAt 从 sys_config.config_key='asset.reconciliation.view.last_refresh_at' 读 ISO8601
	// 无配置则返回 nil
	LastRefreshAt(ctx context.Context) (*time.Time, error)
}

type reconciliationSnapshotServiceImpl struct {
	db *gorm.DB
}

func NewReconciliationSnapshotService(db *gorm.DB) ReconciliationSnapshot {
	return &reconciliationSnapshotServiceImpl{db: db}
}

// reconciliationRefreshTimeout MV 刷新强制上限。
//
// 2026-06-30 生产事故:cron 路径(@every 5m)传入无超时 ctx,REFRESH 在 O(N²) 视图上
// 烧 CPU 27min+,多个 cron 节拍堆叠僵尸后端,级联锁死后续 app 启动(卡在
// cleanupOldConstraints 的 ALTER ops_asset)。RefreshView 不能假设调用方一定传 timeout
// ctx —— 故在此强制 90s 上限,与调用方无关(migration_180 已让视图 O(N+M) 亚秒级,
// 90s 纯属防御网;视图若退化到超 90s 说明有回归,应被取消并告警)。
const reconciliationRefreshTimeout = 90 * time.Second

// RefreshView REFRESH MATERIALIZED VIEW CONCURRENTLY reconciliation_normalized.
//
// D-02: 失败仅 logrus 日志,返回 error 让调用方决定(cron 写 job_log,handler 透传 500)。
//
// 超时双保险(防御纵深):
//  1. Go context WithTimeout(90s) —— pgx/v5 在 ctx 取消时向 PG 发 cancel
//  2. 服务端 SET statement_timeout=90s —— PG 自杀,不依赖驱动 ctx 链路
//
// 用专用 *sql.Conn 保证 SET/REFRESH/RESET 落同一连接(REFRESH ... CONCURRENTLY 不能
// 在事务块里,故不能用 SET LOCAL;专用连接避免 statement_timeout 泄漏到连接池其他查询)。
func (s *reconciliationSnapshotServiceImpl) RefreshView(ctx context.Context) error {
	// 1. Go 侧 timeout(与调用方 ctx 取交集:startup 的 30s 仍生效,cron 的无超时 ctx 被 90s 兜住)
	refreshCtx, cancel := context.WithTimeout(ctx, reconciliationRefreshTimeout)
	defer cancel()

	// 2. 专用连接,statement_timeout 会话级隔离
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("RefreshView: 获取 *sql.DB 失败: %w", err)
	}
	conn, err := sqlDB.Conn(refreshCtx)
	if err != nil {
		return fmt.Errorf("RefreshView: 获取专用连接失败: %w", err)
	}
	defer conn.Close()

	// 3. 服务端 statement_timeout(毫秒)。defer 先于 conn.Close() 执行(LIFO),
	//    用独立 ctx 防止 refreshCtx 已取消时 RESET 也被拒。
	timeoutMs := int64(reconciliationRefreshTimeout / time.Millisecond)
	if _, err := conn.ExecContext(refreshCtx, fmt.Sprintf("SET statement_timeout = %d", timeoutMs)); err != nil {
		return fmt.Errorf("RefreshView: 设置 statement_timeout 失败: %w", err)
	}
	defer func() {
		if _, err := conn.ExecContext(context.Background(), "RESET statement_timeout"); err != nil {
			log.WithError(err).Warn("RefreshView: RESET statement_timeout 失败(连接归还池前)")
		}
	}()

	// 4. REFRESH CONCURRENTLY(不能在事务块里,用 autocommit Exec)
	if _, err := conn.ExecContext(refreshCtx, "REFRESH MATERIALIZED VIEW CONCURRENTLY reconciliation_normalized"); err != nil {
		log.WithError(err).Warn("snapshot refresh failed, retry next cycle")
		return err
	}
	return nil
}

// LastRefreshAt 从 sys_config 读最近一次刷新时间戳。
//
// config_key: 'asset.reconciliation.view.last_refresh_at'
// 值为 ISO8601 字符串(本函数不写,仅读;R1 占位,后续 R2/F1 接入后由 RefreshView 成功路径写入)。
// 无该 config key 或值非合法时间格式则返回 nil, nil。
func (s *reconciliationSnapshotServiceImpl) LastRefreshAt(ctx context.Context) (*time.Time, error) {
	var value string
	if err := s.db.WithContext(ctx).
		Table("sys_config").
		Select("config_value").
		Where("config_key = ?", "asset.reconciliation.view.last_refresh_at").
		Scan(&value).Error; err != nil {
		return nil, err
	}
	if value == "" {
		return nil, nil
	}
	// 尝试 RFC3339 解析
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return &t, nil
	}
	// 尝试 common layout
	if t, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
		return &t, nil
	}
	return nil, nil
}

// ExecuteRefreshViewTask sys_job InvokeTarget='reconciliation:refreshView' 全局函数。
//
// 用于 sys_job 表 cron 调度(@every 5m)。
// 实际语义与 RefreshView 一致,只是作为顶层函数供 InvokeTarget 字符串调用。
func ExecuteRefreshViewTask(ctx context.Context) error {
	// R1 stub: 取不到 db 实例时返回 nil。
	// 实际生产路径: 通过 core.Core.DB().GetDB() 拿到 *gorm.DB,构造 service 调用。
	// 当前 plan 阶段 db 实例由启动器注入,本函数签名仅做兼容。
	// 见 internal/scheduler/ad_sync_tasks.go 实际注入模式。
	//
	// 真正的 service 调用样例(供参考,R1 阶段不直接 import core 避免循环依赖):
	//   svc := NewReconciliationSnapshotService(db)
	//   return svc.RefreshView(ctx)
	log.Info("reconciliation:refreshView invoked (R1 stub, service binding TBD)")
	return nil
}

// ExecuteDetectLayer3Task sys_job InvokeTarget='reconciliation:detectLayer3' 全局函数。
//
// R1 真实执行:RefreshView(确保 MV 最新) + DetectLayer3(写 sys_data_reconciliation)。
// D-07: R1 同步做完整 Layer 3 引擎。
func ExecuteDetectLayer3Task(ctx context.Context) error {
	log.Info("reconciliation:detectLayer3 invoked (R1 stub, service binding TBD)")
	return nil
}

// ExecuteDetectExpiredSilenceTask sys_job InvokeTarget='reconciliation:detectExpiredSilence' 全局函数。
//
// R2 真实实现:R2 接入告警时上线。R1 仅 placeholder。
func ExecuteDetectExpiredSilenceTask(ctx context.Context) error {
	log.Info("reconciliation:detectExpiredSilence invoked (R1 placeholder, R2 implements)")
	return nil
}

// ExecuteCleanupExpiredExceptionsTask sys_job InvokeTarget='reconciliation:cleanupExpiredExceptions' 全局函数。
//
// R3 真实实现:R3 例外规则 CRUD 上线时同时启用临时例外到期清理。
// R1 仅 placeholder。
func ExecuteCleanupExpiredExceptionsTask(ctx context.Context) error {
	log.Info("reconciliation:cleanupExpiredExceptions invoked (R1 placeholder, R3 implements)")
	return nil
}

// StartupRefreshView 启动时调用一次 RefreshView(D-02 避免冷启 0-5min 数据为空)。
//
// 与 ExecuteRefreshViewTask 的区别:
//   - 本函数接收 *gorm.DB 参数,直接构造 service 真实执行 SQL;
//   - ExecuteRefreshViewTask 是 InvokeTarget 字符串的 stub,无 core 依赖。
//
// 调用方:internal/core/core.go Init() 末尾异步调用一次,失败仅 log 不阻断启动。
func StartupRefreshView(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("StartupRefreshView: db is nil")
	}
	svc := NewReconciliationSnapshotService(db)
	return svc.RefreshView(ctx)
}
