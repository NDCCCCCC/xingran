---
type: quick
slug: fix-async-retry-worker-flaky-test
created: 2026-08-20
completed: 2026-08-20
status: complete
description: 修复 pkg/cache TestAsyncRetryWorker_Enqueue 时序竞态 flake(测试断言 bug,非生产代码 bug)
duration: ~12m (debug 5m + 修 1m + 验证 6m)
---

# Summary: 修复 TestAsyncRetryWorker_Enqueue flaky test

## TL;DR

| 指标 | 修复前 | 修复后 |
|------|--------|--------|
| `pkg/cache` 全包测试 10 次跑 | **9 pass / 1 fail** | — |
| `pkg/cache` 全包测试 **15** 次跑 | — | **15 / 15 pass** ✅ |
| CI 同款命令 (`./internal/... ./pkg/... ./cmd/...`) | 失败 1 包 | **全过,exit 0** |
| 改动文件 | — | `pkg/cache/retry_test.go`(only test file) |
| 改动行数 | — | 增 ~30 / 改 ~3 |
| 生产代码 (`retry.go`) | — | **未改动**(无 bug) |

## 根因

**测试断言设计错误(时序竞态),生产代码无 bug。**

```go
// retry_test.go 修复前的 194-209 行
success := worker.Enqueue(ctx, memoryCache, "test_key", "test_value", time.Minute)
assert.True(t, success)

queueSize := worker.QueueSize()       // ① 入队后立即读:0 或 1,取决于 worker 是否已取走
assert.True(t, queueSize >= 0)

time.Sleep(200 * time.Millisecond)   // ② worker 在此期间消费队列

stats := worker.GetStats()
assert.Equal(t, 2, stats["worker_count"])
assert.Equal(t, queueSize, stats["queue_size"])  // ③ 断言"不变"——必然竞态 ❌
```

`worker.QueueSize()` 与 `stats["queue_size"]` 在 sleep 200ms 期间**没有不变量**——worker goroutine 在并发消费队列,实时变化。

### 失败 Case 重现

实测 10 次跑 pkg/cache 全包:

| 跑次 | Enqueue→QueueSize 间 worker 是否取走 | `queueSize` | sleep 后 `stats["queue_size"]` | 断言结果 |
|------|--------------------------------------|-------------|--------------------------------|----------|
| 1 | 否 | 1 | 0(已消费) | ❌ FAIL |
| 2-10 | 是 | 0 | 0(已消费) | ✅ PASS |

**失败率 ~10%**,取决于 worker goroutine 在 `Enqueue` 与 `QueueSize()` 两次调用之间是否被 Go runtime 调度到取走任务。

### 失败机制细节

`AsyncRetryWorker.worker()` (`retry.go:335-346`):
```go
for {
    select {
    case <-w.closeChan:
        return
    case work := <-w.workQueue:
        w.processWork(id, work)
    }
}
```

`workQueue` 是 buffered channel cap=1000。`Enqueue` 是 non-blocking send:
```go
select {
case w.workQueue <- retryWork{...}:
    return true
case <-ctx.Done():
    return false
}
```

只要 channel 没满就立即返回 true(不等消费者)。`QueueSize()` 返回 `len(w.workQueue)`——这一瞬间的瞬时值,无法预测。**测试断言"两次瞬时值相等"是错误的不变量**。

## 修复

### 改动 1:`TestAsyncRetryWorker_Enqueue` 删掉竞态断言

```diff
-    assert.Equal(t, queueSize, stats["queue_size"])
+    statsQueueSize, ok := stats["queue_size"].(int)
+    assert.True(t, ok, "queue_size 应为 int 类型")
+    assert.GreaterOrEqual(t, statsQueueSize, 0)
```

**理由**:worker 并发消费下,`stats["queue_size"]` 随时变化(0~任何在处理的值)。非负断言是唯一可靠不变量。`worker_count=2` 仍是确定值(配置锁定),保留。

### 改动 2:新增 `TestAsyncRetryWorker_QueueSemantics` 确定性测试

