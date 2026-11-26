package browser

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/xingran-next/rpa-worker/internal/logger"
)

// ============================================================================
// Phase 155 / LIVEVIEW-01：CDP screencast 帧流（**ACK 即背压**）
//
// 硬约束（来自 .planning/notes/261123-v156-research-liveview-nocode.md §1）：
//
//  1. `Page.startScreencast` **必须逐帧 ACK**（`Page.screencastFrameAck`），
//     不 ACK 会立刻停止发帧。因此本实现把 ACK 挂在「消费者处理完该帧之后」——
//     前端消费速度即天然限流阀，不需要额外的 timer 节流。
//  2. headless 实测普遍 1–5fps（headful 5–10fps）；靠 `maxWidth/maxHeight`
//     降分辨率 + `quality` 降质 + `everyNthFrame` 降采样三件套拉高，
//     **不追求 60fps**。
//  3. 端到端可感知延迟下限约 80–120ms（含渲染与 JPEG 压缩），UI 必须用
//     「已接管」的显式状态而不是假装零延迟。
//
// 生命周期硬语义（154-SUMMARY §7.4）：
//   - 只操作 `Page.RawPage()`（当前活动页签）；进入 frame 后 ContextPage 变了，
//     但 screencast 必须留在页签句柄上。
//   - Stop 只调 `Page.stopScreencast` 并退出事件泵，**绝不** Close page 或 browser。
//   - switchTab 会让 RawPage 跟随到新页签，调用方需重建 screencast。
// ============================================================================

// ScreencastOptions screencast 启动参数（全零值可用，零值经 normalize 填默认）。
type ScreencastOptions struct {
	// Quality JPEG 质量 [0,100]。默认 50（40–60 是调研给出的甜点区间）。
	Quality int
	// MaxWidth / MaxHeight 最大像素尺寸（Chrome 按比例缩）。
	// 默认 1280×720 —— 与 LIVEVIEW-01「≤3Mbps@1280×720」的验收口径一致。
	MaxWidth  int
	MaxHeight int
	// EveryNthFrame 每 N 帧只发 1 帧（降采样）。默认 2。
	EveryNthFrame int
	// AckTimeout 单帧等待消费者确认的上限。超时则强制 ACK 并计 dropped，
	// 避免一个卡死的消费者把 rod 事件泵（进而把 WaitLoad 等）永久挂住。
	// 默认 800ms（5fps 消费者单帧预算 200ms，留 4× 余量）。
	AckTimeout time.Duration
	// ChannelCapacity 帧 channel 缓冲。默认 2（1 帧在途 + 1 帧预取）。
	ChannelCapacity int
	// Logger 可选；nil 时不输出。
	Logger logger.Logger
}

// 默认参数集中定义，供 config 与测试共用（避免默认值散落）。
const (
	DefaultScreencastQuality         = 50
	DefaultScreencastMaxWidth        = 1280
	DefaultScreencastMaxHeight       = 720
	DefaultScreencastEveryNthFrame   = 2
	DefaultScreencastAckTimeout      = 800 * time.Millisecond
	DefaultScreencastChannelCapacity = 2
)

// normalize 填默认值（越界值回落默认，不 panic）。
func (o ScreencastOptions) normalize() ScreencastOptions {
	out := o
	if out.Quality <= 0 || out.Quality > 100 {
		out.Quality = DefaultScreencastQuality
	}
	if out.MaxWidth <= 0 {
		out.MaxWidth = DefaultScreencastMaxWidth
	}
	if out.MaxHeight <= 0 {
		out.MaxHeight = DefaultScreencastMaxHeight
	}
	if out.EveryNthFrame <= 0 {
		out.EveryNthFrame = DefaultScreencastEveryNthFrame
	}
	if out.AckTimeout <= 0 {
		out.AckTimeout = DefaultScreencastAckTimeout
	}
	if out.ChannelCapacity <= 0 {
		out.ChannelCapacity = DefaultScreencastChannelCapacity
	}
	return out
}

// ScreencastFrame 一帧画面 + 消费确认回调。
//
// 使用约定（**必须遵守**）：从 channel 取出 Frame 后，无论成功推给前端还是失败，
// 都必须在处理结束时调用 `Ack()`（幂等）。不调用会导致下一帧在 AckTimeout 之后
// 被强制放行并计 dropped —— 即「不 ACK = 丢帧」，这正是背压的实现方式。
type ScreencastFrame struct {
	// Data JPEG 原始字节（未 base64）。
	Data []byte
	// Metadata CDP 原始元数据（deviceWidth/Height、scroll 偏移、pageScaleFactor）。
	Metadata *proto.PageScreencastFrameMetadata
	// SessionID CDP 侧帧号（ACK 必须回传同一个）。
	SessionID int
	// Seq 本路 stream 的递增序号（1 起，含 drop 帧，便于前端判断丢帧）。
	Seq int64
	// Width / Height JPEG 实际像素尺寸（由 JPEG SOF 段解析，0 表示解析失败）。
	Width  int
	Height int
	// CapturedAt 本地收到该帧的时间（端到端延迟 = 前端收到时刻 − 本时刻 − 传输耗时）。
	CapturedAt time.Time

	ackOnce sync.Once
	ackFn   func()
}

// Ack 确认该帧已消费。幂等，可安全重复调用。
func (f *ScreencastFrame) Ack() {
	if f == nil {
		return
	}
	f.ackOnce.Do(func() {
		if f.ackFn != nil {
			f.ackFn()
		}
	})
}

// Base64Data 帧内容的 base64 表达（协议字段 LiveFrame.Data）。
func (f *ScreencastFrame) Base64Data() string {
	if f == nil || len(f.Data) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(f.Data)
}

// ScreencastStats 单路推流的运行统计（fps / 带宽实测的数据来源）。
//
// 混合使用 atomic.Int64 与 Atomic 包装：增量走 atomic.Add，快照无锁读取，
// 保证边推流边取数不引入锁竞争（推流路径是延迟敏感的）。
type ScreencastStats struct {
	startedAt time.Time
	frames    int64 // 已投递给消费者的帧
	acks      int64 // 消费者显式 ACK 的帧
	dropped   int64 // 超时被强制 ACK 的帧
	bytes     int64 // 已投递帧的 JPEG 字节总量
	lastFrame atomic.Int64 // 最近一帧的 UnixMilli（0=尚未收帧）
}

// NewScreencastStats 构造统计起点（startedAt 可由调用方给，便于跨 tab 重建时延续）。
func NewScreencastStats(startedAt time.Time) *ScreencastStats {
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	return &ScreencastStats{startedAt: startedAt}
}

// ScreencastStatsSnapshot 统计快照（含派生量：实测 fps 与 Mbps）。
type ScreencastStatsSnapshot struct {
	StartedAt time.Time `json:"startedAt"`
	ElapsedMS int64     `json:"elapsedMs"`
	Frames    int64     `json:"frames"`
	Acks      int64     `json:"acks"`
	Dropped   int64     `json:"dropped"`
	Bytes     int64     `json:"bytes"`
	// FPS 实测帧率 = Frames / 已流逝秒（含 drop，反映「投递给消费者的节奏」）。
	FPS float64 `json:"fps"`
	// Mbps 单路带宽 = Bytes*8 / 已流逝秒 / 1e6。
	Mbps float64 `json:"mbps"`
	// BytesPerFrame 平均帧字节数（带宽核算与降档决策用）。
	BytesPerFrame float64 `json:"bytesPerFrame"`
}

