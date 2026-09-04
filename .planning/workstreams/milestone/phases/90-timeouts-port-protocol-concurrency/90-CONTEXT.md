# Phase 90: TIMEOUTS/PORT/PROTOCOL/CONCURRENCY 常量集中化 - Context

**Gathered:** 2026-09-04
**Status:** Ready for planning

<domain>
## Phase Boundary

抽取 4 个 leaf const pkg(`timeouts.go` / `ports.go` / `protocol.go` / `concurrency.go`),共 10 个常量,覆盖 6 处业务超时(command/LDAP/AD sync/scheduler shutdown)+ 1 处 URL 协议(WS origin)+ 1 处 SNMP 端口 + 1 处并发数 + 3 处 scheduler/cron 内部超时(扩展审计) + 1 处 AD sync 任务超时;通过 AST 锁值测试 + invariants 扫描保证未来不出现重复字面量;`go test ./...` 0 失败回归。

**10 个常量最终定义**:
```go
// pkg/constants/timeouts.go (6 个 time.Duration)
const (
    CommandExecTimeout     = 300 * time.Second  // 设备命令执行超时(command_handler.go:61 + execution_handler.go:104 共用)
    CommandReadTimeout     = 60  * time.Second  // 设备命令读取超时(command_handler.go:98 QuickCommand)
    LDAPConnTimeout        = 30  * time.Second  // LDAP 连接超时(ad_ldap_client.go:74)
    ADSyncTimeout          = 30  * time.Minute  // AD 同步上下文超时(scheduler/ad_sync_tasks.go:42 adSchedulerSyncTimeout 重命名)
    SchedulerShutdownTimeout = 5  * time.Second // cron 引擎关闭 grace(scheduler/cron.go:18 defaultShutdownTimeout 重命名)
    ADSyncTaskTimeout      = 1   * time.Minute  // AD 同步单次任务超时(scheduler/ad_sync_tasks.go:164 内联 1*time.Minute 抽常量)
)

// pkg/constants/ports.go (1 个 int)
const (
    SNMPPort = 161  // SNMP 默认端口(discovery_handler.go:126)
)

// pkg/constants/protocol.go (2 个 string)
const (
    HTTPProto  = "http"
    HTTPSProto = "https"
)

// pkg/constants/concurrency.go (1 个 int)
const (
    CommandConcurrency = 10  // 设备命令执行默认并发数(command_handler.go:58 + execution_handler.go:99 共用)
)
```

**审计源 (2026-09-03)**:
- 6 处业务超时 + 1 处 URL 协议 + 1 处 SNMP 端口 + 1 处并发数 = 9 处硬编码,分布于 5 个 handler + 1 个 service
- 用户讨论后扩展审计至 `internal/scheduler/` cron 内部 3 处 timeout(2 处已命名 const + 1 处内联)
- 总计 **10 个常量** 落地,涉及 **8 个文件**

**v1.29 全局原则 (D-PRINCIPLE)**:
- 行业最佳实践为唯一依据,允许任何形式的重构(D-05 零业务行为变更约束在本 phase 进一步强化:10 个常量值与现状 100% 一致,零行为变更)
- leaf const pkg 模式(纯 const 块,无函数),与 Phase 89 `pagination.go` 一致
- 不加 `Default` 前缀(与 Phase 89 D-08/D-09 删除 `KnowledgeDefaultPageSize` / `AccountPoolDefaultPageSize` 一致:`Default` 是"业务可覆盖"错误抽象)
- 模块特异常量是错误抽象 — 不应有 `KnowledgeCommandTimeout` / `LDAPAdCommandTimeout`

**不在本 phase**:
- CRUD 复用 base.Repository[T](Phase 91)
- 缓存层三处架构统一(Phase 92)
- config_backup 三处 TODO 闭环(Phase 93)
- 前端 API 工厂化(Phase 94)
- v1.28 阶段性收口(Phase 95)
- v1.30+ 候选:WS origin `localhost` / `127.0.0.1` 字面量抽象、scheduler 重试间隔、cron 错误退避策略

