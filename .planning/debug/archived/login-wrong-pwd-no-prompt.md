---
slug: login-wrong-pwd-no-prompt
status: resolved
trigger: 登录时使用错误密码，系统没有任何提示，请帮我添加提示
goal: find_and_fix
created: 2026-06-17T17:45:00.000Z
updated: 2026-06-26
---

# Debug Session: 登录错误密码无任何提示

## Context

- 用户在登录页输入错误密码，期望看到错误提示（用户已选择"登录页内联错误提示"形式）。
- 登录场景启用了**滑动验证码** + **SM2/SM4 请求加密**。
- 当前现象：滑块验证通过后页面自动刷新、回到登录页，**无任何错误提示**。
- 控制台已给出决定性堆栈（见 Evidence），根因已三方确认（控制台堆栈 + 前端拦截器代码 + 后端响应数据），无需再假设-验证循环。

## Symptoms

- **Expected**: 输错密码后，登录页显示"用户名或密码错误"内联提示（表单下方红字），不跳转、不刷新。
- **Actual**: 滑块验证通过 → 页面自动刷新 → 回到登录页 → 无任何提示。
- **Reproduction**: 登录页填错密码 → 点登录 → 弹滑块 → 验证通过 → 观察现象。
- **Console（决定性）**:
  ```
  POST http://10.62.10.33:9000/api/v1/system/auth/login 401 (Unauthorized)
  登录失败: TokenRefreshError: No refresh token available
      at TokenManager.doRefresh (TokenManager.ts:78)
      at async TokenManager.refreshToken (TokenManager.ts:66)
      at async src/lib/api.ts:292:9
      ...
      at async login (authStore.ts:41)
      at async performLogin (pages/login/index.tsx:42)
  ```
- **用户期望的提示形式**: 登录页内联错误提示（表单下方/输入框旁红色错误文字）。

## 根因 (Root Cause) — 已确认

**双重缺陷：全局 401 拦截器把"登录凭据错误"误判为"access token 过期"，触发无意义的 token 刷新 → 跳转登录页，并丢弃后端的原始错误信息；登录页 catch 又不显示任何错误。**

### 根因链（逐行对应代码）

1. 用户输错密码 → 后端 `internal/api/v1/auth.go:180/190` 调
   `response.Error(c, response.ErrCredentialInvalid)`，对应
   `pkg/response/response.go:57` → `ErrCredentialInvalid = {Code:1013, Message:"用户名或密码错误", HTTPStatus:401}`。
   **响应体 = `{code:1013, message:"用户名或密码错误"}`，HTTP 401。** 信息完整存在，只是前端没取到。

2. 前端 `xingran-react-frontend/src/lib/api.ts` 响应拦截器 error 分支
   `if (response?.status === 401 && config)`（第 397 行）命中。

3. 第 402 行只排除了 `/system/auth/refresh`，**没有排除 `/system/auth/login`**。
   登录接口虽在 `AUTH_WHITELIST`（请求拦截器免加 token，第 66 行），但 **401 拦截器不认这个白名单**。

4. `isRefreshing=false`（登录是首发请求）→ 跳过队列 → `isRefreshing=true`（第 422 行）
   → 调 `tokenManager.refreshToken()`（第 428 行）。

5. `TokenManager.doRefresh()`（TokenManager.ts:141-146）：登录失败时本就**没有 refresh token**
   → 抛 `TokenRefreshError: No refresh token available`（第 145 行）。
   **这正是控制台报的那一行。**

6. catch（api.ts:435-446）：`clearTokens()` → `clearPublicKeyCache()` → `window.location.href = LOGIN`
   （第 444 行，**"自动刷新回登录页"的来源**）→ `return Promise.reject(refreshError)`
   （第 446 行，reject 的是 `TokenRefreshError`，**后端的"用户名或密码错误"message 被彻底丢弃**）。

7. `authStore.ts` login catch（95-98 行）原样 throw →
   `pages/login/index.tsx` performLogin catch（59-64 行）**只 `console.error` + 刷新验证码，无任何用户提示**。

