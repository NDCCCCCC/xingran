---
phase: 73
phase_name: P1 重要补齐
slug: p1-pending
date: 2026-08-21
nyquist_dimensions: 8
---

# Phase 73: Validation Strategy (Nyquist)

**Phase:** 73 — P1 重要补齐
**Date:** 2026-08-21
**Nyquist:** ENABLED (8 dimensions tracked)

## Validation Source-of-Truth

| Source | Path | Authority |
|--------|------|-----------|
| ROADMAP.md §Phase 73 | `.planning/ROADMAP.md:130-152` | 8 SC(SC#1..8) |
| REQUIREMENTS.md | `.planning/REQUIREMENTS.md` §IMP-01..06 | 6 requirements |
| CONTEXT.md decisions | `.planning/phases/73-p1-pending/73-CONTEXT.md` §D-01..D-12 | 12 锁定决策 |
| RESEARCH.md validation | `.planning/phases/73-p1-pending/73-RESEARCH.md` §Validation Architecture | 8 维度映射 |
| Coverage baseline | `.planning/coverage-baseline.md` | 21.5% baseline + Phase 73 后 ratchet |

## Nyquist 8-Dimension Coverage Map

| Dim | Name | Phase 73 验证策略 | Source of Truth |
|-----|------|------------------|-----------------|
| **D1** | Functional Correctness | 表驱动 TC 覆盖正常路径 + 边界 + 异常 | 每个 test 文件 TC1/TC2/... |
| **D2** | API Contract | handler 测试断言 `code=0` (success) / `code != 0` (error) | CONTEXT.md D-02 + CLAUDE.md API Response Format |
| **D3** | Error Handling | service 测试覆盖错误分支(返回 error / 写日志) | CONTEXT.md D-02 (portwrite + ad_account 双范本) |
| **D4** | Boundary Cases | 空 DB / 空输入 / 重复请求 / 大数据量 | 每个 test 文件边界 TC |
| **D5** | Security | JWT token 工厂 + 真实 SM4 cipher 加密路径 | CONTEXT.md D-09(真实中间件 + 真实加密) |
| **D6** | Performance | 不强制 phase 73 性能测试(Nyquist 允许跨期) | N/A |
| **D7** | Observability | operlog.Record 被调用断言(handler 层) | CLAUDE.md operlog convention |
| **D8** | Validation Strategy | **本文件即来源** + RESEARCH.md Validation Architecture | `.planning/phases/73-p1-pending/73-VALIDATION.md` |

## Phase 73 Plan → Dimension 映射

| Plan | Stmts | D1 | D2 | D3 | D4 | D5 | D6 | D7 | D8 |
|------|-------|----|----|----|----|----|----|----|----|
| 73-01 (handler 简单:duty+knowledge) | 538 | ✓ | ✓ | ✓ | ✓ | ✓ | — | ✓ | ✓ |
| 73-02 (handler 复杂:rpa+vdi) | 910 | ✓ | ✓ | ✓ | ✓ | ✓ | — | ✓ | ✓ |
| 73-03 (service 简单:duty+knowledge+network) | 326 | ✓ | — | ✓ | ✓ | — | — | — | ✓ |
| 73-04 (service 中等:monitor) | 485 | ✓ | — | ✓ | ✓ | — | — | — | ✓ |

## CI Gate 验证链

```
local dev:
  go test -count=1 ./internal/api/v1/duty/... ./internal/api/v1/knowledge/...
    ↓ exit 0 + per-package ≥70%
  go test -count=1 ./internal/api/v1/rpa/... ./internal/api/v1/vdi/...
    ↓ exit 0 + per-package ≥70%
  go test -count=1 ./internal/services/duty/... ./internal/services/knowledge/... ./internal/services/network/...
    ↓ exit 0 + per-package ≥70%
  go test -count=1 ./internal/services/monitor/...
    ↓ exit 0 + per-package ≥70%
  go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...
    ↓ exit 0 + 全包无失败

CI:
  gh run watch <run-id>
    ↓ CI backend Coverage gate PASSED
    ↓ per-package 验证通过(扩展的 check-coverage.sh)
```

## Coverage Threshold Ratchet

| Stage | Value | Source |
|-------|-------|--------|
| 起点 (Phase 72 后) | 21.5% | `.coverage-threshold` 文件 |
| Phase 73 后(预估) | 26-30% | 8 个 P1 包 2259 stmts × 70% = +1581 covered / 43652 total |
| Phase 74 后(目标) | ≥70% | SCALE-01..03 大块补齐 |

## Phase 73 SC → 验证手段映射

| SC | 描述 | 验证手段 | 计划 |
|----|------|---------|------|
| SC#1 | api/v1/duty ≥70% (265 stmts) | `go test ./internal/api/v1/duty/...` + cover | 73-01 |
| SC#2 | api/v1/knowledge ≥70% (273 stmts) | `go test ./internal/api/v1/knowledge/...` + cover | 73-01 |
| SC#3 | api/v1/rpa ≥70% (612 stmts) | `go test ./internal/api/v1/rpa/...` + cover | 73-02 |
| SC#4 | api/v1/vdi ≥70% (298 stmts) | `go test ./internal/api/v1/vdi/...` + cover | 73-02 |
| SC#5 | services/duty + knowledge ≥70% | `go test ./internal/services/{duty,knowledge}/...` + cover | 73-03 |
| SC#6 | services/monitor + network ≥70% | `go test ./internal/services/{monitor,network}/...` + cover | 73-03 + 73-04 |
| SC#7 | 加权平均 ratchet 到实际值 | `bash .github/scripts/check-coverage.sh` + `.coverage-threshold` amend | 全部 4 plans 完成后原子 commit |
| SC#8 | 零业务代码改动 | `git diff --stat <phase-start>..<phase-end>` 仅 test files | 全部 4 plans 自验 |

## Phase 73 8-Dimension Self-Audit (executor 提交前必填)

| Dim | 73-01 | 73-02 | 73-03 | 73-04 |
|-----|-------|-------|-------|-------|
| D1 | ___ | ___ | ___ | ___ |
| D2 | ___ | ___ | ___ | ___ |
| D3 | ___ | ___ | ___ | ___ |
| D4 | ___ | ___ | ___ | ___ |
| D5 | ___ | ___ | ___ | ___ |
| D6 | N/A | N/A | N/A | N/A |
| D7 | ___ | ___ | ___ | ___ |
| D8 | ✓ | ✓ | ✓ | ✓ |

每个 plan 的 SUMMARY.md 必须包含此表的填写(PASS/FAIL/SKIP)。

---

*Phase 73 Validation Strategy — derived from Nyquist 8-dimension framework + Phase 71/72 patterns*
