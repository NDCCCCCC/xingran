---
last_updated: 2026-08-19
update_trigger: v1.22 milestone INITIATED — ROADMAP drafted (Phases 64-67, 4 phases; design-system 层品牌令牌落地)
last_plan_update: 2026-08-18 — v1.21 SHIPPED + ARCHIVED (Phases 57-62; Phase 63 IN PROGRESS 独立前端工具链工作)
previous_update: 2026-08-13 after v1.21 plans drafted (Phases 57-61) + Phase 62 cross-AI review close-out
---

# Roadmap: XingRan-Next 运维管理系统

## Milestones

- ✅ **v1.0 工位导入部门/用户关联** — Phases 1-2 (shipped 2026-04-16)
- ✅ **v1.1 信息点导入设备端口关联** — Phase 3 (shipped 2026-04-16)
- ✅ **v1.2 可配置仪表盘生产级改造** — Phases 4-7 (shipped 2026-04-21)
- ✅ **v1.3 技术债清理** — Phases 8-10 (shipped 2026-04-27)
- ✅ **v1.4 MAC地址采集优化** — Phase 11 (shipped 2026-05-09)
- ✅ **v1.5 MAC地址历史数据管理** — Phases 12-15 (shipped 2026-06-15)
- ✅ **v1.6 API密钥管理系统** — Phase 16 (shipped 2026-05-19)
- ✅ **v1.7 前后端加密配置同步** — Phase 17 (shipped 2026-05-20)
- ✅ **v1.8 登录端点加密增强** — Phase 18 (shipped 2026-05-21)
- ✅ **v1.9 AD域控集成扩展** — Phases 19-20 (shipped 2026-05-24)
- ✅ **v1.10 网络设备权限修复** — Phase 21 (shipped 2026-05-24)
- ✅ **v1.11 AD组自动同步系统** — Phase 23 (shipped 2026-05-26)
- ✅ **v1.12 深信服桌面云集成 (22A+22B)** — Phases 22A/22B (shipped 2026-06-02)
- ✅ **v1.13 资产管理模块** — Phase 26 (shipped 2026-06-08)
- ✅ **v1.14 全局列自定义** — Phase 27 (shipped 2026-06-09)
- ✅ **v1.15 工位设备关联 + 部门物理位置映射** — Phases 28 + 39 (shipped 2026-06-10 / 2026-06-25)
- ✅ **v1.16 技术债清理 (Tech-Debt Cleanup)** — Phases 40-41 (shipped 2026-06-26)
- ✅ **v1.17 资产对账 (Asset Reconciliation)** — Phases 42-46 + Phase 47 root-cause (shipped 2026-07-03)
- ✅ **v1.18 网络设备硬件清单 (Device Component Serials)** — Phase 48 + Phase 49 gap closure (shipped 2026-07-04)
- ✅ **v1.19 网络设备写命令 (Network Device Port Write Operations)** — Phases 50-55 (shipped 2026-07-08) — see [milestones/v1.19-ROADMAP.md](milestones/v1.19-ROADMAP.md)
- ✅ **v1.20 网络设备 VLAN + 端口绑定** — Phase 56 (shipped 2026-07-10) — see [milestones/v1.20-ROADMAP.md](milestones/v1.20-ROADMAP.md)
- ✅ **v1.21 API Key 认证链修复 + 能力补全 (API Key Auth Chain Repair + Feature Completion)** — Phases 57-62 (shipped 2026-08-18) — see [milestones/v1.21-ROADMAP.md](milestones/v1.21-ROADMAP.md)
- 🚧 **v1.22 前端品牌化改造 (Frontend Brand Design-System)** — Phases 64-67 (in planning; design-system 层品牌令牌落地)

---

## Phases (v1.22) — IN PLANNING

**Milestone Goal:** 把 `brand-spec.md` 的像素实测品牌令牌(深绿 `#156031` × 铜金 `#C09058` × 奶油 `#F0ECE3`)固化进 `xingran-react-frontend/src/design-system/`,让后台内部组件与登录页品牌一致,53 屏业务页面自动继承样式。

**范围边界(用户决策 v1.22 init,锁定):** **仅 design-system 层**。业务页面自动继承样式,不逐屏改造 53 屏;`PROTOTYPE-VS-ACTUAL.md` 里的字段 / 表头 / 路由差异修正转 v1.23+ Future。

**锁定的硬约束:**

- D-01 全局替换,不保留多主题 —— 移除 6 套主题(minimal/glassmorphism/neumorphism/flat2.0/luxury-quiet/ink-amber)与 ThemeSwitcher / ColorSwitcher;保留 light / dark 双模式(单一品牌色相);保留 layoutStore 的布局与密度切换
- D-02 `brand-spec.md` 为唯一色值来源,标注「实测」直接采用,「推导」微调须重跑对比度验证
- D-03 按钮纪律:主按钮一律 `--primary` `#156031` 绿底白字(7.64:1);hover `#2E7444`(5.68:1);铜金 `#C09058` 不做实心主按钮(2.85:1 不达标);必须铜金实心时用 `#B88850` + ≥16px 半粗体白字(3.15:1 大字达标),hover 只许加深至 `#AA7B42`
- D-04 Phase 编号从 **64** 起(Phase 63「前端工具链自动化」已存在且 IN PROGRESS,不占用)
- D-05 仅 design-system 层;业务页面自动继承样式,不逐屏改造 53 屏

