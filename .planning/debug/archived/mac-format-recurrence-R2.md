# Debug Report: MAC Format Recurrence R2 (2026-07-02)

**Trigger**: "mac地址采集又出现了格式问题...请检查是否还有其他定时任务导致该问题"
**Mode**: diagnose-only (no source/db modification)
**Date**: 2026-07-02

---

## 1. Dirty Data Quantification (R2 baseline)

### sys_device_mac_address
- **Total**: 770 rows
- **Bad MAC format** (not `^[A-F0-9]{2}(:[A-F0-9]{2}){5}$`): **525 rows**
- **Bad interface_name** (full prefix or invalid): **761 rows**

### sys_device_mac_history
- **Total**: 18,542 rows
- **Bad MAC**: **18 rows**
- **Bad interface**: **18,522 rows** (full-prefix historical data, mostly legacy)

### Time distribution (last 7 days)
- **ALL** 525 mac_address bad rows were created at `2026-07-02 16:00:01` ~ `16:01:06`
- mac_history bad rows also at `2026-07-02 16:00`

### Per-batch breakdown (by device, last 24h, batched at HH:00 cron)
| device | hostname | vendor | bad_rows | batch_time |
|---|---|---|---|---|
| a0023b78-... | CX-WH-RUITONG-26F-SWL3-HW-S8700 | huawei | 132 | 16:00:01.961 |
| 060e5a69-... | CX-WH-WH-05F-FL-RS8607E-02 | ruijie | 76 | 16:00:01.857 |
| af45ee9c-... | CX-WH-WH-47F-FL-RS8607E-01 | ruijie | 65 | 16:00:01.968 |
| ... | ... | ... | ... | ... |
| **e61e0625-...** | **CX_HB_SY_ZHUSHAN_RJ_RSR10** | **ruijie** | **8** | **16:01:06.254** |
| f27b6387-... | CXHUB-SW-HW-S5735-XN-xianAn-01 | huawei | 6 | 16:01:03.752 |

**Key observation**: All bad batches happen at exactly HH:00:00 ~ HH:01:06 — **aligns precisely with `MAC地址采集` (mac_collection) cron**.

### Sample of bad rows (dev `e61e0625-...`, Ruijie RSR10)
- All 8 MACs have `interface_name = "FastEthernet 1/0"` (with SPACE)
- All MACs in **Cisco dot format** (e.g. `9c7b.ef2f.0222`)
- `mac_type = dynamic`, single vlan

This format CANNOT be produced by normalize.MACAddress (which returns `9C:7B:EF:2F:02:22`) or normalize.InterfaceName (which returns `FE1/0`).

---

## 2. Job Logs (sys_job_log) Evidence

`sys_job_log` confirms at `2026-07-02 16:00:00`:
- **`MAC地址采集` (mac_collection) — duration 90,130ms (one tick) AND 66,229ms (another)** — both succeeded
- `端口状态采集` (port_collection) — succeeded
- `设备信息更新` (device_info_update) — succeeded
- `MAC历史物化视图刷新` (mac_history_matview_refresh) — read-only (refreshes MV)

**Conclusion from logs**: mac_collection cron is the writer. Duration 90s + 66s = ~156 seconds spans exactly the dirty data window 16:00:01 ~ 16:01:06.

**No other cron writes to `sys_device_mac_address` or `sys_device_mac_history`**. The 3 MAC-related cron tasks (mac_history_cleanup, mac_history_purge_monthly, mac_history_matview_refresh) are all DELETE-only or MV REFRESH.

---

## 3. Three Hypotheses Verification

### Hypothesis A (P0): GOCACHE stale binary
**Investigation:**
- Running process: `main.exe` at `C:\Users\CPIC\AppData\Local\Temp\go-build1579591251\b001\exe\main.exe`
  - StartTime: 2026-07-02 15:43:43 (today)
  - Size: 58MB
- Disk binary: `xingran-backend.exe` at `D:\code\ClaudeCode\xingran-go-backend\xingran-backend.exe`
  - Modify: 2026-06-30 13:35 (BEFORE fixes; 126MB)
- GOCACHE size: **1.3G** (still — never cleaned after last fix)
- Source mtimes: mac_collection_service.go at 2026-07-02 00:20; model hooks at 2026-07-01 16:49 — ALL before binary build at 15:43:43
- Last source commit: `d64da6b3` (2026-07-02 00:39 SECURITY suffix), `17459ec9` (2026-07-01 17:40 normalize refactor)

**Binary disassembly verification** (`b322/_pkg_.a` = models package, `b437/_pkg_.a` = services package):
- `DeviceMACAddress.BeforeCreate` IS PRESENT (calls `normalize.MACAddress` at line 46 + `normalize.InterfaceName` at line 47)
- `DeviceMACHistory.BeforeCreate` IS PRESENT (similar)
- `MACCollectionService.parseMACLine` IS PRESENT (calls `pkg/normalize.MACAddress`, `pkg/normalize.InterfaceName` via portcollection/utils.go:21 wrapper, `cleanTimestampFromInterface`)
- All three normalize layers confirmed in binary

**Verdict**: PARTIALLY TRUE — binary was rebuilt today (15:43:43), not stale from June 30. **GOCACHE issue from previous session is no longer the cause.**

