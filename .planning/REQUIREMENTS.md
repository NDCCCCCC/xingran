---
last_updated: 2026-08-19
update_trigger: Phase 69-05 DICT-01 backend status-literal ratchet completed; DICT-01 checkbox promoted to complete
---

# Requirements: XingRan-Next — Milestone v1.22

**Defined:** 2026-08-18
**Milestone:** v1.22 前端品牌化改造 (Frontend Brand Design-System)
**Core Value:** 后台内部视觉与登录页品牌一致 —— 一套像素实测、对比度达标的品牌令牌，全站组件自动继承，消除 indigo/slate 通用色与绿金奶油品牌的冲突。

> **性质:** 本里程碑是**视觉层重构**，不改业务逻辑、不改 API、不改数据模型。落点集中在 `xingran-react-frontend/src/design-system/` 与 `src/index.css`（253 个 CSS 变量层）。
>
> **素材基准:** `655aa291-9bfe-4e94-ad5d-b3c8b2d24984/brand-spec.md`（像素实测 + WCAG 验证）为唯一色值真相源；`admin-design-plan.md` 提供现状侦察与缺口定位；53 张 HTML 原型屏与 `refs/` 截图作为视觉参考，**不逐屏施工**。
>
> **已定位的品牌化缺口（admin-design-plan 实测）:**
> - `--theme-primary: #4f46e5`（indigo 紫蓝）配 `#FEF3C7` 浅金字 —— 主按钮完全脱离品牌
> - `--theme-neutral-100: #f1f5f9`（冷蓝灰 slate）—— 表头底色与暖米画布冲突
> - `--sidebar-bg: #1e293b`（slate 深灰）—— 应为品牌深绿 `#14532D`
> - 登录页品牌化完成度高，后台内部脱节

## 锁定决策 (v1.22 init)

| ID | 决策 |
|----|------|
| **D-01** | **全局替换，不保留多主题** —— 移除 6 套主题（minimal / glassmorphism / neumorphism / flat2.0 / luxury-quiet / ink-amber）与 ThemeSwitcher / ColorSwitcher；保留 light / dark 双模式（仅品牌一套色相）；保留 layoutStore 的布局与密度切换 |
| **D-02** | `brand-spec.md` 为唯一色值来源。标注「实测」的直接采用；「推导」的可微调但须重跑对比度验证 |
| **D-03** | **按钮纪律**：主按钮一律 `--primary` `#156031` 绿底白字（7.64:1）；hover `#2E7444`（5.68:1）。铜金不做实心主按钮（`#C09058` 上白字仅 2.85:1 不达标）；必须铜金实心时用 `#B88850` + ≥16px 半粗体白字（3.15:1 大字达标），hover 只许加深至 `#AA7B42` |
| **D-04** | Phase 编号从 **64** 起（Phase 63「前端工具链自动化」已存在且 IN PROGRESS，不占用） |
| **D-05** | 仅 design-system 层。业务页面自动继承样式，不逐屏改造 53 屏 |

## v1 Requirements

### TOKEN — 品牌令牌层

- [x] **TOKEN-01**: `src/index.css` 的 `--theme-primary*` / `--theme-neutral-*` / `--sidebar-*` 变量组全量改为 brand-spec 实测值（深绿 `#156031` / 铜金 `#C09058` / 奶油 `#F0ECE3` / 白卡 `#FFFFFF` / 描边 `#DBD7CE` / 次级文字 `#707068`），使 253 变量层成为全站品牌单一入口
- [x] **TOKEN-02**: `src/design-system/tokens/colors.ts` 新增 `xingranBrand` 色板常量 —— 绿梯度 6 阶（`#14532D` / `#156031` / `#1A6839` / `#3B784C` / `#598E5E` / `#E9EFEB`）、铜金梯度 4 阶（`#B88850` / `#C09058` / `#C89868` / `#AA7B42`）、奶油中性阶，每个色值带 OKLch 值与 WCAG 对比度注释，作为 TS 侧唯一真相源
- [x] **TOKEN-03**: `src/design-system/components/AntdThemeBridge.tsx` 的 Antd 6 `theme.token` 与 `theme.components` 覆盖全量接品牌令牌，使 Button / Table / Input / Select / Menu / Tabs / Tag / Card 等内置组件自动品牌化，无需逐组件写 CSS override
- [x] **TOKEN-04**: `tokens/shadows.ts` / `spacing.ts` / `typography.ts` 按 brand-spec 的「奶油底衬白卡双层纸感」调性对齐（阴影减弱、圆角统一、字阶收敛），消除与新色彩体系不协调的旧值

