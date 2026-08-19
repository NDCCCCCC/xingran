# Phase 70: settings-page-redesign - Context

**Gathered:** 2026-08-19
**Status:** Ready for planning

<domain>
## Phase Boundary

重构两个设置页面的布局，对齐 v1.22 品牌设计理念（深绿 `#156031` × 铜金 `#C09058` × 奶油纸感 `#F0ECE3`、双层纸感卡片、按钮纪律）：

1. **系统设置页**（`xingran-react-frontend/src/pages/system/settings-page/` + `src/pages/system/settings/`）：Tabs × 3（邮箱配置 / API配置 / 验证码背景图）→ 左侧分类导航 + 右侧内容的新设置页骨架
2. **用户设置页**（`xingran-react-frontend/src/pages/settings/`）：单 Card 垂直表单 → 与系统设置同构的设置页骨架 + 行式设置项
3. **清理多主题时代遗留**：吸收工作区已完成的 default-theme 删除（前后端 -716 行未提交改动）、目录合并、settings 范围内的残留清理

**边界约束（继承且强化）：**
- 纯前端布局重构 —— 不改业务逻辑、不改 API 契约、不改菜单结构（sys_menu 仅允许更新组件路径的数据修正迁移）
- v1.22 已锁定的 D-01~D-05（REQUIREMENTS.md）继续有效：单品牌、brand-spec 唯一色值源、按钮纪律、light/dark 双模式 + layout/density 切换保留
- 全站硬编码色扫描不重做（QA-02 CI 门已存在）

</domain>

<decisions>
## Implementation Decisions

### 系统设置布局形态
- **D-01:** 系统设置页采用**左侧分类导航 + 右侧内容**布局（GitHub/Vercel/GitLab Settings 范式）：左侧竖排分类白卡（邮箱配置 / API配置 / 验证码背景图），右侧渲染对应配置页白卡，衬奶油画布 `#F0ECE3` 形成双层纸感；未来新增分类零成本
- **D-02:** 混合宽度策略 —— 表格/列表类内容（邮箱配置、API配置）撑满容器（对齐 v16 用户管理表格语汇）；纯表单类内容限宽（约 720-880px）避免输入行过长
- **D-03:** 激活分类**URL 参数化**（如 `/system/settings?cat=email`）：可分享、可收藏、刷新/后退自然还原，侧栏 active 态与 URL 同步；**替代**现有 `usePersistedStateController` localStorage 持久化 activeTab 方案
- **D-04:** 窄屏降级 —— `<lg` 断点时左侧竖排导航降级为顶部横向 segmented 控件，内容区全宽；桌面端保持左右结构

### 两页视觉关系
- **D-05:** 用户设置与系统设置**完全同构**：共用同一 SettingsShell 组件。用户设置左导航 = 界面 / 布局 / 数据 3 分类 + 右侧限宽表单；系统设置左导航 = 邮箱 / API / 验证码背景图。**不合并为单一菜单入口**（需动 sys_menu 菜单结构，越纯前端边界，明确不做）
- **D-06:** 用户设置表单采用**行式设置项**：每行 label + 描述 + 右侧控件（Switch/Select 右对齐），分组卡片包裹（界面设置 / 布局设置 / 数据设置）；明暗模式用分段卡片选择器（浅色/深色预览块，非 Radio.Button）

### Tab 内部 CRUD 布局
- **D-07:** 邮箱配置与 API 配置页**完整对齐 v16 用户管理范式**：顶部统计卡行（总配置数 / 启用 / 停用）+ 品牌工具栏（搜索 / 状态筛选 / 深绿主按钮「新增」）+ 双层纸感表格（绿灰表头 `#E9EFEB`、斑马纹与分割线 `#DBD7CE`）
- **D-08:** 验证码背景图页采用**图片网格墙**：缩略图卡片网格 + 图片预览 + 上传 / 启用 / 删除操作；卡片语汇与 v16 统计卡一致（贴合图片资产管理语义，不用表格列表）
- **D-09:** 配置项新增/编辑表单容器**保持 Modal**，仅接品牌令牌（focus 环品牌绿、按钮纪律 D-03 v1.22、统一圆角）；不改 Drawer

### 目录整理与残留清理
- **D-10:** 工作区未提交的 default-theme 清理改动（前后端 -716 行：handler / service / settings_router / defaultThemeApi / default-theme.tsx 等）由本 phase **首个任务原子提交吸收** —— 它正是 phase 目标「清理多主题残留（含已删除的 default-theme 入口）」的一部分
- **D-11:** 三个 settings 目录**合并为单一 `xingran-react-frontend/src/pages/system/settings/`**（SettingsShell + 邮箱/API/验证码分类子页同处），并同步更新 sys_menu 组件路径（一条数据迁移 SQL，属数据修正非业务逻辑变更）；`src/pages/settings/`（用户设置）目录名不变
- **D-12:** 残留清理边界 = **settings 范围**：持久化 tab 状态键、旧 className、settings 页内硬编码色、settingsStore 死字段；不做全仓扫描（QA-02 CI 门已覆盖）

