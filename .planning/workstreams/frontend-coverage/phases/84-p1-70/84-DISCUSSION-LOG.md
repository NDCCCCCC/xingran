# Phase 84: P1 组件层 ≥70% - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-27
**Phase:** 84-P1 组件层 ≥70%
**Areas discussed:** Harness 落地策略 / Floor 粒度与跨 subdir 桶策略 / Plan 切分策略 / 组件测试深度基线

---

## Harness 落地策略

| Option | Description | Selected |
|--------|-------------|----------|
| 补做完整 harness(83 D-04 原计划) | plan 0 落地 `renderWithProviders` + `createApiMock` 完整版;后续所有 P1 测试复用 | ✓ |
| 仅补 renderWithProviders,跳过 createApiMock | plan 0 只做最小 render helper;api 走现有 vi.mock 模式 | |
| 不做 harness,继续 per-test Wrapper | 沿用 BulkWriteDrawer/HealthCard 现有 per-test inline Wrapper 模式 | |

**User's choice:** 补做完整 harness(83 D-04 原计划)
**Notes:** 用户在 AskUserQuestion 中明确选择"补做完整 harness(83 D-04 原计划)"——与 Phase 83 锁定决策对齐;不推迟。后续剩余 3 题用户回复"使用最佳实践方案",由 Claude 按现有锁定决策与代码库现状选最佳实践路径。

---

## Floor 粒度与跨 subdir 桶策略

| Option | Description | Selected |
|--------|-------------|----------|
| 扩展 gate 支持 subdir 行 | 扩展 `.coverage-fe-floors` 引入 9 个 components subdir 行 + `design-system`;gate 同步扩展路径聚合 | ✓ |
| 聚合 components 行(白名单外)≥70% | 维持 gate 现状,`components` 聚合行目标 ≥70%;subdir 视为达成指标不进入 gate | |
| 聚合行 + 五个 COMP 子目标(plan 级自验) | gate 不变,plan 阶段打印/警告 subdir 实测但不 fail | |

**User's choice:** Claude best-practice pick — 扩展 gate 支持 subdir 行
**Notes:** 与 ROADMAP SC 字面"各 subdir ≥70%"对齐;与 82 D-05 已有的 `pages/<subdir>` 二级粒度模式对称;gate 扩展是路径聚合 3 处修改(一次性),后续 bump 即纯数据变更。保留 `components` 聚合行向后兼容 + bump 至 84 终点值。

---

## Plan 切分策略

| Option | Description | Selected |
|--------|-------------|----------|
| 五 plan 串行 + plan 0 harness | 6 个 plan,harness + 五个 COMP 串行 | |
| 三 wave 并行 + plan 0 harness | 4 个 plan,harness + wave 1 shared‖dashboard + wave 2 layout‖CronSelector+captcha+operations + wave 3 network+reconciliation+零散+design-system | ✓ |
| 五 plan 串行,按 stmts 升序 | 5 个 plan,无 harness;按 stmts 升序先清小目录 | |

**User's choice:** Claude best-practice pick — 三 wave 并行 + plan 0 harness
**Notes:** 84 五个组件组都是叶子(无相互依赖),可按 wave 并行;与 83 D-10 "wave 内可并行,wave 间串行"风格一致;wall-clock 减半。

---

## 组件测试深度基线

| Option | Description | Selected |
|--------|-------------|----------|
| 交互 + 子 hook mock(沿用 BulkWriteDrawer 模式) | 模式 A:每个组件测试至少一次 user event + 一次 props 渲染断言;子 hook/store/api mock 走 vi.mock | ✓ |
| 纯渲染 + props 快照(最快) | 模式 B:render + 关键 DOM 断言,无 user event | |
| 混合:核心交互走模式 A,纯展示走模式 B | 按组件复杂度分支,plan 内分类决策 | |

**User's choice:** Claude best-practice pick — 模式 A 锁定(允许纯展示组件简化)
**Notes:** 与 BulkWriteDrawer / HealthCard 既有风格完全对齐(已实测覆盖率含金量的样本);纯展示组件允许单渲染断言(D-12 例外条款);含 canvas/drag 的 captcha 组件按需 mock。

---

## Claude's Discretion

- D-13 polyfill 清单具体边界(ResizeObserver / getComputedStyle 子集)——执行阶段按实际渲染失败实证补齐,不前置
- D-03 mockApiBatch 与单端点 mock 的使用偏好——以简洁优先
- `renderWithProviders` 的 QueryClientProvider 默认注入——按需参数,不默认
- 同一组件内多文件的拆分粒度(单文件 vs `__tests__/` 聚合)——按现有 `__tests__/` 模式参考
- `components/table/` `three/` `DeptTree/` `IconSelect/` `NoticeDetail/` `NotificationBell/` `TargetSelector/` `markdown/` `modal/` `charts/` `asset/` 零散组件清单——wave 3 plan 内按 stmts 量级确认

---

## Deferred Ideas

- antd `Table` / `Form` 全局测试模式 —— 84 不在范围,deferred 到 P2
- Storybook / 视觉回归 —— Out of Scope
- 组件 E2E(Playwright) —— Out of Scope
- MSW 网络层 mock —— 零新依赖纪律,若 P2 真实拦截需求再评估
- `renderWithProviders` store 嵌套 / 自定义 reset 扩展——按实证需求定形
- CI timeout / 分片优化 —— 沿用 82 D-04,先观察