// Snapshot 取当前统计快照。
func (s *ScreencastStats) Snapshot() ScreencastStatsSnapshot {
	if s == nil {
		return ScreencastStatsSnapshot{}
	}
	elapsed := time.Since(s.startedAt)
	out := ScreencastStatsSnapshot{
		StartedAt: s.startedAt,
		ElapsedMS: elapsed.Milliseconds(),
		Frames:    atomic.LoadInt64(&s.frames),
		Acks:      atomic.LoadInt64(&s.acks),
		Dropped:   atomic.LoadInt64(&s.dropped),
		Bytes:     atomic.LoadInt64(&s.bytes),
	}
	if sec := elapsed.Seconds(); sec > 0 {
		out.FPS = float64(out.Frames) / sec
		out.Mbps = float64(out.Bytes) * 8 / sec / 1e6
	}
	if out.Frames > 0 {
		out.BytesPerFrame = float64(out.Bytes) / float64(out.Frames)
	}
	return out
}

// LastFrameAt 最近一帧的接收时间（零值表示还没收到帧）。用于「无帧即停流」判定。
func (s *ScreencastStats) LastFrameAt() time.Time {
	if s == nil {
		return time.Time{}
	}
	ms := s.lastFrame.Load()
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}

// Screencast 一路 screencast 流的句柄。
type Screencast struct {
	page   *rod.Page
	opts   ScreencastOptions
	logger logger.Logger
	stats  *ScreencastStats

	frames chan *ScreencastFrame

	ctx    context.Context
	cancel context.CancelFunc

	pumpDone    chan struct{} // 事件泵退出（唯一能保证"不再有 CDP 回调"的信号）
	framesOnce  sync.Once     // 关闭 frames channel
	stopOnce    sync.Once     // 停止逻辑只跑一次
	finishedCh  chan struct{}
	seq         atomic.Int64
	closed      atomic.Bool
}

// StartScreencast 在页签上启动 screencast。
//
// 返回：
//   - `frames`：帧 channel。**Stop 之后该 channel 会被关闭**，消费方用 `v, ok := <-frames` 判断。
//   - `stats`：统计指针（fps/带宽数据来源，恒非 nil）。
//   - `stop`：停止函数（幂等）。只调 `Page.stopScreencast`，**不关 page、不关 browser**。
//
// page 必须来自 `Page.RawPage()`（当前活动页签）。传 nil 时返回可读错误与已关闭的 channel。
//
// W7 / UI-FIX-03 防御层（方案 B）：无论调用方传入的 page 带不带 deadline，这里都
// 用**干净 ctx** 替换后再发 CDP 调用。历史生产日志：
//
//	Page.startScreencast 失败: context deadline exceeded
//	Page.startScreencast 失败: context canceled
//
// 根因是 page 在池里闲置超过创建期 Timeout（已在上游删掉那个全局 Timeout），
// 但 screencast 是「唯一在 page 闲置后才可能被触发」的操作，多一层防御成本极低。
func StartScreencast(page *rod.Page, opts ScreencastOptions) (
	frames <-chan *ScreencastFrame, stats *ScreencastStats, stop func(), err error,
) {
	opts = opts.normalize()

	if page == nil {
		empty := make(chan *ScreencastFrame)
		close(empty)
		return empty, nil, func() {}, fmt.Errorf("start screencast: page 为 nil")
	}

	// 事件泵需要一个可取消的 page 上下文：ctx 取消后 rod 的 event channel 关闭，
	// EachEvent 的 wait 循环随之退出 —— 这是让泵优雅退出的唯一正确方式
	// （rod 的 wait 只能调用一次，且不会因为单个回调返回 false 就留活口）。
	ctx, cancel := context.WithCancel(context.Background())
	bgPage := page.Context(ctx)

	// W7 / UI-FIX-03 防御层：CDP 直调（Call）必须用**不带 deadline** 的句柄。
	// rod 的 Page.Context(ctx) 是纯替换（不继承旧 ctx 的 deadline），所以这里
	// 用 context.Background() 得到的 cleanPage 保证「无论上游传什么 ctx，
	// startScreencast / screencastFrameAck / stopScreencast 都不会因历史 deadline
	// 立即失败」。事件订阅仍用 bgPage（带可取消 ctx，供 Stop 优雅退出）。
	cleanPage := page.Context(context.Background())

	ch := make(chan *ScreencastFrame, opts.ChannelCapacity)
	st := NewScreencastStats(time.Now())

	s := &Screencast{
		page:       cleanPage,
		opts:       opts,
		logger:     opts.Logger,
		stats:      st,
		frames:     ch,
		ctx:        ctx,
		cancel:     cancel,
		pumpDone:   make(chan struct{}),
		finishedCh: make(chan struct{}),
	}

	// Chrome 对「不可见」页面不发 screencast 帧（headless 下常见）。
	// 主动把页面标记为 active，避免「startScreencast 成功但 0 帧」的假成功。
	if err := (proto.PageSetWebLifecycleState{
		State: proto.PageSetWebLifecycleStateStateActive,
	}).Call(cleanPage); err != nil && opts.Logger != nil {
		opts.Logger.Debug("Page.setWebLifecycleState(active) 失败（继续，可能仍能发帧）", logger.Err(err))
	}

	// **先订阅事件再 startScreencast**：反序会让首帧在订阅建立前发出并被丢弃。
	wait := bgPage.EachEvent(func(e *proto.PageScreencastFrame) bool {
		s.handleFrame(e)
		return s.closed.Load()
	})

	quality := opts.Quality
	maxW := opts.MaxWidth
	maxH := opts.MaxHeight
	nth := opts.EveryNthFrame

	if err := (proto.PageStartScreencast{
		Format:        proto.PageStartScreencastFormatJpeg,
		Quality:       &quality,
		MaxWidth:      &maxW,
		MaxHeight:     &maxH,
		EveryNthFrame: &nth,
	}).Call(cleanPage); err != nil {
		cancel()
		close(ch)
		return make(chan *ScreencastFrame), nil, func() {}, fmt.Errorf("Page.startScreencast 失败: %w", err)
	}

	go func() {
		defer close(s.pumpDone)
		defer s.closeFrames()
		wait()
	}()

	return ch, st, s.Stop, nil
}

