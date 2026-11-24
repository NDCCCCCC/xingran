package rpa

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	svcrpa "github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// ============================================================================
// Phase 155 / LIVEVIEW-02：输入审计的节流器。
//
// 输入是高频事件（鼠标移动按前端 50ms 节流后仍有 20/s），逐条写
// sys_rpa_audit_logs 会在几秒内把表刷爆、并让审计本身成为 IO 瓶颈。做法：
//
//	同一 execution + 同一 operator + 同一 kind 的输入，1 秒窗口内只写一条，
//	窗口内的额外输入累计进 next（kind 相同则累加计数）。
//
// 这满足 LIVEVIEW-02「输入注入有操作审计」的要求，同时把写放大控制在
// 「每 kind 每秒 1 条」。审计条目含 count 字段说明窗口内合并了多少次输入。
// ============================================================================

const (
	// timeoutSeconds handler 侧审计/DB 操作的通用超时（秒）。
	timeoutSeconds = 5
	// inputAuditWindow 同 kind 输入的审计合并窗口。
	inputAuditWindow = time.Second
)

// inputAuditThrottler 按 (executionID, operator, kind) 合并输入审计。
type inputAuditThrottler struct {
	mu   sync.Mutex
	last map[string]time.Time
	n    map[string]int
}

var inputAuditor = &inputAuditThrottler{
	last: map[string]time.Time{},
	n:    map[string]int{},
}

// auditInputThrottled 按窗口合并写入审计；窗口内重复调用只累加计数。
func auditInputThrottled(c *gin.Context, h *LiveViewHandler, executionID, kind string) {
	uid := currentUserID(c)
	key := executionID + "|" + uid + "|" + kind

	inputAuditor.mu.Lock()
	now := time.Now()
	prev, seen := inputAuditor.last[key]
	if seen && now.Sub(prev) < inputAuditWindow {
		inputAuditor.n[key]++
		inputAuditor.mu.Unlock()
		return
	}
	merged := inputAuditor.n[key] + 1
	inputAuditor.n[key] = 0
	inputAuditor.last[key] = now
	inputAuditor.mu.Unlock()

	h.audit(c, executionID, svcrpa.LiveViewEventInput, map[string]interface{}{
		"kind":  kind,
		"count": merged,
	})
}

// warnf 输出 WARN 日志（handler 内需要"可读但非致命"的诊断时用）。
//
// 直接走 pkg/logger 的函数式接口：本包已有多个 handler 用 applogger.Warnf，
// 保持一致；此封装只是给 liveview handler 一个短名字。
func warnf(format string, args ...interface{}) {
	applogger.Warnf(format, args...)
}
