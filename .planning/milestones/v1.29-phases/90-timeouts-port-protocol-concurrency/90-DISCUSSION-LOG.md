# Phase 90: TIMEOUTS/PORT/PROTOCOL/CONCURRENCY 常量集中化 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-04
**Phase:** 90-TIMEOUTS/PORT/PROTOCOL/CONCURRENCY 常量集中化
**Areas discussed:** 常量粒度、超时类型、Protocol helper、验证策略、常量命名、Timeout 复用、审计范围、零变更原则、Scheduler 范围、重命名、实施路径、CLAUDE 同步

---

## Area 1: 常量粒度(拆几个文件?)

| Option | Description | Selected |
|--------|-------------|----------|
| 按职责拆 4 个文件 | 4 个 leaf const 文件:timeouts.go + ports.go + protocol.go + concurrency.go;grep 友好 | ✓ |
| 合并为 timeouts.go 一个文件 | 仅 timeouts.go + protocol.go(ROADMAP 原计划),port+concurrency 合并到 timeouts.go | |
| 单个 combined.go 大文件 | 合并为 timeouts.go 1 个文件,丢 4 个分类 | |

**User's choice:** 按职责拆 4 个文件(与 D-PRINCIPLE "合理抽象"一致)
**Notes:** 选项描述里实际是 4 个文件(timeouts/ports/protocol/concurrency),用户接受 4 文件方案。与 Phase 89 `pagination.go` 单文件不同,Phase 90 因常量类型多样(Duration/int/string)且语义维度不同(timeout/port/protocol/concurrency)拆 4 文件。

---

## Area 2: 超时类型(time.Duration vs int 秒)

| Option | Description | Selected |
|--------|-------------|----------|
| time.Duration 强类型 | 与 Go 标准库 net/http / database/sql / ldap 一致;LDAP 现 `time.Second*30` 可直接复用;handler int 字段需显式转换 | ✓ |
| int 秒原始数字 | 与现状 handler int 字段一致,零转换;LDAP 需转换 | |
| 混合:timeouts.go Duration + 其他 int | timeouts.go time.Duration,ports/protocol/concurrency int/string | |

**User's choice:** time.Duration 强类型
**Notes:** Go 标准库惯例对齐(`net/http.Server.WriteTimeout` / `database/sql.SetConnMaxLifetime` / `ldap.DialURL`)。handler 调用点需 `int(constants.CommandExecTimeout.Seconds())` 显式转换,与 Phase 89 D-15 修订 binding `max=100` 删除的"消除硬编码"哲学一致。

---

## Area 3: Protocol helper(裸 const vs helper 函数)

| Option | Description | Selected |
|--------|-------------|----------|
| helper 函数 BuildOrigin | 在 protocol.go 加 `func BuildOrigin(scheme, host) string`;调用点 `BuildOrigin(HTTPProto, host)` | |
| 裸 const 拼接 | 仅 HTTPProto + HTTPSProto;调用点 `HTTPProto+"://"+host`;与 Phase 89 leaf const 一致 | ✓ |

**User's choice:** 裸 const 拼接(与 Phase 89 一致)
**Notes:** Phase 89 D-19 修订已确立 leaf const pkg 纯 const 习惯,helper 函数违反此惯例。仅 1 处使用,影响面极小。

---

## Area 4: 验证策略(测试锁值对齐程度)

| Option | Description | Selected |
|--------|-------------|----------|
| 全部对齐 Phase 89 | 锁值 + invariants 扫描(全仓 .go 查找 time.Second*N / Timeout=300 等字面量) | ✓ |
| 仅锁值不扫描 | 只锁值不扫描;避免重试/计时器场景误报 | |
| 抽 invariant 扫描为公共工具 | 抽到 pkg/query/literalscan.go,pagination_test.go + timeouts_test.go 都调用 | |

**User's choice:** 全部对齐 Phase 89
**Notes:** 与 Phase 89 一致 + warning 不 fail + 记录到测试日志供人工 triage。Phase 95 closeout 时考虑升级为 fail-on-hit。

---

## Area 5: 常量命名(加不加 Default 前缀)

| Option | Description | Selected |
|--------|-------------|----------|
| 不加 Default 前缀 | `CommandConcurrency=10` + `SNMPPort=161`;与 Phase 89 D-08/D-09 删除 KnowledgeDefaultPageSize 逻辑一致 | ✓ |
| 保留 Default 前缀 | `DefaultCommandConcurrency` + `DefaultSNMPPort`(ROADMAP 原计划) | |
| 不加 Default 但加注释 | 表面一致但实质同 B | |

