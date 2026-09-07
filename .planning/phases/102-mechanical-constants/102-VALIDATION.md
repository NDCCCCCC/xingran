---
phase: 102
slug: mechanical-constants
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-09-07
---

# Phase 102 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify（assert/require，仓库既有） |
| **Config file** | none — go test 原生；守护测试自含 AST 扫描器 |
| **Quick run command** | `go test ./internal/services/system/ ./internal/models/ ./internal/core/ ./internal/scheduler/ ./pkg/constants/ -count=1` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~60 seconds（全量）；quick < 30s |

---

## Sampling Rate

- **After every task commit:** `go build ./...` + 触及包的 quick run（<30s）
- **After every plan wave:** `go test ./...`（0 失败）
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds
- **Phase gate:** 全量套件绿 + 后端 coverage ≥78.33% + 七 gate 不倒退（本相纯后端，前端 gate 应零变化）

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | 01 | 1 | CACHE-01① | — | N/A | unit（快照表驱动） | `go test ./pkg/constants/ -run TestCaptchaCacheKeyEquivalence -count=1` | ❌ W0 | ⬜ pending |
| TBD | 01 | 1 | CACHE-01② | — | N/A | unit（AST 扫描硬失败） | `go test ./internal/services/system/ -run TestCacheKeyInlineResidue -count=1` | ❌ W0 | ⬜ pending |
| TBD | 01 | 1 | CACHE-02① | — | N/A | unit（键值等价快照） | `go test ./internal/services/system/ -run TestCacheKeyEquivalence -count=1` | ❌ W0 | ⬜ pending |
| TBD | 01 | 1 | CACHE-02② | — | N/A | unit（同②扫描器，窄扫 12 文件） | 同 TestCacheKeyInlineResidue | ❌ W0 | ⬜ pending |
| TBD | 01 | 1 | STATUS-01① | — | N/A | regression（既有套件） | `go test ./internal/scheduler/ ./internal/services/... -count=1` | ✅ 既有 | ⬜ pending |
| TBD | 01 | 1 | STATUS-01② | — | N/A | unit（AST 值锁，既有） | `go test ./internal/models/ -run TestStatusConstants -count=1` | ✅ status_constants_test.go | ⬜ pending |
| TBD | 01 | 1 | STATUS-01③ | — | N/A | unit（AST 使用点扫描 + geocoding 白名单） | `go test ./internal/models/ -run TestNoStatusLiteralUsage -count=1` | ❌ W0 | ⬜ pending |
| TBD | 01 | 1 | PAGI-01 | — | N/A | regression（既有 handler/service 测试） | `go test ./internal/api/v1/system/ ./internal/services/system/ -run TestFile -count=1` | ✅ file_handler_test.go + file_service_test.go | ⬜ pending |
| TBD | 01 | 1 | PAGI-01② | — | N/A | 编译期守护（utils 分页文件删除后 build 即守护） | `go build ./...` | ✅ 删除即生效 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*
*Task ID 列在 PLAN.md 生成后由 planner/executor 回填。*

---

## Wave 0 Requirements

- [ ] `pkg/constants/cache_102_test.go`（建议名）— CACHE-01① captcha 键等价快照
- [ ] `internal/services/system/cache_keys_102_test.go`（建议名）— CACHE-02① 键等价快照 + ② 12 文件内联扫描（invariants_92 同款骨架）
- [ ] `internal/models/status_constants_test.go` 扩展（D-102-6 明文同文件）— STATUS-01③ 使用点扫描 + geocoding 白名单表
- [ ] 无框架安装缺口 — go test 原生 + testify 既有

*守护/等价测试文件为本相交付物（Wave 0 = 首个 plan 的守护测试任务），非预置 stub。*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
