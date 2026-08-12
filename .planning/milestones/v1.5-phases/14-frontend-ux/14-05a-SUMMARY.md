---
phase: 14-frontend-ux
plan: 05a
subsystem: shared-components
tags: [frontend, shared, error-handling, empty-state, phase-14]
dependency_graph:
  requires:
    - 14-01
    - 14-03
  provides:
    - EmptyStateWithAction (D-18)
    - ErrorAlertWithRetry (D-20)
    - components/shared barrel 注册 (EmptyStateWithAction / ErrorAlertWithRetry)
  affects:
    - 14-04 (Excel 导出按钮,可选择性使用 ErrorAlertWithRetry)
    - 14-05b (history.tsx / trajectory.tsx 页面三态改造,直接复用)
tech_stack:
  added:
    - React 19 FC + AntD 6 (Empty / Alert / Button / Space)
    - react-router-dom v7 Link (内部跳转)
    - zustand authStore (1007 token 失效自动 logout)
  patterns:
    - 错误对象多态解析(支持 Error 实例 / ApiErrorShape / axios response 结构)
    - useEffect 触发的副作用登出(1007 → logout + location.href)
    - barrel export 追加(不覆盖既有 export)
key_files:
  created:
    - xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx
    - xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx
  modified:
    - xingran-react-frontend/src/components/shared/index.ts
decisions:
  - EmptyStateWithAction 使用 react-router-dom Link(而非 useNavigate)以便嵌套在父级路由上下文,保持 SSR/未来迁移弹性
  - ErrorAlertWithRetry 用 useEffect 触发 logout + location.href 跳转,而非在渲染期间副作用调用(避免 React 警告)
  - 错误对象 code 提取兼容三层结构:error.code / error.response.data.code / error.status,覆盖 axios 拦截器 / 业务异常 / HTTP 状态码三种来源
  - 1006/500 仅展示 showIcon Alert(不附加 description),通用错误码附加 `错误码:${code}` 辅助排查
  - 1007 case 不渲染 Button 重新加载(因为用户已登出跳转,重试无意义)
  - barrel export 顺序:Phase 14 共享组件追加在 BatchExportModal 之后,新增章节注释便于未来按 phase 归类
metrics:
  duration: ~5min
  completed: 2026-06-14
  files_created: 2
  files_modified: 1
  task_count: 1
---

# Phase 14 Plan 05a: 三态共享组件抽取 Summary

将 Phase 14 移动端 + 空/加载/错误三态打磨(原 14-05)中的"空数据"与"错误态"两个组件抽离为可复用 shared 组件,供 14-04(导出按钮)与 14-05b(列表/轨迹页面改造)直接 import 使用。

## What Was Built

### 1. EmptyStateWithAction (`xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx`)

**职责**: 列表/查询结果为 0 时,在页面内联展示空状态 + 可选的"前往某页面"引导按钮。

**Props 接口**:
- `description: string` — 空状态描述文案
- `actionLabel?: string` — 按钮文案(可选)
- `actionPath?: string` — 按钮跳转路径(可选)
- `icon?: ReactNode` — 自定义图标(默认 AntD Empty 简单占位图)

**实现细节**:
- AntD `Empty` 组件 + `Space` + `Button` 组合
- 跳转用 `react-router-dom` 的 `Link` 组件(避免 hook 调用限制)
- 当 `actionLabel` 与 `actionPath` 同时存在时才渲染按钮,否则只展示空描述

**对齐 D-18**: "空数据 = 引导去设备采集",带"前往设备管理"链接。

### 2. ErrorAlertWithRetry (`xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx`)

**职责**: API/查询失败时内联展示错误 Alert,按业务错误码分级文案,支持重试。

**Props 接口**:
- `error: Error | ApiErrorShape | null | undefined` — 错误对象
- `onRetry?: () => void` — 重试回调(可选)

