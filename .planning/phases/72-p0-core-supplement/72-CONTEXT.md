---
phase: 72
phase_name: P0 核心补齐
slug: p0-core-supplement
gathered: 2026-08-21
status: Ready for planning
---

# Phase 72: P0 核心补齐 - Context

**Gathered:** 2026-08-21
**Status:** Ready for planning

<domain>

## Phase Boundary

本阶段交付**P0 核心业务链路的测试补齐**——6 个 CORE 需求覆盖 6 个包(4 个 handler 包 + 2 个 service 包),共 **8204 stmts** 跨 P0 业务路径加权补齐到 **≥70% per-package**。Phase 72 是 v1.26 milestone 的主体交付,Phase 71 的 CI coverage gate 现在才开始真正发挥作用。

**本阶段范围(GOV + CORE-01..06)**:
- CORE-01: `internal/api/v1/workorder` handler 测试补齐(297 stmts,0.0% → ≥70%)
- CORE-02: `internal/api/v1/monitor` handler 测试补齐(518 stmts,0.0% → ≥70%)
- CORE-03: `internal/api/v1/scheduler` handler 测试补齐(152 stmts,0.0% → ≥70%)
- CORE-04: `internal/api/v1/system` handler 测试补齐(3039 stmts,0.5% → ≥70%,按业务子模块横切 14 个 plan)
- CORE-05: `internal/services/workorder` 服务测试补齐(715 stmts,0.6% → ≥70%)
- CORE-06: `internal/services/system` 服务测试增量(3483 stmts,10.2% → ≥70%,按业务子模块横切 14 个 plan,与 CORE-04 配对)

**不在本阶段(SCALE-01..03 属 Phase 74)**:
- IMP-01..06 P1 重要模块清零(Phase 73)
- SCALE-01..03 大块增量与最终达标(Phase 74)
- PR 增量 diff coverage ≥80%(GOV-03 属 Phase 74)
- 工具包强制覆盖(SCALE-02 选做,Phase 74)
- 前端覆盖率治理(Phase 63 已 SHIPPED)

**Phase 72 SC 完整清单(8 项)**:
1. `internal/api/v1/workorder` ≥70%
2. `internal/api/v1/monitor` ≥70%
3. `internal/api/v1/scheduler` ≥70%
4. `internal/api/v1/system` ≥70%(加权平均,**每个子模块 ≥70%**)
5. `internal/services/workorder` ≥70%
6. `internal/services/system` ≥70%(加权平均,**每个子模块 ≥70%**)
7. Phase 72 后加权平均 12.8% → ≥30%,CI gate ratchet 上调至新实际值(手动 commit per D-07)
8. 全部新测试遵循本阶段 D-01 锁定的范本(handler 轻量 / service portwrite),零业务代码改动(测试暴露的确定性 bug 修复走最小 diff 单独说明)

</domain>

<decisions>

## Implementation Decisions

### D-01 (Area 1): 测试范本选择

- **handler 包(4 个)**: 用 `ad_account_handler_test.go` 现有轻量范本
  - **结构**: `glebarez sqlite in-memory` (`:memory:` 真实建表) + **真实 service 调用**(不走 mock service) + **简单 mock struct**(手动实现接口,如 `mockSM4Cipher` 升级为真实 SM4 cipher 见 D-03) + 表驱动 `TC1/TC2/TC3...` 命名
  - **优势**: 与现有 `internal/api/v1/system/ad_account_handler_test.go`、`ad_dept_sync_handler_test.go` 一致,最小重构
  - **不适用**: operlog AST 锁范本(那是治理工具包用的,Phase 72 跨业务包不需要)
- **service 包(2 个)**: 用 `portwrite` 完整范本
  - **结构**: compile-time interface assertion (`var _ portWriteExecutor = (*device.DeviceExecutor)(nil)`) + `testify/mock.Mock` 嵌入的 mock 结构体 + `m.Called(...)` 模式 + 完全 mock 依赖不连真实 DB
  - **参考**: `internal/services/portwrite/port_write_service_test.go` + `internal/services/portwrite/port_write_e2e_test.go`
