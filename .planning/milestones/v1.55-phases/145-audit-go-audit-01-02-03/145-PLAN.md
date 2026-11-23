---
phase: 145
plan: 01
wave: 1
depends_on: []
files_modified:
  - internal/utils/operlog/handler_coverage_test.go   # 新增（或扩 internal/utils/operlog/coverage_test.go）
  - scripts/operlog_e2e_verify.sh                      # 删 :68-151 静态段 + 凭据门禁下移 + 头注释同步
  - tests/e2e/operlog_e2e_verify.sh                    # 同上（重复副本）
  - scripts/e2e/operlog_e2e_verify_test.go             # 简化 driver + 设 cmd.Dir
  - tests/e2e/operlog_e2e_verify_test.go               # 简化 driver + 设 cmd.Dir
  - scripts/README.md                                  # 登记两份脚本权威/重复副本处置 + 静态段去向
  - .github/workflows/ci.yml                           # 确认 go test ./internal/... 覆盖；仅在必要时追加最小一步
  - .planning/phases/145-audit-go-audit-01-02-03/145-SUMMARY.md   # Task 6 产出
autonomous: true
commit: false
reqs:
  - AUDIT-01
  - AUDIT-02
  - AUDIT-03
---

# Phase 145 · Plan 01 — AUDIT 静态门禁 Go 化

> **执行环境**：4 个 executor 并行写同一工作区；Lead 统一集成提交。本 plan **禁止任何 git 写操作**（`git add` / `commit` / `stash` / `checkout` 一律不执行）。`commit: false`。
> **并行冲突面**：`ci.yml` 归 145、`.golangci.yml` 归 148（见 ROADMAP.md:148 全局唯一归属硬登记）。本 phase **不碰 `.golangci.yml`**，也**不碰** `internal/core/security/**`、`internal/services/asset/**`、`internal/services/operations/**`、`xingran-react-frontend/**`。

---

## 1. 必读顺序（开工前按序读完，全部只读）

1. `.planning/STATE.md` — v1.55 active / `## Current Position`（Phase 145 为 W1 首个 phase，状态 `planning`）。
2. `.planning/ROADMAP.md` — **Phase 145 详情块**：需求映射 AUDIT-01/02/03、5 条成功标准、文件面、并行冲突面（`:115` W1 文件面互斥表、`:148` `ci.yml` 归 145 / `.golangci.yml` 归 148）、`:173-181`「明确不算根治」边界、`:150-171` 关键约束与 5 项裁定。
3. `.planning/REQUIREMENTS.md` — `### AUDIT`（`:46-51` AUDIT-01/02/03 全文）、`## Constraints`（`:98-106`）、`## 决策裁定记录`（`:7-16` 5 项裁定）。
4. `.planning/notes/261109-v154-locked-root-cause-analysis.md` — **本 phase 的 research 输入**：`§④`（`:105-115` 根因 / 根治定义 / 方案对比 / 「不应修」边界）、`:23-26` 意外收获（cwd 锚定新发现 + 「静态检查从未进 CI」实证）、`:212` 证据索引。
5. `CLAUDE.md` / `AGENTS.md` — 项目硬约定：`go test ./...` 就近 `*_test.go`；`tests/`（复数）归位；`scripts/README.md` 登记在用脚本；operlog 写操作约定；禁止裸 `c.JSON`；构建产物只落 `out/`；提交前 `go build ./...` + `go test ./...`（本 phase 由 Lead 统一执行）。

**已核对（2026-11-09，只读抽查，根因报告与代码现状一致，无 PLANNING BLOCKED 冲突）**：
- `scripts/operlog_e2e_verify.sh:57-66` 凭据门禁 `exit 1` → `:68-151` 静态段 → `:162` `SKIP_LIVE` → `:167-329` LIVE 段，与报告逐行吻合。
- `:78` `grep -rE "operlog\.(Record|RecordWithBody)\(" internal/ | wc -l`（相对仓库根）、`:83` 阈值 `290`（注释 `:80-82` 写明由 250 提升因 Phase 34 补 25 端点）、`:88` `awk` 抽 `internal/utils/operlog/operlog.go` 的 `var sensitiveKeys = []string{`、`:90` 阈值 `17`、`:105-111` `READONLY_ALLOWLIST` 恰 2 项（`network/mac_history_handler.go`、`network/mac_history_heatmap_handler.go`）、`:115` 遍历 `internal/api/v1/` 的 `*_handler.go` `*Handler` 接收者、`:120/:126` 数 `operlog.Record/RecordWithBody` + legacy `recordOperLog` shim、`:142-150` 0 命中且不在白名单 → `exit 1`。
- `.github/workflows/ci.yml:62-63` `go test ... ./internal/... ./pkg/... ./cmd/...`，**不含** `scripts/e2e/` 与 `tests/e2e/`。
- `scripts/e2e/operlog_e2e_verify_test.go:79-84` 与 `tests/e2e/operlog_e2e_verify_test.go:73-78` 均把 `ADMIN_USER and ADMIN_PASSWORD env vars required` stderr 转 `t.Skipf`；两 driver 均**未设 `cmd.Dir`**。
- 现成基建：`internal/utils/operlog/regression_test.go`（405 行，`go/parser` + `readOperTypeConsts` 锁 25 常量 / 签名 / 23 个 `mandatorySensitiveKeywords`）、`internal/utils/operlog/coverage_test.go`（230 行，`:35` 已 `len(sensitiveKeys) < 17` 断言、`FilterSensitiveParams` 全关键词）；`coverage_test.go:121-230` **无 handler 文件扫描**（本 phase 要补的缺口）。`operlog.go:135` `var sensitiveKeys = []string{` 为多行字符串字面量列表。
- 两份 329 行脚本仅 `:157` 注释路径不同（`scripts/e2e/...` vs `tests/e2e/...`），属重复副本；`scripts/README.md:142` / `:271` 已有登记，需更新。

