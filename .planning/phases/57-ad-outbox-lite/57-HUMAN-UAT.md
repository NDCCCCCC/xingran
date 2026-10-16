---
status: partial
phase: 57-ad-outbox-lite
source: [57-VERIFICATION.md]
started: 2026-10-17T00:00:00+08:00
updated: 2026-10-17T00:00:00+08:00
---

## Current Test

[awaiting human testing]

## Tests

### 1. PG 真实环境验证 ON CONFLICT 部分索引合并路径（QUEUE-02）
expected: 连续两次编辑同一用户 → `ad_push_queue` 仅 1 行 pending（合并而非新增行）。若报 `42P10 there is no unique or exclusion constraint matching the ON CONFLICT specification` → 谓词写错。sqlite 侧 `TestPhase57_EnqueueUpdate_MergeOnSecondEdit` 已提供等价回归保护，但 PG 的冲突目标推断路径只能实机确认。
result: [pending]

### 2. 进程重启不丢已入队行（QUEUE-01 / SC1 后半）
expected: PG 部署后编辑用户入队 → `kill -9` → 重启 → `SELECT * FROM ad_push_queue WHERE status='pending'` 行仍在。
result: [pending]

### 3. AD 端复活链路闭环（QUEUE-06 端到端）
expected: sys 删除用户 → Phase 58 排水器执行 DisableUser → AD 端 UAC 514 → bind 失败（data 533）。执行侧（排水器）Phase 58 才存在，本 phase 只能验证队列行产生（已由测试锁定）；本项本质是 Phase 58 验收项，挂此处防遗漏。
result: [pending]

## Summary

total: 3
passed: 0
issues: 0
pending: 3
skipped: 0
blocked: 0

## Gaps
