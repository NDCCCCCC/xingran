# Phase 70: 系统设置页面布局重构（对齐 v1.22 品牌设计理念） - Research

**Researched:** 2026-08-19
**Domain:** React 19 + Antd 6 前端页面布局重构 / sys_menu 动态路由数据迁移
**Confidence:** HIGH（现状盘点全部经代码实测；Segmented API 经官方文档验证）

## Summary

本 phase 是「纯前端布局重构 + 一条 sys_menu 数据修正迁移」。研究完成了三块：**现状盘点**（两个设置页 + 三个配置子页 + 死目录的全部文件结构、API 调用、状态管理实测）、**v16 范式参考实现定位**（`src/pages/system/user/index.tsx` + `src/index.css` 中的 stat-cards / xr-table-zebra / xr-tag 类名体系，行号级定位，可直接复用）、**迁移与风险**（sys_menu 两条菜单记录的权威值、Migrate208 迁移写法、菜单缓存 30 分钟 TTL 的失效要求、D-10 未提交改动的编译验证）。

最高价值发现：(1) **验证码背景图 status 语义反转**（`1=启用 / 0=禁用`，与 CLAUDE.md 全局 `0=启用` 惯例相反），统计卡与网格墙实现时极易踩坑；(2) **sys_menu 迁移后必须失效 Redis 菜单缓存**（`menu_cache_impl.go` TTL 30 分钟且跨重启存活），否则旧组件路径最长 30 分钟残留导致白屏；(3) `src/pages/system/captcha-background/` 是**死目录**（规范菜单种子无记录、src 内零引用）；(4) 实际路由 URL 是 `/system/settings-page` 与 `/user/settings`，CONTEXT.md 中 `/system/settings?cat=email` 为示意，D-03 落地时参数应挂在真实路径上。

**Primary recommendation:** D-10 首任务原子提交（已验证 `go build ./...` + `tsc --noEmit` 双绿，排除 `.planning/config.json` 与 `asset_columns_schema.json` 两个搭车噪音）→ 新建 SettingsShell（复用 user 页 `Layout/Sider/Content` 250px 分栏先例 + `Grid.useBreakpoint()` + `Segmented block`）→ 三个子页内部对齐 v16 → 目录合并 + Migrate208 数据迁移（含菜单缓存失效）→ 残留清理。

<user_constraints>
## User Constraints (from CONTEXT.md)

> 以下为 CONTEXT.md 锁定决策，逐字复制。研究围绕它们展开，不重新论证方案。

### Locked Decisions

- **D-01:** 系统设置页采用**左侧分类导航 + 右侧内容**布局（GitHub/Vercel/GitLab Settings 范式）：左侧竖排分类白卡（邮箱配置 / API配置 / 验证码背景图），右侧渲染对应配置页白卡，衬奶油画布 `#F0ECE3` 形成双层纸感；未来新增分类零成本
- **D-02:** 混合宽度策略 —— 表格/列表类内容（邮箱配置、API配置）撑满容器（对齐 v16 用户管理表格语汇）；纯表单类内容限宽（约 720-880px）避免输入行过长
- **D-03:** 激活分类**URL 参数化**（如 `/system/settings?cat=email`）：可分享、可收藏、刷新/后退自然还原，侧栏 active 态与 URL 同步；**替代**现有 `usePersistedStateController` localStorage 持久化 activeTab 方案
- **D-04:** 窄屏降级 —— `<lg` 断点时左侧竖排导航降级为顶部横向 segmented 控件，内容区全宽；桌面端保持左右结构
- **D-05:** 用户设置与系统设置**完全同构**：共用同一 SettingsShell 组件。用户设置左导航 = 界面 / 布局 / 数据 3 分类 + 右侧限宽表单；系统设置左导航 = 邮箱 / API / 验证码背景图。**不合并为单一菜单入口**（需动 sys_menu 菜单结构，越纯前端边界，明确不做）
- **D-06:** 用户设置表单采用**行式设置项**：每行 label + 描述 + 右侧控件（Switch/Select 右对齐），分组卡片包裹（界面设置 / 布局设置 / 数据设置）；明暗模式用分段卡片选择器（浅色/深色预览块，非 Radio.Button）
- **D-07:** 邮箱配置与 API 配置页**完整对齐 v16 用户管理范式**：顶部统计卡行（总配置数 / 启用 / 停用）+ 品牌工具栏（搜索 / 状态筛选 / 深绿主按钮「新增」）+ 双层纸感表格（绿灰表头 `#E9EFEB`、斑马纹与分割线 `#DBD7CE`）
- **D-08:** 验证码背景图页采用**图片网格墙**：缩略图卡片网格 + 图片预览 + 上传 / 启用 / 删除操作；卡片语汇与 v16 统计卡一致（贴合图片资产管理语义，不用表格列表）
- **D-09:** 配置项新增/编辑表单容器**保持 Modal**，仅接品牌令牌（focus 环品牌绿、按钮纪律 D-03 v1.22、统一圆角）；不改 Drawer
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
- `src/pages/system/captcha-background/` 目录与 `src/pages/system/settings/captcha-background.tsx` 的关系盘点（侦察发现的疑似重复/残留，由 researcher 确认归属）→ **已裁定，见「死目录裁定」节**

### Deferred Ideas (OUT OF SCOPE)

- **合并用户设置与系统设置为单一菜单入口** —— 需改 sys_menu 菜单结构（数据+权限语义变化），越纯前端边界；未来若有信息架构重组 phase 再评估
- **配置表单容器升级为 Drawer** —— 本 phase 保持 Modal（D-09），交互模式升级可未来单独评估
- **PROTO 系列逐屏对齐（PROTO-01~04）** —— REQUIREMENTS.md Future Requirements，本 phase 仅覆盖设置页两屏，不启动 53 屏对齐

</user_constraints>

<phase_requirements>
## Phase Requirements

本 phase 无 REQ-ID 映射（Requirements: TBD），以 CONTEXT.md D-01~D-12 为需求真相源。映射如下：

