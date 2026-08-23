# Phase 82: 口径修正与治理基建 - Research

**Researched:** 2026-08-23
**Domain:** Vitest 4 全量覆盖率口径 + CI 防倒退 gate（对称后端 v1.26 治理模式）
**Confidence:** HIGH（核心数字全部本地实测复算，Vitest 4 行为经官方文档 + 本地运行双重验证）

## Summary

本 phase 的三件事（口径切换 / 白名单登记 / 4 层 gate）全部有可直接复用的后端 v1.26 参照实现与本地实测数据支撑。研究期间用 CLI 参数覆盖（未改动任何仓库文件）跑了一次全量口径 coverage，实测核对通过：**白名单排除后 statements 3.85%（830/21574，vitest text 摘要显示 3.84%）**，571 个文件进入 per-file 报告（基线 584 = 571 + cad 两目录 13 文件；22602 = 21574 + 1028 完全自洽），与 REQUIREMENTS 基线数字逐位吻合 [VERIFIED: 本地实测 2026-08-23]。

最重要的三个发现：(1) **Vitest 4 自动把测试文件排除在 coverage 报告之外**——即使不配置 `**/*.test.*` exclude，19 个测试文件均未出现在 coverage-final.json 中（实测 TEST files=0），口径不会被测试文件污染；(2) **coverage-final.json 是 5.6MB 单行压缩 JSON 且 Windows 下键为大写盘符+反斜杠绝对路径**——纯 awk 按行解析不可行，gate 脚本推荐「bash 包装 + 内联 `node -e` 扁平化为 TSV + awk 聚合」混合结构（node 是前端工具链固有运行时，仍是零外部依赖）；(3) **D-14 全局阈值 3.8% 的实际余量仅 0.047pp**（实测 3.847%），本地 vs CI 测量漂移可能吃掉余量——但换算下来需新增约 268 条未覆盖语句才会击穿 3.8%，且 D-06 的 −0.5pp ratchet 余量设计已为 per-dir floor 吸收了同类风险；全局阈值 contingency 见 Open Questions Q1。

**Primary recommendation:** 严格对称后端实现——`vitest.config.ts` 加 `coverage.include` + 删 `thresholds` + 白名单 exclude；新建 `check-frontend-coverage.sh`（全局阈值 + per-dir floor，exit 1/4）与 `check-frontend-diff-coverage.sh`（PR diff ≥80%，exit 1）两个 gate 脚本，前者内嵌 ci.yml `frontend` job、后者为独立 PR-only job 复用 coverage artifact；`.coverage-fe-floors` 数据文件 + `.planning/frontend-coverage-baseline.md` 基线文档收口。本研究已产出实测 per-dir 全表，可直接作为 `--init` ratchet 初值（D-06 −0.5pp 已可预计算）。

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**CI 布局与时长预算**
- **D-01:** 全局阈值 gate 与 per-dir floor gate 内嵌在 `ci.yml` 的 `frontend` job 中:将原有 `npm run test` 步骤替换为 `npm run test:coverage`,其后接 gate 步骤。对称后端 `backend` job 布局,避免独立 job 带来的 npm ci + 全量测试重跑。
- **D-02:** PR diff coverage ≥80% gate 通过 artifact 复用实现:新增 `frontend-coverage-diff` job 下载 `frontend` job 上传的 vitest json 产物,基于 `git diff` 计算,零重复测试。
- **D-03:** CI 中保留 `text` / `json` / `html` 三种 reporter,html 目录 zip 后作为 artifact 上传,供本地 debug 下载。
- **D-04:** `frontend` job 的 timeout 保持 15 分钟,先观察全量口径 transform 的实际增量,若不足再在执行阶段末尾调整。

**Per-directory floor 粒度与 ratchet**
- **D-05:** 目录粒度采用 `src/` 下一级目录一个 floor,同时 `pages/` 按二级子目录拆分(operations / system / network / duty / …)。这样与 ROADMAP Phase 83-87 的 wave 验收(例如"pages/operations ≥70%")直接对齐。
- **D-06:** 未达标目录的 ratchet 初值采用保守下界:取当期实测值减去 0.5 个百分点。对称后端 P2_RATCHET 教训(测量噪声会把照抄的实测值变成 gate 失败)。
- **D-07:** per-dir floor 阈值表存储在独立数据文件 `.coverage-fe-floors` 中,文件内同时包含全局阈值行与 per-dir floor 表;gate 脚本读取该文件作为输入。ratchet bump 是纯数据变更,不修改脚本。
- **D-08:** 初始 ratchet 表由 gate 脚本提供 `--init` 模式,从第一次全量口径 coverage json 自动生成(已套用 D-06 的 −0.5pp 余量),人工只审白名单部分。

**白名单登记与同源**
- **D-09:** 前端覆盖率基线文档新建为 `.planning/frontend-coverage-baseline.md`,与后端 `.planning/coverage-baseline.md` 对称并列,包含 ratchet 记录表与白名单登记段。
- **D-10:** 白名单排除的单一真相源是 `xingran-react-frontend/vitest.config.ts` 中的 `coverage.exclude` 数组;gate 脚本做漂移检测,若 cad-editor / cad-elements 仍出现在 coverage json 中则视为配置漂移并失败。
- **D-11:** 白名单复审条件为定量触发:当白名单目录自身语句覆盖率达到 ≥70%(或执行阶段商定的中间 bar)即可启动移除;每个 milestone 收口时强制重审一次。
- **D-12:** 白名单锁死:仅允许 cad-editor 与 cad-elements 两项,未来任何新增必须走 milestone 级显式决策,并重新核算总面积 ≤5%。

**Gate 指标维度**
- **D-13:** 全局阈值 gate 与 per-dir floor gate 只使用 **statements** 维度,与后端 `check-coverage.sh` 加权平均口径对称,也符合 ROADMAP 全篇使用 statements 的表述。
- **D-14:** 全局阈值设定为 **3.8%**(一位小数),白名单后实测约 3.85%,留 0.05pp 最小余量。
- **D-15:** PR diff coverage ≥80% gate 按 **lines** 维度计算:git diff 中的新增/改动行与 vitest json 的 line coverage 对齐(ROADMAP 字面为"新增/改动行覆盖")。
- **D-16:** vitest 原生 `coverage.thresholds` 删除或禁用,全部 gate 逻辑由外部 bash 脚本承担,作为唯一真相源,避免 vitest thresholds 与 `.coverage-fe-floors` 双轨维护导致漂移。

### Claude's Discretion
None — all discussed areas have explicit user decisions.

