---
slug: textfsm-dot1x-template-mismatch
status: resolved
trigger: 华为设备 display dot1x 的 TextFSM 模板匹配失败
created: 2026-05-12
updated: 2026-05-12
---

# Debug: TextFSM dot1x Template Mismatch

## Symptoms

- **Expected**: 行 "GigabitEthernet0/0/1 status: UP  802.1x protocol is Enabled" 应被模板正确匹配并解析出端口名、状态、dot1x协议状态
- **Actual**: 该行不匹配模板中的任何规则，被跳过
- **Error**: TextFSM 日志显示 "行10不匹配任何规则"
- **Timeline**: 新发现的模板不完整问题
- **Reproduction**: 对华为设备执行 display dot1x 命令，解析输出

## Current Focus

- **hypothesis**: 模板 huawei_vrp_display_dot1x.textfsm 缺少匹配端口状态行的规则
- **next_action**: gather initial evidence

## Evidence

- 2026-05-12: `escapeRegexLiteral` 对 `802\.1x` 模式产生了错误的正则输出 `802\\\.1x`（三重转义），应为 `802\.1x`
- 2026-05-12: 根因：`escapeRegexLiteral` 在处理 `\` + `.` 时，因 `.` 不在保留的字母转义列表 `sSdDwWbBnrt` 中，将 `\` 转义为 `\\`，然后又将 `.` 转义为 `\.`，导致 `802\\\.1x`
- 2026-05-12: 次要问题：裸 `.` 在模板中被错误转义为 `\.`（字面点），导致 `^. -> Continue` 规则编译为 `^\.` 只匹配字面点字符，而非 TextFSM 预期的 `^.` 通配符

## Eliminated

- 模板本身缺少规则 -- 排除，模板有正确的 `${INTERFACE}\s+status:.*802\.1x.*` 规则

## Resolution

- **root_cause**: `internal/templates/textfsm.go` 中的 `escapeRegexLiteral` 函数存在两个 bug：(1) `\` + 非字母元字符（如 `\.` `\\` `\(` 等）没有被保留为有效的正则转义序列，而是被双重转义；(2) 裸 `.` 被转义为 `\.`（字面点），但在 TextFSM 模板上下文中裸 `.` 应保留为正则通配符
- **fix**: 修改 `escapeRegexLiteral` 函数：(1) 当 `\` 后跟非字母字符时，保留整个转义序列原样（因为这些是有效的正则元字符转义如 `\. ` `\(` `\\`）；(2) 裸 `.` 不再转义，保持为正则通配符。修复后 `802\.1x` 正确编译为 `802\.1x`（匹配字面点），`^.` 正确编译为 `^.`（匹配任何字符）
- **files_changed**:
  - `internal/templates/textfsm.go` -- 修复 `escapeRegexLiteral` 函数
  - `internal/templates/textfsm_dot1x_test.go` -- 新增测试验证修复

## Specialist Review

- **reviewer**: go (general engineering debug)
- **result**: LOOKS_GOOD -- the fix correctly distinguishes between regex escape sequences (which must be preserved) and literal characters that need escaping
