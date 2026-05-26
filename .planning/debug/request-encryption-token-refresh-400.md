---
status: investigating
trigger: 在参数管理中设置请求加密开关为false，报错：api.ts:404 POST http://10.62.10.33:9000/api/v1/system/auth/refresh 400 (Bad Request)
created: 2026-05-20T15:32:00+08:00
updated: 2026-05-20T15:32:00+08:00
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

## Eliminated

## Resolution

**Root Cause:** TBD

**Fix:** TBD

**Verification:** TBD

**Files Changed:** TBD