</domain>

<decisions>
## Implementation Decisions

### 常量包结构 (Area 1)

- **D-01:** 4 个 leaf const 文件按职责拆分(用户确认方案):
  - `pkg/constants/timeouts.go` — 6 个 `time.Duration` 常量(强类型,Go 标准库惯例对齐 net/http / database/sql / ldap)
  - `pkg/constants/ports.go` — `SNMPPort=161` int(网络默认值,仅 1 个常量)
  - `pkg/constants/protocol.go` — `HTTPProto="http"` + `HTTPSProto="https"` string(WS origin 协议白名单)
  - `pkg/constants/concurrency.go` — `CommandConcurrency=10` int(命令执行并发数,语义独立于 timeout)
  - 与 Phase 89 `pagination.go` 单文件相比,Phase 90 因常量类型多样(Duration/int/string)且语义维度不同(timeout/port/protocol/concurrency)拆 4 文件,grep 友好 + 后续添加 Metric/Prometheus 常量时不影响 timeout

### 类型选择 (Area 2)

- **D-02:** `timeouts.go` 用 `time.Duration` 强类型(用户确认方案):
  - 强类型安全:`time.Duration` 与 handler int 字段转换需 `int(time.Duration.Seconds())` 显式表达
  - LDAPConnTimeout 现为 `time.Second*30` 可直接复用
  - handler int 字段(Timeout=300)调用点需显式转换:`req.Timeout = int(constants.CommandExecTimeout.Seconds())`
  - ports.go / concurrency.go 用 int(与现有字段语义一致);protocol.go 用 string
  - 与 Go 标准库惯例一致(`net/http.Server.WriteTimeout`、`database/sql.SetConnMaxLifetime`、`ldap.DialURL`)

### Protocol helper 设计 (Area 3)

- **D-03:** 不抽 helper 函数,调用点裸拼接 `HTTPProto+"://"+host`(用户确认方案):
  - ws_notice_handler.go:47 现状为 `strings.HasPrefix(origin, "http://"+host)`,迁移后为 `strings.HasPrefix(origin, constants.HTTPProto+"://"+host)`
  - 与 Phase 89 D-19 leaf const 一致性优先(纯 const 块,无函数)
  - 代价:调用点需拼接字符串,但仅 1 处使用,影响面极小

### 测试策略 (Area 4)

- **D-04:** 全部对齐 Phase 89 锁值 + invariants 扫描(用户确认方案):
  - `pkg/constants/{timeouts,ports,protocol,concurrency}_test.go` 4 个锁值测试
    - timeouts 用 `map[string]time.Duration{...}`(因类型为 time.Duration,需自定义类型断言)
    - ports/protocol/concurrency 用 `map[string]int{...}` / `map[string]string{...}`(参考 Phase 89 `expectedPaginationValues`)
  - 命名遵循 `TestXxxConstantStability`(参考 `internal/utils/operlog/regression_test.go:TestOperTypeConstantStability`)
  - 全仓 invariants 扫描,查找以下字面量在业务代码(排除测试文件 `.planning/.coverage-baseline.md` + cron.go + pagination_test.go 等白名单)出现且同 file/func 未引用 `pkg/constants` 时上报 warning:
    - `time.Second * N` / `time.Minute * N` / `time.Hour * N`(timeout 字面量)
    - `Timeout = 300` / `Timeout = 60` / `Timeout = 30`(int 秒超时)
    - `SNMPPort = 161`(int 端口字面量)
    - `Concurrency = 10` / `Concurrency <= 0 { ... = 10 }`(并发数字面量)
    - `"http://"+` / `"https://"+`(协议拼接)
  - invariants 扫描 warning 不 fail,记录到测试日志供人工 triage(与 Phase 89 一致)

### 常量命名 (Area 5)

- **D-05:** 不加 `Default` 前缀(用户确认方案,与 Phase 89 D-08/D-09 删除 `KnowledgeDefaultPageSize` / `AccountPoolDefaultPageSize` 逻辑一致):
  - `CommandConcurrency=10`(不叫 `DefaultCommandConcurrency`)
  - `SNMPPort=161`(不叫 `DefaultSNMPPort`)
  - 理由:`Default` 前缀隐含"业务可覆盖"语义,但本项目所有调用点都引用常量,不存在"覆盖"语义