---

## 2. 目标 / 已确认事实 / Lead 裁决表

### 2.1 目标（一句话）
把 `operlog_e2e_verify.sh` 的静态 3 项断言**从 shell 拆出、改写为 `go/parser` Go 单测**并纳入 `go test ./internal/...`（`ci.yml` 天然硬门禁），**同时用 `runtime.Caller` 锚定仓库根**修掉 cwd 缺陷；LIVE 段保留 shell 且缺凭据仍 `exit 1`。使静态门禁「零凭据 / 零 DB / 任意 cwd 真跑且失败会红」。

### 2.2 已确认事实（直接作为输入，不重新调研）
- **根因**：凭据门禁 `exit 1` 排在静态检查块之前，两个 Go driver 把 `exit 1` 转 `t.Skipf` → CI 两包「凭据缺失 → 可观测 skip → PASS」。
- **cwd 缺陷（决定性）**：静态检查用相对仓库根路径（`grep -r ... internal/`、`operlog.go`、`internal/api/v1/`），两个 driver 都没设 `cmd.Dir`，Go test cwd 是包子目录 → 实测仓库根 `CALLS=373 / HANDLER_FILES=84`，包子目录 `CALLS=0 / 目标文件不存在 / HANDLER_FILES=0`。**仅挪门禁顺序连缓解都不算（假绿）**。
- **CI 事实**：`ci.yml:62-63` 只跑 `./internal/... ./pkg/... ./cmd/...`，`scripts/e2e/` 与 `tests/e2e/` 根本不在 CI 测试范围 → 静态检查从未真正当过门禁。
- **静态 3 项断言（逐字迁移）**：① `operlog.Record|RecordWithBody` 全 `internal/` 调用数 `>= 290`（250→290 因 Phase 34 补 25 端点）② `awk` 抽 `internal/utils/operlog/operlog.go` 的 `var sensitiveKeys = []string{...}` 条目 `>= 17` ③ handler differential：枚举 `internal/api/v1/**/*_handler.go` 中定义 `*Handler` 方法的文件，逐文件数 `operlog.Record/RecordWithBody` + legacy `recordOperLog` shim，0 命中且不在 `READONLY_ALLOWLIST`（2 个只读白名单）→ FAIL。
- **现成基建**：`regression_test.go`（`go/ast` + `go/parser` 锁 25 常量 / 签名 / 强制敏感关键词）、`coverage_test.go`（`FilterSensitiveParams` 全关键词 + `RecordWithBody` 脱敏）；`coverage_test.go` 无 handler 文件扫描。
- **重复副本**：两份 329 行 `operlog_e2e_verify.sh` 仅第 157 行注释路径不同；`scripts/README.md` 需登记（AGENTS.md）。
- **LIVE 段保留**：`:167-329`（backend/psql 探活、登录取 JWT、`assert_logged`、30+ 采样端点、Summary）保留 shell、缺凭据仍 `exit 1`，只把静态段拆出。

### 2.3 Lead 裁决表（不可推翻，写进 executor 上下文）

| 项 | 裁决 | 依据 |
|---|---|---|
| 根治定义 | **静态段 Go 单测化 + cwd 锚定（`runtime.Caller` / 仓库根绝对路径） + CI 硬门禁** 三者同时成立 | ROADMAP.md:33、根因报告 §④「根治（S-M）」 |
| LIVE 段 | **保留 shell**，缺凭据仍 `exit 1`，`SKIP_LIVE` 语义不变；静态段不再依赖任何环境变量 | ROADMAP.md:33、AUDIT-02 |
| cwd 锚定 | 静态 3 项断言必须在**任意 cwd**（仓库根 / 包子目录 / `go test -C`）下取仓库根真值（`CALLS≈373` 而非 `0`）；Go driver 必须设 `cmd.Dir` 或脚本自身 `cd "$(dirname "$0")/.."` | 根因报告 `:25`、`:109`、REQUIREMENTS.md:51 |
| CI 归属 | `ci.yml` 归 145；`.golangci.yml` 归 148（本 phase **不改**） | ROADMAP.md:148 |
| 禁区 | 禁 shell 整体替换为 Go e2e；禁只调门禁顺序；禁改 `.golangci.yml`；禁碰其它 phase 文件面；禁 git 写操作 | ROADMAP.md:173-181、REQUIREMENTS.md:94 |
| 重复副本 | 两份脚本保留（各自配套 driver 在不同包），但在 `scripts/README.md` 标注**权威副本 + 重复副本关系**并同步去静态段；不强行删除其一（避免 driver 路径断裂） | AUDIT-03、AGENTS.md 目录治理 |
| 提交 | 4 executor 并行写同一工作区，**不 commit**；Lead 统一集成提交 | frontmatter `commit: false`、ROADMAP.md:161 |

