---
phase: 54
slug: w5-e2e-real-device-uat-documentation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-07
---

# Phase 54 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Source: `54-RESEARCH.md` §Validation Architecture + §Phase Requirements → Test Map.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + `stretchr/testify v1.11.1` (assert + mock) + `gorm.io/driver/sqlite v1.5.4` (in-memory DB) |
| **Config file** | none — Go 标准 `go test`，无配置文件 |
| **Quick run command** | `go test ./internal/services/portwrite/ -run "TestE2E_" -count=1 -timeout=60s` |
| **Full suite command** | `go test ./... -count=1` (+ 前端 `cd xingran-react-frontend && npm run build && npm run type-check` for SC#6) |
| **Estimated runtime** | ~15s quick（portwrite 包）/ ~120s full Go suite / ~90s 前端 build+type-check |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/services/portwrite/ -count=1 -timeout=60s`（quick：portwrite 包含 Phase 51 mock + Phase 54 e2e）
- **After every plan wave:** Run `go test ./... -count=1`（full Go suite）
- **Before `/gsd:verify-work`:** 三绿全过才能进 verify： (1) `go test ./...` exit 0；(2) `cd xingran-react-frontend && npm run build` exit 0；(3) `cd xingran-react-frontend && npm run type-check` exit 0；(4) `go test ./internal/utils/operlog/ -run "TestOperType|TestRecordSignature|TestFilterSensitive"` exit 0
- **Max feedback latency:** 120 秒（full suite 上限）

---

## Per-Task Verification Map

> 任务 ID 由 planner 在 PLAN.md 细化；本表按 SC / 需求映射验证手段，planner 须为每条补全 task ID。

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 54-01-? | 01 | 1 | SC#1 / SSH-01..05 (5 single happy path) | — | N/A | e2e (service + scrapligo FileTransport) | `go test ./internal/services/portwrite/ -run "TestE2E_.*_HappyPath" -count=1 -timeout=60s` | ❌ W1 创建 | ⬜ pending |
| 54-01-? | 01 | 1 | SC#1 / BATCH-01..04 (1 batch happy path) | — | N/A | e2e batch | `go test ./internal/services/portwrite/ -run "TestE2E_Batch_.*HappyPath" -count=1` | ❌ W1 创建 | ⬜ pending |
| 54-01-? | 01 | 1 | SC#1 / SSH-02 transport_error | — | N/A | e2e error path | `go test ./internal/services/portwrite/ -run "TestE2E_.*TransportError" -count=1` | ❌ W1 创建 | ⬜ pending |
| 54-01-? | 01 | 1 | SC#1 / SSH-02 device_rejected | — | N/A | e2e error path | `go test ./internal/services/portwrite/ -run "TestE2E_.*DeviceRejected" -count=1` | ❌ W1 创建 | ⬜ pending |
| 54-01-? | 01 | 1 | SC#1 / BATCH-02 fail-fast | — | N/A | e2e batch | `go test ./internal/services/portwrite/ -run "TestE2E_Batch_.*FailFast" -count=1` | ❌ W1 创建 | ⬜ pending |
| 54-01-? | 01 | 1 | SC#1 / PORT-06 skipped (NoOp) | — | N/A | e2e NoOp path | `go test ./internal/services/portwrite/ -run "TestE2E_.*NoOp" -count=1` | ❌ W1 创建（Phase 51 已 unit 覆盖 PORT-06，e2e 可选补） | ⬜ pending |
| 54-01-? | 01 | 0 | SC#1 / A1 infra (device test-helper factory) | — | N/A | compile gate | `go build ./internal/device/... && go vet ./internal/device/...` | ❌ W0 创建 | ⬜ pending |
| 54-01-? | 01 | 2 | SC#2 API 文档（6 端点签名 + schema） | — | N/A | manual (docs) + grep | `grep -c "网络设备端口写操作" docs/API响应规范.md` (期望 ≥1) | ❌ W2 创建 | ⬜ pending |
| 54-01-? | 01 | 2 | SC#3 写端点保持 SM2+SM4 加密文档化 | T-54-01 | 写端点不裸传 {portId,reason}，保持 HTTP 加密；config.yaml exclude_paths 不含 /network/ports/write/* | manual (docs) + config grep | `grep -c "/network/ports/write" configs/config.yaml` (期望 0 = 未豁免) | ✅ config.yaml 已实证；❌ W2 docs 改 | ⬜ pending |
| 54-01-? | 01 | 2 | SC#4 UAT 推迟文档（6 项 + WR-02 观察） | — | N/A | manual (planning) | `test -f .planning/phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md && grep -c "pending" 54-HUMAN-UAT.md` (期望 ≥6) | ❌ W2 创建 | ⬜ pending |
| 54-01-? | 01 | 2 | SC#5 README + CHANGELOG + MILESTONES v1.19 | — | N/A | manual (docs) | `test -f CHANGELOG.md && grep -c "v1.19" CHANGELOG.md README.md .planning/MILESTONES.md` | ❌ W2 创建（CHANGELOG 新建） | ⬜ pending |
| 54-01-? | 01 | 2 | SC#6 全量回归三绿 | — | N/A | automated gate | `go test ./... && (cd xingran-react-frontend && npm run build && npm run type-check)` | ✅ infra exists；❌ phase 末跑 | ⬜ pending |
| 54-01-? | 01 | 2 | SC#7 operlog regression 不回归 | — | N/A | unit regression | `go test ./internal/utils/operlog/ -run "TestOperType|TestRecordSignature|TestFilterSensitive" -count=1` | ✅ Exists (`internal/utils/operlog/regression_test.go`) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/device/` 公开测试工厂（A1 决策已锁：device 包加公开 `NewPooledConnectionForTesting`/`ForE2E`，**无 build tag**）— 让 portwrite 测试包跨包构造 `*device.PooledConnection` 注入 FileTransport mock
- [ ] `internal/services/portwrite/port_write_e2e_test.go` — 第一个 task 用 1 个 happy path fixture 验证 `huawei_vrp.yaml` platform 加载 + channel strip prompt 正确（A2 假设验证），再扩展
- [ ] `internal/services/portwrite/testdata/*.fixture` — 6-8 个手写 Huawei VRP fixture（scrapligo `test-fixtures/` 只含 SSH 密钥对，无设备 IO fixture — Pitfall #3）

*Framework install：无 —— go.mod 已锁定所有 Go 依赖；前端 framework 在 Phase 53 已 audit。*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 真机 SSH 写命令（Huawei/H3C/Ruijie × shutdown/description/dot1x） | SC#4 / SSH-01..05 真机部分 | 需现场设备 + 运维操作；SC#4 已显式 deferral | 推迟至 `54-HUMAN-UAT.md` site visit，owner = 现场运维同事；本 phase 仅创建 pending 文档 |
| WR-02 custom-reason 使用频率观察 | SC#4 / Phase 55 依赖 | 需现场观察运维实际操作习惯 | `54-HUMAN-UAT.md` 加观察条目，结果驱动 Phase 55 WR-02 修/不修决策 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
