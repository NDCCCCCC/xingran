---
phase: 53-w4-frontend-drawer-progress-dialog-api-wrappers
plan: 01
subsystem: frontend
tags: [frontend, react, typescript, api-wrapper, network, port-write, foundation]
requires:
  - "Phase 52 W3 6 kebab HTTP endpoints (port_write_router.go:42-47)"
  - "Phase 51 W2 PortResult/BatchResult/BatchWriteRequest Go struct 契约"
provides:
  - "types/network.ts: PortWriteAction / PortResult / BatchWriteRequest / BatchResult TypeScript 接口"
  - "networkApi.ts: 6 个 port-write wrapper 函数 (writeShutdown/writeUndoShutdown/writeDescription/writeDot1xEnable/writeDot1xDisable/batchWritePorts)"
  - "components/network/port-write/constants.ts: PRESET_REASONS / ACTION_TITLE / REASON_MIN / REASON_MAX / DESCRIPTION_MAX 共享常量"
affects:
  - "53-02 UI plan (PortWriteModal / BulkWriteDrawer / ports/index.tsx 改造) — 直接消费本 plan 三文件"
tech-stack:
  added: []
  patterns:
    - "post<T>(url, body) + result.data! wrapper 模式 (镜像 queryMACHistory)"
    - "type-only import 避免运行时循环依赖"
    - "字面量联合类型编译期锁定 action/status 三态 (LANDMINE #3 / T-53-04)"
key-files:
  created:
    - "xingran-react-frontend/src/components/network/port-write/constants.ts"
  modified:
    - "xingran-react-frontend/src/types/network.ts"
    - "xingran-react-frontend/src/lib/api/networkApi.ts"
decisions:
  - "类型放 types/network.ts (与 DevicePortStatus 同处),不放 types/index.ts barrel (LANDMINE #1)"
  - "wrapper 用 post() 包装风格,不用 opsApi factory (D-08: opsApi 是 operations 模块专用)"
  - "wrapper 不 try/catch 不调 message.error (LANDMINE #5: post() 拦截器已弹 Toast,防 double-Toast)"
  - "PRESET_REASONS 预设项 value 扩展为 6 字符 (故障排查处理 等),与 REASON_MIN=5 自洽,不留矛盾给 53-02 校验逻辑"
  - "新增 PortWriteAction 字面量联合类型,作为 action 字段的单一真实源 (BatchWriteRequest.action 复用)"
metrics:
  duration: "8 min"
  completed: "2026-07-07T05:31:39Z"
  tasks: 3
  files: 3
  commits: 3
requirements: [UI-06, BATCH-05]
---

# Phase 53 Plan 01: 类型 + API Wrapper + 共享常量地基 Summary

把 Phase 52 W3 落地的 6 个 kebab HTTP 写端点 + Phase 51 service 层 Go struct 契约翻译成前端可直接消费的 TypeScript 类型 + 6 个 post wrapper 函数 + 5 个 UI 共享常量,为 53-02 (PortWriteModal / BulkWriteDrawer / ports 页改造) 提供零依赖前置地基。

## What Was Built

### 文件改动清单 (3 文件 / 1 新建 / 2 修改)

| 文件 | 类型 | 改动 | Commit |
|------|------|------|--------|
| `xingran-react-frontend/src/types/network.ts` | MODIFY | 末尾追加 4 类型 (PortWriteAction + PortResult + BatchWriteRequest + BatchResult) | `14d433f3` |
| `xingran-react-frontend/src/lib/api/networkApi.ts` | MODIFY | 顶部 type-only import + 末尾追加 6 wrapper + default export 加 6 引用 | `ef79a45b` |
| `xingran-react-frontend/src/components/network/port-write/constants.ts` | CREATE | 新建 port-write 目录 + constants.ts 导出 5 const | `a968f5b2` |

### Task 1: types/network.ts 追加 4 类型

