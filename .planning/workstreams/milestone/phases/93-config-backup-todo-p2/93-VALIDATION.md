---
phase: 93
slug: config-backup-todo-p2
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-09-05
---

# Phase 93 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> 验证需求 V1..V9 定义见 `93-RESEARCH.md` § Validation Architecture。

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (Go 1.24) + testify；前端 vitest（仅 V8 前端切片） |
| **Config file** | none — 沿用既有 go.mod / vitest 配置 |
| **Quick run command** | `go test ./internal/services/ -run "TestCbk93" -count=1` |
| **Full suite command** | `go test ./internal/services/... -count=1` |
| **Estimated runtime** | quick ~20s / full ~180s |

---

## Sampling Rate

- **After every task commit:** Run `go build ./...` + quick run command
- **After every plan wave:** Run `go test ./internal/services/... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green（含 `go test ./...` 0 失败，v1.29 D-03）
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

> Task ID 为占位——planner 生成 PLAN 后以实际 task 编号回填映射（映射关系按 V1..V9 → 任务归属）。

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (93-XX-0Y) | 01 | 1 | BACKUP-CLOSED-01 (V1) | T-93-02 | 压缩产物仅落备份目录 | unit/roundtrip | `go test ./internal/services/ -run TestCbk93Compress` | ❌ W0 | ⬜ pending |
| (93-XX-0Y) | 01 | 1 | BACKUP-CLOSED-02 (V2) | — | 损坏流拒绝解析 | unit | `go test ./internal/services/ -run TestCbk93Decompress` | ❌ W0 | ⬜ pending |
| (93-XX-0Y) | 02 | 2 | BACKUP-CLOSED-03/05 (V3) | T-93-01 | 下发仅限同设备+互斥 | e2e (FileTransport) | `go test ./internal/services/ -run TestCbk93Restore` | ❌ W0 | ⬜ pending |
| (93-XX-0Y) | 02 | 2 | D-10/D-14/D-15 (V6/V7) | T-93-01 | 无备份不下发；状态机 | unit/e2e | `go test ./internal/services/ -run TestCbk93Task` | ❌ W0 | ⬜ pending |
| (93-XX-0Y) | 03 | 2 | BACKUP-CLOSED-04 (V4) | — | 失败场景全覆盖 | unit | `go test ./internal/services/ -run TestCbk93Failure` | ❌ W0 | ⬜ pending |
| (93-XX-0Y) | 03 | 2 | D-17/D-19 (V8) | — | handler taskId 契约 | unit+type-check | `go test ./internal/api/v1/network/...` + `npm run type-check` | ✅ | ⬜ pending |
| (93-XX-0Y) | 03 | 2 | D-13/D-32/D-33 (V9) | — | 锁值防线不回归 | regression | `go test ./internal/utils/operlog/ ./internal/models/` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/services/config_backup_service_93_01_test.go` — V1/V2 压缩解压 stubs（93_NN 系列首个文件）
- [ ] 79_06 helper 复用确认（newCbk7906 / cbk7906Chdir / writeFixture7906 同包可见——同包 `services` 无需导出）

*Existing infra covers the rest（sqlite in-memory + FileTransport fixture 基建已就绪）。*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 真机恢复（华为/H3C/锐捷实机下发） | BACKUP-CLOSED-03 生产语义 | FileTransport 无法覆盖真实设备 CLI 差异 | deferred——仿 v1.18/19 site-visit UAT 模式另行登记（owner = 现场运维同事），非本期验收门槛（D-30） |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
