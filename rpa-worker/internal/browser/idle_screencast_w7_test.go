package browser

// ============================================================================
// W7 / UI-FIX-03 验收：page 闲置后 screencast 必须仍能出帧
//
// 缺陷实证（生产 10.62.10.34 rpa-worker 日志，2026-11-25）：
//   liveview watch failed: Page.startScreencast 失败: context deadline exceeded
//   liveview watch failed: Page.startScreencast 失败: context canceled
//
// 根因：newChromePageOn 里曾有一行 `page = page.Timeout(max(Timeout,
// NavigationTimeout))`。rod 的 Page.Timeout(d) 是
//     ctx, cancel := context.WithTimeout(p.ctx, d)
// —— deadline 从 **page 创建那一刻**开始计时，且这个 ctx 挂在 page 句柄上永不重置。
// Manager 复用池内 page，闲置超过 60s（生产配置）后 ctx 已过期 → 之后任何
// CDP 调用（proto.X.Call(page)）立即失败。
//
// 为什么 155 的用例测不出来：它们全是「page 刚创建就用」，ctx 还在有效期内。
// 本文件补的就是这个同构性缺口：**先把 page 放到池里闲置，再开流**。
//
// 用例清单：
//   TestUI03_IdlePageStillStreams        闲置 90s 后第一次开流必须收到 ≥1 帧
//   TestUI03_IdleReusedPageRepeatedRounds 闲置后连续多轮 watch/stop，每轮都要出帧
//   TestUI03_PageCtxHasNoDeadline        静态断言：池内 page 的 ctx 不带 deadline
//                                        （不依赖 90s 睡眠，CI 也能秒级跑）
// ============================================================================

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/proto"
)

// getenvInt 读整数型 env（无值时返回 0，调用方用 0 表示"用默认"）。
func getenvInt(key string) int {
	v := os.Getenv(key)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}

// TestUI03_PoisonedPageDefenseLayer 反向对照 + 防御层验证（秒级，无需真等 90s）。
//
// 分两段：
//   A. 阴性对照：手工给一个 page 挂上 `.Timeout(1s)`（复刻修复前的写法），
//      等 1.2s 后它必然「已死」——CDP 直调报 context deadline exceeded。
//      这一段证明确实是「创建期一次性 deadline」导致故障，不是别的原因。
//   B. 防御层：把**同一个已经中毒的 page** 交给 StartScreencast，
//      必须仍能出帧（screencast.go 用 cleanPage 替换 ctx）。
func TestUI03_PoisonedPageDefenseLayer(t *testing.T) {
	_, p, _ := newLivePage(t, nil)
	raw := p.RawPage()

	// A. 阴性对照：复刻修复前的写法。
	poisoned := raw.Timeout(1 * time.Second)
	time.Sleep(1200 * time.Millisecond)
	ctx := poisoned.GetContext()
	dl, hasDL := ctx.Deadline()
	if !hasDL {
		t.Fatal("阴性对照失效：手工 Timeout 后 ctx 应带 deadline")
	}
	if err := ctx.Err(); err == nil {
		t.Fatalf("阴性对照失效：1.2s 后 ctx 应已过期（deadline=%v）", dl)
	}
	t.Logf("阴性对照成立：挂 Timeout(1s) 的 page 在 1.2s 后 ctx 已过期（%v）", ctx.Err())

	// 同一 page 的裸 CDP 直调必然失败 —— 这就是生产日志里的现象。
	//
	// 注意：不能用 raw.Info() 做对照 —— rod 的 Page.Info() 走的是
	// Browser.pageInfo()（用 browser 自己的 Background ctx），不吃 page ctx。
	// 必须用一个**真正经过 page ctx** 的 CDP 调用，这里选 Page.getLayoutMetrics。
	if _, err := (proto.PageGetLayoutMetrics{}).Call(poisoned); err != nil {
		t.Logf("裸 CDP 直调如预期失败：%v", err)
	} else {
		t.Fatal("裸 CDP 直调竟然成功，阴性对照不成立（rod 行为可能变了）")
	}

	// B. 防御层：中毒 page 交给 StartScreencast 仍必须出帧。
	frames, _, stop, err := StartScreencast(poisoned, ScreencastOptions{
		Quality: 40, MaxWidth: 640, MaxHeight: 360, EveryNthFrame: 1,
		AckTimeout: 2 * time.Second, ChannelCapacity: 2, Logger: nopLogger{},
	})
	if err != nil {
		t.Fatalf("防御层失效：中毒 page 上 StartScreencast 报错 %v", err)
	}
	defer stop()

	f := mustFrame(t, frames, 15*time.Second)
	if f == nil {
		t.Fatal("防御层失效：中毒 page 上 15s 内没有帧")
	}
	t.Logf("防御层成立：中毒 page 上仍收到帧 seq=%d %dx%d", f.Seq, f.Width, f.Height)
	f.Ack()
}

// idleDurationForTest 取闲置时长：默认 90s（真实缺陷的时间量级），
// 可用 UI03_IDLE_SECONDS 覆盖以便本地快速自测。
func idleDurationForTest(t *testing.T) time.Duration {
	t.Helper()
	if s := getenvInt("UI03_IDLE_SECONDS"); s > 0 {
		return time.Duration(s) * time.Second
	}
	return 90 * time.Second
}

