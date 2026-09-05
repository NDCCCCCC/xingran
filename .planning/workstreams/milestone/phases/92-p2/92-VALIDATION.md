---
phase: 92
slug: p2
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-09-05
---

# Phase 92 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `92-RESEARCH.md` § Validation Architecture (2026-09-05, HIGH confidence).

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test（标准）+ testify + miniredis/v2.38.0 + glebarez/sqlite |
| **Config file** | none（go.mod 驱动；无独立 test config） |
| **Quick run command** | `go build ./... && go test ./internal/services/base/... ./internal/services/system/... ./internal/services/operations/...` |
| **Full suite command** | `go test ./internal/services/...`（SC-4）/ 全量 `go test ./...` |
| **Estimated runtime** | ~60-120 seconds（三包 quick）/ 全量 ~5-8 minutes |

---

## Sampling Rate

- **After every task commit:** Run `go build ./...` + quick 命令（base + system + operations 三包）
- **After every plan wave:** Run `go test ./internal/services/...`（SC-4 口径）
- **Before `/gsd:verify-work`:** Full suite must be green（全量 `go test ./...` + operlog/status AST 锁值防线保持绿）
- **Max feedback latency:** ~120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (planner 填充) | 01 | 1 | CACHE-UNIFY-01 | 缓存键注入（key 逐字不变） | 键构造 diff 级零变更 | unit（fake provider 纯函数） | `go test ./internal/services/base/ -run TestCache -v` | ❌ W0 | ⬜ pending |
| (planner 填充) | 01 | 1 | CACHE-UNIFY-05 | 失效遗漏 | pattern+key 失效 + NoOp 透传 | integration（miniredis） | `go test ./internal/services/base/ -v` | ❌ W0 | ⬜ pending |
| (planner 填充) | 02 | 2 | CACHE-UNIFY-02 | — | 行为不变（29 处迁移） | regression | `go test ./internal/services/system/ -run "Cache" -v` | ✅ | ⬜ pending |
| (planner 填充) | 03 | 3 | CACHE-UNIFY-03 | — | 分发语义不变 | regression + unit | `go test ./internal/services/operations/ -run "Cache\|Floor" -v` | ✅ | ⬜ pending |
| (planner 填充) | 03 | 3 | CACHE-UNIFY-04 | — | GetExpiration 委托行为不变 | regression | `go test ./internal/services/ -run "Dcs7901\|Ccs7901" -v` | ✅ | ⬜ pending |
| (planner 填充) | (任一) | — | D-10② | — | 无 interface{} 闭包式 GetOrSet 残留 | AST/regex 扫描（warning 不 fail） | `go test ./internal/services/system/ -run TestNoInterfaceGetOrSetResidue -v` | ❌ W0 | ⬜ pending |
| (planner 填充) | (全) | — | SC-4 | — | 1688+ 测试 0 回归 | full regression | `go test ./internal/services/...` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/services/base/cache_service_base_test.go` — 覆盖 CACHE-UNIFY-01/05：泛型函数族 hit/miss/TTL(FastForward)/失效/NoOp 透传/GetStats + nil-config 分支（REQUIREMENTS 原文命名的文件）
- [ ] （可选并入上文件）base 包 alias 翻转后 `var _` 编译期断言
- [ ] invariants 扫描测试（system 或 pkg 侧，warning-only，Phase 89/90 模式）
- [ ] Framework install: 无需（全部既有）

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| (none) | — | D-09 决策：SC-5 端到端验证以 miniredis 自动化落地，无强制手动项 | — |

*All phase behaviors have automated verification (per D-09).*

---

## Phase Gate（verify 前置）

- 全量 `go test ./...` 绿
- `internal/utils/operlog/regression_test.go` + `internal/models/status_constants_test.go` AST 锁值防线保持绿
- LOC numstat 双口径报告（样板毛减口径 + repo 全口径，沿 OVR-91-01 先例）
- invariants 扫描输出：interface{} 闭包式 GetOrSet 残留 = 0

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
