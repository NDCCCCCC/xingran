---
phase: 84-p1-70
plan: 01a
wave: 1
status: complete
completed: 2026-08-27
tasks_commits: c76946a (task1-2), a0c7def (task3)
---

## Plan 84-01a Complete

**components/shared 21 文件 892 stmts 覆盖率 22.42% → floor bump 0.0→21.9**

### Tasks
- **Task 1** (`c76946a`): 共享原子组件 8-family 39-tests(D-11/D-12) — atomicComponents/batchOperations/fileUpload/globalSearch/imageGallery/networkExport/columnConfigModal + renderWithProviders/createApiMock harness 被 6 tests import
- **Task 2** (`c76946a`): FPE 5-sub(panZoom/hooks/types/constants) + Excel ops + shared 测试 30 tests PASS — 69 total in shared/__tests__
- **Task 3** (`a0c7def`): floor bump shared→21.9 / baseline ratchet 84-01a 行 / 0 FAIL / 930 tests PASS

### Key decisions
- `@testing-library/user-event` 未安装，统一使用 `fireEvent` + `waitFor`
- BatchExportModal 无 footer cancel 按钮，onCancel 由关闭 X 触发
- GlobalSearch Ctrl+K window listener 在 jsdom 无法测试，改为 trigger click
- DepartmentTreeSelect placeholder 在 `.ant-select-placeholder` div 而非 input attr
- NetworkExport 导出菜单需等待 Dropdown 动画，用 `waitFor` 避免 flaky
- ColumnConfigModal dnd-kit 复杂依赖保留存量测试
- FPE `.tsx` Three.js/WebGL 依赖已有存量覆盖

### Verification
- gate: GLOBAL 18.92% >= 3.80% / 47 PASS / 0 FAIL
- vitest: 930 tests PASS / 87 files / 0 fail
- floor: shared 22.42% >= 21.9%
- baseline ratchet: 84-01a 行追加

### Artifacts
- `xingran-react-frontend/src/components/shared/__tests__/` 7 test files
- `.coverage-fe-floors`: shared 0.0→21.9
- `.planning/frontend-coverage-baseline.md`: 84-01a ratchet row appended
