---
slug: port-write-undo-shutdown-rename
status: resolved
trigger: "用户 2026-07-08 真机测试：'取消关闭' 这个按钮文案奇怪，应该改成 '启用端口'，和 '关闭端口' 对应"
created: 2026-07-08
related:
  - port-write-shutdown-multi-layer-bug (4 层修复，同日)
---

# Debug Session: port-write-undo-shutdown-rename

## 症状

- **Expected behavior:** 端口管理页面的操作按钮应该语义清晰、动作对偶
- **Actual behavior:**
  - 按钮文案是 "取消关闭"（直译自 backend action key `undo_shutdown`）
  - 用户反馈：与"关闭端口"对应，应该叫 **"启用端口"**
- **重要：** 这是 UX 文案 bug，不是逻辑 bug
- **范围：** 要查 backend audit log 的 operType（status=10? 11?）是否也要跟着调整

## 怀疑方向（Phase 1 待验证）

### 假设 A：前端 `ACTION_TITLE` 常量表里有 undo_shutdown 直译
**位置:** `xingran-react-frontend/src/components/network/port-write/constants.ts`（推测）
- 现有 key 应该是 `shutdown → "关闭端口"`, `undo_shutdown → "取消关闭"`
- 修复：把 `undo_shutdown → "启用端口"`，确认 `ACTION_TITLE` 是前端唯一文案源

### 假设 B：BulkWriteDrawer / PortWriteModal 也有写死文案
- 假设 A 修了后，Drawer 标题"批量取消关闭"也要改"批量启用端口"
- 检查：所有引用 `undo_shutdown` action 的前端文案

### 假设 C：backend audit log operType 用错（OperTypeStatus）
- 关闭端口 → OperTypeStatus (=10 状态变更) ✓
- 启用端口 → OperTypeStatus 也行，但 OperTypeEnable (=12) / OperTypeDisable (=13) 更精确
- 验证：port_write_service.go 写 audit log 时用的常量

### 假设 D：菜单权限字符串也带 undo_shutdown
- 查 sys_menu / sys_role_menu 看是否有 `network:port:undo_shutdown` 这种 perm
- 改文案不动 perm key（key 是 internal，对应无影响）

## 当前焦点

- hypothesis: 已验证并修复完成
- next_action: 提交 + 知识库登记

## 证据

- timestamp: 2026-07-08
  type: user_report
  finding: "取消关闭和奇怪，应该改成启用端口，和关闭端口对应"

- timestamp: 2026-07-08
  type: file_inspect
  checked: `xingran-react-frontend/src/components/network/port-write/constants.ts:50`
  found: `undo_shutdown: "取消关闭"` (ACTION_TITLE 共享常量)
  implication: PortWriteModal 标题 + BulkWriteDrawer action selector 都用此源

- timestamp: 2026-07-08
  type: file_inspect
  checked: `xingran-react-frontend/src/pages/network/ports/index.tsx:345`
  found: `{ key: "undo_shutdown", label: "取消关闭", onClick: ... }` (硬编码)
  implication: 行操作菜单的"取消关闭"按钮文案是用户直接看到的地方

- timestamp: 2026-07-08
  type: file_inspect
  checked: `internal/api/v1/network/port_write_handler.go:69-105`
  found: Shutdown/UndoShutdown/EnableDot1x/DisableDot1x 4 个 handler 全用 `operlog.OperTypeStatus(=10)`
  implication: operType 通用但不够精确, Enable/Disable 有更准的 OperTypeEnable(=12)/OperTypeDisable(=13)

- timestamp: 2026-07-08
  type: file_inspect
  checked: `pkg/permission/config.go:188` (search undo_shutdown)
  found: 权限注释含 undo_shutdown 但 perm key 是 internal, 不影响 UI
  implication: perm key 保持不变 (假设 D 验证: 改文案不动 perm key)

## 排除

- hypothesis: 修改后端 action key undo_shutdown → enable 等
  evidence: action key 是 protocol 字段, 改会破 service/handler/fixture 链
  timestamp: 2026-07-08

## 决议

- root_cause:
  1. 前端 ACTION_TITLE 共享常量把 action key 直译为"取消关闭" (constants.ts:50)
  2. 行操作菜单硬编码同样的"取消关闭" (ports/index.tsx:345)
  3. backend operType 用通用 OperTypeStatus(10), 缺少对偶精确度 (handler.go:69/77/93/101)

- fix:
  1. constants.ts: `undo_shutdown: "启用端口"` (与"关闭端口"对偶)
  2. ports/index.tsx:345: `label: "启用端口"`
  3. handler.go: Shutdown → OperTypeDisable(13), UndoShutdown → OperTypeEnable(12),
     EnableDot1x → OperTypeEnable(12), DisableDot1x → OperTypeDisable(13)
  4. handler_test.go:242: `lastBusinessType` 10 → 13, 注释同步更新
  5. perm key / action key / fixture 文件名: 全部保持不变 (protocol 字段)

- verification:
  - go build ./... — pass
  - go test ./internal/api/v1/network/... — pass
  - go test ./internal/services/portwrite/... — pass
  - npm run type-check (frontend) — pass
  - npx vitest run src/components/network/port-write/ — 26/26 pass
  - PortWriteModal.test.tsx 仍 mock `action="undo_shutdown"` (action key 不变, 测试不变)

- files_changed:
  - xingran-react-frontend/src/components/network/port-write/constants.ts
  - xingran-react-frontend/src/pages/network/ports/index.tsx
  - internal/api/v1/network/port_write_handler.go
  - internal/api/v1/network/port_write_handler_test.go
