# Phase 73: P1 重要补齐 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-21
**Phase:** 73-P1 重要补齐
**Areas discussed:** Plan 拆分策略, services/monitor/oper_log_service.go 处理, rpa 公开路由覆盖, CI gate ratchet 目标值

---

## Plan 拆分策略

| Option | Description | Selected |
|--------|-------------|----------|
| **8 plans (1 包 1 plan)** | 每个 P1 包独立 plan(duty-handler / duty-service / knowledge-handler / knowledge-service / rpa-handler / vdi-handler / monitor-service / network-service)。粒度细,executor 单 plan 压力小,8 个原子 commit 清晰 | |
| **4 plans (按类型横切)** | handlers 简单包(duty+knowledge 538 stmts) / handlers 复杂包(rpa+vdi 910 stmts) / services 简单包(duty+knowledge+network 326 stmts) / services 中等包(monitor 485 stmts)。4 个 plan,planner 一次设计,executor 中等粒度 | ✓ |
| **3 plans (按业务域)** | duty + knowledge 域(737 stmts)/ rpa + vdi 域(910 stmts)/ monitor + network 域(612 stmts)。业务域聚类,但跨 service+handler 边界,planner 复杂 | |

**User's choice:** 4 plans(按类型横切),与 Phase 72 D-02 思路一致——按业务子模块/复杂度横切而非按层切。

**Notes:**
- Phase 72 D-02 走"按业务子模块横切"14 plan(services/system + api/v1/system 14 子模块),每个 plan 跨 service+handler+cache。
- Phase 73 包规模较小(8 包 2259 stmts vs Phase 72 14 子模块 8204 stmts),所以改为"按复杂度横切"4 plan,平衡粒度与 executor 压力。
- 8 plans 选项虽最细,但小 plan 间相互等待浪费 context,且 commit 节奏碎。
- 3 plans 选项"按业务域"跨 service+handler 边界,planner 设计复杂(测试模式不同),不采用。

---

## services/monitor/oper_log_service.go 处理

| Option | Description | Selected |
|--------|-------------|----------|
| **仍按 ≥70% 严格测 (推荐)** | Phase 73 SC#6 要求 services/monitor ≥70%(含 oper_log_service 0% 起点)。handler 层 71.2% 不等于 service 层 70%。oper_log_service 是 service 层公共 API,需直接测 | ✓ |
| **跳过 oper_log_service** | 认为 handler 层 71.2% 已覆盖 oper_log 业务路径,service 层可豁免。IMP-06 实际只测 login_log + server + cache_service,oper_log 维持 0%。这会偏离 Phase 72 D-04 严格均衡原则 | |

**User's choice:** 仍按 ≥70% 严格测。Phase 72 D-04 严格均衡原则优先。

**Notes:**
- oper_log_service.go 占 services/monitor 485 stmts 中约 100-150 stmts,豁免则 SC#6 难达。
- handler 层 71.2%(api/v1/monitor)只测 HTTP 包装,middleware 集成;service 层测 CRUD/Record* 私有方法、错误处理、批量操作。两层覆盖不重复,反而互补。
- Phase 72 已写 api/v1/monitor/oper_log_handler_test.go(若存在),其 DDL 可复用到 service 层 test setup。
- 测试模式选 ad_account 范本(glebarez sqlite + 真实 service),不选 portwrite(因 oper_log 需真实 DB 写入验证)。

---

## rpa 公开路由覆盖

| Option | Description | Selected |
|--------|-------------|----------|
| **与认证路由同套测试** | SetupPublicWorkerRouter 公开路由(register/heartbeat/progress)也走 glebarez sqlite + handler 测试。仅不调 JWT 中间件,handler 本身仍测。完整覆盖 7 个 rpa 子 handler | ✓ |
| **公开路由测试跳过** | 只测认证路由(SetupRPARouter 6 个子组 + 公开路由不测),rpa 覆盖率可能仅达 ~55-60%(公开路由 3 endpoint × ~30 stmts 缺口)。不严格满足 SC#3 ≥70% | |

**User's choice:** 与认证路由同套测试。公开路由不豁免。

**Notes:**
- rpa_router.go 有 2 个 router:`SetupPublicWorkerRouter` (无 auth,3 endpoint) + `SetupRPARouter` (有 auth,6 子组 ~30 endpoint)。
- 公开路由不豁免理由:(1) Phase 73 SC#3 要求 api/v1/rpa ≥70%,公开路由 ~50 stmts 不可丢;(2) Worker 是 RPA 系统关键组件(集群 worker 注册/心跳/进度上报),生产故障影响大;(3) 测试 setup 与认证路由共用,差异仅 JWT middleware,实现成本低。
- 测试 setup 需提供两套 router,公共 helper 提取避免重复(planner 决定具体方式)。
- 公开路由与认证路由用不同的 helper 初始化(publicRouter 无 auth),但 DB / SM4 cipher / 测试数据 factory 共享。

---

## CI gate ratchet 目标值

