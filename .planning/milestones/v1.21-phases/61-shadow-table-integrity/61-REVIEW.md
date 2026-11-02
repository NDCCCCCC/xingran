---
phase: 61-shadow-table-integrity
reviewed: 2026-10-21T00:00:00Z
depth: standard
files_reviewed: 7
files_reviewed_list:
  - internal/services/addomain/computer.go
  - internal/services/addomain/computer_78_06_test.go
  - internal/services/addomain/group_sync_service.go
  - internal/services/addomain/shadow_cleanup_service.go
  - internal/services/addomain/shadow_cleanup_test.go
  - internal/services/addomain/sync.go
  - internal/services/addomain/sync_78_05_test.go
findings:
  critical: 1
  warning: 6
  info: 2
  total: 9
status: issues_found
---

# Phase 61: Code Review Report

**Reviewed:** 2026-10-21T00:00:00Z
**Depth:** standard
**Files Reviewed:** 7
**Status:** issues_found

## Summary

Phase 61 delivers three plans in `internal/services/addomain`: (61-01) shadow-table cleanup service soft-deleting vanished AD rows after full sync behind dual gates; (61-02) soft-deleted row revival via Unscoped existing queries + `deleted_at` in `ON CONFLICT` update lists; (61-03) `syncGroupMembers` Delete+Create wrapped in a GORM transaction.

The core mechanisms are sound. I traced the revival path end-to-end for all four entity types (OU/group/user/computer) plus the `SyncSingleGroup` OnConflict path: the existing-row queries are Unscoped, `DeletedAt` is zeroed on hits, and every upsert that can receive a soft-deleted row includes `deleted_at` in `AssignmentColumns`, so a conflicting insert resets `deleted_at=NULL`. The cleanup gates (empty-LDAP reject, >50% reject with strict-greater comparison) exactly mirror the `handleDeletedGroups` precedent, and both sides of the 50% boundary are test-anchored. The managed-set filtering (users: non-empty `sAMAccountName`; computers: non-empty `cn`; OUs: all) matches each sync's skip logic. The transaction rollback is proven by the `FORCED_FAIL` CHECK-injection test.

However, the review found 1 Critical and 6 Warning issues. The Critical one is a pre-existing bug in an in-scope file (`updateSyncLog` writes a nonexistent column, so **every failed AD sync is recorded as perpetually "running"** with no error message — a monitoring blind spot that the Phase 78 test suite explicitly locked in as "current behavior"). Warnings cover silently-swallowed query errors in the new cleanup code, a completeness gap (vanished groups are never cleaned in the scheduled full-sync path), the `IsEnabled default:true` GORM quirk that persists AD-disabled users as enabled, a comment/log that contradicts the actual dedup behavior in `syncComputers`, unsanitized LDAP strings in the incremental group-sync paths, and the stringly-typed `syncType` guard that turns any caller literal drift into a silent cleanup no-op.

## Critical Issues

### CR-01: updateSyncLog writes nonexistent column `error_message` — failed syncs are never recorded as failed

**File:** `internal/services/addomain/sync.go:739` (model: `internal/models/ad_domain.go:207`)
**Issue:** `updates["error_message"] = safe` uses the key `error_message`, but `models.ADSyncLog.ErrorMsg` maps to column `error_msg` (`gorm:"type:text;column:error_msg"`). GORM map-`Updates` emits the raw key as a column name, so the entire `UPDATE sys_ad_sync_log SET sync_status=?, ..., error_message=?` fails with PostgreSQL SQLSTATE 42703 (`column "error_message" does not exist`). Because one bad key poisons the whole statement, **none** of the fields are written: `sync_status` stays `running`, `end_time`/`duration`/`ou_count`/`group_count`/`user_count`/`computer_count`/`error_count` are all lost, and the sanitized error text is never persisted. Every failed AD sync therefore looks like a sync that never finished. Note the success path is unaffected (it skips the `errorMsg != ""` branch), which is why this survives in production unnoticed.

Provenance: pre-existing bug, not introduced by Phase 61 — `TestSync78_UpdateSyncLog_StatusMatrix` (sync_78_05_test.go:1390-1450) documents and **asserts** the broken behavior ("当前生产 bug", frozen under D-78-05c/D-78-10 "无据不改"). This review finding is the missing authority to fix it; the shipped-code behavior is incorrect and it directly degrades the operational integrity of the sync feature this phase extends (cleanup results land in a log whose failure states are never written).

**Fix:**
```go
// sync.go:739
updates["error_msg"] = safe   // was: updates["error_message"] = safe
```
Then flip the assertions in `TestSync78_UpdateSyncLog_StatusMatrix` (sync_78_05_test.go:1433-1443) and `TestSync78_SyncDataInternal_EmptyAccountPool` (sync_78_05_test.go:1553-1554) from "现行为:UPDATE 失败" to asserting `SyncStatus == Failed`, `ErrorMsg == "boom"`, and non-nil `EndTime`. Remove the now-stale "当前生产 bug" commentary blocks.

