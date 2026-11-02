---
slug: remove-vdi-config-ip
description: Remove Config IP functionality from VDI VM list page
created: 2026-06-02
---

# Remove VDI Config IP Functionality

## Objective

Remove the "配置IP" (Config IP) functionality from the virtual machine list page, including frontend buttons/dialogs and backend API endpoints.

## Scope

### Frontend Changes (`xingran-react-frontend/`)
- Remove "配置IP" button from VM list page (`src/pages/vdi/VirtualMachineList/index.tsx`)
- Remove ConfigIP modal dialog component
- Remove `configIP` function from `vdiApi.ts`
- Remove related type definitions

### Backend Changes (`internal/`)
- Remove ConfigIP endpoint from VM handler (`internal/api/v1/vdi/vm_handler.go`)
- Remove ConfigIP route from VM router (`internal/api/v1/vdi/vm_router.go`)
- Remove BatchConfigIP method from VM service (`internal/services/vdi/vm_service_impl.go`)
- Remove ConfigIP interface from VM service interface (`internal/services/vdi/vm_service.go`)

### VDI Client Changes
- Remove ConfigIP method from VDI client (`internal/services/vdi/vdi_client_extended.go`)
- Remove ConfigIP related types from `internal/services/vdi/vdi_types.go`

## Files to Modify

### Frontend
1. `xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx` - Remove ConfigIP button and handler
2. `xingran-react-frontend/src/lib/vdiApi.ts` - Remove configIP method
3. `xingran-react-frontend/src/types/vdi.ts` - Remove VMIPConfigRequest type

### Backend
1. `internal/api/v1/vdi/vm_handler.go` - Remove ConfigIP handler method
2. `internal/api/v1/vdi/vm_router.go` - Remove ConfigIP route
3. `internal/services/vdi/vm_service.go` - Remove BatchConfigIP from interface
4. `internal/services/vdi/vm_service_impl.go` - Remove BatchConfigIP implementation
5. `internal/services/vdi/vdi_client_extended.go` - Remove ConfigIP method
6. `internal/services/vdi/vdi_types.go` - Remove ConfigIPRequestExtended type

## Execution Order

1. Backend changes first (remove API endpoint and service layer)
2. Frontend changes second (remove UI and API client)
3. Test compilation and verify no broken references

## Success Criteria

1. [ ] ConfigIP button removed from VM list page
2. [ ] ConfigIP endpoint removed from backend API
3. [ ] ConfigIP methods removed from service layer
4. [ ] ConfigIP methods removed from VDI client
5. [ ] Frontend compiles without errors
6. [ ] Backend compiles without errors
7. [ ] No broken imports or references
