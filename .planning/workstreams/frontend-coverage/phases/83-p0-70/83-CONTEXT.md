# Phase 83: P0 基建层全清 ≥70% - Context

**Gathered:** 2026-08-24
**Status:** Ready for planning

<domain>
## Phase Boundary

本 phase 交付三件事：

1. **CR-01 gate 修复前置**（82 遗留，verifier 判定建议前置）：修复 `check-frontend-diff-coverage.sh` pathspec 未镜像 vitest `coverage.exclude` 的 BLOCKER（白名单 CAD 文件 / `**/*.d.ts` / `src/test/` 变更的合法 PR 会误触 gate 红），连同 WR-01/02/03 一次清完。
2. **P0 基建层测试补齐**：lib（20 文件 1042 stmts）/ utils（23 文件 950 stmts）/ hooks（27 文件 1050 stmts）/ store（9 文件 589 stmts）/ services+router+constants+types（~626 stmts）共 82 文件 ~3,900 stmts，各目录 statements 覆盖率 ≥70%（INFRA-01~05）。
3. **公共测试 harness 定稿**（QUAL-03）：在 P0 尾声按实证重复需求沉淀 renderWithProviders / mockApi / message mock 三件套，Phase 84（P1 组件层）开工前备好。

**Out of scope**: P1 组件层（Phase 84）、P2 页面层（Phase 85-87）、白名单变更（D-12 锁死）、E2E/视觉回归（REQUIREMENTS Out of Scope）、业务逻辑修改（测试暴露的 bug 修复除外——ROADMAP 范围边界）。

</domain>

<decisions>
## Implementation Decisions

### Gate 修复（82 遗留前置）
- **D-01:** CR-01 修复作为 Phase 83 的**首个 plan**（wave 1 首位），走完整 plan 流程——与 harness 落地同 phase，时序上保证后续 harness PR 不踩 gate 红。
- **D-02:** 修复范围为**全部四项**：CR-01（diff pathspec 镜像 vitest coverage.exclude——白名单/.d.ts/src-test 不进 diff 分母）+ WR-01（json 缺失软跳过改 fail-closed `exit 2`，对齐后端孪生脚本 `check-diff-coverage.sh` 同位置行为）+ WR-02（白名单漂移检测改锚定匹配，`cad-editor-helpers.ts` 类合法新文件不再误判）+ WR-03（floors 数值结构校验，拒绝 `3..8`/`1.2.3` 类手误）。
- **D-03:** 修复验证 = 本地空树合成基线复现修复前后行为（82-03 已证可靠的技术）+ **试验 PR**（含 `src/test/` + `.d.ts` + 白名单文件变更）验证真实 CI 绿；试验 PR 验完关闭**不 merge**。顺带首次真实触发 GOV-04 的 join+阈值主路径（补 82-REVIEW IN-06 缺口——此前该路径只走过 docs-only 空 diff 分支）。

### 公共测试 harness（QUAL-03）
- **D-04:** harness **P0 尾声定稿**——P0 各 plan（纯逻辑测试为主）期间观察实际重复样板，harness 在 P0 尾声（store/组件接壤处）按实证需求定稿；不做前置全家桶，也不推迟到 84。
- **D-05:** `renderWithProviders` **按需注入**——默认 Router + AntD ConfigProvider；Zustand stores 走参数按需注入并自动 reset（对齐 Zustand 官方测试模式 resetBetweenTests），避免全量注入的测试间状态泄漏。
- **D-06:** `mockApi` 采用**端点工厂**形态——`createApiMock(endpoint, response)` 生成 post/get spy，按端点注册成功/失败/延迟响应；零新依赖，贴合现有 wrapped post/get 项目约定（CLAUDE.md 前端 API 规范）。不引入 MSW。

### Mock 与测试策略
- **D-07:** api.ts 加密客户端**双轨**——业务层（hooks/store/组件）测试用 `vi.mock('@/lib/api')` 整模块 mock；api.ts 自身在 INFRA-01 内用真实链路 + mock 加密层（固定密钥/向量）直测加密编排、401 刷新、重试分支。
- **D-08:** sm2/sm4/encoding 国密工具用**真实算法 + 确定性测试向量**直测——加解密往返、篡改密文报错、密钥长度边界，固定密文样本验证；零 mock。
- **D-09:** TokenManager 定时刷新循环用 `vi.useFakeTimers()`——`advanceTimersByTime` 直达过期点；并发刷新队列与 401 重试同样 fake timers + mock api；不使用真实短 TTL（避免 flaky）。

### Plan 结构与验收
- **D-10:** plan 按**依赖层切分**——plan0=CR-01 修复；随后 wave：utils+lib（底层先清，两 plan 并行）→ hooks+store → services/router/constants/types 收尾 + harness 定稿（P0 尾声，D-04）。依赖方向 utils ← lib ← hooks/store，底层未清不测上层。
- **D-11:** per-dir floor **逐 plan bump**——每个 plan 完成即 bump 对应目录 floor 至实测−0.5pp 并同 PR 追加基线文档（延续 82 的 D-06/D-07 纪律：ratchet 是纯数据变更 + 文档同 commit）。
- **D-12:** **plan 级验收**——每个 plan 的 verify 含 `npm run test:coverage` + gate 脚本目录断言 + 159 存量测试不回归（QUAL-01 基线）；phase 级 verify 只做汇总。

