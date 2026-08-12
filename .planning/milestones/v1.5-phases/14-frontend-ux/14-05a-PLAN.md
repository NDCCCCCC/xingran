<!-- Revised 2026-06-14 — plan-checker iteration 1: 修复 BLOCKER 1+2 + WARNING 1-4 -->
---
phase: 14-frontend-ux
plan: 05a
type: execute
wave: 2
depends_on:
  - 14-03
  - 14-01
files_modified:
  - xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx
  - xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx
  - xingran-react-frontend/src/components/shared/index.ts
autonomous: true
requirements:
  - UI-01
  - UI-04

must_haves:
  truths:
    - "EmptyStateWithAction 组件可被任意页面 import,接受 description / actionLabel / actionPath props"
    - "ErrorAlertWithRetry 组件按错误码 1006/1007/500 分级文案,接受 error / onRetry props"
    - "components/shared/index.ts barrel 出口包含两个新组件"
  artifacts:
    - path: "xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx"
      provides: "空数据态组件,接受 description / actionLabel / actionPath props"
      exports: ["default"]
    - path: "xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx"
      provides: "错误态组件,接受 error / onRetry props,内部按错误码分级文案"
      exports: ["default"]
    - path: "xingran-react-frontend/src/components/shared/index.ts"
      provides: "barrel export 包含两个新组件"
      contains: "EmptyStateWithAction|ErrorAlertWithRetry"
  key_links:
    - from: "xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx"
      to: "错误码映射 1006/1007/500"
      via: "switch (error.code) 文案"
      pattern: "1006|1007|500"
    - from: "xingran-react-frontend/src/components/shared/index.ts"
      to: "EmptyStateWithAction.tsx 与 ErrorAlertWithRetry.tsx"
      via: "barrel export"
      pattern: "EmptyStateWithAction.*ErrorAlertWithRetry|ErrorAlertWithRetry.*EmptyStateWithAction"
---

<objective>
为 Phase 14 三态(空/加载/错误)打磨抽离可复用组件 `EmptyStateWithAction` 与 `ErrorAlertWithRetry`,在 components/shared/ 目录下创建这两个组件并加入 barrel 导出。本 plan **仅**完成组件抽取与 barrel 注册,页面改造与设备列表联动入口由 14-05b 负责。

Purpose: 本 plan 输出两个共享组件,14-05b 改造 history.tsx 与 trajectory.tsx 时可直接 import 使用。14-05a 与 14-05b 拆分原因:Wave 2 让 shared 组件就位(14-04 也可提前使用),Wave 3 串行做页面改造,降低 14-05b 单 plan 上下文压力(原 14-05 含 3 任务 / 6 文件)。

Output: 2 个新组件 + 1 个 barrel export 增量。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/14-frontend-ux/14-CONTEXT.md (D-18 / D-19 / D-20 全部锁定)
@.planning/REQUIREMENTS.md (UI-01 / UI-04)
@.planning/CLAUDE.md (ErrorCode 1006/1007/500 来自 API 响应规范)
@.planning/phases/14-frontend-ux/14-05b-PLAN.md (依赖本 plan 提供的 shared 组件)

# 参考实现
@xingran-react-frontend/src/components/shared/ExcelImport.tsx
@xingran-react-frontend/src/pages/operations/buildings/index.tsx
</context>

<tasks>

<task type="auto">
  <name>Task 1: 创建 EmptyStateWithAction 与 ErrorAlertWithRetry 组件 + barrel export</name>
  <files>xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx, xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx, xingran-react-frontend/src/components/shared/index.ts</files>
  <read_first>
    - D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\components\shared\ExcelImport.tsx
    - D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\pages\operations\buildings\index.tsx
    - D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\components\shared\index.ts (现状,需追加 export 不覆盖既有)
  </read_first>
  <action>
    1. 新建 components/shared/EmptyStateWithAction.tsx,FC 组件,props 为 { description: string; actionLabel?: string; actionPath?: string; icon?: React.ReactNode },内部用 AntD Empty 组件 + Button 包裹 Link(useNavigate 实现跳转或 Link 标签)。无 actionLabel/actionPath 时不渲染按钮。
    2. 新建 components/shared/ErrorAlertWithRetry.tsx,FC 组件,props 为 { error: Error 或 ApiError; onRetry?: () = void },内部 switch(error.code 或从 error.message 解析 code):
       - 1006: Alert type=error message=该设备不存在或已被删除,showIcon
       - 1007: 调用 authStore.logout() 或 navigate('/login');Alert message=登录已失效,正在跳转...
       - 500: Alert type=error message=服务暂不可用,请稍后重试,showIcon
       - 其他: Alert type=error message=查询失败:{error.message}
       文案后跟 Button 重新加载 (onClick 调 onRetry)。
    3. 在 components/shared/index.ts 中 export 两个新组件(在已有 export 之后追加,不删除既有 export)。
  </action>
  <verify>
    <automated>cd "D:/CODE/ClaudeCode/xingran-go-backend/xingran-react-frontend" && npx tsc --noEmit -p . 2>&1 | head -30</automated>
  </verify>
  <acceptance_criteria>
    - EmptyStateWithAction.tsx 存在,export default 一个 FC(grep default export 命中 ≥ 1 行)
    - ErrorAlertWithRetry.tsx 存在,含 1006/1007/500 三个错误码分支(grep 命中 3 个数字字符串)
    - components/shared/index.ts 包含 EmptyStateWithAction 与 ErrorAlertWithRetry 两个 export(grep 命中 2 个名字)
    - 既有 components/shared/index.ts 的其他 export 不被删除(grep 既有组件名仍命中)
    - npx tsc --noEmit 退出码 0
  </acceptance_criteria>
  <done>两个三态组件可被任意页面 import 使用,barrel 导出就位,14-05b 可直接复用。</done>
</task>

</tasks>

<verification>
1. 14-05a 完成后,`import { EmptyStateWithAction, ErrorAlertWithRetry } from '@/components/shared'` 在 14-05b 与 14-04 中可解析
2. 既有 shared 组件(如 ExcelImport / ExcelExport)仍可通过同 barrel 导入
3. npx tsc --noEmit 0 错误
</verification>

<success_criteria>
- [ ] EmptyStateWithAction 组件 + props 接口符合 D-18
- [ ] ErrorAlertWithRetry 组件 + 错误码映射符合 D-20
- [ ] barrel export 包含两个新组件
- [ ] 既有的 shared 组件 export 不被破坏
- [ ] Wave 2 并行就位,14-04 / 14-05b 可依赖
</success_criteria>

<output>
Create `.planning/phases/14-frontend-ux/14-05a-SUMMARY.md` when done
</output>
