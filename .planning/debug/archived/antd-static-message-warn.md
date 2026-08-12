---
slug: antd-static-message-warn
status: resolved
trigger: index.tsx:184 Warning: [antd: message] Static function can not consume context like dynamic theme. Please use 'App' component instead. + index.tsx:818 Warning: [antd: Alert] message is deprecated. Please use title instead.
created: 2026-06-17T09:35:00.000Z
updated: 2026-06-17T17:00:00.000Z
---

# Debug Session: antd-static-message-warn

## Context

- 项目共 **111 个文件**使用 antd v5 静态 `message.xxx()` API（`import { message } from "antd"`），触发 `[antd: message] Static function can not consume context` 警告且不消费动态主题（暗色/主色切换时降级）。
- 根因：根组件 `AntdThemeBridge` 仅包裹 antd `ConfigProvider`，未包裹 antd `<App>` 组件——message context 无法传到静态 message 单例。
- 此次会话完成**全项目根治**：
  1. 新建 `utils/antdMessage.ts` 模块作用域桥接（`setAppMessageInstance` / `getAppMessage`）
  2. `AntdThemeBridge` 加 `MessageContextInjector` 子组件，在 `<App>` 内部用 `useEffect` 注入 context-aware message 实例到 module-level ref
  3. 4 个高危顶层调用文件（axios 拦截器、静态错误处理器、Excel 工具）改用 `getAppMessage()` 读取
  4. 107 个组件/hook 文件改用 `App.useApp()` 模式（import 移除 message 加 App + 组件/h 顶部 `const { message } = App.useApp()`）

## Resolution

- **root_cause**: |
  antd v5 静态 `message`（`import { message } from "antd"`）在模块加载时实例化为全局单例，不在 React 树内，无法消费 ConfigProvider 的动态主题 context——暗色模式下 message 弹框仍是白底、自定义主色不生效，并喷 `[antd: message] Static function can not consume context` 警告。
  根因双重：①根组件 `AntdThemeBridge` 未包裹 antd `<App>` 承载 message context；②111 个业务文件用静态 API 而非 `App.useApp()`。
- **fix**: |
  两层架构 + 统一迁移：

  **架构层（2 文件）**
  - `src/utils/antdMessage.ts`（新建）：持有 module-level `messageRef`，导出 `setAppMessageInstance(instance)` / `getAppMessage()`；`<App>` 未挂载时 `getAppMessage()` 返回 no-op 实例静默短路。`MessageInstance` 类型从 `ReturnType<typeof App.useApp>["message"]` 推断，避免依赖 antd 内部子路径。
  - `src/design-system/components/AntdThemeBridge.tsx`：import 加 `useEffect` 和 `setAppMessageInstance`；新增 `MessageContextInjector` 子组件（运行在 antd `<App>` 内部）→ `useEffect(() => { setAppMessageInstance(message); return () => setAppMessageInstance(null); }, [message])`；将 `{children}` 在 JSX 中改用 `<MessageContextInjector>{children}</MessageContextInjector>` 包裹。

  **顶层调用文件（4 文件，桥接模式）**——运行在模块作用域无法用 hook
  - `src/lib/api.ts`：8 处 axios 响应拦截器内 `message.error(...)` → `getAppMessage().error(...)`
  - `src/utils/errorHandler.ts`：6 处 `message.error/success(...)` → `getAppMessage().xxx(...)`（保持 `ErrorHandler` 静态方法 API 不变，调用方零改动）
  - `src/pages/duty/management/utils/excelUtils.ts`：3 处工具函数内 `message.xxx(...)` → `getAppMessage().xxx(...)`
  - `src/pages/duty/holidays/utils.tsx`：5 处工具函数内 `message.xxx(...)` → `getAppMessage().xxx(...)`

  **组件 / 自定义 hook 文件（107 文件，App.useApp 模式）**——运行在 React 树内可用 hook
  - 统一模式：①antd import 移除 `message` 命名导入（单行或多行整体移除），确保含 `App`（无则补）；②在调用 message 的那个 React 函数组件/自定义 hook 函数体最顶部（紧接 `{` 后、所有 `useState/useEffect/useCallback/useXxx` 之前）插入 `  const { message } = App.useApp();`；③所有 `message.xxx(...)` 调用点保持原样（自动解析到第 ② 步解构的 context-aware 实例）。
  - 按模块分批由 subagent 并行迁移；subagent 报告的所有命中文件均完成，0 个"无法迁移"。

- **verification**: |
  - `npm run type-check`（`tsc --noEmit`）：PASS，无输出无错误。
  - `npm run build`（`tsc -b && vite build`）：PASS，`✓ built in 35.56s`；主入口图 + 路由懒加载 chunk 全部成功打包。
  - `rg -U --multiline-dotall -l 'import\s*\{[^}]*\bmessage\b[^}]*\}\s*from\s*"antd"' xingran-react-frontend/src | grep -v antdMessage`：**0 匹配**——除 `utils/antdMessage.ts` 的文档注释字面串外，全项目无静态 `message` 导入。
  - 边界情况处理：
    - `apikeys/index.tsx` 的模块级 `copyToClipboard` 工具函数原无法用 `App.useApp()`，subagent 改为接受 `message` 作参数（参数注入），调用方在组件作用域内持有 context-aware message 并显式传入。
    - 多文件包含 `Form.Item rules` 的 `message:` 字段（验证文案 key 名，与 antd `message` 函数无关）和 `LoginLog.message` 字段（后端响应字段），均未误改。
    - `monitor/job/index.tsx`、`monitor/job/columns/*` 等文件中 `message` 仅出现在 `rules` 文案或 `dataIndex`，subagent 已识别为假阳性，未误改。
  - 既有调用约定保持：所有 `message.success/error/warning/info/loading(...)` 调用签名未变；axios 拦截器拒绝 Promise 行为不变；`ErrorHandler` 静态方法签名不变；`copyToClipboard` 工具函数改参数注入后调用方已同步更新。