---

## 3. 任务（Task 1–6，XML）

> 写集合按任务切分以支撑并行：Task 1 → `internal/utils/operlog/*_test.go`；Task 2 → 两份 `.sh`；Task 3 → 两份 `_test.go` driver；Task 4 → `ci.yml` + `scripts/README.md`；Task 5 = 纯验证（不写）；Task 6 = `145-SUMMARY.md`。**Task 1 与 Task 3 的 `_test.go` 是不同包路径，无写冲突**；Task 4 的 `ci.yml` 与其他任务无交叠。

<task id="1">
  <title>新增 Go 静态扫描单测（3 项断言 + 仓库根 cwd 锚定）</title>
  <files>
    <file>internal/utils/operlog/handler_coverage_test.go</file>
    <file alt="或扩写">internal/utils/operlog/coverage_test.go</file>
  </files>
  <action>
在包 `operlog` 内新建 `handler_coverage_test.go`（或扩 `coverage_test.go`，二者择一、不要同时建两份重复逻辑），实现从 `scripts/operlog_e2e_verify.sh:68-151` **逐字迁移**的 3 项静态断言，全部用 `go/parser` / `go/ast` / `go/token` + `os.ReadFile` + `filepath.WalkDir` 实现（零外部进程、零 `grep`/`awk`、零环境变量、零 DB）。

**A. 仓库根锚定（决定性，必须最先实现）**：
```go
// repoRoot 用 runtime.Caller 锚定本测试文件所在目录，上溯 3 级到仓库根，
// 保证 go test 在任意 cwd（仓库根 / 包子目录 / -C 切换）下都取同一绝对路径。
func repoRoot(t *testing.T) string {
    _, thisFile, _, ok := runtime.Caller(0)
    if !ok { t.Fatal("runtime.Caller failed") }
    root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
    if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
        t.Fatalf("repo root not anchored at %s (go.mod missing): %v", root, err)
    }
    return root
}
```
`internal/utils/operlog/` → `..`=internal/utils → `..`=internal → `..`=仓库根；用 `go.mod` 存在性做兜底断言。所有被扫描路径一律由 `filepath.Join(repoRoot(t), "internal", ...)` 拼出，**禁止**任何 `"internal/"` 相对路径常量。

**B. 断言①（CALLS >= 290）**：`WalkDir(repoRoot/internal/)`，只读 `.go` 文件（**排除 `*_test.go`**，与 shell `grep -r ... internal/` 口径对齐；若 shell 口径含测试文件则以「全 `.go` 含 `_test.go` 计数 >= 373 实测值」为准并在 SUMMARY 记录口径决策），用 `regexp.MustCompile("operlog\\.(Record|RecordWithBody)\\(")` 统计匹配行/调用点数。断言 `calls >= 290`（阈值常量 `const minOperlogCalls = 290`，注释写明「250→290 因 Phase 34 补 25 端点」）。**额外硬断言**：`calls > 0`（cwd 锚定红绿变异守护——若锚定失效在包子目录读到 0 会立即红，而非静默假绿）。

**C. 断言②（sensitiveKeys >= 17）**：用 `go/parser.ParseFile(token.NewFileSet(), filepath.Join(root,"internal/utils/operlog/operlog.go"), nil, 0)` 解析 AST，定位 `var sensitiveKeys = []string{...}` 的 `*ast.CompositeLit`，遍历 `Elts` 统计 `*ast.BasicLit`(token.STRING) 元素数（逐字等价 shell `awk '/var sensitiveKeys = \[\]string\{/,/^\}/' | grep -cE '^\s+"[^"]+",?\s*$'`）。断言 `>= 17`（`const minSensitiveKeys = 17`）。复用 `regression_test.go` 的 AST 基建范式（参考 `readOperTypeConsts` 的 `parser.ParseFile` 用法），**不要**直接 `len(sensitiveKeys)` 了事（须从源码 AST 抽，才能与 shell 同口径、保证「删一个条目即红」）。

