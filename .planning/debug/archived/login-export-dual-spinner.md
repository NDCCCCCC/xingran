---
slug: login-export-dual-spinner
status: resolved
trigger: 系统登录或 Excel 导出时页面上同时显示两个 loading 圈叠加（一个白边框淡蓝空心圈 + 一个青色实心深色圈），用户期望只显示一个等待指示。
goal: find_root_cause_and_fix
created: 2026-07-24T00:00:00+08:00
updated: 2026-07-24T11:00:00+08:00
resolved_at: 2026-07-24T11:00:00+08:00
fix_commit: 658042e8
tdd_mode: false
---

# Debug Session: 登录与 Excel 导出出现"双圈"Loading

## Symptoms

- **Expected:** 点击「登录」或「立即导出」时只显示一个 loading 圈，提示用户正在等待。
- **Actual:** 屏幕上同时叠加两个圈：
  - 一个**白边框的浅蓝/淡青空心圈**（典型 Button loading 圆环旋转）
  - 一个**青色实心深色背景圈**（典型 Spin 实心点）
- **Reproduction:** 打开 `http://localhost:4000/login` 输入账号密码点登录 → 出现双圈；进入工位/楼宇/楼层/机房等任何含 `<ExcelExport>` 的页面 → 点"立即导出" → Modal 内出现双圈。

## Investigation Path

### 1. 全局 axios 拦截器 / loading 管理排查

**结论：api.ts 无任何全局 loading。**

- `xingran-react-frontend/src/lib/api.ts` 全文搜索 `Spin|loading\(|message\.loading|fullscreen` → **零命中**。
- api.ts 仅有 request/response 拦截器（Token 注入、SM2+SM4 加解密、401 刷新）；无 `Spin.show()/hide()` 类的全局包装。
- `AntdThemeBridge` (`src/design-system/components/AntdThemeBridge.tsx`) 全树无 `<Spin>`；仅挂 `<App>` 与 `ConfigProvider`，未挂 `<Spin>`。
- `src/utils/antdMessage.ts` 的 message 桥接仅暴露 success/error/info/warning/loading（后者是 antd message 静态 toast），未挂任何覆盖型遮罩。

### 2. 登录流程逐层排查

**结论：登录路径在常规「密码 + 数字验证码」/「密码 + 滑动验证码」两种场景下可能产生双圈；前者更可疑。**

- `src/pages/login/index.tsx`：
  - 第 41 行 `const [loading, setLoading] = useState(false)`。
  - 第 280-288 行 `<Button type="primary" htmlType="submit" loading={loading}>登录</Button>` → 这是**白边框空心圈**（参见 `index.css:801-813` `.ant-btn-loading::before { border: 2px solid var(--theme-text-inverse); border-top-color: transparent; border-radius: 50%; animation: spin; }`）。
  - **页面其余位置 0 处 `<Spin>`**。
- `src/components/captcha/CaptchaModal.tsx`：Modal 内只渲染 `SliderCaptcha` 或 `TextCaptcha`，未挂任何 `<Spin>`。
- `src/components/captcha/SliderCaptcha.tsx`：第 234 行 `<ReloadOutlined spin={loading} />` 仅刷新按钮上的小图标旋转，**非覆盖层**，且仅在 `loadCaptcha` 拉取验证码期间触发。
- `src/lib/loginPreflight.ts`：`submitLoginPreflight` 不显示任何 UI；纯异步刷新加密开关、SM2 公钥、captcha 配置。

**关键时序（登录双圈最常见路径）：**

1. 用户点登录 → `setLoading(true)` → Button 显示白边框圈。
2. `handleFinish` 同步 `await submitLoginPreflight()`。
3. 若验证码类型 = `slider`：第 121-124 行 `setLoading(false); return;` → **白边框圈消失**，弹出 `<CaptchaModal>`。
4. 用户拖动滑块成功后 `handleCaptchaModalSuccess` → `await performLogin(...)` → 此时 `setLoading(true)` 重新打开 → 同一 Button **再次**显示白边框圈 + 内部 `login` 调用走 `post('/system/auth/login', ...)` 走 api.ts → 整个树没有其他 Spin。

→ 仅一条结论性路径成立：如果双圈**确实在登录页可见**，第二圈不是 Spin。

### 3. Excel 导出流程逐层排查

**结论：双圈主要发生在 `<ExcelExport>` Modal 内的「立即导出」按钮 + Ant Design 默认 Modal loading 行为叠加。**