### Claude's Discretion
- 具体测试文件组织（同目录 `*.test.ts` 放置、describe/it 分组粒度）——按现有 19 个测试文件的既有模式。
- CR-01 pathspec 镜像的实现方式（硬编码镜像列表 vs 脚本内联注释指回 vitest.config.ts 真相源）——由 researcher/planner 定，保持单一真相源原则即可。
- 各 plan 内文件的测试优先级排序（先高扇出后长尾）。

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 需求与锁定决策
- `.planning/workstreams/frontend-coverage/REQUIREMENTS.md` — INFRA-01~05 + QUAL-03 需求定义（§v1 Requirements 与 Phase Mapping）
- `.planning/workstreams/frontend-coverage/phases/82-coverage-caliber-and-governance/82-CONTEXT.md` — Phase 82 锁定决策 D-01~D-16（statements 维度 D-13 / ratchet 纪律 D-06/D-07 / 白名单真相源 D-10 / 白名单锁死 D-12 / vitest thresholds 禁用 D-16）
- `.planning/workstreams/frontend-coverage/ROADMAP.md` — Phase 83 目标、三波 P0→P1→P2 结构、范围边界（仅补测试不修业务逻辑）

### CR-01 修复对象与发现详情
- `.github/scripts/check-frontend-diff-coverage.sh` — CR-01 修复对象（pathspec L135-138）与 WR-01（json 缺失软跳过分支）
- `.github/scripts/check-frontend-coverage.sh` — WR-02（漂移检测 grep）与 WR-03（floors 数值校验）修复对象
- `.planning/workstreams/frontend-coverage/phases/82-coverage-caliber-and-governance/82-REVIEW.md` — CR-01/WR-01~03 完整发现与本地复现记录
- `.planning/workstreams/frontend-coverage/phases/82-coverage-caliber-and-governance/82-VERIFICATION.md` — verifier 对 CR-01 的判定（WARNING 级、fail-closed 方向、83 前置建议）
- `.github/scripts/check-diff-coverage.sh` — 后端孪生脚本（WR-01 fail-closed `exit 2` 的对齐参照）

### Gate 与 ratchet 数据
- `.coverage-fe-floors` — per-dir floor 表（D-11 逐 plan bump 目标）
- `.planning/frontend-coverage-baseline.md` — ratchet 记录表（bump 同 PR 追加）与白名单登记段
- `xingran-react-frontend/vitest.config.ts` — coverage.include/exclude 真相源（CR-01 pathspec 镜像对象）

### 测试基建现状
- `xingran-react-frontend/src/lib/api.ts` — 加密客户端（D-07 双轨的直测对象）
- `xingran-react-frontend/src/test/setup.ts` — 现有 vitest setupFiles（harness 集成点）
- `xingran-react-frontend/src/utils/sm2.ts` / `sm4.ts` — 国密工具（D-08 向量直测对象）
- 现有 19 个测试文件 — 既有测试模式参照（jsdom + @testing-library）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- 现有 19 个测试文件（159 tests 全绿）——P0 测试的既有模式参照与"已知通过样本"。
- `src/test/setup.ts` 已配置 jsdom 环境 setupFiles——harness 三件套的天然挂载点。
- 82-03 的**空树合成基线**技术——CR-01 修复验证的本地复现手段（构造受控 diff 基线跑 gate 脚本）。
- 82-02 gate 脚本的 `--init` 模式——floor bump 后的数据再生成参照。

### Established Patterns
- **bash+awk 零依赖 gate**（82 的锁定范式）：CR-01/WR 修复保持该范式，不引入第三方工具。
- **ratchet 纪律**：floor bump 与基线文档追加同一 commit（后端 D-04 / 前端 82 D-06/D-07）。
- **依赖层推进**：utils（最底层）← lib ← hooks/store——P0 波次的并行边界即依赖边界。
- **Zustand store 测试**：按需注入 + reset（D-05），不用全量 Provider 包裹。

### Integration Points
- `.coverage-fe-floors`：每个 plan 完成时 bump 对应目录行（D-11）。
- `ci.yml` frontend job：gate 步骤读 floors 表——bump 后 CI 自动生效，无需改脚本。
- `vitest.config.ts` coverage.exclude：CR-01 pathspec 镜像的唯一真相源——两处列表必须同步演化。

</code_context>

<specifics>
## Specific Ideas

- CR-01 试验 PR 的变更构成：`src/test/` 新文件 + `.d.ts` 变更 + 白名单目录内文件变更——三类曾被误罚的路径各至少一个文件。
- 波次结构：CR-01 修复（1 plan）→ utils ∥ lib（2 plan 并行）→ hooks + store（同 wave）→ 收尾 + harness 定稿。
- harness 三件套最小集：`renderWithProviders`（Router + ConfigProvider + 按需 stores）/ `createApiMock`（端点工厂）/ message mock；其余工具按 P0 实证重复度增补。
- 国密测试向量：加密→解密往返、密文篡改必须报错、密钥长度边界、固定密文样本。

</specifics>

<deferred>
## Deferred Ideas

- MSW 网络层 mock —— D-06 未采纳（零新依赖优先）；若 P1/P2 出现网络层真实拦截需求再评估。
- CI 缓存/分片优化 —— 82 遗留 deferred，本 phase 不动（41s Test (coverage) 余量充足）。
- harness 全家桶（fireEvent helpers / 表单填充 / IntersectionObserver 等 jsdom 补丁）—— D-04 按 P1/P2 实际需求渐进增补，不前置。
- E2E 测试层、视觉回归 —— REQUIREMENTS v2 候选，本里程碑外。

</deferred>

---

*Phase: 83-P0 基建层全清 ≥70%*
*Context gathered: 2026-08-24 via /gsd:discuss-phase*
