---
slug: scrapligo-queue-panic-windows
status: resolved
trigger: "scrapligo util.(*Queue).Dequeue panic: index out of range [0] with length 0 in Windows local dev, not on Linux server. Connection pool reuses connection 10.62.25.252:22 successfully but panics in GetPrompt on subsequent calls. Memory usage high concurrent with panic."
created: "2026-06-15"
updated: "2026-06-15"
resolved: "2026-06-15"
---

## Symptoms

- **Expected**: Device connection pool returns usable wrapper; subsequent SendCommand/GetPrompt operations succeed
- **Actual**: `panic: runtime error: index out of range [0] with length 0` at `scrapligo/util/queue.go:64`
- **Connection Log**: `INFO[2026-06-15 22:00:01] [连接池] 设备连接成功: CX-WH-RUITONG-25F-SWL2-HW-S5735-1 (10.62.25.252:22)` then immediate panic
- **Stack trace top**:
  ```
  github.com/scrapli/scrapligo/util.(*Queue).Dequeue
  github.com/scrapli/scrapligo/channel.(*Channel).Read      (read.go:129)
  github.com/scrapli/scrapligo/channel.(*Channel).ReadUntilPrompt (read.go:234)
  github.com/scrapli/scrapligo/channel.(*Channel).GetPrompt.func1  (getprompt.go:34)
  ```
- **Environment diff**: Windows 11 local dev crashes; Linux server deployment runs without panic
- **User hypothesis tested**: High memory usage suspected as overflow cause — **REJECTED** (see Analysis)
- **Python env check**: User reports "未安装 scrapli / 不确定" — **NOT RELEVANT** (Go binary uses scrapligo, not Python)

## Current Focus

- hypothesis: scrapligo Queue.Dequeue race + GetPrompt's inner goroutine panic is not caught by outer recover(); Windows goroutine scheduling exposes the race more often
- next_action: Patch internal/device/scrapli_wrapper.go GetPrompt call sites to bypass scrapligo's internal goroutine, or use a wrapper that runs scrapligo in an isolated recover boundary
- test: null
- expecting: null
- reasoning_checkpoint: null
- tdd_checkpoint: null

## Evidence

- 2026-06-15 22:00:01: `internal/device/scrapli_wrapper.go:355` `w.driver.GetPrompt()` is called from OpenContext polling loop; outer `recover()` at line 351 CANNOT catch panics in the inner goroutine spawned at `getprompt.go:22-40`
- 2026-06-15 22:00:01: `internal/device/connection_pool.go:118-119` has recover() for Execute path, but this is a different code path
- 2026-06-15 22:00:01: scrapligo `util/queue.go:53-73` Dequeue has TOCTOU race: `getDepth()` checks via `depthChan` (independent of `q.queue`); between check and `q.queue[0]` access, another Dequeue consumer can drain
- 2026-06-15 22:00:01: scrapligo `channel/getprompt.go:22-40` spawns `go func()` that calls `ReadUntilPrompt → Read → Queue.Dequeue`; panic in this goroutine crashes the entire process (Go semantics)
- 2026-06-15 22:00:01: `scrapligo-snmp-panic.md` (2026-04-20) showed similar root cause; fix was "panic recovery wrapper, RWMutex concurrency control, connection readiness validation" — but ONLY covered SNMP path, NOT the GetPrompt path

## Eliminated

- 2026-06-15 22:00:01: hypothesis: Python scrapli_cli not installed — ELIMINATED. Stack trace shows Go scrapligo, not Python. Backend uses scrapligo (Go library) directly.
- 2026-06-15 22:00:01: hypothesis: local network cannot reach devices — ELIMINATED. Log shows `设备连接成功` (TCP connection succeeded)
- 2026-06-15 22:00:01: hypothesis: memory overflow causes panic — ELIMINATED. Panic is in Queue slice access, not heap exhaustion. Memory growth is a SECONDARY symptom (connection leak on crash), not the cause.

## Analysis

### Root cause chain

1. **scrapligo Queue race (library bug)**:
   - `util/queue.go:53-73` `Dequeue` does depth check via `depthChan`, then locks, then `q.queue[0]`
   - Two concurrent Dequeue consumers can race: A passes depth check, B drains, A panics
   - Trigger: scrapligo spawns multiple `Read()` consumers when `GetPrompt` is called concurrently (e.g., once from OpenContext polling, once from a real operation before the previous ticker iteration has finished)

2. **GetPrompt's inner goroutine panics** (trigger #1):
   - `channel/getprompt.go:22` `go func()` is the only Dequeue consumer for a single GetPrompt call
   - If `GetPrompt` is called twice rapidly (OpenContext ticker at 100ms + real connection check), the inner goroutines can race
   - The Queue race fires, panic happens inside the spawned goroutine

3. **Panic in goroutine crashes the process** (amplifier):
   - Go's panic semantics: a panic in any goroutine, if uncaught, kills the entire program
   - The `recover()` at `scrapli_wrapper.go:351` is in the CALLING goroutine, not the panic's goroutine — `recover()` only works in the panicking goroutine's deferred function
   - Result: `exit status 2` terminates the backend

4. **Memory growth is a leak, not overflow** (consequence):
   - Each crash terminates the process before `Close()` runs on the connection pool
   - On restart, new connections are made
   - If scheduler retries with the same broken logic, repeated crashes leak FD and memory
   - This looks like "memory high" but is actually connection + goroutine churn

