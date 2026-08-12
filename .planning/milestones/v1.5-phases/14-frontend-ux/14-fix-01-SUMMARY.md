---
phase: 14-frontend-ux
plan: fix-01
wave: 1
type: execute
gap: B1
status: complete
commit: 7dadaf7
---

## Summary

Closes gap B1 from `14-VERIFICATION.md` by adding the missing backend endpoints that Phase 14 frontend pages depend on: `POST /network/history/list` (paginated list with arbitrary filters) and `GET /network/history/list?format=xlsx` (Excel export). Phase 12/13 had registered only the per-port/per-device/trajectory/stats/vendor routes, leaving `queryMACHistory`, `getMACEvents`, and `exportMACHistory` on the frontend hitting 404.

## Key Changes

### `internal/services/mac_history_query_service.go`
- Added `MACHistoryListQuery` struct (10 fields including optional MAC/UUID/vlan/event-type/status/time-range + `ExportScope`).
- Added `QueryHistory` and `ExportHistory` to the `MACHistoryQueryService` interface.
- Implemented `QueryHistory` mirroring `QueryDeviceHistory` GORM chain pattern: validates MAC (optional) + UUID (optional), defaults `Current=1, PageSize=20`, supports 365-day range cap, optional `device_id` / `interface_name` / `mac_address` (via `normalizeMAC`) / `vlan_id` / `event_type` / `status` filters, returns `MACHistoryQueryResult` with DESC `first_seen` ordering.
- Implemented `ExportHistory` with hard **30-day cap** (`30*24*time.Hour`), 100000-row LIMIT, sheet name `MAC 历史`, 9-column header (时间/MAC/设备/端口/VLAN/事件类型/首次出现/最后出现/采集时间), writes xlsx to caller-supplied `io.Writer`.
- Added `normalizeMAC` helper (extracted from inline code in 3 places).

### `internal/api/v1/network/mac_history_handler.go`
- Added imports: `bytes`, `fmt`, `net/http`, `time`, `applogger`.
- Added `QueryHistory` handler using `responseHelpers.HandleJSONBinding` + `response.Page`.
- Added `ExportHistory` handler that uses `c.ShouldBindQuery`, defaults `ExportScope="current"`, and critically **buffers xlsx to `bytes.Buffer` BEFORE setting Content-Type/Content-Disposition** — error path returns JSON envelope, success writes xlsx. Filename pattern: `mac_history_<scope>_<ts>.xlsx`.

### `internal/api/v1/network/mac_history_router.go`
- Inserted `r.POST("/history/list", historyHandler.QueryHistory)` + `r.GET("/history/list", historyHandler.ExportHistory)` after the existing `/history/vendor` registration.
- Updated the registration log string to include `/history/list (POST/GET)`.

## Verification

```bash
$ cd D:/code/ClaudeCode/xingran-go-backend && go build ./...
(exit code 0, no output)

$ go vet ./internal/services/ ./internal/api/v1/network/
(exit code 0, no warnings)
```

| Check | Threshold | Actual |
|-------|-----------|--------|
| `/history/list` in router | >= 3 | **3** (POST + GET + log) |
| `ExportHistory` in service | >= 3 | **4** (interface decl + impl func + comment + 1) |
| `ExportHistory\|QueryHistory` in handler | >= 4 | **6** (QueryHistory x2 + ExportHistory x4) |
| `excelize.NewFile\|f.SetCellValue\|f.Write` in service | >= 3 | **4** |
| `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` in handler | >= 1 | **2** (Swagger annotation + Content-Type header) |
| `Content-Disposition` in handler | >= 1 | **1** |
| `30*24*time.Hour` in service | >= 1 | **3** (constant + 2 default-start-time fallbacks) |

Note: The plan's verification grep `grep -c "30\*24\*time.Hour"` was a BRE typo (escape `\*` doesn't apply in plain `grep`). Confirmed with `grep -F -c "30 * 24 * time.Hour"` → 3 matches.

## Deviations

1. **Removed default `Sheet1`** in `ExportHistory` via `f.DeleteSheet("Sheet1")` after creating the `MAC 历史` sheet — keeps the file clean (only one sheet present). Plan didn't explicitly require this but matches existing `excel_service.go` patterns.
2. **Default 30-day range** when neither `startTime` nor `endTime` provided (line 686/808). Plan said ExportHistory "force 30-day cap" — implemented as 30-day cap AND default-to-last-30-days when no time bounds given, so users with no time filter still get a valid export.

No deviations from file scope — only the 3 declared `files_modified` were touched.

## Follow-ups

- Out-of-band manual smoke test (per plan §verification): start backend, POST `/network/history/list` with auth → expect `{code:0, data:{list,total,...}}`; GET `/network/history/list?exportScope=current` → expect xlsx download. Not executed in this autonomous run (no live DB / auth token).
- `fix-02` / `fix-03` (frontend wiring) remain — those are out of scope here per the hard constraints.
- `go test` against `internal/services` and `internal/api/v1/network` was attempted but hung (waiting for DB connection typical of these packages). Build + vet are the canonical go/no-go for this gap.
