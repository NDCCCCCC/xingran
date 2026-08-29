---
phase: 76-test-doubles
plan: "05"
subsystem: testing
tags: [test-doubles, ast-guard, go-parser, for-testing, isolation-contract, infra-05]

# Dependency graph
requires:
  - 76-02（wave 3 排序：scrapli_wrapper.go 工厂抽取后的全仓最终态受验证；无内容依赖）
  - 76-03（wave 3 排序：28 处闭包替换后的全仓最终态受验证；无内容依赖）
  - 76-04（wave 3 排序：agent/server 测试改写后的全仓最终态受验证；无内容依赖）
provides:
  - TestNoProductionForTestingReferences 全仓 AST 守护（生产 .go 文件引用 *ForTesting 后缀符号即 FAIL，报 file:line）
  - 双检测分支自证证据（CallExpr.Fun=Ident 直接调用形态 + SelectorExpr.Sel 限定引用形态，注毒实验双 FAIL→还原→PASS）
  - 三层隔离契约成文（编译器 _test.go 物理隔离 / ForTesting 命名 / AST 兜底，守护测试头注释）
affects: [phase-77, phase-78（新增 ForTesting 符号自动受守护）]

# Tech tracking
tech-stack:
  added: []  # 零新依赖——go/parser + go/ast + io/fs 全 stdlib
  patterns:
    - "全仓 AST 守护：WalkDir(\"../..\")（go test cwd=包目录）+ 目录 SkipDir 过滤 + parser.ParseFile fail-not-skip"
    - "双防呆：scannedFiles==0 即 Fatal（防 cwd 漂移假绿）+ parse 失败即报错（防静默跳过）——照 status_constants_test.go :297-299 范式"
    - "白名单归一化：WalkDir 路径带 ../.. 前缀，须 filepath.Rel(repoRoot, path) 后再精确匹配"
    - "注毒自证实验：毒株绝不进 commit，FAIL 输出含 file:line 与符号名即为守护有效性证据"

key-files:
  created:
    - internal/device/for_testing_guard_test.go
  modified: []

key-decisions:
  - "白名单匹配用 filepath.Rel 归一化（非 PLAN 字面的 ToSlash(path) 直比）：WalkDir 产出路径带 ../.. 前缀，字面比较永不命中，白名单会失效并让守护对 e2e_helpers.go:34 自身内部调用误报"
  - "目录过滤对 WalkDir 根豁免（path == repoRoot）：根路径 Name()=='..' 自身以点开头，字面套用点前缀 SkipDir 会跳过整个仓库（ scannedFiles==0 防呆会接住，但守护完全不可用）"
  - "执行可选的 SelectorExpr 注毒自证：跨包限定引用形态（portwrite 生产文件 var _ = device.NewPooledConnectionForTesting）补全第二检测分支的证据，与计划示例形态一致"
  - "Task 2 零 commit：注毒实验的交付物是证据而非代码（毒株绝不进 commit），守护两分支首跑即正确捕获、无修复可提交"

requirements-completed: [INFRA-05]

# Metrics
duration: 13min
completed: 2026-08-23
---

# Phase 76 Plan 05: 全仓 ForTesting AST 守护测试 Summary

**仿 status_constants_test.go 的 go/parser 范式新建 TestNoProductionForTestingReferences 全仓 AST 守护（WalkDir("../..")+目录 SkipDir 过滤，712 个生产 .go 文件零违规），双毒株注毒实验实证 CallExpr 直接调用与 SelectorExpr 限定引用两检测分支均 file:line 精确报错，三层隔离契约（编译器/命名/AST）在头注释成文——INFRA-05 闭环，Phase 76 五项 INFRA 需求全部交付**

## Performance

- **Duration:** 13 min（07:57:41Z → 08:10:30Z，含 ~7min 后台全量 CI 本地收尾门）
- **Tasks:** 2/2（Task 1 独立 commit；Task 2 证据型任务零 commit，毒株不进库）
- **Files created:** 1（internal/device/for_testing_guard_test.go，127 行 ≥ 60 门槛）

## Accomplishments