| ID | Description | Research Support |
|----|-------------|------------------|
| D-01 | 系统设置左侧分类导航 + 右侧内容 | user 页 `Layout/Sider/Content` 250px 分栏先例（`src/pages/system/user/index.tsx:554-565`）；`.xr-dept-panel` 左面板样式（`src/index.css:5243+`） |
| D-02 | 混合宽度策略（表格撑满 / 表单限宽 720-880px） | 现有三子页均为 Card 包 Table 全宽结构；限宽用 `maxWidth` 内联或工具类 |
| D-03 | URL 参数化替代 persisted activeTab | `useSearchParams` 8 文件先例；tab key = `location.pathname`（`useRouteTabs.ts:98`）→ searchParams 变更不产生新 tab；现有 activeTab 调用点已定位（settings-page/index.tsx:17-21） |
| D-04 | 窄屏 `<lg` segmented 降级 | `Grid.useBreakpoint()` 既有惯例（`network/mac/index.tsx:47-51` 等 4 处）；Segmented `block` prop 官方验证（antd 6.6.0 docs）；`DensitySwitcher.tsx` Segmented 先例 |
| D-05 | 两页共用 SettingsShell 同构 | 两页均为单 index.tsx 顶层组件，重构为 Shell + 分类子组件结构互不冲突；不合并菜单入口（sys_menu 仅改 component 不改 path） |
| D-06 | 用户设置行式设置项 + 明暗分段卡片选择器 | settingsStore 字段清单已盘点（theme.mode / layout.density / layout.sidebar.collapsed / data.defaultPageSize）；现表单为 Form+Divider 结构（settings/index.tsx:62-131） |
| D-07 | 邮箱/API 配置对齐 v16（统计卡+工具栏+表格） | v16 类名体系行号定位（index.css:5100-5184）；**统计 API 缺口已确认**：notification API 无 statistics 端点，list 支持 `Status *int` 筛选（`email_config_service.go:56,232`）→ 前端轻量计数方案 |
| D-08 | 验证码背景图图片网格墙 | 现 API 已提供 `previewUrl`（`types/captcha.ts:80`）与 `getCaptchaBackgroundStatistics`；**status 语义反转陷阱**（1=启用）已记录 |
| D-09 | 表单容器保持 Modal 接品牌令牌 | 现有 5 个 Modal（email 编辑/测试、api 编辑、captcha 上传/编辑）结构完好，仅样式继承 AntdThemeBridge 即可 |
| D-10 | 首任务原子提交 default-theme 清理 | 13 文件 +42/-717 清单已核实；`go build ./...` 与 `tsc --noEmit` 双绿已验证；两个搭车噪音文件已识别（应排除） |
| D-11 | 三目录合并 + sys_menu 数据迁移 | sys_menu 两条记录权威值已从 `menu_catalog_seed.sql` 摘出；Migrate208 注册点（database.go:834-847 双分支）；**菜单缓存 30min TTL 失效要求**已确认 |
| D-12 | settings 范围残留清理 | 清单已盘点：persisted tab key、死目录、死 store actions、硬编码 fallback 色、Tag 语义色 |

</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| 设置页布局骨架（SettingsShell） | Browser / Client（React 组件层） | — | 纯前端路由内组件，无 SSR/API 参与 |
| 激活分类状态（?cat=） | Browser（URL searchParams） | — | D-03 明确 URL 为单一真相源，替代 sessionStorage |
| 用户偏好读写 | Browser（settingsStore/Zustand + configService） | API（现有 /preferences 接口，不变） | 契约原样保留，仅 UI 形态变化 |
| 分类子页数据加载 | Browser（现有 lib API 封装） | API（notificationConfigApi / captcha service，不变） | 不改 API 契约 |
| 菜单 → 组件解析 | Browser（componentLoader import.meta.glob） | Database（sys_menu.component 字段） | D-11 迁移只改 DB 字段值，glob 模式不动 |
| sys_menu 组件路径修正 | Database（Go Migrate208 迁移） | Backend 启动链路（database.go 注册） | 数据修正，非业务逻辑 |
| 菜单缓存失效 | Backend（Redis menu keys） | — | 迁移改 DB 后必须清缓存，属迁移的一部分 |
| 品牌样式 | Browser（index.css 令牌 + AntdThemeBridge） | — | v1.22 已 SHIPPED，本 phase 全部消费不新增 |

## 现状盘点（Current State Inventory）

### 涉及文件全景

| 文件 | 体量 | 角色 | 处置 |
|------|------|------|------|
| `src/pages/system/settings-page/index.tsx` | 62 行 | 系统设置 Tabs×3 壳（activeTab 持久化） | D-11 后删除（并入 system/settings/） |
| `src/pages/system/settings/index.tsx` | 3 行 | barrel（导出 3 个配置页） | 改写为新设置页入口 |
| `src/pages/system/settings/email-config.tsx` | 427 行 / 11.8KB | 邮箱配置 CRUD（Card+Table+2 Modal） | 重构为分类子页 + v16 化 |
| `src/pages/system/settings/api-config.tsx` | 467 行 / 14.6KB | API 通知配置 CRUD（含 Modal 内 Tabs） | 重构为分类子页 + v16 化 |
| `src/pages/system/settings/captcha-background.tsx` | 656 行 / 19KB | 验证码背景图（统计卡+搜索+Table+2 Modal） | 重构为图片网格墙（D-08） |
| `src/pages/settings/index.tsx` | 138 行 | 用户设置（单 Card 垂直表单） | 重构为 SettingsShell + 行式设置项 |
| `src/pages/system/captcha-background/` | 7 文件 | 独立验证码背景管理页（hooks/modals/columns 拆分版） | **死目录，建议删除**（见下） |

[VERIFIED: codebase Read/grep，2026-08-19]

### 各页结构细节

**1. 系统设置壳 `settings-page/index.tsx`**：`Tabs activeKey={activeTab}` × 3（key: `email` / `api` / `captcha-background`，带图标 label），`usePersistedStateController<string>({ keyPrefix: location.pathname, keySuffix: "activeTab", defaultValue: "email" })` —— D-03 替换点、D-12 清理点。Tab 内容直接渲染 barrel 导出的三个页面组件（非 destroyInactiveTabPane 配置，默认全挂载式切换——Antd Tabs 默认懒渲染后保持）。

**2. email-config.tsx**：
- API：`getEmailConfigList({page,pageSize})` / `createEmailConfig` / `updateEmailConfig` / `deleteEmailConfig` / `testEmailConfig`（`@/lib/notificationConfigApi`）
- 状态：`configs/loading` 本地 useState + `usePagination` + 编辑/测试双 Modal
- 现布局：`div.p-6 > Card(title=邮箱服务器配置, extra=新增按钮) > Table` —— **无统计卡、无搜索工具栏**（D-07 需补）
- 表格列：configName(+默认 Tag) / host:port(+SSL Tag) / username / fromName / status Tag / createdAt / 操作(测试/编辑/删除)
- 密码脱敏惯例：更新时 `password === "******"` 则不发送（line 110-112，重构时保留）
- 注意 line 305-309：`Table onChange` 里 `setCurrent/setPageSize` 后又手动 `loadConfigs()`，与 `useEffect([paginationProps.current, pageSize])` 叠加有双请求风险——现有行为，重构时顺手收敛为 useEffect 单一路径

**3. api-config.tsx**：
- API：`getAPINotificationConfigList({page,pageSize,configType})` 等五件套（同 lib）
- 状态：`configTypeFilter`（SMS/WEBHOOK/PUSH 类型筛选）—— D-07 工具栏的现状基础
- 现布局同 email（Card+Table+1 Modal）；Modal 内嵌 `Tabs`（headers/template/auth 三段，line 351-436）—— **Modal 内 Tabs 保留**（D-09 只接令牌不动结构）
- authType 条件表单（BASIC/BEARER/APIKEY 分支渲染，line 392-431）

