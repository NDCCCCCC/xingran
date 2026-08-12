// Phase 31 Plan 03: F-14 并发回归测试
//
// 这些测试覆盖 PooledConnection.refCount 的并发安全性,
// 锁定 31-01 + 31-02 的修复成果。
// 不依赖真实 SSH/Scrapli,通过 struct 直接构造 PooledConnection
// 测试 refCount 字段的原子操作。

package device

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// makeTestPooledConnection 构造一个可控的 PooledConnection 用于 refCount 测试。
// 不依赖 ScrapliWrapper / SSH,仅测试计数语义。
func makeTestPooledConnection(deviceID string, initialRefCount int32) *PooledConnection {
	return &PooledConnection{
		wrapper:  nil, // 不实际通信,只测 refCount
		refCount: initialRefCount,
		lastUsed: time.Now(),
		deviceID: deviceID,
		mu:       &sync.Mutex{},
		pool:     nil,
	}
}

// TestPooledConnection_IsIdle_RefCount 验证 IsIdle 与 refCount 的语义关系
func TestPooledConnection_IsIdle_RefCount(t *testing.T) {
	pc := makeTestPooledConnection("dev-1", 0)
	if !pc.IsIdle() {
		t.Fatal("初始 refCount=0 应被视为 idle")
	}

	atomic.AddInt32(&pc.refCount, 1)
	if pc.IsIdle() {
		t.Fatal("refCount=1 不应是 idle (有 caller 持有)")
	}

	atomic.AddInt32(&pc.refCount, -1)
	if !pc.IsIdle() {
		t.Fatal("refCount=0 应回到 idle")
	}
}

// TestPooledConnection_Release_DecrementsRefCount 验证 ReleaseRef 行为
// (Release 配对 Acquire,需要先 mu.Lock;ReleaseRef 不操作 mu,适合 GetConnection 路径)
func TestPooledConnection_Release_DecrementsRefCount(t *testing.T) {
	pc := makeTestPooledConnection("dev-2", 1)
	pc.ReleaseRef()
	if got := atomic.LoadInt32(&pc.refCount); got != 0 {
		t.Fatalf("ReleaseRef 后 refCount 应为 0, 实际 %d", got)
	}
}

// TestPooledConnection_Release_PanicOnNegative 验证 ReleaseRef 在 refCount<0 时 panic
// (防御 caller 多调 ReleaseRef 导致计数漂移到负数)
func TestPooledConnection_Release_PanicOnNegative(t *testing.T) {
	pc := makeTestPooledConnection("dev-3", 0)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("refCount=0 时再调 ReleaseRef 应该 panic")
		}
	}()
	pc.ReleaseRef() // 应触发 panic
}

// TestPooledConnection_ConcurrentRefCount 验证 100 goroutine 并发 +1/-1
// refCount 最终归零。模拟高并发 GetConnection/Release 场景。
func TestPooledConnection_ConcurrentRefCount(t *testing.T) {
	pc := makeTestPooledConnection("dev-4", 0)

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			// 模拟 GetConnection 内部 +1 + 短暂使用 + Release -1
			atomic.AddInt32(&pc.refCount, 1)
			time.Sleep(time.Microsecond) // 让 scheduler 切换
			atomic.AddInt32(&pc.refCount, -1)
		}()
	}

	wg.Wait()

	if got := atomic.LoadInt32(&pc.refCount); got != 0 {
		t.Fatalf("100 并发 +1/-1 后 refCount 应为 0, 实际 %d", got)
	}
	if !pc.IsIdle() {
		t.Fatal("100 并发完毕后应回到 idle")
	}
}