**D. 断言③（handler differential）**：
- `WalkDir(repoRoot/internal/api/v1/)` 收集所有 `*_handler.go`。
- 对每个文件 `parser.ParseFile`，遍历 `ast.File.Decls` 的 `*ast.FuncDecl`，判定「定义了 `*Handler` 接收者方法」：`fd.Recv != nil` 且接收者类型去 `*` 后以 `Handler` 结尾（逐字对齐 shell `^func \(h \*[A-Za-z]+Handler\) [A-Za-z]+\(` 的语义：接收者名 `h`、单指针接收者、类型名 `XxxHandler`）。命中即把该文件纳入 `handlerFiles`。
- 逐文件统计 `operlog.Record|RecordWithBody` 调用 + legacy `recordOperLog(` shim 调用（用 AST `ast.Inspect` 找 `*ast.CallExpr` 的 `SelectorExpr`/`Ident`，或等价地对该文件源码做行级正则计数——与 shell `grep -cE` 同口径，**须把 shim 一并计入**）。
- `READONLY_ALLOWLIST` 常量**逐字迁移** shell `:105-111` 的 2 项：
  ```go
  var readonlyAllowlist = []string{
      "network/mac_history_handler.go",          // 纯 /history/* 查询 + 1 GET 导出
      "network/mac_history_heatmap_handler.go",  // 仅 QueryHeatmap (POST /history/heatmap, read)
  }
  ```
  用 `strings.HasSuffix(path, wl)` 判定（对齐 shell `[[ "$f" == *"$wl" ]]`）。
- 任一 `handlerFiles` 文件 `count==0` 且不在白名单 → `t.Errorf` 收集后 `t.Fatalf` 列出全部文件（对齐 shell `:142-150` 的 `exit 1`），错误文案含「add it to readonlyAllowlist with a reason」提示。
- **额外硬断言**：`len(handlerFiles) > 0`（cwd 锚定守护；实测仓库根 `HANDLER_FILES=84`）。

**E. 复用约束**：`package operlog`；import 分组按 AGENTS.md（标准库 / 第三方 / 内部）；测试函数命名 `TestStaticOperlogCoverage_Calls` / `TestStaticOperlogCoverage_SensitiveKeys` / `TestStaticOperlogCoverage_HandlerDifferential`（可加 `t.Parallel()`，但 `repoRoot` 用 `runtime.Caller` 不共享可变状态）；参考 `regression_test.go:25-40` 的 import 与 `parser` 用法、`coverage_test.go:30-54` 的阈值写法。测试失败信息必须打印实测数（如 `CALLS=373`），便于 Task 5 红绿变异比对。

**F. 完成判据**：`go test -count=1 ./internal/utils/operlog/` 在仓库根 cwd 与包子目录 cwd 下均 PASS，且打印的 `CALLS` / `HANDLER_FILES` 与 shell 基线 `373` / `84` 同量级（非 0）。
  </action>
</task>

<task id="2">
  <title>两份脚本删除静态段、凭据门禁下移、头注释同步</title>
  <files>
    <file>scripts/operlog_e2e_verify.sh</file>
    <file>tests/e2e/operlog_e2e_verify.sh</file>
  </files>
  <action>
两份 329 行脚本**同步改**（仅 `:157` 注释路径不同，保持该差异或按 Task 4 的 README 处置统一，但两份 LIVE 行为必须一致）。

**删除范围**：删掉 `:68-151` 整段 STATIC COVERAGE CHECK（含 `echo "== Static checks =="` 起到 `:151` `echo "all *Handler-receiver files contain >=1 operlog call..."` 止），一并删除该段引用的 `READONLY_ALLOWLIST` 数组、`CALLS`/`SENSITIVE_KEYS`/`HANDLER_FILES`/`ZERO_OPLOG_HANDLERS`/`direct`/`shim`/`count`/`allowed` 全部局部变量与 `grep`/`awk`/`wc` 静态逻辑。**静态段不再依赖任何环境变量**（已无静态段）。

**保留并调整**：
- 保留 `#!/usr/bin/env bash`、`set -euo pipefail`（`:42`）、`:44-50` 的 `API_BASE`/`DB_*`/`PSQL_EXTRA_ARGS` 默认值。
- **凭据门禁（原 `:52-66`）整体下移到 LIVE 段内、`SKIP_LIVE` 判定之后、登录（`:182-195`）之前**：即把 `ADMIN_USER`/`ADMIN_PASSWORD`/`DEV_MODE` 解析块移到 `:162-165` 的 `if [[ "${SKIP_LIVE:-}" =~ ^(1|true)$ ]]; then ... exit 0; fi` **之后**、backend/psql 探活（`:167-179`）**之前或之后均可但必须在登录取 JWT 之前**；缺凭据仍 `echo "ERROR: ADMIN_USER and ADMIN_PASSWORD env vars required..." >&2; exit 1`（恢复脚本注释声明的语义，行为不变）。
- **LIVE 段 `:167-329` 全部逻辑原样保留**：backend/psql 探活（`exit 2`）、`export PGPASSWORD`、登录取 `TOKEN`、`assert_logged` 函数、30+ 采样端点、`Summary`、`rm -f /tmp/operlog_status.$$`、`FAIL_COUNT` 非 0 时退出码逻辑（`:326-329`）一字不改。`SKIP_LIVE=1` 仍 `exit 0`。
- `:153-160` 的 LIVE 段 banner 注释同步更新：删掉「Reciprocal of the STATIC COVERAGE CHECK banner above」这类指向已删静态段的表述，改为「static coverage assertions now live in `internal/utils/operlog/handler_coverage_test.go` (`go test ./internal/...`)」。