**4. captcha-background.tsx（settings/ 下，活代码）**：
- API：`captchaService.*`（`@/services/captcha`）—— list / upload(postFormData) / update / delete / toggleStatus / preloadCache / **getCaptchaBackgroundStatistics（现成统计端点）**
- 统计返回：`{totalCount, enabledCount, disabledCount, totalUsage?}` —— D-08 网格墙统计卡数据源现成
- **status 语义反转**：`status === 1 ? "启用" : "禁用"`（line 311、331、624）—— 与 CLAUDE.md「0=启用」全局惯例**相反**。后端契约如此，前端重构必须沿用 1=启用（勿"纠正"）[VERIFIED: codebase]
- 现布局：统计卡（Row/Col span=6 + Statistic，line 364-407）+ inline 搜索表单 + 操作按钮组 + Table（含 Image 预览列）+ 上传/编辑 Modal
- line 74-75 有死状态：`previewVisible/previewImage`（`_setPreviewImage` 从未真正使用，隐藏 Image 元素是残留）—— D-12 清理候选
- 上传约束：仅图片 + <2MB + maxCount 1（beforeUpload 返回 false 手动上传）

**5. 用户设置 `settings/index.tsx`**：
- 读写 `useSettingsStore`（initialize / updatePreferences）；表单字段路径 = preferences 嵌套路径：`["theme","mode"]` / `["layout","density"]` / `["layout","sidebar","collapsed"]` / `["data","defaultPageSize"]`
- 现布局：`div.p-6 > Card > Form(vertical) + Divider×3`（界面/布局/数据）+ Radio.Button 明暗 + 保存/重置按钮
- D-06 重构面：Radio.Button → 分段卡片选择器；Form.Item → 行式设置项（label+描述+右对齐控件）；分组 Card
- 重置语义（v1.22 收尾后）：恢复到上一次保存的偏好（form.resetFields + setFieldsValue(preferences)）—— 保留

**6. settingsStore（`src/store/settingsStore.ts`）**：
- 权威字段：`preferences.{theme.mode, layout.{type, sidebar.{collapsed,width,collapsedWidth}, density}, data.{defaultPageSize, pageSizeOptions}, language?}`；persist key `settings-storage`
- 与 themeStore/layoutStore 通过 `window.dispatchEvent("settings-changed")` 衍生同步（syncToStores）
- **死 actions**：`exportPreferences` / `importPreferences`（git grep 全 src 仅 store 自身文件命中）→ D-12 清理候选 [VERIFIED: codebase grep]
- 密度选项：`compact/comfortable/spacious`（types/config.ts:34；现页面 Select 同）—— THEME-03 不回归

### 死目录裁定（Claude's Discretion 项）

`src/pages/system/captcha-background/`（index.tsx 7027B + columns + constants + types + hooks/ + modals/ 共 7 文件）：

- **无静态引用**：全 src `git grep` 对 `system/captcha-background` 的 import 为 0 命中（唯一 consumers 是其内部相对导入）[VERIFIED: codebase grep]
- **无菜单记录**：规范菜单种子 `menu_catalog_seed.sql`（239 条，migration_207 内嵌权威目录）中 `captcha` / `背景图` 均 0 命中；archive/053 时代的「背景图查询」菜单不在规范集内 [VERIFIED: grep + migration_207 注释确认规范集=去重保留集]
- 仅存可达路径是 `import.meta.glob` 的 `/src/pages/**/index.tsx` 模式匹配（无 component 指向则永不加载）
- **裁定：确认重复/残留，属 D-12 清理范围，建议整目录删除**。执行时加一道防御：删除前在运行库 `SELECT * FROM sys_menu WHERE component LIKE 'system/captcha-background%'` 确认 0 行（防存量库有手工菜单）[ASSUMED: 存量库无该记录——规范种子与 dev 去重导入集一致，但生产库未直接查验]

### sys_menu 权威记录（D-11 迁移目标）

来源：`internal/core/db/migrations/menu_catalog_seed.sql` [VERIFIED: codebase Read]

| 菜单 | id | parent（path） | path | component（现状） | 前端路由 URL | 备注 |
|------|----|---------------|------|-------------------|--------------|------|
| 系统设置 | `308d89be-e516-4556-b949-bc22bf6ab759` | 系统管理 `d67f4240…`（`system`） | `settings-page` | `system/settings-page/index` | **`/system/settings-page`** | perms `system:config:list`，可见菜单 |
| 用户设置 | `c480cc01-c2e4-47ff-a958-6c25cfa0cf95` | 用户中心 `6e197ad8…`（`user`，hidden M） | `settings` | `settings/index` | **`/user/settings`** | hidden，perms `system:user:settings` |

**关键推论**：
1. D-11 只改 **component** 字段（`system/settings-page/index` → `system/settings/index`），**path 字段不动** → 路由 URL 保持 `/system/settings-page` 不变，收藏/外链不破坏。CONTEXT.md 中 `/system/settings?cat=email` 是示意；实际落地为 `/system/settings-page?cat=email`。
2. `componentLoader` 约定：DB 存不带 `pages/` 前缀、不带 `.tsx` 后缀的路径（`createLazyComponent` 自动补，componentLoader.tsx:174-203）；glob 模式只匹配 `{index,detail}.tsx` → **合并后新目录必须保持 `index.tsx` 命名**（`system/settings/index.tsx` 作为入口）。
3. 用户设置的 `settings/index` 路径 D-11 下不变（目录名不变），无迁移需求。

### 迁移机制与写法（Migrate208）

- 迁移走 **Go MigrateNNN 函数 + database.go 注册**（archive SQL 不执行）[VERIFIED: codebase + 项目 memory]
- 注册点：`internal/core/db/database.go:803-855` —— PG 分支在 advisory lock 块内（834-842），sqlite 分支独立（843-847）。**数据修正 UPDATE 双方言通用，两个分支都要调**（参考 207 的双分支调用形态）
- 下一个编号：**208**（现最大 207）[VERIFIED: ls migrations]
- 模板参考：`migration_207_menu_catalog_seed.go`（go:embed + 幂等 + 单事务 + WARN 不阻断）；本迁移更简单——单条 UPDATE by id，天然幂等
- **必须附加：菜单缓存失效**。`internal/services/system/menu_cache_impl.go:39-130` 菜单树/路由缓存 TTL 30 分钟（`CacheConfigMenuTree`/`CacheConfigMenuRouter`，Redis key 前缀 `xingran:`），**跨重启存活** → 迁移 UPDATE 后若不失效，前端最长 30 分钟拿到旧 `system/settings-page/index` → `createLazyComponent` Module not found → 白屏/错误组件。迁移函数内对 menu 相关 cache key 做 DeleteByPattern（或等价失效），或迁移后调用现有菜单缓存失效方法 [VERIFIED: codebase Read]

### D-10 未提交改动清单（首任务原子提交依据）

`git status` + `git diff --stat` 实测（2026-08-19）：13 文件，+42/-717 [VERIFIED: git]

| 处置 | 文件 | 说明 |
|------|------|------|
| 删除 | `internal/api/v1/system/default_theme_handler.go`（-108） | 默认主题 handler |
| 删除 | `internal/services/system/default_theme_service.go`（-178） | 默认主题 service |
| 删除 | `internal/services/system/internal/api/v1/system/default_theme_handler.go`（-93） | **误置嵌套副本**（历史事故产物） |
| 删除 | `xingran-react-frontend/src/lib/defaultThemeApi.ts`（-38） | 前端 API 封装 |
| 删除 | `xingran-react-frontend/src/pages/system/settings/default-theme.tsx`（-159） | 默认主题 Tab 页 |
| 修改 | `internal/api/v1/system/settings_router.go` | 移除 ConfigService 注入与 `/config/theme/default` 路由组 |
| 修改 | `internal/services/system/settings_service.go` / `settings_cache_impl.go` | 移除默认主题依赖 |
| 修改 | `src/pages/system/settings-page/index.tsx` / `settings/index.tsx`（barrel）/ `src/pages/settings/index.tsx` | 移除 default-theme Tab 与重置=默认主题逻辑 |

**编译验证（本次研究实测）**：`go build ./...` 通过；`npx tsc --noEmit -p tsconfig.app.json` 零输出通过 → 原子提交安全 [VERIFIED: 本 session 执行]

**两个搭车噪音（建议排除出 D-10 提交）**：
- `internal/services/system/asset_columns_schema.json`：仅 `__generated__` 时间戳变化（prebuild `sync-columns-schema` 自动再生，无实质 diff）
- `.planning/config.json`：GSD 框架自身 `_auto_chain_active: false→true` 开关，与业务清理无关

### 残留清理清单（D-12 汇总）

| # | 项 | 位置 | 动作 |
|---|----|------|------|
| 1 | persisted activeTab 键 | sessionStorage `xingran_table_state_<sanitizePathForKey('/system/settings-page')>_activeTab`（`TABLE_STATE_PREFIX` = `xingran_table_state_`，constants/storage.ts:49） | 删除调用点即可（hook 本身保留，user 页 selectedDeptId 等仍在用）；无需清存量数据（sessionStorage 会话级自灭） |
| 2 | 死目录 | `src/pages/system/captcha-background/`（7 文件） | 整目录删除（+ 删除前 DB 防御查询） |
| 3 | 死 store actions | `settingsStore.exportPreferences/importPreferences` | 删除（零外部引用） |
| 4 | 死组件状态 | `captcha-background.tsx` 的 `previewVisible/previewImage`（line 74-75）与隐藏 Image（line 638-650） | 重构网格墙时自然消除 |
| 5 | 硬编码 fallback 色 | `var(--theme-success, #3f8600)` / `var(--theme-error, #cf1322)`（captcha 统计卡两处文件共 4 点） | 重构时改纯 `var(--theme-success)` 等（v1.22 令牌已保证存在，fallback 是旧防御残留） |
| 6 | Tag 语义色 | email/api/captcha 页 `Tag color="blue/green/orange/cyan/purple"`（Antd preset） | 对齐 v16 语汇换 `xr-tag`/`xr-tag-green`/`xr-tag-gold` 品牌圆点 tag（user 页 line 149-171 先例） |
| 7 | 旧 className | `user-form-input` 等跨页复用类**保留**；settings 页内一次性 className 重构时清理 | 按文件实际重写情况定 |

## Standard Stack

### Core（全部既有依赖，零新增）

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| antd | ^6.1.1（package.json 实测） | Segmented / Grid.useBreakpoint / Image.PreviewGroup / 全部现用组件 | 项目唯一 UI 库；Segmented 官方验证可用 [CITED: ant.design/components/segmented，当前 6.6.0] |
| react-router-dom | ^7.10.1 | `useSearchParams`（D-03） | 项目路由库；8 文件既有先例 [VERIFIED: codebase grep] |
| zustand | ^5.0.9 | settingsStore 读写 | 现状延续，零改动 |
| CSS 令牌层 | v1.22 SHIPPED | `src/index.css` 253 变量 + `tokens/colors.ts` + AntdThemeBridge | **消费不新增**（CONTEXT 边界） |

### Supporting（本 phase 将消费的现有设施）

| Facility | 位置 | 用途 |
|----------|------|------|
| `.stat-cards/.stat-card/.sc-*` | `src/index.css:5100-5152` | D-07/D-08 统计卡与网格卡片直接复用 |
| `.xr-table-zebra` + `.xr-cell-id/.xr-cell-time` | `src/index.css:5155-5184` | D-07 双层纸感表格 |
| `.xr-tag/.xr-tag-green/.xr-tag-gold`、`.xr-row-ops/.xr-op-danger` | `src/index.css`（user 页消费处反查） | 状态/类型标签与操作列 |
| `.xr-dept-panel`（250px Sider 左面板先例） | `src/index.css:5243+` + user/index.tsx:554-565 | SettingsShell 左右分栏同构参考 |
| `usePagination` / `useTableManager` | `src/hooks/` | 列表页分页/表格状态（email/api 沿用 usePagination 即可，无需升级 useTableManager——重构面越小越好） |
| `PageTitle` | `src/design-system/components/PageTitle.tsx` | 备选页头（user 页选择不加——TabBar 已有标题；设置页同理可不加，planner 定） |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Antd `Tabs`（现状） | SettingsShell 左导航（D-01） | 已锁定，不讨论 |
| 自绘分段控件 | Antd `Segmented`（D-04） | 已锁定方向；Segmented block + options 结构官方支持 |
| Drawer 表单 | Modal（D-09） | 已锁定保持 Modal |

**Installation:** 无。本 phase **零新增依赖**。

## Package Legitimacy Audit

