---
phase: 145
plan: 01
status: complete
commit: false
reqs: [AUDIT-01, AUDIT-02, AUDIT-03]
---

# Phase 145 · Plan 01 SUMMARY — AUDIT 静态门禁 Go 化

> **执行口径**：4 个 executor 并行写同一工作区，Lead 统一集成提交。本 phase **未执行任何 git 写操作**（`git add` / `commit` / `stash` / `checkout` / `reset` 一律未用）。`commit: false` 已遵守。

---

## 1. 改动清单（逐文件）

| 文件 | 动作 | 关键 diff |
|---|---|---|
| `internal/utils/operlog/handler_coverage_test.go` | **新增**（~430 行） | `repoRoot(t)` 用 `runtime.Caller(0)` 上溯 3 级 + `go.mod` 兜底锚定仓库根；`minOperlogCalls=290` / `minSensitiveKeys=17`；`operlogCallRE` / `shimCallRE` / `handlerReceiverRE((?m))` 逐字迁移 shell 正则；`readonlyAllowlist` **逐字保留** 2 项；`classifiedHandlers` 方案 C 三组分类（每项带 reason）；`knownGapHandlers` 真实缺口可观测登记；3 个测试：`TestStaticOperlogCoverage_Calls` / `_SensitiveKeys` / `_HandlerDifferential` |
| `scripts/operlog_e2e_verify.sh` | 重写（329→239 行） | 删 `:68-151` 整段 STATIC COVERAGE CHECK；**凭据门禁下移**到 LIVE 段内 `SKIP_LIVE` 判定之后、登录之前；头注释改单一「LIVE E2E CHECK」+ 静态断言去向；`Exit 1` 描述改「credential error (LIVE) or LIVE assertion failure」；LIVE `:167-329`（探活 / 登录 / `assert_logged` / 30+ 端点 / Summary / `FAIL_COUNT`）一字未改 |
| `tests/e2e/operlog_e2e_verify.sh` | 重写（329→239 行） | 同上；与权威副本**仅剩 1 行注释路径差异**（`:49` driver 路径 `scripts/e2e/...` vs `tests/e2e/...`，符合 PLAN「保持该差异」） |
| `scripts/e2e/operlog_e2e_verify_test.go` | 重写（94→~105 行） | 新增 `runtime.Caller` 2 级 `repoRoot` + `cmd.Dir = repoRoot` + `scriptPath` 仓库根绝对路径（保持指向 `scripts/operlog_e2e_verify.sh`）；删假绿 skip 耦合，缺凭据仅 LIVE 可观测 `t.Skipf`、其它非零一律 `t.Fatalf`；删静态输出提取；`SKIP_LIVE=1` 不要求 Summary |
| `tests/e2e/operlog_e2e_verify_test.go` | 重写（88→~101 行） | 同上（保持指向 `tests/e2e/operlog_e2e_verify.sh`），两 driver 未互换脚本指向 |
| `scripts/README.md` | 更新（+~8 行） | `e2e/` 段补「静态 3 项断言在 `handler_coverage_test.go`（`go test ./internal/...` 硬门禁）」；顶层登记表把 `operlog_e2e_verify.sh` 拆为**权威副本** + **重复副本**两行；追加「重复副本处置 = 保留双份 + 登记权威/从属 + 强制同步改动，物理合并由 Lead 单独裁定」；`:271` 历史归档件说明保持不矛盾 |
| `.github/workflows/ci.yml` | **最小 2 行注释**（`Test` step） | 确认 `:62-63` `go test ... ./internal/... ./pkg/... ./cmd/...` 天然含 `internal/utils/operlog` 包；仅在 `Test` step 注释补「Phase 145: static operlog coverage assertions ... are hard-gated here」。**未改既有测试范围、未动 lint step、未动 `:65-76` PG contract step、未碰 `.golangci.yml`** |
| `.planning/phases/145-audit-go-audit-01-02-03/145-SUMMARY.md` | **新增**（本文件） | Task 6 产出 |

