---
slug: ou-page-getoudeptmapping-impor
name: ou-page-getoudeptmapping-impor
description: OU组织管理页面空白 - getOUDeptMapping 导出不存在
status: resolved
trigger: OU组织管理页面空白，前端报错：index.tsx:28 Uncaught SyntaxError: The requested module '/src/lib/adDomainApi.ts?t=1779854188775' does not provide an export named 'getOUDeptMapping' (at index.tsx:28:3)
created: 2026-05-27
updated: 2026-05-27
type: bug
---

## Symptoms

### Expected Behavior
OU 组织管理页面应该正常加载和显示

### Actual Behavior
页面空白，浏览器控制台报错

### Error Messages
```
index.tsx:28 Uncaught SyntaxError: The requested module '/src/lib/adDomainApi.ts?t=1779854188775' does not provide an export named 'getOUDeptMapping' (at index.tsx:28:3)
```

### Timeline
- 之前能正常打开
- 页面加载时就报错

### Reproduction
1. 打开 OU 组织管理页面
2. 页面加载失败

### Environment
- Frontend: React 19.2, TypeScript 5.9, Vite 7.2
- Error location: `index.tsx:28` importing from `/src/lib/adDomainApi.ts`

## Current Focus

**Hypothesis:** adDomainApi.ts 缺少 OU 部门映射相关函数的导出

**Next Action:** 在 adDomainApi.ts 中添加缺失的函数

**Test:** 导航到 OU 组织管理页面，确认页面正常加载

**Expecting:** 页面正常显示，无控制台错误

**Reasoning Checkpoint:** 已确认
- 后端 API 已存在: GET/POST `/system/ou/:ouDn/dept-mapping`
- 前端页面使用但 adDomainApi.ts 未导出
- commit 3b9d733 添加了部门-组映射但未包含 OU-部门映射

**TDD Checkpoint:** 未使用

## Evidence
- timestamp: 2026-05-27
  evidence: |
    错误发生在 index.tsx:28，尝试导入 getOUDeptMapping 但该导出在 adDomainApi.ts 中不存在
    错误类型：SyntaxError - 模块导出缺失
    影响：OU 组织管理页面完全无法加载

- timestamp: 2026-05-27
  evidence: |
    后端已实现完整的 OU 部门映射 API:
    - internal/api/v1/system/ou_mapping_router.go
    - internal/api/v1/system/ou_mapping_handler.go
    - GET /ou/:ouDn/dept-mapping 返回 DeptMappingResponse
    - POST /ou/:ouDn/dept-mapping 接受 {deptId: string}

- timestamp: 2026-05-27
  evidence: |
    前端页面使用:
    - getOUDeptMapping(ouDn) - 获取映射
    - updateOUDeptMapping(ouDn, {deptId}) - 更新映射
    - OUDeptMappingResponse 类型

## Eliminated
- 假设: 路由未注册 -> 已排除，路由已存在
- 假设: 后端未实现 -> 已排除，handler.go 已实现

## Files Under Investigation
- `xingran-react-frontend/src/pages/ad-domain/ous/index.tsx` (line 28)
- `xingran-react-frontend/src/lib/adDomainApi.ts`
- `internal/api/v1/system/ou_mapping_handler.go`

## Resolution

**Root Cause:** 前端 adDomainApi.ts 缺少 OU 部门映射相关的函数导出，后端路由也未注册

**Fix Applied:**
1. ✅ 后端路由注册：在 `internal/api/router.go` 中添加 `SetupOUMappingRouter` 调用
2. ✅ 前端 API 函数：在 `adDomainApi.ts` 中添加：
   - `OUDeptMappingResponse` 类型定义
   - `getOUDeptMapping(ouDn: string)` 函数
   - `updateOUDeptMapping(ouDn: string, data: {deptId: string})` 函数

**Verification:** 待用户验证
- 重启后端服务以加载新路由
- 刷新前端页面，确认 OU 组织管理页面正常加载
- 测试部门映射功能是否正常工作

**Files Changed:**
- ✅ `internal/api/router.go` - 添加 OU 映射路由注册
- ✅ `xingran-react-frontend/src/lib/adDomainApi.ts` - 添加 OU 映射 API 函数和类型

**Additional Findings:**
- commit 3b9d733 添加了部门-组映射但未包含 OU-部门映射
- 后端 handler (`ou_mapping_handler.go`) 已完整实现，只是路由未注册
- 这是一个典型的"后端已实现但前端 API 缺失"的集成问题

## Specialist Hint
TypeScript 模块导出问题 - 已修复，需要重启服务验证
