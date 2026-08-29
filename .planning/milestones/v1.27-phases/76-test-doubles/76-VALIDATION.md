---
phase: 76
slug: test-doubles
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-08-23
---

# Phase 76 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — test-only deps (miniredis/v2, httpmock) installed by 76-01 Task 1 |
| **Quick run command** | `go test ./pkg/cache/ ./internal/device/` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~120 seconds (task-level package runs: <30s each) |

---

## Sampling Rate

- **After every task commit:** Run that task's target package (see Per-Task Map below; all < 30s)
- **After every plan/wave:** Run `go test ./...` + `bash scripts/check-ci-local.sh backend`（plan 收尾门，见各 PLAN verification 块）
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** task 级 ≤ 30s；全量门 ~120s（只在 plan/wave 收尾跑，不嵌入 per-task verify）

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 76-01-01 | 01 | 1 | INFRA-01 | T-76-01-SC | N/A | unit | `[ "$(grep -c "// test-only (v1.27 D-02)" go.mod)" -eq 2 ] && go build ./...` | —（go.mod 编辑） | ⬜ pending |
| 76-01-02 | 01 | 1 | INFRA-01 | T-76-01-02 | N/A | unit | `go test -count=1 ./pkg/cache/` | ❌ new（redis_miniredis_76_01_test.go） | ⬜ pending |
| 76-01-03 | 01 | 1 | INFRA-01 | T-76-01-02 | N/A | unit | `go test -count=1 -run 'TestGeocoding' ./internal/services/operations/` | ❌ new（geocoding_httpmock_76_01_test.go） | ⬜ pending |
| 76-02-01 | 02 | 2 | INFRA-02 | T-76-02-01 | N/A | unit | `go build ./... && go test -count=1 ./internal/device/` | —（生产重构，既有测试守护） | ⬜ pending |
| 76-02-02 | 02 | 2 | INFRA-02 | T-76-02-04 | N/A | unit | `go test -count=1 -run 'TestDriverFactory' ./internal/device/ && go test -count=1 ./internal/device/` | ❌ new（driver_factory_76_02_test.go） | ⬜ pending |
| 76-03-01 | 03 | 2 | INFRA-03 | T-76-03-02 | N/A | unit | `go build ./... && go vet ./internal/services/addomain/ && go test -count=1 ./internal/services/addomain/` | ✅（ldap_client_mock_test.go 既有） | ⬜ pending |
| 76-03-02 | 03 | 2 | INFRA-03 | T-76-03-01 | N/A | unit | `go build ./...` + 双 grep 零残留门（闭包/辅助签名）+ `go test -count=1 ./internal/services/addomain/... ./internal/scheduler/... ./internal/core/security/...` | —（24 处机械替换） | ⬜ pending |
| 76-03-03 | 03 | 2 | INFRA-03 | T-76-03-03 | N/A | unit | `go test -count=1 -run 'TestFailover' ./internal/services/addomain/ && go test -count=1 ./internal/services/addomain/` | ❌ new（failover_client_76_03_test.go） | ⬜ pending |
| 76-04-01 | 04 | 2 | INFRA-04 | T-76-04-01 | N/A | unit | `go test -count=1 -run 'TestSubprocessStub' ./internal/agent/server/ -v && go test -count=1 ./internal/agent/server/` | ❌ new（subprocess_stub_test.go） | ⬜ pending |
| 76-04-02 | 04 | 2 | INFRA-04 | T-76-04-02 | N/A | unit | `! grep -n '"echo"' internal/agent/server/subprocess_pgroup_test.go && go test -count=1 ./internal/agent/server/ -v && git diff --quiet HEAD -- internal/agent/server/subprocess.go` | ✅（既有文件改写） | ⬜ pending |
| 76-05-01 | 05 | 3 | INFRA-05 | T-76-05-02 | N/A | unit | `go test -count=1 -run 'TestNoProductionForTestingReferences' ./internal/device/ -v && go test -count=1 ./internal/device/` | ❌ new（for_testing_guard_test.go） | ⬜ pending |
| 76-05-02 | 05 | 3 | INFRA-05 | T-76-05-01 | N/A | unit（注毒自证） | `git diff --quiet HEAD -- internal/device/connection_pool.go && go test -count=1 -run 'TestNoProductionForTestingReferences' ./internal/device/` | ✅（Task 1 文件） | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

无独立 Wave 0——test-only 依赖的安装与锚定就是 76-01 的 Task 1（go.mod 落地）与 Task 3（httpmock PoC tidy 保活），随 wave 1 首个 plan 一体执行，无需前置步骤。

- [x] `go get github.com/alicebob/miniredis/v2@v2.38.0` — 由 76-01 Task 1 覆盖（INFRA-01）
- [x] `go get github.com/jarcoal/httpmock@v1.4.2` — 由 76-01 Task 1 覆盖（INFRA-01）
- [x] geocoding httpmock PoC test — 由 76-01 Task 3 覆盖（锚定 go.mod，tidy 保活）

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| ubuntu CI double-green, no Docker | INFRA-01 | Local Windows cannot verify CI runner | Push branch, confirm `.github/workflows/ci.yml` go test job passes on ubuntu without Docker |

*If none: "All phase behaviors have automated verification."*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references（无独立 Wave 0，由 76-01 Task 1/3 覆盖）
- [x] No watch-mode flags
- [x] Feedback latency：task 级 ≤ 30s，全量门 ~120s 仅 plan/wave 收尾
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** signed — 2026-08-23（checker 修订轮：Per-Task Map 补齐至 13 行、修正 76-04/76-05 命令与 wave 列、全量测试移至 plan 收尾门后签收）
