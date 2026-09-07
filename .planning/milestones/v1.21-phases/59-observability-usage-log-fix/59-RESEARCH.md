# Phase 59: 可观测性 / 使用日志修复 - Research

**Researched:** 2026-08-13
**Domain:** Go / Gin middleware + 异步 fire-and-forget 日志链路 + GORM 单 INSERT 写入
**Confidence:** HIGH（纯逻辑修复，所有 file:line 经实读核对，关键先例在仓库内）

## Summary

本 phase 是一个**聚焦的逻辑修复**：让 `sys_api_key_usage_log` 真实反映请求结果。根因经 STATE.md §根因调查钉死、CONTEXT.md 已锁定 D-01~D-04 决策，故本研究**不重论证锁定决策**，只确认实现模式、钉死开放细节、产出 Validation Architecture。

实读源码后，CONTEXT.md 所列 file:line **全部准确**（`apikey.go:61-76` 记录点在 `c.Next()` 前、`usage_logger.go:54-75` 复用调用方 ctx + `_ = err` 静默吞错、`apikey_service.go:519` `successCount/total*100`）。`LogUsageRequest` 结构体**已含** `StatusCode/Duration/Success` 字段（`usage_logger.go:20-30`），`models.APIKeyUsageLog` 也已含对应列（`api_key_usage_log.go:8-24`）——**无 schema 变更、无 migration**，确认 CONTEXT.md 判断。

**关键研究发现**（超出 CONTEXT.md 的增量）：
1. **detached-context 的更强先例**：`pkg/cache/redis.go:601-605` 的 L2 异步刷盘与本 phase P2-b **语义近乎相同**（注释原话「不能用请求 ctx 去做异步入队…改用独立 ctx 隔离,真正的客户端取消不应阻塞 L2 异步刷盘」），比 PROJECT.md v1.19 citation 更直接、更贴近——D-02 落地有现成模式可抄。
2. **`c.Next()` 后捕获状态码是仓库既有标准模式**：`pkg/middleware/logger.go:47-49`（`startTime` 前置 + `time.Since` + `c.Writer.Status()`）、`response_encryption.go:101`、`oper_log_service.go:176` 均如此。
3. **异步可测试性推荐 `require.Eventually` 轮询 DB 行**（见 §异步写入可测试性机制）——这是本研究**最需要给出明确推荐**的点。
4. **D-03「sqlite in-memory」措辞与仓库既有实践冲突**：`usage_logger_test.go:18-29` 注释明载曾用 `file::memory:?cache=shared` 因并发写锁撞锁改用**文件 DB（`os.TempDir` + 唯一名 + `busy_timeout=5000`）**；新测试应**沿用文件 DB 模式**，不要退回 in-memory。
5. **`gen_random_uuid()` 是 sqlite 测试陷阱**：模型 `id` 列 `default:gen_random_uuid()`（PG 专有），既有测试用**裸 `CREATE TABLE` DDL**（`id TEXT PRIMARY KEY`）绕过；新测试若误用 `AutoMigrate` 会 INSERT 失败。
6. **记录点后移带来「pre-auth 失败不可日志」的语义边界**（见 §Pattern 1 备注）——planner 须确保 SC#2 失败用例用**下游产生的失败**（RequireScope→403 / handler→429），而非 pre-auth 401（无 resolved key 可记）。

**Primary recommendation:** middleware 改为 `start:=time.Now()` → `c.Next()` → 同步填 `StatusCode=c.Writer.Status()` / `Duration=time.Since(start).Milliseconds()` / `Success = 200<=code<300` → 同步调 `LogUsage(c.Request.Context(), req)`；impl 内部 `logUsageAsync` 改 `context.WithTimeout(context.Background(), 10*time.Second)` 做写入、失败走 `applogger.Errorf`；新增测试用**真实文件 sqlite + `require.Eventually` 轮询**做 SC#1/#3/#4 的 DB 行实证。

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions（锁定，研究仅引用确认，不重论证）

- **D-01（Success 口径）:** `Success = 200 <= statusCode && statusCode < 300`。3xx/4xx/5xx 一律 `false`。直接决定 `apikey_service.go:519` `successRate` 语义。
- **D-02（detach 责任内聚）:** 「请求生命周期免疫」做成 `UsageLogger` **impl 内部**契约——`logUsageAsync`（`usage_logger.go:54`）改用 `context.WithTimeout(context.Background(), ~10s)`，**忽略**调用方 ctx 的取消信号；调用方 ctx 仅用于取请求范围值。
- **D-02a（去冗余双重 goroutine）:** 去掉 middleware 的外层 `go func(){ LogUsage }`（`apikey.go:62`）；修复后 middleware 在 `c.Next()` **之后**同步调 `usageLogger.LogUsage(...)`，由 `LogUsage` 内部 `go logUsageAsync()` 异步执行。
- **D-03（真实 DB 测试）:** 时序/取消测试用 **sqlite** 作真实 DB（SC#1/#4 要求 DB 行实证，fake 不写库无法满足）。`APIKeyUsageLog` 无 JSON/PG 专有列，sqlite 胜任。
- **D-03a（fake/真实 DB 职责正交）:** Phase 57 的 `apikey_integration_test.go`（fake UsageLogger 测认证链 context 键）**原样保留**；本 phase 新增真实 DB 测试与之并存不回归。
- **D-04（写入失败不静默）:** DB 写入失败用既有 **`pkg/logger`**（alias `applogger`）记录，如 `applogger.Errorf("[USAGE_LOG] 写入失败 key=%s path=%s: %v", ...)`。保持 fire-and-forget：仅记录、不阻塞、不 panic。**不**引入 Prometheus metric（deferred）。

### Claude's Discretion（研究给出推荐）