**未改**：`.golangci.yml`（归 148）、`internal/core/security/**`（146）、`internal/services/asset/**`（147）、`internal/services/operations/**`、`xingran-react-frontend/**`、`internal/api/router.go`、`AGENTS.md`、任何其它 phase 的 PLAN/SUMMARY、79 条人工验证跟踪表。

---

## 2. 三个 REQ 如何满足

### AUDIT-01（静态段 Go 单测化 + `go test ./internal/...` 硬门禁）
- `scripts/operlog_e2e_verify.sh:68-151` 的 3 项断言**逐字迁移**为 `handler_coverage_test.go` 的 3 个 Go 单测：
  1. `operlog.Record|RecordWithBody` 全 `internal/` 调用数 **≥290**（阈值 250→290 因 Phase 34 补 25 端点，注释保留）；额外硬断言 `CALLS>0`（cwd 假绿守护）。
  2. `sensitiveKeys` 条目 **≥17**：用 `go/parser.ParseFile` + `go/ast` 从 `operlog.go` 的 `var sensitiveKeys = []string{...}` 的 `*ast.CompositeLit.Elts` 抽 `*ast.BasicLit(token.STRING)`，**不是**直接 `len(sensitiveKeys)`，删一个条目即红（与 shell awk 同口径）。
  3. handler differential：枚举 `internal/api/v1/**/*_handler.go` 中定义 `*Handler` 接收者方法的文件（正则 `(?m)^func \(h \*[A-Za-z]+Handler\) [A-Za-z]+\(`），逐文件数 `operlog.Record/RecordWithBody` + legacy `recordOperLog(` shim，0 命中且不在分类表 → `t.Fatalf`（错误文案含 "add it to ... with a reason"）；额外硬断言 `HANDLER_FILES>0`。
- 零外部进程、零 `grep`/`awk`、零环境变量、零 DB。
- `ci.yml` 的 `./internal/...` 天然覆盖该包 → **CI 硬门禁成立**（默认 0 diff，仅补 2 行确认注释）。

### AUDIT-02（零凭据 / 零 DB / 任意 cwd 真跑且失败会红；LIVE 缺凭据仍 `exit 1`）
- **cwd 锚定**：`repoRoot` 用 `runtime.Caller` + `go.mod` 兜底；所有被扫描路径由 `filepath.Join(repoRoot, "internal", ...)` 拼绝对路径，**无 `"internal/"` 相对路径常量**。两 Go driver 同步设 `cmd.Dir = repoRoot`（`runtime.Caller` 2 级）+ 脚本绝对路径。
- 实测双 cwd（仓库根 / `internal/utils/operlog` 包子目录）`CALLS=373`、`HANDLER_FILES=84`、`SENSITIVE_KEYS=39` **完全相等且非 0**（基线 373 / 84）。
- 反向变异「删前 PASS → 整行删除 `settings_handler.go:89` 的 `operlog.Record` → FAIL（differential 列出该文件 + CALLS=372）→ 还原 PASS」三段证据齐全。
- LIVE 段保留 shell、`SKIP_LIVE=1` → `exit 0`、无凭据且不设 `SKIP_LIVE` → `exit 1` 且原文案不变；两 Go driver 缺凭据 → 可观测 `t.Skipf`（文案明确「static checks now run in go test ./internal/...」），**不再出现「凭据缺失 → 静态检查 PASS」耦合**。

### AUDIT-03（重复副本登记 + 不做 shell 整体替换）
- `scripts/README.md` 登记权威副本（`scripts/operlog_e2e_verify.sh`）+ 重复副本（`tests/e2e/operlog_e2e_verify.sh`，仅 1 行注释路径不同）+ 同步约束 + 未来合并由 Lead 单独裁定；**未擅自删除任一脚本**。
- LIVE 段（探活 / 登录 / `assert_logged` / 30+ 采样端点 / Summary）**仍为 bash**，两 driver 仍 `exec.CommandContext("bash", ...)`，仅简化 + 设 `cmd.Dir`。

