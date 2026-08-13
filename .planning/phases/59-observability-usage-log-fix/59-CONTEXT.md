# Phase 59: 可观测性 / 使用日志修复 - Context

**Gathered:** 2026-08-13
**Status:** Ready for planning

<domain>
## Phase Boundary

让 API Key **使用日志**(`sys_api_key_usage_log`)真实反映请求结果——记录时机移到请求处理完成(`c.Next()`)之后,`StatusCode` / `Duration` / `Success` 取真实值,异步写入不被请求生命周期取消竞态污染,`GetUsageLogSummary` 的 `successRate` 聚合因此可信。

**本 phase 只动使用日志记录链路**,不挂载 MultiAuth(那是 Phase 60 AUTH-03 的 discuss 决策点),不碰限流头编码/密钥哈希(Phase 60),不碰资源级权限矩阵(Phase 61)。日志记录点在 Phase 57 已就绪的认证链 `MultiAuth` goroutine 内,故 **depends on Phase 57**。

**Requirements:** OBSERV-01（记录时机 + 真实 StatusCode/Duration/Success）、OBSERV-02（successRate 可信，P1-2 连锁）、OBSERV-03（独立 context 消除取消竞态 P2-b）

**模型无需迁移:** `models.APIKeyUsageLog` 已含 `StatusCode` / `Duration` / `Success` 列（`usage_logger.go:56-67` 构造时已赋值），当前缺陷是 middleware 根本没填这些字段 + ctx 取消导致写入被丢弃，属纯逻辑修复，**无 schema 变更、无 migration**。

</domain>

<decisions>
## Implementation Decisions

### Success 判定语义（OBSERV-02 successRate 口径）
- **D-01:** `Success` 纯由响应状态码派生（使用日志里 status code 即真相，2xx 不可能是逻辑失败），口径为 **仅 2xx = 成功**：`Success = 200 <= statusCode && statusCode < 300`。3xx/4xx/5xx 一律 `false`。与 SC#1（2xx→Success=true）、SC#2（401/403/429→Success=false）完全对齐；5xx 服务端错误归失败符合直觉；API key JSON 端点几乎不发 3xx，故 3xx 归 false 无实际影响。此口径直接决定 `GetUsageLogSummary.successRate`（`apikey_service.go:519` `successCount/total*100`）的数值语义。

### 异步写入 context 策略（OBSERV-03 / P2-b 修复形状）
- **D-02:** 「请求生命周期免疫」**成为 `UsageLogger` 自身的契约**——在 **impl 内部** detach，而非调用点（middleware）负责。`logUsageAsync`（`usage_logger.go:54`）改用 `context.WithTimeout(context.Background(), ~10s)` 进行 DB 写入，**忽略**调用方传入的 `ctx` 的取消信号（调用方 ctx 仅用于传递请求范围值，不用于取消）。理由：任何调用方（不止 MultiAuth）都自动受保护；超时兜底防止 DB 挂起泄漏 goroutine。
- **D-02a:** **去掉 middleware 冗余 `go func()`**（`apikey.go:62`）。`LogUsage`（`usage_logger.go:47`）内部已 `go logUsageAsync()`，middleware 再包一层 goroutine 是冗余双重异步。修复后 middleware 在 `c.Next()` **之后**同步调用 `usageLogger.LogUsage(c.Request.Context(), req)`，由 LogUsage 内部异步执行。
- **依据:** detached-with-timeout context 是本仓既有先例——PROJECT.md 记录 v1.19 batch `context.WithTimeout(context.Background(), 30*time.Minute)` 规避 `Core.Close()` 截止。本次为单条 INSERT，超时量级远小于 30min（~10s 量级），具体值见 Claude's Discretion。

