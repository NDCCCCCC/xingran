---
phase: 14-frontend-ux
verified: 2026-06-26T16:00:00Z
status: gaps_found
score: 12/19 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 12/19
  gaps_closed:
    - "TrajectoryNode contract: Phase 13 (2026-06-26) fixed TrajectoryNode.MACAddress (json:mac) + queryMACVendor (camelCase vendorName) + MACTrajectoryChart tooltip node-object access (CR-03 + CR-04 + W4-vendor partial mitigation). See 13-VERIFICATION.md status:passed score:18/18 + 13-{07,08,09,10}-SUMMARY.md."
  gaps_remaining:
    - "exportMACHistory returns BaseResponse instead of Blob (14-01 EXPORT-01, CR-01 of 14-REVIEW)"
    - "/network/history/list endpoint does not exist (14-02 + 14-03 blocker)"
    - "ErrorAlertWithRetry 1007 logout fires twice without useRef guard (CR-03 of 14-REVIEW)"
    - "EmptyStateWithAction actionPath type narrowing (CR-04 of 14-REVIEW)"
    - "MAC copy success/error feedback missing (WR-07)"
    - "Excel export end-to-end non-functional (no backend format=xlsx branch + frontend never receives real Blob)"
  regressions: []
gaps:
  - truth: "工具栏出现两个互斥按钮 '导出当前查询' / '导出全量',点击后浏览器下载 .xlsx 文件"
    status: failed
    reason: "exportMACHistory returns the intercepted BaseResponse envelope ({code:0,...}) instead of a Blob. URL.createObjectURL receives a JSON object, so the browser downloads a .xlsx file with JSON contents. No real Excel binary ever reaches the user."
    artifacts:
      - path: "xingran-react-frontend/src/lib/api/networkApi.ts"
        issue: "Lines 130-141: `await api.default.get(...)` returns BaseResponse (unwrapped by interceptor at api.ts:269-391). `return response as Blob` is a type lie — response is `{code:0, message:'success', data:..., ...}`. CR-01 from 14-REVIEW confirmed."
      - path: "xingran-react-frontend/src/pages/network/mac/history.tsx"
        issue: "Line 385: `URL.createObjectURL(blob)` is called on the BaseResponse object. The error path (`blob.size < 1024` → JSON parse) is never reached because the value is not a Blob at all."
    missing:
      - "Bypass the response interceptor for blob requests (use rawAxios or expose a `rawApi` without response interceptors)"
      - "Or change backend to return `{code:0, data:<base64 xlsx>, filename:...}` and convert on frontend"
      - "Or `transformResponse: [(d) => d]` + read `.data` as Blob via raw axios call"
      - "Implement the planned `blob.size < 1024 → FileReader.readAsText → JSON.parse` error path"
  - truth: "查询 MAC 历史记录(列表页主数据源) — POST /network/history/list 返回 list/total/current/pageSize"
    status: failed
    reason: "Backend exposes /network/history/port, /network/history/device, /network/history/trajectory, /network/history/stats, /network/history/vendor only (mac_history_router.go:19-23). No /network/history/list endpoint exists. queryMACHistory will receive 404 at runtime, so UI-01 main list returns no data."
    artifacts:
      - path: "xingran-react-frontend/src/lib/api/networkApi.ts"
        issue: "Line 95: `await post<MACHistoryPageResult>('/network/history/list', params)` — calls a non-existent endpoint."
      - path: "internal/api/v1/network/mac_history_router.go"
        issue: "Only POST /history/port, /history/device, /history/trajectory, /history/stats, /history/vendor are registered."
    missing:
      - "Backend route `POST /network/history/list` that aggregates the device/port queries into a paginated result matching MACHistoryQueryParams"
      - "Or frontend must call `/history/port` or `/history/device` with appropriate params — needs service-layer decision"
  - truth: "ErrorAlertWithRetry 错误态使用组件,1007 token 失效自动 logout + 跳登录(无重复触发)"
    status: failed
    reason: "useEffect at ErrorAlertWithRetry.tsx:80-87 fires on every render where code===1007, with no ref guard. CR-03 confirmed. React StrictMode will invoke it twice; even without StrictMode a single render where the component survives across a parent state change will fire logout() twice, racing the cleanup of sessionStorage tokens."
    artifacts:
      - path: "xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx"
        issue: "Lines 80-87: `useEffect(() => { if (code === 1007) logout().finally(...) }, [code, logout])` — no useRef guard, no cancelled flag."
    missing:
      - "useRef guard so logout fires at most once per error instance"
      - "Cancelled flag in effect cleanup"
      - "(Better) move 1007 handling into a single global response interceptor, not a leaf component"
  - truth: "EmptyStateWithAction 空数据态,actionLabel + actionPath 同时存在才渲染 Link"
    status: failed
    reason: "Boolean(actionLabel && actionPath) passes for actionPath = 0 (numeric), or actionPath = '' would still be falsy but actionPath as object would also pass. The `to={actionPath as string}` cast erases the type system. With React Router v7 an empty-string `to` silently routes to current path; a numeric `to` throws at render. CR-04 confirmed."
    artifacts:
      - path: "xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx"
        issue: "Line 32: visibility check too lax. Line 42: `to={actionPath as string}` discards type narrowing."
    missing:
      - "Narrow actionPath with `typeof actionPath === 'string' && actionPath.length > 0` before rendering"
  - truth: "端点严格按 D-01 锁定为 POST /network/history/list(可在 Phase 14 后端复用)"
    status: failed
    reason: "The 14-01 PLAN explicitly states endpoint is locked to `/network/history/list` and the executor must not assume alternatives. However the backend (Phase 12/13) only registered /history/port, /history/device, etc. — the locked endpoint was never created. UI-01 and UI-02 both depend on it."
    artifacts:
      - path: ".planning/phases/14-frontend-ux/14-01-PLAN.md"
        issue: "Lines 86, 107, 116, 124-126: PLAN explicitly requires endpoint `/network/history/list` but the backend does not expose it."
    missing:
      - "Either add a backend handler `POST /network/history/list` returning MACHistoryPageResult, or amend the plan and frontend to use `/history/port` / `/history/device`"
  - truth: "复制 MAC 后给用户反馈(WR-07)"
    status: failed
    reason: "history.tsx:409-411 uses `void navigator.clipboard?.writeText(mac)` — discards the promise silently. No success or error message to user. Marked WR-07 in 14-REVIEW."
    artifacts:
      - path: "xingran-react-frontend/src/pages/network/mac/history.tsx"
        issue: "Line 410: `void navigator.clipboard?.writeText(mac)` swallows errors."
    missing:
      - "Wrap in try/catch with AntD message.success/error feedback"
  - truth: "Excel 导出 — 后端真实接口返回 application/vnd.openxmlformats-officedocument.spreadsheetml.sheet(UI-02 验收)"
    status: failed
    reason: "Same as the first gap. UI-02 requires a real Excel download. The frontend never receives a real .xlsx because (a) the endpoint doesn't exist, (b) the Blob is actually a JSON envelope. End-to-end the export feature is non-functional."
    artifacts:
      - path: "xingran-react-frontend/src/lib/api/networkApi.ts"
        issue: "exportMACHistory returns non-Blob; no backend `format=xlsx` branch was added either."
      - path: "internal/api/v1/network/mac_history_handler.go"
        issue: "No format query parameter handling; no xlsx generation path. D-14 (backend `format=xlsx` branch) was not implemented."
    missing:
      - "Backend branch: when `format=xlsx` is supplied, return `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` with streaming Excel"
      - "Frontend: actual Blob delivery via rawAxios or post with `responseType: 'blob'` bypassing envelope unwrap"
