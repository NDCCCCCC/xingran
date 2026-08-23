# Phase 59: 可观测性 / 使用日志修复 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-13
**Phase:** 59-可观测性 / 使用日志修复
**Areas discussed:** Success 判定语义, 异步写入 context 策略, 测试验证基座, 写入失败处理 (+ todo 折叠评审)

---

## 前置:灰区选择

用户在 `present_gray_areas` 阶段选择讨论**全部 4 个灰区**(多选)。todo 折叠评审同步进行。

---

## Success 判定语义

`Success` 字段驱动 `GetUsageLogSummary.successRate`。SC#1 锁定 2xx=成功、SC#2 锁定 401/403/429=失败，3xx/5xx 归属未定。

| Option | Description | Selected |
|--------|-------------|----------|
| 仅 2xx = 成功 | `200 <= status < 300` 为 true，3xx/4xx/5xx 均 false。与 SC#1/#2 完全对齐；5xx 归失败符合直觉；API key JSON 端点几乎不返 3xx | ✓ |
| 2xx + 3xx = 成功 | `status < 400` 为 true（含重定向）。HTTP 语义更宽容，但 API key 端点 3xx 罕见，差别极小 | |
| 你来定（Claude 选仅 2xx） | 交给 Claude 按惯例处理 | |

**User's choice:** 仅 2xx = 成功
**Notes:** Success 纯由状态码派生（使用日志里 status code 即真相）。偏好严格口径——宁可 successRate 偏低也要语义干净可预测。3xx/5xx 一律 false。→ CONTEXT D-01

---

## 异步写入 context 策略

P2-b 修复形状。SC#4 已锁定「独立、不被请求生命周期取消的 context」——detach 是定的，开口在「哪一层 detach」+「是否加超时」。

| Option | Description | Selected |
|--------|-------------|----------|
| Impl 内 detach + WithTimeout | `logUsageAsync` 改用 `context.WithTimeout(context.Background(), ~10s)`。「请求生命周期免疫」成为 UsageLogger 自身契约；超时兜底防泄漏；去掉 middleware 冗余 `go func()` | ✓ |
| Middleware 层 detach + 纯 Background | middleware 构造 `context.Background()` 传入；不加超时（写永不被取消，但 DB 挂起泄漏 goroutine）；保护仅限 MultiAuth 调用点 | |
| 你来定（Claude 选 Option 1） | 交给 Claude 按惯例 | |

**User's choice:** Impl 内 detach + WithTimeout
**Notes:** detached-with-timeout 是本仓既有先例（v1.19 batch `WithTimeout(Background, 30min)`）。偏好把「请求生命周期免疫」做成 logger 自身契约而非调用点责任（更鲁棒、责任内聚）。middleware 冗余双重 goroutine（middleware `go func` + LogUsage 内部 `go logUsageAsync`）随修复一并清理。超时量级 ~10s（单条 INSERT）。→ CONTEXT D-02 / D-02a

---

## 测试验证基座

SC#1/#4 要求「数据库行实证而非代码推断」——fake UsageLogger（不写库）无法满足。

| Option | Description | Selected |
|--------|-------------|----------|
| sqlite in-memory | hermetic、零外部依赖、快、CI 友好。APIKeyUsageLog 表无 JSON/PG 专有列，sqlite 完全胜任单 INSERT+回读。与 Phase 57 D-02 一致 | ✓ |
| 测试 PostgreSQL | 与生产方言完全一致（PG 18），零方言风险；但需运行中 PG、较慢、CI 需 PG 服务 | |
| 你来定（Claude 选 sqlite） | 交给 Claude 按惯例 | |

**User's choice:** sqlite in-memory
**Notes:** 已定部分无需再问——Phase 57 的 `apikey_integration_test.go`（fake 认证链）原样保留（测认证、不测时序），与本 phase 新增真实 DB 时序/取消测试职责正交、并存不回归。测试文件落点（middleware 集成 vs services 单元）交 planner。→ CONTEXT D-03 / D-03a

---

## 写入失败处理

当前 `logUsageAsync` 静默吞错（`_ = err`）。保持 fire-and-forget 语义已定（不阻塞主流程）。

| Option | Description | Selected |
|--------|-------------|----------|
| log.Error/Warn | 用既有 `pkg/logger`（参 config_backup_service.go `applogger.Errorf` 模式）记录失败原因。失败可见、零新依赖、不阻塞业务 | ✓ |
| 保持静默 | 现状 `_ = err`。最简，但 usage log 丢失无人知——与「让使用日志可信」目标相悖 | |
| 暴露为指标计数 | 失败递增 counter（Prometheus / 内存）供监控告警。最强可观测，但引入 metric 基建，超 phase 范围 | |

**User's choice:** log.Error/Warn
**Notes:** 项目有既有 `pkg/logger`（提供 Warnf/Infof/Errorf，config_backup_service 用 applogger.Errorf 记设备备份失败）。取向：观测系统自身也要可观测。Prometheus counter 推到后续独立 phase（见 deferred）。→ CONTEXT D-04

---

## Todo 折叠评审

| Option | Description | Selected |
|--------|-------------|----------|
| 不折叠（推荐） | operlog.exclude_paths 与 API Key 使用日志无关，属不同日志系统。标记 reviewed-not-folded | ✓ |
| 折叠进 Phase 59 | 把 operlog 白名单需求纳入本 phase | |

**User's choice:** 不折叠
**Notes:** `operlog-exclude-paths.md`（todo.match-phase score 0.6，关键词「log」+「phase」误匹配）属 operlog（操作日志/审计，`sys_oper_log`）系统，与 Phase 59 的 API Key 使用日志（`sys_api_key_usage_log`）是两套独立系统。Phase 57 已评审拒绝同一 todo（彼时 score 0.2）。记入 CONTEXT deferred「Reviewed Todos」。

---

## Claude's Discretion

- 记录时机实现：`defer` 还是显式 `c.Next()` + 后续语句（任选干净 gin 模式）
- Duration 测量：`time.Since(start)` 取毫秒
- StatusCode 捕获：`c.Writer.Status()`
- 超时精确值：~10s 量级（5–15s 间，单条 INSERT 语义）
- 测试文件落点：middleware 集成 vs services 单元（planner 按可测性选）
- async 写入可测试性等待机制：轮询 DB / 注入同步测试模式 / 其它（planner 择优）
- UserAgent 字段是否顺带捕获（SC 未要求，可选增强）
- 冗余 goroutine 移除的具体重构写法

## Deferred Ideas

- 写入失败暴露为指标计数（Prometheus / 内存 counter 供运营告警）——超 Phase 59 范围，需独立 metric 基建 phase
- UserAgent / 额外请求元数据捕获——可选增强，SC 未要求
