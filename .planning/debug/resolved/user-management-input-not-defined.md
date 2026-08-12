---
slug: user-management-input-not-defined
status: resolved
trigger: "Uncaught ReferenceError: Input is not defined at UserManagement (index.tsx:497:20)"
created: 2026-06-17
updated: 2026-06-17
---

## Current Focus
hypothesis: confirmed
test: applied
expecting: fulfilled
next_action: archive to resolved/ and append knowledge base

## Symptoms
expected: UserManagement page renders without runtime errors
actual: "Uncaught ReferenceError: Input is not defined" thrown when React tries to render the component
errors: ReferenceError: Input is not defined at index.tsx:497:20
reproduction: navigate to /system/user in the running dev server
started: unknown

## Eliminated
(none — root cause found on first hypothesis)

## Evidence
- timestamp: 2026-06-17
  checked: import block at lines 8-24 of user/index.tsx
  found: antd destructure contains Table, Button, Space, Modal, Form, Select, TreeSelect, Tag, Avatar, Card, Row, Col, Statistic, Layout, Alert — NO Input
  implication: missing import
- timestamp: 2026-06-17
  checked: grep for `\bInput\b` in the same file
  found: 10 JSX usages (lines 497, 500, 606, 614, 622, 625, 628, 631, 690, 708); also `Input.Password`
  implication: 10 broken references — one fix (add to import) resolves all
- timestamp: 2026-06-17
  checked: hooks/useUserData.ts and hooks/useUserModals.ts
  found: no `Input` usage
  implication: bug is isolated to index.tsx
- timestamp: 2026-06-17
  checked: full antd component audit (Modal, Select, Form.Item, Tag, Avatar, etc.)
  found: all other antd components used are properly imported
  implication: only Input was missing
- timestamp: 2026-06-17
  checked: `npx tsc --noEmit`
  found: 0 errors
  implication: type-check passes
- timestamp: 2026-06-17
  checked: `npx eslint src/pages/system/user/index.tsx`
  found: 5 errors + 2 warnings — ALL pre-existing (unused useState, unused TreeSelect, unused convertDeptTreeData, exhaustive-deps on setSelectedDeptId, etc.); none related to Input or my change
  implication: my change introduces no new lint issues
- timestamp: 2026-06-17
  checked: `npm run build`
  found: 0 errors in src/pages/system/user/index.tsx; pre-existing errors in src/pages/vdi/VirtualMachineList/index.tsx (vmApi/vdiServerApi/Input/Modal references) that are out of scope
  implication: my change is clean; the build still fails on the unrelated VDI page

## Resolution
root_cause: `<Input>` (and `<Input.Password>`) used in JSX at 10 sites in src/pages/system/user/index.tsx but `Input` was never added to the antd destructure import (lines 8-24). React 19 throws ReferenceError on the first render attempt.
fix: added `Input,` to the antd import destructure between `Form,` and `Select,` to maintain alphabetical order
verification:
  - type-check: PASS (0 errors)
  - lint: no new errors (5 pre-existing errors remain, all unrelated to Input)
  - build: PASS for user file (pre-existing build failures in VirtualMachineList are out of scope)
files_changed:
  - xingran-react-frontend/src/pages/system/user/index.tsx