### Timeout 复用 (Area 6)

- **D-06:** `command_handler.go:61` 与 `execution_handler.go:104` 都为 `Timeout=300`(设备命令执行超时),共用同一常量 `CommandExecTimeout=300s`(用户确认方案):
  - 同一语义(timeout 限制单次设备命令执行总时长),不应拆为 `CommandExecTimeout` + `ExecutionTimeout` 两个常量
  - 与 Phase 89 D-08/D-09 删除"模块特异默认值"逻辑一致 — 业务语义相同的常量合并

### 审计范围扩展 (Area 7)

- **D-07:** scheduler/cron 内部 3 处 timeout 全部抽取(用户确认方案):
  - `internal/scheduler/ad_sync_tasks.go:42 adSchedulerSyncTimeout = 30 * time.Minute` → `pkg/constants/timeouts.go:ADSyncTimeout=30*time.Minute`
  - `internal/scheduler/cron.go:18 defaultShutdownTimeout = 5 * time.Second` → `pkg/constants/timeouts.go:SchedulerShutdownTimeout=5*time.Second`
  - `internal/scheduler/ad_sync_tasks.go:164` 内联 `1*time.Minute`(context.WithTimeout)→ `pkg/constants/timeouts.go:ADSyncTaskTimeout=1*time.Minute`
- **D-08:** scheduler 抽取后重命名为项目惯例名(PascalCase + 无前缀 `Default`),与 Phase 89 D-01 一致:
  - `adSchedulerSyncTimeout` → `ADSyncTimeout`
  - `defaultShutdownTimeout` → `SchedulerShutdownTimeout`
  - 新增 `ADSyncTaskTimeout`

### 业务行为变更清单 (D-PRINCIPLE 强化:零变更)

- **D-09:** 10 个常量值与现状 100% 一致,**零业务行为变更**(用户确认方案):
  - CommandExecTimeout=300s(现状 command_handler:61 + execution_handler:104 = 300 一致)
  - CommandReadTimeout=60s(现状 command_handler:98 = 60 一致)
  - LDAPConnTimeout=30s(现状 ad_ldap_client:74 `time.Second*30` 一致)
  - ADSyncTimeout=30m(现状 adSchedulerSyncTimeout 一致)
  - SchedulerShutdownTimeout=5s(现状 defaultShutdownTimeout 一致)
  - ADSyncTaskTimeout=1m(现状内联 1*time.Minute 一致)
  - SNMPPort=161(现状 discovery_handler:126 = 161 一致)
  - HTTPProto="http" / HTTPSProto="https"(ws_notice_handler:47 字符串拼接现状)
  - CommandConcurrency=10(现状 command_handler:58 + execution_handler:99 = 10 一致)

### 实施策略 (Area 9)

- **D-10:** 无 pilot,一次抽完 10 个常量 + 8 个文件调用点修改(用户确认方案):
  - 理由:Phase 90 变更都是字面量替换,不改业务行为,逻辑比 Phase 89 简单(Phase 89 有 2 个代码路径需 pilot 验证统一逻辑;Phase 90 仅替换字面量引用)
  - plan 拆分依据:按调用点文件分组(网络 handlers 1 个 plan + scheduler 1 个 plan + LDAP/WS 1 个 plan + 锁值测试 1 个 plan),约 3-4 个 plan

### 项目级同步 (Area 10)

