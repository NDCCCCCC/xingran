---
phase: quick
plan: 260528-vxy
subsystem: vdi
tags: [frontend, backend, vdi, vm-creation, ux-improvement]
dependency_graph:
  requires: [vdi-server-table, vdi-resource-group-table]
  provides: [resource-groups-api, create-vm-modal-redesign]
  affects: [vdi-vm-create-flow]
tech-stack:
  added: [antd-slider, vdi-resource-group-api]
  patterns: [cascading-dropdown, auto-name-generation]
key-files:
  created: []
  modified:
    - internal/services/vdi/vm_service.go
    - internal/services/vdi/vm_service_impl.go
    - internal/api/v1/vdi/vm_handler.go
    - internal/api/v1/vdi/vm_router.go
    - xingran-react-frontend/src/types/vdi.ts
    - xingran-react-frontend/src/lib/vdiApi.ts
    - xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx
decisions:
  - Resource groups queried from local DB (not VDI API) for fast dropdown response
  - Memory slider uses MB internally (matching existing data model), displays GB in tooltip
  - Auto-name uses resource group name as prefix with optional custom suffix
metrics:
  duration: 5m 17s
  completed: 2026-05-28
  tasks_completed: 2
  files_modified: 7
---

# Quick Task 260528-vxy: VDI Create VM Form UX Improvement Summary

Redesigned VDI virtual machine creation form with cascading dropdown selectors, slider controls for resource allocation, and auto-generated VM names from resource group names.

## Changes Made

### Task 1: Backend Resource Group List API
- Added `VDIResourceGroupDTO` type and `ListResourceGroups` method to `VMService` interface
- Implemented `ListResourceGroups` in `vm_service_impl.go` querying `sys_vdi_resource_group` table from local DB
- Supports optional `vdi_server_id` filter; returns only enabled groups (`status = 0`)
- Added `ListResourceGroups` handler and registered `POST /vdi/vm/resource-groups` route
- **Commit:** 81e934b

### Task 2: Frontend Create VM Modal Redesign
- Added `VDIResourceGroup` TypeScript type and `listResourceGroups` API method
- Replaced UUID text inputs with `Select` dropdowns for VDI server and resource group
- Resource group dropdown cascades from selected VDI server selection
- VM name auto-generates from selected resource group name + optional custom suffix
- Replaced `InputNumber` with `Slider` controls for CPU (1-16), CPU cores (1-32), memory (0.5-64GB), and disk (20-500GB)
- VDI servers loaded on modal open; resource groups loaded on server selection change
- **Commit:** 371aa50

## Verification

- `go build ./internal/api/v1/vdi/... ./internal/services/vdi/...` -- passed
- `npx tsc --noEmit` -- passed (zero errors)

## Deviations from Plan

None - plan executed exactly as written.

## Self-Check: PASSED

- [x] `internal/services/vdi/vm_service.go` -- FOUND
- [x] `internal/services/vdi/vm_service_impl.go` -- FOUND
- [x] `internal/api/v1/vdi/vm_handler.go` -- FOUND
- [x] `internal/api/v1/vdi/vm_router.go` -- FOUND
- [x] `xingran-react-frontend/src/types/vdi.ts` -- FOUND
- [x] `xingran-react-frontend/src/lib/vdiApi.ts` -- FOUND
- [x] `xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx` -- FOUND
- [x] Commit 81e934b -- FOUND
- [x] Commit 371aa50 -- FOUND