- 记录时机实现：`defer` vs 显式 `c.Next()`+后续语句 → **推荐显式**（见 §Pattern 1）。
- Duration 测量 / StatusCode 捕获 → 仓库既有标准模式（见 §Code Examples）。
- 超时精确值（5–15s）→ **推荐 10s**（单条 INSERT 语义，与 redis.go L2 降级超时同量级）。
- 测试文件落点 → **推荐分工**：middleware 时序/状态（SC#1/#2）落 `internal/middleware/apikey_integration_test.go` 新子测试；cancel-race（SC#4）落 `internal/services/usage_logger_test.go` 新子测试；successRate（SC#3）落 `internal/services/system/apikey_service_test.go` 新子测试。
- async 写入可测试性机制 → **推荐 `require.Eventually` 轮询 DB 行**（见 §异步写入可测试性机制）。
- UserAgent 字段捕获 → 可选增强，SC 未要求，planner 酌情（低成本时可顺带从 `c.Request.UserAgent()` 填）。

### Deferred Ideas (OUT OF SCOPE)

- 写入失败暴露为 Prometheus / 内存 counter 指标——超 phase 范围，独立 metric 基建后再补。
- UserAgent / 额外请求元数据捕获——可选增强。
- `operlog-exclude-paths.md` todo 不折叠（operlog 审计系统与 API Key 使用日志是两套独立系统）。
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description (from REQUIREMENTS.md) | Research Support |
|----|------------------------------------|------------------|
| OBSERV-01 | 记录时机移到 `c.Next()` 之后，`StatusCode`/`Duration`/`Success` 取真实值（修 P1-2） | §Pattern 1（记录时机）+ §Code Examples（gin 仓库标准模式）；`LogUsageRequest` 已含三字段，只需 middleware 在 `c.Next()` 后正确填充 |
| OBSERV-02 | `GetUsageLogSummary` 的 `successRate` 基于真实 `Success` 聚合，不再恒 ≈ 0%（P1-2 连锁） | 纯连锁：`apikey_service.go:519` `successCount/total*100` 不需改；只要 OBSERV-01 把 `Success` 填对，聚合自动可信。SC#3 测试用混合 success 行验证 |
| OBSERV-03 | 异步 goroutine 用独立、不被请求生命周期取消的 context（修 P2-b） | §Pattern 2（detached context）+ §Code Examples（`pkg/cache/redis.go:604` 近乎同义先例）；`logUsageAsync` 改 `context.WithTimeout(context.Background(), 10s)` |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| 捕获真实 StatusCode/Duration/Success | Middleware 层（`internal/middleware/apikey.go`） | — | gin middleware 是唯一能在 `c.Next()` 前后观测 `c.Writer.Status()` 与 `time.Since(start)` 的位置 |
| detach 请求生命周期（独立 ctx） | Service 层（`internal/services/usage_logger.go` impl） | — | D-02 责任内聚：由 `UsageLogger` 自身契约保证，而非每个调用点各自记得 detach |
| fire-and-forget 单 INSERT 写入 | Service 层（`usageLoggerImpl.logUsageAsync`） | DB（`sys_api_key_usage_logs`） | 异步 goroutine 内执行；失败仅 `applogger.Errorf` |
| successRate 聚合 | Service 层（`internal/services/system/apikey_service.go`） | — | `GetUsageLogSummary` 消费 `Success` 字段；本 phase 不改聚合代码 |
| 写入失败可见性 | `pkg/logger`（applogger） | — | D-04 替换 `_ = err` 静默吞错 |

## Standard Stack

**无新增依赖**——本 phase 是纯逻辑修复，全部基于既有 stack。Package Legitimacy Audit 不适用（零新增包）。

### Core（既有，复用）
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/gin-gonic/gin` | 既有 | middleware `c.Next()` / `c.Writer.Status()` | 仓库全局 web 框架 |
| `gorm.io/gorm` + `gorm.io/driver/sqlite` | 既有 | 单 INSERT 写入 + 测试 DB | 项目 ORM；sqlite 已是测试依赖（见 `usage_logger_test.go`） |
| `github.com/stretchr/testify` v1.11.1 | 既有 | `assert`/`require`（含 `require.Eventually`） | TESTING.md 钉死的断言库 |
| `github.com/sirupsen/logrus`（经 `pkg/logger`） | 既有 | `applogger.Errorf` 写入失败记录 | D-04 指定 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff（不采用） |
|------------|-----------|--------------------|
| `require.Eventually` 轮询 DB | `time.Sleep` 固定等待 | 既有测试用 sleep，但 flaky 且慢；新测试用 Eventually 提升确定性（见 §异步可测试性） |
| `require.Eventually` | 注入同步测试模式 / WaitGroup hook | 需改生产代码加测试分支，违反 fire-and-forget 纯粹性；拒绝 |
| 文件 sqlite | `:memory:?cache=shared` | 既有测试已证明并发写锁撞锁；拒绝退回（见 §Pitfall 2） |

## Architecture Patterns

### System Architecture Diagram（修复后数据流）

```
HTTP Request (X-API-Key header)
        │
        ▼
