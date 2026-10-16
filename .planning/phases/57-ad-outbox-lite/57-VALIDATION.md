---
phase: 57
slug: ad-outbox-lite
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-10-16
---

# Phase 57 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Source: `.planning/phases/57-ad-outbox-lite/57-RESEARCH.md` (Validation Architecture section)

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + `github.com/stretchr/testify/assert` + `github.com/stretchr/testify/require` + `github.com/glebarez/sqlite` (sqlite in-memory) |
| **Config file** | None — sqlite tests use `gorm.Open(sqlite.Open(":memory:"))` (precedent `user_service_test.go:49`) |
| **Quick run command** | `go test ./internal/services/system/ -run TestPhase57 -v` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~30 seconds (test suite + build) |

---

## Sampling Rate

- **After every task commit:** `go build ./...` + `go test ./internal/services/system/ -run TestPhase57 -v`
- **After every plan wave:** `go test ./...` + `go vet ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** ~15 seconds (single package) / ~60 seconds (full suite)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 57-01-01 | 01 | 1 | HYGIENE-06 (sqlite migration_215 建表) | — | migration_215 双方言 DDL 在 sqlite 内 in-memory 可建 | unit | `go test ./internal/services/system/ -run TestPhase57_MigrationSetup -v` | ❌ W0 | ⬜ pending |
| 57-01-02 | 01 | 1 | QUEUE-01 (事务内入队 + 进程重启不丢) | T-1 | 同事务原子性由 GORM/DB 提供；INSERT 失败回滚业务写 | unit (sqlite + tx rollback) | `go test ./internal/services/system/ -run TestPhase57_UpdateEnqueue_TxAtomic -v` | ❌ W0 | ⬜ pending |
| 57-01-03 | 01 | 1 | QUEUE-02 (按实体合并) | — | 部分唯一索引夺锁 + ON CONFLICT 文本级谓词 | unit (sqlite + 同实体两次写入) | `go test ./internal/services/system/ -run TestPhase57_UpdateEnqueue_Merge -v` | ❌ W0 | ⬜ pending |
| 57-02-01 | 02 | 2 | QUEUE-06 (Delete + BatchDelete disable 入队) | T-2 | 调用方 service 层；handler RequirePermissions 既有 | unit (sqlite + EnqueueDisable + Batch) | `go test ./internal/services/system/ -run TestPhase57_DeleteEnqueue -v` | ❌ W0 | ⬜ pending |
| 57-02-02 | 02 | 2 | HYGIENE-06 (full sqlite suite green) | — | 直调 migration_215 在 sqlite in-memory 落地 + 全包 sqlite 测试 | unit (full suite) | `go test ./...` | ❌ W0 | ⬜ pending |
| 57-02-03 | 02 | 2 | D-02 前置条件 (无 AD 配置行 → 不入队 + WARN) | — | service 层事务内查 sys_ad_config 启用行 | unit (sqlite + sys_ad_config fixture) | `go test ./internal/services/system/ -run TestPhase57_Precondition -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/services/system/ad_push_queue_test.go` — covers QUEUE-01 (diff + tx rollback on INSERT fail) / QUEUE-02 (同实体合并) / QUEUE-06 (Delete + BatchDelete) / D-02 前置条件
- [ ] `internal/core/db/migrations/migration_215_create_ad_push_queue_test.go` — 双方言 DDL smoke（sqlite 真实可建；PG 由 UAT 覆盖，仿 migration_212 模式无 _test 文件）
- [ ] Framework install: 既有（无新增依赖）

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 进程重启不丢失已入队行 | QUEUE-01 | 需真实部署二进制 + 启停 | 在 PG 部署后 UPDATE sys_user → 入队 → kill -9 进程 → 重启 → `SELECT * FROM ad_push_queue WHERE status='pending'` 验证行存在 |
| ON CONFLICT 谓词不匹配 42P10 | QUEUE-02 | PG 特定错误码，sqlite 不复现 | PG 部署后用 `EXPLAIN` 验证索引命中 + 入队合并测试 |
| D-04 复活链路守门 | QUEUE-06 | 涉及 AD 真实 bind + disable UAC 514 | AD 端真实操作：sys 删除用户 → AD bind 期望失败（UAC 533） |

*If none: "All phase behaviors have automated verification."*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
