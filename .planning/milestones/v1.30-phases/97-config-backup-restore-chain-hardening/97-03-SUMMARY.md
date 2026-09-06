# Plan 97-03 Summary: V130R-03 业务错误码语义化

**Status:** ✅ COMPLETED
**Date:** 2026-09-06
**Plans:** 97-03

## Changes Made

### 1. pkg/response/business_error.go — 新建 BusinessError 类型
- `BusinessError{HTTPStatus, Code, Message}` + `Error()` method
- 任何 service 层可复用

### 2. pkg/response/handler_helpers.go — HandleServiceError 识别 BusinessError
- 识别 `*BusinessError` → 直接 `c.JSON(httpStatus, Response{Code: httpStatus, Data: {bizCode}})`
- 避免 `Error()` -> `toAppError` 把 int 当 error code 的问题

### 3. internal/services/config_restore_task_service.go — 三处错误返回 BusinessError
- 跨设备错误 → `&BusinessError{HTTPStatus:400, Code:400001}`
- 活跃任务冲突（两处）→ `&BusinessError{HTTPStatus:409, Code:409001}`

### 4. pkg/response/response_test.go — 新增 BusinessError 测试
- `TestHandleServiceErrorWithBusinessError`: 400 和 409 场景

### 5. internal/services/config_restore_task_service_97_03_test.go — 回归测试
- `TestV130R03_DeviceMismatchReturns400`
- `TestV130R03_DuplicateRestoreReturns409`
- `TestV130R03_PlainErrorNotBusinessError`

## Verification

| Check | Result |
|-------|--------|
| `go build ./...` | ✅ PASS |
| `TestHandleServiceErrorWithBusinessError` (pkg/response) | ✅ PASS |
| `TestV130R03_DeviceMismatchReturns400` | ✅ PASS |
| `TestV130R03_DuplicateRestoreReturns409` | ✅ PASS |
| `TestV130R03_PlainErrorNotBusinessError` | ✅ PASS |
| Phase 93 `TestCbk93RestoreCrossDeviceRejected` | ✅ PASS |
| Phase 93 `TestCbk93RestoreMutualExclusion` | ✅ PASS |

## Truths Verified

1. ✅ 跨设备恢复 → HTTP 400，业务码 400001
2. ✅ 活跃任务冲突 → HTTP 409，业务码 409001
3. ✅ 普通 error 不走 BusinessError 路径
