package addomain

// Phase 144 · 同步漂移告警通道收口 —— 通道钩子（零循环依赖注入点）。
//
// 背景：internal/scheduler/ad_sync_tasks.go 与 internal/services（根包，
// 含 NoticeService / APISenderService）都 import 了 internal/services/addomain，
// 因此 addomain 不能再反向 import 二者（Go 包级循环依赖，编译期硬失败）。
// 本文件把 emit() 需要的三个外部能力抽象为「函数变量 + 注入点」，由
// internal/api/router.go 的 setupNoticeHub 装配阶段注入真实实现：
//   - WS notifier  ← scheduler 全局 noticeHubAdapter（同一个 adapter 实例）
//   - publisher    ← services.NewNoticeService(db).CreateNoticeWithTargets + PublishNotice
//   - webhook      ← services.APISenderService.SendWithDefaultConfig / Send
//
// 语义与 PLAN 的「方案 A + 经 scheduler.GetNoticeHub() 调用」等价：生产环境
// 广播仍走全局 hub，发布链路仍走 NoticeService，webhook 仍走 APISenderService；
// 额外收益是可单测（注入 mock）、零新包、零新依赖。未注入时 emit() 走
// 安全回退（软失败：仅 WARN，绝不阻断 cron）。

import (
	"context"
	"sync"

	"gorm.io/gorm"
)

// DriftAlertWSNotifier 广播接口，与 scheduler.NoticeHub / noticeHubAdapter 的
// BroadcastToAll(message interface{}) 同签名。
type DriftAlertWSNotifier interface {
	BroadcastToAll(message interface{})
}

// DriftAlertNotice 发布链路入参（本包自定义，避免 import internal/services）。
// 字段语义对齐 services.CreateNoticeRequest / models.Notice。
type DriftAlertNotice struct {
	Title      string
	Content    string
	NoticeType string
	Priority   int
	TargetType int
}

// DriftAlertNoticePublisher 发布链路：由 api 层注入
// services.NewNoticeService(db).CreateNoticeWithTargets + PublishNotice。
// 返回 noticeID；error 由 emit() 仅 WARN（软失败）。
type DriftAlertNoticePublisher func(ctx context.Context, db *gorm.DB, notice DriftAlertNotice) (string, error)

// DriftAlertWebhookMessage webhook 消息体（对齐 services.APIMessage）。
type DriftAlertWebhookMessage struct {
	Title   string
	Content string
	Data    map[string]interface{}
}

// DriftAlertWebhookResult webhook 触发结果。
// Message 必须已脱敏：禁止包含明文 webhook URL / token / secret。
type DriftAlertWebhookResult struct {
	Success    bool
	HTTPCode   int
	RetryCount int
	Message    string
}

// DriftAlertWebhookSender webhook 发送：由 api 层注入
// services.APISenderService.SendWithDefaultConfig / Send。
// configID 为空 = 用默认配置；非空 = 按 configID 取配置。
type DriftAlertWebhookSender func(ctx context.Context, db *gorm.DB, configID string, msg DriftAlertWebhookMessage) DriftAlertWebhookResult

var driftAlertHooks struct {
	sync.RWMutex
	wsNotifier       DriftAlertWSNotifier
	noticePublisher  DriftAlertNoticePublisher
	webhookSender    DriftAlertWebhookSender
}

// SetDriftAlertWSNotifier 注入 WS 广播器（生产传 scheduler 全局 adapter）。
func SetDriftAlertWSNotifier(n DriftAlertWSNotifier) {
	driftAlertHooks.Lock()
	defer driftAlertHooks.Unlock()
	driftAlertHooks.wsNotifier = n
}

// SetDriftAlertNoticePublisher 注入通知发布链路。
func SetDriftAlertNoticePublisher(fn DriftAlertNoticePublisher) {
	driftAlertHooks.Lock()
	defer driftAlertHooks.Unlock()
	driftAlertHooks.noticePublisher = fn
}

// SetDriftAlertWebhookSender 注入 webhook 发送器。
func SetDriftAlertWebhookSender(fn DriftAlertWebhookSender) {
	driftAlertHooks.Lock()
	defer driftAlertHooks.Unlock()
	driftAlertHooks.webhookSender = fn
}

func driftAlertWSNotifier() DriftAlertWSNotifier {
	driftAlertHooks.RLock()
	defer driftAlertHooks.RUnlock()
	return driftAlertHooks.wsNotifier
}

func driftAlertNoticePublisher() DriftAlertNoticePublisher {
	driftAlertHooks.RLock()
	defer driftAlertHooks.RUnlock()
	return driftAlertHooks.noticePublisher
}

func driftAlertWebhookSender() DriftAlertWebhookSender {
	driftAlertHooks.RLock()
	defer driftAlertHooks.RUnlock()
	return driftAlertHooks.webhookSender
}