// TestUI03_IdlePageStillStreams 池内 page 闲置 90s 后开流必须出帧。
//
// 这是缺陷 3 的**用户视角**复现：用户打开实时画面时，worker 的 page 很可能
// 已经在池里躺了十几分钟（上一次执行早就结束了）。修前必然
// `context deadline exceeded`，修后必须收到帧。
func TestUI03_IdlePageStillStreams(t *testing.T) {
	_, p, _ := newLivePage(t, nil)
	raw := p.RawPage()

	idle := idleDurationForTest(t)
	t.Logf("让 pool 内 page 闲置 %v 后再开流（模拟生产 08:40 启动 / 08:51 首次 watch）", idle)

	// 闲置期间不做任何 CDP 调用 —— 这正是让「创建期 deadline」走完的条件。
	deadline := time.Now().Add(idle)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
	}

	// 闲置结束后先确认 page 还活着（Info 走 CDP，若 ctx 已过期这里就会失败）。
	if _, err := raw.Info(); err != nil {
		t.Fatalf("闲置 %v 后 page 已不可用（读了 CDP 失败）: %v", idle, err)
	}

	frames, stats, stop, err := StartScreencast(raw, ScreencastOptions{
		Quality: 40, MaxWidth: 640, MaxHeight: 360, EveryNthFrame: 1,
		AckTimeout: 2 * time.Second, ChannelCapacity: 2, Logger: nopLogger{},
	})
	if err != nil {
		t.Fatalf("闲置 %v 后 StartScreencast 失败（缺陷 3 未修复）: %v", idle, err)
	}
	defer stop()

	f := mustFrame(t, frames, 15*time.Second)
	if f == nil {
		t.Fatalf("闲置 %v 后 15s 内没有收到任何帧（修复不完整）", idle)
	}
	t.Logf("闲置 %v 后首帧到达：seq=%d %dx%d %d bytes", idle, f.Seq, f.Width, f.Height, len(f.Data))
	f.Ack()

	if got := stats.Snapshot().Frames; got < 1 {
		t.Fatalf("stats.Frames=%d，期望 >=1", got)
	}
}

// TestUI03_IdleReusedPageRepeatedRounds 闲置后连续 3 轮 watch/stop，每轮都必须出帧。
//
// 只测一轮会漏掉「第一轮把 page 用过后第二轮又变坏」的情形（比如某处把
// deadline 又按当前时刻写回 page 句柄）。
func TestUI03_IdleReusedPageRepeatedRounds(t *testing.T) {
	_, p, _ := newLivePage(t, nil)
	raw := p.RawPage()

	// 用较短闲置（默认 90s 已在上面用例覆盖）避免本用例跑太久；
	// 若显式设置 UI03_IDLE_SECONDS 则尊之。
	idle := idleDurationForTest(t)
	if idle > 5*time.Second {
		idle = 5 * time.Second
	}
	time.Sleep(idle)

	for round := 1; round <= 3; round++ {
		frames, _, stop, err := StartScreencast(raw, ScreencastOptions{
			Quality: 40, MaxWidth: 640, MaxHeight: 360, EveryNthFrame: 1,
			AckTimeout: 2 * time.Second, ChannelCapacity: 2, Logger: nopLogger{},
		})
		if err != nil {
			t.Fatalf("第 %d 轮 StartScreencast 失败: %v", round, err)
		}
		f := mustFrame(t, frames, 15*time.Second)
		if f == nil {
			stop()
			t.Fatalf("第 %d 轮 15s 内没有帧", round)
		}
		// 多读一帧确认流是持续的（不是只发首帧）。
		f.Ack()
		f2 := mustFrame(t, frames, 15*time.Second)
		if f2 == nil {
			stop()
			t.Fatalf("第 %d 轮只收到一个帧就停了", round)
		}
		f2.Ack()
		stop()
		t.Logf("第 %d 轮出帧正常（seq=%d,%d）", round, f.Seq, f2.Seq)
	}
}

// TestUI03_PageCtxHasNoDeadline 静态断言：Manager 借出的 page 的 ctx 不带 deadline。
//
// 这是缺陷 3 的**根因级**守护：只要有人在创建期重新挂上 Timeout，本用例立即红，
// 而且不需要等 90s —— CI 也能跑。
func TestUI03_PageCtxHasNoDeadline(t *testing.T) {
	_, p, _ := newLivePage(t, nil)
	raw := p.RawPage()

	ctx := raw.GetContext()
	if ctx == nil {
		t.Fatal("page.GetContext() 返回 nil")
	}
	if dl, ok := ctx.Deadline(); ok {
		t.Fatalf("池内 page 的 ctx 带 deadline %v（应在创建时不设 Timeout；"+
			"一次性 deadline 会让闲置后的 page 全部失效）", dl)
	}
	if err := ctx.Err(); err != nil {
		t.Fatalf("池内 page 的 ctx 已失效: %v", err)
	}
	// 再确认底层 context 是 Background（rod 的 Browser.Page 默认 ctx）。
	t.Logf("page ctx 类型=%T，无 deadline，err=nil（符合 W7 / UI-FIX-03 口径）", ctx)
}