- **INFRA-05 落地**：生产 .go 文件（非 _test.go）中出现 `*ForTesting` 后缀符号的 CallExpr.Fun / SelectorExpr.Sel 引用即守护 FAIL，violation 消息含 `fset.Position` 的 file:line:col 与符号名
- **全仓绿**：712 个生产 .go 文件扫描零违规；目录过滤跳过点前缀（.git、主仓 `.claude/worktrees/` 下 7 份 agent 仓库拷贝、scripts/.archive-*）、vendor、node_modules、xingran-react-frontend、testdata、tests
- **白名单生效实证**：internal/device/e2e_helpers.go（生产文件但合法定义+内部引用 ForTesting 符号）经 filepath.Rel 归一化后精确匹配跳过——零违规本身即证明（否则其 :34 `newScrapliWrapperForTesting(d)` 调用必报）
- **双毒株注毒自证**（详见 Deviations 后的实验记录）：两检测分支各自 FAIL（file:line + 符号名）→ git checkout 还原零残留 → 守护 PASS + device 包全量绿
- **三层隔离契约成文**：守护测试头注释按 e2e_helpers.go 契约文案展开（编译器层 _test.go 物理隔离 / 命名层 ForTesting 后缀生产符号 / AST 层本守护兜底），并说明 ForTesting 工厂跳过连接池簿记/设备锁/可达性检查/凭据解密的安全含义
- **收尾门全绿**：`bash scripts/check-ci-local.sh backend` EXIT=0（golangci-lint 2.12.2 与 CI pin 同版本 0 issues + 全量测试 + coverage gate threshold=55.5% passed + P1 8 包 ≥70% + P2 7+3 包 floor 全过）

## 守护自证实验记录（Task 2 交付物）

**毒株 1（CallExpr.Fun=Ident 直接调用形态）**——internal/device/connection_pool.go 尾部临时追加 `var _ = poisonCallForTesting()` + `func poisonCallForTesting() int { return 0 }`：

```
--- FAIL: TestNoProductionForTestingReferences (0.27s)
    for_testing_guard_test.go:122: production ForTesting reference: ..\..\internal\device\connection_pool.go:644:9: call to poisonCallForTesting
FAIL
```

**毒株 2（SelectorExpr.Sel 限定引用形态，计划可选项，已执行）**——internal/services/portwrite/port_write_service.go 尾部临时追加 `var _ = device.NewPooledConnectionForTesting`（该文件本就 import device 包，跨包限定引用即现实误用形态）：

```
--- FAIL: TestNoProductionForTestingReferences (0.32s)
    for_testing_guard_test.go:122: production ForTesting reference: ..\..\internal\device\connection_pool.go:644:9: call to poisonCallForTesting
    for_testing_guard_test.go:122: production ForTesting reference: ..\..\internal\services\portwrite\port_write_service.go:647:9: reference to NewPooledConnectionForTesting
FAIL
```

**还原**：`git checkout -- internal/device/connection_pool.go internal/services/portwrite/port_write_service.go`，`git diff --quiet HEAD` 对两文件均 exit 0（毒株零残留），重跑守护 PASS + `go test -count=1 ./internal/device/` 全绿。毒株未出现在任何 commit。

## Task Commits

1. **Task 1: for_testing_guard_test.go 全仓 AST 守护实现** - `0ea3ab7` (test)
2. **Task 2: 守护自证注毒实验** - 零 commit（证据型任务：毒株绝不进 commit，守护两分支首跑即正确 FAIL，无代码改动可提交；实验记录即本节与上节内容）

## Files Created/Modified

- `internal/device/for_testing_guard_test.go` - 新建 127 行：TestNoProductionForTestingReferences（WalkDir 全仓扫描 + 目录过滤 + 白名单 + 双检测分支 + 双防呆）+ 三层契约头注释

## Decisions Made

