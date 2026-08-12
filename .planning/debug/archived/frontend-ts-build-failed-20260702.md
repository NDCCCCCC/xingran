---
status: resolved
trigger: "diagnose 11 TypeScript errors in build-linux.bat frontend tsc stage; produce minimal patches"
created: 2026-07-02
updated: 2026-07-02
---

## Current Focus
hypothesis: (CONFIRMED) 11 errors fall into 4 atomic groups — type re-export redundancy, missing component prop, optional-field narrowing loss in setState callbacks, type predicate + unsafe post<T>.
test: applied patches A/B/C/D; ran `npx tsc -b` after each.
expecting: 11 -> 0
next_action: report delivered to user; awaiting approval before commit.

## Symptoms
expected: tsc compiles cleanly; no error TS* output.
actual: 11 errors from `npx tsc -b`, all in uncommitted-modified scope (`WorkstationDeviceTable/index.tsx`, `types/operations.ts`) plus callers introduced alongside.
reproduction: `cd xingran-react-frontend && npx tsc -b`
started: 2026-07-02 (introduced by Phase R5 confidence column + same-day edits)

## Eliminated
- hypothesis: "maybe `editingInfoPoint.deviceId` should be made required on InfoPoint"
  evidence: changing InfoPoint shape would ripple to every callsite; minimal fix is local narrowing.
  timestamp: 2026-07-02
- hypothesis: "use `as any` to silence TS2677 on user/index.tsx"
  evidence: user explicitly prefers `r: unknown` type predicate to keep type safety.
  timestamp: 2026-07-02

## Evidence
- 2026-07-02: `DeviceSource = "ad" | "asset" | "manual" | "physical"` and `DeviceSourceLabels`/`DEVICE_SOURCE_LABELS` (incl. `physical: "物理链路"`) are already defined in `src/types/operations.ts:296,329,337`. The component-local `WorkstationDeviceTable/types.ts` redeclared the const without `physical` → TS2741.
- 2026-07-02: `MACEventsTimelineProps` in `src/components/network/MACEventsTimeline.tsx:28` lacked `deviceId`, but `MACHistoryPage.tsx:449,552` passed `deviceId={record.deviceId}` (added by R5).
- 2026-07-02: `InfoPoint.deviceId`/`portId`/`portName` (`src/types/operations.ts:186-189`) and `Department.leader?`/`leaderName?`/`leaderUsername?` (`src/types/system.ts:114-116`) are all optional. `useState` setter callback `prev => ...` does not carry the outer narrowing into the closure, so `editingInfoPoint.deviceId` reverts to `string | undefined`. Hoisting into a local `const x: string` (with explicit annotation) re-narrows and satisfies the state type.
- 2026-07-02: `post<T>(url, data)` returns `Promise<BaseResponse<T>>`; `BaseResponse.data` is `T | undefined`. `assetApi.ts:445` returned `res.data` directly → TS2322.
- 2026-07-02: `user/index.tsx:393` type predicate parameter type was `string | { id: string; roleName?: string; roleKey?: string }`; predicate narrowing to object form must be assignable to the parameter's input type. The boolean guard `typeof r === "object" && ...` excludes string but does not assert it — TS rejects predicate (TS2677) and falls back to `r: string` for the chain, breaking the `.map(r => r.id)` chain (TS2339 ×3). Casting input to `unknown[]` and re-asserting `id` after predicate fixes both.
- 2026-07-02: Clean rebuild (`rm tsconfig.tsbuildinfo* && npx tsc -b`) returns EXIT=0.

## Resolution
root_cause: Four independent TS regression groups introduced alongside R5 physical-link confidence column: (1) redundant local re-definition of `DEVICE_SOURCE_LABELS` lost `physical`; (2) `MACEventsTimelineProps` not extended for `deviceId` prop passed by R5 caller; (3) optional-field narrowing lost inside `useState` setter closures for info-points and dept; (4) `assetApi.refreshReconciliation` returned `T | undefined`, and `user/index.tsx` type predicate parameter type was incompatible with the predicate narrowing.
fix: see Patch A/B/C/D in report body.
verification: `npx tsc -b` reports 0 errors; clean rebuild (after removing tsconfig.tsbuildinfo*) also reports 0 errors with EXIT=0.
files_changed:
- xingran-react-frontend/src/components/operations/WorkstationDeviceTable/types.ts
- xingran-react-frontend/src/components/network/MACEventsTimeline.tsx
- xingran-react-frontend/src/pages/operations/info-points/index.tsx
- xingran-react-frontend/src/pages/system/dept/index.tsx
- xingran-react-frontend/src/pages/system/user/index.tsx
- xingran-react-frontend/src/lib/assetApi.ts

---

## Patch Report (verbatim)

### Root cause table (4 groups)