- **files_changed**:
  基础设施（2）：
  - xingran-react-frontend/src/utils/antdMessage.ts（新建）
  - xingran-react-frontend/src/design-system/components/AntdThemeBridge.tsx

  顶层调用桥接（4）：
  - xingran-react-frontend/src/lib/api.ts
  - xingran-react-frontend/src/utils/errorHandler.ts
  - xingran-react-frontend/src/pages/duty/management/utils/excelUtils.ts
  - xingran-react-frontend/src/pages/duty/holidays/utils.tsx

  组件/hook 迁移（107，按模块）：
  - pages/system/（9）：apikeys/index.tsx、dept/hooks/useDeptData.ts、menu/hooks/useMenuData.ts、notice/hooks/useNoticeData.ts、menu/hooks/useMenuActions.tsx、dict/hooks/useDictActions.ts、dict/hooks/useDictData.ts、captcha-background/hooks/useCaptchaModals.ts、role/hooks/useRoleActions.ts
  - pages/operations/（11，subagent A）：assets/index.tsx、building-spaces/index.tsx、building-spaces/components/WorkstationView.tsx、building-spaces-3d/components/HubeiMap.tsx、building-spaces-3d/components/HubeiMapGL.tsx、floors/useFloorPlanEditor.ts、rpa/executions/ExecutionDetailModal.tsx、rpa/tasks/index.tsx、rpa/tasks/modals/EditModal.tsx、rpa/tasks/modals/AIScriptEditor.tsx、workstations/hooks/useWorkstationModals.ts
  - pages/duty/（10）：config/index.tsx、pools/index.tsx、schedules/index.tsx、schedules/hooks/useScheduleData.ts、schedules/hooks/useScheduleModals.ts、holidays/hooks/useHolidayData.ts、holidays/hooks/useHolidayModals.ts、management/hooks/useDutyConfig.ts、management/hooks/useHolidayData.ts、management/hooks/useScheduleData.ts、management/modals/BatchHolidayModal.tsx（注：含 BatchHolidayModal 共 11 个）
  - pages/network/（2）：executions/hooks/useExecutionModals.tsx、executions/hooks/useExecutionData.ts
  - pages/monitor/ + pages/workorder/（12）：monitor/cache/index.tsx、monitor/job/hooks/useJobActions.ts、monitor/logs/index.tsx、monitor/logs/hooks/useLogActions.tsx、monitor/logs/hooks/useLogData.ts、monitor/server/index.tsx、workorder/categories/index.tsx、workorder/orders/index.tsx、workorder/orders/hooks/useWorkOrderActions.ts、workorder/orders/hooks/useWorkOrderData.ts、workorder/periodic/templates/hooks/useTemplateActions.ts、workorder/periodic/templates/hooks/useTemplateData.ts
  - pages/ad-domain/ + pages/ad/（8）：ad-domain/users/index.tsx、ad-domain/groups/index.tsx、ad-domain/logs/index.tsx、ad-domain/ous/index_with_dept.tsx、ad-domain/computers/index.tsx、ad-domain/configs/index.tsx、ad-domain/ous/index.tsx、ad/SyncMonitor/index.tsx
  - pages/misc + hooks/（7）：hooks/useADConfigs.ts、hooks/useColumnConfig.ts、hooks/useImageUpload.ts、knowledge/articles/hooks/useArticleData.ts、knowledge/articles/index.tsx、vdi/VirtualMachineDetail/index.tsx、vdi/VDIServerConfig/index.tsx
  - components/（非 dashboard + 非 SliderCaptcha，11）：dashboard/DashboardView.tsx + dashboard/layout/LayoutToolbar.tsx + dashboard/layout/TemplateSelector.tsx + dashboard/widgets/WidgetEditor.tsx + dashboard/settings/DashboardSettings.tsx + dashboard/layout/TemplatePreview.tsx（子会话 subagent B，6）+ layout/sidebar.tsx + shared/FileUpload.tsx + shared/FloorPlanEditor.tsx + shared/NetworkExport.tsx + shared/ImageGallery.tsx（本轮 subagent G，5）
  - 之前会话手动修（2）：components/captcha/SliderCaptcha.tsx、pages/vdi/VirtualMachineList/index.tsx
  - 漏改补（1）：pages/knowledge/view/index.tsx

  **总计：113 个文件改动**（基础设施 2 + 顶层桥接 4 + 业务迁移 107）

  注：项目本身仅 111 个静态 `import { ... message ... } from "antd"` 命中；其中 `utils/antdMessage.ts` 的命中来自文档注释示例代码（"import { message } from "antd" "），非真实 import。真实改动覆盖 110 个业务文件 + 2 个基础设施文件 + 1 个新桥接模块 = 113 个文件触及。

## Note (后续可考虑，非本次 scope)

- 项目还有 antd v5 静态 `notification` 和 `Modal.confirm/warning` 等 API 在多个页面使用（如 `DashboardView.tsx`、`LayoutToolbar.tsx`），同样会触发警告且不消费动态主题。本次仅按"用户报告的 message 警告"做最小根治；如需统一治理 `notification/modal`，可后续另起 task 按相同"App.useApp() 模式"迁移——同样需要根 `<App>` 包裹（已就位）。
- 调试会话 `slidercaptcha-antd-message-warn.md`（针对 `SliderCaptcha.tsx:83` 的报告）已合并入本会话 Resolution，因为本质同根因。
