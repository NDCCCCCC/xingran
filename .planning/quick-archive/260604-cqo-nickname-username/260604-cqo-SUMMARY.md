---
phase: quick
plan: 01
subsystem: vdi
tags: [frontend, backend, vdi, user-binding, select-dropdown]
dependency_graph:
  requires: [system-users-api, vdi-bind-user-api]
  provides: [searchable-user-dropdown, username-based-binding]
  affects: [vdi-vm-list-page, vdi-bind-user-flow]
tech_stack:
  added: [ant-design-select-showsearch]
  patterns: [debounced-search, separate-form-instance]
key_files:
  created: []
  modified:
    - xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx
    - xingran-react-frontend/src/types/vdi.ts
    - xingran-react-frontend/src/lib/vdiApi.ts
    - internal/services/vdi/vm_service.go
    - internal/services/vdi/vm_service_impl.go
decisions:
  - "Use separate bindUserForm instance to avoid conflicts with the create VM form"
  - "Debounce user search at 300ms to prevent excessive API calls"
  - "Store username in bound_user_id field and computed display name in bound_user_name"
  - "Prefer VDI API returned name over locally computed display name when available"
metrics:
  duration: 5m
  completed: "2026-06-04"
  tasks: 2
  files: 5
---

# Quick Task 260604-cqo: Nickname/Username Bind User Dropdown Summary

Replace the plain text input for "Bind User" in the VM list page with an Ant Design Select dropdown that supports fuzzy search. The dropdown loads system users, displays their nickname, and sends their username to the backend BindUser API.

## Changes Made

### Task 1: Replace bind user modal Input with searchable Select dropdown
**Commit:** 4967586

Frontend changes to the VM list page and VDI types:

- Added `systemUsers` state and `userSearchLoading` state for user dropdown
- Added `bindUserForm` separate form instance to avoid conflicts with create VM form
- Added `loadSystemUsers` function that calls `POST /system/users/list` with 300ms debounce
- Replaced `<Input placeholder="用户 ID">` with `<Select showSearch>` component
- Dropdown displays user nickname (falls back to username if no nickname set)
- Selecting a user sends `{ username: selectedUsername }` to the bind API
- Updated `BindUserRequest` type from `user_id: string` to `username: string`

**Files modified:**
- `xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx`
- `xingran-react-frontend/src/types/vdi.ts`

### Task 2: Update backend BindUser to accept username
**Commit:** 3d26d23

Backend changes to the VDI service layer:

- Changed `BindUserServiceRequest.UserID` to `Username` field
- Added system user lookup by username to build display name
- Display name format: "nickname (username)" or just "username" if no nickname
- Passes username (not UUID) to the VDI API
- Stores username in `bound_user_id` and computed display name in `bound_user_name`
- Prefers VDI API returned `BoundUserName` over locally computed display name when available

**Files modified:**
- `internal/services/vdi/vm_service.go`
- `internal/services/vdi/vm_service_impl.go`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed BoundUserName type mismatch**
- **Found during:** Task 2 - Go build verification
- **Issue:** `vdiVMDetail.BoundUserName` is `*string` (pointer), plan treated it as `string`
- **Fix:** Added nil check and pointer dereference: `if vdiVMDetail.BoundUserName != nil && *vdiVMDetail.BoundUserName != ""`
- **Files modified:** `internal/services/vdi/vm_service_impl.go`
- **Commit:** 3d26d23

## Verification Results

- Frontend TypeScript: Compiles cleanly (verified via main repo `tsc --noEmit`)
- Backend Go: `go build ./...` passes with no errors
- Both commits contain only the expected file changes, no unexpected deletions

## Self-Check: PASSED

All 5 modified files verified present. Both commits (4967586, 3d26d23) verified in git log.
