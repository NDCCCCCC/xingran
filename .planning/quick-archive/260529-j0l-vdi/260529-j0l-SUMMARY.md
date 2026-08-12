---
phase: quick
plan: 260529-j0l
subsystem: vdi
tags: [vdi, virtual-machine, cleanup, full-stack]
dependency_graph:
  requires: []
  provides: [vdi-vm-status-removal]
  affects: [vdi-vm-list, vdi-sync]
tech_stack:
  added: []
  patterns: [database-migration, dto-cleanup]
key_files:
  created:
    - internal/core/db/migrations/migration_143_vdi_remove_vm_status.go
  modified:
    - internal/models/vdi.go
    - internal/services/vdi/vm_service.go
    - internal/services/vdi/vm_service_impl.go
    - internal/core/db/database.go
    - xingran-react-frontend/src/types/vdi.ts
    - xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx
decisions:
  - "Removed status field entirely rather than deprecating, since it was always 0 and power_state already provides operational status"
metrics:
  duration: 10m
  completed: "2026-05-29T06:00:00Z"
---

# Phase quick Plan 260529-j0l: Remove VDI VM Status Column Summary

Removed the redundant "status" column from the VDI Virtual Machine list page and full backend stack. The status field was always 0 (normal) because VMs are synced from the VDI server and their actual operational state is shown by the "power_state" column.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Remove status field from backend model, DTO, and service | a2ae7f1 | internal/models/vdi.go, internal/services/vdi/vm_service.go, internal/services/vdi/vm_service_impl.go |
| 2 | Add database migration, update frontend types and UI | f222f31 | internal/core/db/migrations/migration_143_vdi_remove_vm_status.go, internal/core/db/database.go, xingran-react-frontend/src/types/vdi.ts, xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx |

## Changes Made

### Backend

- **internal/models/vdi.go**: Removed `Status int` field from `VDIVirtualMachine` struct. `VDIServer.Status`, `VDIResourceGroup.Status`, and `VDIUserBinding.Status` remain untouched.
- **internal/services/vdi/vm_service.go**: Removed `Status` from `VDIVMDTO`, `UpdateVMRequest`, and `ListVMRequest` DTOs.
- **internal/services/vdi/vm_service_impl.go**: Removed `Status: 0` from three VM creation points (new record, soft-deleted re-creation, CreateVM), removed status filter from `ListVMs`, removed status update from `UpdateVM`, removed `Status` from `toDTO` mapping.
- **internal/core/db/migrations/migration_143_vdi_remove_vm_status.go**: New migration that drops the `status` column from `sys_vdi_vm` table.
- **internal/core/db/database.go**: Added `migrations` package import and registered `Migrate143VDIRemoveVMStatus` call after existing credential migration.

### Frontend

- **xingran-react-frontend/src/types/vdi.ts**: Removed `status` from `VirtualMachine`, `VMListParams`, and `UpdateVMRequest` interfaces.
- **xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx**: Removed the "status" column (with green/red Tag rendering) from the VM list table. The "power_state" column remains.

## Decisions Made

1. **Full removal vs deprecation**: Chose to fully remove the field across the stack since it carried no meaningful data (always 0) and `power_state` already provides the VM operational status.

## Deviations from Plan

None - plan executed exactly as written.

## Pre-existing Issues (Out of Scope)

- `internal/services/vdi/vm_service_impl.go` references model fields (`CPUNumber`, `CPUCore`, `CPUPer`, `MemoryPer`, `DiskPer`, `IPType`, `SubnetMask`, `DefaultGateway`, `NameServer`, `AssignIP`) that do not exist on the current `VDIVirtualMachine` model in `internal/models/vdi.go`. This is a pre-existing model-service mismatch unrelated to this task.
- `internal/services/addomain/` has compilation errors from undefined types (`DeptGroupMappingService`, `models.DeptGroupMapping`, etc.) -- pre-existing and out of scope.

## Self-Check

- Models package compiles: YES
- Migrations package compiles: YES
- No `Status` references remain in `VDIVirtualMachine` model: YES
- No `status` in `VirtualMachine`, `VMListParams`, `UpdateVMRequest` types: YES
- Migration file exists and registered: YES
- `power_state` column preserved in frontend: YES
- Commit a2ae7f1 exists: YES
- Commit f222f31 exists: YES