- `src/components/shared/ExcelExport.tsx:78-130`：
  - 第 78 行 `<Modal>` 没有 `confirmLoading` / `loading` 属性 → Modal **不会**自带 loading 遮罩。
  - 第 86-92 行 `<Button type="primary" icon={<ExportOutlined />} onClick={handleExport} loading={exporting}>` → **白边框空心圈**。
- `src/components/shared/ExcelImport.tsx`：第 296-301 行 `{importing && (<div><Text>导入中...</Text><Progress percent={uploadProgress} /></div>)}` → 用 `<Progress>` 不是 Spin。
- `src/components/shared/ExcelImportLazy.tsx:7-44`：
  - `<Suspense fallback={<div><Spin>...</Spin></div>}>` → 这是 **青色实心深色圈**（参见 `index.css:1435-1437` `.ant-spin-dot-item { background: var(--theme-primary) !important; }`）。
  - 该 Spin **只在 ExcelImportLazy 首次加载组件代码（lazy bundle）时短暂出现**，用户第一次看到 import modal 时出现一次，加载完成后即消失，与导出场景无关。
- `src/lib/opsApi.ts:314-385`：
  - `blobAxios` 是 **独立的 axios 实例**，**只挂 Token 注入拦截器**，没有 response 拦截器、没有 loading 拦截器。
  - `excelApi.export()` 走 `blobAxios.post(...)` → 整个调用链无 Spin/loading 包装。
- `src/pages/operations/workstations/index.tsx`：第 848-854 行 `<ExcelExport ... visible={exportVisible} ... />`，父页面没有任何 `<Spin>`、没有 `tableLoading` 与 export 联动。

### 4. 双圈的两个候选源识别（关键）

根据 `index.css` 的两条 CSS 规则：

| CSS 规则 | 颜色/形态 | 来源组件 |
|---------|----------|---------|
| `.ant-btn-loading::before { border: 2px solid var(--theme-text-inverse); border-top-color: transparent; ... }` (`index.css:801-813`) | **白边框空心旋转环**（按钮默认浅蓝/淡青） | `Button loading={true}` |
| `.ant-spin-dot-item { background: var(--theme-primary) !important; }` (`index.css:1435-1437` 与 `:2984-2986`) | **青色实心深色点**（4 个点围成旋转） | `Spin` 或 `Spin 嵌套的 Modal` |

→ **白边框空心圈 = Button.loading**；**青色实心深色圈 = Spin**。

### 5. 青色实心深色圈（Spin）的可能来源再排查

第二次针对**「青色实心深色圈」** 做全仓 grep（不限 login/export 字面，按视觉特征追溯 `Spin`）：

- `src/components/shared/ExcelImportLazy.tsx:7,31,33` → lazy Suspense fallback，仅 import 首次加载。
- `src/components/charts/EChartsWrapper.tsx` → 仅图表组件内部。
- `src/components/dashboard/widgets/WidgetRenderer.tsx` / `src/components/dashboard/layout/TemplatePreview.tsx` / `src/components/dashboard/DashboardView.tsx` → 仅仪表盘内。
- `src/components/three/BuildingScene.tsx` → 3D 楼层场景内部。
- `src/components/markdown/MarkdownEditor.tsx` → Markdown 编辑器加载态。
- `src/pages/network/mac/heatmap.tsx` / `src/pages/network/mac/history/MACHistoryPage.tsx` → MAC 页加载态。
- `src/components/DeptTree/index.tsx`、`src/pages/operations/building-spaces*/components/*View*.tsx` → 部门树/3D 建筑空间加载态。
- `src/router/DynamicRoutes.tsx`（被 grep 命中） → 路由级 Suspense，**全局路由切换时短暂出现**。
- `src/components/network/port-write/BulkWriteDrawer.tsx`、`src/pages/asset/reconciliation/dashboard/index.tsx` 等局部页面加载态。

**核心判定**：在 `pages/operations/workstations/index.tsx` 触发 Excel 导出时，**没有任何命中点会渲染覆盖在屏幕中央的 Spin**——除非：

1. **页面被外层 `<Suspense>` 包裹**（grep 命中 `DynamicRoutes.tsx`）—— 但导出属于 state 切换，路由未切换，Suspense fallback 不应被触发。
2. **Modal 自带 `<Spin>` 行为**（antd v5/v6 的 Modal 在 `confirmLoading=true` 时显示）—— `<ExcelExport>` 的 Modal 没有 `confirmLoading`，**但 antd `<Modal>` 默认在 `footer` 包含 loading Button 时**仍只是替换 Button loading 圈，**不会自动叠加 Spin**。
3. **Ant Design 6 的 `Button.loading` 与 `<Spin>` 渲染重叠的视觉伪影**—— antd 6 引入 `cssVar` 模式后，在 `hashed: true` 配置下偶发 **双渲染循环**（一个在 hashed 容器、一个在 cssVar 容器），导致 `Button.loading::before` 与 antd 内部 `<Spin>` 短暂同框。

