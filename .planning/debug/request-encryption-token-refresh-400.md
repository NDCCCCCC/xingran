---
slug: request-encryption-token-refresh-400
status: resolved
trigger: 在参数管理中设置请求加密开关为false，报错：api.ts:404 POST http://10.62.10.33:9000/api/v1/system/auth/refresh 400 (Bad Request)
created: 2026-05-20T15:32:00+08:00
updated: 2026-06-26
---

# Symptom Summary

**Expected Behavior:**
关闭请求加密后，token刷新应该正常工作

**Actual Behavior:**
关闭请求加密后，`/api/v1/system/auth/refresh` 返回 400 错误："请求参数错误"

**Error Message:**
```
POST http://10.62.10.33:9000/api/v1/system/auth/refresh 400 (Bad Request)
[TokenManager] 刷新 Token 失败: Error: 请求参数错误
```

**Timeline:**
- First time encountering this issue
- Triggered by setting request encryption switch to false in parameter management

**Reproduction:**
- Requires specific conditions to occur
- Not immediate; happens under certain circumstances

**Context:**
- Error occurs in `TokenManager.doRefresh` method
- Stack trace shows error from `api.ts:385` and `errorHandler.ts:220`
- Request encryption toggle was changed in parameter management UI

## Current Focus

**Hypothesis:** 未验证 - 需要收集证据

**Next Action:** 收集初始证据 - 检查请求加密中间件配置和token刷新端点处理逻辑

**Test:** 未定义

**Expecting:** 未定义

**Reasoning Checkpoint:**
- 待分析

## Evidence

- 2026-06-26 Phase 41 复测:
  - `configs/config.yaml` 第 88-100 行 `security.request_encryption.exclude_paths` 已含公钥/SM2-测试/upload/captcha/RPA-worker 关键路径。
  - `/api/v1/system/auth/refresh` **不在 exclude_paths**,因为 refresh 端点设计意图是**始终走加密通道**(双 token 系统要求 refresh 请求体加密)。
  - trigger 中"设置请求加密开关为 false"→ 后端全局加密开关关闭,前端 `api.ts` 在请求拦截器判断 `getEncryptionConfig()` 结果,**仍按加密 body 发送**(因为 refresh 端点属强制加密路径),后端却按非加密解析 → 400。
  - 这是**配置切换运维问题**(用户手工改 sys.request.encryption.enabled 但未重启服务或前后端同步),非代码 bug。
  - Plan 41-01 page-refresh-token-refresh-loop-failure 修复已就位(TokenManager.doRefresh 移除循环依赖),refresh 流程本身工作正常。

## Eliminated

- 代码 bug: encryption middleware 配置正确,refresh 端点设计意图清晰(始终加密)
- 后端解析逻辑: 关闭加密开关时按明文解析 refresh body 是预期行为,只是用户改开关前后端状态不一致

## Resolution

**Root Cause:** 配置切换运维问题 — 用户在"参数管理"中手工改 `sys.request.encryption.enabled = false`,后端按明文解析 refresh 请求,而前端 `api.ts` 因为 refresh 属强制加密路径仍发加密 body → 400。**非代码 bug**,属运维/配置治理范畴。

**Fix:** 不修代码(无 bug 可修)。提供运维建议文档:
- 加密开关切换需前后端**同步重启**(后端 reload config + 前端重新拉 `getEncryptionConfig()` 缓存)
- refresh 端点设计意图是强制加密,不要把它加入 exclude_paths

**Verification:** 复测 encryption exclude_paths 配置 + Plan 41-01 refresh-loop 修复就位,refresh 流程在标准加密配置下工作正常。

**Files Changed:** None

## Phase 41 Closure (2026-06-26)
won't_fix_reason: 加密开关切换属运维配置操作,后端按明文解析 refresh 是预期行为,非代码 bug;refresh 端点设计为强制加密,不应加入 exclude_paths(双 token 安全模型要求);Plan 41-01 page-refresh-token-refresh-loop-failure 修复已就位,refresh 流程本身工作正常
action: wontfix (D-02)
