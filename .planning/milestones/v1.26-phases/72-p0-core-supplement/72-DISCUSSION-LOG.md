# Phase 72: P0 核心补齐 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-21
**Phase:** 72-P0 核心补齐
**Areas discussed:** 测试范本选择, services/system 拆分策略, Handler 测试中间件与加密, 覆盖目标粒度

---

## 测试范本选择

| Option | Description | Selected |
|--------|-------------|----------|
| **portwrite 完整 mock 范本** | interface + testify/mock + 纯 mock 依赖不连 DB | (部分采纳: 仅 service 包) |
| **ad_account_handler 轻量范本** | glebarez sqlite in-memory + 真实 service + 简单 mock struct + 表驱动 TC | ✓ (handler 包采纳) |
| **operlog AST 锁范本** | AST 锁公共 API(常量值/数量/函数签名/敏感关键词) | ✗ (Phase 72 不适用) |
| **operlog AST 锁 + glebarez sqlite hybrid** | handler 也走 AST 锁 | ✗ (boilerplate 太多,跨 6 包不实用) |

**User's choice:** 全用现有 ad_account_handler 轻量范本(handler) + portwrite 完整范本(service)。operlog AST 锁不适用 Phase 72。

**Notes:**
- **澄清轮**: 用户先看到 3 个选项需要澄清 4 个点(portwrite 范本详情 / operlog 范本详情 / 现有 handler 测试示例 / handler-vs-service 划分依据)。我读了三份实际代码:
  - `internal/services/portwrite/port_write_service_test.go` (139 行 + mockDeviceExecutor + mockCollectionSvc)
  - `internal/utils/operlog/regression_test.go` (AST 锁 25 个常量 + 11 个敏感关键词)
  - `internal/api/v1/system/ad_account_handler_test.go` (glebarez sqlite + mockSM4Cipher 简单 stub + TC 命名)
- **关键发现**: 仓库现有约定就是 handler 用 ad_account 范本 + service 用 portwrite 范本。operlog 范本是给治理工具包(operlog, status constants)用的,Phase 72 跨业务包不需要。
- **简化决策**: 不引入新范本,与仓库现有约定一致,planner 直接套用。

---

## services/system 拆分策略

| Option | Description | Selected |
|--------|-------------|----------|
| **按业务子模块横切(14 plan)** | user / role / dept / menu / dict / post / config / notice / settings / dashboard / apikey / profile / file / email_config,每个 plan 跨 service + handler + cache | ✓ |
| **按层横切(3 plan)** | service 1 plan / handler 1 plan / cache 1 plan | ✗ (粒度过粗,单个 plan 跨 3000+ stmts,executor context 压力大) |
| **按 stmts 均匀切(7-9 plan)** | 每 plan 700-900 stmts | ✗ (跨越业务边界,如 user+role 同 plan) |

**User's choice:** 按业务子模块横切。约 14 个 plan,每个 plan 包含 service + handler + cache(如适用)。

**Notes:**
- 已通过 `ls internal/services/system/` 确认 14 个子模块边界(已有 6 个 test 文件零散覆盖 apikey/config/dict/post/role/settings/user)
- 已通过 `ls internal/api/v1/system/` 确认 25 个 handler 文件,大部分有 1:1 对应 service
- **planner 注意**: 已有 9 个 test 文件作为基础(`apikey_service_test.go` / `config_invalidation_test.go` / `config_statistics_test.go` / `dict_statistics_test.go` / `post_statistics_test.go` / `role_service_apperrors_test.go` / `settings_service_test.go` / `user_list_recursive_test.go` / `user_list_status_test.go`)。新测试补齐时不要覆盖已有用例,而是补充未覆盖路径。

---

## Handler 测试中间件与加密策略

| Option | Description | Selected |
|--------|-------------|----------|
| **跳过 auth/CORS,SM2+SM4 用真实 cipher** | 现有 mockSM4Cipher 升级为真实 SM4 cipher,每个 test 独立 init helper | (类似,但保留 auth) |
| **不走任何中间件,直接调 handler 函数** | 跳过所有 middleware,test 代码里手动调 handler.Xxx(c) | ✗ (偏离 ad_account 范本) |
| **走真实中间件 + 真实加密** | JWT auth + SM2+SM4 真实路径,每个 test 独立 init helper | ✓ |

**User's choice:** 走真实中间件 + 真实加密(SM2+SM4 + auth),每个 test 独立初始化 helper。

**Notes:**
- 用户主动选择最重的端到端选项。可能因素:验证 handler 真实集成度;为 Phase 74+ 集成测试打基础。
- **mockSM4Cipher 升级路径**: 现有 `internal/api/v1/system/ad_account_handler_test.go` 的 `mockSM4Cipher` 是简单 stub。Phase 72 新测试不沿用,改用真实 SM4 cipher(setup 中初始化,SM4_KEY 来自 env 或 setup 中生成)。
- **已有测试保留**: 已有 ad_account_handler_test.go / ad_dept_sync_handler_test.go 走 mockSM4Cipher 不重写,避免 scope creep。
- **测试 setup 复杂度**: 每个 test 需初始化 SM4 cipher + JWT token factory + 可能 SM2 公私钥对。预估 setup 代码 +30%,运行时间 +5-10%。
- **风险**: SM2 公私钥生成 ~5-10s,若每个 test 都生成会慢。方案是 setup 中生成一次 + 全 test 共享(只读)。

---

## 覆盖目标粒度

| Option | Description | Selected |
|--------|-------------|----------|
| **加权 total ≥70%** | 只看 package 总体,子模块可相互补偿(高覆盖拉低覆盖) | ✗ (允许伪兑底) |
| **子包全部 ≥70%(严格均衡)** | 每个 plan 子模块都要 ≥70%,user 90% + role 30% 不达标 | ✓ |
| **子包 ≥50% 最低 + 总 ≥70%** | 平衡验收严格性与可执行性 | ✗ (用户选更严格) |

**User's choice:** 子包全部 ≥70%(严格均衡)。

**Notes:**
- **CI 验证脚本扩展**: 现有 `check-coverage.sh` 只算加权平均,Phase 72 完成后需扩展为:
  - 加权 total ≥30%(ratchet 到新值)
  - 每个 CORE 目标子模块 ≥70%
  - 检查方式: `go test -cover` 输出 per-package % 列,逐行 grep 对比
- **planner 风险**: 14 个 system 子包严格 ≥70% 可能拖慢进度。某些难点子包(如 dashboard 跨表查询)需更多测试代码。
- **兜底**: 若某个子包到 Phase 72 末尾仍 < 70%,不可妥协到 < 70% — 走 Phase 73/74 补齐,不能"差不多"放过。

---

## Claude's Discretion

下列子决策未深入讨论,按默认规则走(若 plan 阶段发现需调整,可修改):

- **测试 DB schema 管理**: 沿用 `ad_account_handler_test.go` 模式——`setupTestDB(t)` 显式 `db.Exec("CREATE TABLE ...")` 建表,与 production migration 解耦
- **测试数据生成**: `uuid.NewString()` + 随机字符串 factory,不用共享 fixture 文件
- **Coverage profile 排除范围**: 与 Phase 71 `check-coverage.sh` 一致(74 业务包加权)
- **CI run timeout**: 沿用 ci.yml `-timeout 15m`(Phase 72 全包预估 10-15 分钟)
- **未覆盖文件的容忍**: 单文件覆盖率 < 50% 但不强制(优先保证 ≥70% 加权)
- **新增 testify/mock 依赖**: 沿用现有 `github.com/stretchr/testify v1.x` + `github.com/stretchr/mock` 版本,不升级不新增

---

## Carry-forward Decisions (from Phase 71 + milestone v1.26)

| Decision | Source | Status |
|----------|--------|--------|
| **D-05** bash + awk coverage gate live on main at 12.8% | Phase 71 SC#1-#5 完成 | ✓ Already shipped |
| **D-06** 测试基建沿用 glebarez sqlite in-memory,无新 mock framework | milestone v1.26 D-04 | ✓ Carried forward |
| **D-07** 手动 ratchet — `.coverage-threshold` + `.planning/coverage-baseline.md` 原子 commit | Phase 71 D-04 | ✓ Carried forward |
| **D-08** 0 业务代码改动,测试暴露 bug 走最小 diff | milestone v1.26 | ✓ Carried forward |
| **D-09** quick scan SUMMARY.md 保持不变 | Phase 71 D-03 | ✓ Carried forward |

---

## Deferred Ideas (已记入 CONTEXT.md `<deferred>`,此处仅索引)

- GOV-03 PR 增量 diff coverage ≥80% (Phase 74)
- FUT-02 分支覆盖率
- FUT-03 mutation testing
- FUT-04 PR 评论机器人
- FUT-01 辅助包强制覆盖
- 已有 test 文件的 scope creep(不重写已存在 11 个 test)
- `vladopajic/go-test-coverage` 引入 (Phase 74 重评估)
- 集成测试 / E2E 测试 (Phase 74+)
- 测试覆盖率徽章
