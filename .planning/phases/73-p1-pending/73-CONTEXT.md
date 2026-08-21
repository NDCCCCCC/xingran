---
phase: 73
phase_name: P1 重要补齐
slug: p1-pending
gathered: 2026-08-21
status: Ready for planning
---

# Phase 73: P1 重要补齐 - Context

**Gathered:** 2026-08-21
**Status:** Ready for planning

<domain>

## Phase Boundary

本阶段交付 **P1 重要业务模块的测试补齐**——8 个 IMP 需求覆盖 8 个 0% 测试包(4 个 handler 包 + 4 个 service 包),共 **2259 stmts** 跨 P1 业务路径加权补齐到 **≥70% per-package**。Phase 73 是 v1.26 milestone 的**承上启下**:Phase 72 已 SHIPPED 21.5% 基线(CORE-01..06 完成),本阶段把 P1 业务模块清零,Phase 74 收口到 ≥70% 整体加权平均。

**本阶段范围(IMP-01..06)**:
- IMP-01: `internal/api/v1/duty` handler 测试补齐(265 stmts,0.0% → ≥70%)
- IMP-02: `internal/api/v1/knowledge` handler 测试补齐(273 stmts,0.0% → ≥70%)
- IMP-03: `internal/api/v1/rpa` handler 测试补齐(612 stmts,0.0% → ≥70%,7 个子 handler)
- IMP-04: `internal/api/v1/vdi` handler 测试补齐(298 stmts,0.0% → ≥70%)
- IMP-05: `internal/services/duty` + `internal/services/knowledge` service 测试补齐(114 + 85 stmts,0.0% → ≥70%)
- IMP-06: `internal/services/monitor` + `internal/services/network` service 测试补齐(485 + 127 stmts,0.0% → ≥70%)

**不在本阶段(SCALE-01..03 属 Phase 74)**:
- SCALE-01 P2 大块增量(`services/rpa` 1865 stmts + `services/vdi` 1127 stmts + `api/v1/operations/asset/network`)
- SCALE-02 中等覆盖包与工具包选择性提升
- SCALE-03 最终整体 ≥70%(本阶段完成后预估 27-30%,Phase 74 推到 70%)
- GOV-03 PR 增量 diff coverage ≥80%(Phase 74)
- 辅助包强制覆盖(SCALE-02 选做)

**Phase 73 SC 完整清单(8 项)**:
1. `internal/api/v1/duty` ≥70%(265 stmts)
2. `internal/api/v1/knowledge` ≥70%(273 stmts)
3. `internal/api/v1/rpa` ≥70%(612 stmts)
4. `internal/api/v1/vdi` ≥70%(298 stmts)
5. `internal/services/duty` + `internal/services/knowledge` ≥70%(114 + 85 stmts)
6. `internal/services/monitor` + `internal/services/network` ≥70%(485 + 127 stmts)
7. Phase 73 完成后加权平均 ratchet 到新实际值(沿用 Phase 72 D-07,预估 27-30%)
8. 全部新测试零业务代码改动(沿用 D-08)

**Phase 73 与 Phase 72 关键差异**:
- Phase 72 有 11 个已有 test 文件作为基础;**Phase 73 是 8 个 0% 包,完全无基础**
- Phase 72 最大包 3483 stmts(services/system 14 子模块);Phase 73 最大包 612 stmts(api/v1/rpa 7 子 handler)
- Phase 73 rpa 公开路由(`SetupPublicWorkerRouter` 无 auth)需单独覆盖,与 Phase 72 ad_account 范本不同
- Phase 73 services/monitor 含 `oper_log_service`(与 Phase 72 已 SHIPPED api/v1/monitor handler 71.2% 路径重叠,本阶段 service 层仍需直接测)

</domain>

<decisions>

## Implementation Decisions

### D-01 (Area 1): Plan 拆分策略 — 4 plans 按类型横切