**头注释（`:1-40`）同步更新**：
- 删除「TWO clearly-separated sections / 1. STATIC COVERAGE CHECK ...」整段，改为单一「LIVE E2E CHECK」说明，明确「**静态覆盖断言已迁出至 Go 单测 `internal/utils/operlog/handler_coverage_test.go`，由 `go test ./internal/...`（CI 硬门禁）执行；本脚本不再跑静态检查**」。
- `Required environment variables` 段删掉 `/ STATIC COVERAGE CHECK only` 之类措辞；`Optional` 段 `SKIP_LIVE` 说明改为「skip the LIVE E2E CHECK entirely（静态检查不在此脚本）」。
- `Exit codes` 段保留 `0/1/2`，但 `1` 的描述改为「credential error (LIVE) or LIVE assertion failure」，不再提静态。
- 显式声明「缺凭据 → `exit 1`（LIVE 语义不变）」。

**完成判据**：`grep -n "STATIC\|Static checks\|READONLY_ALLOWLIST\|CALLS\|SENSITIVE_KEYS\|HANDLER_FILES" scripts/operlog_e2e_verify.sh tests/e2e/operlog_e2e_verify.sh` 无命中（除更新的说明性注释外）；`bash -n` 语法检查通过；`SKIP_LIVE=1 bash <script>` 不再要求凭据且 `exit 0`；不设 `SKIP_LIVE` 且无 `ADMIN_USER`/`ADMIN_PASSWORD`/`DEV_MODE` 时 `exit 1` 并打印原凭据错误文案。
  </action>
</task>

<task id="3">
  <title>两个 Go driver 简化 + cmd.Dir 仓库根锚定</title>
  <files>
    <file>scripts/e2e/operlog_e2e_verify_test.go</file>
    <file>tests/e2e/operlog_e2e_verify_test.go</file>
  </files>
  <action>
两份 driver（94 行 / 88 行）同步简化，保留 `SKIP_E2E` 语义与 `t.Context()` 5 分钟超时（`T-34-VER-03`）。

**A. cwd 锚定（必须）**：给 `exec.CommandContext` 设 `cmd.Dir` 到**仓库根**（`runtime.Caller(0)` 上溯：`scripts/e2e/` → `..`=scripts → `..`=仓库根，即 2 级；`tests/e2e/` → `..`=tests → `..`=仓库根，即 2 级），或改用脚本自身 `cd "$(dirname "$0")/.."`（Task 2 已可择一，**两 driver 与两脚本只选一种统一方案，避免双重 cd**）。推荐 driver 侧统一：
```go
_, thisFile, _, _ := runtime.Caller(0)
repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
cmd.Dir = repoRoot
```
并把 `scriptPath` 改为相对 `repoRoot` 的绝对路径（`filepath.Join(repoRoot,"scripts","operlog_e2e_verify.sh")` / `filepath.Join(repoRoot,"tests","e2e","operlog_e2e_verify.sh")`），保留 `filepath.ToSlash`（Windows 反斜杠坑，原 `:49-52` 注释）。
> 注意：`scripts/e2e` driver 原 `scriptPath = ../../scripts/operlog_e2e_verify.sh`（指向 `scripts/operlog_e2e_verify.sh`），`tests/e2e` driver 原 `scriptPath = ./operlog_e2e_verify.sh`（指向 `tests/e2e/operlog_e2e_verify.sh`）——**保持各自指向不变**，仅改为仓库根绝对锚定，勿互换。

**B. 去掉假绿 skip**：删除 `:79-84` / `:73-78` 里把 `bytes.Contains(stderr, "ADMIN_USER and ADMIN_PASSWORD env vars required")` 转 `t.Skipf` 的分支。改法二选一（**推荐后一种**）：
- 方案①：彻底删除该分支，脚本 `exit 1` 一律 `t.Fatalf`（但 `SKIP_E2E`/`SKIP_LIVE` 仍需 skip 语义）；
- 方案②（推荐，保留「LIVE 缺凭据可观测 skip」语义且不掩盖静态失败）：driver 仅在**显式判定为 LIVE 缺凭据**时 `t.Skip`，即判断 `SKIP_LIVE` 未设且 stderr 含凭据错误文案 → `t.Skipf("LIVE skipped: admin credentials not provided...")`；**静态断言已不在脚本内，任何其它非零退出一律 `t.Fatalf`**，不得再出现「凭据缺失 → 静态检查 PASS」的耦合。无论哪种，注释必须写明「静态 3 项断言已迁至 `go test ./internal/...`，本 driver 只承载 LIVE」。