### Hypothesis B (P1): Some other cron task bypasses GORM hook
**Investigation:**
- Searched all cron tasks registered in `internal/scheduler/cron.go` — only `mac_collection` writes to `sys_device_mac_address` / `sys_device_mac_history`
- Searched all production code paths in `internal/services/` writing to these tables: only `internal/services/mac_collection_service.go` line 291-292 (`s.db.WithContext(ctx).Create(&macRecords)`) and `internal/services/mac_history_service.go` line 260 (`Create(&historyRecords)`)
- No `db.Exec("INSERT INTO sys_device_mac_address...")` in production code (only test files)
- No `Session{SkipHooks: true}` usage
- All untracked cmd/* scripts (mac_cleanup, mac_merge_real, mac_purge_*, mac_space_diag) are read-only or DELETE-only against these tables
- `workstation_device_service.go` has uncommitted SELECT-only changes adding `latest_mac_history` CTE — does NOT write

**Verdict**: FALSE — user's hypothesis is WRONG. No other cron writes to MAC tables. mac_collection IS the writer.

### Hypothesis C (P2): M196 DB TRIGGER missing / bypassed
**Investigation:**
- Per memory file `go-run-cache-stale-mac-recurrence.md`: "M196 DB-level TRIGGER" was proposed as P0 defense
- Actual M196 (`internal/core/db/migrations/migration_196_reconciliation_dict_labels_align.go`): **fixes reconciliation dict labels, NOT a TRIGGER**
- DB check: `SELECT tgname FROM pg_trigger WHERE tgrelid IN ('sys_device_mac_address'::regclass, 'sys_device_mac_history'::regclass) AND NOT tgisinternal` returns **0 rows** — **NO TRIGGERS**

**Verdict**: TRUE — defense-in-depth trigger was RECOMMENDED but NEVER IMPLEMENTED. This is a future work item.

---

## 4. Reconciling the Contradiction

The running binary (verified via objdump) contains all normalize fixes. Yet the dirty data was written TODAY (16:00 ~ 16:01) by mac_collection cron with that binary. The Cisco-format MAC `9c7b.ef2f.0222` and "FastEthernet 1/0" (with space) CANNOT be produced by normalize.MACAddress / normalize.InterfaceName.

Possible explanations (in order of likelihood, none verified in this diagnose-only pass):
1. **GORM slice insertion bypasses BeforeCreate hook** — `s.db.Create(&macRecords)` where `macRecords` is `[]*models.DeviceMACAddress` — in some GORM versions, slice-of-pointers insertion may bypass hooks on certain code paths. NEEDS runtime verification.
2. **mac_collection_service.CollectDevice is called concurrently** and a stale connection/cache holds the older (pre-fix) parsed MAC values
3. **`go run` process reload behavior** — running binary at 15:43:43 may have built from a snapshot before final source edits, even though file mtimes look correct. The user's WIP uncommitted changes (stashes show `stash@{0}: pre-merge-2: drop info-point cols`) suggest non-trivial in-progress edits.

**Recommended next step (NOT applied here per diagnose-only constraint)**: Stop the running binary, manually run `go build -o /tmp/test-mac-col.bin cmd/main.go && /tmp/test-mac-col.bin` against the same DB, manually trigger mac_collection, observe whether new inserts are clean. If clean → confirms hypothesis 1 (slice insertion) or 2 (concurrency). If still dirty → hypothesis 3 (build inconsistency).

---

## 5. User's Hypothesis Verdict

**"是否有其他定时任务导致该问题" — Answer: NO.**

Evidence:
- sys_job_log shows only `MAC地址采集` (mac_collection) writes to MAC tables at 16:00
- All other MAC-related cron tasks (mac_history_cleanup, mac_history_purge_monthly, mac_history_matview_refresh) are DELETE-only or VIEW REFRESH
- mac_collection cron is the ONE and ONLY writer

---

## 6. Recommended Next Steps

1. **Stop running binary and verify fresh build**: `taskkill /F /PID 66616` (PID of running main.exe), then `go build -o xingran-backend.exe ./cmd/main.go && ./xingran-backend.exe`, then manually trigger mac_collection and verify bad=0
2. **Implement M196 DB TRIGGER** (as recommended in memory file `go-run-cache-stale-mac-recurrence.md`): BEFORE INSERT/UPDATE trigger on `sys_device_mac_address` + `sys_device_mac_history` that REGEXP_REPLACE-normalizes mac_address and interface_name. This is defense-in-depth — even if Go binary/bypass bugs recur, DB level guarantees clean data.
3. **Document production launch command**: change from `go run cmd/main.go` to `go build -o xingran-backend.exe cmd/main.go && ./xingran-backend.exe` to avoid GOCACHE surprises (per same memory file)
4. **Run M194 clean** after the issue is fixed (idempotent re-normalization)

---

## Files Referenced
- Running binary: `C:\Users\CPIC\AppData\Local\Temp\go-build1579591251\b001\exe\main.exe`
- Production writer: `internal/services/mac_collection_service.go:291`
- Hook source: `internal/models/device_mac_address.go:42`
- Normalize package: `pkg/normalize/mac.go`, `pkg/normalize/iface.go`
- M194 (re-normalize): `internal/core/db/migrations/migration_194_clean_iface_security_suffix.go`
- Verify tool: `scripts/verify/format_unify/main.go`
- Diagnostic script: `cmd/diag_mac_recurrence_r2/main.go` (delete after use)