- **方案**: 8 个 P1 包按复杂度横切为 4 个 plan,平衡粒度与 executor 压力
- **4 个 plan 划分**:
  - **Plan 73-01**: handlers 简单包 — `api/v1/duty` (265 stmts) + `api/v1/knowledge` (273 stmts),共 538 stmts
    - 文件数最少(duty 2 / knowledge 2),单 plan 可独立完成
  - **Plan 73-02**: handlers 复杂包 — `api/v1/rpa` (612 stmts) + `api/v1/vdi` (298 stmts),共 910 stmts
    - rpa 7 子 handler(tasks/workers/executions/credentials/ai/flow)+ vdi 4 文件(server/vm/base)
  - **Plan 73-03**: services 简单包 — `services/duty` (114 stmts) + `services/knowledge` (85 stmts) + `services/network` (127 stmts),共 326 stmts
    - 全部为 cache_impl 单文件,与 portwrite 范本 1:1 匹配
  - **Plan 73-04**: services 中等包 — `services/monitor` (485 stmts,5 文件),单 plan 聚焦
    - 5 文件(cache_service/login_log_service/oper_log_service/server_service/types),跨 CRUD/读/统计,与 rpa service 路径不同
- **不采用**:
  - **8 plans (1 包 1 plan)**:粒度过细,executor 8 次 commit 节奏碎,小 plan 间相互等待浪费 context
  - **3 plans (按业务域)**:跨 service+handler 边界,planner 难以做"业务模块完整链路"设计,且 service/handler 测试模式不同

### D-02 (Area 2): 测试范本 — Phase 72 D-01 直接沿用

- **handler 包**:用 `ad_account_handler_test.go` 现有轻量范本(glebarez sqlite in-memory + 真实 service + 简单 mock struct + 表驱动 TC),与 Phase 72 一致
  - **不绕过 middleware**:JWT auth 真实 token 调真实 middleware;SM2+SM4 真实加密
  - **每个 test 独立初始化 helper**:SM4 cipher 实例 + JWT token factory + SM2 公私钥对(若 env 没设)
- **service 包**:
  - **portwrite 完整范本**(`port_write_service_test.go`): interface 断言 + testify/mock + 完全 mock 依赖不连真实 DB
  - **适配点**: services/monitor 5 文件非全部是 cache_impl,login_log_service / oper_log_service / server_service 需用真实 DB + 真实 service 调用(类似 ad_account 范本),不全部走 portwrite 纯 mock
  - **混合策略**:
    - `services/duty/duty_cache_impl.go` (114 stmts) + `services/knowledge/knowledge_cache_impl.go` (85 stmts) + `services/network/cache_impl.go` (127 stmts) → portwrite 纯 mock
    - `services/monitor/cache_service.go` (cache_impl 部分) → portwrite 纯 mock
    - `services/monitor/login_log_service.go` + `oper_log_service.go` + `server_service.go` → ad_account 范本(glebarez sqlite + 真实 DB + 真实 service 调用)
- **不允许引入**: 新 mock framework(沿用 D-06 milestone 锁定)
- **mockSM4Cipher 升级**: 沿用 Phase 72 D-03,不沿用 mockSM4Cipher 简单 stub,新测试改用真实 SM4 cipher

### D-03 (Area 3): services/monitor/oper_log_service.go 不豁免

- **仍按 ≥70% 严格测**:Phase 72 D-04 严格均衡原则,handler 层 71.2% 不替代 service 层 70%
- **oper_log_service 测试策略**:用 ad_account 范本(真实 DB + 真实 service 调用),与其他 monitor service 一致
  - 与 `api/v1/monitor/oper_log_handler.go` (Phase 72 已 71.2%)路径重叠但测试层级不同——service 层测 CRUD/Record* 方法,handler 层测 HTTP 包装
  - 双重覆盖不浪费:service 层测 service 私有方法 / 错误处理,handler 层测 middleware 集成
- **oper_log 表测试 DDL**:复用 Phase 72 `internal/api/v1/monitor/oper_log_handler_test.go`(若存在)中的 DDL 或新建一份完整 DDL
- **不豁免**:与 Phase 72 D-04 严格均衡原则一致;若豁免,SC#6 services/monitor ≥70% 实际不可达(oper_log 占 485 stmts 中约 100-150 stmts)

### D-04 (Area 4): api/v1/rpa 公开路由 — 与认证路由同套测试

- **SetupPublicWorkerRouter 也走 glebarez sqlite + handler 测试**
- **3 个公开 endpoint 测试**:
  - `POST /workers/register` (handler.Register)
  - `POST /workers/:id/heartbeat` (handler.Heartbeat)
  - `POST /workers/progress` (handler.Progress)