### Deferred Ideas (OUT OF SCOPE)
- 前端具体测试补齐 —— Phase 83(P0 基建层)、Phase 84(P1 组件层)、Phase 85-87(P2 页面层三波)。
- 公共测试 harness 沉淀 —— Phase 83 成功标准之一。
- 若全量 coverage transform 超出 15 分钟,在执行阶段实测后由 Phase 82 末尾决定是否上调 CI timeout。
- CI 缓存/分片优化:当前不引入,先通过 D-04 观察,必要时未来 phase 处理。
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GOV-01 | vitest coverage 切换全量口径——`coverage.include` 显式圈定 `src/**/*.{ts,tsx}`（Vitest 4 已移除 `coverage.all`），白名单目录同步 exclude，未测试文件计入报告 | 官方迁移指南确认 `coverage.all` 已移除、include 未配置时只报告被加载文件 [VERIFIED: vitest.dev/guide/migration]；本地实测 CLI 覆盖运行 571 文件全部进入报告（含 75 个 0 语句文件以 0% 计入）[VERIFIED: 本地实测]；测试文件被 vitest 自动排除（无需新增 exclude）[VERIFIED: 本地实测] |
| GOV-02 | 前端覆盖率基线落盘（起点 3.67% / 22602 stmts + per-dir 快照 + ratchet 记录），与后端 `.planning/coverage-baseline.md` 模式对称 | 后端文档结构已摘录（表列 schema：date/phase_label/weighted_avg/total_stmts/total_covered/0pct_pkg_count/commit/phase_executor/ratchet_from/ratchet_to + per-dir 快照 + 倒退检查清单）[VERIFIED: 仓库文件]；本研究产出实测 per-dir 全表（含 ROADMAP 遗漏的 9 个小目录）可直接作为快照数据 |
| GOV-03 | CI 前端全局阈值 gate 切到全量口径并 ratchet 只升不降（失败即阻断） | 后端 `check-coverage.sh` 第 1/2 节结构可直接移植（awk 聚合 + threshold 文件 + exit 1）；实测白名单后 3.85% ≥ 3.8% 阈值成立，余量 0.047pp，需新增 ~268 未覆盖 stmts 才击穿 [VERIFIED: 本地实测] |
| GOV-04 | PR diff coverage ≥80% gate（复用后端 74-10 自实现 bash+awk 模式，前端版） | 后端 `check-diff-coverage.sh` 三段式（unified=0 diff 解析 → coverage 块 join → gate）结构可复刻；差异点：istanbul json 为单行 5.6MB（需 node 扁平化）+ statementMap 行区间替代 coverage.out 块 [VERIFIED: 仓库文件 + 本地实测 json 格式]；git pathspec 过滤 `xingran-react-frontend/src/*.ts(x)` + `:(exclude)*.test.*` 实测有效 [VERIFIED: git 实测] |
| GOV-05 | per-directory floor gate（白名单外目录 ≥70%，未达标目录走 ratchet 过渡，对称后端 p1/p2_package_check） | 后端第 3/4 节 per-package floor + ratchet 变量 + 独立 exit code（4/5）模式；前端版改为从 `.coverage-fe-floors` 数据文件读取（D-07，比后端硬编码 ratchet 变量更优）；实测 per-dir 全表已产出，D-06 −0.5pp 初值可预计算 |
| QUAL-02 | 白名单治理——cad-editor 804 + cad-elements 224 排除登记（理由 / 面积 / 复审条件），面积 ≤5% 总语句 | cad-editor 8 文件 / cad-elements 5 文件，stmts 804+224=1028 已通过 21574+1028=22602 算术核对 [VERIFIED: 本地实测]；白名单占比 1028/22602=4.55% ≤5% 成立；D-10 漂移检测（json 中出现 cad 路径即失败）实测可行（本次运行 cad 条目数=0） |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| 覆盖率测量口径（全量/白名单/未测文件 0% 计入） | 前端构建工具层（`vitest.config.ts` coverage 配置） | — | Vitest 4 的 include/exclude 是口径的唯一真相源（D-10），测量行为只能在 provider 配置层定义 |
| 全局阈值 gate + per-dir floor gate | CI 流水线层（ci.yml `frontend` job 内嵌步骤） | 仓库 gate 脚本（`check-frontend-coverage.sh`） | D-01 锁定内嵌同一 job 避免重跑测试；脚本逻辑与 CI 编排分离，本地可复算 |
| PR diff coverage ≥80% gate | CI PR-only job 层（`frontend-coverage-diff`） | 仓库 gate 脚本（`check-frontend-diff-coverage.sh`） | D-02 锁定 artifact 复用零重复测试；PR-only 因 push 无 diff base（对称后端 `coverage-diff` job） |
| 阈值/floor 数据（ratchet 状态） | 仓库数据文件层（`.coverage-fe-floors`） | — | D-07：纯数据变更不改脚本，ratchet bump 走普通 PR |
| ratchet 历史与白名单登记 | 治理文档层（`.planning/frontend-coverage-baseline.md`） | — | D-09：人类可读记录，与后端对称并列 |
| coverage 产物（json/html）传递 | CI artifact 层（upload/download-artifact） | — | D-02/D-03：json 供 diff job 消费，html 供本地 debug |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| vitest | 4.1.10（已安装） | 测试运行器 + coverage provider host | 项目既有，`coverage.include` 是全量口径的唯一开关 [VERIFIED: package.json + npx vitest --version] |
| @vitest/coverage-v8 | 4.1.10（已安装） | V8 coverage provider，输出 istanbul 格式 json | 项目既有；v3.2.0 起 AST remapping 使 v8 报告精度与 istanbul 一致 [CITED: vitest.dev/guide/coverage] |
| bash + GNU awk | runner 自带（本地实测 GNU Awk 5.0.0） | gate 脚本零依赖自实现 | 后端 v1.26 锁定范式（D-01），前端保持一致 [VERIFIED: 本地实测] |
| Node.js | 24（CI `node-version: '24'`；本地 24.19.0） | gate 脚本内联 JSON 扁平化（`node -e`） | 前端工具链固有运行时，不构成新增依赖 [VERIFIED: ci.yml + 本地实测] |

### Supporting

无新增。本 phase **不安装任何 npm 包、不引入任何第三方 coverage action**（后端 74-10 已验证 marketplace 上的 diff-coverage action 不可信，fallback 自实现是既定教训）[VERIFIED: check-diff-coverage.sh 头注释]。

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `node -e` 扁平化 coverage-final.json | 纯 awk 解析 JSON | 不可行：实测 json 为单行压缩 5.6MB，嵌套结构，awk 按行解析无能为力；node 是前端 CI 环境必然存在的运行时 |
| `node -e` 扁平化 | `json-summary` reporter（coverage-summary.json） | 诱惑存在但被 D-03 排除（reporter 锁定 text/json/html 三种）；且 summary 无 statementMap 行区间，无法支撑 diff gate |
| 自实现 diff gate | diff-cover / 第三方 action | 已被后端 74-10 决策链否决（supply-chain + 不可验证），前端沿用 |
| vitest 原生 thresholds | 外部 bash gate | D-16 锁定外部 gate 唯一真相源，vitest thresholds 删除 |