**回归性质:** 纯视觉层重构,不动业务逻辑、不改 API、不改数据模型。落点集中在 `xingran-react-frontend/src/design-system/` 与 `src/index.css`(253 个 CSS 变量层)。

**Phase Numbering:** 从 Phase 63 续编(64+)。整数 phase 为计划里程碑工作;小数 phase (如 64.1) 为紧急插入。Phase 63「前端工具链自动化」(.planning/phases/63-frontend-toolchain-automation/) 独立 IN PROGRESS,不占用本 milestone 编号。

- [ ] **Phase 64: 品牌令牌层落地 + 对比度验证** - 把 brand-spec 实测值写进 index.css 253 变量、tokens/colors.ts 新增 xingranBrand 常量、AntdThemeBridge 全量接品牌令牌、tokens/shadows+spacing+typography 调性对齐;含可执行对比度校验断言关键前景/背景对达标
- [ ] **Phase 65: 主题系统收敛** - 删除 6 套主题目录与 ThemeSwitcher / ColorSwitcher / themeStore 类型字段,清理 13 个消费方残留;保留 light / dark 双模式 + layoutStore 布局与密度切换
- [ ] **Phase 66: 通用组件样式 + 硬编码色扫描** - 侧边栏深绿化、表格/卡片双层纸感、按钮纪律落地、表单/标签/ECharts 接令牌;全仓扫描并以 lint / CI 阻止硬编码 `#4F46E5` / `#F1F5F9` / slate 系新增
- [ ] **Phase 67: 构建回归 + 视觉确认** - `npm run build` / `type-check` / `lint` / `test` 全绿、vendor-react gzip 体积不增、6 屏前后截图对比、登录页与后台视觉一致、milestone SHIPPED + ARCHIVED

## Phases (v1.23 启动块 — 部署稳健性 & 文档一致性) — IN PLANNING

- [ ] **Phase 68: 部署稳健性 & 文档一致性（SM2 密钥配置闭环）** - 修正 env 变量名文档-代码不一致(DEPLOY-01)、setup-server.sh secrets.env 模板补 SM2 keys 段(DEPLOY-02)、gen_sm2_keys header 注释路径修正(DEPLOY-03)、getPublicKey handler 500 时打印 useSM2 / sm2PublicKey 状态(DEPLOY-04)、config.sqlite.example.yaml use_sm2 默认值与迁移文档对齐(DEPLOY-05);来源已归档会话 public-key-500-after-subpath-fix.md

## Phases (v1.24 启动块 — 字典与状态值治理) — IN PLANNING

- [ ] **Phase 69: 字典与状态值治理（状态语义单一真相源）** - 后端新增集中状态常量包消灭 50+ 文件裸 0/1 字面量(DICT-01)、盘点 type/category 真枚举字段并 seed 进 sys_dict(DICT-02)、前端 constants.tsx 硬编码 options 分批迁移 useDict(DICT-03)、CLAUDE.md Status Value Convention 改指向常量真相源(DICT-04);现状审计(2026-08-19):字典基础设施完整但消费端近零——后端 GetDictDataByTypeKey 仅 dict_cache_impl 自用(0 个业务 service 读字典),前端 ~78 页仅 4 页用 useDict(~5%),sys_dict seed 仅 network_device_type 一类,状态语义散落后端字面量/前端 constants.tsx/CLAUDE.md 三处手工同步拷贝

## Phases (v1.25 启动块 — 系统设置页面布局重构) — IN PLANNING

- [ ] **Phase 70: 系统设置页面布局重构（对齐 v1.22 品牌设计理念）** - 按品牌设计理念(深绿 × 铜金 × 奶油纸感、双层纸感卡片、按钮纪律 D-03)重新设计系统设置页面布局,清理多主题时代遗留布局残留(含已删除的 default-theme 入口);范围:`xingran-react-frontend/src/pages/system/settings-page/` + `src/pages/settings/`;纯前端布局重构,不改业务逻辑与 API 契约;依赖 Phase 67 已交付的品牌基线

---

## Phase Details

### Phase 64: 品牌令牌层落地 + 对比度验证

**Goal**: design-system 层拥有完整的品牌令牌真相源 —— `index.css` 253 变量全量接 brand-spec 实测值,`tokens/colors.ts` 提供 TS 侧 `xingranBrand` 常量(绿/铜金/奶油三梯度,带 OKLch + WCAG 注释),`AntdThemeBridge` 把 Antd 6 `theme.token` 与 `theme.components` 全量映射到品牌令牌,`tokens/shadows+spacing+typography` 与新色彩调性对齐;并以可执行对比度校验断言关键前景/背景对达标,把品牌基线锁在测试层。

**Depends on**: Nothing (v1.22 第一个 phase;Phase 63 独立进行中,无依赖)

**Requirements**: TOKEN-01, TOKEN-02, TOKEN-03, TOKEN-04, QA-01

**Success Criteria** (what must be TRUE):

1. 打开任一后台页面(如 `/system/user`),DevTools `:root` 上 `--theme-primary` = `#156031`、`--sidebar-bg` = `#14532D`、`--theme-neutral-100` = `#E9EFEB`、`--theme-bg-primary` = `#F0ECE3`、描述 `--theme-primary-hover` = `#2E7444`、品牌 `--theme-primary-active` = `#14542E` 等关键实测值均落位,无 indigo `#4F46E5` 与冷 slate `#1e293b` / `#F1F5F9` 残留;`src/index.css` `grep -E "#4F46E5|#1e293b|#F1F5F9"` 在变量定义段零命中
2. 在 TS 侧 `import { xingranBrand } from "@/design-system/tokens/colors"` 拿到含 OKLch + WCAG 注释的常量集:绿梯度 6 阶(`#14532D` / `#156031` / `#1A6839` / `#3B784C` / `#598E5E` / `#E9EFEB`)、铜金梯度 4 阶(`#B88850` / `#C09058` / `#C89868` / `#AA7B42`)、奶油中性阶(底 / 描边 / 次级文字等),无蓝 / 紫 / indigo;同名常量在 `AntdThemeBridge.tsx` 与组件样式中被引用,值不重复
3. `AntdThemeBridge.tsx` 的 `theme.token.colorPrimary` / `colorInfo` / `colorLink` 与 `theme.components.Button` / `Table` / `Input` / `Select` / `Menu` / `Tabs` / `Tag` / `Card` 等组件级覆盖全部从品牌令牌读取,无硬编码 `#1677ff` / `#4F46E5` 残留;切换 light / dark 时 algorithm 仍为 darkAlgorithm / defaultAlgorithm,密度切换 compactAlgorithm 叠加正确
4. `tokens/shadows.ts` / `spacing.ts` / `typography.ts` 与品牌调性对齐 —— 阴影减弱(暖低饱和)、圆角统一(控件 8px 一档)、字阶收敛(衬线可选、UI 单一无衬线、数据 mono 增强);DevTools computed style 显示新值生效,旧 indigo / slate 调色板不再被消费
5. 可执行对比度校验(单测 `tokens/colors.test.ts` 或脚本 `scripts/check-contrast.mjs`)断言 brand-spec 关键前景/背景对:`#FFFFFF` on `#156031` ≥ 7.6:1、`#E0E0B0` on `#156031` ≥ 5.6:1、`#707068` on `#FFFFFF` ≥ 4.9:1、`#FFFFFF` on `#14532D` ≥ 7.0:1(深绿渐变深端)、暗色模式关键对 ≥ 4.5:1;任一对不达标即 fail,`npm test` 全绿

**Plans**: TBD

**UI hint**: yes

---

### Phase 65: 主题系统收敛

**Goal**: 移除 6 套主题与主题切换能力 —— 整站视觉归一到品牌,ThemeSwitcher / ColorSwitcher / themeStore 主题类型字段与 13 个消费方残留代码全部清零;保留 light / dark 双模式切换(品牌一套色相的深底推导)与 layoutStore 的布局 / 密度切换,确保里程碑不可逆决策(D-01)落实到位且 layout / density 不回归(D-03)。

**Depends on**: Phase 64 (品牌令牌已就位,主题移除才有可回退的真相源)

**Requirements**: THEME-01, THEME-02, THEME-03

**Success Criteria** (what must be TRUE):

1. `design-system/themes/` 下 6 套主题目录(`minimal` / `glassmorphism` / `neumorphism` / `flat2.0` / `luxury-quiet` / `ink-amber`)全部删除,`themes/index.ts` 与 `themes/theme-styles.css` 清理;`ThemeSwitcher.tsx` / `ColorSwitcher.tsx` 组件删除;`themeStore` 主题类型字段(`configuration.style` 等)与 `themes/` 引用清除,`settings/index.tsx` 主题入口移除;13 个消费方(`ConfigProvider` / `header` / `InnovativeLayout` / `TabBar` / `sidebar` / `ThemeProvider` / `main.tsx` / `settings/index.tsx` / `settingsStore` 等)`grep -rn "ThemeSwitcher|ColorSwitcher|getTheme\\(|getMinimalTheme\\(|getGlassmorphismTheme\\(|getInkAmberTheme\\("` 在 `src/` 下零命中,无死代码 / TS 错误 / unused import 警告
2. light / dark 双模式切换仍可在 settings 页操作 —— 暗色模式下 `--theme-bg-primary` / `--theme-text-primary` / `--sidebar-bg` 等关键变量有对应的深底推导(深绿底加深、铜金提亮受控、奶油转深灰纸感),`#FFFFFF` on 深绿底 / `#E0E0B0` on 深底等关键前景/背景对均满足 WCAG AA(Phase 64 的对比度校验脚本在 dark fixture 下全绿)
3. `layoutStore` 的布局切换(`ClassicLayout` / `HybridLayout` / `InnovativeLayout`)与密度切换(`classic` / `comfortable`)完整保留 —— 切换后侧边栏宽 280px ↔ 64px、列表行高密度变化正确,工具栏 / 表单 / 表格不受影响;`settings/layout` 入口与 density switcher 控件正常工作
4. `npm run type-check` / `npm run build` / `npm run lint` / `npm run test` 全绿,移除 6 套主题后 vendor-react 打包体积较 Phase 64 baseline 不增(预期下降);无新增 lint warning(`react/no-unused-prop-types` 等)
5. 验收清单:在 settings 页内,**主题切换器 UI 不再存在**;**颜色自定义(主色 / 侧边栏色)入口不再存在**(D-01 不可逆);**布局 / 密度 / 浅色 / 深色切换入口完整可用**;主页面视觉与登录页品牌一致(深绿 × 铜金 × 奶油)

