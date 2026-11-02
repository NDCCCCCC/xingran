---
id: 260612-d2x
slug: ip
description: 工位管理页面子表格设备信息添加字段ip地址
status: complete
created: 2026-06-12
completed: 2026-06-12
commits:
  - dcde5f0 feat(workstation-device): add ip_address column and model field
  - 31e916a feat(workstation-device): add ipAddress to service DTOs and persistence
  - 8ad0f4a feat(workstation-device-ui): display and edit ipAddress in subtable
---

# Quick Task 260612-d2x: 工位管理页面子表格设备信息添加字段ip地址

## Summary

为工位管理页面"工位设备关联"子表格添加 IP 地址字段 `ipAddress`，覆盖数据库列、后端模型、Service DTO 与持久化逻辑、前端类型、表格列、Modal 表单、编辑/添加/设为主设备三处数据流。

## Commits

| Commit | Subject |
|--------|---------|
| `dcde5f0` | feat(workstation-device): add ip_address column and model field |
| `31e916a` | feat(workstation-device): add ipAddress to service DTOs and persistence |
| `8ad0f4a` | feat(workstation-device-ui): display and edit ipAddress in subtable |

## Changes

### Backend

| File | Change |
|------|--------|
| `internal/models/workstation_device.go` | 新增 `IPAddress *string` 字段（gorm size 64 + 索引） |
| `internal/core/db/migrations/150_add_workstation_device_ip_address.sql` | 新建 SQL：ALTER TABLE 添加 ip_address 列 + 索引 |
| `internal/core/db/migrations/migration_150_add_workstation_device_ip_address.go` | 新建 Go 迁移函数（含 SQL 文件 + inline 兜底） |
| `internal/core/db/database.go` | 注册 `Migrate150AddWorkstationDeviceIPAddress` |
| `internal/services/operations/workstation_device_service.go` | `AddDeviceRequest` / `UpdateDeviceRequest` / `SetPrimaryAndSaveRequest` 新增 `IPAddress *string`；Add/Update/SetPrimaryAndSave 三处实际写入 DB |

### Frontend

| File | Change |
|------|--------|
| `xingran-react-frontend/src/types/operations.ts` | `WorkstationDevice` 与 `DeviceFormData` 新增 `ipAddress?: string` |
| `xingran-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx` | 表格新增"IP地址"列（MAC 后）；handleEdit 加载 ipAddress；handleModalOk 提交时携带；handleSetPrimary 同步 AD/资产设备时携带；Modal 表单新增 IP 输入项（带 IPv4 pattern 校验） |

## Verification

- `go build ./internal/models/ ./internal/core/db/ ./internal/services/operations/` — 通过
- `cd xingran-react-frontend && npx tsc --noEmit -p .` — 通过
- 数据库：迁移 150 在启动时自动执行（首次启动或现存数据库需执行新迁移）

## Behavior

- **手动添加设备**: 序列号失焦后自动匹配资产信息时**不**自动填充 IP（IP 由用户手动输入，避免误填）
- **编辑设备**: IP 字段正确回显
- **AD/资产设备设为主设备**: 若原数据含 IP，会一并保存到数据库
- **表格展示**: 三个来源（手动 / AD / 资产）均新增"IP地址"列展示
- **校验**: 提交时若 IP 不符合 IPv4 格式（x.x.x.x）则提示

## Out of Scope (留待后续)

- 资产表 `ops_asset` 同步 IP 字段（资产系统暂未提供）
- AD 设备列展示 IP（AD 数据由前端从 `ADDeviceMatch.ipAddress` 实时获取，本任务已在 type 中支持，UI 列展示待后续按需启用）
- 自动从 MAC/序列号推导 IP

## Status

✅ Complete