**Installation:**
```bash
# 无。本 phase 零新增依赖。
```

**Version verification:** vitest/`@vitest/coverage-v8` 4.1.10 已在 `xingran-react-frontend/node_modules` 就位并通过 `npx vitest --version` 实测确认（2026-08-23）。

## Package Legitimacy Audit

> 本 phase 不安装任何外部包（config + bash 脚本 + 文档变更），Package Legitimacy Gate 不适用。

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|-------------|-----------|-------------|
| （无新增包） | — | — | — | — | — | N/A |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
PR / push to main
      │
      ▼
┌─ ci.yml ──────────────────────────────────────────────────────┐
│  frontend job（ubuntu, 15min, working-dir: xingran-react-frontend）│
│    npm ci --legacy-peer-deps                                   │
│    → format:check → lint → type-check                          │
│    → npm run test:coverage      ← 替换原 npm run test（D-01）  │
│        vitest --coverage                                       │
│        ├─ coverage.include: src/**/*.{ts,tsx}（全量口径）      │
│        ├─ coverage.exclude: 白名单 cad-* + src/test 等         │
│        ├─ reporter: text / json / html（D-03）                 │
│        └─ 产物: coverage/coverage-final.json + coverage/ (html)│
│    → 全局阈值 + per-dir floor gate（check-frontend-coverage.sh）│
│        ├─ node -e 扁平化 coverage-final.json → TSV             │
│        ├─ awk 聚合全局 statements ≥ .coverage-fe-floors 全局行 │
│        │    不足 → exit 1（CI 红）                             │
│        ├─ awk 按 D-05 粒度聚合 per-dir → floor 表比对          │
│        │    违例 → exit 4（独立 exit code，GOV-05/SC-5）       │
│        └─ 白名单漂移检测：json 含 cad-* 路径 → 失败（D-10）    │
│    → 上传 artifact（json + html，if: always()）                │
│    → npm run build                                             │
│                                                                │
│  frontend-coverage-diff job（PR-only, needs: frontend, D-02）  │
│    checkout（fetch-depth: 0）                                  │
│    → download-artifact（frontend coverage json，零重复测试）   │
│    → check-frontend-diff-coverage.sh                           │
│        ├─ git diff --unified=0 base...HEAD                     │
│        │    × xingran-react-frontend/src/*.ts,*.tsx            │
│        │    − *.test.* / __tests__（新增/改动可执行行）        │
│        ├─ join statementMap 行区间（istanbul json）            │
│        │    行落入 count>0 语句区间 = covered                  │
│        └─ diff coverage < 80% → exit 1 + 未覆盖文件清单        │
└────────────────────────────────────────────────────────────────┘

数据/文档流（ratchet 只升不降）:
  .coverage-fe-floors（全局阈值行 + per-dir floor 表）
      ↑ 纯数据 PR bump（D-07）
  .planning/frontend-coverage-baseline.md（起点 + per-dir 快照 + ratchet 记录表 + 白名单登记，D-09）
      ↑ 同一 commit 内追加（对称后端 D-04 纪律）
```

### Recommended Project Structure

```
.github/
├── workflows/ci.yml                          # [修改] frontend job 加 coverage+gate；新增 frontend-coverage-diff job；严禁触碰 backend job
└── scripts/
    ├── check-coverage.sh                     # [不动] 后端 gate
    ├── check-diff-coverage.sh                # [不动] 后端 diff gate
    ├── check-frontend-coverage.sh            # [新增] 全局阈值 + per-dir floor + 白名单漂移检测
    └── check-frontend-diff-coverage.sh       # [新增] PR diff coverage ≥80%

.coverage-fe-floors                           # [新增] 前端阈值数据文件（全局行 + per-dir floor 表）
.planning/frontend-coverage-baseline.md       # [新增] 前端基线文档（对称 .planning/coverage-baseline.md）

xingran-react-frontend/
├── vitest.config.ts                          # [修改] + coverage.include；+ 白名单 exclude；− thresholds
└── package.json                              # [修改] test:coverage 建议改 "vitest run --coverage"（消除 watch 歧义）
```

### Pattern 1: 后端 gate 脚本结构模板（前端版照抄骨架）

**What:** `check-coverage.sh` 四段式：参数/软跳过 → awk 聚合表 + 内嵌阈值判定（exit 0/1）→ P1 per-package floor（exit 4）→ P2 per-package floor + ratchet 变量（exit 5）。
**When to use:** `check-frontend-coverage.sh` 的结构模板；差异仅在第 1 段之前需要插入 node 扁平化步骤，且 floor 表从数据文件读取而非脚本内硬编码（D-07 优于后端的硬编码 ratchet 变量）。
**关键可复用语义** [VERIFIED: .github/scripts/check-coverage.sh]：
- profile 缺失时软跳过（exit 0）——配合上游 `if: always()` 的 HTML/Upload 步骤
- 聚合表格式 `"%-50s %8d %8d %6.2f%%"` ——本地扫描与 CI 输出 diff-friendly
- awk 数值比较 `exit !(a + 0 >= b + 0)` ——避免字符串比较陷阱
- 防御性收尾：无 `PASS:` 行视为解析失败（exit 1），防空 profile 静默通过
- exit code 语义镜像 `check-status-literals.sh`：0=过 / 1=全局或解析 / 2=用法 / 4=floor 违例

### Pattern 2: diff gate 三段式（git diff 解析 → coverage join → gate）

**What:** `check-diff-coverage.sh` 用 `git diff --unified=0 base...HEAD`（三点 = merge-base）解析 hunk：`+c` 偏移逐行递增，排除空行与注释行；再与 coverage 块区间 join——落在 count>0 块内 = covered，文件不在 profile 中 = 全部按未覆盖计（新文件不给免费通行）。
**When to use:** `check-frontend-diff-coverage.sh` 直接复刻第 1 段（pathspec 换前端）；第 2 段的 coverage.out 块解析换成 istanbul statementMap 区间解析。
**前端版 join 语义** [VERIFIED: 本地实测 statementMap 结构]：
```
statementMap[k] = { start: {line, column}, end: {line, column: 可为 null} }
s[k] = 命中次数
```
行 L 被覆盖 ⟺ ∃k: s[k]>0 且 start.line ≤ L ≤ end.line。与后端「块粒度」语义完全对称。column 为 null 不影响（只用 line）。V8 remap 后 lines 维度本就派生自 statements [CITED: vitest.dev/guide/coverage]，statement 区间推断即 lines 口径的忠实实现。

### Pattern 3: coverage-final.json 扁平化（前端特有步骤）

**What:** 前端版 gate 在 awk 之前先跑一步内联 node 扁平化，把 5.6MB 单行 istanbul JSON 转成 awk 可吃的 TSV。
**When to use:** 两个 gate 脚本共用。实测 json 特征 [VERIFIED: 本地实测]：单行压缩、键为绝对路径（Windows 下 `D:\CODE\...` 大写盘符+反斜杠；CI Linux 下 `/home/runner/work/...` 正斜杠）。
**Example:**
```bash
# bash 包装内（伪代码骨架，Source: 本研究实测验证的模式）
FLAT="$(node -e '
const d = require(require("path").resolve(process.argv[1]));
for (const [p, f] of Object.entries(d)) {
  const s = f.s || {}, sm = f.statementMap || {};
  // 统一路径: 反斜杠→正斜杠, 截取 xingran-react-frontend/src/ 之后
  const n = p.replace(/\\/g, "/");
  const i = n.toLowerCase().indexOf("xingran-react-frontend/src/");
  if (i < 0) continue;
  const rel = n.slice(i);
  let tot = 0, cov = 0; const ranges = [];
  for (const k in s) {
    const m = sm[k]; if (!m) continue;
    tot++; const hit = s[k] > 0; if (hit) cov++;
    ranges.push(m.start.line + "-" + m.end.line + ":" + (hit ? 1 : 0));
  }
  // 每文件一行: 相对路径 TAB stmts TAB covered TAB 区间列表(逗号分隔)
  console.log(rel + "\t" + tot + "\t" + cov + "\t" + ranges.join(","));
}' coverage/coverage-final.json)"
# 其后 awk 按 $1 前缀聚合 per-dir / join diff 行——与后端同构
```

### Pattern 4: 阈值数据文件 + 基线文档双轨（ratchet 纪律）

**What:** 后端用 `.coverage-threshold`（单值 55.5，实测读取）+ `.planning/coverage-baseline.md`（表列 schema：`date / phase_label / weighted_avg / total_stmts / total_covered / 0pct_pkg_count / commit / phase_executor / ratchet_from / ratchet_to`，每 phase 追加一节 + per-package 快照 + 倒退检查清单 + ratchet note）[VERIFIED: 仓库文件]。
**When to use:** 前端版合并为单文件 `.coverage-fe-floors`（全局行 + per-dir 表，D-07）+ `.planning/frontend-coverage-baseline.md`（人类可读，D-09）。**ratchet bump 与基线文档追加必须在同一 commit**（后端 D-04 纪律）。

### Pattern 5: artifact 复用（零重复测试）

**What:** 后端 `coverage-diff` job `needs: backend` + `if: github.event_name == 'pull_request'` + `fetch-depth: 0`（merge-base diff 需全历史）+ `download-artifact@v4` 复用 `coverage.out` [VERIFIED: ci.yml]。
**When to use:** 前端 `frontend-coverage-diff` job 完全对称，artifact 名建议 `frontend-coverage`（与后端 `backend-coverage` 命名对齐），内容 `coverage-final.json` + html 目录。

### Anti-Patterns to Avoid

- **vitest thresholds 与外部 gate 双轨**：D-16 明令删除 vitest thresholds——保留会在口径切换瞬间（3.85% < 旧阈值 24）让 `test:coverage` 直接失败。
- **在 gate 脚本里硬编码 floor/ratchet 值**：后端 P2_RATCHET 硬编码变量导致每次 bump 都要改脚本；D-07 已锁定数据文件，别走回头路。
- **ratchet 初值照抄实测值**：后端 PR #4 round-5 教训（Go patch 版本 instrumentation 漂移、CI/本地分支差异、异步时序方差会把照抄值变成 gate 失败）[VERIFIED: check-coverage.sh L234-241 注释]；D-06 的 −0.5pp 就是为此。
- **触碰 backend job**：ci.yml 是两 workstream 共享编辑区，v1.27 并行改后端覆盖率——本 phase 只增不改前端相关部分 [VERIFIED: STATE.md Blockers]。
- **用字符串比较百分比**：awk 里 `a >= b` 对 "10.0" vs "9.9" 是字符串序；必须 `a+0 >= b+0`（后端范式）。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 覆盖率测量/报告 | 自写 instrumentation | vitest + @vitest/coverage-v8（已有） | V8 runtime coverage + AST remapping，精度等同 istanbul [CITED: vitest.dev/guide/coverage] |
| JSON 解析 | awk 手撕嵌套 JSON | 内联 `node -e`（运行时已有） | json 实测单行 5.6MB 嵌套结构；node 是前端 CI 必然存在的运行时，不算引入依赖 |
| diff hunk 解析 | 从零写 diff 解析器 | 复刻 `check-diff-coverage.sh` 第 1 段 awk（已验证的 unified=0 解析） | 后端已在 PR-only CI 上线运行，边界情况（`\ No newline`、removed line 不推进行号）都已处理 |
| threshold 存储 | 数据库/CI 变量/复杂格式 | 仓库纯文本数据文件（`.coverage-fe-floors`） | ratchet bump = 普通 PR diff，可审计、可 review、CI 与本地同源 |

**Key insight:** 本 phase 的全部「自实现」限定在 gate 判定逻辑（awk 数值比较与聚合）——这是后端验证过的 309 行范围内的小surface；测量、JSON 解析、diff 解析三类复杂度分别由 vitest、node、既有 awk 脚本承担，不新造轮子。

## Common Pitfalls

### Pitfall 1: 测试文件是否会污染口径——已实证不会，但仅限 vitest 自动行为
**What goes wrong:** 担心 `coverage.include: ['src/**/*.{ts,tsx}']` 会把 19 个测试文件（多数在 `src/**/__tests__/` 或 `*.test.*` 命名，散布在 lib/hooks/utils/constants/design-system/pages 内）以近 100% 覆盖率计入分子分母。
**实测结论:** 本地全量口径运行中 coverage-final.json 的 571 个文件里 **0 个测试文件**——vitest 自动排除测试文件，无需在 `coverage.exclude` 手动加 `**/*.test.*` / `**/__tests__/**` [VERIFIED: 本地实测 2026-08-23]。但 `src/test/`（setup.ts 所在）仍需显式 exclude（现状 config 已有）。
**Warning signs:** 若未来 json 中出现 `.test.` 路径，白名单漂移检测同款逻辑可捕获。

