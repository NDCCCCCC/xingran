---
phase: 95-v1-28-ship-v1-29-closeout-audit-p4
plan: 02
subsystem: closeout-audit
tags: [v1.29-closeout, type-check-gate, flaky-fix, seven-gates, milestone-audit, shipped, bookkeeping]
requires:
  - "95-01 校准后口径（45 requirements / CLOSEOUT-01/02 已勾选 / 两 ROADMAP Progress 表 Phase 91-94 已修正）"
  - "94-VERIFICATION tsc 探针先验（全仓 1 处存量错误 useRestoreTask.ts:47）"
  - "95-RESEARCH gate ② 双 flaky 定性（:memory: 并发空库 + 时区日界根因链）"
provides:
  - "type-check gate 真实化（tsc --noEmit -p tsconfig.app.json，3701 文件检查面）+ npm run build 链修复"
  - "gate ② 零失败诚实达成（newNetworkTestEnv 单连接串行化 + TestJbu8003 正午锚定）"
  - ".planning/milestones/v1.29-MILESTONE-AUDIT.md（六段 + 2 新增章节 + 45/45 追溯）"
  - "v1.29 SHIPPED 文档标记（MILESTONES + PROJECT）+ 94-HUMAN-UAT resolved + V130 增补 JOBSTAT-01 + CLOSEOUT-03 记账闭环（两 ROADMAP 45/45）"
affects:
  - "v1.30 规划输入（V130-CANDIDATES：CACHEDEF-01..05 + JOBSTAT-01；type-check:strict 空转注记）"
  - "/gsd-complete-milestone 完整 archive（用户触发）与 push 决策（留用户）"
tech-stack:
  added: []
  patterns:
    - "glebarez :memory: 测试环境单连接串行化 pattern（SetMaxOpenConns(1)，全仓新 pattern）"
    - "sqlite DATE() UTC 取日缺陷的测试侧正午锚定隔离 pattern"
key-files:
  created:
    - ".planning/milestones/v1.29-MILESTONE-AUDIT.md"
    - ".planning/workstreams/milestone/phases/95-v1-28-ship-v1-29-closeout-audit-p4/95-02-SUMMARY.md"
  modified:
    - "xingran-react-frontend/package.json"
    - "xingran-react-frontend/src/pages/network/backups/hooks/useRestoreTask.ts"
    - "internal/api/v1/network/handlers_test_helpers_test.go"
    - "internal/api/v1/api_v1_tail_80_03_test.go"
    - ".planning/MILESTONES.md"
    - ".planning/PROJECT.md"
    - ".planning/REQUIREMENTS.md"
    - ".planning/ROADMAP.md"
    - ".planning/workstreams/milestone/ROADMAP.md"
    - ".planning/workstreams/milestone/STATE.md"
    - ".planning/workstreams/milestone/phases/94-api-p2/94-HUMAN-UAT.md"
decisions:
  - "D-03 修复线成立：实测 1 处错误 < 5 降级线；-p tsconfig.app.json 选型（references 方案证伪、tsc -b 语义混淆否决）；type-check:strict 不修仅注记"
  - "gate ② 双 flaky 全 test-infra 修复零生产文件；生产看板缺陷 JOBSTAT-01 登记 V130 本相不修（job_utils.go:57 零改动守 D-04/D-05 红线）"
  - "gate ② 双口径：CI 三包为主记录 + 字面 ./... 一次性补充留证；跑批裁定记录放行（Pitfall 4 选项 c，锚 e49916b）"
metrics:
  duration: 49min（含七 gate 跑批 ~40min wall time）
  completed: 2026-09-06
---

# Phase 95 Plan 2: v1.29 closeout（D-03 type-check 修复 + flaky 双修复 + 七 gate 跑批 + audit + SHIPPED）Summary

v1.29 收口相执行完毕：type-check gate 从恒真变真 gate（build 链同根因救活）、gate ② 两个 flaky 测试 test-infra 修复后 -count=10 全绿、七 gate 本地全绿逐项留证、v1.29-MILESTONE-AUDIT.md 落盘（commit A `1ef2224`）、SHIPPED 双标记 + 94-HUMAN-UAT 流转 resolved + JOBSTAT-01 V130 登记 + CLOSEOUT-03 记账闭环（commit B `9173cb9`）——v1.29 全部 7 phases / 26 plans / 45 requirements 收口，milestone SHIPPED 2026-09-06。

## What Was Done

### Task 1: D-03 type-check gate 真实化（commit d7e82ff）

- `package.json` scripts.type-check：`tsc --noEmit` → `tsc --noEmit -p tsconfig.app.json`（根 tsconfig `files: []` 空转 → 真实检查面 3701 文件）
- `useRestoreTask.ts:47`：`setTask(result.data)` → `setTask(result.data ?? null)`（TS2345 唯一存量错误）
- 验证链全绿：`npx tsc --noEmit -p tsconfig.app.json` exit 0；`npx tsc -b` exit 0；`npm run type-check` exit 0 耗时 **14s**（非 0.09s 瞬通——恒真断言证伪的判定点）；`npm run build`（tsc -b && vite build）exit 0（115s）
- **降级线未触发**：实测 1 处错误 < 5 文件授权边界，修复线成立

### Task 2: gate ② flaky 双修复（commit e49916b，test-infra 零生产文件）

- `handlers_test_helpers_test.go` newNetworkTestEnv：对底层 `*sql.DB` 调 `SetMaxOpenConns(1)`（附机理注释：glebarez 每连接独立空库 + Phase 93 异步恢复 goroutine 并发取第二连接见空库）；备选 shared-cache DSN 未启用（主方案复验通过无死锁）
- `api_v1_tail_80_03_test.go` TestJbu8003：种子 JobLog CreatedAt 显式 `time.Date(y, m, d, 12, 0, 0, 0, time.Local)` 正午锚定（绕开 job_utils.go:57 本地日界 + glebarez 带偏移写格式 + sqlite DATE() UTC 取日的凌晨窗口陷阱）
- 复验：`-count=10` 10 次全绿（verbose 确认实际执行 10 轮）；同包 count=1 无死锁；TestJbu8003 绿——**复验时本地 ~06:40 +08 正处于原失败窗口（00:00-08:00）内，通过为有效证据**
- job_utils.go 生产代码零改动；生产看板同窗口缺陷登记 V130-CANDIDATES JOBSTAT-01（Task 5 落账）

### Task 3: D-06 七 gate 跑批 + 工作树快照（零代码变更，无 commit）

跑批前快照：HEAD = `e49916bf281b9cdab30aa701e2d9fee063ae7718`；git status --short 快照（4 个未跟踪测试文件 + cov_full_research.out / node_modules/ / xingran-frontend/ 杂物）已写入 audit 报告；跑批后 status 逐字一致，零新增产物（coverage.out 与 coverage/ 均被 .gitignore 覆盖）。

| Gate | 命令 | exit | 关键数字 | 时长 |
|------|------|------|---------|------|
| ① go build | `go build ./...` | 0 | 0 错误 | 14s |
| ② go test（CI 三包，主记录） | `go test -timeout 15m -count=1 -coverprofile=coverage.out -covermode=atomic ./internal/... ./pkg/... ./cmd/...` | 0 | 0 失败 | 540s |
| ⑥ 后端 coverage | `bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold` | 0 | weighted **78.33%**（34111/43546）≥ 77.5；P1 8/8 + P2 10/10 | <1s |
| ③ type-check | `npm run type-check` | 0 | 3701 文件 0 错误，14s 真检查 | 14s |
| ④ lint | `npm run lint` | 0 | 0 errors / 1389 warnings（存量口径） | 32s |
| ⑤ 前端全量 | `npm run test:coverage` | 0 | 554 files / 3800 tests 全过；Statements 59.84% | 695s |
| ⑦ 前端 coverage | `bash .github/scripts/check-frontend-coverage.sh …`（仓库根 cwd） | 0 | 45/45 dirs ≥ floor | <1s |
| ②X 补充留证 | `go test -timeout 15m -count=1 ./...` | 0 | 74 ok / 0 FAIL | 476s |

复确认链（plan verify 命令）GATE6/7/3 二次实测均 exit 0。exit code 全部经重定向后 `$?` 采集（Pitfall 1 防管道掩蔽）。**未 push**（裁定留用户，73-05 先例）。

