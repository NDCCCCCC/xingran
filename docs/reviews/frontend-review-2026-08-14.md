# 前端代码审查报告

**审查日期**: 2026-08-14
**审查范围**: `xingran-react-frontend/src/` 全部 14 个子目录 (313 tsx + 316 ts)
**审查方法**: 4 维度并行深入 + 源码复核验证
**客观基线**: tsc --noEmit 通过 (0 类型错误); ESLint 3887 问题 (2050 可自动修复)

---

## 问题汇总 (去重后 36 个独立问题)

| 等级 | 数量 | 性质 |
|------|------|------|
| P0 致命 | 4 | 运行时死循环 / 内存泄漏 / 权限空转 |
| P1 重要 | 12 | 安全缺陷 / 竞态 / 资源泄漏 |
| P2 建议 | ~20 | 性能 / 类型 / 代码质量 |

---

## P0 致命问题 (4 个, 已全部核实源码)

### P0-1 路由 hasPermission 零调用 — 权限守卫形同虚设
- **位置**: `router/routeConfigManager.ts:90-108`, `router/DynamicRoutes.tsx:55-82`
- **问题**: `hasPermission()` 已实现完整 RBAC 校验, 但全仓库零调用. `meta.permissions` 是死代码. 前端路由只做"是否登录"二值判断.
- **后果**: 已实现但不生效; 安全 100% 依赖后端中间件.

### P0-2 useWidgetPolling 双 effect 共用 intervalRef — interval 泄漏
- **位置**: `hooks/useWidgetPolling.ts:135-145` + `161-163`
- **问题**: 主 effect 和 visibility effect 都往同一个 `intervalRef.current` 写 setInterval. 第二个覆盖 ref 时第一个 interval 永远不会被 clearInterval.
- **后果**: 每次页面 visibility 切换泄漏一个定时器, 内存/资源持续增长.

### P0-3 数组/对象依赖未 memoize — effect 死循环
- **位置**: `useWidgetPolling.ts:145` (widgetIds), `useTableQuery.ts:58` (filters={}), `useWidgetData.ts:186` (widgets), `useRealtimeUpdates.ts:128` (options)
- **问题**: 父组件内联传入的数组/对象每次渲染都是新引用, useEffect 判定为"变化"反复重建定时器/WebSocket.
- **后果**: 死循环或连接风暴. (CLAUDE.md 明确警告的高危点)

### P0-4 useWebSocket 指数退避读闭包旧值失效
- **位置**: `hooks/useWebSocket.ts:112,138`
- **问题**: connect 依赖 reconnectAttempts state, 退避延迟计算读闭包旧值, 重连退避不生效.
- **后果**: 网络抖动时可能产生连接风暴.

---

## P1 重要问题 (12 个, 已核实)

### 安全类

**P1-S1 SM4 存储"加密"密钥与 IV 完全相同** (已计算验证)
- `utils/token/SecureTokenStorageImpl.ts:28-58`
- 两个不同输入前 16 字节恰好都是 "xingran-next-sec", 派生 hex 前 32 字符相同 = `78696e6772616e2d6e6578742d736563`
- SM4-CBC Key=IV, 安全性归零; 密钥硬编码在 JS, 伪加密.

**P1-S2 SM2 登录加密失败回退明文, 生产环境也生效**
- `utils/sm2.ts:139-146` — 注释写"仅开发环境"但无环境判断 (对比 api.ts:262 有判断)

**P1-S3 401 双重登出竞态**
- `lib/api.ts:424-451` 已有刷新逻辑, `errorHandler.ts:214-216` 又独立触发 logout

### 状态管理类

**P1-M1 双重并发刷新锁**
- `TokenManager.ts` refreshLock + `api.ts` isRefreshing 两套锁并存, getAccessToken 绕过 refreshQueue

**P1-M2 layoutStore 在 set updater 内调 get() — DOM 错位**
- `layoutStore.ts:201-217` — get().applyToDOM() 在 updater 内调用, 写入旧状态

**P1-M3 themeStore 模块级事件监听永不移除**
- `themeStore.ts:130-136` — HMR 下重复注册累积

**P1-M4 noticeStore 把 Set 放入响应式 state**
- `noticeStore.ts:17-18` — 直接 mutate state 中的 Set

### 路由类

**P1-R1 allMenus/menus 刻意不一致** — 需后端 RBAC 保证
- `menuApi.ts:7-18` — allMenus(含隐藏)生成路由, menus(不含隐藏)渲染侧边栏

**P1-R2 静态 detail 路由无权限守卫**
- `DynamicRoutes.tsx:14-16,218-220` — notice/:id, my-notices/:id 无菜单节点

**P1-R3 tokenMeta 仅内存, 刷新后自动刷新定时器失效**
- `SecureTokenStorageImpl.ts:141-176` — isAccessTokenExpiringWithin 永远 false

### Hooks 类

**P1-H1 useWidgetData 无 AbortController**
- `hooks/useWidgetData.ts:68-131` — 卸载后 setState 泄漏

**P1-H2 useRealtimeUpdates 重连竞态 + 闭包旧 enabled**
- `hooks/useRealtimeUpdates.ts:112-128`

---

## 修复优先级

1. P0-2/3/4 hooks 死循环与泄漏 (运行时缺陷, 用户可感知)
2. P0-1 路由权限接入 hasPermission
3. P1-S1/S2/S3 安全类
4. P1-M1/M2/M3/M4 状态管理
5. P1-H1/H2 hooks 资源管理
6. P2 代码质量

---

## 亮点 (值得肯定)

- TypeScript 严格模式 0 错误, any 使用极少
- Token 存储架构正确 (AccessToken 内存, RefreshToken sessionStorage, partialize 不持久化 token)
- SM2+SM4 传输加密链路完整 (timestamp+nonce 防重放, 公钥缓存 generation 防覆盖)
- 懒加载覆盖率约 96%
- 401 刷新队列 single-flight 模式正确
- usePagination/useServerSort/useDeptTree 等 hooks 符合 React Query 规范