### Pitfall 2: D-14 全局阈值 3.8% 余量仅 0.047pp
**What goes wrong:** 实测精确值 830/21574 = **3.847%**（vitest text 摘要显示 3.84%，json 复算 3.85%——reporter 舍入口径不一致）。D-14 文档假设「约 3.85% 留 0.05pp」，实际余量 0.047pp。
**Why it happens:** 本地 Windows vs CI Linux 测量漂移（后端经历过 CI/本地 ±0.5pp 级分支差异）可能让 CI 实测落在 3.80 之下。
**How to avoid:** 量化风险：击穿 3.8% 需分母增至 >830/0.038≈21842，即新增 ~268 条未覆盖语句——短期安全。执行顺序上先落口径切换、在 CI 上实测一次全量数字，再以 CI 读数校准 `.coverage-fe-floors` 全局行（纯数据变更）。若 CI 实测 <3.8，阈值行写 CI 实测值（仍满足「失败即阻断」语义）。
**Warning signs:** 首次 CI 运行 frontend gate 即红且报 3.7x%。

### Pitfall 3: coverage-final.json 路径形态跨平台不一致
**What goes wrong:** Windows 键为 `D:\CODE\ClaudeCode\...\src\App.tsx`（大写盘符、反斜杠）；CI Linux 为 `/home/runner/work/...`。gate 脚本若假设固定前缀或固定分隔符，本地与 CI 必有一边 join 失败。
**How to avoid:** 扁平化步骤统一 `replace(/\\/g,"/")` + `toLowerCase().indexOf("xingran-react-frontend/src/")` 截取（本研究脚本已验证此规范化在 Windows 实测 json 上工作）。
**Warning signs:** gate 输出「所有文件 not found in profile」。