### Claude's Discretion
- SettingsShell 组件的归属目录与命名（候选：`design-system/components/` 或 `components/layout/`）
- API 配置页统计卡的指标口径（邮箱已示例：总配置数/启用/停用；API 按现有数据定）
- URL 参数名（`?cat=` 为示例，可用 `?section=` / `?tab=` 等）
- 明暗模式分段卡片选择器的具体视觉细节（UI-SPEC 阶段细化）
- 图片网格墙的列数 / 卡片宽高比 / 空状态文案（UI-SPEC 阶段细化）
- QA 验证方式（建议：QA-04 风格改造前后截图对比 + `npm run build` / `type-check` / `lint` / `test` 回归门）
- `src/pages/system/captcha-background/` 目录与 `src/pages/system/settings/captcha-background.tsx` 的关系盘点（侦察发现的疑似重复/残留，由 researcher 确认归属）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 品牌规范（色值真相源）
- `655aa291-9bfe-4e94-ad5d-b3c8b2d24984/brand-spec.md` — 像素实测品牌令牌 + WCAG 对比度验证，唯一色值来源（v1.22 D-02）
- `.planning/REQUIREMENTS.md` — v1.22 锁定决策 D-01~D-05（按钮纪律：主按钮 `#156031` 绿底白字、hover `#2E7444`；铜金不做实心主按钮）

### 视觉基准
- `.planning/sketch/real-final-v16.webp` — 最新收敛的品牌视觉基准（用户管理页：统计卡行 + 工具栏 + 绿灰表头表格 + 双层纸感卡片 + 深绿主按钮 + 铜金点缀），本 phase 设置页对齐此语汇
- `.planning/phases/64-brand-token-layer-and-contrast-verification/test1-system-user-page.png` — 用户管理页改造后实测截图（v1.22 QA-04 产物，可对照）

### 差异记录
- `655aa291-9bfe-4e94-ad5d-b3c8b2d24984/PROTOTYPE-VS-ACTUAL.md` §6.9 — 系统设置页历史差异（原型=表单 vs 真实=Tabs×4）；本 phase 选择侧栏分类导航形态，**不**按原型表单施工

### v1.22 落地参考
- `.planning/phases/66-component-styles-and-hardcoded-color-scan/66-01-SUMMARY.md` — 组件样式落地模式（表格/卡片/按钮体系如何接令牌）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **v1.22 令牌层（Phase 64 SHIPPED）**：`src/index.css` 253 CSS 变量层、`src/design-system/tokens/colors.ts` 的 `xingranBrand` 色板（绿梯度 6 阶/铜金 4 阶/奶油中性阶，含 OKLch + WCAG 注释）、`AntdThemeBridge.tsx` 全量映射 —— 本 phase 全部**消费不新增**令牌
- **v16 范式参考实现**：`src/pages/system/user/`（用户管理页，Phase 66 品牌化落地）—— 统计卡行 + 工具栏 + 表格的组件组织方式直接参照
- `usePersistedStateController` hook（`src/hooks/usePersistedState`）—— 现有 activeTab 持久化方案，按 D-03 被 URL 参数替代后需清理调用点

### Established Patterns
- **路由解析**：`src/router/componentLoader.tsx` 用 `import.meta.glob` 扫描 `pages/`，组件路径来自 sys_menu 的 component 字段 —— 目录改名（D-11）必须同步 sys_menu 数据迁移，否则白屏
- **状态管理**：`settingsStore`（Zustand）持久化用户偏好（明暗/密度/折叠/分页），用户设置页读写入口
- **两套 Layout**（ClassicLayout / HybridLayout / InnovativeLayout）+ 密度切换（classic/comfortable）—— 设置页运行在其内容区，THEME-03 要求不回归

### Integration Points
- 系统设置菜单项 → sys_menu component 路径（D-11 迁移目标）
- `src/pages/system/settings/` 现有 3 个配置组件（email-config.tsx 11.8KB / api-config.tsx 14.6KB / captcha-background.tsx 19KB）→ 重构为分类子页并内部对齐 v16
- 后端 API 不变：邮箱配置 / API 配置 / 验证码背景图的现有接口与请求加密配置原样保留

</code_context>

<specifics>
## Specific Ideas

- 视觉基准 = `.planning/sketch/real-final-v16.webp`（用户管理页 sketch）：统计卡行带图标圆角块、标题+副标题页头、搜索+筛选+主绿按钮工具栏、绿灰表头表格、白卡衬奶油画布
- PROTOTYPE-VS-ACTUAL §6.9 曾把「原型表单 vs 真实 Tabs×4」标为**极高**差异 —— 本 phase 以侧栏分类导航形态吸收（比表单可承载重 CRUD，比 Tabs 更符合设置页惯例），原型表单形态不作为施工目标
- 明暗模式分段卡片选择器参考现代 settings 页（浅色/深色各一个预览块卡片，选中态品牌绿描边）

</specifics>

<deferred>
## Deferred Ideas

- **合并用户设置与系统设置为单一菜单入口** —— 需改 sys_menu 菜单结构（数据+权限语义变化），越纯前端边界；未来若有信息架构重组 phase 再评估
- **配置表单容器升级为 Drawer** —— 本 phase 保持 Modal（D-09），交互模式升级可未来单独评估
- **PROTO 系列逐屏对齐（PROTO-01~04）** —— REQUIREMENTS.md Future Requirements，本 phase 仅覆盖设置页两屏，不启动 53 屏对齐

### Reviewed Todos (not folded)
- `operlog.exclude_paths` 配置驱动白名单（解决 RPA 心跳日志污染）—— 后端运维项，与设置页重构无关，仅关键词浅匹配（score 0.6），留在原待办
- `v1.17-reconciliation-decisions` 资产对账决策追踪 —— 后端资产模块待办，与本 phase 无关，仅关键词浅匹配（score 0.6），留在原待办

</deferred>

---

*Phase: 70-settings-page-redesign*
*Context gathered: 2026-08-19*