- **测试 setup 与认证路由共用**:唯一差异是不调 JWT auth middleware,其他(SM2+SM4 加密、handler 调用链、DB 验证)一致
- **不豁免**:Phase 73 SC#3 要求 api/v1/rpa ≥70%,公开路由占 ~50 stmts,豁免则达标难
- **rpa_router.go 双 router 处理**:测试 setup 需提供两套 router(SetupPublicWorkerRouter 与 SetupRPARouter),公共 helper 提取避免重复

### D-05 (Area 5): Ratchet 目标值 — 沿用 Phase 72 D-07 手动 ratchet 到实际值

- **ratchet 到新加权平均实际值**(预估 27-30%),不预先设 ≥55% 不切实际目标
- **沿用 Phase 72 模式**:execute plan 末尾原子 commit 更新 `.coverage-threshold` + `.planning/coverage-baseline.md`
- **commit message 格式**: `docs(73): coverage ratchet 21.5% → X.X%`
- **Phase 73 SC#7 修订**:ROADMAP.md 写的"预估达 ≥55%" 实际不可达——8 个 P1 包 2259 stmts 最多贡献 +5-6pp。Phase 73 完成后加权平均预估:
  - Phase 72 后 21.5% (9366/43652 stmts)
  - Phase 73 预计 +5-6pp → 约 26.5-27.5%
  - Phase 74 推到 ≥70%(需 SCALE-01..03 大块补齐贡献 +42-43pp)
- **ROADMAP SC#7 实际达成 = 实际 ratchet 值**(与 Phase 72 SC#7 当初写 ≥30% 实际 21.5% 同模式)

### Carry-forward from Phase 71 + Phase 72

- **D-06 (Phase 71)**: bash + awk coverage gate live on main,12.8% baseline → Phase 72 后 21.5%。Phase 73 任何 PR 不可降到 21.5% 以下
- **D-07 (Phase 72 D-01)**: 测试范本 ad_account(handler) + portwrite(service)直接沿用,无新设计
- **D-08 (Phase 72 D-02)**: services/system 按业务子模块横切——本阶段按复杂度横切(handler 简单/复杂 + service 简单/中等),适配 8 包规模
- **D-09 (Phase 72 D-03)**: 走真实中间件 + 真实加密(JWT + SM2+SM4),不复用 mockSM4Cipher
- **D-10 (Phase 72 D-04)**: per-sub-package ≥70% 严格均衡,不可妥协到子包 < 70%
- **D-11 (Phase 72 D-07)**: 手动 ratchet——Phase 73 execute plan 末尾原子 commit 更新 `.coverage-threshold` + `.planning/coverage-baseline.md`
- **D-12 (milestone D-04)**: 0 业务代码改动,测试暴露的确定性 bug 修复走最小 diff 单独说明
- **D-13 (Phase 71 D-03)**: `.planning/coverage-baseline.md` 继续 append 新行,quick scan SUMMARY.md 保持不变

### Claude's Discretion

下列子决策未深入讨论,按默认规则走(若 plan 阶段发现需调整,可修改):

- **测试 DB schema 管理**:沿用 `ad_account_handler_test.go` 模式——`setupTestDB(t)` 显式 `db.Exec("CREATE TABLE ...")` 建表,与 production migration 解耦
- **测试数据生成**:用 `uuid.NewString()` + 随机字符串 factory,不用共享 fixture 文件
- **Coverage profile 排除范围**:与 Phase 71 `check-coverage.sh` 一致——74 业务包加权,排除 `scripts/`/`tests/scripts/`/`node_modules/`/MAC 清理工具集/`crypto/gen_sm2_keys`/`migrations/`/cmd main/internal docs
- **CI run timeout**: Phase 73 全包测试预估 ~12-18 分钟(74 包 + 8 个新 P1 测试包),沿用 ci.yml `-timeout 15m` 不变;若超过则上调至 20m
- **未覆盖文件的容忍**: 单文件覆盖率 < 50% 但不强制(比如某些 handler 内大段 panic recovery + 错误格式化代码难测,优先保证 ≥70% 加权)
- **新增 testify/mock 依赖**:沿用现有 `github.com/stretchr/testify v1.x` + `github.com/stretchr/mock` 版本,不升级不新增
- **Plan 73-02 rpa 公开路由 helper 抽取**:plan executor 自行决定是否提取共用 setup helper,避免重复

</decisions>

<canonical_refs>

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 规划输入

- `.planning/REQUIREMENTS.md` — IMP-01..06 详细 stmts 数 + 起点 % + 目标 %
- `.planning/quick/260820-backend-test-coverage-scan/SUMMARY.md` — 起点状态 + 扫描方法 + 已知失败(都已修)
- `.planning/quick/260820-backend-test-coverage-scan/per-package-coverage.txt` — 76 行聚合,Phase 73 8 个 IMP 目标的精确起点
- `.planning/coverage-baseline.md` — 当前 21.5% baseline + Phase 72 后状态快照(8 个 P1 包仍 0%)
- `.planning/ROADMAP.md` — Phase 73 完整 SC + Notes(复杂模块可分多 plan 拆分)
- `.planning/PROJECT.md` §Current Milestone:v1.26 — D-04 测试基建锁定 + D-06 milestone 决策
- `.planning/STATE.md` — Phase 71 + Phase 72 已 SHIPPED + CI gate live

### Phase 72 决策范本(强制阅读)

- `.planning/phases/72-p0-core-supplement/72-CONTEXT.md` — **D-01..D-04 + D-05..D-09 决策完整记录**,Phase 73 直接沿用
- `.planning/phases/72-p0-core-supplement/72-DISCUSSION-LOG.md` — Phase 72 讨论过程,Phase 73 可参考讨论模式
- `.planning/phases/72-p0-core-supplement/72-01-PLAN.md` ~ `72-13-PLAN.md` — Phase 72 各 plan 结构与执行模式

### 测试范本(强制阅读)

- `internal/api/v1/system/ad_account_handler_test.go` — **D-02 handler 轻量范本**(glebarez sqlite + 真实 service + 表驱动 TC),所有 P1 handler 包参考
- `internal/services/portwrite/port_write_service_test.go` — **D-02 service 完整范本**(interface 断言 + testify/mock + 纯 mock),P1 cache_impl 类 service 参考
- `internal/services/portwrite/port_write_e2e_test.go` — portwrite e2e 范本(可选参考)
- `internal/utils/operlog/regression_test.go` — operlog AST 锁范本(本阶段不适用,作为"什么不做"参考)
- `internal/api/v1/monitor/oper_log_handler_test.go` — oper_log handler 测试(若 Phase 72 已写),DDL 可复用

### 目标包结构(planner 必须先 map)

- `internal/api/v1/duty/` — 265 stmts handler,2 文件(duty_handler.go / duty_router.go)
- `internal/api/v1/knowledge/` — 273 stmts handler,2 文件(handler.go / router.go)
- `internal/api/v1/rpa/` — 612 stmts handler,7 文件(ai/credential/execution/flow/task/worker/handler_helpers + rpa_router.go)
- `internal/api/v1/vdi/` — 298 stmts handler,4 文件(base/vdi_server/vm + vdi_server_router/vm_router)
- `internal/services/duty/` — 114 stmts service,1 文件(duty_cache_impl.go)
- `internal/services/knowledge/` — 85 stmts service,1 文件(knowledge_cache_impl.go)
- `internal/services/monitor/` — 485 stmts service,5 文件(cache_service/login_log_service/oper_log_service/server_service/types)
- `internal/services/network/` — 127 stmts service,1 文件(cache_impl.go)

### Phase 71 + Phase 72 已落地 CI gate(已 live on main)

- `.github/workflows/ci.yml` — backend job Test step 已带 `-coverprofile=coverage.out -covermode=atomic -count=1`
- `.coverage-threshold` — 当前 `21.5`,Phase 73 后 ratchet
- `.github/scripts/check-coverage.sh` — bash + awk 加权平均 gate,Phase 71 已扩展支持 per-sub-package grep

### 加密与中间件基础(D-09 必读)

- `pkg/crypto/` — SM2/SM4 实现(测试 setup 用)
- `pkg/middleware/` — auth/CORS/加密中间件(handler 测试走真实路径)
- `internal/core/security/jwt.go` — JWT 签名 / 验证(测试 token factory)
- `configs/config.dev.yaml` §security.sm4_key — 测试用 SM4 key 来源

### CLAUDE.md 关键约束(planner 必读)

- **测试基建 (D-04 milestone 锁定)**:沿用已有 glebarez sqlite in-memory,不引入新 mock framework
- **Status Value Convention**:测试中 status 常量引用 `internal/models/`,不写裸 0/1 字面量
- **API Response Format**:handler 测试断言 `code=0` (success) / `code != 0` (error)
- **operlog convention**:handler 测试应验证 `operlog.Record(c, ...)` 被调用

