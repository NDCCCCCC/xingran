---
phase: 76-test-doubles
plan: "04"
subsystem: testing
tags: [test-doubles, re-exec, TestHelperProcess, subprocess, os-exec, windows-ci-parity]

# Dependency graph
requires:
  - phase: 76-test-doubles/76-01
    provides: 无直接依赖（76-01 交付 miniredis/httpmock；本 plan 独立交付 re-exec 模式，仅共享 phase 基线）
provides:
  - TestHelperProcess 集中式 re-exec helper（四形态：default 秒退 / stdout-flood 1000 行 / sleep-until-stdin-close / ignore-sigterm linux）
  - helperStubCommand 测试侧构造模式（os.Args[0] + -test.run=^TestHelperProcess$ 锚定 + GO_WANT_HELPER_PROCESS=1 cmd.Env）
  - 组 B 父环境继承驱动法（t.Setenv + runCommand/runCommandOutput 零值 Env 继承，保住生产函数覆盖）
  - subprocess_pgroup_test.go 5 处 "echo" 实参清零（Windows/CI 平台分歧根源消除）
affects: [77-04, 77-05, phase-78-core-reaper, phase-78-block-02-agent-server]

# Tech tracking
tech-stack:
  added: []  # 零新依赖——Go stdlib 官方模式（GOROOT src/os/os_test.go:2475-2516 先例）
  patterns:
    - "re-exec stub: 子进程 = 测试二进制自身（os.Args[0]），任何平台语义一致，替代平台命令（echo/sleep）"
    - "TestHelperProcess 守卫第一行 + 每分支必达 os.Exit（防挂起 + 防 testing 框架解析形态参数）"
    - "形态参数经 '--' 传位置参数，子进程 os.Args[len-1:] 直取，不经 flag 解析"
    - "生产函数内建 cmd 无法设 Env → t.Setenv 经父进程环境继承驱动（Env 零值语义）"
    - "ignore-sigterm 分支 runtime.GOOS 守卫（Windows 走 default）+ sigterm-armed 就绪标记同步"

key-files:
  created:
    - internal/agent/server/subprocess_stub_test.go
  modified:
    - internal/agent/server/subprocess_pgroup_test.go

key-decisions:
  - "TDD 真 RED/GREEN：Task 1 拆两 commit——RED（仅形态自验测试，Default/StdoutFlood 失败于子进程输出 PASS 而非 hello）→ GREEN（补 TestHelperProcess 后全绿）"
  - "ignore-sigterm 增加 sigterm-armed 就绪标记：父进程先读到标记再发 SIGTERM，消除 handler 安装竞态（否则 CI 上 SIGTERM 可能早于 signal.Notify 到达而走默认终止路径，flaky）"
  - "Windows 发信号用 cmd.Process.Signal(syscall.SIGTERM)（跨平台可编译；syscall.Kill 不存在于 windows syscall 包，直接引用会编译失败）"
  - "组 A 构造点不设 cmd.Env（进程不启动，环境无意义）；组 B 执行点一律 t.Setenv（t.Cleanup 自动恢复）"
  - "-race 抽查在本机跳过：windows/amd64 race detector 需 cgo/gcc，本机无 gcc（plan 标注可选，历史上在 WSL 跑）"

patterns-established:
  - "Pattern: 子进程 stub 一律 re-exec 测试二进制（TestHelperProcess 守卫 + -- 形态参数），禁止 exec.Command(平台命令)"
  - "Pattern: 需要驱动生产 runCommand/runCommandOutput 内建 cmd 时用 t.Setenv 走父环境继承，不改生产签名"

requirements-completed: [INFRA-04]

# Metrics
duration: 16min
completed: 2026-08-23
---

# Phase 76 Plan 04: TestHelperProcess re-exec 子进程 stub Summary

**internal/agent/server 落地 Go stdlib 官方 TestHelperProcess re-exec 模式（四形态集中式 helper + 形态自验测试，TDD 真 RED/GREEN），并将 subprocess_pgroup_test.go 5 处 exec.Command("echo") 全部替换为测试二进制自重执行——echo 是 cmd.exe 内建、Windows 依赖 PATH 恰有 Git Bash echo.exe 才碰巧能跑的平台分歧根源就此清零，生产代码（subprocess.go / sysproc_*.go）零改动**

