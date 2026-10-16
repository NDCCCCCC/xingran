---
phase: 51
slug: w2-portwriteservice-batch-orchestrator-mock-tests
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-06
---

# Phase 51 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `testing` (Go stdlib) + `github.com/stretchr/testify/assert` + `github.com/stretchr/testify/mock` |
| **Config file** | None — Go test convention; `go test ./...` auto-discovers |
| **Quick run command** | `go test ./internal/services/portwrite/... -count=1` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~10 seconds (all mocks, no DB / no real SSH) |

---

## Sampling Rate

- **After every task commit:** `go test ./internal/services/portwrite/... -count=1`
- **After every plan wave:** `go test ./...` + `go vet ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 51-01-01 | 01 | 1 | SSH-02 | T-51-01 | parseConfigError 不泄露 SSH 内部栈 | unit (table-driven) | `go test ./internal/services/portwrite/... -run TestParseConfigError -v` | ❌ W0 | ⬜ pending |
| 51-01-02 | 01 | 1 | SSH-03 | — | ExecuteCustom 复用同 device 连接 | unit (mock executor) | `go test ./internal/services/portwrite/... -run TestShutdown -v` | ❌ W0 | ⬜ pending |
| 51-01-03 | 01 | 1 | SSH-04 | T-51-04 | detached 30min context 不被 HTTP ctx 切断 | unit (mock) | `go test ./internal/services/portwrite/... -run TestBatchWritePorts_DetachedContext -v` | ❌ W0 | ⬜ pending |
| 51-01-04 | 01 | 1 | PORT-06 | T-51-03 | pre-state NoOp 路径；DB 缺失 fallback | unit | `go test ./internal/services/portwrite/... -run TestPreState -v` | ❌ W0 | ⬜ pending |
| 51-01-05 | 01 | 1 | BATCH-04 | T-51-02 | maxBatchSize=50 cap 入口拒绝 | unit | `go test ./internal/services/portwrite/... -run TestBatchWritePorts_ExceedsMax -v` | ❌ W0 | ⬜ pending |
| 51-01-06 | 01 | 1 | BATCH-02 | — | fail-fast 立即停 | unit | `go test ./internal/services/portwrite/... -run TestBatchWritePorts_FailFast -v` | ❌ W0 | ⬜ pending |
| 51-01-07 | 01 | 1 | BATCH-03 | — | partial result 三数组 | unit | `go test ./internal/services/portwrite/... -run TestBatchWritePorts_PartialResult -v` | ❌ W0 | ⬜ pending |
| 51-01-08 | 01 | 1 | AUDIT-04 | T-51-04 | success 路径触发 Enqueue；失败路径不触发 | unit (mock collectionSvc) | `go test ./internal/services/portwrite/... -run TestShutdown_TriggersEnqueue -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/services/portwrite/port_write_service.go` — 6-method service interface + impl + DI 构造
- [ ] `internal/services/portwrite/batch_orchestrator.go` — BatchWriteRequest + BatchResult + 批量循环 + detached context
- [ ] `internal/services/portwrite/parse_error.go` — parseConfigError + WriteError + transport/rejection marker 常量
- [ ] `internal/services/portwrite/pre_state_check.go` — checkPreState 单端口 + 批量统一函数
- [ ] `internal/services/portwrite/port_write_service_test.go` — 单测（含 10+ parseConfigError 用例）
- [ ] Service 字段类型调整: `*device.DeviceExecutor` → `portWriteExecutor` interface（仅暴露 ExecuteCustom）
- [ ] Service 字段类型调整: `*services.DeviceInfoCollectionService` → `portWriteCollectionSvc` interface（仅暴露 Enqueue）

*Existing infrastructure (Phase 50) covers: RenderCommand vendor templates + RenderCommand 表驱动测试 + DevicePortStatus 模型.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| SSH-06 凭据解析 | SSH-06 | 由 device 层覆盖；service 不直调 | Phase 54 真机 UAT 验证 |
| 真实 marker 字串 | SSH-02 | 无真实设备 sample | Phase 54 site-visit 后补 `testdata/write_errors/*.txt` |

*If none: "All phase behaviors have automated verification."*

---

## Validation Sign-Off

- [ ] All tasks have automated verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending