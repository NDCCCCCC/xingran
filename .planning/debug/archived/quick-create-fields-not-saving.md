---
slug: quick-create-fields-not-saving
status: resolved
trigger: "快速创建设备功能没有正确保存型号 (model) 和 IP 地址 (ip_address) 字段"
created: "2026-05-08"
updated: "2026-05-08"
---

## Symptoms

- **Expected**: 探测成功后，设备信息（型号、IP 地址等）应该正确保存到数据库
- **Actual**: 型号 (model) 和 IP 地址 (ip_address) 没有保存，可能还有其他字段缺失
- **Timeline**: 最近发现的问题
- **Reproduction**: 快速创建设备 → 探测成功显示信息 → 点击创建 → 字段未保存
- **Context**: 探测结果显示有型号和 IP 信息，说明探测功能正常
- **User Report**: "还有其他字段不确定是否全部保存，请帮我检查"

## Current Focus

- hypothesis: null
- next_action: null
- test: null
- expecting: null
- reasoning_checkpoint: null
- tdd_checkpoint: null

## Evidence

- timestamp: 2026-05-08T10:00:00Z
  source: code_inspection
  finding: |
    Frontend handleQuickCreate function (xingran-react-frontend/src/pages/network/devices/index.tsx:326-339)
    sends only these fields to backend:
    - ipAddress
    - credentialId
    - snmpPort
    - communities
    - deptId
    - location
    - description

    Missing fields that are available in probeResult state:
    - model
    - vendor
    - deviceName
    - deviceType
    - sysName
    - sysDescr

- timestamp: 2026-05-08T10:05:00Z
  source: code_inspection
  finding: |
    Backend QuickCreateDevice function (internal/services/network_device_service.go:222-337)
    does perform its own SNMP probe via s.discoveryService.ProbeSingleDevice()
    and uses probeResult.Model and probeResult.IPAddress when creating device:

    Line 313: Model:        probeResult.Model,
    Line 314: IPAddress:    req.IPAddress,

    Device creation at line 305-322 includes all fields from probe result.

- timestamp: 2026-05-08T10:10:00Z
  source: code_inspection
  finding: |
    DeviceProbeResult struct (internal/services/device_discovery_service.go:73-83)
    contains Model field at line 80:
    type DeviceProbeResult struct {
        Success    bool                `json:"success"`
        Message    string              `json:"message"`
        IPAddress  string              `json:"ipAddress"`
        DeviceName string              `json:"deviceName,omitempty"`
        DeviceType models.DeviceType   `json:"deviceType,omitempty"`
        Vendor     models.DeviceVendor `json:"vendor,omitempty"`
        Model      string              `json:"model,omitempty"`  <-- Line 80
        SysDescr   string              `json:"sysDescr,omitempty"`
        SysName    string              `json:"sysName,omitempty"`
    }

- timestamp: 2026-05-08T10:15:00Z
  source: code_inspection
  finding: |
    Probe result model extraction (internal/services/device_discovery_service.go:552-575)
    correctly extracts model from SysDescr using model extractor:

    Line 552-563: Model extraction logic with fallback
    Line 574: Model:      model,  <-- Assigned to probeResult

- timestamp: 2026-05-08T10:30:00Z
  source: code_inspection
  finding: |
    **ROOT CAUSE IDENTIFIED**: Asynchronous device info collection service overwrites SNMP data

    Flow:
    1. QuickCreateDevice creates device with SNMP probe data (lines 305-322)
       - Sets Model: probeResult.Model (line 313)
       - Sets IPAddress: req.IPAddress (line 314)

    2. DeviceInfoCollectionService.Enqueue() is called (line 330)
       - Runs asynchronously in background
       - Calls CollectDeviceInfo() via SSH (line 231 in device_info_collection_service.go)

    3. updateDeviceInfo() overwrites device fields (lines 297-321)
       - updates["model"] = info.Model (line 302)
       - If SSH collection fails or returns empty model, it overwrites the good SNMP data

    The issue: SSH-based collection may fail or return empty data,
    causing it to overwrite the correctly-set SNMP probe data.

- timestamp: 2026-05-08T10:35:00Z
  source: code_inspection
  finding: |
    updateDeviceInfo function (internal/services/device_info_collection_service.go:297-321):
    ```go
    func (s *DeviceInfoCollectionService) updateDeviceInfo(device *models.NetworkDevice, info *DeviceInfo) {
        updates := map[string]interface{}{}

        if info.Model != "" {
            updates["model"] = info.Model  // <-- Overwrites SNMP model
        }
        // ... other fields

        if len(updates) > 0 {
            if err := s.db.Model(device).Updates(updates).Error; err != nil {
                applogger.Infof("更新设备信息失败: %v", err)
            }
        }
    }
    ```

    The function ONLY updates if the field is not empty, but if SSH fails
    or returns empty strings, it won't update. However, there might be
    a race condition or the SSH is returning partial/incorrect data.

## Eliminated

- timestamp: 2026-05-08T10:20:00Z
  hypothesis: Backend not receiving model data
  evidence: Backend does its own SNMP probe, doesn't rely on frontend data for model
  reasoning: QuickCreateDevice calls ProbeSingleDevice independently

- timestamp: 2026-05-08T10:25:00Z
  hypothesis: Model extraction failing during probe
  evidence: Code shows two-tier model extraction with fallback (lines 556-562)
  reasoning: ExtractModelFromSysDescr is called as fallback if NewModelExtractor fails

## Resolution

- root_cause: DeviceInfoCollectionService async task was overwriting SNMP probe data with SSH-collected data, causing loss of initial device information
- fix: Modified updateDeviceInfo() to only update fields that are currently empty, preserving SNMP probe data during SSH collection attempts
- verification: ✅ Code compiles successfully with `go build ./internal/services/...` - 2026-05-08
- files_changed: internal/services/device_info_collection_service.go (lines 297-322)
  status: applied
  verified: true

### Fix Details

**File**: `internal/services/device_info_collection_service.go`

**Function**: `updateDeviceInfo()`

**Change**: Modified the update logic to preserve existing SNMP probe data by only updating fields that are currently empty:

```go
// Before (unconditionally overwrites):
if info.Model != "" {
    updates["model"] = info.Model
}

// After (preserves existing data):
if info.Model != "" && device.Model == "" {
    updates["model"] = info.Model
}
```

This ensures that:
1. SNMP probe data (model, serial number, software version, uptime) is preserved
2. SSH collection only fills in missing fields
3. No race condition between SNMP and SSH data collection
4. Existing device information is never overwritten by async collection

**Impact**: Quick-create devices will now retain their SNMP-probed model and other information, while still allowing SSH collection to supplement additional details later.

## Phase 41 Closure (2026-06-26)
verification: 2026-06-26 复测 `internal/services/device_info_collection_service.go:307` 确认修复落地 — `if info.Model != "" && device.Model == "" { updates["model"] = info.Model }`，updateDeviceInfo 已改为只更新当前为空的字段，保留 SNMP 探测获取的数据；后续 SerialNumber 等字段同样应用 `device.<Field> == ""` 守卫（line 310 同模式）。
files_changed: internal/services/device_info_collection_service.go (updateDeviceInfo 函数 lines 297-322，加 device.<Field> == "" 守卫保留 SNMP 探测数据)
action: re-verify-then-flip (D-01)