| 类型 | 来源 | 字段 |
|------|------|------|
| `PortWriteAction` | 新增 (action 字段单一真实源) | `"shutdown" \| "undo_shutdown" \| "description" \| "dot1x_enable" \| "dot1x_disable"` |
| `PortResult` | port_write_service.go:34-42 | `portId, action: PortWriteAction, status: "succeeded" \| "failed" \| "skipped", noOp, currentState?, error?, commandSent?` |
| `BatchWriteRequest` | port_write_service.go:45-50 | `deviceId, action: PortWriteAction, portIds: string[], description?` |
| `BatchResult` | batch_orchestrator.go:15-19 | `succeeded: PortResult[], failed: PortResult[], skipped: PortResult[]` |

**关键:** status 用字面量联合类型 (非 string),编译期锁定三态防 typo 引入第 4 态导致 BulkWriteDrawer 结果分区漏判 (T-53-04 mitigate)。

### Task 2: networkApi.ts 扩展 6 wrapper

| Wrapper | URL (全 kebab) | Request Body | Response |
|---------|----------------|--------------|----------|
| `writeShutdown` | `/network/ports/write/shutdown` | `{portId, reason}` | `Promise<PortResult>` |
| `writeUndoShutdown` | `/network/ports/write/undo-shutdown` | `{portId, reason}` | `Promise<PortResult>` |
| `writeDescription` | `/network/ports/write/description` | `{portId, description, reason?}` | `Promise<PortResult>` |
| `writeDot1xEnable` | `/network/ports/write/dot1x-enable` | `{portId, reason}` | `Promise<PortResult>` |
| `writeDot1xDisable` | `/network/ports/write/dot1x-disable` | `{portId, reason}` | `Promise<PortResult>` |
| `batchWritePorts` | `/network/ports/write/batch` | `BatchWriteRequest` | `Promise<BatchResult>` |

