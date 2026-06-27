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

// RefreshView REFRESH MATERIALIZED VIEW CONCURRENTLY reconciliation_normalized.
//
// D-02: 失败仅 logrus 日志,不返回 error 给调用方(handler 自行处理 500 / cron 写 job_log)。
// 实际策略: 返回 error 让调用方决定(cron 调用方收到 error 后写 job_log;
// handler 调用方把 error 透传给客户端 500)。日志由本方法内置。
func (s *reconciliationSnapshotServiceImpl) RefreshView(ctx context.Context) error {
	if err := s.db.WithContext(ctx).Exec("REFRESH MATERIALIZED VIEW CONCURRENTLY reconciliation_normalized").Error; err != nil {
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