---

# Phase 14: Frontend UX — Verification Report

**Phase Goal:** 基于 Phase 12 (数据模型与采集集成) 与 Phase 13 (查询层与轨迹) 的后端能力,补齐 MAC 地址历史数据管理的完整前端 UX — 包括查询列表页、轨迹可视化页 UX 增强、Excel 导出、菜单与权限注册、与网络设备模块的联动入口,以及移动端响应式适配与状态兜底。

**Status:** gaps_found
**Verified:** 2026-06-15T02:00:00Z

## Goal Achievement

### Observable Truths

| #   | Truth                                                       | Status     | Evidence                                                                                                                |
| --- | ----------------------------------------------------------- | ---------- | ----------------------------------------------------------------------------------------------------------------------- |
| 1   | 用户访问 /network/mac/history 看到 8 列 + 6 时间预设 + 默认 7d            | ✗ FAILED   | UI 渲染就绪 (history.tsx),但后端 `/network/history/list` 不存在,数据加载失败                 |
| 2   | 快捷预设按钮 5 + 自定义 与 RangePicker 互斥同步                                  | ✓ VERIFIED | PRESETS 数组 history.tsx:61-68; handlePresetClick / handleCustomRangeChange                                        |
| 3   | 操作列 "查看事件" 展开行 → MACEventsTimeline;点击事件跳 /network/mac/trajectory    | ✓ VERIFIED | expandedRowRender history.tsx:456-463; MACEventsTimeline.tsx:160-167                                          |
| 4   | 列表行 AntD Table virtual + placeholderData: keepPreviousData              | ✓ VERIFIED | virtual 属性 history.tsx:430; useTableQuery.ts:66                                                            |
| 5   | URL 参数注入(?deviceId=&portName=&startTime=&endTime=&mac=)              | ✓ VERIFIED | useEffect history.tsx:134-154                                                                                  |
| 6   | /network/mac/trajectory 时间预设 + URL 注入 + 自动查询                          | ✓ VERIFIED | trajectory.tsx:96-127; PRESETS + URLSearchParams                                                                  |
| 7   | dataZoom 默认 66/100 范围                                                      | ✓ VERIFIED | MACTrajectoryChart.tsx:53-54, 169-176                                                                          |
| 8   | 停留时长热力 tooltip 4 档(<1h/<24h/<7d/>=7d)                               | ✓ VERIFIED | MACTrajectoryChart.tsx:29-39 + 103-106                                                                          |
| 9   | 右侧 Drawer 嵌入 MACEventsTimeline                                          | ✓ VERIFIED | trajectory.tsx:276-298                                                                                           |
| 10  | 工具栏 "导出当前查询" / "导出全量" 按钮 + 权限可见性                                  | ✗ FAILED   | 按钮渲染正确 history.tsx:707-725,但点击下载的不是 .xlsx 文件(见 CR-01)                   |
| 11  | api.get blob + format=xlsx + createObjectURL 下载模式                       | ✗ FAILED   | networkApi.ts:135 调用走 api.default.get,但 response interceptor 返回 BaseResponse envelope 而非 Blob |
| 12  | blob.size < 1024 → JSON.parse 错误反序列化                                  | ✗ FAILED   | plan 14-04 计划要求,但 14-04 + 14-05b 实施均未实现 size < 1024 分支                                  |
| 13  | exportMACHistory 函数返回 Blob                                                | ✗ FAILED   | networkApi.ts:130-141 `return response as Blob` — 类型谎言,运行时是 JSON 对象                           |
| 14  | EmptyStateWithAction 空数据 + "前往设备管理" 跳转                                | ⚠ PARTIAL  | history.tsx:467-471 + 492-496 调用 EmptyStateWithAction;组件 CR-04 (actionPath=0 时仍渲染 Link)        |
| 15  | ErrorAlertWithRetry 错误码 1006/1007/500 分级文案                              | ⚠ PARTIAL  | ErrorAlertWithRetry.tsx:91-103 实现文案,但 CR-03 (1007 useEffect 重复触发)                             |
| 16  | 移动端 Grid.useBreakpoint xs 自动切换 List 卡片                                 | ✓ VERIFIED | history.tsx:107 isMobile;renderCardList branch                                                              |
| 17  | 网络设备列表/详情 Modal "查看 MAC 历史" 按钮跳 /network/mac/history?deviceId= | ✓ VERIFIED | devices/index.tsx:180-184 + 942-952                                                                         |
| 18  | 菜单 SQL 注册 + 路由说明文档                                                    | ✓ VERIFIED | 14-menu-registration.sql (3 INSERTs + 验证 SELECT); 14-03-ROUTE-SETUP.md (8 checklist items) |
| 19  | 工具栏 exportScope='current'|'all' 契约保留                                    | ✓ VERIFIED | history.tsx:712-719 (`handleExport('current')` / `handleExport('all')`)                                         |