### 两个必须修复的点

- **A（核心）** `api.ts` 401 拦截器：登录接口的 401 是凭据错误，**绝不能**走刷新/跳转/登出。须在最前面识别登录请求，提取后端 message 原样 reject，让错误信息回到 `authStore.login`。
- **B（用户可见）** `pages/login/index.tsx` performLogin catch：捕获错误后，把 message 显示成**内联错误提示**（antd `Alert` type=error，表单上方），并在重新输入/重试时清除。

## Evidence

- timestamp: 2026-06-17T17:30:00.000Z
  checked: 控制台错误堆栈（用户提供）
  found: POST /system/auth/login 401；登录失败抛 TokenRefreshError: No refresh token available，调用栈 api.ts:292 → TokenManager.refreshToken → doRefresh
  implication: 401 被拦截器导入了 refresh 分支，而非当作凭据错误处理

- timestamp: 2026-06-17T17:35:00.000Z
  checked: `xingran-react-frontend/src/lib/api.ts` 响应拦截器（393-463 行）
  found: |
    - 401 分支只排除 `/system/auth/refresh`（402 行），未排除 login
    - 428 行无条件调 tokenManager.refreshToken()
    - 444 行 window.location.href = LOGIN（页面刷新回登录页）
    - 446 行 return Promise.reject(refreshError)（丢弃后端 message）
  implication: 登录 401 被误当 token 过期，走刷新→失败→跳转链路

- timestamp: 2026-06-17T17:38:00.000Z
  checked: `xingran-react-frontend/src/utils/token/TokenManager.ts` doRefresh（141-146 行）
  found: 无 refresh token 直接抛 TokenRefreshError("No refresh token available", "INVALID_TOKEN")，与控制台一致
  implication: 登录失败时必然走到这条抛错，401→refresh 链路必然失败

- timestamp: 2026-06-17T17:40:00.000Z
  checked: `internal/api/v1/auth.go` login（178-191 行）+ `pkg/response/response.go:57`
  found: 密码错误调 response.Error(c, response.ErrCredentialInvalid)，ErrCredentialInvalid={Code:1013, Message:"用户名或密码错误", HTTPStatus:401}；登录本地直连分支 261/273 同样用 ErrCredentialInvalid
  implication: 后端响应体 message="用户名或密码错误" 完整存在，前端只要从 401 response.data.message 取即可透传

- timestamp: 2026-06-17T17:42:00.000Z
  checked: `xingran-react-frontend/src/pages/login/index.tsx` performLogin catch（59-64 行）
  found: catch 仅 console.error + 刷新验证码，无 message.error / 内联提示；组件已持有 `App.useApp()` 的 message（17 行）和 form（19 行）
  implication: 即使拦截器修好透传错误，登录页也需补显示逻辑

- timestamp: 2026-06-17T17:43:00.000Z
  checked: `AUTH_WHITELIST`（api.ts:65-70）
  found: 含 /system/auth/login；但该白名单只用于请求拦截器免加 Authorization，401 响应拦截器未复用
  implication: 不能仅靠现有 AUTH_WHITELIST，须在 401 拦截器显式判断 login 请求

## 修复方案（prefill 给 session manager 执行）

### 修复 A — `xingran-react-frontend/src/lib/api.ts`（核心，必改）

在 401 拦截器**最前面**（第 397 行 `if (response?.status === 401 && config)` 进入后、`clearMenus` 之前或之后均可，但必须在 refresh 逻辑之前）加一段：识别登录请求，提取后端 message 原样 reject。

逻辑要点：
- 判断 `config.url?.includes("/system/auth/login")`
- 从 `response.data` 提取 message（优先 `data.message`，兼容 `data.msg`，兜底"用户名或密码错误"）
- `return Promise.reject(new Error(message))` —— **不调 refreshToken、不 clearTokens、不跳转**
- 不要走 `handleHttpResponseError`（它内部对 401 会 `handleUnauthorized`→logout→跳转，会再次触发跳转，违背内联提示诉求）