**C. 清理过时注释**：删掉头注释里「When SKIP_E2E is unset the test still runs the static checks portion of the bash script」等已失效表述（`:5-8`），改为「static checks are covered by `internal/utils/operlog/handler_coverage_test.go`; this driver runs LIVE only」。删除 `:70-77` 里对 `"calls in internal"` / `"sensitiveKeys entries"` 日志行的提取（静态输出已不存在，保留 `passed` Summary 行提取即可）与 `:89-93` / `:83-87` 的 `<n>/<n> passed` 断言注释对齐 LIVE Summary（`echo "${PASS_COUNT}/${TOTAL} passed"` 仍保留）。保留 `os.Chmod(0o755)` 与 `cmd.Env = os.Environ()`（凭据透传）。

**完成判据**：`go test -count=1 ./scripts/e2e/ ./tests/e2e/` 在 `SKIP_E2E=1` 下可观测 skip；在 `SKIP_LIVE=1` 下脚本 `exit 0`、driver PASS 且不需要凭据；`go vet` 无 `runtime`/`bytes` 未使用 import。
  </action>
</task>

<task id="4">
  <title>ci.yml 确认覆盖 + scripts/README.md 登记（AGENTS.md 不动）</title>
  <files>
    <file>.github/workflows/ci.yml</file>
    <file>scripts/README.md</file>
    <file no-write="移交 SUMMARY" >AGENTS.md</file>
  </files>
  <action>
**A. `ci.yml`**：确认 `:62-63` 的 `go test -timeout 15m -count=1 -coverprofile=coverage.out -covermode=atomic ./internal/... ./pkg/... ./cmd/...` **已天然覆盖** `internal/utils/operlog/handler_coverage_test.go`（`./internal/...` 含该包）。**默认不改 ci.yml**；仅当确认存在排除规则（如 build tag / `-run` 过滤 / 该包被单独 `skip`）时才追加**最小一步**（例如在 `Test` step 后加一行注释或 `go test -count=1 -run TestStaticOperlogCoverage ./internal/utils/operlog/` 的显式最小 step），不得改动 `./internal/...` 既有范围、不得动 lint step、不得动 `:65-76` PG contract step、不得触碰 `.golangci.yml`。在 `Test` step 的注释里补一句「Phase 145: static operlog coverage assertions in ./internal/utils/operlog are hard-gated here」即可视为「确认覆盖」的落点（最小 diff）。

**B. `scripts/README.md`**：更新 `### 顶层在用脚本（登记）` 表（`:132-143`）与相关段落：
- `:142` `| operlog_e2e_verify.sh | 操作日志 e2e |` 改为明确登记：
  ```
  | `operlog_e2e_verify.sh`（`scripts/` 顶层，**权威副本**） | 操作日志 **LIVE e2e**（需 running backend + DB + 凭据）；静态覆盖断言已迁出至 `internal/utils/operlog/handler_coverage_test.go` |
  | `tests/e2e/operlog_e2e_verify.sh`（**重复副本**） | 与权威副本仅 1 行注释路径不同，配套 `tests/e2e/operlog_e2e_verify_test.go`；LIVE 行为与权威副本逐字一致，改动须同步 |
  ```
- `### e2e/ — 端到端测试`（`:110-116`）补一句：「静态 3 项断言不在本目录脚本内，见 `internal/utils/operlog/handler_coverage_test.go`（`go test ./internal/...` 硬门禁）」。
- `:271` 提到 `operlog_e2e_verify.sh` 的归档说明保持一致性（不改归档事实，仅确保不与新登记矛盾）。
- 明确「两脚本重复副本处置 = 保留双份但登记权威/从属关系并强制同步改动；未来合并由 Lead 单独裁定」，不擅自删除任一文件。

**C. `AGENTS.md` 不动**：本 phase 不修改 `AGENTS.md`；若 executor 认为确需补充目录治理条目，写入 `145-SUMMARY.md` 的 Known Gaps 移交 Lead，不在本 plan 执行。

**完成判据**：`ci.yml` diff 为 0 或仅 1 行注释；`scripts/README.md` 两处登记齐全且无「脚本跑静态检查」的过时表述；无 `.golangci.yml` / 其它 phase 文件面改动。
  </action>
</task>

<task id="5">
  <title>自验证（只跑相关包；cwd 红绿变异 + 反向删调用验证）</title>
  <files>
    <file mode="read-only">（不写生产代码；变异用临时副本/回滚，验证后恢复）</file>
  </files>
  <action>
**只跑相关包，不跑全仓**（全量 `go build ./...` / `go test ./...` 由 Lead 集成提交前统一执行，本 phase 不抢跑）：

1. **主验证命令**：
   ```bash
   go test -count=1 ./internal/utils/operlog/ ./scripts/e2e/ ./tests/e2e/
   ```
   预期：operlog 包 3 个新静态断言 PASS；两 driver 在 `SKIP_E2E=1`（或 `SKIP_LIVE=1`）下可观测 skip/PASS，无 `ADMIN_USER` 要求、无 DB/backend 要求。

