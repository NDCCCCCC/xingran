---
phase: 71
phase_name: 治理基线 + CI gate
slug: governance-baseline-and-ci-gate
gathered: 2026-08-20
status: Ready for planning
---

# Phase 71: 治理基线 + CI gate - Context

**Gathered:** 2026-08-20
**Status:** Ready for planning

<domain>

## Phase Boundary

本阶段只交付**后端 CI 守护基建**:让 `.github/workflows/ci.yml` 跑出 coverage 报告、落地加权阈值 gate、记录基线数据,使后续 Phase 72/73/74 补的测试在 CI 上有"防倒退"机制。

**本阶段不补任何业务测试代码**(Phase 72-74 的事),也不改业务逻辑。Pkg/cache flaky test 视为已修(`5ead742`)直接验收。

**Phase 71 范围(GOV-01 + GOV-02 + GOV-04)**:
- GOV-01: 后端 CI job 加 `-coverprofile` + `-covermode=atomic` + 产 profile artifact
- GOV-02: 加权平均阈值 gate(初始 12.8% 基线)<阈值 PR 即 fail,ratchet 机制防倒退
- GOV-04: `.planning/coverage-baseline.md` 记录起点 + 各 phase 后实际数字
- 验收: `go test ./pkg/cache/...` 全过 + `go test ./internal/... ./pkg/... ./cmd/...` 全包 exit 0

**不在本阶段(GOV-03 属 Phase 74)**:PR 增量 diff coverage ≥80% 门槛。

</domain>

<decisions>

## Implementation Decisions

### CI 阈值 gate 工具选型

- **D-01 (Area 1)**: **bash + awk 自检**(零依赖,逻辑全部在 ci.yml 同一段)
  - 阈值写入独立文件 `.coverage-threshold`(纯数字如 `12.8`),CI 读这个文件——ratchet 时只改文件不改 ci.yml,便于审计。
  - awk 脚本放 `.github/scripts/check-coverage.sh`,复用 quick-260820-bcs 加权聚合公式(`internal/services/*` 业务包加权,排除 scripts/migrations 等运维工具)。
  - **Phase 74 启用 GOV-03 diff coverage 时重新评估** `vladopajic/go-test-coverage`(原生支持 base branch diff + PR comment),届时成本/收益再权衡。本阶段不预先引入。
  - 推荐理由:加权口径与 quick-260820-bcs 一致;零依赖;per-package 倒退日志可直接输出到 CI 失败信息便于排错。

### Profile artifact 上传策略

- **D-02 (Area 2)**: **每次 PR 都上传,保留 30 天**
  - 使用 `actions/upload-artifact@v4` + `retention-days: 30`。
  - 同时上传 `coverage.out`(原始 profile)和 `coverage.html`(`go tool cover -html=coverage.out -o coverage.html`)便于人工浏览器查看。
  - gate 失败时不另行下载——30 天内任何 PR 都可拉 artifact 本地调试,无需 fail-only 上传。
  - GitHub 1G artifact 限额下,74 包 coverage.out 约 3MB + html 约 5MB,30 天保留约 8MB/run,远低于限额。

### 基线数据存放位置

- **D-03 (Area 3)**: **新建独立文件 `.planning/coverage-baseline.md`**
  - 与 `quick/260820-backend-test-coverage-scan/SUMMARY.md`(原始只读扫描报告)分离——后者保持 scan 不变性,前者记录 phase 推进后的"状态快照"。
  - 文件结构:起点(2026-08-20,12.8%)→ Phase 71 后 → Phase 72 后 → ... → 最终(≥70%)的时序快照表;每行包含 date / weighted_avg / per-package-detail / commit。
  - quick-260820-bcs SUMMARY.md 不回填,只作为 v1.26 启动前的"原状"证据。

### Ratchet 阈值递进机制

- **D-04 (Area 4)**: **手动 bump(每次 phase execute plan 原子提交)**
  - 每个 phase(72/73/74)的 execute plan 完成后,由该 phase 负责原子提交"更新 .coverage-threshold + 在 coverage-baseline.md 追加新行"两件事。
  - commit message 明示 `coverage ratchet 12.8% → 30.5%`,可审计、可回滚。
  - 阈值与 baseline file 不做双向同步;出现"谁负责更新"的边界模糊时不利于防倒退。
  - 优势:每次 bump 是一个有意图、可 review 的 PR diff;劣势:需 phase plan 作者记得提交(但 ratchet 的本质就是 phase 收口的一部分)。

### Claude's Discretion

下列子决策未深入讨论,按默认规则走(若 plan 阶段发现需调整,可修改):

- **Gate fail 时的 override 机制**:严格无 override——不加 `[skip-coverage]` label 例外,gate fail = PR 必须修。如有真正例外需求(如紧急热修复),走 PR 豁免流程而非 CI 短路。
- **加权口径**:与 quick-260820-bcs 一致——74 个业务包加权平均,排除 `scripts/`、`migrations/`、`cmd/main`、`internal/docs`、密钥生成工具、诊断脚本。涵盖 43652 stmts 口径。
- **Per-package 倒退 PR comment**:不强制——CI 日志输出 per-package 数字即可,Phase 71 不实现 PR comment bot(FUT-04 范畴)。
- **覆盖率扫描时机**:`go test` 单次而非 race-detector 模式(避免时间翻倍);`count=1` 防缓存。
- **Coverage scope 列表存放**:`check-coverage.sh` 内置 awk 过滤规则(基于 SUMMARY.md "排除范围"段),不引入独立配置文件。

</decisions>

<canonical_refs>

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 规划输入