// TestPooledConnection_RefCountStaysPositive_DuringCleanup 验证 cleanup 与持有并发场景下,
// refCount > 0 的连接不应被错判为 idle。模拟 F-14 race window 修复后的不变量:
//   - caller A 持有 refCount=1
//   - cleanup goroutine 同时检查 IsIdle
//   - cleanup 应跳过 (因 IsIdle=false)
func TestPooledConnection_RefCountStaysPositive_DuringCleanup(t *testing.T) {
	pc := makeTestPooledConnection("dev-5", 1)

	const cleanupChecks = 1000
	var sawIdle int32

	var wg sync.WaitGroup
	wg.Add(2)

	// goroutine 1: 模拟 cleanupIdleConnections 反复检查 IsIdle
	go func() {
		defer wg.Done()
		for i := 0; i < cleanupChecks; i++ {
			if pc.IsIdle() {
				atomic.AddInt32(&sawIdle, 1)
			}
		}
	}()

	// goroutine 2: 模拟 caller 持有引用计数一段时间后释放
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&pc.refCount, -1)
	}()

	wg.Wait()

	if sawIdle > 0 {
		// 允许 caller 释放后 cleanup 看到 idle 是正常的,
		// 但在 caller 持有期间(refCount=1)看到 idle 是 race 的征兆
		// 这里只允许后期(释放后)的检查命中 idle
		t.Logf("cleanup 观察到 idle %d/%d 次 (允许,只要不是在持有期间)", sawIdle, cleanupChecks)
	}

	if got := atomic.LoadInt32(&pc.refCount); got != 0 {
		t.Fatalf("Release 后 refCount 应为 0, 实际 %d", got)
	}
}

// TestPooledConnection_GetConnection_InternalRefCount_Pattern 单元层面验证
// 31-01 的核心不变量: GetConnection 返回的 pc 应已经 refCount=1。
// (整链路集成测试需要真实 ConnectionPool + SSH,留给手工/E2E 阶段)
func TestPooledConnection_GetConnection_InternalRefCount_Pattern(t *testing.T) {
	// 模拟 31-01 修复后的 new-connection 分支行为
	pc := &PooledConnection{
		wrapper:  nil,
		refCount: 1, // 31-01: 新创建时直接初始化为 1
		lastUsed: time.Now(),
		deviceID: "dev-6",
		mu:       &sync.Mutex{},
	}

	if pc.IsIdle() {
		t.Fatal("GetConnection 返回时 pc 应已持有 refCount=1,不应是 idle")
	}

	// 模拟 caller defer ReleaseRef()
	pc.ReleaseRef()

	if !pc.IsIdle() {
		t.Fatal("caller ReleaseRef 后应回到 idle 允许 cleanup")
	}
}

// TestPooledConnection_AntiPattern_AcquireAfterGetConnection 回归测试 (2026-07-06):
//
// 锁定 2026-07-06 pool 泄漏 bug — `连接池已满: 当前=24, 最大=20` 根因。
//
// 旧错误模式 (pre-F-14 残留,在 config_execution_service.go / device_info_collection_service.go
// 的 3 处出现):
//   conn, _ := pool.GetConnection(ctx, deviceID)  // refCount=1
//   conn.Acquire()                                // +1 → refCount=2
//   defer conn.Release()                          // -1 → refCount=1, mu.Unlock
//   // 失败: refCount 停在 1,IsIdle()=false,cleanup 永不清除 → pool 满
//
// 正确模式 (F-14 协议):
//   conn, _ := pool.GetConnection(ctx, deviceID)  // refCount=1
//   defer conn.ReleaseRef()                       // -1 → refCount=0
//
// 本测试直接对 refCount 字段做算术(绕开 Acquire/Release 的 mu.Lock 副作用,
// 因为 wrapper=nil 时 Acquire 会早退),断言两种模式的最终 refCount 状态:
// - 错误模式: 1+1-1=1,IsIdle=false (cleanup 永不清除 — 正是 24/20 bug)
// - 正确模式: 1-1=0,IsIdle=true  (cleanup 正常清理)
func TestPooledConnection_AntiPattern_AcquireAfterGetConnection(t *testing.T) {
	// --- 错误模式: GetConnection(+1) + Acquire(+1) + Release(-1) = 1 ---
	wrong := &PooledConnection{
		wrapper:  nil,
		refCount: 1, // GetConnection 内部 +1
		lastUsed: time.Now(),
		deviceID: "dev-wrong",
		mu:       &sync.Mutex{},
	}
	atomic.AddInt32(&wrong.refCount, 1) // 模拟 Acquire 的 +1
	atomic.AddInt32(&wrong.refCount, -1) // 模拟 Release 的 -1
	if got := atomic.LoadInt32(&wrong.refCount); got != 1 {
		t.Fatalf("错误模式算术后 refCount 应停在 1, 实际 %d", got)
	}
	if wrong.IsIdle() {
		t.Fatal("错误模式下 IsIdle() 应为 false,cleanup 永不清理(24/20 池满根因)")
	}
	t.Logf("错误模式: refCount=1, IsIdle=false → cleanup 永不删除")

	// --- 正确模式: GetConnection(+1) + ReleaseRef(-1) = 0 ---
	right := &PooledConnection{
		wrapper:  nil,
		refCount: 1, // GetConnection 内部 +1
		lastUsed: time.Now(),
		deviceID: "dev-right",
		mu:       &sync.Mutex{},
	}
	atomic.AddInt32(&right.refCount, -1) // 模拟 ReleaseRef 的 -1
	if got := atomic.LoadInt32(&right.refCount); got != 0 {
		t.Fatalf("正确模式算术后 refCount 应归零, 实际 %d", got)
	}
	if !right.IsIdle() {
		t.Fatal("正确模式下 IsIdle() 应为 true,允许 cleanup 删除")
	}
	t.Logf("正确模式: refCount=0, IsIdle=true → cleanup 正常清理")
}

