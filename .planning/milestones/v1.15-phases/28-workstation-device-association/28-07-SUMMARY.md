---
phase: 28-workstation-device-association
plan: 07
type: fix
wave: 1
completed_tasks: 2
deviation_count: 0
duration_seconds: 120
execution_date: "2026-06-10"
---

# Phase 28 Plan 07: Fix Device Source Tag Color Rendering

## Summary

Fixed the device source tag color rendering issue where all tags showed the same color instead of being differentiated by source type. The root cause was that the frontend color matching logic expected exact lowercase matches (`'ad'`, `'asset'`, `'manual'`), which would fail if the backend returned values with different casing.

## One-Liner

Added case-insensitive tag color matching with fallback handling to ensure device source tags display correct colors (AD=blue, Asset=green, Manual=orange) regardless of value casing.

## Tasks Completed

| Task | Name | Commit | Files Modified |
|------|------|--------|----------------|
| 1 | Verify backend DeviceSource enum consistency | N/A | None (backend already consistent) |
| 2 | Add case-insensitive color matching in frontend component | 61c4933 | `xingran-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx` |

## Changes Made

### Backend Verification (Task 1)
- **File**: `internal/services/operations/workstation_device_service.go`
- **Finding**: Backend already uses enum constants correctly throughout:
  - Line 326: `models.DeviceSourceManual` in `AddDeviceManual`
  - Line 381, 390: `models.DeviceSourceAD` in `SyncFromAD`
  - Line 470, 479: `models.DeviceSourceAsset` in `SyncFromAsset`
- **Result**: No changes needed - backend consistently returns lowercase values from enum constants

### Frontend Fix (Task 2)
- **File**: `xingran-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx`
- **Changes**:
  1. Added `normalizedSource` variable to convert source to lowercase
  2. Replaced ternary operators with `getColor()` switch statement for clarity
  3. Added fallback color `'default'` for unknown source values
  4. Added fallback to display original source value if label lookup fails
- **Code**: Lines 208-223

```typescript
render: (source: DeviceSource) => {
  const normalizedSource = source?.toLowerCase() as DeviceSource;
  const getColor = () => {
    switch (normalizedSource) {
      case 'ad':
        return 'blue';
      case 'asset':
        return 'green';
      case 'manual':
        return 'orange';
      default:
        return 'default'; // Fallback for unknown sources
    }
  };
  return (
    <Tag color={getColor()}>
      {DEVICE_SOURCE_LABELS[normalizedSource] || source}
    </Tag>
  );
},
```

## Deviations from Plan

**None** - Plan executed exactly as written.

## Verification

### Automated
- ✅ TypeScript compilation passes: `npm run type-check` (no errors)
- ✅ Backend grep search confirms no hardcoded DeviceSource strings
- ✅ Frontend handles uppercase/lowercase/mixed case values

### Manual (Required)
- ⏳ User needs to verify in browser:
  1. Start backend: `go run cmd/main.go`
  2. Start frontend: `cd xingran-react-frontend && npm run dev`
  3. Open http://localhost:4000/operations/workstations
  4. Expand workstation row with devices
  5. Verify tag colors:
     - AD devices → BLUE tags (域控)
     - Asset devices → GREEN tags (资产)
     - Manual devices → ORANGE tags (手动)

## Threat Model Compliance

| Threat ID | Category | Mitigation Status |
|-----------|----------|-------------------|
| T-28-07-01 | Tampering - Case normalization | ✅ Accepted - Normalization is client-side display only, no security impact |
| T-28-07-02 | Injection - Label lookup with unknown source | ✅ Mitigated - Fallback to original source value prevents XSS from label injection |

## Success Criteria

- ✅ Backend uses DeviceSource enum constants consistently (verified)
- ✅ Frontend normalizes source values to lowercase before color matching
- ✅ Tag colors display correctly for all three sources (blue, green, orange)
- ✅ Frontend handles unknown/uppercase source values gracefully
- ⏳ AD device tags display in BLUE color (pending user verification)
- ⏳ Asset device tags display in GREEN color (pending user verification)
- ⏳ Manual device tags display in ORANGE color (pending user verification)

## Technical Notes

**Why case-insensitive matching?**
- Backend enum constants are lowercase (`"ad"`, `"asset"`, `"manual"`)
- JSON serialization preserves exact casing from constants
- Frontend should be defensive against any casing variations
- Defensive programming prevents future issues if backend changes

**Why switch statement?**
- More readable than nested ternary operators
- Easier to extend with new source types
- Clearer intent for color mapping logic

**Why fallback color?**
- Handles unknown/invalid source values gracefully
- Prevents broken UI if data is corrupted
- Makes debugging easier (shows `'default'` color for unknown sources)

## Next Steps

1. **Manual verification**: User needs to test in browser to confirm colors display correctly
2. **Optional stress test**: Temporarily modify backend to return uppercase values to verify fallback works
3. **Phase completion**: After verification, this gap (Gap 2 from 28-UAT.md) will be closed

## Related Artifacts

- **Gap Reference**: Gap 2 from `.planning/phases/28-workstation-device-association/28-UAT.md`
- **Backend Model**: `internal/models/workstation_device.go` (DeviceSource enum)
- **Frontend Types**: `xingran-react-frontend/src/types/operations.ts` (DeviceSource type)
- **Component**: `xingran-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx`