┌─────────────────────────────────────────────────────┐
│ MultiAuth middleware (internal/middleware/apikey.go) │
│                                                     │
│  1. extractAPIKey / isValidKeyFormat                │
│     ├── 失败 → 401 (c.Abort; return)  ◄── pre-auth  │
│     │                                失败: 不可日志  │
│  2. apiKeyService.ValidateAPIKey                    │
│     ├── 失败 → 401 (c.Abort; return)  ◄── pre-auth  │
│  3. IP 白名单 → 403 (c.Abort; return) ◄── pre-auth  │
│  4. setUserContextForAPIKey (7 context 键)          │
│  5. start := time.Now()                             │
│  6. c.Next()  ──────────────► 下游 middleware/handler
│  7. (c.Next() 返回后)                               │
│     code := c.Writer.Status()      ◄── 真实状态码  │
│     dur  := time.Since(start).Milliseconds()        │
│     success := code >= 200 && code < 300  (D-01)    │
│  8. usageLogger.LogUsage(c.Request.Context(), req)  │
│        │  (同步调用; middleware 不再包 go func—D-02a)│
└────────┼────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────┐
│ usageLoggerImpl.LogUsage  (internal/services/)      │
│   go logUsageAsync(ctx, req)   ◄── 内部异步执行     │
└────────┼────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────┐
│ logUsageAsync                                       │
│  detachedCtx, cancel := context.WithTimeout(        │
│      context.Background(), 10*time.Second)  ◄── D-02│
│  defer cancel()                                     │
│  // 忽略调用方 ctx 的取消; 仅 detachedCtx 驱动 DB 写│
│  if err := db.WithContext(detachedCtx).Create(...); │
│     err != nil {                                    │
│       applogger.Errorf("[USAGE_LOG] ...: %v", err)  │ ◄── D-04
│     }   // 替换 _ = err 静默吞错                     │
└─────────────────────────────────────────────────────┘
         │
         ▼ (P2-b 修复: 请求 ctx 取消不再传播到这里)
sys_api_key_usage_logs (StatusCode/Duration/Success 取真实值)
         │
         ▼