> 本 phase 不安装任何外部包。全部能力来自项目既有依赖（antd / react-router-dom / zustand，版本见 package.json 实测）。
>
> **Packages removed due to slopcheck [SLOP] verdict:** none
> **Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
浏览器
  │
  ├─ 菜单加载：sys_menu（DB）── menu_cache_impl（Redis, TTL 30min）── /system/users/menus
  │        │                                                        │
  │        └── Migrate208: component 'system/settings-page/index' → 'system/settings/index'
  │             └── ★ 迁移内必须失效 menu 缓存 key（否则 30min 旧路径白屏）
  │
  ├─ createLazyComponent('system/settings/index') ── import.meta.glob('/src/pages/**/index.tsx')
  │        │
  │        ▼
  ┌─ SettingsShell（共用骨架，两页实例化）─────────────────────────┐
  │  useSearchParams(?cat=) ──→ activeCat（URL 单一真相源, D-03）   │
  │        │                                                      │
  │        ├─ Grid.useBreakpoint().lg ?                           │
  │        │     ├─ true  → Sider(竖排分类卡) + Content(右侧白卡)  │
  │        │     └─ false → Segmented(block, 顶部横向) + 全宽内容  │
  │        │                                                      │
  │        └─ 分类注册表 categories=[{key,label,icon,content,maxWidth?}]
  └──────────────────────────────────────────────────────────────┘
        │                                        │
        ▼ 系统设置实例（/system/settings-page?cat=）   ▼ 用户设置实例（/user/settings?cat=）
        ├─ email: 统计卡行+工具栏+表格(全宽, D-07)      ├─ 界面: 行式项+分段卡片明暗选择器(D-06)
        ├─ api: 同上(类型筛选)                          ├─ 布局: 密度/侧栏折叠行式项
        └─ captcha: 统计卡+图片网格墙(D-08)             └─ 数据: 分页大小行式项（限宽 720-880px, D-02）
        │
        ▼ 数据（不变）
  notificationConfigApi ── POST /system/notification/email-configs/* 等
  captchaService ── POST /system/captcha-backgrounds/*
  settingsStore ── GET/PUT /system/settings/preferences（configService）
```

### Recommended Project Structure（D-11 合并后）

```
xingran-react-frontend/src/pages/
├── system/
│   └── settings/                    # 合并后的唯一系统设置目录
│       ├── index.tsx                # 入口（sys_menu component 指向此处；SettingsShell 系统设置实例）
│       ├── SettingsShell.tsx        # 共用骨架（或放 design-system/components/，Claude 裁量）
│       ├── email-config.tsx         # 邮箱配置分类子页（v16 化）
│       ├── api-config.tsx           # API 配置分类子页（v16 化）
│       └── captcha-background.tsx   # 验证码背景图网格墙
├── settings/
│   └── index.tsx                    # 用户设置（目录名不变，D-11；SettingsShell 用户实例 + 行式设置项）
└── system/
    └── captcha-background/          # ★ 删除（死目录）
```

> SettingsShell 归属裁量建议：`design-system/components/` 更贴合「两页共用骨架 + 令牌消费」性质（与 LayoutSwitcher/DensitySwitcher 同级）；若 planner 认为「页面级布局」语义更强则 `components/layout/`。两处均有先例，无技术差异。

### Pattern 1: v16 列表页范式（D-07 直接套用）

**What:** 统计卡行 + Card 工具栏 + Card 表格的三段式；全部类名消费 index.css 既有定义
**When to use:** email-config / api-config 两个分类子页
**Example:**
```tsx
// Source: src/pages/system/user/index.tsx:531-660（项目内 v16 落地范本，直接对照改写）
<div className="stat-cards">
  <div className="stat-card">
    <div className="stat-label">配置总数</div>
    <div className="stat-value">{stats.total}</div>
    <div className="stat-trend">全部通知渠道</div>
  </div>
  <div className="stat-card sc-green">…启用…</div>
  <div className="stat-card sc-gray">…停用…</div>
</div>
<Card style={{ marginBottom: 14 }}>   {/* 工具栏卡 */}
  <div style={{ display: "flex", justifyContent: "space-between", flexWrap: "wrap", gap: 16 }}>
    <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>…</Form>
    <Space>
      <Button type="primary" icon={<PlusOutlined />}>新增配置</Button>
    </Space>
  </div>
</Card>
<Card>
  <Table className="xr-table-zebra" size="middle" … />
</Card>
```

### Pattern 2: SettingsShell 骨架（D-01/D-03/D-04/D-05）

**What:** 分类注册表驱动的共用壳；URL 参数为唯一激活态真相源；断点降级
**When to use:** 两页共用
**Example:**
```tsx
// 组合要点（均项目内既有先例，非新发明）：
const [searchParams, setSearchParams] = useSearchParams();      // 8 文件先例
const screens = Grid.useBreakpoint();                            // network/mac 4 处先例
const activeCat = categories.some(c => c.key === searchParams.get("cat"))
  ? searchParams.get("cat")! : categories[0].key;                // 非法值回退首分类
const setCat = (key: string) => setSearchParams({ cat: key }, { replace: true });
// replace: true —— 分类切换不污染 history 栈（回退离开页面而非逐分类回退；planner 可改 push 语义）

// 桌面端：user/index.tsx:554-565 的 Layout/Sider/Content 同构
<Layout style={{ background: "transparent" }}>
  <Sider width={220} className="xr-…-panel" …>  {/* 竖排分类白卡，active 品牌绿 */}
    {categories.map(c => <CategoryCard key={c.key} active={c.key === activeCat} …/>)}
  </Sider>
  <Content>{renderActive()}</Content>
</Layout>

// 窄屏：<Segmented block value={activeCat} onChange={setCat}
//   options={categories.map(c => ({ label: c.label, value: c.key, icon: c.icon }))} />
```

### Pattern 3: 行式设置项（D-06）

**What:** 每行 label+描述+右对齐控件的设置行（非表单流式布局）
**When to use:** 用户设置三分组
**Example:**
```tsx
// 行结构建议（Antd 原语组合，样式接令牌）：
<div className="setting-row">  {/* flex justify-between align-center, 分割线 var(--theme-border-primary) */}
  <div>
    <div className="setting-row-label">深色模式</div>
    <div className="setting-row-desc">调暗界面色调，适合夜间使用</div>
  </div>
  <Switch checked={mode === "dark"} onChange={…} />   {/* 或 Select 右对齐 */}
