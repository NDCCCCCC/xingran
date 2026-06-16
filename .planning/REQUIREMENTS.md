---
type: requirements
project: RuoYi-Next
milestone: v1.16
status: planning
last_updated: 2026-06-16
---

# Requirements: v1.16 (planning) — Post-Phase 34 review closure

**Project:** RuoYi-Next 网络运维管理系统
**Milestone:** v1.16 (planning)
**Status:** Planning — fresh requirements for the next milestone
**Last Updated:** 2026-06-16
**Previous:** v1.5 archive at `.planning/milestones/v1.5-REQUIREMENTS.md`

---

## Carry-over Validation (read-only)

The 16 v1.5 requirements (DATA-01..04, QUERY-01..04, UI-01..04, PERF-01..04)
were all completed by Phase 12-15 (SHIPPED 2026-06-15). See archive for
full text and traceability. The verification chain:

| REQ-ID | Phase | Status | Evidence |
|--------|-------|--------|----------|
| DATA-01..02 | 12 | ✅ Complete | Phase 12 SUMMARY + 5/5 plans |
| QUERY-01..04 | 13 | ✅ Complete | Phase 13 SUMMARY + 6/6 plans |
| UI-01..04 | 14 | ✅ Complete | Phase 14 SUMMARY + 10/10 plans |
| PERF-01..04 | 15 | ✅ Complete | Phase 15 SUMMARY + 5/5 plans |

---

## Active: v1.16 Requirements (to be defined)

The v1.16 scope is **not yet defined**. Candidate areas based on the open
items from `gsd-sdk query audit-open` (103 items) and the
state-sync-report (20260616):

### Audit-derived candidates

- **AUDIT-01**: Triage and close 45 debug sessions flagged `root_cause_found`
  but not actioned. Most are stale (last updated 2026-05).
- **AUDIT-02**: Verify 51 quick-task directories in `.planning/quick/` —
  many have commits in main but the directory was never marked done.
- **AUDIT-03**: Close 4 UAT gaps + 3 verification gaps surfaced by recent
  phases (32, 33, 34).

### Carry-over from abandoned plans

- **VDI-22C**: VDI 账号管理与密码 (2 plans, was planned under v1.12/v1.11)
- **VDI-22D**: VDI Web Console 监控 (2 plans, was planned under v1.12/v1.11)
- **VM-25**: 虚拟机数据范围权限配置 (4 plans, depends on Phase 22)
- **DICT-29**: 重构工位状态枚举改为 sys_dict (0 plans, listed in STATE)
- **PERF-32**: Phase 32 剩余 P2 架构债 (5+ 类 in ROADMAP P2-A1..A8 partial)

### Phase 34 review-derived (post-closure)

- **AUDIT-04**: Lock the Phase 34 review's expanded `sensitiveKeys` (34
  entries) via property-based test that asserts every camelCase / snake_case
  variant of password / key / secret / token in the codebase is masked.
  (Defensive: would have caught the original CR-001 bug at code-introduce
  time instead of at review time.)
- **AUDIT-05**: Promote the Phase 34 `operlog_e2e_verify.sh` static +
  live sections to a real CI gate (currently `SKIP_E2E=1` by default so
  live assertions never run).
- **AUDIT-06**: Range-over-int modernization pass on the 1 file flagged
  by `rangeint` linter (cosmetic, single line in regression_test.go).

---

## Out of Scope (v1.16)

| Candidate | Reason | Replacement |
|-----------|--------|-------------|
| New product surfaces (AI features, mobile app) | Not derived from user feedback yet | Wait for v1.16 stakeholder input |
| VDI 22C / 22D | Depends on VDI production stability data from v1.11 | Schedule after v1.17 once 22A/22B is observably stable |
| DICT-29 (sys_dict refactor) | Cosmetic, no functional gain | Defer to v1.18 |

---

## Traceability

| REQ-ID | Description | Phase | Status |
|--------|-------------|-------|--------|
| (none yet) | TBD after v1.16 scope is finalized | TBD | Pending |

---

*This file replaces the v1.5 file (archived to
`.planning/milestones/v1.5-REQUIREMENTS.md`). It will be re-populated
when v1.16 scope is decided via `/gsd:new-milestone`.*
