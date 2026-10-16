---
slug: scrapligo-getprompt-nil-panic
status: awaiting_human_verify
created: 2026-10-16
updated: 2026-10-16
trigger: scrapligo transport.(*Standard).Write nil pointer panic in GetPrompt goroutine crashed the whole backend process (exit status 2)
---

# Debug Session: scrapligo-getprompt-nil-panic

## Symptoms

- expected_behavior: 后端进程不应 panic；设备连接操作失败应返回错误而非崩溃整个进程
- actual_behavior: 整个 Go 后端进程 panic 退出 (exit status 2)。panic 来自 scrapligo GetPrompt 内部 spawn 的 goroutine，发生在 2026-10-16 00:01:58，崩溃前最后一条业务日志是无关的 RPA worker heartbeat (POST /api/v1/rpa/workers/worker-1/heartbeat, 200)
- error_message: |
    panic: runtime error: invalid memory address or nil pointer dereference
    [signal 0xc0000005 code=0x0 addr=0x20 pc=0xe8e19b]

    goroutine 145588 [running]:
    github.com/scrapli/scrapligo/transport.(*Standard).Write(0x21389d0?, {0xc002af1c10?, 0x37087c0?, 0xc0016ebf08?})
            C:/Users/CPIC/go/pkg/mod/github.com/scrapli/scrapligo@v1.4.0/transport/standard.go:269 +0x1b
    github.com/scrapli/scrapligo/transport.(*Transport).Write(...)
            C:/Users/CPIC/go/pkg/mod/github.com/scrapli/scrapligo@v1.4.0/transport/transport.go:212
    github.com/scrapli/scrapligo/channel.(*Channel).Write(0xc0031872c0, {0xc002af1c10, 0x1, 0x1}, 0x0)
            C:/Users/CPIC/go/pkg/mod/github.com/scrapli/scrapligo@v1.4.0/channel/write.go:12 +0xb9
    github.com/scrapli/scrapligo/channel.(*Channel).WriteReturn(...)
            C:/Users/CPIC/go/pkg/mod/github.com/scrapli/scrapligo@v1.4.0/channel/write.go:17
    github.com/scrapli/scrapligo/channel.(*Channel).GetPrompt.func1()
            C:/Users/CPIC/go/pkg/mod/github.com/scrapli/scrapligo@v1.4.0/channel/getprompt.go:25 +0x8d
    created by github.com/scrapli/scrapligo/channel.(*Channel).GetPrompt in goroutine 145576
            C:/Users/CPIC/go/pkg/mod/github.com/scrapli/scrapligo@v1.4.0/channel/getprompt.go:22 +0x112
- timeline: 首次出现（2026-10-16 00:01:58，本地 go run 环境）
- reproduction: 不确定/后台触发，无手动复现路径；仅此一次

## Key Context

- 项目技术栈记录 scrapligo v1.3.3，stack trace 显示 **v1.4.0** —— 库近期升级过（待验证 go.mod / git log）
- 项目有自研 SSH 连接池（按 deviceID 缓存 + LRU 淘汰 idle + connecting 占位），见 internal/device/ 相关代码与记忆 device-connection-pool-lru-eviction
- panic 点 addr=0x20 强烈暗示 transport 的底层 conn 为 nil（已 Close）后仍被 Write
- GetPrompt 会 spawn goroutine 发送回车探测 prompt；若同一连接此刻被池淘汰/关闭 → race → nil deref
- goroutine 145576 调用 GetPrompt（未在 stack 中，说明是普通调用），145588 是其 spawn 的写入 goroutine
- 崩溃时无用户触发的设备操作 → panic 可能来自池内 idle 连接的健康检查/复用路径，或残留采集 goroutine

## Evidence

- timestamp: 2026-10-16T00:00 (investigation start)
  checked: go.mod + stack trace
  found: go.mod has scrapligo v1.4.0 (project docs stale at v1.3.3); prior commit b91815cb fixed a scrapligo Queue.Dequeue race panic before (defense: getPromptMu in wrapper)
  implication: 库已升级，v1.4.0 新行为可能是诱因；历史上已有一次 scrapligo 竞态 panic