- **D-11:** 本 phase 完成后同步在 `CLAUDE.md` `Pagination Constants Convention` 段后新增 `## Timeout/Port/Protocol Constants Convention` 段(用户确认方案):
  - 锁定纪律:所有 timeout/port/protocol/concurrency 字面量走 `pkg/constants/{timeouts,ports,protocol,concurrency}.go`
  - 引用示例:`req.Timeout = int(constants.CommandExecTimeout.Seconds())`(time.Duration→int 转换模式)
  - 禁止 inline 字面量:`time.Second * 30`、`SNMPPort = 161`、`Concurrency = 10`、`"http://"+` 等

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### v1.29 milestone 全局
- `.planning/ROADMAP.md` — v1.29 milestone § Phase 90 (Goal / 8 requirements / 2 plans / 4 Success Criteria)
- `.planning/REQUIREMENTS.md` — `TIMEOUTS-01..08` (本 phase 8 项需求)
- `.planning/PROJECT.md` — v1.29 Current Milestone 段,D-01..D-06 locked decisions + D-PRINCIPLE 行业最佳实践

### Phase 89 上下文(直接承上)
- `.planning/milestones/v1.29-phases/89-pagination-constants/89-CONTEXT.md` — Phase 89 D-01..D-19 全量决策,本 phase 复用其 leaf const + AST 锁值 + invariants 扫描模式
- `.planning/milestones/v1.29-phases/89-pagination-constants/89-DISCUSSION-LOG.md` — Phase 89 决策日志,作为本 phase 讨论风格参考
- `.planning/milestones/v1.29-phases/89-pagination-constants/89-CLAUDE.md` — Phase 89 收口时同步的 CLAUDE.md 段(Pagination Constants Convention),本 phase 90 收口时新增 Timeout/Port/Protocol 段参考其格式

### 现有常量相关代码
- `pkg/constants/pagination.go` — Phase 89 落地的 leaf const 先例(3 个 int 常量,纯 const 块,无函数)
- `pkg/constants/pagination_test.go` — Phase 89 AST 锁值测试先例(`expectedPaginationValues = map[string]int{...}` 模式 + `TestPaginationConstantStability` 命名)

### 现有 timeout/port/protocol/concurrency 业务代码
- `internal/api/v1/network/command_handler.go:58` — `req.Concurrency = 10`(待替换)
- `internal/api/v1/network/command_handler.go:61` — `req.Timeout = 300`(待替换)
- `internal/api/v1/network/command_handler.go:98` — `req.Timeout = 60`(QuickCommand 待替换)
- `internal/api/v1/network/execution_handler.go:99` — `req.Concurrency = 10`(待替换)
- `internal/api/v1/network/execution_handler.go:104` — `req.Timeout = 300`(待替换,与 command_handler:61 共用 `CommandExecTimeout`)
- `internal/api/v1/network/discovery_handler.go:126` — `req.SNMPPort = 161`(待替换)
- `internal/services/ad_ldap_client.go:74` — `c.conn.SetTimeout(time.Second * 30)`(待替换)
- `internal/api/v1/ws_notice_handler.go:47` — `strings.HasPrefix(origin, "http://"+host) || strings.HasPrefix(origin, "https://"+host)`(待替换为 `HTTPProto+"://"+host`)
- `internal/scheduler/ad_sync_tasks.go:42` — `const adSchedulerSyncTimeout = 30 * time.Minute`(待替换 + 重命名为 `ADSyncTimeout`)
- `internal/scheduler/ad_sync_tasks.go:164` — 内联 `context.WithTimeout(context.Background(), 1*time.Minute)`(待替换)
- `internal/scheduler/cron.go:18` — `const defaultShutdownTimeout = 5 * time.Second`(待替换 + 重命名为 `SchedulerShutdownTimeout`)

### AST 锁值测试参考 (内部 pattern)
- `internal/utils/operlog/regression_test.go` — `expectedOperTypeValues = map[string]int{...}` 模式 + `TestOperTypeConstantStability` 命名先例
- `internal/models/status_constants_test.go` — 多文件多 family AST scan 模式(本 phase invariants 扫描可参考)

### 项目级约束
- `CLAUDE.md` — `Status Value Convention` + `Pagination Constants Convention` (本 phase 新增 `Timeout/Port/Protocol Constants Convention` 段参考其格式)
- `CLAUDE.md` — `操作日志记录约定 (operlog convention)`(本 phase 业务行为变更零,无需触发 operlog)
- `CLAUDE.md` — `Compilation & Build Verification` (Phase 90 完成后 `go build ./...` 必须 0 错误)
- `CLAUDE.md` — `Testing` (Phase 90 完成后 `go test ./...` 必须 0 失败,既有 1688+ 测试不回归)