- **不允许引入**: 新 mock framework(D-06 milestone 锁定)
- **现有 ad_account_handler_test.go 升级路径**: 见 D-03

### D-02 (Area 2): services/system + api/v1/system 拆分策略

- **方案**: 按业务子模块横切,每个 plan 覆盖一个子模块的完整链路(service 测试 + handler 测试 + cache 实现测试)
- **预期 ~14 个 plan**:
  - `user` (最大头: user_handler + user_import_handler + user_unlock_handler + user_service + user_cache_impl + user_list_*_test 已存在)
  - `role` (role_handler + role_service + role_cache_impl + role_service_apperrors_test 已存在)
  - `dept` (department_handler + department_service + department_cache_impl)
  - `menu` (menu_handler + fix_menu_handler + menu_service + menu_cache_impl)
  - `dict` (dict_handler + dict_service + dict_cache_impl + dict_statistics_test 已存在)
  - `post` (post_handler + post_service + post_cache_impl + post_statistics_test 已存在)
  - `config` (config_handler + config_service + config_cache_impl + config_invalidation_test + config_statistics_test 已存在)
  - `notice` (notice_handler + notice_user_handler + notice_service + notice_cache_impl)
  - `settings` (settings_handler + settings_service + settings_cache_impl + settings_service_test 已存在)
  - `dashboard` (dashboard_handler + dashboard_service)
  - `apikey` (apikey_handler + apikey_service + apikey_service_test 已存在)
  - `profile` (profile_handler + profile_service)
  - `file` (file_handler + file_service)
  - `email_config` (notification_config_handler + email_config_service + api_notification_config_service)
- **不采用**: 按层横切(service 1 plan / handler 1 plan / cache 1 plan,粒度过粗 executor 压力大); 按 stmts 均匀切(跨越业务边界)
- **planner 注意**: 已有测试文件(9 个)作为基础,新测试补齐时不要覆盖已有用例,而是补充未覆盖路径

### D-03 (Area 3): Handler 测试中间件与加密策略

- **走真实中间件**: handler 测试不绕过 auth / CORS / SM2+SM4 等生产中间件
  - **JWT auth**: 真实 secret key,生成有效 token 调真实 middleware
  - **SM2+SM4 加密**: request body 真实加密,handler 真实解密;response 真实加密,test 真实解密
  - **不绕过**: 不跳过 middleware 直接调 handler 函数(handler 调用方式仍按 D-01 范本,但中间件栈真实)
- **每个 test 独立初始化 helper**:
  - SM4 cipher 实例(用现有 `SM4_KEY` env var,测试 setup 初始化一次)
  - SM2 公私钥对(若 env 没设,setup 中生成临时 key)
  - JWT token factory(签名密钥 + 过期时间)
- **mockSM4Cipher 升级**: 现有 `internal/api/v1/system/ad_account_handler_test.go` 用的 `mockSM4Cipher` 是简单 stub,Phase 72 新测试**不沿用 mockSM4Cipher**,改用真实 SM4 cipher;已有测试保持不动(避免 scope creep)
- **权衡**: 端到端验证更真实,handler 调用链全栈覆盖; 测试 setup 复杂度 +30%,但每个 test 实际运行时间只 +5-10%(setup 共享)

### D-04 (Area 4): 覆盖目标粒度

- **每个子包必须 ≥70%** (严格均衡)
- **不允许** user 90% + role 30% 拼出 总体 70% 的伪兑底
- **CORE-04 / CORE-06 验收**: 每个子模块(user, role, dept, menu, dict, post, config, notice, settings, dashboard, apikey, profile, file, email_config)各自的 `go test -cover` 输出 ≥70.0%
- **CORE-01/02/03/05 验收**: 单一包 ≥70%(无子模块,直接检查)
- **CI 验证脚本扩展**: 现有 `check-coverage.sh` 只能算加权平均。Phase 72 完成后需扩展为:
  - 总体加权 ≥30%(Phase 72 后 ratchet 到的新阈值)
  - 每个 CORE 目标子模块 ≥70%
  - 检查方式: `go test -cover` 输出 per-package % 列,逐行 grep 对比

### Carry-forward from Phase 71 + milestone v1.26