| # | Errors | Root cause |
|---|--------|------------|
| A | TS2741 (1) | Local `DEVICE_SOURCE_LABELS` in `WorkstationDeviceTable/types.ts:18` lacks `physical`; canonical version with all 4 keys exists at `src/types/operations.ts:337` |
| B | TS2322 ×2 (2) | `MACEventsTimelineProps` missing `deviceId`; R5 caller at `MACHistoryPage.tsx:449/552` passes it |
| C | TS2345 ×3 (3) | Optional-field narrowing lost across `useState` callback closure: `editingInfoPoint.deviceId`/`portId`/`portName` (info-points) and `record.leaderName` (dept) |
| D | TS2677 + TS2339 ×3 + TS2322 (5) | (D1) `user/index.tsx` type predicate parameter type union is too wide; `(r): r is {…}` cannot narrow back. (D2) `assetApi.refreshReconciliation` returns `res.data` (`T \| undefined`) when signature promises `T` |

Total: 11 errors → 4 atomic patches → 0 errors

### Patch A — `WorkstationDeviceTable/types.ts`
Replace local 3-key definition with re-export of canonical 4-key map.

```
- import type { WorkstationDevice, DeviceSource, DeviceFormData } from "@/types";
-
- // 导出类型，方便外部使用
- export type { WorkstationDevice, DeviceSource, DeviceFormData };
-
- // 组件 Props 接口
- export interface WorkstationDeviceTableProps {
-   workstationId: string;
-   expandable?: boolean;
-   onDeviceChange?: () => void;
- }
-
- // 设备来源枚举和标签
- export const DEVICE_SOURCE_LABELS: Record<DeviceSource, string> = {
-   ad: "域控",
-   asset: "资产",
-   manual: "手动",
- } as const;
+ import type { WorkstationDevice, DeviceSource, DeviceFormData } from "@/types";
+ // 复用 @/types/operations 中已包含全部 DeviceSource 键(含 physical)的标签表,
+ // 避免本地重复定义遗漏导致 Record<DeviceSource, string> 不完整。
+ export { DEVICE_SOURCE_LABELS } from "@/types/operations";
+
+ // 导出类型，方便外部使用
+ export type { WorkstationDevice, DeviceSource, DeviceFormData };
+
+ // 组件 Props 接口
+ export interface WorkstationDeviceTableProps {
+   workstationId: string;
+   expandable?: boolean;
+   onDeviceChange?: () => void;
+ }
```
Fixes: TS2741 ×1. Verification: 11 → 10.

### Patch B — `components/network/MACEventsTimeline.tsx`
Add optional `deviceId?: string` to props interface.

```
  export interface MACEventsTimelineProps {
    /** MAC 地址(必填) */
    mac: string;
    /** 时间范围起点(ISO 字符串) */
    startTime: string;
    /** 时间范围终点(ISO 字符串) */
    endTime: string;
    /** 单页条数(默认 100) */
    pageSize?: number;
+   /** 设备 ID(可选, R5 物理链路场景传入, 当前未在组件内使用, 仅透传给未来扩展) */
+   deviceId?: string;
  }
```
Fixes: TS2322 ×2. Verification: 10 → 8.

### Patch C — narrowing via local const (atomic patch spans 2 files)
`pages/operations/info-points/index.tsx` (lines 311-330) and `pages/system/dept/index.tsx` (lines 90-107).

For info-points, hoist `editingInfoPoint.deviceId`, `editingInfoPoint.portId`, `editingInfoPoint.portName` into local `const x: string` inside the outer `if` guard so they retain narrowing inside the setter closures.

```
-       if (editingInfoPoint && editingInfoPoint.deviceId) {
-         setSelectedDeviceId(editingInfoPoint.deviceId);
-         // 设备兜底注入(2026-06-30,同 openModal):当前设备可能不在 pageSize:50 列表
-         const devName = editingInfoPoint.deviceName || "";
-         setNetworkDevices(prev =>
-           prev.find(d => d.id === editingInfoPoint.deviceId)
-             ? prev
-             : [...prev, { id: editingInfoPoint.deviceId, deviceName: devName || "未命名设备", ipAddress: "" }]
-         );
-         await loadDevicePorts(editingInfoPoint.deviceId);
-         // 端口兜底注入:loadDevicePorts(pageSize:50)可能没覆盖当前 portId,用 portName 注入
-         if (editingInfoPoint.portId && editingInfoPoint.portName) {
-           setDevicePorts(prev =>
-             prev.find(p => p.id === editingInfoPoint.portId)
-               ? prev
-               : [...prev, { id: editingInfoPoint.portId, interfaceName: editingInfoPoint.portName }]
-           );
-         }
+       if (editingInfoPoint && editingInfoPoint.deviceId) {
+         // 收窄到局部 const,避免 setState 闭包内 editingInfoPoint.deviceId 被推回 string|undefined
+         const deviceId: string = editingInfoPoint.deviceId;
+         setSelectedDeviceId(deviceId);
+         // 设备兜底注入(2026-06-30,同 openModal):当前设备可能不在 pageSize:50 列表
+         const devName = editingInfoPoint.deviceName || "";
+         setNetworkDevices(prev =>
+           prev.find(d => d.id === deviceId)
+             ? prev
+             : [...prev, { id: deviceId, deviceName: devName || "未命名设备", ipAddress: "" }]
+         );
+         await loadDevicePorts(deviceId);
+         // 端口兜底注入:loadDevicePorts(pageSize:50)可能没覆盖当前 portId,用 portName 注入
+         if (editingInfoPoint.portId && editingInfoPoint.portName) {
+           const portId: string = editingInfoPoint.portId;
+           const portName: string = editingInfoPoint.portName;
+           setDevicePorts(prev =>
+             prev.find(p => p.id === portId)
+               ? prev
+               : [...prev, { id: portId, interfaceName: portName }]
+           );
+         }
```

