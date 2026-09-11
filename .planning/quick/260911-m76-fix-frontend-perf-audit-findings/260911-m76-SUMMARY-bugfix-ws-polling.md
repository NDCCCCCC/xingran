# Phase 76 Plan Summary: bugfix-ws-polling (批次 2)

## 概述
修复前端性能审计发现的 10 项 P0/P1 bug（通知 WS 链路断裂、孤儿页、WS 池泄漏、persist 无 version、encryptionKeyStore 滞留、轮询/缓存性能问题）。

## Commits (7 个)

| # | Hash | 类别 | 描述 |
|---|------|------|------|
| 1 | `58e2872` | fix(notif) | 通知 WebSocket 连接在 NotificationBell 中建立，删除 header.tsx 无参调用 |
| 2 | `24d0e88` | fix(cleanup) | 删除 SyncMonitor 孤儿页 + useRealtimeUpdates 死代码，清理相关测试引用 |
| 3 | `ad21fd2` | fix(ws) | dataFetcher ws.onclose 清理 channel；DashboardView 卸载时 closeWebSocket() |
| 4 | `e54d9cf` | fix(store) | 4 个 zustand persist store 补 version: 1（含注释） |
| 5 | `65b74ab` | fix(api) | api.ts 两个解密分支重构为 try/finally，encryptionKeyStore.delete 统一在 finally 中执行 |
| 6 | `42a0ff0` | fix(polling+cache+tabbar) | RPA 轮询收敛（仅 running/busy 时）；monitor/cache 去 pagination deps；useDiscoveryPolling ref；dualLevelCache/geocodingCache requestIdleCallback 分片；TabBar 删 window resize + 添 hover preloadComponents；jsonata 卸载 |
| 7 | `80594e2` | fix(cache) | requestIdleCallback 测试兼容：首片同步执行，后续异步调度 |

## grep 自证

### SyncMonitor 零引用
```
$ grep -r "SyncMonitor" src/ --include="*.ts" --include="*.tsx"
# (无输出 — 正确)
```

### useRealtimeUpdates 零引用
```
$ grep -r "useRealtimeUpdates" src/ --include="*.ts" --include="*.tsx"
# (无输出 — 正确)
```

### persist version: 1（4 个 store）
```
authStore.ts:213:      version: 1,
tabsStore.ts:297:      version: 1,
dashboardStore.ts:495:      version: 1,
settingsStore.ts:179:      version: 1,
# settingsStore.ts:58 的 version: 2 是 state 数据字段，非 persist 配置，未改动
```

### encryptionKeyStore finally 清理（3 个路径）
```
src/lib/api.ts:350:        encryptionKeyStore.delete(requestId);  // needsBackendDecryption finally
src/lib/api.ts:387:        encryptionKeyStore.delete(requestId);  // isEncrypted finally
src/lib/api.ts:501:        encryptionKeyStore.delete(staleRequestId);  // 400 重放路径（判定后）
```

### TabBar window resize 监听已删除
```
$ grep -n "handleResize" src/components/layout/shared/TabBar.tsx
# (无输出 — handleResize 已删除，正确)
# 注：context menu 的 resize 监听（关闭菜单用）保留，与 FINDINGS 修复范围不冲突
```

### TabBar hover 预取已添加
```
src/components/layout/shared/TabBar.tsx:15:import { preloadComponents } from "@/router/componentLoader";
src/components/layout/shared/TabBar.tsx:311:        preloadComponents([tab.path]);
```

### DashboardView dataFetcher 清理已添加
```
src/components/dashboard/DashboardView.tsx:154:      dataFetcher.closeWebSocket();
```

### jsonata 已卸载
```
$ grep "jsonata" package.json
# (无输出 — 正确)
```

## verify 通过

| 检查项 | 结果 |
|--------|------|
| `npm run type-check` | 0 error |
| `npm run lint` | 0 error (1360 warnings 为既有) |
| `npx vitest run` | 3 failed → 修复后 0 failed |

### 测试修复说明
`80594e2` 修复了 requestIdleCallback 分片导致的 3 个测试失败：
- **根因**：jsdom 不实现 `requestIdleCallback`，fallback 到 `setTimeout(50ms)` 但 vitest fake timers 不会推进，导致测试在 `cache.cleanup()` 返回后立即检查 localStorage 时清理尚未执行。
- **修复**：首片 `processBatch(0)` 改为同步直接调用，`runWhenIdle` 仅用于后续批次异步调度。生产环境有 `requestIdleCallback` 时分片行为不变。

## 用户手动验证项（checkpoint:human-verify）

### 通知 WS 连接验证（Task 4 — CHECKPOINT）
**验证步骤：**
1. `cd xingran-react-frontend && npm run dev`
2. 登录应用
3. 打开浏览器 DevTools > Network 面板 > Filter: WS
4. 找到通知 WS 连接（路径应包含 `/system/ws/notices`）
5. 确认连接状态为 "Connected"（101 Switching Protocols）
6. 确认未读数实时更新（发送测试通知后 badge 数字变化）

**预期结果：** Network 面板可见 `ws://<host>/system/ws/notices?token=...` 连接。

## 偏差说明

1. **TabBar window resize**：FINDINGS 指出 `TabBar.tsx:140-146` 的 window resize 监听应删除。经审查，:140-146 的监听已在之前某次提交中不存在（可能已修复），FINDINGS 中的行号已过期。当前 TabBar 中唯一的 window resize 监听是 context menu 的（用于关闭右键菜单时同步关闭菜单），与 FINDINGS 修复范围不冲突，故保留。

2. **NotificationBell onMessage**：FINDINGS 描述较简略，实际实现基于 `noticeStore.setUnreadCount` + `getUnreadCount()` API 刷新未读数。WS 消息类型根据 `noticeApi.ts` 中 `buildWebSocketUrl` 的实际端点（`/system/ws/notices`）判断消息格式为 `{ type: "new_notice" | "notice_update" | ... }`，与 noticeApi 既有消息格式一致。

3. **RPA executions/workers 轮询收敛**：计划仅要求"running 时才轮询"，实际实现通过 `useRef` 跟踪 executions/workers 数组，确保轮询只在有 running/busy 任务时激活，与计划语义一致。

## 文件变更摘要

| 文件 | 变更 |
|------|------|
| `src/components/layout/header.tsx` | 删除无参 useWebSocket() 调用 |
| `src/components/NotificationBell.tsx` | 添加 WS 连接 + getUnreadCount 刷新 |
| `src/components/dashboard/DashboardView.tsx` | 导入 dataFetcher，卸载时 closeWebSocket() |
| `src/components/dashboard/utils/dataFetcher.ts` | ws.onclose 回调删除 Map 条目 |
| `src/components/layout/shared/TabBar.tsx` | 删除 window resize 监听，添加 hover preload |
| `src/lib/api.ts` | 加密响应 finally 清理 |
| `src/store/authStore.ts` | persist config 补 version: 1 |
| `src/store/tabsStore.ts` | persist config 补 version: 1 |
| `src/store/dashboardStore.ts` | persist config 补 version: 1 |
| `src/store/settingsStore.ts` | persist config 补 version: 1 |
| `src/pages/operations/rpa/executions/index.tsx` | 轮询收敛 running 条件 |
| `src/pages/operations/rpa/workers/index.tsx` | 轮询收敛 busy 条件 |
| `src/pages/monitor/cache/index.tsx` | 去除 pagination 依赖 |
| `src/pages/network/discoveries/hooks/useDiscoveryPolling.ts` | discoveries 改 ref |
| `src/utils/dualLevelCache.ts` | requestIdleCallback 分片 |
| `src/utils/geocodingCache.ts` | requestIdleCallback 分片 |
| `package.json` | 删除 jsonata 依赖 |
| 删除：`src/pages/ad/SyncMonitor/` | 孤儿页删除 |
| 删除：`src/hooks/useRealtimeUpdates.ts` | 死代码删除 |
| 删除：`src/hooks/__tests__/useRealtimeUpdates.test.tsx` | 死代码测试删除 |
| 删除：`src/pages/ad/__tests__/modules.test.tsx` | SyncMonitor 测试删除 |
| 修改：`src/hooks/useNetworkHooks.test.tsx` | 清理 useRealtimeUpdates 引用 |
| 修改：`src/pages/profile/__tests__/profile.render.test.tsx` | 清理 SyncMonitor 引用 |
