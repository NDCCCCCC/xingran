---
quick_id: 260911-m76
slug: bugfix-ws-polling
phase: 76-bugfix-ws-polling
plan: 02
type: execute
wave: 1
depends_on: []
files_modified:
  - xingran-react-frontend/src/components/layout/header.tsx
  - xingran-react-frontend/src/components/NotificationBell.tsx
  - xingran-react-frontend/src/pages/ad/SyncMonitor/index.tsx
  - xingran-react-frontend/src/components/dashboard/utils/dataFetcher.ts
  - xingran-react-frontend/src/hooks/useRealtimeUpdates.ts
  - xingran-react-frontend/src/store/authStore.ts
  - xingran-react-frontend/src/store/tabsStore.ts
  - xingran-react-frontend/src/store/dashboardStore.ts
  - xingran-react-frontend/src/store/settingsStore.ts
  - xingran-react-frontend/src/utils/dualLevelCache.ts
  - xingran-react-frontend/src/utils/geocodingCache.ts
  - xingran-react-frontend/src/components/layout/shared/TabBar.tsx
  - xingran-react-frontend/src/router/componentLoader.tsx
autonomous: false
requirements: []
must_haves:
  truths:
    - 通知 WebSocket 连接成功建立（登录后 Network 面板可见 WS 连接）
    - SyncMonitor 孤儿页已删除（零引用）
    - dataFetcher.wsConnections Map 在 DashboardView 卸载时清理
    - zustand persist store 全部补 version: 1
    - encryptionKeyStore 在响应拦截器 finally 统一清理（无论成功/失败）
    - jsonata 死依赖已卸载
  artifacts:
    - path: xingran-react-frontend/src/pages/ad/SyncMonitor/
      deleted: true
    - path: xingran-react-frontend/src/hooks/useRealtimeUpdates.ts
      deleted: true
  key_links:
    - from: NotificationBell.tsx
      to: WebSocket server
      via: useWebSocket with connect call
    - from: api.ts response interceptor
      to: encryptionKeyStore
      via: finally block cleanup
---

<objective>
修复 5 个功能 bug（通知 WS 链路断裂、孤儿页、WS 池泄漏、persist 无 version、encryptionKeyStore 滞留）及其他轮询/缓存问题。
</objective>

<context>
@xingran-react-frontend/src/components/layout/header.tsx
@xingran-react-frontend/src/components/NotificationBell.tsx
@xingran-react-frontend/src/hooks/useWebSocket.ts
@xingran-react-frontend/src/components/dashboard/utils/dataFetcher.ts
@xingran-react-frontend/src/pages/ad/SyncMonitor/index.tsx
@xingran-react-frontend/src/store/authStore.ts
@xingran-react-frontend/src/store/tabsStore.ts
@xingran-react-frontend/src/store/dashboardStore.ts
@xingran-react-frontend/src/store/settingsStore.ts

**WS URL 定位：** grep `DashboardView` 或现有项目内 WS 用法确认通知 WS 端点格式（wss:// 或 ws:// + path）。
**useRealtimeUpdates.ts 删除前确认：** grep 确认仅测试文件引用，无生产引用。
</context>

<tasks>

<task type="auto">
  <name>Task 1: 通知 WebSocket 链路修复（2.1）</name>
  <files>
    xingran-react-frontend/src/components/layout/header.tsx
    xingran-react-frontend/src/components/NotificationBell.tsx
  </files>
  <action>
**header.tsx：** 删除第 22-25 行的无参 `useWebSocket()` 调用及错误注释（第 23-24 行声称模块级单例但实际无此机制）。

**NotificationBell.tsx（约第 72 行附近）：**
1. 确认 useWebSocket options 结构（url/onMessage/onOpen/onClose）
2. 参照 DashboardView.tsx:128 或 grep `ws://|wss://|/ws` 找到通知 WS 端点
3. 在 NotificationBell 组件内调用 `useWebSocket({ url: <通知WS地址>, onMessage: <现有未读数处理逻辑> })`
4. 用 useEffect 触发 connect：`useEffect(() => { connect(); }, [connect])`
5. 删除第 74 行「WebSocket 连接已在 Header 组件中初始化」错误注释
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check && npm run lint 2>&1 | tail -5</automated>
  </verify>
  <done>header.tsx 无参 useWebSocket 调用已删除；NotificationBell.tsx 实际建立 WS 连接；type-check + lint 通过</done>
</task>

<task type="auto">
  <name>Task 2: SyncMonitor 孤儿页删除（2.2）+ useRealtimeUpdates 死代码删除（2.5）</name>
  <files>
    xingran-react-frontend/src/pages/ad/SyncMonitor/
    xingran-react-frontend/src/hooks/useRealtimeUpdates.ts
  </files>
  <action>
**删除前确认零引用：**
```bash
grep -r "SyncMonitor" xingran-react-frontend/src/ --include="*.ts" --include="*.tsx" -l
grep -r "useRealtimeUpdates" xingran-react-frontend/src/ --include="*.ts" --include="*.tsx" -l
```

**SyncMonitor：** 整个 `src/pages/ad/SyncMonitor/` 目录删除（含 index.tsx 及任何子文件）。

**useRealtimeUpdates.ts：** 删除该文件及其测试文件（先 grep 确认测试文件存在）。
</action>
  <verify>
    <automated>grep -r "SyncMonitor\|useRealtimeUpdates" xingran-react-frontend/src/ --include="*.ts" --include="*.tsx" 2>/dev/null | grep -v "node_modules"</automated>
  </verify>
  <done>SyncMonitor 目录已删除，零引用；useRealtimeUpdates.ts 及测试文件已删除</done>
</task>