**User's choice:** 不加 Default 前缀(与 Phase 89 刚删除过的模块特异默认值逻辑一致)
**Notes:** `Default` 前缀隐含"业务可覆盖"语义,但本项目所有调用点都引用常量,不存在"覆盖"语义。Phase 89 D-08/D-09 删除 `KnowledgeDefaultPageSize=100` / `AccountPoolDefaultPageSize=20` 已确立无 `Default` 惯例。

---

## Area 6: Timeout 复用(command_handler:61 + execution_handler:104 共用?)

| Option | Description | Selected |
|--------|-------------|----------|
| 共用一个常量 CommandExecTimeout | 两处 Timeout=300 语义相同(设备命令执行超时),合并为同一常量 | ✓ |
| 保留两个独立常量名 | CommandExecTimeout + ExecutionTimeout(虽然语义一样) | |
| 复用但调用点不同名常量 | 常量名拆,值都是 300s | |

**User's choice:** 共用一个常量 CommandExecTimeout=300s
**Notes:** 与 Phase 89 D-08/D-09 删除"模块特异默认值"逻辑一致。同一语义不应拆为多个常量名。

---

## Area 7: 审计范围(扩展 scheduler/cron?)

| Option | Description | Selected |
|--------|-------------|----------|
| 仅 ROADMAP 6 处 | 严格按 ROADMAP 划定的 6 处(command_handler 58/61/98 + execution_handler 104 + ad_ldap_client 74 + discovery_handler 126 + ws_notice 47) | |
| 扩展到 scheduler/cron 超时 | 同 phase 抽完,发现 N 处额外超时 | ✓ |
| 实际叫 audit 报告引用 | PROJECT.md 明确"仅限审计报告提及的 6 处",不变更范围 | |

**User's choice:** 扩展到 scheduler/cron 超时
**Notes:** scheduler 内部 3 处 timeout(2 处已命名 const + 1 处内联)同样属于"业务硬编码",应一并治理。避免后续 Phase 92/93 等再回头补 scheduler 常量。

---

## Area 8: 零变更原则(常量值与现状是否一致?)

| Option | Description | Selected |
|--------|-------------|----------|
| 全部值与现状一致 | 零业务行为变更;CommandExecTimeout=300s/CommandReadTimeout=60s/LDAPConnTimeout=30s/SNMPPort=161/CommandConcurrency=10 与现状一致 | ✓ |
| 个别值可调整 | LDAPConnTimeout=30s 可能是拍脑袋决定,可与行业最佳实践对齐调为不同值;需列变更清单并用户逐项接受 | |

**User's choice:** 全部值与现状一致(零业务行为变更)
**Notes:** Phase 89 接受 6 处业务行为变更(pagination binding/ad_domain/knowledge 100→10 等);Phase 90 因审计源(2026-09-03)报告的所有现有值已经合理,无需调整。降低风险面:零变更 = 零回归,部署即刻生效。

---

## Area 9: Scheduler 范围(具体抽哪些?)

| Option | Description | Selected |
|--------|-------------|----------|
| 3 处都抽 | adSchedulerSyncTimeout=30m + defaultShutdownTimeout=5s + ad_sync_tasks.go:164 内联 1*time.Minute | ✓ |
| 仅抽 adSchedulerSyncTimeout | 仅 AD 同步上下文;shutdownTimeout 是 scheduler 内部实现,1*time.Minute 是 ad_sync 业务逻辑 | |
| 仅抽外部使用的 2 处 | 抽 2 处,不抽内联 1*time.Minute | |

**User's choice:** 3 处都抽
**Notes:** `internal/scheduler/ad_sync_tasks.go:42 adSchedulerSyncTimeout = 30 * time.Minute` + `internal/scheduler/cron.go:18 defaultShutdownTimeout = 5 * time.Second` + `internal/scheduler/ad_sync_tasks.go:164` 内联 `context.WithTimeout(..., 1*time.Minute)`。覆盖 AD 同步任务完整 timeout 链。

---

## Area 10: 重命名(scheduler 常量抽取后)

