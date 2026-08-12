package asset

import (
	"context"
	"fmt"
	"sync"
	"time"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
	"gorm.io/gorm"
)

// ============================================================================
// Phase 46 R5 — 误修复率监控 service
//
// 锁定的决策(详见 .planning/phases/46-r5/46-CONTEXT.md):
//   - D-C5: 误修复率 = rolled_back / applied(7d 滑动窗口)
//   - 阈值可配置 sys_config "asset.reconciliation.fix.mis_fix_threshold"(默认 0.01)
//   - 超过阈值 → 写 sys_notice + WS 广播 + sys_oper_log 审计
//
// 防告警风暴(W-7 修订):
//   - in-memory 缓存 lastBreachNotifiedAt time.Time
//   - mutex 保护并发安全
//   - 仅在 1h 节流后 或 状态从非 breach 转为 breach 时才推送
//
// 配套 cron(46-02 Task 3): 7,17,27,37,47,57 * * * *
//   - 错峰 generator 3,8,13,18,...(每 10min 第 3 分钟)
//   - 避免 monitor 与 generator 同时触发争用 sys_reconciliation_fix_suggestion 写
// ============================================================================

// FixSuggestionMisFixRateNoticeTitle 误修复率告警标题常量(46-02 / D-C5)
//
// 用于 NoticeService + WS 广播 统一标题,前端可据此过滤。
const FixSuggestionMisFixRateNoticeTitle = "资产对账误修复率超阈告警"

// FixSuggestionMisFixRateNoticeType 误修复率告警类型常量
//
// WS 推送的 Type 字段。前端 useReconciliationWebSocket hook 可据此过滤事件。
const FixSuggestionMisFixRateNoticeType = "fix_suggestion_mis_fix_rate_breach"

// misFixRateThrottleInterval 告警节流时间窗
//
// 防告警风暴:同一 breach 持续期间,1h 内最多推送 1 次。
// 1h 选取依据:误修复率是慢变量(7d 窗口),10min cron 1h 6 次推送没有信息增量。
const misFixRateThrottleInterval = 1 * time.Hour

// FixSuggestionMonitor 误修复率监控 service
//
// 依赖注入:
//   - db: *gorm.DB — 直接查询 sys_reconciliation_fix_suggestion
//   - configService: 读 sys_config "asset.reconciliation.fix.mis_fix_threshold"
//   - noticeHub: WS 广播(可空,nil 时跳过 WS)
//   - noticeService: SysNotice 写入(可空,nil 时跳过 SysNotice)
//   - service: FixSuggestionService — 复用 Stats() 取 7d 窗口数据
type FixSuggestionMonitor struct {
	db            *gorm.DB
	configService system.ConfigService
	noticeHub     *websocket.NoticeHub
	noticeService *services.NoticeService
	service       FixSuggestionService

	// 防告警风暴 in-memory 状态
	mu                    sync.Mutex
	lastBreachNotifiedAt  time.Time
	lastBreached          bool
}

// NewFixSuggestionMonitor 构造函数
func NewFixSuggestionMonitor(
	db *gorm.DB,
	configSvc system.ConfigService,
	noticeHub *websocket.NoticeHub,
	noticeSvc *services.NoticeService,
	svc FixSuggestionService,
) *FixSuggestionMonitor {
	return &FixSuggestionMonitor{
		db:            db,
		configService: configSvc,
		noticeHub:     noticeHub,
		noticeService: noticeSvc,
		service:       svc,
	}
}