- **白名单 filepath.Rel 归一化**：PLAN/RESEARCH 骨架的字面 `filepath.ToSlash(path) == "internal/device/e2e_helpers.go"` 在 WalkDir("../..") 产出路径（带 `../..` 前缀）下永不命中。用 `filepath.Rel(repoRoot, path)` 归一化后精确匹配，保住「精确匹配」语义（key_links pattern `e2e_helpers\.go` 不受影响）
- **目录过滤根豁免**：WalkDir 回调对根自身的调用中 `d.Name()` 为 `".."`（点开头），字面套用点前缀规则会 SkipDir 整个仓库。加 `path == repoRoot` 豁免；此缺陷本会被 scannedFiles==0 防呆接住（FAIL 而非假绿），但守护完全不可用
- **violation 定位用 `fset.Position(x.Pos())`**：输出 file:line:col 三段（如 `connection_pool.go:644:9`），满足 must_haves「报 file:line」且更精确
- **FuncDecl 天然不误报**（计划预判验证属实）：函数名是声明不进 CallExpr/SelectorExpr 分支，e2e_helpers.go 的两个 ForTesting FuncDecl 若无白名单也不会命中，白名单实际防的是 :34 的内部调用

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] PLAN/RESEARCH 骨架的白名单字面比较永不命中**
- **Found during:** Task 1（实现时静态推演 WalkDir 路径形态，未及运行即发现）
- **Issue:** `filepath.ToSlash(path) == "internal/device/e2e_helpers.go"` 中 path 带 `..\..\` 前缀，归一化只切反斜杠不剥前缀 → 白名单失效 → e2e_helpers.go:34 自身内部调用 `newScrapliWrapperForTesting(d)` 会让守护对全仓唯一的合法文件误报 FAIL
- **Fix:** 白名单改用 `filepath.Rel(repoRoot, path)` 归一化后 ToSlash 精确匹配
- **Files modified:** internal/device/for_testing_guard_test.go
- **Verification:** 全仓扫描零违规（e2e_helpers.go 被正确跳过）
- **Committed in:** 0ea3ab7

**2. [Rule 1 - Bug] 点前缀 SkipDir 会跳过 WalkDir 根自身**
- **Found during:** Task 1（实现时静态推演，与上项同源）
- **Issue:** WalkDir 首次回调的根 DirEntry `Name()` 为 `".."`，`strings.HasPrefix(name, ".")` 为真 → 整个仓库被 SkipDir → scannedFiles==0 防呆 Fatal（守护不可用；幸为假红非假绿）
- **Fix:** 目录过滤对 `path == repoRoot` 豁免，仅过滤子目录
- **Files modified:** internal/device/for_testing_guard_test.go
- **Verification:** 712 个文件被扫描（根未被跳过）
- **Committed in:** 0ea3ab7

---

**Total deviations:** 2 auto-fixed（2 × Rule 1，均为计划骨架实现级缺陷，落地首 commit 内修正）
**Impact on plan:** must_haves 五条 truths 全部不受影响（字面比较与根跳过均为「怎么实现」层面，守护语义与白名单意图不变）；key_links 两处 pattern（`WalkDir`、`e2e_helpers\.go`）grep 均命中。

## Issues Encountered

- commitlint `body-max-line-length`（100 字符）拒绝 Task 1 首版 commit body——改短行后通过（与 76-02 的 subject-case 拒绝同类，非代码问题）。
- 收尾门 lint 段出现一条 warning：golangci-lint 处理时短暂看到并行 agent worktree（`agent-aeeecbd2a0a643e45`）下已消失的文件路径——并行执行环境的瞬时产物，lint 仍报 0 issues 且 PASS，非代码问题。

## Verification Results（plan 收尾门全项）

- `go test -count=1 -run 'TestNoProductionForTestingReferences' ./internal/device/ -v` → **PASS**（712 files scanned, 0 violations）
- 注毒实验 FAIL→还原→PASS 双分支记录 → **PASS**（见「守护自证实验记录」节，输出含 file:line 与符号名）
- `git diff --quiet HEAD -- internal/device/connection_pool.go`（毒株零残留）→ **PASS** exit 0
- `go test -count=1 ./internal/device/` → **PASS**（2.0s 全绿）
- `golangci-lint run --timeout=5m ./internal/device/...` → **PASS** 0 issues
- `bash scripts/check-ci-local.sh backend`（phase 收尾全量）→ **PASS EXIT=0**（lint 0 issues + 全量测试 + coverage gate threshold=55.5% passed + P1 8 包 ≥70% + P2 7+3 floor 全过——55.5 gate 无倒退）
- must_haves truths 逐条：CallExpr/SelectorExpr FAIL+file:line ✓ / 全仓零违规+点前缀跳过 ✓ / scannedFiles==0 防呆 ✓（实现且为根跳过缺陷的接住者）/ 注毒实证 ✓ / 三层契约头注释 ✓

## User Setup Required

None - 纯测试基建（stdlib go/parser），无外部依赖。

## Next Phase Readiness

- **Phase 76 全部 5 项 INFRA 需求闭环**（本 plan 是 wave 3 最后一个）：INFRA-01（miniredis+httpmock）→ 02（Driver 工厂）→ 03（LDAPClientIface）→ 04（re-exec stub）→ 05（AST 守护）全部交付
- 此后任何新增 ForTesting 符号自动受守护；误用会在 `go test ./internal/device/` 即刻 file:line 报错（Phase 77/78 大量新增测试替身期间的防呆底线）
- 运维提示：守护 FAIL 输出若含 `.claude/worktrees` 路径即目录过滤失效信号（T-76-05-03），应查点前缀 SkipDir 分支

## Self-Check: PASSED

- 文件存在：internal/device/for_testing_guard_test.go / 76-05-SUMMARY.md 全部 FOUND
- 提交存在：0ea3ab7（Task 1）在 git log 中 FOUND；Task 2 为证据型任务零 commit（设计使然）
- 工作树干净（毒株零残留已由 git diff --quiet 验证；coverage.out 等生成物被 gitignore 覆盖）