### Pitfall 4: frontend job 的 working-directory 陷阱
**What goes wrong:** `frontend` job 有 `defaults.run.working-directory: xingran-react-frontend`——新增 `run:` 步骤里的脚本路径 `bash .github/scripts/...` 会相对 frontend 目录解析而找不到文件；且 `upload-artifact` 的 `path:` 相对 workspace root 而非 working-directory。
**How to avoid:** gate 步骤要么 `working-directory: .` 覆盖默认值后从 root 调 `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors`，要么用 `../.github/scripts/` 相对路径；artifact path 写 `xingran-react-frontend/coverage/...`。
**Warning signs:** CI 报 `No such file or directory: .github/scripts/...`。

### Pitfall 5: `test:coverage` 脚本是 watch 语义
**What goes wrong:** package.json 现值 `"test:coverage": "vitest --coverage"`——非 CI 环境（本地手跑 gate 流程）会进入 watch 模式挂住。
**How to avoid:** 改为 `"vitest run --coverage"`（CI 里 CI=true 也能跑，行为更确定）。与 D-01 的步骤替换兼容，属口径切换的自然组成部分。

### Pitfall 6: include 圈定后旧 exclude 大部分失效但非全部
**What goes wrong:** 现有 exclude 含 `node_modules/`、`**/*.config.*`、`dist/`——官方迁移指南明确：include 圈定到 src 后这些 root 级模式无需保留（「No need to define root level *.config.ts files or node_modules, as we didn't add those in include」）[CITED: vitest.dev/guide/migration]。但 `src/test/`、`**/*.d.ts`（src 内有 3 个：types/externals.d.ts、types/global.d.ts、utils/sm-crypto.d.ts）必须保留，`**/mockData/**` 实测当前 src 内无此目录（可留可删）。
**How to avoid:** 新 exclude 数组 = `['src/test/', '**/*.d.ts', 'src/components/cad-editor/**', 'src/components/cad-elements/**']`（+ 按需 mockData）；用首次全量运行的 per-file 列表验证无意外文件混入。

### Pitfall 7: SC-1「584 个文件」与白名单 exclude 的自相矛盾
**What goes wrong:** SC-1 要求「白名单排除后约 3.85%，584 个 src 文件全部出现在 per-file 报告中」——但 584 = 571 + cad 两目录 13 文件；白名单 exclude 落地后 per-file 报告是 **571** 个文件（本次实测）。两句话不可能同时字面成立。
**How to avoid:** planner 把验证表述拆成两步：口径切换生效瞬间（未排白名单）报告 584 文件；白名单落地后报告 571 文件且 statements 3.85%。VALIDATION.md 按两步分别断言（见 Validation Architecture）。
**Warning signs:** verifier 死抠 584 字面值。

### Pitfall 8: D-05 粒度下 ROADMAP 未列目录的 floor 归属缺口
**What goes wrong:** D-05 粒度 = src 一级目录 + pages 二级拆分。实测发现 ROADMAP per-dir 清单遗漏 9 个实测存在的目录：pages/ad(37)、pages/profile(87)、pages/settings(65)、src/api(8)、(src 根散文件 App.tsx/main.tsx 13 stmts)、components/asset(81)、components/DeptTree(64)、components/IconSelect(56) 及 components 顶层散文件(135, 3 文件)。合计 21 stmts 在 ROADMAP 的 22581 总和中无主（22602−22581=21 恰为 api 8 + 根 13）。「白名单外无无主面积」要求数学成立，floor 表必须显式覆盖它们。
**How to avoid:** per-dir floor 表按 D-05 粒度即「components 整体一个 floor（3958 stmts，含全部子目录）+ pages 每个二级目录一个 floor + 其余每个一级目录一个 floor + `(src root)`/api 显式条目」。pages 实测 16 个二级目录全部入表。ROADMAP 的 login 62 / 零散 224 与实测 95 / 189 有小差异——以 gate 脚本可复算的实测表为准（见 Code Examples 全表）。

### Pitfall 9: 测试失败时 coverage 产物可能不生成
**What goes wrong:** vitest 默认测试失败不产出 coverage 报告（`coverage.reportOnFailure` 默认 false [ASSUMED]）——Test 步骤红了，后续 gate 步骤（非 always）不跑、artifact 步骤（if: always()）拿不到 json。
**How to avoid:** 与后端同款语义：gate 脚本对缺失 profile 软跳过（exit 0 + stderr 提示），job 由 Test 步骤本身的红来阻断。html/json artifact 拿不到属可接受退化（测试失败时开发者先修测试）。

### Pitfall 10: diff gate 的新文件免费通行问题——已被全量口径天然解决
**What goes wrong:** 后端 diff gate 要专门处理「文件不在 profile 中 = 全部按未覆盖计」（新 .go 文件不给免费通行）。
**前端版结论:** 全量口径下所有 src 文件都在 coverage-final.json 中（含 0% 文件），「文件缺失」分支几乎不会触发；但防御性保留该分支仍有价值（diff 涉及 json 生成后被删的文件等边缘情况）。改动行不落入任何 statementMap 区间（纯类型/声明/注释行）不计入分母——与后端「executable 才计分母」语义一致。

## Code Examples

### 目标态 vitest.config.ts（coverage 段）

```typescript
// Source: 官方迁移指南推荐模式 (vitest.dev/guide/migration) + 本研究本地实测验证
coverage: {
  provider: "v8",
  reporter: ["text", "json", "html"],   // D-03: 三种 reporter 保留
  // GOV-01: Vitest 4 移除 coverage.all 后，include 是全量口径唯一开关。
  // 未配置时只报告被 import 文件（旧口径 24.58% 失真的来源）。
  include: ["src/**/*.{ts,tsx}"],
  exclude: [
    "src/test/",                          // setup 目录（保留）
    "**/*.d.ts",                          // src 内 3 个 .d.ts（保留）
    // QUAL-02 白名单（D-10: 单一真相源在此，gate 脚本做漂移检测）
    "src/components/cad-editor/**",       // 804 stmts / 8 文件（实测核对）
    "src/components/cad-elements/**",     // 224 stmts / 5 文件（实测核对）
  ],
  // D-16: thresholds 整段删除——gate 唯一真相源是外部脚本 + .coverage-fe-floors
},
```