**Score:** 12/19 must-haves fully verified; 4 partially verified with anti-patterns; 3 outright failed (CR-01 export blob; missing backend endpoint; CR-03 1007 race).

### Deferred Items

None. Items described as "deferred" in 14-CONTEXT.md (OUI vendor, workstation entry, Gantt node focus, single-MAC export button, PDF export) are tracked separately and don't affect this verification.

### Required Artifacts

| Artifact                                                                                       | Expected                                | Status      | Details                                                                                       |
| ---------------------------------------------------------------------------------------------- | --------------------------------------- | ----------- | --------------------------------------------------------------------------------------------- |
| `xingran-react-frontend/src/lib/api/networkApi.ts`                                               | queryMACHistory / getMACEvents / exportMACHistory | ⚠ PARTIAL  | exports exist; exportMACHistory returns wrong type (CR-01)                                    |
| `xingran-react-frontend/src/components/network/MACEventsTimeline.tsx`                            | cross-page reusable vertical timeline   | ✓ VERIFIED  | 4 event colors + icons match MACTrajectoryChart (D-10); click → navigate to trajectory page  |
| `xingran-react-frontend/src/components/network/index.ts`                                         | barrel export                           | ✓ VERIFIED  | exports both MACTrajectoryChart and MACEventsTimeline                                        |
| `xingran-react-frontend/src/pages/network/mac/history.tsx`                                       | main list page (desktop + mobile)       | ✓ VERIFIED  | 8 columns, virtual scroll, URL params, mobile List, three-state wired                        |
| `xingran-react-frontend/src/pages/network/mac/history/index.tsx`                                 | route re-export shim                    | ✓ VERIFIED  | `export { default } from '../history'`; Vite glob picks this (componentLoader.tsx:33-35)      |
| `xingran-react-frontend/src/pages/network/mac/trajectory.tsx`                                    | trajectory page UX enhancements         | ✓ VERIFIED  | 5 enhancements (presets, URL, dataZoom, heatmap, Drawer) all present                         |
| `xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx`                          | empty-state shared component            | ⚠ PARTIAL  | works in main flow; CR-04 (type-narrow) is a latent bug                                      |
| `xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx`                          | error shared component                  | ⚠ PARTIAL  | 1006/1007/500 cases correct; CR-03 (logout race)                                              |
| `xingran-react-frontend/src/components/shared/index.ts`                                         | barrel export                           | ✓ VERIFIED  | EmptyStateWithAction + ErrorAlertWithRetry exported alongside legacy                         |
| `xingran-react-frontend/src/pages/network/devices/index.tsx`                                     | entry button to MAC history             | ✓ VERIFIED  | row action + detail modal both wired; HistoryOutlined icon                                   |
| `.planning/phases/14-frontend-ux/14-menu-registration.sql`                                     | idempotent sys_menu INSERT              | ✓ VERIFIED  | 3 INSERTs + 1 SELECT verification + rollback                                                |
| `.planning/phases/14-frontend-ux/14-03-ROUTE-SETUP.md`                                        | route + permission doc                  | ✓ VERIFIED  | 8 checklist items, rollback SQL                                                              |

