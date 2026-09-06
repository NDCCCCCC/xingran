# Phase 95: v1.28 SHIP 收口 + v1.29 closeout + audit - Research

**Researched:** 2026-09-06
**Domain:** milestone closeout 文档核对 + 完整 gate 跑批 + milestone audit 报告 + 前端 type-check gate 修复（唯一代码动作）
**Confidence:** HIGH（全部结论基于本会话工具实测，无 WebSearch 依赖项）

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**CLOSEOUT-01/02 核对确认 (Area 1)**
- **D-01: 核对确认而非重写** — MILESTONES.md v1.28 段与 PROJECT.md v1.28 SHIPPED+ARCHIVED 标记均已存在（2026-09-04 收口产物）。plan 95-01 动作 = 逐项核对 CLOSEOUT-01/02 内容清单（45.13% 理由 / 24.87pp 缺口 / Phase 88 备选 / 归档位置登记）+ 补漏（若有缺项才补）。REQUIREMENTS CLOSEOUT-01/02 与 ROADMAP SC-1/SC-2 措辞按现实校准（同 commit，Phase 94 D-01 先例）
- **D-02: frontend-coverage workstream 保留作历史，零移动** — MILESTONES v1.28 段已登记「保留作历史」= REQUIREMENTS「或保留作历史」分支满足；`.archive/` 迁移不做。95-01 仅核对目录完整性（STATE.md/ROADMAP.md/phases 存在性确认）

**Phase 94 HUMAN-UAT 两项裁决 (Area 2)**
- **D-03: type-check gate 空转修复优先** — 修 `npm run type-check` 使 tsc 实际检查源码 + 修复唯一存量错误 `useRestoreTask.ts:47`。**授权边界：若修复暴露 >5 文件新错误或需重设 tsconfig 策略（拆分/重构 config 体系），降级为 audit 报告登记 + V130 候选，零代码改动收口**
- **D-04: 外观级变化接受现状** — asset excel 文件名（后端英文名优先）与错误文案归一：94-03-SUMMARY deviation 6 已登记、测试已锁定、可逆。本相关闭该 UAT 项（裁决 = 接受），audit 报告记录
- **D-05: 94-HUMAN-UAT.md 状态流转** — 两项裁决落档后 UAT frontmatter status: partial → resolved（type-check 修复线）或 status: partial 保留注记（降级线），audit 报告交叉引用

**CLOSEOUT-03 完整 gate + audit (Area 3)**
- **D-06: gate 跑批清单与口径** — ① `go build ./...` 0 错误 ② `go test ./...` 0 失败（全量一次性）③ `npm run type-check`（D-03 修复后为真检查；降级线则如实标注恒真缺陷）④ `npm run lint` 0 errors（1390 warnings pre-existing 口径不变）⑤ `npm run test` 全绿 ⑥ 后端 coverage gate（`bash .github/scripts/check-coverage.sh` 同 CI :77 口径）⑦ 前端 coverage gate（`check-frontend-coverage.sh` 同 CI :183 口径，仓库根执行）。跑批结果逐项写入 audit 报告（命令 + exit code + 关键数字）
- **D-07: v1.29-MILESTONE-AUDIT.md 仿 v1.27 模板** — 落点 `.planning/milestones/v1.29-MILESTONE-AUDIT.md`；章节结构对照 v1.27：Milestone SC 验证（7 项行动 100% + PROJECT.md D-01..D-06 践行）/ Requirements 追溯（41/41）/ Phase 链 SUMMARY 索引（89-95）/ 最终 gate 配置与实测 / 预存缺陷裁决（V130-CANDIDATES 转记确认）/ 结论。**新增章节**：type-check gate 空转缺陷的发现→处置全程记录
- **D-08: 7 项行动确认口径** — 按 PROJECT.md v1.29 Progress 段逐项核对（PAGINATION/TIMEOUTS/CRUD/缓存/config_backup/API 工厂/v1.28 收口），每项引用其 phase 的 VERIFICATION.md/关键 commit 为证据
- **D-09: SC-5 SHIPPED 状态 = MILESTONES.md 文档标记** — v1.29 段标题「🚧 STARTED」→「✅ SHIPPED <date>」+ PROJECT.md v1.29 段 SHIPPED 标记；完整 milestone archive 不在本相

**收口惯例 (Area 4)**
- **D-10: 措辞校准纪律** — 「写入/标记」→「核对确认（已存在）」修订，同 commit 落地（Phase 90 3a2efe5 / 94 D-01 先例）
- **D-11: 文档 commit 纪律** — audit 报告、MILESTONES、PROJECT、REQUIREMENTS、ROADMAP 各文档变更 atomic commit；gate 跑批不产生代码变更（除 D-03 修复线）

### Claude's Discretion

- type-check 修复的具体 tsconfig 方案（references vs -p 指向 vs files 补全；researcher 以 tsc 探针实证选型）→ **本研究已完成实证选型，见 § D-03 修复方案实证**
- D-03 降级线的「>5 文件」判定时机（修复分支试跑后即知）→ **实测 1 处错误，远低于降级线**
- audit 报告各章节的详略粒度与证据引用格式
- gate 跑批的执行顺序与分批 commit 粒度
- 95-01/95-02 的任务切分微调（保持「95-01 文档核对 → 95-02 gate+audit」骨架）
- 7 项行动证据引用的抽查深度
- HUMAN-UAT 状态流转的具体注记格式 → 本研究已捕获 92-HUMAN-UAT resolved 先例格式

### Deferred Ideas (OUT OF SCOPE)

- **milestone 完整 archive**（v1.29-phases/ 迁移）— `/gsd-complete-milestone` 独立工作流
- **V130-CANDIDATES 处置**（WR-01..05 等）— audit 转记确认即可
- **operlog.exclude_paths todo** — 维持 pending
- **frontend-coverage 目录 .archive/ 迁移** — D-02 明示不做
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CLOSEOUT-01 | 核对 MILESTONES.md v1.28 SHIPPED 段（45.13% 理由 + 24.87pp 缺口 + Phase 88 备选 + 归档位置） | § 实测确认 `.planning/MILESTONES.md:32-51` 四件套齐备；动作 = 核对清单逐项 grep；随带核对 REQUIREMENTS CLOSEOUT-01 原文措辞校准（D-01/D-10） |
| CLOSEOUT-02 | 核对 PROJECT.md v1.28 SHIPPED+ARCHIVED 标记 + frontend-coverage workstream 保留作历史 | § 实测确认 `.planning/PROJECT.md:35` 标记存在；workstream 目录 5 项齐备（REQUIREMENTS/ROADMAP/STATE/config.json/phases×7）；零移动 |
| CLOSEOUT-03 | 完整 gate 跑批 + 7 项行动确认 + v1.29-MILESTONE-AUDIT.md 生成 + SHIPPED 标记 | § Validation Architecture 七 gate 命令链（含时长实测）；§ Gate ② 现状警示（2 个失败测试，本研究已定性并给出修复路径）；§ audit 模板字段级映射；§ 证据索引（89/90 无 VERIFICATION.md 的兜底口径）；§ 实测 requirement 总数 45 非 41（措辞校准项） |
</phase_requirements>

## Summary

本相是 v1.29 收尾相：文档核对（CLOSEOUT-01/02 大部分已于 2026-09-04 完成，动作 = 核对确认）、完整 gate 跑批、audit 报告生成、SHIPPED 标记。核心实证工作有三块：(1) D-03 type-check 修复方案经 4 组 tsc 探针实证选型——裸 `tsc --noEmit` 检查 0 文件恒 exit 0（0.09s），真实配置 `-p tsconfig.app.json` 检查 3701 文件（12.97s）全仓恰好 1 处错误（useRestoreTask.ts:47 TS2345），94 verifier 的「仅 1 处」数字复核成立；**附带重大发现：`npm run build`（`tsc -b` 链）当前也因同一处错误 exit 2——修一处 :47 同时救活 type-check 与 build 两个 gate**。(2) 全量 `go test` 实测跑了两轮，**发现 2 个失败测试**（TestBackupHandler_Restore 并发 flake + TestJbu8003_GetJobStatistics 时区日界 flake），均已完成根因定性并给出 test-infra 级修复路径——这是 95-02 规划的阻断级输入。(3) audit 证据侧：89/90 两相**无 VERIFICATION.md**（需 SUMMARY+commit 兜底）；全仓 requirement 实数 **45 项**而非沿用的「41」；BACKUP-CLOSED-01/02 复选框漏勾（实现已存在，纯记账）；两个 ROADMAP Progress 表 stale。

**Primary recommendation:** D-03 走修复线（实测 1 处错误，远低于 >5 降级线）：type-check script 改 `tsc --noEmit -p tsconfig.app.json` + `useRestoreTask.ts:47` 改 `setTask(result.data ?? null)`（单 token 级，同时修复 build 链）；95-02 的 gate ② 先处置 2 个 flaky 测试（均为 test-infra 修复，不触 D-04 生产行为红线）；audit 报告 Requirements 追溯按 **45/45** 撰写并同步校准「41」措辞；本次 gate 实测须在 audit 报告如实记录「v1.29 全程本地验证，168 commits 未推送，CI 状态滞后于 v1.28 末」。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CLOSEOUT-01/02 文档核对 | 规划文档层（.planning/） | — | 纯文档核对 + 措辞校准，零代码 |
| type-check gate 修复（D-03） | Frontend 工具链（package.json + tsconfig） | Frontend 源码（useRestoreTask.ts:47 一行） | gate 语义属构建工具链；存量错误属源码 |
| gate ② flaky 测试处置 | Backend 测试基建（*_test.go） | — | 两个失败均为测试环境缺陷，生产代码零改动 |
| v1.29-MILESTONE-AUDIT.md | 规划文档层（.planning/milestones/） | — | 数据汇编报告，仿 v1.27 模板 |
| HUMAN-UAT 状态流转（D-05） | 规划文档层（94-HUMAN-UAT.md） | — | frontmatter + Tests 段注记 |

## Standard Stack

### Core（本相零新依赖——全部复用既有工具链）

| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| TypeScript | 5.9.3（前端已装） | D-03 真实 type-check（`tsc --noEmit -p tsconfig.app.json`） | 既有 devDependency，探针实测 12.97s/3701 文件 |
| Go | 1.24.5 windows/amd64 | gate ①②（build + test） | 与 go.mod toolchain 一致（ci.yml:36-38 同版本策略） |
| Node.js | 24.19.0 | npm 四件套 | CLAUDE.md 要求 Node 24+ |
| vitest | 4.0.18 | gate ⑤⑦（`npx vitest run` / `npm run test:coverage`） | 既有配置（coverage.include 全 src 口径，v1.28 产物） |
| bash | 4.4.23 (Git Bash msys) | 双 coverage gate 脚本 | check-coverage.sh / check-frontend-coverage.sh 均为 bash+awk 零依赖（D-01 先例）；94-03 已本地跑通 |
| golangci-lint | v2.12.2（CI 侧） | 参考（D-06 清单不含，CI 才跑） | ci.yml:43-49 |

### Alternatives Considered（D-03 方案对比——探针实测数据）

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| **type-check = `tsc --noEmit -p tsconfig.app.json`（推荐）** | type-check = `tsc -b` | `-p` 方案：14.4s，仅查 app（3701 文件），无副产物文件，与 94 verifier 探针完全同款；漏掉 vite.config.ts（223 文件，1.42s），该缺口由 CI :200 `npm run build`（`tsc -b`）兜底。`tsc -b` 方案：~15s，app+node 双覆盖（与 build 共享 tsbuildinfo 缓存），但会在 `node_modules/.tmp/` 写 tsbuildinfo（已 gitignore，无害），且语义上「build mode」与 check gate 职责混淆 |
| — | 根 config 补 references + 保持裸 `tsc --noEmit` | **不可行**：裸 tsc 遇 references 不遍历（这正是当前空转根因）；遍历 references 必须 `--build` 模式，等价于方案二 |
| — | `tsc --build --noEmit` | 冗余：两个子 config 已内置 `noEmit: true`（TS 5.6+ 允许 build 模式 noEmit），加 flag 无增益 |

**Installation:** 无需安装任何包。

## Package Legitimacy Audit

**本相零外部包安装**（纯文档 + 既有工具链 gate 跑批 + 1 行 TS 修复），Package Legitimacy Gate 协议不触发。唯一 script 变更使用已存在的本地 TypeScript 5.9.3。

## Architecture Patterns

### System Architecture Diagram（Phase 95 执行流）