// handleFrame 事件泵内收帧：投递 → 等消费确认 → ACK。
//
// 这里的等待就是 LIVEVIEW-01 要求的背压：消费者处理完上一帧之前本帧不被 ACK，
// Chrome 就不再发下一帧（CDP screencast 的天然流控）。
func (s *Screencast) handleFrame(e *proto.PageScreencastFrame) {
	if e == nil || len(e.Data) == 0 {
		return
	}

	// ACK 函数：无论成功/失败/超时，最终都必须回传，否则 Chrome 侧会积压未确认帧。
	var ackOnce sync.Once
	acked := make(chan struct{})
	ack := func() {
		ackOnce.Do(func() {
			_ = (proto.PageScreencastFrameAck{SessionID: e.SessionID}).Call(s.page)
			close(acked)
		})
	}

	if s.closed.Load() {
		ack()
		return
	}

	seq := s.seq.Add(1)
	w, h := jpegSize(e.Data)

	frame := &ScreencastFrame{
		Data:       e.Data,
		Metadata:   e.Metadata,
		SessionID:  e.SessionID,
		Seq:        seq,
		Width:      w,
		Height:     h,
		CapturedAt: time.Now(),
	}
	frame.ackFn = ack

	atomic.AddInt64(&s.stats.frames, 1)
	atomic.AddInt64(&s.stats.bytes, int64(len(e.Data)))
	s.stats.lastFrame.Store(frame.CapturedAt.UnixMilli())

	// 投递（channel 满则等到 AckTimeout）。用显式 Timer 而非 time.After，
	// 避免每帧产生一个拖到 GC 才回收的 timer。
	deliverTimer := time.NewTimer(s.opts.AckTimeout)
	select {
	case s.frames <- frame:
		deliverTimer.Stop()
	case <-s.ctx.Done():
		deliverTimer.Stop()
		ack()
		return
	case <-deliverTimer.C:
		// 消费者堵住：强制 ACK 并计 dropped，让事件泵继续存活，
		// 否则整个 page 的 CDP 事件（含 WaitLoad）都会被拖死。
		atomic.AddInt64(&s.stats.dropped, 1)
		if s.logger != nil {
			s.logger.Warn("screencast 投递超时，强制 ACK 并丢帧",
				logger.Int64("seq", seq),
				logger.Duration("ack_timeout", s.opts.AckTimeout))
		}
		ack()
		return
	}

	// 等消费者处理完（背压的关键等待），或超时强制放行。
	ackTimer := time.NewTimer(s.opts.AckTimeout)
	select {
	case <-acked:
		ackTimer.Stop()
		atomic.AddInt64(&s.stats.acks, 1)
	case <-s.ctx.Done():
		ackTimer.Stop()
		ack()
	case <-ackTimer.C:
		atomic.AddInt64(&s.stats.dropped, 1)
		if s.logger != nil {
			s.logger.Warn("screencast 消费确认超时，强制 ACK",
				logger.Int64("seq", seq),
				logger.Duration("ack_timeout", s.opts.AckTimeout))
		}
		ack()
	}
}

