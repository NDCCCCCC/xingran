---
phase: 91
slug: crud-base-repository-t-p1
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-09-04
---

# Phase 91 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (Go 1.24, testify + glebarez sqlite in-memory) |
| **Config file** | none — existing infrastructure (v1.26/v1.27 coverage stack) |
| **Quick run command** | `go test ./internal/services/operations/... ./internal/services/base/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~60s quick / ~300s full |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/services/operations/... ./internal/services/base/...`
- **After every plan wave:** Run `go test ./...` + `go build ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 91-01-* | 01 | 1 | CRUD-REUSE-01 | — | N/A | unit | `go test ./internal/services/base/...` | ✅ base_80_05_test.go (适配重写) | ⬜ pending |
| 91-02-* | 02 | 2 | CRUD-REUSE-02 | — | N/A | unit + behavior-lock | `go test ./internal/services/operations/ -run Workstation` | ✅ 533 行行为锁测试（改写） | ⬜ pending |
| 91-03-* | 03 | 3 | CRUD-REUSE-03..05 | — | N/A | unit | `go test ./internal/services/operations/` | ✅ crud_services_test.go + building/floor/asset 各自测试 | ⬜ pending |
| 91-04-* | 04 | 3+ | CRUD-REUSE-06..08 | — | N/A | unit + LOC audit | `go test ./internal/services/operations/...` + `git diff --stat` | ✅ crud_services_test.go | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*注：具体 Task 粒度映射在 PLAN.md 生成后由 planner 细化；上表为 plan 级映射。*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements:
- [x] `internal/services/base/base_80_05_test.go` — GORMRepository 泛型测试（改造后适配重写）
- [x] `internal/services/operations/crud_services_test.go` — wall/door/server_room/dedicated_line/infopoint CRUD 行为锁
- [x] sqlite in-memory 测试基建（v1.27 D-27-01..04）
- [x] `go build ./...` 编译验证链

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| building/asset 软删 Total 计数修复（Count 不滤软删 → 滤） | CRUD-REUSE-03/05 | 微小行为变更，语义需人确认（RESEARCH F3 建议 checkpoint:human-verify） | 对比迁移前后 deleted 行存在时 List 返回的 Total 值 |
| workstation 分页上限语义（如收紧） | CRUD-REUSE-02 | D-05 涉及 map→typed 接线时 extractPagination 上限是否变化需人确认 | 验证 pageSize=10000+ 请求在迁移前后行为一致 |

*若 plan 最终零行为变更则以上两项转为 N/A。*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
