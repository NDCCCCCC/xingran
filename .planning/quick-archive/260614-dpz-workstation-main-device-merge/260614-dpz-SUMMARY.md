---
status: complete
---

# 260614-dpz 工位主设备 - 资产+域控数据合并同步 — 执行总结

**Mode**: quick
**Date**: 2026-06-14
**Slug**: 260614-dpz-workstation-main-device-merge

## Commits

| Task | Scope | Commit |
|------|-------|--------|
| Task 1 | 服务端 `SetPrimaryAndSave` 合并逻辑 | `dcfd552` |
| Task 2 | Handler Swagger 注释 + 前端 opsApi 注释 | `093b730` |

## What Changed

### Task 1 — `internal/services/operations/workstation_device_service.go`
- 接口 `WorkstationDeviceService.SetPrimaryAndSave` 注释补全合并语义说明（字段优先级、失败降级）。
- 方法体内新增"以 device_serial 为键"的合并流程：
  1. 调用现有 `GetADDevices(ctx, req.WorkstationID)` 与 `GetAssetDevices(ctx, req.WorkstationID)` 实时拉取两侧数据。
  2. 构造 `adBySN` / `assetBySN` 两个 map，命中分别记录 `adHit` / `assetHit`。
  3. 合并字段优先级：
     - `deviceName`: AD.DeviceName > req.DeviceName
     - `deviceModel`: asset.DeviceModelName > req.DeviceModel
     - `deviceType`: asset.DeviceTypeName > req.DeviceType
     - `macAddress`: AD.MACAddress > asset.MAC1 > req.MACAddress
     - `ipAddress`: AD.IPAddress > req.IPAddress
     - `responsibleUser`: asset.NowUserName > req.ResponsibleUser
     - `assetID` / `adComputerID`: 命中时填充
  4. 任一来源查询失败时 `logger.Warnf` 后降级继续，不阻塞保存。
  5. 事务内步骤调整为：清理旧 `device_source IN ('ad','asset')` 记录 -> 取消旧主设备 -> 写入合并后的 manual 记录 (IsPrimary=true)，新增 `ADComputerID` 字段填充。
- 方法顶部新增 doc 注释说明合并策略与字段优先级。

### Task 2 — `internal/api/v1/operations/workstation_device_handler.go` + `xingran-react-frontend/src/lib/opsApi.ts`
- Handler 的 `@Description` Swagger 注释扩展为完整描述合并语义（字段优先级、清理旧 AD/资产来源、最终写入 manual 记录）。
- 前端 `workstationDeviceApi.setPrimaryAndSave` 上方补一行注释，说明后端在保存前会自动合并两侧实时数据。
- 未修改 API 路径、未修改 `SetPrimaryAndSaveRequest` 结构、不影响 opsApi 调用方。

## Verification

- `go build ./...` 通过 (exit 0)
- `go vet ./internal/services/operations/...` 通过 (exit 0)
- 接口签名未变化，向后兼容现有 handler / 前端调用

## Out of Scope (as planned)

- `SetPrimaryAndSaveRequest` 结构保持不变
- opsApi URL 路径未变，无新增前端请求
- `SyncFromAD` / `SyncFromAsset` 单独同步逻辑保留
- `priority` 字段策略未调整

## Self-Check

- [x] `go build ./...` 通过
- [x] `go vet ./internal/services/operations/...` 通过
- [x] Task 1 完成, Task 2 完成
- [x] 合并策略在四种组合下都按预期兜底 (AD+Asset 都命中 / 仅 AD / 仅 Asset / 都不命中)
