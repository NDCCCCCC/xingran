---
phase: 55-phase-53-leftover-sweep
reviewed: 2026-07-08T00:00:00Z
depth: standard
files_reviewed: 6
files_reviewed_list:
  - xingran-react-frontend/src/components/network/port-write/constants.ts
  - xingran-react-frontend/src/components/network/port-write/PortWriteModal.tsx
  - xingran-react-frontend/src/components/network/port-write/BulkWriteDrawer.tsx
  - xingran-react-frontend/src/pages/network/ports/index.tsx
  - xingran-react-frontend/src/components/reconciliation/__tests__/HealthCard.test.tsx
  - internal/services/portwrite/batch_orchestrator.go
findings:
  critical: 0
  warning: 1
  info: 2
  total: 3
status: issues_found
---

# Phase 55: Code Review Report

**Reviewed:** 2026-07-08
**Depth:** standard
**Files Reviewed:** 6
**Status:** issues_found

## Summary

Phase 55 是一个纯技术债清理 phase,5 项可执行遗留(WR-02 validator 签名、IN-01 instanceof Error 收窄、IN-02 eslint-disable、CR-02 后端 fallback 归属校验、HealthCard 空态断言)均已落地并通过自验证。整体改动质量高:validator 修复用 `form.getFieldValue("reasonText")` 跨字段取值,两个组件(PortWriteModal 与 BulkWriteDrawer)对齐使用 `Form.useWatch("action", form)` 动态 rules;CR-02 在 fallback 分支前置 DB 查询,deviceID 不匹配时归 Failed + `continue` 而非 `break`(语义区分数据校验 vs SSH 传输失败),正确避免了 SSH 下发到错位设备。

Critical 项 0。Warning 1 项(inline 已注释代码段,符合 Scope Constrainment 但有维护性提示)。Info 2 项(HealthCard 测试残留问题 + CR-02 executeWrite 仍传 req.DeviceID 的微脆弱点)。

未发现以下范围外项(已按 CONTEXT.md 排除):WR-03/WR-04、HealthCard import 80s 性能、全路径归属校验。

## Critical Issues

无。

## Warnings

### WR-01: BulkWriteDrawer.tsx 残留 7 行已注释的 composeReason 实现

**File:** `xingran-react-frontend/src/components/network/port-write/BulkWriteDrawer.tsx:63-69`

**Issue:** 文件 line 63-69 保留了 WR-02 修复前的本地 `composeReason` 函数注释块(7 行,含完整函数体)。代码块以 `// function composeReason(...)` 开头,虽然已注释不会执行,但属于注释掉的代码 —— 是 code smell,且容易被未来的 refactor 当成"历史参考"误恢复。

具体原文:
```typescript
// 55-01 WR-02: 实现下沉到 ./constants, BulkWriteDrawer 仅引用。 */
// function composeReason(reasonSelect: unknown, reasonText: unknown): string | null {
//   if (reasonSelect === REASON_CUSTOM_SENTINEL) {
//     ...
//   }
//   return typeof reasonSelect === "string" && reasonSelect.length > 0 ? reasonSelect : null;
// }
```

**Fix:** 删除 line 63-69 全部注释,仅保留 line 61-62 的 "同 D-02: reasonSelect + reasonText 合并" 注释文字即可。CONTEXT.md Scope Constrainment 允许删除与本阶段改动相关的残留代码 —— 此段正是 WR-02 helper 抽取时遗留的 dead 注释。

## Info

### IN-01: HealthCard.test.tsx line 101 残留 pre-existing 测试失败

**File:** `xingran-react-frontend/src/components/reconciliation/__tests__/HealthCard.test.tsx:101`

**Issue:** 测试 line 101 断言 `screen.getByText("对账健康度")`,但当前 HealthCard.tsx 已不再渲染 Card 标题(单行紧凑版移除了 Card 标题 —— 见 `/gsd-fast` 2026-06-30 重构)。这是 55-01-SUMMARY.md 已记录的 PRE-EXISTING OUT-OF-SCOPE 失败,本阶段范围严格限定为 line 127 的 empty state 断言。

附带观察:line 105-109 的 KPI 标题断言(`正常`/`漂移`/`冲突`/`无数据`/`例外命中`)在 HealthCard.tsx line 94 中渲染为单个 `<span>` 文本 `"正常 N · 漂移 N · 冲突 N [· 无数据 N] [· 例外 N]"` —— `getByText("正常")` 精确匹配很可能失败(sibling node + 文本前缀)。但本阶段不动这些断言是 Scope Constrainment 正确选择,仅作 informational 记录。

**Fix:** 不在本次范围内修复。后续可单开 follow-up phase 一次性修复:把 line 101 改为 regex `getByText(/对账健康度/)` 或删除该断言(line 105-109 同理改 regex 或使用 `getAllByText`)。当前 npm test 套件状态:HealthCard.test.tsx 1/5 失败,整体 64/65 测试通过(见 55-01-SUMMARY.md 验证结果)。

### IN-02: CR-02 executeWrite 调用仍传 req.DeviceID 而非 actualPort.DeviceID

**File:** `internal/services/portwrite/batch_orchestrator.go:98`

**Issue:** CR-02 修复后,fallback 分支 line 98 调用 `s.executeWrite(ctx, portID, req.DeviceID, ...)`,传的是 `req.DeviceID` 而非 `actualPort.DeviceID`。虽然 line 88 的 `actualPort.DeviceID != req.DeviceID` 校验已经确保二者相等,功能正确,但**自文档化与未来重构容错性不足** —— 如果有人后续调整校验顺序或注释,容易形成隐性依赖。

参考对照:line 120 主路径调用 `s.executeWrite(ctx, port.ID, port.DeviceID, ...)` 使用 `port.DeviceID`(从 preStateMap 取出),fallback 路径用 `req.DeviceID`,两条路径风格不一致。

**Fix(可选,非阻塞):** 把 line 98 改为 `s.executeWrite(ctx, portID, actualPort.DeviceID, req.Action, req.Description, operator, "")`,与主路径 line 120 风格对齐。即使 line 88 校验被意外删除,fallback 也不会用错位 deviceID。

```go
// 当前(line 98)
writeResult, werr := s.executeWrite(ctx, portID, req.DeviceID, req.Action, req.Description, operator, "")

// 建议改为
writeResult, werr := s.executeWrite(ctx, portID, actualPort.DeviceID, req.Action, req.Description, operator, actualPort.InterfaceName)
```

附带:`actualPort.InterfaceName` 也可用于审计追踪(原 PLAN.md 中提到的 PortName 字段填入因 PortResult struct 无该字段而被移除,但作为 executeWrite 的最后一参 `preInterfaceName` 是支持的 —— 见 line 120 主路径用法)。

---

_Reviewed: 2026-07-08_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_