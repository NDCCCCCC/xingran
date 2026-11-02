---
slug: workstation-primary-device-serial
quick_id: 260626-lbd
status: in-progress
created: 2026-06-26
---

# Quick Task: 工位管理主表添加主设备序列号列

## 描述
工位管理页面主表（工位列表）添加"主设备序列号"列。当工位设置了主设备（`ops_workstation_device.is_primary = true`）后，在主表该列显示主设备的序列号；未设置则显示 `-`。

## 背景
- 工位设备表 `ops_workstation_device` 有 `device_serial` + `is_primary` 字段
- 一个工位可关联多台设备，`is_primary=true` 的是主设备
- Excel 导出已有现成的主设备子查询模式（`excel_query_builder.go:151`），复用其 `ORDER BY (CASE WHEN is_primary=true THEN 0 ELSE 1 END), priority DESC, created_at ASC LIMIT 1` 逻辑
- 工位列表 SELECT（`workstation_service.go:16 workstationJoinSelect`）目前无主设备字段

## 改动（4 处）

### 后端
1. `internal/models/workstation.go` — Workstation struct 加计算字段
   ```go
   PrimaryDeviceSerial *string `gorm:"-:migration" json:"primaryDeviceSerial,omitempty"` // 主设备序列号(子查询,非表列)
   ```
   用 `gorm:"-:migration"` 避免 AutoMigrate 建列，但保留 Select scan 能力。

2. `internal/services/operations/workstation_service.go:16` — workstationJoinSelect 末尾加子查询：
   ```sql
   , (SELECT device_serial FROM ops_workstation_device
      WHERE workstation_id = sys_workstation.id::text AND deleted_at IS NULL
      ORDER BY (CASE WHEN is_primary = true THEN 0 ELSE 1 END), priority DESC, created_at ASC
      LIMIT 1) as primary_device_serial
   ```
   同时影响 List 和 GetByID（共用 select），一致性 OK。

### 前端
3. `xingran-react-frontend/src/types/operations.ts` — WorkstationOps 加 `primaryDeviceSerial?: string`

4. `xingran-react-frontend/src/pages/operations/workstations/columns.tsx` — 在"所属用户"列后加"主设备序列号"列，`render: (text) => text || "-"`

## 验证
- `go build ./...`
- `npm run type-check`
- 浏览器实测（chrome-devtools）：工位列表出现"主设备序列号"列，有主设备的工位显示序列号，无则 `-`

## 不在范围
- 不改"域控设备"展开行逻辑（那是另一个已诊断的 managed_by 关联问题）
- 不改主设备设置/同步逻辑
- 列不可排序（子查询排序代价高，且无业务需求）
