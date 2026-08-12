---
phase: 25
slug: vm-data-scope-permissions
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-02
---

# Phase 25 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go test (backend), Vitest (frontend) |
| **Config file** | `xingran-react-frontend/vitest.config.ts` |
| **Quick run command** | `go test ./internal/services/vdi/... -v` |
| **Full suite command** | `go test ./... && cd xingran-react-frontend && npm run test` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/services/vdi/... -v`
- **After every plan wave:** Run `go test ./... && cd xingran-react-frontend && npm run test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 25-01-01 | 01 | 1 | D-01 | T-25-01 | DELETE removes only vdi:vm:operate, INSERT adds 5 granular permissions | integration | `go test ./internal/core/db/migrations/... -v -run TestMigration131` | ❌ W0 | ⬜ pending |
| 25-02-01 | 02 | 1 | D-02, D-03 | T-25-02 | ApplyVMDataScopeFilter implements 5 scope rules + NULL handling | unit | `go test ./internal/services/vdi/... -v -run TestApplyVMDataScopeFilter` | ❌ W0 | ⬜ pending |
| 25-02-02 | 02 | 1 | D-02, D-03 | T-25-02 | VMService calls filter before list query | integration | `go test ./internal/services/vdi/... -v -run TestVMServiceListWithFilter` | ❌ W0 | ⬜ pending |
| 25-03-01 | 03 | 2 | D-05 | T-25-03 | DataScopePermission middleware populates context | unit | `go test ./pkg/middleware/... -v -run TestDataScopePermission` | ✅ | ⬜ pending |
| 25-03-02 | 03 | 2 | D-01, D-05 | T-25-03 | RequirePermissions middleware checks granular permissions | unit | `go test ./pkg/middleware/... -v -run TestRequirePermissions` | ✅ | ⬜ pending |
| 25-03-03 | 03 | 2 | D-05 | T-25-03 | StartVM/StopVM/RestartVM handlers call service with fixed actions | integration | `go test ./internal/api/v1/vdi/... -v -run TestVMHandlers` | ❌ W0 | ⬜ pending |
| 25-04-01 | 04 | 2 | D-06 | — | vmOperationButtons maps permissions to buttons | unit | `cd xingran-react-frontend && npm run test -- src/pages/vdi/VirtualMachineList/vmOperationButtons.test.ts` | ❌ W0 | ⬜ pending |
| 25-04-02 | 04 | 2 | D-06 | — | renderOperationButtons filters by authStore.permissions | integration | `cd xingran-react-frontend && npm run test -- src/pages/vdi/VirtualMachineList/index.test.tsx` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/services/vdi/vm_data_scope_filter_test.go` — stubs for D-02, D-03 (5 scope rules + NULL handling)
- [ ] `internal/core/db/migrations/131_add_vdi_granular_permissions_test.go` — stub for D-01 (permission migration)
- [ ] `internal/api/v1/vdi/vm_handler_test.go` — stubs for D-05 (StartVM/StopVM/RestartVM)
- [ ] `xingran-react-frontend/src/pages/vdi/VirtualMachineList/vmOperationButtons.test.ts` — stubs for D-06 (button config)
- [ ] `xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.test.tsx` — stubs for D-06 (dynamic rendering)

*Existing infrastructure:* Go test framework exists, Vitest config exists in frontend

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Frontend button visibility | D-06 | Requires UI interaction + auth state | 1. Create test user with specific permissions (e.g., only vdi:vm:start) 2. Login as test user 3. Navigate to VM list page 4. Verify only "开机" button is visible, other operation buttons hidden |
| Data scope filtering visual verification | D-02, D-03 | Requires realistic test data + user roles | 1. Create VM with bound_user_id = user_A 2. Create user_B in same department 3. Login as user_B with DataScope=3 (本部门) 4. Verify VM is visible in list 5. Login as user_C in different department 6. Verify VM is NOT visible |

*All critical behaviors have automated verification. Manual tests are for UX validation.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (5 new test files needed)
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

---

*Phase: 25-vm-data-scope-permissions*
*VALIDATION.md created: 2026-06-02*
