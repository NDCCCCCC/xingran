---
slug: knowledge-base
status: resolved
skip_audit: true
trigger: GSD Debug Knowledge Base 索引文件（非 debug session）
created: 2026-06-15
updated: 2026-06-25
session_type: meta
---

# GSD Debug Knowledge Base

Resolved debug sessions. Used by `gsd-debugger` to surface known-pattern hypotheses at the start of new investigations.

---

## workstation-ad-asset-404-recurre — 工位设备 /ad /asset /set-primary-and-save 路由 404 反复出现
- **Date:** 2026-06-15
- **Error patterns:** workstation-device, 404, route missing, handler not registered, awaiting_human_verify
- **Root cause:** internal/api/router.go 工位设备路由组中缺失 POST /:id/ad、POST /:id/asset、POST /:id/set-primary-and-save 三个路由注册；handler 方法已实现但从未挂载到 gin.Engine
- **Fix:** 在 router.go:595 后追加 3 行 POST 注册（GetADDevices/GetAssetDevices/SetPrimaryAndSave）
- **Files changed:** internal/api/router.go
- **Recurrence pattern:** 6/13 session 状态停留在 awaiting_human_verify，文档声称已注册但实际未落地；session 关闭后修复永远丢失
- **Prevention:** 1) awaiting_human_verify→resolved 需强约束后端冒烟测试通过 2) 集成测试扫描 handler 方法 vs router 注册一致性 3) 清理散乱 worktree
---

## vm-list-usenavigate-undefi — 虚拟机列表页空白 ReferenceError useNavigate / PlusOutlined
- **Date:** 2026-06-17
- **Error patterns:** ReferenceError, useNavigate, PlusOutlined, react-router-dom, @ant-design/icons, VirtualMachineList, blank page, undefined symbol, index.tsx:37, VDI
- **Root cause:** VirtualMachineList/index.tsx 同时缺两个 import：line 2 只 import 了 useLocation（漏 useNavigate），line 25-28 只 import 了 ReloadOutlined（漏 PlusOutlined）；两者在 hook 调用处（line 37）与 JSX icon 处（line 828）被使用但未 import，导致组件 mount 时抛 ReferenceError、React 卸载整棵子树 → 页面空白。
- **Fix:** 在两个既有 import 块中追加缺失的 symbol（`useNavigate` 加进 react-router-dom import；`PlusOutlined` 加进 @ant-design/icons import），不重写 import 结构。
- **Files changed:** xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx
---

## user-management-input-not-defined — UserManagement 页面 ReferenceError: Input is not defined
- **Date:** 2026-06-17
- **Error patterns:** ReferenceError, Input, not defined, UserManagement, index.tsx:497, antd destructure, missing import, React 19
- **Root cause:** src/pages/system/user/index.tsx 8-24 行 antd 导入块 destructure 列表中漏 `Input`，但 JSX 在 10 处使用 `<Input>` 和 `<Input.Password>`（搜索表单 497/500、编辑弹窗 606/614/622/625/628/631、重置密码 690/708）；React 19 在首次 render 时抛出 ReferenceError 导致整页无法挂载。
- **Fix:** 在 antd destructure 中按字母序在 `Form,` 和 `Select,` 之间插入 `Input,`（仅 1 行 diff），所有 10 处使用随之生效。
- **Files changed:** xingran-react-frontend/src/pages/system/user/index.tsx
- **Related pattern:** 与 vm-list-usenavigate-undefi（VirtualMachineList 缺 useNavigate + PlusOutlined）同源——`npm run lint` 缺少 no-undef 检查时，antd 子模块 destructure 漏写不会在 lint 阶段暴露，仅在浏览器运行时以 ReferenceError 表现。Prevention: 1) 引入 esbuild/vite 的 esm resolver 提示 2) 业务页面统一通过 `import { ... } from '@/lib/antd'` 聚合层 3) 把 `noUncheckedSideEffectImports` 打开让 TS 报错。
---