| Option | Description | Selected |
|--------|-------------|----------|
| 重命名为项目惯例名 | PascalCase + 无前缀 Default;`adSchedulerSyncTimeout` → `ADSyncTimeout`;`defaultShutdownTimeout` → `SchedulerShutdownTimeout` | ✓ |
| 保留原名仅迁移 | 表面一致但与 Phase 89 D-01 命名习惯不一致 | |

**User's choice:** 重命名为项目惯例名
**Notes:** 与 Phase 89 D-01 命名习惯一致。PascalCase + 无前缀 `Default`。

---

## Area 11: 实施路径(pilot 先行?)

| Option | Description | Selected |
|--------|-------------|----------|
| 无 pilot 一次抽完 | 10 个常量 + 8 个文件调用点修改;理由:变更都是字面量替换,不改行为,逻辑比 Phase 89 简单 | ✓ |
| 与 Phase 89 一致 pilot 先行 | 先抽 1-2 个常量+pilot 验证后批量迁移 | |

**User's choice:** 无 pilot 一次抽完
**Notes:** Phase 90 变更都是字面量替换,不改业务行为。Phase 89 有 2 个代码路径需 pilot 验证统一逻辑;Phase 90 仅替换字面量引用,无逻辑复杂度。plan 拆分依据:按调用点文件分组(网络 handlers 1 个 plan + scheduler 1 个 plan + LDAP/WS 1 个 plan + 锁值测试 1 个 plan),约 3-4 个 plan。

---

## Area 12: CLAUDE 同步

| Option | Description | Selected |
|--------|-------------|----------|
| 同步 CLAUDE.md 加 Timeout 段 | 在 Pagination Constants Convention 段后新增 `## Timeout/Port/Protocol Constants Convention` 段 | ✓ |
| 不写 CLAUDE.md | 仅写 CONTEXT.md/SUMMARY;减少文档变更面 | |

**User's choice:** 同步 CLAUDE.md 加 Timeout 段
**Notes:** 与 Phase 89 收口时同步 CLAUDE.md 的惯例一致(89-03 commit `8bb6007`)。锁定纪律:所有 timeout/port/protocol/concurrency 字面量走 `pkg/constants/{timeouts,ports,protocol,concurrency}.go`。

---

## Claude's Discretion

无。所有 12 个灰色区域均由用户通过 AskUserQuestion 明确选择。

---

## Deferred Ideas

### Workstream 同步问题 (非本 phase 范围,需后续 `/gsd:progress` 修复)
- `.planning/workstreams/milestone/STATE.md` 与 `.planning/workstreams/milestone/ROADMAP.md` 仍停留在 v1.27 SHIPPED,未同步到 v1.29
- `gsd-tools.cjs init phase-op 90` 默认查 workstream 路径返回 `phase_found: false`,本 discuss-phase 以 `.planning/ROADMAP.md` 为准继续
- 修复方式待 Phase 95 closeout 评估:workstream 是更新到 v1.29 还是重命名为 `v1.29-milestone`

### 未来 v1.29 其他 phase 引用
- Phase 91 (CRUD 复用 base.Repository[T]):独立的 CRUD 抽象,不影响
- Phase 92 (缓存层统一):缓存层三处架构合并,可考虑把 `CacheServiceBase` 共同方法放 `pkg/query` 或新 `pkg/cache` 抽象
- Phase 93 (config_backup 闭环):独立功能,与常量集中化无交叉

### 概念延展 (超出 Phase 90 范围)
- WS origin `localhost` / `127.0.0.1` 字面量抽象(ws_notice_handler.go:52):业务决策,应通过配置中心实现,v1.30+ 候选
- Scheduler 重试间隔 / cron 错误退避策略 / cleanup goroutine 间隔:scheduler 内部实现细节,v1.30+ 候选
- Prometheus metric 默认 bucket / histogram default / OTLP endpoint 常量:与 timeout 语义不同,v1.30+ 候选
- Phase 95 closeout 时,本 phase 的 invariants 扫描工具可考虑升级为 fail-on-hit

### 业务配置化候选 (v1.30+)
- 把 SNMPPort / CommandConcurrency 做成可配置项(目前 const 化是正确的简化,但运维可能需要按环境调整)
- 把 timeout 做成可配置项(`time.Second * 30` → 可从 config.yaml 读取)以支持不同网络环境
- LDAP 客户端超时单独可配(不同 AD 服务器响应速度不同)
