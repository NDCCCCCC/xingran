---
slug: asset-export-404-error
status: resolved
trigger: 资产导出功能404错误：POST /api/v1/ops/asset/excel/export 返回404
created: 2026-06-08
updated: 2026-06-08
session_type: bug
---

# Debug Session: Asset Export 404 Error

## Symptoms

### Expected Behavior
点击"导出"按钮应该导出资产数据到Excel文件

### Actual Behavior
点击"导出"按钮返回404错误，没有导出文件

### Error Messages
```
POST http://127.0.0.1:4000/api/v1/ops/asset/excel/export 404 (Not Found)
文件位置：opsApi.ts:551
```

### Timeline
新功能，刚实现的资产管理页面

### Reproduction Steps
1. 打开资产管理页面
2. 点击"导出"按钮
3. 预期：下载Excel文件
4. 实际：返回404错误

## Current Focus

hypothesis: Confirmed - incorrect URL paths in assetApi.excel
next_action: Fix applied
test: Verify export button triggers correct endpoint
expecting: POST /api/v1/ops/asset/export returns Excel file
reasoning_checkpoint: Root cause identified and fix applied
tdd_checkpoint:

## Evidence

- 2026-06-08: Backend router registers asset Excel routes via `operations.SetupExcelRouter(assets, "asset", core)` which adds `/import`, `/export`, `/template` directly to the `/ops/asset` group, producing endpoints: `/api/v1/ops/asset/export`, `/api/v1/ops/asset/import`, `/api/v1/ops/asset/template`
- 2026-06-08: Frontend `assetApi.excel` in opsApi.ts used paths with extra `/excel/` segment: `/ops/asset/excel/export`, `/ops/asset/excel/import`, `/ops/asset/excel/template` -- these do not match any backend route
- 2026-06-08: Other entities (building, floor, workstation, etc.) use the generic `excelApi` which correctly constructs paths as `/api/v1/ops/${entityType}/export` without the `/excel/` segment
- 2026-06-08: The `ExcelImport` component (used by asset page) already builds correct URLs: `/api/v1/ops/${entityType}/template` and `/api/v1/ops/${entityType}/import` -- import and template were already working

## Eliminated

- Backend route registration issue: routes are correctly registered
- Missing excel_config entry: "asset" config exists with full column mappings
- Permission middleware blocking: 404 indicates route not found, not 403

## Resolution

root_cause: Frontend `assetApi.excel` in opsApi.ts hardcoded incorrect URL paths with an extra `/excel/` segment (e.g., `/ops/asset/excel/export`) that does not exist on the backend. The backend `SetupExcelRouter` registers routes directly on the entity group without an `/excel/` sub-path, producing `/ops/asset/export`.
fix: Removed the erroneous `/excel/` segment from all three `assetApi.excel` paths in `xingran-react-frontend/src/lib/opsApi.ts`: template path changed from `/ops/asset/excel/template` to `/ops/asset/template`, import from `/ops/asset/excel/import` to `/ops/asset/import`, export from `/api/v1/ops/asset/excel/export` to `/api/v1/ops/asset/export`.
verification: Path now matches the backend route registered by `SetupExcelRouter(assets, "asset", core)` at `/ops/asset/export`.
files_changed: xingran-react-frontend/src/lib/opsApi.ts