- **D-05 (Phase 71 SC 完成)**: bash + awk coverage gate 已 live on main,12.8% baseline。Phase 72 任何 PR 不可降到 12.8% 以下
- **D-06 (milestone D-04)**: 测试基建沿用 `glebarez/sqlite` + `glebarez/go-sqlite` in-memory,无新 mock framework(但 `testify/mock.Mock` 已有依赖,可继续使用)
- **D-07 (Phase 71 D-04)**: 手动 ratchet——Phase 72 execute plan 末尾原子 commit 更新 `.coverage-threshold` + `.planning/coverage-baseline.md`,commit message `docs(72): coverage ratchet X.X% → Y.Y%`
- **D-08 (milestone)**: 0 业务代码改动,测试暴露的确定性 bug 修复走最小 diff 单独说明
- **D-09 (Phase 71 D-03)**: `.planning/coverage-baseline.md` 继续 append 新行,quick scan SUMMARY.md 保持不变

### Claude's Discretion

下列子决策未深入讨论,按默认规则走(若 plan 阶段发现需调整,可修改):

- **测试 DB schema 管理**: 沿用 `ad_account_handler_test.go` 模式——setupTestDB 显式建表 DDL(不依赖 AutoMigrate),与 production migration 解耦
- **测试数据生成**: 用 `uuid.NewString()` + 随机字符串 factory,不用共享 fixture 文件
- **Coverage profile 排除范围**: 与 Phase 71 `check-coverage.sh` 一致——74 业务包加权,排除 `scripts/`/`tests/scripts/`/`node_modules/`/MAC 清理工具集/`crypto/gen_sm2_keys`/`migrations/`/cmd main/internal docs
- **CI run timeout**: Phase 72 全包测试预估 ~10-15 分钟(74 包 + 14 子模块新测试),沿用 ci.yml `-timeout 15m` 不变
- **未覆盖文件的容忍**: 单文件覆盖率 < 50% 但不强制(比如某些 handler 内大段 panic recovery + 错误格式化代码难测,优先保证 ≥70% 加权)
- **新增 testify/mock 依赖**: 沿用现有 `github.com/stretchr/testify v1.x` + `github.com/stretchr/mock` 版本,不升级不新增

</decisions>

<canonical_refs>

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 规划输入

- `.planning/REQUIREMENTS.md` — CORE-01..06 详细 stmts 数 + 起点 % + 目标 %
- `.planning/quick/260820-backend-test-coverage-scan/SUMMARY.md` — 起点状态 + 扫描方法 + 已知失败(都已修)
- `.planning/quick/260820-backend-test-coverage-scan/per-package-coverage.txt` — 76 行聚合,Phase 72 6 个 CORE 目标的精确起点
- `.planning/coverage-baseline.md` — 当前 12.8% baseline + Phase 71 后状态快照
- `.planning/ROADMAP.md` — Phase 72 完整 SC + Notes(services/system 拆分提示 + operlog 范本引用)
- `.planning/PROJECT.md` §Current Milestone:v1.26 — D-04 测试基建锁定 + D-06 milestone 决策
- `.planning/STATE.md` — Phase 71 已 SHIPPED(`cfec2c4`) + CI gate live

### 测试范本(强制阅读)

- `internal/api/v1/system/ad_account_handler_test.go` — **D-01 handler 轻量范本**(glebarez sqlite + 真实 service + 表驱动 TC),Phase 72 handler 包参考
- `internal/services/portwrite/port_write_service_test.go` — **D-01 service 完整范本**(interface 断言 + testify/mock + 纯 mock),Phase 72 service 包参考
- `internal/services/portwrite/port_write_e2e_test.go` — portwrite e2e 范本参考(Phase 72 不强制使用,可选)
- `internal/utils/operlog/regression_test.go` — operlog AST 锁范本(D-01 明确 Phase 72 不适用,作为"什么不做"参考)

### 已有测试基础(planner 应避免重复覆盖)