### THEME — 主题系统收敛

- [x] **THEME-01**: 彻底移除多主题能力 —— 删除 `design-system/themes/` 下 6 套主题目录、`ThemeSwitcher.tsx` / `ColorSwitcher.tsx` 组件、`themeStore` 的主题类型字段与 settings 页主题入口；清理全部 13 个消费方（`ConfigProvider` / `header` / `InnovativeLayout` / `TabBar` / `sidebar` / `ThemeProvider` / `main.tsx` / `settings/index.tsx` / `settingsStore` 等）的残留引用，无死代码、无 TS 错误
- [x] **THEME-02**: 保留 light / dark 双模式切换 —— 品牌色在暗色模式下有对应的深底推导（深绿底加深、铜金提亮受控、奶油转深灰纸感），暗色模式下关键前景/背景对同样满足 WCAG AA
- [x] **THEME-03**: `layoutStore` 的布局切换（ClassicLayout / HybridLayout / InnovativeLayout）与密度切换（classic / comfortable）完整保留且不回归 —— 本里程碑只动颜色，不动布局与间距结构

### COMP — 通用组件样式

- [x] **COMP-01**: 侧边栏深绿化 —— `--sidebar-bg` `#1e293b` → `#14532D`，hover / active 用 `#156031` 底 + `#E0E0B0` 强调文字（5.62:1），折叠态（64px）与展开态（280px）均正确；顶栏保持白底 64px，面包屑与全局搜索 ⌘K 视觉不破
- [x] **COMP-02**: 表格与卡片统一 —— 表头底 `#F1F5F9` → `#E9EFEB` 绿灰淡彩，斑马纹与分割线用 `#DBD7CE`，白卡 `#FFFFFF` 衬奶油画布 `#F0ECE3` 形成双层纸感；表格排序/筛选/选中态、空状态、分页器全部接品牌令牌
- [x] **COMP-03**: 按钮体系落地 D-03 按钮纪律 —— 主按钮 `#156031` 绿底白字、hover `#2E7444`；次级（描边绿）、危险、禁用、链接、图标按钮全套规范；`#FEF3C7` 从按钮前景移除，回归为淡黄标签底；全站无铜金实心主按钮
- [x] **COMP-04**: 表单 / 标签 / 图表接令牌 —— 表单控件 focus 环用品牌绿、校验错误态色阶统一；Tag / Badge（含 SM2 / SM3 / SM4 淡黄标签 `#FEF3C7`）规范化；Tabs 多页签与面包屑；ECharts 图表系列色改用绿金梯度（`#156031` / `#3B784C` / `#C09058` / `#598E5E` / `#C89868`）而非默认蓝紫

### QA — 质量门与防回归

- [x] **QA-01**: 对比度自动验证 —— 提供可执行的对比度校验（脚本或单测）覆盖 brand-spec 列出的关键前景/背景对（白字 on `#156031` ≥7.6:1、`#E0E0B0` on `#156031` ≥5.6:1、`#707068` on 白卡 ≥4.9:1 等），不达标即失败
- [x] **QA-02**: 硬编码色值防回归 —— 全仓扫描并清除 `src/` 下遗留的 `#4F46E5` / `#F1F5F9` / slate 系等非品牌硬编码色值，并以 lint 规则或 CI 检查阻止新增（衔接 Phase 63 前端工具链）
- [x] **QA-03**: 构建回归门 —— `npm run build` / `type-check` / `lint` / `test` 全绿；移除 6 套主题后 bundle 体积不增（预期下降），记录前后对比数值
- [x] **QA-04**: 视觉回归确认 —— 关键屏（仪表盘 / 系统用户 / 工位管理 / 监控仪表盘 / 资产对账看板 / 登录页）改造前后截图对比，人工确认无布局崩坏、无不可读文本、无残留冲突色

## Future Requirements (v1.23+)

本里程碑不交付，来源于 `PROTOTYPE-VS-ACTUAL.md` 差异清单与 53 屏原型：

### PROTO — 原型对齐（逐屏）

- **PROTO-01**: 路由前缀差异修正 —— 24 处原型路径与真实路径不一致（`/workorders` vs `/ops/workorder/orders` 等）
- **PROTO-02**: 逐屏字段 / 表头 / 工具栏差异对齐（53 屏）
- **PROTO-03**: 菜单组结构与命名对齐（密钥列表 `/system/list` 等）
- **PROTO-04**: 空状态 / 统计卡文案与原型对齐

### VIS — 视觉深化

- **VIS-01**: 3D 楼宇可视化（Three.js）配色接品牌令牌
- **VIS-02**: 登录页与后台的过渡动效统一
- **VIS-03**: 打印 / 导出样式品牌化

## Out of Scope

| 项 | 理由 |
|----|------|
| 逐屏改造 53 张原型屏 | 用户明确限定仅 design-system 层；令牌层落地后业务页面自动继承，逐屏施工投入产出比低 |
| 后端任何改动 | 纯前端视觉重构，不涉及 API / 数据模型 / 权限 |
| 布局结构与信息架构调整 | D-01 保留现有 3 套 Layout 与九组菜单结构，只换颜色不换骨架 |
| 保留 6 套主题并新增第 7 套 | 用户明确选择「全局替换，不保留多主题」（不可逆决策） |
| 新增设计资产（图标库 / 插画） | 品牌插画与图标沿用现状，本里程碑不产出新素材 |

## Traceability

15/15 v1.22 requirements mapped to exactly one phase (0 orphans, 0 duplicates):

| Requirement | Phase | Status | Category |
|-------------|-------|--------|----------|
| TOKEN-01 | Phase 64 | Pending | TOKEN (index.css 253 变量层) |
| TOKEN-02 | Phase 64 | Pending | TOKEN (tokens/colors.ts xingranBrand 常量) |
| TOKEN-03 | Phase 64 | Pending | TOKEN (AntdThemeBridge 全量映射) |
| TOKEN-04 | Phase 64 | Pending | TOKEN (shadows/spacing/typography 调性对齐) |
| QA-01 | Phase 64 | Pending | QA (对比度自动验证) |
| THEME-01 | Phase 65 | Delivered (65-01, T9 视觉确认 pending) | THEME (6 套主题 + 切换器移除) |
| THEME-02 | Phase 65 | Delivered (65-01, T9 视觉确认 pending) | THEME (light/dark 双模式保留) |
| THEME-03 | Phase 65 | Delivered (65-01, T9 视觉确认 pending) | THEME (layout/density 不回归) |
| COMP-01 | Phase 66 | Pending | COMP (侧边栏深绿化) |
| COMP-02 | Phase 66 | Pending | COMP (表格/卡片 双层纸感) |
| COMP-03 | Phase 66 | Pending | COMP (按钮体系 D-03 纪律) |
| COMP-04 | Phase 66 | Pending | COMP (表单/标签/ECharts 接令牌) |
| QA-02 | Phase 66 | Pending | QA (硬编码色扫描防回归) |
| QA-03 | Phase 67 | Pending | QA (构建/lint/test/bundle 回归门) |
| QA-04 | Phase 67 | Pending | QA (6 屏前后视觉回归确认) |

**Coverage validation**: All 15 v1.22 requirements (TOKEN-01..04 / THEME-01..03 / COMP-01..04 / QA-01..04) are mapped to exactly one phase. No orphans, no duplicates.

**Phase ordering rationale**:

- Phase 64 (TOKEN + QA-01) first — 品牌令牌是所有后续 phase 的真相源,对比度校验断言 token 值正确性,尽早暴露 brand-spec 实测值与设计目标偏差
- Phase 65 (THEME) second — 多主题移除是高风险机械重构,需要在令牌已就位(避免视觉退化)后执行;依赖令牌层提供回退色源
- Phase 66 (COMP + QA-02) third — 组件样式落地依赖稳定令牌 + 单一品牌上下文;硬编码扫描与组件落地同步,防止新代码破坏品牌
- Phase 67 (QA-03/04) terminal — 构建回归 + 视觉确认是里程碑 SHIPPED 前置门

## v1.24 Requirements

### DICT — 状态与字典单一真相源

- [x] **DICT-01**: 后端以 `internal/models` 状态常量作为语义单一真相源，清除受控 service/handler 范围内的裸 0/1 status 字面量，并以白名单 ratchet 防止回归
- **DICT-02**: 盘点 type/category 真枚举字段并 seed 进 sys_dict
- **DICT-03**: 前端 constants.tsx 硬编码 options 分批迁移 useDict
- **DICT-04**: CLAUDE.md Status Value Convention 改指向常量真相源

## v1.24 Traceability

| Requirement | Phase | Status | Notes |
|-------------|-------|--------|-------|
| DICT-01 | Phase 69 | Complete | Plans 69-01/03/04/05；`--baseline` 终态仅 geocoding F 簇 1 条 |
| DICT-02 | Phase 69 | Complete | Plan 69-02 executed；字典 seed 已落地 |
| DICT-03 | Phase 69 | Complete | Plans 69-06/69-07 executed；前端消费迁移分批完成 |
| DICT-04 | Phase 69 | Complete | Plan 69-08 |