</div>
// 明暗分段卡片选择器：两张预览块卡（mini 界面缩略），选中态品牌绿描边（UI-SPEC 细化）
// 注意：现页面是 Form 批量提交；行式化后建议改为「即改即存」（每行 onChange 直接调
// updateTheme/updateLayout/updateDataPageSize）或保留底部保存按钮 —— planner 决策点
```

### Pattern 4: 图片网格墙（D-08）

**What:** 统计卡行 + 卡片网格（复用 `.stat-card` 语汇）替代 Table
**When to use:** captcha-background 分类页
**要点：**
- 网格容器：CSS grid `repeat(auto-fill, minmax(Npx, 1fr))`（同 `.stat-cards` 思路，N 由 UI-SPEC 定）
- 图片：`<Image>` 自带预览（`preview={{ src: record.previewUrl }}`）——现 Table 已用同 API（captcha-background.tsx:259-267），网格墙直接沿用，无需额外灯箱库
- 卡片操作：编辑/启停/删除（Antd Dropdown 或卡脚 Button 组，UI-SPEC 定）
- 分页/筛选：保留 `usePagination` + 筛选（fileName/形状/难度/状态）；上传 Modal 原样（D-09）

### Anti-Patterns to Avoid

- **手写 hex 色值**：任何新样式用 CSS 变量（`var(--theme-*)`）或 xingranBrand 常量；`npm run lint` 内置 `check-hardcoded-colors.mjs` 会直接挂 [VERIFIED: package.json scripts + Phase 66 交付]
- **Tabs 残留**：新壳不得再用 Antd Tabs 做分类切换（D-01/D-03 已锁定替代方案）
- **动 sys_menu 的 path 字段**：只改 component（改 path = 改 URL = 破坏收藏/权限语义，越界）
- **"纠正" captcha status 语义**：1=启用 是后端契约，前端显示层适配即可
- **Modal 改 Drawer**：D-09 明确不做
- **useEffect 依赖不稳定**：分类注册表、params 对象必须 useMemo/模块级常量（CLAUDE.md 强制条款）

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 图片预览/灯箱 | 自写 overlay/zoom | Antd `<Image preview>` | 现页已用，内置缩放/多图 |
| 响应式断点判断 | window.innerWidth 监听 | `Grid.useBreakpoint()` | 项目惯例，SSR 安全，4 处先例 |
| 分类激活持久化 | 自写 storage 同步 | `useSearchParams` | D-03 锁定；浏览器后退/分享免费获得 |
| 统计卡/表格/标签样式 | 自写 CSS 卡片 | `.stat-cards`/`.xr-table-zebra`/`.xr-tag` 类 | v1.22 已交付且 QA-02 扫描器守护 |
| 分段控件 | 自写 radio 组 | Antd `Segmented`（block/orientation 齐备） | 官方组件，DensitySwitcher 先例 |
| 组件懒加载路径解析 | 改 componentLoader | 只改 DB component 值 | glob 机制已含白名单安全校验，勿动 |

**Key insight:** 本 phase 的全部视觉/交互原语（统计卡、表格语汇、分段控件、图片预览、断点、URL 状态）在项目内或 Antd 内都有现成实现，phase 工作量是「组合与迁移」，不是「造轮子」。

## Runtime State Inventory

> 本 phase 含目录合并 + sys_menu 数据迁移 + 状态存储替换，五类逐项回答。

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| **Stored data** | sys_menu 1 行（系统设置，id `308d89be…`）：component 字段 `system/settings-page/index` → `system/settings/index` | **数据迁移**（Migrate208 UPDATE，幂等） |
| **Live service config** | Redis 菜单缓存（`xingran:menu:*` 系 key，TTL 30min，跨重启存活；menu_cache_impl.go:39-130）；**Redis 在重启后仍会喂旧 component 路径** | 迁移函数内 DeleteByPattern 失效菜单缓存（代码侧，随迁移原子执行） |
| **OS-registered state** | None — verified：纯 Web 前端 + Go 后端，无任务计划/pm2/launchd 注册 | 无 |
| **Secrets/env vars** | None — verified：改动不触密钥；`VITE_ENABLE_REQUEST_ENCRYPTION` 等配置与设置页无关 | 无 |
| **Build artifacts / installed packages** | ① `asset_columns_schema.json` 时间戳再生（prebuild 自动产物）；② sessionStorage 旧键 `xingran_table_state_*settings-page*_activeTab`（会话级自灭，无需清理动作，仅删代码调用点）；③ 死目录 `system/captcha-background/`（glob 打包为懒 chunk，删除即从构建消失） | ① 随 D-10 提交或排除（内容无实质变化）；② 代码删除即可；③ 整目录删除 |

## Common Pitfalls

### Pitfall 1: captcha status 语义反转
**What goes wrong:** 按 CLAUDE.md 全局惯例（0=启用）写统计卡/启停逻辑，验证码背景图实际是 **1=启用 / 0=禁用**
**Why it happens:** 后端契约历史上即如此；全局惯例有例外模块
**How to avoid:** 网格墙/统计卡沿用 `status === 1 ? 启用 : 禁用`；代码注释标注「例外模块」
**Warning signs:** 启停按钮反向、统计卡启用/停用数对调

### Pitfall 2: 迁移后菜单缓存残留 → 白屏
**What goes wrong:** Migrate208 更新 sys_menu 后，Redis 菜单缓存最长 30 分钟返回旧 component 路径 → `createLazyComponent` Module not found → 页面渲染错误组件
**Why it happens:** 菜单缓存 TTL 30min 且 Redis 跨重启存活；迁移只写了 DB
**How to avoid:** 迁移函数内失效 menu 缓存 key（与 UPDATE 同事务/同函数）；验证步骤含「重启后立即访问设置页」
**Warning signs:** 部署后设置页白屏但 30 分钟后自愈

### Pitfall 3: URL 参数与 Tabs/keepAlive 系统冲突
**What goes wrong:** 担心 ?cat= 切换触发新 tab 或 keepAlive 重挂载
**Why it happens:** 对 tab key 生成机制不了解
**How to avoid:** tab key = `location.pathname`（useRouteTabs.ts:98），**searchParams 变更不产生新 tab** [VERIFIED: codebase]；分类切换建议 `replace: true` 避免历史栈堆积
**Warning signs:** 切分类时 tab 数量增加（不应发生，出现即 bug）

### Pitfall 4: 目录合并后 glob 失配
**What goes wrong:** 新目录没有 `index.tsx`（例如叫 `main.tsx` 或 `settings.tsx`）→ import.meta.glob 不匹配 → 菜单指向组件加载失败
**Why it happens:** glob 模式是 `/src/pages/**/{index,detail}.tsx`（componentLoader.tsx:34-47）
**How to avoid:** 合并目录入口必须命名 `index.tsx`；DB component 值不带 `pages/` 前缀不带 `.tsx`
**Warning signs:** console `[createLazyComponent] Module not found`

### Pitfall 5: email/api 统计卡无后端统计端点
**What goes wrong:** 直接照抄 captcha 的 `getCaptchaBackgroundStatistics` 模式 → notification API 没有对应端点
**Why it happens:** notification_config_router.go 只有 CRUD 六+五个端点，无 statistics [VERIFIED: codebase Read]
**How to avoid:** 前端轻量计数：list 请求支持 `Status *int` 筛选（email_config_service.go:56,232）→ 两次 `pageSize=1` 请求只取 `total`（总数可从主列表 total 获得，启用/停用各一次轻请求）；或单次全量拉取（配置量级小）。纯前端边界内解决，**不新增后端端点**
**Warning signs:** 试图给后端加 /statistics 路由（越 D-11 数据修正边界）

### Pitfall 6: Modal 内 Antd Tabs 与外层形态混淆
**What goes wrong:** 重构时把 api-config Modal 内的 headers/template/auth Tabs 也"清理"掉
**How to avoid:** D-09 锁定 Modal 容器与内部结构不动，仅接令牌；只有**页面级**分类 Tabs 被替换
**Warning signs:** diff 里出现 Modal 内 Tabs 删除

### Pitfall 7: useEffect 无限循环（CLAUDE.md 强制条款）
**What goes wrong:** SettingsShell 里 categories 数组/params 对象在依赖里每次渲染新引用 → 死循环
**How to avoid:** categories 模块级常量或 useMemo；searchParams 的消费用原始 string 值做依赖
**Warning signs:** 网络面板 list 请求连发

## Code Examples

### 现有 persisted activeTab（被替换对象）
```tsx
// Source: src/pages/system/settings-page/index.tsx:16-21（现状，D-03 后删除）
const [activeTab, setActiveTab] = usePersistedStateController<string>({
  keyPrefix: location.pathname,
  keySuffix: "activeTab",
  defaultValue: "email",
});
```

### Migrate208 骨架（对照 207 改写）
```go
// Source: 模式取自 internal/core/db/migrations/migration_207_menu_catalog_seed.go + database.go:840-847
func Migrate208UpdateSettingsMenuComponent(db *gorm.DB) error {
	// 数据修正（Phase 70 D-11）：系统设置组件路径目录合并
	res := db.Model(&models.Menu{}).
		Where("id = ?", "308d89be-e516-4556-b949-bc22bf6ab759").
		Where("component = ?", "system/settings-page/index").
		Update("component", "system/settings/index")
	if res.Error != nil { return fmt.Errorf("migration 208: %w", res.Error) }
	// ★ 同函数内失效菜单缓存（Redis menu keys DeleteByPattern）——具体 key 模式
	//   参照 menu_cache_impl.go 的 cacheKey 构造与现有失效方法
	return nil
}
// 注册：database.go PG 分支（advisory lock 块内）+ sqlite 分支 双调用
```

### Segmented 窄屏降级（官方 API）
```tsx
// Source: antd 6 官方文档（block 自 4.20 提供；options 支持 icon/label/value）
<Segmented
  block                     // 撑满父宽
  value={activeCat}
  onChange={(v) => setCat(v as string)}
  options={categories.map((c) => ({ label: c.label, value: c.key, icon: c.icon }))}