- `.planning/quick/260820-backend-test-coverage-scan/SUMMARY.md` — v1.26 milestone 启动前的覆盖率扫描原状(74 业务包 / 43652 stmts / 12.8% / 33 个 0% 包 / 已知测试失败清单 / CI 现状)
- `.planning/quick/260820-backend-test-coverage-scan/per-package-coverage.txt` — per-package stmts/covered/百分比明细(76 行聚合结果)
- `.planning/quick/260820-backend-test-coverage-scan/coverage.out` — 原始 Go atomic coverage profile(2.95MB,Go tool 解析用)
- `.planning/quick/260820-backend-test-coverage-scan/cover-func.txt` — 函数级明细(5152 行,Phase 71 参考格式)

### Milestone 治理文档

- `.planning/ROADMAP.md` — Phase 71 定义段(目标 / 5 SC / Key Decisions / Notes)
- `.planning/REQUIREMENTS.md` — GOV-01 / GOV-02 / GOV-04 三项需求 + Out of Scope + Traceability
- `.planning/PROJECT.md` §Current Milestone:v1.26 + D-01..D-06 锁定决策(加权 ≥70% / glebarez 沿用 / Phase 编号 / Research skipped)
- `.planning/STATE.md` §Quick Tasks Completed — `260820-far` pkg/cache flaky test 修复(`5ead742` 待确认 commit),Phase 71 直接验收

### CI 现状

- `.github/workflows/ci.yml` — 当前 backend job 定义(`go test -timeout 15m -count=1` 无 coverage),Phase 71 增改此文件
- `.github/workflows/deploy.yml` — gate 上游,Phase 71 不动(由 ci.yml 串接)

### 测试范本(可参考)

- `internal/utils/operlog/regression_test.go` — operlog 82.2% 范本,AST 锁公共 API(regression_test 设计模式参考)
- `internal/services/portwrite/*_test.go` — portwrite 85.3% 范本,glebarez sqlite in-memory + table-driven tests
- `internal/middleware/apikey_test.go` — middleware 86.2% 范本,全链路 apikey 测试

### 测试基建(D-04 锁定)

- `go.mod` — glebarez sqlite 已在依赖中(quick-260817-hfl 落地),无需新增
- `internal/core/db/` — sqlite 驱动 + 测试 DB 初始化模式

</canonical_refs>

<code_context>

## Existing Code Insights

### Reusable Assets

- **quick-260820-bcs awk 聚合公式** — `SUMMARY.md` "扫描方法"段的 awk 脚本是基线加权算法,Phase 71 `check-coverage.sh` 复用同一公式,确保 CI gate 与原状扫描口径一致。
- **golangci-lint-action@v9** — ci.yml 已有同类第三方 Action 引入先例,但 Phase 71 gate 不引入第三方——逻辑全部本地化。

### Established Patterns

- **测试基建锁定 glebarez sqlite in-memory** — D-04 锁定,不引新 mock framework。
- **operlog regression_test.go AST 锁值** — 可参考其 regression_test 设计模式:本阶段虽无 AST 锁值需求,但 `check-coverage.sh` 阈值文件可类比——单一真相源 + 测试/CI 双向引用。
- **CI Action 引入先例** — `golangci/golangci-lint-action@v9` 已落地,Action 引入非禁忌;Phase 71 不引入但保留 Phase 74 评估空间。

### Integration Points

- **`.github/workflows/ci.yml` backend job** — `Test` step 改为 `go test -coverprofile=coverage.out -covermode=atomic -count=1 ...`,后接 `check-coverage.sh` gate step;失败仍保留 `coverage.out` 给后续 step 上传。
- **`.github/workflows/ci.yml` 后端 job 末尾** — 新增 `Upload coverage artifact` step,`if: always()`,retention 30 天。
- **新增文件 `.coverage-threshold`** — 仓库根目录,纯数字 `12.8`,CI awk 脚本读此文件作阈值。
- **新增文件 `.github/scripts/check-coverage.sh`** — bash + awk 实现,`exit 1` if weighted_avg < threshold。
- **新增文件 `.planning/coverage-baseline.md`** — 状态快照表(空文件骨架,Phase 71 完成时回填 Phase 71 后数字)。

</code_context>

<specifics>

## Specific Ideas

无具体设计参考——这是工程化基建 phase,所有路径都由标准 Go 工具 + bash/awk 实现,无视觉/UX 决策。

Phase 71 唯一具体参考:`quick-260820-backend-test-coverage-scan/SUMMARY.md` 的扫描方法段(awk 聚合公式)与排除范围(74 业务包 vs scripts/migrations/cmd main/docs 工具)。

</specifics>

<deferred>

## Deferred Ideas

下列想法讨论中被归类为本阶段范围外,**记在这里不丢失**:

- **GOV-03 PR 增量 diff coverage ≥80%** — 属 Phase 74 范围,本阶段不做。Phase 74 时重新评估 `vladopajic/go-test-coverage` 是否值得引入(diff coverage 是其强项)。
- **FUT-02 分支覆盖率** — Go 原生 coverprofile 仅语句级;引入第三方工具或 Go 官方分支支持后再议。
- **FUT-03 mutation testing** — 覆盖率达标后的下一步,工具选型单独评估。
- **FUT-04 覆盖率 PR 评论机器人** — 每条 PR 自动贴 per-file diff coverage 表,GOV-03 达标后的体验增强,Phase 74+ 候选。
- **Gate fail override 机制** — 严格无 override 是默认决策;若未来出现紧急热修复 PR,走 PR 豁免流程而非 CI 短路,不在本阶段内建 escape hatch。
- **`vladopajic/go-test-coverage` 引入** — Phase 71 不引入;Phase 74 启用 diff coverage 时重新评估(Action 内置 base branch diff + PR comment 是 Phase 74 强需求)。

</deferred>

---

*Phase: 71-治理基线 + CI gate*
*Context gathered: 2026-08-20*
</content>
</invoke>