<task type="auto">
  <name>Task 3: dataFetcher wsConnections 泄漏修复（2.5）</name>
  <files>xingran-react-frontend/src/components/dashboard/utils/dataFetcher.ts</files>
  <action>
1. 在 `closeWebSocket` 方法的 `ws.onclose` 中（:218-227 附近）从 `this.wsConnections` Map 删除对应 channel 条目：
   ```ts
   ws.onclose = () => {
     this.wsConnections.delete(channel);
   };
   ```
2. 确认 DashboardView.tsx 卸载时调用 `dataFetcher.closeWebSocket()` 批量清理（grep 确认，或在 DashboardView useEffect cleanup 中添加）
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check 2>&1 | tail -5</automated>
  </verify>
  <done>ws.onclose 回调中从 Map 删除 channel 条目；DashboardView 卸载清理已连接</done>
</task>

<task type="checkpoint:human-verify" gate="blocking">
  <name>Task 4: 通知 WS 连接验证</name>
  <what-built>NotificationBell.tsx 中的 useWebSocket({ url, onMessage, connect }) 调用</what-built>
  <how-to-verify>
1. 启动 dev server: `cd xingran-react-frontend && npm run dev`
2. 登录应用
3. 打开浏览器 DevTools > Network 面板 > Filter: WS
4. 找到通知 WS 连接（ws:// 或 wss:// + 通知端点 path）
5. 确认连接状态为 "Connected"
6. 确认未读数实时更新
  </how-to-verify>
  <resume-signal>Type "approved" or describe issues</resume-signal>
</task>

<task type="auto">
  <name>Task 5: zustand persist 补 version（2.6）</name>
  <files>
    xingran-react-frontend/src/store/authStore.ts
    xingran-react-frontend/src/store/tabsStore.ts
    xingran-react-frontend/src/store/dashboardStore.ts
    xingran-react-frontend/src/store/settingsStore.ts
  </files>
  <action>
对每个 persist store（authStore/tabsStore/dashboardStore/settingsStore），在 persist 配置对象（第二参数）中补 `version: 1`（首次引入即 1）。

**注意：** settingsStore.ts:58 的 `version: 2` 是 state 字段，不是 persist 配置，不要动它。找到真正的 persist 配置位置（通常是 `persist(key, { ... }, { version: 1 })` 的第三个参数）。

加注释说明：
```ts
// version: 1 — 首次引入。字段结构变更时需递增 version 并补 migrate 函数
```
</action>
  <verify>
    <automated>grep -n "version:" xingran-react-frontend/src/store/authStore.ts xingran-react-frontend/src/store/tabsStore.ts xingran-react-frontend/src/store/dashboardStore.ts xingran-react-frontend/src/store/settingsStore.ts</automated>
  </verify>
  <done>4 个 persist store 全部补 version: 1；type-check + lint 通过</done>
</task>

<task type="auto">
  <name>Task 6: api.ts encryptionKeyStore finally 统一清理（2.7）</name>
  <files>xingran-react-frontend/src/lib/api.ts</files>
  <action>
在响应拦截器（`api.interceptors.response.use` 的第一个 handler）的 try 块末（解密成功后 data 处理完成处）与 catch 块末，都加 `encryptionKeyStore.delete(requestId)`。

**关键约束：** 400 重放路径（:500 附近）在判定重放前依赖 encryptionKeyStore 条目存在，删除时机须在重放判定之后。利用 try/catch/finally 结构：
```ts
try {
  // ... 解密和处理逻辑
  encryptionKeyStore.delete(requestId); // 成功后删除
  return data;
} catch (error) {
  encryptionKeyStore.delete(requestId); // 失败后也删除
  return Promise.reject(error);
}
```
</action>
  <verify>
    <automated>grep -n "encryptionKeyStore" xingran-react-frontend/src/lib/api.ts</automated>
  </verify>
  <done>encryptionKeyStore.delete 覆盖成功/失败两条路径；type-check + lint 通过</done>
</task>

<task type="auto">
  <name>Task 7: jsonata 死依赖卸载 + TabBar resize 删除 + requestIdleCallback（2.3/2.8/2.9）</name>
  <files>
    xingran-react-frontend/package.json
    xingran-react-frontend/src/components/layout/shared/TabBar.tsx
    xingran-react-frontend/src/utils/dualLevelCache.ts
    xingran-react-frontend/src/utils/geocodingCache.ts
  </files>
  <action>
**jsonata 卸载：** `cd xingran-react-frontend && npm uninstall jsonata`

**TabBar resize 删除（2.9）：** `TabBar.tsx:140-146` 的 window resize 监听（含 removeEventListener cleanup）全部删除；确认 ResizeObserver 仍覆盖容器尺寸变化。

**dualLevelCache/geocodingCache 定时清理（2.8）：** 将 cleanup 中的 `Object.keys(localStorage)` 逐条 getItem+JSON.parse 改为 `requestIdleCallback`（fallback setTimeout）分片执行，每片处理少量条目（如 10 条），保持 TTL 语义不变。
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check && npm run lint 2>&1 | tail -5</automated>
  </verify>
  <done>jsonata 已卸载；TabBar resize 监听已删除；缓存 cleanup 已改 requestIdleCallback 分片</done>
</task>

</tasks>

<verification>
cd xingran-react-frontend && npm run type-check && npm run lint && npm test
</verification>

<success_criteria>
- 通知 WS 连接成功建立（用户验证）
- SyncMonitor 零引用，useRealtimeUpdates 零引用
- dataFetcher.wsConnections 在 DashboardView 卸载时清理
- 4 个 persist store 全部有 version: 1
- encryptionKeyStore 无论成功/失败响应均清理
- jsonata 已卸载
</success_criteria>

<output>
创建 `.planning/quick/260911-m76-fix-frontend-perf-audit-findings/260911-m76-02-PLAN-SUMMARY.md`
</output>
