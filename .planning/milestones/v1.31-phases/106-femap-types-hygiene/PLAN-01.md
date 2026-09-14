---
phase: 106
plan: "01"
type: execute
wave: 1
depends_on: []
files_modified:
  - xingran-react-frontend/src/constants/status.ts
  - xingran-react-frontend/src/pages/asset/reconciliation/fix-suggestion/index.tsx
  - xingran-react-frontend/src/pages/asset/reconciliation/fix-suggestion/components/FixSuggestionDetailDrawer.tsx
autonomous: true
requirements_addressed: [FEMAP-01, FEMAP-02]
---

<objective>
Add FIX_STATUS_OPTIONS / FIX_STATUS_TAG_CONFIG, MAC_HISTORY_STATUS_OPTIONS / MAC_HISTORY_STATUS_TAG_CONFIG, and MAC_TYPE_OPTIONS / MAC_TYPE_TAG_CONFIG to the shared status constants file, then migrate the duplicate fixStatusColor/fixStatusLabel objects out of the two fix-suggestion files into the centralized constants. Also fix the inconsistency where FixSuggestionDetailDrawer used rolled_back: "magenta" instead of "orange".
</objective>

<tasks>

## Step 1 — Add new constants to `src/constants/status.ts`

Read `src/constants/status.ts` (lines 1–69). Append three new constant blocks before the closing `}` of the file.

**Block A — FixStatus (6-state FixSuggestion workflow):**

```typescript
// 对齐 fixSuggestionApi.FixStatus: pending/accepted/rejected/applied/rolled_back/failed
// 消费方: asset/reconciliation/fix-suggestion
export const FIX_STATUS_OPTIONS: StatusOption[] = [
  { label: "待处理", value: "pending" },
  { label: "已接受", value: "accepted" },
  { label: "已拒绝", value: "rejected" },
  { label: "已应用", value: "applied" },
  { label: "已回滚", value: "rolled_back" },
  { label: "失败", value: "failed" },
];

// rolled_back color: "orange" — Phase 106 FEMAP-01 unification (index.tsx was correct, Drawer had "magenta")
export const FIX_STATUS_TAG_CONFIG: Record<string, { text: string; color: string }> = {
  pending:     { text: "待处理",  color: "gold" },
  accepted:   { text: "已接受",  color: "blue" },
  rejected:   { text: "已拒绝",  color: "default" },
  applied:    { text: "已应用",  color: "green" },
  rolled_back: { text: "已回滚", color: "orange" },
  failed:     { text: "失败",    color: "red" },
};
```

Note: FIX_STATUS_TAG_CONFIG uses `string` key (FixStatus is a string union), not `number`, so the type is `Record<string, { text: string; color: string }>` to match the FixStatus string keys.

**Block B — MAC history status (0=normal/1=stopped, same semantics as NORMAL_STOP):**

```typescript
// 对齐 MACHistoryRecord.status: 0=正常, 1=停用 (与 NORMAL_STOP 语义相同)
// 消费方: network/mac/history
// 注意: MAC 历史页面使用字面颜色 "green"/"red"（非 Ant Design 语义色），保持一致
export const MAC_HISTORY_STATUS_OPTIONS: StatusOption[] = [
  { label: "正常", value: 0 },
  { label: "停用", value: 1 },
];

export const MAC_HISTORY_STATUS_TAG_CONFIG: StatusTagConfig = {
  0: { text: "正常", color: "green" },
  1: { text: "停用", color: "red" },
};
```

**Block C — MAC type (dynamic/static/secure → blue/green/orange):**

```typescript
// 对齐 MACRecord.macType: dynamic/static/secure
// 消费方: network/mac
export const MAC_TYPE_OPTIONS: StatusOption[] = [
  { label: "动态", value: "dynamic" },
  { label: "静态", value: "static" },
  { label: "安全", value: "secure" },
];

export const MAC_TYPE_TAG_CONFIG: Record<string, { text: string; color: string }> = {
  dynamic: { text: "动态", color: "blue" },
  static:  { text: "静态", color: "green" },
  secure:  { text: "安全", color: "orange" },
};
```

## Step 2 — Migrate `fix-suggestion/index.tsx`

Read `src/pages/asset/reconciliation/fix-suggestion/index.tsx`.

**Add import:**
```typescript
import { FIX_STATUS_TAG_CONFIG } from "@/constants/status";
```

**Remove the local `fixStatusColor` and `fixStatusLabel` objects** (lines 68–75 and 77–84 respectively):
```typescript
// DELETE these two objects:
// Lines 68-75:
const fixStatusColor: Record<FixStatus, string> = { ... };
// Lines 77-84:
const fixStatusLabel: Record<FixStatus, string> = { ... };
```

**Update the status column render** (line 382):
- Change: `render: (v: FixStatus) => <Tag color={fixStatusColor[v]}>{fixStatusLabel[v]}</Tag>`
- To: `render: (v: FixStatus) => <Tag color={FIX_STATUS_TAG_CONFIG[v]?.color ?? "default"}>{FIX_STATUS_TAG_CONFIG[v]?.text ?? v}</Tag>`

**Update the action column fallback tag** (line 449):
- Change: `return <Tag>{fixStatusLabel[record.fixStatus]}</Tag>`
- To: `return <Tag>{FIX_STATUS_TAG_CONFIG[record.fixStatus]?.text ?? record.fixStatus}</Tag>`

**Update the Select options** (line 509):
- Change: `options={Object.entries(fixStatusLabel).map(([value, label]) => ({ value, label }))}`
- To: `options={FIX_STATUS_OPTIONS}`

## Step 3 — Migrate `FixSuggestionDetailDrawer.tsx`

Read `src/pages/asset/reconciliation/fix-suggestion/components/FixSuggestionDetailDrawer.tsx`.

**Add import:**
```typescript
import { FIX_STATUS_TAG_CONFIG } from "@/constants/status";
```

**Remove the local `fixStatusColor` and `fixStatusLabel` objects** (lines 22–38 of the original file).

**Fix the color bug:** The original `rolled_back: "magenta"` becomes `"orange"` via the centralized constant.

**Update all usages in the history timeline** (lines 246, 250–251):
- Change: `color: fixStatusColor[h.fixStatus]` → `color: FIX_STATUS_TAG_CONFIG[h.fixStatus]?.color ?? "default"`
- Change: `<Tag color={fixStatusColor[h.fixStatus]}> {fixStatusLabel[h.fixStatus]} </Tag>` → `<Tag color={FIX_STATUS_TAG_CONFIG[h.fixStatus]?.color ?? "default"}> {FIX_STATUS_TAG_CONFIG[h.fixStatus]?.text ?? h.fixStatus} </Tag>`

</tasks>

<success_criteria>
- `src/constants/status.ts` exports 6 new constants: FIX_STATUS_OPTIONS, FIX_STATUS_TAG_CONFIG, MAC_HISTORY_STATUS_OPTIONS, MAC_HISTORY_STATUS_TAG_CONFIG, MAC_TYPE_OPTIONS, MAC_TYPE_TAG_CONFIG
- Both `fix-suggestion/index.tsx` and `FixSuggestionDetailDrawer.tsx` import from `@/constants/status` — no local fixStatusColor/fixStatusLabel copies remain
- `rolled_back` maps to "orange" in both files (was "magenta" in Drawer, now unified)
- All TypeScript compiles with no new errors
- All export signatures (component props, API response shapes) remain identical
</success_criteria>

<output>
Create SUMMARY.md on completion
</output>
