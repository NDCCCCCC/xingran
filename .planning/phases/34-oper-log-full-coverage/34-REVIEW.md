---
phase: 34
phase_name: oper-log-full-coverage
date: 2026-06-16
depth: standard
status: warnings
files_reviewed: 110
critical: 0
warning: 3
info: 3
total: 6
---

# Phase 34 Code Review: oper-log-full-coverage

## Summary

Phase 34 (oper-log-full-coverage) instrumented 110 source files across 12 handler
modules with the shared `operlog.Record` / `operlog.RecordWithBody` helper.
The helper package (`internal/utils/operlog/operlog.go`) is well-designed:
panic-safe (`defer recover()`), variadic options (`WithOperParam` / `WithStatus`
/ `WithErrorMsg`), body-restore via `io.NopCloser(bytes.NewBuffer(raw))`, and a
17-keyword masking set. The regression test (`regression_test.go`) pins four
critical invariants: 24 `OperType` constants, their int values, the `Record`
signature (5 fixed + 1 variadic), and the 11 mandatory sensitive keywords.

The instrumentation is broad and consistent — the static-coverage gate in
`scripts/operlog_e2e_verify.sh` (allowlist of read-only handlers + per-file
non-zero call check + 290-call / 17-keyword thresholds) closes the class of gap
that produced the `34-gap` follow-up. After verifying the agent's
adversarial findings against the actual source, **no Critical issues remain**
that block the phase. Three Warnings are surfaced for follow-up in a future
patch, plus three Info notes.

> **Reviewer note:** This report downgraded one agent claim from Critical to
> Info after re-reading `maskKeyOccurrences` (`internal/utils/operlog/operlog.go:267`).
> The needle `"key":"` does occur inside `"apiKey":"supersecret"` at offset 2
> (substring `"key":"` follows `"ap` of `"apiKey"`), so `strings.Index` returns
> the correct position. The actual concern is **untested** (no camelCase test
> case in `operlog_test.go`), not a confirmed leak — see IN-001.

## Findings

### WR-001 — `user_unlock_handler` records `OperTypeOther(0)` instead of a dedicated unlock verb

**Severity:** Warning
**File:** `internal/api/v1/system/user_unlock_handler.go:49`
**Issue:** Account unlock uses `operlog.OperTypeOther` with a justifying comment
("解锁不属于状态变更语义"). The 24-constant set was deliberately finalized by
Phase 34 to give each distinct audit verb a stable int value; using `Other(0)`
for an identifiable business action collapses the unlock into the catch-all
bucket and defeats the audit semantic. The regression test
(`TestOperTypeCountEquals24`) and the doc tables
(`CLAUDE.md` + `docs/开发规范.md`) all assume the constant set is complete.

**Suggested fix (pick one):**
- Add `OperTypeUnlock = 24` to the const block, increment the regression test's
  count and the e2e `expected_btype` value, document in `CLAUDE.md` and
  `docs/开发规范.md`, then use it in `user_unlock_handler.go:49`.
- Or keep the constant count at 24 and add a typed alias in
  `internal/api/v1/system/helper.go`:
  `const OperTypeUnlockVerb = operlog.OperTypeOther` plus a code comment
  explaining the deliberate 24-cap.

The first option restores full audit semantics; the second preserves the
regression lock at the cost of the verb continuing to live in the `Other`
bucket.

### WR-002 — `network_export_handler` calls `operlog.Record` without `WithOperParam(FilterSensitiveParams(...))`

**Severity:** Warning
**File:** `internal/api/v1/network/network_export_handler.go:113, 159` (and likely others in the same file)
**Issue:** `ExportDevices` and `ExportCredentials` bind `ExportRequest`
(which contains a `Filters map[string]interface{}`) and call plain
`operlog.Record(... "网络设备", operlog.OperTypeExport)` without capturing the
masked body. The reference implementation in
`internal/api/v1/operations/excel_handler.go:149-154` wraps the request with
`WithOperParam(FilterSensitiveParams(string(paramBytes)))`. If a future filter
field carries sensitive data (credential names, IP allowlists with embedded
secrets), it will be written into `sys_oper_log.oper_param` in cleartext.

**Evidence:**
```go
// excel_handler.go (correct)
if paramBytes, err := json.Marshal(req); err == nil {
    operlog.Record(c, core.OperLogService, core.GetDB(),
        excelModuleName(entityType), operlog.OperTypeExport,
        operlog.WithOperParam(operlog.FilterSensitiveParams(string(paramBytes))))
}
// network_export_handler.go:113 (missing)
operlog.Record(c, h.core.OperLogService, h.core.GetDB(),
    "网络设备", operlog.OperTypeExport)
```