</canonical_refs>

<code_context>

## Existing Code Insights

### Reusable Assets

- **glebarez/sqlite v1.11.0 + glebarez/go-sqlite v1.21.2** (已在 go.mod) — 内存数据库,无需 AutoMigrate
- **stretchr/testify v1.x + stretchr/mock** (已在 go.mod) — assert/require + mock.Mock 嵌入模式
- **uuid.NewString()** — 测试数据生成标准
- **Phase 72 coverage gate 扩展点** — `check-coverage.sh` 的 awk 公式可复用,扩展为 per-sub-package grep
- **Phase 72 已 SHIPPED 的 test 范本 11 个**(`ad_account_handler_test.go` / `ad_dept_sync_handler_test.go` / `apikey_service_test.go` / `config_invalidation_test.go` / `config_statistics_test.go` / `dict_statistics_test.go` / `post_statistics_test.go` / `role_service_apperrors_test.go` / `settings_service_test.go` / `user_list_recursive_test.go` / `user_list_status_test.go`)——可借鉴测试结构

### Established Patterns (Phase 73 沿用)

- **handler 测试 DDL 模式** (ad_account_handler_test.go): `setupTestDB(t)` 显式 `db.Exec("CREATE TABLE ...")` 建表,与 production migration 解耦
- **service 测试 mock 模式** (portwrite): interface assertion 在文件顶部,compile-time 锁定 mockability contract
- **表驱动 TC 命名**: `TC1: 描述` / `TC2: 描述` 中文注释 + `TestXxx_yyy` 函数名
- **真实中间件 + mock cipher 升级**: D-09 要求新测试用真实 SM4 cipher,不复用 `mockSM4Cipher` 简单 stub
- **oper_log_service 测试模式** (若 Phase 72 已写 handler 测试):oper_log 表 DDL 可复用,service 层测 Record/List/GetByID/Delete/Clean/BatchDelete 方法

### Integration Points

