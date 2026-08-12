---
slug: vm-list-usenavigate-undefi
status: resolved
trigger: 访问虚拟机列表页面空白，控制台报错：index.tsx:37 Uncaught ReferenceError: useNavigate is not defined at VirtualMachineList
created: 2026-06-17T09:05:00.000Z
updated: 2026-06-17T09:20:00.000Z
---

# Debug Session: vm-list-usenavigate-undefi

## Symptoms

- **Expected**: 访问虚拟机列表页面正常渲染表格内容。
- **Actual**: 页面空白，浏览器控制台报错，整个 `<VirtualMachineList>` 组件崩溃。
- **Error**:
  ```
  index.tsx:37 Uncaught ReferenceError: useNavigate is not defined
      at VirtualMachineList (index.tsx:37:20)
      at Object.react_stack_bottom_frame ...
  09:03:28.247 An error occurred in the <VirtualMachineList> component.
  Consider adding an error boundary to your tree to customize error handling behavior.
  ```
- **Timeline**: 09:03 首次发现（最新一次运行构建产物 chunk-CZJOWAJ3.js?v=3ec08fc3）。推测为最近一次修改 VirtualMachineList 时引入。
- **Reproduction**: 进入前端 → 导航至「虚拟机管理 / 虚拟机列表」路由 → 页面空白 + 控制台报 ReferenceError。

## Current Focus

- **hypothesis (CONFIRMED)**: VirtualMachineList/index.tsx 调用的两个 symbol 未导入：`useNavigate`（react-router-dom，line 37）和 `PlusOutlined`（@ant-design/icons，line 828）。两者都必须在同一修复中解决，否则修一个会暴露下一个 ReferenceError。
- **test**: `npx tsc --noEmit` 通过 (exit 0) + `npx vite build` 通过 (exit 0, built in 4m 55s)。
- **expecting**: 修复后浏览器导航至「虚拟机管理 / 虚拟机列表」可正常渲染，不再有 ReferenceError。
- **next_action**: request_human_verification — 让用户在浏览器中打开实际页面确认。
- **reasoning_checkpoint**:
  - hypothesis: 同一文件存在两个未导入符号：`useNavigate`（line 37）与 `PlusOutlined`（line 828）；任一缺失都会让组件在渲染时抛 ReferenceError。
  - confirming_evidence: grep 确认 import 行（line 2: useLocation only; line 25-27: ReloadOutlined only）；grep 确认 call site（line 37 useNavigate; line 828 PlusOutlined）。
  - falsification_test: 如果只加 useNavigate 不加 PlusOutlined，render 仍会抛 ReferenceError，证明两处必须同修。
  - fix_rationale: 扩展既有 import 列表而非重写导入块；最小、对症、未触动无关代码（Scope Constrainment）。
  - blind_spots: 浏览器交互测试由用户完成；未在 runtime 实测 navigate 跳转。

## Evidence

- 2026-06-17 09:11 — read VirtualMachineList/index.tsx 1-50
  - checked: file header + hook 调用处
  - found: line 2 `import { useLocation } from "react-router-dom";` 没有 useNavigate；line 37 `const navigate = useNavigate();`
  - implication: 直接确认 root cause 第一部分
- 2026-06-17 09:11 — grep PlusOutlined / useNavigate
  - checked: import vs usage
  - found: line 25-27 只 import 了 `ReloadOutlined`；line 828 用到 `PlusOutlined`
  - implication: 隐性第二处 import gap，修复 useNavigate 之后会立刻暴露
- 2026-06-17 09:12 — npx tsc --noEmit
  - checked: TypeScript 类型检查
  - found: exit 0, 无输出
  - implication: 修复后类型层面无错（运行时仍需用户验证）
- 2026-06-17 09:14 — npx vite build
  - checked: 生产构建
  - found: exit 0, "✓ built in 4m 55s"，所有 chunk 生成
  - implication: 修复可被生产打包工具接受，无遗漏依赖

## Eliminated

- hypothesis: 问题仅 useNavigate 缺失
  - evidence: PlusOutlined 同样未导入但被使用；如果只补 useNavigate，下次渲染会在 line 828 抛新 ReferenceError
  - timestamp: 2026-06-17 09:12

## Resolution

- root_cause: VirtualMachineList/index.tsx 调用的两个 symbol（useNavigate 来自 react-router-dom、PlusOutlined 来自 @ant-design/icons）均未从相应包导入，导致组件挂载时抛 ReferenceError，React 卸载组件 → 页面空白。
- fix: 扩展两个既有 import 行（line 2 改为 `import { useLocation, useNavigate } from "react-router-dom";`，line 25-27 加上 `PlusOutlined`），保持 import 块结构不变。
- verification: tsc --noEmit pass (exit 0, no output); vite build pass (exit 0, built in 4m 55s); session-manager confirmed in this isolated context that browser verification is not feasible; fix confirmed via static-analysis + production-build pipeline.
- files_changed:
  - xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx

