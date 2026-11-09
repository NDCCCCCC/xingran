---
phase: 144
plan: 02
wave: 1
depends_on: []
files_modified:
  - xingran-react-frontend/src/pages/system/ad-drift-alert/index.tsx (new)
  - xingran-react-frontend/src/pages/system/ad-drift-alert/components/StatusCards.tsx (new)
  - xingran-react-frontend/src/pages/system/ad-drift-alert/components/TriggerLogTable.tsx (new)
  - xingran-react-frontend/src/pages/system/ad-drift-alert/components/ConfigForm.tsx (new)
  - xingran-react-frontend/src/pages/system/ad-drift-alert/__tests__/index.render.test.tsx (new)
  - xingran-react-frontend/src/lib/adDriftAlertApi.ts (new)
  - xingran-react-frontend/src/lib/__tests__/adDriftAlertApi.test.ts (new)
  - xingran-react-frontend/src/store/noticeStore.ts
  - xingran-react-frontend/src/store/noticeStore.test.ts
  - xingran-react-frontend/src/types/notice.ts
  - xingran-react-frontend/src/constants/routes.ts
  - xingran-react-frontend/src/constants/pageTitles.ts
  - xingran-react-frontend/src/constants/permissionCodes.ts
  - xingran-react-frontend/src/hooks/__tests__/useADDriftAlertWebSocket.test.tsx (new)
autonomous: true
commit: false   # Lead 统一提交，禁止本 executor git add/commit
---

# Phase 144 · 同步漂移告警通道收口 — 前端（ALRT-03）

> 本 PLAN 只做**前端**。后端端点/WS/migration/权限常量由同目录 `PLAN-01-backend.md` 的独立 executor 并行负责，两者文件面互斥（本 PLAN 不碰任何 `*.go` 与 `.planning/**` 之外的后端文件）。

## 必读（按顺序）
1. `.planning/notes/261108-v154-architecture-decisions.md` **§二**（Lead 裁决，尤其「前端 WS：不新建第二条连接」）
2. `.planning/ROADMAP.md` Phase 144 段（成功标准 5 条）
3. `.planning/REQUIREMENTS.md` ALRT-03
4. `AGENTS.md` / `CLAUDE.md` 前端约定（wrapped API / Zustand / useEffect 依赖稳定 / 三态组件）

## 目标
交付「同步漂移告警」前端页面：状态卡 + 触发记录表（分页/筛选/已读）+ 配置表单（9 旋钮 + webhook 开关/下拉）+「立即检测」按钮 + WS 实时 toast；页面由菜单驱动动态路由自动发现。

