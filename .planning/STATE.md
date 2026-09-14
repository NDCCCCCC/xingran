---
gsd_state_version: 1.0
milestone: v1.33
milestone_name: 前端性能治理 (Frontend Performance Remediation)
status: Phase 115 plans all executed — SELECTOR-01~05 + NotificationBell 落位，criterion 5 收口（全库剩余 12 处归属 Phase 116/120）
stopped_at: Completed 115-03-PLAN.md
last_updated: "2026-09-14T03:57:59.751Z"
last_activity: 2026-09-14
progress:
  total_phases: 7
  completed_phases: 2
  total_plans: 6
  completed_plans: 6
  percent: 29
---

# Project State (v1.33 — Frontend Performance Remediation)

## Project Reference

See: `.planning/PROJECT.md` — v1.33 Current Milestone 段

**Core value:** 端到端运维可观测与可审计
**Current focus:** Phase 115 — selector-completion（Zustand selector 收尾）

## Current Position

Phase: 116
Plan: Not started
Status: Phase 115 plans all executed — SELECTOR-01~05 + NotificationBell 落位，criterion 5 收口（全库剩余 12 处归属 Phase 116/120）
Last activity: 2026-09-14

Progress: [██████████] 100% (6/6 milestone plans)

## v1.33 范围摘要

**Goal**: 修复 2026-09-11 前端全量性能审计（`.planning/reviews/20260911-frontend-perf-audit.md`）全部 33 项 findings（2 HIGH + 8 MEDIUM + 3 次级 + 12 LOW）+ 8 项死代码/依赖清理。用户确认全量范围不分批 defer。

**修复域**: H-2 地图聚类 O(n²) / H-1 dashboard N² 级联 / selector 收尾 26 处 / BUNDLE（ExcelImportLazy 推广 + iconUtils 假动态导入）/ RENDER（columns 工厂 + Table virtual）/ DATA（VDI 去重 + 菜单门控）/ 正确性 ×3 / 死代码与僵尸依赖 ×8

**前序背景**: quick task `260911-m76`（PR #19，同日合并）修了 6 批次相邻问题；本审计为 merge 后全面扫描，33+8 项经抽查确认全部残留。

## Milestone Reference

- Roadmap: `.planning/ROADMAP.md`（2026-09-12 生成：Phases 114-120）
- Requirements: `.planning/REQUIREMENTS.md` v1.33 段（38 项，进度追踪表已填充）
- Audit input: `.planning/reviews/20260911-frontend-perf-audit.md`

## Accumulated Context (carried forward from v1.32)

### Decisions preserved

- 七 gate 基线: go build / go test / 后端 coverage ≥78.33 / 前端 45 dirs / lint / type-check / diff coverage
- operlog 25 OperType 常量 + 11 敏感关键词 AST 锁值不回归
- status 0/1 普适规则（Menu visible 例外）
- Phase 92 `base.CacheProvider` / `base.GetOrSetJSON[T]` single authority
- `src/lib/apiFactory.ts` + apiFactory.invariants.test.ts dual-guard
- captcha-background 1=启用语义锁定；agent 裸 c.JSON 有意设计；operlog exclude_paths 挂账

### Blockers

- 无

## Code State