### Task 4: v1.29-MILESTONE-AUDIT.md（commit A `1ef2224`）

- 六段结构 + 2 新增章节（261 行）：SC 验证（D-01..D-06 践行 + D-08 七项行动 100% 确认表）/ Requirements 追溯 **45/45**（逐项 45 行表）/ Phase 链 SUMMARY 索引（89-95 七行）/ 最终 gate 配置与实测（七 gate 命令+exit+数字 + 跑批锚定 + 本地口径声明）/ V130 转记确认（CACHEDEF-01..05）/ 结论（SHIPPED 判定 + 8 项 known gaps + 文档债核销表）
- 新增章节 1「type-check gate 空转缺陷全程」：发现（94-VERIFICATION 探针）→ 定性（solution-style files:[] 恒真，0.09s/Files:0）→ 处置（D-03 修复线落地 + 降级线未触发）→ 附带收益（build 链同根因救活）→ type-check:strict 注记登记不修 → 教训（gate 语义 ≠ gate 结果）
- 新增章节 2「gate ② flaky 双发现」：并发 :memory: 空库机理 + 时区日界根因链（:57）+ JOBSTAT-01 生产缺陷登记
- D-08 证据实名口径：89 引用 89-01/02/03-SUMMARY.md（238283c..3559626）、90 引用 90-01..04-SUMMARY.md（b51f44c..3a2efe5），引用前 ls 存在性核对通过，全文零占位名（grep 89-0M/90-0M/NN-0M = 0）；本地口径声明（CI 滞后：research 定性时点 168 commits / audit 定稿 179 commits，CI 末次绿跑 2026-09-03 = Phase 88 内容）

### Task 5: SHIPPED 标记 + UAT 流转 + V130 登记 + 记账闭环（commit B `9173cb9`）

- REQUIREMENTS V130-CANDIDATES 增补 **JOBSTAT-01**（段标题注记「2026-09-06 Phase 95 增补」，条目含完整根因链 + v1.30+ 候选定位 + 测试侧已隔离说明）
- 94-HUMAN-UAT 流转（D-05 修复线）：status partial → **resolved**；Test 1 result = D-03 修复线落地；Test 2 result = D-04 裁决接受现状；Summary passed 2 / pending 0
- SHIPPED 双标记（D-09）：MILESTONES.md v1.29 段「🚧 STARTED」→「✅ SHIPPED 2026-09-06」+ Status planning → shipped + Plans TBD → 26 + Delivered 七项补实；PROJECT.md v1.29 Progress 末尾补 Phase 95 ✅ + v1.29 ✅ SHIPPED
- 记账闭环：CLOSEOUT-03 勾选带收口注记；两 ROADMAP Phase 95 行 Pending 0/2 → Complete 2/2（2026-09-06/2026-09-06）+ 两 Total 行 → 45/45 done——与 audit「Requirements 追溯 45/45」声明自洽
- 完整 milestone archive 未做（/gsd-complete-milestone 独立工作流，deferred）

## Deviations from Plan

**1. [流程] Task 1 commit 两次被 commitlint 拒绝后重排通过**
- **Found during:** Task 1 commit 步骤
- **Issue:** 首次 commit message 触发 body-max-line-length（>100 字符）+ subject-case（D-03 大写开头）两条规则拒绝
- **Fix:** subject 改小写开头 + body 重排 ≤100 字符/行后重试通过；未用 --no-verify，hooks（lint-staged 含 npm run type-check）全程执行
- **Commit:** d7e82ff

**2. [Rule 1] asset_columns_schema.json 生成时间戳噪声回退**
- **Found during:** Task 2 go build 后 git status 检查
- **Issue:** `npm run build` 的 prebuild（sync-columns-schema）重写了 `internal/services/system/asset_columns_schema.json` 的 `__generated__` 时间戳（内容零变化），污染任务变更面
- **Fix:** `git checkout -- <该文件>` 单文件回退（任务范围仅两测试文件，gate 跑批零代码变更红线 D-11）
- **Files modified:** 无（回退至 HEAD）

