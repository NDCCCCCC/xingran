---
status: partial
phase: 01-后端导入配置
source: [01-VERIFICATION.md]
started: 2026-04-16T17:30:00Z
updated: 2026-04-16T17:30:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. End-to-end import with valid dept/user names

expected: After importing a workstation Excel file with known department names and usernames, sys_workstation.dept_id and sys_workstation.user_id contain correct UUIDs

result: [pending]

### 2. Import with non-existent dept/user names

expected: Import succeeds, dept_id and user_id are NULL for rows with non-existent names, no error raised

result: [pending]

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps
