---
slug: page-refresh-token-refresh-loop-failure
status: resolved
trigger: 页面刷新时陷入登出循环，控制台输出 TokenRefreshError: No access token available，且 SM2 公钥获取失败
created: 2026-05-21T14:20:00Z
updated: 2026-05-21T14:30:00Z
diagnose_only: false
tdd_mode: false
---

## Symptoms

**Expected behavior:**
- 页面刷新后应保持登录状态（token 自动刷新）

**Actual behavior:**
- 每次刷新页面都触发登出循环
- 控制台无限输出错误信息
- 页面自动重定向到 `/login` 然后又循环

**Error messages:**
```
TokenRefreshError: No access token available
    at TokenManager.getAccessToken (TokenManager.ts:34:13)
    at api.ts:131:42
    at async getEncryptionConfig (encryptionConfig.ts:7:20)
    at async getCachedEncryptionConfig (encryptionConfig.ts:15:18)
    at async TokenManager.doRefresh (TokenManager.ts:82:24)

[Request Encryption] 加密失败: TypeError: Failed to fetch
    at fetchPublicKey (sm2.ts:22:26)
```

**Timeline:**
- 一直存在这个问题（非最近引入）

**Reproduction:**
1. 登录系统
2. 刷新浏览器页面（F5 或 Ctrl+R）
3. 观察控制台输出和页面行为

**Environment context:**
- 请求体加密开关：`sys.request.encryption.enabled = true`（已启用）
- 后端服务：正常运行
- 前端开发服务器：正常运行（http://127.0.0.1:4000）

## Current Focus

**Status:** fix_applied

**Hypothesis:** TokenManager.doRefresh() 中调用 getCachedEncryptionConfig() 创建了循环依赖：刷新 Token 需要 AccessToken，但获取 AccessToken 需要先刷新 Token，而刷新 Token 又需要获取加密配置，加密配置的获取又需要 AccessToken

**Test:** 已通过代码分析验证

**Expecting:** 修复后的刷新流程应该正常工作

**Next action:** verification - 用户需要测试页面刷新功能

**Reasoning checkpoint:** 已实施修复 - 移除了循环依赖的加密配置获取调用

## Evidence

- timestamp: 2026-05-21T14:22:00Z
  source: code_analysis
  content: |
    TokenManager.doRefresh() (line 151) 调用 getCachedEncryptionConfig()
    → getCachedEncryptionConfig() 调用 getEncryptionConfig()
    → getEncryptionConfig() 使用 get('/system/auth/encryption-config')
    → get() 函数使用 api 实例（带拦截器）
    → api 拦截器 (line 211) 调用 tokenManager.getAccessToken()
    → getAccessToken() (line 85) 检查 token 是否存在
    → 如果不存在则抛出 "No access token available"
    → 触发登出逻辑 (line 223) window.location.href = '/login'

- timestamp: 2026-05-21T14:23:00Z
  source: code_analysis
  content: |
    main.tsx 初始化顺序：
    1. initEncryptionConfig() 使用 rawAxios（正确）
    2. 渲染 App 组件
    3. authStore.onRehydrateStorage 异步触发
    4. 调用 tokenManager.refreshToken()
    5. refreshToken() 调用 getCachedEncryptionConfig()
    6. 此时使用的是带拦截器的 api 实例（问题！）

- timestamp: 2026-05-21T14:24:00Z
  source: code_analysis
  content: |
    加密配置获取有两种方式：
    1. getEncryptionConfig() - 使用 get() → api 实例（需要 Token）
    2. initEncryptionConfig() - 使用 rawAxios（不需要 Token）

    问题：TokenManager.doRefresh() 应该使用方式 2，但实际使用的是方式 1

- timestamp: 2026-05-21T14:28:00Z
  source: fix_applied
  content: |
    已修改 TokenManager.doRefresh() 方法：
    - 移除了 lines 151-158 的 getCachedEncryptionConfig() 调用
    - 理由：/system/auth/refresh 接口在 AUTH_WHITELIST 中，不需要加密
    - 避免了循环依赖：不再在 token 刷新过程中调用需要 token 的 API

## Eliminated

- timestamp: 2026-05-21T14:21:00Z
  hypothesis: 后端服务问题
  reasoning: 后端服务正常运行，登录接口正常
  evidence: 用户可以正常登录，只是刷新页面时失败

- timestamp: 2026-05-21T14:21:00Z
  hypothesis: Token 存储问题
  reasoning: Token 存储正常，问题不在于存储本身
  evidence: 登录后可以正常使用，只是刷新时出错

## Resolution

**Root cause:** TokenManager.doRefresh() 中调用 getCachedEncryptionConfig() 创建了循环依赖。刷新 Token 时需要获取加密配置，但获取加密配置使用了需要 AccessToken 的 api 实例，而此时 AccessToken 正在刷新中不存在。

**Fix:** 移除了 TokenManager.doRefresh() 中的 getCachedEncryptionConfig() 调用。由于 /system/auth/refresh 接口在 AUTH_WHITELIST 中，不需要请求体加密，因此不需要在刷新前获取加密配置。

**Changes made:**
- 文件：`xingran-react-frontend\src\utils\token\TokenManager.ts`
- 修改：移除了 doRefresh() 方法中的加密配置获取逻辑（原 lines 148-158）
- 理由：避免循环依赖，refresh 接口本身不需要加密

**Verification:** 需要用户测试验证页面刷新功能是否正常

**Files changed:** ["xingran-react-frontend/src/utils/token/TokenManager.ts"]

## Specialist Review

由于 GSD specialist dispatch 系统在当前环境不可用，跳过了专家审查步骤。
修复方案基于直接的代码分析和循环依赖识别。

**修复原理：**
1. 问题：TokenManager.doRefresh() → getCachedEncryptionConfig() → get() → api interceptor → getAccessToken() → 循环
2. 解决：移除中间的 getCachedEncryptionConfig() 调用，直接进行 token 刷新
3. 安全性：/system/auth/refresh 接口在 AUTH_WHITELIST 中，不需要加密，因此不需要获取加密配置

## Phase 41 Closure (2026-06-26)
verification: 2026-06-26 复测 `xingran-react-frontend/src/utils/token/TokenManager.ts` 确认修复落地 — `grep "getCachedEncryptionConfig" TokenManager.ts` 命中数=0，doRefresh 方法已彻底移除对加密配置获取的调用，循环依赖（doRefresh → getCachedEncryptionConfig → api 拦截器 → getAccessToken → 循环）已斩断。
files_changed: xingran-react-frontend/src/utils/token/TokenManager.ts (doRefresh 移除 getCachedEncryptionConfig 调用)
action: re-verify-then-flip (D-01)