/>
// 注意：v6 size 语义为 large|medium|small（v5 的 middle 已更名 medium），默认 medium 即可
```
[CITED: ant.design/components/segmented]

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Antd Tabs 承载设置分类 | 左侧导航 Shell + URL 参数（D-01/D-03） | 本 phase | GitHub/Vercel Settings 惯例对齐 |
| Antd 5 `size="middle"` | Antd 6 `size="medium"` | antd 6.0 | Segmented/Table 等传 size 时注意（默认值不受影响） |
| sessionStorage 持久化 tab | URL searchParams | 本 phase | 可分享/收藏/后退还原 |

**Deprecated/outdated:**
- Antd `List` 组件标记 DEPRECATED（6.6.0 文档）——网格墙不用 List，用自定义 grid 容器 + Image [CITED: ant.design]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | 存量生产库 sys_menu 无 `system/captcha-background%` component 记录（死目录可删） | 死目录裁定 | 中：若有手工菜单，删除后该菜单白屏 → 已建议删除前防御查询（checkpoint 化解） |
| A2 | 分类切换用 `replace: true` 的 history 语义符合预期 | Pattern 2 | 低：若用户期望逐分类后退，改 push 即可（一行） |
| A3 | 用户设置行式化后改「即改即存」或保留保存按钮由 planner 决策（两案均可行） | Pattern 3 | 低：交互偏好问题，不阻塞 |
| A4 | Migrate208 内可直接复用现有菜单缓存失效方法（未逐行核对 cache service 的失效 API 名称） | 迁移机制 | 低：实现时按 menu_cache_impl.go 实际方法名调整 |
| A5 | email/api 统计卡用「主列表 total + 两次 status 筛选轻请求」口径可行（list 的 Status 筛选已确认存在于 service 层，前端 req 结构透传未逐字段核对） | Pitfall 5 | 低：若前端 API 封装未透传 status，加一个可选参数即可（不越契约，service 已支持） |

## Open Questions (RESOLVED)

> 三问实质均已由 approved 的 70-UI-SPEC 裁定闭环，逐条内联标注如下（后续施工以 UI-SPEC 对应条目为准）。

1. **用户设置的保存交互（A3）** —— (RESOLVED → UI-SPEC IC-1)
   - What we know: 现页面是 Form + 底部保存按钮；settingsStore 有单字段 updateTheme/updateLayout/updateDataPageSize 即改即存能力
   - What's unclear: 行式化后是否保留「保存」按钮
   - Recommendation: planner 定；倾向即改即存（现代 settings 惯例，与行式形态匹配），重置按钮语义需同步定义
   - Resolution: 采纳「即改即存」——每行控件 onChange 直接调 settingsStore 单字段更新方法 + message.success("已保存") 轻提示，移除底部「保存/重置」按钮（UI-SPEC IC-1 裁定）
2. **SettingsShell 归属目录（Claude 裁量项）** —— (RESOLVED → UI-SPEC L-1)
   - What we know: `design-system/components/`（与 LayoutSwitcher 同级）或 `components/layout/` 均有先例
   - Recommendation: `design-system/components/`（跨页共用 + 令牌消费性质更贴合）
   - Resolution: 采纳建议——落位 `xingran-react-frontend/src/design-system/components/SettingsShell.tsx`，与 LayoutSwitcher / DensitySwitcher 同级（UI-SPEC L-1 归属目录裁定）
3. **图片网格墙的筛选/分页保留形态** —— (RESOLVED → UI-SPEC L-3)
   - What we know: 现有 fileName/形状/难度/状态筛选 + 分页；图片资产量级通常小
   - What's unclear: 网格墙是否保留完整筛选栏（D-08 未细说）
   - Recommendation: 保留筛选（降为工具栏内紧凑形态）+ 保留分页（list API 契约不变）；UI-SPEC 细化
   - Resolution: 采纳建议——筛选保留但降为工具栏内紧凑形态（文件名/拼图形状/难度/状态），分页保留（usePagination 沿用，list API 契约不变）（UI-SPEC L-3 工具栏卡裁定）

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js ≥24 | 前端构建 | ✓ | 24+（CLAUDE.md 约定，nvm 环境） | — |
| Go 1.24 | 后端编译 + Migrate208 | ✓ | 1.24 | — |
| npm | 前端包管理 | ✓ | — | — |
| PostgreSQL / SQLite | sys_menu 迁移验证 | ✓ | 双方言均需验证（迁移双分支注册） | — |
| Redis | 菜单缓存失效验证 | ✓（dev 环境） | 7.4 | 本地 memory cache（缓存不生效则无残留风险） |

**Missing dependencies with no fallback:** 无
**Missing dependencies with fallback:** 无

## Validation Architecture

> `workflow.nyquist_validation: true`（.planning/config.json）→ 本节必填。

### Test Framework

| Property | Value |
|----------|-------|
| Framework | vitest 4.0.18 + jsdom + @testing-library/react 16.3.2 |
| Config file | `xingran-react-frontend/vitest.config.ts`（globals + setup `src/test/setup.ts`） |
| Quick run command | `cd xingran-react-frontend && npx vitest run src/pages/system/settings/__tests__/` |
| Full suite command | `cd xingran-react-frontend && npm run test -- run`（注意裸 `npm run test` 进 watch 模式） |

### Phase Requirements → Test Map（D-01~D-12 即需求）

| ID | Behavior | Test Type | Automated Command | File Exists? |
|----|----------|-----------|-------------------|-------------|
| D-03 | ?cat= 参数驱动激活分类；非法值回退首分类；replace 语义 | unit（jsdom + MemoryRouter） | `npx vitest run src/pages/system/settings/__tests__/SettingsShell.test.tsx -t "cat"` | ❌ Wave 0 |
| D-04 | `<lg` 断点渲染 Segmented、≥lg 渲染 Sider（mock useBreakpoint） | unit | `npx vitest run src/pages/system/settings/__tests__/SettingsShell.test.tsx -t "breakpoint"` | ❌ Wave 0 |
| D-06 | 行式设置项 onChange → settingsStore 对应字段更新 | unit | `npx vitest run src/pages/settings/__tests__/index.test.tsx` | ❌ Wave 0 |
| D-07/D-08 | 分类注册表完整性（两页各自 3 分类、icon/label/key） | unit（纯数据断言） | `npx vitest run src/pages/system/settings/__tests__/categories.test.ts` | ❌ Wave 0 |
| D-08 | 网格墙 status===1 → 启用徽标（反转语义锁定） | unit | 并入 SettingsShell/网格墙测试文件 | ❌ Wave 0 |
| D-11 | 迁移幂等 + component 值正确 | Go unit（对照 207 test 形态） | `go test ./internal/core/db/migrations/ -run TestMigrate208` | ❌ Wave 0 |
| D-12 | 死代码清理后无悬空导出 | 既有门 | `npm run lint` + `npm run deadcode`（knip） | ✅（既有命令） |
| 全部门 | 构建/类型/测试回归 | gate | `npm run build && npm run type-check && npm run lint && npm run test -- run` | ✅ |
| 视觉 | 两页 × 断点 × 明暗模式截图对比（QA-04 风格） | manual-only（chrome-devtools 辅助，Phase 66 T6 / Phase 67 先例） | — | checkpoint |

manual-only 理由：视觉语汇（双层纸感/统计卡比例/网格墙观感）无像素级自动化基线，沿用 QA-04 人工确认 + 截图归档惯例。

### Sampling Rate

- **Per task commit:** `npx vitest run <本任务触及的测试文件>` + `npm run type-check`（<30s 级）
- **Per wave merge:** `npm run build && npm run lint && npm run test -- run` + `go build ./...`（触后端迁移的任务）
- **Phase gate:** 四门全绿 + 迁移双方言验证（PG + sqlite 启动日志确认 Migrate208 执行）+ 前后截图对比归档，然后 `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `xingran-react-frontend/src/pages/system/settings/__tests__/SettingsShell.test.tsx` — D-03/D-04/D-08
- [ ] `xingran-react-frontend/src/pages/settings/__tests__/index.test.tsx` — D-06
- [ ] `internal/core/db/migrations/migration_208_*_test.go` — D-11 幂等（对照 migration_207_menu_catalog_seed_test.go 写法）
- [ ] 无 conftest 级缺口（`src/test/setup.ts` 已存在）