- **CI gate** (Phase 71+72 已 live) — Phase 73 完成后 `.coverage-threshold` ratchet 到 ~27% (预估)
- **Coverage baseline 文件** (`.planning/coverage-baseline.md`) — Phase 73 后 append 一行新行(同 Phase 72 模式)
- **check-coverage.sh 扩展** — 加 8 个 P1 包 per-package grep 验证(D-10 要求每个包 ≥70%)
- **Tests in services/* 与 api/v1/* 命名** — `*_test.go` 同行同名,git 自动包含
- **rpa_router.go 双 router** — `SetupPublicWorkerRouter` (无 auth) + `SetupRPARouter` (有 auth),测试 setup 需两套 router,公共 helper 提取

### Phase 73 不需要做的事(明确范围)

- 不引入新 mock framework(沿用 testify/mock)
- 不引入新测试 runner(沿用 go test)
- 不改业务逻辑、API 契约、数据模型(测试暴露的 bug 走最小 diff 单独说明)
- 不实现 PR comment bot(FUT-04,Phase 74+)
- 不实现 mutation testing(FUT-03,Phase 74+)
- 不实现分支覆盖率(FUT-02,Phase 74+)
- 不实现 diff coverage(GOV-03,Phase 74)
- 不实现 SCALE-01..03(P2 增量与最终达标,Phase 74)
- 不豁免 services/monitor/oper_log_service(D-03 锁定)
- 不豁免 rpa 公开路由(D-04 锁定)
- 不重写 Phase 72 已落地的 11 个 test 文件(已有基础)
- 不改 Phase 71 已落地的 ci.yml / check-coverage.sh 主体(只扩展 gate 验证逻辑)

</code_context>

<specifics>

## Specific Ideas

**Phase 73 验证 checklist (executor 自验 + CI 验收)**:

1. `go test -count=1 ./internal/api/v1/duty/...` 退出 0,覆盖率 ≥70%
2. `go test -count=1 ./internal/api/v1/knowledge/...` 退出 0,覆盖率 ≥70%
3. `go test -count=1 ./internal/api/v1/rpa/...` 退出 0,覆盖率 ≥70%(含公开路由 3 endpoint)
4. `go test -count=1 ./internal/api/v1/vdi/...` 退出 0,覆盖率 ≥70%
5. `go test -count=1 ./internal/services/duty/... ./internal/services/knowledge/... ./internal/services/network/...` 退出 0,各覆盖率 ≥70%
6. `go test -count=1 ./internal/services/monitor/...` 退出 0,覆盖率 ≥70%(含 oper_log_service)
7. 全包 `go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...` exit 0,无失败
8. 加权平均覆盖率从 21.5% 上升到新实际值(预估 27-30%),`.coverage-threshold` ratchet 同步
9. CI 上 backend Coverage gate 仍 green(ratchet 后新阈值)
10. 0 业务代码改动(D-12 锁定)

**Phase 73 完成时执行的 4 个原子动作**(类似 Phase 72 模式):

1. 重跑全包 `go test -coverprofile` 得新加权平均
2. 跑 `bash .github/scripts/check-coverage.sh`(Phase 71 扩展版支持 per-sub-package)输出 per-package 表
3. 编辑 `.coverage-threshold` 写入新阈值(预估 ~27-30% 实际值)
4. 编辑 `.planning/coverage-baseline.md` 追加 Phase 73 后行(含 per-package 完整表 + 8 个 P1 包逐个 %)
5. 原子 commit (3 文件同 commit)

commit message 格式: `docs(73): coverage ratchet 21.5% → X.X%`

**Coverage-baseline.md Phase 73 后行 schema**(沿用 Phase 71/72 schema):

| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-21 | Phase 73 后 | ~27-30 | ~43652 | ~11800-13000 | ≤23 | TBD (amend) | gsd-execute-phase 73 | 21.5 | ~27-30 |

+ 8 个 P1 包逐个 % 表(新增列)

**Phase 73 Plan 拆分预期(4 plans)**:

```
Plan 73-01: handlers 简单包(duty + knowledge) — 538 stmts
Plan 73-02: handlers 复杂包(rpa + vdi)        — 910 stmts
Plan 73-03: services 简单包(duty + knowledge + network) — 326 stmts
Plan 73-04: services 中等包(monitor)           — 485 stmts
                                          ────────────
                                   total:   2259 stmts
```

每个 plan 内部按子包独立 test 文件,如 `internal/api/v1/duty/duty_handler_test.go` + `internal/api/v1/knowledge/handler_test.go`,git 自动包含。

</specifics>

<deferred>

## Deferred Ideas

下列想法讨论中被归类为本阶段范围外,**记在这里不丢失**:

- **GOV-03 PR 增量 diff coverage ≥80%** — 属 Phase 74 范围,本阶段不做
- **SCALE-01 P2 大块增量**(`services/rpa` 1865 stmts + `services/vdi` 1127 stmts + `api/v1/operations/asset/network`) — 属 Phase 74 范围
- **SCALE-02 中等覆盖包与工具包选择性提升** — 属 Phase 74 范围
- **SCALE-03 整体 ≥70%** — 属 Phase 74 范围(Phase 73 后预估 27-30%)
- **FUT-02 分支覆盖率** — Go 原生 coverprofile 仅语句级;引入第三方工具或 Go 官方分支支持后再议
- **FUT-03 mutation testing** — 覆盖率达标后的下一步,工具选型单独评估
- **FUT-04 PR 评论机器人** — Phase 74+ 候选
- **FUT-01 辅助包强制覆盖**(`internal/models/*`、`cmd` 入口、`internal/docs`) — 低价值高成本,Phase 74 重评估
- **`vladopajic/go-test-coverage` 引入** — Phase 74 启用 GOV-03 diff coverage 时重评估
- **集成测试 / E2E 测试** — Phase 73 只做单元 + 表驱动测试,集成测试留 Phase 74+
- **测试覆盖率徽章** — Phase 74+ 候选,FUT-04 范畴
- **ROADMAP.md Phase 73 SC#7 修订** — SC#7 当前写"预估达 ≥55%"实际不可达(8 个 P1 包 2259 stmts 仅能贡献 +5-6pp);planner 可建议在 plan 阶段更新 ROADMAP.md SC#7 为实际 ratchet 值
- **已有 test 文件的 scope creep** — Phase 72 已 SHIPPED 的 11 个 test 文件,Phase 73 不重写/重构这些,只在它们之上借鉴结构

</deferred>

---

*Phase: 73-P1 重要补齐*
*Context gathered: 2026-08-21 via discuss-phase*