### Key Link Verification

| From                                                  | To                                                          | Via                                  | Status      | Details                                                                                                                                |
| ----------------------------------------------------- | ----------------------------------------------------------- | ------------------------------------ | ----------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `pages/network/mac/history.tsx`                       | `lib/api/networkApi.ts`                                    | `queryMACHistory`                    | ⚠ WIRED-BROKEN | call exists but target endpoint doesn't exist on backend                                                                              |
| `pages/network/mac/history.tsx`                       | `components/network/MACEventsTimeline.tsx`                  | expandedRowRender                    | ✓ WIRED     | renders in expanded row                                                                                                                |
| `components/network/MACEventsTimeline.tsx`            | `/network/mac/trajectory`                                  | `navigate(\`/network/mac/trajectory?...\`)` | ✓ WIRED     |                                                                                                                              |
| `pages/network/mac/history.tsx`                       | `lib/api/networkApi.ts` `exportMACHistory`                 | `exportMACHistory`                   | ✗ BROKEN    | returns BaseResponse envelope, not Blob                                                                                                |
| `pages/network/mac/history.tsx`                       | `URL.createObjectURL + a.download`                          | blob download                        | ✗ BROKEN    | receives non-Blob object; produces broken .xlsx file                                                                                   |
| `pages/network/mac/history.tsx`                       | `useMenuStore.permissions` for `network:mac:export`         | `canExport` boolean                  | ✓ WIRED     | history.tsx:110-112                                                                                                                    |
| `pages/network/devices/index.tsx`                     | `/network/mac/history?deviceId=...`                        | `navigate`                           | ✓ WIRED     | both row action (line 183) and detail Modal footer (line 947)                                                                          |
| `components/network/MACEventsTimeline.tsx`            | `lib/api/networkApi.ts` `getMACEvents`                     | `useQuery(['macEvents', mac, startTime, endTime])` | ✓ WIRED     |                                                                                                                              |
| `pages/network/mac/trajectory.tsx`                    | `components/network/MACTrajectoryChart.tsx`                | `dataZoomStart/End` props            | ✓ WIRED     | chart merges into `option.dataZoom[0]` (MACTrajectoryChart.tsx:173-174)                                                                |
| `pages/network/mac/trajectory.tsx`                    | `components/network/MACEventsTimeline.tsx`                 | Drawer render                        | ✓ WIRED     | trajectory.tsx:286-290                                                                                                                  |