```
95-01 文档核对线（CLOSEOUT-01/02）
  MILESTONES.md:32-51 ──核对──> CLOSEOUT-01 清单（45.13%/24.87pp/Phase 88/归档位置）
  PROJECT.md:35 ───────核对──> CLOSEOUT-02（SHIPPED+ARCHIVED 标记）
  workstreams/frontend-coverage/ ─存在性核对─> D-02 零移动确认
  REQUIREMENTS/ROADMAP 措辞校准 + 记账补漏 ──atomic commit──> 收口

95-02 gate + audit 线（CLOSEOUT-03）
  D-03 修复（先行，gate ③ 前置）:
    package.json type-check script ──改为──> tsc --noEmit -p tsconfig.app.json
    useRestoreTask.ts:47 ──?? null──> 真实配置下 0 错误
    [同根因附带修复] npm run build（tsc -b）──> exit 0
  gate ② 前置处置:
    TestBackupHandler_Restore（并发 flake）──test-infra 修复──> -count=10 绿
    TestJbu8003_GetJobStatistics（时区日界）──seed 修法──> 全天候绿
  七 gate 跑批 ──逐项 exit code + 关键数字──> audit 报告「最终 gate 实测」节
  89-95 证据索引 + 45/45 追溯 + V130 转记 + type-check 缺陷全程记录
  ──> .planning/milestones/v1.29-MILESTONE-AUDIT.md（T4，commit A）
  ──> MILESTONES.md v1.29 段 ✅ SHIPPED + PROJECT.md v1.29 段标记（D-09，T5）
  ──> 94-HUMAN-UAT.md status 流转（D-05，T5）
  ──> JOBSTAT-01 V130 登记 + CLOSEOUT-03 勾选 + 两 ROADMAP Progress 收口 45/45（T5，commit B）
```

### Pattern 1: gate 链合并执行（避免重复跑全量套件）

**What:** D-06 清单 ②⑥ 共享一次 `go test -coverprofile`，⑤⑦ 共享一次 `npm run test:coverage`——CI 正是这么跑的（ci.yml:62→:77 后端、:170→:183 前端）。
**When to use:** 95-02 gate 跑批（省一次全量套件时长，后端 ~8min、前端 coverage ~17min）。
**Example:**

```bash
# 后端（仓库根 cwd）：② 测试全绿 + 产出 coverage.out → ⑥ coverage gate
go test -timeout 15m -count=1 -coverprofile=coverage.out -covermode=atomic \
  ./internal/... ./pkg/... ./cmd/...     # CI 同款三包口径（见 Pitfall 3）
bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold   # CI :77

# 前端（xingran-react-frontend cwd）：⑤ 测试全绿 + 产出 coverage-final.json
npx vitest run --coverage               # 或 npm run test:coverage（~17min 本地实测口径）
# ⑦ coverage gate —— 仓库根基准（CI :182-183 working-directory: .；94-03 T3 注记同款）
cd .. && bash .github/scripts/check-frontend-coverage.sh \
  xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors
```

### Pattern 2: audit 报告六段模板映射（v1.27 → v1.29）

v1.27-MILESTONE-AUDIT.md 实测结构 → v1.29 对应：

| v1.27 段落（实测） | v1.29 对应内容 | 数据源 |
|---|---|---|
| 头部块（Milestone/Phases/Plans/Threshold chain） | 89-95 七相 / 24 plans 执行 / gate 配置链 | STATE.md + ROADMAP Progress |
| `## Milestone SC 验证(SC-a..e)` 逐项表格+证据 | PROJECT.md D-01..D-06 践行 + v1.29 SC 逐项（含 SC-e SHIPPED 自证） | PROJECT.md:19-27 + 各相 VERIFICATION |
| `## Requirements 追溯(19/19)` 表（Req/内容/状态/证据） | **45/45**（非 41，见 Pitfall 6）逐项 | .planning/REQUIREMENTS.md 复选框 + commit |
| `## Phase 链 SUMMARY 索引` 表（Phase/Plans/收口数字/关键 commit） | 89-95 七行 | § 七项行动证据索引（下文） |
| `## 最终 gate 配置`（threshold + 本地验证命令输出） | 七 gate 逐项命令+exit code+数字 | 95-02 实跑 |
| `## BLOCK-05 裁决`（已知缺口豁免格式） | V130-CANDIDATES（CACHEDEF-01..05）转记确认 | REQUIREMENTS.md:122-131 |
| `## QUIRKS`（新增登记） | type-check gate 空转发现→处置全程（**D-07 新增章节**）+ Jbu8003 时区缺陷登记 | 本研究 § Gate ② |
| `## 结论`（SHIPPED 判定 + known gaps + 文档债核销） | 同款 + 文档债核销表（ROADMAP Progress/45→41 措辞等） | 本研究 § 记账补漏清单 |

### Pattern 3: 措辞校准同 commit 纪律（D-10）

先例四期：Phase 90 commit 3a2efe5 / 91 / 92 D-06 / 94 D-01。本相校准对象：REQUIREMENTS CLOSEOUT-01/02「更新/添加」→「核对确认（已存在）」；ROADMAP § Phase 95 SC-1/SC-2 同步；REQUIREMENTS/ROADMAP「41 requirements」→「45 requirements」。

### Anti-Patterns to Avoid

- **把 `npm run build` 排除在 gate 清单外却不记录现状**：build 链当前 exit 2（见 Pitfall 2），audit 必须显式记录（修复线自动解决；降级线则是留给下次 push 的地雷）。
- **在 audit 报告引用不存在的证据文件**：89/90 无 VERIFICATION.md，引用前必须存在性核对（见 Pitfall 7）。
- **降级线触发后仍改代码**：D-03 授权边界明确——>5 文件错误或需重设 tsconfig 策略即零代码改动收口。实测 1 处，不触发。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 覆盖率 gate | 自写覆盖率校验逻辑 | `.github/scripts/check-coverage.sh`（exit 0/1/2/4/5）+ `check-frontend-coverage.sh`（exit 0/1/4/6）+ `.coverage-threshold`（77.5）+ `.coverage-fe-floors`（GLOBAL 3.8 + 45 目录 floor） | CI 同款脚本，exit code 语义已文档化；94-03 已验证本地可跑 |
| audit 报告结构 | 自创章节 | v1.27-MILESTONE-AUDIT.md 六段模板 | 三期先例（v1.19/26/27），D-07 明确仿照 |
| HUMAN-UAT 注记格式 | 自创格式 | 92-HUMAN-UAT.md resolved 先例 | frontmatter status + Tests 段 expected/result + Summary 计数块 |
| type-check 真实化 | 重构 tsconfig 体系 | 改一行 package.json script 指向 tsconfig.app.json | 94 verifier 建议 + 本研究探针实证最小侵入 |

**Key insight:** 本相全部基建（gate 脚本/阈值/模板/先例格式）均已存在，唯一「新」工作是数据汇编 + 两处最小修复；任何超出此范围的重构冲动都应被 D-11「gate 跑批不产生代码变更（除 D-03 修复线）」拦下。

## Runtime State Inventory

不适用（非 rename/refactor/migration 相）。但有一个近亲发现需记录：**工作树未跟踪文件**（见 Pitfall 4）影响 gate 跑批的干净度口径。

## Common Pitfalls

