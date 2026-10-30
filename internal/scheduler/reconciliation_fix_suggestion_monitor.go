package scheduler

import (
	"context"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/asset"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"gorm.io/gorm"
)

// RegisterFixSuggestionMisFixMonitor 注册误修复率监控 cron 任务(46-02 / D-C5)
//
// 设计要点:
//   - 单一 taskType "reconciliation" 复用 reconciliation_tasks.go 的 switch 分发,
//     通过 params["target"] == "monitorFixSuggestionMisFix" 路由到本监控
//   - cron 表达式 "7,17,27,37,47,57 * * * *" — 每 10 分钟的第 7 分钟
//   - 错峰 generator 3,8,13,18,... 至少 4 分钟,避免与 generator 同时争用
//     sys_reconciliation_fix_suggestion 写锁(W-7 修订)
//
// 调用链:
//  cron tick → RegisterReconciliationTasks switch "monitorFixSuggestionMisFix"
//    → fixSuggestionMonitor.CheckAndNotify(ctx)
//    → FixSuggestionService.Stats(ctx, 7) + 阈值判定 + 1h 节流 + WS + SysNotice + operlog
//
// 软失败:CheckAndNotify 内部所有错误仅 applogger.Warnf,不返回 error(cron exit 0)
//   — 监控告警失败不应阻塞对账主流程。
func RegisterFixSuggestionMisFixMonitor(s *Scheduler, db *gorm.DB, cacheSvc cache.Cache, wsSvc *websocket.NoticeHub, noticeSvc *services.NoticeService) {
	_ = cacheSvc // 暂未使用(预留:若 monitor 需要失效 cache 可注入)

	// 构造 monitor(46-02 / D-C5)
	configSvc := system.NewConfigService(db)
	fixSuggestionSvc := asset.NewFixSuggestionService(db, nil, configSvc, wsSvc)
	monitor := asset.NewFixSuggestionMonitor(db, configSvc, wsSvc, noticeSvc, fixSuggestionSvc)

	// 注册到 Scheduler(单 taskType "reconciliation")
	s.RegisterTask("reconciliation", func(ctx context.Context, params map[string]interface{}) error {
		target, _ := params["param"].(string)
		if target != "monitorFixSuggestionMisFix" {
			// 非本监控任务,跳过(由 reconciliation_tasks.go 的 switch 处理)
			return nil
		}
		applogger.Infof("[reconciliation:monitorFixSuggestionMisFix] 开始执行")
		if err := monitor.CheckAndNotify(ctx); err != nil {
			applogger.Errorf("[reconciliation:monitorFixSuggestionMisFix] 失败: %v", err)
			return err
		}
		applogger.Infof("[reconciliation:monitorFixSuggestionMisFix] 完成")
		return nil
	})

	applogger.Infof("误修复率监控已注册(reconciliation:monitorFixSuggestionMisFix, cron 7,17,27,37,47,57 * * * *)")
}