## Performance

- **Duration:** 16 min（07:30:49Z → 07:46:25Z，含一次完整 backend 收尾门 ~10 min）
- **Started:** 2026-08-23T07:30:49Z
- **Completed:** 2026-08-23T07:46:25Z
- **Tasks:** 2/2（Task 1 拆 TDD 两 commit：RED + GREEN）
- **Files modified:** 2（1 新建 helper 测试文件 174 行 ≥ 80 门槛，1 既有测试文件改写）

## Accomplishments

- INFRA-04 全量落地：TestHelperProcess 集中式 helper（default 秒退打印 hello / stdout-flood 1000 行 / sleep-until-stdin-close / ignore-sigterm linux 守卫），守卫第一行 + 每分支必达 os.Exit
- `grep -n '"echo"' internal/agent/server/subprocess_pgroup_test.go` 零命中（自动化验证 exit 1）——5 处替换点按两组改法：组 A（:13/:69 构造点）newCommand 直连、组 B（:29/:40/:58 执行点）t.Setenv 父环境继承 + runCommand/runCommandOutput 生产覆盖保留
- 守卫实证：正常 `go test` 直跑 TestHelperProcess 0.00s 静默通过，整包不挂起不污染输出
- 收尾门全绿：`bash scripts/check-ci-local.sh backend` EXIT=0（lint 0 issues + 全量测试 + coverage gate 56.02% ≥ 55.50%）

## Task Commits

1. **Task 1 RED: 形态自验测试（无 helper，预期失败）** - `298ee21` (test)
2. **Task 1 GREEN: TestHelperProcess 四形态 helper** - `a812b0c` (feat)
3. **Task 2: 5 处 echo 替换（组 A/组 B 两改法）** - `e7f838c` (test)

**Plan metadata:** 本 SUMMARY commit（docs）

## Files Created/Modified

- `internal/agent/server/subprocess_stub_test.go` - 新建 174 行：TestHelperProcess helper（守卫 + 四形态分支 + os.Exit）+ helperStubCommand 构造 helper + 4 个形态自验测试（IgnoreSigterm 非 linux t.Skip）
- `internal/agent/server/subprocess_pgroup_test.go` - 5 处 echo → re-exec：组 A 两处 newCommand(os.Args[0], "-test.run=^TestHelperProcess$", "--", shape)，组 B 三处 t.Setenv + runCommand/runCommandOutput；:40 断言从 non-empty 升级为 Contains "hello"

## Decisions Made

- **TDD 真 RED/GREEN（区别于 76-01 的"交付物即测试"豁免）**：本 plan 的 helper 与自验测试可分离——RED 阶段子进程因 `-test.run=^TestHelperProcess$` 无匹配测试而输出 "PASS"，Default/StdoutFlood 断言失败（实证输出捕获链路真实工作），GREEN 补 helper 后翻转全绿。
- **sigterm-armed 就绪标记**：RESEARCH 骨架的 ignore-sigterm 分支只打印 still-alive；父进程若在 signal.Notify 安装前发 SIGTERM，子进程会走默认终止路径（Wait 返回 signal: terminated）。加 armed 标记后父进程 bufio 读到标记才发信号，确定性同步，CI 不 flaky。
- **发信号 API 选择**：`syscall.Kill` 在 windows syscall 包不存在（编译期失败），改用 `cmd.Process.Signal(syscall.SIGTERM)`——两符号 Windows 均可编译，运行时仅 linux 分支执行。
- **注释措辞避开 `"echo"` 字面量**：文件头说明注释初稿含带引号的 "echo"，会破坏 plan 的字面 grep 零命中验收；改为不带引号措辞。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] 注释中带引号的 "echo" 字面量破坏 grep 零命中验收**
- **Found during:** Task 2（首次 grep 验证）
- **Issue:** 文件头注释写了 `like "echo" — "echo" is ...`，`grep -n '"echo"'` 命中注释行，自动化验证 `! grep` 失败
- **Fix:** 注释改为不带引号措辞（`like echo — a cmd.exe builtin ...`），语义不变
- **Files modified:** internal/agent/server/subprocess_pgroup_test.go
- **Verification:** `grep -n '"echo"'` exit 1（零命中）
- **Committed in:** e7f838c（Task 2 commit 内，提交前已修正）