- timestamp: 2026-10-16
  checked: scrapligo v1.4.0 transport/standard.go, transport/transport.go, channel/{channel,write,getprompt}.go, driver/{generic,network}/driver.go
  found: (1) Standard.Write(269)=`t.writer.Write(b)`；writer 仅在 openSession→StdinPipe 赋值一次；Close() 只 nil session/client 不 nil writer → 正常 close 后 Write 返回 error 而非 nil panic。(2) Transport.Write 不取 implLock（Read/Close 取）。(3) Channel.GetPrompt spawn goroutine 执行 WriteReturn+ReadUntilPrompt，unbuffered chan 握手保证父返回前子完成 write。(4) panic 在 spawn 的 goroutine 里，调用方任何 recover() 都无法捕获 → 进程必死（解释了 IsReady/OpenContext/pc.Execute 三处 recover 全部失效）
  implication: nil writer 需要解释——要么 openSession 从未完成（writer 保持构造时 nil）时有人 Write，要么并发 Open 撕裂 interface 两字
- timestamp: 2026-10-16
  checked: internal/device/scrapli_wrapper.go 全文 + connection_pool.go 全文
  found: (1) 所有 wrapper 操作（SendCommand/GetPrompt/IsReady/SendConfig）经 acquireOp: state==Ready + opMu.RLock；Close 经 opMu.Lock + 100ms sleep → Ready 态操作与 Close 完全互斥，无竞态。(2) OpenContext 的初始化轮询循环（每 100ms GetPrompt）**不持 opMu**（注释自认 line 375），且 OpenContext 自身无状态检查可重复调用。(3) OpenContext ctx.Done 路径 spawn fire-and-forget goroutine 直接 driver.Close()（line 412-418），可与仍在执行的 driver.Open()/GetPrompt 轮询并发。(4) pool 每个 wrapper 只 OpenContext 一次，deviceLock 按设备串行，无重复 Open
  implication: 唯一无保护的并发窗口 = OpenContext 初始化期（Initializing 态）的 GetPrompt vs driver.Close / driver.Open 的交错
- timestamp: 2026-10-16
  checked: 全 codebase scrapligo import 扫描
  found: 生产代码只有 internal/device/scrapli_wrapper.go 一处封装（其余全是 _test.go），全部流量经连接池
  implication: 排除"第二个绕过池的 scrapligo 使用点"假设
- timestamp: 2026-10-16
  checked: scrapligo 上游 GitHub（API）：tags / issues / compare v1.4.0...v1.4.2
  found: v1.4.1/v1.4.2 修复 Channel.Close/read 竞态（#231/#232，send-on-closed-channel panic）+ system transport closed-channel read panic；未修 Write-vs-Close 同步（我们的 panic 签名上游无 fix）；v2 是 purego FFI 重写
  implication: v1 线确认有竞态史（旁证）；bump v1.4.2 可消除 opMu 防不住的库内 read-loop-vs-Close panic 残余暴露
- timestamp: 2026-10-16
  checked: 修复后验证（build/vet/test）
  found: go build ./... 干净；device 全套件 4 次 + 重点 5 次全绿；portwrite(e2e)/portcollection/services(455s) 全绿；-race 本机不可用（gcc 16 触发 runtime/cgo -Werror，最小 cgo 包复现，环境问题）
  implication: 修复无回归；竞态检测验证留给 CI

## Eliminated

- hypothesis: 池淘汰/清理 Close 与 Ready 态操作竞态（用户初始假设：池内 idle 连接被淘汰时 GetPrompt 在飞）
  evidence: wrapper 所有操作经 acquireOp（state==Ready + opMu.RLock），Close 经 opMu.Lock 互斥等待；且 Standard.Close 不 nil writer，write-after-close 只返回 error。refCount 挡住 cleanup/LRU/removeConnection 对活跃连接的关闭
  timestamp: 2026-10-16
- hypothesis: 存在绕过连接池的第二个 scrapligo 使用点（自建 driver/Open/Close）
  evidence: 全库 grep scrapligo import，生产代码仅 scrapli_wrapper.go 一处
  timestamp: 2026-10-16