GetUsageLogSummary.successRate = successCount/total*100  ◄── 自动可信 (OBSERV-02)
```

### Recommended Project Structure

无新增目录/文件类型——仅改 2 个源文件、扩 3 个既有测试文件：
```
internal/
├── middleware/apikey.go                      # 改: 记录点后移 + 填字段 + 去冗余 goroutine
├── services/usage_logger.go                  # 改: detached context + applogger.Errorf
├── middleware/apikey_integration_test.go     # 扩: SC#1/#2 真实 DB 时序/状态子测试
├── services/usage_logger_test.go             # 扩: SC#4 cancel-race 子测试
└── services/system/apikey_service_test.go    # 扩: SC#3 successRate 混合行子测试
```

### Pattern 1: 记录时机——显式 `c.Next()` + 后续语句（推荐，非 defer）

**What:** gin middleware 在 `c.Next()` 返回后捕获响应状态码与耗时。
**When to use:** 任何需要在「下游 handler 完成后」观测响应结果的 middleware。
**推荐写法（显式，非 defer）:**

```go
// Source 模式: pkg/middleware/logger.go:47-49 (VERIFIED 仓库既有标准模式)
return func(c *gin.Context) {
    // ... 验证 key / IP 白名单 / setUserContextForAPIKey ...

    start := time.Now()
    c.Next()  // 下游 handler 执行完毕后返回

    // c.Next() 之后: 状态码/耗时此刻才真实可用
    statusCode := c.Writer.Status()
    duration := time.Since(start).Milliseconds()  // int 毫秒, 对齐 LogUsageRequest.Duration

    userID := ""
    if apiKey.UserID != nil {
        userID = *apiKey.UserID
    }
    usageLogger.LogUsage(c.Request.Context(), &services.LogUsageRequest{
        APIKeyID:   apiKey.ID,
        UserID:     userID,
        Method:     c.Request.Method,
        Path:       c.Request.URL.Path,
        ClientIP:   c.ClientIP(),
        StatusCode: statusCode,
        Duration:   int(duration),
        Success:    statusCode >= 200 && statusCode < 300,  // D-01
        // UserAgent: 字符串指针, 可选 (c.Request.UserAgent())
    })
    // LogUsage 内部 go logUsageAsync(); middleware 不再包外层 goroutine (D-02a)
}
```

**为什么显式而非 defer:** `defer` 写法（`defer func(){ code := c.Writer.Status(); ... LogUsage(...) }()`）功能等价但可读性略差（延迟执行点不直观）；显式 `c.Next()`+后续语句与 `pkg/middleware/logger.go` 一致，planner 任选其一皆可——本研究**推荐显式**，与仓库主流 middleware 对齐。

**⚠️ 关键语义边界（pre-auth 失败不可日志）:** 记录点在 `c.Next()` 之后，意味着只有在 MultiAuth **已 setUserContextForAPIKey 并到达 c.Next()** 的路径才会记录。Pre-auth 失败（key 格式错 / ValidateAPIKey 失败 / IP 白名单拒）走 `c.Abort(); return`，**在 `c.Next()` 之前退出**，故不会记录——这是结构上无法避免的（无 resolved apiKey 可记）。**planner 须确保 SC#2 失败用例用「下游产生的失败」**：RequireScope→403、handler→429、handler→500。这些发生在 `c.Next()` 下游，MultiAuth 返回时 `c.Writer.Status()` 捕获真实失败码 → 记录 `Success=false`。SC#2 若误用 pre-auth 401 测试，会因「无日志行」误判失败。

### Pattern 2: detached-with-timeout context（D-02 落地）

**What:** fire-and-forget 异步写入用独立 `context.Background()` 派生的 timeout context，忽略调用方 ctx 的取消。
**When to use:** 任何「后台任务，应独立于请求生命周期」的异步写入——本仓库已确立的模式。
**近乎同义的仓库先例（比 PROJECT.md v1.19 更贴近）:**

```go
// Source: pkg/cache/redis.go:601-605 (VERIFIED, 注释原话佐证 D-02 依据)
// P1 fix: 不能用请求 ctx 去做 L2 异步入队 —— HTTP 请求 ctx 通常只有 5-30s 截止时间,
// 但 L2 写入是后台任务,应当独立于请求生命周期。改用独立 ctx 隔离,
// 真正的客户端取消不应阻塞 L2 异步刷盘。
enqueueCtx, cancelEnqueue := context.WithTimeout(context.Background(), m.l2Writer.GetFallbackTimeout())
defer cancelEnqueue()
```

**logUsageAsync 落地写法:**

```go
// Source 模式: 套用 pkg/cache/redis.go:604 + cmd/main.go:263 先例
func (s *usageLoggerImpl) logUsageAsync(ctx context.Context, req *LogUsageRequest) {
    // D-02 / OBSERV-03: 用独立 ctx 写 DB, 忽略调用方 ctx 的取消信号。
    // 调用方 ctx (c.Request.Context()) 仅用于传递请求范围值, 不用于取消。
    // 超时兜底防 DB 挂起泄漏 goroutine (单条 INSERT 量级 ~10s)。
    detachedCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    _ = ctx // 显式标记: 不用于本次 DB 写入的取消控制

    usageLog := models.APIKeyUsageLog{
        APIKeyID:   req.APIKeyID,
        UserID:     req.UserID,
        Method:     req.Method,
        Path:       req.Path,
        StatusCode: req.StatusCode,
        ClientIP:   req.ClientIP,
        UserAgent:  req.UserAgent,
        Duration:   req.Duration,
        Success:    req.Success,
        CreatedAt:  time.Now(),
    }

    if err := s.db.WithContext(detachedCtx).Create(&usageLog).Error; err != nil {
        // D-04: 不再 _ = err 静默吞错; 用 applogger 暴露写入失败
        applogger.Errorf("[USAGE_LOG] 写入失败 key=%s path=%s: %v", req.APIKeyID, req.Path, err)
    }
}
```

**调用方 ctx 边界（重要）:** `logUsageAsync(ctx, req)` 签名保留 `ctx` 参数（不破坏接口），但内部**仅用 `detachedCtx` 驱动 DB**。调用方传入的 `c.Request.Context()` 被「降级」为仅可提取请求范围值（如 trace ID）的载体——若未来需要从请求 ctx 取值注入日志，可读 `ctx.Value(...)`，但**绝不**用 `ctx` 做 DB 写入的取消控制。`_ = ctx` 显式标注此契约。

### Anti-Patterns to Avoid

- **❌ middleware 再包外层 goroutine（D-02a 移除项）:** `go func(){ usageLogger.LogUsage(...) }()` 冗余——`LogUsage` 内部已 `go logUsageAsync()`，双层 goroutine 是 P2-b 之外的额外异步层。
- **❌ `time.Sleep` 等待异步落库:** 固定 sleep 是已知 flaky 反模式（既有测试用 100–500ms），CI 高并行下时序漂移导致偶发失败；新测试用 `require.Eventually`。
- **❌ 用 `AutoMigrate(&models.APIKeyUsageLog{})` 建测试表:** `gen_random_uuid()` 是 PG 专有，sqlite 不识别 → INSERT 无默认主键失败。沿用既有 `setupUsageLoggerTestDB` 的裸 `CREATE TABLE` DDL。
- **❌ 退回 `file::memory:?cache=shared`:** 既有测试注释明载并发写锁撞锁（`database table is locked`）。
- **❌ 把 detach 责任放到 middleware 调用点:** D-02 明确责任内聚到 impl——否则任何新调用方（不止 MultiAuth）都得自己记得 detach，易漏。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 异步等待确定性 | `time.Sleep` 固定延迟 / 自造轮询循环 | `require.Eventually(t, func() bool {...}, 2s, 10ms)` | testify v1.11.1 已提供标准实现；固定 sleep flaky 且慢 |
| 独立 ctx + 超时 | 手写 goroutine + select + timer | `context.WithTimeout(context.Background(), 10s)` | 标准库，仓库 redis.go/main.go 既有先例 |
| 写入失败日志 | 自造 log 接口 / metric counter | `applogger.Errorf(...)`（`pkg/logger`） | D-04；既有基建，零新依赖 |
| 测试 DB 表创建 | `AutoMigrate` | 既有裸 `CREATE TABLE` DDL helper | 绕过 `gen_random_uuid()` PG 专有陷阱 |

**Key insight:** 本 phase 的所有「基础设施」（异步、detach、日志、测试 DB、断言轮询）仓库都有现成标准实现或先例——**零造轮子**。

## Common Pitfalls

### Pitfall 1: `gen_random_uuid()` 在 sqlite 失败
**What goes wrong:** 用 `db.AutoMigrate(&models.APIKeyUsageLog{})` 建表后，`LogUsage` 的 INSERT 失败——sqlite 不识别 `default:gen_random_uuid()`（`api_key_usage_log.go:9` 的 PG 专有 tag）。
**Why it happens:** GORM AutoMigrate 把模型 tag 翻译成 DDL `DEFAULT gen_random_uuid()`，sqlite 无此函数。
**How to avoid:** 沿用既有 `setupUsageLoggerTestDB`（`usage_logger_test.go:30-57` / `apikey_integration_test.go:111-137`）的裸 `CREATE TABLE` DDL——`id TEXT PRIMARY KEY`，sqlite 对 TEXT 主键的 INSERT 自动接受应用层未设的空字符串/由 GORM 在应用层生成。两个文件已有 helper，新子测试直接复用。
**Warning signs:** 测试报 `gen_random_uuid: no such function` 或 INSERT 返回 `NOT NULL constraint failed`。

### Pitfall 2: `:memory:?cache=shared` 并发写锁撞锁
**What goes wrong:** 多个子测试/异步 goroutine 并发 INSERT 时报 `database table is locked`，导致行丢失、`assert.Len(logs, N)` 偶发失败。
**Why it happens:** 共享 in-memory DB 是单一连接，SQLite 写锁互斥；fire-and-forget goroutine 与测试主线程争锁。
**How to avoid:** 沿用既有「每测试独立文件 DB（`os.TempDir` + `time.Now().UnixNano()` + `pid` 唯一名）+ `busy_timeout(5000)`」模式。**不用 `t.TempDir()`**——fire-and-forget goroutine 测试结束后仍可能写文件，`t.TempDir` 自动 cleanup 删占用文件会 mark test failed。残留 .db 文件留在系统 temp，由 OS 定期清理（单测 <100KB）。
**Warning signs:** 间歇性 `database table is locked` / 行数断言偶发少一条。

**⚠️ 与 CONTEXT.md D-03 措辞的差异:** D-03 写「sqlite in-memory」，但仓库既有实践（含注释化的迁移决策）是**文件 DB**。本研究判断 D-03 的「in-memory」是 loose 措辞，其核心要求是「真实 DB（非 fake）、hermetic、零外部依赖、CI 友好」——文件 sqlite 同样满足且已验证。**planner 应采用文件 DB 模式**；若坚持 in-memory，须自行解决撞锁（per-test 独立 `:memory:` 无 `cache=shared` 但 goroutine 争同连接锁仍有风险）。

### Pitfall 3: pre-auth 401 不可日志（见 Pattern 1 备注）
**What goes wrong:** SC#2 测试若用「无效 key → 401」期望日志行出现，会失败——pre-auth 失败在 `c.Next()` 前 `c.Abort(); return`，记录点（`c.Next()` 后）不可达。
**How to avoid:** SC#2 失败用例用**下游产生的失败**（RequireScope→403 / handler→429 / handler→500）。
**Warning signs:** SC#2 测试断言「日志行 Success=false」时找不到行。

### Pitfall 4: Phase 57 fake 测试回归风险（已评估，低）
**What goes wrong:** 记录点后移后，Phase 57 `apikey_integration_test.go` 的 `fakeUsageLogger.LogUsage` 时机改变。
**Why 不回归:** fake 的 `LogUsage` 同步 `close(done)`，`waitForLog(t)` 在 `ServeHTTP` 返回后调用——若 fake 已 close 则立刻返回，channel happens-before 保证 `logged=true` 可见。「有效key+缺失scope_403」子测试在修复后会多触发一次 LogUsage（下游 RequireScope 403 后 MultiAuth 返回捕获 403），但该子测试**未断言 `logged`**，仅断 `w.Code==403`，故不回归。
**How to avoid:** Phase 57 fake 测试**原样保留**（D-03a）；新测试独立写，不共用 fake。

## 异步写入可测试性机制（推荐 — 研究焦点 #3）

**背景:** `LogUsage` 内部 `go logUsageAsync()` 是 fire-and-forget；时序/取消测试需等待 goroutine 完成 DB 写入后才能断言行存在。CONTEXT.md 列为 Claude's Discretion，是本研究**最需明确推荐**的点。

**四方案对比:**

| 方案 | 机制 | 生产代码侵入 | 确定性 | 评估 |
|------|------|-------------|--------|------|
| (a) **轮询 DB 至行出现 + 超时** | `require.Eventually(predicate, 2s, 10ms)` 查 `Count()>0` | 零 | 高（标准 idiom） | ✅ **推荐** |
| (b) 注入同步测试模式 | config 开关或字段让 goroutine 改同步 | 中（加测试分支） | 高 | ❌ 违反 fire-and-forget 纯粹性 |
| (c) `sync.WaitGroup` / done channel 注入 | impl 加 `onDone chan struct{}` 测试 hook | 中（加测试 hook 字段） | 高 | △ 可选但侵入 |
| (d) 接口抽象注入 fake | Phase 57 已用 fake | 零 | 高 | ❌ D-03 明确：fake 无法满足 SC#1/#4「DB 行实证」 |

**推荐: 方案 (a) `require.Eventually`**

**理由:**
1. **零生产代码侵入**——`LogUsage`/`logUsageAsync` 签名与实现完全不为测试让步，保持 fire-and-forget 纯粹。
2. **确定性优于既有 `time.Sleep`**——既有 `usage_logger_test.go` 用 `time.Sleep(100–500ms)` 是已知 flaky 反模式（注释提到并发撞锁导致行丢失）；`require.Eventually` 在条件满足时立刻返回（快），超时才失败（确定性失败而非偶发）。
3. **testify v1.11.1 已含**——`require.Eventually(t, condition func() bool, waitFor time.Duration, tick time.Duration, msgAndArgs ...interface{})`，TESTING.md 钉死 testify 是既有约定。
4. **完美适配 SC#4 cancel-race**——`logUsageAsync` 改 detached context 后，即便调用方 ctx 已 cancel，行仍应落库；`require.Eventually` 轮询 DB 行是验证「最终一致」的自然方式。

**注:** 全仓库当前未使用 `require.Eventually`（既有测试多用 `time.Sleep`）。引入它是**升级**而非偏离——新测试用更好 idiom，既有测试不在本 phase 范围内重构（保持 D-03a「不回归」原则）。

**推荐封装（测试 helper）:**

```go
// waitForUsageLog 轮询 DB 至指定 apiKeyID 的日志行数 >= want, 或超时失败。
// 替代 time.Sleep 的确定性版本。用于 fire-and-forget 异步落库的测试同步。
func waitForUsageLog(t *testing.T, db *gorm.DB, apiKeyID string, want int64) {
    t.Helper()
    require.Eventually(t, func() bool {
        var count int64
        db.Model(&models.APIKeyUsageLog{}).Where("api_key_id = ?", apiKeyID).Count(&count)
        return count >= want
    }, 2*time.Second, 10*time.Millisecond, "usage log for key=%s not persisted within 2s", apiKeyID)
}
```

**SC#4 cancel-race 专用测试骨架:**

```go
func TestLogUsage_CancelledCtxStillWrites_D02(t *testing.T) {
    db := setupUsageLoggerTestDB(t)
    logger := NewUsageLogger(db)

    ctx, cancel := context.WithCancel(context.Background())
    cancel() // 预取消: 模拟请求结束 (P2-b 场景)

    err := logger.LogUsage(ctx, &LogUsageRequest{
        APIKeyID:   "cancel-race-key",
        Method:     "GET", Path: "/test",
        StatusCode: 200, Success: true, Duration: 10,
    })
    require.NoError(t, err)

    // 修复前 (复用 c.Request.Context()): ctx 已 cancel → Create 失败 → _ = err 吞掉 → 无行
    // 修复后 (D-02 detached ctx): 忽略调用方 cancel → 行落库
    waitForUsageLog(t, db, "cancel-race-key", 1)

    var log models.APIKeyUsageLog
    require.NoError(t, db.Where("api_key_id = ?", "cancel-race-key").First(&log).Error)
    assert.Equal(t, 200, log.StatusCode)
    assert.True(t, log.Success)
}
```

## Code Examples（均经实读核对）

### gin middleware 标准记录模式（仓库既有）
```go
// Source: pkg/middleware/logger.go:47-49 (VERIFIED)
func logRequest(c *gin.Context, startTime time.Time, bodyBytes []byte) {
    latency := time.Since(startTime)
    statusCode := c.Writer.Status()
    // ... 用 statusCode / latency 构建日志 ...
}
// 调用点: LoggerMiddleware 在 c.Next() 前后捕获 startTime / 响应状态
```

### detached context 写入先例（仓库既有，近乎同义）
```go
// Source: pkg/cache/redis.go:601-605 (VERIFIED, 注释佐证 D-02 依据)
// P1 fix: 不能用请求 ctx 去做 L2 异步入队...改用独立 ctx 隔离,
// 真正的客户端取消不应阻塞 L2 异步刷盘。
enqueueCtx, cancelEnqueue := context.WithTimeout(context.Background(), m.l2Writer.GetFallbackTimeout())
defer cancelEnqueue()
```

### applogger 写入失败记录先例（仓库既有）
```go
// Source: internal/services/config_backup_service.go:247 (VERIFIED)
// 注: 该行实为 applogger.Infof (非 Errorf); D-04 指定用 Errorf 是因 DB 写入
// 失败语义上=error 级别, 比 info 更正确。结构模式 (applogger.Xf + [模块] 前缀) 一致。
applogger.Infof("[配置备份] 备份失败 [%s]: %v", device.DeviceName, err)