### Pitfall 1: 管道掩蔽 exit code（本研究自身踩中）
**What goes wrong:** `go test ... | tail -15; echo $?` 得到的是 tail 的退出码。本研究第一轮全量跑因此把 FAIL 误读为 exit 0。
**Why it happens:** bash `$?` 取管道最后一个命令的状态。
**How to avoid:** 用 `PIPESTATUS[0]`、或重定向到文件后单独检查 `$?`、或 `set -o pipefail`。
**Warning signs:** 输出含 `FAIL` 字样但 exit=0。

### Pitfall 2: `npm run build` 已断（tsc -b exit 2，pre-existing）
**What goes wrong:** `build = tsc -b && vite build`；`tsc -b` 真实检查（与 type-check 空转相反），当前因 useRestoreTask.ts:47 同一错误 exit 2（探针实测，错误输出逐字一致）。
**Why it happens:** a4bfc71（Phase 93，2026-09-05 20:26）引入 :47 错误后无人跑过 build——v1.29 全程 168 commits 未推送，CI 最后绿跑停留在 2026-09-03（Phase 88 内容），build 断链未及暴露。
**How to avoid:** D-03 修复线的一行 `?? null` 同时救活 type-check 与 build；audit 报告把「build 修复」记为 D-03 的同根因附带收益。
**Warning signs:** 降级线若被触发（实测不会），下次 push CI frontend job 必红于 Build 步。

### Pitfall 3: gate ② 口径分歧——D-06 字面 `go test ./...` vs CI 三包口径
**What goes wrong:** CI 实际跑 `./internal/... ./pkg/... ./cmd/...`（ci.yml:62-63，显式排除根 `tests/` 集成测试——需 live 服务）。D-06 字面 `./...` 会额外编译根 tests/（7 个 go 文件；e2e 与 login_encryption 自带 t.Skip 守卫，tests/scripts/* 为 package main 无测试）。v1.26 先例「全仓 65 packages ok」证明 `./...` 历史上可绿。
**How to avoid:** 主跑批用 CI 三包口径（与 ⑥ coverage gate 的 CI :77 锚定逻辑一致）；若要字面 `./...` 可作补充跑并如实记录。二选一需在 audit 报告写明口径。
**Warning signs:** `./...` 比 CI 慢且多编译 tests/scripts 三个 main 包。

### Pitfall 4: 工作树未跟踪文件污染 gate 跑批
**What goes wrong:** 当前工作树有 4 个未跟踪 test 文件（`internal/models/rpa/rpa_model_methods_test.go` + `internal/pkg/cache/manager_coverage_test.go` + `internal/pkg/system/sysmetrics_{common,windows}_test.go`，补测计划 B1-P2 实验，实测三包全绿但 cache 包 49.8s 较慢）+ `cov_full_research.out`（2.8MB 陈旧产物，8-28）+ 空的根 `node_modules/` + 只含 `src/` 的孤儿目录 `xingran-frontend/`。`go test` 会编译执行这些未提交测试，audit 的 gate 结果混入未提交代码。
**How to avoid:** 三选一（planner 裁量）：(a) 跑批前 `git stash -u` 跑完 pop；(b) 把 4 个测试文件按独立 commit 正式入库（若判定属有效补测投资）；(c) 如实记录「gate 实测含 4 个未跟踪测试文件，已单列标注」。任选其一，audit 报告必须写明跑批时的 commit SHA 与工作树状态。
**Warning signs:** `git status` 非 clean 时声称的「gate 全绿」不可复现。

### Pitfall 5: 裸 `npm run test` 的 watch 模式风险
**What goes wrong:** `package.json` `"test": "vitest"`——TTY 交互环境下 vitest 默认 watch 模式可能挂住执行器。
**How to avoid:** gate ⑤ 用 `npx vitest run`（94-03 PLAN verify 链同款）；coverage 场景直接 `npm run test:coverage`（= `vitest run --coverage`，天然 run-once）。

### Pitfall 6: 「41 requirements」是陈旧计数，实数 45
**What goes wrong:** REQUIREMENTS.md:118 / 两份 ROADMAP / CONTEXT D-07 均写「41 requirements」；实测逐类清点 PAGINATION 11 + TIMEOUTS 8 + CRUD-REUSE 8 + CACHE-UNIFY 5 + BACKUP-CLOSED 5 + API-FACTORY 5 + CLOSEOUT 3 = **45**（grep 实证）。
**Why it happens:** 早期草稿计数沿用，PAGINATION/TIMEOUTS 扩容后未回改。
**How to avoid:** audit 报告 Requirements 追溯按 45/45 撰写；「41→45」列入措辞校准清单（D-10 同 commit）。
**Warning signs:** audit 若照抄 41/41 会与自身追溯表行数自相矛盾。

### Pitfall 7: 89/90 两相无 VERIFICATION.md
**What goes wrong:** D-08 要求「每项引用其 phase 的 VERIFICATION.md/关键 commit」——实测 89-pagination-constants/ 与 90-timeouts-port-protocol-concurrency/ 目录仅有 PLAN+SUMMARY+CONTEXT+DISCUSSION-LOG 全套，无 VERIFICATION.md（91/92/93/94 均有）。
**Why it happens:** 89/90 完成于 2026-09-04（verifier 工作流未逐相执行的早期），STATE.md 引用的「90-VERIFICATION gaps」实为 SUMMARY 内记载。
**How to avoid:** audit 证据索引对 89/90 用**各 plan SUMMARY 实名 + 关键 commit 区间**兜底（D-08 措辞本身已留 `/关键 commit` 分支）：89 引用 `89-01-SUMMARY.md` / `89-02-SUMMARY.md` / `89-03-SUMMARY.md` 三份（commits 238283c..3559626），90 引用 `90-01-SUMMARY.md` / `90-02-SUMMARY.md` / `90-03-SUMMARY.md` / `90-04-SUMMARY.md` 四份（commits b51f44c..3a2efe5）；引用前 `ls` 存在性核对，禁用任何「NN-0M-SUMMARY.md」类占位名。另注意 93-VERIFICATION.md 无 YAML frontmatter（goal-backward 内联格式，结论行 `✅ PHASE GOAL ACHIEVED (5/5)`）。

### Pitfall 8: 前端 coverage gate 的 cwd 基准
**What goes wrong:** `check-frontend-coverage.sh` 与 `.coverage-fe-floors` 在仓库根，`coverage-final.json` 在 xingran-react-frontend/coverage/——CI :182 显式 `working-directory: .` 覆盖 job 级 frontend 目录。
**How to avoid:** 命令链按 94-03 T3 基准：前端目录跑 `npm run test:coverage` → `cd ..` 切仓库根跑 gate（94-VALIDATION.md:53 注记原文）。

### Pitfall 9: 记账 stale 清单（95-01 补漏范围）
**What goes wrong:** 实测三处记账滞后：(a) 根 REQUIREMENTS.md BACKUP-CLOSED-01/02 复选框未勾——但 `gzipCompress`(:603)/`gzipDecompress`(:617) 已实现（93 交付），纯记账遗漏（94-VERIFICATION:131 对 API-FACTORY-01/02/03 同类问题的「记账备注」先例）；(b) 根 `.planning/ROADMAP.md` Progress 表 Phase 93「Pending 0/3」+ Phase 94「Pending 0/3」均 stale（实际均完成；Phase 93 plan 数根表写 3、实际 6）；(c) workstream ROADMAP Progress 表 Phase 93「Pending 0/6」stale（Phase 94 行已由 6a36a0b 修为 Complete 3/3）。v1.27 audit 的「文档债核销」段是同类处置先例。
**How to avoid:** 全部列入 95-01 补漏清单，atomic commit（D-11）。

### Pitfall 10: v1.29 全程未推送，CI 状态滞后
**What goes wrong:** `git rev-list origin/main...HEAD` = 0 ahead / **168 behind→ahead**（本地领先 168 commits）；CI 最后一次 run（33713650858，2026-09-03，success）内容为 Phase 88。「gate 全绿」的 audit 记录是**本地实证**，CI 未见证 v1.29 任何代码。
**How to avoid:** audit 报告「最终 gate 实测」节显式声明本地口径 + CI 状态滞后事实；push 决策按 73-05 先例留给 orchestrator/user（不在本相 scope，CONTEXT 未列入也不应擅自扩大）。

## Gate ② 现状警示（95-02 阻断级发现）：2 个失败测试已定性

两轮全量跑（04:49 与 05:00，`-count=1`，三包口径）均复现同样 2 个失败；单包隔离与 `-count=10` 进一步定性：

### 失败 1：TestBackupHandler_Restore（internal/api/v1/network）——并发/连接池 flake
- **现象**：全量两轮均 FAIL（`no such table: sys_config_restore_task`，INSERT 主键 `task-live` 来自 backup_handler_test.go:281 的 `mutual_exclusion_rejected` subtest）；隔离单跑一次 FAIL 一次 PASS；`-count=10` FAIL。
- **根因**：`newNetworkTestEnv`（handlers_test_helpers_test.go:48）用裸 `sqlite.Open(":memory:")`，**未设 `SetMaxOpenConns(1)`**（全仓 grep 无此先例）。glebarez 驱动下每个新连接 = 独立空库；Phase 93 异步恢复任务在 handler 路径引入后台 goroutine（93-03 D-01 detached context 执行链），其与测试主线程并发取第二个池连接时，看到的是空库 → 「no such table」。全量跑 CPU 忙时窗口大（两轮皆中），安静隔离跑多数绿——2026-09-05 Phase 93 白天验证未见即此机理。
- **修复路径（test-infra，不触 D-04）**：`newNetworkTestEnv` 打开后对 `gormDB` 的 `*sql.DB` 设 `SetMaxOpenConns(1)`（或 DSN 换 `file:network_test?mode=memory&cache=shared`）。注意这是全仓新 pattern（先例 0），executor 修复后必须 `-count=10` 复验 + 抽跑同包其余测试无死锁。

### 失败 2：TestJbu8003_GetJobStatistics（internal/api/v1）——时区日界 flake（含潜在生产缺陷）
- **现象**：`有种子_计数正确` subtest 断言今日成功/失败 = 1 实得 0；凌晨（本地 00:00–08:00 +08）窗口确定性失败（本研究 05:01/05:18 两次复现），白天必然通过——解释了历次白天验证零暴露。
- **根因链（源码级实证）**：`job_utils.go:57` `today := time.Now().Format("2006-01-02")`（本地日界）→ 查询 `DATE(created_at) = ?`；glebarez 驱动默认写时间格式 `"2006-01-02 15:04:05.999999999-07:00"`（驱动 sqlite.go:296/352，**保留 +08:00 偏移**）→ sqlite `DATE()` 对带偏移值**换算 UTC 取日期** → 本地 00:00–08:00 间 UTC 日期是前一天 → 永不匹配 → 计数 0。（注：日界行在 :57；同函数 :52 为 `stats["paused"]` 赋值行，与日界无关。）
- **连带发现（潜在生产缺陷）**：生产 `GetJobStatistics` 在早晨窗口同样会把本地当天前 8 小时的 JobLog 计到「昨日」——今日成功/失败看板少计。修复生产属行为变更（D-04 红线 + 需回归测试），**不在本相修**：按 Phase 92 WR-01..05 先例登记 V130-CANDIDATES（audit 转记），audit 报告「新增发现」节记录。
- **本相修复路径（test-infra）**：种子 JobLog 显式 `CreatedAt: time.Date(y, m, d, 12, 0, 0, 0, time.Local)`（正午本地在 +08 下 UTC 同日，全天候稳定）——单文件单测试改动。

> planner 决断点：两处修复均为 test-infra（符合 v1.29 D-04「不修改业务行为」），但属 gate ②「0 失败」的前置工作，应作为 95-02 独立 task（先修 → `-count=10` 验证 → 再跑全量 gate），失败明细与根因写入 audit 报告（v1.29 独有发现，与 type-check 空转同审计价值）。

## Code Examples

### D-03 修复（探针实证后的推荐落地）

```jsonc
// xingran-react-frontend/package.json — scripts 段单行改动
"type-check": "tsc --noEmit -p tsconfig.app.json"
// type-check:strict 同为空转（裸 tsc），CI 未引用；可选同步改为
// "tsc --noEmit -p tsconfig.app.json --strict" 或保持不动并登记（planner 裁量）
```

```typescript
// xingran-react-frontend/src/pages/network/backups/hooks/useRestoreTask.ts:47
// 现状（TS2345）：BaseResponse<T>.data?: T（src/types/base.ts:11）→ result.data: ConfigRestoreTask | undefined
setTask(result.data);
// 修法（单 token 级；:48 isTerminal(result.data?.status) 已容忍 undefined，无需动）：
setTask(result.data ?? null);
```

修复验证命令（executor 执行）：

```bash
cd xingran-react-frontend
npx tsc --noEmit -p tsconfig.app.json   # 期望 exit 0（3701 files, ~14s）
npx tsc -b                               # 期望 exit 0（build 链同根因一并修复）
npm run type-check                       # 期望 exit 0 且非 0.09s 瞬通
```

### 七 gate 跑批命令清单（95-02 主链，含实测时长）

```bash
# === 后端（仓库根）===
go build ./...                                    # gate ①
go test -timeout 15m -count=1 -coverprofile=coverage.out -covermode=atomic \
  ./internal/... ./pkg/... ./cmd/...               # gate ②（本地实测 ~7-9min；CI 同款）
bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold   # gate ⑥（threshold=77.5；exit 0/1/2/4/5）

# === 前端（xingran-react-frontend/）===
npm run type-check                                 # gate ③（D-03 修复后真检查 ~14s）
npm run lint                                       # gate ④（0 errors / ~1389 warnings 存量）
npm run test:coverage                              # gate ⑤⑦ 合并（94 verifier 实测 ~17min；CI frontend job 15min timeout 内含同步）
# gate ⑦ 仓库根基准：
cd .. && bash .github/scripts/check-frontend-coverage.sh \
  xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors   # exit 0/1/4/6；94 末态：GLOBAL 59.79% ≥ 3.8 + 45/45 dirs + lib 90.48% ≥ 87.2
```

### 94-HUMAN-UAT.md 状态流转注记（D-05，仿 92 resolved 先例）

```markdown
---
status: resolved          # 修复线；降级线则保留 partial 并注记
phase: 94-前端 API 工厂化
source: [94-VERIFICATION.md]
started: 2026-09-06T04:00:00+08:00
updated: <裁决时间戳>      # 更新此行
---

## Current Test

[全部判定完成 <date>（Phase 95 closeout 裁决，用户预授权自主运行）]

## Tests

### 1. 前端 type-check gate 空转（项目级前置缺陷，非 94 引入）
expected: （原文保留）
result: ✅ Phase 95 D-03 修复线落地：type-check script 改 tsc --noEmit -p tsconfig.app.json +
        useRestoreTask.ts:47 setTask(result.data ?? null)；真实配置 0 错误，build 链（tsc -b）一并修复

### 2. 两处已登记外观级变化的产品确认
expected: （原文保留）
result: ✅ Phase 95 D-04 裁决 = 接受现状（94-03-SUMMARY deviation 6 登记 + 测试锁定；ignoreContentDisposition 选项留 deferred）

## Summary

total: 2
passed: 2
...
```

### 七项行动证据索引（D-08 audit 主轴预览）

| v1.29 行动 | Phase | VERIFICATION.md | 关键证据/数字 | 关键 commit |
|-----------|-------|-----------------|--------------|-------------|
| PAGINATION 常量集中化 | 89 (3 plans) | **无**（用各 plan SUMMARY 实名兜底：89-01/02/03-SUMMARY.md） | 3 常量 + `NormalizePagination` 唯一入口 + AST Stability/Count 双锁 | 238283c..3559626 |
| TIMEOUTS/PORT/PROTOCOL/CONCURRENCY | 90 (4 plans) | **无**（用各 plan SUMMARY 实名兜底：90-01..04-SUMMARY.md） | 4 leaf const pkg 10 常量 + AST 锁值 10 tests | b51f44c..3a2efe5 |
| CRUD 复用 base.Repository[T] | 91 (4 plans) | `91-VERIFICATION.md` status: **passed** 8/9 (+1 OVR-91-01) | 11/11 services 迁入 GORMRepository[T] + 生产口径 +408 LOC | 5d0008b..963defe |
| 缓存层三处统一 | 92 (4 plans) | `92-VERIFICATION.md` status: **passed** 10/10（resolved 2026-09-05） | 32 处 GetOrSet 收敛 + 42 处失效归一 + 207 口径 + invariants 锁 | 92-04 收口（git log 区间可查） |
| config_backup TODO 闭环 | 93 (6 plans) | `93-VERIFICATION.md`（无 frontmatter，结论 `✅ PHASE GOAL ACHIEVED` 5/5） | gzip helpers (:603/:617) + RestoreConfig + 异步任务 + 93_NN 19 用例 + Migrate211 | a4bfc71（93-05）等 |
| 前端 API 工厂化 | 94 (3 plans) | `94-VERIFICATION.md` status: human_needed（4/4 truths verified；human 项由本相裁决） | apiFactory 单一权威 + 13 文件矩阵 + 覆盖率 59.79% + review 0 Critical | b3bc745..1771e1d |
| v1.28 收口 | 95-01（本相） | 本相核对清单即证据 | MILESTONES:32-51 + PROJECT:35 + workstream 目录 5 项 | 本相 commit |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 裸 `tsc --noEmit`（Vite 脚手架常见误配——solution-style 根 config 需 build 模式才遍历 references） | `tsc --noEmit -p tsconfig.app.json` 显式指向检查面 | 本相（D-03） | type-check 从恒真 gate 变为真 gate；CI :164 同步受益（下次 push 起） |
| `tsc -b` 需 referenced project `composite: true` | TS 5.6+ 允许 build 模式下 `noEmit` 非 composite 项目（本仓 tsconfig.app/node 即此形态，`tsc -b` 实测可跑并写 tsbuildinfo） | TS 5.6（训练知识 + 本仓实测双证） | `build` 与未来 `tsc -b` 型 type-check 可共存 |

**Deprecated/outdated:**
- 「type-check exit 0 = 类型健康」假设：本相实测证伪（0.09s / Files: 0 恒真）——audit 报告应记录「gate 语义≠gate 结果」教训。

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `setTask(result.data ?? null)` 修复 :47 且无行为副作用（undefined→null 语义等价，`activeTask` 派生链 :74 不受影响） | Code Examples | 低——错误信息逐字指明 undefined 不可赋 null 联合；executor 跑 `tsc -p tsconfig.app.json` + useRestoreTask 既有测试即证 |
| A2 | `newNetworkTestEnv` 加 `SetMaxOpenConns(1)` 消除 restore flake 且不引入死锁 | Gate ② | 中——机理推断（:memory: 每连接独立库）成立但未实装验证；executor 必须以 `-count=10` 复验；若死锁则改用 shared-cache DSN 备选 |
| A3 | JobLog 种子 `CreatedAt` 取本地正午使 Jbu8003 全天候稳定 | Gate ② | 低——sqlite DATE() 对 +08:00 偏移换算 UTC，正午 +08 = 04:00 UTC 同日；但 executor 应在不同时段至少复验一次 |
| A4 | 全量 go test 本地 ~7-9min（两轮实测 04:49→04:56 / ~05:00→05:09）；npm test:coverage 本地 ~17min（94 verifier 单源） | Code Examples | 低——仅影响 plan 工时估算；CI 口径 timeout 15m（后端）/15m（前端 job）为硬上限参考 |
| A5 | 根 `tests/` 集成测试在 `go test ./...` 下自跳过不致失败（SKIP_E2E 默认 + SM2 keys 检查守卫 + v1.26 全仓绿先例） | Pitfall 3 | 低——未在本会话实跑 `./...`；若 planner 选字面口径，executor 首跑即验证 |

## Open Questions (RESOLVED)

> 2026-09-06 修订：五问均已裁定并落入 95-02 计划，以下为决策落点注记。

1. **gate ② 两个 flaky 测试的处置方式** — ✅ RESOLVED：**95-02 T2 必做**（修 + `-count=10` 复验，先于全量 gate）；Jbu8003 生产边界缺陷 **JOBSTAT-01 登记 V130-CANDIDATES**（95-02 T5 落账 REQUIREMENTS，本相不修）
   - What we know: 均为 test-infra 缺陷（非生产行为），修复路径明确（见 § Gate ②），符合 D-04 红线
   - What's unclear: planner 是否将其作为 95-02 必做 task（本研究推荐：必做——否则 CLOSEOUT-03「go test 0 失败」无法诚实达成）以及是否同步把 Jbu8003 生产边界缺陷登记 V130-CANDIDATES
   - Recommendation: 95-02 增加「gate ② 前置处置」task（修 + count=10 验证）；生产缺陷走 V130 转记（与 D-07 V130 段合并）→ **已采纳**
2. **D-06 gate ② 字面 `./...` vs CI 三包口径** — ✅ RESOLVED：**双口径**——CI 三包为主记录（audit 主记录）+ 字面 `./...` 一次性补充留证（95-02 T3 步骤 2/3，audit 报告写明口径声明）
   - What we know: CI 用三包口径并注释了排除原因；根 tests/ 自带守卫
   - What's unclear: audit 报告采用哪个口径作为「0 失败」的正式记录
   - Recommendation: 三包口径为主（与 coverage gate CI 锚定一致 + 省时）；字面 `./...` 可作一次性补充跑 → **已采纳为双口径**
3. **未跟踪文件处置（Pitfall 4）** — ✅ RESOLVED：**记录放行**（选项 c）——跑批前快照 commit SHA + `git status --short`，4 个未跟踪测试文件单列标注「随跑批执行且全绿」，3 项杂物登记；不 stash 不入库（95-02 T3 步骤 1；用户 2026-09-06 修订裁定，覆盖本研究 stash 建议）
   - What we know: 4 个未跟踪测试全绿但未提交；另有 3 项杂物
   - What's unclear: 入库（独立 commit）/ stash / 记录放行，三选一
   - Recommendation: 跑批前 `git stash -u` 保证 audit 复现性 → **未采纳，改为记录放行（audit 报告写明工作树状态保证可复现）**
4. **push/CI 实证是否入 scope** — ✅ RESOLVED：**不入 scope，不 push**——audit 如实记录本地口径 + CI 滞后 168 commits；push 决策留用户（73-05 先例）（95-02 T3 步骤 7）
   - What we know: v1.29 全程未推送（168 commits）；D-06/D-11 均未提及 push
   - What's unclear: 「最终 gate 全绿」是否要求 CI 见证
   - Recommendation: 不入 scope（CONTEXT 边界未列）→ **已采纳**
5. **`type-check:strict` 同类空转是否一并修** — ✅ RESOLVED：**不修**——仅修 CI 引用的 `type-check`；`type-check:strict` 在 audit 注记登记（CI 未引用无消费方，避免扩大 D-11「除 D-03 修复线」解释面）（95-02 T1 步骤 1 + T4 章节 g）
   - What we know: 同为裸 tsc（Files: 0），CI 未引用，无消费方证据
   - Recommendation: 最小方案 = 仅修 CI 引用的 `type-check`；`type-check:strict` 在 audit 注记登记 → **已采纳**

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | gate ①②⑥ | ✓ | 1.24.5 (windows/amd64，与 go.mod toolchain 一致) | — |
| Node.js | gate ③④⑤⑦ | ✓ | 24.19.0 | — |
| npm | 前端四件套 | ✓ | 11.17.0 | — |
| TypeScript | D-03 修复验证 | ✓ | 5.9.3（本地 node_modules） | — |
| bash | 双 coverage gate 脚本 | ✓ | 4.4.23 (Git Bash msys；94-03 本地跑通先例) | — |
| vitest | gate ⑤⑦ | ✓ | 4.0.18 | — |
| gh CLI | CI 状态核对（可选） | ✓ | 2.97.0 | 只影响 audit 的 CI 状态陈述 |

**Missing dependencies with no fallback:** none
**Missing dependencies with fallback:** none

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | 本相验证 = gate 跑批 exit code + 文档核对 grep + audit 报告存在性（无新单测框架引入；既有 go test 1.24.5 + vitest 4.0.18 为被验证对象） |
| Config file | 既有：`.coverage-threshold`（77.5）/ `.coverage-fe-floors`（GLOBAL 3.8 + 45 dirs）/ `go.mod` / `vitest.config.ts` |
| Quick run command | `cd xingran-react-frontend && npx tsc --noEmit -p tsconfig.app.json`（~14s）+ `go build ./...` |
| Full suite command | § Code Examples 七 gate 链（合计本地 ~35-45min：go ~9 + npm coverage ~17 + 其余 ~10） |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CLOSEOUT-01 | MILESTONES v1.28 段四件套齐备 + REQUIREMENTS/ROADMAP 措辞校准落地 | 文档核对（grep 断言） | `grep -c "45.13\|24.87\|Phase 88\|保留作历史" .planning/MILESTONES.md`（期望全命中）+ `grep -c "核对确认" .planning/REQUIREMENTS.md` | ✅ 对象文件均存在 |
| CLOSEOUT-02 | PROJECT.md:35 SHIPPED+ARCHIVED 标记 + workstream 目录完整性 | 文档核对（grep + ls） | `sed -n 35p .planning/PROJECT.md \| grep -c "SHIPPED + ARCHIVED"`；`ls .planning/workstreams/frontend-coverage/{STATE,ROADMAP}.md` | ✅ 均存在（本研究已实测） |
| CLOSEOUT-03① | go build 0 错误 | gate | `go build ./...`（exit 0） | ✅ |
| CLOSEOUT-03② | go test 0 失败 | gate（前置：2 个 flaky 修复 + count=10） | `go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...`（exit 0） | 前置修复 ❌ Wave 0（95-02 T2 内做） |
| CLOSEOUT-03③ | type-check 真检查 0 错误 | gate（前置：D-03 两处修复） | `cd xingran-react-frontend && npm run type-check`（exit 0 且 >1s） | 前置修复 ❌ Wave 0（95-02 T1） |
| CLOSEOUT-03④ | lint 0 errors | gate | `npm run lint`（exit 0；warnings ≈1389 存量） | ✅ |
| CLOSEOUT-03⑤ | 前端全量测试绿 | gate | `npx vitest run`（94 末态 554 文件 / 3800 tests） | ✅ |
| CLOSEOUT-03⑥ | 后端 coverage gate | gate | `bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold`（exit 0，≥77.5%） | ✅ |
| CLOSEOUT-03⑦ | 前端 coverage gate | gate | `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors`（exit 0，45/45 dirs） | ✅ |
| CLOSEOUT-03⑧ | v1.29-MILESTONE-AUDIT.md 生成（六段 + type-check 缺陷章节 + 45/45） | 文档存在性 + 结构 grep | `grep -c "^## " .planning/milestones/v1.29-MILESTONE-AUDIT.md`（≥6）；`grep -c "45/45\|type-check" <audit 文件>` | ❌ 本相交付物（95-02 T4） |
| CLOSEOUT-03⑨ | SHIPPED 标记（D-09）+ 94-HUMAN-UAT 状态流转（D-05）+ CLOSEOUT-03 记账闭环 | 文档核对（grep） | `grep -c "✅ SHIPPED" .planning/MILESTONES.md`（v1.29 段）；`head -3 <94-HUMAN-UAT.md> \| grep -c resolved`；`grep -c "45/45" .planning/ROADMAP.md .planning/workstreams/milestone/ROADMAP.md` | ❌ 本相交付物（95-02 T5） |

### Sampling Rate

- **Per task commit:** 涉及 TS 的 commit → `npx tsc --noEmit -p tsconfig.app.json`；涉及 Go test 的 commit → 对应包 `go test -count=1` + flaky 处置 task 附 `-count=10`
- **Per plan (95-02):** 七 gate 全链一次
- **Phase gate:** 全部 gate exit 0 + audit 文件落盘后才可 `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `newNetworkTestEnv` SetMaxOpenConns(1)（或 shared-cache DSN）— gate ② 前置（flake 修复，先于全量跑）→ 95-02 T2
- [ ] `api_v1_tail_80_03_test.go` JobLog 种子 CreatedAt 正午化 — gate ② 前置（日界修复）→ 95-02 T2
- [ ] `package.json` type-check script + `useRestoreTask.ts:47` — gate ③ 前置（D-03）→ 95-02 T1
- [ ] Framework install: 无需（全部既有）

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | 本相零认证面改动 |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | no | 唯一代码改动（TS 类型收紧 `?? null`）不触碰输入面 |
| V6 Cryptography | no | — |

### Known Threat Patterns for 本相工作面

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 恒真 gate 造成虚假保证（type-check 空转 = 验证防线失效） | Repudiation/Falsification | 本相 D-03 正是在修此模式；audit 报告记录「gate 语义 ≠ gate 结果」教训 |
| audit 报告误录敏感值 | Information Disclosure | 报告仅引用 gate exit code / 覆盖率数字 / commit SHA，不粘贴 config/env 内容（项目 operlog 脱敏纪律同源） |
| gate 跑批混入未提交代码（Pitfall 4） | Tampering（审计可信度） | 跑批锚定干净 commit SHA + 工作树状态声明 |

## Sources

### Primary (HIGH confidence) — 全部为本会话工具实测/直读

**tsc 探针（本研究实测，TS 5.9.3）：**
- 裸 `tsc --noEmit --diagnostics` → Files: 0 / 0.09s / exit 0（空转实证）
- `tsc --noEmit -p tsconfig.app.json --diagnostics` → Files: 3701 / 12.97s / 恰 1 错误（useRestoreTask.ts:47 TS2345 全文捕获）
- `tsc --noEmit -p tsconfig.node.json` → Files: 223 / 1.42s / exit 0
- `tsc -b` → exit 2，错误输出与上逐字一致；tsbuildinfo 写入 node_modules/.tmp/（gitignored）

**go test 实测（go 1.24.5，两轮全量 + 单包隔离 + count=10）：**
- 全量两轮均 2 FAIL：TestBackupHandler_Restore（network）/ TestJbu8003_GetJobStatistics（api/v1）
- 隔离跑时绿时红 + count=10 FAIL → flake 定性
- 根因链源码直读：handlers_test_helpers_test.go:48（:memory: 无 MaxOpenConns）/ job_utils.go:57（本地日界；:52 为 stats[paused] 与日界无关）+ glebarez sqlite.go:295-303/348-353（默认写格式带偏移）/ sqlite DATE() 语义（UTC 取日期）

**关键文件直读：** 95-CONTEXT.md / REQUIREMENTS.md（根+workstream）/ STATE.md / ROADMAP.md（根+workstream）/ PROJECT.md / MILESTONES.md / v1.27-MILESTONE-AUDIT.md / 94-VERIFICATION.md / 94-HUMAN-UAT.md / 92-HUMAN-UAT.md / 91+92+93-VERIFICATION.md frontmatter / 94-03-PLAN.md:219（gate 链）/ 94-03-SUMMARY.md:85,116 / 94-VALIDATION.md:53 / ci.yml（:62-77 后端 gate、:145-200 前端 gate）/ check-coverage.sh 头注（exit 0/1/2/4/5）/ .coverage-threshold（77.5）/ .coverage-fe-floors / tsconfig{,.app,.node}.json / package.json / useRestoreTask.ts / types/base.ts:11（data?: T）/ backup_handler_test.go / api_v1_tail_80_03_test.go / config_backup_service.go:603,617

**git/gh 实证：** a4bfc71 = 2026-09-05 20:26 +0800；`origin/main...HEAD` = 0/168；CI 末次 run 33713650858（2026-09-03 success）；VERIFICATION.md 存在性清单（89/90 缺）；requirement 计数 grep = 45；89/90 各 plan SUMMARY 实名清单（89-01/02/03、90-01..04，checker 复核 ls 确认）

## Metadata

**Confidence breakdown:**
- Standard stack / D-03 选型: HIGH — 四组探针实测 + 源码直读，零假设驱动
- Gate ② 定性: HIGH — 两轮全量复现 + 隔离/count=10 交叉验证 + 根因链每一环源码级实证（修复方案本身为 A2/A3 中置信，executor 需 count=10 复验）
- audit 证据面: HIGH — 全部文件存在性与数字逐项实测（45/41 偏差、89/90 缺 VERIFICATION、BACKUP-CLOSED 记账遗漏均为实锤）
- Pitfalls: HIGH — Pitfall 1 为本研究自身踩中复现

**Research date:** 2026-09-06（同日修订：Pitfall 7 证据实名口径 + job_utils.go 日界锚 :52→:57 + Open Questions 五问裁定落点）
**Valid until:** 本相为 closeout 相，事实全部锚定当前 commit（f173991+ 工作树）；只要不 rebase/重置，数字长期有效。若 95-02 执行前有新 commit 入库，需复核 § 记账补漏清单与 § Gate ② 两测试状态。