---

## 3. 关键发现（本 phase 新证据，Lead 方案 C 裁决）

### 3.1 原 shell 静态段在仓库根 cwd 真跑 = `exit 1`（不是「假绿 PASS」）
用 `DEV_MODE=1 SKIP_LIVE=1 bash scripts/operlog_e2e_verify.sh`（凭据门禁过、跳过 LIVE、零 DB/零 backend/零写数据）实测：

```
== Static checks ==
operlog.Record|RecordWithBody calls in internal/: 373      (过 >=290)
sensitiveKeys entries in operlog.go: 39                     (过 >=17)
== Handler-file operlog coverage differential ==
FAIL: 9 个 *_handler.go 定义 *Handler 接收者但 ZERO operlog 调用 ...
EXIT=1
```

PLAN 与根因报告的「已核对」只跑了计数（373 / 84），**没跑 differential**；平时被凭据门禁挡着 + 从未进 CI，故无人见过这 9 项。这进一步印证根因：静态门禁从未真跑过，连自身白名单漂移都没人发现。

### 3.2 判定机制升级（Lead 方案 C，不是无限扩白名单）
原 `READONLY_ALLOWLIST` 只有 2 项且注释要求「Add ONLY if every routed method is a read」→ 自 v1.51/v1.53 漂移、从未维护。本 phase 把 differential 从「逐文件字符串匹配 + 手写白名单」升级为**三组可验证分类 + 一个真实缺口清单**：

| 组 | 文件 | 可验证依据（代码/AST 可核对） |
|---|---|---|
| (a) 纯只读（8 项，含原 2 项逐字保留） | `network/mac_history_handler.go`、`network/mac_history_heatmap_handler.go`（原 2 项）、`asset/reconciliation_statistics_handler.go`、`network/port_write_audit_handler.go`、`network/port_write_preview_handler.go`、`operations/asset_component_handler.go`、`system/ad_login_history_handler.go`、`system/ad_reconciliation_handler.go` | 显式登记每个文件**全部 `*Handler` 方法名清单**，用 AST 抽方法名逐字比对；新增写方法 / 改名立刻红，不放水 |
| (b) 共用编排写端点（1 项） | `network/port_write_physical_handler.go` | `port_write_handler.go:337 execSinglePort` / `:456 BatchWrite` 内 AST 证明存在 `operlog.Record`，且本文件 AST 证明委托 `h.execSinglePort(...)`（`:62`） |
| (c) 自引用例外（1 项） | `monitor/oper_log_handler.go` | `needOperlog=false` + `NewMonitorLogHandler` + `OperLogService.RecordOperLog` 三证据复核（操作日志自身删除不能再写 operlog） |
| (d) **真实缺口**（非放行） | `network/port_write_trunk_handler.go` | 见 §3.3；用可观测 `t.Skip` 子测试 `KNOWN_GAP_...` 登记，从 `knownGapHandlers` 清单移除即 FAIL |

门禁从「靠手写白名单」变成「靠机制分类」——属根治的一部分，不是打补丁。

