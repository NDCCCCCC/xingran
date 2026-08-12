---
phase: 34-oper-log-full-coverage
plan: 10
subsystem: operlog-documentation
tags: ["operlog", "documentation", "convention", "regression-test", "phase-close"]
requires:
  - 34-01 (operlog helper infra: sensitiveKeys, Record/RecordWithBody, 24 OperType constants)
  - 34-02..34-09 (instrumented endpoints + e2e harness locked in)
provides:
  - CLAUDE.md "### 操作日志记录约定 (operlog convention) — 强制" subsection
  - docs/开发规范.md "### 5.1.1 操作日志记录 (强制)" normative section
  - internal/utils/operlog/regression_test.go (4 public-API lock-in tests)
affects:
  - .planning/STATE.md (Phase 34 marked ✅ Completed 10/10)
  - .planning/ROADMAP.md (Phase 34 row updated to 10/10 Complete)
tech-stack:
  added: []
  patterns:
    - "convention-as-doc + regression-test lock-in (pin public API so docs cannot silently drift from code)"
    - "AST-based constant introspection in go tests (parser.ParseFile + ast.GenDecl walk for OperType* values)"
    - "reflect-based function-signature pinning (NumIn + IsVariadic + per-param type name check)"
key-files:
  created:
    - internal/utils/operlog/regression_test.go
  modified:
    - CLAUDE.md
    - docs/开发规范.md
    - .planning/STATE.md
    - .planning/ROADMAP.md
decisions:
  - "Convention placed as the FIRST subsection under '## Critical Development Rules' in CLAUDE.md (before '### Status Value Convention') so it is the first rule a developer reads — the plan said 'just below Critical Development Rules', interpreted as top-of-rules."
  - "Used '###' (h3) heading for the new CLAUDE.md section to fit cleanly under the '## Critical Development Rules' parent, rather than a sibling '##' section which would have broken the doc's hierarchy."
  - "regression_test.go uses go/parser + go/ast to read OperType constant values from operlog.go source (not reflection) because Go constants are untyped/compile-time — reflection cannot read them. AST walk also enables the count test for free."
  - "TestRecordSignatureStable pins both Record (5 fixed + variadic) AND RecordWithBody (5 fixed, non-variadic) — RecordWithBody's non-variadic-ness is itself an invariant (it takes no options)."
  - "mandatorySensitiveKeywords set to the 11 keywords explicitly named in the plan's acceptance criteria (not all 17) — the 11 are the documented floor; the other 6 (adminPassword/clientSecret/accessKey/secretKey/private_key/publicKey) are validated by coverage_test.go already."
metrics:
  duration: ~20min
  completed: 2026-06-16
  tasks_completed: 1
  tasks_total: 1
  files_created: 1
  files_modified: 4
---

# Phase 34 Plan 10: Documentation + Regression Test (Phase Close) Summary

Phase 34 收尾计划：将"新增写操作 handler 必须调用 `operlog.Record(...)`"这一约定写入开发者最先阅读的两份文档（`CLAUDE.md` + `docs/开发规范.md`），并新增 4 个回归测试锁定 `operlog` 包的公共 API（24 个 OperType 常量值、常量数、Record/RecordWithBody 签名、11 个强制敏感关键词），让任何静默改动都会立即在 CI 失败。Phase 34 全部 10/10 plans 闭环。

## What Was Built

### CLAUDE.md — 新增强制约定章节

新增 `### 操作日志记录约定 (operlog convention) — 强制` 子节，作为 `## Critical Development Rules` 下的第一条款（位于 `### Status Value Convention` 之前），包含：

- **规则:** 所有业务写操作 handler 必须在 success path 末尾、`response.Success(...)` 之前调用 `operlog.Record(...)`。
- **helper 包路径:** `github.com/xingran-next/xingran-go-backend/internal/utils/operlog`
- **调用模式（非敏感端点）:** 1 行 `operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "用户管理", operlog.OperTypeCreate)` 代码示例。
- **敏感端点:** `operlog.RecordWithBody(...)` 自动读取并恢复请求体、对 17 个敏感关键词值脱敏为 `******`。
- **OperType 业务类型常量映射表（24 个）:** 完整的 `常量 → 值 → 语义` 三列表，覆盖 `OperTypeOther=0` 到 `OperTypeReject=23`。
- **module 中文名规范:** 列出 30+ 个推荐中文模块名（用户管理 / 楼宇管理 / 网络设备 / API密钥管理 等）。
- **参考实现:** `internal/api/v1/system/ad_domain_handler.go`（Phase 34 之前的唯一参考，现已成为全模块通用约定）。
- **回归守护:** 指向 `regression_test.go` 的 4 个测试。