**Plans:** 1 plan

Plans:

- [x] 65-01-PLAN.md — 主题系统收敛:删除 6 套主题与切换器(T1-T6 机械清除,每任务独立 commit 常绿),THEME-02 暗色推导断言(T7),THEME-03 布局/密度边界验证 + 全量 QA 门 + bundle 对比(T8),视觉冒烟 checkpoint(T9) — **T1-T8 COMPLETE (8 commits 57bdd51..b605d88, -4,357 行, SC#1-4 全过), T9 人工冒烟 PENDING USER CONFIRMATION**, 见 [65-01-SUMMARY.md](../phases/65-theme-system-consolidation/65-01-SUMMARY.md)

**UI hint**: yes

---

### Phase 66: 通用组件样式 + 硬编码色扫描

**Goal**: 侧边栏 / 表格 / 卡片 / 按钮 / 表单 / 标签 / 图表全量接品牌令牌,D-03 按钮纪律(主按钮绿底白字、铜金只做点缀)落地;全仓扫描并以 lint / CI 阻止硬编码 `#4F46E5` / `#F1F5F9` / slate 系新增,确保 v1.22 品牌效果不被后续代码破坏。

**Depends on**: Phase 65 (主题已收敛,组件样式落地才有稳定的应用上下文)

**Requirements**: COMP-01, COMP-02, COMP-03, COMP-04, QA-02

**Success Criteria** (what must be TRUE):

1. 登录后台后侧边栏底色为 `#14532D` 深绿、折叠态 64px / 展开态 280px 均正确,hover / active 强调用 `#156031` 底 + `#E0E0B0` 浅黄文字(对比度 ≥ 5.6:1);顶栏保持白底 64px,面包屑 / 全局搜索 ⌘K / 通知铃铛 / 用户菜单视觉不破;与品牌深绿(`#14532D` → `#1A6839` 渐变)同源
2. 任一业务页面(如工位管理)的表头底色为 `#E9EFEB` 绿灰淡彩、斑马纹与分割线用 `#DBD7CE`、白卡 `#FFFFFF` 衬 `#F0ECE3` 奶油画布形成双层纸感;表格排序 / 筛选 / 选中态 / 空状态 / 分页器全部接品牌令牌;Card 浮起用 1px 暖灰描边而非重阴影
3. 按钮体系满足 D-03 纪律 —— 主按钮 `#156031` 绿底白字、hover `#2E7444`(白字对比度 ≥ 7.6:1 / 5.68:1);次级(描边绿)、危险(红实心)、禁用、链接、图标按钮全套规范;`#FEF3C7` 从按钮前景移除,回归为 SM2 / SM3 / SM4 淡黄标签底;全站 `grep -rE "#C09058|#B88850" src/ --include="*.tsx" --include="*.css"` 中实心 `background-color: solid` / `Button type="primary"` 零命中,铜金仅出现在描边 / 图标 / 图表系列 / Tag 背景等点缀场景
4. 表单控件 focus 环用品牌绿(`#156031` 2px 焦点环)、校验错误态色阶统一(`#BA3630` + 行内错误文案);Tag / Badge 含 SM2 / SM3 / SM4 淡黄标签 `#FEF3C7` 规范化;Tabs 多页签与面包屑接品牌;ECharts 图表系列色为绿金梯度(`#156031` / `#3B784C` / `#598E5E` / `#C09058` / `#C89868`),无默认蓝紫
5. 全仓扫描脚本(`scripts/check-hardcoded-colors.mjs` 或 `eslint-plugin-no-hardcoded-colors` 自定义规则)对 `src/` 下 `.tsx` / `.ts` / `.css` 文件检查硬编码 `#4F46E5` / `#F1F5F9` / slate 系(`#1e293b` / `#334155` / `#475569` 等)及非品牌裸 hex,命中即非零退出;脚本集成进 `npm run lint` 与 Phase 63 的 CI(`frontend-build.yml`);既有命中通过替换为品牌 token 清零,新文件如有违规即 fail(防回归)

**Plans:** 1 plan

Plans:

- [ ] 66-01-PLAN.md — 通用组件样式 + 硬编码色扫描:AntdThemeBridge 四 Gap 补齐(T1)、侧边栏深绿化 + header 解耦(T2)、ECharts 品牌系列色(T3)、全仓色值清除 + 扫描器 lint 门(T4)、四门回归 + Phase 67 视觉清单(T5)、目检 checkpoint(T6)

**UI hint**: yes

---

### Phase 67: 构建回归 + 视觉确认

**Goal**: 全量构建 / 类型 / 单测 / 视觉四门全绿,验证 v1.22 品牌化未引入回归 —— vendor-react 打包后体积不增(预期下降)、关键 6 屏前后截图对比无布局崩坏、登录页与后台视觉一致,里程碑可 SHIPPED + ARCHIVED。

**Depends on**: Phase 66

**Requirements**: QA-03, QA-04

**Success Criteria** (what must be TRUE):

1. `npm run build` 退出码 0,`npm run type-check` / `npm run lint` / `npm run test` 全绿;`vendor-react` 打包后 gzip 体积较 v1.21 baseline(774.96 kB,见 v1.19 W4 / v1.20 close)不增(预期下降 —— 移除 6 套主题节约 ~数 kB),前后对比数值记录到 ROADMAP Progress 段;Phase 64-66 的对比度校验脚本与硬编码扫描在 CI 中运行并通过
2. 关键 6 屏(仪表盘 / 系统用户 / 工位管理 / 监控仪表盘 / 资产对账看板 / 登录页)改造前后截图对比 —— 前后两套 PNG 在同一对比画布并列展示(可视化 diff):**无布局崩坏**、**无不可读文本**、**无残留 indigo / slate 冲突色**;登录页(品牌锚点)与后台内部组件(品牌化落地)视觉一致(深绿 × 铜金 × 奶油纸感),同一品牌语汇贯穿
3. Phase 64-66 的 success criteria 全部复测通过(回归守护):`--theme-primary` / `--sidebar-bg` / `--theme-neutral-100` 等关键变量值正确,AntdThemeBridge 接品牌令牌,主题切换器已移除且 light / dark + layout / density 切换可用,侧边栏 / 表格 / 按钮 / 表单 / ECharts 颜色全量品牌化,硬编码扫描零命中
4. `MILESTONES.md` v1.22 条目落盘、`REQUIREMENTS.md` 仍标记 v1.23+ 候选(PROTO 逐屏对齐 / VIS 视觉深化);里程碑 SHIPPED 报告产出,ROADMAP archived 至 `milestones/v1.22-ROADMAP.md` + `milestones/v1.22-REQUIREMENTS.md`

**Plans**: TBD

---

### Phase 69: 字典与状态值治理（状态语义单一真相源）

**Goal**: 建立状态语义单一真相源 —— 后端新增集中状态常量包消灭 50+ 文件裸 0/1 字面量(DICT-01)、盘点 type/category 真枚举字段并 seed 进 sys_dict(DICT-02)、前端 constants.tsx 硬编码 options 分批迁移 useDict(DICT-03)、CLAUDE.md Status Value Convention 改指向常量真相源(DICT-04);消除状态语义散落后端字面量/前端 constants.tsx/CLAUDE.md 三处手工同步拷贝的漂移风险。

**Depends on**: Nothing（v1.24 启动块独立 phase;不依赖 Phase 68 部署修复）

**Requirements**: DICT-01, DICT-02, DICT-03, DICT-04

**Success Criteria** (what must be TRUE):

1. 后端存在集中状态常量包（如 `internal/constants/`），User/Role/Menu/Dept/Post 等核心实体 status 语义（0=正常/enabled、1=停用/disabled;Menu visible 例外 1=可见/0=隐藏）以具名常量导出,handler/service 层原先的裸 0/1 字面量替换为常量引用;`go build ./...` / `go test ./...` 全绿
2. type/category 真枚举字段盘点完成（network_device_type 之外新识别的枚举字段）,seed 进 `sys_dict_type` / `sys_dict_data`,迁移随启动自动执行,字典管理页面可见可维护
3. 前端 `constants.tsx` 硬编码 options 数组分批迁移 `useDict`,已迁移下拉在字典管理改值后选项随之变化（保留静态 fallback 保证字典接口异常时可用）
4. `CLAUDE.md` Status Value Convention 段落改写为指向后端常量包 + sys_dict 真相源的引用,不再作为独立手工维护的第三份拷贝

**Plans:** 8 plans

Plans:

**Wave 1**

- [x] 69-01-PLAN.md — DICT-01 基建:models 常量补齐(DictStatus/OperLog 成败/VDIServer/Notice/InfoPoint) + status_constants_test.go AST 锁值 + scripts/check-status-literals.sh 四模式 ratchet 守护(含 map/JSON 形态) + 批 1(services/system 六文件)替换
- [x] 69-06-PLAN.md — DICT-03 前端 status 线:src/constants/status.ts 共享常量模块(两套 label 语义组+vitest 锁值) + 7 文件收敛(含 menu VISIBLE 反转保护),status 不进字典

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 69-02-PLAN.md — DICT-02:migration_208 字典 seed(11 组 = 8 组 archive 存量重建 + ops_workstation_type/sys_user_sex/duty_holiday_type 新增,组级幂等+事务+WARN,Status 引用 DictStatus 常量——硬依赖 69-01) + database.go PG/SQLite 双分支挂载 + dev 库行数 smoke
- [x] 69-03-PLAN.md — DICT-01 批 2:operations 9 service + excel_handler + excel_service map/JSON 2 处替换(簇 A + D 专线三态,双包定义陷阱规避)

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 69-04-PLAN.md — DICT-01 批 3:vdi/addomain/notice/knowledge/rpa/scheduler 替换(簇 A + D 账号池三态 + E 反转;批内补缺常量 ADAccountStatus*/RPACredentialStatus* 同步登记锁值测试 watched 集合与 expectedStatusValues)
- [ ] 69-07-PLAN.md — DICT-03 前端字典线:四页 type 下拉 useDict 迁移(sys_user_sex/ops_workstation_type/duty_holiday_type/network_device_type,静态兜底+isDefault 默认值)

**Wave 4** *(blocked on Wave 3 completion)*

- [ ] 69-05-PLAN.md — DICT-01 批 4:workorder/duty/discovery/execution/dispatch/monitor + 两 handler 替换(簇 B/C CASE WHEN 自洽化) + F 簇豁免收口,守护白名单终态只剩 geocoding,go test ./... 终态门

**Wave 5** *(blocked on Wave 4 completion)*

- [ ] 69-08-PLAN.md — DICT-04:CLAUDE.md Status Value Convention 指针化(删 6 行值表格) + 字典链路端到端 blocking checkpoint(改值联动/fallback/零 UX 回归)

---

### Phase 70: 系统设置页面布局重构（对齐 v1.22 品牌设计理念）

**Goal**: 重构系统管理-系统设置页面 —— 按 v1.22 品牌化改造确立的设计理念(深绿 × 铜金 × 奶油纸感、双层纸感卡片、按钮纪律 D-03)重新设计 `xingran-react-frontend/src/pages/system/settings-page/` 与 `src/pages/settings/` 的页面布局,清理多主题时代遗留的布局残留(含已删除的 default-theme 入口),使系统设置页与品牌化后的其余页面视觉语汇一致;纯前端布局重构,不改业务逻辑与 API 契约。

**Depends on**: Phase 67 (v1.22 品牌令牌层与组件样式基线已 SHIPPED;与 Phase 69 字典治理无硬依赖,可独立规划)

**Requirements**: 无 REQ-ID —— 以 `70-CONTEXT.md` 锁定决策 D-01~D-12 为需求真相源(12/12 已映射到 plans)

**Success Criteria** (what must be TRUE):

1. **D-01/D-03/D-04**: 系统设置页(`/system/settings-page`)为左侧分类导航白卡 + 右侧内容白卡的 SettingsShell 布局;激活分类由 URL `?cat=` 参数驱动(非法值回退默认 `email`),切换 replace:true 不产生新 RouteTab;`<lg` 断点降级为顶部 Segmented block 控件——`npx vitest run src/pages/system/settings/__tests__/SettingsShell.test.tsx` 全绿锁定
2. **D-02 混合宽度**: 表格/网格类分类(邮箱/API/验证码)撑满容器,用户设置三类表单限宽 760px——由 `categories.test.ts` 纯数据断言锁定(systemSettings 无 maxWidth / userSettings 均 760)
3. **D-05 两页同构**: 用户设置(`/user/settings`)与系统设置共用同一 `design-system/components/SettingsShell.tsx`(界面/布局/数据 3 分类),不合并菜单入口
4. **D-06 行式设置项**: 用户设置每行 label+描述+右对齐控件(Switch/Select),明暗模式为分段卡片选择器(role=radio 非 Radio.Button),即改即存(updateTheme/updateLayout/updateDataPageSize + 已保存轻提示,无保存/重置按钮)——`src/pages/settings/__tests__/index.test.tsx` 全绿
5. **D-07 v16 范式**: 邮箱/API 配置页为统计卡行(总配置数/启用/停用,status 轻请求零后端改动) + 品牌工具栏(搜索/筛选/深绿「新增配置」) + `.xr-table-zebra` 双层纸感表格三段式;**D-09**: 5 个 Modal 容器结构与宽度不变(email 700/api 800,captcha 现状),api Modal 内 headers/template/auth Tabs 保留
6. **D-08 网格墙**: 验证码背景图为 `.xr-captcha-grid` 图片网格墙(4 统计卡 + 紧凑筛选 + 缩略图卡片 + Image 预览),captcha status 语义反转(1=启用)在统计卡与徽标两处一致——`captcha-background.test.tsx` 全绿
7. **D-10 首提交**: 工作区 default-theme 清理 13 文件(+42/-717)作为 phase 首个原子提交入库(双绿验证 + 噪音排除)
8. **D-11 目录合并**: 三个 settings 目录合并为 `src/pages/system/settings/`,`settings-page/` 删除;sys_menu component 经 Migrate208 迁移(`system/settings-page/index` → `system/settings/index`,id+旧值双条件幂等,双方言注册);迁移后菜单缓存按 changed 标志失效(6 个 menu: key 前缀)——`go test ./internal/core/db/migrations/ -run TestMigrate208` 全绿,重启后侧边栏「系统设置」点击不白屏
9. **D-12 残留清零**: settings 范围内死目录 `system/captcha-background/`(防御查询后删除)、settingsStore 死 actions、preset Tag、persisted activeTab 键、fallback 硬编码色全部清零(grep 门零命中;不做全仓扫描)
10. **回归收口**: `npm run build/type-check/lint/test` + `npm run deadcode` + `go build ./...` 全绿;截图矩阵(≥lg × light/dark 代表页 + <lg Segmented 降级)对照 v16 基准归档并经 checkpoint 确认

**Plans**: 7 plans

Plans:
**Wave 1**

- [ ] 70-01-PLAN.md — D-10 原子提交吸收 default-theme 清理(checkpoint 过目提交范围)

**Wave 2** *(blocked on Wave 1 completion)*

- [ ] 70-02-PLAN.md — 样式契约层(.xr-* 新类) + SettingsShell 共用骨架 + Wave 0 单测
- [ ] 70-03-PLAN.md — 邮箱/API 配置页 v16 三段式改造(Modal 契约不变)

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 70-04-PLAN.md — 验证码背景图图片网格墙 + status 反转语义锁定
- [ ] 70-05-PLAN.md — 用户设置行式化 + 即改即存 + 明暗分段卡片选择器
- [ ] 70-06-PLAN.md — 目录合并 + Migrate208 sys_menu 迁移 + 菜单缓存失效

**Wave 4** *(blocked on Wave 3 completion)*

- [ ] 70-07-PLAN.md — D-12 残留清理 + 注册表测试 + 收口七门 + 截图 checkpoint

---

## Progress

**Execution Order:**
Phases execute in numeric order: 64 → 65 → 66 → 67 (Phase 63 独立 IN PROGRESS,无依赖关系)

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 64. 品牌令牌层落地 + 对比度验证 | v1.22 | 0/TBD | Not started | - |
| 65. 主题系统收敛 | v1.22 | 0/1 | Planned | - |
| 66. 通用组件样式 + 硬编码色扫描 | v1.22 | 0/1 | Planned | - |
| 67. 构建回归 + 视觉确认 | v1.22 | 0/TBD | Not started | - |

---

## Coverage Map

15/15 v1.22 requirements mapped to exactly one phase (0 orphans, 0 duplicates):

| Requirement | Phase | Category |
|-------------|-------|----------|
| TOKEN-01 | Phase 64 | TOKEN (index.css 253 变量) |
| TOKEN-02 | Phase 64 | TOKEN (xingranBrand TS 常量) |
| TOKEN-03 | Phase 64 | TOKEN (AntdThemeBridge) |
| TOKEN-04 | Phase 64 | TOKEN (shadows/spacing/typography 调性对齐) |
| QA-01 | Phase 64 | QA (对比度自动验证) |
| THEME-01 | Phase 65 | THEME (多主题移除) |
| THEME-02 | Phase 65 | THEME (light/dark 双模式保留) |
| THEME-03 | Phase 65 | THEME (layout/density 不回归) |
| COMP-01 | Phase 66 | COMP (侧边栏深绿化) |
| COMP-02 | Phase 66 | COMP (表格 / 卡片) |
| COMP-03 | Phase 66 | COMP (按钮体系 D-03) |
| COMP-04 | Phase 66 | COMP (表单 / 标签 / 图表) |
| QA-02 | Phase 66 | QA (硬编码色扫描防回归) |
| QA-03 | Phase 67 | QA (构建 / lint / test / bundle 回归门) |
| QA-04 | Phase 67 | QA (6 屏前后视觉回归确认) |

**Inter-phase dependency graph (v1.22):**

```
Phase 64 (品牌令牌 + 对比度验证)
   │
   └─→ Phase 65 (主题系统收敛) [depends on 64]
          │
          └─→ Phase 66 (通用组件样式 + 硬编码扫描) [depends on 65]
                 │
                 └─→ Phase 67 (构建回归 + 视觉确认) [depends on 66]
```

Phase 63 (frontend-toolchain-automation) 独立 IN PROGRESS,提供 CI / lint / 测试基建,本 milestone 4 个 phase 的验证门与 lint 集成将直接受益。

---

## Archive: v1.21 Milestone History

<details>
<summary>✅ v1.21 API Key 认证链修复 + 能力补全 (Phases 57-62, shipped 2026-08-18) — 详见 [milestones/v1.21-ROADMAP.md](milestones/v1.21-ROADMAP.md)</summary>

- ✓ **Phase 57**: 认证链核心修复 + 回归测试 — 修复 `setUserContextForAPIKey` 类型断言(P0-2),消除 MultiAuth 死代码(P0-1),集成测试锁住三路径链路
- ✓ **Phase 58**: 前后端路由契约对齐 — 前端 `getAPIKey` / `updateAPIKey` / `deleteAPIKey` 改 POST 对齐后端;`CONTRACT-02` 字段命名 camelCase 对齐
- ✓ **Phase 59**: 可观测性 / 使用日志修复 — 使用日志记录时机后移,异步 goroutine detached context,`successRate` 可信
- ✓ **Phase 60**: 安全加固与启用决策 — MultiAuth 路由挂载启用决策 + API Key 哈希存储决策 + 限流响应头编码修复 + 重复索引移除
- ✓ **Phase 61**: 资源级权限矩阵 + 限流生产调优 — `RequireAPIKeyResourcePermission` resource 参数真实生效 + `RateLimitByScope` 生产接入与调优(conditional on Phase 60 AUTH-03=启用)
- ✓ **Phase 62**: 数据库核心安全加固(跨 AI 评审修复) — internal/core/db 跨 AI 评审(codex + opencode)共识 C1-C7 + 单方 HIGH/MEDIUM 全部清零,迁移安全 / 种子凭据 / 并发保护

**v1.21 shipped 14/14 v1.21 requirements + Phase 62 跨 AI 评审 14 项 review items。** 唯一遗留:Phase 58 SC#1-SC#4 端到端验证因 dev DB 性能延期(代码契约修复已提交且自动化门全绿),见 `58-01-SUMMARY` §Deferred。

</details>

---

## Archive: Pre-v1.21 Milestone History

<details>
<summary>✅ Earlier milestone phase history (v1.0–v1.20) preserved for reference</summary>

- ✓ Phases 1-2 — v1.0 工位导入部门/用户关联 (7 plans)
- ✓ Phase 3 — v1.1 信息点导入设备端口关联 (1 plan)
- ✓ Phases 4-7 — v1.2 可配置仪表盘 (11 plans)
- ✓ Phases 8-10 — v1.3 技术债清理 (9 plans)
- ✓ Phase 11 — v1.4 MAC地址采集优化 (4 plans)
- ✓ Phases 12-15 — v1.5 MAC地址历史数据 (26 plans)
- ✓ Phase 16 — v1.6 API密钥管理 (10 plans)
- ✓ Phases 17 — v1.7 加密配置同步 (6 plans)
- ✓ Phase 18 — v1.8 登录端加密 (4 plans)
- ✓ Phases 19-20 — v1.9 AD域控集成 (11 plans)
- ✓ Phase 21 — v1.10 网络设备权限修复 (1 plan)
- ✓ Phases 22A/22B — v1.12 深信服 VDI (6 plans)
- ✓ Phase 23 — v1.11 AD组自动同步 (18 plans)
- ✓ Phase 26 — v1.13 资产管理 (6 plans)
- ✓ Phase 27 — v1.14 全局列自定义 (1 plan)
- ✓ Phase 28 — v1.15 工位设备关联 (4 plans)
- ✓ Phases 30-34 — 前端性能/P0收尾/P1P2/React best practices/操作日志全模块集成 (~40 plans)
- ✓ Phases 37-39 — 前端部门选择统一/AD账号池统一/部门物理位置映射
- ✓ Phases 40-41 — v1.16 技术债清理 (8 plans)
- ✓ Phases 42-47 — v1.17 资产对账 + 根因修复 (16 plans)
- ✓ Phases 48-49 — v1.18 网络设备硬件清单 + gap closure (5 plans)
- ✓ Phases 50-55 — v1.19 网络设备写命令 (9 plans, 5 build + 1 cleanup)
- ✓ Phase 56 — v1.20 网络设备 VLAN + 端口绑定 (5 plans)

**Total shipped through v1.21**: 173 plans across 62 phases (Phases 1-56 + 57-62).

</details>

*Last updated: 2026-08-18 — v1.22 ROADMAP drafted (4 phases, Phases 64-67, 15 requirements / 4 categories, 100% coverage). v1.21 SHIPPED + ARCHIVED (Phases 57-62 / 14 v1 requirements + Phase 62 cross-AI 14 review items / C1-C7 + 单方 HIGH/MEDIUM). Phase 63 frontend-toolchain-automation 独立 IN PROGRESS,提供 CI / lint / 测试基建;v1.22 Phase 67 的回归门将直接受益于其 CI gate 与 lint-staged hook。Granularity: standard (config),natural delivery boundary 划分为 4 phases (TOKEN+QA-01 / THEME / COMP+QA-02 / QA-03+QA-04),每个 phase 保持应用在边界处可构建。*
