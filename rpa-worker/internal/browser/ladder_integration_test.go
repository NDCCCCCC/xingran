package browser

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// Phase 154 / ENGINE-03 验收：真实页面命用例
//
// 两个 fixture 场景（由 httptest 提供真实 HTML）：
//   A. testid 缺失但 role 存在   → 必须落在 role+name 级
//   B. id 每次变化（动态 id）     → 必须借 id* 容错或文本级命中
// 另含一个「全部 7 级失败」的负向用例，验证错误里带上逐级尝试结果。
// ============================================================================

const ladderFixtureHTML = `<!DOCTYPE html>
<html lang="zh-CN"><body>
  <!-- 场景 A1：没有 data-testid，但 role + 无障碍名存在（应命中第 2 级） -->
  <div role="button" id="save-btn-a1">保存草稿</div>

  <!-- 场景 A2：同样文本但没有 role（应降到第 5 级可见文本） -->
  <span id="plain-save">保存草稿</span>

  <!-- 场景 B：id 动态（request 级变化），但文本稳定 -->
  <div id="row-__SEQ__" class="row">订单编号 __SEQ__</div>

  <!-- 场景 C：只有 aria-label -->
  <div aria-label="关闭对话框">×</div>

  <!-- 场景 D：可见文本 -->
  <span class="tip">操作成功</span>
</body></html>`

func startLadderFixture(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seq := r.URL.Query().Get("seq")
		if seq == "" {
			seq = "8842"
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		body := strings.ReplaceAll(ladderFixtureHTML, "__SEQ__", seq)
		_, _ = fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestLadder_RealPage_RoleBeforeText 场景 A：同一文本，有 role 的元素
// 必须由第 2 级（role+name）命中，而不是先落到第 5 级可见文本 —— 这正是
// 「阶梯顺序」在真实页面上的可观测证据。
func TestLadder_RealPage_RoleBeforeText(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过真 Chromium 用例（-short）")
	}
	m := newTestManager(t, 1)
	p, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer m.Release(p)

	srv := startLadderFixture(t)
	if err := p.Goto(srv.URL + "/"); err != nil {
		t.Fatalf("Goto fixture: %v", err)
	}
	pm := NewPageManager(p, nopLogger{})

	// 录制器在「无 testid 传统」的内网系统上的典型产物就是可见文本/名称，
	// 阶梯必须在 role+name 级命中（不能先落到 visible-text）。
	el, tier, err := pm.findElement(`保存草稿`)
	if err != nil {
		t.Fatalf("阶梯应命中 role+name 级，实际失败: %v", err)
	}
	if tier != TierRoleName {
		t.Fatalf("有 role 的元素应由 role+name(第2级) 命中，实际 %s(score=%d)",
			tier.String(), TierScore(tier))
	}
	text, err := el.Text()
	if err != nil || !strings.Contains(text, "保存") {
		t.Fatalf("命中的元素不是预期按钮: text=%q err=%v", text, err)
	}
	t.Logf("场景 A 命中层级: %s (score=%d)", tier.String(), TierScore(tier))

	// 命中层级必须写进选择器事件信封（供学习分析）
	line := pm.SelectorEnvelopeLine()
	if line == "" {
		t.Fatal("命中后必须产出选择器事件信封行")
	}
	if !strings.Contains(line, `"tier":"`) {
		t.Fatalf("信封必须带 tier 维度: %s", line)
	}
}

// TestLadder_RealPage_DynamicID 场景 B：id 每次变化。
func TestLadder_RealPage_DynamicID(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过真 Chromium 用例（-short）")
	}
	m := newTestManager(t, 1)
	p, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer m.Release(p)

	srv := startLadderFixture(t)
	// 记录时的 id 是 row-1000，运行时页面变成了 row-9001。
	if err := p.Goto(srv.URL + "/?seq=9001"); err != nil {
		t.Fatalf("Goto fixture: %v", err)
	}
	pm := NewPageManager(p, nopLogger{})

	el, tier, err := pm.findElement("#row-1000")
	if err != nil {
		t.Fatalf("动态 id 场景应命中（id* 容错或文本级），实际失败: %v", err)
	}
	if tier == TierTestID {
		t.Fatalf("动态 id 不应被判定为 testid 级: %s", tier.String())
	}
	text, _ := el.Text()
	if !strings.Contains(text, "订单编号") {
		t.Fatalf("命中的元素不是订单行: %q", text)
	}
	t.Logf("场景 B 动态 id 命中层级: %s (score=%d)", tier.String(), TierScore(tier))
}

// TestLadder_RealPage_AllTiersFail 负向用例：错误必须逐级回显。
func TestLadder_RealPage_AllTiersFail(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过真 Chromium 用例（-short）")
	}
	m := newTestManager(t, 1)
	p, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer m.Release(p)

	srv := startLadderFixture(t)
	if err := p.Goto(srv.URL + "/"); err != nil {
		t.Fatalf("Goto fixture: %v", err)
	}
	pm := NewPageManager(p, nopLogger{})

	_, _, err = pm.findElement(`[data-testid="definitely-absent-xyz"]`)
	if err == nil {
		t.Fatal("不存在的元素必须返回 error")
	}
	var ladderErr *LadderError
	if !asLadderError(err, &ladderErr) {
		t.Fatalf("错误类型必须是 *LadderError: %T %v", err, err)
	}
	if len(ladderErr.Attempts) != 7 {
		t.Fatalf("错误必须带 7 级尝试结果，实际 %d", len(ladderErr.Attempts))
	}

	// 全失败也必须落一条 success=false 的信封事件（后端 3 次触发学习告警）
	line := pm.SelectorEnvelopeLine()
	if line == "" || !strings.Contains(line, `"success":false`) {
		t.Fatalf("全失败必须记录失败事件: %q", line)
	}
}

// asLadderError 简化 errors.As（避免额外 import 名字冲突）。
func asLadderError(err error, target **LadderError) bool {
	for err != nil {
		if e, ok := err.(*LadderError); ok {
			*target = e
			return true
		}
		type unwrapper interface{ Unwrap() error }
		if u, ok := err.(unwrapper); ok {
			err = u.Unwrap()
			continue
		}
		break
	}
	return false
}

var _ = time.Second