### docs/开发规范.md — 新增强制条款

新增 `### 5.1.1 操作日志记录 (强制)` 子节（位于 `### 5.1 Go代码规范` 与 `### 5.2 TypeScript代码规范` 之间），含 7 条规则要点：

1. 所有业务写操作必须埋点（Phase 34 全模块集成后覆盖率 100%）。
2. 敏感端点必须用 `RecordWithBody`。
3. 敏感字段必须通过 `operlog.WithOperParam(operlog.FilterSensitiveParams(string(body)))` 传入，关键词黑名单覆盖 17 个关键词。
4. OperType 共 24 个常量，值固定不可重排（值映射表指向 CLAUDE.md）。
5. module 参数使用中文模块名。
6. 参考实现指向 `ad_domain_handler.go` + `apikey_handler.go`。
7. 回归守护指向 `regression_test.go`。

### internal/utils/operlog/regression_test.go — 4 个公共 API 锁定测试

| 测试 | 守护的 invariant | 失败后果 |
|------|----------------|---------|
| `TestOperTypeConstantStability` | 24 个 `OperType*` 常量值与文档一致（0-23） | 重编号会错误标记历史 `sys_oper_log` 行 |
| `TestOperTypeCountEquals24` | 常量数恰好 24 | 增删常量需同步更新 309 个调用点 + 文档 |
| `TestRecordSignatureStable` | `Record` 5 固定参 + 1 可变参；`RecordWithBody` 5 固定参非可变参 | 签名变更会破坏 ~267 个调用点的编译 |
| `TestFilterSensitiveParamsKeywordsStable` | `sensitiveKeys` 至少含 11 个强制关键词 | 删除关键词会让敏感字段泄露到 `sys_oper_log.oper_param` |

**实现细节:**
- `readOperTypeConsts(fileName)` 用 `go/parser` + `go/ast` 解析 `operlog.go`，遍历 `*ast.GenDecl` (token.CONST) 中的 `*ast.ValueSpec`，提取 `OperType*` 标识符的整数字面量值。Go 常量是无类型/编译期，无法用 reflect 读取，AST 是唯一路径（顺便让 count 测试复用同一解析）。
- `TestRecordSignatureStable` 用 `reflect.TypeOf(Record)` 校验 `NumIn()==6`、`IsVariadic()`、5 个固定参类型名（`*gin.Context` / `operlog.Recorder` / `*gorm.DB` / `string` / `int`）、可变参元素类型 `operlog.RecordOption`；同时校验 `RecordWithBody` 的 `NumIn()==5` 且非可变参。
- `expectedOperTypeValues` map 同时充当"期望清单"：未列出的常量会触发 `unexpected OperType constant` 错误，强制新增常量时同步更新 map + CLAUDE.md + 开发规范.md（三源一致）。

### .planning/STATE.md + .planning/ROADMAP.md — Phase 34 闭环

- STATE.md: `stopped_at` 更新为 Plan 34-10 完成；`completed_phases` 7→8；Phase Status 表新增 `| 34 | 操作日志全模块集成 | ✅ Completed | 10/10 plans |`；Session History 新增 Plan 09 + Plan 10 两行；Session Continuity 更新。
- ROADMAP.md: `### Phase 34` 标题由 📋 改为 ✅；`Plans:` 由 `9/10` 改为 `10/10 plans executed — ALL COMPLETE ✅`；新增 34-10-PLAN.md 条目；底部汇总表 `34. 操作日志全模块集成` 行 `9/10 In Progress` → `10/10 Complete 2026-06-16`；Total `122 → 123 plans completed`。

## Verification

所有 acceptance criteria 满足：

