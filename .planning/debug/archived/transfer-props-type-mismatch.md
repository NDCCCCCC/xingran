---
slug: transfer-props-type-mismatch
status: resolved
trigger: TypeScript build errors in ad-domain/ous/index.tsx after code changes
created: 2026-05-28
updated: 2026-05-28
---

# Debug Session: Transfer Props Type Mismatch

## Symptoms

### Expected behavior
Frontend React app should build without TypeScript errors after recent code changes.

### Actual behavior
TypeScript compilation fails with 2 type errors in `src/pages/ad-domain/ous/index.tsx`:
1. **Line 455**: `onChange` handler type mismatch - expects `(targetKeys: Key[], direction: TransferDirection, moveKeys: Key[]) => void` but got `(targetKeys: string[]) => Promise<void>`
2. **Line 468**: `filterOption` return type mismatch - expects `boolean` but got `boolean | undefined`

### Error messages
```
TS2322: Type '(targetKeys: string[]) => Promise<void>' is not assignable to type '(targetKeys: Key[], direction: TransferDirection, moveKeys: Key[]) => void'.
TS2322: Type 'boolean | undefined' is not assignable to type 'boolean'.
```

### Timeline
Started immediately after recent code changes (user reported: "刚才修改代码后出现的")

### Reproduction
Run `npm run build` in xingran-react-frontend directory.

## Current Focus

### Hypothesis
The Ant Design Transfer component props have stricter type requirements in the current version. The `onChange` handler must accept 3 parameters (targetKeys, direction, moveKeys) and `filterOption` must return `boolean` (not `boolean | undefined`).

### Next action
Gather initial evidence by reading the problematic file and examining the Transfer component usage.

### Test plan
- Read `src/pages/ad-domain/ous/index.tsx` lines 450-475
- Compare current implementation with Ant Design Transfer API
- Fix type signatures to match component requirements

### Expected result
TypeScript compilation succeeds after fixing type mismatches.

## Evidence

## Eliminated

## Resolution

### Root cause
(TBD)

### Fix applied
(TBD)

### Verification
(TBD)

### Files changed
(TBD)

### Evidence gathered

**File examined**: `xingran-react-frontend/src/pages/ad-domain/ous/index.tsx`

**Line 161**: `handleGroupTransfer` function signature:
```typescript
const handleGroupTransfer = async (targetKeys: string[]) => {
```

**Line 455**: Transfer component onChange prop:
```typescript
onChange={handleGroupTransfer}
```

**Line 468**: filterOption implementation:
```typescript
filterOption={(inputValue, item) =>
  item.title?.toLowerCase().includes(inputValue.toLowerCase()) ||
  item.description?.toLowerCase().includes(inputValue.toLowerCase())
}
```

**Root cause identified**: 
1. `onChange` handler accepts only `(targetKeys: string[])` but Ant Design Transfer expects `(targetKeys: Key[], direction: TransferDirection, moveKeys: Key[]) => void`
2. `filterOption` returns `boolean | undefined` due to optional chaining (`item.title?....`) but expects `boolean`


## ROOT CAUSE FOUND

The TypeScript build errors were caused by type mismatches between the Ant Design Transfer component's expected prop signatures and the actual implementation:

### Root cause
1. **onChange handler signature mismatch**: The `handleGroupTransfer` function only accepted `(targetKeys: string[])` but Ant Design Transfer's onChange prop expects `(targetKeys: Key[], direction: TransferDirection, moveKeys: Key[]) => void`
2. **filterOption return type mismatch**: The filterOption function returned `boolean | undefined` due to optional chaining (`?.`) on potentially undefined properties, but Transfer expects a strict `boolean` return type

### Fix applied
**File**: `xingran-react-frontend/src/pages/ad-domain/ous/index.tsx`

**Fix 1** (lines 161-165): Updated `handleGroupTransfer` signature:
```typescript
// Before
const handleGroupTransfer = async (targetKeys: string[]) => {

// After  
const handleGroupTransfer = async (
  targetKeys: React.Key[],
  direction: 'right' | 'left',
  moveKeys: React.Key[]
) => {
```

**Fix 2** (lines 174, 176): Convert React.Key[] to string[] in function body:
```typescript
const toAdd = targetKeys.map(String).filter(id => !currentMappedIds.includes(id));
const toRemove = currentMappedIds.filter(id => !targetKeys.map(String).includes(id));
```

**Fix 3** (line 176): Convert groupId to string when finding group:
```typescript
const group = allGroups.find(g => g.id === String(groupId));
```

**Fix 4** (lines 472-476): Ensured filterOption always returns boolean:
```typescript
// Before
filterOption={(inputValue, item) =>
  item.title?.toLowerCase().includes(inputValue.toLowerCase()) ||
  item.description?.toLowerCase().includes(inputValue.toLowerCase())
}

// After
filterOption={(inputValue, item) => {
  const titleMatch = item.title?.toLowerCase().includes(inputValue.toLowerCase()) ?? false;
  const descMatch = item.description?.toLowerCase().includes(inputValue.toLowerCase()) ?? false;
  return titleMatch || descMatch;
}}
```

### Verification
✅ Build succeeded: `npm run build` completed without TypeScript errors
✅ No runtime behavior changes - only type signature updates
✅ React.Key types properly converted to strings for business logic

### Files changed
- `xingran-react-frontend/src/pages/ad-domain/ous/index.tsx` (4 fixes applied)

## DEBUG COMPLETE

