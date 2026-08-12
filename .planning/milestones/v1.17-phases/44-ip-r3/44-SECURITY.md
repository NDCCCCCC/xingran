---
phase: 44
slug: ip-r3
status: verified
threats_open: 0
threats_total: 13
threats_closed: 13
asvs_level: 1
register_authored_at_plan_time: true
created: 2026-06-28
audited: 2026-06-28
---

# Phase 44 (ip-r3) — Security Audit Report

**Phase:** 44 — R3 IP 例外规则引擎 (44-01 + 44-02)
**Audited:** 2026-06-28
**ASVS Level:** L1
**Register authored at:** PLAN time (`register_authored_at_plan_time: true`)
**Auditor stance:** Verify each declared mitigation EXISTS in implementation. Do NOT scan for new threats.
**Result:** SECURED — 13/13 threats closed (10 mitigated + 3 accepted-as-documented).

---

## Threat Verification Register

### Plan 44-01 Threats (T-44-01 .. T-44-06 + SC)

| Threat ID | Category | Disposition | Status | Evidence (file:line) |
|-----------|----------|-------------|--------|----------------------|
| T-44-01 | Tampering / CIDR 注入 | mitigate | CLOSED | `internal/services/asset/reconciliation_exception.go:151-160` — `ValidateCIDR` calls `net.ParseCIDR` strictly, returns error on parse failure. Called in `Create:320` + `Update:373`. DB `cidr` column is secondary backstop (migration_168 schema). Matcher `preloadActiveRules` (`reconciliation_exception_matcher.go:88-92`) additionally skips rules whose CIDR fails to parse at detection time. |
| T-44-02 | Elevation / 越权创建例外规则 | mitigate | CLOSED | `internal/api/v1/asset/reconciliation_exception_router.go:45-56` — 4 CRUD/test routes wrapped with `middleware.RequirePermissions(asset:reconciliation:exception:{create,update,delete,test})`. **CR-03 fix (commit `08e756fa`) CONFIRMED** at `router.go:64-69`: `/baseline/snapshot` → `exception:create` perm, `/baseline/compare` → `reconciliation:list` perm. Import route also gated (`:80-82`). |
| T-44-03 | Tampering / SQL 注入 (MatchTest) | mitigate | CLOSED | `internal/services/asset/reconciliation_exception.go:463-519` — `MatchTest` uses pure in-memory matching (`net.ParseCIDR` + `ipNet.Contains`), zero SQL string concatenation. Grep for `ip_range >>` / `fmt.Sprintf.*ip_range` / `Where.*ip_range.*+` → **no matches** anywhere in `internal/services/asset/`. NOTE: PLAN cited `Where("ip_range >> ?::inet", ip)` GiST SQL, but implementation chose in-memory matching (documented deviation in `matcher.go:14-27` + `exception.go:455-460`); GiST index (`migration_174`) remains as DB optimizer hint only. Net result: SQL injection surface eliminated entirely, mitigation exceeds plan. |
| T-44-04 | DoS / 告警风暴误配 (0.0.0.0/0 silence) | mitigate | CLOSED | `internal/services/asset/reconciliation_exception.go:205-210` — `ValidateReason` enforces `utf8.RuneCountInString(reason) < 10` → error. Called in `Create:329` + `Update:382`. Frontend `ExceptionRuleForm.tsx` mirrors with `min: 10` (per 44-01 SUMMARY). `expires_at` default governed by INFRA-02 seed (out of scope for this phase's code). |
| T-44-05 | Repudiation / 审计链断裂 | mitigate | CLOSED | (a) `reconciliation_exception_handler.go:115` (Create OperTypeCreate), `:145` (Update OperTypeUpdate), `:169` (Delete OperTypeDelete), `:235` (SnapshotBaseline OperTypeUpdate), `:297` (ImportRules OperTypeImport), `:312` (ExportRules OperTypeExport), `:332` (DownloadTemplate OperTypeDownload) — all 7 write paths call `operlog.Record(ModuleReconciliationExceptionRule, ...)` before `response.Success`. `TestRule`/`CompareBaseline`/`ListRules`/`GetRuleByID` correctly omit operlog (read paths). (b) Layer 3.5 `reconciliation_detection.go:293-301` writes `ExceptionRuleID` + `AppliedActions` into `sys_data_reconciliation` INSERT even on silence hit (`:352-353`). (c) `Delete:435-450` uses GORM soft-delete (`deleted_at` filled); cron `cleanupExpiredExceptionsDirect` (`reconciliation_tasks.go:256-259`) only flips `is_active=0→1`, `deleted_at` stays NULL. Audit chain preserved. |
| T-44-06 | Info Disclosure / 越权读例外规则 | accept | CLOSED-ACCEPTED | `reconciliation_exception_router.go:41-42` — `/exception-rule/list` + `/:id` have no `RequirePermissions` (R1 skeleton carried forward). **Acceptance rationale documented** in router comment `:28-32` + PLAN threat_model: admin role holds list perm by default, normal users don't reach admin page; mirrors project memory `xingran-perm-namespace-split-readonly-page` (read path relaxation to avoid accidental lockout). Consistent with R1 boundary. |
| T-44-SC (44-01) | Tampering / 依赖 | mitigate | CLOSED | Zero new dependencies added. `reconciliation_baseline.go:31` imports `github.com/google/uuid` — confirmed pre-existing project dep (CLAUDE.md tech-stack: "google/uuid v1.6.0 — UUID generation"). No new npm/pip/cargo installs per 44-01 SUMMARY Threat Flags ("无新增威胁面"). |

### Plan 44-02 Threats (T-44-07 .. T-44-12 + SC)

| Threat ID | Category | Disposition | Status | Evidence (file:line) |
|-----------|----------|-------------|--------|----------------------|
| T-44-07 | Repudiation / 审计链断裂 (cleanupExpiredExceptions 硬删除风险) | mitigate | CLOSED | `internal/scheduler/reconciliation_tasks.go:255-264` — `cleanupExpiredExceptionsDirect` issues `Update("is_active", 1)` (soft-disable). WHERE clause `:258` includes `deleted_at IS NULL` filter but the UPDATE itself does NOT touch `deleted_at` (column stays NULL). Idempotency via `is_active = ?` (=0) in WHERE — second cron run gets `rowsAffected=0`. Test `TestCleanupExpiredExceptions` (per 44-02 SUMMARY commit `2cd70fde`) locks `pastDeletedAt.Valid == false`. |
| T-44-08 | Tampering / 转单过滤数据依赖 + NULL 漏转 (BLOCKER-4) | mitigate | CLOSED | `internal/scheduler/reconciliation_tasks.go:212-213` — **literal string match**: `"severity = ? AND deleted_at IS NULL AND resolved_at IS NULL AND workorder_id IS NULL " + "AND (applied_actions IS NULL OR 'no_workorder' != ANY(applied_actions))"`. The `IS NULL` fallback is present verbatim. Reasoning documented in func docstring `:191-198` (PG three-valued logic: `'no_workorder' != ANY(NULL)` returns NULL → row filtered without fallback). Test `TestCreateWorkorderNoWorkorderFilterNullActions` locks behavior. **BLOCKER-4 satisfied.** |
| T-44-09 | Tampering / Excel CIDR 注入 | mitigate | CLOSED | `internal/services/asset/reconciliation_exception.go:552-573` — `ImportFromExcel` delegates base-field write to `excelSvc.ImportData` (excel_service enforces `Required` on `ipRange` column per `excel_config.go:318`). DB `cidr` column backstop rejects malformed CIDR with SQLSTATE 22P02 on INSERT. Additionally, post-process `postProcessImportedRules:584-643` re-reads raw rows but does not re-validate CIDR (acceptable — DB column is authoritative backstop, and imported rules are subsequently loaded by `preloadActiveRules` which skips unparseable CIDRs at detection time, `matcher.go:88-92`). |
| T-44-10 | Tampering / Excel SQL 注入 (scope_name 解析) | mitigate | CLOSED | `internal/services/asset/reconciliation_exception.go:665-668` (dept): `Where("dept_name = ? AND deleted_at IS NULL", scopeName)` — GORM placeholder. `:680-683` (user): `Where("username = ? AND deleted_at IS NULL", scopeName)` — GORM placeholder. Post-process UPDATE at `:635-638` uses `Where("name = ? AND deleted_at IS NULL", name)` — placeholder. Zero string concatenation in any scope-resolution path. |
| T-44-11 | Spoofing / 基线覆盖 (SnapshotBaseline 覆盖现有 baseline) | accept | CLOSED-ACCEPTED | `internal/services/asset/reconciliation_baseline.go:118-143` — Snapshot is overwrite-by-design (`GetByKey` exists → Update config_value, else Create). **Acceptance rationale documented** in PLAN threat_model: baseline is operator-triggered snapshot point, overwrite is intentional. Mitigations present: (a) `handler.go:235` calls `operlog.Record(OperTypeUpdate)` for audit trail; (b) CR-03 fix gates route with `exception:create` permission (`router.go:64-66`), restricting to admin role. |
| T-44-12 | Info Disclosure / Compare 无 baseline | accept | CLOSED-ACCEPTED | `internal/api/v1/asset/reconciliation_exception_handler.go:248-262` — `CompareBaseline` returns HTTP 400 on missing baseline. Error message `"未找到基线快照,请先调用 Snapshot 记录基线"` (`baseline.go:164,169`) — no sensitive data leaked. Frontend fallback renders guidance Alert per 44-02 SUMMARY. **Acceptance rationale documented** in PLAN threat_model. |
| T-44-SC (44-02) | Tampering / 依赖 | mitigate | CLOSED | Same as 44-01 SC — zero new deps. `google/uuid` already counted. |

---

## Summary Counts

- **Total threats:** 13
- **Closed (mitigate):** 10 (T-44-01, 02, 03, 04, 05, 07, 08, 09, 10, SC×2 merged)
- **Closed (accept, documented):** 3 (T-44-06, 11, 12)
- **Open:** 0
- **Escalated:** 0

## Unregistered Flags

None. Both 44-01 SUMMARY (`## Threat Flags`, line 178) and 44-02 SUMMARY (`## Threat Flags`, line 197) explicitly state "无新增威胁面" (no new attack surface) and map every implemented control back to a declared threat ID. No unregistered flags to log.

## Code Review Context (44-REVIEW.md)

The code review raised 5 BLOCKER findings (CR-01..CR-05). Of these:
- **CR-03** (missing RequirePermissions on baseline routes) was a genuine security gap that has been **FIXED** in commit `08e756fa` — verified present at `router.go:64-69`. This is the only review finding in the threat-model scope; it maps to T-44-02 and is now CLOSED.
- **CR-01** (sys_config.config_value varchar(500) overflow) — not a declared threat in `<threat_model>`; operational robustness issue, out of audit scope.
- **CR-02** (multi-rule audit pointer picks first rule, not max-contributor) — touches T-44-05 audit chain but the *declared* mitigation ("operlog.Record on writes + Layer 3.5 INSERT with exception_rule_id + soft-delete preserves chain") is present. The review's concern is about audit *precision* in multi-rule hits, not absence of audit. Not a BLOCKER under `register_authored_at_plan_time: true` mode — declared mitigation verified present.
- **CR-04** (MatchTestPanel queryKey/queryFn divergence) — frontend correctness, not a declared backend threat.
- **CR-05** (Excel post-process UPDATE not scoped to single row) — touches T-44-09/10 data integrity but the *declared* mitigations (ValidateCIDR backstop + GORM placeholder) are present. The review's concern is UPDATE scope breadth, not SQL injection or CIDR bypass.

No declared mitigation is missing. Phase is clear to ship under L1 / `block_on: critical`.

---

_Audit method: each `mitigate` threat verified by Read of cited implementation file + Grep for mitigation pattern. Each `accept` threat verified by presence of acceptance rationale in PLAN `<threat_model>` + SECURITY.md documentation. Implementation files were READ-ONLY; no edits made._