**错误码映射**:
| 错误码 | 文案 | 副作用 |
|--------|------|--------|
| `1006` | 该设备不存在或已被删除 | 无 |
| `1007` | 登录已失效,正在跳转... | `authStore.logout()` + `window.location.href = '/login'` |
| `500` | 服务暂不可用,请稍后重试 | 无 |
| 其他 | 查询失败:${error.message} | 无,Alert description 附 `错误码:${code}` |

**实现细节**:
- `extractErrorCode()` 支持三种来源:`error.code` / `error.response.data.code` / `error.status`
- `extractErrorMessage()` 兼容 `Error.message` / `response.data.message` / `response.data.msg`
- 1007 处理放在 `useEffect` 中,避免渲染期间触发副作用导致 React 警告
- 1007 跳转前不渲染"重新加载"按钮(用户已登出,重试无意义)

**对齐 D-20**: "错误状态 = 内联 Alert + 重试",按错误码分级文案。

### 3. Barrel Export 增量 (`xingran-react-frontend/src/components/shared/index.ts`)

**追加(不删除既有)**:
```typescript
// Phase 14 - 三态(空/加载/错误)共享组件
export { default as EmptyStateWithAction } from './EmptyStateWithAction';
export type { EmptyStateWithActionProps } from './EmptyStateWithAction';

export { default as ErrorAlertWithRetry } from './ErrorAlertWithRetry';
export type {
  ErrorAlertWithRetryProps,
  ApiErrorShape,
} from './ErrorAlertWithRetry';
```

既有 export (`ActionButtons` / `ExcelImport` / `ExcelExport` / `FileUpload` / `ImageGallery` / `GlobalSearch` / `DepartmentTreeSelect*` / `FloorPlanEditor*` / `BatchExportModal`) 全部保留。

## Verification

### 自动化检查
```bash
$ npx tsc --noEmit -p . 2>&1
EXIT: 0
```

### 接受标准检查
- EmptyStateWithAction.tsx `export default` 命中 1 行(`EmptyStateWithAction`)
- ErrorAlertWithRetry.tsx 错误码 1006 / 1007 / 500 各有 1 个 case 分支,共 3 个数字字符串
- barrel `index.ts` 包含 `EmptyStateWithAction` 和 `ErrorAlertWithRetry` 各 2 个名字(共 4 行 export)
- 既有 export(`ActionButtons` / `ExcelImport` / `ExcelExport` / `DepartmentTreeSelect*`) 全部保留
- TypeScript 编译退出码 0

## Deviations from Plan

None — plan executed exactly as written.

## Downstream Consumers

| Plan | 文件 | 用法 |
|------|------|------|
| 14-04 | Excel 导出按钮 | 可选用 `ErrorAlertWithRetry` 包裹 blob 解析失败的错误(目前 `api.ts` 拦截器已处理) |
| 14-05b | history.tsx / trajectory.tsx | 列表 `total === 0` → `EmptyStateWithAction` + `actionPath='/network/devices'`;React Query `isError` → `ErrorAlertWithRetry` + `onRetry={refetch}` |

## Commits

| Hash | Description |
|------|-------------|
| `ffb43ac` | feat(14-05a): extract EmptyStateWithAction and ErrorAlertWithRetry shared components |

## Self-Check

### Created files exist
- `D:/CODE/ClaudeCode/xingran-go-backend/xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx` — FOUND
- `D:/CODE/ClaudeCode/xingran-go-backend/xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx` — FOUND

### Commits exist
- `ffb43ac` — FOUND in `git log --oneline`

### Acceptance criteria
- EmptyStateWithAction 默认导出 + props 接口符合 D-18: PASS
- ErrorAlertWithRetry 错误码映射(1006/1007/500)符合 D-20: PASS
- barrel export 包含两个新组件: PASS
- 既有的 shared 组件 export 不被破坏: PASS
- Wave 2 并行就位,14-04 / 14-05b 可依赖: PASS

## Self-Check: PASSED