For dept, hoist `record.leader` and `record.leaderName` similarly.

```
-     if (record.leader) {
-       setDeptUsers(prev =>
-         prev.find(u => u.id === record.leader)
-           ? prev
-           : [...prev, {
-               id: record.leader,
-               username: record.leaderUsername || record.leaderName || "未命名用户",
-               nickname: record.leaderName,
-             }]
-       );
-     }
+     if (record.leader) {
+       // 收窄到局部 const,避免 setState 闭包内 record.leaderName 仍为 string|undefined
+       const leaderId: string = record.leader;
+       const nickname: string = record.leaderName || "";
+       setDeptUsers(prev =>
+         prev.find(u => u.id === leaderId)
+           ? prev
+           : [...prev, {
+               id: leaderId,
+               username: record.leaderUsername || record.leaderName || "未命名用户",
+               nickname,
+             }]
+       );
+     }
```
Fixes: TS2345 ×3. Verification: 8 → 5.

### Patch D1 — `pages/system/user/index.tsx`
Cast predicate input to `unknown[]` and re-assert `id` after the predicate.

```
-       const asObjects = editingUser.roles
-         .filter((r): r is { id: string; roleName?: string; roleKey?: string } => typeof r === "object" && r !== null && "id" in r)
-         .map(r => ({ id: r.id, roleName: r.roleName, roleKey: r.roleKey }));
+       const asObjects = (editingUser.roles as unknown[])
+         .filter((r): r is { id: string; roleName?: string; roleKey?: string } => typeof r === "object" && r !== null && "id" in r)
+         .map(r => ({ id: (r as { id: string }).id, roleName: r.roleName, roleKey: r.roleKey }));
```
Fixes: TS2677 ×1 + TS2339 ×3. Verification: 5 → 1.

### Patch D2 — `lib/assetApi.ts`
Provide empty-object fallback for `res.data`.

```
-   }> => {
-     const res = await post<{
-       inserted: number;
-       skipped: number;
-       skippedSilence: number;
-       skippedThrottle: number;
-     }>("/asset/reconciliation/refresh", {});
-     return res.data;
-   },
+   }> => {
+     const res = await post<{
+       inserted: number;
+       skipped: number;
+       skippedSilence: number;
+       skippedThrottle: number;
+     }>("/asset/reconciliation/refresh", {});
+     // res.data 实际是 T | undefined(BaseResponse 契约),这里业务契约保证非空,
+     // 兜底 {} 以满足返回类型。
+     return res.data ?? { inserted: 0, skipped: 0, skippedSilence: 0, skippedThrottle: 0 };
+   },
```
Fixes: TS2322 ×1. Verification: 1 → 0.

### Verification commands

```bash
# Per-patch count
cd D:/code/ClaudeCode/xingran-go-backend/xingran-react-frontend && npx tsc -b 2>&1 | grep -cE "error TS"
# Expected progression: 11 -> 10 -> 8 -> 5 -> 0

# Final clean rebuild (no incremental cache)
cd D:/code/ClaudeCode/xingran-go-backend/xingran-react-frontend && rm -f tsconfig.tsbuildinfo tsconfig.app.tsbuildinfo tsconfig.node.tsbuildinfo && npx tsc -b
# Expected: no output, exit 0
```

### Commit suggestion (do not auto-commit, awaiting user)
- `git add` the 6 patched files (no `git add .` / `-A`)
- Suggested order: 1 (A), 2 (B), 3 (C — info-points+dept as one commit), 4 (D1), 5 (D2)
  - Or single commit covering all 4 atomic patches if user prefers
- Per CLAUDE.md: ask before `git commit`; show diff first
