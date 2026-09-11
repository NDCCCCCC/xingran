---
phase: 107
plan: "02"
type: execute
wave: 1
depends_on: []
files_modified:
  - xingran-react-frontend/src/components/dashboard/layout/LayoutToolbar.tsx
  - xingran-react-frontend/src/pages/operations/building-spaces-3d/hooks/useGeocoding.ts
  - xingran-react-frontend/src/pages/operations/building-spaces/components/WorkstationView.tsx
  - xingran-react-frontend/src/pages/operations/assets/index.tsx
  - xingran-react-frontend/src/pages/operations/rpa/tasks/modals/AIScriptEditor.tsx
autonomous: true
requirements_addressed: [TODO-05]
---

<objective>
Batch delete frontend TODO comments. Pure comment removal — keep stub implementations.
</objective>

<tasks>

## TASK-1: Delete TODO comments in LayoutToolbar.tsx

**File:** `xingran-react-frontend/src/components/dashboard/layout/LayoutToolbar.tsx`

- **Line 132**: Delete `// TODO: 打开Widget选择器`
  - Keep `message.info("Widget选择器功能待实现");` — valid UX stub
- **Line 139**: Delete `// TODO: 打开仪表盘设置`
  - Keep `message.info("仪表盘设置功能待实现");` — valid UX stub

Both are comment-only removals. Stubs remain to indicate unbuilt features.

</tasks>

<tasks>

## TASK-2: Delete TODO in useGeocoding.ts

**File:** `xingran-react-frontend/src/pages/operations/building-spaces-3d/hooks/useGeocoding.ts`

- **Line 131**: Delete `// TODO: 后端暂时不支持逆地址解析，这里保留接口但返回空值`
- **Line 132**: Delete `// 如果需要，可以在后端添加逆地址解析的 API 端点`
- Keep `setLoading(false); return null;` — frontend already handles the backend gap gracefully

</tasks>

<tasks>

## TASK-3: Delete TODO in WorkstationView.tsx

**File:** `xingran-react-frontend/src/pages/operations/building-spaces/components/WorkstationView.tsx`

- **Line 73**: Delete `// TODO: 打开编辑对话框`
- Keep `message.info(...)` — valid UX stub indicating edit dialog is not yet wired

</tasks>

<tasks>

## TASK-4: Delete TODO in assets/index.tsx

**File:** `xingran-react-frontend/src/pages/operations/assets/index.tsx`

- **Line 582**: Delete `// TODO: 实现编辑功能`
- Keep `message.info("编辑功能待实现");` — valid UX stub

</tasks>

<tasks>

## TASK-5: Delete TODO and commented code in AIScriptEditor.tsx

**File:** `xingran-react-frontend/src/pages/operations/rpa/tasks/modals/AIScriptEditor.tsx`

- **Line 97**: Delete `// TODO: 调用后端 AI API 生成脚本`
- **Lines 98-99**: Delete the commented-out API call:
  ```
  // const result = await post('/rpa/ai/generate', { description });
  // setGeneratedActions(result.data.script.actions);
  ```
- Keep the mock: `await new Promise((resolve) => setTimeout(resolve, 1500));` and `setGeneratedActions(mockGeneratedActions);` — valid mock implementation

</tasks>

<success_criteria>
- All TODO comments removed from 5 frontend files
- Stub implementations (message.info, mock delays) remain intact
- `npm run build` (or type-check) passes on frontend
- No functional behavior changes
</success_criteria>

<output>
Part of combined SUMMARY.md after Wave 1 + Wave 2 complete
</output>