## Warnings

### WR-01: Cleanup row queries swallow errors — feature silently disables with zero observability

**File:** `internal/services/addomain/shadow_cleanup_service.go:74-77, 131-134, 190-193`
**Issue:** All three cleanup functions ignore the error from the stale-rows `Find`:
```go
var staleUsers []models.ADUser
s.db.WithContext(ctx).
    Where("ad_config_id = ? AND deleted_at IS NULL", config.ID).
    Find(&staleUsers)          // error discarded
```
If the query fails (transient connection error, lock timeout, missing table), the slice stays empty, `vanishedDNs` is empty, and the function returns `nil` with no log line — the entire SHADOW-01 feature is silently off. The failure direction is at least safe (no deletion), but a safety-gated feature that can silently no-op with no trace defeats its own auditability. This is new code in this phase, unlike the pre-existing instances of the same pattern elsewhere.

**Fix:**
```go
if err := s.db.WithContext(ctx).
    Where("ad_config_id = ? AND deleted_at IS NULL", config.ID).
    Find(&staleUsers).Error; err != nil {
    return fmt.Errorf("查询存量用户失败: %w", err)
}
```
The caller (`sync.go:173-175`) already degrades cleanup errors to a WARN without failing the sync, so propagating is free and gains the missing observability.

### WR-02: Vanished groups are never cleaned in the scheduled full-sync path

**File:** `internal/services/addomain/shadow_cleanup_service.go:37-48` (design gap; contrast `internal/services/addomain/group_sync_service.go:360-418`)
**Issue:** `cleanupAfterFullSync` covers users/OUs/computers only. Group deletion lives exclusively in `handleDeletedGroups`, whose sole caller is `GroupSyncService.SyncGroupsByConfig` — reachable only from the manual API endpoint (`internal/api/v1/system/ad_domain_handler.go:757`). The scheduler (`internal/scheduler/ad_sync_tasks.go:317-340`) runs `SyncDataByID(ctx, configID, "full")`, which never deletes vanished groups. Consequence: on the common scheduled path, groups deleted in AD remain live in `sys_ad_group` indefinitely, and their `sys_ad_group_member` rows are never replayed away either (`syncGroups` only writes members for LDAP-present groups; `handleDeletedGroups` is the only member-relation cleaner and it is not called). For a phase whose goal is shadow-table integrity, the group shadow table is left permanently stale unless an operator manually clicks group sync.

**Fix:** Call the existing gate-protected cleaner from the full-sync orchestration, e.g. add to `cleanupAfterFullSync` after `cleanupVanishedComputers`:
```go
grpResult := &GroupSyncResult{}
if err := s.handleDeletedGroupsForConfig(ctx, config, groups, grpResult); err != nil {
    return err
}
result.GroupCount = len(groups) // counters: consider a DeletedGroupCount field
```
The cleanest route is extracting the body of `handleDeletedGroups` into a `SyncService` method (it already has both gates) and having `GroupSyncService.handleDeletedGroups` delegate to it, keeping one implementation. Alternatively, document explicitly in the phase artifacts why groups are excluded and add the manual group-sync run to the ops runbook.

### WR-03: AD-disabled users are persisted as `is_enabled = true` (GORM `default:true` zero-value omission)

**File:** `internal/services/addomain/sync.go:570, 600` (root cause: `internal/models/ad_domain.go:128`)
**Issue:** `models.ADUser.IsEnabled` carries `gorm:"default:true"`. For a user with UAC=514 (disabled), `newUser.IsEnabled = false` is the zero value, so GORM **omits** `is_enabled` from the INSERT column list and the DB column default (`true`) applies. The conflict-update path is equally broken: the INSERT omits the column, so `EXCLUDED.is_enabled` evaluates to the column default (`true`) and `DoUpdates: AssignmentColumns([..., "is_enabled", ...])` writes `true` over the existing value. Net effect: every AD-disabled account synced is stored as enabled — on both create (`sync.go:600`) and update (`sync.go:570`). The phase's own test (`sync_78_05_test.go:918-926`) documents this as a known GORM quirk rather than a defect; it is a defect. Impact is limited today (filters/UI, `user.go:63-64, 275-276`), but any consumer trusting `is_enabled` for access decisions would be misled, and admins see disabled accounts shown as enabled after every sync. Provenance: pre-existing; file in review scope and the phase touched both write sites.

**Fix (pick one):**
- Remove `default:true` from the tag and rely on explicit assignment (check other insert sites first), or
- Make the field `*bool` so `false` is no longer the zero value, or
- Force the column into the statement: `.Select("is_enabled", ...)` on the create/upsert calls — heavier, must enumerate all columns.

### WR-04: `computersToCreate` dedup keeps the FIRST duplicate while the comment and log claim "keep last/newest"