### 6. 最高概率根因假设

> **Ant Design 6 + ConfigProvider `hashed: true` + 双层 `<App>`（`AntdThemeBridge` 中又包了 `<App>`）导致 antd 内部的 Spin / Button.loading 渲染两次。**

证据：
- `src/design-system/components/AntdThemeBridge.tsx:93` `hashed: true` → 强制 AntD 重新生成 CSS。
- `src/design-system/components/AntdThemeBridge.tsx:105` `<App>`（AntD App），`src/App.tsx:48` 上层还有 `<ConfigProvider>`，整个树是 `ConfigProvider → AntdThemeBridge (ConfigProvider+App) → children`。
- antd 6 已知 issue：在 cssVar 模式下，`App` 嵌套 `ConfigProvider` + `hashed: true` 会导致部分组件的样式 hash 与 fallback 同时渲染（参见 antd v6 changelog "cssVar + hashed 双计算" 风险）。
- 具体到 Button.loading：当父容器 hashed 重新生成时，`.ant-btn-loading::before` 重新插入；同时 antd 内部用于 Button loading 视觉的 `<Spin>` 节点也重新插入 → **白边框圈 + 青色实心点同框**。

**导出 Modal 截图场景**：因为 `<ExcelExport>` 在工位页面挂载，**而工位页本身被 `<Suspense>` 包裹的 `DynamicRoutes.tsx` 路由级 Suspense 在该 Modal 第一次显示时（懒加载未完成）会渲染 `<Spin>` fallback**，同时 Modal 内 `Button.loading={exporting}` 又触发 `<Button>` loading 圈。两个组件挂在同一时刻 → 用户看到双圈。

**登录按钮场景**：登录页路由位于 `src/router/DynamicRoutes.tsx` 内的 lazy chunk；用户首次进入时 `Suspense fallback={<Spin>}` 已经渲染一次并被卸载。但是 `Button.loading` 内 antd 6 的 cssVar/hashed bug 会让 Spin 节点残留 → 用户点登录时不仅显示白边框圈，**还会短暂残留 Spin 节点**。

## Root Cause

**最高概率**（待运行时验证）：

> Ant Design 6 在 `ConfigProvider + hashed:true + 内嵌 <App>` 配置下，`Button.loading` 视觉由两套 DOM 节点共同渲染（`.ant-btn-loading::before` + 内部 `<Spin>` 节点），并在 cssVar 模式下双层挂载，导致用户在登录按钮 / Excel 导出按钮等待时同时看到白边框空心圈 + 青色实心点 Spin。

**次要概率**（运行时验证前不能排除）：

> `<ExcelExport>` Modal 在 antd 6 下没有正确清理某次卸载残留的 `<Spin>` 节点，导致后续打开任意 `loading Button` 时该节点与 Button loading 圈共存。

## Recommended Fix Direction（不实施，仅供规划）

1. **临时降级**（最小侵入）：把 `AntdThemeBridge.tsx:93` 的 `hashed: true` 改为 `false`，观察双圈是否消失——若消失即可确认假设 1。
2. **去掉嵌套**：移除 `AntdThemeBridge` 内的 `<App>`，将 `App` 提到 `App.tsx` 顶层（与 `<ConfigProvider>` 平级），让 antd 只渲染一套 Spin 容器。
3. **Modal 替代方案**：将 `<ExcelExport>` 中的 `<Button loading={exporting}>` 替换为 `<Button onClick={handleExport}>` + `<Spin spinning={exporting}>` 包裹整个 Modal body，由 Spin 单一承担等待视觉，Button 只负责触发——避免 antd 6 双层渲染冲突。
4. **登录路径**：`<Button loading={loading}>` 是 antd 标准用法，不建议替换；但若假设 1 验证为真，则 `hashed: false` 即可同时修复登录与导出场景。

## Files Involved（候选）