// CheckAndNotify 检测 7d 误修复率,超阈时通知(节流 1h)
//
// 流程:
//  1. 调 FixSuggestionService.Stats(ctx, 7) 取 7d 窗口数据(applied / rolledBack / misFixRate / threshold / thresholdBreached)
//  2. 读 sys_config "asset.reconciliation.fix.mis_fix_threshold"(默认 0.01) — Stats 内部已读,这里用 stats.Threshold
//  3. 若 !stats.ThresholdBreached → 清理 lastBreached 标记,直接 return nil
//  4. 若 stats.ThresholdBreached → 检查节流:
//     - lastBreachNotifiedAt + 1h > now → 跳过(防风暴)
//     - 否则更新 lastBreachNotifiedAt = now + 推送 notice + WS + operlog
//
// 软失败:任何一步失败仅 applogger.Warnf,不影响 cron 退出码。
func (m *FixSuggestionMonitor) CheckAndNotify(ctx context.Context) error {
	if m.service == nil {
		applogger.Warnf("[fix-suggestion:Monitor] service 未注入,跳过本次检查")
		return nil
	}

	// 1. 取 7d 统计
	stats, err := m.service.Stats(ctx, 7)
	if err != nil {
		applogger.Warnf("[fix-suggestion:Monitor] Stats 失败(忽略): %v", err)
		return nil
	}

	// 2-3. 未超阈 → 清理状态 + return
	if !stats.ThresholdBreached {
		m.mu.Lock()
		m.lastBreached = false
		m.mu.Unlock()
		applogger.Debugf("[fix-suggestion:Monitor] 7d misFixRate=%.4f (applied=%d, rolledBack=%d) 未超阈 %.4f,无需通知",
			stats.MisFixRate, stats.Applied, stats.RolledBack, stats.Threshold)
		return nil
	}

	// 4. 超阈 → 检查节流
	now := time.Now()
	shouldNotify, isStateChange := m.shouldNotifyBreach(now)
	if !shouldNotify {
		applogger.Debugf("[fix-suggestion:Monitor] 7d misFixRate=%.4f 超阈但 1h 节流中,跳过推送", stats.MisFixRate)
		return nil
	}

	// 5. 推送 SysNotice + WS + operlog
	noticeContent := fmt.Sprintf("7d 误修复率 %.2f%% 超过阈值 %.2f%%(applied=%d, rolledBack=%d)。请检查 R5 修复建议生成器的 confidence 阈值与样本质量。",
		stats.MisFixRate*100, stats.Threshold*100, stats.Applied, stats.RolledBack)

	applogger.Warnf("[fix-suggestion:Monitor] 7d misFixRate=%.4f 超过阈值 %.4f (applied=%d, rolledBack=%d, stateChange=%v)",
		stats.MisFixRate, stats.Threshold, stats.Applied, stats.RolledBack, isStateChange)

	// 5a. WS 广播(广播给所有在线用户,前端按 Type 过滤)
	if m.noticeHub != nil {
		defer func() {
			// Hub 关闭时 BroadcastToAll 写 channel 会 panic,兜底避免 cron 崩溃
			if r := recover(); r != nil {
				applogger.Warnf("[fix-suggestion:Monitor] WS 广播异常(忽略): %v", r)
			}
		}()
		m.noticeHub.BroadcastToAll(websocket.NoticeMessage{
			Type:      FixSuggestionMisFixRateNoticeType,
			Title:     FixSuggestionMisFixRateNoticeTitle,
			Content:   noticeContent,
			Priority:  2, // 紧急
			Timestamp: now.Unix(),
		})
	}

	// 5b. SysNotice 写入(D-C5 双通道之 SysNotice 通道,与 R2 critical_workorder 同模式)
	if m.noticeService != nil {
		req := &services.CreateNoticeRequest{
			NoticeTitle:   FixSuggestionMisFixRateNoticeTitle,
			NoticeType:    "2", // 2=警告
			NoticeContent: "[fix_suggestion_mis_fix_rate_breach]\n" + noticeContent,
			Priority:      models.PriorityUrgent,
			TargetType:    models.TargetAll,
			IsMarkdown:    false,
		}
		notice, nerr := m.noticeService.CreateNoticeWithTargets(ctx, req, "SYSTEM", "SYSTEM")
		if nerr != nil {
			applogger.Warnf("[fix-suggestion:Monitor] SysNotice 创建失败: %v", nerr)
		} else if perr := m.noticeService.PublishNotice(ctx, notice.ID); perr != nil {
			applogger.Warnf("[fix-suggestion:Monitor] SysNotice 发布失败: %v", perr)
		} else {
			applogger.Infof("[fix-suggestion:Monitor] SysNotice 已发布 notice_id=%s", notice.ID)
		}
	}

	// 5c. sys_oper_log 审计(OperTypeUpdate=2 标记"监控告警")
	//  使用 direct INSERT,无需 gin.Context
	operLog := &models.OperLog{
		Title:        FixSuggestionMisFixRateNoticeTitle,
		BusinessType: 2, // OperTypeUpdate
		Method:       "fix_suggestion_monitor.CheckAndNotify",
		Status:       0, // 0=成功
		OperTime:     now,
		CostTime:     0,
	}
	jsonResult := fmt.Sprintf(`{"misFixRate":%.4f,"threshold":%.4f,"applied":%d,"rolledBack":%d,"stateChange":%v}`,
		stats.MisFixRate, stats.Threshold, stats.Applied, stats.RolledBack, isStateChange)
	operLog.JsonResult = &jsonResult
	if err := m.db.WithContext(ctx).Create(operLog).Error; err != nil {
		applogger.Warnf("[fix-suggestion:Monitor] operlog 写入失败(忽略): %v", err)
	}

	return nil
}

// shouldNotifyBreach 决定是否推送告警(节流 + 状态变化检测)
//
// 返回:
//   - shouldNotify: true = 推送告警
//   - isStateChange: true = 状态从非 breach 转为 breach(可作"首次告警"标记)
//
// 节流规则(46-02 / W-7 修订):
//   - 状态从非 breach → breach:必推送(首次告警)
//   - 状态持续 breach:每 1h 最多推送 1 次
func (m *FixSuggestionMonitor) shouldNotifyBreach(now time.Time) (shouldNotify bool, isStateChange bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	wasBreached := m.lastBreached
	isStateChange = !wasBreached

	// 首次告警 或 距上次推送 > 1h
	if isStateChange || now.Sub(m.lastBreachNotifiedAt) >= misFixRateThrottleInterval {
		m.lastBreachNotifiedAt = now
		m.lastBreached = true
		return true, isStateChange
	}

	return false, false
}