### 行业最佳实践参考
- Go 标准库 `time.Duration`:net/http.Server.WriteTimeout/ReadTimeout, database/sql.SetConnMaxLifetime, ldap.DialURL 均使用 time.Duration
- 项目 v1.27 落地先例:`internal/services/ad_ldap_client.go:74` 现状 `time.Second*30` 是 time.Duration

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/constants/pagination.go:DefaultCurrent/DefaultPageSize/MaxPageSize` — Phase 89 leaf const 先例,本 phase 完全复用其结构(纯 const 块,无函数,PascalCase 命名)
- `pkg/constants/pagination_test.go:TestPaginationConstantStability` — AST 锁值测试模板(`map[string]int{...}` + 同名测试函数 + `if got != expected { t.Errorf(...) }`),本 phase 4 个测试文件复用
- `internal/utils/operlog/regression_test.go:expectedOperTypeValues` — 双锁(常量值 + 常量数 = 25)模式参考
- `internal/models/status_constants_test.go:TestStatusConstantsStability` — AST scan 模板(`watchedStatusPrefixes` + 双向 assertion),本 phase invariants 扫描参考

### Established Patterns (D-PRINCIPLE 对齐依据)
- **Leaf const package pattern**: Go 标准库(`io/fs`, `net/http`, `time`)普遍采用 leaf pkg 纯 const,无业务函数。本 phase 严格遵守(D-01)
- **time.Duration 强类型 pattern**: Go 标准库 net/http/database/sql/ldap 全用 time.Duration,本 phase timeouts.go 全部用 time.Duration(D-02)
- **Single Source of Truth (SSOT)**: D-PRINCIPLE + D-06 要求所有 timeout/port/protocol/concurrency 字面量走 `pkg/constants`,无第二字面量源
- **零业务行为变更 pattern**: Phase 89 接受 6 处业务行为变更;Phase 90 因 D-09 决定所有常量值与现状 100% 一致,零变更,降低风险面
- **Atomic commit pattern**: Phase 75 (QUIRK 全修) 的 "修复 + 同 commit 翻转断言 + 回归用例 + 原子 commit" 五步法,本 phase 借鉴用于常量引入 + 锁值测试 + 调用点迁移

### Integration Points
- `internal/api/v1/network/command_handler.go` 替换 3 处(58/61/98) — 调用方:Dashboard UI 的命令下发页(无需改动,行为不变)
- `internal/api/v1/network/execution_handler.go` 替换 2 处(99/104) — 调用方:Dashboard UI 的配置执行页(无需改动,行为不变)
- `internal/api/v1/network/discovery_handler.go` 替换 1 处(126) — 调用方:Dashboard UI 的设备发现页(无需改动,行为不变)
- `internal/services/ad_ldap_client.go` 替换 1 处(74) — 调用方:AD 同步 scheduler、AD 账号管理(无需改动,行为不变)
- `internal/api/v1/ws_notice_handler.go` 替换 1 处(47) — 调用方:WebSocket 客户端(无需改动,行为不变)
- `internal/scheduler/ad_sync_tasks.go` 替换 2 处(42 常量声明 + 164 内联) — 调用方:scheduler 启动 + AD 同步任务(行为不变)
- `internal/scheduler/cron.go` 替换 1 处(18 常量声明) — 调用方:cron 引擎关闭逻辑(行为不变)

</code_context>

<specifics>
## Specific Ideas

### 命名一致性 (用户输入)
- **不加 `Default` 前缀**:与 Phase 89 D-08/D-09 刚删除 `KnowledgeDefaultPageSize=100` / `AccountPoolDefaultPageSize=20` 逻辑一致
- **重命名 scheduler 常量**:`adSchedulerSyncTimeout` → `ADSyncTimeout`、`defaultShutdownTimeout` → `SchedulerShutdownTimeout`
- **PascalCase + 无 snake_case**:与 Go 标准库惯例对齐

### 调用点 time.Duration→int 转换模式 (示例)
```go
// internal/api/v1/network/command_handler.go:58-62 迁移后
if req.Concurrency == 0 {
    req.Concurrency = constants.CommandConcurrency
}
if req.Timeout == 0 {
    req.Timeout = int(constants.CommandExecTimeout.Seconds())
}
if req.Timeout == 0 {  // QuickCommand
    req.Timeout = int(constants.CommandReadTimeout.Seconds())
}
```

### 协议拼接模式 (示例)
```go
// internal/api/v1/ws_notice_handler.go:47 迁移后
if strings.HasPrefix(origin, constants.HTTPProto+"://"+host) ||
    strings.HasPrefix(origin, constants.HTTPSProto+"://"+host) {
    return true
}
```

### Audit 扩展动机 (用户输入)
> "扩展到 scheduler/cron 超时"
- 原 ROADMAP 仅规划 6 处业务超时(handlers/services),审计时发现 scheduler 内部还有 3 处相关 timeout,这些 timeout 同样属于"业务硬编码",应一并治理
- 避免后续 Phase 92/93 等再回头补 scheduler 常量

### Zero Behavior Change 强化 (用户输入)
> "全部值与现状一致"
- Phase 89 接受 6 处业务行为变更(pagination binding/ad_domain/knowledge 100→10 等),Phase 90 因审计源(2026-09-03)报告的所有现有值已经合理,无需调整
- 降低风险面:零变更 = 零回归,部署即刻生效

</specifics>

<deferred>
## Deferred Ideas

### Workstream 同步问题 (非本 phase 范围,需后续 `/gsd:progress` 修复)
- `.planning/workstreams/milestone/STATE.md` 与 `.planning/workstreams/milestone/ROADMAP.md` 仍停留在 v1.27 SHIPPED,未同步到 v1.29(承袭 Phase 89 同一问题)
- `gsd-tools.cjs init phase-op 90` 默认查 workstream 路径返回 `phase_found: false`,本 discuss-phase 以 `.planning/ROADMAP.md` 为准继续
- 修复方式待 Phase 95 closeout 评估:workstream 是更新到 v1.29 还是重命名为 `v1.29-milestone`

### 未来 v1.29 其他 phase 引用
- **Phase 91 (CRUD 复用 base.Repository[T])**:独立的 CRUD 抽象,不影响
- **Phase 92 (缓存层统一)**:缓存层三处架构合并,可考虑把 `CacheServiceBase` 共同方法放 `pkg/query` 或新 `pkg/cache` 抽象
- **Phase 93 (config_backup 闭环)**:独立功能,与常量集中化无交叉

### 概念延展 (超出 Phase 90 范围)
- WS origin `localhost` / `127.0.0.1` 字面量抽象(ws_notice_handler.go:52):业务决策(开发环境白名单),应通过配置中心实现而非 const 化,v1.30+ 候选
- Scheduler 重试间隔 / cron 错误退避策略 / cleanup goroutine 间隔:scheduler 内部实现细节,非业务超时,v1.30+ 候选
- Prometheus metric 默认 bucket / histogram default / OTLP endpoint 常量:与 timeout 语义不同,v1.30+ 候选
- Phase 95 closeout 时,本 phase 的 invariants 扫描工具可考虑升级为 fail-on-hit(届时若全仓已无字面量,warning 应为 0,加 fail 也无影响)

### 业务配置化候选 (v1.30+)
- 把 SNMPPort / CommandConcurrency 做成可配置项(目前 const 化是正确的简化,但运维可能需要按环境调整)
- 把 timeout 做成可配置项(`time.Second * 30` → 可从 config.yaml 读取)以支持不同网络环境
- LDAP 客户端超时单独可配(不同 AD 服务器响应速度不同)

</deferred>

---

*Phase: 90-TIMEOUTS/PORT/PROTOCOL/CONCURRENCY 常量集中化*
*Context gathered: 2026-09-04*
