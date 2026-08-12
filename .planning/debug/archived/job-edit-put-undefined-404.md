---
slug: job-edit-put-undefined-404
status: resolved
trigger: 修复任务编辑功能bug：job ID 在 PUT 请求中变成 undefined，导致 404 错误
created: "2026-06-05T15:56:00Z"
updated: "2026-06-05T16:15:00Z"
tdd_mode: false
---

# Debug Session: job-edit-put-undefined-404

## Symptoms

### Expected Behavior
编辑任务（job）时应该正确传递 job ID 到 PUT 请求

### Actual Behavior
PUT 请求 URL 变成 `/api/v1/monitor/jobs/undefined`，返回 404 错误

### Error Messages
```
PUT http://10.62.10.33:9000/api/v1/monitor/jobs/undefined 404 (Not Found)
提交失败: Error: SPA routes are handled by Vite dev server in development mode
```

### Timeline
- 发生时间: 2026-06-05 15:56:00
- 位置: useJobActions.ts:86 → index.tsx:197 onOk

### Reproduction
1. 打开任务编辑表单
2. 修改任务内容
3. 点击确定提交
4. 观察 PUT 请求 URL

## Current Focus

### Hypothesis
表单缺少 id 字段的 Form.Item，导致 form.getFieldsValue() 无法获取 id 值

### Next Action
✅ ROOT CAUSE IDENTIFIED - Need to add hidden Form.Item for id field

### Evidence
- timestamp: "2026-06-05T15:56:00Z"
  source: user_report
  detail: "PUT /api/v1/monitor/jobs/undefined 404 错误，job ID 为 undefined"
  location: "useJobActions.ts:86, index.tsx:197"

- timestamp: "2026-06-05T16:10:00Z"
  source: code_analysis
  detail: "useJobActions.ts:86 使用 values.id 构建 PUT URL，但 form 中没有对应的 Form.Item"
  location: "useJobActions.ts:52-78 (openModal), index.tsx:200-266 (Form)"

- timestamp: "2026-06-05T16:12:00Z"
  source: grep_search
  detail: "在 index.tsx 表单中未找到 name='id' 的 Form.Item"
  location: "index.tsx:200-266"

## Eliminated
(empty)

## Resolution

### Root Cause
`xingran-react-frontend/src/pages/monitor/job/index.tsx` 表单中缺少 `id` 字段的 `Form.Item` 组件。虽然 `useJobActions.ts` 的 `openModal` 函数通过 `form.setFieldsValue({ id: record.id, ... })` 设置了 id 值，但由于表单中没有对应的 `Form.Item`，该字段无法被正确维护和提交。

### Fix
在表单中添加一个隐藏的 `Form.Item` 来存储 `id` 字段：

```tsx
<Form.Item
  name="id"
  hidden
>
  <Input />
</Form.Item>
```

位置：添加在表单的第一个字段之前，确保 id 值能够正确传递。

### Verification Steps
1. 打开任务编辑表单
2. 修改任务内容
3. 点击确定提交
4. 验证 PUT 请求 URL 包含正确的 job ID（而不是 undefined）
5. 验证任务更新成功

### Fix Applied
✅ APPLIED at 2026-06-05T16:18:00Z
File: xingran-react-frontend/src/pages/monitor/job/index.tsx
Change: Added hidden Form.Item for 'id' field before the first form field