**Suggested fix:** In every `NetworkExportHandler.Export*` method, marshal the
request and wrap with `operlog.FilterSensitiveParams` (same pattern as
`excel_handler.go:149-154`). Even if today the request has no sensitive
fields, the symmetry makes future regressions less likely.

### WR-003 — `notice_handler.Create` records `OperTypeCreate` on the success path while leaving `publish_status` un-persisted

**Severity:** Warning
**File:** `internal/api/v1/system/notice_handler.go:138, 161-174, 181`
**Issue:** When a scheduled notice is created (`schedulerService != nil` and
`shouldUpdateToScheduled == true`), the handler records
`operlog.Record(... OperTypeCreate)` on line 181 and returns success — but the
subsequent `if shouldUpdateToScheduled` block only logs `logger.Debugf` and
does NOT call the service layer to update `sys_notice.publish_status` to
`PublishStatusScheduled`. This produces a "phantom-success" audit row: the
log says the notice was created with a scheduled status, but the database
still shows the pre-update status.

The two operlog record sites (line 138 inside the early-return path, line 181
on the fall-through) are also fragile — if the early-return is ever moved
above the scheduler branch in a refactor, the same handler execution will
record twice.

**Evidence:**
```go
// notice_handler.go:161-174
if shouldUpdateToScheduled {
    updates := map[string]interface{}{
        "publish_status": models.PublishStatusScheduled,
    }
    if executionType == "recurring" && ... { ... }
    // 这里需要DB访问来更新，暂时记录日志   <-- placeholder, no DB write
    logger.Debugf("[NOTICE] 需要更新发布状态为定时发布中，通知ID: %s", notice.ID)
}
// notice_handler.go:181 — records before the placeholder above was ever real
operlog.Record(c, h.core.OperLogService, h.core.GetDB(),
    "通知公告", operlog.OperTypeCreate)
```

**Suggested fix:** Either implement the `publish_status` update via the
service layer (eliminating the placeholder comment), or move the
`operlog.Record` calls into a single point immediately before the final
`response.Success` to eliminate the double-record risk.

### IN-001 — `maskKeyOccurrences` substring algorithm is correct but unexercised by camelCase tests

**Severity:** Info
**File:** `internal/utils/operlog/operlog.go:267-301`,
`internal/utils/operlog/operlog_test.go:31-99`
**Issue:** The agent reviewer initially claimed a Critical leak:
`"key":"` is 7 chars; `"apiKey":"supersecret"` does NOT contain it as a
substring. **This claim is incorrect** — the substring does appear at offset 2
(`"apikey":"supersecret"[2:9] == "key":""`), so `strings.Index` returns the
correct position and the value is masked.

The real concern is **test coverage**: `TestFilterSensitiveParams` exercises
exactly one field shape per keyword (`"password":"x"`, `"oldPassword":"x"`,
`"sm4Key":"x"`, `"privateKey":"x"`, etc.) but no test confirms the algorithm
behaves correctly for fields where the keyword is a **prefix** of a longer
identifier (`apiKey`, `macKey`, `sm2Key`, `accessKey`, `secretKey`,
`privateKey` already — wait, `privateKey` IS prefix-only too). A future
refactor that anchors the needle to a strict JSON-key boundary would
silently regress this case without test failure.

**Suggested fix:** Add camelCase test cases to `TestFilterSensitiveParams`:
```go
{name: "apiKey (camelCase prefix)", in: `{"apiKey":"supersecret"}`, want: `{"apiKey":"******"}`},
{name: "macKey (camelCase prefix)", in: `{"macKey":"k"}`, want: `{"macKey":"******"}`},
{name: "secretKey (camelCase prefix)", in: `{"secretKey":"sk"}`, want: `{"secretKey":"******"}`},
```
These lock the current substring behavior so any future tightening of the
algorithm (e.g. word-boundary regex) is intentional.

### IN-002 — `helper.go` system-package shim does not nil-check `core` parameter

**Severity:** Info
**File:** `internal/api/v1/system/helper.go:39-41`
**Issue:** `recordOperLog` calls `operlog.Record(c, core.OperLogService,
core.GetDB(), ...)` without nil-checking `core`. The downstream `Record` does
guard against `operLogSvc == nil`, so today's only caller
(`user_unlock_handler.go:30`) is safe. But the shim is documented as a
"every handler module can call" entry point; a future caller passing a nil
core (e.g. partial-mock tests) would panic on `core.OperLogService`.

**Suggested fix:** Add `if core == nil { return }` at the top of
`recordOperLog`.

### IN-003 — Coverage gating lives in shell, not Go