### 3.3 `port_write_trunk_handler.go` 是**真实违反 CLAUDE.md 约定**的遗留缺口（门禁已能真实暴露它）
- `port_write_trunk_handler.go:43-80`：`TrunkWrite` 成功路径直接 `response.Success(c, result)`，**前面没有任何 `operlog.Record` / `RecordWithBody`**；
- `internal/services/portwrite/` 全包 `operlog.` 零命中；`port_write_router.go:72` 把 `POST /write/trunk` 直连 `handler.TrunkWrite`（**不经过** `execSinglePort`/`BatchWrite`）；
- 对照 `port_write_physical_handler.go:62` 确实委托编排，trunk 与它「同构」只在渲染/校验层成立，**审计层不成立**；
- ⇒ 违反 CLAUDE.md「所有业务写操作 handler 必须在 success path 末尾、`response.Success(...)` 之前调用 `operlog.Record(...)`」强制约定，v1.51 Phase 124 遗留。
- **本 phase 不修**（文件面属 portwrite，归 W2 的 148/150 或后续 quick；范围外）。但**绝不静默当合法例外放行**：`knownGapHandlers` 带文件:行号证据，由 `TestStaticOperlogCoverage_HandlerDifferential` 的可观测子测试 `KNOWN_GAP_port_write_trunk_handler.go` 以 `t.Skipf` 登记（**skip 可观测、非放行**），且反向复核「确实仍无 operlog 调用」——若将来补了调用，子测试 `t.Fatalf` 要求把该项从清单摘除，避免清单变成永久放水口。
- **这正是 145 根治价值的实证**：静态门禁零凭据/任意 cwd 真跑后，立即把此前被 skip 掩盖的真实缺口暴露出来。

---

## 4. 验证结果（Task 5 逐项，全部有输出证据）

| # | 命令 | 结果 |
|---|---|---|
| 1 | `SKIP_E2E=1 go test -count=1 ./internal/utils/operlog/ ./scripts/e2e/ ./tests/e2e/` | **PASS**（3 包 ok，EXIT=0） |
| 2 | `bash -n` 两脚本 | 均 OK |
| 3 | `SKIP_LIVE=1`（无凭据）两脚本 | `exit 0`，输出 `SKIP_LIVE=1 set — skipping live-DB portion` |
| 4 | 无凭据且不设 `SKIP_LIVE` 两脚本 | `exit 1`，输出原文案 `ERROR: ADMIN_USER and ADMIN_PASSWORD env vars required...` |
| 5 | `go vet ./internal/utils/operlog/ ./scripts/e2e/ ./tests/e2e/` | EXIT=0 |
| 6 | `go build ./...`（AGENTS.md 强制，改 Go 后） | **BUILD_EXIT=0** |
| 7 | 仓库根 cwd `go test -v -run TestStaticOperlogCoverage` | `CALLS=373 / HANDLER_FILES=84 / SENSITIVE_KEYS=39`，PASS |
| 8 | 包子目录 cwd（`cd internal/utils/operlog && go test ... .`） | `CALLS=373 / HANDLER_FILES=84 / SENSITIVE_KEYS=39`，PASS，**与仓库根相等且非 0** |
| 9 | 包子目录 cwd 下 driver（`cd scripts/e2e && go test ./...`） | ok（cmd.Dir 锚定，无「目标文件不存在」） |
| 10 | 反向变异（整行删 `settings_handler.go:89` 的 `operlog.Record`） | **删前 PASS(EXIT=0) → 删后 FAIL(EXIT=1，differential 列出该文件 + `CALLS=372`) → 还原 PASS(EXIT=0)**；用 `cp` 备份 + `sed -i '/operlog\.Record(/d'` + `trap` 恢复，**未用 git checkout/commit** |
| 11 | `SKIP_LIVE=1` 两 driver | PASS（无需凭据/backend/DB），EXIT=0 |
| 12 | `SKIP_E2E=1` 两 driver | `--- SKIP: TestE2EAllEndpointsLogged`（可观测 skip） |
| 13 | 缺凭据、不设 `SKIP_LIVE` 两 driver | `--- SKIP` + 文案 `LIVE skipped: admin credentials not provided (static checks now run in go test ./internal/...)`，非 Fatalf |
| 14 | `grep -n "./internal/..." .github/workflows/ci.yml` | `:65` 命中（`./internal/... ./pkg/... ./cmd/...`），静态断言落在 `./internal/...` 内 → 硬门禁成立 |
| 15 | 两脚本静态关键词残留复查 | 无命中（`STATIC`/`Static checks`/`READONLY_ALLOWLIST`/`CALLS`/`SENSITIVE_KEYS`/`HANDLER_FILES` 全清） |