### Data-Flow Trace (Level 4)

| Artifact                            | Data Variable               | Source                                           | Produces Real Data | Status             |
| ----------------------------------- | --------------------------- | ------------------------------------------------ | ------------------ | ------------------ |
| `pages/network/mac/history.tsx`     | `list` / `pageData.list`    | `useTableQuery` → `queryMACHistory` → `POST /network/history/list` | NO (404)           | ✗ DISCONNECTED     |
| `pages/network/mac/history.tsx`     | `blob` from export          | `exportMACHistory` → `api.default.get` → interceptor unwraps envelope | NO (JSON object)  | ✗ DISCONNECTED     |
| `pages/network/mac/trajectory.tsx`  | `trajectoryData`            | `useQuery` → `queryMACTrajectory` → `POST /network/history/trajectory` | YES (endpoint exists) | ✓ FLOWING         |
| `components/network/MACEventsTimeline.tsx` | `events`             | `useQuery` → `getMACEvents` → `POST /network/history/list` | NO (404)           | ✗ DISCONNECTED     |

### Behavioral Spot-Checks

| Behavior                                                                  | Command                                                  | Result                                          | Status                |
| ------------------------------------------------------------------------- | -------------------------------------------------------- | ----------------------------------------------- | --------------------- |
| TypeScript compile                                                        | `npx tsc --noEmit -p .`                                  | exit 0                                          | ✓ PASS                |
| `queryMACHistory` reaches a real endpoint                                 | `grep -rn "/history/list" internal/`                     | no matches                                      | ✗ FAIL (endpoint missing) |
| `exportMACHistory` returns a Blob                                         | read `networkApi.ts:130-141`                             | returns `response as Blob` but response is BaseResponse | ✗ FAIL        |
| `URL.createObjectURL` receives a Blob                                     | read `history.tsx:385`                                   | receives BaseResponse object                    | ✗ FAIL                |
| 6 time presets present                                                    | `grep "近 1h\|近 24h\|近 7d\|近 30d\|近 90d" history.tsx` | all 5 present                                   | ✓ PASS                |
| Permission gate on export buttons                                         | `grep "canExport &&" history.tsx`                        | line 707 wraps both buttons                     | ✓ PASS                |
| Device page navigate to MAC history                                       | `grep "navigate(\`/network/mac/history?deviceId=" devices/index.tsx` | two sites                                          | ✓ PASS                |

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| (no conventional probes defined for frontend UX phase) | — | — | SKIP |

### Requirements Coverage

| Requirement | Source Plan              | Description                                          | Status      | Evidence                                                                                                |
| ----------- | ------------------------ | ---------------------------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------- |
| UI-01       | 14-01, 14-03, 14-05b     | MAC 历史查询页面 (时间筛选/分页/操作)                  | ⚠ PARTIAL  | UI 渲染就绪;后端 endpoint 不存在 (data source disconnected)                                              |
| UI-02       | 14-04, 14-05b            | 数据导出功能 (Excel)                                  | ✗ FAILED   | 按钮存在但下载的不是 .xlsx(CR-01)+ 后端无 format=xlsx 分支                                            |
| UI-04       | 14-01, 14-05b            | 历史事件时间线组件                                     | ⚠ PARTIAL  | MACEventsTimeline 组件就位;但 timeline 也调不存在的 /network/history/list,实际数据加载为空            |
| UI-03       | 14-02 (refers Phase 13)  | MAC 轨迹可视化                                        | ✓ VERIFIED | trajectory.tsx + MACTrajectoryChart.tsx 增强,后端 /history/trajectory 已存在                              |

### Anti-Patterns Found

