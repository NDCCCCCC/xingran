# Phase 84: P1 组件层 ≥70% - Context

**Gathered:** 2026-08-27
**Status:** Ready for planning

<domain>
## Phase Boundary

本 phase 交付一件事:

**components/* 五个组件组(白名单外约 4,150 stmts + 顶层 design-system 194 stmts)语句覆盖率全部 ≥70%**——共享组件、仪表盘 Widget 体系、布局骨架与 network/reconciliation 低洼目录全部拉平;并把 83 D-04 承诺但未落地的测试 harness(`renderWithProviders` + `createApiMock`)在 P1 开工前补齐。

**Out of scope**: P2 页面层(Phase 85-87)、白名单变更(D-12 锁死)、业务逻辑修改(测试暴露的 bug 修复除外——ROADMAP 范围边界)、E2E/视觉回归(REQUIREMENTS Out of Scope)。

</domain>

<decisions>
## Implementation Decisions

### Harness 落地(83 D-04 补课)
- **D-01:** Phase 84 `plan 0` 落地 `renderWithProviders` + `createApiMock` 完整版,与 83 D-04/D-05/D-06 锁定决策对齐;不推迟到 84 中段或 P2。
- **D-02:** `renderWithProviders` 默认注入 `<MemoryRouter>` + `<App>`(antd App 提供 message/modal/notification context)。Zustand stores 按需参数注入并自动 reset(对齐 83 D-05,Zustand 官方 resetBetweenTests 模式)。
- **D-03:** `createApiMock(endpoint)` 端点工厂形态——生成 vi.fn() spy,支持 `.mockResolvedValue()` / `.mockRejectedValue()` / `.mockImplementationOnce()` 链式;**不引入 MSW**(零新依赖纪律,与 83 D-06 对齐)。提供可选 `mockApiBatch(handlers: Array<{endpoint, response}>)` 一次注册多端点。

### Floor 粒度与 gate 扩展
- **D-04:** 扩展 `.coverage-fe-floors` 引入 components subdir 行(`components/shared` / `components/dashboard` / `components/layout` / `components/CronSelector` / `components/captcha` / `components/operations` / `components/network` / `components/reconciliation` / `design-system`)——与 ROADMAP SC 字面"subdir ≥70%"对齐,与 82 D-05 的 `pages/<subdir>` 二级粒度模式对称。
- **D-05:** `check-frontend-coverage.sh` 的 awk 路径聚合逻辑需在 3 处(`L219 / L316 / L381`)同步扩展 `components/<subdir>` 分支,与现有 `pages/<subdir>` 完全镜像(改后保持 82 D-07 "ratchet bump 是纯数据变更"——84 subdir 行一旦落地后续 bump 即纯数据)。
- **D-06:** `components` 聚合行保留并 bump 至 84 终点值(白名单外实测 ≥70%)——既满足 ROADMAP SC "全清" 又保留 82 D-05 一级目录粒度向后兼容。`design-system` 不挂在 components 下,而是与 hooks/store/services 同级顶层行(与现状 15.0 行对齐)。

### COMP-04 跨 subdir 桶策略
- **D-07:** CronSelector(316) + captcha(154) + operations(149) 三个 subdir 各自独立 floor——ratchet 互不掩盖,任一 subdir 倒退会被 gate 抓到。

### Plan 切分(三 wave 并行 + plan 0)
- **D-08:** 4 个 plan,plan 0 = harness;wave 1 = `shared`(892) || `dashboard`(1068) 并行;wave 2 = `layout`(507) || `CronSelector+captcha+operations`(619) 并行;wave 3 = `network + reconciliation + 零散 + design-system`(966 stmts)独立收口。
- **D-09:** wave 内并行 PR 互不阻塞,各自 bump 各自 subdir floor(纯数据变更);并行度选择依据 = 同 wave 内组件相互无依赖(都是叶子),且 stmts 量级匹配避免大目录独占 plan。
- **D-10:** 与 83 D-10 风格一致——wave 内可并行,wave 间串行(每 wave bump 后实测覆盖率单调上升再进下一 wave)。

### 组件测试深度基线
- **D-11:** **模式 A 锁定**——每个组件测试至少包含一次 user event(`fireEvent` 或 `@testing-library/user-event`)+ 一次 props 渲染断言;子 hook/store/api mock 走 `vi.mock()` 路径;**对齐 BulkWriteDrawer / HealthCard 既有风格**(已实测覆盖率的两个组件样本)。
- **D-12:** 纯展示组件(如 `ModernTag`、`EmptyStateWithAction`)允许**单一渲染断言**(无 user event),但需有 props 变异(至少 2 个 props 组合)的快照——保证覆盖率含金量而非纯 0%→100% 暴力行覆盖。
- **D-13:** antd `Drawer`/`Modal`/`Select` 渲染需要的 polyfill(`ResizeObserver`、`getComputedStyle` 子集)在 `src/test/setup.ts` 集中沉淀,**不**在每个测试文件重写——延续 BulkWriteDrawer 经验,把 jsdom 补丁提到 setup 层。

### Floor bump 节奏
- **D-14:** 每个 plan 完成即 bump 对应 subdir floor 至实测−0.5pp(沿用 82 D-06 / 83 D-11 噪声余量纪律),同 PR 追加 `.planning/frontend-coverage-baseline.md` ratchet 行。
- **D-15:** `components` 聚合行 floor 在 wave 3 完成后 bump 至白名单外实测≥70% 值(终点目标 = 70.0 - 0.5 = 69.5 一位小数);`design-system` 同步 bump 至 70.0 - 0.5 = 69.5(若白名单外实测 ≥70%)。

### Plan 级验收
- **D-16:** 沿用 83 D-12——每个 plan 的 verify 含 `npm run test:coverage` + `bash .github/scripts/check-frontend-coverage.sh` + QUAL-01 159 存量测试不回归断言;phase 级 verify 仅汇总。

### Claude's Discretion
- 同一组件内多文件的拆分粒度(单文件 `.test.tsx` 还是按组件家族聚合 `__tests__/ComponentGroup.test.tsx`)——按现有 `__tests__/` 目录模式参考(`network/port-write/__tests__/BulkWriteDrawer.test.tsx`、`reconciliation/__tests__/HealthCard.test.tsx`)。
- D-13 polyfill 清单的具体边界(哪个 antd 组件需要哪个 polyfill)——执行阶段按实际渲染失败实证补齐,不前置。
- D-03 `mockApiBatch` 与单端点 mock 的使用偏好——以简洁优先,单端点不够时再批量。
- `renderWithProviders` 是否默认注入 `QueryClientProvider`(若组件用 `@tanstack/react-query`,参考 widgets 体系)——按需参数注入,不默认。

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 需求与锁定决策
- `.planning/workstreams/frontend-coverage/REQUIREMENTS.md` — COMP-01~05 需求定义(§v1 Requirements 与 Phase Mapping)
- `.planning/workstreams/frontend-coverage/ROADMAP.md` — Phase 84 目标、五个组件组边界、`(src root)`/`api`/`design-system` 无主面积处理
- `.planning/workstreams/frontend-coverage/phases/82-coverage-caliber-and-governance/82-CONTEXT.md` — Phase 82 锁定决策 D-05(目录粒度)/D-06(ratchet 余量)/D-07(floors 数据真相源)/D-13(statements 维度)
- `.planning/workstreams/frontend-coverage/phases/83-p0-70/83-CONTEXT.md` — Phase 83 锁定决策 D-04(harness P0 尾声)/D-05(renderWithProviders 按需注入)/D-06(createApiMock 端点工厂)/D-07(api.ts 双轨)/D-10(依赖层切分)/D-11(per-dir floor 逐 plan bump)/D-12(plan 级验收)
- `.planning/PROJECT.md` — v1.28 milestone 锁定决策 D-01~D-04

### Gate 与 ratchet 数据
- `.github/scripts/check-frontend-coverage.sh` — gate 主脚本(D-05 扩展对象:L219 / L316 / L381 三处 awk 路径聚合)
- `.github/scripts/check-frontend-diff-coverage.sh` — PR diff ≥80% gate(83 D-01 CR-01 修复已落地)
- `.coverage-fe-floors` — per-dir floor 表(D-04 增量行 + D-15 聚合行 bump)
- `.planning/frontend-coverage-baseline.md` — ratchet 记录表(D-14 同 PR 追加)
- `xingran-react-frontend/vitest.config.ts` — coverage.include/exclude 真相源(白名单锁死 D-12)

### 测试基建现状
- `xingran-react-frontend/src/test/setup.ts` — 现有 vitest setupFiles(只含 matchMedia polyfill;harness 三件套的天然挂载点 + D-13 polyfill 沉淀点)
- `xingran-react-frontend/src/lib/api.ts` — 加密客户端(83 D-07 双轨直测对象)
- `xingran-react-frontend/vitest.config.ts` — coverage 全量口径配置 + jsdom 环境 + 15s timeout + v8/json/text/html reporter

### 既有组件测试样本(D-11 模式 A 参照)
- `xingran-react-frontend/src/components/network/port-write/__tests__/BulkWriteDrawer.test.tsx` — 交互 + 子 hook mock + ResizeObserver polyfill
- `xingran-react-frontend/src/components/reconciliation/__tests__/HealthCard.test.tsx` — 交互 + 子 hook mock + ECharts mock + 11 个用例覆盖空态/加载/错误/正常路径

### P1 五个组件组目录
- `xingran-react-frontend/src/components/shared/` — 21 文件 892 stmts(COMP-01)
- `xingran-react-frontend/src/components/dashboard/` — 29 文件 1068 stmts(COMP-02,含 `widgets/` `templates/` `utils/` `settings/` `layout/` 子目录)
- `xingran-react-frontend/src/components/layout/` — 16 文件 507 stmts(COMP-03,HybridLayout / Sidebar / Header)
- `xingran-react-frontend/src/components/CronSelector/` `captcha/` `operations/` — 619 stmts 合计(COMP-04)
- `xingran-react-frontend/src/components/network/` `reconciliation/` `零散` + `src/design-system/` — 966 stmts 合计(COMP-05)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- 现有 `BulkWriteDrawer.test.tsx` / `HealthCard.test.tsx` —— D-11 模式 A 的风格样本;harness 三件套设计可参考其 Wrapper 模式并抽象提升。
- 现有 19 个测试文件(159 tests 全绿)——D-11 模式 A 在 hooks/lib/reconciliation 三个目录已实证可走通,84 阶段无新模式探索成本。
- `src/test/setup.ts` —— harness 集成点 + D-13 polyfill 沉淀点(matchMedia 已配,ResizeObserver/getComputedStyle 子集按需扩展)。
- 82-03 的**空树合成基线**技术——CR-01 类回归验证手段(若 84 plan 触发 gate 红,可用此本地复现)。
- 82-02 gate 脚本的 `--init` 模式——84 阶段 subdir 行首次生成时的数据再生成参照。

### Established Patterns
- **bash+awk 零依赖 gate**(82 锁定范式):D-05 的 3 处扩展保持 awk 内联,不引入第三方工具。
- **ratchet 纪律**(82 D-06 / 83 D-11):floor bump 与基线文档追加同一 commit;floors 只升不降。
- **vitest 全量口径**(82 D-01):`coverage.include` 圈定 `src/**/*.{ts,tsx}`,未测试文件以 0% 计入——84 阶段新增测试覆盖即可在报告中可见。
- **现有 per-test inline Wrapper 模式**(BulkWriteDrawer / HealthCard)——84 plan 0 harness 落地后此模式逐步替换,新测试优先用 `renderWithProviders`。
- **依赖方向**:84 五个组件组都是叶子(无相互依赖),与 83 D-10 的"底层先清"不同——84 阶段无依赖约束,可按 stmts 量级 / 复杂度切 wave 并行。

### Integration Points
- `.coverage-fe-floors`:每个 plan 完成 bump 对应 subdir 行(D-14)+ wave 3 末尾 bump 聚合 `components` 与 `design-system` 两行(D-15)。
- `check-frontend-coverage.sh`:D-05 扩展 3 处 awk 路径聚合(plan 0 内与 harness 同步完成,或独立 plan 0.5 提交)。
- `vitest.config.ts` `coverage.include` 全 src 已就绪——新增测试文件无需改配置。
- `src/test/setup.ts`:D-13 polyfill 集中沉淀,后续 P2 复用。
- `src/test/renderWithProviders.tsx` + `src/test/createApiMock.ts`(plan 0 新建,84 阶段所有新测试 import)。

</code_context>

<specifics>
## Specific Ideas

- plan 0 harness 文件落地形态:
  - `src/test/renderWithProviders.tsx` — 默认 `<MemoryRouter><App>{ui}</App></MemoryRouter>` + 可选 stores reset。
  - `src/test/createApiMock.ts` — 端点工厂 + 批量注册 helpers。
  - `src/test/setup.ts` 增补 antd polyfill(`ResizeObserver` / `getComputedStyle` 子集)。
- wave 内并行的 PR 形态:每个 plan 是独立 PR 推到自己的分支;PR 内 bump 对应 subdir floor;CI gate 验证该 subdir ≥ bump 后 floor 即绿。
- `components/dashboard/` 是 84 阶段最大单目录(29 文件 1068 stmts),含 `widgets/` `templates/` `utils/` `settings/` `layout/` 五个子目录——wave 1b plan 可按子目录分批写测试,避免单 PR 文件过多。
- `components/layout/` 含 `HybridLayout` / `ClassicLayout` / `InnovativeLayout` / `Sidebar` / `Header` —— 是 routing 与菜单核心,模式 A 测试覆盖路由跳转 / 菜单折叠 / 主题切换是核心交互场景。
- `components/CronSelector/` 用 `@breejs/later` + `cron-parser` + `cron-validate` —— 可走"真实 cron 字符串解析 + 边界字符串"的纯逻辑测试为主(参考 utils 模式),非纯 antd 组件测试。
- `components/captcha/` 含 `SliderCaptcha` + `TextCaptcha` + `CaptchaModal` —— 含 canvas / drag 交互,可走"mock canvas API + fireEvent drag"模式。
- `components/network/` 现 50.6% / `components/reconciliation/` 现 18.1% —— wave 3 主要拉平这两个已有部分覆盖的目录,`HealthCard` 与 `BulkWriteDrawer` 风格可直接复用。

</specifics>

<deferred>
## Deferred Ideas

- antd `Table` / `Form` 全局测试模式 —— 84 不在 components/table 范围(ROADMAP 把 table 归 P2 页面层)。deferred 到 P2 plan 时再评估。
- Storybook / 视觉回归 —— REQUIREMENTS Out of Scope;deferred 到 v2 候选。
- 组件 E2E(Playwright) —— REQUIREMENTS Out of Scope;deferred 到 v2 候选。
- MSW 网络层 mock —— D-03 仍未采纳(零新依赖优先);若 P2 出现真实拦截需求再评估。
- `renderWithProviders` 的 store 注入 API 扩展(嵌套 store 组合 / 自定义 reset 时机)——84 阶段按实证需求定形,不前置设计。
- `components/table/` / `components/three/` / `components/DeptTree/` / `components/IconSelect/` / `components/NoticeDetail/` / `components/NotificationBell/` / `components/TargetSelector/` / `components/markdown/` / `components/modal/` / `components/charts/` / `components/asset/` —— 这些"零散组件"按 stmts 量级归入 wave 3(若超过 304 stmts 边界),具体清单在 wave 3 plan 内确认。
- 84 阶段不动 CI timeout / 分片优化 —— 沿用 82 D-04,先观察全量口径 transform 实际增量。

</deferred>

---

*Phase: 84-P1 组件层 ≥70%*
*Context gathered: 2026-08-27 via /gsd:discuss-phase*
</content>
</invoke>