**基线数值**：`CALLS=373`（与 shell `grep -r ... internal/` 逐字一致）、`SENSITIVE_KEYS=39`（≥17）、`HANDLER_FILES=84`（与 PLAN 基线一致）。

**计数口径决策**：CALLS 采用「全 `.go` 含 `_test.go`」口径，与 shell `grep -rE ... internal/` 逐字一致（实测 373）；若排除 `_test.go` 为 363，仍 ≥290。测试文件自身的正则字面量用字符串拼接构造、注释避免完整 `operlog.Record(` 文本，避免自引用把 373 抬成 374。sensitiveKeys 从 AST 抽 `[]string{...}` 的 BasicLit 元素（39），与 shell `awk` 口径一致。

---

## 5. Known Gaps（登记，非阻断）

1. **【本 phase 新发现，Lead 已确认】`port_write_trunk_handler.go` 真实违反写操作 operlog 约定**：`POST /network/ports/write/trunk`（`port_write_trunk_handler.go:43-80`）成功路径无 `operlog.Record`，`internal/services/portwrite/` 全包零 operlog，`port_write_router.go:72` 直连 `handler.TrunkWrite`，违反 CLAUDE.md「写操作 success path 在 `response.Success` 之前必须 `operlog.Record`」。v1.51 Phase 124 遗留；**本 phase 不修**（文件面/范围属 portwrite，建议后续归属 W2 148 或独立 quick）。门禁已用 `knownGapHandlers` + 可观测 `t.Skip` 子测试真实暴露它 = 145 根治价值的实证，不伪造闭环、不静默放行。Lead 将作为 v1.55 新增 Known Gap 收进 STATE/MILESTONES。
2. 两脚本重复副本**未物理合并**（已登记权威/从属 + 同步约束，合并由 Lead 单独裁定）；各自配套不同包 driver，强删会断路径。未来若新增编排函数或改端口写编排，须同步维护 `classifiedHandlers` / `knownGapHandlers` 清单（残余风险：显式清单需人工同步）。
3. `AGENTS.md` **未改**（如需补目录治理条目移交 Lead，PLAN Task 4C 明确不在本 phase 执行）。
4. 全量 `go build ./...`（已在本 phase 跑，EXIT=0）+ `go test ./...` + `tsc` + `lint:ci`（含 forbidigo） + `vitest` 由 **Lead 集成提交前统一跑**；本 plan 只跑相关包（已额外跑通 `go build ./...`）。
5. `scripts/e2e/` 与 `tests/e2e/` 仍不在 `ci.yml` 测试范围（LIVE 需凭据，符合禁区「不整体替换」；静态已搬进 `./internal/...`）。
6. `ci.yml` 改动为 2 行确认注释（PLAN 允许「0 或 1 行注释」，2 行仍属最小 diff，未动任何 step/范围）；`.golangci.yml` 归 148，本 phase 未碰。
7. 静态计数口径（是否含 `_test.go`）见 §4：采用含 `_test.go`（373），与 shell 逐字一致。

---

## 6. 账务与禁区自查

- **未执行 `git add` / `git commit` / `git stash` / `git checkout` / `git reset`，由 Lead 统一集成提交**；`commit: false` 已遵守。仅用 `git diff --stat` 做**只读**核对（ci.yml 2 行注释）。
- 未改 `.golangci.yml`（归 148）；未碰 `internal/core/security/**`（146）/ `internal/services/asset/**`（147）/ `internal/services/operations/**` / `xingran-react-frontend/**` / `internal/api/router.go`；未改任何其它 phase 的 PLAN/SUMMARY；未动 79 条人工验证跟踪表。
- 未启动长驻进程（dev server / DB / watcher 均未开）；未连真实 DB；未写生产数据；未伪造 PG 双跑 / 未勾人工验证 / 未把 LIVE 真机结果写成已验证。
- 未做 shell 整体替换为 Go e2e；未只调门禁顺序；cwd 假绿已由 `runtime.Caller` + `cmd.Dir` 双保险根治。