### 实测 per-dir 全表（白名单外，2026-08-23 本地实测，gate floor 表与基线快照的数据源）

| 目录（D-05 粒度） | stmts | covered | 实测 % | 文件数 | D-06 ratchet 初值（−0.5pp，下限 0） |
|---|---:|---:|---:|---:|---:|
| pages/operations | 3611 | 0 | 0.00 | 76 | 0.0 |
| pages/system | 2203 | 60 | 2.72 | 56 | 2.2 |
| pages/network | 1962 | 61 | 3.11 | 61 | 2.6 |
| pages/duty | 1190 | 0 | 0.00 | 40 | 0.0 |
| pages/ad-domain | 1082 | 0 | 0.00 | 8 | 0.0 |
| pages/monitor | 627 | 0 | 0.00 | 26 | 0.0 |
| pages/vdi | 567 | 0 | 0.00 | 4 | 0.0 |
| pages/workorder | 551 | 0 | 0.00 | 14 | 0.0 |
| pages/asset | 475 | 0 | 0.00 | 7 | 0.0 |
| pages/knowledge | 262 | 0 | 0.00 | 7 | 0.0 |
| pages/dashboard-system | 203 | 0 | 0.00 | 7 | 0.0 |
| pages/my-notices | 127 | 0 | 0.00 | 2 | 0.0 |
| pages/login | 95 | 59 | 62.11 | 1 | 61.7 |
| pages/profile | 87 | 0 | 0.00 | 1 | 0.0 |
| pages/settings | 65 | 54 | 83.08 | 1 | 82.6 |
| pages/ad | 37 | 0 | 0.00 | 1 | 0.0 |
| components（含全部子目录） | 3958 | 215 | 5.43 | 152 | 4.9 |
| hooks | 1050 | 85 | 8.10 | 27 | 7.6 |
| lib | 1042 | 114 | 10.94 | 20 | 10.4 |
| utils | 950 | 78 | 8.21 | 23 | 7.7 |
| store | 589 | 28 | 4.75 | 9 | 4.3 |
| router | 272 | 0 | 0.00 | 5 | 0.0 |
| services | 238 | 9 | 3.78 | 13 | 3.3 |
| design-system | 194 | 30 | 15.46 | 13 | 15.0 |
| constants | 84 | 33 | 39.29 | 6 | 38.8 |
| (src root)（App.tsx + main.tsx） | 13 | 0 | 0.00 | 2 | 0.0 |
| api | 8 | 0 | 0.00 | 1 | 0.0 |
| types | 32 | 4 | 12.50 | 22 | 12.0 |
| **合计（白名单外）** | **21574** | **830** | **3.85** | **571** | 全局阈值 3.8（D-14） |

（components 细分参考——dashboard 1068/shared 892(2.80%)/layout 507/network 324(50.62%)/CronSelector 316/captcha 154/reconciliation 144(18.06%)/operations 149/顶层散件 135/asset 81/DeptTree 64/IconSelect 56/其余 91——供 Phase 84 与基线快照用，非 gate 粒度）

### `.coverage-fe-floors` 建议格式（D-07）

```bash
# 全局阈值行（statements, 一位小数）— D-14
GLOBAL 3.8
# per-dir floor 表: 目录 <TAB> floor(%) — 目标 70.0，过渡期 = D-06 ratchet 初值
# ratchet bump = 改这里的数字，不动脚本（D-07）
pages/operations	0.0
pages/login	61.7
components	4.9
lib	10.4
...
```

### check-frontend-coverage.sh 骨架（关键差异点）

```bash
#!/usr/bin/env bash
# 仿 .github/scripts/check-coverage.sh 头注释风格 + exit code 语义
# Usage: check-frontend-coverage.sh <coverage-final.json> <floors-file>
# Exit codes: 0 过 / 1 全局阈值或解析失败 / 2 用法 / 4 per-dir floor 违例 / 6 白名单漂移(D-10)
set -euo pipefail
# ... 参数检查 + profile 缺失软跳过（照抄后端 L46-85）...
# 扁平化（Pattern 3 的 node -e）→ $FLAT
# 全局 gate: awk 'BEGIN{...} {split($1,p,"/"); ...聚合 stmts/covered}' <<< "$FLAT"
#   对称后端 awk L92-131（含 PASS:/FAIL: 行 + exit 1）
# per-dir gate: 从 floors-file 读表（cut -f1/2），逐目录 awk 聚合比对 → 违例计数 → exit 4
# 白名单漂移: grep -Ei 'cad-editor|cad-elements' <<< "$FLAT" 非空 → exit 6
```

### check-frontend-diff-coverage.sh 第 1 段（pathspec 替换）

```bash
# Source: 后端 check-diff-coverage.sh L87-106 逐字复刻，仅换 pathspec（本研究 git 实测验证）
git diff --unified=0 "${BASE_REF}...HEAD" \
  -- 'xingran-react-frontend/src/*.ts' 'xingran-react-frontend/src/*.tsx' \
  ':(exclude)xingran-react-frontend/src/*.test.*' \
  ':(exclude)xingran-react-frontend/src/**/__tests__/**' | awk '
  /^\+\+\+ b\// { file = substr($0, 7); in_hunk = 0; next }
  /^@@ /       { match($0, /\+[0-9]+/); lineno = substr($0, RSTART+1, RLENGTH-1)+0; in_hunk = 1; next }
  in_hunk && /^\+/ {
    line = substr($0, 2)
    # 前端注释语义: // 与 /* 与 * 开头（JSDoc 延续行）均非可执行
    if (line !~ /^[[:space:]]*$/ && line !~ /^[[:space:]]*\/\// && line !~ /^[[:space:]]*\/\*/ && line !~ /^[[:space:]]*\*/) {
      printf "%s\t%d\n", file, lineno
    }
    lineno++; next
  }
  in_hunk && /^-/ { next }
  in_hunk && /^\\/ { next }
'
# git 实测: pathspec 单星跨目录匹配成立（src/*.tsx 命中 310 文件）；:(exclude) 生效（297→288）
# 第 2 段 join: 扁平化 TSV 的区间列（Pattern 3）替代后端 coverage.out 块解析，语义同构
```

### ci.yml frontend job 目标态（D-01/D-02/D-03）