---

**Total deviations:** 1 auto-fixed（1 × Rule 1）
**Impact on plan:** 措辞级修正，无 scope 膨胀；生产代码零改动（git diff subprocess.go / sysproc_windows.go / sysproc_linux.go 全空）。

## TDD 说明

Task 1 标记 `tdd="true"`，本次为**真 RED/GREEN**（与 76-01 的豁免不同）：RED commit `298ee21` 仅含形态自验测试（helperStubCommand + 4 用例），运行结果 Default FAIL（got "PASS\n" 无 hello）/ StdoutFlood FAIL（1 行 < 1000）/ StdinClose PASS（容错形态，RED 下子进程秒退）/ IgnoreSigterm SKIP（Windows）；GREEN commit `a812b0c` 补 TestHelperProcess 后四用例全绿。git log 门：test(...) → feat(...) 序列完整。plan 非 `type: tdd`，无 plan 级三段门。

## Issues Encountered

- `-race` 可选抽查无法在本机执行：windows/amd64 race detector 需 cgo（gcc），本机 PATH 无 gcc（`cgo: C compiler "gcc" not found`）。plan 标注该验证为可选；历史上 -race 在 WSL 跑（memory 先例），留 CI/linux 环境覆盖。
- 后台收尾门与本会话并行跑 golangci-lint 报 `parallel golangci-lint is running`——非代码问题，门的 lint 段已覆盖（0 issues），未单独重跑。

## Verification Results（plan 收尾门全项）

- `grep -n '"echo"' internal/agent/server/subprocess_pgroup_test.go` → **PASS**（exit 1，零命中）
- `git diff --quiet HEAD -- subprocess.go sysproc_windows.go sysproc_linux.go` → **PASS**（exit 0，生产三文件零改动）
- `go test -count=1 ./internal/agent/server/` → **PASS**（Windows 本地全绿，含既有 TestSubprocess_*/TestRunCommand_* 断言语义保持，:40 断言升级为 Contains "hello"）
- `go test -count=1 -run '^TestHelperProcess$' -v` → **PASS**（0.00s 静默通过，守卫不挂起实证）
- `go test -race -count=1 ./internal/agent/server/` → **SKIP**（本机无 gcc/cgo，plan 可选项，CI linux 覆盖）
- `go build ./...` + `go vet ./internal/agent/server/` → **PASS**
- `bash scripts/check-ci-local.sh backend` → **PASS EXIT=0**（版本守卫 + lint 0 issues + 全量测试 + coverage gate 56.02% ≥ 55.50%，P1 8 包 + P2 10 包 floor 全过）

## User Setup Required

None - 无外部服务配置需求（re-exec 子进程 = 测试二进制自身，零外部命令依赖）。

## Next Phase Readiness

- Phase 77 BLOCK-02（77-04/77-05 agent/server ≥70%）与 Phase 78 BLOCK-03（core reaper）可直接复用 TestHelperProcess 四形态 + helperStubCommand 构造模式
- 74-08 记载的 agent-server "env-branch divergence"（echo 依赖）根源已消除：stub = 测试二进制自身，Windows/CI 同构
- 债务提示：ignore-sigterm 形态在 Windows 无实义（runtime.GOOS 守卫走 default）；若 Phase 78 需要 Windows 等价的"忽略终止"形态（如 CTRL_BREAK_EVENT），届时按 sysproc_*.go 拆分哲学扩展

## Self-Check: PASSED

- 文件存在：subprocess_stub_test.go（174 行）/ subprocess_pgroup_test.go / 76-04-SUMMARY.md 全部 FOUND
- 提交存在：298ee21（RED）/ a812b0c（GREEN）/ e7f838c（Task 2）全部在 git log 中 FOUND
- 工作树干净（收尾门生成的 coverage.out 等已被 gitignore 覆盖，gate 日志在 /tmp 未入库）