// 本 phase 落地 (D-04, 升级 severity 为 Errorf):
// applogger.Errorf("[USAGE_LOG] 写入失败 key=%s path=%s: %v", req.APIKeyID, req.Path, err)
```

### 测试 DB helper（既有，复用）
```go
// Source: internal/services/usage_logger_test.go:30-57 (VERIFIED)
// 每测试独立文件 DB + busy_timeout; 不用 t.TempDir (fire-and-forget goroutine 测试后仍写)
func setupUsageLoggerTestDB(t *testing.T) *gorm.DB {
    dbPath := filepath.Join(os.TempDir(), fmt.Sprintf("xingran_usage_%d_%d.db",
        time.Now().UnixNano(), os.Getpid()))
    dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)", dbPath)
    db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
        DisableForeignKeyConstraintWhenMigrating: true,
    })
    require.NoError(t, err)
    err = db.Exec(`CREATE TABLE IF NOT EXISTS sys_api_key_usage_logs (
        id TEXT PRIMARY KEY, api_key_id TEXT NOT NULL, user_id TEXT NOT NULL,
        method TEXT, path TEXT, status_code INTEGER, client_ip TEXT,
        user_agent TEXT, duration INTEGER, success BOOLEAN, created_at DATETIME)`).Error
    require.NoError(t, err)
    return db
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| middleware `go func(){ LogUsage }` + LogUsage 内部 `go logUsageAsync` | 去掉外层 goroutine，仅保留内部异步（D-02a） | 本 phase | 消除冗余双重异步，记录点可干净后移到 `c.Next()` 后 |
| `logUsageAsync(ctx)` 复用调用方 ctx | `context.WithTimeout(context.Background(), 10s)`（D-02） | 本 phase | P2-b 消除：请求取消不再传播到 DB 写入 |
| `_ = err` 静默吞错 | `applogger.Errorf(...)`（D-04） | 本 phase | 写入失败可见，观测系统自身可观测 |
| `time.Sleep` 等待异步 | `require.Eventually` 轮询（推荐） | 本 phase | 新测试确定性，不再 flaky |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | D-03「sqlite in-memory」措辞应理解为 loose，核心要求是「真实 DB + hermetic」，文件 sqlite 同样满足 | §Pitfall 2 / §User Constraints | 低：文件 DB 是仓库既有验证模式；若用户坚持字面 in-memory，planner 须自行解决撞锁 |
| A2 | SC#2 的「401」指下游 401 而非 pre-auth 401（pre-auth 不可日志） | §Pattern 1 / §Pitfall 3 | 中：若 planner 用 pre-auth 401 写 SC#2 测试会误判；本研究已明确规避 |
| A3 | `LogUsage(ctx, req)` 签名保留 `ctx` 参数（不破坏接口），仅内部改用 detachedCtx | §Pattern 2 | 低：grep 确认仅 `apikey.go:67` + 14 处测试调用，签名不变即无破坏 |
| A4 | 超时值 10s 合适（单条 INSERT 语义，与 redis.go L2 降级超时同量级） | §Pattern 2 | 极低：CONTEXT.md 给了 5–15s 区间，10s 居中 |
| A5 | `apikey_service.go:519` 聚合逻辑不需改动（OBSERV-02 是 OBSERV-01 连锁） | §Phase Requirements | 极低：实读确认 `successCount/total*100`，只要 Success 填对即自动可信 |