```yaml
# frontend job steps 追加/替换（严禁触碰 backend job —— 共享编辑区纪律）:
      - name: Test (coverage)          # 替换原 Test 步骤
        run: npm run test:coverage     # 建议 package.json 同步改为 vitest run --coverage

      - name: Coverage gate            # 全局阈值 + per-dir floor + 白名单漂移
        working-directory: .           # 覆盖 job 级 working-directory（Pitfall 4）
        run: bash .github/scripts/check-frontend-coverage.sh \
                xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors

      - name: Upload coverage artifact
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: frontend-coverage
          path: xingran-react-frontend/coverage/   # upload-artifact 自动 zip（含 json + html，D-03）
          retention-days: 30

  frontend-coverage-diff:              # PR-only，对称后端 coverage-diff job（D-02）
    runs-on: ubuntu-latest
    timeout-minutes: 10
    needs: frontend
    if: github.event_name == 'pull_request'
    steps:
      - uses: actions/checkout@v7
        with: { fetch-depth: 0 }       # merge-base 三点 diff 需全历史
      - uses: actions/download-artifact@v4
        with: { name: frontend-coverage }
      - run: bash .github/scripts/check-frontend-diff-coverage.sh \
                xingran-react-frontend/coverage/coverage-final.json origin/${{ github.base_ref }} 80
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `coverage.all: true`（默认全文件入报告） | `coverage.all` 选项整体移除；未配 include 只报被加载文件 | Vitest 4.0（项目已在 4.1.10） | 本 phase 的存在理由：旧口径 24.58% 是失真值，必须显式 include [VERIFIED: vitest.dev/guide/migration] |
| v8-to-istanbul remapping（误差大） | AST based remapping（v8 报告精度=istanbul） | Vitest 3.2.0 起默认 | v8 provider 数字可直接作为 gate 依据，无需换 istanbul provider [CITED: vitest.dev/guide/coverage] |
| `coverage.ignoreEmptyLines` | 已移除——无运行时代码的行自动不计 | Vitest 4.0 | diff gate 的「可执行行」语义部分由 provider 免费实现 [CITED: vitest.dev/guide/migration] |
| Vitest 4 默认 exclude 较宽（dist/cypress/config 等） | 只排除 node_modules 与 .git | Vitest 4.0 | include 圈定后旧 exclude 多数失效；src 内需保留的仅 test/d.ts/白名单（Pitfall 6）[CITED: vitest.dev/guide/migration] |

**Deprecated/outdated:**
- `coverage.all` / `coverage.extensions` / `coverage.ignoreEmptyLines` / `coverage.experimentalAstAwareRemapping`：Vitest 4 已移除，配置中出现会报错或被忽略 [CITED: vitest.dev/guide/migration]。

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | CI Linux 上 vitest 全量口径测量值与本地 Windows 实测 3.847% 一致（±0.1pp 内） | Pitfall 2 / Validation | CI 实测 <3.8 则首次 gate 红；mitigation：先在 CI 实测再校准 `.coverage-fe-floors` 全局行（纯数据变更），D-07 数据文件设计使修正零脚本成本 |
| A2 | vitest `coverage.reportOnFailure` 默认 false（测试失败不生成报告） | Pitfall 9 | 仅影响失败时的 artifact 可得性（if: always() 步骤拿不到 json）；gate 阻断语义不受影响（Test 步骤本身已红）。低风险 |
| A3 | ubuntu-latest runner 满足 bash/awk/node24（bash 与 awk 为 runner 镜像标配） | Environment Availability | 后端 job 已在同一镜像上跑 bash+awk gate 十余个 phase，风险极低；node24 由 setup-node 显式安装 |

**除上述 3 条外，本研究全部关键主张均为 [VERIFIED]（本地实测 / 官方文档 / 仓库文件三方之一以上支撑）。**

## Open Questions

1. **D-14 全局阈值 3.8% 的 CI 校准点**
   - What we know: 本地实测 3.847%（vitest 摘要 3.84%），阈值 3.8 通过，余量 0.047pp；击穿需新增 ~268 未覆盖 stmts。
   - What's unclear: CI Linux 首次实测值（历史上后端 CI/本地有 ±0.5pp 级差异案例）。
   - Recommendation: plan 中把「首次 CI 全量运行读数」设为显式检查点；若 CI <3.8，以 CI 读数写 `.coverage-fe-floors` 全局行（D-14 语义「失败即阻断」不变，数值是 D-07 管辖的数据）。
2. **SC-1「584 文件」的验证口径（Pitfall 7）**
   - What we know: 白名单 exclude 后 per-file 报告为 571 文件；584 含 cad 13 文件。
   - What's unclear: SC-1 字面要求 584 全部出现与其前半句「白名单排除后」矛盾。
   - Recommendation: VALIDATION.md 拆两步断言——口径切换瞬间 584（未排白名单）、白名单落地后 571 + 3.85%。
3. **ROADMAP per-dir 清单与实测的小额差异（login 62 vs 95、零散 224 vs 189）**
   - What we know: 实测表可由 gate 脚本 `--init` 复算，数字自洽（pages 合计 13144 与 ROADMAP 精确一致）。
   - Recommendation: 基线文档快照以实测表为准；ROADMAP 不改（后续 phase 的 wave 验收以 gate 实测为准）。
4. **`(src root)` / `src/api` 共 21 stmts 的 floor 归属（Pitfall 8）**
   - What we know: D-05 粒度下它们是「无主面积」；REQUIREMENTS 要求白名单外无无主面积。
   - Recommendation: floor 表加 `(src root)` 与 `api` 两个显式条目（合计仅 21 stmts，可并入 Phase 83 INFRA-05 收尾覆盖）。
5. **ROADMAP 执行顺序笔误**：「Phase 82 → 81 → 82 → 83…」中的 81 属并行 workstream 保留号——不影响本 phase，提请后续修正。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | vitest 运行 / gate 脚本 node -e | ✓ | 24.19.0（本地）/ 24（CI setup-node 锁定） | — |
| npm | 依赖安装 | ✓ | 11.17.0 | — |
| vitest + @vitest/coverage-v8 | GOV-01 测量 | ✓ | 4.1.10（node_modules 实测） | — |
| GNU bash + awk | gate 脚本 | ✓ | GNU Awk 5.0.0（本地）；CI ubuntu-latest 标配（后端 gate 同栈已验证） | — |
| git（unified=0 diff / pathspec） | diff gate | ✓ | 仓库实测 pathspec 过滤有效 | — |
| GitHub Actions（upload/download-artifact@v4） | D-02 artifact 复用 | ✓ | 后端 coverage-diff job 同款已在产 | — |

**Missing dependencies with no fallback:** 无——全部依赖就位，phase 可直接执行。

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vitest 4.1.10（jsdom + @testing-library；现有 19 文件 159 测试） |
| Config file | `xingran-react-frontend/vitest.config.ts` |
| Quick run command | `cd xingran-react-frontend && npm run test`（纯测试，秒级~分钟级） |
| Full suite command | `cd xingran-react-frontend && npm run test:coverage`（全量口径，本地实测约 4-7 分钟含 transform） |

> 注：本 phase **不新增 vitest 测试文件**（SC 明确「不新增任何测试」）。gate 脚本按后端范式不配单测（CI 即验收场），但每个 gate 脚本必须支持本地对既有 coverage-final.json 干跑验证。

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GOV-01 | 全量口径生效：571 文件（白名单后）全部入报告、未测文件 0% | smoke（产物断言） | `cd xingran-react-frontend && npm run test:coverage && node -e "const d=require('./coverage/coverage-final.json');const k=Object.keys(d);console.log('files='+k.length, 'cad='+k.filter(p=>p.includes('cad-')).length)"` → 期望 files=571, cad=0 | ❌ 执行期即时命令（gate 脚本 --init 亦可复算） |
| GOV-02 | 基线文档落盘且数字可复算 | manual-only（文档）+ 可复算断言 | `node -e`（同上）比对文档记载 21574/830/3.85% | ❌ 文档任务 |
| GOV-03 | 全局阈值 gate 失败即阻断 | integration（gate 干跑） | `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors` → exit 0（3.85≥3.8）；临时把全局行改 5.0 干跑 → exit 1 | ❌ Wave 0（脚本本 phase 新建） |
| GOV-04 | diff gate ≥80% 按行计算并输出未覆盖清单 | integration（gate 干跑） | `bash .github/scripts/check-frontend-diff-coverage.sh <json> HEAD~1 80`（构造已知覆盖/未覆盖 diff 的合成基线验证两分支 exit code） | ❌ Wave 0（脚本本 phase 新建） |
| GOV-05 | per-dir floor 违例独立 exit code | integration（gate 干跑） | 同 GOV-03 脚本：临时把某目录 floor 改为 90.0 → exit 4；白名单漂移注入（临时移除 exclude 跑 coverage）→ exit 6 | ❌ Wave 0 |
| QUAL-02 | 白名单登记三项齐全 + 面积 ≤5% | manual-only（文档审查：理由/面积/复审条件）+ 自动核算 | `node -e` 核算 1028/22602=4.55%（已实测成立） | ❌ 文档任务 |

### Sampling Rate
- **Per task commit:** `cd xingran-react-frontend && npm run test:coverage`（不回归：19 文件 159 测试全绿 + json 正常产出）+ 对应 gate 脚本本地干跑 exit 0。
- **Per wave merge:** 全套 gate 干跑（全局/floor/diff 三脚本全 exit 0）+ gate 失败分支注入演练（每脚本至少一次非零 exit 验证）。
- **Phase gate:** 开一个 PR 触发真实 CI——frontend job 绿（含 gate 步骤）+ frontend-coverage-diff job 出现在 PR checks（PR-only 生效证明）+ push main 无 diff job（PR-only 证明）→ 之后 `/gsd:verify-work`。

### Wave 0 Gaps
- 无 vitest 测试缺口（本 phase 零新增测试，现有基础设施覆盖回归验证）。
- gate 脚本本地干跑清单属执行期任务（脚本本身是本 phase 产出物）。

## Security Domain

> config.json 未显式关闭 security_enforcement（absent = enabled）。本 phase 为 CI/工具链变更，不触碰业务鉴权/会话/加密面。

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | 无鉴权变更 |
| V3 Session Management | no | 无会话变更 |
| V4 Access Control | yes（CI 层） | ci.yml 保持 `permissions: contents: read`；不新增 secrets；artifact 只含 coverage 数据 |
| V5 Input Validation | yes（gate 脚本输入） | 脚本输入全部为工具产物/仓库文件（json、floors 文件、git ref）；沿用后端防御模式——参数缺失 exit 2、profile 缺失软跳过、无 PASS 行按失败处理（拒绝静默通过） |
| V6 Cryptography | no | 无加密变更 |

### Known Threat Patterns for CI gate scripts

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| gate 静默通过（解析失败/空输入被当成功） | Elevation of Privilege（绕过质量门） | 后端「no PASS line → exit 1」防御收尾模式照抄；`set -euo pipefail` |
| 第三方 coverage action 供应链注入 | Tampering | 零第三方 action：后端 74-10 已验证 marketplace 选项不可信，前端沿用 in-repo 自实现 [VERIFIED: check-diff-coverage.sh 头注释] |
| 阈值数据文件被悄悄调低（ratchet 倒退） | Repudiation | `.coverage-fe-floors` 走 PR review + 基线文档同 commit 追加纪律（D-04 后端对称）；CI diff 可见任何下调 |
| 白名单漂移（exclude 被删、面积超标无察觉） | Tampering | D-10 漂移检测独立 exit code；D-12 新增白名单须 milestone 级决策 + 面积重核算 |

## Sources

### Primary (HIGH confidence)
- 本地全量口径 coverage 运行（2026-08-23，CLI 参数覆盖、零仓库改动）——571 文件 / 21574 stmts / 830 covered / 3.85%、per-dir 全表、json 格式/路径形态、测试文件自动排除、cad 目录文件数（8+5）。所有关键数字的最终依据。
- vitest.dev/guide/migration（官方迁移指南）——coverage.all 移除、include 未配置时的行为、Simplified exclude、ignoreEmptyLines 移除、AST remapping。
- vitest.dev/guide/coverage（官方文档）——「By default Vitest will show only files that were imported during test run」、include/exclude 模式、v8 provider AST remapping 自 3.2.0。
- 仓库文件（逐行读取）：`.github/scripts/check-coverage.sh`（四段结构/exit 1-2-4-5/ratchet 教训注释）、`.github/scripts/check-diff-coverage.sh`（三段式/第三方案例否决记录）、`.github/workflows/ci.yml`（backend/frontend/coverage-diff 三 job 结构、working-directory、artifact 模式）、`.planning/coverage-baseline.md`（ratchet 表 schema/快照模式）、`xingran-react-frontend/vitest.config.ts`、`xingran-react-frontend/package.json`、`.coverage-threshold`（55.5）、`.gitignore`（coverage 目录已忽略）。
- git pathspec 实测（2026-08-23）——单星跨目录（310 命中）、`:(exclude)` 生效（297→288）。

### Secondary (MEDIUM confidence)
- 无（本研究未依赖未验证的次级来源）。

### Tertiary (LOW confidence)
- 无。

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 零新增依赖，全部为已安装/已验证组件
- Architecture: HIGH — 对称后端已在产的 v1.26 模式 + 本地实测数据支撑
- Pitfalls: HIGH — 10 条中 8 条有实测/官方文档/仓库文件直接证据，2 条标注 [ASSUMED]（A2/A3，低风险）

**Research date:** 2026-08-23
**Valid until:** 2026-09-22（vitest 4.x 稳定；若期间升级 vitest 大版本需复核 coverage 行为）