**File:** `internal/services/addomain/computer.go:442-459` (comment 438-441, log 455)
**Issue:** The F-02 block implements first-write-wins:
```go
if _, exists := dedupMap[key]; !exists {
    dedupMap[key] = c   // keeps FIRST occurrence
}
```
but the comment states "同一 (config_id, cn) 只保留最后一次写入的记录(后写覆盖前写...)" and the Warnf says "同名 cn 冲突,保留最新". Meanwhile `processComputerEntry`'s comment (computer.go:414-415) says the opposite — "toCreate 同名保留第一条" — which matches the code and the `initialSeen` dedup. Two contradictory contracts in one file plus a factually wrong operator-facing log line: a future maintainer "fixing" the code to match the F-02 comment would desynchronize the create dedup from the login-history `initialSeen` keying, and the "保留最新" log actively misleads incident analysis.

**Fix:** Correct the comment and log to "保留第一条" (keep first), or invert the loop to actually keep the last if last-write-wins is the intended contract — but then also flip the `initialSeen` pairing comment. Cheapest correct change:
```go
applogger.Warnf("[电脑同步] 待创建列表去重: %d → %d 条(同名 cn 冲突,保留第一条)", len(computersToCreate), len(deduped))
```

### WR-05: Incremental group-sync paths write raw LDAP strings — SQLSTATE 22021 crash risk the full-sync path already fixed

**File:** `internal/services/addomain/group_sync_service.go:136-146` (SyncSingleGroup upsert struct) and `268-276` (syncGroupEntries update branch)
**Issue:** Both write `entry.GetAttributeValue("cn"/"description")` directly into `models.ADGroup` without `safeAttr`, while the full-sync counterpart (`sync.go:376-377, 388-389`) sanitizes every attribute with `safeAttr` precisely because go-ldap surfaces raw AD bytes (NUL / invalid UTF-8) that PostgreSQL rejects with SQLSTATE 22021 — the exact failure class documented at `sync.go:705-710` (ad-sync-500-nul-byte). A single poisoned group attribute breaks `SyncSingleGroup` (returns error to the API) or aborts the incremental `syncGroupEntries` batch mid-loop, and — interacting with this phase's changes — the unsanitized write now sits on the revival path (`deleted_at` in `AssignmentColumns`), i.e. code this phase modified. Provenance: pre-existing inconsistency; file modified by this phase.

**Fix:** Mirror the full-sync pipeline:
```go
group := &models.ADGroup{
    ...
    GroupName:   safeAttr(entry.GetAttributeValue("cn"), 255),
    Description: safeAttr(entry.GetAttributeValue("description"), 4000),
    ...
}
```
and the same for `groupName`/`description` locals in `syncGroupEntries` (lines 268-269).

### WR-06: Stringly-typed `syncType` guard makes cleanup silently no-op on any caller literal drift

**File:** `internal/services/addomain/shadow_cleanup_service.go:38`; call sites `internal/scheduler/ad_sync_tasks.go:332, 353-356`; `internal/api/v1/system/ad_domain_handler.go:262-264`
**Issue:** SHADOW-03 gates on `syncType != string(models.ADSyncTypeFull)` with an exact, case-sensitive compare — and cleanup failure/skip produces no log. Every caller passes raw strings: the scheduler hardcodes `"full"`, the job-param path forwards whatever `params["syncType"]` contains, and the API handler forwards the client's value (defaulting only when empty). If anyone passes `"Full"`, `"FULL"`, or the constant value is ever reworded, the entire cleanup feature silently stops executing — no error, no warning, no counter. The phase made this exact string load-bearing for the first time without adding any drift detection.

**Fix:** Normalize at the boundary or compare case-insensitively, and log the skip when a non-full type carries deletable data:
```go
if !strings.EqualFold(syncType, string(models.ADSyncTypeFull)) {
    return nil
}
```
and/or replace the three raw `"full"` literals with `string(models.ADSyncTypeFull)` so the constant is the single source of truth.

## Info

### IN-01: `queryAllComputerNames` is now a misnomer and its contract changed silently

**File:** `internal/services/addomain/computer.go:546-557`
**Issue:** After the Unscoped change the function returns all rows (including soft-deleted) as full structs — not just names, and not only live rows. The doc comment explains the revival rationale but the name still promises "names". Rename to `queryAllComputersIncludingDeleted` (or similar) at the next touch to keep the F-01/F-02-era call graph readable.

### IN-02: Redundant explicit `deleted_at IS NULL` under the default GORM scope

**File:** `internal/services/addomain/shadow_cleanup_service.go:76, 133, 192`; also `internal/services/addomain/group_sync_service.go:378`
**Issue:** These queries filter by model (`models.ADUser` etc.) whose `gorm.DeletedAt` already appends `deleted_at IS NULL`; the explicit predicate duplicates it. Harmless but noise — it also implies the author was unsure which scope was active, which is exactly the confusion that produced the (correct) comment at computer.go:542-545. Drop the explicit predicate or add a one-line comment noting it is belt-and-braces.

---

_Reviewed: 2026-10-21T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
