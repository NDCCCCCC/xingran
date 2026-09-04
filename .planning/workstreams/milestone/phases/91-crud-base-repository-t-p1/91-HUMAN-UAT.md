---
status: partial
phase: 91-CRUD 复用 base.Repository[T]
source: [91-VERIFICATION.md]
started: 2026-09-04
updated: 2026-09-04
---

## Current Test

[awaiting human testing]

## Tests

### 1. 前后端联调 smoke — 工位/楼宇/资产列表
expected: 登录 → 工位列表分页正常（pageSize 上限 100 生效，无白屏/报错）；楼宇/资产列表 Total 与实际行数一致（软删行不再计入 Total，F3 checkpoint 批准的行为变更）；building/floor/asset/workstation CRUD 全流程正常
result: [pending]

### 2. 前端 payload 形态审计 — typed bind 降级语义
expected: 各列表页 status/type 筛选参数均为整数或省略（workstation typed bind 后，畸形数值字段触发整包降级跳过过滤——REVIEW IN-03 记录的迁移前后语义差异）；下拉/筛选行为与迁移前一致
result: [pending]

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps

（LOC ≥800 缺口见 91-VERIFICATION.md frontmatter — 待用户决策，非 UAT 项）