## 已确认事实（不要重新调研）
- 前端路由是**菜单驱动动态路由**：`src/router/componentLoader.tsx:36-53` 用 `import.meta.glob("/src/pages/**/{index,detail}.tsx")` 懒加载，`App.tsx` **不含静态业务路由表**；`src/constants/routes.ts` / `pageTitles.ts` 只放常量。→ **不要改 `App.tsx`**，新建 `src/pages/system/ad-drift-alert/index.tsx` 即被 glob 自动收；后端 migration 238（PLAN-01）会播种 component=`system/ad-drift-alert/index` 的 C 型菜单。
- **全局唯一 WS 连接**：`src/components/NotificationBell.tsx:76-101` 已连 `/system/ws/notices` 并把消息转 `useNoticeStore.getState().handleWsMessage`；`src/store/noticeStore.ts:132-199` 只分流 `new_notice` / `rpa_*` / `port_write_*`；`src/types/notice.ts:200-206` 的 `WebSocketMessageType` 无 drift 类型；`src/hooks/useReconciliationWebSocket.ts` / `useWebSocket.ts` / `src/lib/wsUrl.ts:14-15`（后端只注册 `/notices` 一个 WS 端点）。
- 同款「模块级 listener Set + store 内分流」先例：`noticeStore.ts` 的 RPA / portWrite 订阅（约 L11-22 定义 Set、L183-217 分流与 `on*` 导出）。
- wrapped API：`src/lib/api.ts:531/535/539/543` 的 `post/get/put/del`，返回 `BaseResponse<T>` 直接取 `result.data`；参照 `src/lib/noticeApi.ts:21` 的 `createResourceApi` 或 `src/lib/notificationConfigApi.ts:165-167`（`getAPINotificationConfigList({configType:'webhook'})`）。**禁止裸 axios**。
- 通知配置页 `src/pages/system/settings/api-config.tsx` 已支持 Webhook 类型配置（`configType='webhook'`）；告警页只做「开关 + 下拉选择已有配置」，**不新建配置 CRUD 页**。
- 权限码三方一致：`src/constants/permissionCodes.ts` / `pkg/permission/config.go`（PLAN-01 加）/ `sys_menu.perms`（migration 238 播种）；`scripts/check-permission-codes.mjs` 双向校验会 exit 1。命名以 Lead 裁决为准：`system:adDriftAlert` + `:list` / `:config` / `:edit` / `:trigger`。
- 既有就近测试布局：`src/pages/ad-domain/configs/__tests__/index.render.test.tsx`、`src/store/noticeStore.test.ts`、`src/lib/api/__tests__/networkApi.test.ts`、`src/hooks/__tests__/useReconciliationWebSocket.test.tsx`。
- 端点契约（PLAN-01 实现）：`POST /system/ad-drift-alert/{status,logs,config,config/update,trigger}`；`logs` 分页 `list/total/current/pageSize`（current 从 1）；`status` 返回 `{enabled,tolerance,throttleMinutes,noticeEnabled,wsEnabled,webhookEnabled,baseline,lastAlertAt,lastAlertDrifted,alertCount}`；`config` 返回 9 个旋钮；`config/update` 写 9 旋钮；`trigger` 写操作。
- WS 消息（PLAN-01 实现）：`NoticeMessage{Type:"ad_sync_drift_alert", NoticeID, Title, Content(内嵌业务 JSON), Priority, Timestamp, UnreadCount}`。

## Lead 裁决（必须遵守）
| 项 | 裁决 |
|---|---|
| WS | **不新建第二条 WebSocket**；扩 `src/types/notice.ts` 的 `WebSocketMessageType` 加 `"ad_sync_drift_alert"` + 新增 `ADSyncDriftAlertEvent` 接口；扩 `noticeStore.handleWsMessage` 分支解析 `message.content` 内嵌 JSON 并派发到模块级 `driftAlertListeners` Set（照 portWrite/RPA 模式）+ `App` message.warning 全局提示；导出 `onADDriftAlert()` 订阅（含退订） |
| 页面 | 新建 `src/pages/system/ad-drift-alert/index.tsx` + `components/`（StatusCards / TriggerLogTable / ConfigForm）；用 `useMemo`/primitive 稳定 `useEffect` 依赖，禁止对象字面量直接进依赖（CLAUDE.md） |
| API | 新建 `src/lib/adDriftAlertApi.ts`，**必须** `import { post } from "@/lib/api"`（wrapped），返回 `result.data` 直接用；禁止裸 axios、禁止手动判 `code===0` |
| 路由 | 不改 `App.tsx`；在 `src/constants/routes.ts` 加 `SYSTEM_AD_DRIFT_ALERT`、`src/constants/pageTitles.ts` 加标题；菜单由后端 migration 238 播种 |
| 权限码 | 在 `src/constants/permissionCodes.ts` 增 `adDriftAlert` 组，5 个码**逐字**为 `system:adDriftAlert` / `system:adDriftAlert:list` / `system:adDriftAlert:config` / `system:adDriftAlert:edit` / `system:adDriftAlert:trigger`（与 PLAN-01 的 `pkg/permission/config.go` + migration 238 一致）；运行 `node scripts/check-permission-codes.mjs` 应通过（若后端 PLAN 尚未落地导致反向校验失败，如实记录并在 SUMMARY 说明「待后端完成后由 Lead 复跑」） |
| webhook UI | 告警页「webhook 开关 + 下拉选 `getAPINotificationConfigList({configType:'webhook'})`」；**不展示明文 secret**（配置页已有自己的遮蔽），不在前端日志打印 URL |
| 三态 | 列表/表单用既有 `PageTable` / `EmptyStateWithAction` / `ErrorAlertWithRetry` / `handleApiError` 唯一错误出口；icon-only 按钮带 aria-label |