| File                                              | Line(s)     | Pattern                                                                 | Severity      | Impact                                                                                              |
| ------------------------------------------------- | ----------- | ----------------------------------------------------------------------- | ------------- | --------------------------------------------------------------------------------------------------- |
| `lib/api/networkApi.ts`                          | 134-140     | Dynamic `await import('../api')` + `api.default.get` + `as Blob` cast  | 🛑 BLOCKER   | Fragile import + type lie; returns envelope, not Blob                                               |
| `lib/api/networkApi.ts`                          | 130-141     | No `blob.size < 1024` JSON error parse (planned but never implemented)  | 🛑 BLOCKER   | Error path from plan 14-04 is dead code                                                              |
| `components/shared/ErrorAlertWithRetry.tsx`      | 80-87       | useEffect on `[code, logout]` without ref/cancelled guard               | 🛑 BLOCKER   | 1007 fires logout multiple times; race on StrictMode                                                  |
| `components/shared/EmptyStateWithAction.tsx`     | 32, 42      | `Boolean(actionLabel && actionPath)` + `as string` cast                | ⚠ WARNING   | Empty-string / numeric actionPath still renders Link with empty `to`                                 |
| `pages/network/mac/history.tsx`                   | 409-411     | `void navigator.clipboard?.writeText(mac)`                              | ⚠ WARNING   | No user feedback on copy; errors swallowed                                                          |
| `pages/network/mac/history.tsx` + `history/index.tsx` | both exist | Duplicate page entries                                                  | ⚠ WARNING   | Fragile: Vite glob picks index.tsx but two-source-of-truth drift risk (CR-05)                       |
| `pages/network/mac/trajectory.tsx`                | 53-58       | `queryKey: ['macTrajectory', queryParams]` includes fresh object + `!` non-null assertion | ℹ️ INFO | WR-06 — `enabled` guard already short-circuits, assertion is dead but harmless                            |
| `components/network/MACEventsTimeline.tsx`        | 111-116     | `enabled: !!mac` does not validate startTime/endTime ISO strings         | ⚠ WARNING   | Bad input → 400 from backend (WR-05)                                                                  |
| `pages/network/devices/index.tsx`                 | 570-574     | mount-only useEffect with eslint-disable                                | ⚠ WARNING   | WR-02 — loadDevices/loadStatistics ref instability risk                                              |
| `pages/network/mac/history.tsx`                   | 134-154     | URL-param effect runs once; future navigations to same page won't re-read params | ⚠ WARNING   | WR-01 — works for current UX but brittle                                                         |

### Human Verification Required

Items requiring human testing or user-environment verification (these are not blockers — the code paths exist and can be exercised, but the broken behaviors identified above require either (a) running the app with the backend up after the endpoint is added or (b) a browser to confirm download flow).

### Gaps Summary

The phase contains **3 BLOCKER-level gaps** that prevent UI-01 (main list page) and UI-02 (Excel export) from being observable as functional features:

1. **Missing backend endpoint `/network/history/list`.** The Phase 14 PLAN locked this endpoint (D-01) without verifying the backend actually exposes it. The backend only has `/history/port`, `/history/device`, `/history/trajectory`, `/history/stats`, `/history/vendor`. Both `queryMACHistory` and `getMACEvents` (used by the timeline) hit a 404. UI-01 page loads but renders an empty list forever (or until the user manually retries an error path).

2. **Broken blob download in export (CR-01).** `exportMACHistory` returns the intercepted BaseResponse envelope typed-as-Blob. `URL.createObjectURL({code:0, ...})` produces a "blob:" URL pointing to a JSON object, and the browser saves it as `mac_history_current_<ts>.xlsx`. The user gets a .xlsx-named file with JSON content — corrupted download. The plan's `blob.size < 1024` error detection was never implemented.

3. **Logout race in ErrorAlertWithRetry (CR-03).** `useEffect` on `[code, logout]` re-fires on every render of a 1007 error, double-logout in StrictMode and token-clearance race during navigation.

Additional WARNING-level issues (CR-04 EmptyStateWithAction, WR-01/02/05/06/07, duplicate history/index file pair, copyMAC silent failure) are documented in the anti-patterns table but do not by themselves block the phase goal.

The 4 plans with PASS-level findings (14-02, 14-03, 14-05a, 14-05b) are solid: their core deliverables — trajectory UX enhancements, SQL registration, shared components, mobile responsive List, three-state wiring, device page entry button — are all present and wired correctly. The 2 plans with FAIL-level findings (14-01's data source; 14-04's blob handling) are where the goal is broken.

---

_Verified: 2026-06-15T02:00:00Z_
_Verifier: Claude (gsd-verifier)_