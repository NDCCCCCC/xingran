---
phase: quick
plan: 260713-df0
status: complete
subsystem: operations
tags: [excel-import, workstation, dept-mapping, ad-asset-merge]
---

# Quick 260713-df0 工位导入扩展 - 部门/用户/主设备序列号 Summary

## One-liner

工位 Excel 导入新增"部门代码"+"主设备序列号"两列；跨表主设备同步共用既有 AD/Asset 合并逻辑；新增部门映射表下载端点与前端按钮。

## Tasks Completed

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1 | ExcelConfig + WorkstationDeviceService + ExcelService post-import hook | `41c4ff05` | `internal/services/operations/excel_config.go`, `workstation_device_service.go`, `excel_service.go`, `internal/api/v1/operations/excel_handler.go` |
| 2 | Department mapping template download endpoint | `25ac6ebd` | `internal/api/v1/operations/excel_handler.go`, `internal/api/router.go` |
| 3 | Frontend mapping download button + API | `ad0ac2f9` | `xingran-react-frontend/src/lib/opsApi.ts`, `pages/operations/workstations/index.tsx` |

## Decisions

### D1: `mergeBySerial` 作为 unexported helper,无 adAssetMergeResult 接口暴露
- **Reasoning**: 复用而非重写(PLAN 强制约束)。两个公开方法 (`SetPrimaryAndSave` / `SetPrimaryAndSaveBySerial`) 共享同一段合并代码,避免 quick 260614-dpz 已有实现被分裂。
- **Tradeoff**: helper 与返回值结构都不导出,外部包无法复用;但本场景只有本服务使用,可控。

### D2: `ExcelService` 增加 `deviceService` 字段 + `WithDeviceService` setter 而非构造函数参数
- **Reasoning**: `getExcelService` 已经很长,新增 1 个可选参数会破坏向后兼容;setter 模式兼容所有现存调用。
- **Tradeoff**: 可选依赖通过 setter 注入,可能在测试场景被遗忘;已有 `s.deviceService == nil` 守卫保护。

### D3: post-import hook 走 `service` 层(非 handler 层)
- **Reasoning**: PLAN 决策点明确"优先尝试 service 层方案"。`WorkstationDeviceService` 与 `ExcelService` 同在 `internal/services/operations` 包内,无 import cycle。
- **Outcome**: 选 service 层方案,handler 仅构造时注入 service 实例。

### D4: 部门映射表路由放 `workstations` group 而非独立 `dept` group
- **Reasoning**: PLAN 约束"避免路由冲突 (项目记忆 xingran-excel-import-route-conflict)"。`workstations` 已注册 ops:workstation:* 权限,与 /import /template 一致;独立 dept group 容易与未来模块冲突。
- **Tradeoff**: 文件归属 "工位管理",与 operlog module name "部门管理" 不一致 — operlog 端点使用 `"部门管理"` 而非 `excelEntityModuleNames["workstation"]`,因为这是 dept 导出而非 workstation 模板。

### D5: 不修改 `ExcelImport.tsx` props 接口
- **Reasoning**: PLAN Out-of-Scope 明确禁止。`下载部门映射表` 按钮放在页面 toolbar 而非 import modal 内部,降低耦合。

### D6: `SetPrimaryAndSave` 移除 deviceID 内部依赖,改为兼容警告日志
- **Reasoning**: 新代码以 `req.DeviceSerial` 为查询键,`deviceID` 仅作前端 set-primary-and-save 路由入参(临时 id)。保留入参兼容所有现存调用,避免破坏既有 handler。
- **Tradeoff**: deviceID 入参被 warn 提示忽略,前端代码可后续清理。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Initial Edit accidentally truncated workstation block in excel_config.go**
- **Found during:** Task 1, immediately after first Edit attempt
- **Issue:** Used `replace_all=false` with `old_string` missing the closing brace, accidentally inserted `,\n\t"asset": {` then second Edit didn't fully restore
- **Fix:** Verified current file content with Read after each edit; ran `go build` to detect corruption early
- **Files modified:** `internal/services/operations/excel_config.go`
- **Commit:** `41c4ff05` (rolled into Task 1 commit)

## Files Touched

### Created
- None (purely additive changes)

### Modified

**Backend (4 files):**
- `internal/services/operations/excel_config.go` — WorkstationExcelConfig.Columns +2 列 (deptCode, deviceSerial)
- `internal/services/operations/workstation_device_service.go` — 接口 +1 方法 (SetPrimaryAndSaveBySerial), unexported mergeBySerial helper, adAssetMergeResult 结构体
- `internal/services/operations/excel_service.go` — ExcelService +deviceService 字段 +WithDeviceService setter, postImportWorkstationPrimaryDevice hook
- `internal/api/v1/operations/excel_handler.go` — getExcelService 注入 deviceService, DownloadDeptMappingTemplate handler
- `internal/api/router.go` — 注册 GET /ops/workstation/dept-mapping-template

**Frontend (2 files):**
- `xingran-react-frontend/src/lib/opsApi.ts` — 新增 deptApi.exportMapping()
- `xingran-react-frontend/src/pages/operations/workstations/index.tsx` — 下载部门映射表按钮

## Verification

- `go build ./...` — PASS
- `go vet ./internal/services/operations/... ./internal/api/v1/operations/...` — no warnings
- `npm run type-check` (frontend) — PASS
- 3 atomic commits, one per task
- `mergeBySerial` helper 共用 (both SetPrimaryAndSave + SetPrimaryAndSaveBySerial call s.mergeBySerial)
- ExcelConfig.WorkstationExcelConfig 含 `deptCode` + `deviceSerial` 两列
- `sys_workstation` 表未改 (`PrimaryDeviceSerial` 仍是虚拟字段 `gorm:"->;-:migration"`)
- `ExcelImport.tsx` props 未改
- 既有 `SetPrimaryAndSave` 签名兼容 (deviceID 入参保留,内部忽略)

## Out-of-Scope Verification

- [x] 不改 `sys_workstation` 表结构
- [x] 不改 `ExcelImport.tsx` props 接口
- [x] 不修改既有 `SetPrimaryAndSave` 签名
- [x] 不做前端新组件/新路由
- [x] 不写 DB trigger (走 service 层 + BeforeCreate hook)
- [x] 不重构既有 `deptName`/`userName` 逻辑 (保留按名称匹配作为 deptCode 缺失时的回退)
- [x] operlog 只记映射表下载一次;post-import 主设备同步不重复记 operlog
- [x] 不开放映射表端点给非工位模块