```go
// 不启动 worker 消费者,精确验证入队 → 队列大小 +1 语义
func TestAsyncRetryWorker_QueueSemantics(t *testing.T) {
    config := DefaultRetryConfig()
    worker := NewAsyncRetryWorker(config, 2)
    // 注意: 不调用 Start()——无消费者,队列大小完全由入队驱动
    defer worker.Stop()  // wg=0,close channel 立即 wg.Wait,无副作用

    before := worker.QueueSize()
    assert.Equal(t, 0, before)

    ok := worker.Enqueue(ctx, memoryCache, "k1", "v1", time.Minute)
    assert.True(t, ok)
    assert.Equal(t, before+1, worker.QueueSize())

    ok = worker.Enqueue(ctx, memoryCache, "k2", "v2", time.Minute)
    assert.True(t, ok)
    assert.Equal(t, before+2, worker.QueueSize())

    stats := worker.GetStats()
    statsQueueSize := stats["queue_size"].(int)
    assert.Equal(t, worker.QueueSize(), statsQueueSize)
    assert.Equal(t, 2, stats["worker_count"])
}
```

**理由**:删掉的断言原本想覆盖"队列大小语义"——Enqueue 后队列 +1。新测试用"不启动 worker"的技巧,把"队列大小 = 入队计数"这条不变式测成**确定性的**,零竞态。`Stop()` 在 `Start()` 未调用时是安全的(wg=0 立即返回)。

## 验证

### 1. `go build ./...` 全过

无编译错误。

### 2. `go vet ./pkg/cache/...` 全过

无 lint 错误。

### 3. `go test -count=1 ./pkg/cache/...` 连续 15 次

```
ok  github.com/.../pkg/cache  6.852s
ok  github.com/.../pkg/cache  7.167s
... (15 次全 ok,无 FAIL)
```

修复前同命令 10 次中失败 1 次。

### 4. CI 同款命令(全 module)

```bash
go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...
```

exit code 0,所有 38 个包 `ok`(完整列表见任务 `b92qw2497.output`),无回归。

### 5. 不跑 `-race`(Windows 无 gcc,项目 CI 也不跑 race)

`ci.yml` 的 Test 步骤无 `-race`,本修复不变此行为。

## 影响范围

- **生产代码**:`pkg/cache/retry.go` **未改动**(本就是测试断言 bug,非生产 bug)
- **测试代码**:`pkg/cache/retry_test.go` 改动 ~30 行(新增 1 测试 + 删 1 竞态断言)
- **下游**:AsyncRetryWorker 是 cache retry 机制的核心组件,任何下游用户(L2WriteWorker 等)行为不变
- **CI gate**:无变化(`go test ./...` 同样通过)

## 为什么不是生产 bug

考虑过可能性:**AsyncRetryWorker 真的没消费队列任务?**

否定证据:
- 单跑 `TestAsyncRetryWorker_Enqueue` 5/5 pass
- 改动后 15/15 pass(配合新增 QueueSemantics 测试,消费者确实在工作)
- 重新跑 go test ./... 完整 module(后台跑 10 分钟)所有包 ok,exit 0
- 生产代码逻辑清晰:`worker()` goroutine 是标准的 select-on-channel 模式,只要 Start 被调用就一定工作
- 唯一旁证:`processWork` 调用 `RetryWithContext` 后才更新 stats。如果 worker 真的不消费,`total_attempts` 永远是 0;但实际跑出非零值

**结论**:worker 工作正常,`stats["queue_size"]` 反映真实的瞬时队列长度,测试只是错在断言它不变。

## 给 v1.26 milestone 的输入

这是个**测试设计反例**——`async/concurrent` 测试里"瞬时值不变"断言几乎必 flake。可以在 v1.26 加测试代码 review checklist:

```
□ async/concurrent 测试是否有 sleep + 瞬时值断言?(几乎必 flake)
□ 多 goroutine 测试是否用 channel close + WaitGroup 同步而非 time.Sleep?(可选)
□ 状态断言是否区分"worker 启动 vs 不启动"两种语义?(QueueSemantics 范本)
```

类似反模式可能存在于其他测试(未审计),v1.26 扫覆盖率时一并检查。

## 产出文件

- `pkg/cache/retry_test.go` — 改动 + 新增测试
- `PLAN.md` — 修复方案
- `SUMMARY.md` — 本文件

## 给用户的下一步建议

按 CLAUDE.md 的 git workflow:"Before making git commits, ask for explicit user confirmation."

```
建议 commit message:
  test(cache): fix flaky TestAsyncRetryWorker_Enqueue timing race

  - Remove assertion that queue_size remains unchanged across 200ms
    sleep (worker concurrently consumes, was failing ~10% under load)
  - Add TestAsyncRetryWorker_QueueSemantics: deterministic test
    without consumer goroutines, verifies Enqueue → queue size +1
    invariant exactly

  Production code (retry.go) unchanged — no bug there.
```