## 任务（XML）
<task id="1">
<title>类型 + store 分流（不新建 WS 连接）</title>
<files>src/types/notice.ts, src/store/noticeStore.ts, src/store/noticeStore.test.ts</files>
<action>
① `WebSocketMessageType` 加 `"ad_sync_drift_alert"`；新增 `ADSyncDriftAlertEvent` 接口（memberCount/distinctGroups/baseline/deltas/noticeId/tolerance/throttled/timestamp 等，字段与前端口径对齐）。② `noticeStore.ts` 加模块级 `driftAlertListeners: Set<(e: ADSyncDriftAlertEvent) => void>` + 导出 `onADDriftAlert(listener)`（返回退订函数，照 RPA/portWrite 模式）；`handleWsMessage` 加 `"ad_sync_drift_alert"` 分支：`JSON.parse(message.content)` 容错（解析失败仅 console.warn 不抛）、派发给所有 listeners（try/catch 隔离单个 listener 异常）、`App.useApp()` 无法在 store 直接用 → 用 `message.warning` 的静态导入或把提示放页面订阅回调里（**选一种并写清**）；既有 `new_notice` / `rpa_*` / `port_write_*` 分流保持不变。③ `noticeStore.test.ts` 扩：新类型分流、listener 订阅/退订、解析失败不崩溃、其它类型不误触发。**不要 `new WebSocket`、不要改 `NotificationBell.tsx`**。
</action>
</task>

<task id="2">
<title>API 封装 + 常量 + 权限码</title>
<files>src/lib/adDriftAlertApi.ts (new), src/lib/__tests__/adDriftAlertApi.test.ts (new), src/constants/routes.ts, src/constants/pageTitles.ts, src/constants/permissionCodes.ts</files>
<action>
`adDriftAlertApi.ts` 导出 `getStatus()` / `listTriggerLogs(params)` / `getConfig()` / `updateConfig(payload)` / `triggerOnce()`，全部 `post<T>(...)` from `@/lib/api`，直接返回 `result.data`；类型定义放同文件（或 `src/types/adDriftAlert.ts` 若你选择新建，需在 files_modified 里说明）。`adDriftAlertApi.test.ts` 断言走 wrapped `post`、无裸 axios、路径正确。常量：`SYSTEM_AD_DRIFT_ALERT` 路由 + 页面标题；`permissionCodes.ts` 增 `adDriftAlert` 组（5 码逐字如上）。运行 `node scripts/check-permission-codes.mjs`（在仓库根执行；若因后端 PLAN-01 尚未落地导致反向校验失败，如实记录，不硬改 `.go` 文件）。
</action>
</task>

<task id="3">
<title>页面 + 组件</title>
<files>src/pages/system/ad-drift-alert/index.tsx (new), components/StatusCards.tsx (new), components/TriggerLogTable.tsx (new), components/ConfigForm.tsx (new), __tests__/index.render.test.tsx (new)</files>
<action>
`index.tsx`：懒加载由 glob 自动收；组合三块子组件 + 「立即检测」按钮（`triggerOnce()` 后刷新状态与日志，按钮三态门控用 `system:adDriftAlert:trigger`）+ `onADDriftAlert` 订阅（新告警时 `message.warning` + `queryClient.invalidateQueries` 或本地 refetch，退订放 cleanup）。
`StatusCards`：当前成员数/组数、基线、累计告警次数、节流状态、各通道开关（enabled/wsEnabled/webhookEnabled/noticeEnabled）。
`TriggerLogTable`：`listTriggerLogs` 分页（current 从 1，pageSize）、按时间/状态筛选、`PageTable` + `EmptyStateWithAction` + `ErrorAlertWithRetry`，时间用 dayjs。
`ConfigForm`：9 个 sys_config 旋钮（enabled / tolerance / throttle_minutes / notice_enabled / ws_enabled / webhook_enabled / webhook_config_id / webhook_template / baseline 的展示）+ webhook 下拉（`getAPINotificationConfigList({configType:'webhook'})` from `@/lib/notificationConfigApi`）；保存走 `updateConfig`；**不展示明文 secret**；按钮用 `:edit` 权限门控。
`__tests__/index.render.test.tsx`：渲染断言（mock `adDriftAlertApi`），参照 `src/pages/ad-domain/configs/__tests__/index.render.test.tsx`。
`useEffect` 依赖全部 primitive/useMemo，`params` 对象用 useMemo；不要造成无限请求循环。
</action>
</task>

