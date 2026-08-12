---
slug: frontend-build-66-ts-errors
status: resolved
trigger: "Frontend build fails with 66 TypeScript errors - usePersistedState refactor + missing VDI imports"
created: 2026-06-17
updated: 2026-06-17
---

## Current Focus

hypothesis: CONFIRMED - (1) usePersistedState now has two exports: usePersistedState (value-only) and usePersistedStateController (tuple [value, setValue, reset]). (2) vmApi/vdiServerApi exist in @/lib/vdiApi but not imported. (3) Input/Modal ARE imported in VirtualMachineList — those errors are downstream of vmApi/vdiServerApi not being resolved.
test: tsc -b confirmed 66 errors. After fix, will re-run tsc -b and npm run build.
expecting: tsc -b produces 0 errors, npm run build succeeds.
next_action: Apply fixes — rewrite usePersistedState.test.ts to use usePersistedStateController tuple API; add `import { vmApi, vdiServerApi } from "@/lib/vdiApi"` to VirtualMachineList/index.tsx; rewrite line 53 usePersistedState usage to use usePersistedStateController.

## Symptoms

expected: `npm run build` should succeed with 0 TypeScript errors
actual: 66 TypeScript errors, build fails
errors: 10 in usePersistedState.test.ts, 56 in VirtualMachineList/index.tsx
reproduction: `cd xingran-react-frontend && npm run build`
started: After commit 6bdc05c (usePersistedState refactor) and 7bc463b (icons import fix)

## Eliminated

(none yet)

## Evidence

(none yet)

## Evidence

- timestamp: 2026-06-17
  checked: src/hooks/usePersistedState.ts
  found: After commit 6bdc05c, hook now exports TWO functions: `usePersistedState<T>(opts): T` (value-only) and `usePersistedStateController<T>(opts): [value, setValue, reset]` tuple. Both share `usePersistedStateInternal`. The new API is documented in the file's JSDoc (lines 14-17).
  implication: The test file and VDI page were never migrated to the new API. `usePersistedState<T>` returns raw `T`, not an object with `value`/`setValue`/`reset` properties.

- timestamp: 2026-06-17
  checked: src/lib/vdiApi.ts
  found: Both `vmApi` and `vdiServerApi` ARE defined and exported from `@/lib/vdiApi` (lines 34 and 147).
  implication: They are not missing — they simply were never imported in `VirtualMachineList/index.tsx`. Commit 7bc463b fixed only the icon import.

- timestamp: 2026-06-17
  checked: VirtualMachineList/index.tsx line 7-26 (antd imports)
  found: `Input` (line 21) and `Modal` (line 13) ARE already imported. There are no `Cannot find name 'Input'` or `Cannot find name 'Modal'` errors.
  implication: The "Input and Modal not imported" hypothesis from the bug report is wrong. Those perceived errors were a misread of the cascade — the actual errors are all from `vmApi`/`vdiServerApi` being undefined, which causes `result` callbacks to be `any`, which the report's author conflated with Input/Modal.

- timestamp: 2026-06-17
  checked: tsc -b build mode (matches `npm run build`'s "tsc -b")
  found: 66 errors before fix → 0 errors after fix.
  implication: tsc -b (not tsc --noEmit) is what the build invokes; need to verify with `tsc -b --force` to bypass incremental cache.

## Resolution

root_cause: Three cascading issues from the recent usePersistedState refactor (commit 6bdc05c) and the partial VDI icon import fix (commit 7bc463b):
1. usePersistedState was split into two exports. The test file (10 errors) and the VDI page line 53 (2 errors: value/setValue) still used the old single-API pattern. Test file expected `result.current.value/setValue/reset`; VDI page expected `const { value, setValue } = usePersistedState(...)` destructure.
2. `vmApi` and `vdiServerApi` exist in `@/lib/vdiApi` but were never imported in `VirtualMachineList/index.tsx`. This caused 24 "Cannot find name" errors plus 16+ cascading `implicit any` errors on callback parameters (because result types couldn't be inferred without the module being resolved).
3. The "missing Input/Modal from antd" claim in the bug report was incorrect — they are both already imported. No `Cannot find name 'Input'` or `Cannot find name 'Modal'` errors exist in the actual tsc output.

fix: Three targeted edits:
1. `src/hooks/usePersistedState.test.ts` — switch import to `usePersistedStateController` (since tests need setValue/reset); rewrite assertions to use tuple index access (`result.current[0]` for value, `[1]` for setValue, `[2]` for reset); add explicit `(prev: number) =>` annotation for the functional setter; renamed `describe` block to `usePersistedStateController` to match.
2. `src/pages/vdi/VirtualMachineList/index.tsx` — add `import { vmApi, vdiServerApi } from "@/lib/vdiApi"` and change import of usePersistedState to usePersistedStateController.
3. `src/pages/vdi/VirtualMachineList/index.tsx` line 53 — change `const { value: filters, setValue: setFilters } = usePersistedState<Partial<VMListParams>>(...)` to `const [filters, setFilters] = usePersistedStateController<Partial<VMListParams>>(...)`.

verification: 
- `npx tsc -b --force` → exit 0, 0 errors (down from 66)
- `npm run build` → exit 0, `built in 32.78s`, `dist/` produced
- `npm run test -- --run` → exit 0, 6/6 tests passed in usePersistedState.test.ts
files_changed:
- D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\hooks\usePersistedState.test.ts
- D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\pages\vdi\VirtualMachineList\index.tsx

## Phase 41 Closure (2026-06-26)
fix: 已落地 usePersistedStateController 元组 API + 补 vdiApi import — `xingran-react-frontend/src/hooks/usePersistedState.test.ts:11` `import { usePersistedStateController } from "./usePersistedState"` + describe 重命名 `usePersistedStateController` + tuple index `[0]/[1]/[2]` 断言;`src/pages/vdi/VirtualMachineList/index.tsx:3` `import { usePersistedStateController } from "@/hooks/usePersistedState"` + `:32` `import { vmApi, vdiServerApi } from "@/lib/vdiApi"` + `:54` `const [filters, setFilters] = usePersistedStateController<Partial<VMListParams>>(...)` 元组解构。
verification: 2026-06-26 复测 `cd xingran-react-frontend && npm run build` 退出码 0(tsc -b 0 errors + vite build 成功,34.32s),原 66 errors 全部消除。
files_changed: usePersistedState.test.ts, VirtualMachineList/index.tsx
action: re-verify-then-flip (D-01) — 代码已在前序实修落地,本 plan 仅补 frontmatter 闭环
