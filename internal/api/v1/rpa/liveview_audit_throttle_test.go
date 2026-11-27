package rpa

// ============================================================================
// PR #114 CI 覆盖率补齐：输入审计节流器单测（此前 0/25 语句覆盖）
//
// 该节流器是 LIVEVIEW-02「输入注入有操作审计」的落地件，却从未有测试。
// 它直接决定 sys_rpa_audit_logs 会不会被鼠标移动刷爆（前端 50ms 节流后仍 20 次/秒）。
//
// 测法：只测**节流判定**这一纯内存逻辑，不碰数据库 ——
// 用一个 hub 为 nil 的 handler（h.audit 在 hub==nil 时直接 return），
// 于是整个用例零 IO、零 flake，断言对象是节流器自己的键表（last / n）。
// 「审计真的写进表」由 liveview_handler_contract_test.go 的 HTTP 用例覆盖。
// ============================================================================

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// resetInputAuditor 清空全局节流状态（用例间不串扰）。
func resetInputAuditor() {
	inputAuditor.mu.Lock()
	defer inputAuditor.mu.Unlock()
	inputAuditor.last = map[string]time.Time{}
	inputAuditor.n = map[string]int{}
}

// throttleCall 以指定身份调一次节流器（hub 为 nil → 不写库）。
func throttleCall(t *testing.T, h *LiveViewHandler, userID, executionID, kind string) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{}"))
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req
	ctx.Set("user_id", userID)
	ctx.Set("username", "throttle-user")
	auditInputThrottled(ctx, h, executionID, kind)
}

// throttleKeys 取当前节流键表快照。
func throttleKeys() []string {
	inputAuditor.mu.Lock()
	defer inputAuditor.mu.Unlock()
	keys := make([]string, 0, len(inputAuditor.last))
	for k := range inputAuditor.last {
		keys = append(keys, k)
	}
	return keys
}

// TestInputAuditThrottle_FirstWriteNotSwallowed 首次输入必落一条（键表出现新键）。
func TestInputAuditThrottle_FirstWriteNotSwallowed(t *testing.T) {
	resetInputAuditor()
	defer resetInputAuditor()

	h := &LiveViewHandler{} // hub 为 nil：不写库，纯测判定
	throttleCall(t, h, "h-thr-1", "exec-thr-1", "mousemove")

	require.Len(t, throttleKeys(), 1, "首次输入必须建立节流键（不得被吃掉）")
}

// TestInputAuditThrottle_WindowMerges 同 kind 窗口内 5 次只算 1 条，其余累加进 n。
func TestInputAuditThrottle_WindowMerges(t *testing.T) {
	resetInputAuditor()
	defer resetInputAuditor()

	h := &LiveViewHandler{}
	for i := 0; i < 5; i++ {
		throttleCall(t, h, "h-thr-2", "exec-thr-2", "mousemove")
	}

	require.Len(t, throttleKeys(), 1, "同一窗口内仍只有一个键")
	inputAuditor.mu.Lock()
	merged := inputAuditor.n["exec-thr-2|h-thr-2|mousemove"]
	inputAuditor.mu.Unlock()
	require.Equal(t, 4, merged, "5 次输入里第 1 次写审计、后 4 次累加计数")
}

// TestInputAuditThrottle_DifferentKindSeparate 不同 kind 各自独立（3 个键）。
func TestInputAuditThrottle_DifferentKindSeparate(t *testing.T) {
	resetInputAuditor()
	defer resetInputAuditor()

	h := &LiveViewHandler{}
	for _, kind := range []string{"mousemove", "click", "keydown"} {
		throttleCall(t, h, "h-thr-3", "exec-thr-3", kind)
	}
	require.Len(t, throttleKeys(), 3, "三种 kind 必须各建一个键（按 kind 分组）")
}

// TestInputAuditThrottle_DifferentUserSeparate 不同操作者各自独立（2 个键）。
func TestInputAuditThrottle_DifferentUserSeparate(t *testing.T) {
	resetInputAuditor()
	defer resetInputAuditor()

	h := &LiveViewHandler{}
	throttleCall(t, h, "h-thr-4a", "exec-thr-4", "mousemove")
	throttleCall(t, h, "h-thr-4b", "exec-thr-4", "mousemove")
	require.Len(t, throttleKeys(), 2, "不同操作者必须各建一个键（节流键含 operator）")
}

// TestInputAuditThrottle_DifferentExecutionSeparate 不同执行各自独立。
func TestInputAuditThrottle_DifferentExecutionSeparate(t *testing.T) {
	resetInputAuditor()
	defer resetInputAuditor()

	h := &LiveViewHandler{}
	throttleCall(t, h, "h-thr-5", "exec-thr-5a", "mousemove")
	throttleCall(t, h, "h-thr-5", "exec-thr-5b", "mousemove")
	require.Len(t, throttleKeys(), 2, "不同执行必须各建一个键")
}

// TestInputAuditThrottle_WindowExpires 窗口过期后重新写一条（旧键时间戳被刷新）。
func TestInputAuditThrottle_WindowExpires(t *testing.T) {
	resetInputAuditor()
	defer resetInputAuditor()

	h := &LiveViewHandler{}
	const key = "exec-thr-6|h-thr-6|mousemove"
	throttleCall(t, h, "h-thr-6", "exec-thr-6", "mousemove")

	// 手动把时间戳拨回窗口之前，模拟「1 秒已过」。
	inputAuditor.mu.Lock()
	inputAuditor.last[key] = time.Now().Add(-2 * inputAuditWindow)
	inputAuditor.mu.Unlock()

	throttleCall(t, h, "h-thr-6", "exec-thr-6", "mousemove")

	inputAuditor.mu.Lock()
	seen := time.Since(inputAuditor.last[key])
	inputAuditor.mu.Unlock()
	require.Less(t, seen, inputAuditWindow, "窗口过后必须刷新时间戳（重新允许写审计）")
}