**3. [Scope 补全] 两 ROADMAP 95-02 plan 复选框勾选**
- **Found during:** Task 5 记账闭环编辑
- **Issue:** plan 未明示勾选 `- [ ] 95-02-PLAN.md`，但 plan 完成时该复选框即 stale——与 95-01 deviation 2 同款记账滞后
- **Fix:** 两文件 `- [x] 95-02-PLAN.md`，随 commit B 入库（95-01 先例）
- **Commit:** 9173cb9

**4. [Scope 补全] workstream ROADMAP footer Last updated 刷新**
- **Found during:** Task 5 编辑
- **Issue:** 该文件自身惯例是每次执行刷新 footer（95-01 已刷新至「95-01 完成」），95-02 完成后不刷新即 stale
- **Fix:** footer 首句改「Phase 95 Plan 2 (95-02) 完成，v1.29 全部 7 phases / 45 requirements 收口」，随 commit B 入库
- **Commit:** 9173cb9

**5. [口径校准] audit「168 commits」本地口径数字刷新为 179（保留 168 锚点）**
- **Found during:** Task 4 audit 撰写
- **Issue:** 95-RESEARCH 定性的「本地领先 168 commits」在 95-01/95-02 连续 commit 后实测为 179，照抄 168 会写入不实数字
- **Fix:** audit 写实测 179 并保留「research 定性时点为 168 commits」锚点声明（plan truths 要求的「168 commits」关键词在报告中以研究锚点形式如实呈现）
- **Commit:** 1ef2224

## Verification Results

Plan 全部 automated verify 命令 PASS：

- **T1**: `npm run type-check`（14s）+ `npx tsc -b` exit 0；`grep -c "tsconfig.app.json" package.json` = 1；`grep -c "result.data ?? null" useRestoreTask.ts` = 1；`npm run build` exit 0（115s）
- **T2**: `-count=10` TestBackupHandler_Restore 全绿（verbose 确认 10 轮）+ 同包 count=1 ok + TestJbu8003 PASS；git diff 触碰文件仅两个 *_test.go
- **T3**: 七 gate 逐项 exit 0（表见上文）；复确认链 VERIFY6/7/3 = 0/0/0；跑批前后 git status 逐字一致；`git diff --name-only HEAD` 除已 commit 内容外零新增
- **T4**: audit 存在；`grep -c "^## "` = 8（≥8）；45/45 ×5；89-01-SUMMARY ×4；90-04-SUMMARY ×2；JOBSTAT-01 ×4；168 commits ×1；type-check ×8；占位名 0
- **T5**: JOBSTAT-01 ×2；`- [x] **CLOSEOUT-03**` ×1；UAT status resolved；MILESTONES SHIPPED 标记 ×1；PROJECT SHIPPED 标记 ×1；两 ROADMAP 45/45 各 ×2 + Complete 2/2 各 ×1；UAT pending: 0

## Threat Model Compliance

- **T-95-03（恒真 gate 虚假保证）**: mitigated — type-check 真实化 + 14s 耗时证明非瞬通；audit 章节 1 记录教训
- **T-95-04（audit 误录敏感值）**: mitigated — 报告仅含 exit code / 覆盖率数字 / commit SHA / 测试计数，零 config/env 内容
- **T-95-05（跑批混入未提交代码）**: mitigated — 跑批锚 e49916b + 前后快照逐字一致 + 4 个未跟踪测试文件单列「随跑批执行且全绿」
- **T-95-06（引用不存在证据）**: mitigated — 89/90 实名 SUMMARY ls 核对通过 + 91-94 VERIFICATION 存在性确认 + 零占位名
- **T-95-SC（包安装）**: N/A — 零包安装

## Known Stubs

无。本 plan 无 UI/数据源 stub；唯一代码变更为类型收紧（`?? null`）与 script 配置，零占位实现。

## Self-Check: PASSED

- 全部 artifact 文件存在（v1.29-MILESTONE-AUDIT.md / 两 ROADMAP / REQUIREMENTS / MILESTONES / PROJECT / 94-HUMAN-UAT / STATE.md / SUMMARY 本体）
- 四个任务 commit 均在 git log：d7e82ff（T1）/ e49916b（T2）/ 1ef2224（T4 commit A）/ 9173cb9（T5 commit B）；T3 按设计无 commit（零代码变更）
- 无意外文件删除；coverage 产物未入库（gitignore 覆盖）