<task id="4">
<title>WS 订阅 hook 测试（可选 hook 本体 + 必有测试）</title>
<files>src/hooks/__tests__/useADDriftAlertWebSocket.test.tsx (new)</files>
<action>
若实现中抽了 `useADDriftAlertWebSocket`（复用 `onADDriftAlert` 订阅，不新建连接），补测试；否则直接测 `onADDriftAlert` 订阅在页面中的调用与退订（可在 `index.render.test.tsx` 内覆盖）。断言：收到 `ad_sync_drift_alert` 时刷新回调被调用一次、卸载后退订、其它类型不触发。
</action>
</task>

<task id="5">
<title>自验证（只跑前端相关，不跑全仓）</title>
<action>
`cd xingran-react-frontend && npm run type-check && npx vitest run src/store/noticeStore.test.ts src/lib/__tests__/adDriftAlertApi.test.ts src/pages/system/ad-drift-alert src/hooks/__tests__/useADDriftAlertWebSocket.test.tsx`
`node scripts/check-permission-codes.mjs`（仓库根，如实记录结果）
不要跑 `npm run test` 全量 / Go 命令（Lead 集成后统一跑）。失败只修本 phase 文件面问题，迭代至通过。
</action>
</task>

<task id="6">
<title>写 SUMMARY（不提交）</title>
<action>
写 `.planning/phases/144-alrt-01-02-03/144-02-frontend-SUMMARY.md`：改动清单、ALRT-03 满足情况、验证结果、权限码双向校验结果（含「待后端完成后 Lead 复跑」的如实说明）、Known Gaps、以及「未 git commit」说明。
</action>
</task>

## 验证标准（must_haves）
1. 页面可被菜单驱动 glob 发现（`src/pages/system/ad-drift-alert/index.tsx` 存在），`App.tsx` 未改。
2. 全局 WS 唯一：只扩 `noticeStore` 分流 + 类型 + listener，**无第二条 `new WebSocket`**；收到 `ad_sync_drift_alert` 实时刷新/提示有测试。
3. 5 个 API 全走 `@/lib/api` wrapped `post`，禁止裸 axios；`permissionCodes.ts` 的 5 码与后端逐字一致。
4. 状态卡/日志表/配置表单/立即检测可用；三态组件与 `handleApiError` 唯一错误出口；`useEffect` 依赖稳定。
5. `type-check` + 4 组 vitest 全绿（或失败如实记录）；`check-permission-codes.mjs` 通过或如实记录待后端同步。
6. 未触碰其它 phase 文件面；零新依赖。

## 禁止
- 禁止 `git add` / `git commit` / `git stash` / `git checkout`（Lead 统一集成提交）。
- **禁止修改任何 `*.go` 文件**（后端 PLAN-01 独占）；禁止修改 `pkg/permission/config.go` / `internal/core/db/database.go` / `internal/api/router.go`。
- 禁止修改 `src/components/network/port-write/**`、`src/types/network.ts`、`src/lib/api/networkApi.ts`、`src/components/network/port-write/constants.ts`（Phase 143 独占）。
- 禁止修改 `src/lib/api.ts`（barrel）、`src/components/NotificationBell.tsx`（既有全局连接）、`src/App.tsx`（菜单驱动不用静态路由）。
- 禁止新建第二条 WebSocket 连接、禁止裸 axios、禁止在前端日志打印 webhook secret。
- 禁止启动长驻进程（AGENTS.md）、禁止写生产数据。
