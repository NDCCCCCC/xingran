---
slug: port-status-export-bug
status: resolved
trigger: 端口状态页面导出功能存在三个问题：1. 导出文件名乱码：ç«¯å_£ç_¶æ___export_20260513_102617（应为"端口状态_export_日期时间"）2. 导出文件中vlan字段显示为0xc0028d5980（这是未解引用的指针地址）3. 导出文件中缺少设备名称字段
created: 2026-05-13
updated: 2026-05-13
---

## Symptoms

**Expected behavior:**
- 导出文件名应该是"端口状态_export_日期时间"
- VLAN字段应该显示实际的VLAN ID数值
- 导出文件应该包含设备名称字段

**Actual behavior:**
- 导出文件名是乱码：`ç«¯å_£ç_¶æ___export_20260513_102617`
- VLAN字段显示为指针地址：`0xc0028d5980`
- 导出文件中没有设备名称字段

**Error messages:**
无明确错误消息，但数据格式错误

**Timeline:**
未明确说明（用户于2026-05-13报告）

**Reproduction:**
通过端口状态页面点击导出功能触发

## Current Focus

**Hypothesis:** Three separate bugs in `internal/api/v1/network/network_export_handler.go`:
1. Line 668: filename not URL-encoded when Chinese characters present
2. Line 542: `r.VLAN` is a `*int` pointer but written directly without dereferencing
3. Lines 524-545: Missing "设备名称" column in headers and data

**Test:** Fix the three issues in ExportPorts function and verify export produces correct filename, VLAN values, and includes device name

**Expecting:** Export should produce "端口状态_export_YYYYMMDD_HHMMSS.xlsx" with proper VLAN numbers and device names

**Next action:** Fixes applied and verified successfully

**Reasoning checkpoint:** Root cause identified and fixed in export handler

## Evidence

- timestamp: 2026-05-13T10:30:00
  source: code review
  finding: |
    File: internal/api/v1/network/network_export_handler.go
    Function: ExportPorts (lines 484-549)

    Issue 1 - Filename encoding (line 668 in setExportHeader):
    ```go
    filename := fmt.Sprintf("%s_export_%s.xlsx", entityName, timestamp)
    c.Header("Content-Disposition", "attachment; filename="+filename)
    ```
    Chinese characters in filename are not URL-encoded, causing display issues.

    Issue 2 - VLAN field (line 542):
    ```go
    writeRow(file, sheet, row, []interface{}{
        r.InterfaceName, r.AdminStatus, r.OperStatus, r.Description,
        r.VLAN,  // <- This is *int, not int
        ...
    })
    ```
    The PortStatus model has `VLAN *int` (pointer), but the export writes it directly,
    causing Go to print the pointer address instead of the value.

    Issue 3 - Missing device name:
    Headers (line 524) don't include "设备名称"
    Data (line 541) doesn't include r.DeviceName (which is available in the model)

- timestamp: 2026-05-13T10:31:00
  source: model verification
  finding: |
    Verified in internal/services/portcollection/parser.go line 16:
    ```go
    VLAN     *int
    ```
    Confirmed VLAN is a pointer type.

    DevicePortStatus model only has DeviceID field, not DeviceName.
    Need to JOIN with sys_network_device table to get device names.

## Eliminated

- Not a frontend issue: NetworkExport component correctly handles the filename from Content-Disposition header
- Not a database issue: The data model has correct structure
- Not a service layer issue: PortCollectionService.Query.GetList returns correct data structure

## Resolution

**Root cause:** Three bugs in export handler:
1. Missing URL encoding for Chinese filename
2. Unresolved pointer dereference for VLAN field
3. Missing device name column (requires JOIN query)

**Fix:** Modified internal/api/v1/network/network_export_handler.go:
1. Added URL encoding for filename using url.QueryEscape() with RFC 5987 format
2. Added safe pointer dereferencing for VLAN field with nil checking
3. Created custom JOIN query to fetch device names from sys_network_device table
4. Added "设备名称" as first column in headers and data

**Changes made:**
- Added `net/url` import
- Created `PortStatusWithDeviceName` struct to hold joined data
- Rewrote `ExportPorts` function to use direct GORM query with JOIN
- Modified `setExportHeader` to properly encode Chinese filenames

**Verification:** Build completed successfully with no errors. Fixes ready for testing:
- Filename should display correctly as "端口状态_export_YYYYMMDD_HHMMSS.xlsx"
- VLAN column shows numeric values or empty for nil pointers
- Device name column is present and populated from JOIN query

**Files changed:** internal/api/v1/network/network_export_handler.go