## Security Domain

> 纯前端布局重构 + 数据修正迁移，不新增攻击面；`security_enforcement` 未显式关闭，简表如下。

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | 不触认证链 |
| V3 Session Management | no | 不触 token |
| V4 Access Control | 间接（低） | sys_menu 迁移只改 component 值不改 perms/visible；权限语义零变化 |
| V5 Input Validation | no（存量维持） | 表单校验规则原样保留（email/api Modal 的 required/oneof） |
| V6 Cryptography | no | 不触加密 |

### Known Threat Patterns for React 重构

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 组件路径注入（sys_menu.component → glob） | Tampering | componentLoader 白名单 + 危险模式校验已存在（componentLoader.tsx:18-27）；迁移只写固定常量值，不引入用户输入 |
| XSS via 菜单 meta | Tampering | routeGenerator XSS 模式校验已存在；本 phase 不改 |

## Project Constraints (from CLAUDE.md)

- **纯前端边界**：不改业务逻辑、不改 API 契约（唯一后端改动 = Migrate208 数据修正迁移 + D-10 吸收的既有未提交删除）
- **Status Value Convention**：全局 0=启用/1=停用；**例外：captcha 背景 1=启用**（本次研究确认）
- **前端 API 调用**：只用 `@/lib/api` 包装函数 / 既有 lib 封装，禁 raw axios（本 phase 不新增调用，沿用 notificationConfigApi / captcha service）
- **useEffect 依赖稳定**：对象/数组必须 memoize（CLAUDE.md 强制，Pitfall 7）
- **编译验证纪律**：任何 Go 改动后 `go build ./...`；前端改动后 `npm run build` / `type-check`
- **临时文件清理**：不产生根级 `temp_*.go` / `test_*.go`
- **Git 纪律**：提交前过质量门；D-10 原子提交需用户确认（CLAUDE.md 要求 commit 前显式确认——GSD 工作流内的 phase 提交按流程走，但 D-10 吸收的是既有工作区改动，建议 planner 设 checkpoint 让用户过目提交范围）
- **operlog 约定**：不适用（无业务写操作 handler 改动）
- **GSD Workflow Enforcement**：本 phase 经 `/gsd:plan-phase` 进入，符合入口要求

## Sources

### Primary (HIGH confidence)

- 代码实测（Read/git grep，2026-08-19）：全部现状盘点文件、sys_menu 种子（menu_catalog_seed.sql:84,201,230,232）、迁移注册（database.go:803-855）、菜单缓存 TTL（menu_cache_impl.go:39-130）、componentLoader/routeGenerator、index.css v16 类名（5100-5184, 5243+）、user 页范本、tabsStore/useRouteTabs、settingsStore、types/config.ts、usePersistedState、package.json
- 本 session 执行验证：`go build ./...` 通过；`npx tsc --noEmit` 通过；git diff/status 核对 D-10 清单
- `.planning/phases/66-component-styles-and-hardcoded-color-scan/66-01-SUMMARY.md` — v1.22 组件落地模式与四门回归先例

### Secondary (MEDIUM confidence)

- antd Segmented 官方文档（ant.design/components/segmented，站点版本 6.6.0）— block/options/size(medium)/orientation API 表

### Tertiary (LOW confidence)

- 无（无未经交叉验证的外部检索结论进入正文；A1-A5 假设已在 Assumptions Log 声明）

## Metadata

**Confidence breakdown:**
- 现状盘点：HIGH — 全部文件逐一 Read + grep 交叉
- v16 参考实现：HIGH — 行号级定位（user/index.tsx + index.css）
- 迁移与风险：HIGH — 种子/注册点/缓存 TTL 均代码实测；迁移内缓存失效的具体方法名为 A4 低风险假设
- 组件 API：HIGH — Segmented 官方文档验证 + 项目内双先例

**Research date:** 2026-08-19
**Valid until:** 2026-09-18（项目内部代码为真相源，稳定性高；antd 版本窗口内无破坏性预期）