2. **cwd 锚定红绿变异（决定性）**——同一新单测在**仓库根 cwd** 与**包子目录 cwd** 各跑一次，断言 `CALLS` 取自真实仓库根（≈373）而非 0：
   ```bash
   # 仓库根 cwd
   go test -count=1 -v -run TestStaticOperlogCoverage ./internal/utils/operlog/ 2>&1 | grep -E "CALLS|HANDLER_FILES|PASS|FAIL"
   # 包子目录 cwd（go test -C 或 cd 后相对包路径）
   cd internal/utils/operlog && go test -count=1 -v -run TestStaticOperlogCoverage . 2>&1 | grep -E "CALLS|HANDLER_FILES|PASS|FAIL"
   cd scripts/e2e && go test -count=1 ./...   # driver cmd.Dir 锚定后不出现 "目标文件不存在"
   ```
   两次 `CALLS` 必须相等且 `>= 290`、`HANDLER_FILES > 0`；若任一为 0 → **FAIL，Task 1 锚定没做对，回炉**。在 SUMMARY 记录两次实测数值（基线 `373 / 84`）。

3. **反向验证（故意删一处 → 必须红）**：用可逆方式（`cp` 备份 → `node -e` 或 `sed -i` 改 → 跑 → 恢复，**绝不 git checkout/commit**）：
   - 选一个 `internal/api/v1/**/*_handler.go` 里的一行 `operlog.Record(...)` 临时注释/删除（或删 `operlog.go` 的 `sensitiveKeys` 一个条目），跑 `go test -count=1 ./internal/utils/operlog/`，**必须 FAIL**（CALLS 掉到 `<290` 或 handler differential 列出该文件 / sensitiveKeys `<17`）；
   - 恢复备份后重跑 **必须 PASS**。
   ```bash
   # 示例（node -e，Windows/MSYS 通用；务必 trap 恢复）
   cp internal/api/v1/system/user_handler.go /tmp/user_handler.go.bak
   node -e "const fs=require('fs');const p='internal/api/v1/system/user_handler.go';let s=fs.readFileSync(p,'utf8');s=s.replace(/operlog\.Record\(/, '// operlog.Record(');fs.writeFileSync(p,s)"
   go test -count=1 ./internal/utils/operlog/ ; echo "EXPECT_NONZERO=$?"
   cp /tmp/user_handler.go.bak internal/api/v1/system/user_handler.go
   go test -count=1 ./internal/utils/operlog/ ; echo "EXPECT_ZERO=$?"
   ```
   > 若选中的文件被删调用后仍被其它调用/shim 兜住导致不红，换一个「0 命中」候选文件或直接删 `sensitiveKeys` 条目，确保出现红灯证据。**记录「删前 PASS / 删后 FAIL / 恢复 PASS」三段输出**到 SUMMARY。

4. **CI 覆盖确认**：`grep -n "./internal/..." .github/workflows/ci.yml` 命中 `:62-63`；静态断言落在 `./internal/...` 内 → 硬门禁成立。不启动长驻进程、不连真实 DB、不写生产数据、不跑全仓。

**完成判据**：上述 1–4 全部有输出证据；零 `ADMIN_USER`/`ADMIN_PASSWORD`/`DEV_MODE`/`SKIP_LIVE`（除显式设置的 skip 语义验证外）、零 running backend、零 DB 下静态断言真跑且失败会红。
  </action>
</task>

<task id="6">
  <title>写 145-SUMMARY.md（改动清单 / REQ 映射 / 验证结果 / Known Gaps / 未提交声明）</title>
  <files>
    <file>.planning/phases/145-audit-go-audit-01-02-03/145-SUMMARY.md</file>
  </files>
  <action>
写 `145-SUMMARY.md`，至少含：
- **改动清单**：逐文件列出 Task 1–4 的实际 diff 摘要（新增/删除行数、关键函数/常量名），特别标注「两脚本删 `:68-151` 静态段」「凭据门禁下移到 LIVE 段内」「两 driver 设 `cmd.Dir` / 去假绿 skip」「`ci.yml` 是否改动（默认 0 或 1 行注释）」「`scripts/README.md` 登记权威/重复副本」。
- **3 个 REQ 如何满足**：AUDIT-01（静态 3 项断言 Go 单测化 + `go test ./internal/...` 硬门禁 + 逐字迁移 290/17/白名单）、AUDIT-02（零凭据/零 DB/任意 cwd 真跑且失败会红 + LIVE 缺凭据仍 `exit 1`）、AUDIT-03（重复副本登记 + 未做 shell 整体替换）。
- **验证结果**：Task 5 全部命令与输出摘要（含仓库根/包子目录两次 `CALLS`/`HANDLER_FILES` 实测值、反向删调用「PASS→FAIL→PASS」三段证据、`go test -count=1 ./internal/utils/operlog/ ./scripts/e2e/ ./tests/e2e/` 结果）。
- **Known Gaps**：① 两脚本重复副本未物理合并（已登记权威/从属 + 同步约束，合并由 Lead 单独裁定）；② `AGENTS.md` 未改（如需补目录治理条目移交 Lead）；③ 全量 `go build ./...` + `go test ./...` + `tsc` + `lint:ci` + `vitest` 由 Lead 集成提交前统一跑，本 plan 只跑相关包；④ `scripts/e2e/` 与 `tests/e2e/` 仍不在 `ci.yml` 测试范围（LIVE 需凭据，符合禁区「不整体替换」，静态已搬进 `./internal/...`）；⑤ 静态计数口径（是否含 `_test.go`）的决策记录。
- **明确声明**：「**未执行 `git add` / `git commit` / `git stash` / `git checkout`，由 Lead 统一集成提交**」；`commit: false` 已遵守；未改 `.golangci.yml`（归 148）、未碰其它 phase 文件面、未启动长驻进程、未写生产数据。
  </action>
