---
phase: "03-信息点导入设备端口配置"
plan: "01"
subsystem: "operations/excel-import"
tags: [excel, import, infopoint, device, port, reference-resolution]
dependency_graph:
  requires: [existing infoPoint model with device_id/port_id fields, reference_resolver.go]
  provides: [deviceName/portName ExcelColumn entries, template example values for device/port]
  affects: [excel_config.go, excel_service.go]
tech_stack:
  added: []
  patterns: [ExcelColumn Reference pattern, name-to-ID resolution via ReferenceResolver]
key_files:
  created: []
  modified:
    - internal/services/operations/excel_config.go
    - internal/services/operations/excel_service.go
decisions:
  - D-01: Device uses Reference to sys_network_device.device_name (exact match)
  - D-02: Port uses Reference to sys_device_port_status.interface_name (exact match)
  - D-03: Both columns optional, no Required flag, match failure leaves field empty
  - D-04: Port uses global lookup, no DependsOn scoping to device
metrics:
  duration: "170s"
  completed: "2026-04-16"
  tasks: 2
  files: 2
---

# Phase 03 Plan 01: Add deviceName and portName columns to infoPoint Excel import config Summary

Added "所属设备" and "所属端口" columns to infoPoint Excel import configuration, enabling automatic name-to-ID resolution during import via the existing ReferenceResolver pipeline. Configuration-only change -- no model, migration, or frontend modifications needed.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add device and port ExcelColumn entries to infoPoint import config | 3245fe6 | excel_config.go |
| 2 | Add template example values for device and port columns | ae107f4 | excel_service.go |

## Changes Made

### Task 1: excel_config.go
- Added `{Field: "deviceName", Header: "所属设备", MaxLength: 100, Reference: "sys_network_device.device_name", DBField: "device_id"}` to infoPoint Columns
- Added `{Field: "portName", Header: "所属端口", MaxLength: 100, Reference: "sys_device_port_status.interface_name", DBField: "port_id"}` to infoPoint Columns
- Both inserted between workstationName and status columns
- infoPoint config now has 7 columns (was 5)

### Task 2: excel_service.go
- Extended `sourceDeviceName`/`destDeviceName` case to also match `deviceName`, returning "核心交换机A"
- Extended `sourcePort`/`destPort` case to also match `portName`, returning "Gi0/1"
- Consistent with existing example values across the codebase

## Verification Results

| Check | Result |
|-------|--------|
| `go build ./...` | PASS |
| Existing test suite | Pre-existing failure (DB connection required) -- not caused by this change |
| deviceName Reference grep | PASS -- found at line 210 |
| portName Reference grep | PASS -- found at line 211 |
| deviceName example value grep | PASS -- case returns "核心交换机A" |
| portName example value grep | PASS -- case returns "Gi0/1" |
| No Required flag on new columns | PASS |

## Deviations from Plan

None -- plan executed exactly as written.

## Key Decisions

1. **Device reference field**: `sys_network_device.device_name` -- matches on device name column (per D-01)
2. **Port reference field**: `sys_device_port_status.interface_name` -- matches on interface name, not port_name (per D-02)
3. **Optional columns**: Neither column has `Required: true`, so empty values pass through without blocking import (per D-03)
4. **No DependsOn**: Port lookup is global, not scoped to the device column (per D-04)
5. **Example value reuse**: Added deviceName/portName to existing switch case arms rather than creating separate cases, reducing code duplication

## Deferred Issues

Pre-existing test failure in `internal/services/operations/batch_upserter_test.go` -- test requires active database connection and panics on nil GORM instance. This is out of scope for this plan.

## Self-Check: PASSED

| Item | Status |
|------|--------|
| excel_config.go exists | FOUND |
| excel_service.go exists | FOUND |
| 03-01-SUMMARY.md exists | FOUND |
| Task 1 commit 3245fe6 | FOUND |
| Task 2 commit ae107f4 | FOUND |