## Current Focus

test: 等待用户在生产/本地工作流中确认不再出现进程级 panic（原始症状：exit status 2 崩溃）
expecting: 后端持续运行，设备连接操作失败时返回错误而非进程退出
next_action: awaiting human verify — 用户确认后 archive + knowledge base

## Resolution

root_cause: OpenContext 初始化窗口无互斥（scrapli_wrapper.go）。init goroutine（driver.Open() + GetPrompt 探测循环，每 100ms 一次，全程不持 opMu）与 ctx.Done 路径 spawn 的 fire-and-forget goroutine（直接 w.driver.Close()，不持 opMu、不关 closing、不查状态）可并发交错：teardown 撞上 Open/Write。初始化期 wrapper 尚未入池，这个 ctx.Done closer 是该窗口唯一的 closer，故竞态 100% 局部于此。scrapligo v1.4.0 对 Write 与 Open/Close 无任何内部同步（Transport.Write 不取 implLock；Channel.Close 的 62.5ms 超时走 force t.Close(true) 连 implLock 都绕过），交错时 interface dispatch 撞上 nil/撕裂状态（panic pc=standard.go:269 +0x1b, addr=0x20）。panic 发生在 GetPrompt spawn 的 goroutine 内——Go 中任何 goroutine 的未恢复 panic 都杀死整个进程，故 wrapper 里三处 recover() 全部无效。次要放大器：并发双 Close 会二次 driver.Close() → Channel.Close 内 close(c.Errs) 二次关闭 panic（上游 #231/#232 证实并已在 v1.4.2 修复同区域另一竞态）
fix: |
  internal/device/scrapli_wrapper.go 四处修改：
  1. OpenContext init goroutine 全程持有 opMu.RLock（driver.Open + GetPrompt 探测），拿锁后复查 isClosing（Close 先于 Open 完成的窗口不再裸开驱动）
  2. ctx.Done 路径从 fire-and-forget w.driver.Close() 改为走 w.Close()（setState Closing → 关 closing → opMu.Lock 排队等 init 完全退出后才 driver.Close）——从结构上消灭 teardown-vs-Open/Write 并发
  3. Close() 入口加 closeStarted atomic.Bool CAS 幂等保护（entry 状态检查有 TOCTOU；双 Close 会二次 driver.Close() → close(Errs) panic，与上游 v1.4.2 idempotent Close 同款修复）
  4. Close() 在 opMu.Lock 之前 closeOnce(close(initDone))，WaitForReady 等待者立即放行
  另：scrapligo v1.4.0 → v1.4.2（上游对同区域 Channel.Close/read 竞态的修复；opMu 防不了 scrapligo 自身 read-loop goroutine 在设备自然断连+并发 Close 时的 send-on-closed-channel panic，v1.4.2 修的正是这个）
  新增 internal/device/scrapli_wrapper_initrace_test.go：3 个回归测试（cancel+Close 15 轮 hammer / 挂死探测下 cancel 的优雅降级+WaitForReady 立即放行 / Close-先于-Open 不裸开驱动）
verification: |
  - go build ./... 通过（v1.4.2）
  - go vet ./internal/device/ ./internal/collectors/ ./internal/services/portwrite/ 干净
  - go test ./internal/device/ 全绿（完整套件 4 次 + 重点套件 5 次，一次疑似机器负载 flake 未复现）
  - go test ./internal/services/portwrite/（scrapligo e2e 重度）./internal/services/portcollection/ ./internal/services/（455s 全套）全绿
  - 局限：-race 在本机不可用（MSYS2 gcc 16.1.0 触发 runtime/cgo -Werror 编译失败，最小 cgo 包复现，环境问题非代码问题）——竞态检测验证留给 CI；stash-RED 对比实验不可行（该文件有大量未提交 WIP，stash 会回退全部）
files_changed:
  - internal/device/scrapli_wrapper.go
  - internal/device/scrapli_wrapper_initrace_test.go (new)
  - go.mod
  - go.sum
