---
phase: 13-query-layer-trajectory
plan: 09
type: execute
subsystem: frontend-controlled-component
tags: [react, antd-form, controlled-component, mac-input, gap-closure, CR-02]
dependency_graph:
  requires:
    - 13-08 (W1-CR-02 already known + `queryMACVendor` import in place)
  provides:
    - "MAC Input 由 AntD Form 单一驱动(Single Source of Truth)"
    - "React controlled component 反模式 CR-02 修复"
  affects:
    - "xingran-react-frontend/src/pages/network/mac/trajectory/TrajectoryPage.tsx"
tech-stack:
  added: []
  patterns:
    - "AntD Form.Item + Input + onChange 通过 form.setFieldValue 同步(删除 e.target.value 直接 DOM 写入)"
    - "Option b 路径:不引入 macInput useState,Form 作为 MAC 字段唯一真相源"
key-files:
  created: []
  modified:
    - path: xingran-react-frontend/src/pages/network/mac/trajectory/TrajectoryPage.tsx
      change: "MAC Input onChange 改用 form.setFieldValue,删除 e.target.value = formatted 反模式"
decisions:
  - "Option b(纯 Form 驱动)而非 Option a(Form + macInput 双源):避免状态双源漂移风险"
  - "onBlur 行为完全保留:仍调用 form.setFieldValue(mac, formatted),与 onChange 形成双重保险"
  - "Input 不显式设置 value/defaultValue:由 Form.Item 的 valuePropName=\"value\" 接管"
metrics:
  duration_minutes: 3
  completed_date: 2026-06-26T06:19:01Z
---

# Phase 13 Plan 09: TrajectoryPage MAC Input Controlled Component Fix Summary

**One-liner:** 删除 `e.target.value = formatted` 直接 DOM 写入反模式,改走 AntD Form `setFieldValue` 单一源路径,根除光标跳跃 + 状态双源漂移风险。

## Objective Recap

Phase 13 验证发现 CR-02 反模式:`TrajectoryPage.tsx` MAC Input onChange 中使用 `e.target.value = formatted` 直接 DOM 写入,违反 React 受控组件原则,导致用户键入中间位时光标跳到末尾。Plan 09 采用 Option b(极简路径):不新增 `macInput` useState,完全依赖 AntD Form 作为 MAC 字段唯一真相源。

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | MAC Input onChange 删除直接 DOM 写入,改用 form.setFieldValue | see below | TrajectoryPage.tsx |

## Changes Made

### `xingran-react-frontend/src/pages/network/mac/trajectory/TrajectoryPage.tsx`

**行 232-245(原)**: Input 组件 onChange 直接修改 DOM:
```tsx
onChange={(e) => {
  const formatted = normalizeMACAddress(e.target.value);
  if (formatted) {
    e.target.value = formatted;  // 🛑 反模式
  }
}}
```

**行 232-245(改后)**: 通过 AntD Form `setFieldValue` 同步状态:
```tsx
onChange={(e) => {
  // 实时格式化输入(Phase 13 CR-02:走 AntD Form 单一源,删除 e.target.value = formatted 反模式)
  const formatted = normalizeMACAddress(e.target.value);
  if (formatted) {
    form.setFieldValue("mac", formatted);
  }
}}
onBlur={(e) => {
  // 失焦自动规范化(增强 UX,与 onChange 形成双重保险)
  const formatted = normalizeMACAddress(e.target.value);
  if (formatted) {
    form.setFieldValue("mac", formatted);
  }
}}
```

**关键差异**:
1.  `e.target.value = formatted` 反模式删除 → 走 React 状态路径
2.  onChange 与 onBlur 现在统一调用 `form.setFieldValue("mac", formatted)`,形成"实时同步 + 失焦再确认"双重保险
3.  未新增 `macInput` useState(Option b 选择,Form 作为唯一源)

## Verification Results

### Automated Grep Checks

| Check | Result |
|-------|--------|
| `e.target.value\s*=` in TrajectoryPage.tsx | 0 命中(已删除 DOM 写入) |
| `macInput\|setMacInput` in TrajectoryPage.tsx | 0 命中(Option b:不引入新 state) |
| `form.setFieldValue("mac"` in TrajectoryPage.tsx | 2 命中(onChange line 236 + onBlur line 243) |

### TypeScript Validation

```bash
cd D:/CODE/ClaudeCode/xingran-go-backend/xingran-react-frontend && npm run type-check
# Output: 无错误,退出码 0
```

### ESLint Validation

```bash
cd D:/CODE/ClaudeCode/xingran-go-backend/xingran-react-frontend && npx eslint src/pages/network/mac/trajectory/TrajectoryPage.tsx
# 2 errors, 8 warnings(全部为 13-08 引入的预存在 baseline,本次未引入 NEW error/warning)
# Errors: 行 108 / 120 — react-hooks/exhaustive-deps(setActivePreset 缺失,与 CR-02 修复无关)
```

**结论**: 本次改动未引入任何新的 lint error/warning,2 个 error 为 13-08 锁定的预存在 baseline(无新增恶化)。

## Deviations from Plan

None — 计划 Option b 路径精确执行:

1. 不新增 `macInput` useState ✓
2. onChange 改用 `form.setFieldValue("mac", formatted)`,删除 `e.target.value = formatted` ✓
3. onBlur 行为保留(`form.setFieldValue`) ✓
4. TypeScript 0 退出码 ✓
5. 无 macInput/setMacInput 残留 ✓

## Threat Model Compliance

| Threat ID | Disposition | Status |
|-----------|-------------|--------|
| T-13W9-01 Tampering DOM mutation | mitigate | COMPLETED: 删除 `e.target.value = formatted`,仅留 form.setFieldValue 路径 |
| T-13W9-02 Repudiation 状态双源漂移 | mitigate | COMPLETED: Option b 不引入 macInput state,AntD Form 唯一管理 |
| T-13W9-03 Repudiation 输入状态与 Form state 漂移 | mitigate | COMPLETED: onChange 仅调用 form.setFieldValue(规范化成功时),onBlur 兜底 |
| T-13W9-SC Tampering npm/pip/cargo installs | mitigate | COMPLETED: 无新依赖,仅回调体改写 |

## Impact

- **UX 修复**: 用户键入中间位时光标不再跳到末尾(典型 controlled mutation 副作用消除)
- **代码质量**: 消除 React 受控组件反模式,符合 `CLAUDE.md § Frontend Conventions`
- **架构简化**: AntD Form 成为 MAC 字段唯一真相源,删除任何状态双源风险

## Commit

```
fix(13-09): use form.setFieldValue for MAC input (delete e.target.value direct DOM mutation)
```

## Self-Check: PASSED

- 计划任务 1 完成 ✓
- TypeScript 0 错误 ✓
- 无 NEW lint 错误 ✓
- Grep 验证全部通过 ✓
- 13-08 baseline(2 errors)未恶化 ✓