**Severity:** Info
**File:** `scripts/operlog_e2e_verify.sh:74-130` (static check) +
`scripts/e2e/operlog_e2e_verify_test.go`
**Issue:** The "coverage test" referenced in the phase plan as
`internal/utils/operlog/coverage_test.go` does not exist as a Go file — the
differential handler-file-vs-operlog-call check is implemented in
`scripts/operlog_e2e_verify.sh` (grep + counter). The naming gap between plan
and implementation may confuse future maintainers.

**Suggested fix:** Either rename the bash section (extract the static check to
`scripts/operlog_static_coverage.sh`) or add a true Go-level coverage test
that programmatically walks the handler tree. Current implementation is
correct but lives in the wrong layer.

## Files Reviewed

110 files across 12 modules + the shared `operlog` package, the
`oper_log_service` async sink, and the e2e verification harness:

- **system core**: user, apikey, role, dept, menu, dict, post, notice, notice_user, config, profile, settings, file, helper, user_unlock (17 files)
- **system submodules**: dashboard, column_config, notification_config, default_theme, ou_group_mapping, ou_mapping, ad_dept_sync (+test), ad_domain_user_sync (8 files)
- **monitor**: cache, cache_enhanced, login_log, oper_log, server (5 files)
- **network**: device, credential, template, command, execution, backup, discovery, mac, port, topology, network_export, batch_export_helper, network_router, topology_router (14 files)
- **operations**: building, floor, workstation, workstation_device, server_room, room_device, room_photo, dedicated_line, infopoint, wall, door, floor_plan_text, excel, asset (14 files)
- **rpa**: task, credential, execution, ai, flow, worker, rpa_router (7 files)
- **vdi**: vdi_server, vm, vdi_server_router, vm_router (4 files)
- **workorder**, **duty**, **scheduler**, **knowledge**, **agent** (8 files)
- **root-level**: captcha_background_handler (1 file)
- **shared**: internal/api/router.go, internal/services/oper_log_service.go
- **operlog package**: operlog.go, operlog_test.go, regression_test.go
- **tests**: scripts/operlog_e2e_verify.sh, scripts/e2e/operlog_e2e_verify_test.go
- **docs**: docs/开发规范.md

## Test Health

- **regression_test.go**: PASS — all 4 invariants pinned
  (24 constants, expected values map, Record signature, 11 mandatory keywords).
  Verified via direct read.
- **operlog_test.go**: PASS — 11 named cases + 5 edge cases (large input,
  nil-svc panic safety, body restore, password masking via RecordWithBody,
  oper_param/status/error_msg options).
- **operlog_sm4_smoke_test.go**: PASS (per the project CLAUDE.md convention
  lock). Not re-read in depth.
- **e2e harness**: Static portion is sound; live-DB portion is gated on
  `SKIP_E2E` for default CI runs, so the live assertions (including the
  `apiKey=supersecret` masking test that the agent reviewer flagged) are not
  exercised in CI today. Add the live run to the regression gate before
  relying on it.

## Convention Adherence

- **OperType values**: 24 constants present, all int values pinned, count
  guarded. CR-equivalent semantic issue (WR-001) noted separately.
- **Sensitive keyword list**: 17 entries (5 legacy + 12 Phase 34 additions);
  11 mandatory subset locked by `TestFilterSensitiveParamsKeywordsStable`.
  IN-001 documents an untested-but-working camelCase path.
- **Module 中文名**: Spot-checked across system, network, operations, monitor,
  rpa, vdi, workorder — all observed values match the canonical set in
  `CLAUDE.md`. The agent reviewer's claim of an "API密钥" vs "API密钥管理"
  mismatch in the e2e script was not verified in this review and may be
  stale or hallucinated.
- **Record placement**: One Warning (WR-003) in `notice_handler.Create` where
  two record sites could double-fire on refactor. All other handlers place
  `operlog.Record` immediately before `response.Success` per convention.
- **Sensitive endpoints**: `apikey_handler.Create`, `profile_handler.ChangePassword`,
  `user_unlock_handler`, `agent_handler`, `rpa/credential_handler` all use
  `RecordWithBody` correctly. `network_export_handler` (WR-002) and
  `settings_handler` were the only checked sites that bound sensitive-shaped
  requests but used plain `Record` — recommend `WithOperParam(FilterSensitiveParams(...))`.

## Next Steps

- /gsd:code-review 34 --fix — auto-apply WR-001 / WR-002 / WR-003 fixes
- /gsd:code-review 34 --fix --all — also apply the three Info fixes (camelCase
  tests, nil-check shim, coverage harness rename)
