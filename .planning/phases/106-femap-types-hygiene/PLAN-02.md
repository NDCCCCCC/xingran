---
phase: 106
plan: "02"
type: execute
wave: 1
depends_on: []
files_modified:
  - xingran-react-frontend/src/pages/operations/server-rooms/index.tsx
  - xingran-react-frontend/src/pages/network/mac/history/MACHistoryPage.tsx
  - xingran-react-frontend/src/pages/network/mac/index.tsx
autonomous: true
requirements_addressed: [FEMAP-02, FEMAP-03]
---

<objective>
Replace all inline status-color ternary patterns in server-rooms, MAC history, and MAC list pages with the centralized tag-config constants added in Plan 01. After Plan 01 lands, NORMAL_STOP_TAG_CONFIG, MAC_HISTORY_STATUS_TAG_CONFIG, and MAC_TYPE_TAG_CONFIG will be available in constants/status.ts.
</objective>

<tasks>

## Prerequisites

Plan 01 must be executed first — this plan depends on `FIX_STATUS_TAG_CONFIG`, `MAC_HISTORY_STATUS_TAG_CONFIG`, and `MAC_TYPE_TAG_CONFIG` being added to `constants/status.ts`.

## Step 1 — Update `server-rooms/index.tsx`

Read `src/pages/operations/server-rooms/index.tsx` around line 417.

**Add import at top of file:**
```typescript
import { NORMAL_STOP_TAG_CONFIG } from "@/constants/status";
```

**Replace inline ternary** (line 417–419):
- Before:
  ```tsx
  <Tag color={room.status === 0 ? "success" : "error"}>
    {room.status === 0 ? "正常" : "停用"}
  </Tag>
  ```
- After:
  ```tsx
  <Tag color={NORMAL_STOP_TAG_CONFIG[room.status]?.color ?? "default"}>
    {NORMAL_STOP_TAG_CONFIG[room.status]?.text ?? (room.status === 0 ? "正常" : "停用")}
  </Tag>
  ```

The fallback to inline text is safe because NORMAL_STOP_TAG_CONFIG covers both 0 and 1 — but if TypeScript narrows the type strictly, the inline text path should never hit.

## Step 2 — Update `MACHistoryPage.tsx`

Read `src/pages/network/mac/history/MACHistoryPage.tsx`.

**Add import at top of file:**
```typescript
import { MAC_HISTORY_STATUS_TAG_CONFIG } from "@/constants/status";
```

**Location 1 — status column (around line 290):**
- Before:
  ```tsx
  <Tag color={record.status === 0 ? "green" : "red"}>
    {record.status === 0 ? "正常" : "停用"}
  </Tag>
  ```
- After:
  ```tsx
  <Tag color={MAC_HISTORY_STATUS_TAG_CONFIG[record.status]?.color ?? "default"}>
    {MAC_HISTORY_STATUS_TAG_CONFIG[record.status]?.text ?? (record.status === 0 ? "正常" : "停用")}
  </Tag>
  ```

**Location 2 — status column in detail list (around line 530):**
Apply the same replacement using `record.status`.

Note: Keep `EVENT_TAG_COLOR` usage for the event type column (already centralized) — do not change it.

## Step 3 — Update `network/mac/index.tsx`

Read `src/pages/network/mac/index.tsx` around line 304.

**Add import at top of file:**
```typescript
import { MAC_TYPE_TAG_CONFIG } from "@/constants/status";
```

**Replace inline MAC-type color** (lines 302–306):
- Before:
  ```tsx
  const option = macTypeOptions.find((o) => o.value === macType);
  const color = macType === "dynamic" ? "blue" : macType === "static" ? "green" : "orange";
  return <Tag color={color}>{option?.label || macType}</Tag>;
  ```
- After:
  ```tsx
  return (
    <Tag color={MAC_TYPE_TAG_CONFIG[macType]?.color ?? "default"}>
      {option?.label || macType}
    </Tag>
  );
  ```

Remove the now-unused `color` variable and the inline ternary. The `option` variable can be kept if used elsewhere, or removed if only the label fallback was needed.

</tasks>

<success_criteria>
- `server-rooms/index.tsx` uses `NORMAL_STOP_TAG_CONFIG` — no inline `room.status === 0 ? "success" : "error"`
- `MACHistoryPage.tsx` uses `MAC_HISTORY_STATUS_TAG_CONFIG` in both locations — no inline `record.status === 0 ? "green" : "red"`
- `network/mac/index.tsx` uses `MAC_TYPE_TAG_CONFIG` — no inline `macType === "dynamic" ? "blue" : ...` ternary
- All imports resolve correctly
- TypeScript compiles with no new errors
- All component render output is visually identical (same colors, same labels)
</success_criteria>

<output>
Create SUMMARY.md on completion
</output>