- `internal/api/v1/system/ad_account_handler_test.go` (Phase 36)
- `internal/api/v1/system/ad_dept_sync_handler_test.go` (Phase 36)
- `internal/services/system/apikey_service_test.go`
- `internal/services/system/config_invalidation_test.go`
- `internal/services/system/config_statistics_test.go`
- `internal/services/system/dict_statistics_test.go`
- `internal/services/system/post_statistics_test.go`
- `internal/services/system/role_service_apperrors_test.go`
- `internal/services/system/settings_service_test.go`
- `internal/services/system/user_list_recursive_test.go`
- `internal/services/system/user_list_status_test.go`

### 目标包结构(planner 必须先 map)

- `internal/api/v1/workorder/` — 297 stmts handler,需补齐
- `internal/api/v1/monitor/` — 518 stmts handler,需补齐
- `internal/api/v1/scheduler/` — 152 stmts handler,需补齐
- `internal/api/v1/system/` — 3039 stmts handler,按 14 子模块拆 plan
- `internal/services/workorder/` — 715 stmts service,需补齐
- `internal/services/system/` — 3483 stmts service,按 14 子模块拆 plan

### Phase 71 已落地 CI gate(已 live on main)

- `.github/workflows/ci.yml` — backend job Test step 已带 `-coverprofile=coverage.out -covermode=atomic -count=1`
- `.coverage-threshold` — 当前 `12.8`,Phase 72 后 ratchet
- `.github/scripts/check-coverage.sh` — bash + awk 加权平均 gate,Phase 72 需扩展为子模块验证

### 加密与中间件基础(D-03 必读)

- `pkg/crypto/` — SM2/SM4 实现(测试 setup 用)
- `pkg/middleware/` — auth/CORS/加密中间件(handler 测试走真实路径)
- `internal/core/security/jwt.go` — JWT 签名 / 验证(测试 token factory)
- `configs/config.dev.yaml` §security.sm4_key — 测试用 SM4 key 来源

### CLAUDE.md 关键约束(planner 必读)

- **测试基建 (D-04 锁定)**: 沿用已有 glebarez sqlite in-memory,不引入新 mock framework
- **Status Value Convention**: 测试中 status 常量引用 `internal/models/`,不写裸 0/1 字面量
- **API Response Format**: handler 测试断言 `code=0` (success) / `code != 0` (error)
- **operlog convention**: handler 测试应验证 `operlog.Record(c, ...)` 被调用

</canonical_refs>

<code_context>

## Existing Code Insights

### Reusable Assets

- **glebarez/sqlite v1.11.0 + glebarez/go-sqlite v1.21.2** (已在 go.mod) — 内存数据库,无需 AutoMigrate
- **stretchr/testify v1.x + stretchr/mock** (已在 go.mod) — assert/require + mock.Mock 嵌入模式
- **uuid.NewString()** — 测试数据生成标准
- **Phase 71 coverage gate 扩展点** — `check-coverage.sh` 的 awk 公式可复用,扩展为 per-sub-package grep

### Established Patterns (Phase 72 沿用)

- **handler 测试 DDL 模式** (ad_account_handler_test.go): `setupTestDB(t)` 显式 `db.Exec("CREATE TABLE ...")` 建表,与 production migration 解耦
- **service 测试 mock 模式** (portwrite): interface assertion 在文件顶部,compile-time 锁定 mockability contract
- **表驱动 TC 命名**: `TC1: 描述` / `TC2: 描述` 中文注释 + `TestXxx_yyy` 函数名
- **真实中间件 + mock cipher 升级**: D-03 要求新测试用真实 SM4 cipher,不复用 `mockSM4Cipher` 简单 stub

### Integration Points

- **CI gate** (Phase 71 已 live) — Phase 72 完成后 `.coverage-threshold` ratchet 到 ~30%(预估)
- **Coverage baseline 文件** (`.planning/coverage-baseline.md`) — Phase 72 后 append 一行新行(同 Phase 71 模式)
- **check-coverage.sh 扩展** — 加 per-sub-package grep 验证(D-04 要求每个子模块 ≥70%)
- **Tests in services/system 与 api/v1/system 命名** — `*_test.go` 同行同名,git 自动包含

### Phase 72 不需要做的事(明确范围)

- 不引入新 mock framework(沿用 testify/mock)
- 不引入新测试 runner(沿用 go test)
- 不改业务逻辑、API 契约、数据模型(测试暴露的 bug 走最小 diff 单独说明)
- 不实现 PR comment bot(FUT-04,Phase 74+)
- 不实现 mutation testing(FUT-03,Phase 74+)
- 不实现分支覆盖率(FUT-02,Phase 74+)
- 不实现 diff coverage(GOV-03,Phase 74)
- 不改 Phase 71 已落地的 ci.yml / check-coverage.sh 主体(只扩展 gate 验证逻辑)

</code_context>

<specifics>

## Specific Ideas

**Phase 72 验证 checklist (executor 自验 + CI 验收)**:

1. `go test -count=1 ./internal/api/v1/workorder/...` 退出 0,覆盖率 ≥70%
2. `go test -count=1 ./internal/api/v1/monitor/...` 退出 0,覆盖率 ≥70%
3. `go test -count=1 ./internal/api/v1/scheduler/...` 退出 0,覆盖率 ≥70%
4. `go test -count=1 ./internal/services/workorder/...` 退出 0,覆盖率 ≥70%
5. 对 `internal/api/v1/system/` 14 子模块,每个单独 `go test -count=1 ./internal/api/v1/system/{子模块相关文件路径}` 退出 0,覆盖率 ≥70%
6. 对 `internal/services/system/` 14 子模块,每个单独 `go test -count=1 ./internal/services/system/{子模块相关文件路径}` 退出 0,覆盖率 ≥70%
7. 全包 `go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...` exit 0,无失败
8. 加权平均覆盖率从 12.8% 上升到 ≥30%(预估),`.coverage-threshold` ratchet 同步
9. CI 上 backend Coverage gate 仍 green(ratchet 后新阈值 ≥30%)

**Phase 72 完成时执行的 4 个原子动作**(类似 Phase 71 Task 8 amend):

1. 重跑全包 `go test -coverprofile` 得新加权平均
2. 跑 `bash .github/scripts/check-coverage.sh`(扩展后支持 per-sub-package)输出 per-package 表
3. 编辑 `.coverage-threshold` 写入新阈值(≥30% 实际值)
4. 编辑 `.planning/coverage-baseline.md` 追加 Phase 72 后行(含 per-package 完整表 + 14 子模块逐个 %)
5. 原子 commit (3 文件同 commit)

commit message 格式: `docs(72): coverage ratchet 12.8% → X.X%`

**Coverage-baseline.md Phase 72 后行 schema**(沿用 Phase 71 schema):

| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-21 | Phase 72 后 | ≥30 | ~43652 | ~13000 | ≤10 | TBD (amend) | gsd-execute-phase 72 | 12.8 | ≥30 |

+ 14 子模块逐个 % 表(新增列)

</specifics>

<deferred>

## Deferred Ideas

下列想法讨论中被归类为本阶段范围外,**记在这里不丢失**:

- **GOV-03 PR 增量 diff coverage ≥80%** — 属 Phase 74 范围,本阶段不做
- **FUT-02 分支覆盖率** — Go 原生 coverprofile 仅语句级;引入第三方工具或 Go 官方分支支持后再议
- **FUT-03 mutation testing** — 覆盖率达标后的下一步,工具选型单独评估
- **FUT-04 PR 评论机器人** — Phase 74+ 候选
- **FUT-01 辅助包强制覆盖**(`internal/models/*`、`cmd` 入口、`internal/docs`) — 低价值高成本,Phase 74 重评估
- **Phase 72 内已存在测试文件的 scope creep** — 已有 11 个 test 文件(ad_account_handler_test 等),Phase 72 不重写/重构这些,只在它们之上补充未覆盖路径。如果发现现有测试有 bug,走最小 diff 修复 + commit message 标注
- **`vladopajic/go-test-coverage` 引入** — Phase 74 启用 GOV-03 diff coverage 时重评估
- **集成测试 / E2E 测试** — Phase 72 只做单元 + 表驱动测试,集成测试留 Phase 74+
- **测试覆盖率徽章** — Phase 74+ 候选,FUT-04 范畴

</deferred>

---

*Phase: 72-P0 核心补齐*
*Context gathered: 2026-08-21 via discuss-phase*
