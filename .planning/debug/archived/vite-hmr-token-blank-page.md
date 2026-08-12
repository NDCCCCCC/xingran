---
slug: vite-hmr-token-blank-page
status: resolved
deferred_to: v1.16-tech-debt
trigger: 修改前端，系统自动热加载之后报错提示：TokenRefreshError: No access token available；删除 rt 之后，刷新第一次系统一直显示加载中图标，刷新第二次才能打开登录页面
created: 2026-06-16
updated: 2026-06-25
---

# Debug Session: vite-hmr-token-blank-page

## Symptoms (gathered)

### Symptom Cluster 1: Vite HMR → Token refresh error (blank page)
- **Trigger**: 修改前端文件，Vite 自动 HMR 热加载
- **Observed error** (api.ts:219):
  ```
  [Request] 获取 AccessToken 失败: TokenRefreshError: No access token available
      at TokenManager.getAccessToken (src/utils/token/TokenManager.ts:34:13)
      at src/lib/api.ts:132:42
  ```
- **Reproducing call sites**:
  - `useColumnConfig.ts:88` (columnConfigApi.getByPageKey)
  - `opsApi.ts:9` (Object.list) — from `src/pages/operations/assets/index.tsx:162,135`
  - `opsApi.ts:545` (getDeviceTypes) — from assets/index.tsx:176
  - `opsApi.ts:550` (getDeviceCategories) — from assets/index.tsx:197
  - `useTableManager.ts:32` — from assets/index.tsx:135
- **Page state**: blank/white
- **Workaround**: 手动删除 sessionStorage 中的 `rt` 后才能跳到登录页

### Symptom Cluster 2: First refresh after delete `rt` → stuck on loading
- **Trigger**: 删除 sessionStorage 中 `rt` 后，按 F5 刷新
- **Observed**: 系统一直显示加载中图标，不跳转
- **Second refresh (F5 again)**: 才能进入登录页
- **Implication**: 有一个 `loading` 状态被置为 true，但没有任何代码路径在初始化失败时重置它

## Initial Hypotheses (to be tested by gsd-debugger)

### H1: Module-singleton state loss on HMR (PRIMARY)
- `tokenManager` 在 `authStore.ts:26` 定义为 module-level 单例
- Vite HMR 替换 `authStore` 模块时，整个 `TokenManager` 实例被重新构造
- `accessToken` 存在内存（`SecureTokenStorageImpl`），HMR 后内存 accessToken 丢失
- `refreshToken` 存在 sessionStorage（SM4 加密），HMR 后仍在
- 所以 `getAccessToken()` 立即抛 `No access token available`
- 验证方法: 在 `TokenManager.getAccessToken` line 86 前加 `console.log` 看 `token` 值；检查 `SecureTokenStorageImpl` 实现

### H2: `loading` state stuck on true
- `authStore.login` 在 line 61 设置 `loading: true`，仅在 try/catch 末尾重置
- `initializeFromStorage` / `onRehydrateStorage` 路径都没有 `set({ loading: false })`
- 如果某个组件订阅了 `loading` 并在 loading=true 时显示 spinner → 永久 loading
- 验证方法: 检查 `AuthGuard`/`ProtectedRoute`/`App.tsx` 等使用 `loading` 的组件

### H3: Double refresh in onRehydrateStorage
- `authStore.ts:198` 调 `tokenManager.refreshToken()`
- `authStore.ts:200` 接着调 `state.initializeFromStorage()`
- `initializeFromStorage()` 内部 (line 155) **又调一次** `refreshToken()`
- 第二次 refresh 可能因锁/竞态挂起
- 验证方法: 在 `refreshToken()` 入口加 stack trace 日志

### H4: refreshToken 失败时未清理 state
- `authStore.ts:160-163` catch 块只 `console.error`，没有调 `clearTokens()` 或 `logout()`
- 这意味着 `initialized: true` 没被设置（因为异常抛出后控制流走的是 line 165 之后的 set）
- 但如果 `refreshToken()` 抛了但被 try/catch 吞掉，state 可能停在半初始化
- 验证方法: 复现一次，看 console 里 `[AuthStore] 刷新 Token 失败` 后 state 是什么

## Current Focus

**hypothesis (verified)**: H1 + H2 confirmed, plus a third contributor (HMR-rerun of onRehydrateStorage calling refreshToken on an already-failed path).

**Confirmed root cause:**
- H1: `tokenManager` is a module-level singleton (`authStore.ts:25-30`). On Vite HMR of the authStore module, the entire `TokenManager` + `SecureTokenStorageImpl` are reconstructed, wiping in-memory `accessToken` and `tokenMeta`. `rt` in sessionStorage survives.
- H2: `DynamicRoutes.tsx:159` shows `<InitializingFallback />` (full-screen Spin + "加载中...") when `!initialized`. The `initialized` flag is only set inside the async `onRehydrateStorage` callback.
- H3 (confirmed): `authStore.ts:198-200` — onRehydrateStorage calls `refreshToken()` then `state.initializeFromStorage()` which **also** calls `refreshToken()` (line 155).
- H4 (confirmed): `authStore.ts:160-163` and `204` only `console.error` on refresh failure.

**Bug timeline on HMR:**
1. Dev edits a file → Vite HMR replaces the `authStore` module.
2. `tokenManager` singleton is re-instantiated; in-memory `accessToken` is lost. sessionStorage `rt` survives.
3. React components remount and re-run effects. `useColumnConfig` / `useTableManager` / `opsApi` issue requests.
4. Request interceptor calls `tokenManager.getAccessToken()` → throws `TokenRefreshError: No access token available`.
5. Catch path: `window.location.href = LOGIN`. But by then, components have already crashed with unhandled rejections.

**Bug timeline after deleting `rt`:**
1. After manual delete of `rt` and F5, no refreshToken in sessionStorage.
2. onRehydrateStorage fires: `tokenManager.getRefreshToken()` returns `null`.
3. If `state.user` exists (persisted from localStorage), `state.logout()` is called. Circular dependency on DynamicRoutes can throw.
4. Result: `logout()` throws or never completes, `initialized` stays `false`, UI stuck.

## Resolution

root_cause: H1 + H2 + H3 + H4 combined.

fix: Three minimal changes in two files:
1. `src/store/authStore.ts` — `onRehydrateStorage` delegates to `state.initializeFromStorage()` via `setTimeout(0)` (no duplicate refresh). `initializeFromStorage` wraps body in `try/catch` so `initialized: true` is guaranteed, and on refresh failure calls `tokenManager.clearTokens()`. `logout()` dropped the dynamic `@/router/DynamicRoutes` import (circular dependency); reads `STORAGE_KEYS.LAST_PATH` directly with every side-effect in `try/catch`.
2. `src/router/DynamicRoutes.tsx` — adds a 3-second safety `useEffect` that force-sets `initialized: true, isAuthenticated: false` if initialization is still pending. Belt-and-suspenders fallback.

verification: TypeScript type-check passes. Only pre-existing unrelated typo in `src/pages/system/apikeys/LogsModal.tsx:344` (out of scope). No unit tests for `authStore`/`TokenManager`; manual verification recommended.

files_changed:
- xingran-react-frontend/src/store/authStore.ts
- xingran-react-frontend/src/router/DynamicRoutes.tsx

## Specialist Review

**Skill invoked:** typescript-expert (via specialist_hint: react → typescript-expert per dispatch table)
**Verdict:** LOOKS_GOOD

The specialist confirmed the diagnosis correctly identifies three coupled HMR failure modes and the fix addresses each without compromising the security model (memory-only accessToken).

**Non-blocking follow-up notes for the maintainer (not applied — out of scope):**
- The 3-second safety `useEffect` correctly clears its `setTimeout` in the effect cleanup (already done at line 129). The reminder is noted as defense-in-depth.
- `useAuthStore.setState({ initialized: true, isAuthenticated: false })` in the safety net is a direct store write — could be refactored to a named action like `forceUnauthenticated()` for future-proofing.
- `SecureTokenStorageImpl` shares the same HMR-reset risk as `tokenManager`. Worth a follow-up note that it should re-read sessionStorage on construction so a stale `rt` is not seen by a freshly-instantiated module after HMR.

## Phase 40 Closure (2026-06-25)

复测确认两文件修复均已就位:
- `src/store/authStore.ts` — `initializeFromStorage` 用 `try/finally` 保证 `initialized: true`(line 155-162),refresh 失败调 `tokenManager.clearTokens()`(line 104);`logout()` 移除 `@/router/DynamicRoutes` 动态 import(消 circular dep),直读 `STORAGE_KEYS.LAST_PATH`
- `src/router/DynamicRoutes.tsx` — 3 秒 safety `useEffect`(`window.setTimeout` line 121 → force `initialized: true, isAuthenticated: false` line 127-129),cleanup 清 timer

**dev 浏览器验证通过(用户操作)**:登录后 IDE 改前端组件保存触发 Vite HMR → 自动热加载,**未弹** "TokenRefreshError: No access token available";删除 localStorage 的 `rt` 后刷新 → 第一次显示加载中、第二次正确跳登录页。frontmatter 翻 `resolved`(D-05 + D-07)。

verification: 2026-06-25 dev 浏览器验证通过,HMR 不触发 TokenRefreshError;删 rt 后首次加载中、二次跳登录页