| Option | Description | Selected |
|--------|-------------|----------|
| **Ratchet 到实际值** | 跑全包测得新加权平均,直接 ratchet。预估: 21.5% + 8 个 P1 包贡献 ~5-6pp = ~26.5-27.5%。保守真实,phase 节奏稳 | ✓ |
| **Ratchet 到 ≥55% (Phase 73 SC#7)** | SC#7 说"Phase 73 完成后加权平均预估达 ≥55%"。但仅 8 个 P1 包(2259 stmts)无法贡献 +33pp(从 21.5 到 55)。SC#7 实际不可达,需修订 SC 或后续 Phase 73+74 合并 ratchet | |
| **Ratchet 到 ≥30% (Phase 72 实际)** | Phase 72 SC#7 当时写 ≥30%,实际 ratchet 到 21.5%。Phase 73 沿用保守目标 ≥30%。但需明确 Phase 73 SC#7 不达,deferred 到 Phase 74 | |

**User's choice:** Ratchet 到实际值。沿用 Phase 72 D-07 手动 ratchet 模式,跑全包测得新加权平均诚实 ratchet。

**Notes:**
- Phase 72 D-07 锁定手动 ratchet 到实际值模式,Phase 73 沿用。
- Phase 73 SC#7 写"预估达 ≥55%" 实际不可达:
  - 当前 21.5% × 43652 stmts = 9366 covered stmts
  - 8 个 P1 包 2259 stmts × 70% 目标 = 1581 covered stmts 新增(假设 100% 起点 0%)
  - 21.5% + 1581/43652 = 21.5% + 3.6% = 25.1% 加权(若其他包不动)
  - 实际可能因 Phase 72 子包继续微调、其他包不动,ratchet 预估 26-28%
- ROADMAP SC#7 实际达成 = 实际 ratchet 值(与 Phase 72 SC#7 当初写 ≥30% 实际 21.5% 同模式)。planner 可建议在 plan 阶段更新 ROADMAP.md SC#7 为"预估达 ~27-30%"。
- Phase 73 后加权平均 27-30%,Phase 74 推到 ≥70%(需 SCALE-01..03 大块补齐贡献 +42-43pp)。

---

## Claude's Discretion

下列子决策未深入讨论,按默认规则走(若 plan 阶段发现需调整,可修改):

- **测试 DB schema 管理**:沿用 `ad_account_handler_test.go` 模式——`setupTestDB(t)` 显式 `db.Exec("CREATE TABLE ...")` 建表,与 production migration 解耦
- **测试数据生成**:`uuid.NewString()` + 随机字符串 factory,不用共享 fixture 文件
- **Coverage profile 排除范围**:与 Phase 71 `check-coverage.sh` 一致——74 业务包加权,排除 `scripts/`/`tests/scripts/`/`node_modules/`/MAC 清理工具集/`crypto/gen_sm2_keys`/`migrations/`/cmd main/internal docs
- **CI run timeout**:Phase 73 全包测试预估 ~12-18 分钟(74 包 + 8 个新 P1 测试包),沿用 ci.yml `-timeout 15m` 不变;若超过则上调至 20m
- **未覆盖文件的容忍**:单文件覆盖率 < 50% 但不强制(比如某些 handler 内大段 panic recovery + 错误格式化代码难测,优先保证 ≥70% 加权)
- **新增 testify/mock 依赖**:沿用现有 `github.com/stretchr/testify v1.x` + `github.com/stretchr/mock` 版本,不升级不新增
- **Plan 73-02 rpa 公开路由 helper 抽取**:plan executor 自行决定是否提取共用 setup helper,避免重复

---

## Carry-forward Decisions (from Phase 71 + Phase 72)

| Decision | Source | Status |
|----------|--------|--------|
| **D-06** bash + awk coverage gate live on main at 21.5%(Phase 72 后 ratchet) | Phase 71+72 SC#1-#5 完成 | ✓ Already shipped |
| **D-07** 测试范本 ad_account 轻量(handler)+ portwrite 完整(service)直接沿用 | Phase 72 D-01 | ✓ Carried forward |
| **D-08** Plan 拆分按子模块/复杂度横切 | Phase 72 D-02 | ✓ Adapted (Phase 73 按 4 类型横切) |
| **D-09** 走真实中间件 + 真实加密(JWT + SM2+SM4) | Phase 72 D-03 | ✓ Carried forward |
| **D-10** per-sub-package ≥70% 严格均衡 | Phase 72 D-04 | ✓ Carried forward |
| **D-11** 手动 ratchet——execute plan 末尾原子 commit | Phase 72 D-07 | ✓ Carried forward |
| **D-12** 0 业务代码改动 | milestone v1.26 | ✓ Carried forward |
| **D-13** `.planning/coverage-baseline.md` append 新行 | Phase 71 D-03 | ✓ Carried forward |
| **D-14** 测试基建沿用 glebarez sqlite in-memory,无新 mock framework | milestone v1.26 D-04 | ✓ Carried forward |

---

## Deferred Ideas (已记入 CONTEXT.md `<deferred>`,此处仅索引)

- GOV-03 PR 增量 diff coverage ≥80%(Phase 74)
- SCALE-01 P2 大块增量(Phase 74)
- SCALE-02 中等覆盖包与工具包(Phase 74)
- SCALE-03 整体 ≥70%(Phase 74)
- FUT-02 分支覆盖率
- FUT-03 mutation testing
- FUT-04 PR 评论机器人
- FUT-01 辅助包强制覆盖
- `vladopajic/go-test-coverage` 引入(Phase 74 重评估)
- 集成测试 / E2E 测试(Phase 74+)
- 测试覆盖率徽章
- ROADMAP.md Phase 73 SC#7 修订(planner 可选)
- 已有 test 文件的 scope creep(不重写 Phase 72 已 SHIPPED 的 11 个 test)