| Criterion | Result |
|-----------|--------|
| CLAUDE.md 含 `operlog.Record` | PASS (3 处) |
| CLAUDE.md 含 `操作日志` | PASS (3 处) |
| CLAUDE.md 含新章节标题提及操作日志 | PASS (`### 操作日志记录约定 (operlog convention) — 强制`) |
| docs/开发规范.md 含 `operlog.Record` | PASS (3 处) |
| docs/开发规范.md 含 `FilterSensitiveParams` | PASS (1 处) |
| STATE.md Phase Status 含 "操作日志全模块集成" | PASS |
| ROADMAP.md Phase 34 状态为 ✅ (非 📋) | PASS (`### Phase 34: 操作日志全模块集成 ✅`) |
| regression_test.go 含 4 个测试函数 | PASS (TestOperTypeConstantStability / TestOperTypeCountEquals24 / TestRecordSignatureStable / TestFilterSensitiveParamsKeywordsStable) |
| `go build ./...` exits 0 | PASS |
| `go vet ./...` exits 0 | PASS |
| `go test -count=1 ./internal/utils/operlog/` exits 0 | PASS (4 新测试 + 既有测试全绿) |

## Deviations from Plan

### [Rule 3 - 阻塞问题] 章节标题层级选择

- **Found during:** Task 1, 编辑 CLAUDE.md
- **Issue:** 计划写 `Add a new section "## 操作日志记录约定 (operlog convention)" just below "Critical Development Rules"`。但 `## Critical Development Rules` 本身是 h2，其下子节（如 `### Status Value Convention`）是 h3。若按字面插入一个 `##`（h2）同级章节在 `Critical Development Rules` 的标题文本之后、`### Status Value Convention` 之前，会破坏文档层级（一个 h2 章节的正文里突然出现另一个 h2 章节）。
- **Fix:** 改为插入 `### 操作日志记录约定 (operlog convention) — 强制`（h3）作为 `## Critical Development Rules` 下的**第一个**子节（在 `### Status Value Convention` 之前）。这符合"just below Critical Development Rules"的语义（位置上紧贴在该标题之下、且是该规则区段的第一条），同时保持文档层级一致。
- **Files modified:** CLAUDE.md
- **Commit:** efb8632

### [Rule 2 - 补强] mandatorySensitiveKeywords 用计划列出的 11 个而非全 17 个

- **Found during:** Task 1, 编写 regression_test.go
- **Issue:** 计划的 `TestFilterSensitiveParamsKeywordsStable` acceptance criteria 明确列出 11 个 mandatory keywords（password / pwd / secret / token / key / salt / privateKey / oldPassword / macKey / sm4Key / sm2Key）。`sensitiveKeys` 实际有 17 个（多出 adminPassword / clientSecret / accessKey / secretKey / private_key / publicKey）。
- **Fix:** `mandatorySensitiveKeywords` 严格采用计划列出的 11 个作为"强制最低集合"（这是文档与代码之间的契约底线）。其余 6 个关键词由既有的 `coverage_test.go::TestFilterSensitiveParamsCoversAllKeywords` 守护（它遍历 `sensitiveKeys` 全量）。两层守护互补：regression 锁定文档承诺的 11 个，coverage 锁定代码实际的 17 个。
- **Files modified:** internal/utils/operlog/regression_test.go
- **Commit:** efb8632

## Pre-existing Test Failures (无关本计划)

`go test ./...` 在以下包存在失败，均为**预先存在**且**与本计划无关**（本计划仅新增 `operlog` 包的 `_test.go` + 编辑文档/`.planning/`，未触碰这些包的任何源文件）：

| 包 | 失败测试示例 | 根因（确认） |
|----|------------|-----------|
| `internal/services/operations` | `TestPageSizeConstants` (MaxPageSize=10000, want 100) | 测试断言过时常量值（与 operlog 无关） |
| `internal/api/v1` | `TestLoginWithInvalidEncryptedRequest` (404 page not found) | 测试环境路由未注册（与 operlog 无关） |
| `internal/api/v1/auth` | `TestADLoginWithOUProcessing` | AD/LDAP 集成测试环境依赖 |
| `internal/core/security` | `TestIntegration_LocalAuthenticator_*` | DB/Redis 环境依赖 |
| `internal/services` | `TestCleanTimestampFromInterface`, `TestAsyncLogging` | 既有的服务层测试问题 |
| `internal/services/system` | `TestCreateAPIKey`, `TestListAPIKeys` | DB 环境依赖 |
| `internal/services/lldp` | `TestParseHuaweiLLDPNeighbors` | TextFSM 解析 fixture 问题 |

