---
quick: 260613-vp5-workstation-expand-fix
title: Workstation expand cell — full-area click + stopPropagation
type: execute
status: manual verification pending
---

# Quick 260613-vp5: Workstation expand cell — full-area click + stopPropagation

## One-liner

Restore "click anywhere in the 查看设备 / 收起设备 cell to toggle expand" behavior on the workstation management page and stop the inner button from bubbling so a single click toggles exactly once.

## Tasks

### Task 1: Apply JSX change to workstations/index.tsx — DONE

- File: `xingran-react-frontend/src/pages/operations/workstations/index.tsx`
- Added `import type React from 'react';` after the existing react import block.
- Replaced the bare `<Button>` inside `expandable.expandIcon` with a wrapping `<div role="button" tabIndex={0}>` container.
  - Container `onClick` calls `onExpand(record, e)` so any whitespace click in the cell toggles.
  - Container `onKeyDown` handles `Enter` / `Space` for keyboard accessibility.
  - Inner `<Button>` `onClick` calls `e.stopPropagation()` first, then `onExpand(record, e)`, preventing the click from bubbling to the container and toggling twice.
- Left `expandedRowRender` and `rowExpandable` untouched.

### Task 2: Manual UI verification — PENDING (user to run)

Per the executor constraints, the dev server was NOT started and no browser interaction was performed. The user must complete the checklist below before this quick task can be considered fully verified.

**Steps the user should run locally:**

1. Start the frontend dev server:
   ```bash
   cd xingran-react-frontend
   npm run dev
   ```
2. Open the workstation management page at `/operations/workstations`.
3. Run through these four scenarios:
   - **Scenario A — Button body click toggles exactly once.**
     Click "查看设备" → sub-table expands; click again → collapses. Confirm there is no "click once to expand, click again collapses twice" defect.
   - **Scenario B — Click the cell whitespace next to the button.**
     Click the empty area to the left or right of "查看设备" (do not hit the button itself) → row should expand/collapse just like a button click.
   - **Scenario C — Click on an unrelated column.**
     Click the "工位名称" cell or any other column → row should NOT expand/collapse.
   - **Scenario D (optional) — Keyboard accessibility.**
     Tab to focus the 查看设备 / 收起设备 cell, press Enter or Space → row should expand/collapse.
4. If any scenario fails, fall back per the plan's "风险与回滚" section (e.g. switch container to `onMouseDown`, set `pointer-events: none` on the inner button, or `git revert` this commit) and re-test.

**Completion criterion:** all four scenarios pass — clicking anywhere in the 查看设备 / 收起设备 cell toggles expand/collapse, and clicking the button body itself toggles exactly once per click.

## Verification

- TypeScript: `cd xingran-react-frontend && npm run type-check` → passed (no output, exit 0).
- No new dependencies added.
- Diff is scoped to a single file: `xingran-react-frontend/src/pages/operations/workstations/index.tsx` (69 insertions / 61 deletions — the higher insert count reflects the new wrapping `<div>` block; net behavior change is small).

## Deviations from Plan

None — plan executed as written. Both task commits were atomic and code-only; no doc artifacts were committed (the orchestrator handles those).

## Files Changed

- `xingran-react-frontend/src/pages/operations/workstations/index.tsx` (modified)

## Commits

- `aa46631` — `fix(quick-260613-vp5): make expand cell fully clickable and stop button bubble`

## Manual Verification Status

**Pending** — user must run the dev server and walk through scenarios A–D before this quick task is considered fully verified.