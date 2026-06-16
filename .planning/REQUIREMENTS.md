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

### Phase 35 — operlog 配置驱动排除白名单 (post-Phase 34 review)

- **OPERLOG-01**: 在 `internal/utils/operlog` 包内引入配置驱动的操作日志排除路径机制。`Record` 与 `RecordWithBody` 函数体首行通过 `filepath.Match` + `/*` 后缀通配判定，命中即早退（不调用 `RecordAsync`），未命中走原路径。排除列表在 `core.go` 配置加载后调一次 `operlog.Configure(...)` 注入。默认值由 `configs/config.yaml` 的 `operlog.exclude_paths` 数组决定。
- **OPERLOG-02**: 暴露 `func IsExcludedPath(path string) bool` 给外部使用，行为对齐 `pkg/middleware/request_decryption.go:294-315` 的 `isExcludedPath`（同样 `filepath.Match`，不会跨 `/`）。
- **OPERLOG-03**: 保留 `internal/utils/operlog/regression_test.go` 锁定的所有公共 API：`Record(6 参数+1 variadic)` / `RecordWithBody(5 参数非 variadic)` 签名不变；25 个 OperType 常量值不变；34 个敏感关键词列表不变；任何新增 API 都是扩展（不替换）。
- **OPERLOG-04**: 默认配置 `operlog.exclude_paths` 应包含 `/api/v1/rpa/workers/*/heartbeat`（**仅** heartbeat，不含 progress / register）。`progress` 端点保留审计价值（任务轨迹）；`register` 保留生命周期事件。
- **OPERLOG-05**: 在 `regression_test.go` 新增 `TestExcludedPathsEarlyReturn`：mock Recorder，断言路径命中时 `RecordAsync` 不被调用、路径未命中时仍被调用、断言 `RecordWithBody` 在排除路径上不消费 request body。
- **OPERLOG-06**: 行为可观察 — 配置 `operlog.exclude_paths` 后，启动后端 + 一个 rpa-worker Agent 连续运行 5 分钟，SQL `SELECT COUNT(*) FROM sys_oper_log WHERE oper_url LIKE '%heartbeat%' AND oper_time > NOW() - INTERVAL '5 minutes'` 必须返回 0。

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
| OPERLOG-01..06 | operlog.exclude_paths 配置驱动白名单 | 35 | Planned |

---

*This file replaces the v1.5 file (archived to
`.planning/milestones/v1.5-REQUIREMENTS.md`). It will be re-populated
when v1.16 scope is decided via `/gsd:new-milestone`.*
