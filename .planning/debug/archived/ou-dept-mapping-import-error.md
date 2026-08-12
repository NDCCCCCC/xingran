---
slug: ou-dept-mapping-import-error
status: resolved
trigger: Phase 23 执行后，点击 OU 组织单位菜单提示 getOUDeptMapping 导入错误
created: 2026-05-26
updated: 2026-05-26
resolved: 2026-05-26
---

## Symptoms

**Expected Behavior:
点击 OU 组织单位菜单后应该正常显示相关页面内容

**Actual Behavior:**
页面空白，控制台显示导入错误：
```
index.tsx:28 Uncaught SyntaxError: The requested module '/src/lib/adDomainApi.ts?t=1779772546200' does not provide an export named 'getOUDeptMapping' (at index.tsx:28:3)
```

**Error Messages:**
```javascript
Uncaught SyntaxError: The requested module '/src/lib/adDomainApi.ts?t=1779772546200' does not provide an export named 'getOUDeptMapping' (at index.tsx:28:3)
```

**Timeline:**
Phase 23 执行后开始出现（最近的代码更改）

**Reproduction:**
1. 登录系统
2. 点击左侧菜单的"OU 组织单位"
3. 页面空白，控制台报错

## Current Focus

**Hypothesis:**
`adDomainApi.ts` 文件中缺少 `getOUDeptMapping` 函数的导出，或者函数名不匹配

**Next Action:**
gather initial evidence - 检查 `adDomainApi.ts` 文件和调用该函数的 `index.tsx` 文件

**Test:**
检查 `adDomainApi.ts` 是否有 `getOUDeptMapping` 导出

**Expecting:**
确认是否缺少导出或函数名不匹配

## Evidence

- timestamp: 2026-05-26T10:30:00Z
  - Source: Frontend file analysis
  - Finding: `xingran-react-frontend/src/lib/adDomainApi.ts` does NOT export `getOUDeptMapping` function
  - Impact: Frontend component tries to import non-existent function

- timestamp: 2026-05-26T10:31:00Z
  - Source: Usage analysis
  - Finding: `xingran-react-frontend/src/pages/ad-domain/ous/index.tsx` imports `getOUDeptMapping` from `@/lib/adDomainApi`
  - Code: `import { getOUDeptMapping, updateOUDeptMapping, type OUDeptMappingResponse } from '@/lib/adDomainApi';`

- timestamp: 2026-05-26T10:32:00Z
  - Source: Backend handler analysis
  - Finding: Backend handler `OUMappingHandler` exists in `internal/api/v1/system/ou_mapping_handler.go`
  - Routes: `GET /ou/:ouDn/dept-mapping` and `POST /ou/:ouDn/dept-mapping`
  - Router setup function: `SetupOUMappingRouter()` exists but is NOT called in main router

## Eliminated

- ~~Function name mismatch in backend~~ - Backend handler exists with correct methods
- ~~Missing backend implementation~~ - Backend logic is implemented
- ~~Type definitions missing~~ - Types can be added to adDomainApi.ts

## Resolution

**Root Cause:**
Two issues found:
1. **Frontend**: `adDomainApi.ts` is missing the `getOUDeptMapping`, `updateOUDeptMapping` functions and `OUDeptMappingResponse` type
2. **Backend**: `SetupOUMappingRouter` is not registered in `internal/api/router.go`, so the API endpoints are not accessible

**Fix Applied:**
1. ✅ Added `OUDeptMappingResponse` type and `getOUDeptMapping`, `updateOUDeptMapping` functions to `xingran-react-frontend/src/lib/adDomainApi.ts`
2. ✅ Registered `SetupOUMappingRouter` in `internal/api/router.go` (line 479)

**Verification:**
- ✅ Frontend type-check passed (`npm run type-check`)
- ✅ Backend compilation passed (`go build ./...`)
- ✅ No import errors in adDomainApi.ts

**Files Changed:**
- `xingran-react-frontend/src/lib/adDomainApi.ts` (added OUDeptMappingResponse type and functions)
- `internal/api/router.go` (registered SetupOUMappingRouter)