- origin/main HEAD: `727e370` (PR #19 quick/260911-m76-frontend-perf-audit merge)
- Branch Protection: direct push to main allowed (CI gates active)，PR reviews disabled（单人项目）

### Review Findings 挂账（v1.33 范围外，源自 Phase 114/115 code review）

**Phase 114（090ff9d 前）:**

- **CR-01（安全, pre-existing）**: InfoWindow 存储型 XSS——`building.name`/`address` 未转义拼入百度 InfoWindow HTML（HubeiMap.tsx:414-431 / HubeiMapGL.tsx:407-424）。修复=HTML 转义，行为变更超 v1.33 D-01 锁定范围 → 候选后续 quick task / 安全 milestone。
- **WR-01（逻辑, pre-existing）**: HubeiMapGL `MAX_ZOOM=18` + `filterBuildingsByZoom` 精确匹配 `zoom === 10` → 缩放 11-18 级二级楼宇消失。`zoom === 10` 语义被 RESEARCH 红线 + plan-checker 锁定"既有行为保持原样" → 挂账；修复建议 `>= 10`（对 2D 版行为不变）。
- **WR-02（质量, pre-existing）**: HubeiMap.tsx 与共享层逐字重复 ~100 行（toBase64/HUBEI_BOUNDARY/边界辅助），MAP_CONFIG 同名双源已分叉（10 vs 18）→ 挂账（与 Phase 120 死代码清理相邻但不同项）。
- **IN-01/02/03**: init 竞态残留缺口（无 StrictMode、无用户可见缺陷）/ polygonRef 死引用 / 一致性测试 averagePixelPosition 共同模式盲区 → 低优先级观察项。

**Phase 115（115-REVIEW.md, commit 090ff9d）:**

- **WR-05 → 非缺口**: 12 处 dashboardStore 家族整店订阅 = planner OQ-1 定案的 Phase 116（11 处 DASH-01~04）/Phase 120（useTabSync→DEAD-01 删除）白名单分工，criterion 5 口径即为此设计。
- **WR-01（pre-existing）**: my-notices/detail.tsx:29-41 成功返回空 data 时 `setLoading(false)` 不可达 → 页面永久 Spin。挂账。
- **WR-02（pre-existing）**: duty/my-duty/index.tsx:142-152 `setCurrent` 后 `loadSchedules()` stale closure 旧页码双请求 + pageSize 硬编码 10 与持久化双源不一致 → 错页。挂账。
- **WR-03（pre-existing, 与 114-WR-01 成因不同文件不同）**: building-spaces-3d/index.tsx:55 `level: 2` 无条件硬编码 → level1 恒空，默认 zoom 8 下地图零标记、统计面板与图例矛盾。挂账（与 114-WR-01 需一起修）。
- **WR-04（pre-existing）**: DashboardScopeSelector.tsx:57-88 "系统仪表盘必须 global"约束只在开启方向生效，可造矛盾态。挂账。
- **WR-06（pre-existing）**: detail.test.tsx:73-83 等用例名承诺 markAsRead/navigate/onChange 实际零断言。挂账。
- **Info 摘要**: DashboardScopeSelector 死分支 / FloorView3D 死变量空回调 / profile 假加载态 / 值班类型映射三处重复 / layoutStore DOM 同步两份实现漂移 / dual-form mock 四处手写重复 / **selector 约定无 AST 守护（建议补 apiFactory.invariants.test.ts 式扫描——candidate for follow-up phase）**。

## Session Continuity

Last session: 2026-09-14T03:22:01.883Z
Stopped at: Completed 115-03-PLAN.md

## Performance Metrics

| Phase | Plan | Duration | Notes |
|-------|------|----------|-------|
| Phase 114 P01 | 11min | 2 tasks | 2 files |
| Phase 114 P02 | 12min | 2 tasks | 2 files |
| Phase 114 P03 | 54min | 3 tasks | 3 files |
| Phase 115 P01 | 7min | 2 tasks | 2 files |
| Phase 115 P02 | 6min | 2 tasks | 7 files |
| Phase 115 P03 | 33min | 3 tasks | 10 files |

## Decisions

- [Phase ?]: 114-01: 聚类共享函数独立 cluster.ts 落位（桶=threshold、3×3 邻域、下标升序、保留 sqrt 严格小于），与旧 O(n²) 逐位一致由参考实现对照测试锁定
- [Phase ?]: 114-01: cluster.ts 零 SDK 依赖（仅 BuildingItem 类型 + utils 两函数），像素投影剥离到调用方单遍预计算 Map<id,pixel>
- [Phase 114]: 114-02: 两组件聚类改单遍预计算 Map<id,pixel> + clusterBuildings 消费，n² 地图 API 调用归零；有坐标过滤保留 effect 内保层级+坐标输入集语义
- [Phase 114]: 114-02: HubeiMap 渲染体 5 道 filter 收敛 useMemo [buildings]，两组件 zoomend/tiltend 提升 effect 作用域具名 handler 并补同引用 cleanup
- [Phase 114]: 114-03: 死组件 BuildingMarkers/CityMarkers 删除（仅 git rm 两个 .tsx），@uiw/react-baidu-map src/ 零 import 解锁 Phase 120 DEAD-02；components/ 平行第二套 utils/constants/types 与 global.d.ts 原样保留
- [Phase 114]: 114-03: cleanup 测试构造器 mock 用独立 function 表达式（内联 function 回调被 lint-staged prefer-arrow-callback 改写回箭头致 new 抛错）；TDD RED 门以突变校验等效证明
- [Phase 114]: 114-03: MAP3D-02 人工性能验证按 AUTO_MODE 自动批准，HUMAN-UAT 步骤全文留档 SUMMARY，自动侧由 Plan 01 500ms 性能冒烟兜底
- [Phase 115]: 115-01: useTabs/useLayout hook 内逐字段 selector 化（14/10 个 `useXxxStore((s) => s.field)`），返回对象/2 effect/layoutConfig/6 派生布尔一字不改，消费组件零改动（SELECTOR-01/02）
- [Phase 115]: 115-01: history 字段按 OQ-2 定案保留在 useTabs 返回对象（0 消费方但 D-04 最小 diff 优先）；useShallow 维持全库 0 使用
- [Phase 115]: 115-02: 路由层 RouteGuard/DynamicRoutes 整店订阅改 6 个逐字段 selector，41 行权限判断与 UX-only 注释逐字保留（V4 守护项）；DynamicRoutes 只改订阅形态，effect 依赖/allMenus 数组引用/菜单加载时序/routeConfigManager.initialize/getState-setState 一行不动（Phase 118 DATA-02 前向兼容）
- [Phase 115]: 115-02: 3D 页 5 处 visualizationStore 整店订阅改 11 个字段级 selector（FloorView3D 拆 4 个独立调用，selectedFloor 在 loadWorkstations useCallback 依赖引用语义不变）；action 引用不包 useMemo；不引入 useShallow（维持全库 0 使用）
- [Phase 115]: 115-03: SELECTOR-05 六处（4 action-only + 2 state 字段 user）+ NotificationBell 7 项（OQ-1 定案推荐 a 纳入）全部单字段 selector 化；detail.tsx useCallback 依赖数组与 DashboardScopeSelector 派生计算逐字保留
- [Phase 115]: 115-03: 4 个测试 mock 同任务 dual-form 化（selector ? selector(state) : state），let 变量保持调用时求值不快照；DashboardScopeSelector admin 分支断言语义零弱化（T-115-03-A mitigate）
- [Phase 115]: 115-03: criterion 5 收口——全库无参 useXxxStore() 恰好 12 处且全部命中三组白名单前缀，归属 Phase 116（11 处）/ Phase 120（1 处）；断言口径未修改（T-115-03-C）
