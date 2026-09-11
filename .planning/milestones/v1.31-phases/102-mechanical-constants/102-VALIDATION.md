---
phase: 102
slug: mechanical-constants
status: planned
nyquist_compliant: true
wave_0_complete: true
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
| 102-01-T1 | 01 | 1 | CACHE-01① | T-102-01/02 | 键值逐字符等价 | unit（快照表驱动） | `go test ./pkg/constants/ -run TestCaptchaCacheKeyEquivalence -count=1` | ❌→✅ W0（本任务交付） | ⬜ pending |
| 102-01-T2+T3 | 01 | 1 | CACHE-01 位点替换 | T-102-01 | 行为等价 | regression（既有 core 套件） | `go test ./internal/core/ -count=1` | ✅ 既有 | ⬜ pending |
| 102-02-T1 | 02 | 2 | CACHE-02①（8 模块注册） | T-102-03/04 | 注册值 == 原字面量 | unit（快照 + helper 输出断言） | `go test ./internal/services/system/ -run TestCacheKeyEquivalence -count=1` | ❌→✅ W0（本任务交付） | ⬜ pending |
| 102-02-T2 | 02 | 2 | CACHE-02①（根包 2 格式注册） | T-102-03 | 注册值 == 原字面量 | unit | `go test ./pkg/constants/ -count=1` | ✅（Plan 01 文件追加） | ⬜ pending |
| 102-03-T1+T2 | 03 | 3 | CACHE-02 调用点替换 | T-102-06 | 键值/TTL/调用形态三不变 | regression（六包套件） | `go test ./internal/services/{system,duty,workorder,knowledge,network,rpa}/ -count=1` | ✅ 既有 | ⬜ pending |
| 102-03-T3 | 03 | 3 | CACHE-01② + CACHE-02② | T-102-06/07/08 | 12 文件内联清零 | unit（AST 扫描硬失败，窄扫 D-102-7） | `go test ./internal/services/system/ -run TestCacheKeyInlineResidue -count=1` | ❌→✅ W0（本任务交付） | ⬜ pending |
| 102-04-T1+T2 | 04 | 1 | STATUS-01① | T-102-09/10 | 行为等价（占位符 SQL） | regression（既有套件） | `go test ./internal/scheduler/ ./internal/services/scheduler/ ./internal/services/workorder/ -count=1` | ✅ 既有 | ⬜ pending |
| 102-04-T3（值锁部分） | 04 | 1 | STATUS-01② | T-102-10 | models 值锁不倒退 + WorkOrderStatus 补登记 | unit（AST 值锁） | `go test ./internal/models/ -run TestStatusConstants -count=1` | ✅ status_constants_test.go | ⬜ pending |
| 102-04-T3（扫描部分） | 04 | 1 | STATUS-01③ | T-102-11 | 使用点硬失败 + 白名单 2 条 | unit（AST 使用点扫描，7 形态） | `go test ./internal/models/ -run TestNoStatusLiteralUsage -count=1` | ❌→✅ W0（本任务交付） | ⬜ pending |
| 102-05-T1 | 05 | 1 | PAGI-01 | T-102-12/13 | cap=100 行为等价 | regression（既有 handler 测试） | `go test ./internal/api/v1/system/ -run TestFile -count=1` | ✅ file_handler_test.go + file_service_test.go | ⬜ pending |
| 102-05-T2 | 05 | 1 | PAGI-01② | — | utils 分页符号归零 | 编译期守护 | `go build ./...` | ✅ 删除即生效 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*
*Task ID 已由 planner 回填（2026-09-07，随 102 plan-phase）。*

---

## Wave 0 Requirements

- [x] `pkg/constants/cache_102_test.go`— CACHE-01① captcha 键等价快照（Plan 102-01 T1 交付；根包 2 格式由 102-02 T2 追加）
- [x] `internal/services/system/cache_keys_102_test.go` — CACHE-02① 键等价快照（102-02 T1）+ ② 12 文件内联扫描 TestCacheKeyInlineResidue（102-03 T3，invariants_92 同款骨架）
- [x] `internal/models/status_constants_test.go` 扩展（D-102-6 明文同文件）— STATUS-01③ 使用点扫描 TestNoStatusLiteralUsage + 白名单表（102-04 T3；白名单 2 条 = geocoding + sqlite 视图 DDL，较原稿增补）
- [x] 无框架安装缺口 — go test 原生 + testify 既有

*守护/等价测试文件为本相交付物（Wave 0 = 首个 plan 的守护测试任务），非预置 stub。*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** planned（planner 回填 2026-09-07——Task ID 已映射 5 plans / 13 tasks）
