---
phase: 91
slug: crud-base-repository-t-p1
status: approved
nyquist_compliant: true
wave_0_complete: true
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
| 91-01-T1 | 01 | 1 | CRUD-REUSE-01 | TM-01 (行为回归) | N/A | unit | `go test ./internal/services/base/...` | ✅ | ⬜ pending |
| 91-01-T2 | 01 | 1 | CRUD-REUSE-01 | TM-01 | N/A | unit | `go test ./internal/services/operations/...`（alias 兼容） | ✅ | ⬜ pending |
| 91-01-T3 | 01 | 1 | CRUD-REUSE-07 | — | N/A | contract-lock | `go test ./internal/services/base/ -run TestRepo` | ✅ 新建 service_test.go | ⬜ pending |
| 91-02-T1 | 02 | 2 | CRUD-REUSE-02 | TM-02 | N/A | unit | `go build ./internal/services/operations/... ./internal/api/v1/operations/requests/` | ✅ | ⬜ pending |
| 91-02-T2 | 02 | 2 | CRUD-REUSE-02, 07 | TM-02 | N/A | unit + behavior-lock | `go test ./internal/services/operations/ -run TestImp77 -v` + handler 包套件 | ✅ 改写 3 测试文件 | ⬜ pending |
| 91-02-T3 | 02 | 2 | CRUD-REUSE-02 | — | 分页收紧语义 | checkpoint:human-verify | A2 checkpoint 人工确认 | — | ⬜ pending |
| 91-03-T1 | 03 | 3 | CRUD-REUSE-04, 07 | — | N/A | behavior-baseline | floor 行为基线测试（Wave 0 缺口补齐） | ✅ 新建 | ⬜ pending |
| 91-03-T2 | 03 | 3 | CRUD-REUSE-03, 05 | TM-03 (软删可见性) | F3 软删 Total | unit + checkpoint | `go test ./internal/services/operations/` | ✅ | ⬜ pending |
| 91-03-T3 | 03 | 3 | CRUD-REUSE-04 | — | N/A | unit | `go test ./internal/services/operations/`（P7 装饰器签名锁定） | ✅ | ⬜ pending |
| 91-03-T4 | 03 | 3 | CRUD-REUSE-03, 05 | — | F3/A3 软删语义 | checkpoint:human-verify | A3 checkpoint 人工确认 | — | ⬜ pending |
| 91-04-T1 | 04 | 4 | CRUD-REUSE-06 | — | N/A | unit | `go test ./internal/services/operations/ -run 'Door|Wall'` | ✅ crud_services_test.go | ⬜ pending |
| 91-04-T2 | 04 | 4 | CRUD-REUSE-06, 07 | — | N/A | unit + smoke | `go test ./internal/services/operations/ -run 'ServerRoom|DedicatedLine|FloorPlanText'` | ✅ fpt 冒烟补齐 | ⬜ pending |
| 91-04-T3 | 04 | 4 | CRUD-REUSE-06 | — | N/A | unit | `go test ./internal/services/operations/ -run 'RoomDevice|InfoPoint'` | ✅ | ⬜ pending |
| 91-04-T4 | 04 | 4 | CRUD-REUSE-06, 07, 08 | — | N/A | LOC audit + docs sync + full gate | `git diff --stat` numstat 审计 + `go test ./...` + REQ_SYNC_OK grep | ✅ SUMMARY | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*Task 粒度与 91-{01..04}-PLAN.md（修订版 commit 2872234）一致。*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements:
- [x] `internal/services/base/base_80_05_test.go` — GORMRepository 泛型测试（91-01-T1 适配重写）
- [x] `internal/services/operations/crud_services_test.go` — wall/door/server_room/dedicated_line/infopoint CRUD 行为锁
- [x] sqlite in-memory 测试基建（v1.27 D-27-01..04）
- [x] `go build ./...` 编译验证链
- [x] floor 行为基线测试（91-03-T1 执行期补齐）+ fpt 冒烟（91-04-T2 执行期补齐）— 已在 plans 内显式承接

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| workstation 分页收紧（A2: MaxOptionsPageSize 10000→100 + 负 offset 修复） | CRUD-REUSE-02 | 语义变更需人确认（91-02-T3 checkpoint） | 对比迁移前后 pageSize=10000+ 请求行为 |
| building/asset 软删 Total 修复（A3/F3: Count 不滤软删 → 滤） | CRUD-REUSE-03/05 | 微小行为变更，数据可见性语义需人确认（91-03-T4 checkpoint） | deleted 行存在时 List 返回的 Total 对比迁移前后 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies（2 个 checkpoint 任务为显式 human-verify，符合 checkpoint 协议）
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 120s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-09-04（plans 修订版 2872234 结构校验通过后）