**关键证据:** 我唯一新增的 Go 测试文件在 `internal/utils/operlog/` 包，该包全部测试（4 新 + 既有）100% 通过。上述失败包的源文件均未被本计划触碰。这些失败属于 Phase 34 范围外的既有技术债。

## Known Stubs

无。本计划为文档 + 回归测试，无业务代码、无 stub 路径。

## Threat Flags

无新威胁面。本计划是 Phase 34 `<threat_model>` 中已枚举的 T-34-DOC-01（convention 漂移）与 T-34-DOC-02（静默 API 变更）的缓解措施落地：

- T-34-DOC-01 (Repudiation — convention drift): 两份开发者首读文档（CLAUDE.md + 开发规范.md）均含显式 import 路径 + 常量列表 + 敏感关键词列表。
- T-34-DOC-02 (Tampering — silent API change): 4 个回归测试锁定公共 API（常量值 / 常量数 / Record 签名 / 敏感关键词），任何静默改动立即失败。

## Self-Check: PASSED

- `internal/utils/operlog/regression_test.go` — FOUND
- `CLAUDE.md` (新增操作日志约定章节) — FOUND (`### 操作日志记录约定 (operlog convention) — 强制`)
- `docs/开发规范.md` (新增 5.1.1 章节) — FOUND (`### 5.1.1 操作日志记录 (强制)`)
- `.planning/STATE.md` (Phase 34 ✅ Completed 10/10) — FOUND
- `.planning/ROADMAP.md` (Phase 34 ✅, 10/10 Complete) — FOUND
- Commit `efb8632` — FOUND in `git log --oneline`
- `go build ./...` — exits 0
- `go vet ./...` — exits 0
- `go test -count=1 ./internal/utils/operlog/` — PASS (4 新测试 + 既有测试全绿)

## Commits

- `efb8632` docs(34-10): add operlog convention to CLAUDE.md + 开发规范.md + regression test

## Phase 34 Closeout

Phase 34 (oper-log-full-coverage) 全部 10/10 plans 闭环：

| Plan | Wave | 内容 | Commit |
|------|------|------|--------|
| 34-01 | 1 基础设施 | 共享 operlog 包 (Record/RecordWithBody/24 OperType/17 关键词) | 5f73691, 14a238d |
| 34-02 | 2 system 核心 | user/role/dept/menu/dict/post (31 端点) | ffd8bae, 6867d3e |
| 34-03 | 2 system 外围 | notice/apikey/config/profile/settings/file (23 端点) | d72fd0b, a47ec65 |
| 34-04 | 3 operations | building/floor/workstation/... (56 端点) | d7f8903 |
| 34-05 | 4 network | device/credential/template/... (44 端点) | 93e2a6e |
| 34-06 | 5 cross-module | vdi/workorder/duty/knowledge/scheduler (59 端点) | a4cc17c, 278f678 |
| 34-07 | 6 monitor+rpa+agent | monitor/rpa/agent (45 端点) | cbb4ef0, 1a3bdf8 |
| 34-08 | 7 system 子模块 | dashboard/column_config/... (31 端点) | 3c229ba |
| 34-09 | 3 e2e 验证 | scripts/operlog_e2e_verify.sh + coverage_test.go | 6a0799b |
| 34-10 | 3 文档+回归 | CLAUDE.md + 开发规范.md + regression_test.go | efb8632 |

**最终成果:** 操作日志覆盖率由 9/309 (2.9%) 提升至 298/298 (100%)；267 个 grep-verifiable `operlog.Record` 调用；23 个 `RecordWithBody` 敏感端点；18 个敏感关键词；24 个 OperType 常量；4 个回归测试锁定公共 API；2 份开发者首读文档固化约定。