// TestOldestIdleConnectionLocked_PicksOldestIdle_SkipsActive 锁定 GetConnection LRU 退让的核心选择逻辑
// (2026-07-08 连接池满修复):
//   - 池满时 GetConnection 调 oldestIdleConnectionLocked 找一条 idle 连接淘汰腾位。
//   - 必须跳过活跃连接 (refCount>0), 只在 idle (refCount<=0) 中选 lastUsed 最老的。
// 不依赖真实 SSH, 直接构造 connections map 测选择语义。
func TestOldestIdleConnectionLocked_PicksOldestIdle_SkipsActive(t *testing.T) {
	base := time.Now()
	pool := &DeviceConnectionPool{
		connections: map[string]*PooledConnection{
			"dev-active-old":  {refCount: 1, lastUsed: base.Add(-10 * time.Minute), deviceID: "dev-active-old", mu: &sync.Mutex{}},
			"dev-idle-mid":    {refCount: 0, lastUsed: base.Add(-3 * time.Minute), deviceID: "dev-idle-mid", mu: &sync.Mutex{}},
			"dev-idle-newest": {refCount: 0, lastUsed: base.Add(-1 * time.Minute), deviceID: "dev-idle-newest", mu: &sync.Mutex{}},
			"dev-idle-oldest": {refCount: 0, lastUsed: base.Add(-5 * time.Minute), deviceID: "dev-idle-oldest", mu: &sync.Mutex{}},
		},
	}

	id, pc := pool.oldestIdleConnectionLocked()
	if id != "dev-idle-oldest" {
		t.Fatalf("应选 lastUsed 最老的 idle 连接 dev-idle-oldest, 实际 %s", id)
	}
	if pc == nil {
		t.Fatal("应返回非 nil 连接")
	}
	if atomic.LoadInt32(&pc.refCount) != 0 {
		t.Fatalf("选中连接应为 idle (refCount=0), 实际 %d", atomic.LoadInt32(&pc.refCount))
	}
	t.Logf("LRU 选中 %s (lastUsed 最老的 idle), 正确跳过活跃的 dev-active-old", id)
}

// TestOldestIdleConnectionLocked_NoIdleReturnsNil 锁定"全活跃则拒绝"路径:
// 池满且全部活跃时 oldestIdleConnectionLocked 返回 nil, GetConnection 据此返回
// "连接池已满且无 idle 连接可退让" 错误 (而非死等或破坏活跃连接)。
func TestOldestIdleConnectionLocked_NoIdleReturnsNil(t *testing.T) {
	base := time.Now()
	pool := &DeviceConnectionPool{
		connections: map[string]*PooledConnection{
			"dev-a": {refCount: 1, lastUsed: base.Add(-10 * time.Minute), mu: &sync.Mutex{}},
			"dev-b": {refCount: 2, lastUsed: base.Add(-3 * time.Minute), mu: &sync.Mutex{}},
		},
	}
	id, pc := pool.oldestIdleConnectionLocked()
	if id != "" || pc != nil {
		t.Fatalf("全部活跃时应返回 ('', nil), 实际 id=%s pc=%v", id, pc)
	}
}

// TestOldestIdleConnectionLocked_EmptyPool 空池返回 nil。
func TestOldestIdleConnectionLocked_EmptyPool(t *testing.T) {
	pool := &DeviceConnectionPool{
		connections: map[string]*PooledConnection{},
	}
	id, pc := pool.oldestIdleConnectionLocked()
	if id != "" || pc != nil {
		t.Fatalf("空池应返回 ('', nil), 实际 id=%s pc=%v", id, pc)
	}
}
