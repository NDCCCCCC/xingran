---
slug: slidercaptcha-antd-message-warn
status: resolved
trigger: SliderCaptcha.tsx:83 Warning: [antd: message] Static function can not consume context like dynamic theme. Please use 'App' component instead.
created: 2026-06-17T07:03:50.000Z
updated: 2026-06-17T07:03:50.000Z
---

# Debug Session: slidercaptcha-antd-message-warn

## Context

- 这是 **同根因复现**：前一个已 resolved 会话 `antd-static-message-warn.md` 已诊断 W1（antd v5 静态 `message.xxx()` 无法消费 ConfigProvider 动态主题），并在根组件 `AntdThemeBridge.tsx` 加了 antd `<App component={false}>` 包裹（commit `bba86a1`）。
- 用户在登录滑块验证码组件 `SliderCaptcha.tsx` 又触发同一警告（line 83），说明该组件仍在用静态 `message.xxx()`。
- 由于根组件 `<App>` 已就位，SliderCaptcha 处于其子树内，修复仅需**该文件本地改动**，与 VM list 同款修复模式。

## Symptoms

### W1 · antd message static API（SliderCaptcha.tsx:83）

```
SliderCaptcha.tsx:83 Warning: [antd: message] Static function can not consume context
like dynamic theme. Please use 'App' component instead.
```

- 文件 `xingran-react-frontend/src/components/captcha/SliderCaptcha.tsx` 第 3 行静态导入 `message`，第 42/83/87/93 行调用 `message.error/success(...)`。
- 静态 API 无法拿到 ConfigProvider 的动态 theme context（暗色模式 / 主题色切换时降级）。

## Current Focus

- **hypothesis**:
  - 根组件 `<App>` 已就位（前一会话成果），SliderCaptcha 在其子树内。把静态 `message` import 换成 `App`，并在组件顶部 `const { message } = App.useApp();`，4 处 `message.xxx()` 调用点同作用域自动解析为 context-aware 实例，警告消除。
- **test**:
  - 核实 `AntdThemeBridge.tsx` 当前已含 `<App component={false}>`（前提条件）
  - 应用本地修复
  - `npm run type-check` + `npm run build` 验证
- **expecting**:
  - 警告消失；`message.xxx()` 在动态主题下行为正确；无类型/构建回归
- **next_action**:
  - 已完成修复与验证

## Evidence

- timestamp: 2026-06-17T07:03:50.000Z
  checked: `src/design-system/components/AntdThemeBridge.tsx` (line 26, 82-85)
  found: |
    - import `{ App, ConfigProvider, theme as antdTheme }` from "antd"
    - `<ConfigProvider locale={zhCN} theme={antdThemeConfig}><App component={false}>{children}</App></ConfigProvider>`
    - 已提交：commit `bba86a1 feat(design-system): wrap children in antd <App> for context-aware API`
  implication: 根组件 `<App>` 包裹就位，SliderCaptcha 在其子树内，本地 `App.useApp()` 可用，前提满足

- timestamp: 2026-06-17T07:03:50.000Z
  checked: SliderCaptcha.tsx 修复 diff
  found: |
    - `import { Button, message }` → `import { Button, App }`
    - 组件顶部新增 `const { message } = App.useApp();`
    - 第 42/83/87/93 行 `message.xxx()` 调用点未改（同作用域解析）
  implication: 最小本地修复，与 VM list 同款

## Resolution

- root_cause: |
  SliderCaptcha.tsx 使用 antd 静态 `message.xxx()`（named import），无法消费 ConfigProvider 的动态 theme context。
  根因与前一会话 `antd-static-message-warn.md` 完全相同（W1），只是触发文件不同。
  根组件 `<App>` 包裹此前已由前一会话落地，故本次仅需本地迁移调用方。
- fix: |
  两处本地改动（与 VM list 同款修复模式）：
  1. SliderCaptcha.tsx line 3：`import { Button, message } from "antd"` → `import { Button, App } from "antd"`。
  2. 组件顶部新增 `const { message } = App.useApp();`，使第 42/83/87/93 行的 `message.error/success(...)` 透明地解析为 context-aware 实例（调用点无需改动）。
- verification: |
  - `npm run type-check`（tsc --noEmit）：PASS，无输出无错误。
  - `npm run build`：PASS，`✓ built in 36.11s`。
  - 工作区仅 SliderCaptcha.tsx 一处改动，未触及其它文件。
- files_changed:
  - xingran-react-frontend/src/components/captcha/SliderCaptcha.tsx

## Note (系统范围)

- 仍有约 110 个文件使用静态 `message.xxx()`（前一会话统计 111 文件，本次修复 1 个）。这些文件在用户报告时再逐个按同款模式迁移，**遵守 CLAUDE.md scope rule：不未经请求跨模块改动**。
- 根治方案（一次性全项目迁移到 `App.useApp()`）需用户显式授权大 scope，见前一会话 Scope Decision。