**约束满足:**
- 6 wrapper 函数体 0 try/catch (透传 Promise.reject)
- 6 wrapper 函数体 0 message.error / getAppMessage 调用 (LANDMINE #5 防 double-Toast, post() 拦截器已弹)
- 顶部 `import type { PortResult, BatchResult, BatchWriteRequest } from "@/types/network"` 是 type-only import (避免运行时循环依赖)
- default export 对象追加 6 引用

### Task 3: constants.ts 新建 5 const

| Const | 值 | 来源 |
|-------|----|------|
| `PRESET_REASONS` | `[{label,value}] as const`,4 预设项 (故障排查处理 / 安全合规要求 / 业务变更需要 / 临时测试验证, 每项 6 字符) + 1 sentinel `__custom__` | D-02 |
| `ACTION_TITLE` | `Record<PortWriteAction 联合, string>`,5 action 中文标题 (关闭端口/取消关闭/修改描述/启用 802.1X/停用 802.1X) | D-01 |
| `REASON_MIN` | `5` | D-02 下限 |
| `REASON_MAX` | `200` | D-02 上限 |
| `DESCRIPTION_MAX` | `80` | D-03 跨厂商保守上限 |

**自洽性:** PRESET_REASONS 每项 value 6 字符 ≥ REASON_MIN (5),不留矛盾给 53-02 校验逻辑 (planner 在 Task 3 action 中明确指出的 4 字符陷阱已规避)。

## 为 53-02 提供的 Import 接口清单

```typescript
// 53-02 UI 组件可直接 import:

// 类型 (来自 types barrel 自动透传)
import type {
  PortWriteAction,
  PortResult,
  BatchWriteRequest,
  BatchResult,
} from "@/types/network";
// 或从 barrel:
// import type { PortResult } from "@/types";

// Wrapper 函数 (named import)
import {
  writeShutdown,
  writeUndoShutdown,
  writeDescription,
  writeDot1xEnable,
  writeDot1xDisable,
  batchWritePorts,
} from "@/lib/api/networkApi";
// 或 default import:
// import networkApi from "@/lib/api/networkApi";
// networkApi.writeShutdown(portId, reason)

// 共享常量
import {
  PRESET_REASONS,
  ACTION_TITLE,
  REASON_MIN,
  REASON_MAX,
  DESCRIPTION_MAX,
} from "@/components/network/port-write/constants";
```

## Acceptance Criteria Results

### Task 1 — types/network.ts

| AC | Result |
|----|--------|
| `grep -c "export interface PortResult"` = 1 | ✅ 1 |
| `grep -c "export interface BatchResult"` = 1 | ✅ 1 |
| `grep -c "export interface BatchWriteRequest"` = 1 | ✅ 1 |
| `status` 字段为字面量联合 (非 string) | ✅ `status: "succeeded" \| "failed" \| "skipped"` |
| `npm run type-check` 退出 0 | ✅ EXIT=0 |
| `types/index.ts` 未修改 (barrel 自动透传) | ✅ 未触碰 |

### Task 2 — networkApi.ts

| AC | Result |
|----|--------|
| 6 个 `export const` wrapper | ✅ 6 |
| 6 个 kebab URL `post<>(...)` 调用 | ✅ 6 (实际 post 调用) |
| `from "@/types/network"` type-only import | ✅ 1 |
| wrapper 函数体无 try/catch | ✅ 0 |
| wrapper 函数体无 message.error/getAppMessage | ✅ 0 |
| default export 含 6 新引用 | ✅ 6 |
| `npm run type-check` 退出 0 | ✅ EXIT=0 |

### Task 3 — constants.ts

| AC | Result |
|----|--------|
| 文件存在 | ✅ EXISTS |
| `export const PRESET_REASONS` = 1 | ✅ 1 |
| `export const ACTION_TITLE` = 1 | ✅ 1 |
| `REASON_MIN/REASON_MAX/DESCRIPTION_MAX` = 3 | ✅ 3 |
| `__custom__` sentinel ≥ 1 | ✅ 2 (注释 + value) |
| 5 个 action 键覆盖 | ✅ 5 |
| REASON_MIN=5 / REASON_MAX=200 / DESCRIPTION_MAX=80 | ✅ 精确 |
| PRESET_REASONS 每项 value ≥ 5 字符 | ✅ 6 字符 (Node codePoint 验证) |
| `npm run type-check` 退出 0 | ✅ EXIT=0 |

## Wave 1 Build Verification

| 检查项 | 结果 |
|--------|------|
| `npm run type-check` | ✅ EXIT=0 |
| `npm run build` | ✅ EXIT=0 (built in 1m 31s) |
| vendor-react gzip ≤ 826 kB | ✅ 774.96 kB (Phase 48 baseline 776 + 50 容差) |
| Bundle delta | -1.04 kB vs Phase 48 baseline (零新 npm 依赖) |

## Landmine 规避确认

| Landmine | 规避措施 | 状态 |
|----------|----------|------|
| #1: types 必须放 `types/network.ts` | PortResult/BatchWriteRequest/BatchResult 追加到 DevicePortStatus 同文件 (line 117 之后) | ✅ |
| #5: wrapper 不 try/catch 不弹 Toast | 6 wrapper 函数体仅 `post<T> + result.data!`,无 try/catch / message.error / getAppMessage | ✅ |

## Deviations from Plan

无 — plan 严格按文执行。

唯一 planner 在 Task 3 action 中预先标记的小决策 (PRESET_REASONS value 扩展为 5+ 字符以满足 REASON_MIN=5 自洽) 已按 planner 的修正建议落地: 预设项 value 改为 6 字符 (故障排查处理 / 安全合规要求 / 业务变更需要 / 临时测试验证),既满足 D-02 字面要求也保证 53-02 校验逻辑不被卡。

## Known Stubs

无 — 本 plan 是 foundation plan,无 UI 渲染路径,无数据源接入,纯类型/wrapper/常量定义。

## Threat Flags

无新增安全相关表面。

本 plan 三文件均为纯前端 TS 代码,不引入新网络端点 (wrapper 仅代理 Phase 52 已有的 6 端点),不修改 auth/permission 路径,不触碰数据库 schema。T-53-01 (URL 字面量) / T-53-04 (status 三态锁定) 已通过 acceptance_criteria mitigate。

## Self-Check: PASSED

- FOUND: xingran-react-frontend/src/types/network.ts
- FOUND: xingran-react-frontend/src/lib/api/networkApi.ts
- FOUND: xingran-react-frontend/src/components/network/port-write/constants.ts
- FOUND: 14d433f3 (Task 1 commit)
- FOUND: ef79a45b (Task 2 commit)
- FOUND: a968f5b2 (Task 3 commit)