### 测试验证基座（SC#1/#4 数据库行实证）
- **D-03:** 使用日志的时序/取消测试用 **sqlite in-memory** 作为真实 DB。`APIKeyUsageLog` 表无 JSON/PG 专有列（字段为 string/int/*string/bool/time），sqlite 完全胜任单 INSERT + 回读验证。hermetic、零外部依赖、CI 友好，与 Phase 57 D-02「测试 DB 或 sqlite」一致。SC#1/#4 要求「数据库行实证而非代码推断」——fake UsageLogger（不写库）无法满足，故时序/取消用例必须真实 DB。
- **D-03a:** Phase 57 的 `apikey_integration_test.go`（fake UsageLogger 认证链，断言 context 键）**原样保留**——它测认证、不测时序，与本 phase 新增的真实 DB 测试**职责正交、并存不回归**。两套测试分工：fake 测认证链上下文写入；真实 DB 测使用日志时序/字段/取消鲁棒性。

### 写入失败处理（观测系统自身的可观测性）
- **D-04:** DB 写入失败时用既有 **`pkg/logger`** 记录（替换 `usage_logger.go:73` 的 `_ = err`），如 `logger.Errorf("[USAGE_LOG] 写入失败 key=%s path=%s: %v", req.APIKeyID, req.Path, err)`。参考先例 `internal/services/config_backup_service.go:247` 的 `applogger.Errorf(...)`。**保持 fire-and-forget 语义**——失败仅记录、不阻塞主流程、不 panic、不影响业务请求（使用日志绝不能反过来拖垮业务）。失败可见、零新依赖。不引入 Prometheus metric 计数（超出 phase 范围，见 deferred）。

### 承自 Phase 57 的锁定决策（不重问，仅引用）
- **Phase 57 D-01/D-02:** 认证链测试用手写 fake/stub `UsageLogger` + 真实 `gin.Engine` + `httptest`；另在测试里实例化真实 `NewUsageLogger(db)` 证明构造签名可调用。本 phase 复用该 fake 模式做认证链断言，真实 DB 部分见 D-03。
- **TESTING.md 约定:** 无 gomock；需 DB 用真实连接。
- **中间件位置:** `internal/middleware/`；7 个 gin context 键（`user_id`/`username`/`nickname`/`api_key_id`/`scopes`/`auth_type`/`inherit_perms`）保留（Phase 57 D-04）。

### Claude's Discretion
- **记录时机实现:** 用 `defer` 在 `c.Next()` 返回后捕获状态码，还是显式 `c.Next()` + 后续语句——任选干净的写法（标准 gin 模式）。
- **Duration 测量:** `start := time.Now()` 于函数入口，`time.Since(start)` 于 `c.Next()` 后取毫秒。
- **StatusCode 捕获:** `c.Writer.Status()`。
- **超时精确值:** ~10s 量级，planner 可在 5–15s 间定（单条 INSERT 语义）。
- **测试文件落点:** middleware 集成（`internal/middleware/` 新增时序/取消集成测试，真实 gin+真实 NewUsageLogger+sqlite）还是 services 单元（扩展 `usage_logger_test.go`）——planner 按可测性选。
- **async 写入可测试性:** 时序测试需等待内部异步 goroutine 落库——用轮询 DB / 注入同步测试模式 / 其它机制，planner 择优。
- **UserAgent 字段:** `LogUsageRequest.UserAgent *string` 当前 middleware 未填；是否顺带从 `c.Request.UserAgent()` 捕获属可选增强（SC 未要求），planner 酌情。
- **冗余 goroutine 移除的具体写法:** 见 D-02a，具体重构形式由 planner 定。

### Folded Todos
（无——cross_reference_todos 未折叠任何 todo）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 需求与规划
- `.planning/ROADMAP.md` §Phase 59 — Goal / Depends on (Phase 57) / Requirements (OBSERV-01/02/03) / Success Criteria (SC#1-5)
- `.planning/REQUIREMENTS.md` — OBSERV-01/02/03 定义（行 24-26）+ 需求→phase 映射（行 68-70）
- `.planning/STATE.md` §根因调查结论 — P1-2/P2-b ground-truth 表（文件:行 + 根因， milestone regression scope）
- `.planning/phases/57-auth-chain-core-fix-regression-test/57-CONTEXT.md` — D-01/D-02 测试替身策略 + TESTING.md 约定（直接上游 phase，本 phase 依赖其已就绪的认证链）

### 待修复源码（核心）
- `internal/middleware/apikey.go:61-76` — P1-2+P2-b 使用日志 goroutine（`c.Next()` 前 spawn + 用 `c.Request.Context()` + `LogUsageRequest` 未填 StatusCode/Duration/Success 恒零值）；冗余双重 goroutine（middleware `go func(){LogUsage}` + `LogUsage` 内部 `go logUsageAsync`）
- `internal/services/usage_logger.go` — `UsageLogger` 接口（行 12-17）+ `LogUsageRequest` 结构体（行 20-30，**已含** StatusCode/Duration/Success 字段）+ `logUsageAsync`（行 54-75，双重 goroutine + 静默吞错 `_ = err`）+ `NewUsageLogger(db)` 构造（行 38）
- `internal/services/system/apikey_service.go:465-583` — `UsageSummary` struct（行 466）+ `GetUsageLogSummary` successRate 聚合（行 519 `successCount/total*100`，OBSERV-02 连锁点；success 筛选行 437-439）

### 既有测试基建
- `internal/middleware/apikey_integration_test.go` — Phase 57 既有集成测试（fake UsageLogger 认证链，断言 context 键）；本 phase 新增真实 DB 时序/取消测试与之**并存不回归**
- `internal/services/usage_logger_test.go` — `UsageLogger` 既有测试（可扩展点）
- `internal/services/system/apikey_service_test.go` — `GetUsageLogSummary` 相关既有测试（若有，OBSERV-02 验证可复用）
- `.planning/codebase/TESTING.md` — 无 gomock、需 DB 用真实连接约定（D-03 依据）
- `.planning/codebase/ARCHITECTURE.md` — Handler-Service 分层 + Core DI

### 代码库先例（本次决策依据）
- detached-with-timeout context 先例 — PROJECT.md 记录 v1.19 W2 batch `context.WithTimeout(context.Background(), 30*time.Minute)` 规避 `Core.Close()` 截止（D-02 依据）
- `pkg/logger` 失败记录先例 — `internal/services/config_backup_service.go:247` `applogger.Errorf("[配置备份] 备份失败 [%s]: %v", ...)`（D-04 依据）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `services.UsageLogger` 接口 + `LogUsageRequest`（**已含** StatusCode/Duration/Success 字段）——修复只需 middleware 正确填充 + impl 改 context 来源。
- `NewUsageLogger(db)` 构造函数——Phase 57 D-02 已证明可调用、签名与 MultiAuth 装配兼容。
- `pkg/logger`（Warnf/Infof/Errorf）——写入失败记录（D-04）。
- sqlite GORM——项目 `database.type` 配置支持 sqlite，测试可切 in-memory。
- testify `assert`——既有测试约定（context 键、DB 行断言）。

### Established Patterns
- 中间件位于 `internal/middleware/`（非 `pkg/middleware/`）；gin context 键约定（Phase 57 D-04 保留全部 7 键）。
- 测试紧邻源码 `*_test.go`；无 mock 框架；DB 测试用真实连接（本 phase sqlite in-memory）。
- fire-and-forget 异步 + detached context：v1.19 batch 先例。
- `c.Next()` 前后捕获 gin 响应状态的标准模式：`c.Writer.Status()` + `time.Since(start)`。

### Integration Points
- `MultiAuth(apiKeyService, usageLogger)` 是日志记录点——本 phase 改其 goroutine 时机（移到 c.Next() 后）+ context 来源（D-02）。
- `GetUsageLogSummary` 消费 `Success` 字段做 successRate——OBSERV-02 是 OBSERV-01 的纯连锁，无需单独改聚合逻辑（只要 Success 填对，successRate 自动可信）。

</code_context>

<specifics>
## Specific Ideas

- 用户取向：让使用日志「真实可信」（OBSERV-01），且**观测系统自身也要可观测**（D-04 不静默吞错）。
- 偏好**严格 success 口径**（D-01 仅 2xx）而非宽松口径——宁可 successRate 偏低也要语义干净、可预测。
- 偏好把「请求生命周期免疫」做成 **UsageLogger 自身契约**（D-02 impl 内 detach），而非每个调用点各自记得 detach——更鲁棒、责任内聚。
- 测试取向：SC 的「数据库行实证」是真要求（非代码推断），故宁可引入真实 DB（sqlite）也不用 fake 蒙混。

</specifics>

<deferred>
## Deferred Ideas

- **写入失败暴露为指标计数（Prometheus / 内存 counter 供监控告警）**——D-04 仅用 `pkg/logger` 记录；counter/metric 基建超出 Phase 59 范围。若未来需要 usage-log 丢失率的运营告警，可独立 phase 引入 metric 基建后补。
- **UserAgent / 额外请求元数据捕获**——`LogUsageRequest.UserAgent` 当前未填；顺带捕获属可选增强，SC 未要求，留 Claude's Discretion 或后续 phase。

### Reviewed Todos (not folded)
- `operlog-exclude-paths.md`（todo.match-phase score 0.6，关键词「log」+「phase」误匹配）——属 **operlog（操作日志/审计）系统**（`sys_oper_log` 白名单配置，解决 RPA 心跳日志污染），与 Phase 59 的 **API Key 使用日志**（`sys_api_key_usage_log`）是两套独立系统。Phase 57 已评审并拒绝折叠同一 todo（彼时 score 0.2），本 phase 复评仍**不折叠**。

</deferred>

---

*Phase: 59-可观测性 / 使用日志修复*
*Context gathered: 2026-08-13*
</content>
</invoke>