**[VERIFIED]/[CITED]/[ASSUMED] 标注说明:** 本 phase 所有 file:line 均经实读核对（`[VERIFIED: codebase]`）；`pkg/cache/redis.go`、`pkg/middleware/logger.go`、`config_backup_service.go` 先例均经实读（`[CITED: in-repo]`）；无 `[ASSUMED]` 来自训练数据的未验证断言——A1–A5 均为基于实读证据的推断，已标注风险。

## Open Questions

1. **SC#3 successRate 测试落点**
   - What we know: `GetUsageLogSummary` 在 `internal/services/system/apikey_service.go`，既有 `apikey_service_test.go` 存在。
   - What's unclear: 是否已有 `GetUsageLogSummary` 测试可复用 seed 逻辑。
   - Recommendation: planner 先 grep `apikey_service_test.go` 现有覆盖；若已有则扩展混合 success 行子测试，否则新建——seed 直接 INSERT 若干行（success true/false 混合）后调 `GetUsageLogSummary`，断言 `successRate ∈ (0,100)`。

2. **UserAgent 捕获是否顺带做**
   - What we know: `LogUsageRequest.UserAgent *string` 未填；`c.Request.UserAgent()` 取值成本极低。
   - Recommendation: 若 planner 在改 middleware 时一并加一行 `ua := c.Request.UserAgent(); ...UserAgent: &ua` 是低成本收益；但 SC 未要求，**不做也不阻塞**。保持可选。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | 全 phase | ✓ | 1.24（CLAUDE.md） | — |