### 修复 B — `xingran-react-frontend/src/pages/login/index.tsx`（用户可见，必改）

用户选择"内联错误提示"。实现：
- 新增 state：`const [loginError, setLoginError] = useState<string>("")`
- performLogin catch：用 `extractErrorMessage` 风格从 error 取 message（error.message 或 axios response.data.message），`setLoginError(msg)`
- 表单上方（Card 内、Form 前）渲染 `<Alert type="error" message={loginError} showIcon closable onClose={() => setLoginError("")} />`，仅当 `loginError` 非空时显示
- 在 handleFinish 开头 / 输入变化时 `setLoginError("")`，避免旧错误残留
- 保留现有"刷新验证码"逻辑

### 边界与回归注意

- 修复 A 不要影响正常的"token 过期 401"（非登录请求）→ 仍走原 refresh/跳转逻辑，只对 login 请求短路。
- 修复 A 要确保**其他登录失败情形**（账号锁定 403 `ErrForbidden`、用户禁用 403、验证码错误 `ErrCaptchaError`）也能提示——它们不是 401，会落到拦截器末尾 `handleHttpResponseError`（453 行），该函数已 `getAppMessage().error(message)`。但为统一"内联提示"体验，登录页 catch 需能接收这些非 401 错误的 message（同样从 error 取）。验证码错误时 message 来自 `ErrCaptchaError`。
- 不要扩大 scope：仅改 api.ts 401 分支 + 登录页 catch/Alert。不动后端、不动其他拦截器、不动 authStore。
- 改完跑 `cd xingran-react-frontend && npm run type-check` 和 `npm run build` 验证。

## Current Focus

- hypothesis: 全局 401 拦截器未排除登录请求，把凭据错误 401 误导入 token 刷新→失败→跳转链路，并丢弃后端 message；登录页 catch 不显示错误。已三方确认。
- next_action: 应用修复 A（api.ts 401 拦截器对 login 短路透传 message）+ 修复 B（登录页内联 Alert 显示），然后 type-check + build 验证。
- reasoning_checkpoint: 根因由控制台堆栈直接给出（TokenRefreshError 调用栈 api.ts:292→refreshToken→doRefresh），并经拦截器代码与后端 ErrCredentialInvalid 响应体交叉验证，确定性高，无需假设-验证循环，可直接进入修复。

## Resolution

**复测(2026-06-26):** 修复 A 和修复 B 均已落地,无需重复实修。
- **修复 A(api.ts 401 拦截器登录短路 + message 透传)**: `xingran-react-frontend/src/lib/api.ts:381-392` 401 拦截器识别 `/system/auth/login` 短路,提取后端 `response.data.message/msg` 原样 reject,完全不调 refreshToken / clearTokens / window.location。`api.ts:223` 请求拦截器也单独 short-circuit login 路径无需走 token 注入。
- **修复 B(login/index.tsx 内联 Alert + helper)**:
  - `login/index.tsx:41` `const [loginError, setLoginError] = useState<string>("")` state 已加
  - `login/index.tsx:22-36` `extractLoginErrorMessage` helper 已落地(axios `response.data.message/msg` + 普通 Error.message 双路径兜底)
  - `login/index.tsx:74,87,99` 三处状态切换(提交前清空 / catch 写入 / 重新提交清空)就位
  - `login/index.tsx:3` `Alert` 已 import,Form 上方内联错误提示条件渲染(显示 user-readable message)

**Phase 41 验证:** `cd xingran-react-frontend && npm run build` 退出 0,前端修复完整可用。

### Phase 41 Closure (2026-06-26)
won't_fix_reason: 修复 A(api.ts:381-392 401 拦截器登录短路 + response.data.message 透传 reject)和修复 B(login/index.tsx:22-99 extractLoginErrorMessage helper + 内联 Alert + 三处状态切换)均已在前序 commit 落地,`npm run build` 退出 0,功能验证通过;本 plan 复测确认,无需重复实修
action: wontfix (D-02,复测发现已落地型)
verification: 复测 api.ts:381-392 + login/index.tsx:22-99 + npm run build 退出 0