### Windows vs Linux diff

| Dimension | Windows local | Linux server |
|---|---|---|
| Goroutine scheduler | Preemptive with shorter time slices; `GOMAXPROCS` defaults to NumCPU | Same code, but server has higher idle CPU headroom |
| TCP socket timing | Winsock returns faster on close; initial SSH banner arrives in 1-2 read() calls | Linux TCP stack may coalesce more |
| Memory pressure | Dev machine runs IDE + browser → GC frequent | Server runs single backend → GC less frequent |
| Race window | GC pauses during `getDepth()` check leave the `q.queue` exposed | Lower GC pressure → race rarely fires |

## Resolution

- root_cause: scrapligo v1.3.3 `util/queue.go` Dequeue 的 TOCTOU 竞态 —— `getDepth()` 通过 `depthChan` 检查后、加锁访问 `q.queue[0]` 前，另一个并发的 Dequeue 消费者可能已清空队列，导致 `index out of range [0] with length 0`。该 panic 发生在 `channel.GetPrompt` 启动的内部 goroutine 中，Go 语义下未捕获的 goroutine panic 会杀死整个进程（`exit status 2`），调用方的 `recover()` 无法跨 goroutine 拦截。Windows 本地因 GC 频繁（IDE/浏览器占用）+ goroutine 调度抖动，命中竞态窗口的概率显著高于空载的 Linux 服务器。内存高是连接/goroutine 在崩溃后未正常关闭导致的泄漏堆积（次生症状），而非 panic 的原因。
- fix: 三层防御纵深组合
  1. **根因修复（A）**：scrapligo 升级 v1.3.3 → v1.4.0。v1.4.0 在 `util/queue.go:64` 增加了双重检查 `if len(q.queue) == 0 { return nil }`，注释明确 "Double-check after acquiring lock to prevent race condition"，直接消除 Dequeue 的 TOCTOU 窗口。
  2. **参数调优（B）**：在 `NewScrapliWrapper` / `NewScrapliWrapperWithPort` 中追加 `options.WithReadDelay(50*time.Millisecond)` 和 `options.WithTimeoutOps(60*time.Second)`，放慢读取节奏降低竞态命中率，并兼容 Windows 本地与慢设备。
  3. **业务层互斥（D）**：新增 `ScrapliWrapper.getPromptMu sync.Mutex` 字段及 `GetPrompt() (string, error)` 封装方法，串行化所有 GetPrompt 调用（OpenContext 就绪轮询、IsReady、GetResponse 三处统一改调封装）。注意 v1.4.0 中 `driver.GetPrompt` 返回类型从 `[]byte` 变为 `string`。
- verification: `go build ./...` 通过；`go vet ./internal/device/...` 干净；`go test ./internal/device/...` 全部通过（1.511s）；`go mod tidy` 清理 go.sum 仅保留 v1.4.0。本地实机长跑验证待用户执行（连真实网络设备观察 1-2 小时确认无 panic）。
- files_changed:
  - go.mod (scrapligo v1.3.3 → v1.4.0)
  - go.sum
  - internal/device/scrapli_wrapper.go (struct 新增 getPromptMu；新增 GetPrompt 封装方法；两处构造器追加 ReadDelay/TimeoutOps；三处调用点改用封装)

## Memory Hypothesis Verdict

用户怀疑"内存溢出导致 panic" —— **驳回**。
- panic 栈顶在 `util/queue.go:64` 切片越界访问，非堆耗尽
- Go OOM 表现为 `runtime: out of memory`，不会以 `index out of range` 形式出现
- 内存持续增长是 panic 杀进程后连接未关闭、调度器重试反复建连导致的泄漏（次生结果），不是 panic 的因
- 但内存压力会通过频繁 GC 加剧 goroutine 调度抖动，间接提高竞态命中率 —— 属放大器，非根因

## Candidate Fixes (to be evaluated)

### Fix A: Bypass scrapligo GetPrompt for readiness check
- Replace `OpenContext` polling `GetPrompt` call with a Channel state probe (e.g., check `Channel.readLoopExited` via reflection, or just wait for `initDone` channel from scrapligo's own initialization)
- Pros: avoids the race-prone call site
- Cons: requires knowing scrapligo's internal init signals

### Fix B: Wrap scrapligo calls in subprocess
- Run scrapligo driver in a separate OS process via `go scrapligo run` style wrapper
- On crash, subprocess exits, parent restarts it
- Pros: complete isolation
- Cons: high refactor cost

### Fix C: Add defer/recover to scrapligo source via `replace` directive
- In go.mod, `replace github.com/scrapli/scrapligo => ./local-scrapligo-fork`
- Add `defer func() { recover() }()` to `getprompt.go:22` inner goroutine
- Pros: minimal code change
- Cons: maintain fork; community version may fix it later

### Fix D: Sequential GetPrompt only
- Use `sync.Mutex` to serialize all GetPrompt calls per device
- Pros: prevents concurrent Dequeue race
- Cons: still leaks panics from single-threaded cases

### Recommended: Fix C + Fix D
- Vendor scrapligo with recover patch
- Add per-device mutex around all scrapligo Channel operations
- Keeps minimal change footprint, prevents both the race and the panic propagation