| `gorm.io/driver/sqlite` | 测试 DB（D-03） | ✓ | 既有依赖（`usage_logger_test.go` 已用） | — |
| `github.com/stretchr/testify` v1.11.1 | 断言 + `require.Eventually` | ✓ | TESTING.md 钉死 | — |
| `pkg/logger`（applogger） | D-04 写入失败记录 | ✓ | 仓库内置 | — |
| PostgreSQL | **本 phase 不需要** | — | — | sqlite 文件 DB 替代（D-03） |
| Redis | **本 phase 不需要** | — | — | — |

**Missing dependencies with no fallback:** 无。
**Missing dependencies with fallback:** 无。

## Validation Architecture

> `config.json` `workflow.nyquist_validation: true` → 本段必需。

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` + testify v1.11.1（assert/require） |
| Config file | 无独立配置；测试紧邻源码 `*_test.go`（TESTING.md 约定） |
| Quick run command | `go test ./internal/middleware/... ./internal/services/...` |
| Full suite command | `go test ./...` |
| 真实 DB | sqlite 文件 DB（`setupUsageLoggerTestDB` 既有 helper，per-test 独立文件） |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| OBSERV-01 / SC#1 | 2xx 请求后日志行 `StatusCode∈2xx` / `Duration>0` / `Success=true` | 集成（真实 gin.Engine + 真实 NewUsageLogger + sqlite） | `go test ./internal/middleware/ -run TestMultiAuthUsageLogTiming -v` | ❌ Wave 0 新增 |
| OBSERV-01 / SC#2 | 下游失败（403/429/500）后 `Success=false` / `StatusCode=真实码` | 集成（同上，handler 返回失败码） | `go test ./internal/middleware/ -run TestMultiAuthUsageLogFailure -v` | ❌ Wave 0 新增 |
| OBSERV-02 / SC#3 | 混合 success 行后 `successRate ∈ (0,100)`，不恒 ≈ 0% | 单元（seed DB 行 + 调 GetUsageLogSummary） | `go test ./internal/services/system/ -run TestGetUsageLogSummaryMixed -v` | ❌ Wave 0 新增（落既有 apikey_service_test.go） |
| OBSERV-03 / SC#4 | 调用方 ctx cancel 后日志仍完整写入（detached context） | 单元（NewUsageLogger + 预取消 ctx + require.Eventually） | `go test ./internal/services/ -run TestLogUsageCancelledCtx -v` | ❌ Wave 0 新增（落既有 usage_logger_test.go） |
| SC#5 | 全绿 + 防回归（P1-2 时序 + P2-b 取消） | 全 suite | `go test ./internal/middleware/... ./internal/services/...` | ✅ 既有 + 新增 |

### Success Criteria → 验证机制（断言形式）

| SC# | 验证机制 | 断言形式 | 测试类型 |
|-----|----------|----------|----------|
| SC#1 | 真实 gin + 真实 UsageLogger + sqlite；handler 返回 200；`waitForUsageLog` 轮询行出现 | `assert.Equal(200, row.StatusCode)` / `assert.Greater(row.Duration, 0)` / `assert.True(row.Success)` | 集成 + DB 行实证 |
| SC#2 | handler 下游返回 403/429/500（**非 pre-auth 401**，见 Pitfall 3） | `assert.False(row.Success)` / `assert.Equal(expectedCode, row.StatusCode)` | 集成 + DB 行实证 |
| SC#3 | 直接 INSERT seed 若干 success=true/false 行，调 `GetUsageLogSummary` | `assert.Greater(summary.SuccessRate, 0.0)` / `assert.Less(summary.SuccessRate, 100.0)` | 单元 + DB 行实证 |
| SC#4 | `ctx, cancel := WithCancel(Background()); cancel(); LogUsage(ctx, req)`；`require.Eventually` 等行出现 | 行出现 + 字段正确（`require.Eventually` 内 `Count()>=1`，后续字段 assert） | 单元 + DB 行实证 |
| SC#5 | `go test ./...` exit 0 | `go test` 全绿 | 全 suite |

### 异步写入确定性策略（Nyquist Dimension 8 关键）

- **机制:** `require.Eventually(predicate, 2*time.Second, 10*time.Millisecond)` 轮询 DB 行数。
- **为何确定性:** 条件满足立刻返回（不浪费时间），超时则**确定性失败**（非偶发 flaky）；与 fire-and-forget goroutine 的「最终一致」语义天然对齐。
- **封装:** `waitForUsageLog(t, db, apiKeyID, want)` helper（见 §异步可测试性机制）。
- **SC#4 专用:** 预取消 ctx 后轮询——修复前行永不出现（超时失败），修复后行出现（成功）。这是 P2-b 防回归的核心断言。

### Hermetic 性保证（sqlite 测试 DB）

- **Per-test 独立文件 DB:** `os.TempDir()` + `UnixNano()` + `pid` 唯一名 → 测试间零共享状态。
- **`busy_timeout=5000`:** 写锁排队而非立即报错，消除并发 goroutine 撞锁。
- **不用 `t.TempDir()`:** fire-and-forget goroutine 测试结束后仍可能写文件，自动 cleanup 删占用文件 mark fail；残留文件由 OS 清理（<100KB/test）。
- **裸 `CREATE TABLE` DDL:** 绕过 `gen_random_uuid()` PG 专有陷阱（Pitfall 1）。
- **零外部进程:** 无需启动 PostgreSQL/Redis，CI 友好。

### Sampling Rate
- **Per task commit:** `go test ./internal/middleware/... ./internal/services/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`；SC#1/#4 必须 DB 行实证（非代码推断）。

### Wave 0 Gaps
- [ ] `internal/middleware/apikey_integration_test.go` — 新增 `TestMultiAuthUsageLogTiming`（SC#1）+ `TestMultiAuthUsageLogFailure`（SC#2）子测试；复用既有 `setupUsageLoggerTestDB`，注入**真实** `NewUsageLogger(db)`（非 fake）
- [ ] `internal/services/usage_logger_test.go` — 新增 `TestLogUsageCancelledCtxStillWrites_D02`（SC#4）+ `waitForUsageLog` helper
- [ ] `internal/services/system/apikey_service_test.go` — 新增 `TestGetUsageLogSummaryMixed`（SC#3）混合 success 行聚合
- [ ] Phase 57 既有 fake 测试**原样保留**（D-03a，职责正交不回归）

*(既有测试基建覆盖认证链；本 phase 新增真实 DB 时序/字段/取消鲁棒性测试，fake↔真实 DB 职责正交)*

## Security Domain

> `security_enforcement` 在 config.json 缺省（=启用）。本 phase 是纯逻辑修复，不新增攻击面。

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no（本 phase 不改认证链——那是 Phase 57 已完成 / Phase 60 启用决策） | 既有 JWT dual-token + API Key 认证不变 |
| V3 Session Management | no | 既有 access/refresh token 不变 |
| V4 Access Control | no（不改 RBAC / scope 校验） | 既有 RequireScope 不变 |
| V5 Input Validation | no（不新增用户输入处理） | LogUsageRequest 字段由 middleware 从已认证请求内部状态填充，非用户直接输入 |
| V6 Cryptography | no | 不触碰 SM2/SM3/SM4 |
| V7 Error Handling & Logging | **yes**（本 phase 核心就是日志） | D-04：写入失败用 applogger 记录，**不泄露敏感字段**——LogUsageRequest 仅含 path/method/IP/status/duration，无 key 明文/密码/token |

### Known Threat Patterns for 使用日志链路

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 日志注入（path 含换行/控制字符污染日志） | Tampering | 既有：path 取自 `c.Request.URL.Path`（gin 已规范化）；`applogger.Errorf` 用 `%v` 格式化，logrus JSON formatter 自动转义。本 phase 不引入新风险 |
| 日志导致 DoS（写入失败拖垮业务） | Denial of Service | D-04 保持 fire-and-forget：失败仅记录、不阻塞、不 panic；detached context 10s 超时兜底防 goroutine 泄漏 |
| 敏感字段入日志 | Information Disclosure | LogUsageRequest **无** key 明文/密码/token 字段（仅有 apiKeyID UUID + path/method/IP/status）；`applogger.Errorf` 仅打 apiKeyID + path + err，无敏感数据 |

**结论:** 本 phase 不扩大攻击面，反而**提升**安全态势（写入失败可见 = 观测系统自身可观测，D-04）。

## Sources

### Primary (HIGH confidence)
- **实读源码（本 session）:** `internal/middleware/apikey.go`（全文）、`internal/services/usage_logger.go`（全文）、`internal/services/system/apikey_service.go`（行 465-585）、`internal/models/api_key_usage_log.go`、`internal/models/api_key.go`
- **实读测试（本 session）:** `internal/middleware/apikey_integration_test.go`（Phase 57 fake 测试全文）、`internal/services/usage_logger_test.go`（既有 sqlite 测试全文 + 迁移决策注释）
- **实读先例（本 session）:** `pkg/cache/redis.go:601-605`（detached context 近乎同义先例）、`pkg/middleware/logger.go:47-49`（gin 记录模式）、`internal/services/config_backup_service.go:247`（applogger 先例）、`pkg/logger/logger.go`（`Errorf` API）、`.planning/codebase/TESTING.md`（testify v1.11.1 + 无 gomock 约定）
- **grep 验证:** `LogUsage` 调用点（仅 1 生产 + 14 测试）、`c.Writer.Status()` 既有用例、`context.WithTimeout(context.Background()` 既有先例（redis.go / main.go）、`require.Eventually` 仓库零现用（确认是新引入 idiom）
- **CONTEXT.md（上游决策）:** D-01~D-04 锁定决策引用确认

### Secondary (MEDIUM confidence)
- 无——本 phase 研究全部基于仓库内实读证据，无需外部文档。

### Tertiary (LOW confidence)
- 无——纯 Go/Gin 标准库 + 仓库既有模式，无训练数据断言。

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 零新增依赖，全部既有
- Architecture（记录时机 / detached context / async 可测试性）: HIGH — 仓库内有近乎同义先例（redis.go:604 / logger.go:47）
- Pitfalls: HIGH — `gen_random_uuid` / 撞锁 / pre-auth 401 均经实读测试注释或源码佐证
- Validation Architecture: HIGH — testify v1.11.1 `require.Eventually` 是标准 idiom，sqlite 文件 DB 是仓库验证模式

**Research date:** 2026-08-13
**Valid until:** 2026-09-12（30 天；纯逻辑修复，stack 稳定，不易过期）
