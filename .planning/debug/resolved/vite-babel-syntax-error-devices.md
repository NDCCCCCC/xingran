---
slug: vite-babel-syntax-error-devices
status: resolved
trigger: Vite React-Babel syntax error in devices page - Unexpected token at line 409
created: 2026-05-06T00:00:00Z
updated: 2026-05-06T00:00:00Z
---

# Debug Session: Vite Babel Syntax Error

## Symptoms

### Expected Behavior
页面应该能够正常打开，前端编译通过。

### Actual Behavior
Vite 编译时出现 Babel 语法错误：
```
[plugin:vite:react-babel] D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\pages\network\devices\index.tsx: Unexpected token, expected "," (409:2)
```

### Error Message
```
D:/CODE/ClaudeCode/xingran-go-backend/xingran-react-frontend/src/pages/network/devices/index.tsx:409:2
418 |          },
419 |          body: JSON.stringify({
420 |            entityTypes,
    |                      ^
```

### Timeline
- 刚修改代码后出现

### Reproduction
运行 `npm run dev` 触发编译错误。

## Current Focus

### Hypothesis
`handleBatchDelete` 函数在第 407 行未正确闭合，导致 `handleBatchExport` 函数被错误地嵌套在 try 块中，造成语法错误。

### Root Cause (Confirmed)
1. **主要问题**：多个文件中 `handleBatchDelete` 函数未正确闭合，导致 `handleBatchExport` 函数被错误嵌套
2. **次要问题**：正则表达式跨行导致语法错误
3. **新发现**：多个文件缺少批量导出功能所需的导入和状态定义

### Fix Applied
1. **修复了以下文件中的函数结构问题**：
   - `devices/index.tsx` - ✅ 完成
   - `credentials/index.tsx` - ✅ 完成
   - `mac/index.tsx` - ✅ 完成
   - `ports/index.tsx` - ✅ 完成

2. **修复了以下文件的批量导出功能**：
   - `backups/index.tsx` - ✅ 完成（删除重复导入、添加 handleBatchExport）
   - `command/index.tsx` - ✅ 完成（添加导入、状态、函数）
   - `discoveries/index.tsx` - ✅ 完成（添加导入、状态、函数）
   - `executions/index.tsx` - ✅ 完成（添加导入、状态、函数）
   - `templates/index.tsx` - ✅ 完成（添加导入、状态、函数）

3. **修复了正则表达式跨行问题**（所有涉及批量导出的文件）

4. **修复了 statistics 状态定义混乱问题**（mac/index.tsx, ports/index.tsx）

### Verification
- ✅ 代码编译通过
- ✅ Vite build 成功
- ✅ 无 TypeScript 错误

### Files Changed
- `xingran-react-frontend/src/pages/network/devices/index.tsx`
- `xingran-react-frontend/src/pages/network/credentials/index.tsx`
- `xingran-react-frontend/src/pages/network/mac/index.tsx`
- `xingran-react-frontend/src/pages/network/ports/index.tsx`
- `xingran-react-frontend/src/pages/network/backups/index.tsx`
- `xingran-react-frontend/src/pages/network/command/index.tsx`
- `xingran-react-frontend/src/pages/network/discoveries/index.tsx`
- `xingran-react-frontend/src/pages/network/executions/index.tsx`
- `xingran-react-frontend/src/pages/network/templates/index.tsx`

总计修复了 **9 个文件**的语法错误和缺失的批量导出功能。

## Evidence

- timestamp: 2026-05-06T00:00:00Z
  source: code_analysis
  detail: |
    第 402-461 行代码结构分析显示：
    - handleBatchDelete 从第 402 行开始
    - 第 407 行 post 调用未闭合：`await post('/network/devices/batch-delete', { ids: selectedRowKeys }`
    - 第 409 行直接定义 handleBatchExport，形成非法嵌套
    - 第 450-461 行包含 handleBatchDelete 应有的后续代码

## Eliminated

No eliminated hypotheses yet.

## Resolution

### Files Changed
- `xingran-react-frontend/src/pages/network/devices/index.tsx`

### Verification
- [ ] 代码编译通过
- [ ] 页面正常打开
- [ ] handleBatchDelete 功能正常
- [ ] handleBatchExport 功能正常