// closeFrames 关闭帧 channel（幂等）。由事件泵退出时调用。
func (s *Screencast) closeFrames() {
	s.framesOnce.Do(func() {
		close(s.frames)
	})
}

// Stop 停止 screencast（与 StartScreencast 返回的 stop 等价，幂等）。
//
// 顺序：置 closed → stopScreencast → cancel ctx → 等泵退出 → 关 frames。
// **不关 page、不关 browser**：其它并发推流与同容器的执行不能被牵连。
func (s *Screencast) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		s.closed.Store(true)
		if err := (proto.PageStopScreencast{}).Call(s.page); err != nil && s.logger != nil {
			// page 已关闭时这里会报 "session closed"，属正常回收路径，不升级为错误。
			s.logger.Debug("Page.stopScreencast 返回错误（page 可能已关闭）", logger.Err(err))
		}
		s.cancel()
		// 等泵退出：确保 Stop 返回后不再有 CDP 回调（泄漏测试要求）。
		// 上限取 AckTimeout+200ms，防止 page 假死把 Stop 卡住。
		guarded := time.NewTimer(s.opts.AckTimeout + 200*time.Millisecond)
		select {
		case <-s.pumpDone:
		case <-guarded.C:
			if s.logger != nil {
				s.logger.Warn("screencast 事件泵未在期限退出（page 可能假死）")
			}
		}
		guarded.Stop()
		s.closeFrames()
		close(s.finishedCh)
	})
}

// Done 停止完成的信号 channel（Stop 返回后关闭）。
func (s *Screencast) Done() <-chan struct{} {
	if s == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return s.finishedCh
}

// Header 当前 JPEG 尺寸（针对外部只拿到 Data 时的便利方法）。
func jpegSize(b []byte) (width, height int) {
	if len(b) < 4 || b[0] != 0xFF || b[1] != 0xD8 {
		return 0, 0
	}
	i := 2
	for i+9 <= len(b) {
		if b[i] != 0xFF {
			i++
			continue
		}
		marker := b[i+1]
		// 无长度的独立标记（RSTn / SOI / TEM）。
		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			i += 2
			continue
		}
		if marker == 0xD9 { // EOI
			return 0, 0
		}
		segLen := int(b[i+2])<<8 | int(b[i+3])
		if segLen < 2 || i+2+segLen > len(b) {
			return 0, 0
		}
		// SOF0..SOF15，排除 DHT(0xC4)/JPG(0xC8)/DAC(0xCC)。
		if marker >= 0xC0 && marker <= 0xCF && marker != 0xC4 && marker != 0xC8 && marker != 0xCC {
			height = int(b[i+5])<<8 | int(b[i+6])
			width = int(b[i+7])<<8 | int(b[i+8])
			return width, height
		}
		i += 2 + segLen
	}
	return 0, 0
}

// ErrScreencastStopped 已停止时投递指令的返回错误（调用方据此判定「流已不在」）。
var ErrScreencastStopped = errors.New("screencast 已停止")

// GuessImageSize 暴露 JPEG 尺寸解析给测试与诊断（无副作用）。
func GuessImageSize(data []byte) (width, height int) { return jpegSize(data) }