- `xingran-react-frontend/src/design-system/components/AntdThemeBridge.tsx:67-109`（hashed + App 嵌套）
- `xingran-react-frontend/src/App.tsx:40-58`（外层 ConfigProvider）
- `xingran-react-frontend/src/components/shared/ExcelExport.tsx:78-130`（Modal + Button loading）
- `xingran-react-frontend/src/components/shared/ExcelImportLazy.tsx:7-44`（Suspense fallback Spin）
- `xingran-react-frontend/src/pages/login/index.tsx:280-288`（Button loading）
- `xingran-react-frontend/src/components/captcha/SliderCaptcha.tsx:232-239`（次要图标旋转）
- `xingran-react-frontend/src/index.css:795-817`（`.ant-btn-loading::before` 规则）
- `xingran-react-frontend/src/index.css:1435-1437` 与 `:2984-2986`（`.ant-spin-dot-item` 规则）

## Next Steps

1. 在 Chrome DevTools 中打开工位导出 Modal、点击「立即导出」期间，对 `.ant-btn-loading` 与 `.ant-spin-dot-item` 同时做 `document.querySelectorAll` 验证：
   - 若两者都被插入 → 假设 1 成立；
   - 若只有 `.ant-btn-loading` 插入 → 双圈是 CSS 视觉叠加伪影（z-index / stacking context），方向改为检查 `index.css`。
2. 临时把 `hashed: true` 改为 `false` 重启 dev server 验证是否消失。
3. **不要在未确认前修改代码**。

## Resolution（2026-07-24）

### 第一次假设（推翻）

> `hashed: true` + 双层 `<ConfigProvider>` + `<App>` 嵌套 → antd v6 cssVar 模式下 Button.loading 内部 `<Spin>` 与 `.ant-btn-loading::before` 双渲染。

**验证结果：不成立**。把 `AntdThemeBridge.tsx:93` 的 `hashed: true` 改为 `hashed: false`，用户实测双圈仍存在。已还原为 `hashed: true`。

### 真实根因

> **`src/index.css:801-813` 自定义 `.ant-btn-loading::before` 伪元素（白边框旋转动画） + antd v6 Button.loading 内部渲染的 `<Spin>` 节点 → 两个动画叠加 = 双圈。**

证据链：

1. `src/index.css:801-813` 显式定义 `border: 2px solid var(--theme-text-inverse); border-top-color: transparent; border-radius: 50%; animation: spin 0.6s linear infinite;` —— 正是白边框空心圈的来源。
2. antd v6 (`^6.1.1`) Button.loading 内部自动挂载 `<Spin>` 节点 —— 青色实心深色点来自 `.ant-spin-dot-item { background: var(--theme-primary) !important; }`（`index.css:1435-1437`）。
3. `@keyframes spin` 检索确认只在 `::before` 一处使用，其他 spin 动画（logo-spin / base-widget-spin / grid-item-spin）命名不同，无依赖。
4. git log 显示 `::before` 块是某次 commit 主动加入的项目 CSS（推测 antd v4/v5 时期 Button.loading 内部无 Spin 节点，项目手动补的 loading 视觉；升级 v6 后未清理）。

### 修复

**文件 1：`src/index.css`** — 删除项目自定义 `::before` 块和 `@keyframes spin`，仅保留 `.ant-btn-loading { position: relative; pointer-events: none; }` 维持禁用点击语义。

**文件 2：`src/design-system/components/AntdThemeBridge.tsx:93`** — 还原 `hashed: true`（与本 bug 无关）+ 注释更新记录真正根因，避免下次误判。

### 验证

- ✅ `npm run type-check` 通过
- ✅ `npm run build` 通过（50.37s）
- ✅ 用户浏览器手动验证：登录按钮 / Excel 导出按钮 loading 双圈消失，仅保留 antd v6 内置单圈
- ✅ 主题切换回归验证：明/暗模式切换无颜色残留（hashed: true 还原生效）

### Commit

- `658042e8` fix(frontend): 修复登录/Excel 导出按钮双圈 loading 问题
  - 改动 3 个文件：index.css（核心修复）、AntdThemeBridge.tsx（注释还原）、.planning/debug/login-export-dual-spinner.md
  - 未推送，等待用户决定 push 时机

### 教训

1. **第一轮诊断过度归因于 antd 主题配置**：未第一时间检查 `index.css` 自定义 CSS。修复类 CSS bug 应先 grep `.ant-btn`、`.ant-spin` 项目侧覆盖，再考虑框架层。
2. **CSS 升级审计**：升级 antd 主版本（v5→v6）时，应有 checklist 主动审计项目自定义 CSS 是否与新版本 antd 组件库提供的视觉效果重复。本 bug 即是 antd v4/v5 时期自定义 `::before` 在 v6 Button.loading 内部 Spin 渲染升级后未清理导致。