</task>

---

## 4. 验证标准（must_haves，逐条对应 ROADMAP Phase 145 的 5 条成功标准）

| # | ROADMAP 成功标准 | 本 plan 的可观察 must_have | 承载 Task |
|---|---|---|---|
| 1 | 零凭据 / 零 DB / 任意 cwd 下 `go test ./internal/...` 静态 3 项断言真实执行且失败会红 | `handler_coverage_test.go` 用 `runtime.Caller` 锚定 `go.mod` 所在仓库根；`CALLS>=290` / `sensitiveKeys>=17` / handler differential（2 白名单）三项 `t.Fatalf`；`calls>0` + `len(handlerFiles)>0` 反假绿硬断言；仓库根与包子目录两次跑实测 `CALLS≈373` 相等 | 1, 5 |
| 2 | 仓库根与包子目录分别跑同一 driver，`CALLS` 均 `>=290`、目标 handler 文件均存在 | Task 1 锚定 + Task 3 `cmd.Dir` 双保险；Task 5 双 cwd 红绿变异输出 | 1, 3, 5 |
| 3 | 故意移除一处 `operlog.Record` / 删一个 `sensitiveKeys` 条目，CI 的 `go test ./internal/...` 立即红灯；`ci.yml` 真实覆盖 | Task 5 反向变异「PASS→FAIL→PASS」三段证据；`ci.yml:62-63 ./internal/...` 天然含新单测（默认 0 diff 或 1 行注释） | 4, 5 |
| 4 | 两份重复副本（仅 1 行注释不同）「归并为单一权威并登记」；LIVE 缺凭据 `exit 1` 行为不变 | `scripts/README.md` 登记权威（`scripts/operlog_e2e_verify.sh`）+ 重复副本（`tests/e2e/...`）及同步约束；两脚本删静态段后 `SKIP_LIVE=1` → `exit 0`、无凭据不设 `SKIP_LIVE` → `exit 1` 且文案不变；LIVE `:167-329` 一字未改 | 2, 3, 4 |
| 5 | **不做** shell 整体替换为 Go e2e（LIVE 段保留在 shell） | 静态断言确认两脚本 `:167-329`（探活/登录/`assert_logged`/30+ 端点/Summary）仍为 bash；两 driver 仍 `exec.CommandContext("bash", ...)`，仅简化 + 设 `cmd.Dir` | 2, 3 |

**附加 must_have（禁区/账务）**：`git diff --name-only` 不含 `.golangci.yml`；不含 `internal/core/security/**` / `internal/services/asset/**` / `internal/services/operations/**` / `xingran-react-frontend/**`；无任何 git 写操作；`145-SUMMARY.md` 含「未 git commit，由 Lead 统一集成提交」声明；不伪造闭环（PG 双跑 / 79 条人工验证 / LIVE 真机均不在本 phase 伪造）。

---

## 5. 禁止（硬约束，违反即 FAIL）

1. **git 写操作**：`git add` / `git commit` / `git stash` / `git checkout` / `git reset` 一律禁止（4 executor 并行写同一工作区，Lead 统一提交）。
2. **改 `.golangci.yml`**（归 Phase 148）；改 `ci.yml` 仅限 Task 4 的最小确认 diff。
3. **碰其它 phase 文件面**：`internal/core/security/**`（146）、`internal/services/asset/**`（147）、`internal/services/operations/**`（148/149/150）、`xingran-react-frontend/**`（149/150）、`internal/api/router.go`（149/151）、`base.GORMRepository`（151）。
4. **shell 整体替换为 Go e2e**（把 CI 硬门禁与需凭据 LIVE 重新耦合，复现同类缺陷）；**只调门禁顺序**（cwd 假绿，连缓解都不算）。
5. **启动长驻进程**（dev server / DB / watcher 等一律不开）；**写生产数据**；跑全仓 `go test ./...`（由 Lead 集成时统一跑，本 plan 只跑相关包）。
6. **不伪造闭环**：不声称 PG 双跑、不勾 79 条人工验证、不把 LIVE 真机结果写成已验证。
7. 不删任一 `operlog_e2e_verify.sh`（重复副本按 Task 4 登记处置，物理合并移交 Lead）；不互换两 driver 的脚本指向。
